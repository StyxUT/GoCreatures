package main

import (
	"slices"
)

const (
	herbivoreNutritionPerBiomass   = 15
	herbivorePlantSeekingValue     = 30 * herbivoreNutritionPerBiomass
	carcassEnergyFraction          = 0.5
	carcassDecayPerTick            = 0.05
	carnivoreMealFraction          = 0.1
	carnivoreHandlingDuration      = 3
	cloneBaseImmobileTicks         = 10
	carnivoreFightNutritionMin     = 20
	carnivoreCriticalNutrition     = 25
	carnivoreFightHerbHealth       = 40
	carnivoreFightRivalHealth      = 60
	carnivoreRetreatHealth         = 35
	herbivoreCriticalHungerPercent = 15
	herbivoreStarvationRiskPercent = 5
	// Herbivores feed through the 70% reproduction minimum; once eligible, cloning takes priority.
	herbivoreBreedingReservePercent = 80
	herbivoreCloneCooldown          = 20
	carnivoreCloneCooldown          = 40
	cloneEnergyFraction             = 0.2
	cloneNutritionReserve           = 0.5
	cloneFoodPerHerbivoreFraction   = 6.0
)

func cloneCooldownFor(creature *Creature) int {
	if creature.isCarnivore() {
		return carnivoreCloneCooldown
	}
	return herbivoreCloneCooldown
}

func canReproduce(c *Creature, env [][]environment, pctHealth, pctAge float64) bool {
	if c == nil || c.immobileTicks > 0 || c.cloneCooldown > 0 || c.maxAge <= 0 || c.maxHealth <= 0 || pctAge <= 20 || pctAge >= 90 || pctHealth <= 70 || c.maxNutrition <= 0 {
		return false
	}
	budget := int(float64(c.maxNutrition) * cloneEnergyFraction)
	minimumEnergy := int(float64(c.maxNutrition) * (cloneEnergyFraction + cloneNutritionReserve))
	if c.nutrition < minimumEnergy || c.nutrition < budget {
		return false
	}
	if c.isCarnivore() {
		return observablePreyValue(c, env)/float64(observableCarnivoreCount(c, env)) >= float64(c.maxNutrition)*cloneNutritionReserve
	}
	return observablePlantValue(c, env)/float64(observableHerbivoreCount(c, env)) >= float64(c.maxNutrition)*cloneFoodPerHerbivoreFraction
}

func observableHerbivoreCount(creature *Creature, env [][]environment) int {
	count := 0
	if creature == nil || creature.Y < 0 || creature.Y >= len(env) || creature.X < 0 || creature.X >= len(env[creature.Y]) {
		return 1
	}
	count += herbivoreCount(env[creature.Y][creature.X].creatureList)
	for _, location := range creature.observed {
		if location.Env != nil {
			count += herbivoreCount(location.Env.creatureList)
		}
	}
	return max(count, 1)
}

func herbivoreCount(creatures []*Creature) int {
	count := 0
	for _, creature := range creatures {
		if !creature.isCarnivore() && creature.health > 0 {
			count++
		}
	}
	return count
}

func observableCarnivoreCount(creature *Creature, env [][]environment) int {
	count := 0
	if creature == nil || creature.Y < 0 || creature.Y >= len(env) || creature.X < 0 || creature.X >= len(env[creature.Y]) {
		return 1
	}
	count += carnivoreCount(env[creature.Y][creature.X].creatureList)
	for _, location := range creature.observed {
		if location.Env != nil {
			count += carnivoreCount(location.Env.creatureList)
		}
	}
	return max(count, 1)
}

func carnivoreCount(creatures []*Creature) int {
	count := 0
	for _, creature := range creatures {
		if creature.isCarnivore() && creature.health > 0 {
			count++
		}
	}
	return count
}

