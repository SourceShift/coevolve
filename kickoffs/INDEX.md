# Coevolve CLI — Epic Roadmap

**Architecture (corrected 2026-07-06):** the CLI **is opencode** (the runtime the
user works in). opencode can **spawn mini-ork** — for orchestrated multi-file
work — via a **mini-ork MCP server**. mini-ork brings **ContextNest** (memory) +
**TraceOtter** (learning). Direction: opencode -> mini-ork -> {CN, TraceOtter}.

**Built BY mini-ork** (dogfood) against THIS repo as a normal target
(/Volumes/docker-ssd/ps/coevolve), with mini-ork reached via MINI_ORK_ROOT
(a mo-fix checkout). Never build inside the mini-ork tree (framework-self-edit
corruption). All panels render **real data only** (provenance-typed).

The extensibility foundation (coevolve/{capabilities,events,registry}.py,
coevolve/integrations/base.py -- verified) is the MCP server's backend:
integrations become MCP tools.

| # | Epic | Status |
|---|------|--------|
| 01 | Extensibility foundation (-> MCP backend) | done |
| 02 | **mini-ork MCP server** -- mo_run/mo_stop/mo_resume/mo_status/mo_list tools (opencode spawns mini-ork) | todo |
| 03 | ContextNest + TraceOtter MCP tools -- cn_recall/cn_write, traceotter_distill/_stats | todo |
| 04 | Coevolve CLI bundle -- opencode configured to load the mini-ork MCP; branding; coevolve launcher; cl_opencode.sh for mini-ork's own node dispatch | todo |
| 05 | Live-flow surfacing -- stream mini-ork run events into opencode's UI + coevolve watch/logs | todo |
| 06 | Dashboards (real-data-only) -- cost/router/learning/topology as MCP tools + coevolve <cmd> + web Console wiring | todo |
| 07 | Run control -- stop/resume/rollback through MCP + mini-ork resume/rollback; bridge-state + supervisor-reaps | todo |
| 08 | opencode serve/SSE live worker (vendor omnigent client) -- live within-run streaming + abort | todo |
| 09 | End-to-end proof -- inside opencode, spawn mini-ork on a real task; see CN recall + TraceOtter distill + cost; stop/resume | todo |
| 10 | Docs + packaging -- extend-guide, omnigent attribution, demo recipe | todo |

Contracts (do not fork): coevolve/capabilities.py, coevolve/events.py,
coevolve/integrations/base.py, coevolve/registry.py.
Design source: researcher/tmp/Coevolve Console overview/Coevolve CLI.dc.html
omnigent (Apache-2.0) = pattern reference for opencode-MCP + extensibility.
