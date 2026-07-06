// ContextNest HTTP client — the ONLY way this package talks to CN. CN is a
// separate service (Rust/axum) with its own substrate; we never read its
// SQLite/embeddings store directly, only its REST API. Every call has a
// short timeout and returns (zero-value, false) when CN is unreachable —
// callers render an honest "ContextNest offline" state, never fabricated
// data. Shape reference: mo-fix/lib/cn_client.sh (bash sibling client) and
// ContextNest's src/api/{substrate,field,prompt_context,inbox}.rs.
package seams

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
)

// cnTimeout is intentionally tight — a dashboard refresh must never hang on
// a dead ContextNest instance.
const cnTimeout = 2 * time.Second

var cnHTTPClient = &http.Client{Timeout: cnTimeout}

// CNBaseURL resolves ContextNest's base URL: $CN_BASE_URL, else the
// standard local default.
func CNBaseURL() string {
	if v := os.Getenv("CN_BASE_URL"); v != "" {
		return v
	}
	return "http://127.0.0.1:28080"
}

func cnGet(path string, out any) bool {
	req, err := http.NewRequest(http.MethodGet, CNBaseURL()+path, nil)
	if err != nil {
		return false
	}
	resp, err := cnHTTPClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	if out == nil {
		return true
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return false
	}
	return true
}

func cnGetText(path string) (string, bool) {
	req, err := http.NewRequest(http.MethodGet, CNBaseURL()+path, nil)
	if err != nil {
		return "", false
	}
	resp, err := cnHTTPClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return string(buf), true
}

// CNHealth reports whether ContextNest's substrate is reachable, via
// GET /api/v1/substrate/health. Body is a nested stats object (fragments/
// basins/connections/decay counts), not a bare status field — a 200 with a
// decodable body is treated as healthy.
func CNHealth() bool {
	var health struct {
		Fragments struct {
			Total int `json:"total"`
		} `json:"fragments"`
	}
	return cnGet("/api/v1/substrate/health", &health)
}

// Basin is one topic-cluster from ContextNest's field/basins endpoint.
// Field names match ContextNest's real BasinSummary (src/api/field.rs):
// there is no free-text "representative" field, only a numeric centroid and
// a dominant-kind Label.
type Basin struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Source   string         `json:"source"` // "attractor" | "project"
	Mass     int            `json:"mass"`   // active fragment count
	ByKind   map[string]int `json:"by_kind"`
	Sessions []string       `json:"sessions"`
}

// CNBasins fetches topic-cluster basins via GET /api/v1/field/basins.
// Returns (nil, false) when CN is unreachable or the response can't be
// decoded — never a fabricated list.
func CNBasins() ([]Basin, bool) {
	var resp struct {
		Basins []Basin `json:"basins"`
	}
	if !cnGet("/api/v1/field/basins", &resp) {
		return nil, false
	}
	return resp.Basins, true
}

// CNCapsule fetches the deterministic kind-ordered prompt-context capsule
// (markdown text) via GET /api/v1/prompt-context/capsule. Returns
// (text, false) when CN is unreachable — text is always "" in that case.
func CNCapsule() (string, bool) {
	return cnGetText("/api/v1/prompt-context/capsule")
}

// InboxItem is one attention-inbox entry from ContextNest's inbox endpoint.
// Field names match ContextNest's real InboxHit (src/api/inbox.rs): kind
// and urgency live inside Metadata, not as top-level struct fields.
type InboxItem struct {
	ID         string         `json:"id"`
	SessionID  string         `json:"session_id"`
	Content    string         `json:"content"`
	Importance float32        `json:"importance"`
	Metadata   map[string]any `json:"metadata"`
}

// Kind returns the item's metadata["kind"] as a string, or "" if absent.
func (i InboxItem) Kind() string { return i.metaString("kind") }

// Urgency returns the item's metadata["urgency"] as a string, or "" if
// absent.
func (i InboxItem) Urgency() string { return i.metaString("urgency") }

func (i InboxItem) metaString(key string) string {
	if i.Metadata == nil {
		return ""
	}
	if v, ok := i.Metadata[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// CNInbox fetches attention-inbox items via GET /api/v1/inbox. Returns
// (nil, false) when CN is unreachable.
func CNInbox() ([]InboxItem, bool) {
	var resp struct {
		Items []InboxItem `json:"items"`
	}
	if !cnGet("/api/v1/inbox", &resp) {
		return nil, false
	}
	return resp.Items, true
}
