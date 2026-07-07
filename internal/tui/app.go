// Package tui is the Bubble Tea dashboard — the 8-mode Coevolve control plane.
// Modes self-register; digit keys 0-7 switch. Data comes from mini-ork via the
// seams (real data only, provenance-gated).
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sourceshift/coevolve/internal/seams"
)

// Coevolve palette (from the design), exported so mode files share it.
var (
	Teal   = lipgloss.Color("#2BC4A8")
	Amber  = lipgloss.Color("#E6A24A")
	Violet = lipgloss.Color("#8B7CD8")
	Green  = lipgloss.Color("#3FBE86")
	Red    = lipgloss.Color("#E0705F")
	Muted  = lipgloss.Color("#5A6475")
	FG     = lipgloss.Color("#EAEEF4")
	Subtle = lipgloss.Color("#3E4654")

	TitleStyle = lipgloss.NewStyle().Foreground(FG).Bold(true)
	SubStyle   = lipgloss.NewStyle().Foreground(Muted)
	TealStyle  = lipgloss.NewStyle().Foreground(Teal)
	AmberStyle = lipgloss.NewStyle().Foreground(Amber)
	HeadStyle  = lipgloss.NewStyle().Foreground(Teal).Bold(true)

	tabActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("#07090D")).Background(Teal).Bold(true).Padding(0, 1)
	tabInactive = lipgloss.NewStyle().Foreground(Muted).Padding(0, 1)
	statusStyle = lipgloss.NewStyle().Foreground(Muted).Background(lipgloss.Color("#0E1218")).Padding(0, 1)
)

// Bar renders a horizontal bar of `frac` (0..1) at `width` cells in `color`.
func Bar(frac float64, width int, color lipgloss.Color) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	fill := int(frac * float64(width))
	on := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", fill))
	off := lipgloss.NewStyle().Foreground(Subtle).Render(strings.Repeat("░", width-fill))
	return on + off
}

// Model is the root Bubble Tea model.
type Model struct {
	active        int
	width, height int
	statusRun     string
	statusLane    string
	statusCost    string
	statusCN      string
	// overlays
	paletteOpen  bool
	paletteQuery string
	paletteIdx   int
	helpOpen     bool
}

func New() Model {
	return Model{statusRun: "idle", statusLane: "—", statusCost: "€0.00", statusCN: "CN ?"}
}

func (m Model) Init() tea.Cmd { return tickCmd() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ms := Modes()
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case RefreshMsg:
		m.refreshStatus()
		for _, md := range ms { // broadcast refresh to all modes
			if c := md.Update(msg); c != nil {
				cmds = append(cmds, c)
			}
		}
		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)
	case tea.KeyMsg:
		if m.helpOpen { // any key closes the help overlay
			m.helpOpen = false
			return m, nil
		}
		if m.paletteOpen {
			return m.updatePalette(msg)
		}
		// If the active mode owns a focused input, keystrokes go to it (typing);
		// only ctrl+c and ctrl+k stay global.
		if m.active < len(ms) {
			if ic, ok := ms[m.active].(InputCapturer); ok && ic.CapturesInput() {
				switch msg.String() {
				case "ctrl+c":
					return m, tea.Quit
				case "ctrl+k":
					m.paletteOpen, m.paletteQuery, m.paletteIdx = true, "", 0
					return m, nil
				case "tab": // cycle tabs even while the prompt is focused
					m.active = (m.active + 1) % len(ms)
					return m, nil
				case "shift+tab":
					m.active = (m.active - 1 + len(ms)) % len(ms)
					return m, nil
				default:
					return m, ms[m.active].Update(msg)
				}
			}
		}
		switch s := msg.String(); s {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "ctrl+k", ":":
			m.paletteOpen, m.paletteQuery, m.paletteIdx = true, "", 0
			return m, nil
		case "?":
			m.helpOpen = true
			return m, nil
		case "0", "1", "2", "3", "4", "5", "6", "7":
			if d := int(s[0] - '0'); d < len(ms) {
				m.active = d
			}
			return m, nil
		case "left":
			if m.active > 0 {
				m.active--
			}
			return m, nil
		case "right":
			if m.active < len(ms)-1 {
				m.active++
			}
			return m, nil
		case "tab":
			m.active = (m.active + 1) % len(ms)
			return m, nil
		case "shift+tab":
			m.active = (m.active - 1 + len(ms)) % len(ms)
			return m, nil
		}
	}
	// route everything else to the active mode
	if m.active < len(ms) {
		if c := ms[m.active].Update(msg); c != nil {
			cmds = append(cmds, c)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) refreshStatus() {
	if c, ok := seams.TotalCost(); ok {
		m.statusCost = fmt.Sprintf("€%.2f", c)
	}
	if _, ok := seams.DB(); ok {
		m.statusRun = "db ok"
	} else {
		m.statusRun = "db —"
	}
}

func (m Model) updatePalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cmds := filterCommands(m.paletteQuery)
	switch msg.String() {
	case "esc":
		m.paletteOpen = false
	case "up":
		if m.paletteIdx > 0 {
			m.paletteIdx--
		}
	case "down":
		if m.paletteIdx < len(cmds)-1 {
			m.paletteIdx++
		}
	case "enter":
		if m.paletteIdx < len(cmds) {
			if d := cmds[m.paletteIdx].JumpDigit; d >= 0 && d < len(Modes()) {
				m.active = d
			}
		}
		m.paletteOpen = false
	case "backspace":
		if n := len(m.paletteQuery); n > 0 {
			m.paletteQuery = m.paletteQuery[:n-1]
			m.paletteIdx = 0
		}
	default:
		if s := msg.String(); len(s) == 1 {
			m.paletteQuery += s
			m.paletteIdx = 0
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "starting Coevolve…"
	}
	body := m.body()
	if m.helpOpen {
		body = renderHelp(m.width, max(1, m.height-8))
	} else if m.paletteOpen {
		body = renderPalette(m.paletteQuery, m.paletteIdx, m.width, max(1, m.height-8))
	}
	return m.header() + "\n" + body + "\n" + m.statusBar()
}

func (m Model) header() string {
	brand := TitleStyle.Render("coevolve") + SubStyle.Render("  sovereign ai-dev platform")
	var tabs []string
	for i, md := range Modes() {
		chip := fmt.Sprintf("%d %s", md.Meta().Digit, md.Meta().Key)
		if i == m.active {
			tabs = append(tabs, tabActive.Render(chip))
		} else {
			tabs = append(tabs, tabInactive.Render(chip))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	rule := SubStyle.Render(strings.Repeat("─", max(0, m.width)))
	return brand + "\n" + bar + "\n" + rule
}

func (m Model) body() string {
	ms := Modes()
	h := max(1, m.height-8)
	if m.active >= len(ms) {
		return lipgloss.NewStyle().Height(h).Render(SubStyle.Render("no modes registered"))
	}
	return lipgloss.NewStyle().Height(h).Render(ms[m.active].View(m.width, h))
}

func (m Model) statusBar() string {
	segs := []string{
		"◐ " + m.statusRun,
		"lane " + m.statusLane,
		"session " + m.statusCost,
		TealStyle.Render("●") + " " + m.statusCN,
	}
	left := statusStyle.Render(strings.Join(segs, "   "))
	right := lipgloss.NewStyle().Foreground(Subtle).Render("coevolve · real-data-only")
	gap := max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
