# EPIC-01 — Extensibility foundation (MCP backend)
Status: done · Depends: none
## Goal
Capability-declaring plugin registry + normalized events + provenance so the
mini-ork MCP server and every tool/mode is additive. Also serves as the MCP
server's backend (integrations become tools).
## Deliverables (shipped, verified 2026-07-06)
- coevolve/capabilities.py, events.py (Provenance/"all real data"),
  integrations/base.py (Orchestrator/Memory/Learning/Worker/Router ABCs),
  registry.py (@register + entry-points).
## Acceptance (met)
- Smoke: worker self-registers, core resolves by role+capability, not_yet Value flagged non-real.
