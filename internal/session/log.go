// Package session writes a per-session JSONL transcript (like claude/codex/opencode)
// to $XDG_CONFIG/coevolve/sessions/<id>.jsonl — one JSON event per line. Useful
// for inspecting what a run did, replay, and TraceOtter distillation.
package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event is one line in the transcript.
type Event struct {
	TS       string         `json:"ts"`
	Type     string         `json:"type"` // session_start | user | assistant | tool | cost | status | run_start | run_end
	Text     string         `json:"text,omitempty"`
	Err      bool           `json:"err,omitempty"`
	Markdown bool           `json:"markdown,omitempty"`
	Meta     map[string]any `json:"meta,omitempty"`
}

// Log is an append-only JSONL session transcript.
type Log struct {
	mu   sync.Mutex
	f    *os.File
	ID   string
	Path string
}

// New opens a fresh session transcript. Never returns nil; a Log with no file
// (open failed) silently no-ops on Append.
func New() *Log {
	id := time.Now().Format("20060102-150405")
	dir := Dir()
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, id+".jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	l := &Log{ID: id, Path: path}
	if err == nil {
		l.f = f
	}
	l.Append(Event{Type: "session_start", Meta: map[string]any{"id": id}})
	return l
}

// Append writes one event (thread-safe, best-effort).
func (l *Log) Append(ev Event) {
	if l == nil || l.f == nil {
		return
	}
	if ev.TS == "" {
		ev.TS = time.Now().UTC().Format(time.RFC3339Nano)
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.f.Write(append(b, '\n'))
}

// Close flushes + closes the transcript.
func (l *Log) Close() {
	if l != nil && l.f != nil {
		_ = l.f.Close()
	}
}

// Dir is $XDG_CONFIG/coevolve/sessions (or ~/coevolve/sessions).
func Dir() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		if h, e := os.UserHomeDir(); e == nil {
			dir = h
		}
	}
	return filepath.Join(dir, "coevolve", "sessions")
}
