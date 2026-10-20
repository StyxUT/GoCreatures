package main

// Deterministic decision tests rely on test_seed.go initializing RNG with seed 1.
// These tests assert movement direction and fight triggering remain stable across runs.

import (
	"math"
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
	env[1][1].plants[PlantGrass].biomass = 5

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
	}, stats, nil)

	// Assert
	expectedNutrition := 50 - 10*3 // lost 10 during fight (3 rounds)
	if attacker.nutrition != expectedNutrition {
		t.Errorf("expected nutrition to be %d, got %d (eat should be skipped)", expectedNutrition, attacker.nutrition)
	}

	if env[1][1].plants[PlantGrass].biomass != 5 {
		t.Errorf("expected grass biomass to remain at 5 (no eating), got %.2f", env[1][1].plants[PlantGrass].biomass)
	}
}

// Test creature cloning (may disperse)
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

	intent := ActionIntent{Creature: parent, Type: "clone"}

	// Act
	resolveClone(intent, env, CreatureFactory().create)

	// Assert
	all := flattenCreatureList(env)
	if len(all) != 2 {
		t.Fatalf("expected total 2 creatures, got %d", len(all))
	}
	// Find clone
	var clone *Creature
	for _, c := range all {
		if c != parent {
			clone = c
			break
		}
	}
	if clone == nil {
		t.Fatalf("clone not found")
	}
	if clone.health != 30 {
		t.Errorf("expected clone health 30, got %d", clone.health)
	}
	if clone.nutrition != 20 {
		t.Errorf("expected offspring energy budget 20, got %d", clone.nutrition)
	}
	if parent.health != 30 {
		t.Errorf("expected parent health 30, got %d", parent.health)
	}
	if parent.nutrition != 81 {
		t.Errorf("expected parent to retain 81 nutrition after energy transfer, got %d", parent.nutrition)
	}
	if parent.immobileTicks != cloneBaseImmobileTicks {
		t.Errorf("expected parent immobile ticks %d, got %d", cloneBaseImmobileTicks, parent.immobileTicks)
	}
	if clone.immobileTicks != cloneBaseImmobileTicks {
		t.Errorf("expected clone immobile ticks %d, got %d", cloneBaseImmobileTicks, clone.immobileTicks)
	}
	if clone.X != parent.X || clone.Y != parent.Y {
		t.Errorf("expected clone to remain in birth cell, got (%d,%d) parent (%d,%d)", clone.X, clone.Y, parent.X, parent.Y)
	}
}

func TestCloneTransfersARealEnergyBudgetAndStartsCooldown(t *testing.T) {
	parent := &Creature{X: 0, Y: 0, health: 100, maxHealth: 100, nutrition: 80, maxNutrition: 100, generation: 1, ancestor: &Creature{colorCode: "x"}}
	env := [][]environment{{{creatureList: []*Creature{parent}}}}
	before := parent.nutrition
	resolveClone(ActionIntent{Creature: parent, Type: "clone"}, env, CreatureFactory().create)
	child := env[0][0].creatureList[1]
	if parent.nutrition+child.nutrition != before {
		t.Fatalf("parent/offpsring stored energy = %d+%d, want conserved total %d", parent.nutrition, child.nutrition, before)
	}
	if child.nutrition <= 0 || parent.cloneCooldown == 0 || child.cloneCooldown == 0 {
		t.Fatalf("clone energy/cooldown not assigned: parent=%+v child=%+v", parent, child)
	}
}

func TestPredatorKillCreatesFiniteCarcassInsteadOfImmediateReward(t *testing.T) {
	predator := &Creature{health: 300, maxHealth: 300, nutrition: 10, maxNutrition: 1000, diet: DietCarnivore}
	prey := &Creature{health: 1, maxHealth: 100, nutrition: 20, maxNutrition: 400, diet: DietHerbivore}
	env := [][]environment{{{creatureList: []*Creature{predator, prey}}}}
	stats := &TickStats{}
	resolveFight(ActionIntent{Creature: predator, TargetCreature: prey}, stats, nil, env)
	if predator.nutrition != 5 {
		t.Fatalf("kill awarded immediate nutrition beyond fight costs: got %d, want 5", predator.nutrition)
	}
	if got := env[0][0].carcass.energy; got <= 0 || got > float64(prey.maxNutrition)*carcassEnergyFraction {
		t.Fatalf("carcass energy %.2f is not finite prey-derived energy", got)
	}
	for range 5 {
		resolveEat(ActionIntent{Creature: predator, Type: "eat"}, env, stats)
	}
	if env[0][0].carcass.energy < 0 || predator.nutrition > predator.maxNutrition || predator.handlingTicks == 0 {
		t.Fatalf("repeated feeding duplicated energy: carcass %.2f, predator nutrition %d", env[0][0].carcass.energy, predator.nutrition)
	}
}

func TestCarcassDecaysAndHandlingTimeBlocksConsecutiveMeals(t *testing.T) {
	predator := &Creature{X: 0, Y: 0, health: 100, maxHealth: 100, nutrition: 0, maxNutrition: 1000, age: 1, maxAge: 100, diet: DietCarnivore}
	env := [][]environment{{{carcass: carcassState{energy: 100}, creatureList: []*Creature{predator}}}}
	resolveEat(ActionIntent{Creature: predator, Type: "eat"}, env, &TickStats{})
	remaining := env[0][0].carcass.energy
	if remaining != 0 || predator.nutrition != 100 || predator.handlingTicks != carnivoreHandlingDuration+1 {
		t.Fatalf("first meal energy state: carcass %.2f, nutrition %d, handling %d", remaining, predator.nutrition, predator.handlingTicks)
	}
	resolveEat(ActionIntent{Creature: predator, Type: "eat"}, env, &TickStats{})
	if predator.nutrition != 100 {
		t.Fatalf("handling time allowed duplicate meal, nutrition is %d", predator.nutrition)
	}
	env[0][0].carcass.energy = 100
	processEnvironment(env, &TickStats{}, nil)
	if math.Abs(env[0][0].carcass.energy-95) > 1e-9 {
		t.Fatalf("carcass decay = %.2f, want 95", env[0][0].carcass.energy)
	}
}

func TestCarnivoreHandlingBlocksThreeFullFollowingTicks(t *testing.T) {
	predator := &Creature{X: 0, Y: 0, health: 100, maxHealth: 100, nutrition: 0, maxNutrition: 1000, age: 1, maxAge: 100, diet: DietCarnivore}
	env := [][]environment{{{carcass: carcassState{energy: 1000}, creatureList: []*Creature{predator}}}}
	resolveEat(ActionIntent{Creature: predator, Type: "eat"}, env, &TickStats{})
	stats := &TickStats{}
	for tick := 1; tick <= carnivoreHandlingDuration; tick++ {
		simulateTick(env, func(*Creature) *Creature { return nil }, stats, nil)
		if stats.Eats != 0 || stats.Sleeps == 0 {
			t.Fatalf("handling tick %d did not suppress feeding: eats=%d sleeps=%d", tick, stats.Eats, stats.Sleeps)
		}
	}
	simulateTick(env, func(*Creature) *Creature { return nil }, stats, nil)
	if stats.Eats != 1 {
		t.Fatalf("carnivore did not resume feeding after handling duration: %+v", stats)
	}
}

