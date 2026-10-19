package main

import (
	"math"
	"math/rand"
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
		traits := plantTraits[index]
		if plant.Generation != 0 || plant.RegrowthRate != traits.regrowth || plant.SpreadProbability != traits.spreadProbability || plant.NutritionalYield != traits.nutrition {
			t.Errorf("founder species %v has non-baseline state: %+v", plant.Species, plant)
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

func TestNewPlantColonyInheritsItsParentsTraitsAndGeneration(t *testing.T) {
	defer ResetRand(1)
	SetRand(rand.New(&scriptedPlantRandom{values: []int64{0, 0, 0, 1 << 62}}))
	env := make([][]environment, 3)
	for y := range env {
		env[y] = make([]environment, 3)
		for x := range env[y] {
			env[y][x].capacity = plantCellCapacity
		}
	}
	parent := plantPopulation{
		biomass: 25, generation: 4, regrowthRate: 0.083,
		spreadProbability: 1, nutritionalYield: 15.25,
	}
	env[1][1].plants[PlantGrass] = parent

	processEnvironment(env, &TickStats{}, nil)

	child, ok := PlantCellAt(env, 2, 1)
	if !ok || !child.Plants[PlantGrass].Present {
		t.Fatal("expected grass colony to establish at its selected neighbor")
	}
	colony := child.Plants[PlantGrass]
	if colony.Generation != parent.generation+1 {
		t.Fatalf("daughter generation = %d, want %d", colony.Generation, parent.generation+1)
	}
	if colony.RegrowthRate != parent.regrowthRate || colony.SpreadProbability != parent.spreadProbability || colony.NutritionalYield != parent.nutritionalYield {
		t.Fatalf("daughter did not inherit parent traits: %+v", colony)
	}
}

func TestPlantEstablishmentMutatesAtMostOneTraitWithinBounds(t *testing.T) {
	defer ResetRand(1)
	SetRand(rand.New(&scriptedPlantRandom{values: []int64{0, 0, 0, 0, 0, 0, (1 << 63) - 1 - (1 << 53)}}))
	env := make([][]environment, 3)
	for y := range env {
		env[y] = make([]environment, 3)
		for x := range env[y] {
			env[y][x].capacity = plantCellCapacity
		}
	}
	parent := plantPopulation{
		biomass: 25, regrowthRate: 0.08, spreadProbability: 1, nutritionalYield: 15,
	}
	env[1][1].plants[PlantGrass] = parent

	processEnvironment(env, &TickStats{}, nil)

	daughter, _ := PlantCellAt(env, 2, 1)
	child := daughter.Plants[PlantGrass]
	changed := 0
	if child.RegrowthRate != parent.regrowthRate {
		changed++
		relative := math.Abs(child.RegrowthRate/parent.regrowthRate - 1)
		if relative > 0.0025+1e-12 {
			t.Fatalf("regrowth mutation changed %.5f relative, limit is 0.0025", relative)
		}
	}
	if child.SpreadProbability != parent.spreadProbability {
		changed++
	}
	if child.NutritionalYield != parent.nutritionalYield {
		changed++
	}
	if changed != 1 {
		t.Fatalf("mutation changed %d traits, want exactly one: %+v", changed, child)
	}
	if child.RegrowthRate < plantTraits[PlantGrass].regrowth*0.8 || child.RegrowthRate > plantTraits[PlantGrass].regrowth*1.2 {
		t.Fatalf("regrowth mutation outside species bounds: %.8f", child.RegrowthRate)
	}
	if got := env[1][1].plants[PlantGrass]; got.regrowthRate != parent.regrowthRate || got.spreadProbability != parent.spreadProbability || got.nutritionalYield != parent.nutritionalYield {
		t.Fatalf("daughter mutation changed its parent cohort: %+v", got)
	}
	base := plantTraits[PlantGrass]
	if base.regrowth != 0.08 || base.spreadProbability != 0.02 || base.nutrition != 15 {
		t.Fatalf("daughter mutation changed species baselines: %+v", base)
	}
}

func TestEachPlantTraitMutationStaysInsideSpeciesBounds(t *testing.T) {
	defer ResetRand(1)
	for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
		base := plantTraits[species]
		baseValues := []float64{base.regrowth, base.spreadProbability, base.nutrition}
		for traitIndex := 0; traitIndex < len(baseValues); traitIndex++ {
			minimum, maximum := baseValues[traitIndex]*0.8, baseValues[traitIndex]*1.2
			for _, test := range []struct {
				name  string
				start float64
				draw  int64
			}{
				{name: "lower bound", start: minimum, draw: (1 << 63) - 1 - (1 << 53)},
				{name: "upper bound", start: maximum, draw: 0},
			} {
				traits := plantPopulation{
					regrowthRate: base.regrowth, spreadProbability: base.spreadProbability,
					nutritionalYield: base.nutrition,
				}
				switch traitIndex {
				case 0:
					traits.regrowthRate = test.start
				case 1:
					traits.spreadProbability = test.start
				case 2:
					traits.nutritionalYield = test.start
				}
				SetRand(rand.New(&scriptedPlantRandom{values: []int64{
					0, 0, 0, 0, int64(traitIndex) << 32, test.draw,
				}}))
				env := make([][]environment, 5)
				for y := range env {
					env[y] = make([]environment, 5)
					for x := range env[y] {
						env[y][x].capacity = plantCellCapacity
					}
				}
				traits.biomass = plantSpread[species].threshold + 1
				traits.spreadProbability = 1
				// For this trait, test the cohort's actual bounded probability.
				if traitIndex == 1 {
					traits.spreadProbability = test.start
				}
				sourceX, sourceY := 2, 2
				env[sourceY][sourceX].plants[species] = traits
				processEnvironment(env, &TickStats{}, nil)
				dx, dy := plantSpread[species].offsets[0][0], plantSpread[species].offsets[0][1]
				child := env[sourceY+dy][sourceX+dx].plants[species]
				value := []float64{child.regrowthRate, child.spreadProbability, child.nutritionalYield}[traitIndex]
				if value < minimum || value > maximum {
					t.Errorf("%v trait %d %s mutation produced %.8f outside [%.8f, %.8f]", species, traitIndex, test.name, value, minimum, maximum)
				}
				if value == test.start || math.Abs(value/test.start-1) > plantMutationMagnitude+1e-12 {
					t.Errorf("%v trait %d %s mutation exceeded limit: %.8f -> %.8f", species, traitIndex, test.name, test.start, value)
				}
			}
		}
	}
}

func TestColonyTraitsAffectFoodSeekingEatingAndRegrowth(t *testing.T) {
	env := make([][]environment, 3)
	for y := range env {
		env[y] = make([]environment, 3)
		for x := range env[y] {
			env[y][x].capacity = plantCellCapacity
		}
	}
	// This inherited cohort is more nutritious and grows faster than baseline grass.
	env[1][2].plants[PlantGrass] = plantPopulation{
		biomass: 10, regrowthRate: 0.16, spreadProbability: 0.02, nutritionalYield: 30,
	}
	env[2][1].plants[PlantGrass] = plantPopulation{
		biomass: 19, regrowthRate: 0.08, spreadProbability: 0.02, nutritionalYield: 15,
	}
	creature := &Creature{
		X: 1, Y: 1, health: 100, maxHealth: 100,
		nutrition: 500, maxNutrition: 1000, maxAge: 100, vision: 8,
	}
	intent := creature.decideAction(env)
	if intent.Type != "move" || intent.TargetX != 2 || intent.TargetY != 1 {
		t.Fatalf("food seeking ignored local nutritional yield: %+v", intent)
	}

	creature.X, creature.Y = 2, 1
	resolveEat(ActionIntent{Creature: creature, Type: "eat"}, env, &TickStats{})
	if creature.nutrition != 650 {
		t.Fatalf("eating gained %d nutrition, want 150 on top of 500", creature.nutrition)
	}
	before := env[1][2].plants[PlantGrass].biomass
	processEnvironment(env, &TickStats{}, nil)
	if got := env[1][2].plants[PlantGrass].biomass - before; got <= env[2][1].plants[PlantGrass].biomass-19 {
		t.Fatalf("higher regrowth trait did not affect growth: high=%.4f, baseline=%.4f", got, env[2][1].plants[PlantGrass].biomass-19)
	}
}

func TestInheritedSpreadProbabilityChangesEstablishmentOutcome(t *testing.T) {
	defer ResetRand(1)
	// This draw is just above the grass baseline and just below a +0.25% trait.
	chanceDraw := int64(0.020025 * float64(uint64(1)<<63))
	spread := func(probability float64) float64 {
		SetRand(rand.New(&scriptedPlantRandom{values: []int64{chanceDraw, 0, 0, 1 << 62}}))
		env := [][]environment{{
			{capacity: plantCellCapacity, plants: [plantSpeciesCount]plantPopulation{{
				biomass: 20, regrowthRate: 0.08, spreadProbability: probability, nutritionalYield: 15,
			}}},
			{capacity: plantCellCapacity},
		}}
		stats := &TickStats{}
		processEnvironment(env, stats, nil)
		return stats.BiomassTransferred[PlantGrass]
	}

	baselineTransfers := spread(plantTraits[PlantGrass].spreadProbability)
	mutatedTransfers := spread(plantTraits[PlantGrass].spreadProbability * 1.0025)
	if baselineTransfers != 0 || mutatedTransfers != 1 {
		t.Fatalf("spread probability did not change the seeded outcome: baseline %.0f, mutated %.0f", baselineTransfers, mutatedTransfers)
	}
}

func TestFailedEstablishmentAndOrdinaryRegrowthDoNotMutateColonies(t *testing.T) {
	defer ResetRand(1)
	parent := plantPopulation{
		biomass: 25, generation: 3, regrowthRate: 0.081,
		spreadProbability: 1, nutritionalYield: 15.1,
	}
	env := make([][]environment, 3)
	for y := range env {
		env[y] = make([]environment, 3)
		for x := range env[y] {
			env[y][x].capacity = plantCellCapacity
		}
	}
	env[1][1].plants[PlantGrass] = parent
	env[1][2].plants[PlantGrass] = plantPopulation{
		biomass: 2, generation: 7, regrowthRate: 0.079,
		spreadProbability: 0.021, nutritionalYield: 14.9,
	}
	SetRand(rand.New(&scriptedPlantRandom{values: []int64{0, 0, 1 << 62}}))
	stats := &TickStats{}
	processEnvironment(env, stats, nil)
	if stats.BiomassTransferred[PlantGrass] != 0 {
		t.Fatalf("existing same-species colony was overwritten: %+v", stats.BiomassTransferred)
	}
	if got := env[1][1].plants[PlantGrass]; got.generation != parent.generation || got.regrowthRate != parent.regrowthRate || got.spreadProbability != parent.spreadProbability || got.nutritionalYield != parent.nutritionalYield {
		t.Fatalf("failed establishment mutated parent traits: %+v", got)
	}
	if got := env[1][2].plants[PlantGrass]; got.generation != 7 || got.regrowthRate != 0.079 || got.spreadProbability != 0.021 || got.nutritionalYield != 14.9 {
		t.Fatalf("existing colony traits changed during ordinary regrowth: %+v", got)
	}

	regrowthOnly := environment{
		capacity: plantCellCapacity,
		plants: [plantSpeciesCount]plantPopulation{{
			biomass: 5, generation: 2, regrowthRate: 0.087,
			spreadProbability: 0.022, nutritionalYield: 15.3,
		}},
	}
	SetRand(rand.New(&scriptedPlantRandom{}))
	regrowthEnv := [][]environment{{regrowthOnly}}
	processEnvironment(regrowthEnv, &TickStats{}, nil)
	if got := regrowthEnv[0][0].plants[PlantGrass]; got.generation != 2 || got.regrowthRate != 0.087 || got.spreadProbability != 0.022 || got.nutritionalYield != 15.3 {
		t.Fatalf("ordinary regrowth changed plant traits: %+v", got)
	}
}

func TestExtinctColonyLosesItsTraitsAndRecolonizesFromNewParent(t *testing.T) {
	defer ResetRand(1)
	creature := &Creature{X: 0, Y: 0, maxNutrition: 1000}
	env := [][]environment{{
		{capacity: plantCellCapacity, plants: [plantSpeciesCount]plantPopulation{{
			biomass: 0.5, generation: 7, regrowthRate: 0.081,
			spreadProbability: 0.021, nutritionalYield: 15.1,
		}}},
		{capacity: plantCellCapacity},
	}}
	resolveEat(ActionIntent{Creature: creature, Type: "eat"}, env, &TickStats{})
	state, _ := PlantCellAt(env, 0, 0)
	grass := state.Plants[PlantGrass]
	if grass.Present || grass.Biomass != 0 || grass.Generation != 0 {
		t.Fatalf("extinct colony still has local state: %+v", grass)
	}
	if grass.RegrowthRate != plantTraits[PlantGrass].regrowth || grass.SpreadProbability != plantTraits[PlantGrass].spreadProbability || grass.NutritionalYield != plantTraits[PlantGrass].nutrition {
		t.Fatalf("extinct colony retained non-founder traits: %+v", grass)
	}

	parent := plantPopulation{
		biomass: 25, generation: 2, regrowthRate: 0.09,
		spreadProbability: 1, nutritionalYield: 16,
	}
	env[0][1].plants[PlantGrass] = parent
	SetRand(rand.New(&scriptedPlantRandom{values: []int64{0, 1 << 32, 0, 1 << 62}}))
	processEnvironment(env, &TickStats{}, nil)
	recolonized, _ := PlantCellAt(env, 0, 0)
	child := recolonized.Plants[PlantGrass]
	if !child.Present || child.Generation != parent.generation+1 || child.RegrowthRate != parent.regrowthRate || child.SpreadProbability != parent.spreadProbability || child.NutritionalYield != parent.nutritionalYield {
		t.Fatalf("recolonized plant did not inherit new parent's traits: %+v", child)
	}
}

func TestSeededPlantMutationRateIsRareRelativeToCreatureCloning(t *testing.T) {
	defer ResetRand(1)
	ResetRand(20261019)
	parent := plantPopulation{
		regrowthRate:      plantTraits[PlantGrass].regrowth,
		spreadProbability: plantTraits[PlantGrass].spreadProbability,
		nutritionalYield:  plantTraits[PlantGrass].nutrition,
	}
	const establishments = 100_000
	mutations := 0
	for range establishments {
		child := inheritPlantColony(parent, PlantGrass)
		changes := 0
		for _, pair := range [][2]float64{
			{parent.regrowthRate, child.regrowthRate},
			{parent.spreadProbability, child.spreadProbability},
			{parent.nutritionalYield, child.nutritionalYield},
		} {
			if pair[0] != pair[1] {
				changes++
				if math.Abs(pair[1]/pair[0]-1) > plantMutationMagnitude+1e-12 {
					t.Fatalf("seeded mutation exceeded magnitude: %v -> %v", pair[0], pair[1])
				}
			}
		}
		if changes > 1 {
			t.Fatalf("one establishment changed %d traits", changes)
		}
		if changes == 1 {
			mutations++
		}
	}
	if mutations < 50 || mutations > 150 {
		t.Fatalf("seeded mutation sample = %d/%d, outside the expected rare range", mutations, establishments)
	}
	t.Logf("seeded sample: %d mutations per %d successful establishments (%.3f%%); creature clones perturb five continuous traits by about ±1%% per clone", mutations, establishments, 100*float64(mutations)/establishments)
}

type scriptedPlantRandom struct {
	values []int64
	index  int
}

func (s *scriptedPlantRandom) Int63() int64 {
	if s.index >= len(s.values) {
		return 1 << 62
	}
	value := s.values[s.index]
	s.index++
	return value
}

func (s *scriptedPlantRandom) Seed(int64) {}

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
