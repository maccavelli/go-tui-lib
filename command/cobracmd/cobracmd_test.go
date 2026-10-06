package cobracmd_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/command/cli"
	"github.com/maccavelli/go-tui-lib/command/cobracmd"
	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/tuitest"
)

// find is the command at path in root's tree, or nil.
func find(root *cobra.Command, path ...string) *cobra.Command {
	c, rest, err := root.Find(path)
	if err != nil || len(rest) > 0 || c == root {
		return nil
	}
	return c
}

func TestTree(t *testing.T) {
	f := newFixture(t)
	root := f.tree(t)
	if c := find(root, "tui", "only"); c != nil {
		t.Errorf("a command off the CLI surface is in the tree: %s", c.CommandPath())
	}
	if find(root, "tui") != nil {
		t.Error("the namespace of a command off the CLI surface is in the tree")
	}
	secret := find(root, "secret", "cmd")
	if secret == nil || !secret.Hidden || !find(root, "secret").Hidden {
		t.Fatalf("a hidden command is missing, or not hidden: %v", secret)
	}
	if code, out, _ := run(root, "secret cmd"); code != cli.ExitOK || out != "found\n" {
		t.Errorf("a hidden command does not run: %d %q", code, out)
	}
	resize := find(root, "workspace", "resize")
	if resize == nil {
		t.Fatal("workspace resize is missing")
	}
	want := map[string]string{"command.id": "workspace.resize", "command.danger": "ui", "command.surfaces": command.AllSurfaces.String(), "cobracmd.node": "command"}
	for k, v := range want {
		if resize.Annotations[k] != v {
			t.Errorf("annotation %s = %q, want %q", k, resize.Annotations[k], v)
		}
	}
	if resize.Use != "resize SPLIT N" || resize.Short != "Move a split" || !slices.Equal(resize.Aliases, []string{"rs"}) {
		t.Errorf("resize: Use %q, Short %q, Aliases %q", resize.Use, resize.Short, resize.Aliases)
	}
	if c := find(root, "workspace", "rs"); c != resize {
		t.Error("the slash alias does not find the command")
	}
	for _, flag := range []string{"split", "delta", "quiet", "args", "json", "yes"} {
		if resize.Flags().Lookup(flag) == nil {
			t.Errorf("resize has no --%s", flag)
		}
	}
	if resize.Flags().ShorthandLookup("q") == nil {
		t.Error("quiet has no -q")
	}
	if h := find(root, "tools", "echo").Flags().Lookup("secret"); h == nil || !h.Hidden {
		t.Error("a hidden property's flag is missing, or shown")
	}
}

func TestGroups(t *testing.T) {
	f := newFixture(t)
	root := f.tree(t)
	var groups []string
	for _, g := range root.Groups() {
		groups = append(groups, g.ID)
	}
	if !slices.Equal(groups, []string{"Files", "Workspace"}) {
		t.Errorf("the root's groups are %q", groups)
	}
	for _, path := range [][]string{{"workspace"}, {"files"}, {"workspace", "resize"}, {"files", "delete"}} {
		c := find(root, path...)
		if c == nil || c.GroupID == "" || !c.Parent().ContainsGroup(c.GroupID) {
			t.Errorf("%q: group %q is not its parent's", path, c.GroupID)
		}
	}
	if g := find(root, "tools").GroupID; g != "" {
		t.Errorf("tools, of commands with no category, is in group %q", g)
	}
	// Cobra checks the groups when it runs.
	if code, _, _ := run(root, "workspace resize sidebar 4"); code != cli.ExitOK {
		t.Errorf("exit %d", code)
	}
}

