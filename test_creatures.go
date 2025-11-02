package main

// RelatedCreatures returns two creatures sharing the same ancestor but different parents.
// Aggression values can be provided; if zero use defaults.
func RelatedCreatures(aggressionA, aggressionB int) (c1, c2 *Creature) {
	ancestor := &Creature{num: 1000}
	parentA := &Creature{ancestor: ancestor}
	parentB := &Creature{ancestor: ancestor}
	if aggressionA == 0 {
		aggressionA = 50
	}
	if aggressionB == 0 {
		aggressionB = 10
	}
	c1 = &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 0, maxAge: 100, nutrition: 50, maxNutrition: 100, aggression: aggressionA, vision: 0, parent: parentA, ancestor: ancestor}
	c2 = &Creature{X: 1, Y: 1, health: 100, maxHealth: 100, age: 0, maxAge: 100, nutrition: 50, maxNutrition: 100, aggression: aggressionB, vision: 0, parent: parentB, ancestor: ancestor}
	return
}

func DeadCreature() *Creature {
	return &Creature{
		num:       1,
		colorCode: "101",
		health:    0,
		maxHealth: 100,
		age:       50,
		maxAge:    100,
	}
}

func StarvingCreature() *Creature {
	return &Creature{
		num:          2,
		colorCode:    "102",
		nutrition:    5,
		maxNutrition: 100,
		health:       100,
		maxHealth:    100,
		X:            1,
		Y:            1,
	}
}

func AggressiveCreature() *Creature {
	return &Creature{
		num:        3,
		colorCode:  "103",
		aggression: 100,
		X:          1,
		Y:          1,
	}
}

func HealthyCreature() *Creature {
	return &Creature{
		num:          4,
		colorCode:    "104",
		health:       100,
		maxHealth:    101,
		age:          50,
		maxAge:       100,
		nutrition:    101,
		maxNutrition: 100,
		X:            1,
		Y:            1,
	}
}

func HighVisionCreature() *Creature {
	return &Creature{
		num:          5,
		colorCode:    "105",
		vision:       20.11,
		health:       100,
		nutrition:    40,
		maxNutrition: 100,
	}
}

func LowVisionCreature() *Creature {
	return &Creature{
		num:       6,
		colorCode: "106",
		vision:    0.11,
	}
}

func HighVisionHungryCreature() *Creature {
	return &Creature{
		num:          7,
		colorCode:    "107",
		vision:       20.11,
		nutrition:    35,
		maxNutrition: 100,
	}
}

func FedCreature() *Creature {
	return &Creature{
		num:          8,
		colorCode:    "108",
		nutrition:    80,
		maxNutrition: 100,
		health:       20,
		maxHealth:    100,
		metabolism:   12,
		X:            1,
		Y:            1,
	}
}
