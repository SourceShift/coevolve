// Package run drives the actual work: it sends a prompt to the configured main
// LLM (opencode) and streams its output, or — for /run — spawns a full mini-ork
// orchestration. This is what makes Coevolve's command section behave like
// Claude/opencode: you type, it works, the flow streams back.
package run

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
func Start(cfg Config, input string) *Handle {
	trimmed := strings.TrimSpace(input)
	if trimmed != "" && !strings.HasPrefix(trimmed, "/run ") {
		return StartServe(context.Background(), cfg, trimmed)
	}
	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan Line, 64)
	cmd, banner := build(ctx, cfg, trimmed)

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
		pump := func(r io.Reader, isErr bool) {
			sc := bufio.NewScanner(r)
			sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for sc.Scan() {
				txt := stripANSI(sc.Text())
				if strings.TrimSpace(txt) == "" {
					continue
				}
				row, keep := miniOrkRow(txt) // render the loop inline (design style)
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

// build chooses the command + a human banner for the input.
func build(ctx context.Context, cfg Config, input string) (*exec.Cmd, string) {
	if task, ok := strings.CutPrefix(input, "/run "); ok {
		kick := writeKickoff(task)
		c := exec.CommandContext(ctx, filepath.Join(cfg.MiniOrkRoot, "bin", "mini-ork"), "run", kick)
		c.Dir = cfg.TargetCWD
		dry := "1"
		if cfg.Live {
			dry = "0"
		}
		c.Env = append(os.Environ(),
			"MINI_ORK_ROOT="+cfg.MiniOrkRoot,
			"MO_TARGET_CWD="+cfg.TargetCWD,
			"MO_OPENCODE_MODEL="+cfg.WorkerModel,
			"MO_IMPLEMENTER_LANE=opencode",
			"MINI_ORK_DRY_RUN="+dry,
		)
		mode := "dry-run (set COEVOLVE_LIVE=1 to spend)"
		if cfg.Live {
			mode = "LIVE"
		}
		return c, fmt.Sprintf("› mini-ork run · %s · worker=%s · %s", task, cfg.WorkerModel, mode)
	}
	// default: the configured main LLM via opencode, streaming its work
	c := exec.CommandContext(ctx, "opencode", "run", "-m", cfg.WorkerModel,
		"--pure", "--dangerously-skip-permissions", input)
	c.Dir = cfg.TargetCWD
	return c, fmt.Sprintf("› %s  (main LLM: %s)", input, cfg.WorkerModel)
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

func writeKickoff(task string) string {
	dir := filepath.Join(os.TempDir(), "coevolve-kickoffs")
	_ = os.MkdirAll(dir, 0o755)
	p := filepath.Join(dir, fmt.Sprintf("kickoff-%d.md", time.Now().UnixNano()))
	_ = os.WriteFile(p, []byte("# Task\n\n"+task+"\n"), 0o644)
	return p
}
