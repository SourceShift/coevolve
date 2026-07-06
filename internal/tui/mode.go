package tui

import (
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ModeMeta identifies a mode in the tab bar.
type ModeMeta struct {
	Key   string // "Home", "Cost", …
	Title string
	Digit int // 0-7
}

// Mode is one screen of the dashboard. Modes self-register (init → RegisterMode)
// and implement this contract, so adding a mode is additive — the same
// extensibility idiom as integrations.
type Mode interface {
	Meta() ModeMeta
	// Update handles messages routed to this mode; returns a cmd (e.g. async load).
	Update(msg tea.Msg) tea.Cmd
	// View renders the mode body within (w,h) — chrome (tabs/status) excluded.
	View(w, h int) string
}

// RefreshMsg is broadcast on a timer so modes reload real data from the seams.
type RefreshMsg struct{}

var modeRegistry []Mode

// RegisterMode adds a mode (call from init()).
func RegisterMode(m Mode) { modeRegistry = append(modeRegistry, m) }

// Modes returns registered modes ordered by digit.
func Modes() []Mode {
	out := append([]Mode(nil), modeRegistry...)
	sort.Slice(out, func(i, j int) bool { return out[i].Meta().Digit < out[j].Meta().Digit })
	return out
}

// tickCmd schedules the next refresh.
func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return RefreshMsg{} })
}
