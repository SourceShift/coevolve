package modes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sourceshift/coevolve/internal/seams"
	"github.com/sourceshift/coevolve/internal/tui"
)

// homeMode is mode 0 — the run-flow home screen. Live dispatch from the
// dashboard is a later epic, so for now it replays the most recent real run's
// node sequence from the state DB. Labeled "last run", never as a live feed.
type homeMode struct {
	head   seams.RunRow
	headOK bool

	steps   []seams.HomeStep
	stepsOK bool

	lanes   []seams.LaneStat
	lanesOK bool
}

func init() { tui.RegisterMode(&homeMode{}) }

func (m *homeMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Home", Title: "run / chat stream", Digit: 0}
}

func (m *homeMode) Update(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(tui.RefreshMsg); ok {
		m.head, m.headOK = seams.RunRow{}, false
		if runs, ok := seams.RecentRuns(1); ok && len(runs) > 0 {
			m.head, m.headOK = runs[0], true
		}
		if m.headOK {
			m.steps, m.stepsOK = seams.RunSteps(m.head.RunID)
		} else {
			m.steps, m.stepsOK = nil, false
		}
		m.lanes, m.lanesOK = seams.LaneLeaderboard()
	}
	return nil
}

func (m *homeMode) View(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.HeadStyle.Render("0 · HOME") +
		tui.SubStyle.Render("   last run · replay from state DB (live dispatch not wired yet)") + "\n\n")

	rightW := 32
	if w-rightW-4 < 24 {
		rightW = 0 // too narrow to split — stack instead
	}
	leftW := w - rightW - 2
	if leftW < 1 {
		leftW = w
	}

	left := m.renderStream(leftW, h-8)
	if rightW == 0 {
		b.WriteString(left + "\n\n" + m.renderRail(w))
	} else {
		right := m.renderRail(rightW)
		b.WriteString(homeJoinCols(left, right, leftW, rightW) + "\n\n")
	}

	b.WriteString(m.renderPromptRow(w))
	return b.String()
}

// ── left: the last-run stream ────────────────────────────────────────────────

func (m *homeMode) renderStream(w, maxRows int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("LAST RUN") + tui.SubStyle.Render("   llm_calls · real, not live") + "\n")

	if !m.headOK {
		if !m.lanesOK {
			b.WriteString(tui.SubStyle.Render("   state DB not found (MINI_ORK_DB) — offline\n"))
		} else {
			b.WriteString(tui.SubStyle.Render("   no runs recorded in state DB yet\n"))
		}
		return b.String()
	}

	b.WriteString(fmt.Sprintf("   %s  %s  %s  %s\n",
		tui.TealStyle.Render(m.head.RunID),
		tui.SubStyle.Render("task "+orDash(m.head.Recipe)),
		tui.SubStyle.Render("status "+orDash(m.head.Status)),
		tui.AmberStyle.Render(money(m.head.CostUSD, true))))

	if !m.stepsOK {
		b.WriteString(tui.SubStyle.Render("   step trace unavailable\n"))
		return b.String()
	}
	if len(m.steps) == 0 {
		b.WriteString(tui.SubStyle.Render("   no llm_calls rows for this run_id\n"))
		return b.String()
	}

	avail := maxRows - 3
	if avail < 0 {
		avail = 0
	}
	max := len(m.steps)
	if max > avail {
		max = avail
	}
	shown := m.steps
	truncated := false
	if max < len(shown) {
		shown = shown[:max]
		truncated = true
	}
	for i, s := range shown {
		b.WriteString(fmt.Sprintf("   %3d  %s %-16s %-9s %-16s %8s %7dms\n",
			i+1,
			homeStatusGlyph(s.Status),
			homeTrunc(s.Actor, 16),
			homeTrunc(s.Provider, 9),
			homeTrunc(s.ModelID, 16),
			tui.AmberStyle.Render(fmt.Sprintf("€%.3f", s.CostUSD)),
			s.DurationMS))
	}
	if truncated {
		b.WriteString(tui.SubStyle.Render(fmt.Sprintf("   … %d more steps (window too short)\n", len(m.steps)-max)))
	}
	return b.String()
}

// ── right rail: PIPELINE / ROUTER SPLIT / MEMORY / COST cards ────────────────

func (m *homeMode) renderRail(w int) string {
	var b strings.Builder
	b.WriteString(m.railPipeline(w) + "\n")
	b.WriteString(m.railRouterSplit(w) + "\n")
	b.WriteString(m.railMemory(w) + "\n")
	b.WriteString(m.railCost(w))
	return b.String()
}

