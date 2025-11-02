# AGENTS.md
Build / Run: `go run .` (Go 1.24)
Tests (all): `go test ./...`
Single test: `go test -run '^TestName$' -count=1 ./...`
Race / benchmarks: `go test -race ./...`, `go test -bench . ./...`
Lint/format: `go fmt ./...`; `go vet ./...`; optional: `staticcheck ./...` if installed.
Imports: group blank lines (stdlib, third-party, local), use `goimports`.
Naming: camelCase for locals/fields; export only if cross-package later; keep test helpers in `_test.go` when possible.
Types: favor built-in `int` unless size matters; keep struct fields contiguous logically (identity, genetics, stats).
Errors: return `error` values; avoid panic except unrecoverable terminal/raw mode setup.
Concurrency: only per‑creature decision goroutines; no shared mutation inside them; join before resolution.
Atomics: use `atomic` only for pause flags; otherwise channel or mutex; avoid data races (run `go test -race`).
Randomness: seed once in `init`; for deterministic tests consider injecting `*rand.Rand`.
Mutations: prefer pure helpers; minimize global state (current globals: `safeRunes`, `nextRuneIndex`, `paused`, `exitCh`, `isTest`).
Collections: reuse slices via reslicing (pattern already in environment filtering & observation).
Stats: update `TickStats` only inside simulation tick / action resolvers to keep ordering deterministic.
Add missing helpers symmetrically (`max(int,int)`) before using; clamp nutrition with `min`, not `max` (current bug in `resolveEat`).
Tests: structure Arrange/Act/Assert blocks; avoid randomness inside tests or guard with deterministic seeding.
No Cursor / Copilot rule files present; if added, mirror key rules here.
Future DB/reporting: isolate I/O behind interface; do not import DB packages into core simulation loop.
