package main

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestSeededEcologyLongRunIsRepeatableAndBounded(t *testing.T) {
	replay := func(seed int64) (string, TickStats, [plantSpeciesCount]int) {
		ResetRand(seed)
		env := make([][]environment, 9)
		for y := range env {
			env[y] = make([]environment, 9)
			for x := range env[y] {
				env[y][x].capacity = plantCellCapacity
			}
		}
		env[1][1].plants[PlantGrass] = plantPopulation{biomass: 45, regrowthRate: plantTraits[PlantGrass].regrowth, spreadProbability: plantTraits[PlantGrass].spreadProbability, nutritionalYield: plantTraits[PlantGrass].nutrition}
		env[4][4].plants[PlantGrass] = plantPopulation{biomass: 5, regrowthRate: plantTraits[PlantGrass].regrowth, spreadProbability: plantTraits[PlantGrass].spreadProbability, nutritionalYield: plantTraits[PlantGrass].nutrition}
		env[4][4].plants[PlantClover] = plantPopulation{biomass: 45, regrowthRate: plantTraits[PlantClover].regrowth, spreadProbability: plantTraits[PlantClover].spreadProbability, nutritionalYield: plantTraits[PlantClover].nutrition}
		env[7][7].plants[PlantBramble] = plantPopulation{biomass: 45, regrowthRate: plantTraits[PlantBramble].regrowth, spreadProbability: plantTraits[PlantBramble].spreadProbability, nutritionalYield: plantTraits[PlantBramble].nutrition}
		stats := TickStats{}
		for tick := 0; tick < 2500; tick++ {
			stats.Tick = tick + 1
			simulateTick(env, func(*Creature) *Creature { return nil }, &stats, nil)
			for y := range env {
				for x := range env[y] {
					total := 0.0
					for species, population := range env[y][x].plants {
						if population.biomass < 0 {
							t.Fatalf("seed %d tick %d has negative species %d biomass at (%d,%d)", seed, tick, species, x, y)
						}
						total += population.biomass
						traits := plantTraits[species]
						if population.biomass > 0 && (population.regrowthRate < traits.regrowth*0.8 || population.regrowthRate > traits.regrowth*1.2 || population.spreadProbability < traits.spreadProbability*0.8 || population.spreadProbability > traits.spreadProbability*1.2 || population.nutritionalYield < traits.nutrition*0.8 || population.nutritionalYield > traits.nutrition*1.2) {
							t.Fatalf("seed %d tick %d has out-of-bounds traits at (%d,%d): %+v", seed, tick, x, y, population)
						}
					}
					if total > env[y][x].capacity+1e-9 {
						t.Fatalf("seed %d tick %d exceeds capacity at (%d,%d): %.8f", seed, tick, x, y, total)
					}
				}
			}
		}
		counts := stats.PlantOccupiedCells
		var state strings.Builder
		fmt.Fprintf(&state, "%+v\n", stats)
		mixedCells := 0
		for _, row := range env {
			for _, cell := range row {
				fmt.Fprintf(&state, "%+v\n", cell.plants)
				present := 0
				for _, population := range cell.plants {
					if population.biomass > 0 {
						present++
					}
				}
				if present > 1 {
					mixedCells++
				}
			}
		}
		if mixedCells == 0 {
			t.Fatalf("seed %d did not preserve any mixed-species cells", seed)
		}
		return state.String(), stats, counts
	}

	for _, seed := range []int64{3, 19, 2026} {
		first, stats, occupied := replay(seed)
		second, _, _ := replay(seed)
		if first != second {
			t.Fatalf("seed %d produced different world or statistics on replay", seed)
		}
		for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
			if occupied[species] <= 1 {
				t.Errorf("seed %d did not show a propagation front for species %d: %d occupied cells", seed, species, occupied[species])
			}
		}
		if stats.PlantCumulative.BiomassAdded <= 0 || math.IsNaN(stats.PlantCumulative.BiomassAdded) || math.IsInf(stats.PlantCumulative.BiomassAdded, 0) {
			t.Fatalf("seed %d has invalid cumulative growth %.8f", seed, stats.PlantCumulative.BiomassAdded)
		}
	}
}

func TestGrazedPlantCellRegrowsTowardItsPriorBiomass(t *testing.T) {
	ResetRand(91)
	env := [][]environment{{{capacity: plantCellCapacity, plants: [plantSpeciesCount]plantPopulation{{biomass: 20}}}}}
	creature := &Creature{X: 0, Y: 0, maxNutrition: 1000}
	resolveEat(ActionIntent{Creature: creature, Type: "eat"}, env, &TickStats{})
	clearing := env[0][0].plants[PlantGrass].biomass
	for range 100 {
		processEnvironment(env, nil, nil)
	}
	if env[0][0].plants[PlantGrass].biomass <= clearing {
		t.Fatalf("grazed clearing did not regrow: after grazing %.2f, after regrowth %.2f", clearing, env[0][0].plants[PlantGrass].biomass)
	}
}

func TestSimulatedGrazingIsCountedOnceBySpeciesAndCumulatively(t *testing.T) {
	ResetRand(22)
	creature := &Creature{
		X: 0, Y: 0, health: 100, maxHealth: 100, nutrition: 0, maxNutrition: 1000,
		maxAge: 100, metabolism: 0, vision: 0,
	}
	env := [][]environment{{{
		capacity:     plantCellCapacity,
		plants:       [plantSpeciesCount]plantPopulation{{biomass: 10}, {biomass: 5}, {biomass: 0}},
		creatureList: []*Creature{creature},
	}}}
	stats := &TickStats{}
	simulateTick(env, func(*Creature) *Creature { return nil }, stats, nil)
	consumed := stats.PlantBiomassConsumed[PlantGrass] + stats.PlantBiomassConsumed[PlantClover] + stats.PlantBiomassConsumed[PlantBramble]
	if consumed <= 0 || math.Abs(consumed-stats.BiomassConsumed) > 1e-9 {
		t.Fatalf("per-species grazing %.3f does not reconcile with total %.3f", consumed, stats.BiomassConsumed)
	}
	cumulative := stats.PlantCumulative.BiomassConsumed[PlantGrass] + stats.PlantCumulative.BiomassConsumed[PlantClover] + stats.PlantCumulative.BiomassConsumed[PlantBramble]
	if math.Abs(cumulative-consumed) > 1e-9 {
		t.Fatalf("cumulative grazing %.3f does not match tick grazing %.3f", cumulative, consumed)
	}
}
