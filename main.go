package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"time"
)

type TickStats struct {
	Tick             int
	TotalCreatures   int
	TotalDeaths      int
	AgeDeaths        int
	StarvationDeaths int
	FightDeaths      int
	TotalAge         int
	TotalHealth      int
	TotalNutrition   int
	Clones           int
	Eats             int
	Moves            int
	Fights           int
	Sleeps           int
	FoodAdded        int
	HealthAdded      int
	FoodConsumed     int
}

// TODO //
// Use creature vision
// make more intelligent movement like avoid other creatures, or move toward food
var safeRunes []rune
var nextRuneIndex int

func main() {
	rows, cols := 38, 81
	creatureFac := CreatureFactory()
	env := initializeMap(rows, cols, creatureFac)
	var tickCount int

	tickCh := startTicker(300 * time.Millisecond)

	for range tickCh {
		tickCount++
		var tickStats TickStats
		var ancestryStats []AncestryStats

		tickStats.Tick = tickCount

		simulateTick(env, creatureFac, &tickStats)
		collectAncestryStats(env, &ancestryStats)
		printMap(env)
		printTickStats(&tickStats)

		fmt.Println("\n*** Ancestor Leaderboard ***")
		printAncestryStats(&ancestryStats)

		if len(ancestryStats) == 1 {
			printNewestClones(env)
			break
		}

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
func initializeMap(rows int, cols int, createCreature func(p *Creature) *Creature) [][]environment {
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

// Main tick loop
func simulateTick(env [][]environment, createCreature func(*Creature) *Creature, tickStats *TickStats) {
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