func TestCarnivoreForagesToObservedCarcassAndConsumesItOverActions(t *testing.T) {
	predator := &Creature{X: 4, Y: 4, health: 100, maxHealth: 100, nutrition: 20, maxNutrition: 100, age: 10, maxAge: 100, vision: 81, diet: DietCarnivore}
	env := make([][]environment, 9)
	for y := range env {
		env[y] = make([]environment, 9)
	}
	env[4][4].creatureList = []*Creature{predator}
	env[4][6].carcass.energy = 50
	SpiralLook2D_N(env, predator)
	if found, ok := selectCarcassLocation(predator); !ok || found.X != 6 || found.Y != 4 {
		t.Fatalf("carcass not observed for foraging: found=%+v ok=%v observed=%+v", found, ok, predator.observed)
	}
	intent := predator.decideAction(env)
	if intent.Type != "move" || intent.TargetX != 5 || intent.TargetY != 4 {
		t.Fatalf("hungry carnivore did not move toward observed carcass: %+v", intent)
	}
	resolveMove(intent, env)
	intent = predator.decideAction(env)
	if intent.Type != "move" || intent.TargetX != 6 || intent.TargetY != 4 {
		t.Fatalf("carnivore did not continue toward carcass: %+v", intent)
	}
	resolveMove(intent, env)
	intent = predator.decideAction(env)
	if intent.Type != "eat" {
		t.Fatalf("carnivore at carcass did not eat: %+v", intent)
	}
	resolveEat(intent, env, &TickStats{})
	if predator.nutrition != 28 || env[4][6].carcass.energy != 40 {
		t.Fatalf("carcass feeding did not transfer finite energy: nutrition %d, carcass %.2f", predator.nutrition, env[1][2].carcass.energy)
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
	}, stats, nil)

	if attacker.health > 0 {
		t.Errorf("expected attacker to die, got health %d", attacker.health)
	}
	if stats.FightDeaths == 0 {
		t.Errorf("expected death to be recorded in FightDeaths")
	}
}

func TestResolveFight_KillDoesNotAwardNutritionBeforeCarcassFeeding(t *testing.T) {
	attacker := &Creature{health: 500, maxHealth: 500, nutrition: 10, maxNutrition: 200, diet: DietCarnivore}
	defender := &Creature{health: 5, maxHealth: 100, nutrition: 10, maxNutrition: 100, diet: DietHerbivore}
	stats := &TickStats{}

	resolveFight(ActionIntent{Creature: attacker, TargetCreature: defender}, stats, nil)

	if defender.health > 0 {
		t.Fatalf("expected defender to die, got health %d", defender.health)
	}
	expected := 10 - 5
	if attacker.nutrition != expected {
		t.Fatalf("expected attacker nutrition %d, got %d", expected, attacker.nutrition)
	}
}

func TestResolveFight_KillNutritionIsNotGrantedImmediately(t *testing.T) {
	attacker := &Creature{health: 500, maxHealth: 500, nutrition: 95, maxNutrition: 100, diet: DietCarnivore}
	defender := &Creature{health: 5, maxHealth: 100, nutrition: 10, maxNutrition: 100, diet: DietHerbivore}

	resolveFight(ActionIntent{Creature: attacker, TargetCreature: defender}, &TickStats{}, nil)

	if attacker.nutrition != 90 {
		t.Fatalf("expected only fight cost before eating carcass, got %d", attacker.nutrition)
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
	env[1][1].plants[PlantGrass].biomass = 4 // less than 5 to test partial consumption
	env[1][1].creatureList = append(env[1][1].creatureList, creature)

	intent := ActionIntent{
		Creature: creature,
		Type:     "eat",
	}

	stats := &TickStats{}

	// Act
	resolveEat(intent, env, stats)

	// Assert
	expectedNutritionGain := int(16.5 * 4)

	if creature.nutrition != expectedNutritionGain {
		t.Errorf("expected nutrition to be %d, but got %d", expectedNutritionGain, creature.nutrition)
	}
	if env[1][1].plants[PlantGrass].biomass != 0 {
		t.Errorf("expected grass biomass to be 0, but got %.2f", env[1][1].plants[PlantGrass].biomass)
	}
	if stats.BiomassConsumed != 4 {
		t.Errorf("expected BiomassConsumed to be 4, but got %.2f", stats.BiomassConsumed)
	}
}

func TestResolveEat_CarnivoreCannotEatPlantFood(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	creature := CarnivoreCreature()
	creature.nutrition = 10
	env[1][1].plants[PlantGrass].biomass = 5
	env[1][1].creatureList = []*Creature{creature}
	stats := &TickStats{}

	resolveEat(ActionIntent{Creature: creature, Type: "eat"}, env, stats)

	if creature.nutrition != 10 {
		t.Fatalf("expected carnivore nutrition unchanged, got %d", creature.nutrition)
	}
	if env[1][1].plants[PlantGrass].biomass != 5 {
		t.Fatalf("expected grass biomass unchanged, got %.2f", env[1][1].plants[PlantGrass].biomass)
	}
	if stats.BiomassConsumed != 0 {
		t.Fatalf("expected BiomassConsumed 0, got %.2f", stats.BiomassConsumed)
	}
}

// Test creature moves toward empty location (crowded dispersion branch)
func TestCreatureMovesTowardEmptyLocation(t *testing.T) {
	ResetRand(1)
	// Create 3x3 environment
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	// Creature with vision and high nutrition (avoids food-seeking branch)
	c := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 80, maxNutrition: 100, vision: 5}
	// Add additional creatures to crowd center
	env[1][1].creatureList = []*Creature{c, &Creature{}, &Creature{}}
	// Ensure an obviously empty target cell
	SpiralLook2D_N(env, c)
	intent := c.decideAction(env)
	if intent.Type != "move" {
		t.Fatalf("Expected move action, got %s", intent.Type)
	}
	if intent.TargetX == 1 && intent.TargetY == 1 {
		t.Fatalf("Expected movement away from crowded cell")
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
	env[1][2].plants[PlantGrass].biomass = 50
	env[2][2].plants[PlantGrass].biomass = 40
	env[1][1].plants[PlantGrass].biomass = 0
	env[0][0].plants[PlantGrass].biomass = 0

	// Place creature at (1, 1)
	c := HighVisionHungryCreature()
	c.X = 1
	c.Y = 1

	// Scan surroundings
	SpiralLook2D_N(env, c)

	// Sort or evaluate observed cells
	SortEnvironmentListByNutritionalValue(c.observed)

	// Decide what action to take
	intent := c.decideAction(env)
	// Expected move: (1,1) -> (2,1), so dx=+1, dy=0
	if intent.Type != "move" {
		t.Errorf("Expected move action, got %s", intent.Type)
	}
	if intent.TargetX != 2 || intent.TargetY != 1 {
		t.Errorf("Expected target (2,1), got (%d,%d)", intent.TargetX, intent.TargetY)
	}
}

func TestDecideAction_CarnivoreDoesNotChoosePlantEat(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	c := CarnivoreCreature()
	c.nutrition = 20
	c.maxNutrition = 100
	env[1][1].plants[PlantGrass].biomass = 10
	env[1][1].creatureList = []*Creature{c}

	intent := c.decideAction(env)

	if intent.Type == "eat" {
		t.Fatalf("expected carnivore not to choose plant eat")
	}
}

func TestDecideAction_CarnivorePrefersHerbivoreInSameCell(t *testing.T) {
	hunter := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 40, maxNutrition: 100, age: 0, maxAge: 100, diet: DietCarnivore, ancestor: &Creature{num: 1}}
	herbivore := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 50, maxNutrition: 100, diet: DietHerbivore, ancestor: &Creature{num: 2}}
	rivalCarnivore := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 50, maxNutrition: 100, diet: DietCarnivore, ancestor: &Creature{num: 3}}
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	env[1][1].creatureList = []*Creature{hunter, rivalCarnivore, herbivore}

	intent := hunter.decideAction(env)

	if intent.Type != "fight" {
		t.Fatalf("expected fight intent, got %s", intent.Type)
	}
	if intent.TargetCreature != herbivore {
		t.Fatalf("expected herbivore target preference in same cell")
	}
}

