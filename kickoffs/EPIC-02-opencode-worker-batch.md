# EPIC-02 — opencode worker (batch) + lane registration
Status: 🟡 partial  ·  Depends: 01

## Goal
opencode is a selectable mini-ork worker lane; its traces feed TraceOtter.

## Deliverables
- `lib/providers/cl_opencode.sh` ✅ — executable wrapper + shim (opencode json →
  claude-stream-json rows + .tokens/.turns.jsonl/.cost sidecars). Verified:
  TraceOtter parser reads real tokens (19,841 in / 3 out) through it.
- ⬜ `config/providers.yaml` — add `opencode:` `{kind: executable, family:
  opencode, script: lib/providers/cl_opencode.sh, model: <authed model>}`.
- ⬜ `config/agents.yaml` — add lane `worker_opencode: opencode`; document
  MO_CHEAP_LANE / MO_FRONTIER_LANE / MO_OPENCODE_MODEL knobs.

## Acceptance
- `MINI_ORK_DRY_RUN=0` node dispatch with lane=opencode produces
  `llm_calls.provider='opencode'` + a shim-written agent-<node>.stream.jsonl.
- No edits to lib/llm-dispatch.sh (kind:executable + wrapper-wins precedence).
