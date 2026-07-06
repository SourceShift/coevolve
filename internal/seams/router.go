// Router-mode queries. This file only ADDS to the seams surface — it never
// redefines DB()/DBPath() or the shared row types declared in db.go.
package seams

// TotalCalls is the all-time real count of llm_calls rows — powers Router
// mode's header "window" figure.
func TotalCalls() (int, bool) {
	db, ok := DB()
	if !ok {
		return 0, false
	}
	defer db.Close()
	var n int
	if db.QueryRow(`SELECT COUNT(*) FROM llm_calls`).Scan(&n) != nil {
		return 0, false
	}
	return n, true
}

// LaneLeaderboardWithReward joins LaneLeaderboard (share/latency/cost per
// model_id, from llm_calls) with RewardByLane (avg reward_g per
// agent_version_id, from execution_traces) by exact lane-name match. Lanes
// with no graded execution_traces rows keep RewardG == nil — the caller
// renders that as "—", never a fabricated value. Both source queries are
// real; this is a real-data join, not a synthesized metric.
func LaneLeaderboardWithReward() ([]LaneStat, bool) {
	stats, ok := LaneLeaderboard()
	if !ok {
		return nil, false
	}
	rewards, rOK := RewardByLane()
	if !rOK {
		return stats, true // leaderboard itself is still real; reward join just unavailable
	}
	byLane := make(map[string]float64, len(rewards))
	for _, r := range rewards {
		byLane[r.Lane] = r.RewardG
	}
	out := make([]LaneStat, len(stats))
	copy(out, stats)
	for i := range out {
		if v, found := byLane[out[i].Lane]; found {
			vv := v
			out[i].RewardG = &vv
		}
	}
	return out, true
}
