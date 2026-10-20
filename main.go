package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/term"
)

type TickStats struct {
	Tick                   int
	TotalCreatures         int
	TotalCarnivores        int
	TotalHerbivores        int
	TotalDeaths            int
	AgeDeaths              int
	StarvationDeaths       int
	FightDeaths            int
	TotalAge               int
	TotalHealth            int
	TotalNutrition         int
	Clones                 int
	Eats                   int
	Moves                  int
	Fights                 int
	Sleeps                 int
	BiomassAdded           float64
	BiomassRegrown         float64
	BiomassRecovered       float64
	HealthAdded            int
	BiomassConsumed        float64
	BiomassTransferred     [plantSpeciesCount]float64
	PlantBiomassConsumed   [plantSpeciesCount]float64
	PlantRecoveries        [plantSpeciesCount]int
	PlantBiomassRecovered  [plantSpeciesCount]float64
	PlantBiomass           [plantSpeciesCount]float64
	PlantOccupiedCells     [plantSpeciesCount]int
	PlantEstablishments    [plantSpeciesCount]int
	PlantMutations         [plantSpeciesCount]int
	PlantTraits            [plantSpeciesCount]PlantTraitSummary
	PlantCapacity          float64
	PlantCumulative        PlantCumulativeTotals
	CarcassesCreated       int
	CarcassEnergyCreated   float64
	CarcassEnergyEaten     float64
	Reproduction           [reproductionDietCount]ReproductionStats
	ReproductionCumulative [reproductionDietCount]ReproductionStats
}

type PlantTraitSummary struct {
	RegrowthRate      float64
	SpreadProbability float64
	NutritionalYield  float64
}

type PlantCumulativeTotals struct {
	BiomassAdded       float64
	BiomassRegrown     float64
	BiomassRecovered   [plantSpeciesCount]float64
	Recoveries         [plantSpeciesCount]int
	BiomassTransferred [plantSpeciesCount]float64
	BiomassConsumed    [plantSpeciesCount]float64
	Establishments     [plantSpeciesCount]int
	Mutations          [plantSpeciesCount]int
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
const plantMutationProbability = 0.001
const plantMutationMagnitude = 0.0025
const plantRecoveryDelay = 20
const plantRecoveryBiomass = 5.0
const plantRecoveryReserve = 10.0

var plantSpread = [plantSpeciesCount]struct {
	threshold float64
	offsets   [][2]int
}{
	{threshold: 20, offsets: [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}},
	{threshold: 30, offsets: [][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}},
	{threshold: 40, offsets: [][2]int{{-2, -2}, {-1, -2}, {0, -2}, {1, -2}, {2, -2}, {-2, -1}, {-1, -1}, {0, -1}, {1, -1}, {2, -1}, {-2, 0}, {-1, 0}, {1, 0}, {2, 0}, {-2, 1}, {-1, 1}, {0, 1}, {1, 1}, {2, 1}, {-2, 2}, {-1, 2}, {0, 2}, {1, 2}, {2, 2}}},
}

type PlantState struct {
	Species           PlantSpecies
	Present           bool
	Biomass           float64
	Generation        int
	GrowthProgress    float64
	RegrowthRate      float64
	SpreadProbability float64
	NutritionalYield  float64
	RecoveryEligible  bool
	RecoveryReserve   float64
	DormantTicks      int
}

type PlantCellState struct {
	Capacity float64
	Plants   [plantSpeciesCount]PlantState
}

type plantPopulation struct {
	biomass           float64
	generation        int
	growthProgress    float64
	regrowthRate      float64
	spreadProbability float64
	nutritionalYield  float64
	recoveryEligible  bool
	recoveryReserve   float64
	dormantTicks      int
}

type environment struct {
	creatureList []*Creature
	capacity     float64
	plants       [plantSpeciesCount]plantPopulation
	carcass      carcassState
}

type carcassState struct {
	energy float64
}

var plantTraits = [plantSpeciesCount]struct {
	regrowth          float64
	spreadProbability float64
	nutrition         float64
}{
	{regrowth: 0.0968, spreadProbability: 0.02, nutrition: 16.5},
	{regrowth: 0.0484, spreadProbability: 0.01, nutrition: 22},
	{regrowth: 0.0242, spreadProbability: 0.005, nutrition: 11},
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
		regrowthRate, spreadProbability, nutritionalYield := population.regrowthRate, population.spreadProbability, population.nutritionalYield
		if regrowthRate <= 0 {
			regrowthRate = traits.regrowth
		}
		if spreadProbability <= 0 {
			spreadProbability = traits.spreadProbability
		}
		if nutritionalYield <= 0 {
			nutritionalYield = traits.nutrition
		}
		state.Plants[i] = PlantState{
			Species:           PlantSpecies(i),
			Present:           population.biomass > 0,
			Biomass:           population.biomass,
			Generation:        population.generation,
			GrowthProgress:    population.growthProgress,
			RegrowthRate:      regrowthRate,
			SpreadProbability: spreadProbability,
			NutritionalYield:  nutritionalYield,
			RecoveryEligible:  population.recoveryEligible,
			RecoveryReserve:   population.recoveryReserve,
			DormantTicks:      population.dormantTicks,
		}
	}
	return state, true
}

