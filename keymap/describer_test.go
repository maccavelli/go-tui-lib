package keymap

import (
	"context"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/command"
)

// TestFromRegistry: FromRegistry gives a registered command's title, and
// none for a command it does not hold or one with no title.
func TestFromRegistry(t *testing.T) {
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
	d := DescriberFunc(func(id command.ID) (string, bool) { return "Save the file", id == "app.save" })
	if got, ok := d.Describe("app.save"); !ok || got != "Save the file" {
		t.Errorf("DescriberFunc: %q, %v", got, ok)
	}
}

// TestShown: Shown is the bindings help shows for a command, the primary
// first, each a copy, and none for a command bound to nothing or hidden.
func TestShown(t *testing.T) {
	p := Path{command.Global, pane}
	k := mustNew(t, Settings{}, nil, []Binding{
		bind(t, ProgramLayer, command.Global, "ctrl+s", "save"),
		bind(t, ProgramLayer, command.Global, "ctrl+x s", "save"),
		bind(t, ProgramLayer, command.Global, "g", "outer"),
		bind(t, ProgramLayer, pane, "g g", "top"),
	})
	shown := k.Shown("save", p)
	if len(shown) != 2 || shown[0].Keys.String() != "ctrl+x s" || shown[1].Keys.String() != "ctrl+s" {
		t.Fatalf("Shown(save) = %v; want ctrl+x s, then ctrl+s, as Bindings orders them", shown)
	}
	if want := k.Bindings("save", p); shown[0].Keys.String() != want[0].String() {
		t.Errorf("Shown's primary is %v, Bindings' %v", shown[0].Keys, want[0])
	}
	shown[0].Keys[0] = Stroke{Code: 'z'}
	if again := k.Shown("save", p); again[0].Keys.String() != "ctrl+x s" {
		t.Errorf("changing what Shown returned changed the keymap: %v", again[0].Keys)
	}
	for _, id := range []command.ID{"none", "outer"} {
		if got := k.Shown(id, p); len(got) != 0 {
			t.Errorf("Shown(%s) = %v; want nothing", id, got)
		}
	}
}
