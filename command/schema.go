package command

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// schemaDialect is the JSON Schema dialect every schema SchemaOf emits
// names.
const schemaDialect = "https://json-schema.org/draft/2020-12/schema"

// JSON Schema's type names.
const (
	tObject  = "object"
	tArray   = "array"
	tString  = "string"
	tNumber  = "number"
	tInteger = "integer"
	tBoolean = "boolean"
	tNull    = "null"
)

// durationPattern is a Go duration's syntax, as time.ParseDuration reads
// it.
const durationPattern = `^[-+]?(0|([0-9]+(\.[0-9]*)?|\.[0-9]+)(ns|us|µs|μs|ms|s|m|h))+$`

// jsonSchemaer is implemented by a type that gives its own schema.
type jsonSchemaer interface{ JSONSchema() Schema }

var (
	schemaerType  = reflect.TypeFor[jsonSchemaer]()
	durationType  = reflect.TypeFor[time.Duration]()
	errNotSupport = errors.New("is not supported")
)

// SchemaOf emits a JSON Schema 2020-12 document for A.
//
// A struct is an object whose properties are its exported fields, named
// by their json tags, in field order, with additionalProperties false. A
// field is required unless it is a pointer, has omitempty or omitzero, or
// has a default. The tags, aligned with Kong's
// (docs/decisions/0006-MADR-command-registry.md A1, A5):
//
//	json:"name,omitzero"  the property name; omitzero, omitempty and embed
//	help:"…"              the description
//	default:"…"           the default, in the field's type
//	enum:"a,b"            the allowed values, comma-separated
//	arg:""                positional, in field order
//	short:"d"             a one-letter flag
//	hidden:""             in the schema, not in help
//	placeholder:"PANE"    the value's name in help
//	group:"…"             the flag's help group
//	schema:"min=…,max=…,minLen=…,maxLen=…,secret"
//
// secret is written as "writeOnly": true, and masks the value in audit
// records. arg, short, placeholder, group and hidden are written in one
// "x-cli" object, which JSON Schema treats as an annotation.
//
// Supported types: string, bool, the integer and float kinds, slices and
// arrays, structs, pointers, maps with string keys, time.Duration (a
// string such as "1m30s"), []byte (a base64 string), and a type with its
// own JSONSchema() Schema method. Any other type, a recursive type, and a
// scalar enum field that is neither required nor defaulted are errors.
func SchemaOf[A any]() (Schema, error) {
	n, err := schemaFor(reflect.TypeFor[A](), map[reflect.Type]bool{})
	if err != nil {
		return nil, err
	}
	return n.encode(true)
}

// node is one schema, before it is written.
type node struct {
	typ, desc, pattern, encoding string
	enum                         []any
	def                          any
	hasDef                       bool
	min, max                     *float64
	minLen, maxLen               *int
	writeOnly                    bool
	items, addl                  *node
	props                        []prop
	required                     []string
	closed                       bool
	cli                          cliInfo
	raw                          jsontext.Value // a type's own schema
}

type prop struct {
	name string
	n    *node
}

// cliInfo is a property's "x-cli" object: what a command line needs and
// JSON Schema has no keyword for.
type cliInfo struct {
	Arg         bool   `json:"arg,omitzero"`
	Short       string `json:"short,omitzero"`
	Placeholder string `json:"placeholder,omitzero"`
	Group       string `json:"group,omitzero"`
	Hidden      bool   `json:"hidden,omitzero"`
}

func schemaFor(t reflect.Type, seen map[reflect.Type]bool) (*node, error) {
	if raw, ok, err := ownSchema(t); ok || err != nil {
		return &node{raw: raw}, err
	}
	if t == durationType {
		return &node{typ: tString, pattern: durationPattern}, nil
	}
	switch t.Kind() {
	case reflect.String:
		return &node{typ: tString}, nil
	case reflect.Bool:
		return &node{typ: tBoolean}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &node{typ: tInteger}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		zero := 0.0
		return &node{typ: tInteger, min: &zero}, nil
	case reflect.Float32, reflect.Float64:
		return &node{typ: tNumber}, nil
	case reflect.Pointer:
		return schemaFor(t.Elem(), seen)
	case reflect.Slice, reflect.Array:
		if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
			return &node{typ: tString, encoding: "base64"}, nil
		}
		items, err := schemaFor(t.Elem(), seen)
		if err != nil {
			return nil, err
		}
		return &node{typ: tArray, items: items}, nil
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("command: schema: %s: a map key must be a string", t)
		}
		addl, err := schemaFor(t.Elem(), seen)
		if err != nil {
			return nil, err
		}
		return &node{typ: tObject, addl: addl}, nil
	case reflect.Struct:
		if seen[t] {
			return nil, fmt.Errorf("command: schema: %s is recursive", t)
		}
		seen[t] = true
		defer delete(seen, t)
		n := &node{typ: tObject, closed: true}
		return n, structFields(t, n, seen)
	}
	return nil, fmt.Errorf("command: schema: %s %w", t, errNotSupport)
}

