package command

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// maxSchemaDepth bounds how deeply a schema may nest, so a loaded schema
// cannot exhaust the stack.
const maxSchemaDepth = 64

// rule is a schema compiled for checking arguments. It enforces type,
// required, enum, minimum, maximum, minLength, maxLength, items,
// properties and additionalProperties, applies default, and knows which
// values are secret (writeOnly) and which properties are positional
// (x-cli arg). Every other keyword is carried in the schema and ignored
// here.
type rule struct {
	types          []string
	enum           []any
	def            any
	hasDef         bool
	min, max       *float64
	minLen, maxLen *int
	secret         bool
	arg            bool
	items, addl    *rule
	props          []prule // in document order
	required       []string
	closed         bool
	anySecret      bool // this rule or one below it is secret
}

type prule struct {
	name string
	r    *rule
}

func (r *rule) prop(name string) *rule {
	for _, p := range r.props {
		if p.name == name {
			return p.r
		}
	}
	return nil
}

func (r *rule) is(t string) bool { return slices.Contains(r.types, t) }

// compileRule reads schema s.
func compileRule(s Schema) (*rule, error) {
	return compileAt(s, 0)
}

func compileAt(v jsontext.Value, depth int) (*rule, error) {
	if depth > maxSchemaDepth {
		return nil, fmt.Errorf("nested deeper than %d", maxSchemaDepth)
	}
	var kw map[string]jsontext.Value
	if err := json.Unmarshal(v, &kw); err != nil {
		return nil, fmt.Errorf("not a JSON object: %w", err)
	}
	r := &rule{}
	if err := r.scalars(kw); err != nil {
		return nil, err
	}
	if err := r.children(kw, depth); err != nil {
		return nil, err
	}
	r.anySecret = r.secret || (r.items != nil && r.items.anySecret) || (r.addl != nil && r.addl.anySecret)
	for _, p := range r.props {
		r.anySecret = r.anySecret || p.r.anySecret
	}
	return r, nil
}

