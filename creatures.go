package main

import (
	"bytes"
	"fmt"
	"golang.org/x/text/width"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
	"unicode"
)

type creature struct {
	num          int
	parent       *creature
	ancestor     *creature
	char         rune
	colorCode    string
	health       int
	maxHealth    int
	X, Y         int
	age          int
	maxAge       int
	metabolism   int
	nutrition    int
	maxNutrition int
	aggression   int
}

type environment struct {
	creatureList []*creature
	food         int
	maxFood      int
}

type ActionIntent struct {
	Creature *creature
	Type     string // e.g., "move", "rest", etc.
	TargetX  int
	TargetY  int
}

type TickStats struct {
	Tick             int
	TotalCreatures   int
	TotalDeaths      int
	AgeDeaths        int
	StarvationDeaths int
	TotalAge         int
	TotalHealth      int
	TotalNutrition   int
	Clones           int
	Eats             int
	Moves            int
	Fights           int
	FoodAdded        int
	HealthAdded      int
	FoodConsumed     int
}

type AncestryStats struct {
	Ancestor     *creature
	LineageCount int
}

// TODO //
// Add ancestor reportint
//
// Add combat
// less likely to fight when; plenty of food, same a parent, same ancestor
// damage given based on what?
// damage taken based on what?
//
// Add move cost
// apply nutrition cost to moving
//
// Add rest
// reducted nutrition cost if nothing done
// increased healing if needed (normal nutrition cost)
//
// Add vision
// vision score could determine distance, and direction of sight
//
//	3 cells (front, front left, front right of last move direction
//	front left and right/ just front but far?
//
// will need to rack the directon of last movement
// make more intelligent movement like avoid other creatures, or move toward food
var safeRunes []rune
var nextRuneIndex int

func main() {
	rows, cols := 40, 70
	creatureFac := creatureFactory()
	env := initializeMap(rows, cols, creatureFac)
	var tickCount int

	tickCh := startTicker(100 * time.Millisecond)

	for range tickCh {
		tickCount++
		var tickStats TickStats
		var ancestryStats []AncestryStats

		tickStats.Tick = tickCount

		simulateTick(env, creatureFac, &tickStats)
		collectAncestryStats(env, &ancestryStats)
		printMap(env)
		printTickStats(&tickStats)
		printAncestryStats(&ancestryStats)

		if tickStats.TotalCreatures == 0 {
			fmt.Println("All creatures are gone. Simulation ended.")
			break
		}
	}
}

func init() {
	safeRunes = buildSingleWidthRunePool(33, 0x02AF) // safe visible range
}

func startTicker(tickInterval time.Duration) <-chan time.Time {
	ticker := time.NewTicker(tickInterval)
	tickCh := make(chan time.Time)

	go func() {
		for t := range ticker.C {
			tickCh <- t
		}
	}()

	return tickCh
}

// Initialize and populate the map
func initializeMap(rows int, cols int, createCreature func(p *creature) *creature) [][]environment {
	matrix := make([][]environment, rows)
	for i := range matrix {
		matrix[i] = make([]environment, cols)
		for j := range matrix[i] {
			// Set location max and current food
			maxFood := rand.Intn(80)
			matrix[i][j].maxFood = maxFood                                            // set max food
			matrix[i][j].food = int(float64(maxFood) * (float64(rand.Intn(10)) / 20)) // set current food a perentage of maxFood

			// Populate with creatures
			if rand.Intn(20) != 0 { // populate a subset of the map locations
				continue
			}
			creature := createCreature(nil)
			creature.X = j
			creature.Y = i
			matrix[i][j].creatureList = append(matrix[i][j].creatureList, creature)
		}
	}
	return matrix
}

