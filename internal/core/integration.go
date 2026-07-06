package core

import (
	"context"
	"fmt"
	"sort"
)

// Kind is the role an integration plays in the loop.
type Kind string

const (
	KindOrchestrator Kind = "orchestrator" // mini-ork
	KindMemory       Kind = "memory"       // ContextNest
	KindLearning     Kind = "learning"     // TraceOtter
	KindWorker       Kind = "worker"       // opencode
	KindRouter       Kind = "router"       // lane router
)

// Mode is how core talks to the integration.
type Mode string

const (
	ModeCLISubprocess Mode = "cli_subprocess"
	ModeNativeServer  Mode = "native_server"
	ModeHTTPAPI       Mode = "http_api"
)

// Capabilities is the declared feature matrix every integration ships — the
// backbone of extensibility (a new backend adds a row + self-registers).
type Capabilities struct {
	Kind          Kind
	Mode          Mode
	Interrupt     bool     // can a run be stopped mid-flight?
	CostReporting bool     // emits real per-call cost
	Permissions   bool     // gates tool actions through policy
	Streaming     bool     // emits events as work proceeds
	Resume        bool     // supports stop→resume
	Models        []string // concrete model ids it can drive (worker)
	Notes         string
}

// Health is an integration's live availability.
type Health struct {
	Available bool
	Degraded  bool
	Detail    string // e.g. "ContextNest offline", "cold-start"
}

// Recall is a memory hit fed into a run.
type Recall struct {
	Text   string
	Source string // basin / session / capsule id
	AtomID string
	Prov   Provenance
}

// LaneChoice is a router decision.
type LaneChoice struct {
	Lane      string
	Model     string
	Advantage *float64 // nil = cold start
	RunsCount int
	Prov      Provenance
}

// RunRequest is everything a run needs, assembled by core before dispatch.
type RunRequest struct {
	Task            string
	TaskClass       string
	ObjectiveDomain string
	RunDir          string
	TargetCWD       string
	Recalls         []Recall
	Lane            *LaneChoice
	DryRun          bool
}

// Integration is the base every backend implements.
type Integration interface {
	Name() string
	Capabilities() Capabilities
	Health() Health
}

// Orchestrator drives a run through the loop, emitting normalized events. (mini-ork)
type Orchestrator interface {
	Integration
	// Run starts a run and streams events on the returned channel until closed.
	Run(ctx context.Context, req RunRequest) (<-chan Event, error)
	Stop(runID string) error
	Resume(runID string) error
	Rollback(runID string) error
	ListRuns(limit int) ([]RunSummary, error)
}

// RunSummary is a row in the Runs mode.
type RunSummary struct {
	RunID   string
	Recipe  string
	Agent   string
	Verdict string
	Iters   int
	Cost    Value
}

// Memory is recall + outcome write-back. (ContextNest)
type Memory interface {
	Integration
	Recall(query string, limit int) ([]Recall, error)
	WriteOutcome(outcome string, atomIDs []string, evidence string) error
}

// Learning distils traces into a training set / skills. (TraceOtter)
type Learning interface {
	Integration
	Distill(runsDir string) (map[string]any, error)
}

// Worker executes a node's work against a model. (opencode)
type Worker interface {
	Integration
	SupportsModel(model string) bool
}

// Router chooses lane+model per node from learned advantage. (lane_router)
type Router interface {
	Integration
	PickLane(taskClass, nodeType, objectiveDomain, codeRegion string) (LaneChoice, error)
	Recompute() error
}

// ── registry: decorator-free self-registration via init() + Register ──────────

var registry = map[Kind]map[string]Integration{}

// Register adds an integration under its declared Kind. Call from init().
func Register(i Integration) {
	k := i.Capabilities().Kind
	if registry[k] == nil {
		registry[k] = map[string]Integration{}
	}
	registry[k][i.Name()] = i
}

// Get resolves a backend by role + name.
func Get(k Kind, name string) (Integration, error) {
	if m, ok := registry[k]; ok {
		if i, ok := m[name]; ok {
			return i, nil
		}
	}
	return nil, fmt.Errorf("no %s integration named %q", k, name)
}

// AllOf returns every registered integration of a role (name-sorted).
func AllOf(k Kind) []Integration {
	var out []Integration
	for _, i := range registry[k] {
		out = append(out, i)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Name() < out[b].Name() })
	return out
}

// FirstOf returns the first registered integration of a role, or nil.
func FirstOf(k Kind) Integration {
	if all := AllOf(k); len(all) > 0 {
		return all[0]
	}
	return nil
}

// Registered snapshots the registry for `coevolve capabilities`.
func Registered() map[Kind][]string {
	out := map[Kind][]string{}
	for k, m := range registry {
		for name := range m {
			out[k] = append(out[k], name)
		}
		sort.Strings(out[k])
	}
	return out
}
