package main

import "testing"

func avoidanceWorld(rows, cols, x, y int) ([][]environment, *Creature) {
	env := make([][]environment, rows)
	for y := range env {
		env[y] = make([]environment, cols)
	}
	c := &Creature{num: 1, X: x, Y: y, health: 100, maxHealth: 100,
		nutrition: 30, maxNutrition: 100, age: 50, maxAge: 100,
		vision: float64(rows * cols), diet: DietHerbivore}
	env[y][x].creatureList = []*Creature{c}
	return env, c
}

func addAvoidancePredator(env [][]environment, x, y int) *Creature {
	c := &Creature{X: x, Y: y, health: 100, diet: DietCarnivore}
	env[y][x].creatureList = append(env[y][x].creatureList, c)
	return c
}

func TestHerbivoreEscapesBeforeOtherActions(t *testing.T) {
	for _, test := range []struct {
		name      string
		x, y      int
		predators [][2]int
	}{
		{"adjacent", 2, 2, [][2]int{{1, 2}}},
		{"multiple", 2, 2, [][2]int{{1, 2}, {4, 4}}},
		{"edge", 0, 2, [][2]int{{0, 3}}},
		{"shared cell", 2, 2, [][2]int{{2, 2}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Arrange: a hungry, clone-eligible herbivore on food with threats.
			env, c := avoidanceWorld(5, 5, test.x, test.y)
			env[c.Y][c.X].capacity = plantCellCapacity
			env[c.Y][c.X].plants[PlantGrass] = plantPopulation{biomass: 20, nutritionalYield: 15}
			for _, p := range test.predators {
				addAvoidancePredator(env, p[0], p[1])
			}
			if test.name == "shared cell" {
				c.vision = 0
			}

			// Act.
			intent := c.decideAction(env)
			threats := observedCarnivoreLocations(c, &env[c.Y][c.X])

			// Assert: a legal step improves distance from the closest threat.
			if intent.Type != "move" || intent.TargetX < 0 || intent.TargetX >= 5 || intent.TargetY < 0 || intent.TargetY >= 5 {
				t.Fatalf("expected in-bounds escape, got %+v", intent)
			}
			if max(absInt(intent.TargetX-c.X), absInt(intent.TargetY-c.Y)) != 1 {
				t.Fatal("escape is not a single step")
			}
			if nearestCarnivoreDistance(intent.TargetX, intent.TargetY, threats) <= nearestCarnivoreDistance(c.X, c.Y, threats) {
				t.Fatal("escape did not improve nearest-predator distance")
			}
			if again := c.decideAction(env); again != intent {
				t.Fatal("escape tie-breaking is not deterministic")
			}
			resolveMove(intent, env)
			if c.X != intent.TargetX || c.Y != intent.TargetY {
				t.Fatal("escape intent did not resolve into movement")
			}
		})
	}
}

func TestHerbivoreAvoidanceRespectsPerceptionAndImmobility(t *testing.T) {
	for _, name := range []string{"no predator", "dead predator", "unseen predator", "immobile"} {
		t.Run(name, func(t *testing.T) {
			// Arrange.
			previousRand := rng
			t.Cleanup(func() { SetRand(previousRand) })
			ResetRand(2) // Successful breeding draw, independent of other tests.
			env, c := avoidanceWorld(5, 5, 2, 2)
			if name != "no predator" {
				predator := addAvoidancePredator(env, 1, 2)
				if name == "dead predator" {
					predator.health = 0
				}
			}
			want := "clone"
			if name == "unseen predator" {
				c.vision = 1 // Only the eastern cell is observed.
			}
			if name == "immobile" {
				c.immobileTicks = 2
				want = "sleep"
			} else {
				c.nutrition = 80
				env[c.Y][c.X].capacity = plantCellCapacity
				env[c.Y][c.X].plants[PlantGrass] = plantPopulation{biomass: 50}
			}

			// Act / Assert.
			if intent := c.decideAction(env); intent.Type != want {
				t.Fatalf("expected %s, got %+v", want, intent)
			}
		})
	}
}

