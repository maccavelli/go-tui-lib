package cobracmd

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/maccavelli/go-tui-lib/command"
)

// property is one argument of a command's schema, as the shell sees it.
// It is read as command/cli reads it, so that both front ends take the
// same flags.
type property struct {
	name, desc string
	types      []string
	items      string // an array's item type
	enum       []any
	required   bool
	hasDefault bool
	def        any
	min, max   *float64
	cli        struct {
		Arg         bool   `json:"arg"`
		Short       string `json:"short"`
		Placeholder string `json:"placeholder"`
		Group       string `json:"group"`
		Hidden      bool   `json:"hidden"`
	}
}

func (p *property) is(t string) bool { return slices.Contains(p.types, t) }

// The value names pflag reads: a boolean flag's type, which its usage
// prints without a value name, and the name of a JSON value.
const (
	typeBool  = "bool"
	valueJSON = "JSON"
)

// JSON Schema's type names, as the shell reads them.
const (
	tString  = "string"
	tInteger = "integer"
	tNumber  = "number"
	tBoolean = "boolean"
	tObject  = "object"
	tArray   = "array"
)

// properties reads the top-level properties of schema, in document order,
// which is the order positionals take (docs/decisions/0006-MADR-command-registry.md A5).
func properties(schema command.Schema) ([]*property, error) {
	if len(bytes.TrimSpace(schema)) == 0 {
		return nil, nil
	}
	var top struct {
		Properties jsontext.Value `json:"properties"`
		Required   []string       `json:"required"`
	}
	if err := json.Unmarshal(schema, &top); err != nil {
		return nil, err
	}
	if len(top.Properties) == 0 {
		return nil, nil
	}
	d := jsontext.NewDecoder(bytes.NewReader(top.Properties))
	if _, err := d.ReadToken(); err != nil {
		return nil, err
	}
	var out []*property
	for d.PeekKind() != '}' {
		tok, err := d.ReadToken()
		if err != nil {
			return nil, err
		}
		name := tok.String()
		v, err := d.ReadValue()
		if err != nil {
			return nil, err
		}
		p, err := readProperty(name, v)
		if err != nil {
			return nil, err
		}
		p.required = slices.Contains(top.Required, name)
		out = append(out, p)
	}
	return out, nil
}

func readProperty(name string, v jsontext.Value) (*property, error) {
	var raw struct {
		Type        jsontext.Value `json:"type"`
		Description string         `json:"description"`
		Enum        []any          `json:"enum"`
		Default     jsontext.Value `json:"default"`
		Minimum     *float64       `json:"minimum"`
		Maximum     *float64       `json:"maximum"`
		Items       struct {
			Type jsontext.Value `json:"type"`
		} `json:"items"`
		CLI jsontext.Value `json:"x-cli"`
	}
	if err := json.Unmarshal(v, &raw); err != nil {
		return nil, fmt.Errorf("property %s: %w", name, err)
	}
	p := &property{name: name, desc: raw.Description, enum: raw.Enum, min: raw.Minimum, max: raw.Maximum}
	p.types = typeList(raw.Type)
	if it := typeList(raw.Items.Type); len(it) > 0 {
		p.items = it[0]
	}
	if len(raw.Default) > 0 {
		p.hasDefault = true
		if err := json.Unmarshal(raw.Default, &p.def); err != nil {
			return nil, fmt.Errorf("property %s: default: %w", name, err)
		}
	}
	if len(raw.CLI) > 0 {
		if err := json.Unmarshal(raw.CLI, &p.cli); err != nil {
			return nil, fmt.Errorf("property %s: x-cli: %w", name, err)
		}
	}
	return p, nil
}

// typeList is a schema's "type", a string or a list of strings.
func typeList(v jsontext.Value) []string {
	var one string
	if json.Unmarshal(v, &one) == nil && one != "" {
		return []string{one}
	}
	var many []string
	if json.Unmarshal(v, &many) == nil {
		return many
	}
	return nil
}

