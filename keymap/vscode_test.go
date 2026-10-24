package keymap

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// marker marks, in a test's file, the byte an error must point at.
const marker = "‸"

// marked is src without its marker, and the marker's line and column, the
// column in bytes.
func marked(t *testing.T, src string) (string, int, int) {
	t.Helper()
	i := strings.Index(src, marker)
	if i < 0 || strings.Count(src, marker) != 1 {
		t.Fatalf("want one marker in %q", src)
	}
	before := src[:i]
	return before + src[i+len(marker):], strings.Count(before, "\n") + 1, i - strings.LastIndex(before, "\n")
}

// load is LoadVSCode over src, named "keys.json", and its errors, each of
// which must be a *LoadError.
func load(t *testing.T, src string, ids func(command.ID) bool) ([]Binding, []*LoadError) {
	t.Helper()
	bs, err := LoadVSCode(strings.NewReader(src), "keys.json", ids)
	var out []*LoadError
	for _, e := range unjoin(err) {
		le, ok := errors.AsType[*LoadError](e)
		if !ok {
			t.Fatalf("an error is %T, not a *LoadError: %v", e, e)
		}
		out = append(out, le)
	}
	return bs, out
}

// placeless is bs with no Location, for comparing what was loaded.
func placeless(bs []Binding) []Binding {
	out := slices.Clone(bs)
	for i := range out {
		out[i].Location = Location{}
	}
	return out
}

// known is an ids function accepting only the IDs given.
func known(ids ...command.ID) func(command.ID) bool {
	return func(id command.ID) bool { return slices.Contains(ids, id) }
}

