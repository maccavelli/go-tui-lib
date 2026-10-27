package workspace

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// scoped is a fake that names key scopes.
type scoped struct {
	*fake
	scopes []command.Scope
}

func (s *scoped) KeyContexts() []command.Scope { return s.scopes }

// Update records msg and returns s, not the fake inside it, which the
// workspace would keep in its place.
func (s *scoped) Update(msg tea.Msg) (Pane, tea.Cmd) {
	s.fake.Update(msg)
	return s, nil
}

// path is a key path's scopes, joined by a space.
func path(p keymap.Path) string {
	var out []string
	for _, s := range p {
		out = append(out, s.String())
	}
	return strings.Join(out, " ")
}

// mustKeymap builds a keymap over DefaultRules that must have no error.
func mustKeymap(t *testing.T, s keymap.Settings, layers ...[]keymap.Binding) *keymap.Keymap {
	t.Helper()
	k, err := keymap.New(s, DefaultRules(), layers...)
	if err != nil {
		t.Fatalf("keymap.New: %v", err)
	}
	return k
}

// user is a user-layer binding of keys to id in scope; no keys and remove
// make a removal.
func user(t *testing.T, scope command.Scope, keys string, id command.ID, args string, remove bool) keymap.Binding {
	t.Helper()
	b := keymap.Binding{Command: id, Scope: scope, Remove: remove, Location: keymap.Location{Layer: keymap.UserLayer}}
	if keys != "" {
		q, err := keymap.ParseSequence(keys, nil)
		if err != nil {
			t.Fatal(err)
		}
		b.Keys = q
	}
	if args != "" {
		b.Args = jsontext.Value(args)
	}
	return b
}

// ctrl is ctrl and a letter, pressed.
func ctrl(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl} }

// TestDefaultRulesGolden: MADR §9's defaults, with A4's commands, pinned.
func TestDefaultRulesGolden(t *testing.T) {
	var b strings.Builder
	for _, d := range DefaultRules() {
		args := string(d.Args)
		if args == "" {
			args = "-"
		}
		fmt.Fprintf(&b, "%-10s %-16s %-27s %s\n", d.Scope, d.Keys, d.Command, args)
	}
	tuitest.Text(t, "default-rules", b.String())
	if _, err := keymap.New(keymap.Settings{}, DefaultRules()); err != nil {
		t.Errorf("DefaultRules do not build clean: %v", err)
	}
	// Each command's first key is its primary, which the help shows.
	r := newRig(t, 120, 40)
	for id, want := range map[command.ID]string{"workspace.focus.index": "alt+1", "workspace.resize.direction": "alt+shift+left"} {
		if got := r.w.helpBinding(id).Help().Key; got != want {
			t.Errorf("%s's help key is %q; want %q", id, got, want)
		}
	}
}

// TestKeyPath: the path for a focused pane, a KeyContexter pane, a
// non-modal overlay and a modal one, with a mode last.
func TestKeyPath(t *testing.T) {
	r := newRig(t, 120, 40)
	if got, want := path(r.w.KeyPath()), "global workspace pane:main"; got != want {
		t.Errorf("a focused pane: %q; want %q", got, want)
	}
	r.w.SetPane("main", &scoped{fake: r.main, scopes: []command.Scope{"kind:editor", "mode:vim-normal"}})
	if got, want := path(r.w.KeyPath()), "global workspace pane:main kind:editor mode:vim-normal"; got != want {
		t.Errorf("a KeyContexter: %q; want %q", got, want)
	}
	r.w.Push(Overlay{ID: "tip", Pane: &fake{id: "tip"}, Width: 10, Height: 4})
	if got, want := path(r.w.KeyPath()), "global workspace pane:main kind:editor mode:vim-normal overlay overlay:tip"; got != want {
		t.Errorf("a non-modal overlay: %q; want %q", got, want)
	}
	r.w.Push(Overlay{ID: "dlg", Pane: &scoped{fake: &fake{id: "dlg"}, scopes: []command.Scope{"mode:dialog"}}, Width: 20, Height: 6, Modal: true})
	pop := r.w.PushMode("mode:modal")
	if got, want := path(r.w.KeyPath()), "global overlay overlay:dlg mode:dialog mode:modal"; got != want {
		t.Errorf("a modal overlay: %q; want %q", got, want)
	}
	pop()
}

