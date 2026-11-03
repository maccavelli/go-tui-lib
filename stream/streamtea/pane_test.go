package streamtea_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/frame"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inputfilter/inputfiltertea"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/stream/streamtea"
	"github.com/maccavelli/go-tui-lib/v2/theme"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// asciiTheme is the theme the behaviour tests draw with: no colour, ASCII
// glyphs.
var asciiTheme = theme.New(colorprofile.ASCII, theme.Unknown, glyph.ASCII())

// prog plays a program's event loop around a real workspace whose one
// pane, "doc", is a DocPane with no chrome, inside a synctest bubble: a
// command runs on a goroutine of its own, and settle gives each message
// to the workspace until the bubble is idle.
type prog struct {
	w     *workspace.Workspace
	p     *streamtea.DocPane
	d     *stream.Doc
	mu    sync.Mutex
	queue []tea.Msg
	seen  []tea.Msg
}

// keys is a keymap of the workspace's and the pane's defaults, and the
// user's layers.
func keys(t *testing.T, layers ...[]keymap.Binding) *keymap.Keymap {
	t.Helper()
	k, err := keymap.New(keymap.Settings{}, append(workspace.DefaultRules(), streamtea.DefaultRules()...), layers...)
	if err != nil {
		t.Fatalf("keymap.New: %v", err)
	}
	return k
}

// newProg is a width × height workspace over d with no chrome, drawn with
// th, its keys through km.
func newProg(t *testing.T, width, height int, d *stream.Doc, th theme.Theme, km *keymap.Keymap, o ...streamtea.Option) *prog {
	t.Helper()
	return newProgWith(t, width, height, d, th, km, false, o...)
}

// newProgWith is newProg, with the workspace's borders when borders is
// true.
func newProgWith(t *testing.T, width, height int, d *stream.Doc, th theme.Theme, km *keymap.Keymap, borders bool, o ...streamtea.Option) *prog {
	t.Helper()
	g := &prog{d: d}
	chrome := workspace.None
	if borders {
		chrome = workspace.Borders
	}
	g.w = workspace.New(layout.Pane{ID: "doc"}, map[layout.PaneID]workspace.Pane{"doc": placeholder{}},
		workspace.WithTheme(th), workspace.WithChrome(chrome), workspace.WithBindings(km),
		workspace.WithoutBackgroundQuery())
	g.p = streamtea.NewDocPane(g.w, d, o...)
	g.run(g.w.Init())
	g.run(g.w.SetPane("doc", g.p))
	g.send(tea.WindowSizeMsg{Width: width, Height: height})
	g.settle()
	g.w.Render() // as a program draws after each update, so that the mouse finds the pane
	return g
}

// placeholder holds the pane's place until SetPane puts the pane there.
type placeholder struct{}

func (placeholder) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return placeholder{}, nil }
func (placeholder) View(int, int) string                     { return "" }

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

func (g *prog) send(m tea.Msg) {
	g.seen = append(g.seen, m)
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

// named is the keys the tests press, by name.
var named = map[string]tea.KeyPressMsg{
	"up": {Code: tea.KeyUp}, "down": {Code: tea.KeyDown},
	"pgup": {Code: tea.KeyPgUp}, "pgdown": {Code: tea.KeyPgDown},
	"home": {Code: tea.KeyHome}, "end": {Code: tea.KeyEnd},
	"k": {Code: 'k', Text: "k"}, "x": {Code: 'x', Text: "x"},
}

// key presses a named key and settles.
func (g *prog) key(s string) {
	k, ok := named[s]
	if !ok {
		panic("no key " + s)
	}
	g.send(k)
	g.settle()
}

// shown is the frame's rows, each trimmed of the spaces after it.
func (g *prog) shown() []string {
	var out []string
	for l := range strings.SplitSeq(ansi.Strip(g.w.Render()), "\n") {
		out = append(out, strings.TrimRight(l, " "))
	}
	return out
}

// echo is a renderer that draws the source as it is, a row a line.
var echo = stream.RendererFunc(func(src string, _ int) (string, error) { return src, nil })

// numbered is a document of lines "line 1" to "line n", for echo.
func numbered(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	return b.String()
}

// window is the rows "line from" to "line to", with the marks the pane
// draws at its right edge: up on the first row, down on the last.
func window(from, to int, up, down bool) []string {
	var out []string
	for i := from; i <= to; i++ {
		out = append(out, fmt.Sprintf("line %d", i))
	}
	if up {
		out[0] = fmt.Sprintf("%-19s^", out[0])
	}
	if down {
		out[len(out)-1] = fmt.Sprintf("%-19sv", out[len(out)-1])
	}
	return out
}

func equal(a, b []string) bool { return strings.Join(a, "\n") == strings.Join(b, "\n") }

// TestFollow: the pane shows the tail as it grows; scrolled up, it stays
// put while the document grows; back at the end, it follows again.
// WithoutFollow starts at the start and stays there.
func TestFollow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := stream.NewDoc(echo)
		d.Append(numbered(10))
		g := newProg(t, 20, 3, d, asciiTheme, keys(t))
		if got := g.shown(); !equal(got, window(8, 10, true, false)) {
			t.Fatalf("following: %q", got)
		}
		g.key("up")
		if got := g.shown(); !equal(got, window(7, 9, true, true)) {
			t.Fatalf("scrolled up a row: %q", got)
		}
		d.Append("line 11\n")
		if got := g.shown(); !equal(got, window(7, 9, true, true)) {
			t.Fatalf("scrolled up, the document grown: %q; want it where it was", got)
		}
		g.key("down")
		g.key("down")
		if got := g.shown(); !equal(got, window(9, 11, true, false)) {
			t.Fatalf("back at the end: %q", got)
		}
		d.Append("line 12\n")
		if got := g.shown(); !equal(got, window(10, 12, true, false)) {
			t.Fatalf("following again: %q", got)
		}

		d2 := stream.NewDoc(echo)
		d2.Append(numbered(10))
		g2 := newProg(t, 20, 3, d2, asciiTheme, keys(t), streamtea.WithoutFollow())
		if got := g2.shown(); !equal(got, window(1, 3, false, true)) {
			t.Fatalf("WithoutFollow: %q; want the start", got)
		}
		g2.key("end")
		d2.Append("line 11\n")
		if got := g2.shown(); !equal(got, window(8, 10, true, true)) {
			t.Fatalf("WithoutFollow at the end, the document grown: %q; want it where it was", got)
		}
	})
}