func TestMount(t *testing.T) {
	f := newFixture(t)
	served := 0
	program := func() *cobra.Command {
		root := &cobra.Command{Use: "prog"}
		root.AddGroup(&cobra.Group{ID: "Workspace", Title: "Workspace commands:"})
		root.AddCommand(&cobra.Command{Use: "serve", RunE: func(*cobra.Command, []string) error { served++; return nil }})
		return root
	}
	root := program()
	if err := cobracmd.Mount(root, f.r); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := run(root, "serve"); code != cli.ExitOK || served != 1 {
		t.Errorf("the program's own command: exit %d, served %d", code, served)
	}
	if code, out, _ := run(root, "workspace resize sidebar 4 --json"); code != cli.ExitOK || out != `{"split":"sidebar","delta":4}`+"\n" {
		t.Errorf("a mounted command: exit %d %q", code, out)
	}
	if code, out, _ := run(root, "workspace.resize sidebar 4"); code != cli.ExitOK || out != "moved sidebar\n" {
		t.Errorf("a mounted command by its ID: exit %d %q", code, out)
	}
	if code, out, _ := run(root, "list"); code != cli.ExitOK || !strings.Contains(out, "workspace resize") {
		t.Errorf("list: exit %d %q", code, out)
	}
	n := 0
	for _, g := range root.Groups() {
		if g.ID == "Workspace" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("the program's group Workspace is there %d times", n)
	}
	for _, clash := range []string{"list", "workspace", "describe"} {
		root := program()
		root.AddCommand(&cobra.Command{Use: clash})
		before := len(root.Commands())
		err := cobracmd.Mount(root, f.r)
		if err == nil || !strings.Contains(err.Error(), `"`+clash+`"`) {
			t.Errorf("a program with %q: %v", clash, err)
		}
		if len(root.Commands()) != before {
			t.Errorf("a failed Mount changed the program's commands")
		}
	}
	under := program()
	tools := &cobra.Command{Use: "tools2"}
	under.AddCommand(tools)
	if err := cobracmd.Mount(tools, f.r); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := run(under, "tools2 workspace.resize sidebar 4"); code != cli.ExitOK || out != "moved sidebar\n" {
		t.Errorf("a command mounted below the root, by its ID: exit %d %q", code, out)
	}
}

func TestEnumFlag(t *testing.T) {
	f := newFixture(t)
	root := f.tree(t)
	code, out, errOut := run(root, "tools echo --name x --mode medium")
	if code != cli.ExitUsage || out != "" || !strings.Contains(errOut, "argument /mode") || !strings.Contains(errOut, "fast, slow") {
		t.Errorf("a value not in the enum: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	for line, want := range map[string]string{
		"__complete tools echo --mode ":   "fast\nslow\n:4\n",
		"__complete tools level ":         "low\nhigh\n:4\n",
		"__complete tools level low ":     ":0\n",
		"__complete describe workspace.r": "workspace.resize\tMove a split\n:4\n",
	} {
		args := splitArgs(line)
		if strings.HasSuffix(line, " ") {
			args = append(args, "")
		}
		var b, e bytes.Buffer
		cobracmd.Run(context.Background(), root, args, nil, &b, &e)
		if got, _, _ := strings.Cut(b.String(), "Completion ended"); got != want {
			t.Errorf("%q: completions %q, want %q", line, got, want)
		}
	}
}

func TestHelpGolden(t *testing.T) {
	f := newFixture(t)
	for name, line := range map[string]string{
		"root": "help", "resize": "workspace resize --help", "echo": "help tools.echo",
		"level": "tools level -h", "namespace": "workspace --help", "list": "list --help",
		"delete": "files delete --help",
	} {
		tuitest.Golden(t, "help-"+name, tuitest.Matrix{Widths: []int{60, 100}}, func(c tuitest.Case) string {
			p := colorprofile.ASCII
			if c.Color {
				p = colorprofile.TrueColor
			}
			th := theme.New(p, theme.Dark, glyph.For(c.UTF8))
			code, out, errOut := run(f.tree(t, cobracmd.WithTheme(th), cobracmd.WithWidth(c.Width)), line)
			if code != cli.ExitOK || errOut != "" {
				t.Fatalf("%s: exit %d, stderr %q", line, code, errOut)
			}
			return out
		})
	}
}

func TestHelpDefaults(t *testing.T) {
	f := newFixture(t)
	_, out, _ := run(f.tree(t), "help")
	for i := range len(out) {
		if out[i] >= 0x80 || out[i] == 0x1b {
			t.Fatalf("the default help is not plain ASCII: %q", out)
		}
	}
	_, unicode, _ := run(f.tree(t, cobracmd.WithGlyphs(glyph.Unicode())), "tools level --help")
	if !strings.Contains(unicode, "[VALUE…]") {
		t.Errorf("WithGlyphs is not used:\n%s", unicode)
	}
}

func TestCompletionScripts(t *testing.T) {
	f := newFixture(t)
	root := f.tree(t)
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		code, out, errOut := run(root, "completion "+shell)
		if code != cli.ExitOK || errOut != "" || !strings.Contains(out, "app") {
			t.Fatalf("%s: exit %d, stderr %q, stdout %.300q", shell, code, errOut, out)
		}
		tuitest.Text(t, "completion-"+shell, out)
	}
}