// Globals (limited; see AGENTS.md guidance)
var safeRunes []rune
var nextRuneIndex int
var isTest bool = false
var paused int32
var currentReportView int32
var leaderboardRequestCh = make(chan struct{}, 1)
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
	var tickStats TickStats
	showLeaderboard := false

	tickCh := startTicker(time.Duration(tickIntervalMillis) * time.Millisecond)

	for {
		select {
		case <-exitCh:
			fmt.Print("\r\nExited simulation.")
			return
		case <-leaderboardRequestCh:
			showLeaderboard = !showLeaderboard
		case <-tickCh:
			processSpawnRequests(env, creatureFac)
			if atomic.LoadInt32(&paused) == 1 {
				continue
			}

			tickCount++
			tickStats.Tick = tickCount

			simulateTick(env, creatureFac.create, &tickStats, &carnivoreTotals)
			collectAncestryStats(env, &ancestryStats)
			printMap(env)
			width, _, err := term.GetSize(int(os.Stdout.Fd()))
			if err != nil {
				width = 81
			}
			view := reportView(atomic.LoadInt32(&currentReportView))
			printSelectedReport(&tickStats, &carnivoreTotals, view, width)

			if showLeaderboard {
				printAncestorLeaderboard(&ancestryStats)
			}
			if view == reportViewHelp {
				fmt.Printf("\r\nUp/Down arrows adjust speed (%dms/tick)", currentTickInterval()/time.Millisecond)
			}
			fmt.Print("\r\nPress '?' to toggle the help menu")

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
			if RandIntn(1000) >= 45 { // 4.5% initial spawn candidate rate
				continue
			}

			diet := DietHerbivore
			if RandIntn(100) < 5 { // ~5% of initial spawns are carnivores
				diet = DietCarnivore
			} else if RandIntn(2) == 0 { // Retain half of initial herbivore candidates.
				continue
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
	centers := 6 * max(1, (cellCount+299)/300)
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
					biomass:           biomass * scale,
					regrowthRate:      plantTraits[species].regrowth,
					spreadProbability: plantTraits[species].spreadProbability,
					nutritionalYield:  plantTraits[species].nutrition,
					recoveryEligible:  biomass*scale > 0,
					recoveryReserve:   recoveryReserveFor(biomass * scale),
				}
			}
		}
	}
}

