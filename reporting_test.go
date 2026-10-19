package main

import (
	"bytes"
	"io"
	"math"
	"os"
	"strings"
	"testing"
)

func TestPrintNewestClones(t *testing.T) {
	isTest = true

	defer func() {
		isTest = false
	}()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printNewestClones panicked: %v", r)
		}
	}()

	// Arrange
	creatures := []*Creature{
		AggressiveCreature(),
		HealthyCreature(),
	}

	env := make([][]environment, 2)
	for i := range env {
		env[i] = make([]environment, 2) // now 2x2 grid
	}
	env[1][1].creatureList = creatures

	// Act
	PrintNewestClones(env)

	// Assert
	// panics will be caught and reported by defer/recover
}

func TestPrintAncestryStats_IncludesDiet(t *testing.T) {
	isTest = true
	defer func() {
		isTest = false
	}()

	stats := []AncestryStats{{Ancestor: &Creature{diet: DietCarnivore}, LineageCount: 2}}
	output := captureStdout(t, func() {
		printAncestryStats(&stats)
	})

	if !strings.Contains(output, "Diet: Carnivore") {
		t.Fatalf("expected diet column in output, got %q", output)
	}
}

func TestPrintTickStats_DoesNotPanicWithZeroCreatures(t *testing.T) {
	tickStats := &TickStats{}
	output := captureStdout(t, func() {
		printTickStats(tickStats, nil)
	})

	if tickStats.TotalCreatures != 0 {
		t.Fatalf("expected zero-creature stats to remain unchanged, got %d", tickStats.TotalCreatures)
	}
	if !strings.Contains(output, "Creatures: 0 | Carnivores: 0 | Herbivores: 0") {
		t.Fatalf("expected zero-creature counts in output, got %q", output)
	}
}

func TestPrintTickStats_ReadOnly(t *testing.T) {
	// Arrange: the renderer must not complete or correct its input.
	stats := TickStats{TotalDeaths: 7, AgeDeaths: 1, TotalCreatures: 2, TotalHealth: 150}
	before := stats

	// Act
	output := captureStdout(t, func() { printTickStats(&stats, nil) })

	// Assert
	if stats != before || !strings.Contains(output, "TotalDeaths: 7") {
		t.Fatalf("reporting changed completed statistics: %+v / %q", stats, output)
	}
}

func TestPrintTickStatsReportsEcologySnapshotAndCumulativeEvents(t *testing.T) {
	stats := TickStats{
		PlantBiomass:        [plantSpeciesCount]float64{42, 25, 10},
		PlantOccupiedCells:  [plantSpeciesCount]int{3, 2, 1},
		PlantEstablishments: [plantSpeciesCount]int{2, 1, 0},
		PlantMutations:      [plantSpeciesCount]int{1, 0, 0},
		PlantTraits: [plantSpeciesCount]PlantTraitSummary{
			{RegrowthRate: 0.080123, SpreadProbability: 0.020123, NutritionalYield: 15.125},
		},
		PlantCumulative: PlantCumulativeTotals{
			Establishments: [plantSpeciesCount]int{7, 4, 2},
			Mutations:      [plantSpeciesCount]int{2, 1, 0},
		},
	}
	output := captureStdout(t, func() { printTickStats(&stats, nil) })
	for _, want := range []string{
		"Grass: 42.00 (3 cells)", "established 2", "mutations 1",
		"regrowth 0.080123", "spread 0.020123", "yield 15.125",
		"Cumulative establishments: grass 7, clover 4, bramble 2",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("plant report missing %q: %s", want, output)
		}
	}
}

func TestPlantReportUsesFewerRowsWithoutCombiningSpeciesData(t *testing.T) {
	stats := TickStats{
		PlantBiomass: [plantSpeciesCount]float64{42, 25, 10},
		PlantTraits: [plantSpeciesCount]PlantTraitSummary{
			{RegrowthRate: 0.08, SpreadProbability: 0.02, NutritionalYield: 15},
			{RegrowthRate: 0.04, SpreadProbability: 0.01, NutritionalYield: 20},
			{RegrowthRate: 0.02, SpreadProbability: 0.005, NutritionalYield: 10},
		},
	}
	output := formatPlantStats(&stats)
	lines := strings.Split(strings.TrimSuffix(output, "\r\n"), "\r\n")
	if len(lines) != 3 {
		t.Fatalf("plant report has %d rows, want 3: %q", len(lines), output)
	}
	for _, want := range []string{"Grass: 42.00", "Clover: 25.00", "Bramble: 10.00"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("compact species row missing %q: %q", want, lines[1])
		}
	}
}

func TestSimulationPlantReportsAreFinalSnapshotsWithCumulativeEvents(t *testing.T) {
	ResetRand(304)
	env := [][]environment{{
		{capacity: plantCellCapacity, plants: [plantSpeciesCount]plantPopulation{{biomass: 10, regrowthRate: 0.081, spreadProbability: 0.019, nutritionalYield: 15.25}}},
		{capacity: plantCellCapacity, plants: [plantSpeciesCount]plantPopulation{{biomass: 20, regrowthRate: 0.083, spreadProbability: 0.021, nutritionalYield: 16.25}}},
	}}
	stats := &TickStats{}
	noClone := func(*Creature) *Creature { return nil }

	simulateTick(env, noClone, stats, nil)
	firstGrowth := stats.BiomassAdded
	if stats.PlantOccupiedCells[PlantGrass] != 2 {
		t.Fatalf("occupied grass cells = %d, want 2", stats.PlantOccupiedCells[PlantGrass])
	}
	grassAtLeft := env[0][0].plants[PlantGrass]
	grassAtRight := env[0][1].plants[PlantGrass]
	wantRegrowth := (grassAtLeft.regrowthRate*grassAtLeft.biomass + grassAtRight.regrowthRate*grassAtRight.biomass) / (grassAtLeft.biomass + grassAtRight.biomass)
	if math.Abs(stats.PlantTraits[PlantGrass].RegrowthRate-wantRegrowth) > 1e-12 {
		t.Fatalf("biomass-weighted regrowth rate = %.8f, want %.8f", stats.PlantTraits[PlantGrass].RegrowthRate, wantRegrowth)
	}
	if stats.PlantBiomass[PlantGrass] != env[0][0].plants[PlantGrass].biomass+env[0][1].plants[PlantGrass].biomass {
		t.Fatal("plant biomass report does not match final world snapshot")
	}

	simulateTick(env, noClone, stats, nil)
	if stats.BiomassAdded <= 0 || stats.BiomassAdded >= firstGrowth*1.1 {
		t.Fatalf("second tick growth is not a per-tick value: first %.6f, second %.6f", firstGrowth, stats.BiomassAdded)
	}
	if math.Abs(stats.PlantCumulative.BiomassAdded-firstGrowth-stats.BiomassAdded) > 1e-9 {
		t.Fatalf("cumulative growth %.8f does not equal tick sum %.8f", stats.PlantCumulative.BiomassAdded, firstGrowth+stats.BiomassAdded)
	}
	if stats.PlantTraits[PlantClover] != (PlantTraitSummary{}) {
		t.Fatalf("extinct species summary should be zero, got %+v", stats.PlantTraits[PlantClover])
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()
	_ = w.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy stdout: %v", err)
	}
	_ = r.Close()
	return buf.String()
}