// Generate a creature and set it's characteristics
func creatureFactory() func(parent *creature) *creature {
	n := 1

	return func(parent *creature) *creature {
		var color string
		var maxAge int
		var maxHealth int
		var health int
		var metabolism int
		var nutrition int
		var maxNutrition int
		var aggression int

		if parent == nil {
			// Set base values for ancestor creatures
			color = generateColor(n, 100)
			maxAge = rand.Intn(200)
			maxHealth = rand.Intn(200) + 300
			health = int(rand.Intn(maxHealth / 5))
			metabolism = rand.Intn(25) + 1
			maxNutrition = rand.Intn(500) + 200
			nutrition = int(rand.Intn(maxNutrition / 4))
			aggression = rand.Intn(100)
		} else {
			color = parent.ancestor.colorCode
			maxAge = int(float64(parent.maxAge) * (1 + (rand.Float64()*0.2 - 0.1)))             // set maxAge to parent's max age + or - 10%
			maxHealth = int(float64(parent.maxHealth) * (1 + (rand.Float64()*0.2 - 0.1)))       // set maxHealth to parent's max health + or - 10%
			metabolism = int(float64(parent.metabolism) * (1 + (rand.Float64()*0.2 - 0.1)))     // set metabolism to parent's metabolism + or - 10%
			maxNutrition = int(float64(parent.maxNutrition) * (1 + (rand.Float64()*0.2 - 0.1))) // set maxNutrition to parent's maxNutrition + or - 10%
			aggression = parent.aggression + (rand.Intn(3) - 1)                                 // add or remove 1
		}

		c := &creature{
			num:          n,
			colorCode:    color,
			char:         nextSafeRune(),
			parent:       parent,
			health:       health,
			maxHealth:    maxHealth,
			X:            0,
			Y:            0,
			age:          0,
			maxAge:       maxAge,
			metabolism:   metabolism,
			nutrition:    nutrition,
			maxNutrition: maxNutrition,
			aggression:   aggression,
		}

		// set root creature's parent and ancestor to be self
		if parent == nil {
			c.ancestor = c
		} else {
			c.ancestor = parent.ancestor
		}

		n++
		return c
	}
}

// Helper function to flatten the creature list
func flattenCreatureList(env [][]environment) []*creature {
	var all []*creature
	for i := range env {
		for j := range env[i] {
			all = append(all, env[i][j].creatureList...)
		}
	}
	return all
}

// Print creatures and map
func printMap(env [][]environment) {
	var buf bytes.Buffer
	buf.WriteString("\033[H\033[2J") // clear screen

	for i := range env {
		for j := range env[i] {
			cell := env[i][j]
			bg := foodToColor(cell.food)
			creatures := cell.creatureList
			count := len(creatures)

			switch {
			case count == 0:
				buf.WriteString(bg + "   " + "\033[0m")
			case count == 1:
				c := creatures[0]
				fmt.Fprintf(&buf, "%s %s%c \033[0m", bg, c.colorCode, c.char)
			case count <= 3:
				buf.WriteString(bg)
				for k := range count {
					c := creatures[k]
					fmt.Fprintf(&buf, "%s%c", c.colorCode, c.char)
				}
				// pad to 3 characters
				for k := count; k < 3; k++ {
					buf.WriteByte(' ')
				}
				buf.WriteString("\033[0m")
			default:
				first := creatures[0]
				remaining := count - 1
				fmt.Fprintf(&buf, "%s%s%c+%d\033[0m", bg, first.colorCode, first.char, remaining)
			}
		}
		buf.WriteString("\n")
	}
	fmt.Print(buf.String())
}

func printTickStats(tickStats *TickStats) {
	fmt.Printf("Tick: %d | Creatures: %d | TotalDeaths: %d | AgeDeaths: %d | StarvationDeaths: %d | Clones: %d\nFoodAdded: %d | FoodConsumed: %d | Eats: %d | Moves: %d | Fights: %d\n",
		tickStats.Tick, tickStats.TotalCreatures, tickStats.TotalDeaths, tickStats.AgeDeaths, tickStats.StarvationDeaths, tickStats.Clones, tickStats.FoodAdded, tickStats.FoodConsumed, tickStats.Eats, tickStats.Moves, tickStats.Fights)
	fmt.Printf("Avg Age: %.1f | Avg Health: %.1f | Avg Nutrition: %.1f\n",
		float64(tickStats.TotalAge)/float64(tickStats.TotalCreatures), float64(tickStats.TotalHealth)/float64(tickStats.TotalCreatures), float64(tickStats.TotalNutrition)/float64(tickStats.TotalCreatures))
}

