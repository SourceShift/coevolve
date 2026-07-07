package plan

import "testing"

// TestSuggest guards the interactive router: a submitted task must always map
// to a real recipe (never the classifier's `generic` dead-end that emits
// "could not resolve recipe").
func TestSuggest(t *testing.T) {
	cases := map[string]string{
		"create a mini ork to find bugs in the mini ork": "bug-audit-fe-be",
		"add a --version flag to scripts/foo.sh":         "code-fix",
		"audit the architecture for scalability":         "refactor-audit",
		"write up the README for the new module":         "docs",
	}
	for task, want := range cases {
		if got := Suggest(task).Name; got != want {
			t.Errorf("Suggest(%q) = %q, want %q", task, got, want)
		}
	}
	if Suggest("xyzzy nonsense").Name != Default().Name {
		t.Error("unmatched task must fall back to Default, never empty/generic")
	}
}