// scalars reads r's keywords that hold plain values.
func (r *rule) scalars(kw map[string]jsontext.Value) error {
	if t, ok := kw["type"]; ok {
		if t.Kind() == '"' {
			var s string
			if err := json.Unmarshal(t, &s); err != nil {
				return err
			}
			r.types = []string{s}
		} else if err := json.Unmarshal(t, &r.types); err != nil {
			return fmt.Errorf("type: %w", err)
		}
	}
	if d, ok := kw["default"]; ok {
		r.hasDef = true
		if err := json.Unmarshal(d, &r.def); err != nil {
			return fmt.Errorf("default: %w", err)
		}
	}
	var cli struct {
		Arg bool `json:"arg"`
	}
	for name, dst := range map[string]any{
		"enum": &r.enum, "minimum": &r.min, "maximum": &r.max,
		"minLength": &r.minLen, "maxLength": &r.maxLen, "writeOnly": &r.secret,
		"required": &r.required, "x-cli": &cli,
	} {
		if raw, ok := kw[name]; ok {
			if err := json.Unmarshal(raw, dst); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	r.arg = cli.Arg
	return nil
}

// children reads items, properties and additionalProperties.
func (r *rule) children(kw map[string]jsontext.Value, depth int) error {
	var err error
	if v, ok := kw["items"]; ok && v.Kind() == '{' {
		if r.items, err = compileAt(v, depth+1); err != nil {
			return fmt.Errorf("items: %w", err)
		}
	}
	if v, ok := kw["additionalProperties"]; ok {
		switch v.Kind() {
		case 'f':
			r.closed = true
		case '{':
			if r.addl, err = compileAt(v, depth+1); err != nil {
				return fmt.Errorf("additionalProperties: %w", err)
			}
		}
	}
	v, ok := kw["properties"]
	if !ok {
		return nil
	}
	// Read properties in document order: positionals follow it (A5).
	d := jsontext.NewDecoder(bytes.NewReader(v))
	if t, err := d.ReadToken(); err != nil || t.Kind() != '{' {
		return errors.New("properties: not a JSON object")
	}
	for d.PeekKind() != '}' {
		tok, err := d.ReadToken()
		if err != nil {
			return fmt.Errorf("properties: %w", err)
		}
		name := tok.String() // the token is void after the next read
		pv, err := d.ReadValue()
		if err != nil {
			return fmt.Errorf("properties: %w", err)
		}
		pr, err := compileAt(pv, depth+1)
		if err != nil {
			return fmt.Errorf("properties.%s: %w", name, err)
		}
		r.props = append(r.props, prule{name, pr})
	}
	return nil
}

// prepare checks raw against r and fills its defaults, returning the
// arguments a handler gets. Empty arguments are an empty object.
func (r *rule) prepare(raw []byte) ([]byte, error) {
	empty := len(bytes.TrimSpace(raw)) == 0
	if empty {
		raw = []byte("{}")
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, &ArgError{Reason: "not valid JSON: " + reasonOf(err)}
	}
	filled := r.fill(v)
	if err := r.check(v, ""); err != nil {
		return nil, err
	}
	if !filled && !empty {
		return raw, nil
	}
	out, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return nil, &ArgError{Reason: reasonOf(err)}
	}
	return out, nil
}

// fill adds each absent property that has a default, in v and below it,
// and reports whether it added one.
func (r *rule) fill(v any) bool {
	filled := false
	switch x := v.(type) {
	case map[string]any:
		for _, p := range r.props {
			if c, ok := x[p.name]; ok {
				filled = p.r.fill(c) || filled
			} else if p.r.hasDef {
				x[p.name] = clone(p.r.def)
				filled = true
			}
		}
	case []any:
		if r.items != nil {
			for _, c := range x {
				filled = r.items.fill(c) || filled
			}
		}
	}
	return filled
}

// clone copies a decoded JSON value, so a default is never shared.
func clone(v any) any {
	switch x := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(x))
		for k, c := range x {
			m[k] = clone(c)
		}
		return m
	case []any:
		s := make([]any, len(x))
		for i, c := range x {
			s[i] = clone(c)
		}
		return s
	}
	return v
}

// check reports the first way v, at JSON Pointer path, breaks r.
func (r *rule) check(v any, path string) error {
	if len(r.types) > 0 && !slices.ContainsFunc(r.types, func(t string) bool { return isType(v, t) }) {
		return &ArgError{Path: path, Reason: fmt.Sprintf("is %s, not %s", jsonType(v), strings.Join(r.types, " or "))}
	}
	if r.enum != nil && !slices.ContainsFunc(r.enum, func(e any) bool { return reflect.DeepEqual(e, v) }) {
		return &ArgError{Path: path, Reason: "is not one of " + enumText(r.enum)}
	}
	switch x := v.(type) {
	case float64:
		if r.min != nil && x < *r.min {
			return &ArgError{Path: path, Reason: fmt.Sprintf("is less than the minimum, %s", number(*r.min))}
		}
		if r.max != nil && x > *r.max {
			return &ArgError{Path: path, Reason: fmt.Sprintf("is more than the maximum, %s", number(*r.max))}
		}
	case string:
		n := utf8.RuneCountInString(x)
		if r.minLen != nil && n < *r.minLen {
			return &ArgError{Path: path, Reason: fmt.Sprintf("is shorter than %d characters", *r.minLen)}
		}
		if r.maxLen != nil && n > *r.maxLen {
			return &ArgError{Path: path, Reason: fmt.Sprintf("is longer than %d characters", *r.maxLen)}
		}
	case []any:
		if r.items != nil {
			for i, c := range x {
				if err := r.items.check(c, path+"/"+strconv.Itoa(i)); err != nil {
					return err
				}
			}
		}
	case map[string]any:
		return r.checkObject(x, path)
	}
	return nil
}

