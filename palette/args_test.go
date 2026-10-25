package palette

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/theme"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// deployArgs has one parameter of each kind the argument stage asks for,
// in this order: a required string, an integer with a default and bounds, a
// required boolean, an enum with a default, a secret, and a hidden one it
// skips.
type deployArgs struct {
	Target string `json:"target" help:"where to deploy" placeholder:"HOST"`
	Count  int    `json:"count" default:"1" schema:"min=1,max=9" help:"how many copies"`
	Force  bool   `json:"force" help:"replace what runs"`
	Mode   string `json:"mode" enum:"fast,safe" default:"safe"`
	Token  string `json:"token,omitempty" schema:"secret" help:"the API token"`
	Trace  string `json:"trace,omitempty" hidden:""`
}

// tagArgs has an array, which the stage does not ask for.
type tagArgs struct {
	Tags []string `json:"tags"`
}

// deploying is a registry with app.deploy, whose handler records the
// arguments it was run with, and app.tag.
func deploying(t *testing.T) (*command.Registry, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var got []string
	reg := command.NewRegistry()
	if err := reg.Register(
		command.MustNew("app.deploy", "Deploy", func(_ context.Context, inv *command.Invocation, _ deployArgs) (command.Result, error) {
			mu.Lock()
			got = append(got, string(inv.Args))
			mu.Unlock()
			return command.Result{Text: "deployed"}, nil
		}, command.WithDanger(command.Mutating)),
		command.MustNew("app.tag", "Tag release", func(context.Context, *command.Invocation, tagArgs) (command.Result, error) {
			mu.Lock()
			got = append(got, "tagged")
			mu.Unlock()
			return command.Result{}, nil
		}, command.WithDanger(command.Mutating)),
	); err != nil {
		t.Fatal(err)
	}
	return reg, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...)
	}
}

// spying is a provider whose hits record the arguments the picker runs
// them with, before their own Run.
type spying struct {
	Provider
	got *[]string
}

func (s spying) Search(ctx context.Context, q Search, yield func(Hit) bool) error {
	return s.Provider.Search(ctx, q, func(h Hit) bool {
		run := h.Run
		h.Run = func(a jsontext.Value) tea.Cmd {
			*s.got = append(*s.got, string(a))
			return run(a)
		}
		return yield(h)
	})
}

// help is the picker's help line.
func (g *prog) help() string {
	lines := strings.Split(g.view(), "\n")
	return lines[len(lines)-1]
}

// TestArgumentStage: a command's parameters are asked for in order; a
// required one refuses an empty value, an integer one a word and a value
// outside its bounds, each with the reason in the help line; an empty
// value takes the default; a boolean and an enum are chosen from the list,
// which starts on the default; a hidden one is skipped; a secret is drawn
// masked; and the command runs with the values as one JSON object.
func TestArgumentStage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reg, got := deploying(t)
		var logs bytes.Buffer
		f := NewFrecency()
		g := newProg(t, keys(t), nil, WithRecents(f), WithLogger(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
		g.reg = reg
		var sent []string
		g.register(spying{FromRegistry(reg), &sent})
		g.open()
		g.typed("deploy")
		g.key("enter")

		if !strings.Contains(g.view(), "target: HOST") || !strings.Contains(g.help(), "where to deploy") || !strings.Contains(g.help(), "argument 1 of 5") {
			t.Fatalf("the first parameter's frame:\n%s", g.view())
		}
		g.key("enter")
		if !strings.Contains(g.help(), "target: required") {
			t.Fatalf("an empty required value: help %q", g.help())
		}
		g.typed("web")
		g.key("enter")

		for _, c := range []struct{ typed, why string }{
			{"x", "count: not a whole number"},
			{"0", "count: at least 1"},
			{"12", "count: at most 9"},
		} {
			g.typed(c.typed)
			g.key("enter")
			if !strings.Contains(g.help(), c.why) {
				t.Fatalf("count %q: help %q; want %q", c.typed, g.help(), c.why)
			}
			g.key("esc") // clears the refused value
		}
		if !strings.Contains(g.view(), "count: count (1)") {
			t.Fatalf("count's placeholder shows its default:\n%s", g.view())
		}
		g.key("enter") // empty: the default

		if rows := texts(g); len(rows) != 2 || rows[0] != "true" || rows[1] != "false" {
			t.Fatalf("force's list is %q", rows)
		}
		g.key("down")
		g.key("enter")

		if rows := texts(g); len(rows) != 2 || g.p.sel != 1 {
			t.Fatalf("mode's list is %q, selection %d; want it on the default, safe", rows, g.p.sel)
		}
		g.key("enter")

		g.typed("hunter2")
		if v := g.view(); strings.Contains(v, "hunter2") || !strings.Contains(v, "token: *******") {
			t.Fatalf("the secret is drawn:\n%s", v)
		}
		g.key("enter")

		want := `{"target":"web","count":1,"force":false,"mode":"safe","token":"hunter2"}`
		if len(sent) != 1 || sent[0] != want {
			t.Fatalf("the picker ran the hit with %q; want %s", sent, want)
		}
		if a := got(); len(a) != 1 || !strings.Contains(a[0], `"target":"web"`) {
			t.Fatalf("the command ran with %q; want once, with the values", a)
		}
		if sel := filter[SelectedMsg](g.seen); len(sel) != 1 || sel[0].Hit.ID != "app.deploy" {
			t.Fatalf("SelectedMsg %+v", sel)
		}
		if g.p.id != "" {
			t.Fatal("the picker is still open")
		}
		var stored bytes.Buffer
		if _, err := f.WriteTo(&stored); err != nil {
			t.Fatal(err)
		}
		for what, s := range map[string]string{"the recents": stored.String(), "the log": logs.String()} {
			if strings.Contains(s, "hunter2") {
				t.Errorf("the secret is in %s: %s", what, s)
			}
		}
		if f.Score("app.deploy") != 1 {
			t.Errorf("recents: app.deploy scored %v; want 1", f.Score("app.deploy"))
		}
	})
}

