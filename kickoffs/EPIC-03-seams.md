# EPIC-03 — Seams layer
Status: ⬜  ·  Depends: 01

## Goal
Thin, typed adapters over the existing bash/python seams — the only place that
knows how to talk to mini-ork's CLIs, DB, CN, router, traceotter. Everything
above uses these, never raw subprocess.

## Deliverables (`coevolve/seams/`)
- `run.py` — classify / plan / execute / verify / resume / rollback (env:
  MINI_ORK_ROOT/HOME/DB, MINI_ORK_PLAN_PATH, MINI_ORK_RUN_DIR, MO_GIVEN_PLAN,
  MINI_ORK_DRY_RUN). Parse key=value stdout.
- `contextnest.py` — cn_capsule/cn_retrieve/cn_basins/cn_inbox/cn_sessions_*/
  cn_outcome_post (source lib/cn_client.sh OR import mini_ork.cn_client);
  graceful-degrade to offline.
- `router.py` — lane_router_preferred_lane / lane_router_recompute_advantages.
- `db.py` — read-only sqlite over MINI_ORK_DB: llm_calls, execution_traces,
  run_artifacts (cost sums, reward_g, spend-by-provider, per-run rows).
- `traceotter.py` — bin/mini-ork-traceotter --json (distilled counts/skills).
- `topology.py` — bin/mini-ork-topology (node DAG + lane→provider).
- `health.py` — cn_available / opencode presence / provider reachability.

## Acceptance
- Each seam has a unit smoke (dry-run/offline-safe); returns typed data or a
  DEGRADED marker; no exceptions when a backend is down.