// TestChanged: after a draw, a message that changes nothing leaves Changed
// false, as does a scroll down at the end; a new version, a size and a
// scroll make it true.
func TestChanged(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := stream.NewDoc(echo)
		d.Append(numbered(10))
		g := newProg(t, 20, 3, d, asciiTheme, keys(t))
		g.w.Render()
		for _, c := range []struct {
			name string
			do   func()
			want bool
		}{
			{"an unrelated message", func() { g.send(struct{ unrelated int }{}); g.settle() }, false},
			{"a scroll down at the end", func() { g.key("down") }, false},
			{"an unbound key", func() { g.key("x") }, false},
			{"a frame with no Buffer", func() { g.send(frame.DueMsg{At: time.Now()}); g.settle() }, false},
			{"a new version", func() { d.Append("more\n") }, true},
			{"a scroll", func() { g.key("up") }, true},
			{"a size", func() { g.send(tea.WindowSizeMsg{Width: 30, Height: 4}); g.settle() }, true},
		} {
			c.do()
			if got := g.p.Changed(); got != c.want {
				t.Errorf("after %s: Changed %v; want %v", c.name, got, c.want)
			}
			g.w.Render()
			if g.p.Changed() {
				t.Errorf("after %s and a draw: Changed true", c.name)
			}
		}
	})
}

// TestPull: on each frame the pane pulls its Buffer's lines, one a frame
// by the adaptive pacer, all of them by Immediate; once the Buffer is
// closed and drained, the Doc is finished.
func TestPull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		for _, c := range []struct {
			name  string
			o     []streamtea.Option
			after string
		}{
			{"the adaptive pacer", nil, "a\n"},
			{"Immediate", []streamtea.Option{streamtea.WithPacer(frame.Immediate())}, "a\nb\n"},
		} {
			s := frame.NewScheduler()
			b := stream.NewBuffer(s)
			d := stream.NewDoc(echo)
			g := newProg(t, 20, 5, d, asciiTheme, keys(t), append(c.o, streamtea.WithBuffer(b))...)
			b.Write([]byte("a\nb\n"))
			g.send(<-s.Frames())
			g.settle()
			if d.Source() != c.after {
				t.Errorf("%s: after one frame the source is %q; want %q", c.name, d.Source(), c.after)
			}
			b.Close()
			for !d.Finished() {
				select {
				case m := <-s.Frames():
					g.send(m)
					g.settle()
				case <-time.After(5 * time.Second):
					t.Fatalf("%s: no frame in 5 s: source %q, not finished", c.name, d.Source())
				}
			}
			if got := g.shown(); !equal(got[:3], []string{"a", "b", ""}) {
				t.Errorf("%s: finished, drawn %q", c.name, got)
			}
			s.Close()
		}
	})
}

// TestWheel: a wheel notch scrolls a row, and a coalesced wheel message
// its notches, both through the workspace.
func TestWheel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := stream.NewDoc(echo)
		d.Append(numbered(10))
		g := newProg(t, 20, 3, d, asciiTheme, keys(t))
		g.send(tea.MouseWheelMsg{X: 1, Y: 1, Button: tea.MouseWheelUp})
		g.settle()
		if got := g.shown(); !equal(got, window(7, 9, true, true)) {
			t.Fatalf("a notch up: %q", got)
		}
		g.send(inputfiltertea.CoalescedWheelMsg{Pointer: tea.Mouse{X: 1, Y: 1, Button: tea.MouseWheelUp}, DY: -4})
		g.settle()
		if got := g.shown(); !equal(got, window(3, 5, true, true)) {
			t.Fatalf("four notches up, coalesced: %q", got)
		}
		g.send(tea.MouseWheelMsg{X: 1, Y: 1, Button: tea.MouseWheelDown})
		g.settle()
		if got := g.shown(); !equal(got, window(4, 6, true, true)) {
			t.Fatalf("a notch down: %q", got)
		}
	})
}

