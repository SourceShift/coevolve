// Package modes holds the dashboard mode screens. Each mode self-registers
// (init → tui.RegisterMode) and reads REAL data from the seams; anything that
// needs the not-yet-trained local model is rendered as "not available yet".
package modes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/seams"
	"github.com/sourceshift/coevolve/internal/tui"
)

type costMode struct {
	total, today     float64
	totalOK, todayOK bool
	spend            []seams.ProviderSpend
	spendOK          bool
}

func init() { tui.RegisterMode(&costMode{}) }

func (c *costMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Cost", Title: "metrics · savings · budget", Digit: 6}
}

func (c *costMode) Update(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(tui.RefreshMsg); ok {
		c.total, c.totalOK = seams.TotalCost()
		c.today, c.todayOK = seams.TodayCost()
		c.spend, c.spendOK = seams.SpendByProvider()
	}
	return nil
}

func (c *costMode) View(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.HeadStyle.Render("6 · COST") + tui.SubStyle.Render("   llm_calls economics · savings vs frontier-only") + "\n\n")

	// KPI row — all REAL from llm_calls; savings is NOT_YET (needs a trained
	// local baseline to compare against) so we say so, not fake a percentage.
	b.WriteString(kpi("today", money(c.today, c.todayOK), tui.Amber))
	b.WriteString("   ")
	b.WriteString(kpi("lifetime", money(c.total, c.totalOK), tui.Amber))
	b.WriteString("   ")
	b.WriteString(kpi("saved vs frontier-only", tui.SubStyle.Render("not available yet"), tui.Muted))
	b.WriteString("\n" + tui.SubStyle.Render("   (savings needs a trained local baseline — see Learning)") + "\n\n")

	// Spend-by-provider — REAL.
	b.WriteString(tui.TitleStyle.Render("SPEND BY PROVIDER") + tui.SubStyle.Render("   llm_calls · real") + "\n")
	if !c.spendOK {
		b.WriteString(tui.SubStyle.Render("   state DB not found (MINI_ORK_DB) — offline\n"))
		return b.String()
	}
	b.WriteString(tui.SubStyle.Render(fmt.Sprintf("   %-16s %8s %12s %12s", "provider", "calls", "tokens", "cost")) + "\n")
	max := len(c.spend)
	if max > h-10 {
		max = h - 10
	}
	if max < 0 {
		max = 0
	}
	for _, s := range c.spend[:max] {
		b.WriteString(fmt.Sprintf("   %-16s %8d %12s %12s\n",
			s.Provider, s.Calls, humanInt(s.Tokens),
			tui.AmberStyle.Render(fmt.Sprintf("€%.2f", s.CostUSD))))
	}
	return b.String()
}

// ── small shared render helpers (used by other modes too) ────────────────────

func kpi(label, value string, _ any) string {
	return tui.SubStyle.Render(label) + "  " + tui.TitleStyle.Render(value)
}

func money(v float64, ok bool) string {
	if !ok {
		return "—"
	}
	return fmt.Sprintf("€%.2f", v)
}

func humanInt(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	default:
		return fmt.Sprintf("%d", n)
	}
}
