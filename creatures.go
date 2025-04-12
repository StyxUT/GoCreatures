package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
)

type Creature struct {
	num          int
	parent       *Creature
	ancestor     *Creature
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
	creatureList []*Creature
	food         int
	maxFood      int
}

type ActionIntent struct {
	Creature       *Creature
	Type           string // e.g., "move", "rest", etc.
	TargetX        int
	TargetY        int
	TargetCreature *Creature
}

type AncestryStats struct {
	Ancestor     *Creature
	LineageCount int
}

// Generate a creature and set it's characteristics
func CreatureFactory() func(parent *Creature) *Creature {
	n := 1

	return func(parent *Creature) *Creature {
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
			maxAge = int(float64(parent.maxAge) * (1 + (rand.Float64()*0.02 - 0.01)))             // set maxAge to parent's max age + or - 1%
			maxHealth = int(float64(parent.maxHealth) * (1 + (rand.Float64()*0.02 - 0.01)))       // set maxHealth to parent's max health + or - 1%
			metabolism = int(float64(parent.metabolism) * (1 + (rand.Float64()*0.02 - 0.01)))     // set metabolism to parent's metabolism + or - 1%
			maxNutrition = int(float64(parent.maxNutrition) * (1 + (rand.Float64()*0.02 - 0.01))) // set maxNutrition to parent's maxNutrition + or - 1%
			aggression = parent.aggression + (rand.Intn(3) - 1)                                   // add or remove 1
		}

		c := &Creature{
			num:          n,
			colorCode:    color,
			char:         NextSafeRune(),
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
func flattenCreatureList(env [][]environment) []*Creature {
	var all []*Creature
	for i := range env {
		for j := range env[i] {
			all = append(all, env[i][j].creatureList...)
		}
	}
	return all
}

// Iterate over each creature and collect their action intentions
func collectIntents(creatures []*Creature, env [][]environment) []ActionIntent {
	intentCh := make(chan ActionIntent, len(creatures))
	var wg sync.WaitGroup

	for _, c := range creatures {
		wg.Add(1)
		go func(cr *Creature) {
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
			if cell.food < cell.maxFood && rand.Intn(40) == 0 {
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
func processHealth(c *Creature, tickStats *TickStats) {
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
func processNutrition(c *Creature, tickStats *TickStats) {
	c.nutrition -= c.metabolism
	tickStats.TotalNutrition += c.nutrition
}

// Helper function to sort the creature list by creature.num
func SortCreatureList(env [][]environment) []*Creature {
	creatures := flattenCreatureList(env)

	sort.Slice(creatures, func(i, j int) bool {
		return creatures[i].num > creatures[j].num
	})
	return creatures
}

// Helper function to sort a map of ancestors
func mapToSortedAncestryStats(am map[*Creature]int, ancestryStats *[]AncestryStats) {
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
