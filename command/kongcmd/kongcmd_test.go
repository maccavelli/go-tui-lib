package kongcmd_test

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/command/cli"
	"github.com/maccavelli/go-tui-lib/command/kongcmd"
	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/tuitest"
	"github.com/maccavelli/go-tui-lib/when"
)

// spellArgs has the Go names command's spelling of Kong's names must get
// right: an acronym, digits, a single letter, and a json name that is not
// Kong's spelling, given name:"".
type spellArgs struct {
	PaneID     string `json:"pane-id,omitzero" help:"an acronym"`
	HTTPServer string `json:"http-server,omitzero" help:"an acronym before a word"`
	Base64Data string `json:"base-64-data,omitzero" help:"digits"`
	X          bool   `json:"x,omitzero" help:"one letter"`
	MaxItems   int    `json:"max_items,omitzero" name:"max_items" help:"a name tag"`
}

// readers are the argument structs of the fixture and spellArgs, as
// command.New accepts them.
var readers = map[string]func() (any, command.Schema, error){
	"resize": reader[resizeArgs],
	"kinds":  reader[kindsArgs],
	"level":  reader[levelArgs],
	"path":   reader[pathArgs],
	"spell":  reader[spellArgs],
}

func reader[A any]() (any, command.Schema, error) {
	if _, err := command.New("t.cmd", "T", func(context.Context, *command.Invocation, A) (command.Result, error) {
		return command.Result{}, nil
	}, command.WithDanger(command.UI)); err != nil {
		return nil, nil, err
	}
	s, err := command.SchemaOf[A]()
	return new(A), s, err
}

// read is what a Kong model says of a struct's arguments: each name's
// requiredness, choices, default and, for a positional, its place.
type read struct {
	required bool
	enum     string
	def      string
	position int // -1 for a flag
}

func readKong(t *testing.T, n *kong.Node) map[string]read {
	t.Helper()
	out := map[string]read{}
	for _, f := range n.Flags {
		if slices.Contains([]string{"help", "args", "json", "yes"}, f.Name) {
			continue
		}
		out[f.Name] = read{f.Required, f.Enum, f.Default, -1}
	}
	for i, p := range n.Positional {
		out[p.Name] = read{p.Required, p.Enum, p.Default, i}
	}
	return out
}

