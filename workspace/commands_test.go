package workspace_test

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/tuitest"
	"github.com/maccavelli/go-tui-lib/workspace"
)

// drain runs cmd and gives the workspace every message it yields, as a
// program would, a few rounds deep. The registry's own messages are not
// the workspace's, and are skipped.
func drain(ws *workspace.Workspace, cmd tea.Cmd) {
	queue := []tea.Cmd{cmd}
	for round := 0; round < 4 && len(queue) > 0; round++ {
		var next []tea.Cmd
		for _, c := range queue {
			if c == nil {
				continue
			}
			switch m := c().(type) {
			case tea.BatchMsg:
				next = append(next, m...)
			case command.ResultMsg, command.PromptMsg, nil:
			default:
				next = append(next, ws.Update(m))
			}
		}
		queue = next
	}
}

// rig is the agent session at 120 by 30 with its commands registered.
func rig(t *testing.T, o ...workspace.CommandOption) (*workspace.Workspace, *command.Registry) {
	t.Helper()
	th := theme.New(colorprofile.NoTTY, theme.Unknown, glyph.ASCII())
	ws := session(agentRoot(false, layout.FullWidth), th, 120, 30)
	r := command.NewRegistry()
	if err := r.Register(workspace.Commands(ws, o...)...); err != nil {
		t.Fatal(err)
	}
	return ws, r
}

// dispatch runs id with args through r from a key, and drains the result
// into ws. It returns the ResultMsg's error.
func dispatch(t *testing.T, ws *workspace.Workspace, r *command.Registry, id command.ID, args string) error {
	t.Helper()
	cmd := r.Dispatch(context.Background(), command.Request{ID: id, Args: []byte(args), Origin: command.OriginKey, WhenContext: ws.WhenContext()})
	var err error
	var effects []tea.Cmd
	for _, m := range flatten(cmd) {
		switch m := m.(type) {
		case command.ResultMsg:
			err = m.Err
			effects = append(effects, m.Result.Cmd)
		}
	}
	for _, e := range effects {
		drain(ws, e)
	}
	return err
}

// flatten runs cmd and every command a BatchMsg holds, one level of
// registry batching, and returns the messages that are not batches.
func flatten(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	m := cmd()
	b, ok := m.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{m}
	}
	var out []tea.Msg
	for _, c := range b {
		if c == nil {
			continue
		}
		// Only the registry's messages are kept here; the command's own
		// effect comes back in ResultMsg.Result.Cmd, which dispatch drains.
		if msg := c(); msg != nil {
			switch msg.(type) {
			case command.ResultMsg, command.PromptMsg:
				out = append(out, msg)
			}
		}
	}
	return out
}

func layouts() map[string]layout.Node {
	return map[string]layout.Node{"right": agentRoot(false, layout.FullWidth), "left": agentRoot(true, layout.FullWidth)}
}

