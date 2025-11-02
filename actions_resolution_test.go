package main

import (
	"testing"
)

// Test that a creature involved in a fight does not take any other action
func TestResolveActions_SkipEatAfterFight(t *testing.T) {
	// Arrange: create two creatures that will fight
	attacker := &Creature{X: 1, Y: 1, health: 100, nutrition: 50}
	defender := &Creature{X: 1, Y: 1, health: 100, nutrition: 50}

	// Place them in the same environment cell with food
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	env[1][1].creatureList = []*Creature{attacker, defender}
	env[1][1].food = 5

	stats := &TickStats{}

	// Intent 1: fight
	fightIntent := ActionIntent{
		Creature:       attacker,
		TargetCreature: defender,
		Type:           "fight",
	}

	// Intent 2: attacker also tries to eat in same tick
	eatIntent := ActionIntent{
		Creature: attacker,
		Type:     "eat",
	}

	// Act
	resolveActions([]ActionIntent{fightIntent, eatIntent}, env, func(c *Creature) *Creature {
		return &Creature{maxHealth: 100, maxNutrition: 100}
	}, stats)

	// Assert
	expectedNutrition := 50 - 10*3 // lost 10 during fight (3 rounds)
	if attacker.nutrition != expectedNutrition {
		t.Errorf("expected nutrition to be %d, got %d (eat should be skipped)", expectedNutrition, attacker.nutrition)
	}

	if env[1][1].food != 5 {
		t.Errorf("expected food to remain at 5 (no eating), got %d", env[1][1].food)
	}
}

// Test creature cloning
func TestResolveClone(t *testing.T) {
	// Arrange
	parent := HealthyCreature()
	ancestor := DeadCreature()
	parent.ancestor = ancestor

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
	resolveClone(intent, env, CreatureFactory())

	// Assert
	creatures := env[parent.Y][parent.X].creatureList
	if len(creatures) != 2 {
		t.Errorf("expected 2 creatures at (%d,%d), but got %d", parent.X, parent.Y, len(creatures))
	}

	clone := creatures[1]
	if clone.health != 30 {
		t.Errorf("expected clone health to be 29, but got %d", clone.health)
	}
	if clone.nutrition != 30 {
		t.Errorf("expected clone nutrition to be 29, but got %d", clone.nutrition) 
	}

	if parent.health != 30 {
		t.Errorf("expected parent health to be 30 , but got %d", parent.health)
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
	expectedNutritionGain := 15 * 4

	if creature.nutrition != expectedNutritionGain {
		t.Errorf("expected nutrition to be %d, but got %d", expectedNutritionGain, creature.nutrition)
	}
	if env[1][1].food != 0 {
		t.Errorf("expected food to be 0, but got %d", env[1][1].food)
	}
	if stats.FoodConsumed != 4 {
		t.Errorf("expected FoodConsumed to be 4, but got %d", stats.FoodConsumed)
	}
}

// Test creature moves toward empty location
func TestCreatureMovesTowardEmptyLocation(t *testing.T) {
	// Create 3x3 environment with food in a known spot
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}

	// Seed each location with a creature
	for i := range env {
		for j := range env[i] {
			env[i][j].creatureList = append(env[i][j].creatureList, HealthyCreature())
		}
	}

	// Place test creature at (1, 1)
	c := HighVisionHungryCreature()
	c.X = 1
	c.Y = 1

	// Add add two more creatures to 1,1
	env[1][1].creatureList = append(env[1][1].creatureList, HealthyCreature())
	env[1][1].creatureList = append(env[1][1].creatureList, HealthyCreature())

	// Empty creatures from 2,2
	env[2][2].creatureList = nil

	// Scan surroundings
	SpiralLook2D_N(env, c)

	// Sort observed cells
	SortEnvironmentListByOccupantsDesc(c.observed)

	// Decide what action to take
	intent := c.decideAction(env)
	// Expected move: (1,1) → (2,1), so dx=+1, dy=0
	if intent.Type != "move" {
		t.Errorf("Expected move action, got %s", intent.Type)
	}
	if intent.TargetX != 2 || intent.TargetY != 2 {
		t.Errorf("Expected target (2,1), got (%d,%d)", intent.TargetX, intent.TargetY)
	}
}

// Test creature moves toward location with most food
func TestCreatureMovesTowardFood(t *testing.T) {
	// Create 3x3 environment with food in a known spot
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}

	// Set food at (2, 1) to be the highest
	// env[y][x]
	env[1][2].food = 50 
	env[2][2].food = 40 
	env[1][1].food = 29
	env[0][0].food = 0

	// Place creature at (1, 1)
	c := HighVisionHungryCreature()
	c.X = 1
	c.Y = 1

	// Scan surroundings
	SpiralLook2D_N(env, c)

	// Sort or evaluate observed cells
	SortEnvironmentListByFood(c.observed)

	// Decide what action to take
	intent := c.decideAction(env)
	// Expected move: (1,1) → (2,1), so dx=+1, dy=0
	if intent.Type != "move" {
		t.Errorf("Expected move action, got %s", intent.Type)
	}
	if intent.TargetX != 2 || intent.TargetY != 1 {
		t.Errorf("Expected target (2,1), got (%d,%d)", intent.TargetX, intent.TargetY)
	}
}


func TestProcessHealth(t *testing.T) {
	// Arrange	
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}

	var tickStats TickStats
	c := FedCreature()
	c2 := StarvingCreature()
	// Act
	processHealth(c, &tickStats)
	processHealth(c2, &tickStats)

	// Assert
	if c.health != 44{
		t.Errorf("Expected creature healh of 44, got %d", c.health)
	}
	if c2.health != 90 {
		t.Errorf("Expected creature healh of 90, got %d", c2.health)
	}
}

// Test ResolveActions results in correct intent counts
func TestResolveActionIntentCounts(t *testing.T) {
	// Arrange
	intents := make([]ActionIntent, 3)
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	tickStats := &TickStats{}
	createCreature := func(parent *Creature) *Creature {
		return &Creature{}
	}

	intents = append(intents, ActionIntent{
		Creature: HealthyCreature(),
		Type: "move",
		TargetX: 1,
		TargetY: 2,
		},
	)
	intents = append(intents, ActionIntent{
		Creature: AggressiveCreature(),
		Type: "move",
		TargetX: 1,
		TargetY: 2,
		},
	)
	intents = append(intents, ActionIntent{
		Creature: FedCreature(),
		Type: "sleep",
		},
	)
	intents = append(intents, ActionIntent{
		Creature: StarvingCreature(),
		Type: "eat",
		},
	)
	intents = append(intents, ActionIntent{
		Creature: LowVisionCreature(),
		TargetCreature: HighVisionHungryCreature(),
		Type: "fight",
		},
	)
	intents = append(intents, ActionIntent{
		Creature: HighVisionCreature(),
		Type: "clone",
		},
	)

	// Act
	resolveActions(intents, env, createCreature, tickStats)

	// Assert
	if tickStats.Moves != 2 {
		t.Errorf("Expected move count of 2, but got %d", tickStats.Moves)
	}
	if tickStats.Sleeps != 1 {
		t.Errorf("Expected sleeps count of 1, but got %d", tickStats.Sleeps)
	}
	if tickStats.Eats != 1 {
		t.Errorf("Expected eats count of 1, but got %d", tickStats.Eats)
	}
	if tickStats.Fights != 2 {
		t.Errorf("Expected fights count of 2, but got %d", tickStats.Fights)
	}
	if tickStats.Clones != 1 {
		t.Errorf("Expected clones count of 1, but got %d", tickStats.Clones)
	}
}