// ownSchema is t's own JSONSchema(), if t or *t has one.
func ownSchema(t reflect.Type) (jsontext.Value, bool, error) {
	var v reflect.Value
	switch {
	case t.Kind() == reflect.Pointer:
		return nil, false, nil // schemaFor takes the element's
	case t.Implements(schemaerType):
		v = reflect.New(t).Elem()
	case reflect.PointerTo(t).Implements(schemaerType):
		v = reflect.New(t)
	default:
		return nil, false, nil
	}
	s, ok := v.Interface().(jsonSchemaer)
	if !ok {
		return nil, false, nil
	}
	raw := jsontext.Value(bytes.Clone(s.JSONSchema()))
	if !raw.IsValid() || raw.Kind() != '{' {
		return nil, true, fmt.Errorf("command: schema: %s's JSONSchema is not a JSON object", t)
	}
	return raw, true, nil
}

// structFields adds t's fields to n, flattening embedded structs.
func structFields(t reflect.Type, n *node, seen map[reflect.Type]bool) error {
	for f := range t.Fields() {
		tag := f.Tag.Get("json")
		if tag == "-" || (!f.IsExported() && !f.Anonymous) {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		o, err := jsonOptions(t, f, opts)
		if err != nil {
			return err
		}
		// json/v2 embeds an embedded struct unless the field names itself.
		ft := f.Type
		if f.Anonymous && name == "" {
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				if err := structFields(ft, n, seen); err != nil {
					return err
				}
				continue
			}
		}
		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		if err := addField(n, t, f, name, o, seen); err != nil {
			return err
		}
	}
	return nil
}

type fieldOpts struct{ omit bool }

func jsonOptions(t reflect.Type, f reflect.StructField, opts string) (fieldOpts, error) {
	var o fieldOpts
	for opt := range strings.SplitSeq(opts, ",") {
		switch opt {
		case "":
		case "omitzero", "omitempty":
			o.omit = true
		case "embed": // an embedded field without a name is embedded anyway
		default:
			return o, fmt.Errorf("command: schema: %s.%s: json option %q is not supported", t, f.Name, opt)
		}
	}
	return o, nil
}

// addField adds the property for field f of struct t.
func addField(n *node, t reflect.Type, f reflect.StructField, name string, o fieldOpts, seen map[reflect.Type]bool) error {
	for _, p := range n.props {
		if p.name == name {
			return fmt.Errorf("command: schema: %s: two fields are named %q", t, name)
		}
	}
	pn, err := schemaFor(f.Type, seen)
	if err != nil {
		return fmt.Errorf("%w (field %s.%s)", err, t, f.Name)
	}
	if err := applyTags(pn, f); err != nil {
		return fmt.Errorf("command: schema: %s.%s: %w", t, f.Name, err)
	}
	optional := o.omit || pn.hasDef || f.Type.Kind() == reflect.Pointer
	if !optional {
		n.required = append(n.required, name)
	}
	if len(pn.enum) > 0 && pn.typ != tArray && optional && !pn.hasDef {
		return fmt.Errorf("command: schema: %s.%s: an enum must be required or have a default", t, f.Name)
	}
	n.props = append(n.props, prop{name, pn})
	return nil
}

// applyTags reads f's tags other than json into n.
func applyTags(n *node, f reflect.StructField) error {
	tags := []string{"help", "default", "enum", "schema", "arg", "short", "hidden", "placeholder", "group"}
	if n.raw != nil {
		for _, k := range tags {
			if _, ok := f.Tag.Lookup(k); ok {
				return fmt.Errorf("tag %s on a type with its own schema", k)
			}
		}
		return nil
	}
	n.desc = f.Tag.Get("help")
	_, n.cli.Arg = f.Tag.Lookup("arg")
	_, n.cli.Hidden = f.Tag.Lookup("hidden")
	n.cli.Short = f.Tag.Get("short")
	n.cli.Placeholder = f.Tag.Get("placeholder")
	n.cli.Group = f.Tag.Get("group")
	if err := applySchemaTag(n, f.Tag.Get("schema")); err != nil {
		return err
	}
	scalar := f.Type
	for scalar.Kind() == reflect.Pointer {
		scalar = scalar.Elem()
	}
	if e, ok := f.Tag.Lookup("enum"); ok {
		target, et := n, scalar
		if n.typ == tArray {
			target, et = n.items, scalar.Elem()
		}
		for v := range strings.SplitSeq(e, ",") {
			x, err := scalarValue(et, v)
			if err != nil {
				return fmt.Errorf("enum value %q: %w", v, err)
			}
			target.enum = append(target.enum, x)
		}
	}
	if d, ok := f.Tag.Lookup("default"); ok {
		x, err := scalarValue(scalar, d)
		if err != nil {
			return fmt.Errorf("default %q: %w", d, err)
		}
		if len(n.enum) > 0 && !slices.Contains(n.enum, x) {
			return fmt.Errorf("default %q is not one of its enum", d)
		}
		n.def, n.hasDef = x, true
	}
	return nil
}

