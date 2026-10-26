package command

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"slices"
	"testing"
)

func TestBuiltins(t *testing.T) {
	r := registryOf(t, nil,
		cmd("shown", UI),
		cmd("hidden", UI, func(c *Command) { c.Hidden = true }),
		cmd("off", UI, func(c *Command) { c.When = "on" }),
		cmd("shell", UI, func(c *Command) { c.Surfaces = SurfaceCLI }),
	)
	for _, id := range []ID{BuiltinList, BuiltinDescribe, BuiltinQuit} {
		c, ok := r.Lookup(id)
		if !ok || c.Slash != "" || len(c.Aliases) != 0 {
			t.Errorf("%s: registered %v, slash %q %v; want registered with no slash name", id, ok, c.Slash, c.Aliases)
		}
	}
	listed := func(o Origin) []ID {
		t.Helper()
		res, err := r.Run(t.Context(), Request{ID: BuiltinList, Origin: o})
		if err != nil {
			t.Fatalf("command.list from %s: %v", o, err)
		}
		infos, _ := res.Value.([]ManifestCommand)
		var out []ID
		for _, i := range infos {
			out = append(out, i.ID)
		}
		return out
	}
	for o, want := range map[Origin][]ID{
		OriginAgent:   {BuiltinQuit, BuiltinDescribe, BuiltinList, "shown"},
		OriginCLI:     {BuiltinDescribe, BuiltinList, "shell", "shown"},
		OriginProgram: {BuiltinQuit, BuiltinDescribe, BuiltinList, "shell", "shown"},
	} {
		if got := listed(o); !slices.Equal(got, want) {
			t.Errorf("command.list from %s = %v, want %v", o, got, want)
		}
	}

	res, err := r.Run(t.Context(), Request{ID: BuiltinDescribe, Args: []byte(`{"id":"workspace.x"}`), Origin: OriginAgent})
	if ae, ok := errors.AsType[*ArgError](err); !ok || ae.Path != "/id" {
		t.Errorf("describing an unknown ID: %v", err)
	}
	res, err = r.Run(t.Context(), Request{ID: BuiltinDescribe, Args: []byte(`{"id":"command.describe"}`), Origin: OriginAgent})
	info, _ := res.Value.(ManifestCommand)
	if err != nil || info.ID != BuiltinDescribe || info.Danger != "read-only" || len(info.Args) == 0 {
		t.Fatalf("command.describe: %+v, %v", info, err)
	}
	var schema map[string]any
	if err := json.Unmarshal(info.Args, &schema); err != nil || schema["$schema"] != schemaDialect {
		t.Errorf("command.describe's schema: %s, %v", info.Args, err)
	}

	msgs := collect(r.Dispatch(t.Context(), Request{ID: BuiltinQuit, Origin: OriginKey}))
	if len(msgs) != 2 || !slices.Contains(msgs, any(QuitRequestMsg{})) || resultOf(t, msgs).Err != nil {
		t.Errorf("app.quit sent %v, want a QuitRequestMsg and its ResultMsg", msgs)
	}
	if _, err := r.Run(t.Context(), Request{ID: BuiltinQuit, Origin: OriginCLI}); !errors.Is(err, ErrUnavailable) {
		t.Errorf("app.quit from the shell: %v", err)
	}
	for id, d := range map[ID]Danger{BuiltinList: ReadOnly, BuiltinDescribe: ReadOnly, BuiltinQuit: UI} {
		if c, _ := r.Lookup(id); c.Danger != d {
			t.Errorf("%s is %s, want %s", id, c.Danger, d)
		}
	}
}

// TestWithoutBuiltins: without the option a registry holds its three
// commands on every surface that offers them; with it, none
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D4).
func TestWithoutBuiltins(t *testing.T) {
	if BuiltinList != "command.list" || BuiltinDescribe != "command.describe" || BuiltinQuit != "app.quit" {
		t.Errorf("the built-in IDs are %s, %s and %s", BuiltinList, BuiltinDescribe, BuiltinQuit)
	}
	builtins := []ID{BuiltinQuit, BuiltinDescribe, BuiltinList}
	held := func(r *Registry) map[string][]ID {
		var all, tools, manifest, complete []ID
		for c := range r.All() {
			all = append(all, c.ID)
		}
		for _, tool := range r.MCPTools(nil) {
			tools = append(tools, ID(tool.Name))
		}
		for _, c := range r.Manifest().Commands {
			manifest = append(manifest, c.ID)
		}
		for _, id := range r.Complete("", nil, "") {
			complete = append(complete, ID(id))
		}
		return map[string][]ID{"All": all, "MCPTools": tools, "Manifest": manifest, "Complete": complete}
	}
	for name, got := range held(NewRegistry()) {
		want := builtins
		if name == "Complete" {
			want = []ID{BuiltinDescribe, BuiltinList} // app.quit is not offered on the shell
		}
		if !slices.Equal(got, want) {
			t.Errorf("NewRegistry(): %s holds %v, want %v", name, got, want)
		}
	}
	bare := registryOf(t, []RegistryOption{WithoutBuiltins()}, cmd("own", ReadOnly))
	for name, got := range held(bare) {
		if !slices.Equal(got, []ID{"own"}) {
			t.Errorf("WithoutBuiltins: %s holds %v, want only the program's own", name, got)
		}
	}
	if bare.Version() != 1 {
		t.Errorf("WithoutBuiltins: version %d after one Register, want 1", bare.Version())
	}
	if err := bare.Register(cmd(BuiltinQuit, UI)); err != nil {
		t.Errorf("WithoutBuiltins: registering a program's own %s: %v", BuiltinQuit, err)
	}
}

func TestNewNeedsDanger(t *testing.T) {
	run := func(context.Context, *Invocation, NoArgs) (Result, error) { return Result{}, nil }
	if _, err := New("x", "X", run); err == nil {
		t.Error("New without WithDanger succeeded")
	}
	c, err := New("x", "X", run, WithDanger(ReadOnly), WithSlash("x", "y"), WithMeta("icon", "x"),
		WithDescription("d"), WithCategory("c"), WithArgHint("h"), WithOutput(Schema(`{}`)), WithWhen("a"),
		WithScope("pane"), WithIdempotent(), WithOpenWorld(), WithSurfaces(SurfaceKey), WithMode(Async),
		WithExclusive(), WithWhileBusy(), WithHidden())
	if err != nil {
		t.Fatal(err)
	}
	if c.Danger != ReadOnly || c.Slash != "x" || !slices.Equal(c.Aliases, []string{"y"}) || c.Meta["icon"] != "x" ||
		c.Description != "d" || c.Category != "c" || c.ArgHint != "h" || string(c.Output) != `{}` || c.When != "a" ||
		c.Scope != "pane" || !c.Idempotent || !c.OpenWorld || c.Surfaces != SurfaceKey || c.Mode != Async ||
		!c.Exclusive || !c.WhileBusy || !c.Hidden {
		t.Errorf("the options did not all land: %+v", c)
	}
	if _, err := New[int]("x", "X", func(context.Context, *Invocation, int) (Result, error) { return Result{}, nil }, WithDanger(UI)); err == nil {
		t.Error("New with non-object arguments succeeded")
	}
	if _, err := New[NoArgs]("x", "X", nil, WithDanger(UI)); err == nil {
		t.Error("New with no run function succeeded")
	}
	if _, err := New("X", "X", run, WithDanger(UI)); err == nil {
		t.Error("New with a malformed ID succeeded")
	}
}
