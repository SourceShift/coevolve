package modes

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/sourceshift/coevolve/internal/plan"
	"github.com/sourceshift/coevolve/internal/run"
	"github.com/sourceshift/coevolve/internal/session"
	"github.com/sourceshift/coevolve/internal/tui"
)

// capPresets are the selectable hard budget caps ($, MO_DAILY_BUDGET_USD).
var capPresets = []float64{1, 2, 3, 4, 5, 10, 25}

// preflight is the interactive plan card shown after a task is submitted and
// before any spend: the user sees which recipe (topology + lanes) will run and
// the cost cap, and confirms or adjusts. Every task goes through mini-ork —
// this is the guide, not a bypass.
type preflight struct {
	task    string
	options []plan.Recipe // installed, suggested-first
	idx     int           // selected recipe
	capIdx  int           // index into capPresets
	live    bool
}

func (p *preflight) recipe() plan.Recipe { return p.options[p.idx] }
func (p *preflight) cap() float64        { return capPresets[p.capIdx] }

// homeMode is the command surface — like Claude/opencode: type a task, the
// configured main LLM does the work and streams back (tool calls live, prose as
// rendered markdown); `/run <task>` escalates to a full mini-ork orchestration.
type homeMode struct {
	input   textinput.Model
	feed    []run.Line
	running bool
	handle  *run.Handle
	cfg     run.Config
	focused bool

	md      *glamour.TermRenderer // cached markdown renderer
	mdWidth int
	mdCache map[string]string // rendered markdown by width:text (avoids per-keystroke re-render)

	history []string // past commands (persisted), oldest→newest
	histIdx int      // browse cursor; == len(history) means "current draft"
	draft   string   // in-progress input stashed while browsing history

	sess *session.Log // per-session JSONL transcript

	pf *preflight // non-nil while showing the plan card (awaiting confirm)
}

type homeLineMsg struct {
	line run.Line
	ok   bool
}

func init() {
	ti := textinput.New()
	ti.Placeholder = "describe a task, or /run <task> for full orchestration…"
	ti.Prompt = "› "
	ti.Focus()
	hist := loadHistory()
	tui.RegisterMode(&homeMode{
		input: ti, cfg: run.DefaultConfig(), focused: true,
		history: hist, histIdx: len(hist),
		sess: session.New(),
	})
}

// logType classifies a streamed line for the session transcript.
func logType(l run.Line) string {
	t := strings.TrimSpace(l.Text)
	switch {
	case l.Markdown:
		return "assistant"
	case l.Err:
		return "status"
	case strings.HasPrefix(t, "●"), strings.HasPrefix(t, "  ⎿"):
		return "tool"
	case strings.HasPrefix(t, "✓") && strings.Contains(t, "tok"):
		return "cost"
	default:
		return "output"
	}
}

func (m *homeMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Home", Title: "command · run mini-ork · live", Digit: 0}
}

func (m *homeMode) CapturesInput() bool { return m.focused }

// InputBusy reports whether the REPL is mid-composition or mid-preflight — when
// so, bare digits stay with the mode (recipe pick / typing) instead of switching
// tabs.
func (m *homeMode) InputBusy() bool {
	return m.pf != nil || strings.TrimSpace(m.input.Value()) != ""
}

func (m *homeMode) wait() tea.Cmd {
	h := m.handle
	return func() tea.Msg {
		if h == nil {
			return homeLineMsg{ok: false}
		}
		l, ok := <-h.Lines
		return homeLineMsg{line: l, ok: ok}
	}
}

func (m *homeMode) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case homeLineMsg:
		if msg.ok {
			m.feed = append(m.feed, msg.line)
			m.sess.Append(session.Event{Type: logType(msg.line), Text: msg.line.Text, Err: msg.line.Err, Markdown: msg.line.Markdown})
			return m.wait()
		}
		m.running = false
		m.sess.Append(session.Event{Type: "run_end"})
		return nil
	case tea.KeyMsg:
		if m.pf != nil { // plan card is up — its keys take precedence
			return m.updatePreflight(msg)
		}
		switch msg.String() {
		case "i":
			if !m.focused {
				m.focused = true
				m.input.Focus()
				return nil
			}
		case "esc":
			m.focused = false
			m.input.Blur()
			return nil
		case "ctrl+u":
			if m.running && m.handle != nil {
				m.handle.Stop()
			}
			return nil
		case "up":
			if m.focused && len(m.history) > 0 {
				if m.histIdx == len(m.history) {
					m.draft = m.input.Value() // stash the current draft
				}
				if m.histIdx > 0 {
					m.histIdx--
					m.input.SetValue(m.history[m.histIdx])
					m.input.CursorEnd()
				}
				return nil
			}
		case "down":
			if m.focused && len(m.history) > 0 {
				if m.histIdx < len(m.history) {
					m.histIdx++
				}
				if m.histIdx >= len(m.history) {
					m.histIdx = len(m.history)
					m.input.SetValue(m.draft)
				} else {
					m.input.SetValue(m.history[m.histIdx])
				}
				m.input.CursorEnd()
				return nil
			}
		case "enter":
			v := strings.TrimSpace(m.input.Value())
			if v == "" || m.running {
				return nil
			}
			m.feed = append(m.feed, run.Line{Text: tui.TealStyle.Render("› ") + v})
			m.sess.Append(session.Event{Type: "user", Text: v})
			m.input.SetValue("")
			m.history = appendHistory(m.history, v)
			m.histIdx = len(m.history)
			m.draft = ""
			// Every task goes through mini-ork — but guide the user first: open
			// the pre-flight plan card (recipe · topology · cost) to confirm.
			m.openPreflight(v)
			return nil
		}
		if m.focused {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return cmd
		}
	}
	return nil
}