func TestDecideAction_CarnivoreMovesTowardHerbivorePreferredPrey(t *testing.T) {
	hunter := &Creature{X: 2, Y: 2, health: 100, maxHealth: 100, nutrition: 50, maxNutrition: 100, age: 0, maxAge: 100, vision: 20, diet: DietCarnivore, ancestor: &Creature{num: 1}, num: 11}
	herbivore := &Creature{X: 4, Y: 2, health: 100, maxHealth: 100, diet: DietHerbivore, ancestor: &Creature{num: 2}}
	rivalCarnivore := &Creature{X: 2, Y: 4, health: 100, maxHealth: 100, diet: DietCarnivore, ancestor: &Creature{num: 3}}
	env := make([][]environment, 5)
	for i := range env {
		env[i] = make([]environment, 5)
	}
	env[2][2].creatureList = []*Creature{hunter}
	env[2][4].creatureList = []*Creature{herbivore}
	env[4][2].creatureList = []*Creature{rivalCarnivore}

	intent := hunter.decideAction(env)

	if intent.Type != "move" {
		t.Fatalf("expected move intent toward prey, got %s", intent.Type)
	}
	if intent.TargetX != 3 || intent.TargetY != 2 {
		t.Fatalf("expected move toward herbivore at (4,2), got (%d,%d)", intent.TargetX, intent.TargetY)
	}
}

func TestDecideAction_CarnivoreMovesTowardRivalCarnivoreWhenNoHerbivore(t *testing.T) {
	hunterAncestor := &Creature{num: 1}
	hunter := &Creature{X: 2, Y: 2, health: 100, maxHealth: 100, nutrition: 50, maxNutrition: 100, age: 0, maxAge: 100, vision: 20, diet: DietCarnivore, ancestor: hunterAncestor, num: 5}
	sameAncestryCarnivore := &Creature{X: 4, Y: 4, health: 100, maxHealth: 100, diet: DietCarnivore, ancestor: hunterAncestor}
	rivalCarnivore := &Creature{X: 2, Y: 4, health: 100, maxHealth: 100, diet: DietCarnivore, ancestor: &Creature{num: 99}}
	env := make([][]environment, 5)
	for i := range env {
		env[i] = make([]environment, 5)
	}
	env[2][2].creatureList = []*Creature{hunter}
	env[4][4].creatureList = []*Creature{sameAncestryCarnivore}
	env[4][2].creatureList = []*Creature{rivalCarnivore}

	intent := hunter.decideAction(env)

	if intent.Type != "move" {
		t.Fatalf("expected move intent toward rival carnivore, got %s", intent.Type)
	}
	if intent.TargetX != 2 || intent.TargetY != 3 {
		t.Fatalf("expected move toward rival carnivore at (2,4), got (%d,%d)", intent.TargetX, intent.TargetY)
	}
}

func TestDecideAction_CarnivoreHighNutritionSleepsWithoutAdjacentPrey(t *testing.T) {
	env := make([][]environment, 5)
	for i := range env {
		env[i] = make([]environment, 5)
	}
	hunter := &Creature{X: 2, Y: 2, health: 100, maxHealth: 100, nutrition: 90, maxNutrition: 100, vision: 20, diet: DietCarnivore, ancestor: &Creature{num: 1}}
	distantHerbivore := &Creature{X: 4, Y: 4, health: 100, maxHealth: 60, diet: DietHerbivore, ancestor: &Creature{num: 2}}
	env[2][2].creatureList = []*Creature{hunter}
	env[4][4].creatureList = []*Creature{distantHerbivore}

	intent := hunter.decideAction(env)

	if intent.Type != "sleep" {
		t.Fatalf("expected high-nutrition carnivore to sleep, got %s", intent.Type)
	}
}

func TestDecideAction_CarnivoreAdjacentHerbivoreAttacksImmediately(t *testing.T) {
	env := make([][]environment, 5)
	for i := range env {
		env[i] = make([]environment, 5)
	}
	hunter := &Creature{X: 2, Y: 2, health: 100, maxHealth: 100, nutrition: 50, maxNutrition: 100, vision: 8, diet: DietCarnivore, ancestor: &Creature{num: 1}}
	adjacentHerbivore := &Creature{X: 3, Y: 2, health: 100, maxHealth: 60, diet: DietHerbivore, ancestor: &Creature{num: 2}}
	env[2][2].creatureList = []*Creature{hunter}
	env[2][3].creatureList = []*Creature{adjacentHerbivore}

	intent := hunter.decideAction(env)

	if intent.Type != "fight" {
		t.Fatalf("expected immediate adjacent fight, got %s", intent.Type)
	}
	if intent.TargetCreature != adjacentHerbivore {
		t.Fatalf("expected target adjacent herbivore")
	}
}

