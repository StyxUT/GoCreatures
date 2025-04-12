package main

import (
	"testing"
)

// TestNextSafeRune verifies that nextSafeRune() returns the expected runes in sequence.
func TestNextSafeRune(t *testing.T) {
	safeRunes = []rune{'α', 'β', 'γ'} // test-safe pool
	nextRuneIndex = 0                 // reset index

	tests := []rune{'α', 'β', 'γ', 'α', 'β'}
	for i, expected := range tests {
		r := NextSafeRune()
		if r != expected {
			t.Errorf("Test %d: expected %q, got %q", i, expected, r)
		}
	}
}
