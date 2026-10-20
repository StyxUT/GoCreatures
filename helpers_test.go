package main

import (
	"testing"
)

func TestSpiralLook2D_N_LowVisionCreature(t *testing.T) {
	// Arrange
	env := make([][]environment, 5)
	for i := range env {
		env[i] = make([]environment, 5)
	}

	cLowVision := LowVisionCreature()
	cLowVision.X, cLowVision.Y = 2, 2 // center
	cLowVision.observed = make([]ObservedLocation, 0)

	// Act
	SpiralLook2D_N(env, cLowVision)

	// Assert
	expected := int(cLowVision.vision)
	if len(cLowVision.observed) != expected {
		t.Errorf("expected to observe %d locations, got %d", expected, len(cLowVision.observed))
	}

	seen := map[*environment]bool{}
	for i, loc := range cLowVision.observed {
		if loc.Env == nil {
			t.Errorf("nil pointer observed at index %d", i)
		}
		if seen[loc.Env] {
			t.Errorf("duplicate environment pointer observed at index %d", i)
		}
		seen[loc.Env] = true
	}
}

func TestSpiralLook2D_N_HighVisionCreature(t *testing.T) {
	// Arrange
	env := make([][]environment, 5)
	for i := range env {
		env[i] = make([]environment, 5)
	}

	cHighVision := HighVisionCreature()
	cHighVision.X, cHighVision.Y = 0, 0 // off center
	cHighVision.observed = make([]ObservedLocation, 0)

	// Act
	SpiralLook2D_N(env, cHighVision)

	// Assert
	expected := 8 // 8 not 20 due to vision exceeing bounds
	if len(cHighVision.observed) != expected {
		t.Errorf("expected to observe %d locations, got %d", expected, len(cHighVision.observed))
	}

	// Optional: ensure all references are non-nil and distinct
	seen := map[*environment]bool{}
	for i, loc := range cHighVision.observed {
		if loc.Env == nil {
			t.Errorf("nil pointer observed at index %d", i)
		}
		if loc.X == cHighVision.X && loc.Y == cHighVision.Y {
			t.Errorf("current cell was included in observations at index %d", i)
		}
		if seen[loc.Env] {
			t.Errorf("duplicate environment pointer observed at index %d", i)
		}
		seen[loc.Env] = true
	}
}

func TestSpiralLook2D_N_EmptyWorldAndInvalidPositionAreSafe(t *testing.T) {
	creature := &Creature{X: 5, Y: 0, observed: []ObservedLocation{{X: 1, Y: 1}}}
	SpiralLook2D_N(nil, creature)
	SpiralLook2D_N([][]environment{{}}, creature)
	SpiralLook2D_N([][]environment{{{}}}, creature)
	if len(creature.observed) != 0 {
		t.Fatalf("invalid perception retained stale observations: %+v", creature.observed)
	}
}

func TestMapToSortedAncestryStatsKeepsTopFive(t *testing.T) {
	ancestors := make([]*Creature, 6)
	counts := make(map[*Creature]int, len(ancestors))
	for i := range ancestors {
		ancestors[i] = &Creature{num: i + 1}
		counts[ancestors[i]] = i + 1
	}
	stats := []AncestryStats{{LineageCount: 99}}

	mapToSortedAncestryStats(counts, &stats)

	if len(stats) != 5 {
		t.Fatalf("got %d ancestry entries, want 5", len(stats))
	}
	for i, stat := range stats {
		if stat.LineageCount != 6-i {
			t.Errorf("stats[%d].LineageCount = %d, want %d", i, stat.LineageCount, 6-i)
		}
	}
}

func TestMapToSortedAncestryStatsEmptyInput(t *testing.T) {
	stats := []AncestryStats{{LineageCount: 1}}
	mapToSortedAncestryStats(nil, &stats)
	if len(stats) != 0 {
		t.Fatalf("got %d entries for empty input, want 0", len(stats))
	}
}
