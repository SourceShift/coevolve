package modes

import (
	"strings"
	"testing"

	"github.com/sourceshift/coevolve/internal/tui"
)

func TestCostRealData(t *testing.T) {
	var cm tui.Mode
	for _, m := range tui.Modes() {
		if m.Meta().Key == "Cost" {
			cm = m
		}
	}
	if cm == nil {
		t.Fatal("Cost mode not registered")
	}
	cm.Update(tui.RefreshMsg{})
	out := cm.View(96, 24)
	t.Log("\n" + out)
	if !strings.Contains(out, "SPEND BY PROVIDER") {
		t.Fatal("missing spend table")
	}
}
