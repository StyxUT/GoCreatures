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
	if !strings.Contains(got, ".") || !strings.Contains(got, "*") || !strings.Contains(got, "#") {
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
	if plain != "A+3" {
		t.Fatalf("cell did not stay three columns under creature overlay: %q", plain)
	}
}

func TestRenderMapPreservesCarnivoreBracketsAndCrowdCounts(t *testing.T) {
	carnivore := func(char rune) *Creature {
		return &Creature{char: char, colorCode: "\033[38;5;15m", diet: DietCarnivore}
	}
	herbivore := func(char rune) *Creature {
		return &Creature{char: char, colorCode: "\033[38;5;15m", diet: DietHerbivore}
	}
	env := [][]environment{{
		{capacity: plantCellCapacity, creatureList: []*Creature{carnivore('C')}},
		{capacity: plantCellCapacity, creatureList: []*Creature{carnivore('C'), carnivore('D')}},
		{capacity: plantCellCapacity, creatureList: []*Creature{herbivore('A'), herbivore('B'), herbivore('C'), herbivore('D')}},
	}}
	output := renderMap(env)
	line := strings.Split(strings.TrimPrefix(output, "\033[H\033[2J"), "\n")[0]
	plain := regexp.MustCompile("\\x1b\\[[0-9;]*m").ReplaceAllString(line, "")
	if plain != "[C][2]A+3" {
		t.Fatalf("carnivore and crowd cues do not fit three columns per cell: %q", plain)
	}
}

func TestPlantTextureDensityBandsAndSpeciesAreDistinct(t *testing.T) {
	for _, test := range []struct {
		biomass float64
		texture rune
		color   string
	}{
		{5, '.', "\033[48;5;100m"}, {25, '|', "\033[48;5;64m"},
		{50, 'I', "\033[48;5;28m"}, {75, 'H', "\033[48;5;34m"},
	} {
		color, got := plantTexture(PlantGrass, test.biomass, plantCellCapacity)
		if got != test.texture {
			t.Errorf("grass at %.0f%% density: got %q, want %q", test.biomass, got, test.texture)
		}
		if color != test.color {
			t.Errorf("grass at %.0f%% density: color %q, want %q", test.biomass, color, test.color)
		}
	}
	for species, want := range []rune{'I', 'o', 'X'} {
		_, got := plantTexture(PlantSpecies(species), 50, plantCellCapacity)
		if got != want {
			t.Errorf("species %v at medium density: got %q, want %q", species, got, want)
		}
	}
}

func TestRenderMapDistinguishesEverySpeciesCombination(t *testing.T) {
	ansi := regexp.MustCompile("\\x1b\\[[0-9;]*m")
	for combination := 0; combination < 1<<plantSpeciesCount; combination++ {
		cell := environment{capacity: plantCellCapacity}
		var want strings.Builder
		for species := PlantSpecies(0); species < plantSpeciesCount; species++ {
			if combination&(1<<species) != 0 {
				cell.plants[species].biomass = 25
				_, texture := plantTexture(species, 25, plantCellCapacity)
				want.WriteRune(texture)
			} else {
				want.WriteByte(' ')
			}
		}
		output := renderMap([][]environment{{cell}})
		line := strings.Split(strings.TrimPrefix(output, "\033[H\033[2J"), "\n")[0]
		if got := ansi.ReplaceAllString(line, ""); got != want.String() {
			t.Errorf("species combination %03b rendered %q, want %q", combination, got, want.String())
		}
	}
}
