package when

import (
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/go-tui-lib/internal/enum"
)

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "when"

// Kind is what a Value holds.
type Kind uint8

// Kinds. The zero Kind is a zero Value's, which holds nothing.
const (
	KindBool Kind = iota + 1
	KindNumber
	KindString
	KindList
)

var kindNames = enum.Names[Kind]{
	Pkg: pkgName, Type: "Kind",
	Tokens: []string{"nothing", "boolean", "number", "string", "list"},
}

// String is the kind's token, as Check's errors name it.
func (k Kind) String() string { return kindNames.String(k) }

// MarshalText is the kind's token. A kind with no token is an error.
func (k Kind) MarshalText() ([]byte, error) { return kindNames.Marshal(k) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (k *Kind) UnmarshalText(b []byte) error { return kindNames.Unmarshal(b, k) }

// Value is a context key's value: a boolean, a number, a string or a list
// of strings. Build one with BoolValue, NumberValue, StringValue or
// ListValue.
type Value struct {
	kind Kind
	b    bool
	n    float64
	s    string
	l    []string
}

// BoolValue is a boolean value.
func BoolValue(b bool) Value { return Value{kind: KindBool, b: b} }

// NumberValue is a number value. An int is held as a number.
func NumberValue(f float64) Value { return Value{kind: KindNumber, n: f} }

// StringValue is a string value.
func StringValue(s string) Value { return Value{kind: KindString, s: s} }

// ListValue is a list of strings, which `in` tests membership of.
func ListValue(l []string) Value { return Value{kind: KindList, l: slices.Clone(l)} }

// Kind is what v holds.
func (v Value) Kind() Kind { return v.kind }

// Bool is v's truth when its key stands alone in an expression: false for
// false, 0, "", an empty list and the zero Value; true otherwise.
func (v Value) Bool() bool {
	switch v.kind {
	case KindBool:
		return v.b
	case KindNumber:
		return v.n != 0
	case KindString:
		return v.s != ""
	case KindList:
		return len(v.l) > 0
	}
	return false
}

// Number is v's number, and whether it is one.
func (v Value) Number() (float64, bool) { return v.n, v.kind == KindNumber }

// String is v's canonical text: true or false, a number with no exponent,
// the string itself, or a list's items joined with commas.
func (v Value) String() string {
	switch v.kind {
	case KindBool:
		return strconv.FormatBool(v.b)
	case KindNumber:
		return formatNumber(v.n)
	case KindString:
		return v.s
	case KindList:
		return strings.Join(v.l, ",")
	}
	return ""
}

// List is v's list, and whether it is one.
func (v Value) List() ([]string, bool) { return slices.Clone(v.l), v.kind == KindList }

func formatNumber(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// Context gives an expression the values of its keys.
type Context interface {
	// Value is key's value, and whether key is set.
	Value(key string) (Value, bool)
}

// Map is a simple Context.
type Map map[string]Value

// Value is m[key], and whether it is set.
func (m Map) Value(key string) (Value, bool) {
	v, ok := m[key]
	return v, ok
}

// Layered is a Context that asks each of cs in turn: the first that holds
// a key gives its value. A workspace layers the top overlay's context, then
// the focused pane's, then its own.
func Layered(cs ...Context) Context { return layered(slices.Clone(cs)) }

type layered []Context

func (l layered) Value(key string) (Value, bool) {
	for _, c := range l {
		if c == nil {
			continue
		}
		if v, ok := c.Value(key); ok {
			return v, true
		}
	}
	return Value{}, false
}
