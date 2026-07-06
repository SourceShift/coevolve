package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Command is one entry in the ⌘K palette. JumpDigit >=0 switches to that mode
// when selected; -1 means the command is an action stub (dispatch wired later).
type Command struct {
	Name      string
	Group     string
	Help      string
	JumpDigit int
}

// commandCatalogue mirrors the design's palette (mini-ork subcommands, grouped).
// View commands jump to their mode; the rest are action stubs for now.
var commandCatalogue = []Command{
	// Runs
	{"run", "Runs", "start a task through the loop", 0},
	{"plan", "Runs", "plan a task", -1},
	{"execute", "Runs", "execute the current plan", -1},
	{"verify", "Runs", "verify an artifact", -1},
	{"resume", "Runs", "resume a run", 1},
	{"rollback", "Runs", "roll a run back", 1},
	{"spawn", "Runs", "spawn a sub-agent", 1},
	{"epics", "Runs", "epics & dependency cascade", 1},
	{"scheduler", "Runs", "the priority scheduler", 1},
	// Learning
	{"reflect", "Learning", "extract gradients from runs", 2},
	{"improve", "Learning", "propose a candidate workflow", 2},
	{"eval", "Learning", "held-out eval gate", 2},
	{"promote", "Learning", "promote a candidate", 2},
	{"self-improve", "Learning", "the outer improvement loop", 2},
	// Router
	{"route", "Router", "router & LLM performance", 3},
	{"lane", "Router", "lane leaderboard", 3},
	{"classify", "Router", "classify a task's route", 3},
	{"topology", "Router", "node DAG & lane→provider", 5},
	// Memory
	{"basins", "Memory", "ContextNest attractor basins", 4},
	{"capsule", "Memory", "the context capsule", 4},
	{"inbox", "Memory", "attention inbox", 4},
	{"sessions", "Memory", "sessions by file/feature/intent", 4},
	{"retrieve", "Memory", "retrieve fragments", 4},
	// Ops
	{"metrics", "Ops", "cost, savings & budget", 6},
	{"usage-report", "Ops", "usage report", 6},
	{"lifetime", "Ops", "lifetime economics", 6},
	{"logs", "Ops", "live logs & artifacts", 7},
	{"serve", "Ops", "start the web console", -1},
	{"watchdog", "Ops", "the run watchdog", -1},
	{"review", "Ops", "reviewer panel", -1},
	{"bugs", "Ops", "collected bugs", -1},
	{"update", "Ops", "update mini-ork", -1},
	{"init", "Ops", "init a project", -1},
}

// fuzzyMatch reports whether all runes of q appear in s in order (subsequence).
func fuzzyMatch(s, q string) bool {
	s, q = strings.ToLower(s), strings.ToLower(q)
	i := 0
	for _, r := range s {
		if i < len(q) && rune(q[i]) == r {
			i++
		}
	}
	return i == len(q)
}

func filterCommands(q string) []Command {
	if q == "" {
		return commandCatalogue
	}
	var out []Command
	for _, c := range commandCatalogue {
		if fuzzyMatch(c.Name, q) || fuzzyMatch(c.Group, q) {
			out = append(out, c)
		}
	}
	return out
}

var (
	paletteBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Teal).Padding(0, 1)
	palSel     = lipgloss.NewStyle().Foreground(lipgloss.Color("#07090D")).Background(Teal)
	palGroup   = lipgloss.NewStyle().Foreground(Amber)
)

// renderPalette draws the command palette centered over the given size.
func renderPalette(query string, idx, width, height int) string {
	cmds := filterCommands(query)
	var b strings.Builder
	b.WriteString(TealStyle.Render("⌘K ") + TitleStyle.Render("run a command…") + "  " + SubStyle.Render(query+"▏") + "\n")
	b.WriteString(SubStyle.Render(strings.Repeat("─", 44)) + "\n")
	lastGroup := ""
	shown := 0
	for i, c := range cmds {
		if shown >= height-6 {
			break
		}
		if c.Group != lastGroup {
			b.WriteString(palGroup.Render(strings.ToUpper(c.Group)) + "\n")
			lastGroup = c.Group
		}
		jump := ""
		if c.JumpDigit >= 0 {
			jump = SubStyle.Render("  →" + string(rune('0'+c.JumpDigit)))
		}
		line := padRight(c.Name, 14) + SubStyle.Render(c.Help) + jump
		if i == idx {
			b.WriteString(palSel.Render("▸ "+padRight(c.Name, 14)) + SubStyle.Render(c.Help) + jump + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
		shown++
	}
	b.WriteString(SubStyle.Render("↑↓ navigate · ⏎ select · esc close · " + itoa(len(cmds)) + " commands"))
	return centerOverlay(paletteBox.Render(b.String()), width, height)
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// centerOverlay places content roughly centered within width×height.
func centerOverlay(content string, width, height int) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
