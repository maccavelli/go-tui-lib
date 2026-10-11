package keymap

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/tuitest"
)

// fires reports whether the Dispatcher runs id for msg along p.
func fires(k *Keymap, p Path, msg tea.Msg, id command.ID) bool {
	o, _ := NewDispatcher(k).Update(msg, p, nil)
	return o.Command == id
}

// TestKeyBindingMatches: over Step 2's table of keys as ultraviolet
// decodes them, a one-stroke binding's KeyBinding matches with key.Matches
// as BubblesKeys does, so exactly when the Dispatcher runs the command
// wherever the table has Matches and key.Matches agree. A release or
// repeat stroke, which a key.Binding cannot tell from a press, is never
// matched.
func TestKeyBindingMatches(t *testing.T) {
	for _, c := range keyCases {
		msg := tea.KeyPressMsg(c.key)
		for text, want := range c.strokes {
			k := mustNew(t, Settings{Features: allFeatures, Facts: Releases}, nil, []Binding{bind(t, ProgramLayer, command.Global, text, "cmd.x")})
			kb := k.KeyBinding("cmd.x", global, nil)
			if got := fires(k, global, msg, "cmd.x"); got != want[0] {
				t.Errorf("%s: the dispatcher runs %q: %v, want %v", c.name, text, got, want[0])
			}
			wantKB := want[1] && mustStroke(t, text).Event == Press
			if got := key.Matches(msg, kb); got != wantKB {
				t.Errorf("%s: key.Matches(%q's KeyBinding %q) = %v, want %v", c.name, text, kb.Keys(), got, wantKB)
			}
		}
	}
	k := mustNew(t, Settings{}, nil, []Binding{bind(t, ProgramLayer, command.Global, "shift+a", "cmd.x")})
	if got := k.KeyBinding("cmd.x", global, nil).Keys(); !slices.Equal(got, []string{"shift+a", "A"}) {
		t.Errorf("shift+a's keys are %q; want both spellings", got)
	}
}

// TestKeyBindingScopeDecides: a binding an inner scope hides, as the
// dispatcher reads the path (amendment A6), gives a disabled KeyBinding.
func TestKeyBindingScopeDecides(t *testing.T) {
	p := Path{command.Global, "pane:a"}
	k := mustNew(t, Settings{}, nil, []Binding{
		bind(t, ProgramLayer, command.Global, "g", "cmd.g"),
		bind(t, ProgramLayer, "pane:a", "g g", "cmd.top"),
		bind(t, ProgramLayer, command.Global, "ctrl+k", "cmd.outer"),
		bind(t, ProgramLayer, "pane:a", "ctrl+k", "cmd.inner"),
	})
	for _, c := range []struct {
		id   command.ID
		keys string
		on   bool
	}{{"cmd.g", "g", false}, {"cmd.outer", "ctrl+k", false}, {"cmd.inner", "ctrl+k", true}} {
		kb := k.KeyBinding(c.id, p, nil)
		msg := press(t, c.keys)
		if kb.Enabled() != c.on || key.Matches(msg, kb) != c.on || fires(k, p, msg, c.id) != c.on {
			t.Errorf("%s along %v: enabled %v, key.Matches %v, the dispatcher %v; want %v",
				c.id, p, kb.Enabled(), key.Matches(msg, kb), fires(k, p, msg, c.id), c.on)
		}
	}
}

// TestKeyBindingChord: a chord, and a release stroke, are enabled and
// listed as their full text, which help shows and key.Matches never
// matches; the help key is the primary sequence.
func TestKeyBindingChord(t *testing.T) {
	k := mustNew(t, Settings{Features: allFeatures, Facts: Releases}, nil, []Binding{
		bind(t, ProgramLayer, command.Global, "ctrl+s", "cmd.save"),
		bind(t, ProgramLayer, command.Global, "ctrl+x s", "cmd.save"),
		bind(t, ProgramLayer, command.Global, "ctrl+x n", "cmd.new"),
		bind(t, ProgramLayer, command.Global, "a:release", "cmd.talk"),
	})
	nw := k.KeyBinding("cmd.new", global, nil)
	if !nw.Enabled() || !slices.Equal(nw.Keys(), []string{"ctrl+x n"}) || nw.Help().Key != "ctrl+x n" {
		t.Errorf("the chord: enabled %v, keys %q, help %q", nw.Enabled(), nw.Keys(), nw.Help().Key)
	}
	for _, s := range []string{"ctrl+x", "n", "x", "space"} {
		if key.Matches(press(t, s), nw) {
			t.Errorf("the chord matches %s", s)
		}
	}
	talk := k.KeyBinding("cmd.talk", global, nil)
	if !slices.Equal(talk.Keys(), []string{"a:release"}) || key.Matches(press(t, "a"), talk) || key.Matches(release(t, "a"), talk) {
		t.Errorf("the release stroke: keys %q", talk.Keys())
	}
	save := k.KeyBinding("cmd.save", global, nil)
	if want := k.Bindings("cmd.save", global); save.Help().Key != want[0].String() ||
		!slices.Equal(save.Keys(), []string{want[0].String(), want[1].String()}) {
		t.Errorf("two sequences: keys %q, help %q; Bindings %v", save.Keys(), save.Help().Key, want)
	}
	if !key.Matches(press(t, "ctrl+s"), save) || key.Matches(press(t, "ctrl+x"), save) {
		t.Error("the one-stroke sequence beside a chord does not match as it should")
	}
	shown := help.New().ShortHelpView([]key.Binding{nw})
	if !strings.Contains(shown, "ctrl+x n") {
		t.Errorf("help shows %q, not the chord", shown)
	}
}