// applySchemaTag reads a schema tag's min, max, minLen, maxLen and secret.
func applySchemaTag(n *node, tag string) error {
	if tag == "" {
		return nil
	}
	for item := range strings.SplitSeq(tag, ",") {
		k, v, _ := strings.Cut(item, "=")
		var err error
		switch k {
		case "secret":
			n.writeOnly = true
		case "min", "max":
			if n.typ != tInteger && n.typ != tNumber {
				return fmt.Errorf("schema %s on a %s", k, n.typ)
			}
			var f float64
			if f, err = strconv.ParseFloat(v, 64); err == nil {
				if k == "min" {
					n.min = &f
				} else {
					n.max = &f
				}
			}
		case "minLen", "maxLen":
			if n.typ != tString {
				return fmt.Errorf("schema %s on a %s", k, n.typ)
			}
			var i int
			if i, err = strconv.Atoi(v); err == nil && i >= 0 {
				if k == "minLen" {
					n.minLen = &i
				} else {
					n.maxLen = &i
				}
			} else if err == nil {
				err = errors.New("is negative")
			}
		default:
			return fmt.Errorf("schema key %q is not min, max, minLen, maxLen or secret", k)
		}
		if err != nil {
			return fmt.Errorf("schema %s=%s: %w", k, v, err)
		}
	}
	return nil
}

// scalarValue is s as a value of type t, as JSON holds it.
func scalarValue(t reflect.Type, s string) (any, error) {
	if t == durationType {
		if _, err := time.ParseDuration(s); err != nil {
			return nil, err
		}
		return s, nil
	}
	switch t.Kind() {
	case reflect.String:
		return s, nil
	case reflect.Bool:
		return strconv.ParseBool(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, t.Bits())
		return float64(i), err
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(s, 10, t.Bits())
		return float64(u), err
	case reflect.Float32, reflect.Float64:
		return strconv.ParseFloat(s, t.Bits())
	}
	return nil, fmt.Errorf("a %s %w as a tag value", t, errNotSupport)
}

// encode writes n as JSON, with $schema when it is the top.
func (n *node) encode(top bool) (Schema, error) {
	var b bytes.Buffer
	e := jsontext.NewEncoder(&b)
	if err := n.write(e, top); err != nil {
		return nil, err
	}
	return Schema(bytes.TrimSpace(b.Bytes())), nil
}

// write writes n's keywords in a fixed order.
func (n *node) write(e *jsontext.Encoder, top bool) error {
	if n.raw != nil {
		return e.WriteValue(n.raw)
	}
	w := objWriter{e: e}
	w.token(jsontext.BeginObject)
	if top {
		w.member("$schema", schemaDialect)
	}
	w.memberIf(n.typ != "", "type", n.typ)
	w.memberIf(n.desc != "", "description", n.desc)
	w.memberIf(n.enum != nil, "enum", n.enum)
	w.memberIf(n.hasDef, "default", n.def)
	w.memberIf(n.min != nil, "minimum", n.min)
	w.memberIf(n.max != nil, "maximum", n.max)
	w.memberIf(n.minLen != nil, "minLength", n.minLen)
	w.memberIf(n.maxLen != nil, "maxLength", n.maxLen)
	w.memberIf(n.pattern != "", "pattern", n.pattern)
	w.memberIf(n.encoding != "", "contentEncoding", n.encoding)
	w.memberIf(n.writeOnly, "writeOnly", true)
	if n.items != nil {
		w.node("items", n.items)
	}
	if len(n.props) > 0 {
		w.token(jsontext.String("properties"))
		w.token(jsontext.BeginObject)
		for _, p := range n.props {
			w.node(p.name, p.n)
		}
		w.token(jsontext.EndObject)
	}
	w.memberIf(len(n.required) > 0, "required", n.required)
	switch {
	case n.closed:
		w.member("additionalProperties", false)
	case n.addl != nil:
		w.node("additionalProperties", n.addl)
	}
	w.memberIf(n.cli != cliInfo{}, "x-cli", n.cli)
	w.token(jsontext.EndObject)
	return w.err
}

// objWriter writes members to an encoder, keeping the first error.
type objWriter struct {
	e   *jsontext.Encoder
	err error
}

func (w *objWriter) token(t jsontext.Token) {
	if w.err == nil {
		w.err = w.e.WriteToken(t)
	}
}

func (w *objWriter) member(name string, v any) {
	w.token(jsontext.String(name))
	if w.err == nil {
		w.err = json.MarshalEncode(w.e, v, json.Deterministic(true))
	}
}

func (w *objWriter) memberIf(ok bool, name string, v any) {
	if ok {
		w.member(name, v)
	}
}

func (w *objWriter) node(name string, n *node) {
	w.token(jsontext.String(name))
	if w.err == nil {
		w.err = n.write(w.e, false)
	}
}