func printAncestryStats(ancestryStats *[]AncestryStats) {
	fmt.Println("*** Ancestor Leaderboard ***")
	for i, a := range *ancestryStats {
		fmt.Printf("%s%d. Ancestor: %c\t| LineageCount: %d\t| MaxHealth: %d\t| MaxAge: %d\t| MaxNutrition: %d\t| Metabolism: %d\n", a.Ancestor.colorCode, i+1, a.Ancestor.char, a.LineageCount, a.Ancestor.maxHealth, a.Ancestor.maxAge, a.Ancestor.maxNutrition, a.Ancestor.metabolism)
	}
}

// Main tick loop
func simulateTick(env [][]environment, createCreature func(*creature) *creature, tickStats *TickStats) {
	creatures := flattenCreatureList(env)

	// Environment simulation updates
	processEnvironment(env, tickStats)

	// Creature simulation updates
	for _, c := range creatures {
		c.age++ // age creatures
		tickStats.TotalAge += c.age
		processNutrition(c, tickStats)
		processHealth(c, tickStats)
	}

	intents := collectIntents(creatures, env)
	resolveActions(intents, env, createCreature, tickStats)
}

// Determine which action a creature will take
func (c *creature) decideAction(env [][]environment) ActionIntent {
	location := env[c.Y][c.X]
	// Fight if sharing location with non-relative and aggressive
	// implementation may need to remove other creatures action
	//if location.creatureList lenth is > 1
	//if rand.Intn(101) < c.aggression
	//and c.parent != other creatures c.parent
	//...if ancestor is the same drop aggression by 50%

	// Clone if between 20% & 90% maxAge && >= 90% of maxHealth
	if float64(c.age) >= 0.2*float64(c.maxAge) &&
		float64(c.age) < 0.9*float64(c.maxAge) &&
		float64(c.health) >= 0.75*float64(c.maxHealth) {
		return ActionIntent{
			Creature: c,
			Type:     "clone",
		}
	}

	// If nutrition is low, and location food > 0 eat
	if (float64(c.nutrition)/float64(c.maxNutrition)) < 0.95 && location.food > 1 {
		return ActionIntent{
			Creature: c,
			Type:     "eat",
		}
	}

	// Choose random direction: N, S, E, W
	dx, dy := 0, 0
	switch rand.Intn(4) {
	case 0:
		dx = 1 // east
	case 1:
		dx = -1 // west
	case 2:
		dy = 1 // south
	case 3:
		dy = -1 // north
	}

	return ActionIntent{
		Creature: c,
		Type:     "move",
		TargetX:  c.X + dx,
		TargetY:  c.Y + dy,
	}
}

// Iterate over each creature and collect their action intentions
func collectIntents(creatures []*creature, env [][]environment) []ActionIntent {
	intentCh := make(chan ActionIntent, len(creatures))
	var wg sync.WaitGroup

	for _, c := range creatures {
		wg.Add(1)
		go func(cr *creature) {
			defer wg.Done()
			intent := cr.decideAction(env)
			intentCh <- intent
		}(c)
	}

	wg.Wait()
	close(intentCh)

	var intents []ActionIntent
	for intent := range intentCh {
		intents = append(intents, intent)
	}
	return intents
}

// Iterate over creature intentions
//
//	Group intentions by type
//	Resovle intentions in desired order
func resolveActions(intents []ActionIntent, env [][]environment, createCreature func(*creature) *creature, tickStats *TickStats) {
	// Phase 1: Group intents by type
	var moveIntents, cloneIntents, eatIntents, fightIntents []ActionIntent

	for _, intent := range intents {
		switch intent.Type {
		case "fight":
			tickStats.Fights++
			fightIntents = append(fightIntents, intent)
		case "move":
			tickStats.Moves++
			moveIntents = append(moveIntents, intent)
		case "clone":
			tickStats.Clones++
			cloneIntents = append(cloneIntents, intent)
		case "eat":
			tickStats.Eats++
			eatIntents = append(eatIntents, intent)
		}
	}
	// Phase 2: Resolve in safe order
	for _, intent := range fightIntents {
		resolveFight(intent, env)
	}
	for _, intent := range eatIntents {
		resolveEat(intent, env, tickStats)
	}
	for _, intent := range cloneIntents {
		resolveClone(intent, env, createCreature)
	}
	for _, intent := range moveIntents {
		resolveMove(intent, env)
	}
}

