package main

import (
	"math/rand"
	"sync"
)

type Creature struct {
	num          int
	parent       *Creature
	ancestor     *Creature
	generation		int
	char         rune
	colorCode    string
	health       int
	maxHealth    int
	X, Y         int
	age          int
	maxAge       int
	metabolism   int
	nutrition    int
	maxNutrition int
	aggression   int
	vision  		float64
	observed []*environment
}

type environment struct {
	creatureList []*Creature
	food         int
	maxFood      int
}

type ActionIntent struct {
	Creature       *Creature
	Type           string // e.g., "move", "rest", etc.
	TargetX        int
	TargetY        int
	TargetCreature *Creature
}

type AncestryStats struct {
	Ancestor     *Creature
	LineageCount int
}

// Generate a creature and set it's characteristics
func CreatureFactory() func(parent *Creature) *Creature {
	n := 1

	return func(parent *Creature) *Creature {
		var color string
		var generation int
		var maxAge int
		var maxHealth int
		var health int
		var metabolism int
		var nutrition int
		var maxNutrition int
		var aggression int
		var vision float64 // float to support gradual change before having an effect

		if parent == nil {
			// Set base values for ancestor creatures
			color = generateColor(n, 100)
			generation = 1
			maxAge = rand.Intn(200)
			maxHealth = rand.Intn(200) + 300
			health = int(rand.Intn(maxHealth / 5))
			metabolism = rand.Intn(25) + 1
			maxNutrition = rand.Intn(500) + 200
			nutrition = int(rand.Intn(maxNutrition / 4))
			aggression = rand.Intn(100)
			vision = rand.Float64()*8
		} else {
			color = parent.ancestor.colorCode
			generation = parent.generation + 1
			maxAge = int(float64(parent.maxAge) * (1 + (rand.Float64()*0.02 - 0.01)))             // set maxAge to parent's max age + or - 1%
			maxHealth = int(float64(parent.maxHealth) * (1 + (rand.Float64()*0.02 - 0.01)))       // set maxHealth to parent's max health + or - 1%
			metabolism = int(float64(parent.metabolism) * (1 + (rand.Float64()*0.02 - 0.01)))     // set metabolism to parent's metabolism + or - 1%
			maxNutrition = int(float64(parent.maxNutrition) * (1 + (rand.Float64()*0.02 - 0.01))) // set maxNutrition to parent's maxNutrition + or - 1%
			aggression = parent.aggression + (rand.Intn(3) - 1)                                   // add or remove 1
			vision = parent.vision + (rand.Float64()*0.2 - 0.1)
		}

		c := &Creature{
			num:          n,
			colorCode:    color,
			char:         NextSafeRune(),
			parent:       parent,
			generation: generation,
			health:       health,
			maxHealth:    maxHealth,
			X:            0,
			Y:            0,
			age:          0,
			maxAge:       maxAge,
			metabolism:   metabolism,
			nutrition:    nutrition,
			maxNutrition: maxNutrition,
			aggression:   aggression,
			vision: vision,
		}

		// set root creature's parent and ancestor to be self
		if parent == nil {
			c.ancestor = c
		} else {
			c.ancestor = parent.ancestor
		}

		n++
		return c
	}
}

// Iterate over each creature and collect its action intention
func collectIntents(creatures []*Creature, env [][]environment) []ActionIntent {
	intentCh := make(chan ActionIntent, len(creatures))
	var wg sync.WaitGroup

	for _, c := range creatures {
		wg.Add(1)
		go func(cr *Creature) {
			defer wg.Done()
			intent := cr.decideAction(env)
			intentCh <- intent
		}(c)
	}

	wg.Wait()
	close(intentCh)

	var intents []ActionIntent
	for intent := range intentCh {
		intents = append(intents, intent)
	}
	return intents
}

// Process environmental updates including removal of dead creatures
func processEnvironment(env [][]environment, tickStats *TickStats) {
	totalCreatures := 0
	totalDeaths := 0
	ageDeaths := 0
	starvationDeaths := 0

	for i := range env {
		for j := range env[i] {
			cell := &env[i][j]
			// Grow food
			if cell.food < cell.maxFood && rand.Intn(40) == 0 {
				tickStats.FoodAdded++
				cell.food++
			}

			// Manage dead creatures
			newList := cell.creatureList[:0] // reuse underlying array
			for _, c := range cell.creatureList {
				switch {
				case c.age >= c.maxAge:
					ageDeaths++
					totalDeaths++
				case c.health <= 0:
					starvationDeaths++
					totalDeaths++
				default:
					newList = append(newList, c)
					totalCreatures++
				}

				cell.creatureList = newList
			}
		}
	}
	tickStats.TotalCreatures = totalCreatures
	tickStats.TotalDeaths = totalDeaths
	tickStats.AgeDeaths = ageDeaths
	tickStats.StarvationDeaths = starvationDeaths
}

// Process creature health
func processHealth(c *Creature, tickStats *TickStats) {
	tickStats.TotalHealth += c.health
	if c.health >= c.maxHealth {
		return
	}

	pctNutrition := (float64(c.nutrition) / float64(c.maxNutrition)) * 100
	switch {
	case pctNutrition <= 25:
		tickStats.HealthAdded--
		c.health--
	case pctNutrition > 50 && pctNutrition < 75:
		tickStats.HealthAdded += c.metabolism
		c.health = c.health + c.metabolism
	case pctNutrition >= 75:
		tickStats.HealthAdded += (c.metabolism * 2)
		c.health = c.health + (c.metabolism * 2)
	}

	// Ensure health doesn't exceed maxHealth
	c.health = min(c.maxHealth, c.health)
}

// Process creature nutrition
func processNutrition(c *Creature, tickStats *TickStats) {
	c.nutrition -= c.metabolism
	tickStats.TotalNutrition += c.nutrition
}

