# EPIC-10 — End-to-end proof + docs
Status: ⬜  ·  Depends: 06 (min) / 09 (full)

## Goal
The single command that proves the mixture works, plus docs.

## Deliverables
- Proof run: `coevolve run "add a --version flag to scripts/foo.sh" --lane cheap`
  showing all three systems participated (per plan verification).
- `coevolve/README.md` — architecture, extend-guide (add an integration / a mode),
  omnigent attribution.
- A short demo recipe for the deck / DocSend.

## Acceptance (the honest proof)
- mini-ork: task_class + plan.json + runs/<id>/ + execution_traces.reward_g + llm_calls.
- ContextNest: non-empty capsule + cn_outcome_post recorded (or explicit offline).
- opencode+router: route→opencode→model; shim sidecars; llm_calls.provider='opencode'.
- TraceOtter: distilled count includes this run.
- NOT_YET panels labeled, never faked.
