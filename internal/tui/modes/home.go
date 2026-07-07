package modes

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/sourceshift/coevolve/internal/run"
	"github.com/sourceshift/coevolve/internal/tui"
)

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
	})
}

func (m *homeMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Home", Title: "command · run mini-ork · live", Digit: 0}
}

func (m *homeMode) CapturesInput() bool { return m.focused }

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
			return m.wait()
		}
		m.running = false
		return nil
	case tea.KeyMsg:
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
			m.input.SetValue("")
			m.history = appendHistory(m.history, v)
			m.histIdx = len(m.history)
			m.draft = ""
			m.handle = run.Start(m.cfg, v)
			m.running = true
			return m.wait()
		}
		if m.focused {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return cmd
		}
	}
	return nil
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
		b.WriteString(tui.SubStyle.Render("  type a task below — the main LLM does it and streams here.\n"))
		b.WriteString(tui.SubStyle.Render("  prefix /run to orchestrate the full mini-ork loop instead.\n"))
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

	status := ""
	if m.running {
		status = tui.AmberStyle.Render("  ● running (ctrl+u to stop)")
	}
	b.WriteString("\n" + m.input.View() + status + "\n")
	hint := "tab: switch tab · enter: run · /run <task>: orchestration · esc: unfocus · ⌘K: palette"
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
