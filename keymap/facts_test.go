package keymap

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/termcap"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// newline is a newline default: shift+enter, else alt+enter where it
// arrives, else ctrl+j (MADR amendment A1).
func newline(t *testing.T) Default {
	return Default{
		Binding: bind(t, DefaultLayer, command.Global, "shift+enter", "newline"),
		Alternatives: []Alternative{
			{Keys: seq(t, "alt+enter"), Facts: AltEnter},
			{Keys: seq(t, "ctrl+j")},
		},
	}
}

// terminals are fact sets planted from 0003-REPORT §9 and amendment A9.
var terminals = []struct {
	name     string
	features Features
	facts    KeyFacts
	newline  string
}{
	{"kitty, with the protocol", Disambiguate | EventTypes, ShiftEnter | AltEnter | CtrlEnter | Releases, "shift+enter"},
	{"VTE before 8200: shift+enter arrives as CR", 0, AltEnter, "alt+enter"},
	{"tmux before 3.3", 0, AltEnter, "alt+enter"},
	{"a VS Code-family host", 0, AltEnter | HostTakesCtrlQ | HostTakesCtrlI, "alt+enter"},
	{"Apple Terminal: Option types characters", 0, 0, "ctrl+j"},
}

// TestAlternatives: the newline default binds each of its three keys under
// the fact sets of the terminals that deliver them, reporting nothing.
func TestAlternatives(t *testing.T) {
	for _, c := range terminals {
		k, err := New(Settings{Features: c.features, Facts: c.facts}, []Default{newline(t)})
		if err != nil || len(k.Conflicts()) > 0 {
			t.Errorf("%s: New: %v, %v", c.name, err, k.Conflicts())
		}
		if got := fmt.Sprint(k.Bindings("newline", global)); got != "["+c.newline+"]" {
			t.Errorf("%s: newline is bound to %s; want %s", c.name, got, c.newline)
		}
	}
}

// TestNoAlternative: a default none of whose keys can be delivered binds
// nothing and reports nothing.
func TestNoAlternative(t *testing.T) {
	d := newline(t)
	d.Alternatives = d.Alternatives[:1]
	k, err := New(Settings{}, []Default{d})
	if err != nil || len(k.Conflicts()) > 0 || len(k.Bindings("newline", global)) != 0 {
		t.Errorf("New: %v, %v, bound %v; want nothing", err, k.Conflicts(), k.Bindings("newline", global))
	}
}

// TestHostKeys: a host's taking ctrl+q bars a default's ctrl+q, which
// binds its alternative, but not a user's own ctrl+q.
func TestHostKeys(t *testing.T) {
	quit := Default{
		Binding:      bind(t, DefaultLayer, command.Global, "ctrl+q", "quit"),
		Alternatives: []Alternative{{Keys: seq(t, "ctrl+x ctrl+c")}},
	}
	user := []Binding{bind(t, UserLayer, "pane:a", "ctrl+q", "close")}
	for facts, want := range map[KeyFacts]string{0: "ctrl+q", HostTakesCtrlQ: "ctrl+x ctrl+c", HostTakesCtrlI: "ctrl+q"} {
		k, err := New(Settings{Facts: facts}, []Default{quit}, user)
		if err != nil {
			t.Fatalf("%s: %v", facts, err)
		}
		if got := k.Bindings("quit", global); len(got) != 1 || got[0].String() != want {
			t.Errorf("with %s, quit is bound to %v; want %s", facts, got, want)
		}
		if got := k.Bindings("close", Path{"pane:a"}); len(got) != 1 {
			t.Errorf("with %s, the user's ctrl+q is not bound", facts)
		}
	}
}

// TestReleaseRule: a release default binds its alternative until releases
// are reported; a user's release binding without the fact is NotSupported,
// naming it, and binds after Rebuild.
func TestReleaseRule(t *testing.T) {
	talk := Default{
		Binding:      bind(t, DefaultLayer, command.Global, "space:release", "talk"),
		Alternatives: []Alternative{{Keys: seq(t, "space")}},
	}
	user := []Binding{bind(t, UserLayer, command.Global, "a:release", "shout")}
	k, err := New(Settings{Features: EventTypes}, []Default{talk}, user)
	var c BindingConflict
	if !errors.As(err, &c) || c.Kind != NotSupported || !strings.Contains(err.Error(), "needs event-types and releases") {
		t.Fatalf("without Releases: %v; want a:release NotSupported, naming the fact", err)
	}
	if got := fmt.Sprint(k.Bindings("talk", global), k.Bindings("shout", global)); got != "[space] []" {
		t.Errorf("without Releases: talk and shout are bound to %s", got)
	}
	k2, err := k.Rebuild(EventTypes, Releases)
	if err != nil {
		t.Fatalf("Rebuild with Releases: %v", err)
	}
	if got := fmt.Sprint(k2.Bindings("talk", global), k2.Bindings("shout", global)); got != "[space:release] [a:release]" {
		t.Errorf("with Releases: talk and shout are bound to %s", got)
	}
}

// TestRebuildFacts: Rebuild with new facts changes the binding, and the
// help and the shortcut follow.
func TestRebuildFacts(t *testing.T) {
	k := mustNew(t, Settings{Facts: AltEnter}, []Default{newline(t)})
	key := func(k *Keymap) string { return k.Help(global, nil, "newline").ShortHelp()[0].Help().Key }
	if got := key(k); got != "alt+enter" {
		t.Errorf("the help shows %q; want alt+enter", got)
	}
	k2, err := k.Rebuild(Disambiguate, ShiftEnter|AltEnter)
	if err != nil {
		t.Fatal(err)
	}
	if got := key(k2); got != "shift+enter" {
		t.Errorf("after Rebuild, the help shows %q; want shift+enter", got)
	}
	if got := k2.Shortcut("newline", global, "darwin", glyph.Unicode()); got != "⇧enter" {
		t.Errorf("after Rebuild, the shortcut is %q", got)
	}
	k3, _ := k2.Rebuild(0, 0)
	if got := key(k3); got != "ctrl+j" {
		t.Errorf("after a Rebuild with no facts, the help shows %q; want ctrl+j", got)
	}
}

