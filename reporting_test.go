package main

import (
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
