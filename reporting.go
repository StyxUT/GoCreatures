package main

import (
	"fmt"
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
	fmt.Printf("Tick: %d | TotalDeaths: %d | AgeDeaths: %d | StarvationDeaths: %d | FightDeaths: %d \n\rBiomassAdded: %.2f | BiomassConsumed: %.2f | Clones: %d | Eats: %d | Moves: %d | Fights: %d | Sleeps: %d\r\n",
		tickStats.Tick, tickStats.TotalDeaths, tickStats.AgeDeaths, tickStats.StarvationDeaths, tickStats.FightDeaths, tickStats.BiomassAdded, tickStats.BiomassConsumed, tickStats.Clones, tickStats.Eats, tickStats.Moves, tickStats.Fights, tickStats.Sleeps)
	fmt.Printf("Avg Age: %.1f | Avg Health: %.1f | Avg Nutrition: %.1f\r\n",
		float64(tickStats.TotalAge)/float64(avgDivisor), float64(tickStats.TotalHealth)/float64(avgDivisor), float64(tickStats.TotalNutrition)/float64(avgDivisor))
	fmt.Printf("Creatures: %d | Carnivores: %d | Herbivores: %d\r\n", tickStats.TotalCreatures, tickStats.TotalCarnivores, tickStats.TotalHerbivores)
	fmt.Printf("Vegetation | Grass: %.2f | Clover: %.2f | Bramble: %.2f | Capacity: %.2f\r\n",
		tickStats.PlantBiomass[PlantGrass], tickStats.PlantBiomass[PlantClover], tickStats.PlantBiomass[PlantBramble], tickStats.PlantCapacity)
	if carnivoreTotals != nil {
		fmt.Printf("Carnivore Totals | Eats: %d | Kills: %d | Clones: %d | Moves: %d | Fights: %d | Sleeps: %d | AgeDeaths: %d | StarvationDeaths: %d | FightDeaths: %d\r\n", carnivoreTotals.Eats, carnivoreTotals.Kills, carnivoreTotals.Clones, carnivoreTotals.Moves, carnivoreTotals.Fights, carnivoreTotals.Sleeps, carnivoreTotals.AgeDeaths, carnivoreTotals.StarvationDeaths, carnivoreTotals.FightDeaths)
	}
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
