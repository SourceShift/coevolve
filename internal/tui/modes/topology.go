package modes

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/tui"
)

type topologyMode struct{}

func init() { tui.RegisterMode(&topologyMode{}) }

func (m *topologyMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Topology", Title: "node DAG · lane→provider · health", Digit: 5}
}

func (m *topologyMode) Update(msg tea.Msg) tea.Cmd { return nil }

func (m *topologyMode) View(w, h int) string {
	return tui.HeadStyle.Render("5 · Topology") + tui.SubStyle.Render("   node DAG · lane→provider · health") +
		"\n\n" + tui.SubStyle.Render("panels wiring to real mini-ork data (in progress)")
}