func resolveFight(intent ActionIntent, env [][]environment) {

}

// Resolve "Move" creature action
func resolveMove(intent ActionIntent, env [][]environment) {
	c := intent.Creature
	oldX, oldY := c.X, c.Y
	newX, newY := intent.TargetX, intent.TargetY

	// Bounds check
	if newY < 0 || newY >= len(env) || newX < 0 || newX >= len(env[0]) {
		return // invalid move, ignore
	}

	// Remove from old cell
	oldCell := &env[oldY][oldX]
	for i, creature := range oldCell.creatureList {
		if creature == c {
			oldCell.creatureList = append(oldCell.creatureList[:i], oldCell.creatureList[i+1:]...)
			break
		}
	}

	// Add to new cell
	env[newY][newX].creatureList = append(env[newY][newX].creatureList, c)
	c.X = newX
	c.Y = newY
}

// Resolve "Clone" creature action
func resolveClone(intent ActionIntent, env [][]environment, createCreature func(*creature) *creature) {
	parent := intent.Creature
	x, y := parent.X, parent.Y

	// Create clone
	clone := createCreature(parent)
	clone.X = x
	clone.Y = y
	clone.health = int(float64(clone.maxHealth) * 0.1)
	clone.nutrition = int(float64(clone.maxNutrition) * 0.1)

	// Add clone to map
	env[y][x].creatureList = append(env[y][x].creatureList, clone)

	// Reduce parent's health by 60%
	parent.health = int(float64(parent.health) * 0.3)
}

// Resolve "Eat" creature action
func resolveEat(intent ActionIntent, env [][]environment, tickStats *TickStats) {
	creature := intent.Creature
	location := &env[creature.Y][creature.X]

	toConsume := min(5, location.food)
	creature.nutrition += 10 * toConsume

	tickStats.FoodConsumed += toConsume
	location.food -= toConsume
}

// Process environmental updates including removal of dead creatures
func processEnvironment(env [][]environment, tickStats *TickStats) {
	totalCreatures := 0
	totalDeaths := 0
	ageDeaths := 0
	starvationDeaths := 0

	for i := range env {
		for j := range env[i] {
			cell := &env[i][j]
			// Grow food
			if cell.food < cell.maxFood && rand.Intn(20) == 0 {
				tickStats.FoodAdded++
				cell.food++
			}

			// Manage dead creatures
			newList := cell.creatureList[:0] // reuse underlying array
			for _, c := range cell.creatureList {
				switch {
				case c.age >= c.maxAge:
					ageDeaths++
					totalDeaths++
				case c.health <= 0:
					starvationDeaths++
					totalDeaths++
				default:
					newList = append(newList, c)
					totalCreatures++
				}

				cell.creatureList = newList
			}
		}
	}
	tickStats.TotalCreatures = totalCreatures
	tickStats.TotalDeaths = totalDeaths
	tickStats.AgeDeaths = ageDeaths
	tickStats.StarvationDeaths = starvationDeaths
}

// Process creature health
func processHealth(c *creature, tickStats *TickStats) {
	tickStats.TotalHealth += c.health
	if c.health >= c.maxHealth {
		return
	}

	pctNutrition := (float64(c.nutrition) / float64(c.maxNutrition)) * 100
	switch {
	case pctNutrition <= 25:
		tickStats.HealthAdded--
		c.health--
	case pctNutrition > 50 && pctNutrition < 75:
		tickStats.HealthAdded += c.metabolism
		c.health = c.health + c.metabolism
	case pctNutrition >= 75:
		tickStats.HealthAdded += (c.metabolism * 2)
		c.health = c.health + (c.metabolism * 2)
	}

	// Ensure health doesn't exceed maxHealth
	c.health = min(c.maxHealth, c.health)
}