func TestDecideAction_CarnivoreSleepsWhenBreedingChanceIsRejected(t *testing.T) {
	previousRand := rng
	t.Cleanup(func() { SetRand(previousRand) })
	ResetRand(1) // Reject the nonzero breeding chance around low-value prey.
	env := make([][]environment, 5)
	for i := range env {
		env[i] = make([]environment, 5)
	}
	hunter := &Creature{X: 2, Y: 2, health: 90, maxHealth: 100, nutrition: 90, maxNutrition: 100, age: 50, maxAge: 100, vision: 8, diet: DietCarnivore, ancestor: &Creature{num: 1}}
	adjacentHerbivore := &Creature{X: 3, Y: 2, health: 100, maxHealth: 60, diet: DietHerbivore, ancestor: &Creature{num: 2}}
	env[2][2].creatureList = []*Creature{hunter}
	env[2][3].creatureList = []*Creature{adjacentHerbivore}

	intent := hunter.decideAction(env)

	if intent.Type != "sleep" {
		t.Fatalf("expected sleep after a rejected breeding draw, got %s", intent.Type)
	}
}

func TestReproductionRequiresNutritionLocalFoodAndExpiredCooldown(t *testing.T) {
	ResetRand(2)
	env := make([][]environment, 3)
	for y := range env {
		env[y] = make([]environment, 3)
		for x := range env[y] {
			env[y][x].capacity = plantCellCapacity
		}
	}
	parent := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 80, maxNutrition: 100, age: 50, maxAge: 100, vision: 8}
	env[1][1].creatureList = []*Creature{parent}
	env[1][1].plants[PlantGrass].biomass = 50
	if intent := parent.decideAction(env); intent.Type != "clone" {
		t.Fatalf("fed creature with local food did not reproduce: %+v", intent)
	}
	parent.cloneCooldown = 1
	if intent := parent.decideAction(env); intent.Type == "clone" {
		t.Fatalf("creature reproduced during cooldown: %+v", intent)
	}
	parent.cloneCooldown = 0
	env[1][1].plants[PlantGrass].biomass = 0
	if intent := parent.decideAction(env); intent.Type == "clone" {
		t.Fatalf("creature reproduced without observable food: %+v", intent)
	}
	parent.nutrition = 60
	env[1][1].plants[PlantGrass].biomass = 50
	if intent := parent.decideAction(env); intent.Type == "clone" {
		t.Fatalf("energy-poor creature reproduced: %+v", intent)
	}
	parent.nutrition = 80
	env[1][1].creatureList = append(env[1][1].creatureList, &Creature{diet: DietHerbivore, health: 100}, &Creature{diet: DietHerbivore, health: 100})
	ResetRand(2)
	if intent := parent.decideAction(env); intent.Type != "clone" {
		t.Fatalf("competition should reduce but not categorically bar breeding: %+v", intent)
	}
}

func TestHerbivoreBuildsBreedingReserveAndReproduces(t *testing.T) {
	env := [][]environment{{{capacity: plantCellCapacity, plants: [plantSpeciesCount]plantPopulation{{biomass: 1000, nutritionalYield: 15}}}}}
	parent := &Creature{X: 0, Y: 0, health: 100, maxHealth: 100, nutrition: 200, maxNutrition: 500, age: 30, maxAge: 100, diet: DietHerbivore, ancestor: &Creature{num: 1}}
	env[0][0].creatureList = []*Creature{parent}
	stats := &TickStats{}
	factory := CreatureFactory()

	for tick := 0; tick < 20 && stats.Clones == 0; tick++ {
		simulateTick(env, factory.create, stats, nil)
	}
	if stats.Clones == 0 {
		t.Fatalf("healthy herbivore did not reproduce from accessible food: nutrition=%d biomass=%.2f intent=%+v", parent.nutrition, env[0][0].plants[PlantGrass].biomass, parent.decideAction(env))
	}
}

func TestBreedingChanceRespondsGraduallyToFoodAndCompetition(t *testing.T) {
	noFood := breedingProbability(0, 1, 500)
	scarceFood := breedingProbability(250, 1, 500)
	ampleFood := breedingProbability(2500, 1, 500)
	competedFood := breedingProbability(2500, 2, 500)

	if noFood <= 0 || noFood >= 1 || ampleFood >= 1 {
		t.Fatalf("breeding chances must remain bounded and nonzero without food: none=%f ample=%f", noFood, ampleFood)
	}
	if !(noFood < scarceFood && scarceFood < ampleFood) {
		t.Fatalf("more food should gradually increase breeding chance: none=%f scarce=%f ample=%f", noFood, scarceFood, ampleFood)
	}
	if competedFood >= ampleFood {
		t.Fatalf("more competitors should reduce breeding chance: alone=%f competed=%f", ampleFood, competedFood)
	}
}

func TestHerbivoreReproductionChanceIsReducedByTenPercent(t *testing.T) {
	creature := &Creature{X: 0, Y: 0, maxNutrition: 500}
	env := [][]environment{{{plants: [plantSpeciesCount]plantPopulation{{biomass: 1000, nutritionalYield: 15}}, creatureList: []*Creature{creature}}}}
	base := breedingProbability(observablePlantValue(creature, env), observableHerbivoreCount(creature, env), creature.maxNutrition)
	if got := reproductionProbability(creature, env); math.Abs(got-base*0.9) > 1e-12 {
		t.Fatalf("herbivore reproduction chance = %.6f, want 90%% of %.6f", got, base)
	}
}

func TestReproductionCooldownBlocksConfiguredNumberOfTicks(t *testing.T) {
	ResetRand(1)
	env := [][]environment{{{capacity: plantCellCapacity, plants: [plantSpeciesCount]plantPopulation{{biomass: 100, nutritionalYield: 15}}}}}
	parent := &Creature{X: 0, Y: 0, health: 100, maxHealth: 100, nutrition: 100, maxNutrition: 100, age: 30, maxAge: 100, diet: DietHerbivore, ancestor: &Creature{num: 1}, cloneCooldown: herbivoreCloneCooldown + 1}
	env[0][0].creatureList = []*Creature{parent}
	stats := &TickStats{}
	factory := CreatureFactory()
	for tick := 1; tick <= herbivoreCloneCooldown; tick++ {
		simulateTick(env, factory.create, stats, nil)
		if stats.Clones != 0 {
			t.Fatalf("reproduced on cooldown tick %d of %d", tick, herbivoreCloneCooldown)
		}
	}
	simulateTick(env, factory.create, stats, nil)
	if stats.Clones != 1 {
		t.Fatalf("did not reproduce after cooldown expired: clones=%d cooldown=%d intent=%+v age=%d nutrition=%d plants=%.2f", stats.Clones, parent.cloneCooldown, parent.decideAction(env), parent.age, parent.nutrition, env[0][0].plants[PlantGrass].biomass)
	}
}

func TestDecideAction_CarnivoreSleepHasSecondPriority(t *testing.T) {
	env := make([][]environment, 5)
	for i := range env {
		env[i] = make([]environment, 5)
	}
	hunter := &Creature{X: 2, Y: 2, health: 100, maxHealth: 100, nutrition: 90, maxNutrition: 100, age: 10, maxAge: 100, vision: 8, diet: DietCarnivore, ancestor: &Creature{num: 1}}
	adjacentHerbivore := &Creature{X: 3, Y: 2, health: 100, maxHealth: 100, diet: DietHerbivore, ancestor: &Creature{num: 2}}
	env[2][2].creatureList = []*Creature{hunter}
	env[2][3].creatureList = []*Creature{adjacentHerbivore}

	intent := hunter.decideAction(env)

	if intent.Type != "sleep" {
		t.Fatalf("expected high-nutrition carnivore to sleep before attacking, got %s", intent.Type)
	}
}