func TestBuiltinsMatchMethods(t *testing.T) {
	overlay := workspace.Overlay{ID: "help", Pane: &footer{text: "help"}, Width: 20, Height: 5, Modal: true}
	for _, c := range []struct {
		name   string
		setup  func(ws *workspace.Workspace)
		id     command.ID
		args   string
		method func(ws *workspace.Workspace) tea.Cmd
	}{
		{"focus", nil, "workspace.focus", `{"pane":"logs"}`, func(ws *workspace.Workspace) tea.Cmd { return ws.Focus("logs") }},
		{"next", nil, "workspace.focus.next", "", (*workspace.Workspace).FocusNext},
		{"prev", nil, "workspace.focus.prev", "", (*workspace.Workspace).FocusPrev},
		{"zoom", nil, "workspace.zoom", `{"pane":"logs"}`, func(ws *workspace.Workspace) tea.Cmd { return ws.Zoom("logs") }},
		{"zoom focused", nil, "workspace.zoom", "", func(ws *workspace.Workspace) tea.Cmd { return ws.Zoom(ws.Focused()) }},
		{"unzoom", func(ws *workspace.Workspace) { drain(ws, ws.Zoom("logs")) }, "workspace.zoom", `{"pane":"logs"}`,
			func(ws *workspace.Workspace) tea.Cmd { return ws.Zoom("logs") }},
		{"toggle", nil, "workspace.toggle", `{"pane":"metrics"}`, func(ws *workspace.Workspace) tea.Cmd { return ws.Toggle("metrics") }},
		{"resize", nil, "workspace.resize", `{"split":"sidebar","delta":5}`, func(ws *workspace.Workspace) tea.Cmd { return ws.Resize("sidebar:0", 5) }},
		{"layout", nil, "workspace.layout.use", `{"name":"left"}`, func(ws *workspace.Workspace) tea.Cmd { return ws.SetLayout(layouts()["left"]) }},
		{"reset", func(ws *workspace.Workspace) { drain(ws, ws.Zoom("logs")) }, "workspace.layout.reset", "",
			func(ws *workspace.Workspace) tea.Cmd { return ws.SetState(layout.State{}) }},
		{"state", nil, "workspace.state.set", `{"state":{"version":1,"zoom":"metrics"}}`,
			func(ws *workspace.Workspace) tea.Cmd { return ws.SetState(layout.State{Version: 1, Zoom: "metrics"}) }},
		{"close", func(ws *workspace.Workspace) { drain(ws, ws.Push(overlay)) }, "workspace.overlay.close", "", (*workspace.Workspace).Pop},
		{"theme", nil, "workspace.theme.set", `{"background":"light"}`, func(ws *workspace.Workspace) tea.Cmd { return ws.SetBackground(theme.Light) }},
	} {
		byMethod, _ := rig(t, workspace.WithLayouts(layouts()))
		byCommand, r := rig(t, workspace.WithLayouts(layouts()))
		if c.setup != nil {
			c.setup(byMethod)
			c.setup(byCommand)
		}
		drain(byMethod, c.method(byMethod))
		if err := dispatch(t, byCommand, r, c.id, c.args); err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if byMethod.Render() != byCommand.Render() {
			t.Errorf("%s: the frames differ:\n%s\n---\n%s", c.name, byMethod.Render(), byCommand.Render())
		}
		if !statesEqual(byMethod.State(), byCommand.State()) || byMethod.Focused() != byCommand.Focused() {
			t.Errorf("%s: state %+v focus %s, want %+v focus %s", c.name, byCommand.State(), byCommand.Focused(), byMethod.State(), byMethod.Focused())
		}
	}
}

func statesEqual(a, b layout.State) bool {
	ja, err1 := json.Marshal(a, json.Deterministic(true))
	jb, err2 := json.Marshal(b, json.Deterministic(true))
	return err1 == nil && err2 == nil && string(ja) == string(jb)
}

