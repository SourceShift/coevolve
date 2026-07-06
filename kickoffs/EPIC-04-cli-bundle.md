# EPIC-04 — Coevolve CLI bundle (opencode as the shell)
Status: todo · Depends: 02,03
## Goal
Package opencode AS the Coevolve CLI: configured to load the mini-ork+CN+TraceOtter
MCP, branded, with mini-ork using cl_opencode.sh for its own node dispatch.
## Deliverables
- `coevolve/cli.py` — `coevolve` launcher: writes/points an opencode config that
  registers the MCP server(s), sets sane defaults, then execs `opencode` (or
  `opencode run` for headless). Subcommands: `coevolve tui|run|runs|cost|capabilities`.
- opencode config template (per-project XDG, isolated) enabling the MCP tools.
- Wire mini-ork's own implementer lane to cl_opencode.sh (config in mo-fix) so
  spawned mini-ork runs also use opencode as worker.
- pyproject console-script `coevolve`.
## Acceptance
- `coevolve` opens opencode with `mo_run`/`cn_recall`/… available as tools.
