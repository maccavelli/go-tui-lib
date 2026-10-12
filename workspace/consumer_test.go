package workspace

import (
	"image/color"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/keymap"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
)

// consumer is a fake that runs the commands in ids itself, or every
// command with all; on, when set, answers each CommandMsg.
type consumer struct {
	*fake
	ids map[command.ID]bool
	all bool
	on  func(CommandMsg) tea.Cmd
}

func (c *consumer) ConsumesCommand(id command.ID) bool { return c.all || c.ids[id] }

// Update records msg and returns c, not the fake inside it.
func (c *consumer) Update(msg tea.Msg) (Pane, tea.Cmd) {
	c.fake.Update(msg)
	var cmd tea.Cmd
	if m, ok := msg.(CommandMsg); ok && c.on != nil {
		// Called apart from the return: go-critic v0.15.0's evalOrder
		// checker panics on "return c, c.on(m)", taking the function
		// field for a method with no receiver.
		cmd = c.on(m)
	}
	return c, cmd
}

func (c *consumer) commands() []CommandMsg { return filter[CommandMsg](c.msgs) }

func newConsumer(id string, ids ...command.ID) *consumer {
	c := &consumer{fake: &fake{id: id}, ids: map[command.ID]bool{}}
	for _, i := range ids {
		c.ids[i] = true
	}
	return c
}

// consumerRig is a rig whose keymap binds ctrl+k to x.y, globally, with
// args.
func consumerRig(t *testing.T) rig {
	t.Helper()
	km := mustKeymap(t, keymap.Settings{}, []keymap.Binding{user(t, "", "ctrl+k", "x.y", `{"n":1}`, false)})
	return newRig(t, 120, 40, WithBindings(km))
}

// requests is the RequestMsgs among msgs.
func requests(msgs []tea.Msg) []RequestMsg { return filter[RequestMsg](msgs) }

// TestModalConsumer: a modal overlay that consumes x.y holds its
// CommandMsg when Update returns, before any command runs, with the
// request a RequestMsg would carry; and no RequestMsg is made.
func TestModalConsumer(t *testing.T) {
	r := consumerRig(t)
	dlg := newConsumer("dlg", "x.y")
	r.w.Push(Overlay{ID: "dlg", Pane: dlg, Width: 20, Height: 6, Modal: true})
	cmd := r.w.Update(ctrl('k'))
	got := dlg.commands()
	if len(got) != 1 {
		t.Fatalf("the modal consumer holds %d CommandMsgs when Update returns; want 1", len(got))
	}
	req := got[0].Request
	if req.ID != "x.y" || string(req.Args) != `{"n":1}` || req.Origin != command.OriginKey || req.WhenContext == nil {
		t.Errorf("the request is %+v", req)
	}
	if v, ok := OverlayKey().Get(req.WhenContext); !ok || v != "dlg" {
		t.Errorf("the request's when values give the overlay %q, %v", v, ok)
	}
	if rs := requests(run(cmd)); len(rs) != 0 {
		t.Errorf("a consumed command also made %v", rs)
	}
	if len(dlg.keys()) != 0 {
		t.Errorf("the consumer also got the key: %v", dlg.keys())
	}
}