// TestBlank: comments and trailing commas become spaces, line breaks and
// strings stay, and every byte keeps its offset.
func TestBlank(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"[1, // c\n2]", "[1,     \n2]"},
		{"[1, /* a\r\nb */ 2,]", "[1,     \r\n     2 ]"},
		{`["//", "/*", "a\"//"] // x`, `["//", "/*", "a\"//"]     `},
		{"{\"a\": 1, /* c */ }", "{\"a\": 1          }"},
		{"[1,\n// c\n]", "[1 \n    \n]"},
		{"[1, 2]", "[1, 2]"},
		{"\xef\xbb\xbf[]", "   []"},
		{"[1 / 2]", "[1 / 2]"},
	} {
		got, _, err := blank([]byte(c.in))
		if err != nil || string(got) != c.want {
			t.Errorf("blank(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	if _, at, err := blank([]byte("[1, /* open")); !errors.Is(err, errComment) || at != 4 {
		t.Errorf("an unterminated comment: at %d, %v; want 4, %v", at, err, errComment)
	}
}

// TestJSONC: line and block comments, comments holding a quote or "//",
// "//" and "/*" inside strings, and trailing commas are read.
func TestJSONC(t *testing.T) {
	src := `// keybindings, with "quotes" and // slashes in a comment
[
	/* a block "comment" // with slashes,
	   over two lines */
	{
		"key": "ctrl+k ctrl+s", // the save chord
		"command": "app.save", /* a comment, then a trailing comma */
		"when": "editorFocus",
	},
	{"key": "alt+/", "command": "app.comment", "args": {"text": "// not /* a comment", "q": "\"//"},},
	// a trailing comma, and a comment before the bracket
]
`
	bs, errs := load(t, src, nil)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	want := []Binding{
		{Keys: seq(t, "ctrl+k ctrl+s"), Command: "app.save", When: "editorFocus"},
		{Keys: seq(t, "alt+/"), Command: "app.comment", Args: jsontext.Value(`{"text":"// not /* a comment","q":"\"//"}`)},
	}
	if got := placeless(bs); !reflect.DeepEqual(got, want) {
		t.Errorf("loaded %+v\nwant %+v", got, want)
	}
	if l := bs[0].Location; l != (Location{Layer: UserLayer, File: "keys.json", Pointer: "/0", Line: 5, Col: 2}) {
		t.Errorf("the first binding's location is %+v", l)
	}
}

// TestErrorPlaces: each error names the file, and the line, column and
// pointer of the bad part, which a marker shows; a comment before it, on
// its line or over several, moves nothing.
func TestErrorPlaces(t *testing.T) {
	var syntax *jsontext.SyntacticError
	var whenErr *when.SyntaxError
	for _, c := range []struct {
		name, src string
		pointer   jsontext.Pointer
		is        func(error) bool
	}{
		{"a bad key, after a block comment over two lines", "[\n\t/* two\n\t   lines */ {\"key\": ‸\"ctrl+nope\", \"command\": \"app.x\"}\n]",
			"/0/key", func(err error) bool { return errors.Is(err, ErrNotation) }},
		{"an unknown member, after a line comment", "[{\"key\": \"a\", // c\n \"command\": \"app.x\", ‸\"keys\": \"b\"}]", "/0/keys", nil},
		{"a bad when", `[{"key": "a", "command": "app.x", "when": ‸"a &&"}]`,
			"/0/when", func(err error) bool { return errors.As(err, &whenErr) }},
		{"an unknown command", `[{"key": "a", "command": ‸"app.nope"}]`, "/0/command", nil},
		{"a malformed command", `[{"key": "a", "command": ‸"App.X"}]`, "/0/command", nil},
		{"a key that is no string", `[{"key": ‸1, "command": "app.x"}]`, "/0/key", nil},
		{"no key", `[{"key": "a", "command": "app.x"}, ‸{"command": "app.x"}]`, "/1", nil},
		{"no command", `[‸{"key": "a"}]`, "/0", nil},
		{"an element that is no object", `[{"key": "a", "command": "app.x"}, ‸3]`, "/1", nil},
		{"a member twice", `[{"key": "a", ‸"key": "b", "command": "app.x"}]`, "/0/key", nil},
		{"a member twice in args", `[{"key": "a", "command": "app.x", "args": {"n": {"m": 1, ‸"m": 2}}}]`,
			"/0/args/n/m", func(err error) bool { return errors.As(err, &syntax) }},
		{"an empty context", `[{"key": "a", "command": "app.x", "context": ‸""}]`, "/0/context", nil},
		{"a column counts bytes", `[{"key": "a", "command": "app.x", "args": "üü", ‸"x": 1}]`, "/0/x", nil},
		{"no array", `// c` + "\n" + `‸{"key": "a"}`, "", nil},
		{"a missing comma", `[{"key": "a" ‸"command": "app.x"}]`,
			"/0", func(err error) bool { return errors.As(err, &syntax) }},
		{"trailing data", `[] ‸x`, "", func(err error) bool { return errors.As(err, &syntax) }},
		{"an unterminated comment", "[\n  ‸/* never closed\n]", "", func(err error) bool { return errors.Is(err, errComment) }},
		{"the end of the file", "[{\"key\": \"a\", \"command\": \"app.x\"}‸", "", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			src, line, col := marked(t, c.src)
			_, errs := load(t, src, known("app.x"))
			if len(errs) != 1 {
				t.Fatalf("errors: %v; want one", errs)
			}
			e := errs[0]
			if e.File != "keys.json" || e.Line != line || e.Col != col || e.Pointer != c.pointer {
				t.Errorf("at %s:%d:%d %q; want keys.json:%d:%d %q: %v", e.File, e.Line, e.Col, e.Pointer, line, col, c.pointer, e)
			}
			if c.is != nil && !c.is(e) {
				t.Errorf("%v (%T) is not the kind of error wanted", e.Err, e.Err)
			}
		})
	}
}

// TestThreeErrors: three bad bindings give three errors, in the file's
// order, and the good bindings are returned.
func TestThreeErrors(t *testing.T) {
	src := `[
	{"key": "a", "command": "app.a"},
	{"key": "ctrl+", "command": "app.b"},
	{"key": "c", "command": "app.c"},
	{"key": "d", "command": "app.d", "colour": "red"},
	{"key": "e", "command": "app.e", "when": "(x"},
	{"key": "f", "command": "app.f"},
]`
	bs, errs := load(t, src, nil)
	var got []string
	for _, e := range errs {
		got = append(got, fmt.Sprintf("%d %s", e.Line, e.Pointer))
	}
	if want := []string{"3 /1/key", "5 /3/colour", "6 /4/when"}; !slices.Equal(got, want) {
		t.Errorf("errors at %q; want %q", got, want)
	}
	var ids []command.ID
	for _, b := range bs {
		ids = append(ids, b.Command)
	}
	if want := []command.ID{"app.a", "app.c", "app.f"}; !slices.Equal(ids, want) {
		t.Errorf("loaded %v; want %v", ids, want)
	}
}

