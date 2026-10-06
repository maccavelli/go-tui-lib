package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/maccavelli/go-tui-lib/tuitest"
)

var userSrc = Source{Kind: User}

// describe is c as the LoadDir golden lists it.
func describeCmd(c *Command) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", c.ID)
	for _, kv := range [][2]string{
		{"title", c.Title}, {"description", c.Description}, {"slash", c.Slash},
		{"aliases", strings.Join(c.Aliases, ", ")}, {"category", c.Category},
		{"kind", c.Kind.String()}, {"danger", c.Danger.String()}, {"when", c.When},
		{"hidden", fmt.Sprint(c.Hidden)}, {"arghint", c.ArgHint}, {"source", c.Source.String()},
		{"args", string(c.Args)},
	} {
		if kv[1] != "" {
			fmt.Fprintf(&b, "  %s: %s\n", kv[0], kv[1])
		}
	}
	return b.String()
}

func TestLoadDirGolden(t *testing.T) {
	cmds, errs := LoadDir(os.DirFS("testdata/commands"), userSrc)
	if len(errs) != 0 {
		t.Fatalf("errors: %v", errs)
	}
	var b strings.Builder
	for i := range cmds {
		b.WriteString(describeCmd(&cmds[i]))
	}
	tuitest.Text(t, "loaddir", b.String())

	r := NewRegistry()
	if conflicts := r.ReplaceSource(userSrc, cmds); len(conflicts) != 0 {
		t.Fatalf("conflicts: %+v", conflicts)
	}
	req, err := r.ParseSlash("/git:commit parser terse")
	if err != nil {
		t.Fatal(err)
	}
	if string(req.Args) != `{"SCOPE":"parser","STYLE":"terse"}` {
		t.Errorf("placeholders are not positional in order of first use: %s", req.Args)
	}
}

// TestArgumentsTakesRest: a file that uses $ARGUMENTS takes any slash
// tail, its positionals first and the rest left in Raw (A6, D23).
func TestArgumentsTakesRest(t *testing.T) {
	cmds, errs := LoadDir(os.DirFS("testdata/commands"), userSrc)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	r := NewRegistry()
	r.ReplaceSource(userSrc, cmds)
	for line, want := range map[string][2]string{
		"/notes 100 or so":                     {`{}`, "# Summarise the notes\n\nSummarise the notes so far, in 100 or so words."},
		"/git:commit parser terse and a=b too": {`{"SCOPE":"parser","STYLE":"terse"}`, "Write a commit message for parser in the terse style, then parser again.\n\nNotes: parser terse and a=b too"},
	} {
		req, err := r.ParseSlash(line)
		if err != nil || string(req.Args) != want[0] {
			t.Errorf("%q: %s, %v; want %s", line, req.Args, err, want[0])
			continue
		}
		if res, err := r.Run(t.Context(), req); err != nil || res.Text != want[1] {
			t.Errorf("%q expands to %q, %v; want %q", line, res.Text, err, want[1])
		}
	}
	two, _ := LoadDir(fstest.MapFS{"two.md": {Data: []byte("$A and $B")}}, Source{Kind: Project})
	r.ReplaceSource(Source{Kind: Project}, two)
	if _, err := r.ParseSlash("/two x y and more"); err == nil {
		t.Error("a file without $ARGUMENTS took stray words")
	}
}

