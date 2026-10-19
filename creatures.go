package main

type DietType string

const (
	DietHerbivore DietType = "Herbivore"
	DietCarnivore DietType = "Carnivore"
)

type Creature struct {
	num               int
	parent            *Creature
	ancestor          *Creature
	generation        int
	char              rune
	colorCode         string
	health            int
	maxHealth         int
	X, Y              int
	age               int
	maxAge            int
	immobileTicks     int
	metabolism        int
	nutrition         int
	nutritionProgress float64
	maxNutrition      int
	diet              DietType
	aggression        int
	vision            float64
	observed          []ObservedLocation
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

type creatureFactory struct {
	nextID   int
	nextRune int
}

func CreatureFactory() *creatureFactory {
	return &creatureFactory{nextID: 1}
}

// create preserves random ancestor creation and inherited clone creation.
func (f *creatureFactory) create(parent *Creature) *Creature {
	diet := DietHerbivore
	if parent != nil {
		diet = parent.dietType()
	} else if RandIntn(5) == 0 {
		diet = DietCarnivore
	}
	return f.createWithDiet(parent, diet)
}

// createForDiet allocates exactly one ancestor, with stats derived from its diet.
func (f *creatureFactory) createForDiet(diet DietType) *Creature {
	if diet != DietHerbivore && diet != DietCarnivore {
		return nil
	}
	return f.createWithDiet(nil, diet)
}

func (f *creatureFactory) createWithDiet(parent *Creature, diet DietType) *Creature {
	n := f.nextID
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

		maxAge = RandIntn(400)
		maxHealth = RandIntn(200) + 300
		metabolism = RandIntn(25) + 1
		metabolism = max(1, metabolism/2)
		maxNutrition = RandIntn(500) + 200
		if diet == DietCarnivore {
			maxHealth *= 3
			maxNutrition *= 10
		}

		health = int(RandIntn(maxHealth / 2))
		nutrition = int(RandIntn(maxNutrition / 2))
		if diet == DietCarnivore {
			nutrition = int(float64(maxNutrition) * 0.9)
		}
		aggression = RandIntn(100)
		vision = 5 + (RandFloat64() * 20)
		if diet == DietCarnivore {
			vision *= 5
		}
	} else {
		// Set clone values based on parent
		color = parent.ancestor.colorCode
		generation = parent.generation + 1
		maxAge = int(float64(parent.maxAge) * (1 + (RandFloat64()*0.02 - 0.01)))
		maxHealth = int(float64(parent.maxHealth) * (1 + (RandFloat64()*0.02 - 0.01)))
		health = int(float64(maxHealth+parent.health) / 2 * 0.3)
		metabolism = int(float64(parent.metabolism) * (1 + (RandFloat64()*0.02 - 0.01)))
		maxNutrition = int(float64(parent.maxNutrition) * (1 + (RandFloat64()*0.02 - 0.01)))
		nutrition = int(float64(maxNutrition+parent.nutrition) / 2 * 0.3)
		diet = parent.dietType()
		aggression = parent.aggression + (RandIntn(3) - 1)
		vision = parent.vision + (RandFloat64()*0.2 - 0.1)
	}

	c := &Creature{
		num:          n,
		colorCode:    color,
		char:         safeRunes[f.nextRune%len(safeRunes)],
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
		diet:         diet,
		aggression:   aggression,
		vision:       vision,
	}

	// set root creature's parent and ancestor to be self
	if parent == nil {
		c.ancestor = c
	} else {
		c.ancestor = parent.ancestor
	}

	f.nextID++
	f.nextRune++
	return c
}

func (c *Creature) dietType() DietType {
	if c == nil || c.diet == "" {
		return DietHerbivore
	}
	return c.diet
}

func (c *Creature) isCarnivore() bool {
	return c != nil && c.dietType() == DietCarnivore
}

// Collect decisions in world order before resolving any actions. Keeping random
// draws on the tick goroutine makes both draw allocation and intent order stable.
func collectIntents(creatures []*Creature, env [][]environment) []ActionIntent {
	intents := make([]ActionIntent, 0, len(creatures))
	for _, c := range creatures {
		if c.health > 0 {
			intents = append(intents, c.decideAction(env))
		}
	}
	return intents
}

// Process creature health
func processHealth(c *Creature, tickStats *TickStats, carnivoreTotals *CarnivoreActionTotals) {
	pctNutrition := (float64(c.nutrition) / float64(c.maxNutrition)) * 100
	startingHealth := c.health
	if c.health <= 0 {
		return
	}

	switch {
	case pctNutrition > 70:
		heal := c.metabolism * 5
		tickStats.HealthAdded += heal
		c.health += heal
	case pctNutrition > 50:
		heal := c.metabolism * 2
		tickStats.HealthAdded += heal
		c.health += heal
	case pctNutrition > 10:
		tickStats.HealthAdded += c.metabolism
		c.health += c.metabolism
	case pctNutrition < 5:
		c.health -= c.metabolism * 2
	default:
		// No health change in the 5-10% nutrition band.
	}

	if c.health <= 0 && startingHealth > 0 {
		tickStats.StarvationDeaths++
		if carnivoreTotals != nil && c.isCarnivore() {
			carnivoreTotals.StarvationDeaths++
		}
	}

	// Ensure health doesn't exceed maxHealth
	c.health = min(c.maxHealth, c.health)

}

// Process creature nutrition
func processNutrition(c *Creature) {
	c.nutrition -= int(c.metabolism / 2)
}
