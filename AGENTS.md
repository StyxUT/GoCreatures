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
Randomness: use central RNG wrappers (`RandIntn`, `RandFloat64`); seed once in `init`. For deterministic tests inject with `SetRand(rand.New(rand.NewSource(seed)))` or call `ResetRand(seed)`.
Mutations: prefer pure helpers; minimize global state (current globals: `safeRunes`, `nextRuneIndex`, `paused`, `exitCh`, `isTest`).
Collections: reuse slices via reslicing (pattern already in environment filtering & observation).
Stats: update `TickStats` only inside simulation tick / action resolvers to keep ordering deterministic.
Add missing helpers symmetrically (`max(int,int)`) before using; clamp nutrition with `min`, not `max` (current bug in `resolveEat`).
Tests: structure Arrange/Act/Assert blocks; avoid randomness inside tests or guard with deterministic seeding.
No Cursor / Copilot rule files present; if added, mirror key rules here.
Future DB/reporting: isolate I/O behind interface; do not import DB packages into core simulation loop.

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:7510c1e2 -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md for details and anti-patterns.

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
