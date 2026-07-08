# Coevolve

A terminal control plane for **agentic software delivery** — an 8-mode Bubble Tea
dashboard that drives the [mini-ork](https://) orchestration loop with
**opencode** as the LLM worker, and renders the live flow, costs, memory, and
learning in one place. Every panel shows **real data only** (provenance-gated —
never mock numbers).

[![Go](https://img.shields.io/badge/go-1.25-00ADD8.svg)](go.mod)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.0.1-green.svg)](cmd/coevolve/main.go)
[![TUI](https://img.shields.io/badge/tui-Bubble%20Tea-8B7CD8.svg)](https://github.com/charmbracelet/bubbletea)

---

## What it is

Coevolve is the extensible CLI over four cooperating subsystems:

| Role           | Integration     | What it provides                          |
| -------------- | --------------- | ----------------------------------------- |
| Orchestrator   | **mini-ork**    | the classify→plan→execute→verify loop     |
| Worker         | **opencode**    | the LLM engine that does the actual work  |
| Memory         | **ContextNest** | run capsules, basins, the knowledge graph |
| Learning       | **TraceOtter**  | the usage + research learning flywheel    |
| Router         | lane router     | picks a model lane (opus/sonnet/codex/…)  |

You type a task in the **Home** command line; plain text goes to the configured
main LLM (opencode) and streams back like Claude/opencode, while `/run <task>`
(or any task that names mini-ork) spawns a full orchestration you watch node by
node — with per-node cost, lane, and status pulled straight from mini-ork's
`state.db`.

## Requirements

Coevolve is a **front-end over a stack** — it doesn't do the work itself, it
drives and visualizes the subsystems below. On its own the binary builds and
launches, but every panel will read "offline" until you wire in its sources.

**To build**

- **Go 1.25+**

**Required at runtime** (without these, Coevolve has nothing to drive)

- **[opencode](https://opencode.ai)** CLI, authenticated — the LLM worker for
  both chat and the implementer lane. Coevolve inherits your
  `~/.local/share/opencode/auth.json` and env keys (OpenAI/Cloudflare/etc.).
- **[mini-ork](https://)** checkout — the orchestrator. `/run` shells out to it,
  and its `state.db` is the source for the **Runs, Router, Topology, Cost, Logs,
  and Learning** panels. Point Coevolve at it with `MINI_ORK_ROOT`.

**Optional — enables specific modes**

- **ContextNest** — a separate HTTP service (Rust/axum) that backs the
  **ContextNest** mode (capsules · basins · graph). Coevolve talks to it over
  REST at `CN_BASE_URL` (default `http://127.0.0.1:28080`); when it's down, mode
  4 renders "ContextNest offline" and the rest of the app is unaffected.
- **TraceOtter** — the learning/GRPO layer. Its signal (rewards by lane, objective
  domains) rides in mini-ork's `state.db`, so the **Learning** mode lights up as
  soon as mini-ork has recorded traces — no extra service to run.

> **In short:** you need **opencode** + a **mini-ork** checkout to get real work
> and data; **ContextNest** is an add-on for the memory mode; **TraceOtter** data
> comes for free with mini-ork.

## Installation

```bash
git clone https://github.com/sourceshift/coevolve.git
cd coevolve
go build -o bin/coevolve ./cmd/coevolve
```

Then run the binary directly, or `go install ./cmd/coevolve` to put `coevolve` on
your `PATH`.

## Quick Start

```bash
# point at your mini-ork checkout, then launch the dashboard
export MINI_ORK_ROOT=/path/to/mini-ork
./bin/coevolve
```

In the TUI:

1. Land on **Home** (mode `0`). Type a request and press Enter.
2. Plain prose → streams from the main LLM. `/run <task>` → mini-ork orchestration.
3. `/run` opens an **interactive pre-flight**: pick the recipe, a hard budget cap,
   and dry-run vs. live, then confirm. The loop renders inline.
4. Switch modes with digits `0`–`7` (or `←`/`→`) to inspect runs, cost, memory,
   topology, and logs.

## Commands

```
coevolve [tui]          launch the 8-mode dashboard (default)
coevolve capabilities   list registered integrations by role (headless smoke)
coevolve version        print the version
```

`capabilities` needs no TTY — it's a quick check that the extensibility spine is
wired.

## The 8 modes

| # | Mode          | Shows                                          |
| - | ------------- | ---------------------------------------------- |
| 0 | **Home**      | command line · run mini-ork · live node stream |
| 1 | **Runs**      | runs & epics · scheduler · spawn tree          |
| 2 | **Learning**  | the learning loop · usage + research           |
| 3 | **Router**    | router & LLM performance                       |
| 4 | **ContextNest** | capsule · basins · knowledge graph           |
| 5 | **Topology**  | node DAG · lane→provider · health              |
| 6 | **Cost**      | metrics · savings · budget                     |
| 7 | **Logs**      | live `llm_calls` · run artifacts               |

Modes self-register at startup, so adding a mode is a single new file under
`internal/tui/modes/`.

## Keybindings

| Action                     | Keys        |
| -------------------------- | ----------- |
| Switch mode                | `0`–`7`     |
| Prev / next mode           | `←` / `→` · `Shift+Tab` / `Tab` |
| Command palette            | `⌘K` / `:`  |
| Keybindings overlay        | `?`         |
| Quit                       | `q` / `^C^C` (double) |
| Close overlay              | `esc`       |
| Learning flywheel toggle   | `r`         |
| Logs filter all/ok/fail    | `a` / `o` / `f` |
| ContextNest tabs           | `b c i s g` |

While the Home prompt is focused, keystrokes type normally; a bare digit only
switches tabs when the input is empty.

## Configuration

Coevolve is configured entirely through the environment (sane fallbacks apply):

| Variable                | Purpose                                                     |
| ----------------------- | ---------------------------------------------------------- |
| `MINI_ORK_ROOT`         | path to the mini-ork checkout (the orchestrator)           |
| `MINI_ORK_HOME`         | override for mini-ork's home dir (state.db / runs)         |
| `MINI_ORK_DB`           | explicit path to mini-ork's `state.db`                     |
| `MO_TARGET_CWD`         | the repo the work operates on (default: cwd)               |
| `COEVOLVE_WORKER_MODEL` | main LLM as opencode `provider/model` (also `MO_OPENCODE_MODEL`) |
| `COEVOLVE_LIVE`         | `1` to allow real spend; otherwise runs go dry             |
| `MO_DAILY_BUDGET_USD`   | hard budget cap for a run (set via the `/run` pre-flight)  |
| `CN_BASE_URL`           | ContextNest base URL                                       |

DB resolution order: `MINI_ORK_DB` → `MINI_ORK_HOME/state.db` →
`MINI_ORK_ROOT/.mini-ork/state.db`. When the DB is unreachable, modes render an
honest "offline" state rather than fabricated data.

## How it works

```
you ─▶ Home command line
         │
         ├─ plain text ──▶ opencode serve (HTTP+SSE) ──▶ streamed prose/tools
         │
         └─ /run <task> ─▶ mini-ork loop ──▶ nodes ──▶ state.db ──▶ live panels
                              (classify→plan→execute→verify→reflect→…)
```

For chat, Coevolve spawns an isolated `opencode serve` on a free port, creates a
session pinned to your worker model, and reads the SSE event stream — surfacing
text, tool calls, tokens, and cost as they happen. For `/run`, it shells out to
mini-ork and tails the run's own `state.db` so every node row you see is the real
one mini-ork just wrote.

## Project layout

```
cmd/coevolve/       entrypoint + subcommand dispatch
internal/core/      integration registry, events, provenance (the spine)
internal/tui/       Bubble Tea root model, palette, keys, palette overlay
internal/tui/modes/ the 8 self-registering dashboard modes
internal/run/       run controller · opencode serve worker · mini-ork spawn
internal/seams/     read-only data seams over mini-ork's DB / ContextNest / logs
internal/plan/      recipe catalog + suggestion for the /run pre-flight
internal/session/   per-session JSONL run log
kickoffs/           the build roadmap (10 epics)
reference/          the original Python prototype (contracts)
```

## Design principles

- **Real data only.** Panels are provenance-gated; if the source is missing, the
  panel says so — it never invents a number.
- **Extensible spine.** Integrations register by `Kind` (orchestrator, memory,
  learning, worker, router); modes register themselves. Adding either is one file.
- **Single-user, credential-inheriting.** No isolated XDG — it uses your real
  authenticated opencode config, so a run works exactly as `opencode` would.

## License

[Apache License 2.0](LICENSE). Architecture informed by omnigent (Apache-2.0).
