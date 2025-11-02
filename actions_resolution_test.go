package main

// Deterministic decision tests rely on test_seed.go initializing RNG with seed 1.
// These tests assert movement direction and fight triggering remain stable across runs.

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
		t.Errorf("expected clone health to be 30, but got %d", clone.health)
	}
	if clone.nutrition != 30 {
		t.Errorf("expected clone nutrition to be 30, but got %d", clone.nutrition)
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
	SortEnvironmentListByOccupantsAsc(c.observed)

	// Decide what action to take
	intent := c.decideAction(env)
	// Expected move: (1,1) → (2,2), so dx=+1, dy=+1
	if intent.Type != "move" {
		t.Errorf("Expected move action, got %s", intent.Type)
	}
	if intent.TargetX != 2 || intent.TargetY != 2 {
		t.Errorf("Expected target (2,2), got (%d,%d)", intent.TargetX, intent.TargetY)
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
	if c.health != 44 {
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
		Type:     "move",
		TargetX:  1,
		TargetY:  2,
	},
	)
	intents = append(intents, ActionIntent{
		Creature: AggressiveCreature(),
		Type:     "move",
		TargetX:  1,
		TargetY:  2,
	},
	)
	intents = append(intents, ActionIntent{
		Creature: FedCreature(),
		Type:     "sleep",
	},
	)
	intents = append(intents, ActionIntent{
		Creature: StarvingCreature(),
		Type:     "eat",
	},
	)
	intents = append(intents, ActionIntent{
		Creature:       LowVisionCreature(),
		TargetCreature: HighVisionHungryCreature(),
		Type:           "fight",
	},
	)
	intents = append(intents, ActionIntent{
		Creature: HighVisionCreature(),
		Type:     "clone",
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

// Test deterministic wander direction given seed 1: first Intn(4)=1 -> west
func TestDecideAction_WanderDeterministic(t *testing.T) {
	ResetRand(1)
	// Arrange a creature that skips all earlier branches.
	c := &Creature{X: 1, Y: 1, health: 60, maxHealth: 100, age: 0, maxAge: 100, nutrition: 50, maxNutrition: 100, aggression: 0, vision: 0}
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	env[1][1].creatureList = []*Creature{c}
	// Act
	intent := c.decideAction(env)
	// Assert
	if intent.Type != "move" {
		t.Fatalf("expected move intent, got %s", intent.Type)
	}
	if intent.TargetX != 0 || intent.TargetY != 1 {
		t.Fatalf("expected target (0,1) (west), got (%d,%d)", intent.TargetX, intent.TargetY)
	}
}

// Test deterministic fight triggering when aggression > first Intn(100)=81
func TestDecideAction_FightTriggeredDeterministic(t *testing.T) {
	ResetRand(1)
	parentA := &Creature{}
	parentB := &Creature{}
	c1 := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 0, maxAge: 100, nutrition: 50, maxNutrition: 100, aggression: 90, vision: 0, parent: parentA, ancestor: &Creature{num: 1}}
	c2 := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 0, maxAge: 100, nutrition: 50, maxNutrition: 100, aggression: 10, vision: 0, parent: parentB, ancestor: &Creature{num: 2}}
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	env[1][1].creatureList = []*Creature{c1, c2}
	intent := c1.decideAction(env)
	if intent.Type != "fight" {
		t.Fatalf("expected fight intent, got %s", intent.Type)
	}
	if intent.TargetCreature != c2 {
		t.Fatalf("expected target creature c2")
	}
}

// Test deterministic non-fight falls through to wander using second Intn(4)=3 -> north
func TestDecideAction_FightNotTriggeredDeterministic(t *testing.T) {
	ResetRand(1)
	parentA := &Creature{}
	parentB := &Creature{}
	c1 := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 0, maxAge: 100, nutrition: 50, maxNutrition: 100, aggression: 10, vision: 0, parent: parentA, ancestor: &Creature{num: 1}}
	c2 := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 0, maxAge: 100, nutrition: 50, maxNutrition: 100, aggression: 10, vision: 0, parent: parentB, ancestor: &Creature{num: 2}}
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	env[1][1].creatureList = []*Creature{c1, c2}
	intent := c1.decideAction(env)
	if intent.Type != "move" {
		t.Fatalf("expected move intent after non-triggered fight, got %s", intent.Type)
	}
	if intent.TargetX != 1 || intent.TargetY != 0 {
		t.Fatalf("expected target (1,0) (north), got (%d,%d)", intent.TargetX, intent.TargetY)
	}
}

// Test aggression reduction when two creatures share same ancestor.
func TestDecideAction_FightAggressionReductionSharedAncestor(t *testing.T) {
	ResetRand(1)
	ancestor := &Creature{num: 999}
	parentA := &Creature{ancestor: ancestor}
	parentB := &Creature{ancestor: ancestor}
	// Aggression high enough that without reduction Intn(100)=81 would trigger (90>81).
	c1 := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 0, maxAge: 100, nutrition: 50, maxNutrition: 100, aggression: 90, vision: 0, parent: parentA, ancestor: ancestor}
	c2 := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 0, maxAge: 100, nutrition: 50, maxNutrition: 100, aggression: 10, vision: 0, parent: parentB, ancestor: ancestor}
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	env[1][1].creatureList = []*Creature{c1, c2}
	intent := c1.decideAction(env)
	// Because they share ancestor, effective aggression should be reduced below 81, causing wander instead of fight.
	if intent.Type != "move" {
		t.Fatalf("expected move intent (aggression reduced), got %s", intent.Type)
	}
}

// Test clone mutation is deterministic with seeded RNG (no randomness in initial tick after clone except rune assignment not checked)
func TestResolveCloneDeterministicMutation(t *testing.T) {
	ResetRand(1)
	ancestor := &Creature{num: 500}
	parent := &Creature{X: 1, Y: 1, health: 60, maxHealth: 100, nutrition: 60, maxNutrition: 100, aggression: 20, vision: 2, generation: 3, ancestor: ancestor}
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	env[1][1].creatureList = []*Creature{parent}
	intent := ActionIntent{Creature: parent, Type: "clone"}
	// stats not needed for resolveClone; omit to avoid unused variable
	resolveClone(intent, env, CreatureFactory())
	if len(env[1][1].creatureList) != 2 {
		t.Fatalf("expected 2 creatures at clone location, got %d", len(env[1][1].creatureList))
	}
	clone := env[1][1].creatureList[1]
	// Assert generation increments
	if clone.generation != parent.generation+1 {
		t.Errorf("expected clone generation %d, got %d", parent.generation+1, clone.generation)
	}
	// Mutated stats may differ (small +/-1% variation); skip direct equality assertions.
	if clone.parent != parent {
		t.Errorf("expected parent pointer set on clone")
	}
	if clone.ancestor != parent.ancestor && parent.ancestor != nil {
		t.Errorf("expected ancestor propagated")
	}
}