func observablePreyValue(creature *Creature, env [][]environment) float64 {
	if creature == nil || creature.Y < 0 || creature.Y >= len(env) || creature.X < 0 || creature.X >= len(env[creature.Y]) {
		return 0
	}
	value := env[creature.Y][creature.X].carcass.energy
	for _, prey := range env[creature.Y][creature.X].creatureList {
		if prey != creature && !prey.isCarnivore() && prey.health > 0 {
			value += float64(max(prey.maxNutrition, prey.maxHealth)) * carcassEnergyFraction
		}
	}
	for _, location := range creature.observed {
		if location.Env == nil {
			continue
		}
		value += location.Env.carcass.energy
		for _, prey := range location.Env.creatureList {
			if !prey.isCarnivore() && prey.health > 0 {
				value += float64(max(prey.maxNutrition, prey.maxHealth)) * carcassEnergyFraction
			}
		}
	}
	return value
}

func observablePlantValue(creature *Creature, env [][]environment) float64 {
	if creature == nil || creature.Y < 0 || creature.Y >= len(env) || creature.X < 0 || creature.X >= len(env[creature.Y]) {
		return 0
	}
	value := plantNutritionalValue(&env[creature.Y][creature.X])
	for _, location := range creature.observed {
		value += plantNutritionalValue(location.Env)
	}
	return value
}

func plantForagingScore(location ObservedLocation) float64 {
	if location.Env == nil {
		return 0
	}
	competitors := 0
	for _, creature := range location.Env.creatureList {
		if !creature.isCarnivore() && creature.health > 0 {
			competitors++
		}
	}
	return plantNutritionalValue(location.Env) / float64(max(competitors, 1))
}

func cloneImmobileTicks(c *Creature) int {
	if c == nil {
		return cloneBaseImmobileTicks
	}

	metabolismPct := max(c.metabolism, 0)
	reduction := (cloneBaseImmobileTicks*metabolismPct + 50) / 100
	return max(cloneBaseImmobileTicks-reduction, 0)
}

// Determine which action a creature will take
func (c *Creature) decideAction(env [][]environment) ActionIntent {
	location := env[c.Y][c.X]

	pctHealth := float64(c.health) / float64(c.maxHealth) * 100
	pctAge := float64(c.age) / float64(c.maxAge) * 100
	pctNutrition := float64(c.nutrition) / float64(c.maxNutrition) * 100

	// Observe area based on current location and vision
	SpiralLook2D_N(env, c)
	canMove := c.immobileTicks == 0
	if c.isCarnivore() {
		return decideCarnivoreAction(c, env, location, pctHealth, pctAge, pctNutrition, canMove)
	}

	var threats []ObservedLocation
	if canMove {
		threats = observedCarnivoreLocations(c, &env[c.Y][c.X])
		if destination, ok := selectHerbivoreEscape(c, env, threats); ok {
			escape := ActionIntent{Creature: c, Type: "move", TargetX: destination.X, TargetY: destination.Y}
			if pctNutrition > herbivoreCriticalHungerPercent {
				return escape
			}
			food := decideHerbivoreAction(c, env, location, pctHealth, pctAge, pctNutrition, canMove)
			if food.Type == "eat" {
				if pctNutrition <= herbivoreStarvationRiskPercent {
					return food
				}
				return escape
			}
			if food.Type == "move" && nearestCarnivoreDistance(food.TargetX, food.TargetY, threats) > 0 {
				return food
			}
			return escape
		}
	}
	intent := decideHerbivoreAction(c, env, location, pctHealth, pctAge, pctNutrition, canMove)
	if intent.Type == "move" && len(threats) > 0 {
		distance := nearestCarnivoreDistance(intent.TargetX, intent.TargetY, threats)
		if distance == 0 || distance < nearestCarnivoreDistance(c.X, c.Y, threats) {
			return ActionIntent{Creature: c, Type: "sleep"}
		}
	}
	return intent
}