// TestPushMode: a mode goes at the end of the path; pop removes its own,
// whatever was pushed since, and a second pop does nothing.
func TestPushMode(t *testing.T) {
	r := newRig(t, 120, 40)
	popA := r.w.PushMode("mode:a")
	popB := r.w.PushMode("mode:b")
	if got := path(r.w.KeyPath()); !strings.HasSuffix(got, "pane:main mode:a mode:b") {
		t.Errorf("two modes: %q", got)
	}
	popA()
	if got := path(r.w.KeyPath()); !strings.HasSuffix(got, "pane:main mode:b") {
		t.Errorf("after the first's pop: %q; want mode:b left", got)
	}
	popA()
	if got := path(r.w.KeyPath()); !strings.HasSuffix(got, "pane:main mode:b") {
		t.Errorf("after a second pop: %q; want it unchanged", got)
	}
	popB()
	if got, want := path(r.w.KeyPath()), "global workspace pane:main"; got != want {
		t.Errorf("after both pops: %q; want %q", got, want)
	}
}

// TestRebind: a user rule rebinds workspace.zoom, and the help follows; a
// "-workspace.zoom" rule alone removes it, and its key reaches the pane.
func TestRebind(t *testing.T) {
	km := mustKeymap(t, keymap.Settings{}, []keymap.Binding{
		user(t, "workspace", "", "workspace.zoom", "", true),
		user(t, "workspace", "ctrl+g", "workspace.zoom", "", false),
	})
	r := newRig(t, 120, 40, WithBindings(km))
	r.w.Update(press("alt+z"))
	if r.w.State().Zoom != "" || len(r.main.keys()) != 1 {
		t.Fatalf("alt+z after the rebind: zoom %q, main keys %v", r.w.State().Zoom, r.main.keys())
	}
	run(r.w.Update(ctrl('g')))
	if r.w.State().Zoom != "main" {
		t.Fatalf("ctrl+g did not zoom: %+v", r.w.State())
	}
	if h := r.w.ShortHelp(); len(h) != 2 || h[1].Help().Key != "ctrl+g" || h[1].Help().Desc != "zoom" {
		t.Errorf("the short help is %v; want ctrl+g zoom", h)
	}
	r = newRig(t, 120, 40, WithBindings(mustKeymap(t, keymap.Settings{}, []keymap.Binding{user(t, "workspace", "", "workspace.zoom", "", true)})))
	r.w.Update(press("alt+z"))
	if r.w.State().Zoom != "" || len(r.main.keys()) != 1 {
		t.Errorf("alt+z after the removal: zoom %q, main keys %v", r.w.State().Zoom, r.main.keys())
	}
	if h := r.w.ShortHelp(); len(h) != 1 {
		t.Errorf("the short help after the removal is %v; want next pane alone", h)
	}
}

// chordKeymap binds the program's ctrl+x n, with a timeout of a
// nanosecond, so that its tea.Tick fires at once.
func chordKeymap(t *testing.T) *keymap.Keymap {
	prog := keymap.Binding{Keys: mustSeq(t, "ctrl+x n"), Command: "app.new", Location: keymap.Location{Layer: keymap.ProgramLayer}}
	return mustKeymap(t, keymap.Settings{Timeout: time.Nanosecond}, []keymap.Binding{prog})
}

