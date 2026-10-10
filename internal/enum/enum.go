// Package enum gives an exported enum over uint8 or int its text forms from one
// table of tokens: the token String prints, and the MarshalText and
// UnmarshalText pair, which accept exactly those tokens
// (docs/decisions/0014-MADR-native-integration-api.md W0.4;
// docs/decisions/0014-PLAN-canonicalization.md Step 6). Names is the table
// of an enum, and Bits of a bit set.
//
// The errors name the package and the enum, as "termcap: unknown Mux", so a
// package that moves onto this one keeps its texts.
//
// Stability: internal.
package enum

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Value is the kinds of integer an enum may be built on.
type Value interface{ ~uint8 | ~int }

// Names is an enum's tokens: Tokens[i] is the value i's. Pkg and Type
// name the package and the enum in the errors, as "termcap" and "Mux".
type Names[T Value] struct {
	Pkg, Type string
	Tokens    []string
}

// String is v's token, or the stringer form "Type(N)" for a v with none.
func (n Names[T]) String(v T) string {
	if i, ok := index(n.Tokens, v); ok {
		return n.Tokens[i]
	}
	return n.Type + "(" + strconv.Itoa(int(v)) + ")"
}

// Marshal is v's token. A v with none is an error naming the package and
// the enum.
func (n Names[T]) Marshal(v T) ([]byte, error) {
	i, ok := index(n.Tokens, v)
	if !ok {
		return nil, fmt.Errorf("%s: %s %d has no name", n.Pkg, n.Type, v)
	}
	return []byte(n.Tokens[i]), nil
}

// Unmarshal sets *v to the value whose token b is. A b that is not exactly
// one of the tokens, case included, is an error naming the package and the
// enum, and leaves *v as it was.
func (n Names[T]) Unmarshal(b []byte, v *T) error {
	i := slices.Index(n.Tokens, string(b))
	if i < 0 {
		return fmt.Errorf("%s: unknown %s %q", n.Pkg, n.Type, b)
	}
	*v = T(i)
	return nil
}

// Bits is a bit set's tokens: Tokens[i] is the bit 1<<i's, and None is
// the empty set's. A set's text is its bits' tokens, lowest first, joined
// with "|".
type Bits[T ~uint8] struct {
	Pkg, Type string
	Tokens    []string
	None      string
}

// known is the bits Tokens names.
func (s Bits[T]) known() T { return T(1<<len(s.Tokens) - 1) }

// String is v's text, with any bits Tokens does not name last, in hex.
func (s Bits[T]) String(v T) string {
	if v == 0 {
		return s.None
	}
	var out []string
	for i, tok := range s.Tokens {
		if v&(1<<i) != 0 {
			out = append(out, tok)
		}
	}
	if rest := v &^ s.known(); rest != 0 {
		out = append(out, fmt.Sprintf("%#x", uint8(rest)))
	}
	return strings.Join(out, "|")
}

// Marshal is v's text. A v with a bit Tokens does not name is an error
// naming the package and the set.
func (s Bits[T]) Marshal(v T) ([]byte, error) {
	if rest := v &^ s.known(); rest != 0 {
		return nil, fmt.Errorf("%s: %s %#x has no name", s.Pkg, s.Type, uint8(rest))
	}
	return []byte(s.String(v)), nil
}

// Unmarshal sets *v to the set b names: None, or tokens joined with "|",
// in any order. Anything else, case included, is an error naming the
// package and the set, and leaves *v as it was.
func (s Bits[T]) Unmarshal(b []byte, v *T) error {
	if string(b) == s.None {
		*v = 0
		return nil
	}
	var set T
	for tok := range strings.SplitSeq(string(b), "|") {
		i := slices.Index(s.Tokens, tok)
		if i < 0 {
			return fmt.Errorf("%s: unknown %s %q", s.Pkg, s.Type, b)
		}
		set |= 1 << i
	}
	*v = set
	return nil
}

// index is v as an index into names, and whether names has an entry for
// it. A negative v has none.
func index[T Value](names []string, v T) (int, bool) {
	i := int(v)
	return i, i >= 0 && i < len(names)
}

// Name is v's entry in names, or its number when names has none.
func Name[T Value](names []string, v T) string {
	if i, ok := index(names, v); ok {
		return names[i]
	}
	return strconv.Itoa(int(v))
}

// Marshal is Names{pkg, kind, names}.Marshal(v).
func Marshal[T Value](pkg, kind string, names []string, v T) ([]byte, error) {
	return Names[T]{pkg, kind, names}.Marshal(v)
}

// Unmarshal is Names{pkg, kind, names}.Unmarshal(b, v).
func Unmarshal[T Value](pkg, kind string, names []string, b []byte, v *T) error {
	return Names[T]{pkg, kind, names}.Unmarshal(b, v)
}
