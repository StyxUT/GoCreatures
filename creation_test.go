package main

import "testing"

func TestCreatureCreation_RequestedDietAndAllocation(t *testing.T) {
	// Arrange
	ResetRand(11)
	factory := CreatureFactory()

	// Act / Assert: alternate diets, exceeding the old retry cap in total births.
	for i := range 200 {
		diet := DietHerbivore
		if i%2 == 1 {
			diet = DietCarnivore
		}
		c := factory.createForDiet(diet)
		if c == nil || c.diet != diet || c.num != i+1 || c.char != safeRunes[i%len(safeRunes)] || c.ancestor != c || c.parent != nil || c.generation != 1 {
			t.Fatalf("birth %d has wrong traits or consumed extra identity/rune: %+v", i, c)
		}
		if diet == DietCarnivore {
			if c.maxHealth < 900 || c.maxHealth > 1497 || c.maxNutrition < 2000 || c.maxNutrition > 6990 || c.vision < 25 || c.vision >= 125 || c.nutrition != int(float64(c.maxNutrition)*0.9) {
				t.Fatalf("carnivore stats don't match diet: %+v", c)
			}
		} else if c.maxHealth < 300 || c.maxHealth > 499 || c.maxNutrition < 200 || c.maxNutrition > 699 || c.vision < 5 || c.vision >= 25 {
			t.Fatalf("herbivore stats don't match diet: %+v", c)
		}
	}
}

func TestCreatureCreation_InvalidDietDoesNotAllocate(t *testing.T) {
	// Arrange
	ResetRand(12)
	factory := CreatureFactory()

	// Act
	if c := factory.createForDiet("unknown"); c != nil {
		t.Fatal("invalid diet created a creature")
	}
	got := factory.createForDiet(DietHerbivore)
	ResetRand(12)
	want := CreatureFactory().createForDiet(DietHerbivore)

	// Assert: invalid requests consume neither randomness nor allocation state.
	got.ancestor, want.ancestor = nil, nil
	if got.num != want.num || got.char != want.char || got.maxHealth != want.maxHealth || got.maxNutrition != want.maxNutrition || got.vision != want.vision {
		t.Fatalf("invalid request changed the following birth: got %+v, want %+v", got, want)
	}
}
