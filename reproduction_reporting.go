package main

const (
	herbivoreReproductionIndex = iota
	carnivoreReproductionIndex
	reproductionDietCount
)

type ReproductionStats struct {
	Evaluated            int
	Eligible             int
	NutritionBlocked     int
	HealthBlocked        int
	AgeBlocked           int
	CooldownBlocked      int
	ImmobilityBlocked    int
	ProbabilityChecks    int
	ProbabilityRejected  int
	OtherActionSelected  int
	CloneIntents         int
	ResolutionSuppressed int
	Births               int
}

type reproductionDecisionState struct {
	considered bool
	passed     bool
}

func tryReproduce(creature *Creature, env [][]environment, healthPercent, agePercent float64) bool {
	if !canReproduce(creature, env, healthPercent, agePercent) {
		return false
	}
	probability := reproductionProbability(creature, env)
	creature.reproductionDecision = reproductionDecisionState{
		considered: true,
		passed:     RandFloat64() < probability,
	}
	return creature.reproductionDecision.passed
}

func recordReproductionDecision(creature *Creature, intent ActionIntent, stats *TickStats) {
	if creature == nil || stats == nil {
		return
	}
	diet := reproductionDietIndex(creature)
	current := &stats.Reproduction[diet]
	current.Evaluated++
	blockers := reproductionBlockers(creature)
	current.NutritionBlocked += blockers.NutritionBlocked
	current.HealthBlocked += blockers.HealthBlocked
	current.AgeBlocked += blockers.AgeBlocked
	current.CooldownBlocked += blockers.CooldownBlocked
	current.ImmobilityBlocked += blockers.ImmobilityBlocked
	if blockers.total() != 0 {
		return
	}
	current.Eligible++
	if creature.reproductionDecision.considered {
		current.ProbabilityChecks++
		if !creature.reproductionDecision.passed {
			current.ProbabilityRejected++
		}
	}
	if intent.Type == "clone" {
		current.CloneIntents++
		return
	}
	if creature.reproductionDecision.considered {
		return
	}
	current.OtherActionSelected++
}

type reproductionBlockerCounts struct {
	NutritionBlocked  int
	HealthBlocked     int
	AgeBlocked        int
	CooldownBlocked   int
	ImmobilityBlocked int
}

func (blockers reproductionBlockerCounts) total() int {
	return blockers.NutritionBlocked + blockers.HealthBlocked + blockers.AgeBlocked + blockers.CooldownBlocked + blockers.ImmobilityBlocked
}

func reproductionBlockers(creature *Creature) reproductionBlockerCounts {
	var blockers reproductionBlockerCounts
	if creature == nil {
		return blockers
	}
	minimumNutrition := 0
	if creature.maxNutrition > 0 {
		minimumNutrition = int(float64(creature.maxNutrition) * (cloneEnergyFraction + cloneNutritionReserve))
	}
	if creature.maxNutrition <= 0 || creature.nutrition < minimumNutrition {
		blockers.NutritionBlocked = 1
	}
	if creature.maxHealth <= 0 || float64(creature.health)/float64(max(creature.maxHealth, 1))*100 <= 70 {
		blockers.HealthBlocked = 1
	}
	if creature.maxAge <= 0 {
		blockers.AgeBlocked = 1
	} else {
		agePercent := float64(creature.age) / float64(creature.maxAge) * 100
		if agePercent <= 20 || agePercent >= 90 {
			blockers.AgeBlocked = 1
		}
	}
	if creature.cloneCooldown > 0 {
		blockers.CooldownBlocked = 1
	}
	if creature.immobileTicks > 0 {
		blockers.ImmobilityBlocked = 1
	}
	return blockers
}

func reproductionDietIndex(creature *Creature) int {
	if creature.isCarnivore() {
		return carnivoreReproductionIndex
	}
	return herbivoreReproductionIndex
}

func addReproductionStats(total *ReproductionStats, tick ReproductionStats) {
	total.Evaluated += tick.Evaluated
	total.Eligible += tick.Eligible
	total.NutritionBlocked += tick.NutritionBlocked
	total.HealthBlocked += tick.HealthBlocked
	total.AgeBlocked += tick.AgeBlocked
	total.CooldownBlocked += tick.CooldownBlocked
	total.ImmobilityBlocked += tick.ImmobilityBlocked
	total.ProbabilityChecks += tick.ProbabilityChecks
	total.ProbabilityRejected += tick.ProbabilityRejected
	total.OtherActionSelected += tick.OtherActionSelected
	total.CloneIntents += tick.CloneIntents
	total.ResolutionSuppressed += tick.ResolutionSuppressed
	total.Births += tick.Births
}
