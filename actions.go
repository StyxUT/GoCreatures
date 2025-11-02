package main

import (
	"math/rand"
	"slices"
)

// Determine which action a creature will take
func (c *Creature) decideAction(env [][]environment) ActionIntent {
	location := env[c.Y][c.X]

	pctHealth := float64(c.health) / float64(c.maxHealth) * 100
	pctAge := float64(c.age) / float64(c.maxAge) * 100
	pctNutrition := float64(c.nutrition) / float64(c.maxNutrition) * 100

	// Observe area based on current location and vision
	SpiralLook2D_N(env, c)

	modAggression := c.aggression
	var targetCreature *Creature
	// Fight if sharing location with non-relative, healthy, and aggressive
	if len(location.creatureList) > 1 && pctHealth > 50 {
		// Find first non-self, non-parent creature at the same location
		for _, tc := range location.creatureList {
			if tc != c && tc.parent != nil && c.parent != nil && tc.parent != c.parent {
				targetCreature = tc
				break
			}
		}

		// Reduce aggression by 50% if shared ancestor
		if targetCreature != nil && targetCreature.ancestor.num == c.ancestor.num {
			modAggression = int(modAggression / 3) // if share ancestor cut aggression by 66%
		}

		// Fight if aggressive enough
		if targetCreature != nil && modAggression > rand.Intn(100) {
			return ActionIntent{
				Creature:       c,
				Type:           "fight",
				TargetCreature: targetCreature,
			}
		}
	}

	// Clone if between 20% & 90% maxAge && >= 70% of maxHealth
	if pctAge > 20 && pctAge < 90 && pctHealth > 70 {

		return ActionIntent{
			Creature: c,
			Type:     "clone",
		}
	}

	// If hungry and location.food is low, try to move toward more food
	// 		TODO: reduce location preference if others are in the area
	SortEnvironmentListByFood(c.observed)
	if len(c.observed) > 0 && pctNutrition > 30 && pctNutrition < 70 && location.food < 30 && location.food < c.observed[0].Env.food {
		target := c.observed[0]
		dx := sign(target.X - c.X)
		dy := sign(target.Y - c.Y)

		return ActionIntent{
			Creature: c,
			Type:     "move",
			TargetX:  c.X + dx,
			TargetY:  c.Y + dy,
		}
	}

	// If nutrition is low, and location food > 0 eat
	if pctNutrition <= 40 && location.food > 1 {
		return ActionIntent{
			Creature: c,
			Type:     "eat",
		}
	}

	// If crowded move toward an empty location
	if len(c.observed) > 0 && len(location.creatureList) > 2 {
		SortEnvironmentListByOccupantsAsc(c.observed)
		target := c.observed[0]
		dx := sign(target.X - c.X)
		dy := sign(target.Y - c.Y)

		return ActionIntent{
			Creature: c,
			Type:     "move",
			TargetX:  c.X + dx,
			TargetY:  c.Y + dy,
		}
	}

	// If nutrition is high, or health is low and > 20% nutrition, sleep
	if pctNutrition > 70 || (pctNutrition > 40 && pctHealth < 60) {
		return ActionIntent{
			Creature: c,
			Type:     "sleep",
		}
	}

	// If high health and aggressive, move toward enemy

	// If no other action, wonder in random direction: N, S, E, W
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
func resolveActions(intents []ActionIntent, env [][]environment, creatureFac func(*Creature) *Creature, tickStats *TickStats) {
	// Phase 1: Group intents by type
	var moveIntents, cloneIntents, eatIntents, fightIntents, sleepIntents []ActionIntent
	fought := make(map[*Creature]bool)

	for _, intent := range intents {
		switch intent.Type {
		case "fight":
			fightIntents = append(fightIntents, intent)
		case "move":
			moveIntents = append(moveIntents, intent)
		case "clone":
			cloneIntents = append(cloneIntents, intent)
		case "eat":
			eatIntents = append(eatIntents, intent)
		case "sleep":
			sleepIntents = append(sleepIntents, intent)
		}
	}
	// Phase 2: Resolve in safe order
	for _, intent := range fightIntents {
		fought[intent.Creature] = true
		fought[intent.TargetCreature] = true
		resolveFight(intent, tickStats)
	}
	// Fights metric counts distinct participant creatures, not fight pair events.
	// A single fight between A and B increments twice (once per participant) so tests expect 2.
	// This allows multiple fight intents involving same creature in one tick to still only count it once.
	// If event-level counting is desired later, track unique (attacker,target) pairs instead.
	// count fights in this way to account for a creature being involved in multiple fights
	for _, f := range fought {
		if f {
			tickStats.Fights++
		}
	}

	for _, intent := range eatIntents {
		if !fought[intent.Creature] {
			tickStats.Eats++
			resolveEat(intent, env, tickStats)
		}
	}

	for _, intent := range cloneIntents {
		if !fought[intent.Creature] {
			tickStats.Clones++
			resolveClone(intent, env, creatureFac)
		}
	}

	for _, intent := range moveIntents {
		if !fought[intent.Creature] {
			tickStats.Moves++
			resolveMove(intent, env)
		}
	}

	for _, intent := range sleepIntents {
		if !fought[intent.Creature] {
			tickStats.Sleeps++
			resolveSleep(intent, tickStats)
		}
	}
}

// Resolve "Fight" action
func resolveFight(intent ActionIntent, tickStats *TickStats) {
	c := intent.Creature
	tc := intent.TargetCreature

	// three attacks each
	for range 3 {
		if c.health > 0 {
			//TODO: consider reducing attack based on being young or old
			// 			reduce attack by:	absolute value of age - (maxAge / 2)
			cAttack := int((c.health + c.nutrition) / 50) // damage advantage to the agressor
			tc.health -= cAttack
			tc.nutrition -= 5
			c.nutrition -= 5
		} else {
			break
		}
		if tc.health > 0 {
			tcAttack := int((tc.health + tc.nutrition) / 60) // damage disadvantage to the defender
			c.health -= tcAttack
			c.nutrition -= 5
			tc.nutrition -= 5
		} else {
			break
		}
	}

	if c.health <= 0 {
		tickStats.FightDeaths++
	}
	if tc.health <= 0 {
		tickStats.FightDeaths++
	}
}

// Apply a nutrition cost scaled by metabolism.
// Base is multiplied by max(1, metabolism/10) for gentle scaling.
func applyNutritionCost(c *Creature, base int) {
	factor := max(int(c.metabolism/10), 1)
	c.nutrition -= max(base*factor, 1)
}

// Resolve "Move" creature action
func resolveMove(intent ActionIntent, env [][]environment) {
	c := intent.Creature
	oldX, oldY := c.X, c.Y
	newX, newY := intent.TargetX, intent.TargetY

	applyNutritionCost(c, 1)

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
func resolveClone(intent ActionIntent, env [][]environment, createCreature func(*Creature) *Creature) {
	parent := intent.Creature
	x, y := parent.X, parent.Y

	// Create clone
	clone := createCreature(parent)
	clone.X = x
	clone.Y = y

	// Add clone to map
	env[y][x].creatureList = append(env[y][x].creatureList, clone)

	// Reduce parent's health by 70%
	parent.health = int(float64(parent.health) * 0.3)
}

// Resolve "Eat" creature action
func resolveEat(intent ActionIntent, env [][]environment, tickStats *TickStats) {
	c := intent.Creature
	location := &env[c.Y][c.X]

	toConsume := min(5, location.food)
	gain := 15 * toConsume
	if c.maxNutrition > 0 {
		c.nutrition = min(c.nutrition+gain, c.maxNutrition)
	} else {
		c.nutrition += gain
	}

	tickStats.FoodConsumed += toConsume
	location.food -= toConsume
}

// Resolve "Sleep" creature action
func resolveSleep(intent ActionIntent, tickStats *TickStats) {
	c := intent.Creature
	applyNutritionCost(c, 1)

	processHealth(c, tickStats)
}