func TestDecideAction_CarnivoreLowHealthRetreatsInsteadOfFighting(t *testing.T) {
	env := make([][]environment, 5)
	for i := range env {
		env[i] = make([]environment, 5)
	}
	hunter := &Creature{X: 2, Y: 2, health: 30, maxHealth: 100, nutrition: 50, maxNutrition: 100, vision: 8, diet: DietCarnivore, ancestor: &Creature{num: 1}, num: 9}
	adjacentHerbivore := &Creature{X: 3, Y: 2, health: 100, maxHealth: 100, diet: DietHerbivore, ancestor: &Creature{num: 2}}
	env[2][2].creatureList = []*Creature{hunter}
	env[2][3].creatureList = []*Creature{adjacentHerbivore}

	intent := hunter.decideAction(env)

	if intent.Type != "move" {
		t.Fatalf("expected low-health carnivore to retreat with move, got %s", intent.Type)
	}
	if intent.TargetX == 3 && intent.TargetY == 2 {
		t.Fatalf("expected retreat move away from adjacent prey")
	}
}

func TestDecideAction_CarnivoreLowNutritionFightsAdjacentHerbivore(t *testing.T) {
	env := make([][]environment, 5)
	for i := range env {
		env[i] = make([]environment, 5)
	}
	hunter := &Creature{X: 2, Y: 2, health: 30, maxHealth: 100, nutrition: 20, maxNutrition: 100, vision: 8, diet: DietCarnivore, ancestor: &Creature{num: 1}}
	adjacentHerbivore := &Creature{X: 3, Y: 2, health: 100, maxHealth: 100, diet: DietHerbivore, ancestor: &Creature{num: 2}}
	env[2][2].creatureList = []*Creature{hunter}
	env[2][3].creatureList = []*Creature{adjacentHerbivore}

	intent := hunter.decideAction(env)

	if intent.Type != "fight" {
		t.Fatalf("expected starving carnivore to fight adjacent herbivore, got %s", intent.Type)
	}
	if intent.TargetCreature != adjacentHerbivore {
		t.Fatalf("expected target adjacent herbivore")
	}
}

func TestDecideAction_CarnivoreLowHealthSameCellAvoidsFight(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	hunter := &Creature{X: 1, Y: 1, health: 55, maxHealth: 100, nutrition: 50, maxNutrition: 100, vision: 6, diet: DietCarnivore, ancestor: &Creature{num: 1}}
	herbivore := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, diet: DietHerbivore, ancestor: &Creature{num: 2}}
	env[1][1].creatureList = []*Creature{hunter, herbivore}

	intent := hunter.decideAction(env)

	if intent.Type == "fight" {
		t.Fatalf("expected carnivore below herbivore fight health threshold not to fight")
	}
}

func TestCanCarnivoreFightTarget_UsesHealthByTargetType(t *testing.T) {
	hunter := &Creature{diet: DietCarnivore, ancestor: &Creature{num: 1}}
	herbivore := &Creature{diet: DietHerbivore, health: 100, ancestor: &Creature{num: 2}}
	rival := &Creature{diet: DietCarnivore, health: 100, ancestor: &Creature{num: 3}}

	if canCarnivoreFightTarget(hunter, herbivore, 40, 50, false) {
		t.Fatalf("expected herbivore fight to fail at 40%% health threshold")
	}
	if canCarnivoreFightTarget(hunter, rival, 60, 50, false) {
		t.Fatalf("expected rival carnivore fight to fail at 60%% health threshold")
	}
}

func TestCanCarnivoreFightTarget_RequiresMinimumNutrition(t *testing.T) {
	hunter := &Creature{diet: DietCarnivore, ancestor: &Creature{num: 1}}
	herbivore := &Creature{diet: DietHerbivore, health: 100, ancestor: &Creature{num: 2}}
	rival := &Creature{diet: DietCarnivore, health: 100, ancestor: &Creature{num: 3}}

	if !canCarnivoreFightTarget(hunter, herbivore, 30, 20, true) {
		t.Fatalf("expected adjacent herbivore fight to succeed during starvation")
	}
	if canCarnivoreFightTarget(hunter, rival, 100, 20, false) {
		t.Fatalf("expected rival carnivore fight to fail under low nutrition")
	}
}

func TestCarnivoreWillFight_HealthExtremes(t *testing.T) {
	if carnivoreWillFight(0) {
		t.Fatalf("expected zero-health willingness to always be false")
	}
	if !carnivoreWillFight(100) {
		t.Fatalf("expected full-health willingness to always be true")
	}
}

