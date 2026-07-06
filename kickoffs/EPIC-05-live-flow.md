# EPIC-05 — Live-flow surfacing
Status: todo · Depends: 02,04
## Goal
Show the mini-ork run unfolding live (the design's Home stream) while opencode
has spawned it — node·lane·route·cost·parity·duration + recall/reroute/result.
## Deliverables
- Stream NODE events from a spawned mini-ork run back into the opencode surface
  (tool-result progress) AND a standalone `coevolve watch <run_id>` / `coevolve run`
  renderer (render/stream.py) tailing .live.log + DB. Provenance-gated (real only).
## Acceptance
- `coevolve run "<task>"` (or mo_run from opencode) shows the live flow with real
  route/cost as nodes complete.