// TestKeyBindingDisabled: an unbound command, and one the user's file
// removes, give a disabled binding, which help skips.
func TestKeyBindingDisabled(t *testing.T) {
	k := mustNew(t, Settings{},
		defaults(bind(t, DefaultLayer, command.Global, "q", "cmd.quit"), bind(t, DefaultLayer, command.Global, "ctrl+s", "cmd.save")),
		[]Binding{{Command: "cmd.quit", Remove: true, Location: Location{Layer: UserLayer}}})
	for _, id := range []command.ID{"cmd.none", "cmd.quit"} {
		if kb := k.KeyBinding(id, global, nil); kb.Enabled() || key.Matches(press(t, "q"), kb) {
			t.Errorf("%s: enabled %v, keys %q", id, kb.Enabled(), kb.Keys())
		}
	}
	h := k.Help(global, nil, "cmd.none", "cmd.quit", "cmd.save")
	if got := help.New().ShortHelpView(h.ShortHelp()); got != help.New().ShortHelpView([]key.Binding{k.KeyBinding("cmd.save", global, nil)}) {
		t.Errorf("the short help is %q; want cmd.save's alone", got)
	}
	if full := h.FullHelp(); len(full) != 1 || len(full[0]) != 1 || full[0][0].Help().Desc != "cmd.save" {
		t.Errorf("the full help is %v; want cmd.save alone", full)
	}
}

// TestDescriber: the help's description is the Describer's title, or the
// ID when it has none or there is no Describer; FromRegistry gives a
// registered command's title.
func TestDescriber(t *testing.T) {
	k := mustNew(t, Settings{}, nil, []Binding{
		bind(t, ProgramLayer, command.Global, "ctrl+s", "app.save"),
		bind(t, ProgramLayer, command.Global, "ctrl+q", "app.quit"),
	})
	d := DescriberFunc(func(id command.ID) (string, bool) { return "Save the file", id == "app.save" })
	for _, c := range []struct {
		id   command.ID
		d    Describer
		want string
	}{{"app.save", d, "Save the file"}, {"app.quit", d, "app.quit"}, {"app.save", nil, "app.save"}} {
		if got := k.KeyBinding(c.id, global, c.d).Help().Desc; got != c.want {
			t.Errorf("%s: %q; want %q", c.id, got, c.want)
		}
	}
	r := command.NewRegistry()
	run := func(context.Context, *command.Invocation, command.NoArgs) (command.Result, error) {
		return command.Result{}, nil
	}
	if err := r.Register(command.MustNew("app.save", "Save file", run, command.WithDanger(command.UI)),
		command.MustNew("app.untitled", "", run, command.WithDanger(command.UI))); err != nil {
		t.Fatal(err)
	}
	reg := FromRegistry(r)
	if got, ok := reg.Describe("app.save"); !ok || got != "Save file" {
		t.Errorf("FromRegistry: %q, %v", got, ok)
	}
	for _, id := range []command.ID{"app.none", "app.untitled"} {
		if got, ok := reg.Describe(id); ok {
			t.Errorf("FromRegistry describes %s: %q", id, got)
		}
	}
	if got := k.KeyBinding("app.save", global, reg).Help().Desc; got != "Save file" {
		t.Errorf("the help through FromRegistry is %q", got)
	}
}

