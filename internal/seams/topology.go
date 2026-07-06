// Topology-mode queries: workflow node → lane → provider resolution, provider
// health, and the watchdog/grounded-rejection safety strip. Same rule as the
// rest of this package — real data only, "not available" reported honestly
// when a source is missing or empty.
package seams

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// WorkflowNodes is mini-ork's universal loop, in pipeline order. It is a
// fixed constant (not queried) — the DAG shape itself doesn't change per
// run, only which lane/provider serves each node.
var WorkflowNodes = []string{"classify", "plan", "execute", "verify", "review", "reflect"}

// nodeLane maps each universal-loop node to the agents.yaml lane that serves
// it. mini-ork's config doesn't literally spell "classify"/"execute"/"review"
// as lane keys, so this is the documented, best-effort mapping onto the
// nearest canonical loop-role lane (config/agents.yaml `lanes:`).
var nodeLane = map[string]string{
	"classify": "decomposer",
	"plan":     "planner",
	"execute":  "worker",
	"verify":   "verifier",
	"review":   "reviewer",
	"reflect":  "reflector",
}

// NodeLane returns the lane name mini-ork's config assigns to a workflow node.
func NodeLane(node string) string { return nodeLane[node] }

// AgentsYAMLPath resolves config/agents.yaml: $MINI_ORK_ROOT/config/agents.yaml,
// else the repo-relative default used by this dashboard's own dev setup.
func AgentsYAMLPath() string {
	root := os.Getenv("MINI_ORK_ROOT")
	if root == "" {
		root = "/Volumes/docker-ssd/ps/mo-fix"
	}
	return filepath.Join(root, "config", "agents.yaml")
}

// LaneProviderMap parses the `lanes:` block of config/agents.yaml into a
// lane -> model map (e.g. "reviewer" -> "opus"). It's a tiny indentation-aware
// line scan, not a full YAML parser: sufficient for this flat "key: value"
// block and avoids adding a YAML dependency for one map.
func LaneProviderMap() (map[string]string, bool) {
	f, err := os.Open(AgentsYAMLPath())
	if err != nil {
		return nil, false
	}
	defer f.Close()

	out := map[string]string{}
	inLanes := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)

		if !inLanes {
			if trimmed == "lanes:" && indent == 0 {
				inLanes = true
			}
			continue
		}
		if trimmed == "" {
			continue
		}
		if indent == 0 {
			break // next top-level key — lanes: block is over
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// strip trailing "# comment"
		if i := strings.Index(trimmed, "#"); i >= 0 {
			trimmed = strings.TrimSpace(trimmed[:i])
		}
		key, val, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	if err := sc.Err(); err != nil || len(out) == 0 {
		return nil, false
	}
	return out, true
}

// LaneProviderFallback derives a lane -> provider map from llm_calls when
// agents.yaml can't be read: each distinct actor's most-recently-used
// provider. actor values (planner, worker, reviewer, reflector, …) line up
// with agents.yaml lane names for the canonical loop roles.
func LaneProviderFallback() (map[string]string, bool) {
	db, ok := DB()
	if !ok {
		return nil, false
	}
	defer db.Close()
	rows, err := db.Query(`
		SELECT actor, provider FROM (
			SELECT actor, provider,
			       ROW_NUMBER() OVER (PARTITION BY actor ORDER BY ts DESC) AS rn
			FROM llm_calls WHERE actor IS NOT NULL AND actor <> ''
		) WHERE rn = 1`)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var actor, provider string
		if rows.Scan(&actor, &provider) == nil {
			out[actor] = provider
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// ProviderHealth is per-provider status derived from llm_calls: most recent
// call's status ("success"/"failed") and an average call duration (labeled
// p50 like the rest of this package's latency fields — it's a mean, not a
// true percentile; SQLite has no built-in median).
type ProviderHealth struct {
	Provider     string
	Calls        int
	P50MS        int
	LatestStatus string // "success" | "failed"
	Healthy      bool   // LatestStatus == "success"
}

// ProviderHealthByProvider powers Topology mode's provider-health panel.
func ProviderHealthByProvider() ([]ProviderHealth, bool) {
	db, ok := DB()
	if !ok {
		return nil, false
	}
	defer db.Close()
	rows, err := db.Query(`
		SELECT provider, COUNT(*), CAST(COALESCE(AVG(duration_ms),0) AS INT),
		       (SELECT status FROM llm_calls l2
		        WHERE l2.provider = l1.provider ORDER BY l2.ts DESC LIMIT 1)
		FROM llm_calls l1 GROUP BY provider ORDER BY 2 DESC`)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []ProviderHealth
	for rows.Next() {
		var h ProviderHealth
		if rows.Scan(&h.Provider, &h.Calls, &h.P50MS, &h.LatestStatus) == nil {
			h.Healthy = h.LatestStatus == "success"
			out = append(out, h)
		}
	}
	return out, true
}

// SafetyStrip is the watchdog-abort / grounded-rejection summary. Both
// tables are real (append-only, schema present in state.db); counts of 0
// mean "queried, nothing recorded yet" — not "not available".
type SafetyStrip struct {
	WatchdogAborts       int
	WatchdogAvoided      int // verified_failure_avoided = 'true'
	GroundedRejections   int
	UnconsumedRejections int
}

// SafetyStripStatus powers Topology mode's safety strip.
func SafetyStripStatus() (SafetyStrip, bool) {
	db, ok := DB()
	if !ok {
		return SafetyStrip{}, false
	}
	defer db.Close()

	var s SafetyStrip
	if db.QueryRow(`SELECT COUNT(*) FROM watchdog_aborts`).Scan(&s.WatchdogAborts) != nil {
		return SafetyStrip{}, false
	}
	db.QueryRow(`SELECT COUNT(*) FROM watchdog_aborts WHERE verified_failure_avoided = 'true'`).Scan(&s.WatchdogAvoided)
	if db.QueryRow(`SELECT COUNT(*) FROM grounded_rejections`).Scan(&s.GroundedRejections) != nil {
		return SafetyStrip{}, false
	}
	db.QueryRow(`SELECT COUNT(*) FROM grounded_rejections WHERE consumed_by_reflector_ts IS NULL`).Scan(&s.UnconsumedRejections)
	return s, true
}
