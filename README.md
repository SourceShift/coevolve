# Coevolve

Extensible CLI that runs the **mini-ork** loop with **opencode** as the LLM
worker, shows the live flow + costs, supports stop/resume, and integrates
**ContextNest** (memory) + **TraceOtter** (learning). All panels render real
data only. Architecture informed by omnigent (Apache-2.0).

mini-ork is reached at runtime via `MINI_ORK_ROOT` (points at a mini-ork checkout).
Build roadmap: `kickoffs/` (10 epics). Contracts: `coevolve/{capabilities,events,registry}.py`, `coevolve/integrations/base.py`.
