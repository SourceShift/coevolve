package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var keybindings = [][2]string{
	{"switch mode", "0–7"},
	{"prev / next mode", "← / →"},
	{"command palette", "⌘K / :"},
	{"keybindings (this)", "?"},
	{"quit", "q / ^C"},
	{"close overlay", "esc"},
	{"learning flywheel toggle", "r"},
	{"logs filter all/ok/fail", "a / o / f"},
	{"contextnest tabs", "b c i s g"},
}

var helpBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Amber).Padding(0, 1)

func renderHelp(width, height int) string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("KEYBINDINGS") + "\n")
	b.WriteString(SubStyle.Render(strings.Repeat("─", 34)) + "\n")
	for _, kb := range keybindings {
		b.WriteString(padRight(kb[0], 26) + TealStyle.Render(kb[1]) + "\n")
	}
	b.WriteString(SubStyle.Render("\nesc / ? to close"))
	return centerOverlay(helpBox.Render(b.String()), width, height)
}