// openPreflight builds the plan card for a submitted task: it suggests a recipe
// from the task text, lists the other installed recipes so the user can switch,
// and picks a sane default cost cap. Nothing spends until the user confirms.
func (m *homeMode) openPreflight(task string) {
	task = strings.TrimSpace(strings.TrimPrefix(task, "/run "))
	installed := plan.Installed(plan.RecipesRoot(m.cfg.MiniOrkRoot))
	suggested := plan.Suggest(task)
	// Put the suggestion first so idx 0 is the recommended plan.
	opts := []plan.Recipe{suggested}
	for _, r := range installed {
		if r.Name != suggested.Name {
			opts = append(opts, r)
		}
	}
	// Default cap = the suggestion's suggested cap (nearest preset).
	capIdx := 3 // $4
	for i, c := range capPresets {
		if c >= suggested.CapUSD {
			capIdx = i
			break
		}
	}
	m.pf = &preflight{task: task, options: opts, idx: 0, capIdx: capIdx, live: m.cfg.Live}
}

// updatePreflight handles keys while the plan card is up.
func (m *homeMode) updatePreflight(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.pf = nil
		m.feed = append(m.feed, run.Line{Text: tui.SubStyle.Render("  ✗ cancelled"), Err: true})
		return nil
	case "r", "down", "right":
		m.pf.idx = (m.pf.idx + 1) % len(m.pf.options)
		return nil
	case "shift+r", "up", "left":
		m.pf.idx = (m.pf.idx - 1 + len(m.pf.options)) % len(m.pf.options)
		return nil
	case "c":
		m.pf.capIdx = (m.pf.capIdx + 1) % len(capPresets)
		return nil
	case "l":
		m.pf.live = !m.pf.live
		return nil
	case "enter":
		pf := m.pf
		rec := pf.recipe()
		m.pf = nil
		spec := run.Spec{Task: pf.task, Recipe: rec.Name, CapUSD: pf.cap(), Live: pf.live}
		mode := "dry-run"
		if pf.live {
			mode = tui.AmberStyle.Render("LIVE")
		}
		m.feed = append(m.feed, run.Line{Text: tui.TealStyle.Render(
			"  → dispatching ") + rec.Title + tui.SubStyle.Render(
			" · "+rec.Name+" · cap $"+strconv.FormatFloat(pf.cap(), 'f', 0, 64)+" · "+mode)})
		m.sess.Append(session.Event{Type: "run_start", Text: pf.task, Meta: map[string]any{
			"recipe": rec.Name, "cap_usd": pf.cap(), "live": pf.live}})
		m.handle = run.StartSpec(m.cfg, spec)
		m.running = true
		return m.wait()
	}
	return nil
}

// renderPreflight draws the plan card.
func (m *homeMode) renderPreflight(w int) string {
	pf := m.pf
	rec := pf.recipe()
	label := lipgloss.NewStyle().Foreground(tui.Muted)
	key := lipgloss.NewStyle().Foreground(tui.Violet)
	val := lipgloss.NewStyle().Foreground(tui.FG)
	row := func(k, v string) string {
		return "    " + label.Width(10).Render(k) + v + "\n"
	}
	// Topology: show first ~6 nodes then a count.
	nodes := rec.Nodes
	shown := nodes
	tail := ""
	if len(nodes) > 6 {
		shown = nodes[:6]
		tail = fmt.Sprintf(" … (%d nodes)", len(nodes))
	} else {
		tail = fmt.Sprintf("  (%d nodes)", len(nodes))
	}
	topo := tui.TealStyle.Render(strings.Join(shown, " → ")) + tui.SubStyle.Render(tail)

	mode := "dry-run  " + tui.SubStyle.Render("(no spend)")
	if pf.live {
		mode = tui.AmberStyle.Render("LIVE  ") + tui.SubStyle.Render("(real spend)")
	}

	var b strings.Builder
	b.WriteString(key.Render("  ◆ mini-ork pre-flight ") + tui.SubStyle.Render("— confirm the plan before spend\n\n"))
	b.WriteString(row("task", val.Render(truncate(pf.task, maxWidth(w)-14))))
	b.WriteString(row("recipe", val.Render(rec.Title)+tui.SubStyle.Render("   "+key.Render("r")+" change")))
	b.WriteString("    " + label.Width(10).Render("") + tui.SubStyle.Render(truncate(rec.Purpose, maxWidth(w)-14)) + "\n")
	b.WriteString(row("topology", topo))
	b.WriteString(row("lanes", tui.SubStyle.Render(rec.Lanes)))
	b.WriteString(row("max cost", val.Render("$"+strconv.FormatFloat(pf.cap(), 'f', 0, 64))+tui.SubStyle.Render("   "+key.Render("c")+" change")))
	b.WriteString(row("mode", mode+tui.SubStyle.Render("   "+key.Render("l")+" toggle")))
	b.WriteString("\n  " + tui.SubStyle.Render(key.Render("enter")+" run · "+key.Render("r/c/l")+" adjust · "+key.Render("esc")+" cancel"))
	return b.String()
}