// panes is workspace.panes's value, read back through JSON.
type paneRow struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	X       int    `json:"x"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Focused bool   `json:"focused"`
	Hidden  bool   `json:"hidden"`
}

func panesOf(t *testing.T, r *command.Registry) []paneRow {
	t.Helper()
	res, err := r.Run(context.Background(), command.Request{ID: "workspace.panes", Origin: command.OriginAgent})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(res.Value)
	if err != nil {
		t.Fatal(err)
	}
	var rows []paneRow
	if err := json.Unmarshal(b, &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestPanesCommand(t *testing.T) {
	ws, r := rig(t)
	if err := dispatch(t, ws, r, "workspace.toggle", `{"pane":"metrics"}`); err != nil {
		t.Fatal(err)
	}
	rows := panesOf(t, r)
	var ids []string
	for _, p := range rows {
		ids = append(ids, p.ID)
	}
	// The ring (session, logs), the footer, which takes no focus, then the
	// hidden metrics (A8).
	if want := []string{"session", "logs", "footer", "metrics"}; !slices.Equal(ids, want) {
		t.Fatalf("panes = %v, want %v", ids, want)
	}
	plan := ws.Plan()
	for _, p := range rows {
		r, placed := plan.Panes[layout.PaneID(p.ID)]
		if p.Hidden == placed || p.Focused != (layout.PaneID(p.ID) == ws.Focused()) || p.X != r.X || p.Width != r.W || p.Height != r.H {
			t.Errorf("%s: %+v; plan %+v placed %v, focused %s", p.ID, p, r, placed, ws.Focused())
		}
		if p.Title == "" {
			t.Errorf("%s has no title", p.ID)
		}
	}

	// The ring's order, not the tree's, comes first.
	th := theme.New(colorprofile.NoTTY, theme.Unknown, glyph.ASCII())
	ringed := session(agentRoot(false, layout.FullWidth), th, 120, 30, workspace.WithFocusRing("logs", "metrics", "session"))
	rr := command.NewRegistry()
	if err := rr.Register(workspace.Commands(ringed)...); err != nil {
		t.Fatal(err)
	}
	ids = ids[:0]
	for _, p := range panesOf(t, rr) {
		ids = append(ids, p.ID)
	}
	if want := []string{"logs", "metrics", "session", "footer"}; !slices.Equal(ids, want) {
		t.Errorf("with a focus ring: panes = %v, want %v", ids, want)
	}
}

func TestResizeSplit(t *testing.T) {
	ws, r := rig(t)
	before := ws.Plan().Panes["session"]
	if err := dispatch(t, ws, r, "workspace.resize", `{"split":"sidebar","delta":4}`); err != nil {
		t.Fatal(err)
	}
	if after := ws.Plan().Panes["session"]; after.W != before.W+4 {
		t.Errorf("session is %d wide after resize, want %d", after.W, before.W+4)
	}
	if err := dispatch(t, ws, r, "workspace.resize", `{"split":"sidebar:0","delta":-4}`); err != nil {
		t.Fatal(err)
	}
	if after := ws.Plan().Panes["session"]; after.W != before.W {
		t.Errorf("session is %d wide after resizing sidebar:0 back, want %d", after.W, before.W)
	}
	// A named split of three panes has two separators, row:0 and row:1, and
	// an unnamed split in it one positional separator.
	var positional string
	free := workspace.New(layout.Split{Name: "row", Axis: layout.Horizontal, Children: []layout.Child{
		{Node: layout.Pane{ID: "a"}}, {Node: layout.Pane{ID: "b"}},
		{Node: layout.Split{Axis: layout.Vertical, Children: []layout.Child{{Node: layout.Pane{ID: "c"}}, {Node: layout.Pane{ID: "d"}}}}},
	}}, map[layout.PaneID]workspace.Pane{"a": &footer{}, "b": &footer{}, "c": &footer{}, "d": &footer{}})
	free.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	for _, s := range free.Plan().Separators {
		if !s.Resizable {
			positional = s.ID
		}
	}
	if positional == "" {
		t.Fatal("no positional separator to try")
	}
	fr := command.NewRegistry()
	if err := fr.Register(workspace.Commands(free)...); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		ws    *workspace.Workspace
		r     *command.Registry
		split string
	}{{ws, r, "session"}, {ws, r, "nothing"}, {free, fr, positional}, {free, fr, "row"}} {
		err := dispatch(t, c.ws, c.r, "workspace.resize", `{"split":"`+c.split+`","delta":1}`)
		if ae, ok := errors.AsType[*command.ArgError](err); !ok || ae.Path != "/split" {
			t.Errorf("split %q: %v, want an ArgError at /split", c.split, err)
		}
	}
}

func TestUnknownPanes(t *testing.T) {
	ws, r := rig(t)
	for _, id := range []command.ID{"workspace.focus", "workspace.zoom", "workspace.toggle"} {
		err := dispatch(t, ws, r, id, `{"pane":"typo"}`)
		if ae, ok := errors.AsType[*command.ArgError](err); !ok || ae.Path != "/pane" {
			t.Errorf("%s of an unknown pane: %v, want an ArgError at /pane", id, err)
		}
	}
	if len(ws.State().Hidden) != 0 || ws.State().Zoom != "" {
		t.Errorf("an unknown pane changed the state: %+v", ws.State())
	}
	if err := dispatch(t, ws, r, "workspace.focus", `{"pane":"footer"}`); err == nil {
		t.Error("the footer, which takes no focus, was focused")
	}
	drain(ws, ws.Toggle("metrics"))
	if err := dispatch(t, ws, r, "workspace.toggle", `{"pane":"metrics"}`); err != nil {
		t.Errorf("toggling a hidden pane back: %v", err)
	}
}

func TestLayoutUseEnum(t *testing.T) {
	_, r := rig(t, workspace.WithLayouts(layouts()))
	c, ok := r.Lookup("workspace.layout.use")
	if !ok {
		t.Fatal("no workspace.layout.use")
	}
	var schema struct {
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(c.Args, &schema); err != nil {
		t.Fatal(err)
	}
	if got := schema.Properties["name"].Enum; !slices.Equal(got, []string{"left", "right"}) {
		t.Errorf("enum = %v, want [left right]", got)
	}
	ws, r2 := rig(t, workspace.WithLayouts(layouts()))
	if err := dispatch(t, ws, r2, "workspace.layout.use", `{"name":"top"}`); err == nil {
		t.Error("a layout outside the enum was used")
	}
	_, plain := rig(t)
	if _, ok := plain.Lookup("workspace.layout.use"); ok {
		t.Error("workspace.layout.use is registered without WithLayouts")
	}
}

func TestLayoutReset(t *testing.T) {
	ws, r := rig(t)
	fresh, _ := rig(t)
	for _, c := range [][2]string{
		{"workspace.resize", `{"split":"sidebar","delta":3}`},
		{"workspace.toggle", `{"pane":"logs"}`},
		{"workspace.zoom", `{"pane":"session"}`},
	} {
		if err := dispatch(t, ws, r, command.ID(c[0]), c[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := dispatch(t, ws, r, "workspace.layout.reset", ""); err != nil {
		t.Fatal(err)
	}
	if !statesEqual(ws.State(), layout.State{}) || ws.Render() != fresh.Render() {
		t.Errorf("after reset: %+v\n%s", ws.State(), ws.Render())
	}
}

func TestStateSetRefusesBadState(t *testing.T) {
	ws, r := rig(t)
	before := ws.Render()
	err := dispatch(t, ws, r, "workspace.state.set", `{"state":{"version":99}}`)
	if ae, ok := errors.AsType[*command.ArgError](err); !ok || ae.Path != "/state" {
		t.Errorf("a state of an unknown version: %v, want an ArgError at /state", err)
	}
	if ws.Render() != before || ws.Err() != nil {
		t.Errorf("the refused state stayed: %v", ws.Err())
	}
	res, err := r.Run(context.Background(), command.Request{ID: "workspace.overlay.close", Origin: command.OriginKey})
	if err != nil || res.Text == "" {
		t.Errorf("closing with no overlay: %+v, %v", res, err)
	}
}

func TestAgentMayZoom(t *testing.T) {
	ws, r := rig(t)
	if res := r.CallMCP(context.Background(), "workspace.zoom", []byte(`{"pane":"logs"}`), "agent"); res.IsError {
		t.Fatalf("an agent's zoom: %+v", res)
	}
	if ws.State().Zoom != "logs" {
		t.Errorf("zoom = %q, want logs", ws.State().Zoom)
	}

	// Through WithLoop, as a real program runs it: the agent's goroutine
	// waits, and the loop runs the command (A7).
	th := theme.New(colorprofile.NoTTY, theme.Unknown, glyph.ASCII())
	ws2 := session(agentRoot(false, layout.FullWidth), th, 120, 30)
	sent := make(chan tea.Msg, 1)
	lr := command.NewRegistry(command.WithLoop(func(m tea.Msg) { sent <- m }))
	if err := lr.Register(workspace.Commands(ws2)...); err != nil {
		t.Fatal(err)
	}
	done := make(chan command.MCPCallResult, 1)
	go func() {
		done <- lr.CallMCP(context.Background(), "workspace.zoom", []byte(`{"pane":"logs"}`), "agent")
	}()
	lm, ok := (<-sent).(command.LoopMsg)
	if !ok {
		t.Fatal("CallMCP did not send a LoopMsg")
	}
	if ws2.State().Zoom != "" {
		t.Fatal("the workspace changed before the loop ran the command")
	}
	drain(ws2, lm.Run())
	if res := <-done; res.IsError || ws2.State().Zoom != "logs" {
		t.Errorf("through the loop: %+v, zoom %q", res, ws2.State().Zoom)
	}
}

func TestKeysStillWork(t *testing.T) {
	ws, _ := rig(t)
	was := ws.Focused()
	drain(ws, ws.Update(tea.KeyPressMsg{Code: '.', Mod: tea.ModAlt}))
	if ws.Focused() == was {
		t.Errorf("alt+. did not move focus from %s", was)
	}
	drain(ws, ws.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModAlt}))
	if ws.State().Zoom != ws.Focused() {
		t.Errorf("alt+z did not zoom %s: %+v", ws.Focused(), ws.State())
	}
}

func TestCommandsGolden(t *testing.T) {
	steps := []struct {
		name string
		id   command.ID
		args string
	}{
		{"commands-zoomed", "workspace.zoom", `{"pane":"logs"}`},
		{"commands-restored", "workspace.zoom", `{"pane":"logs"}`},
		{"commands-light", "workspace.theme.set", `{"background":"light"}`},
	}
	for i, s := range steps {
		tuitest.Golden(t, s.name, tuitest.Matrix{Widths: []int{80, 160}}, func(c tuitest.Case) string {
			p, g := c.Profile(), c.Glyphs()
			// A workspace that follows the background, built for the case's
			// profile and glyphs, starting dark.
			build := func(_ colorprofile.Profile, bg theme.Background) theme.Theme { return theme.New(p, bg, g) }
			ws := session(agentRoot(false, layout.FullWidth), theme.New(p, theme.Dark, g), c.Width, 30,
				workspace.WithThemeBuilder(build))
			r := command.NewRegistry()
			if err := r.Register(workspace.Commands(ws)...); err != nil {
				t.Fatal(err)
			}
			for _, prior := range steps[:i+1] {
				if err := dispatch(t, ws, r, prior.id, prior.args); err != nil {
					t.Fatalf("%s: %v", prior.name, err)
				}
			}
			return ws.Render()
		})
	}
}
