package main

import (
	"math"
	"testing"
)

func TestSeededWorldStartsWithAllPlantSpeciesWithinSharedCapacity(t *testing.T) {
	ResetRand(41)
	env := initializeMap(1, 1, CreatureFactory())

	state, ok := PlantCellAt(env, 0, 0)
	if !ok {
		t.Fatal("expected plant state for the only cell")
	}

	biomass := 0.0
	for index, plant := range state.Plants {
		if plant.Species != PlantSpecies(index) {
			t.Fatalf("plant slot %d has species %v", index, plant.Species)
		}
		if plant.Biomass <= 0 {
			t.Errorf("species %v was not seeded", plant.Species)
		}
		biomass += plant.Biomass
	}
	if biomass > state.Capacity {
		t.Fatalf("combined biomass %.2f exceeds capacity %.2f", biomass, state.Capacity)
	}
}

func TestPlantSnapshotsAreDetachedAndEmptyMapsAreSafe(t *testing.T) {
	env := initializeMap(1, 1, CreatureFactory())
	state, ok := PlantCellAt(env, 0, 0)
	if !ok {
		t.Fatal("expected state for initialized cell")
	}
	state.Capacity = -1
	state.Plants[PlantGrass].Biomass = -1
	after, _ := PlantCellAt(env, 0, 0)
	if after.Capacity != plantCellCapacity || after.Plants[PlantGrass].Biomass < 0 {
		t.Fatalf("mutating the snapshot changed simulation state: %+v", after)
	}

	empty := initializeMap(0, 0, CreatureFactory())
	if len(empty) != 0 {
		t.Fatalf("zero-sized world has %d rows, want 0", len(empty))
	}
	if _, ok := PlantCellAt(empty, 0, 0); ok {
		t.Fatal("zero-sized world unexpectedly returned a plant cell")
	}
}

func TestPlantRegrowthRetainsFractionalGrowthWithinSharedCapacity(t *testing.T) {
	env := [][]environment{{
		{
			capacity: plantCellCapacity,
			plants: [plantSpeciesCount]plantPopulation{
				{biomass: 20},
				{biomass: 20},
				{biomass: 20},
			},
		},
	}}
	stats := &TickStats{}
	before, _ := PlantCellAt(env, 0, 0)

	processEnvironment(env, stats, nil)

	after, _ := PlantCellAt(env, 0, 0)
	for i, plant := range after.Plants {
		if plant.Biomass <= before.Plants[i].Biomass {
			t.Errorf("species %v did not regrow", plant.Species)
		}
	}
	biomass := 0.0
	for _, plant := range after.Plants {
		biomass += plant.Biomass
	}
	if biomass > after.Capacity {
		t.Fatalf("combined biomass %.4f exceeds capacity %.2f", biomass, after.Capacity)
	}
}

func TestPlantPopulationsStayWithinCapacityDuringRegrowth(t *testing.T) {
	ResetRand(7)
	env := initializeMap(4, 4, CreatureFactory())
	for tick := 0; tick < 500; tick++ {
		processEnvironment(env, &TickStats{}, nil)
		for y := range env {
			for x := range env[y] {
				state, _ := PlantCellAt(env, x, y)
				total := 0.0
				for _, plant := range state.Plants {
					if plant.Biomass < 0 {
						t.Fatalf("negative %v biomass at (%d,%d): %.8f", plant.Species, x, y, plant.Biomass)
					}
					total += plant.Biomass
				}
				if total > state.Capacity+1e-9 {
					t.Fatalf("biomass %.8f exceeds capacity %.2f at (%d,%d)", total, state.Capacity, x, y)
				}
			}
		}
	}
}

