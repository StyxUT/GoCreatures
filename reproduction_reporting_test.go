package main

import (
	"strings"
	"testing"
)

func TestTickDiagnosticsCountOverlappingReproductionBlockers(t *testing.T) {
	ResetRand(44)
	creature := &Creature{
		X: 0, Y: 0, health: 60, maxHealth: 100, nutrition: 10, maxNutrition: 100,
		age: 5, maxAge: 100, cloneCooldown: 2, immobileTicks: 2, vision: 0,
	}
	env := [][]environment{{{creatureList: []*Creature{creature}}}}
	stats := &TickStats{}
	simulateTick(env, func(*Creature) *Creature { return nil }, stats, nil)

	herbivore := stats.Reproduction[herbivoreReproductionIndex]
	if herbivore.Evaluated != 1 || herbivore.Eligible != 0 {
		t.Fatalf("expected one evaluated, ineligible herbivore: %+v", herbivore)
	}
	if herbivore.NutritionBlocked != 1 || herbivore.HealthBlocked != 1 || herbivore.AgeBlocked != 1 || herbivore.CooldownBlocked != 1 || herbivore.ImmobilityBlocked != 1 {
		t.Fatalf("simultaneous hard blockers should each be counted: %+v", herbivore)
	}
}

func TestTickDiagnosticsSeparateProbabilityRejectionFromPriorityActions(t *testing.T) {
	parent := &Creature{X: 0, Y: 0, health: 100, maxHealth: 100, nutrition: 100, maxNutrition: 100, age: 30, maxAge: 100, vision: 0}
	world := [][]environment{{{creatureList: []*Creature{parent}}}}
	seed := int64(1)
	for ; seed < 10000; seed++ {
		ResetRand(seed)
		if RandFloat64() >= breedingChanceFloor {
			break
		}
	}
	ResetRand(seed)
	_ = RandFloat64() // the breeding decision's single probability draw
	wantNextRandom := RandIntn(1000)
	ResetRand(seed)
	stats := &TickStats{}
	intent := collectIntents([]*Creature{parent}, world, stats)[0]
	if got := RandIntn(1000); got != wantNextRandom {
		t.Fatalf("reproduction diagnostics changed RNG sequence: got next draw %d, want %d", got, wantNextRandom)
	}
	decision := stats.Reproduction[herbivoreReproductionIndex]
	if intent.Type == "clone" || decision.ProbabilityChecks != 1 || decision.ProbabilityRejected != 1 || decision.OtherActionSelected != 0 {
		t.Fatalf("food-starved breeding should record one chance rejection: intent=%q stats=%+v", intent.Type, decision)
	}

	carnivore := &Creature{X: 0, Y: 0, health: 100, maxHealth: 100, nutrition: 75, maxNutrition: 100, age: 30, maxAge: 100, diet: DietCarnivore}
	carrionWorld := [][]environment{{{carcass: carcassState{energy: 50}, creatureList: []*Creature{carnivore}}}}
	priorityStats := &TickStats{}
	priorityIntent := collectIntents([]*Creature{carnivore}, carrionWorld, priorityStats)[0]
	carnivoreStats := priorityStats.Reproduction[carnivoreReproductionIndex]
	if priorityIntent.Type != "eat" || carnivoreStats.Eligible != 1 || carnivoreStats.OtherActionSelected != 1 || carnivoreStats.ProbabilityChecks != 0 {
		t.Fatalf("carcass feeding should count as action priority, not chance rejection: intent=%q stats=%+v", priorityIntent.Type, carnivoreStats)
	}
}

