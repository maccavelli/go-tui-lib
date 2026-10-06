package cli

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/go-tui-lib/command"
)

// property is one argument of a command's schema, as the shell sees it.
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

// argValue is a flag.Value that sets one property of the arguments.
type argValue struct {
	p    *property
	obj  map[string]any
	last **command.ArgError // the parse's last argument error
}

func (v argValue) String() string { return "" }

func (v argValue) IsBoolFlag() bool { return v.p.is(tBoolean) }

// Set reads one value: an array appends its item, anything else is set
// once, an object or map as JSON.
func (v argValue) Set(text string) error {
	if err := v.assign(text); err != nil {
		*v.last = &command.ArgError{Path: "/" + v.p.name, Reason: err.Error()}
		return *v.last
	}
	return nil
}

// assign is Set without the error's wrapping.
func (v argValue) assign(text string) error {
	if v.p.is(tArray) {
		item, err := value(itemType(v.p), text)
		if err != nil {
			return err
		}
		var list []any
		if l, ok := v.obj[v.p.name].([]any); ok {
			list = l
		}
		v.obj[v.p.name] = append(list, item)
		return nil
	}
	if _, ok := v.obj[v.p.name]; ok {
		return errors.New("is given twice")
	}
	x, err := value(v.p.typeOf(), text)
	if err != nil {
		return err
	}
	v.obj[v.p.name] = x
	return nil
}

// itemType is an array's item type, string when the schema does not say.
func itemType(p *property) string {
	if p.items != "" {
		return p.items
	}
	return tString
}

// parsed is a command line's arguments and the shell's own flags.
type parsed struct {
	obj    map[string]any
	raw    string // --args
	asJSON bool
	yes    bool
	help   bool
}

// parse reads a command's flags and positional values against props. An
// unknown flag or a positional too many is a usage error, and a value its
// property cannot take an *command.ArgError.
func parse(props []*property, args []string) (parsed, error) {
	out := parsed{obj: map[string]any{}}
	var last *command.ArgError
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&out.raw, "args", "", "")
	fs.BoolVar(&out.asJSON, "json", false, "")
	fs.BoolVar(&out.yes, "yes", false, "")
	fs.BoolVar(&out.help, "help", false, "")
	fs.BoolVar(&out.help, "h", false, "")
	var positional []*property
	for _, p := range props {
		v := argValue{p: p, obj: out.obj, last: &last}
		if fs.Lookup(p.name) != nil {
			return out, fmt.Errorf("the property %q takes the name of a flag the shell keeps", p.name)
		}
		fs.Var(v, p.name, "")
		if s := p.cli.Short; s != "" && fs.Lookup(s) == nil {
			fs.Var(v, s, "")
		}
		if p.cli.Arg {
			positional = append(positional, p)
		}
	}
	next := 0
	for {
		if err := fs.Parse(args); err != nil {
			if last != nil {
				return out, last
			}
			return out, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return out, nil
		}
		for next < len(positional) && !positional[next].is(tArray) && out.obj[positional[next].name] != nil {
			next++
		}
		if next >= len(positional) {
			return out, fmt.Errorf("%q is one positional value too many", args[0])
		}
		p := positional[next]
		if err := (argValue{p: p, obj: out.obj, last: &last}).Set(args[0]); err != nil {
			return out, err
		}
		if !p.is(tArray) {
			next++
		}
		args = args[1:]
	}
}

// request is the JSON arguments: --args, then each flag and positional
// over it.
func (p parsed) request() ([]byte, error) {
	obj := map[string]any{}
	if strings.TrimSpace(p.raw) != "" {
		if err := json.Unmarshal([]byte(p.raw), &obj); err != nil {
			return nil, &command.ArgError{Reason: "--args is not a JSON object: " + err.Error()}
		}
	}
	maps.Copy(obj, p.obj)
	return json.Marshal(obj, json.Deterministic(true))
}
