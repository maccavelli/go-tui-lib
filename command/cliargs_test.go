package command

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// ParseArgs, Params and Complete
// (docs/decisions/0014-PLAN-component-native-forms.md Step 5).

type deployArgs struct {
	Target  string        `json:"target" arg:"" help:"where to deploy" placeholder:"HOST"`
	Files   []string      `json:"files,omitzero" arg:""`
	Count   int           `json:"count,omitzero" short:"n" schema:"min=1,max=9"`
	Verbose bool          `json:"verbose,omitzero" short:"v"`
	Mode    string        `json:"mode,omitzero" enum:"fast,safe" default:"safe" group:"Behaviour"`
	Tags    []string      `json:"tags,omitzero" enum:"a,b"`
	Wait    time.Duration `json:"wait,omitzero"`
	Token   string        `json:"token,omitzero" hidden:"" schema:"secret"`
}

func cliRegistry(t testing.TB) *Registry {
	t.Helper()
	return registryOf(t, nil,
		testCommand(t, "deploy", func(context.Context, *Invocation, deployArgs) (Result, error) { return Result{}, nil }),
		testCommand(t, "review", func(context.Context, *Invocation, reviewArgs) (Result, error) { return Result{}, nil }, WithSlash("review")),
		testCommand(t, "files.add", func(context.Context, *Invocation, addArgs) (Result, error) { return Result{}, nil }, WithSlash("add")),
		testCommand(t, "workspace.resize", func(context.Context, *Invocation, resizeArgs) (Result, error) { return Result{}, nil }, WithSlash("resize")),
		cmd("keys.only", UI, func(c *Command) { c.Surfaces = SurfaceKey }),
		cmd("secret.tool", UI, func(c *Command) { c.Hidden = true }),
	)
}

func TestParseArgs(t *testing.T) {
	r := cliRegistry(t)
	ok := func(id ID, want string, args ...string) {
		t.Helper()
		req, err := r.ParseArgs(id, args, OriginCLI)
		if err != nil {
			t.Errorf("%s %q: %v", id, args, err)
			return
		}
		if string(req.Args) != want || req.ID != id || req.Origin != OriginCLI || req.Raw != quoteArgs(args) {
			t.Errorf("%s %q:\n got %s %q\nwant %s", id, args, req.Args, req.Raw, want)
		}
	}
	fails := func(id ID, want string, args ...string) {
		t.Helper()
		_, err := r.ParseArgs(id, args, OriginCLI)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s %q: %v; want an error with %q", id, args, err, want)
		}
		if _, isArg := errors.AsType[*ArgError](err); id != "no.such" && !isArg {
			t.Errorf("%s %q: %v is not an *ArgError", id, args, err)
		}
	}
	base := `"mode":"safe","target":"h"`
	ok("deploy", `{`+base+`}`, "h")
	ok("deploy", `{"count":3,`+base+`}`, "h", "--count=3")
	ok("deploy", `{"count":3,`+base+`}`, "--count", "3", "h")
	ok("deploy", `{"count":3,`+base+`}`, "-n", "3", "h")
	ok("deploy", `{"count":3,`+base+`}`, "-n=3", "h")
	ok("deploy", `{`+base+`,"verbose":true}`, "h", "--verbose")
	ok("deploy", `{`+base+`,"verbose":true}`, "-v", "h")
	ok("deploy", `{`+base+`,"verbose":false}`, "--verbose=false", "h")
	ok("deploy", `{"files":["x"],`+base+`,"verbose":true}`, "-v", "h", "x") // a boolean never takes the next word
	ok("deploy", `{"files":["a","b","c"],`+base+`}`, "h", "a", "b", "c")
	ok("deploy", `{"files":["-v","--count=2"],`+base+`}`, "h", "--", "-v", "--count=2")
	ok("deploy", `{"files":["-"],`+base+`}`, "h", "-")
	ok("deploy", `{"mode":"fast","target":"h"}`, "--mode", "fast", "h")
	ok("deploy", `{"mode":"safe","tags":["a","b"],"target":"h"}`, "--tags=a", "h", "--tags", "b")
	ok("deploy", `{`+base+`,"wait":"1m30s"}`, "h", "--wait=1m30s")
	ok("deploy", `{"mode":"safe","target":"h h"}`, "h h")
	ok("deploy", `{"mode":"safe","target":"--target"}`, "--target", "--target")
	ok("deploy", `{"mode":"safe","target":"h","token":"s"}`, "--token=s", "h") // hidden still parses
	ok("review", `{"focus":"fix the tests"}`, "fix", "the", "tests")
	ok("review", `{"focus":"x"}`, "--focus", "x")
	ok("review", `{}`)
	ok("files.add", `{"files":["a","b"],"force":true}`, "a", "-f", "b")
	ok("workspace.resize", `{"delta":-3,"split":"s"}`, "s", "--", "-3")
	ok("keys.only", ``) // no schema: no Args, and Raw only

	fails("no.such", "unknown command")
	fails("deploy", "/target: is required")
	fails("deploy", "/target: is required", "--count", "3")
	fails("deploy", "--bogus is not a known flag", "--bogus", "h")
	fails("deploy", "-x is not a known flag", "-x", "h")
	fails("deploy", "-count is not a known flag", "-count", "3", "h") // one dash is a short
	fails("deploy", "--count needs a value", "h", "--count")
	fails("deploy", `"x" is not a number`, "h", "--count", "x")
	fails("deploy", "/count", "h", "--count", "12")
	fails("deploy", "/mode", "h", "--mode", "slow")
	fails("deploy", "is given twice", "h", "--count=1", "-n", "2")
	fails("deploy", "is not true or false", "h", "--verbose=maybe")
	fails("workspace.resize", "one positional value too many", "s", "1", "2")

	// A duration's text is checked when the handler decodes it, as for a
	// slash line: the rule does not read the schema's pattern.
	req, err := r.ParseArgs("deploy", []string{"h", "--wait=soon"}, OriginCLI)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(t.Context(), req); err == nil || !strings.Contains(err.Error(), "/wait") {
		t.Errorf("Run with --wait=soon: %v; want an *ArgError for /wait", err)
	}
	fails("review", "is given twice", "--focus", "x", "y")
}