func TestReproductionDiagnosticsSeparateSuppressedIntentFromSuccessfulBirth(t *testing.T) {
	seedForChance := func(chance float64) int64 {
		for seed := int64(1); ; seed++ {
			ResetRand(seed)
			if RandFloat64() < chance {
				return seed
			}
		}
	}
	newParent := func() *Creature {
		return &Creature{X: 0, Y: 0, health: 100, maxHealth: 100, nutrition: 100, maxNutrition: 100, age: 30, maxAge: 100, vision: 0, ancestor: &Creature{colorCode: "x"}}
	}

	parent := newParent()
	env := [][]environment{{{capacity: plantCellCapacity, creatureList: []*Creature{parent}}}}
	ResetRand(seedForChance(breedingChanceFloor))
	stats := &TickStats{}
	simulateTick(env, CreatureFactory().create, stats, nil)
	herbivore := stats.Reproduction[herbivoreReproductionIndex]
	if herbivore.CloneIntents != 1 || herbivore.Births != 1 || herbivore.ResolutionSuppressed != 0 || stats.ReproductionCumulative[herbivoreReproductionIndex].Births != 1 || stats.TotalCreatures != 2 {
		t.Fatalf("successful birth was not counted consistently: tick=%+v cumulative=%+v", herbivore, stats.ReproductionCumulative[herbivoreReproductionIndex])
	}
	simulateTick(env, CreatureFactory().create, stats, nil)
	if stats.Reproduction[herbivoreReproductionIndex].Evaluated != 2 || stats.ReproductionCumulative[herbivoreReproductionIndex].Evaluated != 3 || stats.ReproductionCumulative[herbivoreReproductionIndex].Births != 1 {
		t.Fatalf("per-tick diagnostics did not reset or cumulative diagnostics did not persist: tick=%+v cumulative=%+v", stats.Reproduction[herbivoreReproductionIndex], stats.ReproductionCumulative[herbivoreReproductionIndex])
	}

	parent = newParent()
	hunter := &Creature{X: 0, Y: 0, health: 500, maxHealth: 500, nutrition: 1000, maxNutrition: 1000, diet: DietCarnivore}
	env = [][]environment{{{capacity: plantCellCapacity, creatureList: []*Creature{parent, hunter}}}}
	ResetRand(seedForChance(breedingChanceFloor))
	stats = &TickStats{}
	cloneIntent := collectIntents([]*Creature{parent}, env, stats)[0]
	if cloneIntent.Type != "clone" {
		t.Fatalf("expected deterministic clone intent before combat, got %+v", cloneIntent)
	}
	resolveActions([]ActionIntent{
		{Creature: hunter, Type: "fight", TargetCreature: parent},
		cloneIntent,
	}, env, CreatureFactory().create, stats, nil)
	herbivore = stats.Reproduction[herbivoreReproductionIndex]
	if herbivore.CloneIntents != 1 || herbivore.ResolutionSuppressed != 1 || herbivore.Births != 0 {
		t.Fatalf("combat-suppressed reproduction misreported: %+v", herbivore)
	}
}

func TestTickReportShowsReproductionBlockersAndCumulativeBirthsByDiet(t *testing.T) {
	stats := TickStats{}
	stats.Reproduction[herbivoreReproductionIndex] = ReproductionStats{
		Evaluated: 3, Eligible: 1, NutritionBlocked: 2, HealthBlocked: 1,
		ProbabilityChecks: 1, ProbabilityRejected: 1, CloneIntents: 1,
		ResolutionSuppressed: 1,
	}
	stats.Reproduction[carnivoreReproductionIndex] = ReproductionStats{
		Evaluated: 2, Eligible: 1, AgeBlocked: 1, OtherActionSelected: 1,
		CloneIntents: 1, Births: 1,
	}
	stats.ReproductionCumulative[herbivoreReproductionIndex] = ReproductionStats{
		Evaluated: 10, Eligible: 4, NutritionBlocked: 6, ProbabilityChecks: 3,
		ProbabilityRejected: 2, OtherActionSelected: 1, CloneIntents: 1,
		ResolutionSuppressed: 1, Births: 4,
	}
	stats.ReproductionCumulative[carnivoreReproductionIndex] = ReproductionStats{
		Evaluated: 9, Eligible: 5, AgeBlocked: 4, ProbabilityChecks: 5,
		ProbabilityRejected: 2, OtherActionSelected: 2, CloneIntents: 3,
		ResolutionSuppressed: 1, Births: 5,
	}
	output := captureStdout(t, func() { printTickStats(&stats, nil) })
	for _, expected := range []string{
		"Reproduction Herbivore | tick evaluated 3 eligible 1",
		"blocked n/h/a/c/i 2/1/0/0/0",
		"chance rejected 1/1",
		"intents/suppressed/births 1/1/0",
		"total evaluated 10 eligible 4 blocked 6/0/0/0/0 chance 2/3 other 1 intents/suppressed/births 1/1/4",
		"Reproduction Carnivore | tick evaluated 2 eligible 1",
		"chance rejected 0/0 other 1 intents/suppressed/births 1/0/1",
		"total evaluated 9 eligible 5 blocked 0/0/4/0/0 chance 2/5 other 2 intents/suppressed/births 3/1/5",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("reproduction report missing %q: %s", expected, output)
		}
	}
}

func TestReproductionReportingDoesNotConsumeRandomness(t *testing.T) {
	ResetRand(20261020)
	want := RandIntn(10000)
	ResetRand(20261020)
	stats := &TickStats{}
	captureStdout(t, func() { printTickStats(stats, nil) })
	if got := RandIntn(10000); got != want {
		t.Fatalf("reporting changed the random stream: got %d, want %d", got, want)
	}
}
