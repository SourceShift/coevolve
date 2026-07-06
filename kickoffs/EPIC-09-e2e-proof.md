# EPIC-09 — End-to-end proof
Status: todo · Depends: 05,06,07
## Goal
The single demo: inside opencode, spawn mini-ork on a real task; watch the live
flow; see ContextNest recall + TraceOtter distill + real cost; stop/resume.
## Acceptance (honest, all real)
- opencode calls mo_run → mini-ork: task_class + plan.json + runs/<id>/ +
  execution_traces.reward_g + llm_calls (provider reflects the worker).
- cn_recall returned real atoms; cn_write recorded the outcome (or explicit offline).
- traceotter_stats distilled count includes this run.
- Live flow rendered with real route/cost; NOT_YET panels labeled, never faked.
- A run stopped + resumed successfully.