// readSchema is what a schema says of the same.
func readSchema(t *testing.T, s command.Schema) map[string]read {
	t.Helper()
	var doc struct {
		Properties map[string]struct {
			Enum    []any `json:"enum"`
			Default any   `json:"default"`
			CLI     struct {
				Arg bool `json:"arg"`
			} `json:"x-cli"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(s, &doc); err != nil {
		t.Fatal(err)
	}
	// Positionals take the properties' document order (A5).
	var order []string
	dec := strings.Split(string(s), `"properties":{`)[1]
	for name := range doc.Properties {
		order = append(order, name)
	}
	slices.SortFunc(order, func(a, b string) int {
		return strings.Index(dec, `"`+a+`":`) - strings.Index(dec, `"`+b+`":`)
	})
	out := map[string]read{}
	pos := 0
	for _, name := range order {
		p := doc.Properties[name]
		r := read{required: slices.Contains(doc.Required, name), position: -1}
		var enum []string
		for _, e := range p.Enum {
			b, _ := json.Marshal(e)
			enum = append(enum, strings.Trim(string(b), `"`))
		}
		r.enum = strings.Join(enum, ",")
		if p.Default != nil {
			b, _ := json.Marshal(p.Default)
			r.def = strings.Trim(string(b), `"`)
		}
		if p.CLI.Arg {
			r.position = pos
			pos++
		}
		out[name] = r
	}
	return out
}

// TestOneStructTwoReaders reads each argument struct New accepts directly
// in Kong, and through this package's grammar, and compares both with its
// schema: names, requiredness, choices, defaults and positional order
// (A1, A12, D47).
func TestOneStructTwoReaders(t *testing.T) {
	for name, get := range readers {
		grammar, schema, err := get()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := readSchema(t, schema)
		k, err := kong.New(grammar, kong.Name("x"), kong.Exit(func(int) {}), kong.Writers(io.Discard, io.Discard))
		if err != nil {
			t.Fatalf("%s: Kong refuses the struct: %v", name, err)
		}
		direct := readKong(t, k.Model.Node)
		if !reflect.DeepEqual(direct, want) {
			t.Errorf("%s read directly by Kong:\n%v\nthe schema:\n%v", name, direct, want)
		}

		r := command.NewRegistry()
		c, err := command.New("t.cmd", "T", func(context.Context, *command.Invocation, command.NoArgs) (command.Result, error) {
			return command.Result{}, nil
		}, command.WithDanger(command.UI))
		if err != nil {
			t.Fatal(err)
		}
		c.Args = schema
		if err := r.Register(c); err != nil {
			t.Fatal(err)
		}
		a, err := kongcmd.New(r)
		if err != nil {
			t.Fatal(err)
		}
		p, err := a.Parser()
		if err != nil {
			t.Fatalf("%s: the adapter's grammar: %v", name, err)
		}
		node := byID(p.Model.Node, "t.cmd")
		if node == nil {
			t.Fatalf("%s: no node has the ID t.cmd", name)
		}
		adapted := readKong(t, node)
		if !reflect.DeepEqual(adapted, want) {
			t.Errorf("%s through the adapter:\n%v\nthe schema:\n%v", name, adapted, want)
		}
	}
}

// byID is the node below n whose id tag is id.
func byID(n *kong.Node, id string) *kong.Node {
	if n.Tag != nil && n.Tag.Get("id") == id {
		return n
	}
	for _, c := range n.Children {
		if found := byID(c, id); found != nil {
			return found
		}
	}
	return nil
}

func TestNestedSelection(t *testing.T) {
	f := newFixture(t)
	s := f.shell(t)
	for line, want := range map[string]string{
		"workspace focus logs":          "focused logs\n",
		"workspace focus next":          "next\n",
		"workspace.focus logs":          "focused logs\n",
		"workspace resize sidebar -q 2": "moved sidebar\n",
		"workspace rs sidebar 2":        "moved sidebar\n",
	} {
		if code, out, errOut := s.run(line); code != cli.ExitOK || out != want {
			t.Errorf("%q: exit %d, stdout %q, stderr %q; want %q", line, code, out, errOut, want)
		}
	}
	if code, _, _ := s.run("workspace focus"); code != cli.ExitUsage {
		t.Errorf("a command missing its positional: exit %d", code)
	}
}

func TestHelpStopsParse(t *testing.T) {
	f := newFixture(t)
	s := f.shell(t)
	// --name is required: a parse that went on after the help would fail.
	code, out, errOut := s.run("tools echo --help")
	if code != cli.ExitOK || errOut != "" || !strings.Contains(out, "Usage:") {
		t.Errorf("--help without a required flag: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	if _, ok := f.args["tools.echo"]; ok {
		t.Error("the command ran after --help")
	}
	if code, out, _ := s.run("help workspace.resize"); code != cli.ExitOK || !strings.Contains(out, "SPLIT N") {
		t.Errorf("help by ID: exit %d %q", code, out)
	}
	if code, _, _ := s.run("help no such"); code != cli.ExitUsage {
		t.Errorf("help of an unknown command: exit %d", code)
	}
}

// program is a Kong program's own grammar.
type program struct {
	Serve struct{} `cmd:"" help:"Serve"`
	Debug bool     `help:"debug output"`
}

func TestMount(t *testing.T) {
	f := newFixture(t)
	a, err := kongcmd.New(f.r, kongcmd.WithName("prog"))
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	var g program
	p, err := kong.New(&g, append(a.Options(), kong.Writers(&out, &errOut))...)
	if err != nil {
		t.Fatal(err)
	}
	kctx, code, handled := a.Run(context.Background(), p, []string{"--debug", "serve"})
	if handled || code != cli.ExitOK || kctx == nil || kctx.Command() != "serve" || !g.Debug {
		t.Errorf("the program's own command: handled %v, exit %d, %v", handled, code, kctx)
	}
	if _, code, handled := a.Run(context.Background(), p, []string{"workspace", "resize", "sidebar", "4"}); !handled || code != cli.ExitOK || out.String() != "moved sidebar\n" {
		t.Errorf("a registry command: handled %v, exit %d, %q", handled, code, out.String())
	}
	var clash struct {
		List struct{} `cmd:""`
	}
	if _, err := kong.New(&clash, a.Options()...); err == nil || !strings.Contains(err.Error(), `"list"`) {
		t.Errorf("a program with its own list: %v", err)
	}
	var clashAlias struct {
		Thing struct{} `cmd:"" aliases:"workspace"`
	}
	if _, err := kong.New(&clashAlias, a.Options()...); err == nil || !strings.Contains(err.Error(), `"workspace"`) {
		t.Errorf("a program with an alias the registry takes: %v", err)
	}
}

func TestHelpGolden(t *testing.T) {
	t.Setenv("COLUMNS", "20") // Kong's own printer would read it
	f := newFixture(t)
	for name, line := range map[string]string{
		"root": "--help", "resize": "workspace resize --help", "echo": "help tools.echo",
		"level": "tools level -h", "namespace": "workspace --help", "focus": "workspace focus --help",
		"list": "list --help", "delete": "files delete --help",
	} {
		tuitest.Golden(t, "help-"+name, tuitest.Matrix{Widths: []int{60, 100}}, func(c tuitest.Case) string {
			p := colorprofile.ASCII
			if c.Color {
				p = colorprofile.TrueColor
			}
			th := theme.New(p, theme.Dark, glyph.For(c.UTF8))
			code, out, errOut := f.shell(t, kongcmd.WithTheme(th), kongcmd.WithWidth(c.Width)).run(line)
			if code != cli.ExitOK || errOut != "" {
				t.Fatalf("%s: exit %d, stderr %q", line, code, errOut)
			}
			for l := range strings.Lines(out) {
				if w := len([]rune(strings.TrimRight(stripANSI(l), "\n"))); w > c.Width {
					t.Errorf("%s at %d: a line %d wide: %q", line, c.Width, w, l)
				}
			}
			return out
		})
	}
}

// stripANSI removes SGR sequences.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func TestHelpProgramNode(t *testing.T) {
	f := newFixture(t)
	a, err := kongcmd.New(f.r, kongcmd.WithName("prog"))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	var g program
	p, err := kong.New(&g, append(a.Options(), kong.Writers(&out, &out))...)
	if err != nil {
		t.Fatal(err)
	}
	a.Run(context.Background(), p, []string{"--help"})
	tuitest.Text(t, "help-mounted", out.String())
}

func TestCompletionScripts(t *testing.T) {
	f := newFixture(t)
	s := f.shell(t)
	for _, sh := range []string{"bash", "zsh", "fish", "powershell"} {
		code, out, errOut := s.run("completion " + sh)
		if code != cli.ExitOK || errOut != "" || !strings.Contains(out, "app __complete") && !strings.Contains(out, "__complete") {
			t.Fatalf("%s: exit %d, stderr %q", sh, code, errOut)
		}
		tuitest.Text(t, "completion-"+sh, out)
	}
	if code, _, _ := s.run("completion tcsh"); code != cli.ExitUsage {
		t.Errorf("an unknown shell: exit %d", code)
	}
}

func TestComplete(t *testing.T) {
	f := newFixture(t)
	s := f.shell(t)
	for line, want := range map[string][]string{
		"__complete ":                   {"command\tThe command commands", "files\tThe files commands", "tools\tThe tools commands", "workspace\tThe workspace commands", "list\tList the commands you can run now", "describe\tDescribe a command as JSON", "schema\tPrint a command's arguments schema", "help\tShow the help of the program or of a command", "completion\tGenerate the completion script for a shell"},
		"__complete wor":                {"workspace\tThe workspace commands"},
		"__complete tools echo --mode ": {"fast", "slow"},
		"__complete tools echo --m":     {"--mode\thow"},
		"__complete tools level ":       {"low", "high"},
		"__complete tools level low ":   nil,
		"__complete workspace ":         {"focus\tFocus a pane", "resize\tMove a split"},
		"__complete workspace focus ":   {"next\tFocus the next pane"},
		"__complete completion ":        {"bash", "zsh", "fish", "powershell"},
	} {
		code, out, _ := s.run(line)
		got := slices.Collect(strings.Lines(out))
		for i := range got {
			got[i] = strings.TrimSuffix(got[i], "\n")
		}
		if code != cli.ExitOK || !sameSet(got, want) {
			t.Errorf("%q: exit %d, %q, want %q", line, code, got, want)
		}
		if strings.Contains(out, "secret") {
			t.Errorf("%q completes a hidden command: %q", line, out)
		}
	}
}

func sameSet(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

// TestNoProcessState shows that Run reads no os.Args, and that nothing
// reaches the process's standard streams, from a parser given no
// writers either.
func TestNoProcessState(t *testing.T) {
	f := newFixture(t)
	a, err := kongcmd.New(f.r)
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Args
	t.Cleanup(func() { os.Args = saved })
	os.Args = []string{"prog", "tools", "text"}

	stdout, stderr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = wOut, wErr
	p, err := a.Parser()
	if err == nil {
		for _, line := range []string{"tools text", "tools fail", "--help", "workspace resize --help", "no such", "tools echo --bogus", "completion zsh", "__complete tools echo --mode ", "list --json", ""} {
			a.Run(context.Background(), p, splitArgs(line))
		}
		a.Run(context.Background(), p, nil)
	}
	os.Stdout, os.Stderr = stdout, stderr
	if err != nil {
		t.Fatal(err)
	}
	if err := wOut.Close(); err != nil {
		t.Fatal(err)
	}
	if err := wErr.Close(); err != nil {
		t.Fatal(err)
	}
	for name, r := range map[string]io.Reader{"stdout": rOut, "stderr": rErr} {
		b, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		if len(b) > 0 {
			t.Errorf("a parser without writers wrote to the process's %s: %q", name, b)
		}
	}
	if _, ok := f.args["tools.text"]; !ok {
		t.Error("tools text did not run")
	}
	clear(f.args)
	s := f.shell(t)
	if code, out, errOut := s.run(""); code != cli.ExitUsage || out != "" || errOut == "" {
		t.Errorf("no arguments ran os.Args: exit %d, stdout %q", code, out)
	}
	if len(f.args) > 0 {
		t.Errorf("no arguments ran %v", f.args)
	}
}

func TestResolver(t *testing.T) {
	f := newFixture(t)
	ctx := when.Map{"config.mode": when.StringValue("slow"), "config.tags": when.ListValue([]string{"a", "b"}), "config.debug": when.BoolValue(true)}
	a, err := kongcmd.New(f.r, kongcmd.WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	var g program
	p, err := kong.New(&g, append(a.Options(), kong.Writers(&out, &out), kong.Resolvers(a.Resolver()))...)
	if err != nil {
		t.Fatal(err)
	}
	if _, code, _ := a.Run(context.Background(), p, []string{"tools", "echo", "--name", "x"}); code != cli.ExitOK || f.args["tools.echo"] != `{"mode":"slow","name":"x","tags":["a","b"]}` {
		t.Errorf("settings: exit %d, %s, %q", code, f.args["tools.echo"], out.String())
	}
	if _, code, _ := a.Run(context.Background(), p, []string{"tools", "echo", "--name", "x", "--mode", "fast"}); code != cli.ExitOK || f.args["tools.echo"] != `{"mode":"fast","name":"x","tags":["a","b"]}` {
		t.Errorf("the command line wins: exit %d, %s", code, f.args["tools.echo"])
	}
	if _, _, handled := a.Run(context.Background(), p, []string{"serve"}); handled || !g.Debug {
		t.Errorf("a program's own flag from settings: handled %v, debug %v", handled, g.Debug)
	}
}

func TestNoEnvTag(t *testing.T) {
	f := newFixture(t)
	p, err := f.shell(t).a.Parser()
	if err != nil {
		t.Fatal(err)
	}
	var walk func(n *kong.Node)
	walk = func(n *kong.Node) {
		for _, fl := range n.Flags {
			if len(fl.Envs) > 0 {
				t.Errorf("%s --%s reads the environment: %v", n.Path(), fl.Name, fl.Envs)
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(p.Model.Node)
}

// TestRunsAreIndependent runs one parser several times.
func TestRunsAreIndependent(t *testing.T) {
	f := newFixture(t)
	s := f.shell(t)
	if code, _, _ := s.run("files delete a --yes"); code != cli.ExitOK {
		t.Fatalf("with --yes: exit %d", code)
	}
	if code, _, _ := s.run("files delete a"); code != cli.ExitRefused {
		t.Errorf("--yes outlived its run: exit %d", code)
	}
	s.run("tools echo --name x --tags a --json")
	// The registry fills the schema's default, mode, itself.
	if _, out, _ := s.run("tools echo --name y"); out != "echo\n" || f.args["tools.echo"] != `{"mode":"fast","name":"y"}` {
		t.Errorf("a flag outlived its run: %q, %s", out, f.args["tools.echo"])
	}
}

func TestDestructiveNeedsYes(t *testing.T) {
	f := newFixture(t)
	code, out, errOut := f.shell(t).run("files delete notes.txt")
	if code != cli.ExitRefused || out != "" || !strings.Contains(errOut, "--yes") {
		t.Errorf("without --yes: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	var asked []string
	s := f.shell(t, kongcmd.WithConfirm(func(p string) bool { asked = append(asked, p); return true }))
	if code, _, _ := s.run("files delete a"); code != cli.ExitOK || len(asked) != 1 || !strings.Contains(asked[0], "files delete") {
		t.Errorf("confirmed: exit %d, asked %q", code, asked)
	}
}

// failing is a writer that fails every write.
type failing struct{}

func (failing) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestWriteFails(t *testing.T) {
	f := newFixture(t)
	a, err := kongcmd.New(f.r)
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.Parser(kong.Writers(failing{}, io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"tools text", "--help", "list --json", "__complete "} {
		if _, code, _ := a.Run(context.Background(), p, splitArgs(line)); code != cli.ExitFailed {
			t.Errorf("%q to a failing stdout: exit %d", line, code)
		}
	}
}

func TestBrokenCommand(t *testing.T) {
	r := command.NewRegistry()
	c, err := command.New("tools.clash", "Clash", func(context.Context, *command.Invocation, command.NoArgs) (command.Result, error) {
		return command.Result{Text: "ran"}, nil
	}, command.WithDanger(command.UI))
	if err != nil {
		t.Fatal(err)
	}
	c.Args = command.Schema(`{"type":"object","properties":{"json":{"type":"boolean"}}}`)
	if err := r.Register(c); err != nil {
		t.Fatal(err)
	}
	a, err := kongcmd.New(r)
	if err != nil {
		t.Fatalf("one command's clash broke the grammar: %v", err)
	}
	var out, errOut bytes.Buffer
	p, err := a.Parser(kong.Writers(&out, &errOut))
	if err != nil {
		t.Fatal(err)
	}
	if _, code, _ := a.Run(context.Background(), p, []string{"tools", "clash"}); code != cli.ExitUsage || !strings.Contains(errOut.String(), "a flag the shell keeps") {
		t.Errorf("a clashing property: exit %d %q", code, errOut.String())
	}
	if _, code, _ := a.Run(context.Background(), p, []string{"list"}); code != cli.ExitOK {
		t.Errorf("the rest of the grammar: exit %d", code)
	}
}

func TestChildNamedLikeParent(t *testing.T) {
	r := command.NewRegistry()
	for _, id := range []command.ID{"a.b", "a.b.b"} {
		c, err := command.New(id, "T", func(context.Context, *command.Invocation, command.NoArgs) (command.Result, error) {
			return command.Result{}, nil
		}, command.WithDanger(command.UI))
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Register(c); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := kongcmd.New(r); err == nil || !strings.Contains(err.Error(), "a.b") {
		t.Errorf("a command whose child is named like it: %v", err)
	}
}

func TestNilArguments(t *testing.T) {
	if _, err := kongcmd.New(nil); err == nil {
		t.Error("New(nil) built an adapter")
	}
}