// texts is the rows the picker draws: its providers', or the stage's.
func texts(g *prog) []string {
	var out []string
	for _, r := range g.p.rows() {
		out = append(out, r.hit.Text)
	}
	return out
}

// TestArgumentsFallThrough: a hit with a parameter the stage does not ask
// for, an array here, and a command whose parameters cannot be read, send
// SelectedMsg and run nothing; a hit whose parameters are all hidden runs at
// once, with no arguments.
func TestArgumentsFallThrough(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reg, got := deploying(t)
		g := newProg(t, keys(t), nil)
		g.reg = reg
		g.register(FromRegistry(reg))
		g.open()
		g.typed("tag")
		g.key("enter")
		if sel := filter[SelectedMsg](g.seen); len(sel) != 1 || sel[0].Hit.ID != "app.tag" || g.p.id != "" {
			t.Fatalf("an array parameter: SelectedMsg %+v, open %q; want app.tag, closed", sel, g.p.id)
		}
		if a := got(); len(a) != 0 {
			t.Fatalf("an array parameter ran %q", a)
		}
	})

	bad := command.Command{ID: "bad.schema", Title: "Bad schema", Danger: command.UI, Args: command.Schema(`{"type":`)}
	if h := (&registry{reg: command.NewRegistry()}).hit(bad, fuzzy.Score{}, nil); h.Run != nil || h.Params != nil {
		t.Fatalf("a schema that cannot be read: Run set %v, Params %v; want neither", h.Run != nil, h.Params)
	}

	synctest.Test(t, func(t *testing.T) {
		var ran []string
		record := func(a jsontext.Value) tea.Cmd { ran = append(ran, string(a)); return nil }
		hidden := Hit{ID: "h", Text: "hidden only", Params: []command.Param{{Name: "x", Type: "string", Hidden: true}}, Run: record}
		tags := Hit{ID: "t", Text: "tags of a list", Params: []command.Param{{Name: "tags", Type: "array", Items: "string", Enum: []any{"a", "b"}}}, Run: record}
		g := newProg(t, keys(t), nil)
		g.register(&fake{name: "f", search: yields(hidden)})
		g.open()
		g.typed("hidden")
		g.key("enter")
		if len(ran) != 1 || ran[0] != "" || g.p.id != "" {
			t.Fatalf("all hidden: ran %q, open %q; want one run with no arguments, closed", ran, g.p.id)
		}
		ran = nil
		g = newProg(t, keys(t), nil)
		g.register(&fake{name: "f", search: yields(tags)})
		g.open()
		g.typed("tags")
		g.key("enter")
		if len(ran) != 0 || g.p.id != "" || g.p.args != nil {
			t.Fatalf("an array with enum items: ran %q, open %q; want nothing run, closed", ran, g.p.id)
		}
	})

	// A completion pop-up asks for nothing: the pane uses the hit.
	synctest.Test(t, func(t *testing.T) {
		h := Hit{ID: "d", Text: "deploy", Params: []command.Param{{Name: "target", Type: "string", Required: true}}}
		g := newProg(t, keys(t), nil, WithoutInput())
		g.register(&fake{name: "f", search: yields(h)})
		g.open()
		g.send(SetQuery("dep"))
		g.settle()
		g.key("enter")
		if sel := filter[SelectedMsg](g.seen); len(sel) != 1 || g.p.args != nil || g.p.id != "" {
			t.Fatalf("completion: SelectedMsg %+v, stage %v, open %q; want one, no stage, closed", sel, g.p.args != nil, g.p.id)
		}
	})
}

