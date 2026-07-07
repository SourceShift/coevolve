# Bug Audit: coevolve Go codebase

**Task class:** bug_audit_coevolve
**Target:** /Volumes/docker-ssd/ps/coevolve (self — dogfood)
**Mode:** report-only · no code mutation

## Objective

Find bugs, logic errors, race conditions, nil-pointer risks, resource leaks,
incorrect SQL queries, and architectural regressions in the coevolve Go
codebase. This is a **read-only audit** — produce a ranked report, do not edit
any file.

## Scope

All `.go` files under:
- `cmd/coevolve/main.go`
- `internal/core/`
- `internal/run/`
- `internal/seams/`
- `internal/tui/`

## Lenses

1. **kimi (contract-violation lens)** — check every Go interface implementation
   against its contract: does `seams` package return `(zero, false)` correctly
   on all error paths? Do all `core.Integration` implementations satisfy their
   interface? Are there nil-interface returns, unhandled errors, or resource
   leaks (deferred Close/Unlock)?

2. **minimax (user-impact lens)** — what bugs would a user actually hit? Race
   conditions in the TUI (concurrent map writes, stale data), nil-pointer
   dereferences in mode views, incorrect cost display, broken /run flow,
   history file corruption, SSE stream hangs.

3. **codex (logic-error lens)** — SQL query correctness (wrong columns, missing
   WHERE clauses, incorrect aggregation), off-by-one in truncation/bar rendering,
   incorrect ANSI stripping, race conditions in miniOrkEnricher, incorrect
   context cancellation in serveWorker.

4. **opus (arch-regression lens)** — does the code still match the architecture
   described in the README, INDEX.md, and package doc comments? Broken
   extensibility contracts, incorrect interface implementations, provenance
   honesty violations, resource leaks (deferred Close/Unlock), dead code.

## Output

A single ranked report with:
- Bug ID, severity (critical/high/medium/low), file:line
- Description of the bug
- Evidence (code excerpt)
- Recommended fix (one sentence)
