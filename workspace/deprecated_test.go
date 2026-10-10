package workspace

import (
	"testing"

	"github.com/maccavelli/go-tui-lib/layout"
)

// TestDeprecatedOptionsStillWork: each old name does what its new name does,
// through v0.9.x (docs/decisions/0014-PLAN-canonicalization.md Step 4). It
// sits in the package, so that staticcheck does not report the deprecated
// names it uses on purpose.
func TestDeprecatedOptionsStillWork(t *testing.T) {
	pane := map[layout.PaneID]Pane{"a": &fake{id: "a"}}
	for _, c := range []struct {
		name string
		o    []Option
		want bool
	}{
		{"WithMouse(false)", []Option{WithMouse(false)}, false},
		{"WithMouse(true) after WithoutMouse", []Option{WithoutMouse(), WithMouse(true)}, true},
		{"WithoutMouse", []Option{WithoutMouse()}, false},
		{"neither", nil, true},
	} {
		if w := New(layout.Pane{ID: "a"}, pane, c.o...); w.mouse != c.want {
			t.Errorf("%s: mouse %v, want %v", c.name, w.mouse, c.want)
		}
	}
}
