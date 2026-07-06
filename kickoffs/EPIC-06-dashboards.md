# EPIC-06 — Dashboards (real-data-only)
Status: todo · Depends: 03,05
## Goal
The design's dashboard capabilities, real data only, as MCP tools + subcommands.
## Deliverables (coevolve/dash/ + mcp tools)
- cost (llm_calls sums, budget caps; savings labeled est.), router (lane share/$/
  lat/reward_g; parity/win% = NOT_YET), learning (reward_g by lane; retrain-history
  = NOT_YET), topology (node DAG + lane→provider + health), contextnest (basins/
  capsule/inbox/sessions). Each as `coevolve <cmd>` and an MCP tool. Optional: wire
  the existing web Console to the same data.
## Acceptance
- Each surface shows measured data or an explicit NOT_YET/offline marker.
