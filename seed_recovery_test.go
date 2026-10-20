package main

import (
	"math"
	"testing"
)

func TestGrazedSeedLocationsRecoverAllSpeciesThroughSimulationTicks(t *testing.T) {
	// Arrange: retain each founding species/location pair, then graze the world bare.
	previousRand := rng
	t.Cleanup(func() { SetRand(previousRand) })
	ResetRand(41)
	env := initializeMap(12, 12, CreatureFactory())
	type seedLocation struct {
		x, y    int
		species PlantSpecies
	}
	seeded := make(map[seedLocation]bool)
	var seedCounts [plantSpeciesCount]int
	stats := &TickStats{}
	for y := range env {
		for x := range env[y] {
			cell := &env[y][x]
			cell.creatureList = nil
			for species, plant := range cell.plants {
				if plant.biomass > 0 {
					seeded[seedLocation{x, y, PlantSpecies(species)}] = true
					seedCounts[species]++
				}
			}
			herbivore := &Creature{X: x, Y: y, maxNutrition: 100}
			for {
				biomass := 0.0
				for _, plant := range cell.plants {
					biomass += plant.biomass
				}
				if biomass == 0 {
					break
				}
				resolveEat(ActionIntent{Creature: herbivore, Type: "eat"}, env, stats)
			}
		}
	}
	for species, count := range seedCounts {
		if count == 0 || count == len(env)*len(env[0]) {
			t.Fatalf("species %d needs both seeded and unseeded locations, got %d", species, count)
		}
	}

	// Act / Assert: no biomass returns before the delay; every original pair recovers.
	for tick := 1; tick <= plantRecoveryDelay; tick++ {
		simulateTick(env, nil, stats, nil)
		for y := range env {
			for x := range env[y] {
				for species, plant := range env[y][x].plants {
					want := 0.0
					if tick == plantRecoveryDelay && seeded[seedLocation{x, y, PlantSpecies(species)}] {
						want = plantRecoveryBiomass
					}
					if plant.biomass != want {
						t.Fatalf("tick %d species %d at (%d,%d): biomass %g, want %g", tick, species, x, y, plant.biomass, want)
					}
				}
			}
		}
	}
	for species, count := range seedCounts {
		want := float64(count) * plantRecoveryBiomass
		if stats.PlantRecoveries[species] != count || stats.PlantCumulative.Recoveries[species] != count || stats.PlantBiomassRecovered[species] != want || stats.PlantBiomass[species] != want {
			t.Fatalf("species %d recovery accounting does not match %d seed locations: %+v", species, count, stats)
		}
		if stats.PlantEstablishments[species] != 0 || stats.BiomassTransferred[species] != 0 {
			t.Fatalf("species %d recovery counted as spread", species)
		}
	}

	// Recovered colonies resume ordinary growth without spending more reserves.
	simulateTick(env, nil, stats, nil)
	for location := range seeded {
		plant := env[location.y][location.x].plants[location.species]
		if plant.biomass <= plantRecoveryBiomass || plant.recoveryReserve != plantRecoveryReserve-plantRecoveryBiomass {
			t.Fatalf("recovered colony did not resume normal growth at %+v: %+v", location, plant)
		}
	}
	if stats.BiomassRegrown <= 0 || stats.BiomassRecovered != 0 || math.Abs(stats.BiomassAdded-stats.BiomassRegrown) > 1e-9 {
		t.Fatalf("post-recovery growth accounting is incorrect: %+v", stats)
	}
}
