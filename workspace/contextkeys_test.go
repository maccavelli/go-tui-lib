package workspace

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/when"
)

// TestKeysIgnoreReassignment: a program that reassigns the deprecated Key
// variables changes nothing the workspace publishes or lists
// (docs/decisions/0014-PLAN-hardening.md Step 8, finding H11). It uses only
// names that existed before the step, and sits in the package so that the
// deprecated names are not reported by the linter.
func TestKeysIgnoreReassignment(t *testing.T) {
	saved := []any{KeyFocusedPane, KeyZoomed, KeyHiddenPanes, KeyOverlay, KeyModal, KeyWidth, KeyHeight}
	t.Cleanup(func() {
		KeyFocusedPane, KeyZoomed, KeyHiddenPanes = saved[0].(when.Key[string]), saved[1].(when.Key[bool]), saved[2].(when.Key[[]string])
		KeyOverlay, KeyModal = saved[3].(when.Key[string]), saved[4].(when.Key[bool])
		KeyWidth, KeyHeight = saved[5].(when.Key[int]), saved[6].(when.Key[int])
	})
	KeyFocusedPane = when.NewKey[string]("bogus.focusedPane", "")
	KeyZoomed = when.NewKey[bool]("bogus.zoomed", "")
	KeyHiddenPanes = when.NewKey[[]string]("bogus.hiddenPanes", "")
	KeyOverlay = when.NewKey[string]("bogus.overlay", "")
	KeyModal = when.NewKey[bool]("bogus.modal", "")
	KeyWidth = when.NewKey[int]("bogus.width", "")
	KeyHeight = when.NewKey[int]("bogus.height", "")

	w := New(layout.Pane{ID: "a"}, map[layout.PaneID]Pane{"a": &fake{id: "a"}}, WithFocus("a"))
	w.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	c := w.WhenContext()
	keys := ContextKeys()
	for _, name := range []string{"focusedPane", "zoomed", "hiddenPanes", "overlay", "modal", "width", "height"} {
		if _, ok := c.Value("workspace." + name); !ok {
			t.Errorf("WhenContext does not publish workspace.%s", name)
		}
		if _, ok := c.Value("bogus." + name); ok {
			t.Errorf("WhenContext publishes the reassigned bogus.%s", name)
		}
		if _, ok := keys["workspace."+name]; !ok {
			t.Errorf("ContextKeys does not list workspace.%s", name)
		}
		if _, ok := keys["bogus."+name]; ok {
			t.Errorf("ContextKeys lists the reassigned bogus.%s", name)
		}
	}
	if v, _ := c.Value("workspace.width"); v.String() != "40" {
		t.Errorf("workspace.width = %v, want 40", v)
	}
}
