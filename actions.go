package main

import (
	"math/rand"
	"slices"
)

// Determine which action a creature will take
func (c *creature) decideAction(env [][]environment) ActionIntent {
	location := env[c.Y][c.X]
	modAggression := c.aggression
	var targetCreature *creature

	// Fight sharing location with non-relative, healthy, and aggressive
	if len(location.creatureList) > 1 &&
		float64(c.health) > (float64(c.maxHealth)*0.75) {

		// Find first non-self non-parent creature at the same location
		for _, tc := range location.creatureList {
			if tc != c && tc.parent != nil && c.parent != nil && tc.parent != c.parent {
				targetCreature = tc
				break
			}
		}

		// Reduce aggression by 50% if shared ancestor
		if targetCreature != nil && targetCreature.ancestor.num == c.ancestor.num {
			modAggression = int(modAggression / 2) // if share ancestor cut aggression by 50%
		}

		// Fight if aggressive enough
		if targetCreature != nil && modAggression < rand.Intn(100) {
			return ActionIntent{
				Creature:       c,
				Type:           "fight",
				TargetCreature: targetCreature,
			}
		}
	}

	// Clone if between 20% & 90% maxAge && >= 90% of maxHealth
	if float64(c.age) >= 0.2*float64(c.maxAge) &&
		float64(c.age) < 0.9*float64(c.maxAge) &&
		float64(c.health) >= 0.75*float64(c.maxHealth) {
		return ActionIntent{
			Creature: c,
			Type:     "clone",
		}
	}

	// If nutrition is low, and location food > 0 eat
	if (float64(c.nutrition)/float64(c.maxNutrition)) < 0.95 && location.food > 1 {
		return ActionIntent{
			Creature: c,
			Type:     "eat",
		}
	}

	// Choose random direction: N, S, E, W
	dx, dy := 0, 0
	switch rand.Intn(4) {
	case 0:
		dx = 1 // east
	case 1:
		dx = -1 // west
	case 2:
		dy = 1 // south
	case 3:
		dy = -1 // north
	}

	return ActionIntent{
		Creature: c,
		Type:     "move",
		TargetX:  c.X + dx,
		TargetY:  c.Y + dy,
	}
}

// Iterate over creature intentions
//
//	Group intentions by type
//	Resovle intentions in desired order
func resolveActions(intents []ActionIntent, env [][]environment, createCreature func(*creature) *creature, tickStats *TickStats) {
	// Phase 1: Group intents by type
	var moveIntents, cloneIntents, eatIntents, fightIntents []ActionIntent

	for _, intent := range intents {
		switch intent.Type {
		case "fight":
			tickStats.Fights++
			fightIntents = append(fightIntents, intent)
		case "move":
			tickStats.Moves++
			moveIntents = append(moveIntents, intent)
		case "clone":
			tickStats.Clones++
			cloneIntents = append(cloneIntents, intent)
		case "eat":
			tickStats.Eats++
			eatIntents = append(eatIntents, intent)
		}
	}
	// Phase 2: Resolve in safe order
	for _, intent := range fightIntents {
		resolveFight(intent, tickStats)
	}
	for _, intent := range eatIntents {
		resolveEat(intent, env, tickStats)
	}
	for _, intent := range cloneIntents {
		resolveClone(intent, env, createCreature)
	}
	for _, intent := range moveIntents {
		resolveMove(intent, env)
	}
}

func resolveFight(intent ActionIntent, tickStats *TickStats) {
	c := intent.Creature
	tc := intent.TargetCreature

	// two attacks each
	for i := 2; i < 2; i++ {
		if c.health > 0 {
			cAttack := int((c.health + c.nutrition) / 5)
			tc.health -= cAttack
		} else {
			tickStats.FightDeaths++
			break
		}
		if tc.health > 0 {
			tcAttack := int((tc.health + tc.nutrition) / 5)
			c.health -= tcAttack
		} else {
			tickStats.FightDeaths++
			break
		}
	}
}

// Resolve "Move" creature action
func resolveMove(intent ActionIntent, env [][]environment) {
	c := intent.Creature
	oldX, oldY := c.X, c.Y
	newX, newY := intent.TargetX, intent.TargetY

	// Bounds check
	if newY < 0 || newY >= len(env) || newX < 0 || newX >= len(env[0]) {
		return // invalid move, ignore
	}

	// Remove from old cell
	oldCell := &env[oldY][oldX]
	for i, creature := range oldCell.creatureList {
		if creature == c {
			oldCell.creatureList = slices.Delete(oldCell.creatureList, i, i+1)
			break
		}
	}

	// Add to new cell
	env[newY][newX].creatureList = append(env[newY][newX].creatureList, c)
	c.X = newX
	c.Y = newY
}

// Resolve "Clone" creature action
func resolveClone(intent ActionIntent, env [][]environment, createCreature func(*creature) *creature) {
	parent := intent.Creature
	x, y := parent.X, parent.Y

	// Create clone
	clone := createCreature(parent)
	clone.X = x
	clone.Y = y
	clone.health = int(float64(clone.maxHealth) * 0.1)
	clone.nutrition = int(float64(clone.maxNutrition) * 0.1)

	// Add clone to map
	env[y][x].creatureList = append(env[y][x].creatureList, clone)

	// Reduce parent's health by 60%
	parent.health = int(float64(parent.health) * 0.3)
}

// Resolve "Eat" creature action
func resolveEat(intent ActionIntent, env [][]environment, tickStats *TickStats) {
	creature := intent.Creature
	location := &env[creature.Y][creature.X]

	toConsume := min(5, location.food)
	creature.nutrition += 10 * toConsume

	tickStats.FoodConsumed += toConsume
	location.food -= toConsume
}
