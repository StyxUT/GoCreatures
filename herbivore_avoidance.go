package main

// Only living predators in the current cell or the creature's observations
// influence avoidance. A shared cell is known even with zero vision.
func observedCarnivoreLocations(c *Creature, current *environment) []ObservedLocation {
	var threats []ObservedLocation
	add := func(location ObservedLocation) {
		for _, other := range location.Env.creatureList {
			if other != nil && other != c && other.health > 0 && other.isCarnivore() {
				threats = append(threats, location)
				return
			}
		}
	}
	add(ObservedLocation{Env: current, X: c.X, Y: c.Y})
	for _, location := range c.observed {
		add(location)
	}
	return threats
}

// Chebyshev distance matches diagonal movement and adjacent carnivore attacks.
func nearestCarnivoreDistance(x, y int, threats []ObservedLocation) int {
	distance := 1 << 30
	for _, threat := range threats {
		distance = min(distance, max(absInt(x-threat.X), absInt(y-threat.Y)))
	}
	return distance
}

func selectHerbivoreEscape(c *Creature, env [][]environment, threats []ObservedLocation) (ObservedLocation, bool) {
	if len(threats) == 0 {
		return ObservedLocation{}, false
	}
	currentDistance := nearestCarnivoreDistance(c.X, c.Y, threats)
	bestDistance := currentDistance
	var candidates []ObservedLocation
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			x, y := c.X+dx, c.Y+dy
			if dx == 0 && dy == 0 || y < 0 || y >= len(env) || x < 0 || x >= len(env[y]) {
				continue
			}
			distance := nearestCarnivoreDistance(x, y, threats)
			if distance <= currentDistance || distance < bestDistance {
				continue
			}
			if distance > bestDistance {
				bestDistance = distance
				candidates = candidates[:0]
			}
			candidates = append(candidates, ObservedLocation{X: x, Y: y})
		}
	}
	if len(candidates) == 0 {
		return ObservedLocation{}, false
	}
	// Stable identity-based ties avoid shared RNG and use no unseen cell contents.
	return chooseByModulo(candidates, c.num), true
}