func TestResolveFight_CarnivoreKillCreatesCarcassNotMeal(t *testing.T) {
	attacker := &Creature{health: 500, maxHealth: 500, nutrition: 10, maxNutrition: 200, diet: DietCarnivore}
	defender := &Creature{health: 5, maxHealth: 100, nutrition: 10, maxNutrition: 100, diet: DietHerbivore}
	env := [][]environment{{{creatureList: []*Creature{attacker, defender}}}}
	tickStats := &TickStats{}
	totals := &CarnivoreActionTotals{}

	resolveFight(ActionIntent{Creature: attacker, TargetCreature: defender}, tickStats, totals, env)

	if tickStats.Eats != 0 || tickStats.CarcassesCreated != 1 || tickStats.CarcassEnergyCreated != 50 {
		t.Fatalf("kill event stats are not separated from feeding: %+v", tickStats)
	}
	if totals.Eats != 0 {
		t.Fatalf("kill should not count as meal, got %d eats", totals.Eats)
	}
	if totals.Kills != 1 {
		t.Fatalf("expected carnivore totals kills to increment, got %d", totals.Kills)
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
	processHealth(c, &tickStats, nil)
	processHealth(c2, &tickStats, nil)

	// Assert
	if c.health != 80 {
		t.Errorf("Expected creature healh of 80, got %d", c.health)
	}
	if c2.health != 100 {
		t.Errorf("Expected creature healh of 100, got %d", c2.health)
	}
}

func TestProcessHealth_NutritionBandRules(t *testing.T) {
	tickStats := &TickStats{}

	high := &Creature{health: 50, maxHealth: 200, nutrition: 160, maxNutrition: 200, metabolism: 4}
	processHealth(high, tickStats, nil)
	if high.health != 70 {
		t.Fatalf("expected >70%% nutrition to heal by 5x metabolism to 70, got %d", high.health)
	}

	mid := &Creature{health: 40, maxHealth: 200, nutrition: 120, maxNutrition: 200, metabolism: 4}
	processHealth(mid, tickStats, nil)
	if mid.health != 48 {
		t.Fatalf("expected >50%% nutrition to heal by 2x metabolism to 48, got %d", mid.health)
	}

	low := &Creature{health: 30, maxHealth: 200, nutrition: 30, maxNutrition: 200, metabolism: 4}
	processHealth(low, tickStats, nil)
	if low.health != 34 {
		t.Fatalf("expected >10%% nutrition to heal by 1x metabolism to 34, got %d", low.health)
	}

	noChange := &Creature{health: 25, maxHealth: 200, nutrition: 10, maxNutrition: 200, metabolism: 4}
	processHealth(noChange, tickStats, nil)
	if noChange.health != 25 {
		t.Fatalf("expected 5-10%% nutrition band to produce no health change, got %d", noChange.health)
	}

	damage := &Creature{health: 25, maxHealth: 200, nutrition: 9, maxNutrition: 200, metabolism: 4}
	processHealth(damage, tickStats, nil)
	if damage.health != 17 {
		t.Fatalf("expected <5%% nutrition to take 2x metabolism damage to 17, got %d", damage.health)
	}
}

func TestProcessHealth_HealthClampedToMax(t *testing.T) {
	tickStats := &TickStats{}
	c := &Creature{health: 98, maxHealth: 100, nutrition: 90, maxNutrition: 100, metabolism: 4}

	processHealth(c, tickStats, nil)

	if c.health != 100 {
		t.Fatalf("expected health clamped to 100, got %d", c.health)
	}
	if tickStats.HealthAdded != 20 {
		t.Fatalf("expected healing attempt of 20 to still be recorded, got %d", tickStats.HealthAdded)
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
	mover := AggressiveCreature()
	mover.health = 100
	attacker := LowVisionCreature()
	attacker.health = 100
	defender := HighVisionHungryCreature()
	defender.health = 100

	intents = append(intents, ActionIntent{
		Creature: HealthyCreature(),
		Type:     "move",
		TargetX:  1,
		TargetY:  2,
	},
	)
	intents = append(intents, ActionIntent{
		Creature: mover,
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
		Creature:       attacker,
		TargetCreature: defender,
		Type:           "fight",
	},
	)
	intents = append(intents, ActionIntent{
		Creature: HighVisionCreature(),
		Type:     "clone",
	},
	)

	// Act
	resolveActions(intents, env, createCreature, tickStats, nil)

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

func TestResolveActions_BlocksReciprocalMoveSwaps(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}

	a := &Creature{X: 1, Y: 1, health: 100, nutrition: 100, metabolism: 10, diet: DietCarnivore}
	b := &Creature{X: 2, Y: 1, health: 100, nutrition: 100, metabolism: 10, diet: DietCarnivore}
	env[1][1].creatureList = []*Creature{a}
	env[1][2].creatureList = []*Creature{b}

	stats := &TickStats{}
	intents := []ActionIntent{
		{Creature: a, Type: "move", TargetX: 2, TargetY: 1},
		{Creature: b, Type: "move", TargetX: 1, TargetY: 1},
	}

	resolveActions(intents, env, func(parent *Creature) *Creature { return &Creature{} }, stats, nil)

	if a.X != 1 || a.Y != 1 {
		t.Fatalf("expected creature a to stay at (1,1), got (%d,%d)", a.X, a.Y)
	}
	if b.X != 2 || b.Y != 1 {
		t.Fatalf("expected creature b to stay at (2,1), got (%d,%d)", b.X, b.Y)
	}
	if len(env[1][1].creatureList) != 1 || env[1][1].creatureList[0] != a {
		t.Fatalf("expected cell (1,1) to still contain creature a")
	}
	if len(env[1][2].creatureList) != 1 || env[1][2].creatureList[0] != b {
		t.Fatalf("expected cell (2,1) to still contain creature b")
	}
	if stats.Moves != 0 {
		t.Fatalf("expected reciprocal swap moves to be blocked, got %d moves", stats.Moves)
	}
	if a.nutrition != 100 || b.nutrition != 100 {
		t.Fatalf("expected no nutrition cost for blocked reciprocal swaps, got a=%d b=%d", a.nutrition, b.nutrition)
	}
}

func TestResolveMove_BoundsCheckRejectsNegative(t *testing.T) {
	env := make([][]environment, 2)
	for i := range env {
		env[i] = make([]environment, 2)
	}

	c := &Creature{X: 0, Y: 0, health: 100, nutrition: 10, metabolism: 10}
	env[0][0].creatureList = []*Creature{c}

	resolveMove(ActionIntent{Creature: c, Type: "move", TargetX: -1, TargetY: 0}, env)

	if c.X != 0 || c.Y != 0 {
		t.Fatalf("expected creature to remain at (0,0), got (%d,%d)", c.X, c.Y)
	}
	if len(env[0][0].creatureList) != 1 || env[0][0].creatureList[0] != c {
		t.Fatalf("expected origin cell to still contain creature after invalid move")
	}
	if c.nutrition != 9 {
		t.Fatalf("expected invalid move to still pay nutrition cost, got %d", c.nutrition)
	}
}

func TestResolveMove_BoundsCheckRejectsExceedsMap(t *testing.T) {
	env := make([][]environment, 2)
	for i := range env {
		env[i] = make([]environment, 2)
	}

	c := &Creature{X: 1, Y: 1, health: 100, nutrition: 10, metabolism: 10}
	env[1][1].creatureList = []*Creature{c}

	resolveMove(ActionIntent{Creature: c, Type: "move", TargetX: 2, TargetY: 1}, env)

	if c.X != 1 || c.Y != 1 {
		t.Fatalf("expected creature to remain at (1,1), got (%d,%d)", c.X, c.Y)
	}
	if len(env[1][1].creatureList) != 1 || env[1][1].creatureList[0] != c {
		t.Fatalf("expected origin cell to still contain creature after invalid move")
	}
	if c.nutrition != 9 {
		t.Fatalf("expected invalid move to still pay nutrition cost, got %d", c.nutrition)
	}
}

func TestSelectAdjacentCarnivorePrey_OutOfBoundsSafety(t *testing.T) {
	hunter := &Creature{X: 0, Y: 0, diet: DietCarnivore, ancestor: &Creature{num: 1}}
	herbivore := &Creature{X: 1, Y: 0, health: 100, diet: DietHerbivore, ancestor: &Creature{num: 2}}
	env := make([][]environment, 2)
	for i := range env {
		env[i] = make([]environment, 2)
	}
	env[0][0].creatureList = []*Creature{hunter}
	env[0][1].creatureList = []*Creature{herbivore}

	target := selectAdjacentCarnivorePrey(hunter, env)

	if target != herbivore {
		t.Fatalf("expected edge-position hunter to find adjacent herbivore, got %#v", target)
	}
}

func TestSelectCarnivorePreyLocation_TieBreakerByCreatureNum(t *testing.T) {
	hunter := &Creature{X: 2, Y: 2, diet: DietCarnivore, ancestor: &Creature{num: 1}, num: 5}
	preyA := &Creature{health: 100, diet: DietHerbivore, ancestor: &Creature{num: 2}}
	preyB := &Creature{health: 100, diet: DietHerbivore, ancestor: &Creature{num: 3}}
	left := environment{creatureList: []*Creature{preyA}}
	right := environment{creatureList: []*Creature{preyB}}
	hunter.observed = []ObservedLocation{
		{Env: &left, X: 1, Y: 2},
		{Env: &right, X: 3, Y: 2},
	}

	chosen, ok := selectCarnivorePreyLocation(hunter)

	if !ok {
		t.Fatalf("expected tied prey candidates to produce a choice")
	}
	if chosen.X != 3 || chosen.Y != 2 {
		t.Fatalf("expected creature num tie-break to choose (3,2), got (%d,%d)", chosen.X, chosen.Y)
	}
}

func TestResolveActions_TracksCarnivoreTotals(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}

	carnivoreMover := &Creature{X: 1, Y: 1, health: 100, diet: DietCarnivore}
	carnivoreSleeper := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 50, maxNutrition: 100, diet: DietCarnivore}
	herbivoreEater := &Creature{X: 1, Y: 1, health: 100, diet: DietHerbivore}
	carnivoreAttacker := &Creature{X: 1, Y: 1, health: 100, nutrition: 100, diet: DietCarnivore}
	herbivoreDefender := &Creature{X: 1, Y: 1, health: 1, nutrition: 100, diet: DietHerbivore}

	env[1][1].plants[PlantGrass].biomass = 5
	env[1][1].creatureList = []*Creature{carnivoreMover, carnivoreSleeper, herbivoreEater, carnivoreAttacker, herbivoreDefender}

	tickStats := &TickStats{}
	totals := &CarnivoreActionTotals{}
	intents := []ActionIntent{
		{Creature: carnivoreMover, Type: "move", TargetX: 1, TargetY: 1},
		{Creature: carnivoreSleeper, Type: "sleep"},
		{Creature: herbivoreEater, Type: "eat"},
		{Creature: carnivoreAttacker, Type: "fight", TargetCreature: herbivoreDefender},
	}

	resolveActions(intents, env, func(parent *Creature) *Creature { return &Creature{} }, tickStats, totals)

	if totals.Moves != 1 {
		t.Fatalf("expected carnivore moves total 1, got %d", totals.Moves)
	}
	if totals.Sleeps != 1 {
		t.Fatalf("expected carnivore sleeps total 1, got %d", totals.Sleeps)
	}
	if totals.Fights != 1 {
		t.Fatalf("expected carnivore fights total 1, got %d", totals.Fights)
	}
	if totals.Eats != 0 {
		t.Fatalf("kill should create carcass but not count a meal, got %d", totals.Eats)
	}
	if totals.Kills != 1 {
		t.Fatalf("expected carnivore kills total 1, got %d", totals.Kills)
	}
	if totals.Clones != 0 {
		t.Fatalf("expected carnivore clones total 0, got %d", totals.Clones)
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
	c1, c2 := RelatedCreatures(90, 10)
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
	parent := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 80, maxNutrition: 100, aggression: 20, vision: 2, generation: 3, ancestor: ancestor}
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	env[parent.Y][parent.X].creatureList = []*Creature{parent}
	intent := ActionIntent{Creature: parent, Type: "clone"}
	resolveClone(intent, env, CreatureFactory().create)
	all := flattenCreatureList(env)
	if len(all) != 2 {
		t.Fatalf("expected total 2 creatures, got %d", len(all))
	}
	var clone *Creature
	for _, c := range all {
		if c != parent {
			clone = c
			break
		}
	}
	if clone == nil {
		t.Fatalf("clone not found")
	}
	if clone.generation != parent.generation+1 {
		t.Errorf("expected clone generation %d, got %d", parent.generation+1, clone.generation)
	}
	if clone.parent != parent {
		t.Errorf("expected parent pointer set on clone")
	}
	if clone.ancestor != parent.ancestor && parent.ancestor != nil {
		t.Errorf("expected ancestor propagated")
	}
	if parent.health >= 60 {
		t.Errorf("expected parent health reduced below 60, got %d", parent.health)
	}
	if parent.nutrition != 60 {
		t.Errorf("expected parent nutrition 60 after 20%% transfer, got %d", parent.nutrition)
	}
	if parent.immobileTicks != cloneBaseImmobileTicks || clone.immobileTicks != cloneBaseImmobileTicks {
		t.Errorf("expected both parent and clone immobile for %d ticks, got parent=%d clone=%d", cloneBaseImmobileTicks, parent.immobileTicks, clone.immobileTicks)
	}
	if clone.X != parent.X || clone.Y != parent.Y {
		t.Errorf("expected clone to remain in birth cell, got clone (%d,%d) parent (%d,%d)", clone.X, clone.Y, parent.X, parent.Y)
	}
}

