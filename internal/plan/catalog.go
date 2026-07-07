// Package plan turns a free-text task into a concrete mini-ork run proposal:
// which recipe (and therefore which node topology + lane mix) will run, and a
// sane cost cap. This is what makes Coevolve INTERACTIVE — every task goes
// through mini-ork (written in stone), but the user sees and confirms the plan
// (recipe · topology · max cost) before any spend, and can pick a different
// recipe when the auto-suggestion is wrong.
//
// The catalog is curated from mini-ork's real recipes/ (node lists are the
// actual workflow.yaml node ids, not invented). Installed() intersects it with
// what's actually present in a given mini-ork checkout so we never offer a
// recipe that isn't there.
package plan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Recipe is one runnable mini-ork topology the CLI can propose.
type Recipe struct {
	Name     string   // recipe dir name, e.g. "bug-audit-fe-be" (passed to `mini-ork run <name> <kickoff>`)
	Title    string   // human label for the picker
	Purpose  string   // one-line what-it-does
	Nodes    []string // real topology (workflow.yaml node ids)
	Lanes    string   // human summary of the lane mix
	Keywords []string // task-text triggers for auto-suggestion
	CapUSD   float64  // suggested hard budget cap (MO_DAILY_BUDGET_USD)
}

// Catalog is the curated set surfaced in the interactive picker. Values are
// real: node lists come from each recipe's workflow.yaml.
var Catalog = []Recipe{
	{
		Name:    "code-fix",
		Title:   "Code fix",
		Purpose: "Single-patch change: bug fix, small feature, or one-function refactor — actually edits code.",
		Nodes:   []string{"planner", "implementer", "typecheck", "test", "reviewer", "publisher", "rollback"},
		Lanes:   "code: codex · review: opus",
		Keywords: []string{"fix", "add", "implement", "patch", "change", "bug in", "broken",
			"make it", "wire", "flag", "endpoint", "function"},
		CapUSD: 3,
	},
	{
		Name:    "bug-audit-fe-be",
		Title:   "Bug audit (FE+BE)",
		Purpose: "Find bugs across a feature surface — 2 heterogeneous lenses → ranked bug list. Report-only, no code mutation.",
		Nodes:   []string{"planner", "kimi_lens", "minimax_lens", "synthesizer", "lens_completeness", "publisher", "rollback"},
		Lanes:   "lenses: kimi + minimax · synth: opus",
		Keywords: []string{"find bugs", "bug audit", "audit", "review", "look for bugs",
			"find issues", "problems", "vulnerabilit", "what's wrong"},
		CapUSD: 4,
	},
	{
		Name:    "refactor-audit",
		Title:   "Refactor / architecture audit",
		Purpose: "Multi-lens audit for scalability, security, perf, or architectural-shape concerns → ranked findings.",
		Nodes:   []string{"planner", "glm_lens", "kimi_lens", "codex_lens", "opus_lens", "minimax_lens", "synthesizer", "lens_completeness", "publisher", "rollback"},
		Lanes:   "5 lenses: glm+kimi+codex+opus+minimax",
		Keywords: []string{"refactor", "architecture", "scalability", "bottleneck", "tech debt",
			"performance", "design review", "deep audit", "thorough"},
		CapUSD: 6,
	},
	{
		Name:     "silent-catch-audit",
		Title:    "Silent-catch audit (JS/TS)",
		Purpose:  "Read-only audit that detects silent catch / error-swallowing anti-patterns and ranks swallowed failures.",
		Nodes:    []string{"planner", "structural_lens", "semantic_lens", "adversarial_lens", "findings_reviewer", "audit_shape", "publisher", "rollback"},
		Lanes:    "3 lenses: structural+semantic+adversarial",
		Keywords: []string{"silent catch", "error swallow", "swallowed", "catch anti", "empty catch"},
		CapUSD:   4,
	},
	{
		Name:     "ui-audit",
		Title:    "UI audit",
		Purpose:  "5-lens UI surface audit — a11y, perf, visual consistency, interaction, edge cases → findings.",
		Nodes:    []string{"planner", "a11y_lens", "perf_lens", "visual_lens", "interaction_lens", "edge_lens", "synthesizer", "findings_completeness", "publisher", "rollback"},
		Lanes:    "5 lenses (a11y/perf/visual/interaction/edge)",
		Keywords: []string{"ui audit", "frontend audit", "accessibility", "a11y", "design audit", "visual"},
		CapUSD:   5,
	},
	{
		Name:     "docs",
		Title:    "Docs edit",
		Purpose:  "Documentation-only edit: READMEs, roadmaps, positioning — verified via grep assertions + link integrity.",
		Nodes:    []string{"planner", "doc_editor", "grep_assert", "link_verifier", "publisher"},
		Lanes:    "doc: opus · verify: mechanical",
		Keywords: []string{"docs", "documentation", "readme", "roadmap", "write up", "positioning", "explain in"},
		CapUSD:   2,
	},
	{
		Name:     "research-synthesis",
		Title:    "Research synthesis",
		Purpose:  "Multi-lens research synthesis on a topic/question — web + literature + code-pattern lenses → synthesis.",
		Nodes:    []string{"planner", "glm_lens", "kimi_lens", "codex_lens", "opus_lens", "synthesizer", "source_completeness", "publisher", "rollback"},
		Lanes:    "4 research lenses + synth",
		Keywords: []string{"research", "synthesis", "literature", "compare", "survey", "investigate", "how do others"},
		CapUSD:   5,
	},
}

// Default is the fallback recipe when nothing matches — never leaves the user
// on task_class=generic (which has no recipe and dead-ends the run).
func Default() Recipe { return Catalog[0] } // code-fix

// Suggest scores the catalog against the task text and returns the best match,
// falling back to Default when nothing scores. Never returns a zero Recipe.
func Suggest(task string) Recipe {
	t := strings.ToLower(task)
	best, bestScore := Default(), 0
	for _, r := range Catalog {
		score := 0
		for _, kw := range r.Keywords {
			if strings.Contains(t, kw) {
				// Longer, more specific phrases outweigh single words.
				score += 1 + strings.Count(kw, " ")*2
			}
		}
		if score > bestScore {
			best, bestScore = r, score
		}
	}
	return best
}

// Installed returns the catalog filtered to recipes whose dir exists under
// recipesRoot (…/mini-ork/recipes), so the picker only offers real ones. If the
// root can't be read, the full catalog is returned (best-effort).
func Installed(recipesRoot string) []Recipe {
	ents, err := os.ReadDir(recipesRoot)
	if err != nil {
		return Catalog
	}
	present := map[string]bool{}
	for _, e := range ents {
		if e.IsDir() {
			present[e.Name()] = true
		}
	}
	var out []Recipe
	for _, r := range Catalog {
		if present[r.Name] {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return Catalog
	}
	return out
}

// RecipesRoot resolves the recipes dir for a mini-ork root.
func RecipesRoot(miniOrkRoot string) string {
	return filepath.Join(miniOrkRoot, "recipes")
}

// IndexOf returns the position of recipe name in list, or 0 if absent.
func IndexOf(list []Recipe, name string) int {
	for i, r := range list {
		if r.Name == name {
			return i
		}
	}
	return 0
}

// sortByTitle is used only for deterministic display when needed.
func sortByTitle(rs []Recipe) {
	sort.Slice(rs, func(i, j int) bool { return rs[i].Title < rs[j].Title })
}

var _ = sortByTitle
