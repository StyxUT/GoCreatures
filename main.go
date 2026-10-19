package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/term"
)

type TickStats struct {
	Tick               int
	TotalCreatures     int
	TotalCarnivores    int
	TotalHerbivores    int
	TotalDeaths        int
	AgeDeaths          int
	StarvationDeaths   int
	FightDeaths        int
	TotalAge           int
	TotalHealth        int
	TotalNutrition     int
	Clones             int
	Eats               int
	Moves              int
	Fights             int
	Sleeps             int
	BiomassAdded       float64
	HealthAdded        int
	BiomassConsumed    float64
	BiomassTransferred [plantSpeciesCount]float64
	PlantBiomass       [plantSpeciesCount]float64
	PlantCapacity      float64
}

type CarnivoreActionTotals struct {
	Eats             int
	Kills            int
	Clones           int
	Moves            int
	Fights           int
	Sleeps           int
	AgeDeaths        int
	StarvationDeaths int
	FightDeaths      int
}

// PlantSpecies has a stable order used by cell snapshots and deterministic ties.
type PlantSpecies uint8

const (
	PlantGrass PlantSpecies = iota
	PlantClover
	PlantBramble
	plantSpeciesCount
)

const plantCellCapacity = 100.0

var plantSpread = [plantSpeciesCount]struct {
	threshold   float64
	probability float64
	offsets     [][2]int
}{
	{threshold: 20, probability: 0.02, offsets: [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}},
	{threshold: 30, probability: 0.01, offsets: [][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}},
	{threshold: 40, probability: 0.005, offsets: [][2]int{{-2, -2}, {-1, -2}, {0, -2}, {1, -2}, {2, -2}, {-2, -1}, {-1, -1}, {0, -1}, {1, -1}, {2, -1}, {-2, 0}, {-1, 0}, {1, 0}, {2, 0}, {-2, 1}, {-1, 1}, {0, 1}, {1, 1}, {2, 1}, {-2, 2}, {-1, 2}, {0, 2}, {1, 2}, {2, 2}}},
}

type PlantState struct {
	Species          PlantSpecies
	Present          bool
	Biomass          float64
	GrowthProgress   float64
	RegrowthRate     float64
	NutritionalYield float64
}

type PlantCellState struct {
	Capacity float64
	Plants   [plantSpeciesCount]PlantState
}

type plantPopulation struct {
	biomass          float64
	growthProgress   float64
	regrowthRate     float64
	nutritionalYield float64
}

type environment struct {
	creatureList []*Creature
	capacity     float64
	plants       [plantSpeciesCount]plantPopulation
}

var plantTraits = [plantSpeciesCount]struct {
	regrowth  float64
	nutrition float64
}{
	{regrowth: 0.08, nutrition: 15},
	{regrowth: 0.04, nutrition: 20},
	{regrowth: 0.02, nutrition: 10},
}

// PlantCellAt returns a value snapshot in stable grass, clover, bramble order.
// Mutating the returned value cannot modify the simulation state.
func PlantCellAt(env [][]environment, x, y int) (PlantCellState, bool) {
	if y < 0 || y >= len(env) || x < 0 || x >= len(env[y]) {
		return PlantCellState{}, false
	}
	cell := &env[y][x]
	state := PlantCellState{Capacity: cell.capacity}
	for i, population := range cell.plants {
		traits := plantTraits[i]
		regrowthRate, nutritionalYield := population.regrowthRate, population.nutritionalYield
		if regrowthRate <= 0 {
			regrowthRate = traits.regrowth
		}
		if nutritionalYield <= 0 {
			nutritionalYield = traits.nutrition
		}
		state.Plants[i] = PlantState{
			Species:          PlantSpecies(i),
			Present:          population.biomass > 0,
			Biomass:          population.biomass,
			GrowthProgress:   population.growthProgress,
			RegrowthRate:     regrowthRate,
			NutritionalYield: nutritionalYield,
		}
	}
	return state, true
}

