# EPIC-02 — mini-ork MCP server (opencode spawns mini-ork)
Status: todo · Depends: 01
## Goal
An MCP server exposing mini-ork orchestration as tools so **opencode can spawn
mini-ork**. This is the central enabler of the whole CLI.
## Deliverables (coevolve/mcp/)
- server.py — an MCP server (stdio) exposing tools:
  - `mo_run(task, lane?, dry_run?)` → starts a mini-ork run on the caller's cwd;
    returns run_id + streams/collects NODE events.
  - `mo_status(run_id)`, `mo_list(limit)`, `mo_stop(run_id)`, `mo_resume(run_id)`,
    `mo_rollback(run_id)`.
  - `mo_classify(task)` (cheap, dry-run) for a quick route preview.
- Backed by the EPIC-01 integrations (MiniOrkOrchestrator + seams). Reaches
  mini-ork via MINI_ORK_ROOT; runs target the CALLER's repo (MO_TARGET_CWD).
- Register the server so `opencode mcp` can load it.
## Acceptance
- `opencode` configured with this MCP can call `mo_run("add --version to foo.sh")`
  and a real mini-ork run executes; `mo_stop`/`mo_resume` work; tools return typed JSON.
