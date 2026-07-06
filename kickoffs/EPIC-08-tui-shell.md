# EPIC-08 — Textual TUI shell
Status: ⬜  ·  Depends: 06

## Goal
The full-screen 8-mode TUI shell with a mode registry (modes plug in like
integrations), status bar, command palette, keybindings.

## Deliverables (`coevolve/tui/`)
- `app.py` — Textual App; digit bindings 0–7; status bar (run·lane·session
  cost·CN health·throttles); prompt row (submit a new run).
- `modes/base.py` — `Mode` base + `@register_mode` (same registry idiom).
- `palette.py` — ⌘K / `:` command palette (the ~38 mini-ork subcommands).
- `keys.py` — `?` keybindings overlay.

## Acceptance
- `coevolve tui` launches, switches modes by digit, status bar updates live,
  palette opens and can trigger a run.
