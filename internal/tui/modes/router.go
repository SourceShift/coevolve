package modes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/seams"
	"github.com/sourceshift/coevolve/internal/tui"
)

type routerMode struct {
	totalCalls   int
	totalCallsOK bool
	lanes        []seams.LaneStat
	lanesOK      bool
}

func init() { tui.RegisterMode(&routerMode{}) }

func (m *routerMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Router", Title: "router & LLM perf", Digit: 3}
}

func (m *routerMode) Update(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(tui.RefreshMsg); ok {
		m.totalCalls, m.totalCallsOK = seams.TotalCalls()
		m.lanes, m.lanesOK = seams.LaneLeaderboardWithReward()
	}
	return nil
}

func (m *routerMode) View(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.HeadStyle.Render("3 · ROUTER") +
		tui.SubStyle.Render("   router & LLM perf · window: "+routerWindow(m.totalCalls, m.totalCallsOK)) + "\n\n")

	if !m.lanesOK && !m.totalCallsOK {
		b.WriteString(tui.SubStyle.Render("   state DB not found (MINI_ORK_DB) — offline") + "\n")
		return b.String()
	}

	// Health strip — these need a trained local model to grade against a
	// frontier baseline, which doesn't exist yet, so every tile says so
	// honestly rather than showing a fabricated percentage.
	b.WriteString(tui.TitleStyle.Render("HEALTH") + tui.SubStyle.Render("   needs trained local baseline") + "\n")
	b.WriteString(routerNAtile("parity gate p≥X") + "   " +
		routerNAtile("escalation rate") + "   " +
		routerNAtile("quality vs frontier") + "   " +
		routerNAtile("collapse-guard") + "\n\n")

	// Lane leaderboard — the real centerpiece: share/latency/cost from
	// llm_calls, reward_g from execution_traces where graded traces exist.
	b.WriteString(tui.TitleStyle.Render("LANE LEADERBOARD") + tui.SubStyle.Render("   llm_calls + execution_traces · real") + "\n")
	if !m.lanesOK {
		b.WriteString(tui.SubStyle.Render("   state DB not found (MINI_ORK_DB) — offline") + "\n\n")
	} else if len(m.lanes) == 0 {
		b.WriteString(tui.SubStyle.Render("   no llm_calls rows yet") + "\n\n")
	} else {
		b.WriteString(tui.SubStyle.Render(fmt.Sprintf("   %-18s %-16s %8s %10s   %-16s",
			"lane", "share", "p50 ms", "cost", "reward_g")) + "\n")
		max := len(m.lanes)
		if avail := h - 14; avail > 0 && max > avail {
			max = avail
		}
		if max < 0 {
			max = 0
		}
		for _, s := range m.lanes[:max] {
			b.WriteString(fmt.Sprintf("   %-18s %s %5.1f%% %8dms %10s   %s",
				truncateLane(s.Lane, 18),
				tui.Bar(s.SharePct/100, 10, tui.Teal), s.SharePct,
				s.P50MS,
				tui.AmberStyle.Render(fmt.Sprintf("€%.2f", s.CostUSD)),
				routerRewardCell(s.RewardG)) + "\n")
		}
		b.WriteString("\n")
	}

	// Retune — the threshold is a static echo of the configured gate; live
	// retune isn't wired (it routes through improve→eval, not built yet).
	b.WriteString(tui.TitleStyle.Render("RETUNE") + tui.SubStyle.Render("   parity threshold · static") + "\n")
	b.WriteString("   " + tui.TealStyle.Render("p ≥ 0.62") +
		tui.SubStyle.Render("  (configured gate; live retune routes through improve→eval — not wired yet)") + "\n")
	return b.String()
}

// ── router-scoped render helpers (prefixed to avoid clashing with cost.go) ───

func routerWindow(n int, ok bool) string {
	if !ok {
		return "—"
	}
	return fmt.Sprintf("%d calls (all-time, real)", n)
}

func routerNAtile(label string) string {
	return tui.SubStyle.Render(label+": ") + tui.SubStyle.Render("not available yet")
}

func routerRewardCell(v *float64) string {
	if v == nil {
		return tui.SubStyle.Render("—")
	}
	color := tui.Green
	if *v < 0 {
		color = tui.Red
	}
	frac := (*v + 1) / 2
	return tui.Bar(frac, 8, color) + " " + fmt.Sprintf("%+.2f", *v)
}

func truncateLane(lane string, n int) string {
	if len(lane) <= n {
		return lane
	}
	if n <= 1 {
		return lane[:n]
	}
	return lane[:n-1] + "…"
}