// TestArgumentsGolden: the argument stage's frames across the matrix
// {colour, no colour} × {UTF-8, ASCII} at 60 and 100 columns, each line
// within its width: a list of values starting on the default, a secret as
// it is typed, drawn masked, and a refused value with its reason. No frame
// holds the secret.
func TestArgumentsGolden(t *testing.T) {
	min1 := 1.0
	scenes := []struct {
		name  string
		param command.Param
		typed string
		enter bool
	}{
		{"choice", command.Param{Name: "mode", Type: "string", Enum: []any{"fast", "safe", "slow"}, HasDefault: true, Default: "safe", Description: "How to roll the release out"}, "", false},
		{"secret", command.Param{Name: "token", Type: "string", Secret: true, Required: true, Description: "The API token, never kept"}, "hunter2", false},
		{"refused", command.Param{Name: "count", Type: "integer", Min: &min1, Placeholder: "N", Description: "How many copies"}, "0", true},
	}
	synctest.Test(t, func(t *testing.T) {
		for _, s := range scenes {
			tuitest.Golden(t, "picker-args-"+s.name, tuitest.Matrix{Widths: []int{60, 100}}, func(c tuitest.Case) string {
				th := theme.New(c.Profile(), theme.Unknown, c.Glyphs())
				g := newProg(t, keys(t), []workspace.Option{workspace.WithTheme(th)})
				g.send(tea.WindowSizeMsg{Width: c.Width + 2, Height: 14})
				g.register(&fake{name: "commands", search: yields(Hit{ID: "app.deploy", Text: "Deploy", Params: []command.Param{s.param}})})
				g.open(WithSize(c.Width+2, 12))
				g.typed("deploy")
				g.key("enter")
				g.typed(s.typed)
				if s.enter {
					g.key("enter")
				}
				out := g.p.View(g.p.width, g.p.height)
				tuitest.Fits(t, out, c.Width, g.w.WidthMethod())
				if strings.Contains(out, "hunter2") {
					t.Errorf("%s: the frame holds the secret:\n%s", s.name, out)
				}
				g.send(workspace.Close("palette"))
				g.settle()
				return out
			})
		}
	})
}

// TestArgumentChoices: an enum's values are filtered by what is typed,
// tab completes the selected one, and a value that matches nothing is
// refused; esc on an empty value closes the picker, running nothing.
func TestArgumentChoices(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var ran []string
		h := Hit{ID: "m", Text: "Set mode", Params: []command.Param{{Name: "mode", Type: "string", Required: true, Enum: []any{"fast", "safe", "slow"}}},
			Run: func(a jsontext.Value) tea.Cmd { ran = append(ran, string(a)); return nil }}
		g := newProg(t, keys(t), nil)
		g.register(&fake{name: "f", search: yields(h)})
		g.open()
		g.typed("mode")
		g.key("enter")
		g.typed("sl")
		if rows := texts(g); len(rows) != 1 || rows[0] != "slow" {
			t.Fatalf("sl: %q; want slow", rows)
		}
		g.key("tab")
		if g.p.input.Value() != "slow" {
			t.Fatalf("tab: the input is %q", g.p.input.Value())
		}
		g.typed("zz")
		g.key("enter")
		if !strings.Contains(g.help(), "mode: pick one of its values") {
			t.Fatalf("no match: help %q", g.help())
		}
		g.key("esc")
		g.key("enter")
		if len(ran) != 1 || ran[0] != `{"mode":"fast"}` {
			t.Fatalf("ran %q; want the first value", ran)
		}

		ran = nil
		g.open()
		g.typed("mode")
		g.key("enter")
		g.key("esc")
		if g.p.id != "" || len(ran) != 0 {
			t.Fatalf("esc on an empty value: open %q, ran %q; want closed, nothing run", g.p.id, ran)
		}
		g.send(workspace.Close("palette"))
		g.settle()
	})
}
