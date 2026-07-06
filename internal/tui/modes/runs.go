package modes

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/tui"
)

type runsMode struct{}

func init() { tui.RegisterMode(&runsMode{}) }

func (m *runsMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Runs", Title: "runs & epics · scheduler · spawn tree", Digit: 1}
}

func (m *runsMode) Update(msg tea.Msg) tea.Cmd { return nil }

func (m *runsMode) View(w, h int) string {
	return tui.HeadStyle.Render("1 · Runs") + tui.SubStyle.Render("   runs & epics · scheduler · spawn tree") +
		"\n\n" + tui.SubStyle.Render("panels wiring to real mini-ork data (in progress)")
}