// TestConsumerOrder: with no modal overlay, the top overlay that consumes
// takes the command before a lower one and before the focused pane; the
// focused pane takes it when no overlay does; with none, RequestMsg.
func TestConsumerOrder(t *testing.T) {
	r := consumerRig(t)
	pane := newConsumer("main", "x.y")
	r.w.SetPane("main", pane)
	low, top := newConsumer("low", "x.y"), newConsumer("top", "x.y")
	r.w.Push(Overlay{ID: "low", Pane: low, Width: 10, Height: 4})
	r.w.Push(Overlay{ID: "top", Pane: top, Width: 10, Height: 4})

	r.w.Update(ctrl('k'))
	if len(top.commands()) != 1 || len(low.commands()) != 0 || len(pane.commands()) != 0 {
		t.Fatalf("top %d, low %d, pane %d; want the top overlay alone", len(top.commands()), len(low.commands()), len(pane.commands()))
	}
	delete(top.ids, "x.y")
	r.w.Update(ctrl('k'))
	if len(low.commands()) != 1 || len(pane.commands()) != 0 {
		t.Fatalf("with the top not consuming: low %d, pane %d; want the lower overlay", len(low.commands()), len(pane.commands()))
	}
	delete(low.ids, "x.y")
	r.w.Update(ctrl('k'))
	if len(pane.commands()) != 1 {
		t.Fatalf("with no overlay consuming, the focused pane got %d", len(pane.commands()))
	}
	delete(pane.ids, "x.y")
	msgs := run(r.w.Update(ctrl('k')))
	if rs := requests(msgs); len(rs) != 1 || rs[0].Request.ID != "x.y" {
		t.Fatalf("with no consumer: %v; want one RequestMsg for x.y", msgs)
	}
	for _, c := range []*consumer{pane, low, top} {
		if len(c.keys()) != 0 {
			t.Errorf("%s got the key itself: %v", c.id, c.keys())
		}
	}
}

// TestModalConsumerRefuses: a modal overlay that does not consume keeps
// the panes beneath from being offered the command, which goes to the
// program.
func TestModalConsumerRefuses(t *testing.T) {
	r := consumerRig(t)
	pane := newConsumer("main", "x.y")
	r.w.SetPane("main", pane)
	r.w.Push(Overlay{ID: "dlg", Pane: newConsumer("dlg"), Width: 20, Height: 6, Modal: true})
	msgs := run(r.w.Update(ctrl('k')))
	if len(pane.commands()) != 0 {
		t.Errorf("the pane beneath a modal overlay was given %v", pane.commands())
	}
	if rs := requests(msgs); len(rs) != 1 {
		t.Errorf("got %v; want one RequestMsg", msgs)
	}
}

// TestConsumedInKeyOrder: ten keys bound to ten commands reach the consumer
// in the order they were sent, each before its Update returns and before
// any command runs.
func TestConsumedInKeyOrder(t *testing.T) {
	// Ten letters whose ctrl form is no other key on a legacy terminal, as
	// ctrl+i is tab and ctrl+m enter.
	letters := []rune("abdefgknop")
	var binds []keymap.Binding
	var ids []command.ID
	for _, c := range letters {
		id := command.ID("x." + string(c))
		ids = append(ids, id)
		binds = append(binds, user(t, "", "ctrl+"+string(c), id, "", false))
	}
	r := newRig(t, 120, 40, WithBindings(mustKeymap(t, keymap.Settings{}, binds)))
	dlg := newConsumer("dlg", ids...)
	r.w.Push(Overlay{ID: "dlg", Pane: dlg, Width: 20, Height: 6, Modal: true})
	for i, c := range letters {
		r.w.Update(ctrl(c))
		if got := dlg.commands(); len(got) != i+1 || got[i].Request.ID != ids[i] {
			t.Fatalf("after key %d the consumer holds %v; want %v in order", i, got, ids[:i+1])
		}
	}
}

// TestOwnCommandsFirst: the workspace's own commands still run though a
// pane consumes every command, and esc still closes an overlay that
// consumes every command but not esc.
func TestOwnCommandsFirst(t *testing.T) {
	r := newRig(t, 120, 40)
	pane := newConsumer("main")
	pane.all = true
	r.w.SetPane("main", pane)
	run(r.w.Update(press("alt+z")))
	if r.w.State().Zoom != "main" || len(pane.commands()) != 0 {
		t.Fatalf("alt+z: zoom %q, the pane was given %v", r.w.State().Zoom, pane.commands())
	}
	all := newConsumer("all")
	all.all = true
	r.w.Push(Overlay{ID: "all", Pane: all, Width: 20, Height: 6, Modal: true})
	run(r.w.Update(press("esc")))
	if len(r.w.Overlays()) != 0 || len(all.commands()) != 0 {
		t.Errorf("esc: overlays %v, the overlay was given %v", r.w.Overlays(), all.commands())
	}
}