func (m *homeMode) railPipeline(_ int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("PIPELINE") + tui.SubStyle.Render("  node sequence · real") + "\n")
	if !m.headOK || len(m.steps) == 0 {
		b.WriteString(tui.SubStyle.Render(" no run to trace\n"))
		return b.String()
	}
	// Collapse the raw step trace into the distinct actors in first-seen
	// order — the pipeline's node sequence — with a per-node ok/fail glyph.
	type node struct {
		actor string
		calls int
		ok    bool
	}
	var order []string
	byActor := map[string]*node{}
	for _, s := range m.steps {
		n, seen := byActor[s.Actor]
		if !seen {
			n = &node{actor: s.Actor, ok: true}
			byActor[s.Actor] = n
			order = append(order, s.Actor)
		}
		n.calls++
		if s.Status != "success" {
			n.ok = false
		}
	}
	for _, a := range order {
		n := byActor[a]
		glyph := tui.TealStyle.Render("✓")
		if !n.ok {
			glyph = lipglossRed("✗")
		}
		b.WriteString(fmt.Sprintf(" %s %-16s ×%d\n", glyph, homeTrunc(n.actor, 16), n.calls))
	}
	return b.String()
}

func (m *homeMode) railRouterSplit(_ int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("ROUTER SPLIT") + tui.SubStyle.Render("  all-time · real") + "\n")
	if !m.lanesOK || len(m.lanes) == 0 {
		b.WriteString(tui.SubStyle.Render(" no llm_calls yet\n"))
		return b.String()
	}
	var frontierCalls, localCalls, total int
	for _, l := range m.lanes {
		total += l.Calls
		if homeIsFrontier(l.Lane) {
			frontierCalls += l.Calls
		} else {
			localCalls += l.Calls
		}
	}
	if total == 0 {
		b.WriteString(tui.SubStyle.Render(" no calls yet\n"))
		return b.String()
	}
	frontierFrac := float64(frontierCalls) / float64(total)
	b.WriteString(fmt.Sprintf(" local   %s %2.0f%%\n", tui.Bar(1-frontierFrac, 14, tui.Teal), 100*(1-frontierFrac)))
	b.WriteString(fmt.Sprintf(" frontier%s %2.0f%%\n", tui.Bar(frontierFrac, 14, tui.Amber), 100*frontierFrac))
	b.WriteString(tui.SubStyle.Render(" heuristic: model_id has opus/claude/sonnet/gpt\n"))
	b.WriteString(tui.SubStyle.Render(" → frontier, else local (imprecise, no lane table)\n"))
	return b.String()
}

func (m *homeMode) railMemory(_ int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("MEMORY") + tui.SubStyle.Render("  recall · ContextNest") + "\n")
	b.WriteString(" " + tui.SubStyle.Render("recall hit rate: not available yet") + "\n")
	b.WriteString(tui.SubStyle.Render(" (needs a ContextNest query wired here)\n"))
	return b.String()
}

func (m *homeMode) railCost(_ int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("COST") + tui.SubStyle.Render("  this run · real") + "\n")
	if !m.headOK {
		b.WriteString(tui.SubStyle.Render(" —\n"))
		return b.String()
	}
	b.WriteString(" " + kpi("total", money(m.head.CostUSD, true), tui.Amber) + "\n")
	b.WriteString(" " + kpi("steps", fmt.Sprintf("%d", len(m.steps)), tui.Amber) + "\n")
	return b.String()
}

// ── bottom: prompt row (visual only — submit is not wired) ───────────────────

func (m *homeMode) renderPromptRow(w int) string {
	box := "› describe a task…"
	chips := "/run   /plan   /mem"
	line := tui.SubStyle.Render(strings.Repeat("─", max0(w))) + "\n"
	line += tui.TitleStyle.Render(box) + "    " + tui.TealStyle.Render(chips) + "\n"
	line += tui.SubStyle.Render("  (visual only — submitting a task here is not wired; live dispatch is a later epic)")
	return line
}

// ── small local helpers (prefixed "home" per the shared-API contract) ────────

func homeStatusGlyph(status string) string {
	switch status {
	case "success":
		return tui.TealStyle.Render("✓")
	case "failed":
		return lipglossRed("✗")
	default:
		return tui.SubStyle.Render("·")
	}
}

func homeIsFrontier(modelID string) bool {
	l := strings.ToLower(modelID)
	for _, kw := range []string{"opus", "claude", "sonnet", "gpt"} {
		if strings.Contains(l, kw) {
			return true
		}
	}
	return false
}

func homeTrunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// lipglossRed renders a glyph in the shared Red palette color without
// redefining a package-level style (Cost/Router don't need Red today).
func lipglossRed(s string) string {
	return lipgloss.NewStyle().Foreground(tui.Red).Render(s)
}

// homeJoinCols lays the stream and rail side by side, left column padded/cut
// to leftW so the rail starts at a stable column regardless of stream width.
func homeJoinCols(left, right string, leftW, rightW int) string {
	lLines := strings.Split(left, "\n")
	rLines := strings.Split(right, "\n")
	n := len(lLines)
	if len(rLines) > n {
		n = len(rLines)
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		var l, r string
		if i < len(lLines) {
			l = lLines[i]
		}
		if i < len(rLines) {
			r = rLines[i]
		}
		b.WriteString(homePad(l, leftW) + " │ " + r)
		if i < n-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func homePad(s string, w int) string {
	pad := w - lipgloss.Width(s)
	if pad <= 0 {
		return s
	}
	return s + strings.Repeat(" ", pad)
}
