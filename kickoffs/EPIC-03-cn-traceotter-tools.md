# EPIC-03 — ContextNest + TraceOtter MCP tools
Status: todo · Depends: 02
## Goal
Expose memory + learning to opencode (and to mini-ork runs) as MCP tools.
## Deliverables (coevolve/mcp/)
- `cn_recall(query, limit)`, `cn_write(outcome, atom_ids, evidence)` (ContextNestMemory).
- `traceotter_distill(runs_dir?)`, `traceotter_stats()` (TraceOtterLearning).
- All degrade gracefully (CN offline → explicit DEGRADED result, never fake).
## Acceptance
- From opencode, `cn_recall("webhook idempotency")` returns real hits or an
  offline marker; `traceotter_stats()` returns the real distilled counts.
