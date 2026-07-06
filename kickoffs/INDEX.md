# Coevolve CLI — Epic Roadmap

The extensible CLI that runs the mini-ork loop with **opencode** as the LLM
worker, shows the **live flow + costs**, supports **stop/resume**, and
integrates **ContextNest** (memory) + **TraceOtter** (learning). All panels
render **real data only** (provenance-typed; nothing aspirational shown as real).
Architecture informed by omnigent (Apache-2.0) — see `docs` in each epic.

Build order is sequential; each epic is an independently verifiable deliverable.

| # | Epic | Status |
|---|------|--------|
| 01 | Extensibility foundation | ✅ done |
| 02 | opencode worker (batch) + lane registration | 🟡 partial (wrapper done; lane reg pending) |
| 03 | Seams layer | ⬜ not started |
| 04 | Core integrations (Orchestrator/Memory/Learning/Router/Worker) | ⬜ |
| 05 | Pipeline + run controller (stop/resume/rollback) | ⬜ |
| 06 | `coevolve run` CLI + live Home renderer | ⬜ |
| 07 | opencode serve/SSE live worker (vendor omnigent client) | ⬜ |
| 08 | Textual TUI shell (modes registry, palette, keybindings) | ⬜ |
| 09 | The 8 modes (real-data-only) | ⬜ |
| 10 | End-to-end proof + docs | ⬜ |

Canonical contracts (do not fork): `coevolve/capabilities.py`,
`coevolve/events.py`, `coevolve/integrations/base.py`, `coevolve/registry.py`.
Design source of truth: `/Volumes/docker-ssd/Migration/Development/researcher/tmp/Coevolve Console overview/Coevolve CLI.dc.html`.