func decideHerbivoreAction(c *Creature, env [][]environment, location environment, pctHealth, pctAge, pctNutrition float64, canMove bool) ActionIntent {
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
		if targetCreature != nil && modAggression > RandIntn(100) {
			return ActionIntent{
				Creature:       c,
				Type:           "fight",
				TargetCreature: targetCreature,
			}
		}
	}

	// Hungry herbivores compare mixed vegetation by total nutritional value.
	slices.SortStableFunc(c.observed, func(a, b ObservedLocation) int {
		if plantForagingScore(a) > plantForagingScore(b) {
			return -1
		}
		if plantForagingScore(a) < plantForagingScore(b) {
			return 1
		}
		return 0
	})
	currentPlantValue := plantNutritionalValue(&env[c.Y][c.X])
	if canMove && len(c.observed) > 0 && (pctNutrition <= herbivoreCriticalHungerPercent || (pctNutrition > 30 && pctNutrition < 70)) && currentPlantValue < herbivorePlantSeekingValue && currentPlantValue < plantForagingScore(c.observed[0]) {
		bestPlantValue := plantForagingScore(c.observed[0])
		var bests []ObservedLocation
		for _, ol := range c.observed {
			if plantForagingScore(ol) == bestPlantValue {
				bests = append(bests, ol)
			} else {
				break
			}
		}
		chosen := chooseByModulo(bests, c.num)
		dx := sign(chosen.X - c.X)
		dy := sign(chosen.Y - c.Y)

		return ActionIntent{
			Creature: c,
			Type:     "move",
			TargetX:  c.X + dx,
			TargetY:  c.Y + dy,
		}
	}

	if canReproduce(c, env, pctHealth, pctAge) && pctNutrition > herbivoreCriticalHungerPercent {
		return ActionIntent{Creature: c, Type: "clone"}
	}

	// Feed toward a breeding reserve above the 70% clone minimum.
	if !c.isCarnivore() && pctNutrition <= herbivoreBreedingReservePercent && plantBiomassAvailable(&env[c.Y][c.X]) > 0 {
		return ActionIntent{
			Creature: c,
			Type:     "eat",
		}
	}

	// If crowded move toward an empty location
	if canMove && len(c.observed) > 0 && len(location.creatureList) > 2 {
		SortEnvironmentListByOccupantsAsc(c.observed)
		leastCount := len(c.observed[0].Env.creatureList)
		var leasts []ObservedLocation
		for _, ol := range c.observed {
			if len(ol.Env.creatureList) == leastCount {
				leasts = append(leasts, ol)
			} else {
				break
			}
		}
		chosen := chooseByModulo(leasts, c.num)
		dx := sign(chosen.X - c.X)
		dy := sign(chosen.Y - c.Y)

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

	if !canMove {
		return ActionIntent{
			Creature: c,
			Type:     "sleep",
		}
	}

	// If high health and aggressive, move toward enemy

	// If no other action, wonder in random direction: N, S, E, W
	dx, dy := 0, 0
	switch RandIntn(4) {
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

func decideCarnivoreAction(c *Creature, env [][]environment, location environment, pctHealth, pctAge, pctNutrition float64, canMove bool) ActionIntent {
	if c.handlingTicks > 0 {
		return ActionIntent{Creature: c, Type: "sleep"}
	}
	if location.carcass.energy > 0 && pctNutrition < 80 {
		return ActionIntent{Creature: c, Type: "eat"}
	}
	if canMove && pctNutrition < 80 {
		if carcass, ok := selectCarcassLocation(c); ok {
			dx, dy := sign(carcass.X-c.X), sign(carcass.Y-c.Y)
			return ActionIntent{Creature: c, Type: "move", TargetX: c.X + dx, TargetY: c.Y + dy}
		}
	}

	adjacentTarget := selectAdjacentCarnivorePrey(c, env)
	if adjacentTarget != nil && !adjacentTarget.isCarnivore() && pctNutrition <= carnivoreCriticalNutrition {
		if canCarnivoreFightTarget(c, adjacentTarget, pctHealth, pctNutrition, true) {
			return ActionIntent{Creature: c, Type: "fight", TargetCreature: adjacentTarget}
		}
	}
	if canReproduce(c, env, pctHealth, pctAge) && pctNutrition > carnivoreCriticalNutrition {
		return ActionIntent{Creature: c, Type: "clone"}
	}

	if canMove && pctHealth < carnivoreRetreatHealth && pctNutrition > carnivoreCriticalNutrition {
		if retreatLocation, ok := selectCarnivoreRetreatLocation(c, location); ok {
			dx := sign(retreatLocation.X - c.X)
			dy := sign(retreatLocation.Y - c.Y)
			return ActionIntent{Creature: c, Type: "move", TargetX: c.X + dx, TargetY: c.Y + dy}
		}
	}

	if pctNutrition > 70 || (pctNutrition > 40 && pctHealth < 60) {
		return ActionIntent{Creature: c, Type: "sleep"}
	}

	if len(location.creatureList) > 1 {
		if targetCreature := selectPreferredCarnivorePrey(c, location.creatureList); canCarnivoreFightTarget(c, targetCreature, pctHealth, pctNutrition, false) {
			return ActionIntent{Creature: c, Type: "fight", TargetCreature: targetCreature}
		}
	}

	if canCarnivoreFightTarget(c, adjacentTarget, pctHealth, pctNutrition, true) {
		return ActionIntent{Creature: c, Type: "fight", TargetCreature: adjacentTarget}
	}

	if canMove {
		if preyLocation, ok := selectCarnivorePreyLocation(c); ok {
			dx := sign(preyLocation.X - c.X)
			dy := sign(preyLocation.Y - c.Y)
			return ActionIntent{Creature: c, Type: "move", TargetX: c.X + dx, TargetY: c.Y + dy}
		}
	}

	if !canMove {
		return ActionIntent{Creature: c, Type: "sleep"}
	}

	if len(c.observed) > 0 && len(location.creatureList) > 2 {
		SortEnvironmentListByOccupantsAsc(c.observed)
		leastCount := len(c.observed[0].Env.creatureList)
		var leasts []ObservedLocation
		for _, ol := range c.observed {
			if len(ol.Env.creatureList) == leastCount {
				leasts = append(leasts, ol)
			} else {
				break
			}
		}
		chosen := chooseByModulo(leasts, c.num)
		dx := sign(chosen.X - c.X)
		dy := sign(chosen.Y - c.Y)
		return ActionIntent{Creature: c, Type: "move", TargetX: c.X + dx, TargetY: c.Y + dy}
	}

	dx, dy := 0, 0
	switch RandIntn(4) {
	case 0:
		dx = 1
	case 1:
		dx = -1
	case 2:
		dy = 1
	case 3:
		dy = -1
	}

	return ActionIntent{Creature: c, Type: "move", TargetX: c.X + dx, TargetY: c.Y + dy}
}

func selectCarcassLocation(creature *Creature) (ObservedLocation, bool) {
	if creature == nil {
		return ObservedLocation{}, false
	}
	var selected ObservedLocation
	energy := 0.0
	for _, location := range creature.observed {
		if location.Env != nil && location.Env.carcass.energy > energy {
			selected, energy = location, location.Env.carcass.energy
		}
	}
	return selected, energy > 0
}

func canCarnivoreFightTarget(hunter, target *Creature, pctHealth, pctNutrition float64, adjacent bool) bool {
	if hunter == nil || target == nil || target == hunter || target.health <= 0 {
		return false
	}

	if target.isCarnivore() {
		if sharesAncestor(hunter, target) {
			return false
		}
		if pctNutrition <= carnivoreFightNutritionMin {
			return false
		}
		if pctHealth <= carnivoreFightRivalHealth {
			return false
		}
		return carnivoreWillFight(pctHealth)
	}

	if adjacent && pctNutrition <= carnivoreCriticalNutrition {
		return true
	}
	if pctHealth <= carnivoreFightHerbHealth {
		return false
	}
	if adjacent {
		return true
	}
	return carnivoreWillFight(pctHealth)
}

func carnivoreWillFight(pctHealth float64) bool {
	chance := int(pctHealth + 0.5)
	if chance <= 0 {
		return false
	}
	if chance >= 100 {
		return true
	}
	return RandIntn(100) < chance
}

func selectCarnivoreRetreatLocation(hunter *Creature, current environment) (ObservedLocation, bool) {
	if hunter == nil || len(hunter.observed) == 0 {
		return ObservedLocation{}, false
	}

	leastCount := 1 << 30
	retreatCandidates := make([]ObservedLocation, 0)
	for _, ol := range hunter.observed {
		occupants := len(ol.Env.creatureList)
		if occupants < leastCount {
			leastCount = occupants
			retreatCandidates = append(retreatCandidates[:0], ol)
			continue
		}
		if occupants == leastCount {
			retreatCandidates = append(retreatCandidates, ol)
		}
	}

	if len(retreatCandidates) == 0 {
		return ObservedLocation{}, false
	}

	currentOccupants := len(current.creatureList)
	if leastCount >= currentOccupants {
		return ObservedLocation{}, false
	}

	farthestDistance := -1
	farthestCandidates := make([]ObservedLocation, 0, len(retreatCandidates))
	for _, ol := range retreatCandidates {
		distance := absInt(ol.X-hunter.X) + absInt(ol.Y-hunter.Y)
		if distance > farthestDistance {
			farthestDistance = distance
			farthestCandidates = append(farthestCandidates[:0], ol)
			continue
		}
		if distance == farthestDistance {
			farthestCandidates = append(farthestCandidates, ol)
		}
	}

	if len(farthestCandidates) == 0 {
		return ObservedLocation{}, false
	}

	return chooseByModulo(farthestCandidates, hunter.num), true
}

func selectAdjacentCarnivorePrey(hunter *Creature, env [][]environment) *Creature {
	if hunter == nil || len(env) == 0 || len(env[0]) == 0 {
		return nil
	}
	height := len(env)
	width := len(env[0])

	var herbivorePrey *Creature
	var rivalCarnivorePrey *Creature

	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			x := hunter.X + dx
			y := hunter.Y + dy
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}
			target := selectPreferredCarnivorePrey(hunter, env[y][x].creatureList)
			if target == nil {
				continue
			}
			if !target.isCarnivore() {
				if herbivorePrey == nil {
					herbivorePrey = target
				}
				continue
			}
			if rivalCarnivorePrey == nil {
				rivalCarnivorePrey = target
			}
		}
	}

	if herbivorePrey != nil {
		return herbivorePrey
	}
	return rivalCarnivorePrey
}

