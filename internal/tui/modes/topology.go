// Package modes: Topology mode — the mini-ork universal loop (a fixed node
// DAG) rendered with its REAL lane->provider resolution (config/agents.yaml,
// falling back to llm_calls when that file can't be read), REAL per-provider
// health from llm_calls, and a REAL watchdog/grounded-rejection safety strip.
// Same provenance rule as every other mode: "not available yet" only when a
// source genuinely has nothing, offline marker only when the state DB itself
// is missing.
package modes

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sourceshift/coevolve/internal/seams"
	"github.com/sourceshift/coevolve/internal/tui"
)

type topologyMode struct {
	dbOK bool

	lanes       map[string]string
	lanesOK     bool
	lanesSource string

	health   []seams.ProviderHealth
	healthOK bool

	safety   seams.SafetyStrip
	safetyOK bool
}

func init() { tui.RegisterMode(&topologyMode{}) }

func (m *topologyMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Topology", Title: "node DAG · lane→provider · health", Digit: 5}
}

func (m *topologyMode) Update(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(tui.RefreshMsg); ok {
		_, m.dbOK = seams.DB()

		if lm, ok := seams.LaneProviderMap(); ok {
			m.lanes, m.lanesOK = lm, true
			m.lanesSource = "config/agents.yaml lanes"
		} else if lm, ok := seams.LaneProviderFallback(); ok {
			m.lanes, m.lanesOK = lm, true
			m.lanesSource = "llm_calls actor→provider (agents.yaml unreadable)"
		} else {
			m.lanes, m.lanesOK = nil, false
			m.lanesSource = ""
		}

		m.health, m.healthOK = seams.ProviderHealthByProvider()
		m.safety, m.safetyOK = seams.SafetyStripStatus()
	}
	return nil
}

func (m *topologyMode) View(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.HeadStyle.Render("5 · TOPOLOGY") +
		tui.SubStyle.Render("   node DAG · lane→provider · health") + "\n\n")

	if !m.dbOK {
		b.WriteString(tui.SubStyle.Render("   state DB not found (MINI_ORK_DB) — offline\n"))
		return b.String()
	}

	// NODE DAG — mini-ork's universal loop is a fixed 6-node chain; each
	// node's box shows node/lane and its resolved provider.
	b.WriteString(tui.TitleStyle.Render("NODE DAG") +
		tui.SubStyle.Render("   classify→plan→execute→verify→review→reflect · "+topoSourceLabel(m)) + "\n")
	b.WriteString("   " + topoRenderChain(m.lanes, m.lanesOK) + "\n\n")

	// LANE -> PROVIDER table.
	b.WriteString(tui.TitleStyle.Render("LANE → PROVIDER") + tui.SubStyle.Render("   "+topoSourceLabel(m)) + "\n")
	if m.lanesOK && len(m.lanes) > 0 {
		b.WriteString(topoRenderLaneTable(m.lanes, h))
	} else {
		b.WriteString(tui.SubStyle.Render("   not available yet\n"))
	}
	b.WriteString("\n")

	// PROVIDER HEALTH — most-recent call status + mean duration per provider.
	b.WriteString(tui.TitleStyle.Render("PROVIDER HEALTH") + tui.SubStyle.Render("   llm_calls · real") + "\n")
	if m.healthOK && len(m.health) > 0 {
		b.WriteString(tui.SubStyle.Render(fmt.Sprintf("   %-14s %8s %10s   %s\n", "provider", "calls", "p50", "status")))
		for _, hh := range m.health {
			mark := tui.TealStyle.Render("● healthy")
			if !hh.Healthy {
				mark = lipgloss.NewStyle().Foreground(tui.Red).Render("● degraded")
			}
			b.WriteString(fmt.Sprintf("   %-14s %8d %8dms   %s\n", hh.Provider, hh.Calls, hh.P50MS, mark))
		}
	} else {
		b.WriteString(tui.SubStyle.Render("   not available yet\n"))
	}
	b.WriteString("\n")

	// SAFETY STRIP — watchdog_aborts / grounded_rejections. Both tables are
	// real (append-only, present in schema); a count of 0 means "queried,
	// nothing recorded yet", which we still show as real, not "unavailable".
	b.WriteString(tui.TitleStyle.Render("SAFETY") +
		tui.SubStyle.Render("   watchdog_aborts · grounded_rejections · real") + "\n")
	if m.safetyOK {
		b.WriteString(fmt.Sprintf("   %s %s   %s %s %s\n",
			tui.SubStyle.Render("watchdog aborts:"), tui.TitleStyle.Render(humanInt(int64(m.safety.WatchdogAborts))),
			tui.SubStyle.Render("grounded rejections:"), tui.TitleStyle.Render(humanInt(int64(m.safety.GroundedRejections))),
			tui.SubStyle.Render(fmt.Sprintf("(%d unconsumed)", m.safety.UnconsumedRejections))))
	} else {
		b.WriteString(tui.SubStyle.Render("   not available yet\n"))
	}
	return b.String()
}

// ── topology-mode-local helpers (prefixed "topo" to avoid clashing with other modes) ──

func topoSourceLabel(m *topologyMode) string {
	if m.lanesOK {
		return m.lanesSource
	}
	return "not available yet"
}

// topoRenderChain draws the fixed 6-node universal loop as a horizontal
// chain, each box labeled node/lane -> resolved provider ("?" when the lane
// map itself is unavailable).
func topoRenderChain(lanes map[string]string, ok bool) string {
	parts := make([]string, 0, len(seams.WorkflowNodes))
	for _, node := range seams.WorkflowNodes {
		lane := seams.NodeLane(node)
		provider := "?"
		if ok {
			if p, exists := lanes[lane]; exists {
				provider = p
			}
		}
		box := tui.TealStyle.Render(node) + tui.SubStyle.Render("/"+lane+" → ") + tui.AmberStyle.Render(provider)
		parts = append(parts, "["+box+"]")
	}
	return strings.Join(parts, tui.SubStyle.Render(" ─ "))
}

func topoRenderLaneTable(lanes map[string]string, h int) string {
	keys := make([]string, 0, len(lanes))
	for k := range lanes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	max := len(keys)
	if avail := h - 30; avail > 0 && avail < max {
		max = avail
	}
	if max <= 0 {
		max = len(keys)
	}

	var b strings.Builder
	for _, k := range keys[:max] {
		b.WriteString(fmt.Sprintf("   %-16s %s\n", k, tui.AmberStyle.Render(lanes[k])))
	}
	return b.String()
}
