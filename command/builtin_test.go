package command

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestBuiltins(t *testing.T) {
	r := registryOf(t, nil,
		cmd("shown", UI),
		cmd("hidden", UI, func(c *Command) { c.Hidden = true }),
		cmd("off", UI, func(c *Command) { c.When = "on" }),
		cmd("shell", UI, func(c *Command) { c.Surfaces = SurfaceCLI }),
	)
	for _, id := range []ID{idList, idDescribe, idQuit} {
		c, ok := r.Lookup(id)
		if !ok || c.Slash != "" || len(c.Aliases) != 0 {
			t.Errorf("%s: registered %v, slash %q %v; want registered with no slash name", id, ok, c.Slash, c.Aliases)
		}
	}
	listed := func(o Origin) []ID {
		t.Helper()
		res, err := r.Run(t.Context(), Request{ID: idList, Origin: o})
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
		OriginAgent:   {idQuit, idDescribe, idList, "shown"},
		OriginCLI:     {idDescribe, idList, "shell", "shown"},
		OriginProgram: {idQuit, idDescribe, idList, "shell", "shown"},
	} {
		if got := listed(o); !slices.Equal(got, want) {
			t.Errorf("command.list from %s = %v, want %v", o, got, want)
		}
	}

	res, err := r.Run(t.Context(), Request{ID: idDescribe, Args: []byte(`{"id":"workspace.x"}`), Origin: OriginAgent})
	if ae, ok := errors.AsType[*ArgError](err); !ok || ae.Path != "/id" {
		t.Errorf("describing an unknown ID: %v", err)
	}
	res, err = r.Run(t.Context(), Request{ID: idDescribe, Args: []byte(`{"id":"command.describe"}`), Origin: OriginAgent})
	info, _ := res.Value.(ManifestCommand)
	if err != nil || info.ID != idDescribe || info.Danger != "read-only" || len(info.Args) == 0 {
		t.Fatalf("command.describe: %+v, %v", info, err)
	}
	var schema map[string]any
	if err := json.Unmarshal(info.Args, &schema); err != nil || schema["$schema"] != schemaDialect {
		t.Errorf("command.describe's schema: %s, %v", info.Args, err)
	}

	msgs := collect(r.Dispatch(t.Context(), Request{ID: idQuit, Origin: OriginKey}))
	if len(msgs) != 2 || !slices.Contains(msgs, tea.Msg(QuitRequestMsg{})) || resultOf(t, msgs).Err != nil {
		t.Errorf("app.quit sent %v, want a QuitRequestMsg and its ResultMsg", msgs)
	}
	if _, err := r.Run(t.Context(), Request{ID: idQuit, Origin: OriginCLI}); !errors.Is(err, ErrUnavailable) {
		t.Errorf("app.quit from the shell: %v", err)
	}
	for id, d := range map[ID]Danger{idList: ReadOnly, idDescribe: ReadOnly, idQuit: UI} {
		if c, _ := r.Lookup(id); c.Danger != d {
			t.Errorf("%s is %s, want %s", id, c.Danger, d)
		}
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
