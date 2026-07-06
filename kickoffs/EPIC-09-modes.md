# EPIC-09 — The 8 modes (real-data-only)
Status: ⬜  ·  Depends: 08

## Goal
All modes from the design, each bound to real seams; anything needing the
not-yet-trained model rendered as an explicit "not available yet" / "offline".

## Deliverables (`coevolve/tui/modes/` + adapters)
- `home.py` — live run stream + right rail (pipeline/router-split/memory/cost).
- `runs.py` — runs table + spawn trees + scheduler counters; **stop/resume/rollback actions**.
- `cost.py` — session/today/lifetime (real llm_calls), budget caps, spend-by-provider; savings labeled est.
- `router.py` — lane share/$/lat/reward_g (real); parity/win% = NOT_YET.
- `contextnest.py` — basins/capsule/inbox/sessions/graph (real or offline chip).
- `topology.py` — node DAG + lane→provider + provider health.
- `learning.py` — reward_g by lane (real); retrain-history/adoption = NOT_YET.
- `logs.py` — tail agent stream + llm_calls + run artifacts.

## Acceptance
- Every panel shows measured data or an honest NOT_YET/DEGRADED marker; runs
  can be stopped/resumed/rolled-back from the Runs mode.
