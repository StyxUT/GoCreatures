package main

import (
	"testing"
)

// Test that ProcessEnvironment removes dead creatures
func TestProcessEnvironment_RemoveDeadCreatures(t *testing.T) {
	// Arrange
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}

	c1 := DeadCreature()
	c2 := DeadCreature()
	c3 := DeadCreature()
	c4 := HealthyCreature()
	c5 := DeadCreature()

	env[1][1].creatureList = append(env[1][1].creatureList, c1)
	env[1][1].creatureList = append(env[1][1].creatureList, c2)
	env[1][1].creatureList = append(env[1][1].creatureList, c4)
	env[1][2].creatureList = append(env[1][2].creatureList, c3)
	env[0][1].creatureList = append(env[0][1].creatureList, c5)

	tickStats := TickStats{}

	// Act
	processEnvironment(env, &tickStats)
	cellCount := len(env[1][1].creatureList)
	listCount := len(flattenCreatureList(env))

	// Assert
	if cellCount != 1 {
		t.Errorf("expected 1 living creature, but got %d", cellCount)
	}
	if listCount != 1 {
		t.Errorf("expected 1 living creature in the flattened creature list, but got %d", listCount)
	}
}
