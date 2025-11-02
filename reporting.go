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

func printTickStats(tickStats *TickStats) {
	tickStats.TotalDeaths = tickStats.AgeDeaths + tickStats.StarvationDeaths + tickStats.FightDeaths
	tickStats.TotalCreatures = max(tickStats.TotalCreatures, 1)
	fmt.Printf("Tick: %d | Creatures: %d | TotalDeaths: %d | AgeDeaths: %d | StarvationDeaths: %d | FightDeaths: %d \n\rFoodAdded: %d | FoodConsumed: %d | Clones: %d | Eats: %d | Moves: %d | Fights: %d | Sleeps: %d\r\n",
		tickStats.Tick, tickStats.TotalCreatures, tickStats.TotalDeaths, tickStats.AgeDeaths, tickStats.StarvationDeaths, tickStats.FightDeaths, tickStats.FoodAdded, tickStats.FoodConsumed, tickStats.Clones, tickStats.Eats, tickStats.Moves, tickStats.Fights, tickStats.Sleeps)
	fmt.Printf("Avg Age: %.1f | Avg Health: %.1f | Avg Nutrition: %.1f\r\n",
		float64(tickStats.TotalAge)/float64(tickStats.TotalCreatures), float64(tickStats.TotalHealth)/float64(tickStats.TotalCreatures), float64(tickStats.TotalNutrition)/float64(tickStats.TotalCreatures))
}

func printAncestryStats(ancestryStats *[]AncestryStats) {
	for i, a := range *ancestryStats {
		color := ""
		charStr := ""

		if !isTest {
			color = a.Ancestor.colorCode
			charStr = fmt.Sprintf("%c", a.Ancestor.char)
		}

		fmt.Printf("%s%d. Ancestor: %s\t| LineageCount: %d\t| Generation: %d\t| MaxHealth: %d\t| MaxAge: %d\t| MaxNutrition: %d\t| Metabolism: %d \t| Aggression: %d \t| Vision: %.2f\r\n", color, i+1, charStr, a.LineageCount, a.Ancestor.generation, a.Ancestor.maxHealth, a.Ancestor.maxAge, a.Ancestor.maxNutrition, a.Ancestor.metabolism, a.Ancestor.aggression, a.Ancestor.vision)
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
