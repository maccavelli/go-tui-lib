package palette

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"image/color"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/command/commandtea"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/theme"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// pane is the workspace's one pane, beneath the picker, recording what it
// is sent; a modal dialog in a test is one too.
type pane struct{ msgs []tea.Msg }

func (p *pane) Update(m tea.Msg) (workspace.Pane, tea.Cmd) { p.msgs = append(p.msgs, m); return p, nil }
func (p *pane) View(int, int) string                       { return "" }

// filter is the messages of type T among msgs.
func filter[T any](msgs []tea.Msg) []T {
	var out []T
	for _, m := range msgs {
		if v, ok := m.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

// prog plays a program's event loop around a real workspace and a picker,
// inside a synctest bubble: a command runs on a goroutine of its own, its
// message is queued, and settle hands each to the program's update, which
// dispatches a RequestMsg to its registry and gives the rest to the
// workspace, until the bubble is idle.
type prog struct {
	t     *testing.T
	w     *workspace.Workspace
	p     *Picker
	main  *pane
	reg   *command.Registry
	mu    sync.Mutex
	queue []tea.Msg
	seen  []tea.Msg // every message the program was given
}

// asciiTheme is the theme the behaviour tests draw with: no colour, ASCII
// glyphs.
var asciiTheme = theme.New(colorprofile.ASCII, theme.Unknown, glyph.ASCII())

// newProg is an 80 × 24 workspace whose keys go through km, or its
// defaults when km is nil, with a picker labelling hints for Linux.
func newProg(t *testing.T, km *keymap.Keymap, wopts []workspace.Option, opts ...Option) *prog {
	main := &pane{}
	g := newProgOn(t, layout.Pane{ID: "main"}, map[layout.PaneID]workspace.Pane{"main": main}, km, wopts, opts...)
	g.main = main
	return g
}

// newProgOn is newProg over the layout root and its panes.
func newProgOn(t *testing.T, root layout.Node, panes map[layout.PaneID]workspace.Pane, km *keymap.Keymap, wopts []workspace.Option, opts ...Option) *prog {
	g := &prog{t: t}
	if wopts == nil {
		wopts = []workspace.Option{workspace.WithTheme(asciiTheme)}
	}
	wopts = append(wopts, workspace.WithoutBackgroundQuery())
	if km != nil {
		wopts = append(wopts, workspace.WithBindings(km))
	}
	g.w = workspace.New(root, panes, wopts...)
	g.run(g.w.Init())
	g.send(tea.WindowSizeMsg{Width: 80, Height: 24})
	g.p = NewPicker(g.w, append([]Option{WithGOOS("linux")}, opts...)...)
	g.settle()
	return g
}

// keys is a keymap of the workspace's and the palette's defaults, and the
// user's layers.
func keys(t *testing.T, layers ...[]keymap.Binding) *keymap.Keymap {
	t.Helper()
	k, err := keymap.New(keymap.Settings{}, append(workspace.DefaultRules(), DefaultRules()...), layers...)
	if err != nil {
		t.Fatalf("keymap.New: %v", err)
	}
	return k
}

// user is a user-layer binding of keys to id in scope, or its removal.
func user(t *testing.T, scope command.Scope, keys string, id command.ID, remove bool) keymap.Binding {
	t.Helper()
	q, err := keymap.ParseSequence(keys, nil)
	if err != nil {
		t.Fatal(err)
	}
	return keymap.Binding{Keys: q, Command: id, Scope: scope, Remove: remove, Location: keymap.Location{Layer: keymap.UserLayer}}
}

func (g *prog) register(p Provider) {
	g.t.Helper()
	if err := g.p.Register(p, Availability{}); err != nil {
		g.t.Fatal(err)
	}
}

func (g *prog) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		if m := cmd(); m != nil {
			g.mu.Lock()
			g.queue = append(g.queue, m)
			g.mu.Unlock()
		}
	}()
}

// send is the program's update.
func (g *prog) send(m tea.Msg) {
	g.seen = append(g.seen, m)
	if r, ok := m.(workspace.RequestMsg); ok {
		if g.reg != nil {
			g.run(commandtea.Dispatch(context.Background(), g.reg, r.Request))
		}
		return
	}
	g.run(g.w.Update(m))
}

func (g *prog) settle() {
	for {
		synctest.Wait()
		g.mu.Lock()
		q := g.queue
		g.queue = nil
		g.mu.Unlock()
		if len(q) == 0 {
			return
		}
		for _, m := range q {
			if b, ok := m.(tea.BatchMsg); ok {
				for _, c := range b {
					g.run(c)
				}
				continue
			}
			g.send(m)
		}
	}
}

// open opens the picker directly, as a program does, and settles.
func (g *prog) open(opts ...OverlayOption) {
	g.run(g.p.Open(opts...))
	g.settle()
}

