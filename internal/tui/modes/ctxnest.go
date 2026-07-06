package modes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/seams"
	"github.com/sourceshift/coevolve/internal/tui"
)

// ctxTab is one of the ContextNest mode's internal tabs, switchable by key
// while this mode is active (see contextnestMode.Update).
type ctxTab int

const (
	ctxTabBasins ctxTab = iota
	ctxTabCapsule
	ctxTabInbox
	ctxTabSessions
	ctxTabGraph
)

type contextnestMode struct {
	tab ctxTab

	healthy bool

	basins   []seams.Basin
	basinsOK bool

	capsule   string
	capsuleOK bool

	inbox   []seams.InboxItem
	inboxOK bool
}

func init() { tui.RegisterMode(&contextnestMode{}) }

func (m *contextnestMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "ContextNest", Title: "capsule · basins · graph", Digit: 4}
}

func (m *contextnestMode) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tui.RefreshMsg:
		m.healthy = seams.CNHealth()
		m.basins, m.basinsOK = seams.CNBasins()
		m.capsule, m.capsuleOK = seams.CNCapsule()
		m.inbox, m.inboxOK = seams.CNInbox()
	case tea.KeyMsg:
		switch msg.String() {
		case "b":
			m.tab = ctxTabBasins
		case "c":
			m.tab = ctxTabCapsule
		case "i":
			m.tab = ctxTabInbox
		case "s":
			m.tab = ctxTabSessions
		case "g":
			m.tab = ctxTabGraph
		}
	}
	return nil
}

func (m *contextnestMode) View(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.HeadStyle.Render("4 · CONTEXTNEST") + tui.SubStyle.Render("   capsule · basins · graph") + "  ")
	if m.healthy {
		b.WriteString(tui.TealStyle.Render("● substrate up"))
	} else {
		b.WriteString(tui.SubStyle.Render("○ substrate offline"))
	}
	b.WriteString("\n")
	b.WriteString(m.tabBar() + "\n\n")

	switch m.tab {
	case ctxTabBasins:
		b.WriteString(m.viewBasins(w, h))
	case ctxTabCapsule:
		b.WriteString(m.viewCapsule(w, h))
	case ctxTabInbox:
		b.WriteString(m.viewInbox(w, h))
	case ctxTabSessions:
		b.WriteString(m.viewSessions())
	case ctxTabGraph:
		b.WriteString(m.viewGraph())
	}
	return b.String()
}

func (m *contextnestMode) tabBar() string {
	tabs := []struct {
		key   ctxTab
		label string
	}{
		{ctxTabBasins, "b·basins"},
		{ctxTabCapsule, "c·capsule"},
		{ctxTabInbox, "i·inbox"},
		{ctxTabSessions, "s·sessions"},
		{ctxTabGraph, "g·graph"},
	}
	var chips []string
	for _, t := range tabs {
		if t.key == m.tab {
			chips = append(chips, tui.TealStyle.Render("["+t.label+"]"))
		} else {
			chips = append(chips, tui.SubStyle.Render(" "+t.label+" "))
		}
	}
	return strings.Join(chips, " ")
}

// cnOffline renders the honest "ContextNest offline" marker shared by every
// tab when the substrate is unreachable — never fabricate basins/fragments.
func cnOffline() string {
	return tui.SubStyle.Render(fmt.Sprintf("   ContextNest offline (%s)\n", seams.CNBaseURL()))
}

func (m *contextnestMode) viewBasins(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("BASINS") + tui.SubStyle.Render("   topic clusters · /api/v1/field/basins") + "\n")
	if !m.basinsOK {
		b.WriteString(cnOffline())
		return b.String()
	}
	if len(m.basins) == 0 {
		b.WriteString(tui.SubStyle.Render("   no basins yet\n"))
		return b.String()
	}
	maxMass := 1
	for _, bs := range m.basins {
		if bs.Mass > maxMass {
			maxMass = bs.Mass
		}
	}
	max := len(m.basins)
	if lim := h - 6; lim > 0 && max > lim {
		max = lim
	}
	for _, bs := range m.basins[:max] {
		name := bs.Label
		if name == "" {
			name = bs.ID
		}
		frac := float64(bs.Mass) / float64(maxMass)
		b.WriteString(fmt.Sprintf("   %-24s %-10s traces=%-4d %s\n",
			name, bs.Source, bs.Mass, tui.Bar(frac, 20, tui.Violet)))
	}
	return b.String()
}

func (m *contextnestMode) viewCapsule(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("CAPSULE") + tui.SubStyle.Render("   /api/v1/prompt-context/capsule") + "\n")
	if !m.capsuleOK {
		b.WriteString(cnOffline())
		return b.String()
	}
	if strings.TrimSpace(m.capsule) == "" {
		b.WriteString(tui.SubStyle.Render("   capsule is empty (no atoms matched the default window)\n"))
		return b.String()
	}
	lines := strings.Split(m.capsule, "\n")
	max := len(lines)
	if lim := h - 4; lim > 0 && max > lim {
		max = lim
	}
	for _, ln := range lines[:max] {
		b.WriteString("   " + ln + "\n")
	}
	return b.String()
}

func (m *contextnestMode) viewInbox(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("INBOX") + tui.SubStyle.Render("   attention items · /api/v1/inbox") + "\n")
	if !m.inboxOK {
		b.WriteString(cnOffline())
		return b.String()
	}
	if len(m.inbox) == 0 {
		b.WriteString(tui.SubStyle.Render("   inbox is empty\n"))
		return b.String()
	}
	max := len(m.inbox)
	if lim := h - 6; lim > 0 && max > lim {
		max = lim
	}
	for _, it := range m.inbox[:max] {
		urgency := it.Urgency()
		if urgency == "" {
			urgency = "—"
		}
		kind := it.Kind()
		if kind == "" {
			kind = "?"
		}
		content := strings.ReplaceAll(strings.TrimSpace(it.Content), "\n", " ")
		if len(content) > 100 {
			content = content[:97] + "..."
		}
		b.WriteString(fmt.Sprintf("   [%-4s %-8s] %s\n", urgency, kind, content))
	}
	return b.String()
}

func (m *contextnestMode) viewSessions() string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("SESSIONS") + tui.SubStyle.Render("   by-file · by-feature · by-intent") + "\n")
	b.WriteString(tui.SubStyle.Render("   not available yet (needs a session/query focus to drive by-file/by-feature/by-intent lookups)\n"))
	return b.String()
}

func (m *contextnestMode) viewGraph() string {
	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render("GRAPH") + tui.SubStyle.Render("   /api/v1/connections") + "\n")
	b.WriteString(tui.SubStyle.Render("   not available yet (needs a focused fragment/node id — connections are neighbour lookups, not a listing endpoint)\n"))
	return b.String()
}
