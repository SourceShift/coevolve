// Logs-mode queries: recent llm_calls rows as a log stream, plus a listing of
// files in the most recent run-artifact directory on disk. Both read REAL
// state — no synthetic rows, no fabricated file lists.
package seams

import (
	"os"
	"path/filepath"
	"sort"
)

// LogEntry is one llm_calls row rendered as a log-stream line.
type LogEntry struct {
	TS           string
	Provider     string
	ModelID      string
	Actor        string
	Status       string
	FinishReason string
	ErrorMessage string
	CostUSD      float64
	RunID        string
	Traceparent  string
}

// RecentLogCalls powers Logs mode's log stream — the most recent llm_calls
// rows, newest first.
func RecentLogCalls(limit int) ([]LogEntry, bool) {
	db, ok := DB()
	if !ok {
		return nil, false
	}
	defer db.Close()
	rows, err := db.Query(`SELECT ts, provider, model_id, COALESCE(actor,''), status,
		COALESCE(finish_reason,''), COALESCE(error_message,''), COALESCE(cost_usd,0),
		COALESCE(run_id,''), COALESCE(traceparent,'')
		FROM llm_calls ORDER BY ts DESC LIMIT ?`, limit)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []LogEntry
	for rows.Next() {
		var e LogEntry
		if rows.Scan(&e.TS, &e.Provider, &e.ModelID, &e.Actor, &e.Status,
			&e.FinishReason, &e.ErrorMessage, &e.CostUSD, &e.RunID, &e.Traceparent) == nil {
			out = append(out, e)
		}
	}
	return out, true
}

// ArtifactFile is one file inside a run-artifact directory.
type ArtifactFile struct {
	Name string
	Size int64
}

// RunsDir resolves mini-ork's run-artifact root directory: derived from
// $MINI_ORK_DB's directory when set (mirrors DBPath's primary knob), else
// $MINI_ORK_HOME/runs, else $MINI_ORK_ROOT/.mini-ork/runs, else the default
// mini-ork checkout's .mini-ork/runs.
func RunsDir() string {
	if p := os.Getenv("MINI_ORK_DB"); p != "" {
		return filepath.Join(filepath.Dir(p), "runs")
	}
	if h := os.Getenv("MINI_ORK_HOME"); h != "" {
		return filepath.Join(h, "runs")
	}
	if r := os.Getenv("MINI_ORK_ROOT"); r != "" {
		return filepath.Join(r, ".mini-ork", "runs")
	}
	return "/Volumes/docker-ssd/ps/mini-ork/.mini-ork/runs"
}

// LatestRunArtifacts lists the files (name + size) inside the most recently
// modified run directory under RunsDir(). Returns the directory's base name,
// its files, and false when the runs root or its newest entry can't be read.
func LatestRunArtifacts() (string, []ArtifactFile, bool) {
	root := RunsDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", nil, false
	}
	var newestName string
	var newestMod int64 = -1
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if mt := info.ModTime().UnixNano(); mt > newestMod {
			newestMod = mt
			newestName = e.Name()
		}
	}
	if newestName == "" {
		return "", nil, false
	}
	dirEntries, err := os.ReadDir(filepath.Join(root, newestName))
	if err != nil {
		return newestName, nil, false
	}
	var out []ArtifactFile
	for _, f := range dirEntries {
		if f.IsDir() {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		out = append(out, ArtifactFile{Name: f.Name(), Size: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return newestName, out, true
}