// key presses a named key and settles.
func (g *prog) key(s string) {
	g.send(keyMsg(s))
	g.settle()
}

// typed types each character of s, settling after each.
func (g *prog) typed(s string) {
	for _, r := range s {
		g.key(string(r))
	}
}

// keyMsg is the key s names: a named key, ctrl and a letter, or a
// character.
func keyMsg(s string) tea.KeyPressMsg {
	switch s {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	}
	if c, ok := strings.CutPrefix(s, "ctrl+"); ok && len(c) == 1 {
		return tea.KeyPressMsg{Code: rune(c[0]), Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

// requests is the RequestMsgs the program was given.
func (g *prog) requests() []workspace.RequestMsg { return filter[workspace.RequestMsg](g.seen) }

// texts is the rows' texts.
func (g *prog) texts() []string {
	var out []string
	for _, r := range g.p.eng.rows() {
		out = append(out, r.hit.Text)
	}
	return out
}

// view is the picker drawn at its content size, escape sequences removed.
func (g *prog) view() string { return ansi.Strip(g.p.View(g.p.width, g.p.height)) }

// ran records the hits run, in order.
type ran struct {
	mu  sync.Mutex
	ids []string
}

func (r *ran) list() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.ids)
}

// runs is n hits, a1 … an, best first, each recording its run in r.
func runs(r *ran, n int) []Hit {
	var hs []Hit
	for i := range n {
		id := string(rune('a'+i)) + "1"
		hs = append(hs, Hit{ID: id, Text: id, Score: fuzzy.Score{Grade: fuzzy.Exact, Value: n - i}, Run: func(jsontext.Value) tea.Cmd {
			r.mu.Lock()
			r.ids = append(r.ids, id)
			r.mu.Unlock()
			return nil
		}})
	}
	return hs
}

// discovering is a provider whose recent hits are hs, for any query.
type discovering struct{ *fake }

func newDiscovering(name string, hs ...Hit) discovering {
	return discovering{&fake{name: name, search: yields(hs...)}}
}

func (d discovering) Discover(ctx context.Context, yield func(Hit) bool) error {
	return d.search(ctx, Search{}, yield)
}

// TestOpenKey: ctrl+p comes back as RequestMsg for palette.open; with
// OpenCommand's command in the registry, dispatching it opens the picker;
// and with a modal overlay that is not a picker open, ctrl+p is not
// matched, as !workspace.modal says.
func TestOpenKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		g.key("ctrl+p")
		if rs := g.requests(); len(rs) != 1 || rs[0].Request.ID != OpenID {
			t.Fatalf("ctrl+p: %v; want one RequestMsg for palette.open", rs)
		}
		if o := g.w.Overlays(); len(o) != 0 {
			t.Fatalf("with no registry, ctrl+p opened %v", o)
		}
		g.reg = command.NewRegistry()
		if err := g.reg.Register(g.p.OpenCommand(OpenID, "Open the palette")); err != nil {
			t.Fatal(err)
		}
		g.key("ctrl+p")
		if o := g.w.Overlays(); !slices.Equal(o, []string{"palette"}) {
			t.Fatalf("dispatching palette.open: overlays %v; want the picker", o)
		}
		g.key("esc")
		if o := g.w.Overlays(); len(o) != 0 {
			t.Fatalf("esc with an empty query left %v", o)
		}

		dlg := &pane{}
		g.run(g.w.Push(workspace.Overlay{ID: "dlg", Pane: dlg, Width: 20, Height: 6, Modal: true}))
		g.settle()
		n := len(g.requests())
		g.key("ctrl+p")
		if len(g.requests()) != n || len(filter[tea.KeyPressMsg](dlg.msgs)) != 1 {
			t.Errorf("ctrl+p under a modal dialog: %d more requests, the dialog got %v; want none, and the key",
				len(g.requests())-n, filter[tea.KeyPressMsg](dlg.msgs))
		}
	})
}