func selectPreferredCarnivorePrey(hunter *Creature, occupants []*Creature) *Creature {
	var herbivorePrey *Creature
	var rivalCarnivorePrey *Creature

	for _, target := range occupants {
		if target == nil || target == hunter || target.health <= 0 {
			continue
		}
		if !target.isCarnivore() {
			if herbivorePrey == nil {
				herbivorePrey = target
			}
			continue
		}
		if !sharesAncestor(hunter, target) && rivalCarnivorePrey == nil {
			rivalCarnivorePrey = target
		}
	}

	if herbivorePrey != nil {
		return herbivorePrey
	}
	return rivalCarnivorePrey
}

func selectCarnivorePreyLocation(hunter *Creature) (ObservedLocation, bool) {
	type preyCandidate struct {
		location ObservedLocation
		rank     int
		distance int
		count    int
	}

	const noRank = 99
	best := preyCandidate{rank: noRank, distance: 1 << 30, count: -1}
	tied := make([]ObservedLocation, 0)

	for _, ol := range hunter.observed {
		herbivores, rivalCarnivores := countCarnivorePreyByType(hunter, ol.Env.creatureList)
		rank := noRank
		count := 0
		switch {
		case herbivores > 0:
			rank = 0
			count = herbivores
		case rivalCarnivores > 0:
			rank = 1
			count = rivalCarnivores
		default:
			continue
		}

		distance := absInt(ol.X-hunter.X) + absInt(ol.Y-hunter.Y)
		candidate := preyCandidate{location: ol, rank: rank, distance: distance, count: count}

		if candidate.rank < best.rank ||
			(candidate.rank == best.rank && candidate.distance < best.distance) ||
			(candidate.rank == best.rank && candidate.distance == best.distance && candidate.count > best.count) {
			best = candidate
			tied = append(tied[:0], candidate.location)
			continue
		}

		if candidate.rank == best.rank && candidate.distance == best.distance && candidate.count == best.count {
			tied = append(tied, candidate.location)
		}
	}

	if len(tied) == 0 {
		return ObservedLocation{}, false
	}
	return chooseByModulo(tied, hunter.num), true
}

