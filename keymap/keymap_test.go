package keymap

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/when"
)

func seq(t testing.TB, s string) Sequence {
	t.Helper()
	q, err := ParseSequence(s, nil)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

// bind is a binding of keys to id in scope, made in code in layer l.
func bind(t testing.TB, l Layer, scope command.Scope, keys string, id command.ID) Binding {
	t.Helper()
	return Binding{Keys: seq(t, keys), Command: id, Scope: scope, Location: Location{Layer: l}}
}

// mustNew builds a keymap that must have no error.
func mustNew(t testing.TB, s Settings, defaults []Default, layers ...[]Binding) *Keymap {
	t.Helper()
	k, err := New(s, defaults, layers...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return k
}

// lookup is the command a sequence resolves to, or "" when none.
func lookup(t testing.TB, k *Keymap, p Path, ctx when.Context, keys string) command.ID {
	t.Helper()
	r, ok := k.Lookup(p, ctx, seq(t, keys))
	if ok != (r.Command != "") {
		t.Fatalf("Lookup(%q) = %+v, %v: ok and the command disagree", keys, r, ok)
	}
	return r.Command
}

func defaults(bs ...Binding) []Default {
	ds := make([]Default, len(bs))
	for i, b := range bs {
		ds[i] = Default{Binding: b}
	}
	return ds
}

const (
	ws   command.Scope = "workspace"
	pane command.Scope = "pane:a"
)

// TestResolution: the innermost scope first, then the later layer, then
// the later binding, with when filtering through the when package (MADR
// §3).
func TestResolution(t *testing.T) {
	busy := Binding{Keys: seq(t, "w"), Command: "b.w", Scope: ws, When: "busy", Location: Location{Layer: ProgramLayer}}
	k := mustNew(t, Settings{},
		defaults(
			bind(t, DefaultLayer, command.Global, "x", "g.x"),
			bind(t, DefaultLayer, ws, "x", "ws.x"),
			bind(t, DefaultLayer, ws, "z", "d.z"),
		),
		[]Binding{
			bind(t, ProgramLayer, pane, "y", "pane.y"),
			bind(t, ProgramLayer, ws, "z", "p.z"),
			bind(t, ProgramLayer, ws, "w", "a.w"),
			busy,
		})
	full := Path{command.Global, ws, pane}
	on := when.Map{"busy": when.BoolValue(true)}
	for _, c := range []struct {
		p    Path
		ctx  when.Context
		keys string
		want command.ID
	}{
		{full, nil, "x", "ws.x"},                // the innermost scope with x
		{Path{command.Global}, nil, "x", "g.x"}, // workspace is off the path
		{full, nil, "y", "pane.y"},
		{Path{command.Global, ws}, nil, "y", ""}, // pane:a is off the path
		{full, nil, "z", "p.z"},                  // the later layer
		{full, on, "w", "b.w"},                   // the later binding, its when holding
		{full, nil, "w", "a.w"},                  // busy unset: the earlier binding
		{full, when.Map{"busy": when.BoolValue(false)}, "w", "a.w"},
		{full, nil, "q", ""},
		{nil, nil, "x", ""}, // an empty path resolves nothing
	} {
		if got := lookup(t, k, c.p, c.ctx, c.keys); got != c.want {
			t.Errorf("Lookup(%v, %q) = %q, want %q", c.p, c.keys, got, c.want)
		}
	}
	r, _ := k.Lookup(full, on, seq(t, "w"))
	if r.Binding.When != "busy" || r.Binding.Scope != ws || r.Binding.Location.Layer != ProgramLayer {
		t.Errorf("Resolved.Binding = %+v; want the busy binding", r.Binding)
	}
	if _, ok := k.Lookup(full, nil, nil); ok {
		t.Error("Lookup of no keys resolved")
	}
}

// TestScopeDecides: along a path, the innermost scope with a binding whose
// keys are the sequence, start it, or start with it decides it, and outer
// scopes are not tried (amendment A6).
func TestScopeDecides(t *testing.T) {
	editing := Binding{Keys: seq(t, "j j"), Command: "pane.jj", Scope: pane, When: "editing"}
	k := mustNew(t, Settings{},
		defaults(
			bind(t, DefaultLayer, command.Global, "g", "g.g"),
			bind(t, DefaultLayer, command.Global, "h h", "g.hh"),
			bind(t, DefaultLayer, command.Global, "j", "g.j"),
		),
		[]Binding{
			bind(t, ProgramLayer, pane, "g g", "pane.gg"),
			bind(t, ProgramLayer, pane, "h", "pane.h"),
			editing,
		})
	p := Path{command.Global, pane}
	on := when.Map{"editing": when.BoolValue(true)}
	for _, c := range []struct {
		ctx  when.Context
		keys string
		want command.ID
	}{
		{nil, "g", ""}, // pane:a's g g decides g
		{nil, "g g", "pane.gg"},
		{nil, "h", "pane.h"},
		{nil, "h h", ""}, // pane:a's h decides h h
		{nil, "j", "g.j"},
		{on, "j", ""}, // j j's when holds, so pane:a decides j
		{on, "j j", "pane.jj"},
	} {
		if got := lookup(t, k, p, c.ctx, c.keys); got != c.want {
			t.Errorf("Lookup(%q) = %q, want %q", c.keys, got, c.want)
		}
	}
	if got := lookup(t, k, Path{command.Global}, nil, "g"); got != "g.g" {
		t.Errorf("Lookup(g) on the global path = %q, want g.g", got)
	}
	var shadows []string
	for _, c := range k.Conflicts() {
		if c.Kind != Shadow {
			t.Errorf("Conflicts() has a %s", c.Kind)
		}
		shadows = append(shadows, c.Keys.String())
	}
	slices.Sort(shadows)
	if want := []string{"g g", "h", "j j"}; !slices.Equal(shadows, want) {
		t.Errorf("the Shadows of global bindings are %q, want %q", shadows, want)
	}
}

// TestRemoval: a removal with keys removes those keys of the command in
// the scope, and fails when none is bound; one with no keys removes every
// binding of the command in the scope; a when limits it (MADR §3).
func TestRemoval(t *testing.T) {
	user := func(keys string, id command.ID, w string) Binding {
		b := Binding{Command: id, Scope: ws, When: w, Remove: true,
			Location: Location{Layer: UserLayer, File: "keybindings.json", Line: 4, Col: 3, Pointer: "/1"}}
		if keys != "" {
			b.Keys = seq(t, keys)
		}
		return b
	}
	ds := defaults(
		bind(t, DefaultLayer, ws, "a", "cmd.a"),
		bind(t, DefaultLayer, ws, "b", "cmd.a"),
		bind(t, DefaultLayer, command.Global, "a", "cmd.a"),
		bind(t, DefaultLayer, ws, "c", "cmd.c"),
		bind(t, DefaultLayer, ws, "d", "cmd.c"),
	)
	ds = append(ds, Default{Binding: bind(t, DefaultLayer, ws, "shift+enter", "newline"), Alternatives: []Alternative{{Keys: seq(t, "ctrl+j")}}})
	whens := []Binding{bind(t, ProgramLayer, ws, "q", "cmd.q"), bind(t, ProgramLayer, ws, "q", "cmd.q")}
	whens[1].When = "x"
	k, err := New(Settings{Features: Disambiguate}, ds, whens, []Binding{
		user("a", "cmd.a", ""),
		user("z", "cmd.a", ""),
		user("", "cmd.c", ""),
		user("", "cmd.none", ""),
		user("q", "cmd.q", "x"),
		user("ctrl+j", "newline", ""),
	})
	if err == nil || strings.Count(err.Error(), "\n") != 0 ||
		!strings.Contains(err.Error(), "keybindings.json:4:3 (/1): z in workspace: removes z from cmd.a, which is not bound there") {
		t.Fatalf("New: %v; want one error, the removal of z, named", err)
	}
	p := Path{command.Global, ws}
	for keys, want := range map[string]command.ID{
		"a": "cmd.a", // the global binding; the workspace's is removed
		"b": "cmd.a", "c": "", "d": "", "q": "cmd.q", "shift+enter": "", "ctrl+j": "",
	} {
		if got := lookup(t, k, p, when.Map{"x": when.BoolValue(true)}, keys); got != want {
			t.Errorf("after the removals, Lookup(%q) = %q, want %q", keys, got, want)
		}
	}
	if got := lookup(t, k, Path{ws}, nil, "a"); got != "" {
		t.Errorf("Lookup(a) in workspace = %q; the removal left it", got)
	}
	n := 0
	for b := range k.Rules() {
		if b.Command == "cmd.q" {
			n++
			if b.When != "" {
				t.Errorf("the removal with when x left %+v", b)
			}
		}
	}
	if n != 1 {
		t.Errorf("%d cmd.q bindings left, want 1", n)
	}
}

// TestFeatures: a default whose keys need missing features binds its one
// alternative, and its own keys after Rebuild; one with no alternative
// binds nothing and reports nothing; a binding the terminal cannot
// deliver is NotSupported, kept, and binds after Rebuild (MADR §5,
// amendments A1 and A6).
func TestFeatures(t *testing.T) {
	ds := []Default{
		{Binding: bind(t, DefaultLayer, command.Global, "ctrl+m", "newline"), Alternatives: []Alternative{{Keys: seq(t, "ctrl+j")}}},
		{Binding: bind(t, DefaultLayer, command.Global, "shift+enter", "submit")},
	}
	user := []Binding{bind(t, UserLayer, command.Global, "ctrl+i", "indent")}
	p := Path{command.Global}
	k, err := New(Settings{}, ds, user)
	var c BindingConflict
	if !errors.As(err, &c) || c.Kind != NotSupported || c.Keys.String() != "ctrl+i" ||
		!strings.Contains(err.Error(), "user layer: ctrl+i: needs disambiguate") {
		t.Fatalf("New with no features: %v; want ctrl+i NotSupported", err)
	}
	for id, want := range map[command.ID]string{"newline": "[ctrl+j]", "submit": "[]", "indent": "[]"} {
		if got := fmt.Sprint(k.Bindings(id, p)); got != want {
			t.Errorf("with no features, Bindings(%s) = %s, want %s", id, got, want)
		}
	}
	k2, err := k.Rebuild(Disambiguate, 0)
	if err != nil {
		t.Fatalf("Rebuild(disambiguate): %v", err)
	}
	for id, want := range map[command.ID]string{"newline": "ctrl+m", "submit": "shift+enter", "indent": "ctrl+i"} {
		if got := k2.Bindings(id, p); len(got) != 1 || got[0].String() != want {
			t.Errorf("after Rebuild, Bindings(%s) = %v, want %s", id, got, want)
		}
	}
	if got := lookup(t, k, p, nil, "ctrl+j"); got != "newline" {
		t.Errorf("Rebuild changed the keymap it was called on: Lookup(ctrl+j) = %q", got)
	}
	for keys, want := range map[string]Features{
		"a": 0, "ctrl+m": Disambiguate, "ctrl+[": Disambiguate, "alt+]": Disambiguate, "alt+[": Disambiguate,
		"ctrl+i": Disambiguate, "ctrl+i:release": Disambiguate | EventTypes, "a:repeat": EventTypes,
		"[KeyA]": AlternateKeys, "x [Slash]:release": AlternateKeys | EventTypes, "enter": 0,
	} {
		if got := needs(seq(t, keys)); got != want {
			t.Errorf("needs(%q) = %v, want %v", keys, got, want)
		}
	}
}

// TestFeaturesFrom: each Kitty flag is the feature of its bit, and the
// flags this package does not name are dropped.
func TestFeaturesFrom(t *testing.T) {
	for flags, want := range map[int]Features{
		0: 0, ansi.KittyDisambiguateEscapeCodes: Disambiguate, ansi.KittyReportEventTypes: EventTypes,
		ansi.KittyReportAlternateKeys: AlternateKeys, ansi.KittyReportAllKeysAsEscapeCodes: AllKeysAsEscapes,
		ansi.KittyReportAssociatedKeys: AssociatedText, ansi.KittyAllFlags: allFeatures,
		1<<6 | ansi.KittyReportEventTypes: EventTypes,
	} {
		if got := FeaturesFrom(flags); got != want {
			t.Errorf("FeaturesFrom(%#x) = %v, want %v", flags, got, want)
		}
	}
}

// TestLeader: <leader> expands to Settings.Leader in New, and a binding
// that names it with no leader set, or grows past MaxStrokes with it, is
// an error naming the binding (MADR §6, amendment A6).
func TestLeader(t *testing.T) {
	b := Binding{Keys: seq(t, "<leader> f"), Command: "find", Location: Location{Layer: UserLayer, File: "k.json", Line: 2, Col: 9}}
	k := mustNew(t, Settings{Leader: "ctrl+x"}, nil, []Binding{b})
	if got := lookup(t, k, Path{command.Global}, nil, "ctrl+x f"); got != "find" {
		t.Errorf("Lookup(ctrl+x f) = %q, want find", got)
	}
	for r := range k.Rules() {
		if r.Keys.String() != "ctrl+x f" {
			t.Errorf("Rules() gives %q, want the expanded ctrl+x f", r.Keys)
		}
	}
	long := b
	long.Keys = seq(t, "<leader> a b c")
	for _, c := range []struct {
		leader string
		b      Binding
		part   string
	}{
		{"", b, `k.json:2:9: <leader> f: keymap: bad key notation "<leader> f": <leader>, and no leader is set`},
		{"ctrl+x ctrl+y", long, "5 strokes with the leader, more than 4"},
		{"<leader>", b, "Settings.Leader names <leader>"},
		{"ctrl+foo", b, "Settings.Leader"},
	} {
		_, err := New(Settings{Leader: c.leader}, nil, []Binding{c.b})
		if !errors.Is(err, ErrNotation) || !strings.Contains(err.Error(), c.part) {
			t.Errorf("leader %q, binding %s: %v; want ErrNotation naming %q", c.leader, c.b.Keys, err, c.part)
		}
	}
}

// TestBindingsOrder: Bindings is in resolution order, the primary first,
// leaving out a sequence a binding with no when hides.
func TestBindingsOrder(t *testing.T) {
	guarded := bind(t, ProgramLayer, pane, "alt+p", "other")
	guarded.When = "editing"
	k := mustNew(t, Settings{},
		defaults(
			bind(t, DefaultLayer, command.Global, "ctrl+p", "palette"),
			bind(t, DefaultLayer, ws, "alt+p", "palette"),
		),
		[]Binding{
			bind(t, ProgramLayer, pane, "ctrl+p", "other"),
			guarded,
		},
		[]Binding{bind(t, UserLayer, ws, "f1", "palette")})
	for _, c := range []struct {
		p    Path
		want string
	}{
		{Path{command.Global, ws}, "[f1 alt+p ctrl+p]"},
		{Path{command.Global, ws, pane}, "[f1 alt+p]"}, // pane:a's ctrl+p hides the global one
		{Path{command.Global}, "[ctrl+p]"},
		{nil, "[]"},
	} {
		if got := fmt.Sprint(k.Bindings("palette", c.p)); got != c.want {
			t.Errorf("Bindings(palette, %v) = %s, want %s", c.p, got, c.want)
		}
	}
}

// TestRules: the bindings in effect, in the order New took them.
func TestRules(t *testing.T) {
	k := mustNew(t, Settings{},
		defaults(bind(t, DefaultLayer, command.Global, "a", "one"), bind(t, DefaultLayer, ws, "b", "two")),
		[]Binding{bind(t, ProgramLayer, ws, "c", "three"), {Command: "two", Scope: ws, Remove: true}},
		[]Binding{bind(t, UserLayer, ws, "d", "four")})
	var got []string
	for b := range k.Rules() {
		got = append(got, fmt.Sprintf("%s=%s@%s", b.Keys, b.Command, b.Location.Layer))
	}
	if want := []string{"a=one@default", "c=three@program", "d=four@user"}; !slices.Equal(got, want) {
		t.Errorf("Rules() = %q, want %q", got, want)
	}
}

// TestLocationText: a location names its file, line, column and pointer,
// or the layer of a binding made in code.
func TestLocationText(t *testing.T) {
	for _, c := range []struct {
		l    Location
		want string
	}{
		{Location{}, "default layer"},
		{Location{Layer: UserLayer, File: "keybindings.json", Line: 12, Col: 5, Pointer: "/3/key"}, "keybindings.json:12:5 (/3/key)"},
		{Location{Layer: UserLayer, File: "keybindings.json"}, "keybindings.json"},
		{Location{Layer: ProgramLayer, Pointer: "/0"}, "program layer (/0)"},
	} {
		if got := c.l.String(); got != c.want {
			t.Errorf("%+v.String() = %q, want %q", c.l, got, c.want)
		}
	}
}

// TestImmutable: a keymap is safe for concurrent reads, and the slices
// given to New are its own copies.
func TestImmutable(t *testing.T) {
	layer := []Binding{bind(t, ProgramLayer, ws, "ctrl+k ctrl+c", "comment"), bind(t, UserLayer, command.Global, "x", "cut")}
	layer[0].Args = []byte(`{"n":1}`)
	reserved := []string{"ctrl+c"}
	k := mustNew(t, Settings{Reserved: reserved}, nil, layer)
	layer[0].Keys[1] = mustStroke(t, "ctrl+u")
	layer[0].Args[1] = 'X'
	layer[1] = bind(t, UserLayer, command.Global, "y", "paste")
	reserved[0] = "x"
	p := Path{command.Global, ws}
	if r, ok := k.Lookup(p, nil, seq(t, "ctrl+k ctrl+c")); !ok || string(r.Args) != `{"n":1}` {
		t.Errorf("after the caller's changes, Lookup = %+v, %v", r, ok)
	}
	if got := lookup(t, k, p, nil, "x"); got != "cut" {
		t.Errorf("after the caller's changes, Lookup(x) = %q, want cut", got)
	}
	k2, _ := k.Rebuild(0, 0)
	if got := lookup(t, k2, p, nil, "x"); got != "cut" {
		t.Errorf("Rebuild took the caller's changes: Lookup(x) = %q, want cut", got)
	}
	q := seq(t, "ctrl+k ctrl+c")
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 500 {
				if r, _ := k.Lookup(p, nil, q); r.Command != "comment" {
					t.Errorf("concurrent Lookup = %+v", r)
					return
				}
				_ = k.Bindings("cut", p)
				for range k.Rules() {
				}
				_ = k.Conflicts()
			}
		})
	}
	wg.Wait()
}

// BenchmarkLookup: a 200-binding keymap, over a five-scope path.
func BenchmarkLookup(b *testing.B) {
	scopes := Path{command.Global, ws, "pane:editor", "kind:editor", "mode:vim-normal"}
	var layer []Binding
	var keys []Sequence
	for i := range 200 {
		q := seq(b, fmt.Sprintf("ctrl+%c %c", 'a'+rune(i%26), 'a'+rune(i/26)))
		keys = append(keys, q)
		layer = append(layer, Binding{Keys: q, Command: command.ID(fmt.Sprintf("cmd.c%d", i)), Scope: scopes[i%len(scopes)]})
	}
	k := mustNew(b, Settings{Features: Disambiguate}, nil, layer) // ctrl+i and ctrl+m among the keys
	for i := 0; b.Loop(); i++ {
		if _, ok := k.Lookup(scopes, nil, keys[i%len(keys)]); !ok {
			b.Fatalf("Lookup(%s) did not resolve", keys[i%len(keys)])
		}
	}
}
