package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync/atomic"
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

type environment struct {
	creatureList []*Creature
	food         int
	maxFood      int
}

// TODO //
// Use creature vision
//
//	make more intelligent movement like avoid other creatures
//
// Add reporting into Postgress
var safeRunes []rune
var nextRuneIndex int
var isTest bool = false
var paused int32
var exitCh = make(chan struct{})

func main() {
	// Start key input goroutine (pause on space, exit on 'e')
	startKeyListener()

	rows, cols := 38, 81
	creatureFac := CreatureFactory()
	env := initializeMap(rows, cols, creatureFac)
	var tickCount int
	var ancestryStats []AncestryStats

	tickCh := startTicker(300 * time.Millisecond)

	for {
		select {
		case <-exitCh:
			fmt.Print("\r\nExited simulation.")
			return

		case <-tickCh:
			if atomic.LoadInt32(&paused) == 1 {
				continue // skip this tick
			}

			tickCount++
			var tickStats TickStats

			tickStats.Tick = tickCount

			simulateTick(env, creatureFac, &tickStats)
			collectAncestryStats(env, &ancestryStats)
			printMap(env)
			printTickStats(&tickStats)

			fmt.Print("\r\n*** Ancestor Leaderboard ***\r\n")
			printAncestryStats(&ancestryStats)
			fmt.Print("\r\nPress <space> to pause or 'e' to exit")

			if len(ancestryStats) <= 1 {
				PrintNewestClones(env)
				close(exitCh)
				return
			}
			if tickStats.TotalCreatures == 0 {
				fmt.Println("All creatures are gone. Simulation ended.")
				close(exitCh)
				return
			}
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
			maxFood := rand.Intn(120)
			matrix[i][j].maxFood = maxFood                                            // set max food
			matrix[i][j].food = int(float64(maxFood) * (float64(rand.Intn(10)) / 20)) // set current food a perentage of maxFood

			// Populate with creatures
			if rand.Intn(100) != 0 { // populate a subset of the map locations
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
func simulateTick(env [][]environment, creatureFac func(*Creature) *Creature, tickStats *TickStats) {

	// Environment simulation updates
	processEnvironment(env, tickStats)

	creatures := flattenCreatureList(env)
	
	// Creature simulation updates
	for _, c := range creatures {
		c.age++ // age creatures
		tickStats.TotalAge += c.age
		processNutrition(c, tickStats)
		processHealth(c, tickStats)
	}

	intents := collectIntents(creatures, env)
	resolveActions(intents, env, creatureFac, tickStats)
	tickStats.TotalCreatures = len(flattenCreatureList(env))
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
		buf.WriteString("\n\r")
	}
	fmt.Print(buf.String())
}

// Process environmental updates including removal of dead creatures
func processEnvironment(env [][]environment, tickStats *TickStats) {
	for i := range env {
		for j := range env[i] {
			cell := &env[i][j]
			// Grow food
			if cell.food < cell.maxFood && rand.Intn(40) == 0 {
				tickStats.FoodAdded++
				cell.food++
			}

			// Manage dead creatures
			newList := cell.creatureList[:0] // reuse underlying array
			for _, c := range cell.creatureList {
				switch {
				case c.age >= c.maxAge:
					tickStats.AgeDeaths++
				case c.health <= 0:
					// do nothing
				default:
					newList = append(newList, c)
				}
				cell.creatureList = newList
			}
		}
	}
}
