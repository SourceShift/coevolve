// Package modes: Runs mode — runs & epics, scheduler counters, dependency
// cascade. Same provenance rule as every other mode: real data from the
// seams, "not available yet" when a table/column genuinely has nothing to
// show, and a hard offline marker when the state DB itself is missing.
package modes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sourceshift/coevolve/internal/seams"
	"github.com/sourceshift/coevolve/internal/tui"
)

// runsDailyBudgetCapUSD is the configured daily spend cap shown alongside the
// REAL today-spend sum. It is a policy constant, not a DB fact.
const runsDailyBudgetCapUSD = 50.0

type runsMode struct {
	dbOK bool

	todayCost float64
	todayOK   bool

	counts   seams.EpicCounts
	countsOK bool

	epics   []seams.EpicRow
	epicsOK bool

	deps   []seams.EpicDepRow
	depsOK bool

	runs   []seams.RunRow
	runsOK bool
}

func init() { tui.RegisterMode(&runsMode{}) }

func (m *runsMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Runs", Title: "runs & epics · scheduler · spawn tree", Digit: 1}
}

func (m *runsMode) Update(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(tui.RefreshMsg); ok {
		_, m.dbOK = seams.DB()
		m.todayCost, m.todayOK = seams.TodayCost()
		m.counts, m.countsOK = seams.EpicStatusSummary()
		m.epics, m.epicsOK = seams.RecentEpics(10)
		m.deps, m.depsOK = seams.DependencyCascade(8)
		m.runs, m.runsOK = seams.RecentRuns(12)
	}
	return nil
}

