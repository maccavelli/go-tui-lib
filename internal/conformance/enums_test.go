package conformance

import (
	"encoding"
	"fmt"
	"go/constant"
	"go/types"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/launch"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/termcap"
	"github.com/maccavelli/go-tui-lib/termsvc"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/when"
	"github.com/maccavelli/go-tui-lib/workspace"
)

// as is n as the enum T.
func as[T ~uint8 | ~int](n int64) any { return T(n) }

// enums builds a value of each enum the scan finds, keyed by package and
// type name. The scan decides what an enum is: an exported type of a
// public package whose underlying type is an integer. The table only lets
// the test build values, so an enum the scan finds and the table lacks
// fails, and so does an entry the scan does not find.
var enums = map[string]func(int64) any{
	"command.Danger":        as[command.Danger],
	"command.Format":        as[command.Format],
	"command.Kind":          as[command.Kind],
	"command.Mode":          as[command.Mode],
	"command.Origin":        as[command.Origin],
	"command.SourceKind":    as[command.SourceKind],
	"command.Surface":       as[command.Surface],
	"command.Verdict":       as[command.Verdict],
	"glyph.Tier":            as[glyph.Tier],
	"launch.Choice":         as[launch.Choice],
	"launch.Target":         as[launch.Target],
	"layout.Axis":           as[layout.Axis],
	"layout.SizeKind":       as[layout.SizeKind],
	"layout.Span":           as[layout.Span],
	"termcap.Brand":         as[termcap.Brand],
	"termcap.Disposition":   as[termcap.Disposition],
	"termcap.Editor":        as[termcap.Editor],
	"termcap.Mux":           as[termcap.Mux],
	"termcap.Origin":        as[termcap.Origin],
	"termcap.Platform":      as[termcap.Platform],
	"termcap.Support":       as[termcap.Support],
	"termsvc.ActivityState": as[termsvc.ActivityState],
	"termsvc.Display":       as[termsvc.Display],
	"termsvc.Policy":        as[termsvc.Policy],
	"termsvc.Protocol":      as[termsvc.Protocol],
	"termsvc.Route":         as[termsvc.Route],
	"termsvc.SkipReason":    as[termsvc.SkipReason],
	"termsvc.Status":        as[termsvc.Status],
	"termsvc.Urgency":       as[termsvc.Urgency],
	"theme.Background":      as[theme.Background],
	"theme.BorderStyle":     as[theme.BorderStyle],
	"when.Kind":             as[when.Kind],
	"workspace.AnchorKind":  as[workspace.AnchorKind],
	"workspace.Chrome":      as[workspace.Chrome],
}

// bitSets are the enums whose values are sets of bits, with
// internal/enum's Bits text.
var bitSets = map[string]bool{"command.Surface": true}

// enumType is one enum the scan found: its key in enums, its package's
// name and its own, its type, and the values of its constants with the
// zero value, ascending.
type enumType struct {
	key, pkg, name string
	typ            types.Type
	values         []int64
}

