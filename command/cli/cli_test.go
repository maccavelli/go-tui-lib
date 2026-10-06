package cli_test

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/command/cli"
	"github.com/maccavelli/go-tui-lib/tuitest"
	"github.com/maccavelli/go-tui-lib/when"
)

type resizeArgs struct {
	Split string `json:"split" arg:"" help:"the split whose separator moves" placeholder:"SPLIT"`
	Delta int    `json:"delta" arg:"" help:"cells to give the pane before it; negative takes" schema:"min=-200,max=200"`
	Quiet bool   `json:"quiet,omitzero" short:"q" group:"Output" help:"say less"`
}

type kindsArgs struct {
	Name   string         `json:"name" help:"a name"`
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

// fixture is a registry for the shell, and the last arguments each
// command was run with.
type fixture struct {
	r    *command.Registry
	args map[command.ID]string
}

func mk[A any](t *testing.T, f *fixture, id command.ID, title string, run func(A) (command.Result, error), opts ...command.Option) command.Command {
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

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{r: command.NewRegistry(), args: map[command.ID]string{}}
	ui := command.WithDanger(command.UI)
	cmds := []command.Command{
		mk(t, f, "workspace.resize", "Move a split", func(a resizeArgs) (command.Result, error) {
			return command.Result{Value: a, Text: "moved " + a.Split}, nil
		}, ui, command.WithCategory("Workspace"), command.WithDescription("Moves a named split's separator by delta cells, as far as the panes' sizes allow.")),
		mk(t, f, "files.delete", "Delete a file", func(a pathArgs) (command.Result, error) {
			return command.Result{Text: "deleted " + a.Path}, nil
		}, command.WithDanger(command.Destructive), command.WithCategory("Files")),
		mk(t, f, "tools.echo", "Echo the arguments", func(a kindsArgs) (command.Result, error) {
			return command.Result{Value: a, Text: "echo"}, nil
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

// run runs args and returns the exit code, stdout and stderr.
func (f *fixture) run(args string, o ...cli.Option) (int, string, string) {
	var out, errOut bytes.Buffer
	code := cli.Run(context.Background(), f.r, splitArgs(args), &out, &errOut, append([]cli.Option{cli.WithName("app")}, o...)...)
	return code, out.String(), errOut.String()
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

func TestHelpGolden(t *testing.T) {
	f := newFixture(t)
	for _, width := range []int{60, 100} {
		for name, args := range map[string]string{"registry": "help", "resize": "help workspace resize", "echo": "tools echo --help"} {
			code, out, errOut := f.run(args, cli.WithWidth(width))
			if code != cli.ExitOK || errOut != "" {
				t.Fatalf("%s: exit %d, stderr %q", args, code, errOut)
			}
			for i := range len(out) {
				if out[i] >= 0x80 {
					t.Fatalf("%s at %d: help is not ASCII: %q", args, width, out)
				}
			}
			tuitest.Text(t, "help-"+name+"."+itoa(width), out)
		}
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestFlagsEqualArgs(t *testing.T) {
	f := newFixture(t)
	want := `{"delta":4,"split":"sidebar"}`
	for _, line := range []string{
		"workspace resize --split sidebar --delta 4",
		"workspace resize sidebar 4",
		"workspace.resize --args '{\"split\":\"sidebar\",\"delta\":4}'",
		"workspace resize --delta=4 sidebar",
		"workspace resize -split=sidebar 4",
	} {
		code, _, errOut := f.run(line)
		if code != cli.ExitOK || f.args["workspace.resize"] != want {
			t.Errorf("%s: exit %d %q; args %s, want %s", line, code, errOut, f.args["workspace.resize"], want)
		}
	}
	if code, _, _ := f.run("workspace resize sidebar 4 -q"); code != cli.ExitOK || f.args["workspace.resize"] != `{"delta":4,"quiet":true,"split":"sidebar"}` {
		t.Errorf("a short flag: %s", f.args["workspace.resize"])
	}
}

func TestFlagKinds(t *testing.T) {
	f := newFixture(t)
	code, _, errOut := f.run("tools echo --name x --n 3 --on --mode slow --tags a --tags b --state '{\"a\":1}'")
	want := `{"mode":"slow","n":3,"name":"x","on":true,"state":{"a":1},"tags":["a","b"]}`
	if code != cli.ExitOK || f.args["tools.echo"] != want {
		t.Errorf("exit %d %q; args %s, want %s", code, errOut, f.args["tools.echo"], want)
	}
	if code, _, _ := f.run("tools echo --name x"); code != cli.ExitOK || f.args["tools.echo"] != `{"mode":"fast","name":"x"}` {
		t.Errorf("the default: %s", f.args["tools.echo"])
	}
	for line, frag := range map[string]string{
		"tools echo --name x --n many":        "argument /n",
		"tools echo --name x --on=maybe":      "argument /on",
		"tools echo --name x --mode medium":   "argument /mode",
		"tools echo --name x --state notjson": "argument /state",
		"tools echo --name x --bogus 1":       "bogus",
		"tools echo --name x stray":           "positional",
		"tools echo --n 1":                    "argument /name",
		"tools echo --name x --name y":        "given twice",
		"tools echo --args '[1]'":             "--args",
		"workspace resize sidebar 999":        "argument /delta",
	} {
		code, out, errOut := f.run(line)
		if code != cli.ExitUsage || !strings.Contains(errOut, frag) || out != "" {
			t.Errorf("%s: exit %d, stdout %q, stderr %q; want exit 2 naming %q", line, code, out, errOut, frag)
		}
	}
}

func TestJSONOutput(t *testing.T) {
	f := newFixture(t)
	if _, out, _ := f.run("workspace resize sidebar 4 --json"); out != `{"split":"sidebar","delta":4}`+"\n" {
		t.Errorf("--json: %q", out)
	}
	if _, out, _ := f.run("workspace resize sidebar 4"); out != "moved sidebar\n" {
		t.Errorf("text: %q", out)
	}
	if _, out, _ := f.run("tools text --json"); out != "null\n" {
		t.Errorf("--json with no value: %q, want null", out)
	}
}

func TestDestructiveNeedsYes(t *testing.T) {
	f := newFixture(t)
	code, out, errOut := f.run("files delete notes.txt")
	if code != cli.ExitRefused || out != "" || !strings.Contains(errOut, "--yes") {
		t.Errorf("without --yes: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	if code, out, _ := f.run("files delete notes.txt --yes"); code != cli.ExitOK || out != "deleted notes.txt\n" {
		t.Errorf("with --yes: exit %d %q", code, out)
	}
	var asked []string
	confirm := func(answer bool) cli.Option {
		return cli.WithConfirm(func(p string) bool { asked = append(asked, p); return answer })
	}
	if code, _, _ := f.run("files delete a", confirm(true)); code != cli.ExitOK {
		t.Errorf("confirmed: exit %d", code)
	}
	if code, _, _ := f.run("files delete a", confirm(false)); code != cli.ExitRefused {
		t.Errorf("declined: exit %d", code)
	}
	if len(asked) != 2 || !strings.Contains(asked[0], "files delete") {
		t.Errorf("prompts: %q", asked)
	}
	if code, _, _ := f.run("tools text", confirm(false)); code != cli.ExitOK || len(asked) != 2 {
		t.Errorf("a command that is not destructive asked: %q", asked)
	}
}

func TestCLISurface(t *testing.T) {
	f := newFixture(t)
	_, list, _ := f.run("list")
	_, help, _ := f.run("help")
	for _, out := range []string{list, help} {
		if strings.Contains(out, "tui only") || strings.Contains(out, "secret cmd") || strings.Contains(out, "app quit") {
			t.Errorf("a command off the CLI surface, or hidden, is listed:\n%s", out)
		}
		if !strings.Contains(out, "workspace resize") || !strings.Contains(out, "command list") {
			t.Errorf("a CLI command is missing:\n%s", out)
		}
	}
	for _, line := range []string{"tui only", "tui.only", "app quit"} {
		if code, _, errOut := f.run(line); code != cli.ExitUsage || !strings.Contains(errOut, "unknown command") {
			t.Errorf("%s: exit %d %q, want an unknown command", line, code, errOut)
		}
	}
	if code, out, _ := f.run("secret cmd"); code != cli.ExitOK || out != "found\n" {
		t.Errorf("a hidden command does not run: %d %q", code, out)
	}
}

func TestExitCodes(t *testing.T) {
	f := newFixture(t)
	ready := cli.WithContext(when.Map{"ready": when.BoolValue(true)})
	for _, c := range []struct {
		line string
		o    []cli.Option
		want int
	}{
		{"tools text", nil, cli.ExitOK},
		{"tools fail", nil, cli.ExitFailed},
		{"tools later", nil, cli.ExitFailed},
		{"tools later", []cli.Option{ready}, cli.ExitOK},
		{"no such", nil, cli.ExitUsage},
		{"", nil, cli.ExitUsage},
		{"files delete x", nil, cli.ExitRefused},
		{"describe", nil, cli.ExitUsage},
		{"list --bogus", nil, cli.ExitUsage},
	} {
		if code, _, _ := f.run(c.line, c.o...); code != c.want {
			t.Errorf("%q: exit %d, want %d", c.line, code, c.want)
		}
	}
}

func TestVerbs(t *testing.T) {
	f := newFixture(t)
	_, out, _ := f.run("list --json")
	var m command.Manifest
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("list --json: %v\n%s", err, out)
	}
	var ids []command.ID
	for _, c := range m.Commands {
		ids = append(ids, c.ID)
	}
	if !slices.Contains(ids, "workspace.resize") || slices.Contains(ids, "tui.only") || slices.Contains(ids, "secret.cmd") || m.Format != 1 {
		t.Errorf("list --json: %v", ids)
	}
	if _, out, _ := f.run("describe workspace resize"); !strings.Contains(out, `"id":"workspace.resize"`) {
		t.Errorf("describe: %s", out)
	}
	if _, out, _ := f.run("schema workspace.resize"); !strings.Contains(out, `"split"`) || !strings.Contains(out, "2020-12") {
		t.Errorf("schema: %s", out)
	}
	if code, _, _ := f.run("schema tui only"); code != cli.ExitUsage {
		t.Errorf("schema of a TUI-only command: exit %d", code)
	}
}

func TestWritersOnly(t *testing.T) {
	f := newFixture(t)
	code, out, errOut := f.run("tools fail")
	if code != cli.ExitFailed || out != "" || !strings.Contains(errOut, "it broke") {
		t.Errorf("a failure: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	code, out, errOut = f.run("tools text")
	if code != cli.ExitOK || out != "hello\n" || errOut != "" {
		t.Errorf("a success: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	code, out, errOut = f.run("")
	if code != cli.ExitUsage || out != "" || !strings.Contains(errOut, "Usage:") {
		t.Errorf("no arguments: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
}