// helpKeymap is a keymap over a global, a workspace and a pane scope, and
// the path through them, for the help's tests.
func helpKeymap(t *testing.T) (*Keymap, Path) {
	t.Helper()
	return mustNew(t, Settings{}, nil, []Binding{
		bind(t, ProgramLayer, command.Global, "ctrl+q", "app.quit"),
		bind(t, ProgramLayer, command.Global, "?", "app.help"),
		bind(t, ProgramLayer, command.Global, "ctrl+shift+s", "app.save"),
		bind(t, ProgramLayer, "workspace", "alt+.", "workspace.focus.next"),
		bind(t, ProgramLayer, "workspace", "alt+z", "workspace.zoom"),
		bind(t, ProgramLayer, "pane:a", "ctrl+s", "app.save"),
		bind(t, ProgramLayer, "pane:a", "ctrl+x n", "app.new"),
	}), Path{command.Global, "workspace", "pane:a"}
}

// columns is a full help's help keys, column by column.
func columns(full [][]key.Binding) string {
	var cols []string
	for _, c := range full {
		var keys []string
		for _, b := range c {
			keys = append(keys, b.Help().Key)
		}
		cols = append(cols, strings.Join(keys, ","))
	}
	return strings.Join(cols, " | ")
}

// TestHelpColumns: the short help is the IDs given, in order; the full
// help lists them, each once, in its primary sequence's scope's column,
// innermost scope first; with no IDs, every command along the path.
func TestHelpColumns(t *testing.T) {
	k, p := helpKeymap(t)
	h := k.Help(p, nil, "workspace.zoom", "app.quit", "app.save", "app.none")
	var short []string
	for _, b := range h.ShortHelp() {
		short = append(short, fmt.Sprintf("%s:%v", b.Help().Key, b.Enabled()))
	}
	if want := []string{"alt+z:true", "ctrl+q:true", "ctrl+s:true", ":false"}; !slices.Equal(short, want) {
		t.Errorf("the short help is %q; want %q", short, want)
	}
	if got, want := columns(h.FullHelp()), "ctrl+s | alt+z | ctrl+q"; got != want {
		t.Errorf("the full help is %q; want %q", got, want)
	}
	all := k.Help(p, nil)
	// Each scope's commands in resolution order, the later binding first;
	// app.save once, in pane:a's column, where its primary sequence is.
	if got, want := columns(all.FullHelp()), "ctrl+x n,ctrl+s | alt+z,alt+. | ?,ctrl+q"; got != want {
		t.Errorf("the full help of every command is %q; want %q", got, want)
	}
	if got, want := len(all.ShortHelp()), 6; got != want {
		t.Errorf("the short help of every command has %d bindings; want %d", got, want)
	}
	twice := k.Help(Path{command.Global, command.Global}, nil)
	if got, want := columns(twice.FullHelp()), "ctrl+shift+s,?,ctrl+q"; got != want {
		t.Errorf("a path naming global twice: %q; want %q", got, want)
	}
	h.ShortHelp()[0] = key.Binding{}
	h.FullHelp()[0][0] = key.Binding{}
	if columns(h.FullHelp()) != "ctrl+s | alt+z | ctrl+q" || h.ShortHelp()[0].Help().Key != "alt+z" {
		t.Error("changing what ShortHelp or FullHelp returned changed the help")
	}
}

// helpModel is a bubbles help model drawn with a case's glyphs and a
// theme's styles, as workspace.Help draws it.
func helpModel(c tuitest.Case) help.Model {
	th := theme.New(c.Profile(), theme.Dark, c.Glyphs())
	g, st := th.Glyphs, th.Styles
	m := help.New()
	m.ShortSeparator = " " + g.Bullet + " "
	m.Ellipsis = g.Ellipsis
	m.Styles = help.Styles{
		Ellipsis: st.Muted, ShortKey: st.Body, ShortDesc: st.Muted, ShortSeparator: st.Muted,
		FullKey: st.Body, FullDesc: st.Muted, FullSeparator: st.Muted,
	}
	m.SetWidth(c.Width)
	return m
}

// TestHelpGolden: the short and full help through help.Model, across the
// matrix of 0001-MADR §6; at 40 cells the short help ends in the
// ellipsis.
func TestHelpGolden(t *testing.T) {
	k, p := helpKeymap(t)
	d := DescriberFunc(func(id command.ID) (string, bool) {
		title, ok := map[command.ID]string{
			"app.quit": "quit", "app.help": "help", "app.save": "save", "app.new": "new file",
			"workspace.focus.next": "next pane", "workspace.zoom": "zoom",
		}[id]
		return title, ok
	})
	h := k.Help(p, d)
	m := tuitest.Matrix{Widths: []int{40, 80}}
	tuitest.Golden(t, "help-short", m, func(c tuitest.Case) string { return helpModel(c).ShortHelpView(h.ShortHelp()) })
	tuitest.Golden(t, "help-full", m, func(c tuitest.Case) string { return helpModel(c).FullHelpView(h.FullHelp()) })
}
