// coevolve — the extensible CLI over mini-ork + ContextNest + TraceOtter, with
// opencode as the worker engine. A Bubble Tea 8-mode dashboard (real data only).
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sourceshift/coevolve/internal/core"
	"github.com/sourceshift/coevolve/internal/tui"
)

const version = "0.0.1"

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "", "tui":
		runTUI()
	case "version", "--version", "-v":
		fmt.Printf("coevolve %s\n", version)
	case "capabilities":
		printCapabilities()
	default:
		fmt.Fprintf(os.Stderr, "coevolve %s\nusage: coevolve [tui|capabilities|version]\n", version)
		os.Exit(2)
	}
}

func runTUI() {
	p := tea.NewProgram(tui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "coevolve:", err)
		os.Exit(1)
	}
}

// printCapabilities lists registered integrations by role — a headless smoke of
// the extensibility spine (no TUI, no TTY needed).
func printCapabilities() {
	reg := core.Registered()
	if len(reg) == 0 {
		fmt.Println("no integrations registered yet (built in EPIC-04)")
		return
	}
	for _, k := range []core.Kind{core.KindOrchestrator, core.KindMemory, core.KindLearning, core.KindWorker, core.KindRouter} {
		if names := reg[k]; len(names) > 0 {
			fmt.Printf("%-13s %v\n", k, names)
		}
	}
}
