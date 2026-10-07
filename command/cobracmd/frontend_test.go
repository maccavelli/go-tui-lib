package cobracmd_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/command/cli"
	"github.com/maccavelli/go-tui-lib/command/cobracmd"
	"github.com/maccavelli/go-tui-lib/when"
)

// The shared front-end cases: the same command lines, run through
// command/cli and this package, build the same request, write the same
// stdout and return the same exit code
// (docs/decisions/0006-MADR-command-registry.md A1, "One conformance test
// for three front ends"). kongcmd carries a copy of them.

type resizeArgs struct {
	Split string `json:"split" arg:"" help:"the split whose separator moves" placeholder:"SPLIT"`
	Delta int    `json:"delta" arg:"" help:"cells to give the pane before it; negative takes" schema:"min=-200,max=200"`
	Quiet bool   `json:"quiet,omitzero" short:"q" group:"Output" help:"say less"`
}

type kindsArgs struct {
	Name   string         `json:"name" required:"" help:"a name"`
	N      int            `json:"n,omitzero" help:"a count"`
	On     bool           `json:"on,omitzero" help:"a switch"`
	Mode   string         `json:"mode,omitzero" enum:"fast,slow" default:"fast" help:"how"`
	Tags   []string       `json:"tags,omitzero" help:"labels"`
	State  map[string]int `json:"state,omitzero" help:"a map"`
	Secret string         `json:"secret,omitzero" hidden:"" help:"not in help"`
}

type pathArgs struct {
	Path string `json:"path" arg:"" help:"the file"`
}

type levelArgs struct {
	Level string   `json:"level" arg:"" enum:"low,high" help:"how loud"`
	Words []string `json:"words,omitzero" arg:"" optional:"" help:"what to say"`
}

// fixture is a registry for the shell, and the last arguments each
// command was run with.
type fixture struct {
	r    *command.Registry
	args map[command.ID]string
}

