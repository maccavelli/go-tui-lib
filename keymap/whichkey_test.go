package keymap

import (
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// whichKeys is a keymap over a global and a pane scope, for ActiveKeys
// and Explain.
func whichKeys(t *testing.T) []Binding {
	zoom := bind(t, ProgramLayer, "pane:a", "ctrl+x z", "zoom")
	zoom.When = "x"
	return []Binding{
		bind(t, ProgramLayer, command.Global, "ctrl+x n", "new"),
		bind(t, ProgramLayer, command.Global, "ctrl+x s", "save"),
		bind(t, ProgramLayer, command.Global, "ctrl+q", "quit"),
		bind(t, ProgramLayer, command.Global, "g", "outer.g"),
		bind(t, ProgramLayer, command.Global, "ctrl+x k k", "killall"),
		bind(t, ProgramLayer, "pane:a", "g g", "top"),
		bind(t, ProgramLayer, "pane:a", "ctrl+x s", "pane.save"),
		zoom,
	}
}

// active is ActiveKeys, written "label:command" or "label…".
func active(d *Dispatcher, p Path, ctx when.Context) string {
	var out []string
	for _, a := range d.ActiveKeys(p, ctx, "linux", glyph.ASCII()) {
		if a.Continues {
			out = append(out, a.Label+"…")
		} else {
			out = append(out, a.Label+":"+string(a.Command))
		}
	}
	return strings.Join(out, " ")
}

// TestActiveKeys: with nothing pending, the path's first strokes; after a
// prefix, every completing and continuing stroke, each decided by the
// innermost scope that has it, and a when that fails hides one; labels
// are Label's; and the dispatcher is unchanged.
func TestActiveKeys(t *testing.T) {
	p := Path{command.Global, "pane:a"}
	d, _ := dispatcher(t, Settings{}, whichKeys(t)...)
	if got, want := active(d.Dispatcher, p, nil), "ctrl+x… g… ctrl+q:quit"; got != want {
		t.Errorf("first strokes: %q; want %q", got, want)
	}
	d.Update(press(t, "ctrl+x"), p, nil)
	before := d.Pending()
	if got, want := active(d.Dispatcher, p, nil), "s:pane.save k… n:new"; got != want {
		t.Errorf("after ctrl+x: %q; want %q", got, want)
	}
	if got, want := active(d.Dispatcher, p, when.Map{"x": when.BoolValue(true)}), "z:zoom s:pane.save k… n:new"; got != want {
		t.Errorf("after ctrl+x, with x: %q; want %q", got, want)
	}
	if got := d.ActiveKeys(p, nil, "darwin", glyph.Unicode()); len(got) == 0 || got[0].Label != "s" || got[0].Stroke.String() != "s" {
		t.Errorf("the first active key is %+v", got)
	}
	if !slices.Equal(d.Pending(), before) {
		t.Errorf("ActiveKeys changed the pending sequence: %v, was %v", d.Pending(), before)
	}
	if got, want := active(d.Dispatcher, Path{command.Global}, nil), "ctrl+x… g:outer.g ctrl+q:quit"; got != want {
		t.Errorf("on another path, which a key would cancel: %q; want %q", got, want)
	}
	first, _ := d.Update(press(t, "s"), p, nil)
	if first.Command != "pane.save" {
		t.Errorf("s after ctrl+x ran %q; ActiveKeys said pane.save", first.Command)
	}
	if got := NewDispatcher(nil).ActiveKeys(p, nil, "linux", glyph.ASCII()); got != nil {
		t.Errorf("with no keymap: %v", got)
	}
}

// TestActiveKeysPrefix: of a key that completes one binding and continues
// another in one scope, the one it completes wins, as Update decides.
func TestActiveKeysPrefix(t *testing.T) {
	// "h j" is bound later, so it comes first in resolution order.
	k, _ := New(Settings{}, nil, []Binding{
		bind(t, ProgramLayer, command.Global, "h", "help"),
		bind(t, ProgramLayer, command.Global, "h j", "jump"),
	})
	if got, want := active(NewDispatcher(k), global, nil), "h:help"; got != want {
		t.Errorf("a Prefix pair: %q; want %q", got, want)
	}
}

// TestEditPending: while pending, esc clears the sequence and is not
// replayed, and backspace removes one stroke and leaves the rest pending
// with a new timer; a binding that takes either wins; with both fields
// set, each ends the sequence as MADR §6 says.
func TestEditPending(t *testing.T) {
	bs := append(whichKeys(t), bind(t, ProgramLayer, command.Global, "ctrl+x esc", "abort"))
	p := Path{command.Global, "pane:a"}
	d, c := dispatcher(t, Settings{}, bs...)
	d.Update(press(t, "ctrl+x"), p, nil)
	d.Update(press(t, "k"), p, nil)
	ticks := len(c.timers)
	o, cmd := d.Update(press(t, "backspace"), p, nil)
	if !o.Handled || len(o.Replay) != 0 || d.Pending().String() != "ctrl+x" || cmd == nil || len(c.timers) != ticks+1 {
		t.Errorf("backspace after ctrl+x k: %+v, pending %v, %d timers; want ctrl+x pending and a new timer", o, d.Pending(), len(c.timers)-ticks)
	}
	if o, _ := d.Update(press(t, "s"), p, nil); o.Command != "pane.save" {
		t.Errorf("s after the backspace: %+v; want pane.save", o)
	}
	d.Update(press(t, "ctrl+x"), p, nil)
	if o, _ := d.Update(press(t, "backspace"), p, nil); !o.Handled || len(d.Pending()) != 0 {
		t.Errorf("backspace of the last stroke: %+v, pending %v", o, d.Pending())
	}
	d.Update(press(t, "g"), p, nil)
	if o, _ := d.Update(press(t, "esc"), p, nil); !o.Handled || len(o.Replay) != 0 || len(d.Pending()) != 0 {
		t.Errorf("esc after g: %+v, pending %v; want cleared, nothing replayed", o, d.Pending())
	}
	if o, _ := d.Update(press(t, "esc"), p, nil); o.Handled {
		t.Errorf("esc with nothing pending: %+v; want the pane's", o)
	}
	d.Update(press(t, "g"), p, nil)
	if o, _ := d.Update(press(t, "alt+esc"), p, nil); o.Handled || replayed(o) != "[g]" {
		t.Errorf("alt+esc after g: %+v; want g replayed, as an unmatched key", o)
	}
	d.Update(press(t, "ctrl+x"), p, nil)
	if o, _ := d.Update(press(t, "esc"), p, nil); o.Command != "abort" {
		t.Errorf("esc after ctrl+x, which ctrl+x esc binds: %+v; want abort", o)
	}
	d, _ = dispatcher(t, Settings{EscReplays: true, BackspaceReplays: true}, bs...)
	for _, key := range []string{"esc", "backspace"} {
		d.Update(press(t, "g"), p, nil)
		o, _ := d.Update(press(t, key), p, nil)
		if o.Handled || replayed(o) != "[g]" || len(d.Pending()) != 0 {
			t.Errorf("%s after g, with the fields set: %+v; want g replayed and %s the pane's", key, o, key)
		}
	}
}

// names is bindings' commands.
func names(bs []Binding) string {
	var out []string
	for _, b := range bs {
		out = append(out, string(b.Command))
	}
	return strings.Join(out, " ")
}

// TestExplain: the raw and normalised key, the winner first, and the
// shadowed bindings, for a planted keymap with a Shadow; the pending
// sequence is read first; and the dispatcher is unchanged.
func TestExplain(t *testing.T) {
	bs := []Binding{
		bind(t, ProgramLayer, command.Global, "ctrl+k", "global.k"),
		bind(t, ProgramLayer, "pane:a", "ctrl+k", "pane.k"),
		bind(t, ProgramLayer, command.Global, "ctrl+j", "global.j"),
		bind(t, ProgramLayer, "pane:a", "ctrl+j x", "pane.jx"),
		bind(t, ProgramLayer, "pane:a", "ctrl+j y", "pane.jy"),
		bind(t, ProgramLayer, command.Global, "ctrl+x s", "save"),
		bind(t, ProgramLayer, command.Global, "ctrl+]", "bracket"),
		bind(t, ProgramLayer, command.Global, "@", "at"),
	}
	user := []Binding{bind(t, UserLayer, command.Global, "ctrl+k", "user.k")}
	k, err := New(Settings{GOOS: "windows"}, defaults(bs...), user)
	if err != nil || len(k.Conflicts()) == 0 {
		t.Fatalf("New: %v, %v; want a Shadow", err, k.Conflicts())
	}
	d := NewDispatcher(k)
	pane := Path{command.Global, "pane:a"}
	for _, c := range []struct {
		key               string
		p                 Path
		matches, shadowed string
	}{
		{"ctrl+k", pane, "pane.k", "user.k global.k"},
		{"ctrl+k", global, "user.k", "global.k"},
		{"ctrl+j", pane, "pane.jy pane.jx", "global.j"},
	} {
		k := press(t, c.key)
		x := d.Explain(k, c.p, nil)
		if x.Raw != k || x.Changed || x.Stroke.String() != c.key ||
			names(x.Matches) != c.matches || names(x.Shadowed) != c.shadowed {
			t.Errorf("%s along %v: %+v; want matches %q, shadowed %q", c.key, c.p, x, c.matches, c.shadowed)
		}
	}
	// A Prefix pair, which New reports and keeps: the binding the key
	// completes wins, though the other comes first in resolution order.
	pk, _ := New(Settings{}, nil, []Binding{
		bind(t, ProgramLayer, command.Global, "h", "help"),
		bind(t, ProgramLayer, command.Global, "h j", "jump"),
	})
	if x := NewDispatcher(pk).Explain(press(t, "h"), global, nil); names(x.Matches) != "help jump" {
		t.Errorf("h, of a Prefix pair: %+v; want help first", x)
	}
	x := d.Explain(termkey.KeyEvent{Code: '5', Mod: termkey.ModCtrl}, global, nil)
	if !x.Changed || x.Normalized.Code != ']' || x.Stroke.String() != "ctrl+]" || names(x.Matches) != "bracket" {
		t.Errorf("legacy ctrl+5: %+v; want ctrl+], bracket", x)
	}
	altGr := termkey.KeyEvent{Code: '@', BaseCode: 'q', Text: "@", Mod: termkey.ModCtrl | termkey.ModAlt}
	if x := d.Explain(altGr, global, nil); x.Matches != nil || !x.Changed || x.Stroke.String() != "@" {
		t.Errorf("AltGr on Windows: %+v; want text, meeting nothing", x)
	}
	d.Key(press(t, "ctrl+x"), global, nil)
	before := d.Pending()
	if x := d.Explain(termkey.KeyEvent{Code: 's'}, global, nil); names(x.Matches) != "save" {
		t.Errorf("s after ctrl+x: %+v; want save", x)
	}
	if !slices.Equal(d.Pending(), before) {
		t.Errorf("Explain changed the pending sequence: %v, was %v", d.Pending(), before)
	}
}