func (r *rule) checkObject(x map[string]any, path string) error {
	for _, name := range r.required {
		if _, ok := x[name]; !ok {
			return &ArgError{Path: path + "/" + pointerToken(name), Reason: "is required"}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(x)) {
		p := path + "/" + pointerToken(name)
		switch c := r.prop(name); {
		case c != nil:
			if err := c.check(x[name], p); err != nil {
				return err
			}
		case r.closed:
			return &ArgError{Path: p, Reason: "is not a known argument"}
		case r.addl != nil:
			if err := r.addl.check(x[name], p); err != nil {
				return err
			}
		}
	}
	return nil
}

func isType(v any, t string) bool {
	switch t {
	case tObject:
		_, ok := v.(map[string]any)
		return ok
	case tArray:
		_, ok := v.([]any)
		return ok
	case tString:
		_, ok := v.(string)
		return ok
	case tBoolean:
		_, ok := v.(bool)
		return ok
	case tNumber:
		_, ok := v.(float64)
		return ok
	case tInteger:
		f, ok := v.(float64)
		return ok && f == math.Trunc(f)
	case tNull:
		return v == nil
	}
	return false
}

func jsonType(v any) string {
	switch x := v.(type) {
	case map[string]any:
		return "an object"
	case []any:
		return "an array"
	case string:
		return "a string"
	case bool:
		return "a boolean"
	case float64:
		if x == math.Trunc(x) {
			return "an integer"
		}
		return "a number"
	}
	return tNull
}

func enumText(enum []any) string {
	parts := make([]string, len(enum))
	for i, e := range enum {
		b, err := json.Marshal(e)
		if err != nil {
			b = fmt.Append(nil, e)
		}
		parts[i] = string(b)
	}
	return strings.Join(parts, ", ")
}

func number(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// pointerToken escapes name as one JSON Pointer token (RFC 6901).
func pointerToken(name string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(name)
}

// secretMask replaces a secret value in an audit record.
const secretMask = "***"

// mask is raw with every secret value replaced, for an audit record. When
// raw holds a secret and cannot be read, the record gets no arguments.
func (r *rule) mask(raw []byte) []byte {
	if r == nil || !r.anySecret || len(bytes.TrimSpace(raw)) == 0 {
		return raw
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	out, err := json.Marshal(r.masked(v), json.Deterministic(true))
	if err != nil {
		return nil
	}
	return out
}

func (r *rule) masked(v any) any {
	if r.secret {
		return secretMask
	}
	switch x := v.(type) {
	case map[string]any:
		for k, c := range x {
			if p := r.prop(k); p != nil {
				x[k] = p.masked(c)
			} else if r.addl != nil {
				x[k] = r.addl.masked(c)
			}
		}
	case []any:
		if r.items != nil {
			for i, c := range x {
				x[i] = r.items.masked(c)
			}
		}
	}
	return v
}

// durationUnmarshalers decode a time.Duration from its string form, which
// encoding/json/v2 has no default for.
var durationUnmarshalers = json.UnmarshalFunc(func(b []byte, d *time.Duration) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	x, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = x
	return nil
})

// decodeArgs decodes raw into an A strictly: unknown members are errors.
func decodeArgs[A any](raw []byte) (A, error) {
	var a A
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = []byte("{}")
	}
	err := json.Unmarshal(raw, &a, json.RejectUnknownMembers(true), json.WithUnmarshalers(durationUnmarshalers))
	if err != nil {
		return a, argError(err)
	}
	return a, nil
}

// argError turns a decoding error into an *ArgError with its path.
func argError(err error) *ArgError {
	if se, ok := errors.AsType[*json.SemanticError](err); ok {
		return &ArgError{Path: string(se.JSONPointer), Reason: reasonOf(err)}
	}
	if se, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		return &ArgError{Path: string(se.JSONPointer), Reason: reasonOf(err)}
	}
	return &ArgError{Reason: reasonOf(err)}
}

// reasonOf is err's innermost message.
func reasonOf(err error) string {
	for {
		inner := errors.Unwrap(err)
		if inner == nil {
			return err.Error()
		}
		err = inner
	}
}
