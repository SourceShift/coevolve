# EPIC-07 — Run control (stop/resume/rollback)
Status: todo · Depends: 02
## Goal
Robust stop/resume/rollback through the MCP + mini-ork, no orphaned processes.
## Deliverables
- control.py — RunController: bridge-state file + last_event_id cursor
  (omnigent pattern); supervisor task whose finally reaps the worker; maps to
  mini-ork-resume / mini-ork-rollback + process signalling.
- mo_stop/mo_resume/mo_rollback wired to it.
## Acceptance
- A running task can be stopped mid-flight (worker reaped, no orphan) and resumed.
