package main

import (
	"testing"
)

// Test creature cloaning
func TestResolveClone(t *testing.T) {
	// Arrange
	parent := HealthyCreature()
	parent.X, parent.Y = 2, 3

	env := make([][]environment, 5)
	for i := range env {
		env[i] = make([]environment, 5)
	}

	env[parent.Y][parent.X].creatureList = append(env[parent.Y][parent.X].creatureList, parent)

	intent := ActionIntent{
		Creature: parent,
		Type:     "Clone", // not used as resolveClone() is called directly
	}

	// Act
	resolveClone(intent, env, func(original *Creature) *Creature {
		// clone function can return a shallow copy or deep copy
		return &Creature{
			maxHealth:    original.maxHealth,
			maxNutrition: original.maxNutrition,
		}
	})

	// Assert
	creatures := env[parent.Y][parent.X].creatureList
	if len(creatures) != 2 {
		t.Fatalf("expected 2 creatures at (%d,%d), got %d", parent.X, parent.Y, len(creatures))
	}

	clone := creatures[1]
	if clone.health != int(float64(clone.maxHealth)*0.1) {
		t.Errorf("expected clone health to be 10%% of maxHealth, got %d", clone.health)
	}
	if clone.nutrition != int(float64(clone.maxNutrition)*0.1) {
		t.Errorf("expected clone nutrition to be 10%% of maxNutrition, got %d", clone.nutrition)
	}

	expectedParentHealth := int(float64(100) * 0.3) // assuming HealthyCreature() has 100 health
	if parent.health != expectedParentHealth {
		t.Errorf("expected parent health to be reduced to %d, got %d", expectedParentHealth, parent.health)
	}
}

// Test creature fighting
func TestResolveFight(t *testing.T) {
	attacker := &Creature{health: 1, nutrition: 1}
	defender := &Creature{health: 100, nutrition: 100}
	stats := &TickStats{}

	resolveFight(ActionIntent{
		Creature:       attacker,
		TargetCreature: defender,
	}, stats)

	if attacker.health > 0 {
		t.Errorf("expected attacker to die, got health %d", attacker.health)
	}
	if stats.FightDeaths == 0 {
		t.Errorf("expected death to be recorded in FightDeaths")
	}
}

// Test creature eating
func TestResolveEat(t *testing.T) {
	// Arrange
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}

	creature := &Creature{
		X:         1,
		Y:         1,
		nutrition: 0,
	}

	// Place food at the creature's location
	env[1][1].food = 4 // less than 5 to test min()
	env[1][1].creatureList = append(env[1][1].creatureList, creature)

	intent := ActionIntent{
		Creature: creature,
		Type:     "eat",
	}

	stats := &TickStats{}

	// Act
	resolveEat(intent, env, stats)

	// Assert
	expectedNutritionGain := 10 * 4
	expectedRemainingFood := 0
	expectedFoodConsumed := 4

	if creature.nutrition != expectedNutritionGain {
		t.Errorf("expected nutrition to be %d, got %d", expectedNutritionGain, creature.nutrition)
	}

	if env[1][1].food != expectedRemainingFood {
		t.Errorf("expected food to be %d, got %d", expectedRemainingFood, env[1][1].food)
	}

	if stats.FoodConsumed != expectedFoodConsumed {
		t.Errorf("expected FoodConsumed to be %d, got %d", expectedFoodConsumed, stats.FoodConsumed)
	}
}
