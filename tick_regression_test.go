package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestSimulateTick_SeededReplay(t *testing.T) {
	// Arrange: include random wandering, births, competition, and both diets.
	replay := func() string {
		ResetRand(42)
		factory := CreatureFactory()
		env := make([][]environment, 5)
		for y := range env {
			env[y] = make([]environment, 5)
			for x := range env[y] {
				env[y][x] = environment{
					capacity: plantCellCapacity,
					plants:   [plantSpeciesCount]plantPopulation{{biomass: 5}, {biomass: 2}, {biomass: 1}},
				}
				for i := range 2 {
					diet := DietHerbivore
					if (x+y+i)%4 == 0 {
						diet = DietCarnivore
					}
					c := factory.createForDiet(diet)
					c.X, c.Y = x, y
					c.health, c.maxHealth = 60, 100
					c.nutrition, c.maxNutrition = 30, 100
					c.age, c.maxAge, c.metabolism, c.vision = 1, 100, 0, 0
					if (x+y+i)%3 == 0 {
						c.age, c.health = 30, 90
					}
					env[y][x].creatureList = append(env[y][x].creatureList, c)
				}
			}
		}
		var history strings.Builder
		var totals CarnivoreActionTotals
		for tick := 1; tick <= 20; tick++ {
			// Act: record every tick, including identities and cell order.
			stats := TickStats{Tick: tick}
			simulateTick(env, factory.create, &stats, &totals)
			fmt.Fprintf(&history, "%+v %+v\n", stats, totals)
			for _, row := range env {
				for _, cell := range row {
					fmt.Fprintf(&history, "plants=%+v capacity=%.1f\n", cell.plants, cell.capacity)
					for _, c := range cell.creatureList {
						parent := 0
						if c.parent != nil {
							parent = c.parent.num
						}
						value := *c
						value.parent, value.ancestor, value.observed = nil, nil, nil
						fmt.Fprintf(&history, "%+v parent=%d ancestor=%d\n", value, parent, c.ancestor.num)
					}
				}
			}
		}
		return history.String()
	}

	// Assert: new factories also reproduce rune allocation without global resets.
	want := replay()
	for attempt := range 3 {
		if got := replay(); got != want {
			t.Fatalf("seeded simulation diverged on replay %d", attempt)
		}
	}
}

func TestSimulateTick_DeathPrecedesActions(t *testing.T) {
	// Arrange: age takes precedence at tick entry; already-dead creatures aren't events.
	starving := &Creature{health: 1, maxHealth: 100, nutrition: 1, maxNutrition: 100, metabolism: 4, maxAge: 100, diet: DietCarnivore}
	aged := &Creature{health: 1, maxHealth: 100, age: 100, maxAge: 100, nutrition: 0, metabolism: 4}
	alreadyDead := &Creature{health: 0, age: 100, maxAge: 100}
	env := [][]environment{{{creatureList: []*Creature{starving, aged, alreadyDead}}}}
	stats := TickStats{Tick: 7, TotalHealth: 999, TotalDeaths: 999}
	var totals CarnivoreActionTotals
	noBirth := func(*Creature) *Creature { t.Fatal("dead creature attempted to clone"); return nil }

	// Act
	simulateTick(env, noBirth, &stats, &totals)

	// Assert
	if stats.Tick != 7 || stats.TotalDeaths != 2 || stats.AgeDeaths != 1 || stats.StarvationDeaths != 1 || totals.StarvationDeaths != 1 {
		t.Fatalf("unexpected death accounting: %+v / %+v", stats, totals)
	}
	if stats.TotalCreatures != 0 || stats.TotalHealth != 0 || stats.TotalNutrition != 0 || stats.TotalAge != 0 || len(env[0][0].creatureList) != 0 {
		t.Fatalf("dead creatures included in final snapshot: %+v", stats)
	}
	if stats.Eats+stats.Fights+stats.Moves+stats.Sleeps+stats.Clones != 0 {
		t.Fatalf("dead creatures acted: %+v", stats)
	}
	simulateTick(env, noBirth, &stats, &totals)
	if stats.TotalDeaths != 0 || totals.StarvationDeaths != 1 {
		t.Fatalf("death counted again: %+v / %+v", stats, totals)
	}
}

func TestSimulateTick_CompetingKillsCountOnce(t *testing.T) {
	// Arrange: both hunters target the same adjacent prey before resolution.
	hunter := func(x int) *Creature {
		return &Creature{X: x, health: 100, maxHealth: 100, nutrition: 20, maxNutrition: 100, maxAge: 100, diet: DietCarnivore}
	}
	a, b := hunter(0), hunter(2)
	prey := &Creature{X: 1, health: 1, maxHealth: 100, nutrition: 50, maxNutrition: 100, maxAge: 100, immobileTicks: 3}
	env := [][]environment{{{creatureList: []*Creature{a}}, {creatureList: []*Creature{prey}}, {creatureList: []*Creature{b}}}}
	var stats TickStats
	var totals CarnivoreActionTotals

	// Act
	simulateTick(env, CreatureFactory().create, &stats, &totals)

	// Assert
	if stats.FightDeaths != 1 || stats.TotalDeaths != 1 || totals.Kills != 1 || stats.Eats != 1 || stats.Fights != 2 || totals.Fights != 1 {
		t.Fatalf("duplicate fight credit: %+v / %+v", stats, totals)
	}
	if stats.TotalCreatures != 2 || stats.TotalCarnivores != 2 || stats.TotalHerbivores != 0 || len(env[0][1].creatureList) != 0 || b.nutrition != 20 {
		t.Fatalf("invalid survivor snapshot or second kill reward: %+v, second nutrition=%d", stats, b.nutrition)
	}
}

func TestSimulateTick_SnapshotIncludesSleepAndClone(t *testing.T) {
	// Arrange
	ResetRand(9)
	factory := CreatureFactory()
	parent := factory.createForDiet(DietHerbivore)
	parent.age, parent.maxAge = 30, 100
	parent.health, parent.maxHealth = 100, 100
	parent.nutrition, parent.maxNutrition, parent.metabolism = 90, 100, 0
	sleeper := &Creature{X: 1, health: 50, maxHealth: 100, nutrition: 90, maxNutrition: 100, metabolism: 2, maxAge: 100}
	env := [][]environment{{{creatureList: []*Creature{parent}}, {creatureList: []*Creature{sleeper}}}}
	var stats TickStats

	// Act
	simulateTick(env, factory.create, &stats, nil)

	// Assert: snapshot is after clone costs and both health updates for sleeping.
	if stats.Clones != 1 || stats.Sleeps != 1 || stats.TotalCreatures != 3 || sleeper.health != 70 {
		t.Fatalf("unexpected actions: %+v", stats)
	}
	age, health, nutrition := 0, 0, 0
	for _, c := range flattenCreatureList(env) {
		age += c.age
		health += c.health
		nutrition += c.nutrition
	}
	if stats.TotalAge != age || stats.TotalHealth != health || stats.TotalNutrition != nutrition {
		t.Fatalf("snapshot differs from surviving creatures: %+v; want age=%d health=%d nutrition=%d", stats, age, health, nutrition)
	}
	before := stats
	captureStdout(t, func() { printTickStats(&stats, nil) })
	if stats != before {
		t.Fatalf("reporting mutated simulation statistics")
	}
}
