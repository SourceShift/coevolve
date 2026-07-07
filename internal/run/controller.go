// Package run drives the actual work: it sends a prompt to the configured main
// LLM (opencode) and streams its output, or — for /run — spawns a full mini-ork
// orchestration. This is what makes Coevolve's command section behave like
// Claude/opencode: you type, it works, the flow streams back.
package run

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sourceshift/coevolve/internal/seams"
)

// Config is the resolved runtime: where mini-ork lives, where work happens, and
// the main LLM the worker uses.
type Config struct {
	MiniOrkRoot string // MINI_ORK_ROOT (a mo-fix checkout)
	TargetCWD   string // repo the work operates on
	WorkerModel string // the configured main LLM (opencode provider/model)
	Live        bool   // false → mini-ork runs go dry (no spend) unless opted in
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// DefaultConfig resolves from the environment with sane fallbacks.
func DefaultConfig() Config {
	cwd, _ := os.Getwd()
	return Config{
		MiniOrkRoot: env("MINI_ORK_ROOT", "/Volumes/docker-ssd/ps/mo-fix"),
		TargetCWD:   env("MO_TARGET_CWD", cwd),
		WorkerModel: env("COEVOLVE_WORKER_MODEL", env("MO_OPENCODE_MODEL", "deepinfra/deepseek-ai/DeepSeek-V4-Flash")),
		Live:        os.Getenv("COEVOLVE_LIVE") == "1",
	}
}

// Line is one streamed output line. Markdown lines are a completed block of
// assistant prose the renderer should format (bold/headers/code) via glamour.
type Line struct {
	Text     string
	Err      bool
	Markdown bool
}

// Handle is a running job; Cancel stops it (and reaps the process).
type Handle struct {
	Lines  <-chan Line
	cancel context.CancelFunc
}

func (h *Handle) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
}

// Start dispatches `input`. Plain text → the configured main LLM (opencode),
// streaming its work. "/run <task>" → a full mini-ork orchestration. Returns a
// Handle whose Lines channel closes when the job ends.
// mentionsMiniOrk detects a task that clearly wants the orchestrator (so it's
// routed to mini-ork even without the /run prefix), rather than the chat LLM.
func mentionsMiniOrk(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "mini-ork") || strings.Contains(l, "mini ork") ||
		strings.Contains(l, "miniork") || strings.Contains(l, "orchestrate")
}

func Start(cfg Config, input string) *Handle {
	trimmed := strings.TrimSpace(input)
	// Plain text → the main LLM (opencode). Explicit /run OR a task naming
	// mini-ork → full orchestration.
	if trimmed != "" && !strings.HasPrefix(trimmed, "/run ") && !mentionsMiniOrk(trimmed) {
		return StartServe(context.Background(), cfg, trimmed)
	}
	return StartSpec(cfg, Spec{Task: strings.TrimSpace(strings.TrimPrefix(trimmed, "/run "))})
}

// Spec is a fully-resolved mini-ork run: the task, the explicit recipe to run
// (so we never dead-end on task_class=generic), and a hard cost cap. This is
// what the interactive pre-flight produces once the user confirms the plan.
type Spec struct {
	Task   string  // the work to do
	Recipe string  // explicit recipe name; "" → let mini-ork classify
	CapUSD float64 // hard budget cap (MO_DAILY_BUDGET_USD); 0 → leave unset
	Live   bool    // spend real money; false → dry-run
}

// StartSpec dispatches mini-ork for a confirmed Spec and streams design-style
// node rows enriched with real per-node cost/lane.
func StartSpec(cfg Config, spec Spec) *Handle {
	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan Line, 64)
	cmd, banner := buildSpec(ctx, cfg, spec)

	go func() {
		defer close(out)
		out <- Line{Text: banner}
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err := cmd.Start(); err != nil {
			out <- Line{Text: "failed to start: " + err.Error(), Err: true}
			return
		}
		done := make(chan struct{}, 2)
		enr := &miniOrkEnricher{cfg: cfg}
		defer enr.close()
		pump := func(r io.Reader, isErr bool) {
			sc := bufio.NewScanner(r)
			sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for sc.Scan() {
				txt := stripANSI(sc.Text())
				if strings.TrimSpace(txt) == "" {
					continue
				}
				row, keep := enr.row(txt) // design-style node rows + real cost/lane
				if !keep {
					continue
				}
				select {
				case out <- Line{Text: row, Err: isErr}:
				case <-ctx.Done():
					done <- struct{}{}
					return
				}
			}
			done <- struct{}{}
		}
		go pump(stdout, false)
		go pump(stderr, true)
		<-done
		<-done
		err := cmd.Wait()
		if ctx.Err() != nil {
			out <- Line{Text: "■ stopped", Err: true}
			return
		}
		if err != nil {
			out <- Line{Text: fmt.Sprintf("■ exited: %v", err), Err: true}
			return
		}
		out <- Line{Text: "✓ done"}
	}()

	return &Handle{Lines: out, cancel: cancel}
}

