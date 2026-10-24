package keymap

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// labelled are the sequences the label golden files show: every modifier
// alone and together, every arrow, the display aliases, and a chord, the
// leader placeholder, an event and a scan code.
var labelled = []string{
	"ctrl+k", "alt+k", "shift+k", "meta+k", "hyper+k", "super+k",
	"ctrl+alt+shift+super+k", "ctrl+alt+shift+meta+hyper+super+k",
	"up", "down", "left", "right", "shift+up", "ctrl+alt+left", "super+right",
	"pgup", "pgdown", "delete", "ctrl+delete", "esc", "enter", "space", "tab", "f5", "?",
	"ctrl+x ctrl+s", "<leader> f", "a:release", "ctrl+[KeyA]",
}

// TestLabelGolden: each sequence's label on darwin, linux and windows, in
// UTF-8 and ASCII.
func TestLabelGolden(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "windows"} {
		for _, utf8 := range []bool{true, false} {
			charset := map[bool]string{true: "utf8", false: "ascii"}[utf8]
			var b strings.Builder
			for _, s := range labelled {
				fmt.Fprintf(&b, "%-36s %s\n", s, seq(t, s).Label(goos, glyph.For(utf8)))
			}
			tuitest.Text(t, "labels."+goos+"."+charset, b.String())
		}
	}
}

// TestLabelASCII: an ASCII label is ASCII, and a UTF-8 one on darwin draws
// each modifier as its one-cell glyph.
func TestLabelASCII(t *testing.T) {
	for _, s := range labelled {
		for _, goos := range []string{"darwin", "linux", "windows"} {
			if l := seq(t, s).Label(goos, glyph.ASCII()); strings.ContainsFunc(l, func(r rune) bool { return r > 0x7e }) {
				t.Errorf("%s on %s: the ASCII label %q is not ASCII", s, goos, l)
			}
		}
	}
	u := glyph.Unicode().Keys
	if got, want := seq(t, "ctrl+alt+shift+super+k").Label("darwin", glyph.Unicode()), u.Control+u.Option+u.Shift+u.Command+"k"; got != want {
		t.Errorf("darwin's label is %q; want %q", got, want)
	}
}

// TestLeaderLabel: a keymap's bindings have <leader> expanded, so a
// label shows the leader's own keys; the placeholder alone shows
// <leader>.
func TestLeaderLabel(t *testing.T) {
	k := mustNew(t, Settings{Leader: "ctrl+x"}, nil, []Binding{bind(t, ProgramLayer, command.Global, "<leader> f", "cmd.find")})
	q := k.Bindings("cmd.find", global)[0]
	for _, c := range []struct {
		goos string
		g    glyph.Set
		want string
	}{{"darwin", glyph.Unicode(), "⌃x f"}, {"darwin", glyph.ASCII(), "ctrl+x f"}, {"linux", glyph.Unicode(), "ctrl+x f"}} {
		if got := q.Label(c.goos, c.g); got != c.want {
			t.Errorf("%s: %q; want %q", c.goos, got, c.want)
		}
	}
	if got := seq(t, "<leader> f").Label("darwin", glyph.Unicode()); got != "<leader> f" {
		t.Errorf("the placeholder's label is %q", got)
	}
}

// parts is a canonical or VS Code spelling's modifiers and key, the event
// left out.
func parts(text string) []string {
	text = strings.TrimSuffix(strings.TrimSuffix(text, ":release"), ":repeat")
	if strings.HasSuffix(text, "++") || text == "+" {
		return append(strings.Split(strings.TrimSuffix(text, "++"), "+"), "+")
	}
	return strings.Split(text, "+")
}

// TestAliases: every parse alias reads as its canonical name, and the
// schema's key pattern accepts it; String never writes one, nor either
// spelling a display alias, which Label writes; and each table is the
// caller's to change.
func TestAliases(t *testing.T) {
	aliases := ParseAliases()
	want := []string{"cmd", "del", "escape", "option", "pagedown", "pageup", "return", "win"}
	if got := slices.Sorted(maps.Keys(aliases)); !slices.Equal(got, want) {
		t.Errorf("ParseAliases has %q; want %q", got, want)
	}
	for alias, canon := range aliases {
		in, canonIn := alias, canon
		if _, isKey := keyAliases[alias]; !isKey {
			in, canonIn = alias+"+a", canon+"+a"
		}
		if got, want := mustStroke(t, in), mustStroke(t, canonIn); got != want {
			t.Errorf("%q reads as %s; want %s", in, got, want)
		}
		if !keyRE.MatchString(in) {
			t.Errorf("the schema's key pattern refuses %q", in)
		}
	}
	display := DisplayAliases()
	if !maps.Equal(display, map[string]string{"pgdown": "pgdn", "delete": "del"}) {
		t.Errorf("DisplayAliases is %v", display)
	}
	for canon, short := range display {
		if got := mustStroke(t, canon).Label("linux", glyph.ASCII()); got != short {
			t.Errorf("%s's label is %q; want %q", canon, got, short)
		}
	}
	r := rand.New(rand.NewPCG(9, 9))
	for range 5000 {
		s := randomStroke(r)
		for _, p := range parts(s.String()) {
			if _, ok := aliases[p]; ok {
				t.Fatalf("%q writes the parse alias %q", s.String(), p)
			}
		}
		for _, text := range []string{s.String(), s.VSCode()} {
			for _, p := range parts(text) {
				if slices.Contains(slices.Collect(maps.Values(display)), p) {
					t.Fatalf("%q writes the display alias %q", text, p)
				}
			}
		}
	}
	ParseAliases()["return"] = "x"
	DisplayAliases()["delete"] = "x"
	if ParseAliases()["return"] != "enter" || DisplayAliases()["delete"] != "del" {
		t.Error("changing a returned table changed the package's")
	}
}

// TestCopyStrokes: the copy keys, each its own slice.
func TestCopyStrokes(t *testing.T) {
	var got []string
	for _, s := range CopyStrokes() {
		got = append(got, s.String())
	}
	if want := []string{"ctrl+c", "super+c", "ctrl+shift+c"}; !slices.Equal(got, want) {
		t.Errorf("CopyStrokes is %q; want %q", got, want)
	}
	CopyStrokes()[0] = Stroke{}
	if CopyStrokes()[0].String() != "ctrl+c" {
		t.Error("changing CopyStrokes' slice changed the next")
	}
}
