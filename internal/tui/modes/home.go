package modes

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/tui"
)

type homeMode struct{}

func init() { tui.RegisterMode(&homeMode{}) }

func (m *homeMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Home", Title: "run / chat stream", Digit: 0}
}

func (m *homeMode) Update(msg tea.Msg) tea.Cmd { return nil }

func (m *homeMode) View(w, h int) string {
	return tui.HeadStyle.Render("0 · Home") + tui.SubStyle.Render("   run / chat stream") +
		"\n\n" + tui.SubStyle.Render("panels wiring to real mini-ork data (in progress)")
}