func TestParseArgsRaw(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, ""},
		{[]string{"a", "--b=c", "d/e:f@g%h+i,j"}, "a --b=c d/e:f@g%h+i,j"},
		{[]string{"two words", ""}, "'two words' ''"},
		{[]string{"it's", `"q"`, "$HOME", "a\tb"}, `'it'"'"'s' '"q"' '$HOME' 'a` + "\t" + `b'`},
	} {
		if got := quoteArgs(tc.args); got != tc.want {
			t.Errorf("quoteArgs(%q) = %q, want %q", tc.args, got, tc.want)
		}
		words, err := splitWords(tc.want)
		texts := make([]string, len(words))
		for i, w := range words {
			texts[i] = w.text
		}
		if err != nil || len(texts) != len(tc.args) || (len(texts) > 0 && !slices.Equal(texts, tc.args)) {
			t.Errorf("%q splits back to %q, %v", tc.want, texts, err)
		}
	}
	r := cliRegistry(t)
	if req, err := r.ParseArgs("deploy", []string{"--unknown"}, OriginCLI); err == nil {
		t.Errorf("an unknown flag passed: %+v", req)
	}
}

// TestParseArgsMatchesSlash: where both grammars can say the same thing,
// ParseArgs and ParseSlash prepare the same arguments.
func TestParseArgsMatchesSlash(t *testing.T) {
	r := cliRegistry(t)
	for _, tc := range []struct {
		id    ID
		args  []string
		slash string
	}{
		{"review", []string{"fix", "the", "tests"}, "/review fix the tests"},
		{"review", nil, "/review"},
		{"files.add", []string{"a", "b", "--force"}, "/add a b force=true"},
		{"files.add", []string{"a", "--force=false"}, "/add a force=false"},
		{"workspace.resize", []string{"sidebar", "4"}, "/resize sidebar 4"},
		{"workspace.resize", []string{"--delta=4", "sidebar"}, "/resize delta=4 sidebar"},
		{"workspace.resize", []string{"two words", "1", "-q"}, `/resize "two words" 1 quiet=true`},
	} {
		a, err := r.ParseArgs(tc.id, tc.args, OriginCLI)
		if err != nil {
			t.Errorf("ParseArgs %q: %v", tc.args, err)
			continue
		}
		s, err := r.ParseSlash(tc.slash)
		if err != nil {
			t.Errorf("ParseSlash %q: %v", tc.slash, err)
			continue
		}
		if string(a.Args) != string(s.Args) {
			t.Errorf("%q gives %s; %q gives %s", tc.args, a.Args, tc.slash, s.Args)
		}
	}
}