// TestPickerKeys: in an open picker, ctrl+n and ctrl+p, down and up, and
// pgdown and pgup move the selection, as commands the picker takes, so no
// RequestMsg is made; printable keys reach the input and search again;
// tab completes the query to the selected hit; enter runs it and closes
// the picker.
func TestPickerKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		var r ran
		g.register(&fake{name: "items", search: yields(runs(&r, 5)...)})
		g.open()
		g.typed("x")
		if got := g.texts(); !slices.Equal(got, []string{"a1", "b1", "c1", "d1", "e1"}) {
			t.Fatalf("rows %q", got)
		}
		for _, step := range []struct {
			key  string
			want int
		}{{"ctrl+n", 1}, {"ctrl+n", 2}, {"ctrl+p", 1}, {"down", 2}, {"up", 1}, {"pgdown", 4}, {"pgup", 0}, {"up", 0}} {
			g.key(step.key)
			if g.p.sel != step.want {
				t.Fatalf("%s: selection %d; want %d", step.key, g.p.sel, step.want)
			}
		}
		if rs := g.requests(); len(rs) != 0 {
			t.Fatalf("the picker's keys made %v", rs)
		}
		if v := g.p.input.Value(); v != "x" {
			t.Fatalf("the input holds %q; want x", v)
		}
		g.key("down")
		g.key("tab")
		if v := g.p.input.Value(); v != "b1" {
			t.Fatalf("tab: the input holds %q; want b1", v)
		}
		g.key("down")
		g.key("enter")
		if got := r.list(); !slices.Equal(got, []string{"b1"}) {
			t.Fatalf("enter ran %q; want b1, the second row after tab searched again", got)
		}
		if o := g.w.Overlays(); len(o) != 0 {
			t.Fatalf("enter left %v open", o)
		}
	})
}

// TestKeysRunInOrder: "down, down, enter" given to the workspace in one
// burst, before any command runs, runs the third row; keys after the enter,
// in the same burst, do nothing, as the picker waits to be closed.
func TestKeysRunInOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		var r ran
		g.register(newDiscovering("items", runs(&r, 5)...))
		g.open()
		for _, k := range []string{"down", "down", "enter", "down", "enter"} {
			g.send(keyMsg(k))
		}
		g.settle()
		if got := r.list(); !slices.Equal(got, []string{"c1"}) {
			t.Fatalf("down, down, enter, down, enter ran %q; want c1 alone", got)
		}
	})
}

// TestDisabledDoesNotRun: a disabled hit shows its reason, on its row and
// in the help line, and enter does not run it or close the picker.
func TestDisabledDoesNotRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		var r ran
		hs := runs(&r, 2)
		hs[0].Disabled = "not here"
		g.register(newDiscovering("items", hs...))
		g.open()
		if v := g.view(); strings.Count(v, "not here") != 2 {
			t.Fatalf("the reason is not on the row and in the help line:\n%s", v)
		}
		g.key("enter")
		if got, o := r.list(), g.w.Overlays(); len(got) != 0 || !slices.Equal(o, []string{"palette"}) {
			t.Fatalf("enter on a disabled hit ran %q, overlays %v", got, o)
		}
		g.key("down")
		g.key("enter")
		if got := r.list(); !slices.Equal(got, []string{"b1"}) {
			t.Errorf("enter on the next hit ran %q; want b1", got)
		}
	})
}

// TestScrollFollowsSelection: moving the selection past the window's edge
// scrolls it, one line at a time, both ways.
func TestScrollFollowsSelection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		var r ran
		g.register(newDiscovering("items", runs(&r, 20)...))
		g.open(WithSize(40, 12)) // 8 result lines
		for range 9 {
			g.key("down")
		}
		if g.p.sel != 9 || g.p.top != 2 {
			t.Fatalf("nine downs: selection %d, top %d; want 9 and 2", g.p.sel, g.p.top)
		}
		if lines := strings.Split(g.view(), "\n"); !strings.Contains(lines[1], "c1") || !strings.Contains(lines[8], "j1") {
			t.Fatalf("the window is not c1 to j1:\n%s", g.view())
		}
		for range 9 {
			g.key("up")
		}
		if g.p.sel != 0 || g.p.top != 0 {
			t.Errorf("nine ups: selection %d, top %d; want 0 and 0", g.p.sel, g.p.top)
		}
	})
}

// TestGroupHeaders: under ByProvider each provider's rows follow a header
// naming its group, and the selection moves over the headers.
func TestGroupHeaders(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil, WithGrouping(ByProvider))
		g.register(newDiscovering("alpha", hit("a1", fuzzy.Exact, 1), hit("a2", fuzzy.Exact, 0)))
		g.register(newDiscovering("beta", hit("b1", fuzzy.Exact, 9)))
		g.open()
		var got []string
		for _, l := range strings.Split(g.view(), "\n")[1:6] {
			got = append(got, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), ">")))
		}
		if want := []string{"alpha", "a1", "a2", "beta", "b1"}; !slices.Equal(got, want) {
			t.Fatalf("lines %q; want %q", got, want)
		}
		g.key("down")
		g.key("down")
		if r, ok := g.p.selected(); !ok || r.hit.Text != "b1" {
			t.Errorf("two downs select %q; want b1", r.hit.Text)
		}
	})
}