// scanEnums is every enum of every module's public packages.
func scanEnums(t *testing.T) []enumType {
	t.Helper()
	root := repoRoot(t)
	var out []enumType
	for _, dir := range modules(t, root) {
		_, _, decl := scanModule(t, root, dir)
		for _, d := range decl {
			b, ok := d.obj.Type().Underlying().(*types.Basic)
			if d.obj.IsAlias() || !ok || b.Info()&types.IsInteger == 0 {
				continue
			}
			e := enumType{key: d.pkg + "." + d.name, pkg: d.obj.Pkg().Name(), name: d.name, typ: d.obj.Type(), values: []int64{0}}
			scope := d.obj.Pkg().Scope()
			for _, n := range scope.Names() {
				c, ok := scope.Lookup(n).(*types.Const)
				if !ok || !types.Identical(c.Type(), e.typ) {
					continue
				}
				v, exact := constant.Int64Val(c.Val())
				if !exact {
					t.Fatalf("%s.%s is not an int64", e.key, n)
				}
				if !slices.Contains(e.values, v) {
					e.values = append(e.values, v)
				}
			}
			slices.Sort(e.values)
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		t.Fatal("the scan found no enum")
	}
	return out
}

// texts is v's String, MarshalText and a new *T's UnmarshalText, or why
// it lacks one.
func texts(v any) (fmt.Stringer, encoding.TextMarshaler, reflect.Value, encoding.TextUnmarshaler, string) {
	s, ok := v.(fmt.Stringer)
	if !ok {
		return nil, nil, reflect.Value{}, nil, "String"
	}
	m, ok := v.(encoding.TextMarshaler)
	if !ok {
		return nil, nil, reflect.Value{}, nil, "MarshalText"
	}
	p := reflect.New(reflect.TypeOf(v))
	u, ok := p.Interface().(encoding.TextUnmarshaler)
	if !ok {
		return nil, nil, reflect.Value{}, nil, "UnmarshalText"
	}
	return s, m, p, u, ""
}

// TestEveryEnumRoundTrips: every enum the scan finds has String and
// MarshalText, and UnmarshalText on its pointer; and each of its constants,
// and its zero value, has one lowercase token of its own, which String
// prints, MarshalText writes and UnmarshalText reads back
// (AGENTS.md, "API conventions" rule 5;
// docs/decisions/0014-PLAN-canonicalization.md Step 6).
func TestEveryEnumRoundTrips(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range scanEnums(t) {
		seen[e.key] = true
		for _, m := range []struct {
			recv types.Type
			name string
		}{{e.typ, "String"}, {e.typ, "MarshalText"}, {types.NewPointer(e.typ), "UnmarshalText"}} {
			if types.NewMethodSet(m.recv).Lookup(nil, m.name) == nil {
				t.Errorf("%s has no %s method: an exported enum has String, MarshalText and UnmarshalText", e.key, m.name)
			}
		}
		conv, ok := enums[e.key]
		if !ok {
			t.Errorf("%s is an enum the table lacks: add it to enums", e.key)
			continue
		}
		tokens := map[string]int64{}
		for i, n := range e.values {
			v := conv(n)
			s, m, p, u, missing := texts(v)
			if missing != "" {
				t.Errorf("%s(%d) has no %s method", e.key, n, missing)
				break
			}
			b, err := m.MarshalText()
			if err != nil {
				t.Errorf("%s(%d).MarshalText: %v", e.key, n, err)
				continue
			}
			if string(b) != s.String() || string(b) != strings.ToLower(string(b)) {
				t.Errorf("%s(%d): String %q, MarshalText %q; want one lowercase token", e.key, n, s.String(), b)
			}
			if other, dup := tokens[string(b)]; dup {
				t.Errorf("%s: %d and %d share the token %q", e.key, other, n, b)
			}
			tokens[string(b)] = n
			// Start from another value, so that an UnmarshalText that does
			// nothing cannot pass for the zero value.
			p.Elem().Set(reflect.ValueOf(conv(e.values[(i+1)%len(e.values)])))
			if err := u.UnmarshalText(b); err != nil || p.Elem().Interface() != v {
				t.Errorf("%s: UnmarshalText(%q) = %v, %v; want %v", e.key, b, p.Elem().Interface(), err, n)
			}
		}
	}
	for key := range enums {
		if !seen[key] {
			t.Errorf("enums lists %s, which the scan did not find", key)
		}
	}
}

// TestOutOfRangeText: a value no constant names prints the stringer form
// "Type(N)", and MarshalText refuses it, naming the package and the enum
// (docs/decisions/0014-PLAN-canonicalization.md Step 6). A bit set prints
// the bits it cannot name in hex, after the ones it can.
func TestOutOfRangeText(t *testing.T) {
	n := 0
	for _, e := range scanEnums(t) {
		conv, ok := enums[e.key]
		if !ok {
			continue // TestEveryEnumRoundTrips reports it
		}
		if bitSets[e.key] {
			var all int64
			for _, v := range e.values {
				all |= v
			}
			bit := int64(1) << 7
			if all&bit != 0 || all&1 == 0 {
				t.Fatalf("%s: the test wants bit 0 named and bit 7 free, in %#x", e.key, all)
			}
			s, m, _, _, missing := texts(conv(1 | bit))
			if missing != "" {
				t.Errorf("%s has no %s method", e.key, missing)
				continue
			}
			low, _, _, _, _ := texts(conv(1))
			if want := fmt.Sprintf("%s|%#x", low, bit); s.String() != want {
				t.Errorf("%s(%#x).String() = %q, want %q", e.key, 1|bit, s.String(), want)
			}
			if _, err := m.MarshalText(); err == nil || err.Error() != fmt.Sprintf("%s: %s %#x has no name", e.pkg, e.name, bit) {
				t.Errorf("%s(%#x).MarshalText: %v", e.key, 1|bit, err)
			}
			n++
			continue
		}
		out := []int64{e.values[len(e.values)-1] + 1}
		if b, _ := e.typ.Underlying().(*types.Basic); b.Info()&types.IsUnsigned == 0 {
			out = append(out, e.values[0]-1)
		}
		for _, v := range out {
			s, m, _, u, missing := texts(conv(v))
			if missing != "" {
				t.Errorf("%s has no %s method", e.key, missing)
				break
			}
			want := fmt.Sprintf("%s(%d)", e.name, v)
			if s.String() != want {
				t.Errorf("%s(%d).String() = %q, want %q", e.key, v, s.String(), want)
			}
			if _, err := m.MarshalText(); err == nil || err.Error() != fmt.Sprintf("%s: %s %d has no name", e.pkg, e.name, v) {
				t.Errorf("%s(%d).MarshalText: %v", e.key, v, err)
			}
			if err := u.UnmarshalText([]byte(want)); err == nil {
				t.Errorf("%s: UnmarshalText(%q) took the out-of-range text", e.key, want)
			}
			n++
		}
	}
	if n == 0 {
		t.Fatal("no enum was checked")
	}
}
