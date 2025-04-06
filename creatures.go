package main

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
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

// TODO //
// Add combat
// less likely to fight when; plenty of food, same a parent, same ancestor
// damage given based on what?
// damage taken based on what?

// set background color based on amount of food present

func main() {
	rows, cols := 40, 70
	creatureFac := creatureFactory()
	env := initializeMap(rows, cols, creatureFac)
	var tickCount int

	tickCh := startTicker(500 * time.Millisecond)

	for range tickCh {
		tickCount++
		simulateTick(env, creatureFac)
		creatureCount := countCreatures(env)
		printMap(env, tickCount, creatureCount)

		if creatureCount == 0 {
			fmt.Println("All creatures are gone. Simulation ended.")
			break
		}
	}
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

func creatureFactory() func(parent *creature) *creature {
	n := 1
	r := 257 // start after control characters

	return func(parent *creature) *creature {
		var color string
		var maxAge int
		var maxHealth int
		var health int
		var metabolism int
		var nutrition int
		var maxNutrition int

		if parent == nil {
			// Set base values for ancestor creatures
			color = generateColor(n, 100)
			maxAge = rand.Intn(200)
			maxHealth = rand.Intn(200) + 300
			health = int(rand.Intn(maxHealth / 5))
			metabolism = rand.Intn(25) + 1
			maxNutrition = rand.Intn(500) + 200
			nutrition = int(rand.Intn(maxNutrition / 4))
		} else {
			color = parent.ancestor.colorCode
			maxAge = int(float64(parent.maxAge) * (1 + (rand.Float64()*0.2 - 0.1)))             // set maxAge to parent's max age + or - 10%
			maxHealth = int(float64(parent.maxHealth) * (1 + (rand.Float64()*0.2 - 0.1)))       // set maxHealth to parent's max health + or - 10%
			metabolism = int(float64(parent.metabolism) * (1 + (rand.Float64()*0.2 - 0.1)))     // set metabolism to parent's metabolism + or - 10%
			maxNutrition = int(float64(parent.maxNutrition) * (1 + (rand.Float64()*0.2 - 0.1))) // set maxNutrition to parent's maxNutrition + or - 10%
		}

		c := &creature{
			num:          n,
			char:         rune(r),
			colorCode:    color,
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
		}

		// set root creature's parent and ancestor to be self
		if parent == nil {
			c.ancestor = c
		} else {
			c.ancestor = parent.ancestor
		}

		n++
		r++
		return c
	}
}

func flattenCreatureList(env [][]environment) []*creature {
	var all []*creature
	for i := range env {
		for j := range env[i] {
			all = append(all, env[i][j].creatureList...)
		}
	}
	return all
}

func printMap(env [][]environment, tickCount int, creatureCount int) {
	var buf bytes.Buffer

	buf.WriteString("\033[H\033[2J") // clear screen

	for i := range env {
		for j := range env[i] {
			cell := env[i][j]
			creatures := cell.creatureList
			if len(creatures) == 0 {
				bg := foodToColor(cell.food)
				buf.WriteString(bg + "   " + "\033[0m")
			} else if len(creatures) == 1 {
				fmt.Fprintf(&buf, " %s%c\033[0m ", creatures[0].colorCode, creatures[0].char)
			} else {
				first := creatures[0].char
				fmt.Fprintf(&buf, "%c+%d", first, len(creatures)-1)
			}
		}
		buf.WriteString("\n")
	}
	fmt.Fprintf(&buf, "CreatureCount: %v; TickCount: %v\n", creatureCount, tickCount)

	// Now flush it all at once
	fmt.Print(buf.String())
}

func simulateTick(env [][]environment, createCreature func(*creature) *creature) {
	creatures := flattenCreatureList(env)

	processEnvironment(env)

	// background simulation actions
	for _, c := range creatures {
		c.age++ // age creatures
		processNutrition(c)
		processHealth(c)
	}

	intents := collectIntents(creatures, env)
	resolveActions(intents, env, createCreature)
}

func (c *creature) decideAction(env [][]environment) ActionIntent {
	location := env[c.Y][c.X]

	// Clone if between 20% & 90% maxAge && >= 90% of maxHealth
	if float64(c.age) >= 0.2*float64(c.maxAge) &&
		float64(c.age) < 0.9*float64(c.maxAge) &&
		float64(c.health) >= 0.75*float64(c.maxHealth) {
		return ActionIntent{
			Creature: c,
			Type:     "clone",
		}
	}

	//If nutrition is low, and location food > 0 eat
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

//func collectIntents(creatures []*creature, env [][]environment) []ActionIntent {
//	var intents []ActionIntent
//	for _, c := range creatures {
//		intents = append(intents, c.decideAction(env))
//	}
//	return intents
//}

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

func resolveActions(intents []ActionIntent, env [][]environment, createCreature func(*creature) *creature) {
	// Phase 1: Group intents by type
	var moveIntents, cloneIntents, eatIntents []ActionIntent

	for _, intent := range intents {
		switch intent.Type {
		case "move":
			moveIntents = append(moveIntents, intent)
		case "clone":
			cloneIntents = append(cloneIntents, intent)
		case "eat":
			eatIntents = append(eatIntents, intent)
		}
	}

	// Phase 2: Resolve in safe order
	// Optional: shuffle moveIntents to avoid move-order bias
	for _, intent := range eatIntents {
		resolveEat(intent, env)
	}
	for _, intent := range cloneIntents {
		resolveClone(intent, env, createCreature)
	}
	for _, intent := range moveIntents {
		resolveMove(intent, env)
	}
}

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

func resolveEat(intent ActionIntent, env [][]environment) {
	creature := intent.Creature
	location := &env[creature.Y][creature.X]

	toConsume := min(5, location.food)
	creature.nutrition += 10 * toConsume

	location.food -= toConsume
}

func processEnvironment(env [][]environment) {
	for i := range env {
		for j := range env[i] {
			cell := &env[i][j]
			// Grow food
			if cell.food < cell.maxFood && rand.Intn(10) == 0 {
				cell.food++
			}

			// Remove dead creatures
			newList := cell.creatureList[:0] // reuse underlying array
			for _, c := range cell.creatureList {
				if c.age < c.maxAge && c.health > 0 {
					newList = append(newList, c)
				}
				if countCreatures(env) == 0 {
					fmt.Println("All creatures are gone. Simulation ended.")
					break
				}
			}
			cell.creatureList = newList
		}
	}
}

func processHealth(c *creature) {
	if c.health >= c.maxHealth {
		return
	}

	pctNutrition := (float64(c.nutrition) / float64(c.maxNutrition)) * 100
	switch {
	case pctNutrition <= 25:
		c.health--
	case pctNutrition > 50 && pctNutrition < 75:
		c.health = c.health + c.metabolism
	case pctNutrition >= 75:
		c.health = c.health + (c.metabolism * 2)
	}

	// ensure health doesn't exceed maHealth
	c.health = min(c.maxHealth, c.health)
}

func processNutrition(c *creature) {
	c.nutrition -= c.metabolism
}

func generateColor(index, total int) string {
	if total <= 0 {
		total = 1
	}
	hue := float64(index%total) / float64(total)
	r, g, b := hsvToRGB(hue, 1.0, 1.0)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

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

func countCreatures(env [][]environment) int {
	count := 0
	for i := range env {
		for j := range env[i] {
			count += len(env[i][j].creatureList)
		}
	}
	return count
}

func foodToColor(food int) string {
	switch {
	case food > 75:
		return "\033[48;5;34m" // bright green background
	case food > 50:
		return "\033[48;5;28m" // green
	case food > 25:
		return "\033[48;5;94m" // olive
	case food > 0:
		return "\033[48;5;130m" // brown
	default:
		return "\033[0m" // default background
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
