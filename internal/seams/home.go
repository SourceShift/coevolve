// Package seams (home.go) adds Home mode's one extra query: the step-by-step
// llm_calls trace for a single run_id. Everything else Home needs (the head
// row, the router leaderboard) already exists in db.go and is reused as-is —
// see the seams.RecentRuns / seams.LaneLeaderboard doc comments there.
package seams

// HomeStep is one llm_calls row belonging to a run, in chronological order —
// this is what Home mode renders as the "last run" node-by-node stream.
type HomeStep struct {
	Actor      string
	Provider   string
	ModelID    string
	CostUSD    float64
	DurationMS int
	Status     string
	TS         string
}

// RunSteps returns every llm_calls row for runID ordered oldest-first, i.e.
// the run's real node sequence. Real data only; (nil,false) when the DB is
// unreachable or runID is empty.
func RunSteps(runID string) ([]HomeStep, bool) {
	if runID == "" {
		return nil, false
	}
	db, ok := DB()
	if !ok {
		return nil, false
	}
	defer db.Close()
	rows, err := db.Query(`SELECT COALESCE(actor,''), provider, model_id,
		cost_usd, duration_ms, status, ts
		FROM llm_calls WHERE run_id = ? ORDER BY ts ASC`, runID)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []HomeStep
	for rows.Next() {
		var s HomeStep
		if rows.Scan(&s.Actor, &s.Provider, &s.ModelID, &s.CostUSD, &s.DurationMS, &s.Status, &s.TS) == nil {
			out = append(out, s)
		}
	}
	return out, true
}
