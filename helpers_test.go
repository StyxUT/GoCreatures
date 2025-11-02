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
		if seen[loc.Env] {
			t.Errorf("duplicate environment pointer observed at index %d", i)
		}
		seen[loc.Env] = true
	}
}