// value parses text as a value of type t: a number, a boolean, JSON for
// an object or array, or the text itself.
func value(t, text string) (any, error) {
	switch t {
	case tInteger, tNumber:
		f, err := strconv.ParseFloat(text, 64)
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, fmt.Errorf("%q is not a number", text)
		}
		return f, nil
	case tBoolean:
		b, err := strconv.ParseBool(text)
		if err != nil {
			return nil, fmt.Errorf("%q is not true or false", text)
		}
		return b, nil
	case tObject, tArray:
		var v any
		if err := json.Unmarshal([]byte(text), &v); err != nil {
			return nil, fmt.Errorf("%q is not JSON: %w", text, err)
		}
		return v, nil
	}
	return text, nil
}

// typeOf is p's own type for parsing, the first it names, else string.
func (p *property) typeOf() string {
	for _, t := range p.types {
		if t != "null" {
			return t
		}
	}
	return tString
}

// itemType is an array's item type, string when the schema does not say.
func itemType(p *property) string {
	if p.items != "" {
		return p.items
	}
	return tString
}

// choices is an enum's values as the shell writes them.
func choices(p *property) []string {
	out := make([]string, len(p.enum))
	for i, e := range p.enum {
		out[i] = fmt.Sprint(e)
	}
	return out
}

// state is one command's arguments for one run: what its flags and
// positionals set, and the shell's own flags. Run clears it before each
// run, because Cobra keeps a tree's flag values between executions.
type state struct {
	obj    map[string]any
	last   *command.ArgError // the parse's last argument error
	raw    string            // --args
	asJSON bool
	yes    bool
}

func (s *state) reset() { *s = state{obj: map[string]any{}} }

// resetter is a flag value Run clears before each run.
type resetter interface{ reset() }

// argValue is a pflag.Value that sets one property of the arguments. Its
// type comes from the schema, never from a Go type.
type argValue struct {
	p  *property
	st *state
}

// String is the flag's default as Cobra's usage shows it: false for a
// boolean, which is unset until given, and nothing for the rest, whose
// defaults the registry applies.
func (v argValue) String() string {
	if v.p.is(tBoolean) {
		return "false"
	}
	return ""
}

// Type is the value's name in Cobra's flag usage, which man and Markdown
// pages print; "bool" tells pflag's usage to print none.
func (v argValue) Type() string {
	if v.p.is(tBoolean) {
		return typeBool
	}
	return placeholder(v.p)
}

func (v argValue) reset() { v.st.reset() }

// Set reads one value: an array appends its item, anything else is set
// once, an object or map as JSON, and an enum only to one of its values.
func (v argValue) Set(text string) error {
	if err := v.assign(text); err != nil {
		v.st.last = &command.ArgError{Path: "/" + v.p.name, Reason: err.Error()}
		return v.st.last
	}
	return nil
}

func (v argValue) assign(text string) error {
	if len(v.p.enum) > 0 && !v.p.is(tArray) && !slices.Contains(choices(v.p), text) {
		return fmt.Errorf("%q is not one of %s", text, strings.Join(choices(v.p), ", "))
	}
	return place(v.p, v.st.obj, text)
}

// place sets p in obj from text: an array appends its item, anything else
// is set once.
func place(p *property, obj map[string]any, text string) error {
	if p.is(tArray) {
		item, err := value(itemType(p), text)
		if err != nil {
			return err
		}
		var list []any
		if l, ok := obj[p.name].([]any); ok {
			list = l
		}
		obj[p.name] = append(list, item)
		return nil
	}
	if _, ok := obj[p.name]; ok {
		return errors.New("is given twice")
	}
	x, err := value(p.typeOf(), text)
	if err != nil {
		return err
	}
	obj[p.name] = x
	return nil
}

// switchValue is one of the shell's own boolean flags, --json and --yes.
type switchValue struct{ on *bool }

func (v switchValue) String() string { return strconv.FormatBool(*v.on) }
func (v switchValue) Type() string   { return typeBool }
func (v switchValue) reset()         { *v.on = false }