// TestReachable: commands bound only off the path, behind a when that
// fails, or hidden by an inner scope's binding whose when holds are not
// reachable; each command comes once.
func TestReachable(t *testing.T) {
	gated := func(b Binding, w string) Binding { b.When = w; return b }
	k := mustNew(t, Settings{}, nil, []Binding{
		bind(t, ProgramLayer, command.Global, "ctrl+q", "quit"),
		bind(t, ProgramLayer, command.Global, "ctrl+w", "quit"),
		bind(t, ProgramLayer, "pane:b", "x", "offpath"),
		gated(bind(t, ProgramLayer, command.Global, "w", "gated"), "x"),
		bind(t, ProgramLayer, command.Global, "g", "outer.g"),
		bind(t, ProgramLayer, "pane:a", "g g", "top"),
		bind(t, ProgramLayer, command.Global, "h", "outer.h"),
		gated(bind(t, ProgramLayer, "pane:a", "h", "inner.h"), "y"),
	})
	p := Path{command.Global, "pane:a"}
	for _, c := range []struct {
		ctx  when.Map
		want []command.ID
	}{
		{when.Map{}, []command.ID{"top", "outer.h", "quit"}},
		{when.Map{"x": when.BoolValue(true)}, []command.ID{"top", "outer.h", "gated", "quit"}},
		{when.Map{"y": when.BoolValue(true)}, []command.ID{"inner.h", "top", "quit"}},
	} {
		got := slices.Collect(k.Reachable(p, c.ctx))
		if !slices.Equal(got, c.want) {
			t.Errorf("with %v, Reachable = %v; want %v", c.ctx, got, c.want)
		}
	}
	for range k.Reachable(p, nil) {
		break // stopping early is allowed
	}
}

// TestShortcut: the shortcut is the primary sequence's label, and "" for
// a command unbound, or whose last binding a user's "-" removes.
func TestShortcut(t *testing.T) {
	ds := defaults(bind(t, DefaultLayer, command.Global, "ctrl+s", "save"), bind(t, DefaultLayer, command.Global, "ctrl+q", "quit"))
	k := mustNew(t, Settings{}, ds, []Binding{{Command: "quit", Remove: true, Location: Location{Layer: UserLayer}}})
	for _, c := range []struct {
		id         command.ID
		goos, want string
	}{{"save", "darwin", "⌃s"}, {"save", "linux", "ctrl+s"}, {"quit", "linux", ""}, {"none", "linux", ""}} {
		if got := k.Shortcut(c.id, global, c.goos, glyph.Unicode()); got != c.want {
			t.Errorf("Shortcut(%s, %s) = %q; want %q", c.id, c.goos, got, c.want)
		}
	}
}

// TestFactsFrom: the facts termcap's capabilities give (amendment A9).
func TestFactsFrom(t *testing.T) {
	kitty := ansi.KittyDisambiguateEscapeCodes | ansi.KittyReportEventTypes
	caps := func(b termcap.Brand, flags int) termcap.Caps {
		return termcap.Caps{Brand: termcap.Fact[termcap.Brand]{Value: b}, KeyboardFlags: flags}
	}
	for _, c := range []struct {
		name string
		caps termcap.Caps
		want KeyFacts
	}{
		{"kitty with the protocol", caps(termcap.BrandKitty, kitty), ShiftEnter | AltEnter | CtrlEnter | Releases},
		{"kitty, disambiguation only", caps(termcap.BrandKitty, ansi.KittyDisambiguateEscapeCodes), ShiftEnter | AltEnter | CtrlEnter},
		{"VTE, legacy", caps(termcap.BrandVTE, 0), AltEnter},
		{"an unknown terminal, as over SSH", caps(termcap.BrandUnknown, 0), AltEnter},
		{"Apple Terminal", caps(termcap.BrandAppleTerminal, 0), 0},
		{"Apple Terminal, were it to accept the flag", caps(termcap.BrandAppleTerminal, kitty), ShiftEnter | Releases},
		{"iTerm2 with the protocol", caps(termcap.BrandITerm2, kitty), ShiftEnter | CtrlEnter | Releases},
		{"VS Code", caps(termcap.BrandVSCode, 0), AltEnter | HostTakesCtrlQ | HostTakesCtrlI},
		{"Cursor", caps(termcap.BrandCursor, 0), AltEnter | HostTakesCtrlQ | HostTakesCtrlI},
		{"Windsurf", caps(termcap.BrandWindsurf, 0), AltEnter | HostTakesCtrlQ | HostTakesCtrlI},
		{"Zed", caps(termcap.BrandZed, 0), AltEnter | HostTakesCtrlQ | HostTakesCtrlI},
	} {
		if got := FactsFrom(c.caps); got != c.want {
			t.Errorf("%s: FactsFrom = %s; want %s", c.name, got, c.want)
		}
	}
	k := mustNew(t, Settings{Facts: FactsFrom(caps(termcap.BrandVSCode, 0))}, []Default{newline(t)})
	if got := fmt.Sprint(k.Bindings("newline", global)); got != "[alt+enter]" {
		t.Errorf("in VS Code, newline is bound to %s; want alt+enter", got)
	}
}