func TestExitCodesAndNamespaces(t *testing.T) {
	f := newFixture(t)
	root := f.tree(t)
	for line, want := range map[string]int{
		"workspace":           cli.ExitUsage,
		"workspace bogus":     cli.ExitUsage,
		"help":                cli.ExitOK,
		"help workspace":      cli.ExitOK,
		"help bogus":          cli.ExitUsage,
		"help workspace.nope": cli.ExitUsage,
		"--bogus":             cli.ExitUsage,
		"completion bash":     cli.ExitOK,
	} {
		if code, _, _ := run(root, line); code != want {
			t.Errorf("%q: exit %d, want %d", line, code, want)
		}
	}
	code, out, errOut := run(root, "workspace")
	if out != "" || !strings.Contains(errOut, "workspace resize") || !strings.Contains(errOut, "Usage:") {
		t.Errorf("a bare namespace: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	if _, _, errOut := run(root, "workspace bogus"); !strings.Contains(errOut, `unknown command "workspace bogus"`) {
		t.Errorf("an unknown command in a namespace: %q", errOut)
	}
}

// TestRunsAreIndependent runs one tree several times: Cobra keeps flag
// values between executions, and Run clears them.
func TestRunsAreIndependent(t *testing.T) {
	f := newFixture(t)
	root := f.tree(t)
	if code, _, _ := run(root, "files delete a --yes"); code != cli.ExitOK {
		t.Fatalf("with --yes: exit %d", code)
	}
	if code, _, _ := run(root, "files delete a"); code != cli.ExitRefused {
		t.Errorf("--yes outlived its run: exit %d", code)
	}
	if code, out, _ := run(root, "tools text --help"); code != cli.ExitOK || !strings.Contains(out, "Usage:") {
		t.Fatalf("--help: exit %d %q", code, out)
	}
	if _, out, _ := run(root, "tools text"); out != "hello\n" {
		t.Errorf("--help outlived its run: %q", out)
	}
	run(root, "tools echo --name x --tags a --json")
	if _, out, _ := run(root, "tools echo --name y"); out != "echo\n" || f.args["tools.echo"] != `{"mode":"fast","name":"y"}` {
		t.Errorf("a flag outlived its run: %q, %s", out, f.args["tools.echo"])
	}
}

func TestDestructiveNeedsYes(t *testing.T) {
	f := newFixture(t)
	root := f.tree(t)
	code, out, errOut := run(root, "files delete notes.txt")
	if code != cli.ExitRefused || out != "" || !strings.Contains(errOut, "--yes") {
		t.Errorf("without --yes: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	var asked []string
	confirmed := f.tree(t, cobracmd.WithConfirm(func(p string) bool { asked = append(asked, p); return true }))
	if code, _, _ := run(confirmed, "files delete a"); code != cli.ExitOK || len(asked) != 1 || !strings.Contains(asked[0], "files delete") {
		t.Errorf("confirmed: exit %d, asked %q", code, asked)
	}
}

// TestNoProcessState shows that Run reads no os.Args, and writes to no
// standard stream.
func TestNoProcessState(t *testing.T) {
	f := newFixture(t)
	root := f.tree(t)
	saved := os.Args
	t.Cleanup(func() { os.Args = saved })
	os.Args = []string{"prog", "tools", "text"}
	if code, out, _ := run(root, "list"); code != cli.ExitOK || !strings.Contains(out, "workspace resize") {
		t.Errorf("list ran something else: exit %d %q", code, out)
	}
	var out, errOut bytes.Buffer
	if code := cobracmd.Run(context.Background(), root, nil, nil, &out, &errOut); code != cli.ExitUsage || out.Len() != 0 || !strings.Contains(errOut.String(), "Usage:") {
		t.Errorf("no arguments ran os.Args: exit %d, stdout %q", code, out.String())
	}

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
	for _, line := range []string{"tools text", "tools fail", "help", "workspace resize --help", "no such", "tools echo --bogus", "completion zsh", "__complete tools echo --mode ''", "list --json"} {
		run(root, line)
	}
	os.Stdout, os.Stderr = stdout, stderr
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
			t.Errorf("Run wrote to the process's %s: %q", name, b)
		}
	}
}

// failing is a writer that fails every write.
type failing struct{}

func (failing) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestWriteFails(t *testing.T) {
	f := newFixture(t)
	root := f.tree(t)
	for _, line := range []string{"tools text", "help", "list --json"} {
		if code := cobracmd.Run(context.Background(), root, splitArgs(line), nil, failing{}, io.Discard); code != cli.ExitFailed {
			t.Errorf("%q to a failing stdout: exit %d", line, code)
		}
	}
}

func TestSecondNew(t *testing.T) {
	f := newFixture(t)
	first, second := f.tree(t), f.tree(t)
	for _, root := range []*cobra.Command{first, second, first} {
		var out, errOut bytes.Buffer
		cobracmd.Run(context.Background(), root, []string{"__complete", "tools", "echo", "--mode", ""}, nil, &out, &errOut)
		if !strings.HasPrefix(out.String(), "fast\nslow\n") {
			t.Errorf("completion in a second tree: %q %q", out.String(), errOut.String())
		}
	}
	if err := cobracmd.Mount(&cobra.Command{Use: "other"}, f.r); err != nil {
		t.Errorf("a third tree: %v", err)
	}
}

func TestBrokenSchema(t *testing.T) {
	r := command.NewRegistry()
	type clashArgs struct {
		JSON bool `json:"json,omitzero" help:"a property named like a shell flag"`
	}
	c, err := command.New("tools.clash", "Clash", func(context.Context, *command.Invocation, clashArgs) (command.Result, error) {
		return command.Result{Text: "ran"}, nil
	}, command.WithDanger(command.UI))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Register(c); err != nil {
		t.Fatal(err)
	}
	root, err := cobracmd.New(r)
	if err != nil {
		t.Fatalf("one command's clash broke the tree: %v", err)
	}
	if code, _, errOut := run(root, "tools clash"); code != cli.ExitUsage || !strings.Contains(errOut, "a flag the shell keeps") {
		t.Errorf("a clashing property: exit %d %q", code, errOut)
	}
	if code, _, _ := run(root, "list"); code != cli.ExitOK {
		t.Errorf("the rest of the tree: exit %d", code)
	}
}

func TestNilArguments(t *testing.T) {
	if _, err := cobracmd.New(nil); err == nil {
		t.Error("New(nil) built a tree")
	}
	if err := cobracmd.Mount(nil, command.NewRegistry()); err == nil {
		t.Error("Mount into nil")
	}
}
