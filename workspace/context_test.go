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
	if v, _ := workspace.ModalKey().Get(c); v {
		t.Error("workspace.modal with none open")
	}
}

func TestContextKeys(t *testing.T) {
	ws, _ := rig(t)
	drain(ws, ws.Toggle("metrics"))
	drain(ws, ws.Zoom("session"))
	c := ws.WhenContext()
	if v, ok := workspace.ZoomedKey().Get(c); !ok || !v {
		t.Error("workspace.zoomed is not true")
	}
	if v, ok := workspace.HiddenPanesKey().Get(c); !ok || !slices.Contains(v, "metrics") || !slices.Equal(v, paneStrings(ws.Plan().Hidden)) {
		t.Errorf("workspace.hiddenPanes = %v, want Plan().Hidden %v", v, ws.Plan().Hidden)
	}
	if v, ok := workspace.WidthKey().Get(c); !ok || v != 120 {
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

// TestKeyFunctions: each key function returns the key the workspace
// publishes and ContextKeys lists, by name and kind
// (docs/decisions/0014-PLAN-hardening.md Step 8).
func TestKeyFunctions(t *testing.T) {
	keys := workspace.ContextKeys()
	fns := map[string]when.Kind{
		workspace.FocusedPaneKey().Name: workspace.FocusedPaneKey().Kind(),
		workspace.ZoomedKey().Name:      workspace.ZoomedKey().Kind(),
		workspace.HiddenPanesKey().Name: workspace.HiddenPanesKey().Kind(),
		workspace.OverlayKey().Name:     workspace.OverlayKey().Kind(),
		workspace.ModalKey().Name:       workspace.ModalKey().Kind(),
		workspace.WidthKey().Name:       workspace.WidthKey().Kind(),
		workspace.HeightKey().Name:      workspace.HeightKey().Kind(),
	}
	if len(fns) != 7 || len(keys) != 7 {
		t.Errorf("the functions give %d distinct keys, and ContextKeys %d; want 7 each", len(fns), len(keys))
	}
	for name, kind := range fns {
		if k, ok := keys[name]; !ok || k != kind {
			t.Errorf("%s: ContextKeys has %v, %v; want %v", name, k, ok, kind)
		}
	}
	for _, c := range [][2]string{
		{workspace.FocusedPaneKey().Name, "workspace.focusedPane"}, {workspace.ZoomedKey().Name, "workspace.zoomed"},
		{workspace.HiddenPanesKey().Name, "workspace.hiddenPanes"}, {workspace.OverlayKey().Name, "workspace.overlay"},
		{workspace.ModalKey().Name, "workspace.modal"}, {workspace.WidthKey().Name, "workspace.width"},
		{workspace.HeightKey().Name, "workspace.height"},
	} {
		if c[0] != c[1] {
			t.Errorf("a key function returns %s, want %s", c[0], c[1])
		}
	}
	ws, _ := rig(t)
	c := ws.WhenContext()
	if v, ok := workspace.FocusedPaneKey().Get(c); !ok || v != string(ws.Focused()) {
		t.Errorf("FocusedPaneKey = %q, %v; want %q", v, ok, ws.Focused())
	}
	if v, ok := workspace.HeightKey().Get(c); !ok || v <= 0 {
		t.Errorf("HeightKey = %d, %v", v, ok)
	}
	if v, ok := workspace.OverlayKey().Get(c); !ok || v != "" {
		t.Errorf("OverlayKey with none open = %q, %v", v, ok)
	}
}
