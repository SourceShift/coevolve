package modes

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/tui"
)

type learningMode struct{}

func init() { tui.RegisterMode(&learningMode{}) }

func (m *learningMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Learning", Title: "learning loop · usage + research", Digit: 2}
}

func (m *learningMode) Update(msg tea.Msg) tea.Cmd { return nil }

func (m *learningMode) View(w, h int) string {
	return tui.HeadStyle.Render("2 · Learning") + tui.SubStyle.Render("   learning loop · usage + research") +
		"\n\n" + tui.SubStyle.Render("panels wiring to real mini-ork data (in progress)")
}
