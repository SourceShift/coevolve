# Build the opencode serve+SSE agentic worker (Go)

## Scope (ONLY these files are in scope)
- `internal/run/opencode_serve.go` (new)
- `internal/run/controller.go` (edit: route plain-text tasks through the new ServeWorker)

## Success command (this proves the run succeeded)
`cd /Volumes/docker-ssd/ps/coevolve && go build ./... && go vet ./internal/run/ ./internal/tui/...`

## Task
Make the Coevolve run console replicate the claude/opencode agentic flow (streaming text
+ visible tool calls + per-turn cost) reliably, via `opencode serve` + SSE (because
`opencode run` hangs on headless tool work).

Add `internal/run/opencode_serve.go` with a `ServeWorker` that (Go stdlib only — net/http,
bufio, encoding/json):
1. Spawns `opencode serve --hostname 127.0.0.1 --port <random> --pure` (exec.Cmd), with a
   per-session temp XDG_DATA_HOME/XDG_CONFIG_HOME so it never touches the user's global config.
   Poll `GET http://127.0.0.1:<port>/session` until status < 500 (ready).
2. `POST /session` (body `{}`) → parse `{"id":...}`.
3. `POST /session/{id}/model` with the provider/model (split cfg.WorkerModel on first "/").
4. Opens `GET /event` (SSE, text/event-stream); parse `data: {json}` frames. Each event is
   `{"type":..., "properties":...}`. Handle: `message.part.updated` where properties.part.type ∈
   text (part.text) / tool (part.tool + part.state{status,input,output}) /
   step-finish (part.tokens{input,output,cache}, part.cost); `session.idle` (done);
   `session.error`; `permission.asked` → `POST /session/{id}/permission/{reqID}/reply` body
   `{"response":"once"}` (fail-closed default reject if unclear).
5. `POST /session/{id}/prompt_async` body `{"parts":[{"type":"text","text":<prompt>}]}`.
6. Emit lines on the existing `run.Line` channel (Text/Err) — render tool events as
   `● <tool>(<input summary>)` then `  ⎿ <output first line>`, text as-is, and a final
   `✓ <inTok>→<outTok> tok · $<cost>` from step-finish. Reuse `stripANSI`.
7. Abort: `POST /session/{id}/abort`; Stop() aborts + kills the serve process (no orphan).

Then in `internal/run/controller.go`: in `build()`, for a plain-text task (NOT starting with
`/run `), use `ServeWorker` instead of `exec`-ing `opencode run`. Keep `/run` → mini-ork.

Keep all existing tests green; `go build ./...` and `go vet` must pass.