func mustSeq(t *testing.T, s string) keymap.Sequence {
	t.Helper()
	q, err := keymap.ParseSequence(s, nil)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

// TestChord: a chord prefix typed in a pane is held, and replayed to it
// when the timeout fires; a completed chord of the program's own comes
// back as a RequestMsg from a key, with the workspace's when values.
func TestChord(t *testing.T) {
	r := newRig(t, 120, 40, WithBindings(chordKeymap(t)))
	msgs := run(r.w.Update(ctrl('x')))
	if len(r.main.keys()) != 0 {
		t.Fatalf("the prefix reached the pane at once: %v", r.main.keys())
	}
	i := slices.IndexFunc(msgs, func(m tea.Msg) bool { _, ok := m.(keymap.TimeoutMsg); return ok })
	if i < 0 {
		t.Fatalf("no timer from the prefix: %v", msgs)
	}
	r.w.Update(msgs[i])
	if k := r.main.keys(); len(k) != 1 || k[0].String() != "ctrl+x" {
		t.Fatalf("after the timeout the pane got %v; want ctrl+x", k)
	}
	r.main.reset()
	r.w.Update(ctrl('x'))
	msgs = run(r.w.Update(press("n")))
	j := slices.IndexFunc(msgs, func(m tea.Msg) bool { _, ok := m.(RequestMsg); return ok })
	if j < 0 || len(r.main.keys()) != 0 {
		t.Fatalf("ctrl+x n: %v, pane keys %v; want a RequestMsg and nothing for the pane", msgs, r.main.keys())
	}
	req := msgs[j].(RequestMsg).Request
	if req.ID != "app.new" || req.Origin != command.OriginKey || req.WhenContext == nil {
		t.Fatalf("the request is %+v", req)
	}
	if v, ok := FocusedPaneKey().Get(req.WhenContext); !ok || v != "main" {
		t.Errorf("the request's when values give the focused pane %q, %v", v, ok)
	}
}

// TestSetBindings: a new keymap cancels a pending sequence, and nil is
// DefaultRules again.
func TestSetBindings(t *testing.T) {
	km := chordKeymap(t)
	r := newRig(t, 120, 40, WithBindings(km))
	r.w.Update(ctrl('x'))
	r.w.SetBindings(km)
	msgs := run(r.w.Update(press("n")))
	if slices.ContainsFunc(msgs, func(m tea.Msg) bool { _, ok := m.(RequestMsg); return ok }) || len(r.main.keys()) != 1 {
		t.Errorf("n after SetBindings: %v, pane keys %v; want n for the pane", msgs, r.main.keys())
	}
	r.w.SetBindings(nil)
	run(r.w.Update(press("alt+z")))
	if r.w.State().Zoom != "main" {
		t.Errorf("alt+z after SetBindings(nil): %+v", r.w.State())
	}
}

// TestOwnCommandError: the workspace's own command, matched with bad
// arguments, reports its error as a ResultMsg.
func TestOwnCommandError(t *testing.T) {
	km := mustKeymap(t, keymap.Settings{}, []keymap.Binding{user(t, "workspace", "ctrl+o", "workspace.focus", `{"pane":"nope"}`, false)})
	r := newRig(t, 120, 40, WithBindings(km))
	msgs := run(r.w.Update(ctrl('o')))
	i := slices.IndexFunc(msgs, func(m tea.Msg) bool { _, ok := m.(command.ResultMsg); return ok })
	if i < 0 {
		t.Fatalf("ctrl+o gave %v; want a ResultMsg", msgs)
	}
	if res := msgs[i].(command.ResultMsg); res.Err == nil || res.Request.ID != "workspace.focus" || res.Request.Origin != command.OriginKey {
		t.Errorf("the result is %+v", res)
	}
}

// TestEscConsumerThroughTheEngine: esc closes a plain overlay, and reaches
// an overlay that consumes it instead.
func TestEscConsumerThroughTheEngine(t *testing.T) {
	r := newRig(t, 120, 40)
	eater := &fake{id: "eater", esc: true}
	r.w.Push(Overlay{ID: "eater", Pane: eater, Width: 20, Height: 6})
	r.w.Update(press("esc"))
	if len(r.w.Overlays()) != 1 || len(eater.keys()) != 1 {
		t.Fatalf("esc on an EscConsumer: overlays %v, its keys %v", r.w.Overlays(), eater.keys())
	}
	eater.esc = false
	run(r.w.Update(press("esc")))
	if len(r.w.Overlays()) != 0 {
		t.Errorf("esc did not close the overlay")
	}
}

// TestWithoutOwnCommands: by default a workspace.* key runs in place; under
// the option it comes back as a RequestMsg, which a registry holding
// Commands runs through its gate policy, gate and auditor
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D4).
func TestWithoutOwnCommands(t *testing.T) {
	r := newRig(t, 120, 40)
	was := r.w.Focused()
	if reqs := requests(run(r.w.Update(press("alt+.")))); len(reqs) != 0 || r.w.Focused() == was {
		t.Fatalf("by default alt+. gave %v, focus %s from %s; want focus moved and no RequestMsg", reqs, r.w.Focused(), was)
	}

	r = newRig(t, 120, 40, WithoutOwnCommands())
	was = r.w.Focused()
	ctx := r.w.WhenContext()
	reqs := requests(run(r.w.Update(press("alt+."))))
	if len(reqs) != 1 || r.w.Focused() != was {
		t.Fatalf("under the option alt+. gave %v, focus %s from %s; want one RequestMsg and focus unmoved", reqs, r.w.Focused(), was)
	}
	req := reqs[0].Request
	if req.ID != "workspace.focus.next" || req.Origin != command.OriginKey || !reflect.DeepEqual(req.WhenContext, ctx) {
		t.Fatalf("the request is %+v; want workspace.focus.next from a key, with the workspace's when values", req)
	}

	var audited []command.Record
	var asked []command.ID
	gate := command.GateFunc(func(_ context.Context, inv *command.Invocation) (command.Verdict, error) {
		asked = append(asked, inv.Command.ID)
		return command.RejectOnce, nil
	})
	reg := command.NewRegistry(command.WithGate(gate), command.WithAuditor(command.AuditorFunc(func(rec command.Record) { audited = append(audited, rec) })))
	if err := reg.Register(Commands(r.w)...); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Run(t.Context(), req); err != nil || r.w.Focused() == was {
		t.Fatalf("the registry ran the request: %v, focus %s; want focus moved from %s", err, r.w.Focused(), was)
	}
	if len(audited) != 1 || audited[0].ID != req.ID || audited[0].Origin != command.OriginKey || len(asked) != 0 {
		t.Errorf("the auditor holds %+v, and the gate was asked about %v; want the one request, the gate unasked", audited, asked)
	}

	ui := command.GatePolicyFunc(func(c command.Command, _ command.Origin) bool { return c.Danger == command.UI })
	gated := command.NewRegistry(command.WithGate(gate), command.WithGatePolicy(ui))
	if err := gated.Register(Commands(r.w)...); err != nil {
		t.Fatal(err)
	}
	was = r.w.Focused()
	if _, err := gated.Run(t.Context(), req); !errors.Is(err, command.ErrRefused) || r.w.Focused() != was || !slices.Equal(asked, []command.ID{req.ID}) {
		t.Errorf("under a policy that gates every UI command: %v, focus %s from %s, the gate asked about %v; want it refused by the gate", err, r.w.Focused(), was, asked)
	}
}