// Globals (limited; see AGENTS.md guidance)
var safeRunes []rune
var nextRuneIndex int
var isTest bool = false
var paused int32
var tickIntervalMillis int64 = 300
var exitCh = make(chan struct{})
var tickSpeedCh = make(chan time.Duration, 1)
var spawnDietCh = make(chan DietType, 32)
var exitOnce sync.Once

const (
	minTickInterval  = 50 * time.Millisecond
	maxTickInterval  = 1000 * time.Millisecond
	tickIntervalStep = 50 * time.Millisecond
)

func signalExit() { exitOnce.Do(func() { close(exitCh) }) }

func main() {
	// Enter raw mode once and ensure restoration even on early return.
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		if oldState, err := term.MakeRaw(fd); err == nil {
			defer term.Restore(fd, oldState)
		}
	}

	startKeyListener() // handles pause + manual exit

	rows, cols := 38, 81
	creatureFac := CreatureFactory()
	env := initializeMap(rows, cols, creatureFac)
	var tickCount int
	var ancestryStats []AncestryStats
	var carnivoreTotals CarnivoreActionTotals

	tickCh := startTicker(time.Duration(tickIntervalMillis) * time.Millisecond)

	for {
		select {
		case <-exitCh:
			fmt.Print("\r\nExited simulation.")
			return
		case <-tickCh:
			processSpawnRequests(env, creatureFac)
			if atomic.LoadInt32(&paused) == 1 {
				continue
			}

			tickCount++
			var tickStats TickStats
			tickStats.Tick = tickCount

			simulateTick(env, creatureFac.create, &tickStats, &carnivoreTotals)
			collectAncestryStats(env, &ancestryStats)
			printMap(env)
			printTickStats(&tickStats, &carnivoreTotals)

			fmt.Print("\r\n*** Ancestor Leaderboard ***\r\n")
			printAncestryStats(&ancestryStats)
			fmt.Print("\r\nPress <space> to pause or 'e' to exit")
			fmt.Printf("\r\nUp/Down arrows adjust speed (%dms/tick)", currentTickInterval()/time.Millisecond)
			fmt.Print("\r\nPress 'c' to spawn Carnivore, 'h' to spawn Herbivore")

			if len(ancestryStats) <= 1 {
				PrintNewestClones(env)
				signalExit()
				return
			}
			if tickStats.TotalCreatures == 0 {
				fmt.Println("All creatures are gone. Simulation ended.")
				signalExit()
				return
			}
		}
	}
}

func init() { safeRunes = buildSingleWidthRunePool(33, 0x02AF) }

func startTicker(tickInterval time.Duration) <-chan time.Time {
	atomic.StoreInt64(&tickIntervalMillis, tickInterval.Milliseconds())
	ticker := time.NewTicker(tickInterval)
	ch := make(chan time.Time)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-exitCh:
				return
			case interval := <-tickSpeedCh:
				ticker.Reset(interval)
			case t := <-ticker.C:
				ch <- t
			}
		}
	}()
	return ch
}

func currentTickInterval() time.Duration {
	return time.Duration(atomic.LoadInt64(&tickIntervalMillis)) * time.Millisecond
}

func clampTickInterval(interval time.Duration) time.Duration {
	if interval < minTickInterval {
		return minTickInterval
	}
	if interval > maxTickInterval {
		return maxTickInterval
	}
	return interval
}

func setTickInterval(interval time.Duration) time.Duration {
	interval = clampTickInterval(interval)
	atomic.StoreInt64(&tickIntervalMillis, interval.Milliseconds())
	select {
	case tickSpeedCh <- interval:
	default:
		select {
		case <-tickSpeedCh:
		default:
		}
		tickSpeedCh <- interval
	}
	return interval
}

func adjustTickInterval(delta time.Duration) time.Duration {
	return setTickInterval(currentTickInterval() + delta)
}