func countCarnivorePreyByType(hunter *Creature, occupants []*Creature) (herbivores int, rivalCarnivores int) {
	for _, target := range occupants {
		if target == nil || target == hunter || target.health <= 0 {
			continue
		}
		if !target.isCarnivore() {
			herbivores++
			continue
		}
		if !sharesAncestor(hunter, target) {
			rivalCarnivores++
		}
	}
	return herbivores, rivalCarnivores
}

func sharesAncestor(a, b *Creature) bool {
	if a == nil || b == nil || a.ancestor == nil || b.ancestor == nil {
		return false
	}
	return a.ancestor == b.ancestor
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Iterate over creature intentions
//
//	Group intentions by type
//	Resovle intentions in desired order
func resolveActions(intents []ActionIntent, env [][]environment, creatureFac func(*Creature) *Creature, tickStats *TickStats, carnivoreTotals *CarnivoreActionTotals) {
	// Phase 1: Group intents by type
	var moveIntents, cloneIntents, eatIntents, fightIntents, sleepIntents []ActionIntent
	fought := make(map[*Creature]bool)

	for _, intent := range intents {
		if intent.Creature == nil || intent.Creature.health <= 0 {
			continue
		}
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
		if intent.Creature.health <= 0 || intent.TargetCreature == nil || intent.TargetCreature.health <= 0 || intent.TargetCreature == intent.Creature {
			continue
		}
		fought[intent.Creature] = true
		fought[intent.TargetCreature] = true
		resolveFight(intent, tickStats, carnivoreTotals, env)
		if carnivoreTotals != nil {
			if intent.Creature.isCarnivore() {
				carnivoreTotals.Fights++
			}
			if intent.TargetCreature != nil && intent.TargetCreature.isCarnivore() {
				carnivoreTotals.Fights++
			}
		}
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
			if intent.Creature.isCarnivore() {
				cell := &env[intent.Creature.Y][intent.Creature.X]
				if cell.carcass.energy <= 0 || intent.Creature.handlingTicks > 0 {
					continue
				}
			}
			tickStats.Eats++
			if carnivoreTotals != nil && intent.Creature.isCarnivore() {
				carnivoreTotals.Eats++
			}
			resolveEat(intent, env, tickStats)
		}
	}

	for _, intent := range cloneIntents {
		if !fought[intent.Creature] {
			tickStats.Clones++
			if carnivoreTotals != nil && intent.Creature.isCarnivore() {
				carnivoreTotals.Clones++
			}
			resolveClone(intent, env, creatureFac)
		}
	}

	swapBlocked := reciprocalMoveCreatures(moveIntents)
	for _, intent := range moveIntents {
		if !fought[intent.Creature] && !swapBlocked[intent.Creature] {
			tickStats.Moves++
			if carnivoreTotals != nil && intent.Creature.isCarnivore() {
				carnivoreTotals.Moves++
			}
			resolveMove(intent, env)
		}
	}

	for _, intent := range sleepIntents {
		if !fought[intent.Creature] {
			tickStats.Sleeps++
			if carnivoreTotals != nil && intent.Creature.isCarnivore() {
				carnivoreTotals.Sleeps++
			}
			resolveSleep(intent, tickStats, carnivoreTotals)
		}
	}
}