// TestCommands: each key of DefaultRules moves as its command is named,
// and a key the user binds to one does too; ConsumesCommand takes the
// pane's six commands and no other.
func TestCommands(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q, err := keymap.ParseSequence("k", nil)
		if err != nil {
			t.Fatal(err)
		}
		user := []keymap.Binding{{Keys: q, Command: streamtea.ScrollUpID, Scope: streamtea.DocPaneScope,
			Location: keymap.Location{Layer: keymap.UserLayer}}}
		d := stream.NewDoc(echo)
		d.Append(numbered(20))
		g := newProg(t, 20, 4, d, asciiTheme, keys(t, user))
		for _, c := range []struct {
			key      string
			from, to int
		}{
			{"up", 16, 19},
			{"k", 15, 18},
			{"pgup", 11, 14},
			{"home", 1, 4},
			{"down", 2, 5},
			{"pgdown", 6, 9},
			{"end", 17, 20},
		} {
			g.key(c.key)
			if got := g.shown(); !equal(got, window(c.from, c.to, c.from > 1, c.to < 20)) {
				t.Errorf("after %s: %q; want lines %d to %d", c.key, got, c.from, c.to)
			}
		}
		for _, id := range []command.ID{streamtea.ScrollUpID, streamtea.ScrollDownID, streamtea.PageUpID,
			streamtea.PageDownID, streamtea.TopID, streamtea.BottomID} {
			if !g.p.ConsumesCommand(id) {
				t.Errorf("ConsumesCommand(%s) false", id)
			}
		}
		if g.p.ConsumesCommand("palette.open") {
			t.Error("ConsumesCommand(palette.open) true")
		}
		if ks := g.p.KeyContexts(); len(ks) != 1 || ks[0] != "kind:docpane" {
			t.Errorf("KeyContexts %v", ks)
		}
	})
}

// TestTitle: WithTitle's title, and none without it.
func TestTitle(t *testing.T) {
	w := workspace.New(layout.Pane{ID: "doc"}, map[layout.PaneID]workspace.Pane{"doc": placeholder{}})
	if got := streamtea.NewDocPane(w, stream.NewDoc(nil), streamtea.WithTitle("Answer")).Title(); got != "Answer" {
		t.Errorf("Title %q", got)
	}
	if got := streamtea.NewDocPane(w, stream.NewDoc(nil)).Title(); got != "" {
		t.Errorf("Title %q; want none", got)
	}
}

// wide is a renderer whose one row is wider than any pane here.
var wide = stream.RendererFunc(func(string, int) (string, error) { return strings.Repeat("x", 40), nil })

// TestTheme: a row too wide is cut with the theme's Ellipsis; a new theme
// draws its own, and of two literal themes, both of ID 0, each draws its
// own, never the other's from the cache.
func TestTheme(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := stream.NewDoc(wide)
		d.Append("x\n")
		uni := theme.New(colorprofile.ASCII, theme.Unknown, glyph.Unicode())
		g := newProg(t, 10, 2, d, uni, keys(t))
		row := func() string { return g.shown()[0] }
		if got := row(); got != "xxxxxxxx…" {
			t.Fatalf("the Unicode theme: %q", got)
		}
		g.w.SetTheme(asciiTheme)
		if got := row(); got != "xxxxxxxx~" {
			t.Fatalf("a new theme, the ASCII one: %q", got)
		}
		for _, e := range []string{"!", "?"} {
			lit := theme.Theme{Glyphs: glyph.ASCII()}
			lit.Glyphs.Ellipsis = e
			g.w.SetTheme(lit)
			if got := row(); got != "xxxxxxxx"+e {
				t.Fatalf("a literal theme with Ellipsis %q: %q", e, got)
			}
		}
	})
}

// TestMethod: rows are fitted with the workspace's width method, so when
// the terminal reports grapheme clustering, an emoji with a skin tone,
// four cells by wcwidth and two as a grapheme, is padded again.
func TestMethod(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := stream.NewDoc(nil)
		d.Append("👍🏽\n")
		g := newProg(t, 10, 1, d, asciiTheme, keys(t))
		for _, c := range []struct {
			m    ansi.Method
			pads int
		}{{ansi.WcWidth, 5}, {ansi.GraphemeWidth, 7}} {
			if c.m == ansi.GraphemeWidth {
				g.send(tea.ModeReportMsg{Mode: ansi.ModeUnicodeCore, Value: ansi.ModeSet})
				g.settle()
			}
			v := g.p.View(10, 1)
			if want := "👍🏽" + strings.Repeat(" ", c.pads+1); v != want {
				t.Errorf("under %v: %q; want %d spaces of padding and the marks' column", c.m, v, c.pads)
			}
		}
	})
}