// TestCompleteKeepsPrefix: tab completes a prefixed query to the selected
// hit under its prefix; accepting one of "?"'s rows types its prefix, and
// runs nothing.
func TestCompleteKeepsPrefix(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		g.register(prefixed{fake: &fake{name: "commands", search: yields(hit("Open file", fuzzy.Prefix, 1))}, prefix: ">"})
		g.register(&fake{name: "other", search: yields(hit("zzz", fuzzy.Prefix, 9))})
		g.open()
		g.typed(">op")
		g.key("tab")
		if v := g.p.input.Value(); v != ">Open file" {
			t.Fatalf("tab under a prefix: the input holds %q; want >Open file", v)
		}
		g.key("esc")
		g.typed("?")
		if got := g.texts(); !slices.Equal(got, []string{">"}) {
			t.Fatalf("? lists %q", got)
		}
		g.key("enter")
		if v, o := g.p.input.Value(), g.w.Overlays(); v != ">" || !slices.Equal(o, []string{"palette"}) {
			t.Errorf("enter on ?'s row: input %q, overlays %v; want > and open", v, o)
		}
	})
}

// braced draws its text between braces, which a row's text here lacks.
var braced = lipgloss.NewStyle().Transform(func(s string) string { return "{" + s + "}" })

// TestRowHighlights: in a drawn row, each highlight covers the cells
// fuzzy.Cells gives in the workspace's width method, past the selection
// marker, on a row whose ZWJ family before the match is 6 cells under
// WcWidth and 2 under GraphemeWidth (0008-MADR A5.3).
func TestRowHighlights(t *testing.T) {
	const text = "👨‍👩‍👧 ab cd ef"
	for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		synctest.Test(t, func(t *testing.T) {
			th := asciiTheme
			th.Styles.Body, th.Styles.Accent = lipgloss.NewStyle(), braced
			g := newProg(t, keys(t), []workspace.Option{workspace.WithTheme(th), workspace.WithWidthMethod(m)})
			g.register(&fake{name: "items", search: yields(Hit{ID: "f", Text: text})})
			g.open()
			g.typed("cdef")
			match, _ := fuzzy.NewMatcher("cdef").Match(text)
			var want []fuzzy.CellRange
			for _, c := range fuzzy.Cells(text, match.Ranges, m) {
				want = append(want, fuzzy.CellRange{Start: c.Start + 2, End: c.End + 2}) // past "> "
			}
			row := strings.Split(g.view(), "\n")[1]
			if got := bracedCells(row, m); len(want) != 2 || !slices.Equal(got, want) {
				t.Errorf("method %d: highlighted cells %v; want %v, in %q", m, got, want, row)
			}
		})
	}
}

// bracedCells is the cells of line drawn between braces, the braces not
// counted, measured with m.
func bracedCells(line string, m ansi.Method) []fuzzy.CellRange {
	var got []fuzzy.CellRange
	at := 0
	for {
		before, rest, ok := strings.Cut(line, "{")
		at += m.StringWidth(before)
		if !ok {
			return got
		}
		piece, after, _ := strings.Cut(rest, "}")
		w := m.StringWidth(piece)
		got = append(got, fuzzy.CellRange{Start: at, End: at + w})
		at += w
		line = after
	}
}

// TestEscClearsThenCloses: esc clears a non-empty query and searches
// again; a second esc closes the picker.
func TestEscClearsThenCloses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		g.register(&fake{name: "items", search: yields(hit("one", fuzzy.Exact, 1))})
		g.open()
		g.typed("ab")
		g.key("esc")
		if v, o := g.p.input.Value(), g.w.Overlays(); v != "" || !slices.Equal(o, []string{"palette"}) {
			t.Fatalf("esc with a query: input %q, overlays %v; want it cleared and open", v, o)
		}
		g.key("esc")
		if o := g.w.Overlays(); len(o) != 0 {
			t.Fatalf("esc with no query left %v", o)
		}
	})
}

// TestRebind: a user binding in kind:picker moves the selection with its
// key, and the default it removes no longer does.
// TestDefaultRulesUnchanged: DefaultRules is v2.1.0's, binding for
// binding, read from that tag's palette/keys.go, now that it is built from
// the exported IDs and CompletionRules; and the IDs keep their values.
func TestDefaultRulesUnchanged(t *testing.T) {
	const picker, popup = "kind:picker", "overlay:completion"
	v210 := [][4]string{
		{string(command.Global), "ctrl+p", "palette.open", "!workspace.modal"},
		{picker, "up", "palette.select.prev", ""},
		{picker, "ctrl+p", "palette.select.prev", ""},
		{picker, "down", "palette.select.next", ""},
		{picker, "ctrl+n", "palette.select.next", ""},
		{picker, "pgup", "palette.page.prev", ""},
		{picker, "pgdown", "palette.page.next", ""},
		{picker, "enter", "palette.accept", ""},
		{picker, "tab", "palette.complete", ""},
		{popup, "up", "palette.select.prev", ""},
		{popup, "ctrl+p", "palette.select.prev", ""},
		{popup, "down", "palette.select.next", ""},
		{popup, "ctrl+n", "palette.select.next", ""},
		{popup, "tab", "palette.accept", ""},
		{popup, "enter", "palette.accept", ""},
	}
	var got [][4]string
	for _, d := range DefaultRules() {
		got = append(got, [4]string{string(d.Scope), d.Keys.String(), string(d.Command), d.When})
		if d.Location.Layer != keymap.DefaultLayer {
			t.Errorf("%s %s is on layer %v, want the default layer", d.Scope, d.Keys, d.Location.Layer)
		}
	}
	if !slices.Equal(got, v210) {
		t.Errorf("DefaultRules =\n%q\nwant v2.1.0's\n%q", got, v210)
	}
	ids := []command.ID{OpenID, SelectNextID, SelectPrevID, PageNextID, PagePrevID, AcceptID, CompleteID}
	want := []command.ID{"palette.open", "palette.select.next", "palette.select.prev", "palette.page.next", "palette.page.prev", "palette.accept", "palette.complete"}
	if !slices.Equal(ids, want) || PickerScope != picker {
		t.Errorf("the IDs are %v and the scope %s; want %v and %s", ids, PickerScope, want, picker)
	}
}