// TestWithoutOwnCommandsEsc: under the option an EscConsumer still takes
// esc first, and a plain overlay's esc comes back as a RequestMsg for
// workspace.overlay.close.
func TestWithoutOwnCommandsEsc(t *testing.T) {
	r := newRig(t, 120, 40, WithoutOwnCommands())
	eater := &fake{id: "eater", esc: true}
	r.w.Push(Overlay{ID: "eater", Pane: eater, Width: 20, Height: 6})
	if reqs := requests(run(r.w.Update(press("esc")))); len(reqs) != 0 || len(r.w.Overlays()) != 1 || len(eater.keys()) != 1 {
		t.Fatalf("esc on an EscConsumer: %v, overlays %v, its keys %v; want it delivered to the overlay", reqs, r.w.Overlays(), eater.keys())
	}
	eater.esc = false
	reqs := requests(run(r.w.Update(press("esc"))))
	if len(reqs) != 1 || reqs[0].Request.ID != cmdOverlayClose || len(r.w.Overlays()) != 1 {
		t.Errorf("esc on a plain overlay: %v, overlays %v; want a RequestMsg for %s and the overlay open", reqs, r.w.Overlays(), cmdOverlayClose)
	}
}

// TestReleaseStillBroadcast: a release no binding takes reaches every
// pane, as before the engine.
func TestReleaseStillBroadcast(t *testing.T) {
	r := newRig(t, 120, 40)
	rel := tea.KeyReleaseMsg{Code: 'x', Text: "x"}
	r.w.Update(rel)
	for _, f := range []*fake{r.main, r.side, r.logs} {
		if !f.got(rel) {
			t.Errorf("%s did not get the release", f.id)
		}
	}
}
