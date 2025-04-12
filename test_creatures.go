package testhelpers

func DeadCreature() *Creature {
	return &Creature{
		health:    0,
		maxHealth: 100,
	}
}

func StarvingCreature() *Creature {
	return &Creature{
		nutrition:    0,
		maxNutrition: 100,
		health:       100,
		maxHealth:    100,
	}
}

func AggressiveCreature() *Creature {
	return &Creature{
		aggression: 100,
	}
}