func TestHerbivoreEatsHighestYieldFirstAndTracksFractionalBiomass(t *testing.T) {
	creature := &Creature{X: 0, Y: 0, nutrition: 0, maxNutrition: 200}
	env := [][]environment{{
		{
			capacity: plantCellCapacity,
			plants: [plantSpeciesCount]plantPopulation{
				{biomass: 4.5},
				{biomass: 0.2},
				{biomass: 0.5},
			},
		},
	}}
	stats := &TickStats{}

	resolveEat(ActionIntent{Creature: creature, Type: "eat"}, env, stats)

	if got, want := env[0][0].plants[PlantClover].biomass, 0.0; got != want {
		t.Errorf("clover biomass = %.2f, want %.2f (highest yield first)", got, want)
	}
	if got, want := env[0][0].plants[PlantGrass].biomass, 0.0; got != want {
		t.Errorf("grass biomass = %.2f, want %.2f", got, want)
	}
	if got, want := env[0][0].plants[PlantBramble].biomass, 0.2; math.Abs(got-want) > 1e-9 {
		t.Errorf("bramble biomass = %.2f, want %.2f (lower-yield food is consumed last)", got, want)
	}
	if got, want := stats.BiomassConsumed, 5.0; got != want {
		t.Errorf("biomass consumed = %.2f, want %.2f", got, want)
	}
	if creature.nutrition != 74 || creature.nutritionProgress != 0.5 {
		t.Errorf("nutrition progress = %d + %.2f, want 74 + 0.50", creature.nutrition, creature.nutritionProgress)
	}
}

func TestHerbivoreUsesStableSpeciesOrderForEqualYields(t *testing.T) {
	creature := &Creature{X: 0, Y: 0, maxNutrition: 1000}
	env := [][]environment{{
		{
			plants: [plantSpeciesCount]plantPopulation{
				{biomass: 5, nutritionalYield: 20},
				{biomass: 5, nutritionalYield: 20},
				{biomass: 5, nutritionalYield: 10},
			},
		},
	}}
	resolveEat(ActionIntent{Creature: creature, Type: "eat"}, env, &TickStats{})
	if env[0][0].plants[PlantGrass].biomass != 0 || env[0][0].plants[PlantClover].biomass != 5 {
		t.Fatalf("equal-yield species tie did not follow stable order: grass=%.2f clover=%.2f", env[0][0].plants[PlantGrass].biomass, env[0][0].plants[PlantClover].biomass)
	}
}

func TestHerbivoreSeeksGreatestTotalNutritionalValue(t *testing.T) {
	env := make([][]environment, 3)
	for y := range env {
		env[y] = make([]environment, 3)
		for x := range env[y] {
			env[y][x].capacity = plantCellCapacity
		}
	}
	// Clover is worth more nutritionally despite having less biomass than grass.
	env[1][2].plants[PlantClover].biomass = 16 // value 320
	env[2][1].plants[PlantGrass].biomass = 21  // value 315
	creature := &Creature{
		X: 1, Y: 1, health: 100, maxHealth: 100,
		nutrition: 50, maxNutrition: 100, maxAge: 100, vision: 8,
	}
	env[1][1].creatureList = []*Creature{creature}

	intent := creature.decideAction(env)

	if intent.Type != "move" || intent.TargetX != 2 || intent.TargetY != 1 {
		t.Fatalf("expected movement toward higher nutritional value at (2,1), got %+v", intent)
	}
}

func TestHerbivoreEatingCapsNutritionButAccountsForPartialBiomass(t *testing.T) {
	creature := &Creature{X: 0, Y: 0, nutrition: 95, maxNutrition: 100}
	env := [][]environment{{
		{plants: [plantSpeciesCount]plantPopulation{{}, {}, {biomass: 0.6}}},
	}}
	stats := &TickStats{}

	resolveEat(ActionIntent{Creature: creature, Type: "eat"}, env, stats)

	if creature.nutrition != 100 || creature.nutritionProgress != 0 {
		t.Fatalf("nutrition = %d + %.2f, want capped at 100", creature.nutrition, creature.nutritionProgress)
	}
	if math.Abs(stats.BiomassConsumed-0.6) > 1e-9 || env[0][0].plants[PlantBramble].biomass != 0 {
		t.Fatalf("partial biomass accounting mismatch: consumed %.3f, remaining %.3f", stats.BiomassConsumed, env[0][0].plants[PlantBramble].biomass)
	}
}

func TestZeroCapacityCellDoesNotRegrow(t *testing.T) {
	cell := environment{plants: [plantSpeciesCount]plantPopulation{{biomass: 0.5}}}
	added := regrowPlants(&cell)
	if added != 0 || cell.plants[PlantGrass].biomass != 0.5 {
		t.Fatalf("zero-capacity cell regrew: added %.2f, biomass %.2f", added, cell.plants[PlantGrass].biomass)
	}
}

