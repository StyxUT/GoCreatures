package main

import (
	"math/rand"
	"slices"
)

// Determine which action a creature will take
func (c *Creature) decideAction(env [][]environment) ActionIntent {
	location := env[c.Y][c.X]
	modAggression := c.aggression
	var targetCreature *Creature

	// Fight if sharing location with non-relative, healthy, and aggressive
	if len(location.creatureList) > 1 &&
		float64(c.health) > (float64(c.maxHealth)*0.75) {

		// Find first non-self, non-parent creature at the same location
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

	// Clone if between 20% & 90% maxAge && >= 70% of maxHealth
	if float64(c.age) >= 0.2*float64(c.maxAge) &&
		float64(c.age) < 0.9*float64(c.maxAge) &&
		float64(c.health) >= 0.75*float64(c.maxHealth) {
		return ActionIntent{
			Creature: c,
			Type:     "clone",
		}
	}

	// If nutrition is low, and location food > 0 eat
	if float64(c.nutrition)/float64(c.maxNutrition) < 0.95 && location.food > 1 {
		return ActionIntent{
			Creature: c,
			Type:     "eat",
		}
	}

	// If nutrition is high, or health is low and > 50% nutrition, sleep
	if float64(c.nutrition)/float64(c.maxNutrition) > 0.80 || (float64(c.nutrition)/float64(c.maxNutrition) > 0.50 && float64(c.health)/float64(c.maxHealth) < 0.50) {
		return ActionIntent{
			Creature: c,
			Type: "sleep",
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
func resolveActions(intents []ActionIntent, env [][]environment, createCreature func(*Creature) *Creature, tickStats *TickStats) {
	// Phase 1: Group intents by type
	var moveIntents, cloneIntents, eatIntents, fightIntents, sleepIntents []ActionIntent
	fought := make(map[*Creature]bool)

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
		case "sleep":
			tickStats.Sleeps++
			sleepIntents = append(sleepIntents, intent)
		}
	}
	// Phase 2: Resolve in safe order
  for _, intent := range fightIntents {
 	  fought[intent.Creature] = true
 	  fought[intent.TargetCreature] = true
 	  resolveFight(intent, tickStats)
   }
 
 	for _, intent := range eatIntents {
 	  if !fought[intent.Creature] {
 		  resolveEat(intent, env, tickStats)
 	  }
  }
 
  for _, intent := range cloneIntents {
 	  if !fought[intent.Creature] {
 		  resolveClone(intent, env, createCreature)
 	  }
  } 
 
  for _, intent := range moveIntents {
  	if !fought[intent.Creature] {
  	  resolveMove(intent, env)
    }
  }
 
  for _, intent := range sleepIntents {
 	  if !fought[intent.Creature] {
 	    resolveSleep(intent, tickStats)
 	  }
  }
}

func resolveFight(intent ActionIntent, tickStats *TickStats) {
	c := intent.Creature
	tc := intent.TargetCreature

	// two attacks each
	for range 2 {
		if c.health > 0 {
			cAttack := int((c.health + c.nutrition) / 5)
			tc.health -= cAttack
			tc.nutrition -= 5
			c.nutrition -= 5
		} else {
			tickStats.FightDeaths++
			break
		}
		if tc.health > 0 {
			tcAttack := int((tc.health + tc.nutrition) / 5)
			c.health -= tcAttack
			c.nutrition -= 5
			tc.nutrition -= 5
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

	c.nutrition -= 5

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
	clone.health = int(float64(clone.maxHealth) * 0.1)
	clone.nutrition = int(float64(clone.maxNutrition) * 0.1)

	// Add clone to map
	env[y][x].creatureList = append(env[y][x].creatureList, clone)

	// Reduce parent's health by 60%
	parent.health = int(float64(parent.health) * 0.3)
}

// Resolve "Eat" creature action
func resolveEat(intent ActionIntent, env [][]environment, tickStats *TickStats) {
	c := intent.Creature
	location := &env[c.Y][c.X]

	toConsume := min(5, location.food)
	c.nutrition += 10 * toConsume

	tickStats.FoodConsumed += toConsume
	location.food -= toConsume
}

// Resolve "Sleep" creature action
func resolveSleep(intent ActionIntent, tickStats *TickStats) {
	c := intent.Creature
	tickStats.Sleeps++
	c.nutrition -= 2

	processHealth(c, tickStats)
}

	
