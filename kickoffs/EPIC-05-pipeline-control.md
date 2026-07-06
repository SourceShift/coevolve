# EPIC-05 — Pipeline + run controller
Status: ⬜  ·  Depends: 04

## Goal
Assemble the spine and expose run control (start/stop/resume/rollback/list),
using omnigent's bridge-state + supervisor-reaps-on-teardown pattern.

## Deliverables
- `coevolve/pipeline.py` — order: classify → cn recall → router lane → plan
  (or MO_GIVEN_PLAN) → execute (opencode worker) → verify → cn_outcome_post →
  router.recompute → traceotter distill. Each stage yields normalized Events.
- `coevolve/control.py` — `RunController`: start (spawn+supervise), stop
  (interrupt+reap, no orphans), resume, rollback, list_runs; bridge-state file
  with last_event_id cursor.

## Acceptance
- `coevolve run --dry-run "<task>"` emits ordered NODE events end-to-end (no
  spend). stop() during a run reaps the worker with no orphaned process.