func TestPlantDoesNotRegrowWithoutALocalPopulation(t *testing.T) {
	cell := environment{capacity: plantCellCapacity}
	stats := &TickStats{}
	processEnvironment([][]environment{{cell}}, stats, nil)
	if stats.BiomassAdded != 0 {
		t.Fatalf("empty cell gained %.4f biomass without a local population", stats.BiomassAdded)
	}
}

func TestPlantColoniesSpreadLocallyWithoutSameTickCascade(t *testing.T) {
	spreadObserved := false
	for seed := int64(0); seed < 1000 && !spreadObserved; seed++ {
		ResetRand(seed)
		env := make([][]environment, 5)
		for y := range env {
			env[y] = make([]environment, 5)
			for x := range env[y] {
				env[y][x].capacity = plantCellCapacity
			}
		}
		env[2][2].plants[PlantGrass] = plantPopulation{biomass: 20}
		stats := &TickStats{}
		processEnvironment(env, stats, nil)

		spreadCells := 0
		for y := range env {
			for x := range env[y] {
				if env[y][x].plants[PlantGrass].biomass > 0 && (x != 2 || y != 2) {
					spreadCells++
					if abs(x-2)+abs(y-2) != 1 {
						t.Fatalf("grass spread beyond a cardinal neighbor: (%d,%d)", x, y)
					}
				}
			}
		}
		if spreadCells > 1 {
			t.Fatalf("one tick cascaded into %d new colonies", spreadCells)
		}
		if spreadCells == 1 {
			if stats.BiomassTransferred[PlantGrass] != 1 {
				t.Fatalf("spread transfer = %.2f, want 1 biomass", stats.BiomassTransferred[PlantGrass])
			}
			spreadObserved = true
		}
	}
	if !spreadObserved {
		t.Fatal("grass did not spread with any of 1000 deterministic seeds")
	}
}

func TestEachPlantSpeciesUsesItsDocumentedSpreadNeighborhood(t *testing.T) {
	observed := [plantSpeciesCount]bool{}
	for seed := int64(0); seed < 2000; seed++ {
		ResetRand(seed)
		env := make([][]environment, 7)
		for y := range env {
			env[y] = make([]environment, 7)
			for x := range env[y] {
				env[y][x].capacity = plantCellCapacity
			}
		}
		for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
			env[3][3].plants[species] = plantPopulation{biomass: plantSpread[species].threshold}
		}
		processEnvironment(env, &TickStats{}, nil)
		for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
			if observed[species] {
				continue
			}
			for y := range env {
				for x := range env[y] {
					if env[y][x].plants[species].biomass != 1 || (x == 3 && y == 3) {
						continue
					}
					dx, dy := abs(x-3), abs(y-3)
					switch species {
					case PlantGrass:
						if dx+dy != 1 {
							t.Fatalf("grass spread to (%d,%d), outside cardinal neighborhood", x, y)
						}
					case PlantClover:
						if max(dx, dy) != 1 {
							t.Fatalf("clover spread to (%d,%d), outside eight neighbors", x, y)
						}
					case PlantBramble:
						if max(dx, dy) > 2 {
							t.Fatalf("bramble spread to (%d,%d), beyond distance two", x, y)
						}
					}
					observed[species] = true
				}
			}
		}
		if observed[PlantGrass] && observed[PlantClover] && observed[PlantBramble] {
			return
		}
	}
	t.Fatalf("did not observe each species spreading: %v", observed)
}

func TestTickStatisticsSnapshotVegetationState(t *testing.T) {
	env := [][]environment{{
		{
			capacity: plantCellCapacity,
			plants: [plantSpeciesCount]plantPopulation{
				{biomass: 10}, {biomass: 5}, {biomass: 2},
			},
		},
	}}
	stats := &TickStats{}
	simulateTick(env, func(*Creature) *Creature { return nil }, stats, nil)
	state, _ := PlantCellAt(env, 0, 0)

	if stats.PlantCapacity != state.Capacity {
		t.Fatalf("reported capacity %.2f, cell capacity %.2f", stats.PlantCapacity, state.Capacity)
	}
	for i, plant := range state.Plants {
		if stats.PlantBiomass[i] != plant.Biomass {
			t.Errorf("reported species %v biomass %.4f, cell biomass %.4f", plant.Species, stats.PlantBiomass[i], plant.Biomass)
		}
	}
}
