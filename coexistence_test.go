package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestSeededCoexistenceDiagnostic(t *testing.T) {
	run := func(seed int64, recoveryEnabled bool) string {
		ResetRand(seed)
		env := make([][]environment, 7)
		for y := range env {
			env[y] = make([]environment, 7)
			for x := range env[y] {
				env[y][x].capacity = plantCellCapacity
			}
		}
		seedPlants(env)
		if !recoveryEnabled {
			for y := range env {
				for x := range env[y] {
					for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
						env[y][x].plants[species].recoveryEligible = false
						env[y][x].plants[species].recoveryReserve = 0
					}
				}
			}
		}
		for i := 0; i < 8; i++ {
			diet := DietHerbivore
			if i < 2 {
				diet = DietCarnivore
			}
			c := &Creature{
				num: i + 1, X: (i*3 + 1) % 7, Y: (i*2 + 1) % 7,
				health: 100, maxHealth: 100, nutrition: 85, maxNutrition: 100,
				age: 5, maxAge: 2000, metabolism: 2, vision: 8, diet: diet,
				ancestor: &Creature{num: i + 1},
			}
			if diet == DietCarnivore {
				c.health, c.maxHealth = 300, 300
				c.maxNutrition, c.nutrition = 200, 170
			}
			env[c.Y][c.X].creatureList = append(env[c.Y][c.X].creatureList, c)
		}
		var history strings.Builder
		stats := TickStats{}
		factory := CreatureFactory()
		var carnivoreTotals CarnivoreActionTotals
		clones := 0
		deaths := 0
		herbivoreDeaths := 0
		carnivoreDeaths := 0
		herbivoreBirths := 0
		carnivoreBirths := 0
		plantGrowth := 0.0
		plantConsumption := 0.0
		carcassFood := 0.0
		herbivoreExtinctionTick := 0
		carnivoreExtinctionTick := 0
		for tick := 1; tick <= 500; tick++ {
			stats.Tick = tick
			carnivoreDeathsBefore := carnivoreTotals.AgeDeaths + carnivoreTotals.StarvationDeaths + carnivoreTotals.FightDeaths
			carnivoreBirthsBefore := carnivoreTotals.Clones
			simulateTick(env, factory.create, &stats, &carnivoreTotals)
			clones += stats.Clones
			carnivoreBirths += carnivoreTotals.Clones - carnivoreBirthsBefore
			herbivoreBirths += stats.Clones - (carnivoreTotals.Clones - carnivoreBirthsBefore)
			carnivoreDeaths += carnivoreTotals.AgeDeaths + carnivoreTotals.StarvationDeaths + carnivoreTotals.FightDeaths - carnivoreDeathsBefore
			deaths += stats.TotalDeaths
			herbivoreDeaths += stats.TotalDeaths - (carnivoreTotals.AgeDeaths + carnivoreTotals.StarvationDeaths + carnivoreTotals.FightDeaths - carnivoreDeathsBefore)
			plantGrowth += stats.BiomassAdded
			plantConsumption += stats.BiomassConsumed
			carcassFood += stats.CarcassEnergyEaten
			if stats.TotalHerbivores == 0 && herbivoreExtinctionTick == 0 {
				herbivoreExtinctionTick = tick
			}
			if stats.TotalCarnivores == 0 && carnivoreExtinctionTick == 0 {
				carnivoreExtinctionTick = tick
			}
			if tick%100 == 0 {
				fmt.Fprintf(&history, "tick=%d creatures=%d herbivores=%d carnivores=%d plants=%.1f/%.1f/%.1f births=%d(h=%d,c=%d) deaths=%d(h=%d,c=%d) plantGrowth=%.1f plantConsumed=%.1f carcassFood=%.1f kills=%d fights=%d\n",
					tick, stats.TotalCreatures, stats.TotalHerbivores, stats.TotalCarnivores,
					stats.PlantBiomass[PlantGrass], stats.PlantBiomass[PlantClover], stats.PlantBiomass[PlantBramble],
					clones, herbivoreBirths, carnivoreBirths, deaths, herbivoreDeaths, carnivoreDeaths,
					plantGrowth, plantConsumption, carcassFood, carnivoreTotals.Kills, carnivoreTotals.Fights)
			}
		}
		fmt.Fprintf(&history, "firstExtinctionTick=herbivore:%d carnivore:%d (0 means none by tick 500)\n", herbivoreExtinctionTick, carnivoreExtinctionTick)
		return history.String()
	}

	for _, seed := range []int64{5, 23, 91, 2026, 9127} {
		for _, recoveryEnabled := range []bool{false, true} {
			first := run(seed, recoveryEnabled)
			if replay := run(seed, recoveryEnabled); replay != first {
				t.Fatalf("seed %d (recovery enabled %t) was not repeatable", seed, recoveryEnabled)
			}
			mode := "baseline without recovery"
			if recoveryEnabled {
				mode = "with recovery"
			}
			t.Logf("seed %d %s trajectory:\n%s", seed, mode, first)
		}
	}
}
