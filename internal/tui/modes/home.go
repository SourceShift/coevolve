package modes

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/run"
	"github.com/sourceshift/coevolve/internal/tui"
)

// homeMode is the command surface — like Claude/opencode: type a task, the
// configured main LLM (opencode) does the work and streams back; `/run <task>`
// escalates to a full mini-ork orchestration.
type homeMode struct {
	input   textinput.Model
	feed    []string
	running bool
	handle  *run.Handle
	cfg     run.Config
	focused bool
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
	tui.RegisterMode(&homeMode{input: ti, cfg: run.DefaultConfig(), focused: true})
}

func (m *homeMode) Meta() tui.ModeMeta {
	return tui.ModeMeta{Key: "Home", Title: "command · run mini-ork · live", Digit: 0}
}

// CapturesInput routes keystrokes here while the prompt is focused.
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
			m.feed = append(m.feed, homeRenderLine(msg.line))
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
		case "enter":
			v := strings.TrimSpace(m.input.Value())
			if v == "" || m.running {
				return nil
			}
			m.feed = append(m.feed, tui.TealStyle.Render("› ")+v)
			m.input.SetValue("")
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

func homeRenderLine(l run.Line) string {
	if l.Err {
		return tui.SubStyle.Render(l.Text)
	}
	return l.Text
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
	for _, l := range m.feed[start:] {
		b.WriteString(l + "\n")
	}

	status := ""
	if m.running {
		status = tui.AmberStyle.Render("  ● running (ctrl+u to stop)")
	}
	b.WriteString("\n" + m.input.View() + status + "\n")
	hint := "enter: run · /run <task>: full orchestration · esc: unfocus · ⌘K: palette"
	if !m.focused {
		hint = "i: focus input · digits: switch modes · " + hint
	}
	b.WriteString(tui.SubStyle.Render(hint))
	return b.String()
}
