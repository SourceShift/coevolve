# Competitive Landscape

How the **Coevolve mixture** — mini-ork (orchestrator) · ContextNest (memory) ·
TraceOtter (learning) · Coevolve (TUI control plane) — compares to the closest
projects in the field. Grounded in reading the **actual source** of six cloned
repos, one mapped to each component of the mixture (2026-07).

Method: Context7 library discovery + jina web search surfaced candidates; six
repos were shallow-cloned and analyzed head-to-head against the matching
component by parallel agents reading their real code (file:line cited in the
per-repo notes below).

---

## Scoreboard

| Your component | Closest analog | Stack / size | Overlap | Verdict |
|---|---|---|---|---|
| **mini-ork** (orchestrator) | **Bernstein** | Python, 554K LOC | **~55–60%** | Building-block **and** competitor |
| mini-ork (alt) | AWS CAO (`awslabs/cli-agent-orchestrator`) | Python, 39K LOC | ~20–25% | Complement |
| **ContextNest** (memory) | **AgentMemory** (`rohitg00/agentmemory`) | TypeScript, 38K LOC | **~55%** | Complement / competitor |
| **TraceOtter** (learning) | **OpenPipe ART** | Python, 66K LOC | ~10–15% | Complement (orthogonal) |
| TraceOtter / thesis | SIA (`hexo-ai/sia`) | Python, 9K LOC | ~15–20% | Complement (aspirational) |
| **Whole system** | **Plandex** | Go + Postgres + Bubbletea | ~35–40% | Closest shipped competitor |

---

## The three findings that matter

### 1. Your orchestrator has a real rival — Bernstein

Bernstein independently arrived at nearly the same shape (plan → execute →
verify → lanes → recipes) and is *more mature* on the axes it shares:

- **Deterministic zero-LLM-token scheduler** — orchestration logic (batching,
  spawn/retry, stall detection) is pure state-machine code; only task
  decomposition and judging call an LLM. Verified in code, not marketing.
- **Janitor verify** — rejects rubber-stamped diffs via completion signals
  (`path_exists`, `test_passes`, `file_contains`) + `llm_judge`, plus
  commit-attribution to reject empty/orphaned diffs.
- Cascade routing (opus → sonnet → codex → gemini → qwen), Ed25519-signed audit
  journals for compliance, and **file-based state** (a deliberate ADR rejecting
  SQLite — the opposite of mini-ork's `state.db`).

**What it lacks is exactly the mixture's moat: no GRPO/learning loop, no
ContextNest-grade memory.**

### 2. The learning axis is still uncontested — for a subtle reason

- **OpenPipe ART** is the only project with a *real* GRPO trainer — but it trains
  **one model's weights** (GPU + vLLM + LoRA). The mixture's actual problem
  (TraceOtter → Coevolve's `RewardByLane`) is **lane-routing-policy learning
  across externally-owned frontier models** — you don't own opus/sonnet's weights
  to train them. So ART is *orthogonal, not competitive.* Do **not** wrap it.
- **SIA**'s "harness + weights" self-improvement turned out to **delegate RL to an
  external Tinker API** rather than compute gradients — the thesis collision is
  partly aspirational.

**Nobody combines a learning loop with orchestration + memory + TUI + cost.**
That is the whole-mixture moat, confirmed by reading code.

### 3. Honest internal caveat about *our* side

TraceOtter today does **not** *run* GRPO — it exports verl-formatted datasets and
only does SFT via `llamafactory-cli`. Coevolve's "GRPO reward-by-lane"
(`internal/seams/db.go` → `RewardByLane`) is a **SQL read of a precomputed
`reward_g` column**. So the current "learning loop" is reward-curation + a
routing-policy read, not a live training loop.

This is the *right* design for lane-routing — but the "GRPO learning loop"
framing overstates what is wired end-to-end. ART shows what a real in-repo GRPO
trainer looks like if that is ever wanted.

---

## Borrow list (ranked by value)

1. **Bernstein — deterministic zero-token scheduler.** Tighten mini-ork so
   scheduling ticks never call an LLM; only classify/plan/judge do. Repro + cost win.
2. **Bernstein — janitor verify taxonomy.** Structural signals + `llm_judge` +
   **commit-attribution** to reject empty/orphaned diffs. Stronger than pass/fail.
3. **ART — RULER** (LLM-as-judge, group-relative pairwise scoring, no labels).
   TraceOtter's `eval.py` could judge lanes pairwise on the same task instead of
   rule-based `route_correct`.
4. **AgentMemory — RRF fusion** (BM25 + vector + graph) + **4-tier consolidation/
   decay** (working → episodic → semantic → procedural). ContextNest could add BM25
   as a cheap fallback and express basins as episodic → semantic tiers.
5. **CAO — inbox/status-driven delivery** (deliver to a lane only on
   IDLE/COMPLETED transition, pub/sub). Avoids busy-polling in the lane router.
6. **Plandex — git-backed plan versioning** (shell out to real git per plan for
   history/rewind/branch). Lighter than a custom event log for run history.

---

## Threat assessment

- **Bernstein** — the only real competitor to a *component* (mini-ork) and markets
  in the same category. You win on learning + memory; it out-matures the
  orchestration core. **Biggest watch item.**
- **Plandex** — closest *whole-system* shipped product (same Go + Bubbletea stack,
  persistent plan + roles + routing + cost). **No learning loop, no dedicated
  memory service** — it validates the architecture while leaving both
  differentiators open.
- **AgentMemory** — overlaps ContextNest most, but its closed "iii engine" storage
  makes it un-adoptable and non-transparent. ContextNest's Rust/SQLite + basins +
  inbox are defensible.

**Net:** no single project — and no combination — replicates the full mixture. The
defensible moat is the **integration** of a routing-policy learning loop + a
transparent memory service + orchestrator + real-data TUI. The biggest honest
risks are (a) Bernstein out-maturing mini-ork's orchestration core, and (b) the
learning loop being less "live GRPO" than the pitch implies.

---

## Appendix — repos analyzed

| Repo | URL | Maps to |
|---|---|---|
| OpenPipe ART | github.com/OpenPipe/ART | TraceOtter |
| SIA | github.com/hexo-ai/sia | thesis (harness + weights) |
| AWS CAO | github.com/awslabs/cli-agent-orchestrator | mini-ork |
| AgentMemory | github.com/rohitg00/agentmemory | ContextNest |
| Bernstein | github.com/chernistry/bernstein | mini-ork (verify) |
| Plandex | github.com/plandex-ai/plandex | whole system |