func TestFrontMatterErrors(t *testing.T) {
	fsys := fstest.MapFS{}
	cases := map[string]string{
		"unknown.md":   "---\ntitle: x\nauthor: me\n---\nbody",
		"dup.md":       "---\ntitle: x\ndescription: y\ntitle: z\n---\nbody",
		"nested.md":    "---\ntitle: x\naliases:\n  - a\n---\nbody",
		"flow.md":      "---\naliases: [a, b]\n---\nbody",
		"open.md":      "---\ntitle: x\n",
		"danger.md":    "---\ndanger: risky\n---\nbody",
		"when.md":      "---\ntitle: x\nwhen: a &&\n---\nbody",
		"hidden.md":    "---\nhidden: yes\n---\nbody",
		"slash.md":     "---\nslash: a b\n---\nbody",
		"alias.md":     "---\naliases: a, /b\n---\nbody",
		"noarg.md":     "---\narg.FOCUS: what\n---\nbody without it",
		"notkv.md":     "---\njust words\n---\nbody",
		"emptyval.md":  "---\ntitle: \"\"\n---\nbody",
		"nullname.md":  "---\ntitle: x\n---\nbody",
		"bad-utf8.md":  "\xff\xfe",
		"___.md":       "a name with no letters",
		"lower.md":     "---\narg.focus: what\n---\n$FOCUS",
		"blankkey.md":  "---\n: value\n---\nbody",
		"pipe.md":      "---\ndescription: |\n  more\n---\nbody",
		"dangerOK.md":  "---\ndanger: ui\n---\nfine",
		"crlf-ok.md":   "---\r\ntitle: x\r\n---\r\nbody",
		"no-fm-ok.md":  "plain body",
		"quote-ok.md":  "---\ndescription: \"Review: staged\"\n---\nbody",
		"comment-x.md": "---\n# a comment\n---\nbody",
	}
	want := map[string]string{
		"unknown.md": ":3:", "dup.md": ":4:", "nested.md": ":3:", "flow.md": ":2:", "open.md": ":1:",
		"danger.md": ":2:", "when.md": ":3:", "hidden.md": ":2:", "slash.md": ":2:", "alias.md": ":2:",
		"noarg.md": ":2:", "notkv.md": ":2:", "emptyval.md": ":2:", "bad-utf8.md": "not UTF-8",
		"___.md": "no letter or digit", "lower.md": ":2:", "blankkey.md": ":2:", "pipe.md": ":2:",
		"comment-x.md": ":2:",
	}
	for name, body := range cases {
		fsys[name] = &fstest.MapFile{Data: []byte(body)}
	}
	cmds, errs := LoadDir(fsys, userSrc)
	got := map[string]string{}
	for _, err := range errs {
		msg := err.Error()
		for name := range cases {
			if strings.HasPrefix(msg, "command: "+name+":") {
				got[name] = msg
			}
		}
	}
	for name, frag := range want {
		if !strings.Contains(got[name], frag) {
			t.Errorf("%s: %q, want an error naming the file and %q", name, got[name], frag)
		}
	}
	if len(errs) != len(want) {
		t.Errorf("%d errors, want %d: %v", len(errs), len(want), errs)
	}
	loaded := map[ID]bool{}
	for _, c := range cmds {
		loaded[c.ID] = true
	}
	for _, id := range []ID{"user.dangerok", "user.crlf-ok", "user.no-fm-ok", "user.quote-ok", "user.nullname"} {
		if !loaded[id] {
			t.Errorf("%s did not load", id)
		}
	}
	for _, c := range cmds {
		if c.ID == "user.quote-ok" && c.Description != "Review: staged" {
			t.Errorf("a quoted value: %q", c.Description)
		}
	}
}

func TestNoFrontMatter(t *testing.T) {
	cmds, errs := LoadDir(os.DirFS("testdata/commands"), userSrc)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	for _, c := range cmds {
		if c.ID == "user.notes" {
			if c.Kind != Prompt || c.Description != "Summarise the notes" || c.Title != "notes" {
				t.Errorf("notes.md: %s", describeCmd(&c))
			}
			return
		}
	}
	t.Error("notes.md did not load")
}

func TestDefaultDanger(t *testing.T) {
	cmds, _ := LoadDir(fstest.MapFS{"x.md": {Data: []byte("---\ntitle: x\n---\nbody")}, "y.md": {Data: []byte("body")}}, userSrc)
	if len(cmds) != 2 {
		t.Fatalf("%d commands", len(cmds))
	}
	for _, c := range cmds {
		if c.Danger != Mutating {
			t.Errorf("%s is %s, want mutating", c.ID, c.Danger)
		}
	}
}

func TestRootRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "commands")
	if err := os.Mkdir(cmdDir, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(dir, "outside.md")
	for p, body := range map[string]string{outside: "outside the root", filepath.Join(cmdDir, "ok.md"): "inside"} {
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(cmdDir, "leak.md")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	root, err := os.OpenRoot(cmdDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	cmds, errs := LoadDir(root.FS(), userSrc)
	if len(cmds) != 1 || cmds[0].ID != "user.ok" {
		t.Errorf("loaded %v", cmds)
	}
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "leak.md") {
		t.Errorf("errors %v, want one naming leak.md", errs)
	}
}

func TestLoadDirSources(t *testing.T) {
	fsys := fstest.MapFS{"x.md": {Data: []byte("body")}}
	for src, want := range map[Source]ID{
		{Kind: Project}:                   "project.x",
		{Kind: Plugin, Name: "My Plugin"}: "plugin.my-plugin.x",
	} {
		cmds, errs := LoadDir(fsys, src)
		if len(errs) != 0 || len(cmds) != 1 || cmds[0].ID != want || cmds[0].Source != src {
			t.Errorf("%s: %v %v, want %s", src, cmds, errs, want)
		}
	}
	for _, src := range []Source{{Kind: Builtin}, {Kind: MCP, Name: "s"}, {Kind: ACP, Name: "a"}, {Kind: Plugin}} {
		if _, errs := LoadDir(fsys, src); len(errs) != 1 {
			t.Errorf("%s: %v, want one error", src, errs)
		}
	}
}
