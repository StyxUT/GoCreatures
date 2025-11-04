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
