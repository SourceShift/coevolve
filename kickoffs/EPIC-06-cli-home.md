# EPIC-06 — `coevolve run` CLI + live Home renderer
Status: ⬜  ·  Depends: 05

## Goal
The user-facing entrypoint + the live run-flow (Mode 0) as streaming output,
matching the approved preview; provenance-gated (real numbers only).

## Deliverables
- `coevolve/cli.py` — `coevolve run "<task>" [--lane cheap|frontier] [--dry-run]`,
  `coevolve tui`, `coevolve capabilities`, `coevolve runs`, `coevolve stop/resume`.
  (click or argparse; rich for color.)
- `coevolve/render/stream.py` — Home renderer: per-step node·lane·route·cost·
  (savings est.)·duration + recall/reroute/result rows + a run summary
  (router split, session cost). NOT_YET fields shown as "—".
- `pyproject.toml` — coevolve package + `coevolve` console-script + textual/rich deps.

## Acceptance
- `coevolve run --dry-run "<task>"` prints the live flow; the real run
  (`--lane cheap`) executes via opencode and shows real route/cost.