// TestClose: Close closes the named overlay, lower or top, with Pop's focus
// messages; an ID not open changes nothing; and an overlay that closes
// itself from its own Update leaves the others intact.
func TestClose(t *testing.T) {
	r := newRig(t, 120, 40)
	r.w.Push(Overlay{ID: "a", Pane: &fake{id: "a"}, Width: 10, Height: 4})
	dlg := &fake{id: "dlg"}
	r.w.Push(Overlay{ID: "b", Pane: &fake{id: "b"}, Width: 10, Height: 4})
	r.w.Push(Overlay{ID: "dlg", Pane: dlg, Width: 20, Height: 6, Modal: true})

	if cmd := r.w.Update(Close("nope")); cmd != nil || !slices.Equal(r.w.Overlays(), []string{"a", "b", "dlg"}) {
		t.Fatalf("Close of an ID not open: %v, cmd %v", r.w.Overlays(), cmd != nil)
	}
	r.w.Update(Close("a"))
	if !slices.Equal(r.w.Overlays(), []string{"b", "dlg"}) {
		t.Fatalf("Close(a) left %v", r.w.Overlays())
	}
	r.main.reset()
	dlg.reset()
	run(r.w.Update(Close("dlg")))
	if !slices.Equal(r.w.Overlays(), []string{"b"}) {
		t.Fatalf("Close(dlg) left %v", r.w.Overlays())
	}
	if !dlg.got(PaneBlurMsg{}) || !r.main.got(PaneFocusMsg{}) {
		t.Errorf("Close of the modal overlay: it got blur %v, the focused pane focus %v", dlg.got(PaneBlurMsg{}), r.main.got(PaneFocusMsg{}))
	}

	km := mustKeymap(t, keymap.Settings{}, []keymap.Binding{user(t, "", "ctrl+k", "x.y", "", false)})
	r = newRig(t, 120, 40, WithBindings(km))
	r.w.Push(Overlay{ID: "under", Pane: &fake{id: "under"}, Width: 10, Height: 4})
	self := newConsumer("self", "x.y")
	self.on = func(CommandMsg) tea.Cmd { return func() tea.Msg { return Close("self") } }
	r.w.Push(Overlay{ID: "self", Pane: self, Width: 20, Height: 6, Modal: true})
	for _, m := range run(r.w.Update(ctrl('k'))) {
		r.w.Update(m)
	}
	if !slices.Equal(r.w.Overlays(), []string{"under"}) {
		t.Errorf("an overlay closing itself left %v; want under", r.w.Overlays())
	}
}

// TestThemeAndBindings: Theme follows SetTheme, and the terminal's
// background when the workspace follows it; Bindings is DefaultRules' by
// default, and follows SetBindings.
func TestThemeAndBindings(t *testing.T) {
	w := New(layout.Pane{ID: "main"}, map[layout.PaneID]Pane{"main": &fake{id: "main"}})
	w.Init()
	w.Update(tea.BackgroundColorMsg{Color: color.Black})
	if bg := w.Theme().Background; bg != theme.Dark {
		t.Errorf("after a dark background the theme's is %v", bg)
	}
	w.Update(tea.BackgroundColorMsg{Color: color.White})
	if bg := w.Theme().Background; bg != theme.Light {
		t.Errorf("after a light background the theme's is %v", bg)
	}
	w.SetTheme(asciiTheme)
	if got := w.Theme(); got.Profile != asciiTheme.Profile || got.Glyphs != asciiTheme.Glyphs || got.Background != asciiTheme.Background {
		t.Errorf("Theme does not follow SetTheme: profile %v, background %v", got.Profile, got.Background)
	}

	if !slices.ContainsFunc(slices.Collect(w.Bindings().Rules()), func(b keymap.Binding) bool { return b.Command == "workspace.zoom" }) {
		t.Error("the default Bindings have no workspace.zoom")
	}
	km := chordKeymap(t)
	w.SetBindings(km)
	if w.Bindings() != km {
		t.Error("Bindings does not follow SetBindings")
	}
	w.SetBindings(nil)
	if w.Bindings() == km || w.Bindings() == nil {
		t.Error("SetBindings(nil) did not go back to DefaultRules'")
	}
}
