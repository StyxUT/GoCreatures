package main

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

func HealthyCreature() *Creature {
	return &Creature{
		health: 100,
	}
}

func HighVisionCreature() *Creature {
	return &Creature{
		vision: 20.11,
	}
}

func LowVisionCreature() *Creature {
	return &Creature{
		vision: 0.11,
	}
}
