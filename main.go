package main

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/term"
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

// Globals (limited; see AGENTS.md guidance)
var safeRunes []rune
var nextRuneIndex int
var isTest bool = false
var paused int32
var exitCh = make(chan struct{})
var exitOnce sync.Once

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

	tickCh := startTicker(300 * time.Millisecond)

	for {
		select {
		case <-exitCh:
			fmt.Print("\r\nExited simulation.")
			return
		case <-tickCh:
			if atomic.LoadInt32(&paused) == 1 {
				continue
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
	ticker := time.NewTicker(tickInterval)
	ch := make(chan time.Time)
	go func() {
		for t := range ticker.C {
			ch <- t
		}
	}()
	return ch
}

func initializeMap(rows, cols int, createCreature func(p *Creature) *Creature) [][]environment {
	matrix := make([][]environment, rows)
	for i := range matrix {
		matrix[i] = make([]environment, cols)
		for j := range matrix[i] {
			maxFood := RandIntn(120)
			matrix[i][j].maxFood = maxFood
			matrix[i][j].food = int(float64(maxFood) * (float64(RandIntn(10)) / 20))
			if RandIntn(100) != 0 {
				continue
			}
			c := createCreature(nil)
			c.X, c.Y = j, i
			matrix[i][j].creatureList = append(matrix[i][j].creatureList, c)
		}
	}
	return matrix
}

func simulateTick(env [][]environment, creatureFac func(*Creature) *Creature, tickStats *TickStats) {
	processEnvironment(env, tickStats)
	creatures := flattenCreatureList(env)
	for _, c := range creatures {
		c.age++
		tickStats.TotalAge += c.age
		processNutrition(c, tickStats)
		processHealth(c, tickStats)
	}
	intents := collectIntents(creatures, env)
	resolveActions(intents, env, creatureFac, tickStats)
	tickStats.TotalCreatures = len(flattenCreatureList(env))
}

func printMap(env [][]environment) {
	var buf bytes.Buffer
	buf.WriteString("\033[H\033[2J")
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

func processEnvironment(env [][]environment, tickStats *TickStats) {
	for i := range env {
		for j := range env[i] {
			cell := &env[i][j]
			if cell.food < cell.maxFood && RandIntn(40) == 0 {
				tickStats.FoodAdded++
				cell.food++
			}
			newList := cell.creatureList[:0]
			for _, c := range cell.creatureList {
				switch {
				case c.age >= c.maxAge:
					tickStats.AgeDeaths++
				case c.health <= 0:
					// dead: skip
				default:
					newList = append(newList, c)
				}
				cell.creatureList = newList
			}
		}
	}
}
