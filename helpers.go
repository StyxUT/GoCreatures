package main

import (
	"fmt"
	"math"
	"sort"
)

// Helper function to return min of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper function to return max of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Helper function to view N locations around a starting location and callback
func SpiralLook2D_N(env [][]environment, c *Creature) {
	height := len(env)
	width := len(env[0])

	c.observed = c.observed[:0] // Clear previous observations while reusing slice memory

	// Direction vectors: right, down, left, up
	dx := []int{1, 0, -1, 0}
	dy := []int{0, 1, 0, -1}

	// Setup
	n := int(c.vision) // max number of cells to observe
	x, y := c.X, c.Y   // start at creature's current position
	viewed := 0        // counter for observed locations
	stepSize := 1      // number of steps per leg of the spiral
	dir := 0           // direction index (0:right, 1:down, 2:left, 3:up)

	// Begin spiral outward until N cells are observed or map bounds are exceeded
	for radius := 1; viewed < n && radius < max(width, height); radius++ {
		for range 2 { // each ring has two legs in a direction
			for range stepSize {
				x += dx[dir]
				y += dy[dir]

				// If within bounds, record the location
				if x >= 0 && x < width && y >= 0 && y < height {
					c.observed = append(c.observed, ObservedLocation{Env: &env[y][x], X: x, Y: y})
					viewed++
					if viewed == n {
						return
					}
				}
			}
			dir = (dir + 1) % 4 // turn 90 degrees clockwise
		}
		stepSize++ // increase step size for the next ring of the spiral
	}
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
func generateColor(index, _ int) string {
	// Golden-ratio hue offset for better distinction
	hue := math.Mod(0.61803398875*float64(index), 1.0)

	r, g, b := hsvToRGB(hue, 0.85, 0.95)
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

// Helper function to determine background color
//
//	Location background color determined by food amount
func foodToColor(food int) string {
	switch {
	case food > 100:
		return "\033[48;5;76m"
	case food > 90:
		return "\033[48;5;77m"
	case food > 80:
		return "\033[48;5;34m"
	case food > 70:
		return "\033[48;5;35m"
	case food > 60:
		return "\033[48;5;28m"
	case food > 50:
		return "\033[48;5;29m"
	case food > 40:
		return "\033[48;5;70m"
	case food > 30:
		return "\033[48;5;64m"
	case food > 20:
		return "\033[48;5;142m"
	case food > 10:
		return "\033[48;5;100m"
	case food > 5:
		return "\033[48;5;58m"
	case food > 1:
		return "\033[48;5;136m"
	default:
		return "\033[048;5;94m"
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

// Helper function to sort a list of environments by food
func SortEnvironmentListByFood(ol []ObservedLocation) {
	sort.Slice(ol, func(i, j int) bool {
		return ol[i].Env.food > ol[j].Env.food
	})
}

// Helper function to sort a list of environments by occupants desc
// SortEnvironmentListByOccupantsAsc sorts observed locations by ascending occupant count (fewest creatures first).
func SortEnvironmentListByOccupantsAsc(ol []ObservedLocation) {
	sort.Slice(ol, func(i, j int) bool {
		return len(ol[i].Env.creatureList) < len(ol[j].Env.creatureList)
	})
}

// Helper function to clamp a value at 1, 0, or -1
func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	default:
		return 0
	}
}

// chooseByModulo spreads selection among tied observed locations deterministically.
// It expects a slice of ObservedLocation holding only tied candidates.
func chooseByModulo(tied []ObservedLocation, creatureNum int) ObservedLocation {
	if len(tied) == 0 {
		return ObservedLocation{}
	}
	if len(tied) == 1 {
		return tied[0]
	}
	idx := creatureNum % len(tied)
	return tied[idx]
}