// renderMarkdown formats an assistant prose block (cached renderer per width).
func (m *homeMode) renderMarkdown(md string, width int) string {
	// Cache by width:text so glamour (expensive) runs ONCE per block, not on
	// every keystroke — this is what keeps typing snappy with an answer on screen.
	key := strconv.Itoa(width) + ":" + md
	if v, ok := m.mdCache[key]; ok {
		return v
	}
	if m.md == nil || m.mdWidth != width {
		// Fixed dark style (matches the Coevolve palette) — AutoStyle degrades
		// to plain when it can't detect a TTY inside the alt-screen.
		r, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return md
		}
		m.md, m.mdWidth = r, width
		m.mdCache = map[string]string{} // width changed → drop stale renders
	}
	out, err := m.md.Render(md)
	if err != nil {
		return md
	}
	res := strings.TrimRight(out, "\n")
	if m.mdCache == nil {
		m.mdCache = map[string]string{}
	}
	m.mdCache[key] = res
	return res
}

func (m *homeMode) View(w, h int) string {
	var b strings.Builder
	b.WriteString(tui.HeadStyle.Render("0 · HOME") +
		tui.SubStyle.Render("   command · runs mini-ork with the configured LLM: "+m.cfg.WorkerModel) + "\n\n")

	maxLines := h - 6
	if maxLines < 1 {
		maxLines = 1
	}
	start := 0
	if len(m.feed) > maxLines {
		start = len(m.feed) - maxLines
	}
	if len(m.feed) == 0 {
		b.WriteString(tui.SubStyle.Render("  describe a task — mini-ork runs it. You'll confirm the recipe, topology & cost cap first.\n"))
		if m.sess != nil {
			b.WriteString(tui.SubStyle.Render("  session log · " + m.sess.Path + "\n"))
		}
	}
	wrap := lipgloss.NewStyle().Width(maxWidth(w))
	for _, l := range m.feed[start:] {
		switch {
		case l.Markdown:
			b.WriteString(m.renderMarkdown(l.Text, maxWidth(w)) + "\n")
		case l.Err:
			b.WriteString(wrap.Render(tui.SubStyle.Render(l.Text)) + "\n")
		default:
			b.WriteString(wrap.Render(l.Text) + "\n")
		}
	}

	if m.pf != nil { // plan card replaces the input row while confirming
		b.WriteString("\n" + m.renderPreflight(w) + "\n")
		return b.String()
	}

	status := ""
	if m.running {
		status = tui.AmberStyle.Render("  ● running (ctrl+u to stop)")
	}
	b.WriteString("\n" + m.input.View() + status + "\n")
	hint := "tab: switch tab · enter: plan & run (mini-ork) · esc: unfocus · ⌘K: palette"
	if !m.focused {
		hint = "i: focus input · digits: switch modes · " + hint
	}
	b.WriteString(tui.SubStyle.Render(hint))
	return b.String()
}

func maxWidth(w int) int {
	if w < 20 {
		return 20
	}
	return w - 2
}

// ── command history persistence (like claude/opencode) ───────────────────────

func histFile() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		if h, e := os.UserHomeDir(); e == nil {
			dir = h
		}
	}
	return filepath.Join(dir, "coevolve", "history")
}

func loadHistory() []string {
	b, err := os.ReadFile(histFile())
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		if t := strings.TrimSpace(l); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// appendHistory adds cmd (skipping consecutive dupes), caps at 500, and persists.
func appendHistory(hist []string, cmd string) []string {
	if n := len(hist); n > 0 && hist[n-1] == cmd {
		return hist
	}
	hist = append(hist, cmd)
	if len(hist) > 500 {
		hist = hist[len(hist)-500:]
	}
	p := histFile()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte(strings.Join(hist, "\n")+"\n"), 0o644)
	return hist
}
