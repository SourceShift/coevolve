// Additive queries for the Runs mode. Same rules as db.go: read-only, real
// data only, (zero-value,false) when the DB or expected columns are absent —
// callers render "not available yet" rather than fabricate.
package seams

// EpicRow is one row from mini-ork's `epics` table.
type EpicRow struct {
	ID      string
	Title   string
	Status  string
	Lane    string
	GroupID string
}

// EpicCounts buckets epics.status into scheduler-shaped counters. mini-ork's
// real status vocabulary is "not started" | "in progress" | "blocked" | "done"
// (no literal "in review" state) so InReview is derived from pr_url being set
// on a not-yet-done epic — a real signal, honestly often zero.
type EpicCounts struct {
	Ready    int // status = 'not started'
	Running  int // status = 'in progress'
	Blocked  int // status = 'blocked'
	InReview int // pr_url set AND status <> 'done'
	Done     int // status = 'done'
	Total    int
}

// EpicDepRow is one row from `epic_dependencies`, optionally joined to the
// downstream epic's current status.
type EpicDepRow struct {
	FromEpicID string
	ToEpicID   string
	Kind       string
	Resolved   bool
	ToStatus   string // status of ToEpicID, "" if unknown
}

// RecentEpics powers the Runs mode EPICS panel: most-recently-updated first.
func RecentEpics(limit int) ([]EpicRow, bool) {
	db, ok := DB()
	if !ok {
		return nil, false
	}
	defer db.Close()
	rows, err := db.Query(`SELECT id, title, status, COALESCE(lane,''), COALESCE(group_id,'')
		FROM epics ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []EpicRow
	for rows.Next() {
		var e EpicRow
		if rows.Scan(&e.ID, &e.Title, &e.Status, &e.Lane, &e.GroupID) == nil {
			out = append(out, e)
		}
	}
	return out, true
}

// EpicStatusSummary powers the scheduler counters row.
func EpicStatusSummary() (EpicCounts, bool) {
	db, ok := DB()
	if !ok {
		return EpicCounts{}, false
	}
	defer db.Close()
	var c EpicCounts
	err := db.QueryRow(`SELECT
		COALESCE(SUM(CASE WHEN status = 'not started' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status = 'in progress' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status = 'blocked' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status <> 'done' AND pr_url IS NOT NULL AND pr_url <> '' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END),0),
		COUNT(*)
		FROM epics`).Scan(&c.Ready, &c.Running, &c.Blocked, &c.InReview, &c.Done, &c.Total)
	if err != nil {
		return EpicCounts{}, false
	}
	return c, true
}

// DependencyCascade powers the DEPENDENCY CASCADE panel: unresolved edges
// first (they are the live cascade), most-recently-created first.
func DependencyCascade(limit int) ([]EpicDepRow, bool) {
	db, ok := DB()
	if !ok {
		return nil, false
	}
	defer db.Close()
	rows, err := db.Query(`SELECT d.from_epic_id, d.to_epic_id, d.kind,
		CASE WHEN d.resolved_at IS NULL THEN 0 ELSE 1 END, COALESCE(e.status,'')
		FROM epic_dependencies d LEFT JOIN epics e ON e.id = d.to_epic_id
		ORDER BY (d.resolved_at IS NULL) DESC, d.created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []EpicDepRow
	for rows.Next() {
		var d EpicDepRow
		var resolved int
		if rows.Scan(&d.FromEpicID, &d.ToEpicID, &d.Kind, &resolved, &d.ToStatus) == nil {
			d.Resolved = resolved != 0
			out = append(out, d)
		}
	}
	return out, true
}