func TestResolveCloneTransfersAnExplicitNutritionBudget(t *testing.T) {
	parent := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 95, maxNutrition: 100, generation: 2, ancestor: &Creature{colorCode: "aaa"}}
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	env[parent.Y][parent.X].creatureList = []*Creature{parent}

	resolveClone(ActionIntent{Creature: parent, Type: "clone"}, env, CreatureFactory().create)

	if parent.nutrition != 75 {
		t.Fatalf("expected parent nutrition 75 after transferring 20 energy, got %d", parent.nutrition)
	}
	if parent.immobileTicks != cloneBaseImmobileTicks {
		t.Fatalf("expected parent immobile for %d ticks, got %d", cloneBaseImmobileTicks, parent.immobileTicks)
	}
	all := flattenCreatureList(env)
	if len(all) != 2 {
		t.Fatalf("expected 2 total creatures after clone, got %d", len(all))
	}
}

func TestDecideAction_ImmobileCreatureCannotMove(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	c := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 10, maxAge: 100, nutrition: 50, maxNutrition: 100, vision: 6, immobileTicks: 2}
	env[1][1].plants[PlantGrass].biomass = 0
	env[1][2].plants[PlantGrass].biomass = 60
	env[1][1].creatureList = []*Creature{c}

	intent := c.decideAction(env)

	if intent.Type != "sleep" {
		t.Fatalf("expected immobile creature to choose a stationary action, got %s", intent.Type)
	}
}

func TestDecideAction_ImmobileCreatureCannotClone(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	c := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 50, maxAge: 100, nutrition: 80, maxNutrition: 100, immobileTicks: 1}
	env[1][1].creatureList = []*Creature{c}

	intent := c.decideAction(env)

	if intent.Type == "clone" {
		t.Fatalf("expected immobile creature not to clone")
	}
	if intent.Type != "sleep" {
		t.Fatalf("expected stationary fallback action, got %s", intent.Type)
	}
}