func (m *runsMode) View(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.HeadStyle.Render("1 · RUNS & EPICS") +
		tui.SubStyle.Render("   scheduler · dependency cascade · recent runs") + "\n\n")

	if !m.dbOK {
		b.WriteString(tui.SubStyle.Render("   state DB not found (MINI_ORK_DB) — offline") + "\n")
		return b.String()
	}

	// KPI row — today's spend is REAL (llm_calls); the €50 cap is a
	// configured policy value, not queried.
	b.WriteString(kpi("today", money(m.todayCost, m.todayOK), tui.Amber))
	b.WriteString("   ")
	b.WriteString(kpi("daily cap", fmt.Sprintf("€%.2f", runsDailyBudgetCapUSD), tui.Muted))
	if m.todayOK {
		frac := m.todayCost / runsDailyBudgetCapUSD
		b.WriteString("   " + tui.Bar(frac, 20, runsBudgetColor(frac)))
	}
	b.WriteString("\n\n")

	// Scheduler counters — derived from epics.status.
	b.WriteString(tui.TitleStyle.Render("SCHEDULER") + tui.SubStyle.Render("   epics.status · real") + "\n")
	if m.countsOK && m.counts.Total > 0 {
		b.WriteString("   " +
			runsCounterChip("ready", m.counts.Ready, tui.Teal) + "   " +
			runsCounterChip("running", m.counts.Running, tui.Amber) + "   " +
			runsCounterChip("in-review", m.counts.InReview, tui.Violet) + "   " +
			runsCounterChip("blocked", m.counts.Blocked, tui.Red) + "   " +
			runsCounterChip("done", m.counts.Done, tui.Green) +
			"\n")
	} else {
		b.WriteString(tui.SubStyle.Render("   not available yet\n"))
	}
	b.WriteString("\n")

	// EPICS — from the epics table.
	b.WriteString(tui.TitleStyle.Render("EPICS") + tui.SubStyle.Render("   epics · real") + "\n")
	if m.epicsOK && len(m.epics) > 0 {
		b.WriteString(tui.SubStyle.Render(fmt.Sprintf("   %-22s %-34s %-12s %-10s", "id", "title", "status", "lane/group")) + "\n")
		max := len(m.epics)
		if max > 8 {
			max = 8
		}
		for _, e := range m.epics[:max] {
			laneGroup := e.Lane
			if e.GroupID != "" {
				if laneGroup != "" {
					laneGroup += "/"
				}
				laneGroup += e.GroupID
			}
			if laneGroup == "" {
				laneGroup = "—"
			}
			b.WriteString(fmt.Sprintf("   %-22s %-34s %s %s\n",
				runsTrunc(e.ID, 22), runsTrunc(e.Title, 34),
				runsStatusStyle(e.Status).Render(fmt.Sprintf("%-12s", e.Status)),
				tui.SubStyle.Render(runsTrunc(laneGroup, 24))))
		}
	} else {
		b.WriteString(tui.SubStyle.Render("   not available yet\n"))
	}
	b.WriteString("\n")

	// DEPENDENCY CASCADE — from epic_dependencies, unresolved edges first.
	b.WriteString(tui.TitleStyle.Render("DEPENDENCY CASCADE") + tui.SubStyle.Render("   epic_dependencies · real") + "\n")
	if m.depsOK && len(m.deps) > 0 {
		for _, d := range m.deps {
			mark := tui.AmberStyle.Render("○ unresolved")
			if d.Resolved {
				mark = tui.TealStyle.Render("● resolved")
			}
			downstream := d.ToStatus
			if downstream == "" {
				downstream = "?"
			}
			b.WriteString(fmt.Sprintf("   %-18s -> %-18s (%s)  %s  %s\n",
				runsTrunc(d.FromEpicID, 18), runsTrunc(d.ToEpicID, 18), d.Kind, mark,
				runsStatusStyle(downstream).Render(downstream)))
		}
	} else {
		b.WriteString(tui.SubStyle.Render("   not available yet\n"))
	}
	b.WriteString("\n")

	// RUNS — from llm_calls, grouped by run_id.
	b.WriteString(tui.TitleStyle.Render("RECENT RUNS") + tui.SubStyle.Render("   llm_calls · real") + "\n")
	if !m.runsOK {
		b.WriteString(tui.SubStyle.Render("   not available yet\n"))
		return b.String()
	}
	if len(m.runs) == 0 {
		b.WriteString(tui.SubStyle.Render("   not available yet\n"))
		return b.String()
	}
	b.WriteString(tui.SubStyle.Render(fmt.Sprintf("   %-12s %-24s %-12s %6s %10s", "run_id", "recipe/feature", "status", "iters", "cost")) + "\n")
	max := len(m.runs)
	if max > h-24 && h-24 > 0 {
		max = h - 24
	}
	if max <= 0 || max > len(m.runs) {
		max = len(m.runs)
	}
	for _, r := range m.runs[:max] {
		b.WriteString(fmt.Sprintf("   %-12s %-24s %s %6d %10s\n",
			runsTrunc(r.RunID, 12), runsTrunc(r.Recipe, 24),
			runsStatusStyle(r.Status).Render(fmt.Sprintf("%-12s", r.Status)),
			r.Iters, tui.AmberStyle.Render(fmt.Sprintf("€%.2f", r.CostUSD))))
	}
	return b.String()
}

// ── runs-mode-local helpers (prefixed to avoid clashing with other modes) ────

func runsCounterChip(label string, n int, color lipgloss.Color) string {
	return tui.SubStyle.Render(label+" ") + lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%d", n))
}

func runsStatusStyle(status string) lipgloss.Style {
	switch status {
	case "done":
		return lipgloss.NewStyle().Foreground(tui.Green)
	case "in progress", "running":
		return lipgloss.NewStyle().Foreground(tui.Amber)
	case "blocked":
		return lipgloss.NewStyle().Foreground(tui.Red)
	case "not started":
		return lipgloss.NewStyle().Foreground(tui.Muted)
	case "":
		return tui.SubStyle
	default:
		return lipgloss.NewStyle().Foreground(tui.Violet)
	}
}

func runsBudgetColor(frac float64) lipgloss.Color {
	switch {
	case frac >= 0.9:
		return tui.Red
	case frac >= 0.6:
		return tui.Amber
	default:
		return tui.Green
	}
}

func runsTrunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