// TestLoadRemoval: "-command" removes with a key, and with none removes
// every key of the command in its scope.
func TestLoadRemoval(t *testing.T) {
	bs, errs := load(t, `[
		{"key": "ctrl+k", "command": "-app.x"},
		{"command": "-app.y", "context": "pane:a"},
	]`, nil)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	want := []Binding{
		{Keys: seq(t, "ctrl+k"), Command: "app.x", Remove: true},
		{Command: "app.y", Scope: "pane:a", Remove: true},
	}
	if got := placeless(bs); !reflect.DeepEqual(got, want) {
		t.Fatalf("loaded %+v\nwant %+v", got, want)
	}
	defaults := defaults(
		bind(t, DefaultLayer, command.Global, "ctrl+k", "app.x"),
		bind(t, DefaultLayer, command.Global, "ctrl+j", "app.x"),
		bind(t, DefaultLayer, "pane:a", "a", "app.y"),
		bind(t, DefaultLayer, "pane:a", "b", "app.y"),
	)
	k := mustNew(t, Settings{}, defaults, bs)
	var left []string
	for b := range k.Rules() {
		left = append(left, b.Keys.String()+" "+string(b.Command))
	}
	if want := []string{"ctrl+j app.x"}; !slices.Equal(left, want) {
		t.Errorf("bound after the removals: %q; want %q", left, want)
	}
}

// TestLoadShapes: an empty file, one of comments, and a byte order mark
// hold no bindings and no error; "global" is the global scope.
func TestLoadShapes(t *testing.T) {
	for _, src := range []string{"", " \n", "// none yet\n", "/* none */", "\xef\xbb\xbf[]", "[]"} {
		if bs, errs := load(t, src, nil); len(bs) > 0 || len(errs) > 0 {
			t.Errorf("%q: %v, %v; want nothing", src, bs, errs)
		}
	}
	bs, errs := load(t, `[{"key": "a", "command": "app.x", "context": "global", "when": ""}]`, nil)
	if len(errs) > 0 || len(bs) != 1 || bs[0].Scope != command.Global || bs[0].When != "" {
		t.Errorf(`"context": "global": %+v, %v`, bs, errs)
	}
}

// TestLoadErrorText: the message names as much of the place as is known,
// and Unwrap is the error there. A reader's error is a *LoadError naming
// the file.
func TestLoadErrorText(t *testing.T) {
	bad := errors.New("bad")
	for _, c := range []struct {
		e    LoadError
		want string
	}{
		{LoadError{File: "k.json", Line: 2, Col: 3, Pointer: "/0/key", Err: bad}, "keymap: k.json:2:3 (/0/key): bad"},
		{LoadError{Line: 2, Col: 3, Err: bad}, "keymap: 2:3: bad"},
		{LoadError{File: "k.json", Err: bad}, "keymap: k.json: bad"},
		{LoadError{Err: bad}, "keymap: bad"},
	} {
		if got := c.e.Error(); got != c.want {
			t.Errorf("Error() = %q; want %q", got, c.want)
		}
		if !errors.Is(&c.e, bad) {
			t.Errorf("%q does not unwrap to its error", c.want)
		}
	}
	_, err := LoadVSCode(iotest.ErrReader(bad), "k.json", nil)
	var le *LoadError
	if !errors.As(err, &le) || le.File != "k.json" || !errors.Is(err, bad) {
		t.Errorf("a reader's error: %v", err)
	}
}