func simulateTick(env [][]environment, creatureFac func(*Creature) *Creature, tickStats *TickStats, carnivoreTotals *CarnivoreActionTotals) {
	// Per-tick values reset; plant cumulative totals survive across calls.
	cumulativePlants := tickStats.PlantCumulative
	cumulativeReproduction := tickStats.ReproductionCumulative
	*tickStats = TickStats{Tick: tickStats.Tick, PlantCumulative: cumulativePlants, ReproductionCumulative: cumulativeReproduction}
	processEnvironment(env, tickStats, carnivoreTotals)
	creatures := flattenCreatureList(env)
	for _, c := range creatures {
		c.age++
		if c.immobileTicks > 0 {
			c.immobileTicks--
		}
		if c.cloneCooldown > 0 {
			c.cloneCooldown--
		}
		if c.handlingTicks > 0 {
			c.handlingTicks--
		}
		processNutrition(c)
		processHealth(c, tickStats, carnivoreTotals)
	}
	removeDeadCreatures(env)
	creatures = flattenCreatureList(env)
	intents := collectIntents(creatures, env, tickStats)
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
				population := cell.plants[species]
				tickStats.PlantBiomass[species] += population.biomass
				if population.biomass <= 0 {
					continue
				}
				tickStats.PlantOccupiedCells[species]++
				traits := plantTraits[species]
				regrowth, spread, nutrition := population.regrowthRate, population.spreadProbability, population.nutritionalYield
				if regrowth <= 0 {
					regrowth = traits.regrowth
				}
				if spread <= 0 {
					spread = traits.spreadProbability
				}
				if nutrition <= 0 {
					nutrition = traits.nutrition
				}
				biomass := population.biomass
				tickStats.PlantTraits[species].RegrowthRate += regrowth * biomass
				tickStats.PlantTraits[species].SpreadProbability += spread * biomass
				tickStats.PlantTraits[species].NutritionalYield += nutrition * biomass
			}
		}
	}
	for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
		biomass := tickStats.PlantBiomass[species]
		if biomass > 0 {
			tickStats.PlantTraits[species].RegrowthRate /= biomass
			tickStats.PlantTraits[species].SpreadProbability /= biomass
			tickStats.PlantTraits[species].NutritionalYield /= biomass
		}
		tickStats.PlantCumulative.BiomassTransferred[species] += tickStats.BiomassTransferred[species]
		tickStats.PlantCumulative.BiomassRecovered[species] += tickStats.PlantBiomassRecovered[species]
		tickStats.PlantCumulative.Recoveries[species] += tickStats.PlantRecoveries[species]
		tickStats.PlantCumulative.BiomassConsumed[species] += tickStats.PlantBiomassConsumed[species]
		tickStats.PlantCumulative.Establishments[species] += tickStats.PlantEstablishments[species]
		tickStats.PlantCumulative.Mutations[species] += tickStats.PlantMutations[species]
	}
	tickStats.PlantCumulative.BiomassAdded += tickStats.BiomassAdded
	tickStats.PlantCumulative.BiomassRegrown += tickStats.BiomassRegrown
	for diet := 0; diet < reproductionDietCount; diet++ {
		addReproductionStats(&tickStats.ReproductionCumulative[diet], tickStats.Reproduction[diet])
	}
}

func printMap(env [][]environment) {
	fmt.Print(renderMap(env))
}