// buildSpec constructs the mini-ork command for a confirmed Spec + a banner.
// When Spec.Recipe is set we pass it explicitly (`mini-ork run <recipe>
// <kickoff>`), which skips the classifier — the fix for tasks that would
// otherwise classify as `generic` and dead-end with "could not resolve recipe".
func buildSpec(ctx context.Context, cfg Config, spec Spec) (*exec.Cmd, string) {
	task := strings.TrimSpace(spec.Task)
	kick := writeKickoff(task)
	bin := filepath.Join(cfg.MiniOrkRoot, "bin", "mini-ork")
	var args []string
	if spec.Recipe != "" {
		args = []string{"run", spec.Recipe, kick}
	} else {
		args = []string{"run", kick}
	}
	c := exec.CommandContext(ctx, bin, args...)
	c.Dir = cfg.TargetCWD
	live := spec.Live || cfg.Live
	dry := "1"
	if live {
		dry = "0"
	}
	c.Env = append(os.Environ(),
		"MINI_ORK_ROOT="+cfg.MiniOrkRoot,
		"MO_TARGET_CWD="+cfg.TargetCWD,
		"MO_OPENCODE_MODEL="+cfg.WorkerModel,
		"MO_IMPLEMENTER_LANE=codex", // reliable headless coder (opencode-run hangs on tools)
		"MINI_ORK_NONINTERACTIVE=1", // auto-answer the profile gate from the kickoff
		"MINI_ORK_DRY_RUN="+dry,
	)
	if spec.CapUSD > 0 {
		c.Env = append(c.Env, fmt.Sprintf("MO_DAILY_BUDGET_USD=%.2f", spec.CapUSD))
	}
	mode := "dry-run"
	if live {
		mode = "LIVE"
	}
	rec := spec.Recipe
	if rec == "" {
		rec = "auto"
	}
	return c, fmt.Sprintf("› mini-ork · %s · %s · %s", rec, mode, task)
}

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// miniOrkRow maps mini-ork's key=value stage output to a clean node-stream row
// matching the design (● <node> · <detail>). Drops noise (returns keep=false),
// passes through real agent/error output. This makes /run read like the design's
// Home run-stream instead of raw stdout.
func miniOrkRow(line string) (row string, keep bool) {
	l := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(l, "task_class="):
		return "● classify · " + strings.TrimPrefix(l, "task_class="), true
	case strings.HasPrefix(l, "run_id="):
		return "  " + l, true
	case strings.HasPrefix(l, "plan_path="), strings.HasPrefix(l, "plan_status="):
		return "● plan · ready", true
	case strings.HasPrefix(l, "profile_status=needs_answers"):
		return "⚠ needs answers · auto-answering from kickoff", true
	case strings.HasPrefix(l, "artifact_path="):
		p := strings.Trim(strings.TrimPrefix(l, "artifact_path="), `"`)
		if p == "" {
			return "● execute", true
		}
		return "● execute · " + filepath.Base(p), true
	case strings.Contains(l, `"pass":false`), strings.Contains(l, "REQUEST_CHANGES"):
		return "● verify · ✗", true
	case strings.Contains(l, `"pass":true`), strings.HasPrefix(l, "verdict=") && strings.Contains(l, "approve"):
		return "● verify · ✓", true
	case strings.HasPrefix(l, "reflect"), strings.Contains(l, "gradient"):
		return "● reflect · → ContextNest", true
	case strings.HasPrefix(l, "workflow_version="), strings.HasPrefix(l, "kickoff="),
		strings.HasPrefix(l, "profile_confidence="), strings.HasPrefix(l, "profile_questions="),
		strings.HasPrefix(l, "profile_path="), strings.HasPrefix(l, "[dry-run]"):
		return "", false // drop noise
	default:
		return line, true // pass through real agent output / errors
	}
}

// miniOrkEnricher augments the /run node-stream with REAL per-node cost + lane,
// read from the run's own .mini-ork/state.db as mini-ork writes to it.
type miniOrkEnricher struct {
	cfg      Config
	mu       sync.Mutex
	runID    string
	db       *sql.DB
	dbTried  bool
	lastCost float64
}

func (e *miniOrkEnricher) row(line string) (string, bool) {
	l := strings.TrimSpace(line)
	if id, ok := strings.CutPrefix(l, "run_id="); ok {
		e.mu.Lock()
		e.runID = id
		e.mu.Unlock()
	}
	row, keep := miniOrkRow(line)
	if !keep {
		return "", false
	}
	if strings.HasPrefix(row, "●") {
		if lane, cost, ok := e.nodeStat(); ok {
			row += fmt.Sprintf("  · %s · €%.4f", lane, cost)
		}
	}
	return row, keep
}

// nodeStat returns the lane + per-node cost delta since the previous node.
func (e *miniOrkEnricher) nodeStat() (lane string, cost float64, ok bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.runID == "" {
		return "", 0, false
	}
	if !e.dbTried {
		e.dbTried = true
		p := filepath.Join(e.cfg.TargetCWD, ".mini-ork", "state.db")
		if db, ok := seams.OpenAt(p); ok {
			e.db = db
		}
	}
	if e.db == nil {
		return "", 0, false
	}
	var total float64
	_ = e.db.QueryRow(`SELECT COALESCE(SUM(cost_usd),0) FROM llm_calls WHERE run_id=?`, e.runID).Scan(&total)
	_ = e.db.QueryRow(`SELECT COALESCE(model_id,'') FROM llm_calls WHERE run_id=? ORDER BY ts DESC LIMIT 1`, e.runID).Scan(&lane)
	delta := total - e.lastCost
	e.lastCost = total
	if lane == "" && delta <= 0 {
		return "", 0, false // nothing new to attribute
	}
	return lane, delta, true
}

func (e *miniOrkEnricher) close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.db != nil {
		_ = e.db.Close()
	}
}

func writeKickoff(task string) string {
	dir := filepath.Join(os.TempDir(), "coevolve-kickoffs")
	_ = os.MkdirAll(dir, 0o755)
	p := filepath.Join(dir, fmt.Sprintf("kickoff-%d.md", time.Now().UnixNano()))
	_ = os.WriteFile(p, []byte("# Task\n\n"+task+"\n"), 0o644)
	return p
}