func TestDecideAction_ImmobileCreatureCanEat(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	c := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 50, maxAge: 100, nutrition: 30, maxNutrition: 100, immobileTicks: 2}
	env[1][1].plants[PlantGrass].biomass = 5
	env[1][1].creatureList = []*Creature{c}

	intent := c.decideAction(env)

	if intent.Type != "eat" {
		t.Fatalf("expected immobile hungry creature to eat, got %s", intent.Type)
	}
}

func TestResolveClone_PreservesDiet(t *testing.T) {
	parent := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 90, maxNutrition: 100, age: 50, maxAge: 100, diet: DietCarnivore, ancestor: &Creature{colorCode: "aaa", diet: DietCarnivore}}
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	env[1][1].creatureList = []*Creature{parent}

	resolveClone(ActionIntent{Creature: parent, Type: "clone"}, env, CreatureFactory().create)

	all := flattenCreatureList(env)
	if len(all) != 2 {
		t.Fatalf("expected 2 creatures, got %d", len(all))
	}
	for _, c := range all {
		if c != parent && c.dietType() != DietCarnivore {
			t.Fatalf("expected clone to inherit carnivore diet, got %s", c.dietType())
		}
	}
}

// abs helper for test-only distance checks
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// New dispersion tests
// Food tie-break should pick any of the best-food cells and not remain in place.
func TestDecideAction_FoodDispersionTieBreak(t *testing.T) {
	ResetRand(1)
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	c := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 10, maxAge: 200, nutrition: 50, maxNutrition: 100, aggression: 0, vision: 6, num: 1}
	env[1][1].creatureList = []*Creature{c}
	// Three adjacent best-food cells
	env[1][2].plants[PlantGrass].biomass = 60
	env[2][1].plants[PlantGrass].biomass = 60
	env[0][1].plants[PlantGrass].biomass = 60
	env[1][1].plants[PlantGrass].biomass = 10
	intent := c.decideAction(env)
	if intent.Type != "move" {
		t.Fatalf("expected move intent, got %s", intent.Type)
	}
	if intent.TargetX == 1 && intent.TargetY == 1 {
		t.Fatalf("expected movement outward toward food")
	}
	if env[intent.TargetY][intent.TargetX].plants[PlantGrass].biomass != 60 {
		t.Fatalf("expected target grass biomass 60, got %.2f", env[intent.TargetY][intent.TargetX].plants[PlantGrass].biomass)
	}
}

// Crowded tie-break should move to one of least occupied cells.
func TestDecideAction_CrowdedDispersionTieBreak(t *testing.T) {
	ResetRand(1)
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	c := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 10, maxAge: 200, nutrition: 80, maxNutrition: 100, aggression: 0, vision: 6, num: 5}
	env[1][1].creatureList = []*Creature{c, &Creature{}, &Creature{}}
	intent := c.decideAction(env)
	if intent.Type != "move" {
		t.Fatalf("expected move intent, got %s", intent.Type)
	}
	if intent.TargetX == 1 && intent.TargetY == 1 {
		t.Fatalf("did not move despite crowding")
	}
}

func TestResolveClone_SetsImmobileTicksOnParentAndClone(t *testing.T) {
	ResetRand(1)
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	parent := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 10, maxAge: 200, nutrition: 80, maxNutrition: 100, aggression: 0, vision: 1, num: 10}
	// Ensure ancestor present to avoid nil deref in CreatureFactory clone branch
	parent.ancestor = &Creature{colorCode: "aaa"}
	env[1][1].creatureList = []*Creature{parent}
	intent := ActionIntent{Creature: parent, Type: "clone"}
	resolveClone(intent, env, CreatureFactory().create)
	all := flattenCreatureList(env)
	if len(all) != 2 {
		t.Fatalf("expected total 2 creatures, got %d", len(all))
	}
	var clone *Creature
	for _, c2 := range all {
		if c2 != parent {
			clone = c2
			break
		}
	}
	if clone == nil {
		t.Fatalf("clone not found")
	}
	if parent.health >= 100 {
		t.Fatalf("expected parent health reduction, got %d", parent.health)
	}
	if parent.nutrition != 60 {
		t.Fatalf("expected parent nutrition 60 after transferring 20 energy, got %d", parent.nutrition)
	}
	if parent.immobileTicks != cloneBaseImmobileTicks || clone.immobileTicks != cloneBaseImmobileTicks {
		t.Fatalf("expected immobile ticks of %d, got parent=%d clone=%d", cloneBaseImmobileTicks, parent.immobileTicks, clone.immobileTicks)
	}
	if clone.X != parent.X || clone.Y != parent.Y {
		t.Fatalf("expected clone to remain in parent cell, got clone (%d,%d) parent (%d,%d)", clone.X, clone.Y, parent.X, parent.Y)
	}
}

func TestResolveClone_ImmobileTicksReducedByMetabolismPercent(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	parent := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 80, maxNutrition: 100, metabolism: 21, generation: 2, ancestor: &Creature{colorCode: "aaa"}}
	env[parent.Y][parent.X].creatureList = []*Creature{parent}

	resolveClone(ActionIntent{Creature: parent, Type: "clone"}, env, CreatureFactory().create)

	all := flattenCreatureList(env)
	if len(all) != 2 {
		t.Fatalf("expected total 2 creatures, got %d", len(all))
	}
	var clone *Creature
	for _, c := range all {
		if c != parent {
			clone = c
			break
		}
	}
	if clone == nil {
		t.Fatalf("clone not found")
	}

	if parent.immobileTicks != 8 {
		t.Fatalf("expected parent immobile ticks 8 for metabolism 21, got %d", parent.immobileTicks)
	}
	if clone.immobileTicks != 8 {
		t.Fatalf("expected clone immobile ticks 8 for metabolism 21, got %d", clone.immobileTicks)
	}
}

func TestResolveClone_ImmobileTicksClampedToZero(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	parent := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 80, maxNutrition: 100, metabolism: 130, generation: 2, ancestor: &Creature{colorCode: "aaa"}}
	env[parent.Y][parent.X].creatureList = []*Creature{parent}

	resolveClone(ActionIntent{Creature: parent, Type: "clone"}, env, CreatureFactory().create)

	all := flattenCreatureList(env)
	if len(all) != 2 {
		t.Fatalf("expected total 2 creatures, got %d", len(all))
	}
	var clone *Creature
	for _, c := range all {
		if c != parent {
			clone = c
			break
		}
	}
	if clone == nil {
		t.Fatalf("clone not found")
	}

	if parent.immobileTicks != 0 {
		t.Fatalf("expected parent immobile ticks clamped to 0, got %d", parent.immobileTicks)
	}
	if clone.immobileTicks != 0 {
		t.Fatalf("expected clone immobile ticks clamped to 0, got %d", clone.immobileTicks)
	}
}
