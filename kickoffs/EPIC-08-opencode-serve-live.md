# EPIC-08 — opencode serve/SSE live worker
Status: todo · Depends: 04
## Goal
Live within-run streaming + true abort via long-lived isolated `opencode serve`.
## Deliverables
- coevolve/vendor/opencode/ — vendor omnigent's opencode_native_client.py +
  opencode_native_app_server.py (Apache-2.0, keep NOTICE). Per-session XDG
  isolation + OPENCODE_SERVER_PASSWORD + env allowlist; httpx REST + GET /event SSE
  + /prompt_async + /abort; model pinned per-prompt.
- Use it for mini-ork's implementer node when live streaming is wanted.
## Acceptance
- Live text/tool sub-events stream; abort stops mid-flight + reaps serve process.
