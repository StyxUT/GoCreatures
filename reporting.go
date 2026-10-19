package main

import (
	"fmt"
	"strings"
)

// Collect creature ancestry stats
func collectAncestryStats(env [][]environment, ancestryStats *[]AncestryStats) {
	creatures := flattenCreatureList(env)
	var am = make(map[*Creature]int)
	for _, c := range creatures {
		am[c.ancestor]++
	}
	mapToSortedAncestryStats(am, ancestryStats)
}

func printTickStats(tickStats *TickStats, carnivoreTotals *CarnivoreActionTotals) {
	avgDivisor := max(tickStats.TotalCreatures, 1)
	fmt.Printf("Tick: %d | TotalDeaths: %d | AgeDeaths: %d | StarvationDeaths: %d | FightDeaths: %d \n\rBiomassAdded: %.2f | BiomassTransferred: %.2f | BiomassConsumed: %.2f | Clones: %d | Eats: %d | Moves: %d | Fights: %d | Sleeps: %d\r\n",
		tickStats.Tick, tickStats.TotalDeaths, tickStats.AgeDeaths, tickStats.StarvationDeaths, tickStats.FightDeaths, tickStats.BiomassAdded, tickStats.BiomassTransferred[PlantGrass]+tickStats.BiomassTransferred[PlantClover]+tickStats.BiomassTransferred[PlantBramble], tickStats.BiomassConsumed, tickStats.Clones, tickStats.Eats, tickStats.Moves, tickStats.Fights, tickStats.Sleeps)
	fmt.Printf("Avg Age: %.1f | Avg Health: %.1f | Avg Nutrition: %.1f\r\n",
		float64(tickStats.TotalAge)/float64(avgDivisor), float64(tickStats.TotalHealth)/float64(avgDivisor), float64(tickStats.TotalNutrition)/float64(avgDivisor))
	fmt.Printf("Creatures: %d | Carnivores: %d | Herbivores: %d\r\n", tickStats.TotalCreatures, tickStats.TotalCarnivores, tickStats.TotalHerbivores)
	fmt.Print(formatPlantStats(tickStats))
	if carnivoreTotals != nil {
		fmt.Printf("Carnivore Totals | Eats: %d | Kills: %d | Clones: %d | Moves: %d | Fights: %d | Sleeps: %d | AgeDeaths: %d | StarvationDeaths: %d | FightDeaths: %d\r\n", carnivoreTotals.Eats, carnivoreTotals.Kills, carnivoreTotals.Clones, carnivoreTotals.Moves, carnivoreTotals.Fights, carnivoreTotals.Sleeps, carnivoreTotals.AgeDeaths, carnivoreTotals.StarvationDeaths, carnivoreTotals.FightDeaths)
	}
}

func formatPlantStats(stats *TickStats) string {
	names := [plantSpeciesCount]string{"Grass", "Clover", "Bramble"}
	var output strings.Builder
	fmt.Fprintf(&output, "Vegetation | Capacity: %.2f\r\n", stats.PlantCapacity)
	for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
		trait := stats.PlantTraits[species]
		if species > 0 {
			output.WriteString(" | ")
		}
		fmt.Fprintf(&output,
			"%s: %.2f (%d cells) | established %d | mutations %d | transferred %.2f | consumed %.2f | regrowth %.6f | spread %.6f | yield %.3f",
			names[species], stats.PlantBiomass[species], stats.PlantOccupiedCells[species],
			stats.PlantEstablishments[species], stats.PlantMutations[species],
			stats.BiomassTransferred[species], stats.PlantBiomassConsumed[species],
			trait.RegrowthRate, trait.SpreadProbability, trait.NutritionalYield)
	}
	output.WriteString("\r\n")
	fmt.Fprintf(&output,
		"Cumulative establishments: grass %d, clover %d, bramble %d | mutations: grass %d, clover %d, bramble %d | growth %.2f | transferred %.2f | consumed %.2f\r\n",
		stats.PlantCumulative.Establishments[PlantGrass], stats.PlantCumulative.Establishments[PlantClover], stats.PlantCumulative.Establishments[PlantBramble],
		stats.PlantCumulative.Mutations[PlantGrass], stats.PlantCumulative.Mutations[PlantClover], stats.PlantCumulative.Mutations[PlantBramble],
		stats.PlantCumulative.BiomassAdded,
		stats.PlantCumulative.BiomassTransferred[PlantGrass]+stats.PlantCumulative.BiomassTransferred[PlantClover]+stats.PlantCumulative.BiomassTransferred[PlantBramble],
		stats.PlantCumulative.BiomassConsumed[PlantGrass]+stats.PlantCumulative.BiomassConsumed[PlantClover]+stats.PlantCumulative.BiomassConsumed[PlantBramble])
	return output.String()
}

func printAncestryStats(ancestryStats *[]AncestryStats) {
	for i, a := range *ancestryStats {
		color := ""
		charStr := ""
		reset := ""

		if !isTest {
			color = a.Ancestor.colorCode
			charStr = fmt.Sprintf("%c", a.Ancestor.char)
			reset = "\033[0m"
		}

		fmt.Printf("%s%d. Ancestor: %s\t| Diet: %s\t| LineageCount: %d\t| Generation: %d\t| MaxHealth: %d\t| MaxAge: %d\t| MaxNutrition: %d\t| Metabolism: %d \t| Aggression: %d \t| Vision: %.2f%s\r\n", color, i+1, charStr, a.Ancestor.dietType(), a.LineageCount, a.Ancestor.generation, a.Ancestor.maxHealth, a.Ancestor.maxAge, a.Ancestor.maxNutrition, a.Ancestor.metabolism, a.Ancestor.aggression, a.Ancestor.vision, reset)
	}
}

func PrintNewestClones(env [][]environment) {
	fmt.Print("\r\n*** Newest Clones ***\r\n")
	sortedCreatures := SortCreatureList(env)
	creatures := sortedCreatures[:min(len(sortedCreatures), 3)]
	var as []AncestryStats
	for i, c := range creatures {
		as = append(as, AncestryStats{c, i})
	}
	printAncestryStats(&as)
}
