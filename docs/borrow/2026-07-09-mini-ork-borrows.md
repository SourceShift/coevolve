# Borrow: harden mini-ork with two proven patterns from Bernstein

Status: **proposed** · Created: 2026-07-09 · Source: [competitive-landscape.md](../competitive-landscape.md)

Two patterns from Bernstein (`chernistry/bernstein`) were verified as *real,
load-bearing code* (not marketing) during the competitive clone-and-analyze pass.
Both map cleanly onto mini-ork's core loop and are the two highest-value steals.

Reference clone (kept): `scratchpad/compare/bernstein/`.

---

## Task 1 — Deterministic zero-LLM-token scheduler

**Status:** not started
**Last worked on:** —
**Why:** Bernstein's orchestrator is a *deterministic scheduler that never calls an
LLM directly* — batching, spawn/retry, and stall detection are pure state-machine
code; only task decomposition and judging touch an LLM
(`src/bernstein/core/orchestration/orchestrator.py:251-256`). This gives byte-identical
reruns (an LLM-call cache keyed by model+prompt+params, `deterministic.py`) and
cuts token spend on coordination to zero. mini-ork currently interleaves LLM calls
into scheduling ticks.

**Subtasks**
1. Audit mini-ork's scheduler loop: enumerate every LLM call on the
   classify → plan → execute → verify → reflect → improve → eval → promote path
   and tag each as `scheduling` vs `decision` (decompose/judge). — *not started*
2. Move all `scheduling`-tagged calls out of the tick loop; the tick becomes pure
   Go/state-machine code. Only `decision` steps may call a lane. — *not started*
3. Add an optional record/replay LLM-call cache keyed by
   (model, prompt, provider, temp, max_tokens) for reproducible reruns. — *not started*
4. Verify: same kickoff → same schedule (assert identical `state.db` run shape on
   a dry-run replay). — *not started*

**Acceptance:** a mini-ork run's scheduling decisions are reconstructable and cost
zero tokens; only decompose/judge steps appear in `llm_calls` as `scheduling`-free.

---

## Task 2 — Janitor-style verify (structural signals + commit-attribution)

**Status:** not started
**Last worked on:** —
**Why:** Bernstein's `janitor.py` verifies task completion with a taxonomy of
*concrete completion signals* — `path_exists`, `glob_exists`, `test_passes`,
`file_contains` — plus `llm_review`/`llm_judge` (Claude Sonnet, 0.7 confidence) as
the *only* LLM-touching verification path, and rejects rubber-stamped or orphaned
diffs via **commit-attribution** (`src/bernstein/.../janitor.py:44-100`). This is
strictly stronger than a pass/fail LLM verify and directly hardens mini-ork's
`verify` stage.

**Subtasks**
1. Define a completion-signal schema for mini-ork recipes: structural checks
   (`path_exists`/`test_passes`/`file_contains`/`glob_exists`) declared per recipe. — *not started*
2. Implement a verify stage that runs structural signals first (cheap, deterministic),
   then falls back to an `llm_judge` lane only when structural signals are
   inconclusive. — *not started*
3. Add commit-attribution: reject a task marked "done" whose diff is empty, orphaned,
   or not attributable to the spawned agent. — *not started*
4. Wire results into `state.db` so Coevolve's Runs/Logs modes surface *why* a task
   passed/failed (which signal fired). — *not started*

**Acceptance:** a mini-ork task cannot be marked complete on an LLM rubber-stamp
alone; verify records which concrete signal(s) satisfied it, visible in the TUI.

---

## Deferred / lower-priority borrows (tracked, not scheduled)

- **ART RULER** pairwise LLM-judge for TraceOtter `eval.py` (lanes judged relative on
  the same task vs rule-based `route_correct`).
- **AgentMemory** RRF fusion (BM25 + vector + graph) + 4-tier consolidation/decay for
  ContextNest.
- **CAO** inbox/status-driven delivery (deliver to a lane only on IDLE/COMPLETED) to
  drop busy-polling in the lane router.
- **Plandex** git-backed plan versioning (real git subprocess per plan) for run history.
