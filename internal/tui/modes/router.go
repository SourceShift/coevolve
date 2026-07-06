package modes

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/tui"
)

type routerMode struct{}

func init() { tui.RegisterMode(&routerMode{}) }

func (m *routerMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Router", Title: "router & LLM perf", Digit: 3}
}

func (m *routerMode) Update(msg tea.Msg) tea.Cmd { return nil }

func (m *routerMode) View(w, h int) string {
	return tui.HeadStyle.Render("3 · Router") + tui.SubStyle.Render("   router & LLM perf") +
		"\n\n" + tui.SubStyle.Render("panels wiring to real mini-ork data (in progress)")
}
