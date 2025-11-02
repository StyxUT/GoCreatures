package main

import (
	"math/rand"
	"testing"
)

// TestDeterministicRNG ensures that injecting a deterministic RNG produces stable sequences.
func TestDeterministicRNG(t *testing.T) {
	// Arrange
	seed := int64(1)
	ResetRand(seed)
	firstInts := []int{RandIntn(10), RandIntn(10), RandIntn(10)}
	firstFloat := RandFloat64()

	// Reset and re-run
	ResetRand(seed)
	secondInts := []int{RandIntn(10), RandIntn(10), RandIntn(10)}
	secondFloat := RandFloat64()

	// Assert identical sequences
	for i := range firstInts {
		if firstInts[i] != secondInts[i] {
			v := rand.New(rand.NewSource(seed)).Int() // additional detail
			t.Fatalf("expected deterministic int sequence, mismatch at %d: %d vs %d (seed sanity %d)", i, firstInts[i], secondInts[i], v)
		}
	}
	if firstFloat != secondFloat {
		t.Fatalf("expected deterministic float sequence, got %f vs %f", firstFloat, secondFloat)
	}
}
