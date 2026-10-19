package main

import (
	"bytes"
	"io"
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
