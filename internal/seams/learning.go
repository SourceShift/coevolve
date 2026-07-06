// Learning-mode queries: GRPO reward-by-lane (via RewardByLane in db.go) plus
// the objective_domain distinct list. Real data only — missing/empty is
// reported honestly via the (…, bool) convention used across seams.
package seams

// ObjectiveDomains returns the distinct objective_domain values seen in
// execution_traces (Learning mode's header line). Real if the DB has rows.
func ObjectiveDomains() ([]string, bool) {
	db, ok := DB()
	if !ok {
		return nil, false
	}
	defer db.Close()
	rows, err := db.Query(`SELECT DISTINCT objective_domain FROM execution_traces
		WHERE objective_domain <> '' ORDER BY 1`)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if rows.Scan(&d) == nil {
			out = append(out, d)
		}
	}
	return out, true
}
