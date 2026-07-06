# EPIC-04 — Core integrations
Status: ⬜  ·  Depends: 03

## Goal
Concrete backends implementing the role ABCs + self-registering + declaring
capabilities + honest health().

## Deliverables (`coevolve/integrations/`)
- `miniork.py` — `MiniOrkOrchestrator(Orchestrator)`: run() emits NODE events;
  stop/resume/rollback via seams. Capabilities: ORCHESTRATOR, interrupt=True,
  resume=COLD_ONLY(+WARM later).
- `contextnest.py` — `ContextNestMemory(Memory)`: recall/write_outcome.
- `traceotter.py` — `TraceOtterLearning(Learning)`: distill.
- `router.py` — `LaneRouterIntegration(Router)`: pick_lane/recompute.
- `opencode.py` — `OpencodeWorker(Worker)`: batch mode (cl_opencode.sh),
  Capabilities.models = authed opencode models; cost_reporting=True.

## Acceptance
- `coevolve capabilities` lists all 5 with correct kinds; health() reflects real
  state (CN offline → DEGRADED, opencode missing → unavailable).