func TestHerbivoreDoesNotSeekFoodTowardPredatorsWhenTrapped(t *testing.T) {
	// Arrange: both neighboring steps are closer to a known predator.
	env, c := avoidanceWorld(1, 5, 2, 0)
	c.age = 10 // Not clone eligible.
	c.nutrition = 50
	addAvoidancePredator(env, 0, 0)
	addAvoidancePredator(env, 4, 0)
	env[0][3].capacity = plantCellCapacity
	env[0][3].plants[PlantGrass] = plantPopulation{biomass: 50, nutritionalYield: 15}

	// Act / Assert.
	if intent := c.decideAction(env); intent.Type != "sleep" {
		t.Fatalf("expected to stay rather than approach a predator for food, got %+v", intent)
	}
}

func TestTrappedHerbivoreCanStillEat(t *testing.T) {
	// Arrange: no neighboring cells exist, so feeding remains useful.
	env, c := avoidanceWorld(1, 1, 0, 0)
	c.age = 10
	addAvoidancePredator(env, 0, 0)
	env[0][0].capacity = plantCellCapacity
	env[0][0].plants[PlantGrass] = plantPopulation{biomass: 20, nutritionalYield: 15}

	// Act / Assert.
	if intent := c.decideAction(env); intent.Type != "eat" {
		t.Fatalf("expected feeding when no escape exists, got %+v", intent)
	}
}

func TestCriticallyHungryHerbivoreRisksCloserPredatorToReachKnownFood(t *testing.T) {
	env := make([][]environment, 1)
	env[0] = make([]environment, 5)
	for x := range env[0] {
		env[0][x].capacity = plantCellCapacity
	}
	herbivore := &Creature{X: 2, Y: 0, health: 100, maxHealth: 100, nutrition: 10, maxNutrition: 100, age: 10, maxAge: 100, vision: 5}
	predator := &Creature{X: 4, Y: 0, health: 100, diet: DietCarnivore}
	env[0][2].creatureList = []*Creature{herbivore}
	env[0][3].plants[PlantGrass] = plantPopulation{biomass: 50}
	env[0][4].creatureList = []*Creature{predator}

	intent := herbivore.decideAction(env)
	if intent.Type != "move" || intent.TargetX != 3 || intent.TargetY != 0 {
		t.Fatalf("critically hungry herbivore did not pursue known food: %+v", intent)
	}
}

func TestStarvationRiskPrioritizesEscapeOverLocalFood(t *testing.T) {
	env := make([][]environment, 1)
	env[0] = make([]environment, 5)
	for x := range env[0] {
		env[0][x].capacity = plantCellCapacity
	}
	herbivore := &Creature{X: 2, Y: 0, health: 100, maxHealth: 100, nutrition: 10, maxNutrition: 100, age: 10, maxAge: 100, vision: 5}
	predator := &Creature{X: 1, Y: 0, health: 100, diet: DietCarnivore}
	env[0][2].creatureList = []*Creature{herbivore}
	env[0][2].plants[PlantGrass] = plantPopulation{biomass: 20}
	env[0][1].creatureList = []*Creature{predator}

	intent := herbivore.decideAction(env)
	if intent.Type != "move" || intent.TargetX != 3 {
		t.Fatalf("hungry herbivore chose feeding over a safe escape: %+v", intent)
	}
}

func TestHerbivoreForagingDiscountsCrowdedFoodCells(t *testing.T) {
	env, herbivore := avoidanceWorld(1, 5, 2, 0)
	herbivore.nutrition = 50
	herbivore.num = 1
	herbivore.vision = 10
	env[0][2].plants[PlantGrass].biomass = 0
	env[0][0].plants[PlantGrass] = plantPopulation{biomass: 80}
	env[0][4].plants[PlantGrass] = plantPopulation{biomass: 100}
	for i := 0; i < 3; i++ {
		env[0][4].creatureList = append(env[0][4].creatureList, &Creature{diet: DietHerbivore, health: 100})
	}

	intent := herbivore.decideAction(env)
	if intent.Type != "move" || intent.TargetX != 1 {
		t.Fatalf("herbivore ignored per-grazer food availability: %+v", intent)
	}
}