func processSpawnRequests(env [][]environment, createCreature *creatureFactory) {
	for {
		select {
		case diet := <-spawnDietCh:
			spawnCreatureWithDiet(env, createCreature, diet)
		default:
			return
		}
	}
}

func spawnCreatureWithDiet(env [][]environment, createCreature *creatureFactory, diet DietType) {
	if len(env) == 0 || len(env[0]) == 0 {
		return
	}

	rows := len(env)
	cols := len(env[0])
	x := RandIntn(cols)
	y := RandIntn(rows)

	c := createCreature.createForDiet(diet)
	if c == nil {
		return
	}
	c.X = x
	c.Y = y
	env[y][x].creatureList = append(env[y][x].creatureList, c)
}

func initializeMap(rows, cols int, createCreature *creatureFactory) [][]environment {
	matrix := make([][]environment, rows)
	for i := range matrix {
		matrix[i] = make([]environment, cols)
		for j := range matrix[i] {
			matrix[i][j].capacity = plantCellCapacity
			if RandIntn(1000) >= 90 { // 9.0% initial spawn rate
				continue
			}

			diet := DietHerbivore
			if RandIntn(100) < 5 { // ~5% of initial spawns are carnivores
				diet = DietCarnivore
			}

			c := createCreature.createForDiet(diet)
			if c == nil {
				continue
			}
			c.X, c.Y = j, i
			matrix[i][j].creatureList = append(matrix[i][j].creatureList, c)
		}
	}
	seedPlants(matrix)
	return matrix
}

func seedPlants(env [][]environment) {
	rows := len(env)
	if rows == 0 {
		return
	}
	cols := len(env[0])
	if cols == 0 {
		return
	}
	cellCount := rows * cols
	centers := max(1, (cellCount+299)/300)
	requested := make([][][plantSpeciesCount]float64, rows)
	for y := range requested {
		requested[y] = make([][plantSpeciesCount]float64, len(env[y]))
	}

	for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
		for range centers {
			x, y := RandIntn(cols), RandIntn(rows)
			requested[y][x][species] += 20
			for _, offset := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				haloX, haloY := x+offset[0], y+offset[1]
				if haloY >= 0 && haloY < rows && haloX >= 0 && haloX < len(env[haloY]) {
					requested[haloY][haloX][species] += 5
				}
			}
		}
	}

	for y := range env {
		for x := range env[y] {
			total := 0.0
			for _, biomass := range requested[y][x] {
				total += biomass
			}
			scale := 1.0
			if total > env[y][x].capacity {
				scale = env[y][x].capacity / total
			}
			for species, biomass := range requested[y][x] {
				env[y][x].plants[species] = plantPopulation{
					biomass:          biomass * scale,
					regrowthRate:     plantTraits[species].regrowth,
					nutritionalYield: plantTraits[species].nutrition,
				}
			}
		}
	}
}

func simulateTick(env [][]environment, creatureFac func(*Creature) *Creature, tickStats *TickStats, carnivoreTotals *CarnivoreActionTotals) {
	// Each result describes this tick; cumulative carnivore totals are separate.
	*tickStats = TickStats{Tick: tickStats.Tick}
	processEnvironment(env, tickStats, carnivoreTotals)
	creatures := flattenCreatureList(env)
	for _, c := range creatures {
		c.age++
		if c.immobileTicks > 0 {
			c.immobileTicks--
		}
		processNutrition(c)
		processHealth(c, tickStats, carnivoreTotals)
	}
	removeDeadCreatures(env)
	creatures = flattenCreatureList(env)
	intents := collectIntents(creatures, env)
	resolveActions(intents, env, creatureFac, tickStats, carnivoreTotals)
	removeDeadCreatures(env)
	remaining := flattenCreatureList(env)
	tickStats.TotalDeaths = tickStats.AgeDeaths + tickStats.StarvationDeaths + tickStats.FightDeaths
	tickStats.TotalCreatures = len(remaining)
	for _, c := range remaining {
		tickStats.TotalAge += c.age
		tickStats.TotalHealth += c.health
		tickStats.TotalNutrition += c.nutrition
		if c.isCarnivore() {
			tickStats.TotalCarnivores++
		} else {
			tickStats.TotalHerbivores++
		}
	}
	for y := range env {
		for x := range env[y] {
			cell := &env[y][x]
			tickStats.PlantCapacity += cell.capacity
			for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
				tickStats.PlantBiomass[species] += cell.plants[species].biomass
			}
		}
	}
}

