package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func render(m Model) string {
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 96, Height: 20})
	return nm.(Model).View()
}

func TestRendersEightModes(t *testing.T) {
	out := render(New())
	for _, name := range []string{"Home", "Runs", "Learning", "Router", "ContextNest", "Topology", "Cost", "Logs"} {
		if !strings.Contains(out, name) {
			t.Fatalf("tab bar missing mode %q", name)
		}
	}
	if !strings.Contains(out, "real-data-only") {
		t.Fatal("status bar missing honesty tag")
	}
}

func TestDigitSwitchesMode(t *testing.T) {
	m := New()
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 96, Height: 20})
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'6'}})
	if got := nm.(Model).active; got != 6 {
		t.Fatalf("digit 6 → active=%d, want 6", got)
	}
	if !strings.Contains(nm.(Model).View(), "COST") {
		t.Fatal("mode 6 body should show COST")
	}
}

func TestPrintDashboard(t *testing.T) {
	m := New()
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 96, Height: 16})
	nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'6'}})
	t.Log("\n" + nm.(Model).View())
}
