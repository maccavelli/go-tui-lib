package workspace_test

import (
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/when"
	"github.com/maccavelli/go-tui-lib/workspace"
)

// contexter is a focusable pane that publishes when keys.
type contexter struct{ keys when.Map }

func (p *contexter) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return p, nil }
func (p *contexter) View(int, int) string                     { return "" }
func (p *contexter) WhenContext() when.Context                { return p.keys }

func TestWhenContextLayers(t *testing.T) {
	pane := &contexter{keys: when.Map{
		"k": when.StringValue("pane"), "p": when.StringValue("pane"), "workspace.width": when.NumberValue(1),
	}}
	ws := workspace.New(layout.Split{Axis: layout.Horizontal, Children: []layout.Child{
		{Node: layout.Pane{ID: "editor"}}, {Node: layout.Pane{ID: "other"}},
	}}, map[layout.PaneID]workspace.Pane{"editor": pane, "other": &transcript{}}, workspace.WithFocus("editor"))
	ws.Update(tea.WindowSizeMsg{Width: 60, Height: 10})
	over := &contexter{keys: when.Map{"k": when.StringValue("overlay")}}
	drain(ws, ws.Push(workspace.Overlay{ID: "picker", Pane: over, Width: 20, Height: 5, Modal: true}))

	c := ws.WhenContext()
	for k, want := range map[string]string{
		"k": "overlay", "p": "pane", "workspace.width": "1",
		"workspace.focusedPane": "editor", "workspace.overlay": "picker", "workspace.modal": "true",
		"workspace.zoomed": "false", "workspace.height": "10",
	} {
		if v, ok := c.Value(k); !ok || v.String() != want {
			t.Errorf("%s = %v, %v; want %s", k, v, ok, want)
		}
	}
	drain(ws, ws.Pop())
	c = ws.WhenContext()
	if v, _ := c.Value("k"); v.String() != "pane" {
		t.Errorf("k after the overlay closed = %v, want the pane's", v)
	}
	if v, _ := c.Value("workspace.overlay"); v.String() != "" {
		t.Errorf("workspace.overlay with none open = %q", v.String())
	}
	if v, _ := workspace.KeyModal.Get(c); v {
		t.Error("workspace.modal with none open")
	}
}

func TestContextKeys(t *testing.T) {
	ws, _ := rig(t)
	drain(ws, ws.Toggle("metrics"))
	drain(ws, ws.Zoom("session"))
	c := ws.WhenContext()
	if v, ok := workspace.KeyZoomed.Get(c); !ok || !v {
		t.Error("workspace.zoomed is not true")
	}
	if v, ok := workspace.KeyHiddenPanes.Get(c); !ok || !slices.Contains(v, "metrics") || !slices.Equal(v, paneStrings(ws.Plan().Hidden)) {
		t.Errorf("workspace.hiddenPanes = %v, want Plan().Hidden %v", v, ws.Plan().Hidden)
	}
	if v, ok := workspace.KeyWidth.Get(c); !ok || v != 120 {
		t.Errorf("workspace.width = %d", v)
	}
	keys := workspace.ContextKeys()
	if len(keys) != 7 {
		t.Errorf("ContextKeys has %d keys, want 7", len(keys))
	}
	if errs := when.Check(when.MustParse("workspace.width >= 80 && metrics in workspace.hiddenPanes && !workspace.modal"), keys); len(errs) != 1 {
		// metrics is not a key; everything else fits.
		t.Errorf("Check: %v", errs)
	}
	if !when.MustParse("session == workspace.focusedPane || workspace.zoomed").Eval(c) {
		t.Error("a when over the workspace's keys is false")
	}
}

func paneStrings(ids []layout.PaneID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}
