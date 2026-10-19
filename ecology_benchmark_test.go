package main

import (
	"os"
	"testing"
)

// These default-world benchmarks keep the same 38x81 setup and seed across
// revisions; the pre-vegetation comparison adapts only legacy API signatures.
func BenchmarkDefaultWorldTickAndRender(b *testing.B) {
	ResetRand(42)
	factory := CreatureFactory()
	env := initializeMap(38, 81, factory)
	var stats TickStats
	output, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	previousStdout := os.Stdout
	os.Stdout = output
	b.Cleanup(func() {
		os.Stdout = previousStdout
		_ = output.Close()
	})
	b.ResetTimer()
	for range b.N {
		simulateTick(env, factory.create, &stats, nil)
		printMap(env)
	}
}

func BenchmarkDefaultWorldTick(b *testing.B) {
	ResetRand(42)
	factory := CreatureFactory()
	env := initializeMap(38, 81, factory)
	var stats TickStats
	b.ResetTimer()
	for range b.N {
		simulateTick(env, factory.create, &stats, nil)
	}
}

func BenchmarkDefaultWorldRender(b *testing.B) {
	ResetRand(42)
	env := initializeMap(38, 81, CreatureFactory())
	output, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	previousStdout := os.Stdout
	os.Stdout = output
	b.Cleanup(func() {
		os.Stdout = previousStdout
		_ = output.Close()
	})
	b.ResetTimer()
	for range b.N {
		printMap(env)
	}
}

func BenchmarkDefaultWorldPlantMaintenance(b *testing.B) {
	ResetRand(42)
	factory := CreatureFactory()
	env := initializeMap(38, 81, factory)
	for y := range env {
		for x := range env[y] {
			env[y][x].creatureList = nil
		}
	}
	var stats TickStats
	b.ResetTimer()
	for range b.N {
		simulateTick(env, factory.create, &stats, nil)
	}
}
