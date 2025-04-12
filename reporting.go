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
	fmt.Printf("Tick: %d | Creatures: %d | TotalDeaths: %d | AgeDeaths: %d | StarvationDeaths: %d | FightDeaths: %d | Clones: %d\nFoodAdded: %d | FoodConsumed: %d | Eats: %d | Moves: %d | Fights: %d | Sleeps: %d\n",
		tickStats.Tick, tickStats.TotalCreatures, tickStats.TotalDeaths, tickStats.AgeDeaths, tickStats.StarvationDeaths, tickStats.FightDeaths, tickStats.Clones, tickStats.FoodAdded, tickStats.FoodConsumed, tickStats.Eats, tickStats.Moves, tickStats.Fights, tickStats.Sleeps)
	fmt.Printf("Avg Age: %.1f | Avg Health: %.1f | Avg Nutrition: %.1f\n",
		float64(tickStats.TotalAge)/float64(tickStats.TotalCreatures), float64(tickStats.TotalHealth)/float64(tickStats.TotalCreatures), float64(tickStats.TotalNutrition)/float64(tickStats.TotalCreatures))
}

func printAncestryStats(ancestryStats *[]AncestryStats) {
	for i, a := range *ancestryStats {
		fmt.Printf("%s%d. Ancestor: %c\t| LineageCount: %d\t| MaxHealth: %d\t| MaxAge: %d\t| MaxNutrition: %d\t| Metabolism: %d \t| Aggression: %d\n", a.Ancestor.colorCode, i+1, a.Ancestor.char, a.LineageCount, a.Ancestor.maxHealth, a.Ancestor.maxAge, a.Ancestor.maxNutrition, a.Ancestor.metabolism, a.Ancestor.aggression)
	}
}