// Process creature nutrition
func processNutrition(c *creature, tickStats *TickStats) {
	c.nutrition -= c.metabolism
	tickStats.TotalNutrition += c.nutrition
}

// Collect creature ancestry stats
func collectAncestryStats(env [][]environment, ancestryStats *[]AncestryStats) {
	creatures := flattenCreatureList(env)
	var am = make(map[*creature]int)
	for _, c := range creatures {
		am[c.ancestor]++
	}
	mapToSortedAncestryStats(am, ancestryStats)
}

// Helper function to sort a map of ancestors
func mapToSortedAncestryStats(am map[*creature]int, ancestryStats *[]AncestryStats) {
	stats := make([]AncestryStats, 0, len(am))
	for a, count := range am {
		stats = append(stats, AncestryStats{Ancestor: a, LineageCount: count})
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].LineageCount > stats[j].LineageCount
	})
	*ancestryStats = stats[:min(5, len(stats))]
}

// Helper function to create a color to be used by a creature
func generateColor(index, total int) string {
	if total <= 0 {
		total = 1
	}
	hue := float64(index%total) / float64(total)
	r, g, b := hsvToRGB(hue, 1.0, 1.0)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// Helper function to convert a color from HSV (Hue, Saturation, Value) to RGB (Red, Green, Blue)
func hsvToRGB(h, s, v float64) (int, int, int) {
	h = h * 6
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h, 2)-1))
	m := v - c

	var r, g, b float64
	switch {
	case 0 <= h && h < 1:
		r, g, b = c, x, 0
	case 1 <= h && h < 2:
		r, g, b = x, c, 0
	case 2 <= h && h < 3:
		r, g, b = 0, c, x
	case 3 <= h && h < 4:
		r, g, b = 0, x, c
	case 4 <= h && h < 5:
		r, g, b = x, 0, c
	case 5 <= h && h < 6:
		r, g, b = c, 0, x
	default:
		r, g, b = 0, 0, 0
	}

	return int((r + m) * 255), int((g + m) * 255), int((b + m) * 255)
}

// Count total number of creatures in the environment
//func countCreatures(env [][]environment) int {
//	count := 0
//	for i := range env {
//		for j := range env[i] {
//			count += len(env[i][j].creatureList)
//		}
//	}
//	return count
//}

// Helper function to determine background color
//
//	Location background color determined by food amount
func foodToColor(food int) string {
	switch {
	case food > 100:
		return "\033[48;5;76m" // bright green background
	case food > 90:
		return "\033[48;5;77m" // green
	case food > 80:
		return "\033[48;5;34m" // olive
	case food > 70:
		return "\033[48;5;35m" // olive
	case food > 60:
		return "\033[48;5;28m" // olive
	case food > 50:
		return "\033[48;5;29m" // olive
	case food > 40:
		return "\033[48;5;70m" // olive
	case food > 30:
		return "\033[48;5;64m" // olive
	case food > 20:
		return "\033[48;5;142m" // olive
	case food > 10:
		return "\033[48;5;100m" // olive
	case food > 5:
		return "\033[48;5;58m" // olive
	case food > 1:
		return "\033[48;5;136m" // brown
	default:
		return "\033[048;5;94m" // default background
	}
}

// Helper function to return min of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper function to get the next safe (single width) rune
func nextSafeRune() rune {
	if nextRuneIndex >= len(safeRunes) {
		// fallback if you run out (wrap around or panic/log)
		nextRuneIndex = 0
	}
	r := safeRunes[nextRuneIndex]
	nextRuneIndex++
	return r
}

// Helper function to build a list of single width runes
func buildSingleWidthRunePool(start, end rune) []rune {
	var runes []rune
	for r := start; r <= end; r++ {
		if unicode.IsPrint(r) && isSingleColumnRune(r) {
			runes = append(runes, r)
		}
	}
	return runes
}

// Helper function to determine if a rune is single with rune
func isSingleColumnRune(r rune) bool {
	p := width.LookupRune(r)
	k := p.Kind()
	return k == width.Neutral || k == width.EastAsianNarrow || k == width.EastAsianAmbiguous
}