func printMap(env [][]environment) {
	fmt.Print(renderMap(env))
}

func renderMap(env [][]environment) string {
	var buf bytes.Buffer
	const reset = "\033[0m"
	const darkRedAccent = "\033[38;5;88m"
	buf.WriteString("\033[H\033[2J")
	for i := range env {
		for j := range env[i] {
			cell := env[i][j]
			creatures := cell.creatureList
			for position := 0; position < int(plantSpeciesCount); position++ {
				plant := cell.plants[position]
				bg, texture := plantTexture(PlantSpecies(position), plant.biomass, cell.capacity)
				buf.WriteString(bg)
				if position < len(creatures) && creatures[position] != nil {
					creature := creatures[position]
					if creature.isCarnivore() {
						fmt.Fprintf(&buf, "%s%c", darkRedAccent, creature.char)
					} else {
						fmt.Fprintf(&buf, "%s%c", creature.colorCode, creature.char)
					}
				} else {
					buf.WriteRune(texture)
				}
				buf.WriteString(reset)
			}
		}
		buf.WriteString("\n\r")
	}
	buf.WriteString("Grass |I  Clover *o@  Bramble :xX#  (soil .; sparse 1-9%, medium 10-39%, rich 40-69%, dense 70-100%; creature glyphs cover texture)\n")
	return buf.String()
}

func cellHasCarnivore(creatures []*Creature) bool {
	for _, c := range creatures {
		if c.isCarnivore() {
			return true
		}
	}
	return false
}

func processEnvironment(env [][]environment, tickStats *TickStats, carnivoreTotals *CarnivoreActionTotals) {
	added, transferred := updatePlants(env)
	if tickStats != nil {
		tickStats.BiomassAdded += added
		for species, amount := range transferred {
			tickStats.BiomassTransferred[species] += amount
		}
	}
	for i := range env {
		for j := range env[i] {
			cell := &env[i][j]
			newList := cell.creatureList[:0]
			for _, c := range cell.creatureList {
				switch {
				case c.health <= 0:
					// Already dead: do not count another cause of death.
				case c.age >= c.maxAge:
					tickStats.AgeDeaths++
					if carnivoreTotals != nil && c.isCarnivore() {
						carnivoreTotals.AgeDeaths++
					}
				default:
					newList = append(newList, c)
				}
			}
			cell.creatureList = newList
		}
	}
}

type plantSpreadIntent struct {
	sourceX, sourceY int
	targetX, targetY int
	species          PlantSpecies
	priority         float64
}

