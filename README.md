# GoCreatures

## Purpose
Personal project primarily for learning the Go programming language through a small autonomous creature simulation. Code favors clarity and experimentation over production features.

## What It Does (High Level)
A tick‑based simulation where creatures (runes) take actions (move, eat, reproduce, etc.). Actions are resolved deterministically per tick given the RNG state. Statistics are collected for reporting and tests assert behavior under controlled seeds.

## Running
```bash
go run .
```
Uses Go 1.24 (per local setup). The main program runs a simulation loop until exit.

## Tests
Run all tests:
```bash
go test ./...
```
Single test (example):
```bash
go test -run '^TestResolveEat$' -count=1 ./...
```
Race detector:
```bash
go test -race ./...
```
Benchmarks (if any added later):
```bash
go test -bench . ./...
```

## Deterministic Randomness
Central RNG wrappers (`RandIntn`, `RandFloat64`). For deterministic tests use `ResetRand(seed)` or inject with `SetRand`. Avoid ad‑hoc `rand.*` calls elsewhere.

Decisions run sequentially in row-major cell order, preserving creature order
within each cell. All decisions are collected before action resolution. This
keeps random draw allocation and competing intents reproducible without shared
RNG access from goroutines. Replay requires the same initial world, seed and
ordered spawn requests; create a fresh `CreatureFactory()` for each world.

## Tick and creation contracts

- At tick entry, remove previously dead creatures without counting another death,
  then remove living creatures already at their maximum age as age deaths.
- Apply aging, nutrition and health; remove starvation deaths before decisions.
  Resolve fights, eating, cloning, movement and sleeping in that order. Only
  living actors and living fight targets are eligible; a kill is credited once.
- After actions, remove dead creatures and compute population, age, health and
  nutrition totals from the survivors, including newborn clones. `TickStats` is
  reset per call (preserving `Tick`); carnivore totals remain cumulative.
- Reporting reads completed statistics. `HealthAdded` retains its existing
  attempted-healing meaning; `Fights` counts distinct participants, while
  cumulative carnivore fights count participation in executed fights.
- The creature creation module owns diet-derived and inherited traits, IDs and
  rune allocation. `createForDiet` creates one ancestor directly; unsupported
  diets return nil without consuming creation state. `create(parent)` creates
  inherited clones (or a random-diet ancestor for nil). Use one factory per world.

Domain terms are defined in [CONTEXT.md](CONTEXT.md).

## Code Layout (Selected Files)
- `main.go` – entry point & simulation startup.
- `creatures.go` – creature definitions & state.
- `actions.go` / `actions_resolution_test.go` – action definitions and resolution logic.
- `random.go` / `rng_test.go` – RNG wrapper & tests.
- `runes.go` / `runes_test.go` – rune handling utilities.
- `reporting.go` / `reporting_test.go` – stats & reporting helpers.
- `helpers.go` / tests – small pure helper functions.

## Style & Conventions
Follow simple Go idioms: prefer pure helpers, avoid global mutable state (only a few controlled globals), group imports (stdlib / third‑party / local), keep tests deterministic.

## Contributing / Scope
Personal learning sandbox.

---
This README intentionally brief; deeper docs may evolve with the project.
