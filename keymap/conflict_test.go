package keymap

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/when"
)

// fileBinding is a user binding read from keybindings.json's n-th entry.
func fileBinding(t *testing.T, n int, scope command.Scope, keys string, id command.ID) Binding {
	t.Helper()
	b := bind(t, UserLayer, scope, keys, id)
	b.Location = Location{Layer: UserLayer, File: "keybindings.json", Line: 10 + n, Col: 5,
		Pointer: jsontext.Pointer(fmt.Sprintf("/%d/key", n))}
	return b
}

// TestConflicts: each error kind, planted in one keymap, comes back from
// New, all of them joined, each naming its binding's location; a Shadow
// is listed by Conflicts and does not fail (MADR §4, amendment A6).
func TestConflicts(t *testing.T) {
	ds := defaults(
		bind(t, DefaultLayer, ws, "ctrl+x n", "file.new"),
		bind(t, DefaultLayer, command.Global, "ctrl+s", "file.save"),
		bind(t, DefaultLayer, ws, "ctrl+o", "file.open"),
	)
	program := []Binding{bind(t, ProgramLayer, command.Global, "ctrl+c", "app.quit")} // the program may bind its reserved key
	user := []Binding{
		fileBinding(t, 0, command.Global, "ctrl+d", "one"),
		fileBinding(t, 1, command.Global, "ctrl+d", "two"),
		fileBinding(t, 2, ws, "ctrl+x", "cut"),
		fileBinding(t, 3, command.Global, "ctrl+c a", "copy.all"),
		fileBinding(t, 4, command.Global, "ctrl+i", "indent"),
		fileBinding(t, 5, ws, "ctrl+s", "ws.save"),
		fileBinding(t, 6, ws, "ctrl+o", "ws.open"),
		fileBinding(t, 7, pane, "ctrl+d", "pane.d"), // shadows global's two ctrl+d, and is no Duplicate
	}
	k, err := New(Settings{Reserved: []string{"ctrl+c"}}, ds, program, user)
	if k == nil {
		t.Fatal("New returned no keymap")
	}
	var got []string
	for _, e := range unjoin(err) {
		var c BindingConflict
		if !errors.As(e, &c) {
			t.Errorf("New: %v is not a BindingConflict", e)
			continue
		}
		got = append(got, c.Kind.String()+" "+c.Keys.String())
	}
	want := []string{"reserved ctrl+c", "not-supported ctrl+i", "duplicate ctrl+d", "prefix ctrl+x"}
	if !slices.Equal(got, want) {
		t.Fatalf("New's errors are %q, want %q:\n%v", got, want, err)
	}
	for _, part := range []string{
		"keybindings.json:13:5 (/3/key): ctrl+c a: starts with ctrl+c, which the program reserves",
		"keybindings.json:14:5 (/4/key): ctrl+i: needs disambiguate, which the terminal has not reported",
		"keybindings.json:12:5 (/2/key): ctrl+x in workspace: one of ctrl+x and ctrl+x n (file.new at default layer) starts the other",
		"keybindings.json:11:5 (/1/key): ctrl+d: bound twice in one layer with the same when, also to one at keybindings.json:10:5 (/0/key)",
	} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("New's error lacks %q:\n%v", part, err)
		}
	}
	var shadows []string
	for _, c := range k.Conflicts() {
		shadows = append(shadows, c.Kind.String()+" "+c.Keys.String()+" "+string(c.Scope))
	}
	slices.Sort(shadows)
	if want := []string{"shadow ctrl+d pane:a", "shadow ctrl+d pane:a", "shadow ctrl+o workspace", "shadow ctrl+s workspace"}; !slices.Equal(shadows, want) {
		t.Errorf("Conflicts() = %q, want %q", shadows, want)
	}
	for _, c := range k.Conflicts() {
		if c.Keys.String() == "ctrl+s" && !strings.Contains(c.Error(), "keybindings.json:15:5 (/5/key): ctrl+s in workspace: hides file.save's ctrl+s at default layer") {
			t.Errorf("the Shadow reads %q", c.Error())
		}
	}
	// The keymap holds what built: a Reserved binding is dropped, a
	// NotSupported one waits, and a Duplicate or Prefix pair stays, the
	// later winning.
	p := Path{command.Global, ws}
	for keys, want := range map[string]command.ID{
		"ctrl+c a": "", "ctrl+c": "app.quit", "ctrl+i": "", "ctrl+d": "two", "ctrl+s": "ws.save", "ctrl+x": "cut",
	} {
		if got := lookup(t, k, p, when.Map{}, keys); got != want {
			t.Errorf("Lookup(%q) = %q, want %q", keys, got, want)
		}
	}
}

// unjoin is the errors errors.Join joined.
func unjoin(err error) []error {
	if j, ok := err.(interface{ Unwrap() []error }); ok {
		return j.Unwrap()
	}
	if err == nil {
		return nil
	}
	return []error{err}
}

// TestConflictText: a conflict with no bindings still names its kind and
// keys.
func TestConflictText(t *testing.T) {
	c := BindingConflict{Kind: Prefix, Keys: seq(t, "ctrl+x")}
	if got := c.Error(); got != "keymap: prefix ctrl+x" {
		t.Errorf("Error() = %q", got)
	}
}
