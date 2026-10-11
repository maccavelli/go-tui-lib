package keymap

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"math/rand/v2"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/command"
)

// keyRE is the schema's key pattern, as Go reads it.
var keyRE = regexp.MustCompile(keyPattern)

// focusArgs is a command's args, with a schema from command.SchemaOf.
type focusArgs struct {
	Index int `json:"index" schema:"min=1,max=9"`
}

// TestSchema: the schema is JSON, names draft 2020-12, lists every ID and
// its "-" form once, requires a key unless the command removes, and gives
// an ID with an args schema an if/then pair, without the args schema's
// "$schema".
func TestSchema(t *testing.T) {
	argSchema, err := command.SchemaOf[focusArgs]()
	if err != nil {
		t.Fatal(err)
	}
	ids := []command.ID{"workspace.focus.index", "workspace.zoom", "app.flag", "workspace.zoom"}
	out, err := Schema(ids, func(id command.ID) jsontext.Value {
		switch id {
		case "workspace.focus.index":
			return argSchema
		case "app.flag":
			return jsontext.Value("true")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !jsontext.Value(out).IsValid() {
		t.Fatalf("not valid JSON:\n%s", out)
	}
	var doc struct {
		Schema string `json:"$schema"`
		Type   string `json:"type"`
		Items  struct {
			Properties map[string]struct {
				Pattern string   `json:"pattern"`
				Enum    []string `json:"enum"`
			} `json:"properties"`
			Required             []string `json:"required"`
			AdditionalProperties bool     `json:"additionalProperties"`
			AllOf                []struct {
				If struct {
					Properties map[string]map[string]string `json:"properties"`
				} `json:"if"`
				Then struct {
					Required   []string                  `json:"required"`
					Properties map[string]jsontext.Value `json:"properties"`
				} `json:"then"`
			} `json:"allOf"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Schema != "https://json-schema.org/draft/2020-12/schema" || doc.Type != "array" {
		t.Errorf("$schema %q, type %q", doc.Schema, doc.Type)
	}
	if !slices.Equal(doc.Items.Required, []string{"command"}) || doc.Items.AdditionalProperties {
		t.Errorf("required %q, additionalProperties %v", doc.Items.Required, doc.Items.AdditionalProperties)
	}
	want := []string{"workspace.focus.index", "workspace.zoom", "app.flag", "-workspace.focus.index", "-workspace.zoom", "-app.flag"}
	if got := doc.Items.Properties["command"].Enum; !slices.Equal(got, want) {
		t.Errorf("the command enum is %q; want %q", got, want)
	}
	if got := doc.Items.Properties["key"].Pattern; got != keyPattern {
		t.Errorf("the key pattern is not keyPattern")
	}
	rules := doc.Items.AllOf
	if len(rules) != 3 {
		t.Fatalf("allOf has %d rules; want 3", len(rules))
	}
	if rules[0].If.Properties["command"]["pattern"] != "^[^-]" || !slices.Equal(rules[0].Then.Required, []string{"key"}) {
		t.Errorf("the first rule does not require a key: %+v", rules[0])
	}
	var stripped jsontext.Value
	for _, r := range rules[1:] {
		args := r.Then.Properties["args"]
		switch r.If.Properties["command"]["const"] {
		case "workspace.focus.index":
			stripped = args
		case "app.flag":
			if string(args) != "true" {
				t.Errorf("app.flag's args schema is %s", args)
			}
		default:
			t.Errorf("a rule for %q", r.If.Properties["command"]["const"])
		}
	}
	var gotArgs, wantArgs map[string]jsontext.Value
	if err := json.Unmarshal(stripped, &gotArgs); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(argSchema, &wantArgs); err != nil {
		t.Fatal(err)
	}
	if _, ok := wantArgs["$schema"]; !ok {
		t.Fatal(`SchemaOf wrote no "$schema"; the test needs one`)
	}
	delete(wantArgs, "$schema")
	if !reflect.DeepEqual(compactAll(gotArgs), compactAll(wantArgs)) {
		t.Errorf("the args schema is %s; want %s without $schema", stripped, argSchema)
	}
}

// compactAll is m with each value compact.
func compactAll(m map[string]jsontext.Value) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		v = slices.Clone(v)
		if err := v.Compact(); err != nil {
			panic(err)
		}
		out[k] = string(v)
	}
	return out
}

// TestSchemaErrors: a malformed ID, and an args schema that is not JSON or
// is neither an object nor a boolean, are errors.
func TestSchemaErrors(t *testing.T) {
	if _, err := Schema([]command.ID{"Bad ID"}, nil); err == nil {
		t.Error("a malformed ID: no error")
	}
	for _, s := range []string{"{", "[1]", `"x"`} {
		_, err := Schema([]command.ID{"app.x"}, func(command.ID) jsontext.Value { return jsontext.Value(s) })
		if err == nil || !strings.Contains(err.Error(), "app.x") {
			t.Errorf("args schema %s: %v", s, err)
		}
	}
	if out, err := Schema(nil, nil); err != nil || !jsontext.Value(out).IsValid() {
		t.Errorf("no IDs: %v", err)
	}
}

// TestKeyPattern: the pattern compiles, uses only what Go and ECMA-262
// share, accepts every canonical and VS Code spelling Step 2's property
// test makes, every name and alias in either case, and <leader>, and
// refuses what is not the grammar.
func TestKeyPattern(t *testing.T) {
	for _, goOnly := range []string{`(?i`, `(?P`, `(?<`, `\p`, `\P`, `\z`, `\A`, `\Q`} {
		if strings.Contains(keyPattern, goOnly) {
			t.Errorf("the pattern uses %s, which ECMA-262 does not have", goOnly)
		}
	}
	r := rand.New(rand.NewPCG(7, 13))
	for range 5000 {
		q := Sequence{randomStroke(r)}
		for range r.IntN(MaxStrokes) {
			q = append(q, randomStroke(r))
		}
		for _, s := range []string{q.String(), q.VSCode()} {
			if !keyRE.MatchString(s) {
				t.Fatalf("the pattern refuses %q", s)
			}
		}
	}
	var names []string
	for n := range nameCodes {
		names = append(names, n, strings.ToUpper(n))
	}
	for _, vs := range vscodeNames {
		names = append(names, vs, strings.ToUpper(vs))
	}
	for n := range modifierNames {
		names = append(names, n+"+a", strings.ToUpper(n)+"+A")
	}
	for n := range scanCodes {
		names = append(names, "["+n+"]", "["+strings.ToLower(n)+"]")
	}
	names = append(names, "<leader> f", "<LEADER>", "Ctrl+Shift+A", "ctrl++", "+", "a:release", "A:Repeat", "::release", "ü")
	for _, s := range names {
		if _, err := ParseSequence(s, nil); err != nil {
			t.Fatalf("%q is not a key: %v", s, err)
		}
		if !keyRE.MatchString(s) {
			t.Errorf("the pattern refuses %q", s)
		}
	}
	for _, s := range []string{"", " a", "a ", "a  b", "a b c d e", "ctrl+", "ctlr+a", "ctrl+nope", "[KeyAA]", "a:press", "<leader>+a", "\t"} {
		if keyRE.MatchString(s) {
			t.Errorf("the pattern accepts %q", s)
		}
	}
}