func renderMap(env [][]environment) string {
	var buf strings.Builder
	columns := 0
	for _, row := range env {
		columns += len(row) * int(plantSpeciesCount)
	}
	buf.Grow(columns*24 + len(env)*2 + 160)
	const reset = "\033[0m"
	const darkRedAccent = "\033[38;5;88m"
	buf.WriteString("\033[H\033[2J")
	for i := range env {
		for j := range env[i] {
			cell := env[i][j]
			creatures := cell.creatureList
			hasCarnivore := cellHasCarnivore(creatures)
			for position := 0; position < int(plantSpeciesCount); position++ {
				plant := cell.plants[position]
				bg, texture := plantTexture(PlantSpecies(position), plant.biomass, cell.capacity)
				buf.WriteString(bg)
				glyph, foreground := texture, ""
				switch {
				case hasCarnivore && len(creatures) == 1:
					if position == 0 {
						glyph, foreground = '[', darkRedAccent
					} else if position == 1 {
						glyph, foreground = creatures[0].char, creatures[0].colorCode
					} else {
						glyph, foreground = ']', darkRedAccent
					}
				case hasCarnivore && len(creatures) > 1:
					glyph, foreground = [3]rune{'[', rune('0' + min(len(creatures), 9)), ']'}[position], darkRedAccent
				case len(creatures) > 3:
					first := creatures[0]
					switch position {
					case 0:
						glyph, foreground = first.char, first.colorCode
					case 1:
						glyph = '+'
					case 2:
						glyph = rune('0' + min(len(creatures)-1, 9))
					}
				case len(creatures) == 1:
					if position == 1 {
						glyph, foreground = creatures[0].char, creatures[0].colorCode
					} else {
						glyph = ' '
					}
				default:
					if position < len(creatures) && creatures[position] != nil {
						glyph, foreground = creatures[position].char, creatures[position].colorCode
					}
				}
				if foreground != "" {
					buf.WriteString(foreground)
				}
				buf.WriteRune(glyph)
				buf.WriteString(reset)
			}
		}
		buf.WriteString("\n\r")
	}
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
	update := updatePlants(env)
	if tickStats != nil {
		tickStats.BiomassAdded += update.biomassAdded
		tickStats.BiomassRegrown += update.regrown
		for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
			tickStats.PlantBiomassRecovered[species] += update.recovered[species]
			tickStats.PlantRecoveries[species] += update.recoveries[species]
			tickStats.BiomassRecovered += update.recovered[species]
			tickStats.BiomassTransferred[species] += update.transferred[species]
			tickStats.PlantEstablishments[species] += update.establishments[species]
			tickStats.PlantMutations[species] += update.mutations[species]
		}
	}
	for i := range env {
		for j := range env[i] {
			cell := &env[i][j]
			cell.carcass.energy *= 1 - carcassDecayPerTick
			if cell.carcass.energy < 0.01 {
				cell.carcass = carcassState{}
			}
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

type plantUpdateResult struct {
	biomassAdded   float64
	regrown        float64
	recoveries     [plantSpeciesCount]int
	recovered      [plantSpeciesCount]float64
	transferred    [plantSpeciesCount]float64
	establishments [plantSpeciesCount]int
	mutations      [plantSpeciesCount]int
}

// updatePlants bases growth and spread decisions on one tick-start snapshot,
// then applies growth and propagule transfers without within-tick cascades.
func updatePlants(env [][]environment) plantUpdateResult {
	var result plantUpdateResult
	var intents []plantSpreadIntent
	for y := range env {
		for x := range env[y] {
			for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
				colony := env[y][x].plants[species]
				config := plantSpread[species]
				spreadProbability := colony.spreadProbability
				if spreadProbability <= 0 {
					spreadProbability = plantTraits[species].spreadProbability
				}
				if colony.biomass < config.threshold || RandFloat64() >= spreadProbability || len(config.offsets) == 0 {
					continue
				}
				offset := config.offsets[RandIntn(len(config.offsets))]
				targetX, targetY := x+offset[0], y+offset[1]
				if targetY < 0 || targetY >= len(env) || targetX < 0 || targetX >= len(env[targetY]) || env[targetY][targetX].plants[species].biomass > 0 {
					continue
				}
				intents = append(intents, plantSpreadIntent{
					sourceX: x, sourceY: y, targetX: targetX, targetY: targetY,
					species: species, priority: RandFloat64(),
				})
			}
		}
	}

	for y := range env {
		for x := range env[y] {
			regrown := regrowPlants(&env[y][x])
			result.regrown += regrown
			result.biomassAdded += regrown
			for species, amount := range recoverPlants(&env[y][x]) {
				result.recovered[species] += amount
				if amount > 0 {
					result.recoveries[species]++
				}
				result.biomassAdded += amount
			}
		}
	}
	slices.SortFunc(intents, func(a, b plantSpreadIntent) int {
		if a.priority != b.priority {
			if a.priority < b.priority {
				return -1
			}
			return 1
		}
		if a.targetY != b.targetY {
			return a.targetY - b.targetY
		}
		if a.targetX != b.targetX {
			return a.targetX - b.targetX
		}
		if a.sourceY != b.sourceY {
			return a.sourceY - b.sourceY
		}
		if a.sourceX != b.sourceX {
			return a.sourceX - b.sourceX
		}
		if a.species < b.species {
			return -1
		}
		if a.species > b.species {
			return 1
		}
		return 0
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
		child, mutated := inheritPlantColonyWithMutation(source.plants[intent.species], intent.species)
		source.plants[intent.species].biomass--
		if source.plants[intent.species].biomass <= 1e-12 {
			clearLivingPlant(&source.plants[intent.species])
		}
		founder := target.plants[intent.species]
		child.recoveryEligible = founder.recoveryEligible
		child.recoveryReserve = founder.recoveryReserve
		child.dormantTicks = 0
		target.plants[intent.species] = child
		result.transferred[intent.species]++
		result.establishments[intent.species]++
		if mutated {
			result.mutations[intent.species]++
		}
	}
	return result
}

func inheritPlantColony(parent plantPopulation, species PlantSpecies) plantPopulation {
	child, _ := inheritPlantColonyWithMutation(parent, species)
	return child
}

func inheritPlantColonyWithMutation(parent plantPopulation, species PlantSpecies) (plantPopulation, bool) {
	traits := plantTraits[species]
	regrowthRate, spreadProbability, nutritionalYield := parent.regrowthRate, parent.spreadProbability, parent.nutritionalYield
	if regrowthRate <= 0 {
		regrowthRate = traits.regrowth
	}
	if spreadProbability <= 0 {
		spreadProbability = traits.spreadProbability
	}
	if nutritionalYield <= 0 {
		nutritionalYield = traits.nutrition
	}
	child := plantPopulation{
		biomass: 1, generation: parent.generation + 1,
		regrowthRate: regrowthRate, spreadProbability: spreadProbability,
		nutritionalYield: nutritionalYield,
	}
	if RandFloat64() < plantMutationProbability {
		mutatePlantTrait(&child, species, RandIntn(3), (RandFloat64()*2-1)*plantMutationMagnitude)
		return child, true
	}
	return child, false
}

func mutatePlantTrait(colony *plantPopulation, species PlantSpecies, traitIndex int, relativeChange float64) {
	base := plantTraits[species]
	if relativeChange < -plantMutationMagnitude {
		relativeChange = -plantMutationMagnitude
	}
	if relativeChange > plantMutationMagnitude {
		relativeChange = plantMutationMagnitude
	}

	var value, baseValue float64
	switch traitIndex {
	case 0:
		value, baseValue = colony.regrowthRate, base.regrowth
	case 1:
		value, baseValue = colony.spreadProbability, base.spreadProbability
	case 2:
		value, baseValue = colony.nutritionalYield, base.nutrition
	default:
		return
	}
	minimum, maximum := baseValue*0.8, baseValue*1.2
	value = minFloat(maximum, maxFloat(minimum, value*(1+relativeChange)))
	switch traitIndex {
	case 0:
		colony.regrowthRate = value
	case 1:
		colony.spreadProbability = value
	case 2:
		colony.nutritionalYield = value
	}
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

func recoveryReserveFor(initialBiomass float64) float64 {
	if initialBiomass <= 0 {
		return 0
	}
	return plantRecoveryReserve
}

func recoverPlants(cell *environment) [plantSpeciesCount]float64 {
	var recovered [plantSpeciesCount]float64
	if cell == nil || cell.capacity <= 0 {
		return recovered
	}
	totalBiomass := 0.0
	for _, population := range cell.plants {
		totalBiomass += maxFloat(population.biomass, 0)
	}
	freeCapacity := maxFloat(cell.capacity-totalBiomass, 0)
	for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
		population := &cell.plants[species]
		if !population.recoveryEligible || population.biomass > 0 || population.recoveryReserve <= 0 {
			continue
		}
		if population.dormantTicks < plantRecoveryDelay {
			population.dormantTicks++
		}
		if population.dormantTicks < plantRecoveryDelay || freeCapacity <= 0 {
			continue
		}
		amount := minFloat(plantRecoveryBiomass, minFloat(population.recoveryReserve, freeCapacity))
		if amount <= 0 {
			continue
		}
		traits := plantTraits[species]
		population.biomass = amount
		population.generation = 0
		population.growthProgress = 0
		population.regrowthRate = traits.regrowth
		population.spreadProbability = traits.spreadProbability
		population.nutritionalYield = traits.nutrition
		population.recoveryReserve -= amount
		population.dormantTicks = 0
		freeCapacity -= amount
		recovered[species] = amount
	}
	return recovered
}

// clearLivingPlant removes a colony while preserving any founder's dormant roots.
func clearLivingPlant(population *plantPopulation) {
	if population == nil {
		return
	}
	eligible, reserve := population.recoveryEligible, population.recoveryReserve
	dormantTicks := population.dormantTicks
	*population = plantPopulation{
		recoveryEligible: eligible,
		recoveryReserve:  reserve,
		dormantTicks:     dormantTicks,
	}
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
