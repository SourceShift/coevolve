// Package run: ServeWorker drives an isolated `opencode serve` instance over
// its HTTP + SSE surface. It is the plain-text (non "/run") path from Start().
package run

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type serveWorker struct {
	cfg        Config
	prompt     string
	ctx        context.Context
	cancel     context.CancelFunc
	cmd        *exec.Cmd
	baseURL    string
	httpClient *http.Client
	sessionID  string
	tmpDir     string
	out        chan Line
	stderrBuf  *ringBuffer
	failed     bool
	seen       map[string]struct{} // dedupe repeated part updates
}

type ringBuffer struct {
	mu  sync.Mutex
	buf []byte
	max int
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf = append(r.buf, p...)
	if len(r.buf) > r.max {
		r.buf = r.buf[len(r.buf)-r.max:]
	}
	return len(p), nil
}

func (r *ringBuffer) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return string(r.buf)
}

func pickFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func mkTempXDG() (string, error) {
	return os.MkdirTemp("", "coevolve-oc-*")
}

func (w *serveWorker) spawnServe(ctx context.Context) error {
	port, err := pickFreePort()
	if err != nil {
		return err
	}
	w.baseURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	w.cmd = exec.CommandContext(ctx, "opencode", "serve",
		"--hostname", "127.0.0.1",
		"--port", strconv.Itoa(port),
		"--pure")
	w.cmd.Dir = w.cfg.TargetCWD
	// Single-user CLI: inherit the user's authenticated opencode config
	// (~/.local/share/opencode/auth.json) — do NOT isolate XDG, which would
	// hide the credentials and cause "Authentication error". Env keys
	// (OPENAI/CLOUDFLARE/etc.) also pass through via os.Environ().
	w.cmd.Env = os.Environ()
	w.stderrBuf = &ringBuffer{max: 8192}
	w.cmd.Stderr = w.stderrBuf
	return w.cmd.Start()
}

func (w *serveWorker) waitReady(ctx context.Context, deadline time.Duration) error {
	start := time.Now()
	dctx, dcancel := context.WithTimeout(ctx, deadline)
	defer dcancel()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		req, _ := http.NewRequestWithContext(dctx, "GET", w.baseURL+"/session", nil)
		resp, err := w.httpClient.Do(req)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
		select {
		case <-dctx.Done():
			return fmt.Errorf("elapsed %v: %v; stderr: %s",
				time.Since(start), dctx.Err(), w.stderrBuf.String())
		case <-ticker.C:
		}
	}
}

func (w *serveWorker) killServe() {
	if w.cmd != nil && w.cmd.Process != nil {
		_ = w.cmd.Process.Kill()
		_ = w.cmd.Wait()
	}
	if w.tmpDir != "" {
		_ = os.RemoveAll(w.tmpDir)
	}
}

func (w *serveWorker) postJSON(path string, body any) ([]byte, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(w.ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", w.baseURL+path, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("POST %s: %s: %s", path, resp.Status, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}

func (w *serveWorker) createSession() error {
	body, err := w.postJSON("/session", map[string]any{})
	if err != nil {
		return err
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("decode /session: %w", err)
	}
	if out.ID == "" {
		return fmt.Errorf("empty session id from /session")
	}
	w.sessionID = out.ID
	return nil
}

func (w *serveWorker) setModel(providerModel string) error {
	parts := strings.SplitN(providerModel, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("worker model %q not in provider/model form", providerModel)
	}
	_, err := w.postJSON("/session/"+w.sessionID+"/model",
		map[string]any{"providerID": parts[0], "modelID": parts[1]})
	return err
}

func (w *serveWorker) sendPrompt(text string) error {
	// opencode pins the model PER PROMPT (setModel alone doesn't stick — it
	// falls back to the config default). Include providerID/modelID here.
	body := map[string]any{
		"parts": []map[string]any{{"type": "text", "text": text}},
	}
	if parts := strings.SplitN(w.cfg.WorkerModel, "/", 2); len(parts) == 2 {
		body["model"] = map[string]any{"providerID": parts[0], "modelID": parts[1]}
	}
	_, err := w.postJSON("/session/"+w.sessionID+"/prompt_async", body)
	return err
}

func (w *serveWorker) replyPermission(reqID, response string) error {
	if reqID == "" {
		return fmt.Errorf("empty permission request id")
	}
	_, err := w.postJSON("/session/"+w.sessionID+"/permission/"+reqID+"/reply",
		map[string]any{"response": response})
	return err
}

func (w *serveWorker) abort() {
	if w.sessionID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", w.baseURL+"/session/"+w.sessionID+"/abort", nil)
	if err != nil {
		return
	}
	resp, err := w.httpClient.Do(req)
	if err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func (w *serveWorker) streamEvents(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", w.baseURL+"/event", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	sseClient := &http.Client{}
	resp, err := sseClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		var ev struct {
			Type       string          `json:"type"`
			Properties json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case "message.part.updated":
			w.handlePartUpdated(ev.Properties)
		case "session.idle":
			return nil
		case "session.error":
			w.failed = true
			w.emit(Line{Text: "■ session error: " + extractError(ev.Properties), Err: true})
			return nil
		case "permission.asked":
			var p struct {
				ID string `json:"id"`
			}
			_ = json.Unmarshal(ev.Properties, &p)
			if p.ID == "" {
				w.emit(Line{Text: "■ permission request missing id — rejecting", Err: true})
				continue
			}
			go func(id string) { _ = w.replyPermission(id, "once") }(p.ID)
		}
	}
	return sc.Err()
}

func (w *serveWorker) handlePartUpdated(raw json.RawMessage) {
	var upd struct {
		Part struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			Tool  string `json:"tool"`
			State struct {
				Status string          `json:"status"`
				Input  json.RawMessage `json:"input"`
				Output string          `json:"output"`
			} `json:"state"`
			Tokens struct {
				Input  int `json:"input"`
				Output int `json:"output"`
				Cache  int `json:"cache"`
			} `json:"tokens"`
			Cost float64 `json:"cost"`
		} `json:"part"`
	}
	if err := json.Unmarshal(raw, &upd); err != nil {
		return
	}
	p := upd.Part
	switch p.Type {
	case "text":
		t := strings.TrimSpace(p.Text)
		if t == "" || t == strings.TrimSpace(w.prompt) {
			return // skip empty + the echoed user prompt
		}
		w.emit(Line{Text: stripANSI(p.Text)})
	case "tool":
		switch p.State.Status {
		case "running":
			if w.once("run:" + p.Tool + string(p.State.Input)) {
				w.emit(Line{Text: fmt.Sprintf("● %s(%s)", p.Tool, inputSummary(p.State.Input))})
			}
		case "completed":
			if first := firstNonEmptyLine(p.State.Output); first != "" && w.once("done:"+p.Tool+string(p.State.Input)) {
				w.emit(Line{Text: "  ⎿ " + first})
			}
		}
	case "step-finish":
		w.emit(Line{Text: fmt.Sprintf("✓ %d→%d tok · $%.4f",
			p.Tokens.Input, p.Tokens.Output, p.Cost)})
	}
}

func inputSummary(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	s := strings.ReplaceAll(string(raw), "\n", " ")
	if len(s) > 80 {
		s = s[:77] + "..."
	}
	return s
}

// extractError digs the human message out of opencode's session.error props,
// which nest it under a few shapes. Falls back to the raw JSON so the real
// cause is never silently empty.
func extractError(raw json.RawMessage) string {
	var e struct {
		Error struct {
			Name string `json:"name"`
			Data struct {
				Message string `json:"message"`
			} `json:"data"`
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &e) == nil {
		switch {
		case e.Error.Data.Message != "":
			return e.Error.Data.Message
		case e.Error.Message != "":
			return e.Error.Message
		case e.Error.Name != "":
			return e.Error.Name
		case e.Message != "":
			return e.Message
		}
	}
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return "(no detail)"
	}
	if len(s) > 300 {
		s = s[:297] + "..."
	}
	return s
}

func firstNonEmptyLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			return t
		}
	}
	return ""
}

