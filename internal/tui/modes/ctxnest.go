package modes

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/tui"
)

type contextnestMode struct{}

func init() { tui.RegisterMode(&contextnestMode{}) }

func (m *contextnestMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "ContextNest", Title: "capsule · basins · graph", Digit: 4}
}

func (m *contextnestMode) Update(msg tea.Msg) tea.Cmd { return nil }

func (m *contextnestMode) View(w, h int) string {
	return tui.HeadStyle.Render("4 · ContextNest") + tui.SubStyle.Render("   capsule · basins · graph") +
		"\n\n" + tui.SubStyle.Render("panels wiring to real mini-ork data (in progress)")
}
