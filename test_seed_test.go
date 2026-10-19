package main

// init seeds deterministic RNG for tests. Adjust seed if needed.
// This ensures all tests using randomness are reproducible.
func init() {
	ResetRand(1)
}