func reciprocalMoveCreatures(moveIntents []ActionIntent) map[*Creature]bool {
	blocked := make(map[*Creature]bool)

	for i := 0; i < len(moveIntents); i++ {
		a := moveIntents[i]
		if a.Creature == nil {
			continue
		}
		for j := i + 1; j < len(moveIntents); j++ {
			b := moveIntents[j]
			if b.Creature == nil {
				continue
			}
			if a.TargetX == b.Creature.X && a.TargetY == b.Creature.Y && b.TargetX == a.Creature.X && b.TargetY == a.Creature.Y {
				blocked[a.Creature] = true
				blocked[b.Creature] = true
			}
		}
	}

	return blocked
}

// Resolve "Fight" action
func resolveFight(intent ActionIntent, tickStats *TickStats, carnivoreTotals *CarnivoreActionTotals, worlds ...[][]environment) {
	c := intent.Creature
	tc := intent.TargetCreature
	if c == nil || tc == nil || c == tc || c.health <= 0 || tc.health <= 0 {
		return
	}

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
		if carnivoreTotals != nil && c.isCarnivore() {
			carnivoreTotals.FightDeaths++
		}
		feedCarnivoreOnKill(tc, c, tickStats, carnivoreTotals, worlds...)
	}
	if tc.health <= 0 {
		tickStats.FightDeaths++
		if carnivoreTotals != nil && tc.isCarnivore() {
			carnivoreTotals.FightDeaths++
		}
		feedCarnivoreOnKill(c, tc, tickStats, carnivoreTotals, worlds...)
	}
}