func TestRebind(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		km := keys(t, []keymap.Binding{
			user(t, PickerScope, "down", SelectNextID, true),
			user(t, PickerScope, "ctrl+j", SelectNextID, false),
		})
		g := newProg(t, km, nil)
		var r ran
		g.register(newDiscovering("items", runs(&r, 3)...))
		g.open()
		g.key("ctrl+j")
		if g.p.sel != 1 {
			t.Fatalf("ctrl+j: selection %d; want 1", g.p.sel)
		}
		g.key("down")
		if g.p.sel != 1 {
			t.Errorf("down, unbound: selection %d; want 1", g.p.sel)
		}
	})
}

// TestWithoutRules: without palette.DefaultRules, down reaches the input,
// and the selection stays.
func TestWithoutRules(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, nil, nil)
		var r ran
		g.register(newDiscovering("items", runs(&r, 3)...))
		g.open()
		g.key("down")
		if g.p.sel != 0 || len(g.requests()) != 0 {
			t.Errorf("down: selection %d, requests %v; want 0 and none", g.p.sel, g.requests())
		}
		g.typed("q")
		if v := g.p.input.Value(); v != "q" {
			t.Errorf("the input holds %q; want q", v)
		}
	})
}

// TestThemeFollows: the picker draws with the workspace's theme of the
// moment, so a background report restyles it without reopening.
func TestThemeFollows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), []workspace.Option{workspace.WithProfile(colorprofile.TrueColor)})
		g.register(newDiscovering("items", Hit{ID: "x", Text: "x", Help: "the help"}))
		g.send(tea.BackgroundColorMsg{Color: color.Black})
		g.open()
		dark := g.p.View(g.p.width, g.p.height)
		g.send(tea.BackgroundColorMsg{Color: color.White})
		g.settle()
		light := g.p.View(g.p.width, g.p.height)
		if g.w.Theme().Background != theme.Light {
			t.Fatalf("the theme's background is %v", g.w.Theme().Background)
		}
		if dark == light {
			t.Error("the view did not change with the background")
		}
		if ansi.Strip(dark) != ansi.Strip(light) {
			t.Errorf("the text changed with the background:\n%s\n%s", ansi.Strip(dark), ansi.Strip(light))
		}
	})
}

// TestHitTextSanitized: a hit's text holding an escape sequence and a
// newline draws as one row of its visible text.
func TestHitTextSanitized(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		g.register(newDiscovering("items\x1b]0;title\x07", Hit{ID: "e", Text: "evil\x1b[31mred\nline", Help: "a\rb"}))
		g.open()
		raw := g.p.View(g.p.width, g.p.height)
		if strings.Contains(raw, "\x1b[31m") || strings.Contains(raw, "\x1b]0;") {
			t.Fatalf("the view holds the hit's escape sequences: %q", raw)
		}
		v := g.view()
		if n := strings.Count(v, "\n") + 1; n != g.p.height {
			t.Fatalf("%d lines; want %d", n, g.p.height)
		}
		if !strings.Contains(v, "evilred line") || !strings.Contains(v, "a b") {
			t.Errorf("the row is not one line of the visible text:\n%s", v)
		}
		if name := g.p.eng.sources[0].name; name != "items" {
			t.Errorf("the provider's name is %q", name)
		}
	})
}

