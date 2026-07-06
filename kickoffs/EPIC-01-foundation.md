# EPIC-01 — Extensibility foundation
Status: ✅ done  ·  Depends: none

## Goal
A capability-declaring plugin registry + normalized event model + provenance so
every backend and UI mode is additive (no core edits to extend).

## Deliverables (shipped)
- `coevolve/capabilities.py` — Capabilities + IntegrationMode/Resume/Streaming/Kind.
- `coevolve/events.py` — Event envelope + DataOrigin/Provenance/Value ("all real data").
- `coevolve/integrations/base.py` — role ABCs (Orchestrator/Memory/Learning/Worker/Router).
- `coevolve/registry.py` — @register decorator + entry-point discovery.
- `coevolve/__init__.py`.

## Acceptance (met)
- Smoke: a Worker self-registers, core resolves by role+capability, a `not_yet`
  Value is flagged non-real. Verified 2026-07-06.