// updatePlants bases growth and spread decisions on one tick-start snapshot,
// then applies growth and propagule transfers without within-tick cascades.
func updatePlants(env [][]environment) (float64, [plantSpeciesCount]float64) {
	var transferred [plantSpeciesCount]float64
	snapshot := make([][][plantSpeciesCount]plantPopulation, len(env))
	for y := range env {
		snapshot[y] = make([][plantSpeciesCount]plantPopulation, len(env[y]))
		for x := range env[y] {
			snapshot[y][x] = env[y][x].plants
		}
	}

	var intents []plantSpreadIntent
	for y := range snapshot {
		for x := range snapshot[y] {
			for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
				colony := snapshot[y][x][species]
				config := plantSpread[species]
				if colony.biomass < config.threshold || RandFloat64() >= config.probability || len(config.offsets) == 0 {
					continue
				}
				offset := config.offsets[RandIntn(len(config.offsets))]
				targetX, targetY := x+offset[0], y+offset[1]
				if targetY < 0 || targetY >= len(snapshot) || targetX < 0 || targetX >= len(snapshot[targetY]) || snapshot[targetY][targetX][species].biomass > 0 {
					continue
				}
				intents = append(intents, plantSpreadIntent{
					sourceX: x, sourceY: y, targetX: targetX, targetY: targetY,
					species: species, priority: RandFloat64(),
				})
			}
		}
	}

	added := 0.0
	for y := range env {
		for x := range env[y] {
			added += regrowPlants(&env[y][x])
		}
	}
	sort.Slice(intents, func(i, j int) bool {
		a, b := intents[i], intents[j]
		if a.priority != b.priority {
			return a.priority < b.priority
		}
		if a.targetY != b.targetY {
			return a.targetY < b.targetY
		}
		if a.targetX != b.targetX {
			return a.targetX < b.targetX
		}
		if a.sourceY != b.sourceY {
			return a.sourceY < b.sourceY
		}
		if a.sourceX != b.sourceX {
			return a.sourceX < b.sourceX
		}
		return a.species < b.species
	})
	for _, intent := range intents {
		source := &env[intent.sourceY][intent.sourceX]
		target := &env[intent.targetY][intent.targetX]
		if source.plants[intent.species].biomass < 1 || target.plants[intent.species].biomass > 0 {
			continue
		}
		total := 0.0
		for _, population := range target.plants {
			total += population.biomass
		}
		if target.capacity-total < 1 {
			continue
		}
		source.plants[intent.species].biomass--
		target.plants[intent.species] = plantPopulation{
			biomass: 1, regrowthRate: source.plants[intent.species].regrowthRate,
			nutritionalYield: source.plants[intent.species].nutritionalYield,
		}
		transferred[intent.species]++
	}
	return added, transferred
}

func regrowPlants(cell *environment) float64 {
	if cell == nil || cell.capacity <= 0 {
		return 0
	}

	totalBiomass := 0.0
	for _, population := range cell.plants {
		totalBiomass += maxFloat(population.biomass, 0)
	}
	freeCapacity := maxFloat(cell.capacity-totalBiomass, 0)
	if freeCapacity == 0 {
		return 0
	}
	freeFraction := freeCapacity / cell.capacity
	totalProgress := 0.0
	for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
		if cell.plants[species].biomass <= 0 {
			cell.plants[species].growthProgress = 0
			continue
		}
		regrowthRate := cell.plants[species].regrowthRate
		if regrowthRate <= 0 {
			regrowthRate = plantTraits[species].regrowth
		}
		cell.plants[species].growthProgress += regrowthRate * freeFraction
		totalProgress += cell.plants[species].growthProgress
	}
	if totalProgress == 0 {
		return 0
	}
	growth := minFloat(freeCapacity, totalProgress)
	growthFraction := growth / totalProgress
	allocated := 0.0
	lastGrowing := PlantGrass
	for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
		if cell.plants[species].growthProgress > 0 {
			lastGrowing = species
		}
	}
	for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
		population := &cell.plants[species]
		progress := population.growthProgress
		share := progress * growthFraction
		if species == lastGrowing {
			share = growth - allocated
		}
		population.biomass += share
		if growth == totalProgress {
			population.growthProgress = 0
		} else {
			population.growthProgress = maxFloat(progress-share, 0)
		}
		allocated += share
	}
	return growth
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Removal never records a death: the live-to-dead transition owns that event.
func removeDeadCreatures(env [][]environment) {
	for y := range env {
		for x := range env[y] {
			cell := &env[y][x]
			living := cell.creatureList[:0]
			for _, c := range cell.creatureList {
				if c.health > 0 {
					living = append(living, c)
				}
			}
			clear(cell.creatureList[len(living):])
			cell.creatureList = living
		}
	}
}