// once returns true the first time it sees key (dedupes repeated part updates).
func (w *serveWorker) once(key string) bool {
	if w.seen == nil {
		w.seen = map[string]struct{}{}
	}
	if _, ok := w.seen[key]; ok {
		return false
	}
	w.seen[key] = struct{}{}
	return true
}

func (w *serveWorker) emit(l Line) {
	select {
	case w.out <- l:
	case <-w.ctx.Done():
	}
}

// StartServe launches an isolated `opencode serve`, drives it with `prompt`,
// and returns a Handle whose Lines channel closes when the run ends.
func StartServe(ctx context.Context, cfg Config, prompt string) *Handle {
	runCtx, cancel := context.WithCancel(ctx)
	out := make(chan Line, 64)
	w := &serveWorker{
		cfg:        cfg,
		prompt:     prompt,
		ctx:        runCtx,
		cancel:     cancel,
		out:        out,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	go func() {
		defer close(out)
		defer w.killServe()

		out <- Line{Text: fmt.Sprintf("› %s  (main LLM: %s)", prompt, cfg.WorkerModel)}

		tmp, err := mkTempXDG()
		if err != nil {
			out <- Line{Text: "failed to create temp dir: " + err.Error(), Err: true}
			return
		}
		w.tmpDir = tmp

		if err := w.spawnServe(runCtx); err != nil {
			out <- Line{Text: "failed to spawn opencode serve: " + err.Error(), Err: true}
			return
		}

		readyStart := time.Now()
		if err := w.waitReady(runCtx, 20*time.Second); err != nil {
			out <- Line{Text: fmt.Sprintf("opencode serve not ready (elapsed %v): %v",
				time.Since(readyStart), err), Err: true}
			return
		}

		if err := w.createSession(); err != nil {
			out <- Line{Text: "failed to create session: " + err.Error(), Err: true}
			return
		}
		if err := w.setModel(cfg.WorkerModel); err != nil {
			out <- Line{Text: "failed to set model: " + err.Error(), Err: true}
			return
		}

		streamDone := make(chan struct{})
		go func() {
			defer close(streamDone)
			if err := w.streamEvents(runCtx); err != nil && runCtx.Err() == nil {
				w.emit(Line{Text: "SSE stream error: " + err.Error(), Err: true})
			}
		}()

		if err := w.sendPrompt(prompt); err != nil {
			out <- Line{Text: "failed to send prompt: " + err.Error(), Err: true}
			w.abort()
			<-streamDone
			return
		}

		select {
		case <-streamDone:
			if runCtx.Err() != nil {
				out <- Line{Text: "■ stopped", Err: true}
				return
			}
			if w.failed {
				return // session.error already emitted; don't claim success
			}
			out <- Line{Text: "✓ done"}
		case <-runCtx.Done():
			w.abort()
			<-streamDone
			out <- Line{Text: "■ stopped", Err: true}
		}
	}()

	return &Handle{Lines: out, cancel: cancel}
}
