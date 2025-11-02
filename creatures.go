package main

import (
	"math/rand"
	"sync"
)

type Creature struct {
	num          int
	parent       *Creature
	ancestor     *Creature
	generation   int
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
	vision       float64
	observed     []ObservedLocation
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

type ObservedLocation struct {
	Env *environment
	X   int
	Y   int
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
			health = int(rand.Intn(maxHealth / 2))
			metabolism = rand.Intn(25) + 1
			maxNutrition = rand.Intn(500) + 200
			nutrition = int(rand.Intn(maxNutrition / 2))
			aggression = rand.Intn(100)
			vision = 1 + (rand.Float64() * 20)
		} else {
		  // Set clone values based on parent	
			color = parent.ancestor.colorCode
			generation = parent.generation + 1
			maxAge = int(float64(parent.maxAge) * (1 + (rand.Float64()*0.02 - 0.01)))
			maxHealth = int(float64(parent.maxHealth) * (1 + (rand.Float64()*0.02 - 0.01)))
			health = int(float64(maxHealth + parent.health) / 2 * 0.3)
			metabolism = int(float64(parent.metabolism) * (1 + (rand.Float64()*0.02 - 0.01)))
			maxNutrition = int(float64(parent.maxNutrition) * (1 + (rand.Float64()*0.02 - 0.01)))
			nutrition = int(float64(maxNutrition + parent.nutrition) / 2 * 0.3)
			aggression = parent.aggression + (rand.Intn(3) - 1)                                 
			vision = parent.vision + (rand.Float64()*0.2 - 0.1)
		}

		c := &Creature{
			num:          n,
			colorCode:    color,
			char:         NextSafeRune(),
			parent:       parent,
			generation:   generation,
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
			vision:       vision,
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

// Process creature health
func processHealth(c *Creature, tickStats *TickStats) {
	pctNutrition := (float64(c.nutrition) / float64(c.maxNutrition)) * 100
	startingHealth := c.health
	if c.health <= 0 {
		return
	}

	switch {
	  case pctNutrition <= 10: // lose 10% of max health
	  	c.health -= int(float64(c.maxHealth) * 0.1)	
	  case pctNutrition <= 30: // lose 5% of max health
	  	c.health -= int(float64(c.maxHealth) * 0.05)	
	  case pctNutrition <= 50:  // 2% of max health
	  	c.health -= int(float64(c.maxHealth) * 0.02)	
	  case pctNutrition <= 70:  // 2% of max health
	  	tickStats.HealthAdded += c.metabolism
	  	c.health += c.metabolism
	  default:
	  	tickStats.HealthAdded += (c.metabolism * 2)
	  	c.health += (c.metabolism * 2)
	}
	
  if c.health <= 0 && startingHealth > 0 {
		tickStats.StarvationDeaths++
	}

	// Ensure health doesn't exceed maxHealth
	c.health = min(c.maxHealth, c.health)

	tickStats.TotalHealth += c.health
}

// Process creature nutrition
func processNutrition(c *Creature, tickStats *TickStats) {
	c.nutrition -= int(c.metabolism / 2)
	tickStats.TotalNutrition += c.nutrition
}