// TestHints: a command hit's hint is the key along the path at open, so a
// binding of the focused pane's scope shows though the overlay's path
// lacks it, and it follows SetBindings.
func TestHints(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t, []keymap.Binding{user(t, "pane:main", "ctrl+k", "x.run", false)}), nil)
		g.register(newDiscovering("items", Hit{ID: "x", Text: "Run x", Command: "x.run"}))
		g.open()
		if v := g.view(); !strings.Contains(v, "ctrl+k") {
			t.Fatalf("no hint ctrl+k:\n%s", v)
		}
		g.w.SetBindings(keys(t, []keymap.Binding{user(t, "pane:main", "ctrl+y", "x.run", false)}))
		if v := g.view(); !strings.Contains(v, "ctrl+y") || strings.Contains(v, "ctrl+k") {
			t.Errorf("after SetBindings the hint is not ctrl+y:\n%s", v)
		}
	})
}

// TestOnlyVisibleMatched: of 200 hits, only the rows drawn are matched for
// their ranges.
func TestOnlyVisibleMatched(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		var hs []Hit
		for i := range 200 {
			hs = append(hs, hit(strings.Repeat("a", 1+i%7)+string(rune('A'+i%26)), fuzzy.Prefix, 200-i))
		}
		g.register(&fake{name: "many", search: yields(hs...)})
		g.open(WithSize(40, 12))
		g.typed("a")
		g.p.matched = 0
		g.p.View(g.p.width, g.p.height)
		if v := g.p.visible(); g.p.matched == 0 || g.p.matched > v {
			t.Errorf("%d rows matched for one view of %d visible rows", g.p.matched, v)
		}
	})
}

// TestCursor: the workspace's cursor is the input's, offset into the
// overlay's content: past the prompt and the query.
func TestCursor(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		g.open()
		g.typed("ab")
		c := g.w.Cursor()
		// The 72 × 16 overlay is centred in 80 × 24 at (4, 4); its content
		// starts inside the border, at (5, 5); the ASCII prompt "> " is two
		// cells.
		if c == nil || c.X != 5+2+2 || c.Y != 5 {
			t.Fatalf("cursor %+v; want (9, 5)", c)
		}
	})
}

// TestMouse: the wheel scrolls the results and keeps the selection drawn;
// a click runs the row under it.
func TestMouse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		var r ran
		g.register(newDiscovering("items", runs(&r, 20)...))
		g.open(WithSize(40, 12)) // at (20, 6) in 80 × 24; content at (21, 7), 10 rows: 8 results
		g.w.Render()
		g.send(tea.MouseWheelMsg{X: 25, Y: 10, Button: tea.MouseWheelDown})
		g.settle()
		if g.p.top != 1 || g.p.sel != 1 {
			t.Fatalf("wheel down: top %d, selection %d; want 1 and 1", g.p.top, g.p.sel)
		}
		g.send(tea.MouseWheelMsg{X: 25, Y: 10, Button: tea.MouseWheelUp})
		g.settle()
		if g.p.top != 0 {
			t.Fatalf("wheel up: top %d; want 0", g.p.top)
		}
		g.send(tea.MouseClickMsg{X: 25, Y: 7 + 3, Button: tea.MouseLeft}) // the third result line
		g.settle()
		if got := r.list(); !slices.Equal(got, []string{"c1"}) {
			t.Fatalf("a click on the third row ran %q; want c1", got)
		}
		if o := g.w.Overlays(); len(o) != 0 {
			t.Errorf("the click left %v open", o)
		}
	})
}

// marked is a style that draws its text between \x02 and \x03, so a test
// sees which pieces were drawn highlighted.
var marked = lipgloss.NewStyle().Transform(func(s string) string { return "\x02" + s + "\x03" })

// highlighted is the cells of out drawn between \x02 and \x03, measured
// with m.
func highlighted(out string, m ansi.Method) []fuzzy.CellRange {
	var got []fuzzy.CellRange
	at := 0
	for {
		before, rest, ok := strings.Cut(out, "\x02")
		at += m.StringWidth(before)
		if !ok {
			return got
		}
		piece, after, _ := strings.Cut(rest, "\x03")
		w := m.StringWidth(piece)
		got = append(got, fuzzy.CellRange{Start: at, End: at + w})
		at += w
		out = after
	}
}

