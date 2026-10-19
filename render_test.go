package main

import (
	"regexp"
	"strings"
	"testing"
)

func TestRenderMapShowsMixedPlantSpeciesInThreeColumns(t *testing.T) {
	env := [][]environment{{{capacity: plantCellCapacity, plants: [plantSpeciesCount]plantPopulation{
		{biomass: 5}, {biomass: 25}, {biomass: 80},
	}}}}

	got := renderMap(env)
	if !strings.Contains(got, "|") || !strings.Contains(got, "*") || !strings.Contains(got, "#") {
		t.Fatalf("mixed vegetation lacks species/density textures: %q", got)
	}
	if strings.Count(strings.Split(strings.TrimSpace(got), "\n")[0], "\033[") == 0 {
		t.Fatalf("plant backgrounds missing: %q", got)
	}
}

func TestRenderMapKeepsThreeColumnsAndCreatureOverlay(t *testing.T) {
	creature := &Creature{char: 'A', colorCode: "\033[38;5;15m", diet: DietHerbivore}
	env := [][]environment{{{capacity: plantCellCapacity, plants: [plantSpeciesCount]plantPopulation{{biomass: 1}, {biomass: 20}, {biomass: 80}}, creatureList: []*Creature{creature, creature, creature, creature}}}}
	first := renderMap(env)
	if got := renderMap(env); first != got {
		t.Fatal("unchanged vegetation rendered differently")
	}
	line := strings.Split(strings.TrimPrefix(first, "\033[H\033[2J"), "\n")[0]
	plain := regexp.MustCompile("\\x1b\\[[0-9;]*m").ReplaceAllString(line, "")
	if plain != "AAA" {
		t.Fatalf("cell did not stay three columns under creature overlay: %q", plain)
	}
}

func TestPlantTextureDensityBandsAndSpeciesAreDistinct(t *testing.T) {
	for _, test := range []struct {
		biomass float64
		texture rune
	}{
		{5, '.'}, {25, '|'}, {50, 'I'}, {75, 'H'},
	} {
		_, got := plantTexture(PlantGrass, test.biomass, plantCellCapacity)
		if got != test.texture {
			t.Errorf("grass at %.0f%% density: got %q, want %q", test.biomass, got, test.texture)
		}
	}
	for species, want := range []rune{'I', 'o', 'X'} {
		_, got := plantTexture(PlantSpecies(species), 50, plantCellCapacity)
		if got != want {
			t.Errorf("species %v at medium density: got %q, want %q", species, got, want)
		}
	}
}