func mk[A any](t testing.TB, f *fixture, id command.ID, title string, run func(A) (command.Result, error), opts ...command.Option) command.Command {
	t.Helper()
	c, err := command.New(id, title, func(_ context.Context, inv *command.Invocation, a A) (command.Result, error) {
		f.args[id] = string(inv.Args)
		return run(a)
	}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func newFixture(t testing.TB) *fixture {
	t.Helper()
	f := &fixture{r: command.NewRegistry(), args: map[command.ID]string{}}
	ui := command.WithDanger(command.UI)
	cmds := []command.Command{
		mk(t, f, "workspace.resize", "Move a split", func(a resizeArgs) (command.Result, error) {
			return command.Result{Value: a, Text: "moved " + a.Split}, nil
		}, ui, command.WithCategory("Workspace"), command.WithSlash("resize", "rs"),
			command.WithDescription("Moves a named split's separator by delta cells, as far as the panes' sizes allow.")),
		mk(t, f, "files.delete", "Delete a file", func(a pathArgs) (command.Result, error) {
			return command.Result{Text: "deleted " + a.Path}, nil
		}, command.WithDanger(command.Destructive), command.WithCategory("Files")),
		mk(t, f, "tools.echo", "Echo the arguments", func(a kindsArgs) (command.Result, error) {
			return command.Result{Value: a, Text: "echo"}, nil
		}, ui),
		mk(t, f, "tools.level", "Say something at a level", func(a levelArgs) (command.Result, error) {
			return command.Result{Value: a, Text: a.Level + ": " + strings.Join(a.Words, " ")}, nil
		}, ui),
		mk(t, f, "tools.text", "Say hello", func(command.NoArgs) (command.Result, error) {
			return command.Result{Text: "hello"}, nil
		}, ui),
		mk(t, f, "tools.fail", "Fail", func(command.NoArgs) (command.Result, error) {
			return command.Result{}, errors.New("it broke")
		}, ui),
		mk(t, f, "tools.later", "Not yet", func(command.NoArgs) (command.Result, error) {
			return command.Result{Text: "now"}, nil
		}, ui, command.WithWhen("ready")),
		mk(t, f, "tui.only", "In the TUI only", func(command.NoArgs) (command.Result, error) {
			return command.Result{}, nil
		}, ui, command.WithSurfaces(command.SurfaceKey|command.SurfacePalette)),
		mk(t, f, "secret.cmd", "Hidden", func(command.NoArgs) (command.Result, error) {
			return command.Result{Text: "found"}, nil
		}, ui, command.WithHidden()),
	}
	if err := f.r.Register(cmds...); err != nil {
		t.Fatal(err)
	}
	return f
}

// splitArgs splits a test's command line at spaces, keeping 'quoted'
// words whole.
func splitArgs(s string) []string {
	var out []string
	for i, part := range strings.Split(s, "'") {
		if i%2 == 1 {
			out = append(out, part)
			continue
		}
		out = append(out, strings.Fields(part)...)
	}
	return out
}

// tree is a fixture's registry under New.
func (f *fixture) tree(t testing.TB, o ...cobracmd.Option) *cobra.Command {
	t.Helper()
	root, err := cobracmd.New(f.r, append([]cobracmd.Option{cobracmd.WithName("app")}, o...)...)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// run runs a command line through root, and returns the exit code, stdout
// and stderr.
func run(root *cobra.Command, line string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := cobracmd.Run(context.Background(), root, splitArgs(line), nil, &out, &errOut)
	return code, out.String(), errOut.String()
}

// sharedCases are command lines in both front ends' common syntax, and the
// options each runs with.
var sharedCases = []struct {
	line  string
	ready bool
	yes   *bool // WithConfirm's answer, when set
}{
	{line: "workspace resize --split sidebar --delta 4"},
	{line: "workspace resize sidebar 4"},
	{line: `workspace.resize --args '{"split":"sidebar","delta":4}'`},
	{line: "workspace resize --delta=4 sidebar"},
	{line: "workspace resize sidebar 4 -q"},
	{line: "workspace resize sidebar 4 --json"},
	{line: "workspace.resize sidebar 4 --json"},
	{line: "workspace resize sidebar 999"},
	{line: `tools echo --name x --n 3 --on --mode slow --tags a --tags b --state '{"a":1}'`},
	{line: "tools echo --name x --json"},
	{line: "tools echo --name x --n many"},
	{line: "tools echo --name x --on=maybe"},
	{line: "tools echo --name x --mode medium"},
	{line: "tools echo --name x --state notjson"},
	{line: "tools echo --name x --bogus 1"},
	{line: "tools echo --name x stray"},
	{line: "tools echo --n 1"},
	{line: "tools echo --name x --name y"},
	{line: "tools echo --args '[1]'"},
	{line: `tools echo --args '{"name":"y","n":2}' --n 5 --json`},
	{line: "tools level high say it loud --json"},
	{line: "tools level medium"},
	{line: "tools level --level low --words a --words b --json"},
	{line: "tools text"},
	{line: "tools text --json"},
	{line: "tools fail"},
	{line: "tools later"},
	{line: "tools later", ready: true},
	{line: "files delete notes.txt"},
	{line: "files delete notes.txt --yes"},
	{line: "files delete notes.txt", yes: new(true)},
	{line: "files delete notes.txt", yes: new(false)},
	{line: "secret cmd"},
	{line: "tui only"},
	{line: "tui.only"},
	{line: "app quit"},
	{line: "no such"},
	{line: ""},
	{line: "list"},
	{line: "list --json"},
	{line: "list --bogus"},
	{line: "list extra"},
	{line: "describe workspace resize"},
	{line: "describe workspace.resize"},
	{line: "describe"},
	{line: "describe tui only"},
	{line: "schema workspace.resize"},
	{line: "schema tools text"},
	{line: "schema tui only"},
	{line: "command list --json"},
	{line: "command describe --id workspace.resize --json"},
}

func TestSharedFrontEndCases(t *testing.T) {
	viaCLI, viaCobra := newFixture(t), newFixture(t)
	for _, c := range sharedCases {
		var cliOpts []cli.Option
		var cobraOpts []cobracmd.Option
		if c.ready {
			ctx := when.Map{"ready": when.BoolValue(true)}
			cliOpts = append(cliOpts, cli.WithContext(ctx))
			cobraOpts = append(cobraOpts, cobracmd.WithContext(ctx))
		}
		if c.yes != nil {
			answer := *c.yes
			cliOpts = append(cliOpts, cli.WithConfirm(func(string) bool { return answer }))
			cobraOpts = append(cobraOpts, cobracmd.WithConfirm(func(string) bool { return answer }))
		}
		clear(viaCLI.args)
		clear(viaCobra.args)
		var out, errOut bytes.Buffer
		wantCode := cli.Run(context.Background(), viaCLI.r, splitArgs(c.line), &out, &errOut, append([]cli.Option{cli.WithName("app")}, cliOpts...)...)
		code, gotOut, gotErr := run(viaCobra.tree(t, cobraOpts...), c.line)
		if code != wantCode {
			t.Errorf("%q: exit %d, command/cli %d\nstderr: %s\ncommand/cli's: %s", c.line, code, wantCode, gotErr, errOut.String())
		}
		if gotOut != out.String() {
			t.Errorf("%q: stdout\n%s\ncommand/cli's\n%s", c.line, gotOut, out.String())
		}
		if (gotErr == "") != (errOut.Len() == 0) {
			t.Errorf("%q: stderr %q, command/cli's %q", c.line, gotErr, errOut.String())
		}
		for id, args := range viaCLI.args {
			if viaCobra.args[id] != args {
				t.Errorf("%q: %s ran with %s, command/cli's %s", c.line, id, viaCobra.args[id], args)
			}
		}
		if len(viaCobra.args) != len(viaCLI.args) {
			t.Errorf("%q: ran %v, command/cli %v", c.line, viaCobra.args, viaCLI.args)
		}
	}
}
