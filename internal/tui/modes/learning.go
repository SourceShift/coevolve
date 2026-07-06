// Learning mode: the reflect→improve→eval→promote flywheel. The only mature
// real signal today is GRPO reward_g per lane (execution_traces) — that's the
// centerpiece. Retrain-cycle history, arxiv-adoption, and promotion history
// have no data yet, so they render as "not available yet", never fabricated.
package modes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/seams"
	"github.com/sourceshift/coevolve/internal/tui"
)

type learningMode struct {
	research bool // false = usage view, true = research view

	rewards   []seams.LaneReward
	rewardsOK bool

	domains   []string
	domainsOK bool
}

func init() { tui.RegisterMode(&learningMode{}) }

func (m *learningMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Learning", Title: "learning loop · usage + research", Digit: 2}
}

func (m *learningMode) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tui.RefreshMsg:
		m.rewards, m.rewardsOK = seams.RewardByLane()
		m.domains, m.domainsOK = seams.ObjectiveDomains()
	case tea.KeyMsg:
		if msg.String() == "r" {
			m.research = !m.research
		}
	}
	return nil
}

func (m *learningMode) View(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.HeadStyle.Render("2 · LEARNING LOOP") +
		tui.SubStyle.Render("   "+learnDomainLabel(m.domains, m.domainsOK)) + "\n")
	b.WriteString(learnToggleLine(m.research) + "\n\n")

	if m.research {
		b.WriteString(m.viewResearch(w, h))
	} else {
		b.WriteString(m.viewUsage(w, h))
	}
	return b.String()
}

// ── usage view — REAL centerpiece: GRPO reward_g by lane ────────────────────

func (m *learningMode) viewUsage(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("GRPO REWARD BY LANE") +
		tui.SubStyle.Render("   execution_traces · avg(reward_g) · real") + "\n")

	if !m.rewardsOK {
		b.WriteString(tui.SubStyle.Render("   state DB not found (MINI_ORK_DB) — offline\n"))
		return b.String()
	}
	if len(m.rewards) == 0 {
		b.WriteString(tui.SubStyle.Render("   no graded traces yet (reward_g IS NULL for all rows)\n"))
	}
	max := len(m.rewards)
	if lim := h - 14; max > lim {
		max = lim
	}
	if max < 0 {
		max = 0
	}
	barWidth := 24
	for _, r := range m.rewards[:max] {
		frac := (r.RewardG + 1) / 2 // map reward_g [-1,+1] -> bar [0,1]
		color := tui.Green
		if r.RewardG < 0 {
			color = tui.Red
		}
		b.WriteString(fmt.Sprintf("   %-14s %s %+.2f %s\n",
			r.Lane, tui.Bar(frac, barWidth, color), r.RewardG,
			tui.SubStyle.Render(fmt.Sprintf("(n=%d)", r.Samples))))
	}

	b.WriteString("\n" + tui.TitleStyle.Render("REFLECT → IMPROVE → EVAL → PROMOTE") + "\n")
	b.WriteString(tui.SubStyle.Render("   " + learnNotAvailable("no promotion history yet — retrain cycles haven't run") + "\n"))
	return b.String()
}

// ── research view — arxiv R&D cycle: all not-available-yet except the ────────
// static blast-radius policy tiers, which are real config, not data.

func (m *learningMode) viewResearch(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("ARXIV R&D CYCLE") + "\n")
	b.WriteString(tui.SubStyle.Render("   " + learnNotAvailable("no arxiv-driven proposals ingested yet") + "\n\n"))

	b.WriteString(tui.TitleStyle.Render("ADOPTED FROM RESEARCH") + "\n")
	b.WriteString(tui.SubStyle.Render("   " + learnNotAvailable("no adoption events recorded yet") + "\n\n"))

	b.WriteString(tui.TitleStyle.Render("BLAST RADIUS") + tui.SubStyle.Render("   static policy · real config") + "\n")
	tiers := []struct{ scope, action string }{
		{"prompt / workflow tweak", "auto-adopt"},
		{"router-policy change", "awaiting sign-off"},
		{"safety-critical change", "never auto"},
	}
	for _, t := range tiers {
		b.WriteString(fmt.Sprintf("   %-24s %s\n", t.scope, tui.AmberStyle.Render(t.action)))
	}
	return b.String()
}

// ── small helpers (prefixed "learn" — money/kpi/humanInt already live in cost.go) ──

func learnDomainLabel(domains []string, ok bool) string {
	if !ok {
		return "objective_domain: —"
	}
	if len(domains) == 0 {
		return "objective_domain: (none yet)"
	}
	return "objective_domain: " + strings.Join(domains, ", ")
}

func learnToggleLine(research bool) string {
	view := "USAGE"
	if research {
		view = "RESEARCH"
	}
	return tui.SubStyle.Render("   [r] toggle usage/research · viewing ") + tui.TealStyle.Render(view)
}

func learnNotAvailable(reason string) string {
	return "not available yet — " + reason
}