func (v switchValue) Set(text string) error {
	b, err := strconv.ParseBool(text)
	if err != nil {
		return fmt.Errorf("%q is not true or false", text)
	}
	*v.on = b
	return nil
}

// textValue is --args.
type textValue struct{ s *string }

func (v textValue) String() string { return *v.s }
func (v textValue) Type() string   { return valueJSON }
func (v textValue) reset()         { *v.s = "" }

func (v textValue) Set(text string) error {
	*v.s = text
	return nil
}

// reserved are the flag names the shell keeps for itself, as command/cli
// does.
var reserved = []string{"args", "json", "yes", "help", "h"}

// addFlags gives c a flag for each property, and the shell's own flags. A
// property named like one of the shell's flags, or with a short name that
// is taken or longer than one letter, cannot be a flag: the first is
// returned as an error the command reports when it runs, as command/cli
// does, and the second loses its short name.
func addFlags(c *cobra.Command, props []*property, st *state) error {
	fs := c.Flags()
	fs.SortFlags = false
	fs.Var(textValue{&st.raw}, "args", "the arguments as one JSON object")
	addSwitch(fs, &st.asJSON, "json", "print the result's value as JSON")
	addSwitch(fs, &st.yes, "yes", "run a destructive command without asking")
	var clash error
	for _, p := range props {
		if slices.Contains(reserved, p.name) {
			if clash == nil {
				clash = fmt.Errorf("the property %q takes the name of a flag the shell keeps", p.name)
			}
			continue
		}
		v := argValue{p: p, st: st}
		short := p.cli.Short
		if len(short) != 1 || short[0] >= 0x80 || short == "h" || fs.ShorthandLookup(short) != nil {
			short = ""
		}
		f := fs.VarPF(v, p.name, short, describe(p))
		if p.is(tBoolean) {
			f.NoOptDefVal = "true"
		}
		if p.cli.Hidden {
			f.Hidden = true
		}
	}
	return clash
}

func addSwitch(fs *pflag.FlagSet, on *bool, name, usage string) {
	f := fs.VarPF(switchValue{on}, name, "", usage)
	f.NoOptDefVal = "true"
}

// positionals are the properties taken in order from the command line.
func positionals(props []*property) []*property {
	var out []*property
	for _, p := range props {
		if p.cli.Arg && !slices.Contains(reserved, p.name) {
			out = append(out, p)
		}
	}
	return out
}

// placeArgs sets the positional values in a copy of obj, in order,
// skipping a positional its flag already set; an array takes every value
// left. A value too many, or one its property cannot take, is an error.
func placeArgs(pos []*property, obj map[string]any, args []string) (map[string]any, error) {
	out := maps.Clone(obj)
	if out == nil {
		out = map[string]any{}
	}
	next := 0
	for _, a := range args {
		for next < len(pos) && !pos[next].is(tArray) && out[pos[next].name] != nil {
			next++
		}
		if next >= len(pos) {
			return nil, fmt.Errorf("%q is one positional value too many", a)
		}
		p := pos[next]
		if len(p.enum) > 0 && !p.is(tArray) && !slices.Contains(choices(p), a) {
			return nil, &command.ArgError{Path: "/" + p.name, Reason: fmt.Sprintf("%q is not one of %s", a, strings.Join(choices(p), ", "))}
		}
		if err := place(p, out, a); err != nil {
			return nil, &command.ArgError{Path: "/" + p.name, Reason: err.Error()}
		}
		if !p.is(tArray) {
			next++
		}
	}
	return out, nil
}

// request is the JSON arguments: --args, then each flag and positional
// over it.
func request(raw string, obj map[string]any) ([]byte, error) {
	out := map[string]any{}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return nil, &command.ArgError{Reason: "--args is not a JSON object: " + err.Error()}
		}
	}
	maps.Copy(out, obj)
	return json.Marshal(out, json.Deterministic(true))
}
