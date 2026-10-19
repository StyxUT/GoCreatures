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

// plantBiomassToColor preserves a simple density cue until per-species
// rendering is layered onto the same cell state.
func plantBiomassToColor(biomass, capacity float64) string {
	if capacity <= 0 || biomass <= 0 {
		return "\033[48;5;94m"
	}
	density := biomass / capacity
	switch {
	case density >= 0.7:
		return "\033[48;5;34m"
	case density >= 0.4:
		return "\033[48;5;28m"
	case density >= 0.1:
		return "\033[48;5;64m"
	default:
		return "\033[48;5;100m"
	}
}

// plantTexture assigns a fixed three-column position and species-specific
// foreground/background cues. It depends only on the cell snapshot, not RNG.
func plantTexture(species PlantSpecies, biomass, capacity float64) (string, rune) {
	if biomass <= 0 || capacity <= 0 {
		return "\033[48;5;94m", '.'
	}
	density := biomass / capacity
	shade := 0
	switch {
	case density >= 0.7:
		shade = 3
	case density >= 0.4:
		shade = 2
	case density >= 0.1:
		shade = 1
	}
	backgrounds := [plantSpeciesCount][4]int{
		{100, 64, 28, 34}, // olive grass
		{108, 30, 24, 23}, // teal clover
		{97, 96, 54, 53},  // muted purple bramble
	}
	textures := [plantSpeciesCount][4]rune{
		{'.', '|', 'I', 'H'},
		{'.', '*', 'o', '@'},
		{':', 'x', 'X', '#'},
	}
	if species >= plantSpeciesCount {
		return "\033[48;5;94m", '.'
	}
	return fmt.Sprintf("\033[48;5;%dm", backgrounds[species][shade]), textures[species][shade]
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

// SortEnvironmentListByNutritionalValue ranks cells using the same vegetation
// source and yield calculation as herbivore eating.
func SortEnvironmentListByNutritionalValue(ol []ObservedLocation) {
	sort.SliceStable(ol, func(i, j int) bool {
		return plantNutritionalValue(ol[i].Env) > plantNutritionalValue(ol[j].Env)
	})
}

func plantNutritionalValue(cell *environment) float64 {
	if cell == nil {
		return 0
	}
	value := 0.0
	for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
		value += maxFloat(cell.plants[species].biomass, 0) * plantNutritionalYield(cell, species)
	}
	return value
}

func plantBiomassAvailable(cell *environment) float64 {
	if cell == nil {
		return 0
	}
	biomass := 0.0
	for _, population := range cell.plants {
		biomass += maxFloat(population.biomass, 0)
	}
	return biomass
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
