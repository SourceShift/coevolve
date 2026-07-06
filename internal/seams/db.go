// Package seams is the ONLY layer that talks to mini-ork's surfaces: its SQLite
// state DB (read-only), its bin/ CLIs (subprocess), and ContextNest's HTTP API.
// Everything above (integrations, TUI modes) reads typed data from here — never
// raw SQL or subprocess. Real data only; missing/empty is reported honestly.
package seams

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // pure-Go driver, no cgo
)

// DBPath resolves mini-ork's state.db: $MINI_ORK_DB, else
// $MINI_ORK_HOME/state.db, else <root>/.mini-ork/state.db.
func DBPath() string {
	if p := os.Getenv("MINI_ORK_DB"); p != "" {
		return p
	}
	if h := os.Getenv("MINI_ORK_HOME"); h != "" {
		return filepath.Join(h, "state.db")
	}
	if r := os.Getenv("MINI_ORK_ROOT"); r != "" {
		return filepath.Join(r, ".mini-ork", "state.db")
	}
	return filepath.Join(".mini-ork", "state.db")
}

// DB opens the state DB read-only. Returns (nil,false) when absent — callers
// render a DEGRADED marker rather than fabricate.
func DB() (*sql.DB, bool) {
	p := DBPath()
	if _, err := os.Stat(p); err != nil {
		return nil, false
	}
	db, err := sql.Open("sqlite", "file:"+p+"?mode=ro")
	if err != nil {
		return nil, false
	}
	if err := db.Ping(); err != nil {
		return nil, false
	}
	return db, true
}

// ── typed rows ────────────────────────────────────────────────────────────────

type ProviderSpend struct {
	Provider string
	Calls    int
	Tokens   int64
	CostUSD  float64
}

type LaneReward struct {
	Lane    string
	Samples int
	RewardG float64
}

type LaneStat struct {
	Lane      string
	Calls     int
	SharePct  float64
	P50MS     int
	CostUSD   float64
	RewardG   *float64 // nil when no graded traces yet
}

type RunRow struct {
	RunID   string
	Recipe  string
	Status  string
	CostUSD float64
	Iters   int
}

// ── queries (all read-only, all real) ────────────────────────────────────────

// SpendByProvider powers Cost mode's spend table.
func SpendByProvider() ([]ProviderSpend, bool) {
	db, ok := DB()
	if !ok {
		return nil, false
	}
	defer db.Close()
	rows, err := db.Query(`SELECT provider, COUNT(*), COALESCE(SUM(total_tokens),0),
		COALESCE(SUM(cost_usd),0) FROM llm_calls GROUP BY provider ORDER BY 4 DESC`)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []ProviderSpend
	for rows.Next() {
		var s ProviderSpend
		if rows.Scan(&s.Provider, &s.Calls, &s.Tokens, &s.CostUSD) == nil {
			out = append(out, s)
		}
	}
	return out, true
}

// TotalCost is the all-time real spend (Cost mode lifetime KPI).
func TotalCost() (float64, bool) {
	db, ok := DB()
	if !ok {
		return 0, false
	}
	defer db.Close()
	var v sql.NullFloat64
	if db.QueryRow(`SELECT SUM(cost_usd) FROM llm_calls`).Scan(&v) != nil {
		return 0, false
	}
	return v.Float64, true
}

// TodayCost is real spend since local midnight (approx via ts date()).
func TodayCost() (float64, bool) {
	db, ok := DB()
	if !ok {
		return 0, false
	}
	defer db.Close()
	var v sql.NullFloat64
	err := db.QueryRow(`SELECT COALESCE(SUM(cost_usd),0) FROM llm_calls
		WHERE date(ts) = date('now','localtime')`).Scan(&v)
	if err != nil {
		return 0, false
	}
	return v.Float64, true
}

// RewardByLane powers Learning mode's GRPO reward-by-lane bars.
func RewardByLane() ([]LaneReward, bool) {
	db, ok := DB()
	if !ok {
		return nil, false
	}
	defer db.Close()
	rows, err := db.Query(`SELECT agent_version_id, COUNT(*), AVG(reward_g)
		FROM execution_traces WHERE reward_g IS NOT NULL AND agent_version_id <> ''
		GROUP BY agent_version_id ORDER BY 3 DESC`)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []LaneReward
	for rows.Next() {
		var r LaneReward
		if rows.Scan(&r.Lane, &r.Samples, &r.RewardG) == nil {
			out = append(out, r)
		}
	}
	return out, true
}

// LaneLeaderboard powers Router mode: share/$/latency/reward_g per lane (real).
func LaneLeaderboard() ([]LaneStat, bool) {
	db, ok := DB()
	if !ok {
		return nil, false
	}
	defer db.Close()
	var total int
	db.QueryRow(`SELECT COUNT(*) FROM llm_calls`).Scan(&total)
	rows, err := db.Query(`SELECT model_id, COUNT(*),
		CAST(COALESCE(AVG(duration_ms),0) AS INT), COALESCE(SUM(cost_usd),0)
		FROM llm_calls GROUP BY model_id ORDER BY 2 DESC`)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []LaneStat
	for rows.Next() {
		var s LaneStat
		if rows.Scan(&s.Lane, &s.Calls, &s.P50MS, &s.CostUSD) == nil {
			if total > 0 {
				s.SharePct = 100 * float64(s.Calls) / float64(total)
			}
			out = append(out, s)
		}
	}
	return out, true
}

// RecentRuns powers Runs mode.
func RecentRuns(limit int) ([]RunRow, bool) {
	db, ok := DB()
	if !ok {
		return nil, false
	}
	defer db.Close()
	rows, err := db.Query(`SELECT run_id, COALESCE(MAX(feature_name),''),
		COALESCE(MAX(status),''), COALESCE(SUM(cost_usd),0), COUNT(DISTINCT iter)
		FROM llm_calls WHERE run_id IS NOT NULL AND run_id <> ''
		GROUP BY run_id ORDER BY MAX(ts) DESC LIMIT ?`, limit)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []RunRow
	for rows.Next() {
		var r RunRow
		if rows.Scan(&r.RunID, &r.Recipe, &r.Status, &r.CostUSD, &r.Iters) == nil {
			out = append(out, r)
		}
	}
	return out, true
}