func feedCarnivoreOnKill(predator, prey *Creature, tickStats *TickStats, carnivoreTotals *CarnivoreActionTotals, worlds ...[][]environment) {
	if predator == nil || prey == nil || !predator.isCarnivore() || predator.health <= 0 || len(worlds) == 0 {
		return
	}
	if prey.Y < 0 || prey.Y >= len(worlds[0]) || prey.X < 0 || prey.X >= len(worlds[0][prey.Y]) {
		return
	}
	energy := float64(max(prey.maxNutrition, prey.maxHealth)) * carcassEnergyFraction
	worlds[0][prey.Y][prey.X].carcass.energy += energy
	if tickStats != nil {
		tickStats.CarcassesCreated++
		tickStats.CarcassEnergyCreated += energy
	}
	if carnivoreTotals != nil {
		carnivoreTotals.Kills++
	}
}

func eatCarcass(creature *Creature, cell *environment, tickStats *TickStats) {
	if creature == nil || cell == nil || !creature.isCarnivore() || creature.handlingTicks > 0 || cell.carcass.energy <= 0 {
		return
	}
	meal := minFloat(cell.carcass.energy, maxFloat(float64(creature.maxNutrition)*carnivoreMealFraction, 1))
	if meal <= 0 {
		return
	}
	cell.carcass.energy -= meal
	applyPlantNutrition(creature, meal)
	// Counters decay at tick entry, so retain one extra to block the full
	// configured number of following decisions.
	creature.handlingTicks = carnivoreHandlingDuration + 1
	if tickStats != nil {
		tickStats.CarcassEnergyEaten += meal
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
	if parent == nil || parent.maxNutrition <= 0 {
		return
	}
	energyBudget := int(float64(parent.maxNutrition) * cloneEnergyFraction)
	minimumEnergy := int(float64(parent.maxNutrition) * (cloneEnergyFraction + cloneNutritionReserve))
	if parent.nutrition < minimumEnergy || parent.nutrition < energyBudget || parent.cloneCooldown > 0 {
		return
	}
	x, y := parent.X, parent.Y

	// Create clone
	clone := createCreature(parent)
	if clone == nil {
		return
	}
	clone.nutrition = energyBudget
	clone.nutritionProgress = 0
	clone.X = x
	clone.Y = y

	// Add clone to map
	env[y][x].creatureList = append(env[y][x].creatureList, clone)

	// Reduce parent's health by 70% after offspring stats are derived from pre-cost parent state.
	parent.health = int(float64(parent.health) * 0.3)
	parent.nutrition -= energyBudget
	parent.immobileTicks = cloneImmobileTicks(parent)
	clone.immobileTicks = cloneImmobileTicks(clone)
	// Cooldowns decay at tick entry, so keep one extra count to block the
	// configured number of full subsequent decisions.
	parent.cloneCooldown = cloneCooldownFor(parent) + 1
	clone.cloneCooldown = cloneCooldownFor(clone) + 1
}

// Resolve "Eat" creature action
func resolveEat(intent ActionIntent, env [][]environment, tickStats *TickStats) {
	c := intent.Creature
	location := &env[c.Y][c.X]
	if c.isCarnivore() {
		eatCarcass(c, location, tickStats)
		return
	}

	order := []int{int(PlantGrass), int(PlantClover), int(PlantBramble)}
	slices.SortStableFunc(order, func(a, b int) int {
		aYield := plantNutritionalYield(location, PlantSpecies(a))
		bYield := plantNutritionalYield(location, PlantSpecies(b))
		switch {
		case aYield > bYield:
			return -1
		case aYield < bYield:
			return 1
		default:
			return 0
		}
	})

	remaining := 5.0
	consumed := 0.0
	var consumedBySpecies [plantSpeciesCount]float64
	nutritionGain := 0.0
	for _, index := range order {
		population := &location.plants[index]
		amount := minFloat(maxFloat(population.biomass, 0), remaining)
		if amount == 0 {
			continue
		}
		population.biomass -= amount
		if population.biomass <= 1e-12 {
			*population = plantPopulation{}
		}
		consumed += amount
		consumedBySpecies[index] += amount
		remaining -= amount
		nutritionGain += amount * plantNutritionalYield(location, PlantSpecies(index))
		if remaining <= 0 {
			break
		}
	}

	applyPlantNutrition(c, nutritionGain)
	if tickStats != nil {
		tickStats.BiomassConsumed += consumed
		for species, amount := range consumedBySpecies {
			tickStats.PlantBiomassConsumed[species] += amount
		}
	}
}

func plantNutritionalYield(cell *environment, species PlantSpecies) float64 {
	yield := cell.plants[species].nutritionalYield
	if yield <= 0 {
		yield = plantTraits[species].nutrition
	}
	return yield
}

func applyPlantNutrition(creature *Creature, gain float64) {
	if creature.maxNutrition <= 0 {
		progress := creature.nutritionProgress + gain
		wholeNutrition := int(progress)
		creature.nutrition += wholeNutrition
		creature.nutritionProgress = progress - float64(wholeNutrition)
		return
	}
	if creature.nutrition >= creature.maxNutrition {
		creature.nutrition = creature.maxNutrition
		creature.nutritionProgress = 0
		return
	}

	available := float64(creature.maxNutrition-creature.nutrition) - creature.nutritionProgress
	if gain > available {
		gain = maxFloat(available, 0)
	}
	progress := creature.nutritionProgress + gain
	wholeNutrition := int(progress)
	creature.nutrition += wholeNutrition
	creature.nutritionProgress = progress - float64(wholeNutrition)
	if creature.nutrition >= creature.maxNutrition {
		creature.nutrition = creature.maxNutrition
		creature.nutritionProgress = 0
	}
}

// Resolve "Sleep" creature action
func resolveSleep(intent ActionIntent, tickStats *TickStats, carnivoreTotals *CarnivoreActionTotals) {
	c := intent.Creature
	applyNutritionCost(c, 1)

	processHealth(c, tickStats, carnivoreTotals)
}