// TestHighlightCells: each highlight covers the cells fuzzy.Cells gives,
// under WcWidth and GraphemeWidth, on a row with a ZWJ family, whose widths
// differ, before the match (0008-MADR A5.3); and a highlight in the part
// truncation cuts off is not drawn.
func TestHighlightCells(t *testing.T) {
	const s = "👨\u200d👩\u200d👧 ab cd ef"
	mt := fuzzy.NewMatcher("cdef")
	match, ok := mt.Match(s)
	if !ok {
		t.Fatal("no match")
	}
	for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		cs := fuzzy.Cells(s, match.Ranges, m)
		out := styled(s, cs, 100, m, lipgloss.NewStyle(), marked, "...")
		if got := highlighted(out, m); len(cs) != 2 || !slices.Equal(got, cs) {
			t.Errorf("method %d: highlighted cells %v; want %v", m, got, cs)
		}
		if plain := strings.NewReplacer("\x02", "", "\x03", "").Replace(out); plain != s {
			t.Errorf("method %d: the row reads %q", m, plain)
		}
		w := m.StringWidth(s)
		exact := styled(s, cs, w, m, lipgloss.NewStyle(), marked, "...")
		if plain := strings.NewReplacer("\x02", "", "\x03", "").Replace(exact); plain != s {
			t.Errorf("method %d: a row of exactly its %d cells was cut: %q", m, w, plain)
		}
		cutAt := w - 4 // "ef" and the space before it cut off, with room for "..."
		out = styled(s, cs, cutAt, m, lipgloss.NewStyle(), marked, "...")
		plain := strings.NewReplacer("\x02", "", "\x03", "").Replace(out)
		if m.StringWidth(plain) > cutAt || !strings.HasSuffix(plain, "...") {
			t.Errorf("method %d: cut to %d cells: %q, %d cells", m, cutAt, plain, m.StringWidth(plain))
		}
		for _, c := range highlighted(out, m) {
			if c.End > cutAt-3 {
				t.Errorf("method %d: a highlight %v reaches past the cut at %d", m, c, cutAt-3)
			}
		}
	}
}

// blocking is a provider whose recent hits never end: Discover waits for
// its cancellation.
type blocking struct{ stopping }

func (b blocking) Discover(ctx context.Context, _ func(Hit) bool) error {
	<-ctx.Done()
	return ctx.Err()
}

// TestClosedMidSearch: a picker closed while a search runs, by esc and by
// Close while another overlay covers it, cancels the search and stops the
// provider, and leaves no goroutine (0008-MADR A6).
func TestClosedMidSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		slow := blocking{stopping{&fake{name: "slow", search: echo}}}
		g.register(slow)
		g.open()
		if g.p.eng.pending != 1 {
			t.Fatalf("%d searches running; want the discoverer's", g.p.eng.pending)
		}
		g.key("esc")
		if o := g.w.Overlays(); len(o) != 0 {
			t.Fatalf("esc left %v", o)
		}
		if n := slow.stops.Load(); n != 1 {
			t.Fatalf("closed by esc: stopped %d times; want 1", n)
		}

		g.open()
		g.typed("a")
		g.run(g.w.Push(workspace.Overlay{ID: "dlg", Pane: &pane{}, Width: 20, Height: 6, Modal: true}))
		g.settle()
		g.send(workspace.Close("palette"))
		g.settle()
		if o := g.w.Overlays(); !slices.Equal(o, []string{"dlg"}) {
			t.Fatalf("Close(palette) left %v", o)
		}
		if n := slow.stops.Load(); n != 2 {
			t.Fatalf("closed while covered: stopped %d times in all; want 2", n)
		}
	})
	noLeaks(t)
}

// TestOpenWhileOpen: opening an open picker cancels the last opening's
// searches and stops its providers, under another ID too.
func TestOpenWhileOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		slow := blocking{stopping{&fake{name: "slow"}}}
		g.register(slow)
		g.open()
		g.open()
		if n, o := slow.stops.Load(), g.w.Overlays(); n != 1 || !slices.Equal(o, []string{"palette"}) {
			t.Fatalf("opened again: stopped %d, overlays %v; want 1 and the picker", n, o)
		}
		g.open(WithID("other"))
		if n, o := slow.stops.Load(), g.w.Overlays(); n != 2 || !slices.Equal(o, []string{"other"}) {
			t.Fatalf("opened as other: stopped %d, overlays %v; want 2 and other alone", n, o)
		}
		g.send(workspace.Close("other"))
		g.settle()
	})
	noLeaks(t)
}

// TestRegisterRefusesBadWhen: a When that does not parse is an error, and
// the provider is not added.
func TestRegisterRefusesBadWhen(t *testing.T) {
	p := NewPicker(workspace.New(layout.Pane{ID: "main"}, map[layout.PaneID]workspace.Pane{"main": &pane{}}))
	if err := p.Register(&fake{name: "x"}, Availability{When: "a &&"}); err == nil || len(p.eng.sources) != 0 {
		t.Errorf("Register: %v, %d sources; want an error and none", err, len(p.eng.sources))
	}
}

// TestTwoPickers: one picker's batches, broadcast to every overlay, never
// reach another's rows.
func TestTwoPickers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newProg(t, keys(t), nil)
		g.register(newDiscovering("mine", hit("mine", fuzzy.Exact, 1)))
		other := NewPicker(g.w)
		if err := other.Register(newDiscovering("theirs", hit("theirs", fuzzy.Exact, 1)), Availability{}); err != nil {
			t.Fatal(err)
		}
		g.run(other.Open(WithID("other"), WithSize(30, 8)))
		g.settle()
		g.run(g.p.Open(WithSize(30, 8)))
		g.settle()
		if got := g.texts(); !slices.Equal(got, []string{"mine"}) {
			t.Errorf("the top picker's rows %q", got)
		}
		var theirs []string
		for _, r := range other.eng.rows() {
			theirs = append(theirs, r.hit.Text)
		}
		if !slices.Equal(theirs, []string{"theirs"}) {
			t.Errorf("the lower picker's rows %q", theirs)
		}
		g.send(workspace.Close("palette"))
		g.send(workspace.Close("other"))
		g.settle()
	})
}

