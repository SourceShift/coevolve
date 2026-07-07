# EPIC-08 — opencode serve + SSE agentic worker (Go)

Build the Go client that makes the Coevolve run console replicate the Claude/opencode/Kimi
agentic flow (streaming text + visible tool calls + per-turn cost), reliably — because
`opencode run` hangs on headless tool work; the robust path is `opencode serve` + SSE.

## Deliverable: `internal/run/opencode_serve.go` (new) + wire `internal/run/controller.go`

Implement a struct `ServeWorker` that:
1. Spawns an isolated server: `opencode serve --hostname 127.0.0.1 --port <random> --pure`
   as an exec.Cmd; poll `GET http://127.0.0.1:<port>/session` until it returns <500 (ready).
   Use a per-session temp XDG dir (env XDG_DATA_HOME/XDG_CONFIG_HOME) so it never touches
   the user's global opencode config.
2. Creates a session: `POST /session` (body `{}`) → parse `{id}`.
3. Sets the model: `POST /session/{id}/model` with the configured provider/model (split
   the "provider/model" string on the first "/").
4. Opens the SSE stream: `GET /event` (text/event-stream). Parse SSE frames (`data: {json}`);
   each event is `{type, properties}`. Handle: `message.part.updated` (properties.part with
   part.type ∈ text(part.text) / tool(part.tool + part.state{status,input,output,title}) /
   step-finish(part.tokens{input,output,cache}, part.cost)); `session.idle` (turn done);
   `session.error`; `permission.asked` → auto-reply via `POST /session/{id}/permission/{reqID}/reply`
   body `{response:"once"}` (or "always"); default fail-closed if unclear.
5. Sends the prompt: `POST /session/{id}/prompt_async` body `{parts:[{type:"text", text:<prompt>}]}`.
6. Emits normalized events on a channel: reuse/extend the existing `run.Line` OR add a typed
   `run.Event` {Kind: "text"|"tool"|"cost"|"done"|"error", Text, Tool, Input, Output, InTok, OutTok, CostUSD}.
7. Abort: `POST /session/{id}/abort`; on Stop(), abort + kill the serve process (no orphan).

Then update `controller.go`: for a plain-text task (not `/run`), use `ServeWorker` instead of
`opencode run`, so tool-agentic work streams. Keep `/run` → mini-ork.

## Constraints
- Go stdlib only (net/http, bufio, encoding/json) — no new deps.
- All existing tests + `go build ./... && go vet ./internal/...` must stay green.
- Real: verify against a live `opencode serve` that a task which reads a file streams a
  `tool` event then `text` + `cost` (use model `openai/gpt-5.2`).

## Acceptance
- `internal/run/opencode_serve.go` exists; controller uses it for plain tasks; a real task
  that invokes a tool streams tool + text + cost events; Stop() reaps the serve process.
