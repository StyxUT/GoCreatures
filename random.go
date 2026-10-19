package main

import (
	"math/rand"
	"time"
)

// Central RNG instance to allow deterministic injection in tests. Simulation
// draws belong to the tick goroutine; terminal input only queues requests.
var rng *rand.Rand

func init() {
	if rng == nil { // seed once
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
}

// SetRand allows tests or callers to inject a deterministic RNG.
func SetRand(r *rand.Rand) { rng = r }

// ResetRand resets the RNG to a new determinstic seed.
func ResetRand(seed int64) { rng = rand.New(rand.NewSource(seed)) }

// RandIntn proxies to the central RNG.
func RandIntn(n int) int { return rng.Intn(n) }

// RandFloat64 proxies to the central RNG.
func RandFloat64() float64 { return rng.Float64() }