// workspaceDefaults are MADR §9's defaults, with amendment A4's commands,
// as bindings.
func workspaceDefaults(t *testing.T) []Binding {
	ws := command.Scope("workspace")
	bs := []Binding{
		bind(t, DefaultLayer, ws, "alt+.", "workspace.focus.next"),
		bind(t, DefaultLayer, ws, "alt+,", "workspace.focus.prev"),
	}
	for n := 1; n <= 9; n++ {
		b := bind(t, DefaultLayer, ws, fmt.Sprintf("alt+%d", n), "workspace.focus.index")
		b.Args = jsontext.Value(fmt.Sprintf(`{"index":%d}`, n))
		bs = append(bs, b)
	}
	bs = append(bs, bind(t, DefaultLayer, ws, "alt+z", "workspace.zoom"))
	for _, d := range []string{"left", "right", "up", "down"} {
		b := bind(t, DefaultLayer, ws, "alt+shift+"+d, "workspace.resize.direction")
		b.Args = jsontext.Value(`{"direction":"` + d + `"}`)
		bs = append(bs, b)
	}
	return append(bs, bind(t, DefaultLayer, "overlay", "esc", "workspace.overlay.close"))
}

// userFile is a planted user file with each kind of member and key.
const userFile = `[
	{"key": "<leader> f", "command": "app.find", "when": "editorFocus && !readOnly"},
	{"key": "Ctrl+Shift+A", "command": "app.all", "context": "pane:a"},
	{"key": "ctrl+[KeyA]", "command": "app.scan"},
	{"key": "ctrl+a:release", "command": "app.release", "context": "mode:vim-normal"},
	{"key": "escape g", "command": "app.chord", "args": {"n": [1, 2, {"deep": true}], "s": "é\"\\"}},
	{"key": "ctrl+k", "command": "-app.save"},
	{"command": "-app.quit", "when": "dirty"},
	{"key": "alt+/", "command": "app.comment", "context": "global"},
]`

// TestExportRoundTrip: what ExportVSCode writes, LoadVSCode reads back as
// the same bindings, for the workspace's defaults and a user's file.
func TestExportRoundTrip(t *testing.T) {
	user, errs := load(t, userFile, nil)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	for name, bs := range map[string][]Binding{"the workspace defaults": workspaceDefaults(t), "a user file": user} {
		var b bytes.Buffer
		if err := ExportVSCode(&b, slices.Values(bs)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got, errs := load(t, b.String(), nil)
		if len(errs) > 0 {
			t.Fatalf("%s: %v\n%s", name, errs, b.String())
		}
		if !reflect.DeepEqual(placeless(got), placeless(bs)) {
			t.Errorf("%s: read back\n%+v\nwant\n%+v\nfrom\n%s", name, placeless(got), placeless(bs), b.String())
		}
	}
}

// TestExportText: the exporter writes VS Code's spelling and layout.
func TestExportText(t *testing.T) {
	bs := []Binding{
		{Keys: seq(t, "esc ctrl+shift+a"), Command: "app.x", When: "a", Args: jsontext.Value(`{"n":1}`), Scope: "pane:a"},
		{Keys: seq(t, "super+pgup"), Command: "app.y", Remove: true},
		{Command: "app.z", Remove: true},
	}
	var b bytes.Buffer
	if err := ExportVSCode(&b, slices.Values(bs)); err != nil {
		t.Fatal(err)
	}
	want := `[
	{
		"key": "escape ctrl+shift+a",
		"command": "app.x",
		"when": "a",
		"args": {
			"n": 1
		},
		"context": "pane:a"
	},
	{
		"key": "cmd+pageup",
		"command": "-app.y"
	},
	{
		"command": "-app.z"
	}
]
`
	if b.String() != want {
		t.Errorf("wrote\n%s\nwant\n%s", b.String(), want)
	}
	b.Reset()
	if err := ExportVSCode(&b, slices.Values([]Binding(nil))); err != nil || b.String() != "[]\n" {
		t.Errorf("no bindings: %q, %v", b.String(), err)
	}
	if err := ExportVSCode(&b, slices.Values([]Binding{{Command: "app.x", Args: jsontext.Value("{")}})); err == nil {
		t.Error("bad args: no error")
	}
}
