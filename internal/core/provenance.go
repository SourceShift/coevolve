// Package core holds Coevolve's framework-agnostic contracts: the provenance
// ("all real data") layer, the normalized event model, and the pluggable
// integration interfaces + registry. The Bubble Tea TUI and every seam program
// against these, so backends and modes are additive.
package core

import "fmt"

// Origin records where a displayed value came from. The renderers refuse to
// show a bare number for anything that is not REAL — the "all real data" rule
// enforced at the type level (ported from the Python prototype).
type Origin int

const (
	Real      Origin = iota // measured (DB, opencode usage, CN hit, verify verdict)
	Estimated               // computed from real inputs (tokens × list price)
	NotYet                  // needs a capability we don't have (trained model)
	Degraded                // seam reachable but empty/offline right now
)

func (o Origin) String() string {
	switch o {
	case Real:
		return "real"
	case Estimated:
		return "estimated"
	case NotYet:
		return "not_yet"
	case Degraded:
		return "degraded"
	default:
		return "unknown"
	}
}

// Provenance is where a value came from.
type Provenance struct {
	Origin Origin
	Source string // e.g. "llm_calls.cost_usd", "cn_capsule", "list-price"
}

func (p Provenance) IsReal() bool { return p.Origin == Real }

func RealP(source string) Provenance      { return Provenance{Real, source} }
func EstimatedP(source string) Provenance { return Provenance{Estimated, source} }
func NotYetP(source string) Provenance    { return Provenance{NotYet, source} }
func DegradedP(source string) Provenance  { return Provenance{Degraded, source} }

// Value is a number/string carrying its provenance. Renderers gate on Prov:
// only Real values render as a figure; others render as a marker.
type Value struct {
	V    any
	Prov Provenance
}

func RealVal(v any, source string) Value { return Value{v, RealP(source)} }
func NotYetVal(source string) Value      { return Value{nil, NotYetP(source)} }

// Render returns the display string, honouring the honesty rule: non-real
// values never show a fabricated figure.
func (val Value) Render() string {
	switch val.Prov.Origin {
	case Real:
		return fmt.Sprintf("%v", val.V)
	case Estimated:
		return fmt.Sprintf("%v (est.)", val.V)
	case Degraded:
		return "—"
	default: // NotYet
		return "not available yet"
	}
}
