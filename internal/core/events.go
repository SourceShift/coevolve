package core

import "time"

// EventType is the normalized vocabulary of the live run-flow, consumed
// identically by every mode (and, later, any external surface).
type EventType string

const (
	EvRunStarted  EventType = "run_started"
	EvNode        EventType = "node"       // a mini-ork node (classify/plan/execute/…)
	EvStatus      EventType = "status"     // status change (running/escalated/done/failed)
	EvUsage       EventType = "usage"      // cumulative cost/token tick
	EvRecall      EventType = "recall"     // a ContextNest recall fed into the run
	EvReroute     EventType = "reroute"    // router auto-rerouted (quota/parity)
	EvPermission  EventType = "permission" // worker asked to perform a gated action
	EvLog         EventType = "log"        // a raw log line
	EvRunFinished EventType = "run_finished"
)

// Route classifies a node's dispatch target.
type Route string

const (
	RouteLocal    Route = "local"
	RouteFrontier Route = "frontier"
	RouteUnknown  Route = "unknown"
)

// Event is the single envelope. Numbers the UI shows are Values so provenance
// rides along and the honesty rule cannot be bypassed in a panel.
type Event struct {
	Type  EventType
	RunID string
	TS    time.Time
	Seq   int    // monotonic per run (cursor / dedupe key)
	Node  string // classify | plan | execute | verify | reflect | …
	Route Route
	Lane  string // resolved lane name
	Model string // resolved provider/model
	Text  string
	Cost  Value            // per-node/cumulative cost (Real from llm_calls, or NotYet)
	Data  map[string]Value // extra typed fields
}
