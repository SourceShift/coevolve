// Package tui is the Bubble Tea dashboard — the 8-mode Coevolve control plane.
// Each mode is a screen; digit keys 0-7 switch. Data comes from mini-ork via the
// seams (real data only, provenance-gated).
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type modeDef struct {
	key   string
	title string
}

// The 8 modes from the design (Coevolve CLI.dc.html).
var modes = []modeDef{
	{"Home", "run / chat stream"},
	{"Runs", "runs & epics · scheduler · spawn tree"},
	{"Learning", "learning loop · usage + research"},
	{"Router", "router & LLM perf"},
	{"ContextNest", "capsule · basins · graph"},
	{"Topology", "node DAG · lane→provider · health"},
	{"Cost", "metrics · savings · budget"},
	{"Logs", "live .live.log · artifacts"},
}

// Coevolve palette (from the design): teal=local/healthy, amber=frontier/cost,
// violet=memory, graphite grounds.
var (
	teal   = lipgloss.Color("#2BC4A8")
	amber  = lipgloss.Color("#E6A24A")
	muted  = lipgloss.Color("#5A6475")
	fg     = lipgloss.Color("#EAEEF4")
	subtle = lipgloss.Color("#3E4654")

	tabActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("#07090D")).Background(teal).Bold(true).Padding(0, 1)
	tabInactive = lipgloss.NewStyle().Foreground(muted).Padding(0, 1)
	titleStyle  = lipgloss.NewStyle().Foreground(fg).Bold(true)
	subStyle    = lipgloss.NewStyle().Foreground(muted)
	statusStyle = lipgloss.NewStyle().Foreground(muted).Background(lipgloss.Color("#0E1218")).Padding(0, 1)
	dotTeal     = lipgloss.NewStyle().Foreground(teal).Render("●")
	dotAmber    = lipgloss.NewStyle().Foreground(amber).Render("●")
)

// Model is the root Bubble Tea model.
type Model struct {
	active        int
	width, height int
	// live status segments (wired to real seams in later epics)
	statusRun   string
	statusLane  string
	statusCost  string
	statusCN    string
}

// New builds the dashboard model.
func New() Model {
	return Model{
		active:     0,
		statusRun:  "idle",
		statusLane: "—",
		statusCost: "€0.00",
		statusCN:   "CN ?",
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch s := msg.String(); s {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "0", "1", "2", "3", "4", "5", "6", "7":
			m.active = int(s[0] - '0')
		case "left", "h":
			if m.active > 0 {
				m.active--
			}
		case "right", "l":
			if m.active < len(modes)-1 {
				m.active++
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "starting Coevolve…"
	}
	return strings.Join([]string{
		m.header(),
		m.body(),
		m.statusBar(),
	}, "\n")
}

func (m Model) header() string {
	brand := titleStyle.Render("coevolve") + subStyle.Render("  sovereign ai-dev platform")
	var tabs []string
	for i, md := range modes {
		chip := fmt.Sprintf("%d %s", i, md.key)
		if i == m.active {
			tabs = append(tabs, tabActive.Render(chip))
		} else {
			tabs = append(tabs, tabInactive.Render(chip))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	rule := subStyle.Render(strings.Repeat("─", max(0, m.width)))
	return brand + "\n" + bar + "\n" + rule
}

func (m Model) body() string {
	md := modes[m.active]
	// Placeholder body — each mode's real panels land in EPIC-05/06/09.
	h := max(1, m.height-8)
	title := titleStyle.Render(fmt.Sprintf("%d · %s", m.active, strings.ToUpper(md.key)))
	sub := subStyle.Render(md.title)
	hint := subStyle.Render("digits 0-7 switch · ←/→ prev/next · q quit   (panels wire to real mini-ork data in the next epics)")
	content := title + "\n" + sub + "\n\n" + hint
	return lipgloss.NewStyle().Height(h).Render(content)
}

func (m Model) statusBar() string {
	segs := []string{
		"◐ " + m.statusRun,
		"lane " + m.statusLane,
		"session " + m.statusCost,
		dotTeal + " " + m.statusCN,
	}
	left := statusStyle.Render(strings.Join(segs, "   "))
	right := lipgloss.NewStyle().Foreground(subtle).Render("coevolve · real-data-only")
	gap := max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
