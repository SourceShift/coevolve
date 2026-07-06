package modes

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/tui"
)

type logsMode struct{}

func init() { tui.RegisterMode(&logsMode{}) }

func (m *logsMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Logs", Title: "live .live.log · artifacts", Digit: 7}
}

func (m *logsMode) Update(msg tea.Msg) tea.Cmd { return nil }

func (m *logsMode) View(w, h int) string {
	return tui.HeadStyle.Render("7 · Logs") + tui.SubStyle.Render("   live .live.log · artifacts") +
		"\n\n" + tui.SubStyle.Render("panels wiring to real mini-ork data (in progress)")
}
