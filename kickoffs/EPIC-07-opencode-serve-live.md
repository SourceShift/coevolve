# EPIC-07 — opencode serve/SSE live worker
Status: ⬜  ·  Depends: 04

## Goal
Live within-node streaming + real stop/abort, via a long-lived isolated
`opencode serve`, vendored from omnigent (Apache-2.0, with attribution).

## Deliverables
- `coevolve/vendor/opencode/` — vendored `opencode_native_client.py` (httpx REST
  + `GET /event` SSE + `/prompt_async` + `/abort` + permissions) and
  `opencode_native_app_server.py` (spawn `opencode serve` on loopback + per-session
  XDG isolation + OPENCODE_SERVER_PASSWORD + env allowlist). Keep NOTICE/attribution.
- `coevolve/integrations/opencode_serve.py` — `OpencodeServeWorker(Worker)`:
  Capabilities NATIVE_SERVER, streaming=SSE, interrupt=True, resume=WARM_REATTACH,
  permissions=True, cost_reporting=True (per-message cost from events). Maps
  opencode part-events → normalized Events; model pinned per-prompt.

## Acceptance
- A run streams live text/tool sub-events; stop() aborts mid-flight and reaps the
  serve process; cost badge updates from real per-message usage.