// failing is a provider whose search fails.
func failing(name string) *fake {
	return &fake{name: name, search: func(context.Context, Search, func(Hit) bool) error { return errors.New("boom") }}
}

// stalled yields hs, then waits for its cancellation, so its search is
// still running when the picker draws.
func stalled(hs ...Hit) func(context.Context, Search, func(Hit) bool) error {
	return func(ctx context.Context, q Search, yield func(Hit) bool) error {
		if err := yields(hs...)(ctx, q, yield); err != nil {
			return err
		}
		<-ctx.Done()
		return ctx.Err()
	}
}

// scene is a golden frame: its providers, and what is typed.
type scene struct {
	name      string
	providers func() []Provider
	query     string
	layers    [][]keymap.Binding
}

// TestPickerGolden: the picker's frames across the matrix {colour, no
// colour} × {UTF-8, ASCII} at 60 and 100 columns, each line within its
// width under the workspace's method: an empty query with recent hits; a
// query with highlights, a truncated highlight, danger badges, a disabled
// row and key hints; a provider's failure; and still searching.
func TestPickerGolden(t *testing.T) {
	scenes := []scene{
		{name: "recents", providers: func() []Provider {
			return []Provider{newDiscovering("recent",
				Hit{ID: "r1", Text: "Open settings", Help: "Edit the program's settings", Score: fuzzy.Score{Value: 3}},
				Hit{ID: "r2", Text: "Toggle sidebar", Score: fuzzy.Score{Value: 2}},
				Hit{ID: "r3", Text: "Focus logs", Score: fuzzy.Score{Value: 1}},
			)}
		}},
		{name: "query", query: "op", layers: [][]keymap.Binding{{user(t, "", "ctrl+r", "file.recent", false)}}, providers: func() []Provider {
			return []Provider{&fake{name: "commands", search: yields(
				Hit{ID: "o1", Text: "Open file", Hint: "ctrl+o", Help: "Open a file by its path", Score: fuzzy.Score{Grade: fuzzy.Prefix, Value: 9}},
				Hit{ID: "o2", Text: "Open recent", Command: "file.recent", Score: fuzzy.Score{Grade: fuzzy.Prefix, Value: 8}},
				Hit{ID: "o3", Text: "Options of the open preferences panel, which is long enough to be cut", Score: fuzzy.Score{Grade: fuzzy.Prefix, Value: 7}},
				Hit{ID: "o4", Text: "Drop open tables", Danger: command.Destructive, Score: fuzzy.Score{Grade: fuzzy.Boundary, Value: 6}},
				Hit{ID: "o5", Text: "Stop open servers", Danger: command.Mutating, Score: fuzzy.Score{Grade: fuzzy.Boundary, Value: 5}},
				Hit{ID: "o6", Text: "Copy path", Disabled: "no file is open", Score: fuzzy.Score{Grade: fuzzy.Subsequence, Value: 4}},
			)}}
		}},
		{name: "error", query: "x", providers: func() []Provider {
			return []Provider{&fake{name: "good", search: yields(hit("x marks", fuzzy.Prefix, 1))}, failing("files")}
		}},
		{name: "searching", query: "lo", providers: func() []Provider {
			return []Provider{&fake{name: "slow", search: stalled(hit("logs", fuzzy.Prefix, 2), hit("lock", fuzzy.Prefix, 1))}}
		}},
	}
	synctest.Test(t, func(t *testing.T) {
		for _, s := range scenes {
			tuitest.Golden(t, "picker-"+s.name, tuitest.Matrix{Widths: []int{60, 100}}, func(c tuitest.Case) string {
				th := theme.New(c.Profile(), theme.Unknown, c.Glyphs())
				g := newProg(t, keys(t, s.layers...), []workspace.Option{workspace.WithTheme(th)})
				g.send(tea.WindowSizeMsg{Width: c.Width + 2, Height: 14})
				for _, p := range s.providers() {
					g.register(p)
				}
				g.open(WithSize(c.Width+2, 12))
				g.typed(s.query)
				time.Sleep(frame) // a search still running is drained once its frame ends
				g.settle()
				out := g.p.View(g.p.width, g.p.height)
				tuitest.Fits(t, out, c.Width, g.w.WidthMethod())
				g.send(workspace.Close("palette"))
				g.settle()
				return out
			})
		}
	})
}
