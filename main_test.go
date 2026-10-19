package main

import (
	"sync/atomic"
	"testing"
	"time"
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
	processEnvironment(env, &tickStats, nil)
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

func TestSimulateTick_DecrementsImmobileTicks(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	creature := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 50, maxNutrition: 100, age: 10, maxAge: 100, immobileTicks: 3}
	env[1][1].creatureList = []*Creature{creature}

	var stats TickStats
	simulateTick(env, func(parent *Creature) *Creature { return &Creature{} }, &stats, nil)

	if creature.immobileTicks != 2 {
		t.Fatalf("expected immobile ticks to decrement to 2, got %d", creature.immobileTicks)
	}
}

func TestAdjustTickIntervalClampsWithinBounds(t *testing.T) {
	atomic.StoreInt64(&tickIntervalMillis, 300)

	fast := clampTickInterval(10 * time.Millisecond)
	if fast != minTickInterval {
		t.Fatalf("expected lower clamp %v, got %v", minTickInterval, fast)
	}

	slow := clampTickInterval(2 * time.Second)
	if slow != maxTickInterval {
		t.Fatalf("expected upper clamp %v, got %v", maxTickInterval, slow)
	}
}

func TestSimulateTick_TracksCarnivoreAndHerbivoreCounts(t *testing.T) {
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	carnivore := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 50, maxNutrition: 100, age: 10, maxAge: 100, diet: DietCarnivore}
	herbivore := &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, nutrition: 50, maxNutrition: 100, age: 10, maxAge: 100, diet: DietHerbivore}
	env[1][1].creatureList = []*Creature{carnivore, herbivore}

	var stats TickStats
	simulateTick(env, func(parent *Creature) *Creature { return &Creature{} }, &stats, nil)

	if stats.TotalCarnivores != 1 {
		t.Fatalf("expected 1 carnivore, got %d", stats.TotalCarnivores)
	}
	if stats.TotalHerbivores != 1 {
		t.Fatalf("expected 1 herbivore, got %d", stats.TotalHerbivores)
	}
}

func TestCreateCreatureForDiet_HerbivoreNeverHasCarnivoreBoostedStats(t *testing.T) {
	ResetRand(1)
	create := CreatureFactory()

	for range 200 {
		c := create.createForDiet(DietHerbivore)
		if c == nil {
			t.Fatalf("expected herbivore creature")
		}
		if c.dietType() != DietHerbivore {
			t.Fatalf("expected herbivore diet, got %s", c.dietType())
		}
		if c.maxHealth > 499 {
			t.Fatalf("expected herbivore maxHealth <= 499, got %d", c.maxHealth)
		}
		if c.vision > 25 {
			t.Fatalf("expected herbivore vision <= 25, got %.2f", c.vision)
		}
	}
}

func TestSpawnCreatureWithDiet_HerbivoreNeverHasCarnivoreBoostedStats(t *testing.T) {
	ResetRand(1)
	env := make([][]environment, 3)
	for i := range env {
		env[i] = make([]environment, 3)
	}
	create := CreatureFactory()

	for range 100 {
		spawnCreatureWithDiet(env, create, DietHerbivore)
	}
	if got := len(flattenCreatureList(env)); got != 100 {
		t.Fatalf("expected 100 births, got %d", got)
	}

	for _, c := range flattenCreatureList(env) {
		if c.dietType() != DietHerbivore {
			t.Fatalf("expected requested herbivore diet, got %s", c.dietType())
		}
		if c.maxHealth > 499 {
			t.Fatalf("expected spawned herbivore maxHealth <= 499, got %d", c.maxHealth)
		}
		if c.vision > 25 {
			t.Fatalf("expected spawned herbivore vision <= 25, got %.2f", c.vision)
		}
	}
}

func TestInitializeMap_HerbivoreOnlySpawnNeverHasCarnivoreBoostedStats(t *testing.T) {
	ResetRand(1)
	create := CreatureFactory()
	env := initializeMap(80, 80, create)

	for _, c := range flattenCreatureList(env) {
		if c.dietType() != DietHerbivore {
			continue
		}
		if c.maxHealth > 499 {
			t.Fatalf("expected initialized herbivore maxHealth <= 499, got %d", c.maxHealth)
		}
		if c.vision > 25 {
			t.Fatalf("expected initialized herbivore vision <= 25, got %.2f", c.vision)
		}
	}
}