func TestParams(t *testing.T) {
	c := MustNew("deploy", "Deploy", func(context.Context, *Invocation, deployArgs) (Result, error) { return Result{}, nil }, WithDanger(UI))
	ps, err := Params(c)
	if err != nil {
		t.Fatal(err)
	}
	f := func(v float64) *float64 { return &v }
	want := []Param{
		{Name: "target", Type: "string", Description: "where to deploy", Placeholder: "HOST", Required: true, Positional: true},
		{Name: "files", Type: "array", Items: "string", Positional: true, Rest: true},
		{Name: "count", Type: "integer", Short: "n", Min: f(1), Max: f(9)},
		{Name: "verbose", Type: "boolean", Short: "v"},
		{Name: "mode", Type: "string", Group: "Behaviour", HasDefault: true, Default: "safe", Enum: []any{"fast", "safe"}},
		{Name: "tags", Type: "array", Items: "string", Enum: []any{"a", "b"}},
		{Name: "wait", Type: "string"},
		{Name: "token", Type: "string", Hidden: true, Secret: true},
	}
	if len(ps) != len(want) {
		t.Fatalf("Params = %d params, want %d: %+v", len(ps), len(want), ps)
	}
	for i := range want {
		if !reflect.DeepEqual(ps[i], want[i]) {
			t.Errorf("param %d:\n got %+v\nwant %+v", i, ps[i], want[i])
		}
	}
	if ps, err := Params(Command{ID: "none"}); ps != nil || err != nil {
		t.Errorf("no schema: %v, %v", ps, err)
	}
	if _, err := Params(Command{ID: "bad", Args: Schema(`[1]`)}); err == nil || !strings.HasPrefix(err.Error(), "command: bad: args schema: ") {
		t.Errorf("a bad schema: %v", err)
	}
	loose := Command{ID: "loose", Args: Schema(`{"type":"object","properties":{"x":{"type":"string","description":5,"x-cli":{"arg":true,"short":7}}}}`)}
	if ps, err := Params(loose); err != nil || len(ps) != 1 || !ps[0].Positional || ps[0].Description != "" || ps[0].Short != "" {
		t.Errorf("annotations of the wrong type: %+v, %v; want them left out, and arg kept", ps, err)
	}
}

func TestComplete(t *testing.T) {
	r := cliRegistry(t)
	for _, tc := range []struct {
		id      ID
		args    []string
		partial string
		want    []string
	}{
		{"", nil, "", []string{"command.describe", "command.list", "deploy", "files.add", "review", "workspace.resize"}},
		{"", nil, "d", []string{"deploy"}},
		{"deploy", nil, "-", []string{"--count", "--files", "--mode", "--tags", "--target", "--verbose", "--wait", "-n", "-v"}},
		{"deploy", nil, "--c", []string{"--count"}},
		{"deploy", []string{"h"}, "--mode=", []string{"--mode=fast", "--mode=safe"}},
		{"deploy", nil, "--verbose=", []string{"--verbose=false", "--verbose=true"}},
		{"deploy", nil, "--tags=b", []string{"--tags=b"}},
		{"deploy", []string{"--mode"}, "", []string{"fast", "safe"}},
		{"deploy", []string{"--mode"}, "f", []string{"fast"}},
		{"deploy", []string{"-v"}, "", nil},                            // a boolean takes no value
		{"deploy", []string{"--count"}, "", nil},                       // no enum
		{"deploy", []string{"--", "--mode"}, "", nil},                  // after --
		{"deploy", nil, "x", nil},                                      // a positional
		{"deploy", nil, "--bogus=", nil},                               // no such flag
		{"no.such", nil, "-", nil},                                     // no such command
		{"keys.only", nil, "-", nil},                                   // no schema
		{"workspace.resize", nil, "-", []string{"--delta", "--split"}}, // quiet is hidden
	} {
		got := r.Complete(tc.id, tc.args, tc.partial)
		if !slices.Equal(got, tc.want) {
			t.Errorf("Complete(%q, %q, %q) = %q, want %q", tc.id, tc.args, tc.partial, got, tc.want)
		}
	}
}
