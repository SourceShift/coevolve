package modes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sourceshift/coevolve/internal/seams"
	"github.com/sourceshift/coevolve/internal/tui"
)

// logFilter selects which llm_calls rows the stream shows.
type logFilter int

const (
	logFilterAll logFilter = iota
	logFilterOK
	logFilterFail
)

func (f logFilter) label() string {
	switch f {
	case logFilterOK:
		return "ok"
	case logFilterFail:
		return "fail"
	default:
		return "all"
	}
}

type logsMode struct {
	entries   []seams.LogEntry
	entriesOK bool

	artifactDir string
	artifacts   []seams.ArtifactFile
	artifactsOK bool

	filter logFilter
}

func init() { tui.RegisterMode(&logsMode{}) }

func (m *logsMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Logs", Title: "live llm_calls · run artifacts", Digit: 7}
}

func (m *logsMode) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tui.RefreshMsg:
		m.entries, m.entriesOK = seams.RecentLogCalls(40)
		m.artifactDir, m.artifacts, m.artifactsOK = seams.LatestRunArtifacts()
	case tea.KeyMsg:
		switch msg.String() {
		case "a":
			m.filter = logFilterAll
		case "o":
			m.filter = logFilterOK
		case "f":
			m.filter = logFilterFail
		}
	}
	return nil
}

func (m *logsMode) View(w, h int) string {
	var b strings.Builder
	live := tui.SubStyle.Render("○ offline")
	if m.entriesOK {
		live = tui.TealStyle.Render("● live") + tui.SubStyle.Render(" · llm_calls")
	}
	b.WriteString(tui.HeadStyle.Render("7 · LOGS") + "  " + live + "\n")
	b.WriteString(tui.SubStyle.Render(fmt.Sprintf("   filter: %s (a)ll (o)k (f)ail", m.filterChip())) + "\n\n")

	if !m.entriesOK {
		b.WriteString(tui.SubStyle.Render("   state DB not found (MINI_ORK_DB) — offline\n"))
		return b.String()
	}

	asideW := 34
	if w > 0 && w-asideW < 30 {
		asideW = 0 // too narrow to split — stream only
	}
	streamW := w - asideW - 3
	if streamW < 20 {
		streamW = w
		asideW = 0
	}

	stream := m.renderStream(streamW, h)
	if asideW == 0 {
		b.WriteString(stream)
		return b.String()
	}
	aside := m.renderArtifacts(asideW, h)
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, stream, "   ", aside))
	return b.String()
}

func (m *logsMode) filterChip() string {
	labels := []logFilter{logFilterAll, logFilterOK, logFilterFail}
	var chips []string
	for _, f := range labels {
		if f == m.filter {
			chips = append(chips, tui.TealStyle.Render("["+f.label()+"]"))
		} else {
			chips = append(chips, f.label())
		}
	}
	return strings.Join(chips, " ")
}

func (m *logsMode) renderStream(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("STREAM") + tui.SubStyle.Render("   recent llm_calls · newest first") + "\n")

	max := h - 6
	if max < 1 {
		max = 1
	}
	shown := 0
	for _, e := range m.entries {
		ok := e.Status == "success"
		switch m.filter {
		case logFilterOK:
			if !ok {
				continue
			}
		case logFilterFail:
			if ok {
				continue
			}
		}
		if shown >= max {
			break
		}
		shown++
		b.WriteString(logLine(e, w))
		b.WriteString("\n")
	}
	if shown == 0 {
		b.WriteString(tui.SubStyle.Render("   (no calls match filter)\n"))
	}
	return b.String()
}

func logLine(e seams.LogEntry, _ int) string {
	glyph := tui.TealStyle.Render("✓")
	statusColor := tui.Green
	if e.Status != "success" {
		glyph = lipgloss.NewStyle().Foreground(tui.Red).Render("✗")
		statusColor = tui.Red
	}
	ts := shortTS(e.TS)
	who := e.Provider
	if e.Actor != "" {
		who = e.Provider + ":" + e.Actor
	}
	cost := fmt.Sprintf("€%.4f", e.CostUSD)
	run := shortRunID(e.RunID)
	line := fmt.Sprintf(" %s %s  %-22s %s  %8s  %s",
		glyph, tui.SubStyle.Render(ts), truncate(who, 22),
		lipgloss.NewStyle().Foreground(statusColor).Render(fmt.Sprintf("%-7s", e.Status)),
		cost, tui.SubStyle.Render(run))
	if e.ErrorMessage != "" {
		line += "  " + lipgloss.NewStyle().Foreground(tui.Red).Render(truncate(e.ErrorMessage, 40))
	}
	return line
}

func (m *logsMode) renderArtifacts(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("ARTIFACTS") + "\n")
	if !m.artifactsOK {
		b.WriteString(tui.SubStyle.Render("  runs dir not found\n"))
		return b.String()
	}
	b.WriteString(tui.SubStyle.Render(truncate("  "+m.artifactDir, w)) + "\n")
	max := h - 6
	if max < 1 {
		max = 1
	}
	if len(m.artifacts) == 0 {
		b.WriteString(tui.SubStyle.Render("  (empty)\n"))
		return b.String()
	}
	shown := m.artifacts
	if len(shown) > max {
		shown = shown[:max]
	}
	for _, f := range shown {
		b.WriteString(fmt.Sprintf("  %-*s %8s\n", w-11, truncate(f.Name, w-11), humanSize(f.Size)))
	}
	if len(m.artifacts) > max {
		b.WriteString(tui.SubStyle.Render(fmt.Sprintf("  … +%d more\n", len(m.artifacts)-max)))
	}
	return b.String()
}

// ── small local helpers (logs-prefixed to avoid clashing with other modes) ──

func shortTS(ts string) string {
	// "2026-07-04T10:25:12.778Z" -> "10:25:12"
	i := strings.IndexByte(ts, 'T')
	if i < 0 || i+9 > len(ts) {
		return ts
	}
	return ts[i+1 : i+9]
}

func shortRunID(id string) string {
	if len(id) <= 16 {
		return id
	}
	return id[:16] + "…"
}

func truncate(s string, w int) string {
	if w <= 0 || lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	if w <= 1 || len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%dB", n)
	}
}
