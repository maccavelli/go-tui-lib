// Package enum gives an exported enum over uint8 or int its text forms from one
// table of names: the name String prints, and the MarshalText and
// UnmarshalText pair, which accept exactly those names
// (docs/decisions/0014-MADR-native-integration-api.md W0.4;
// docs/decisions/0013-PLAN-cli-integration-helpers.md Step 2).
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
)

// Value is the kinds of integer an enum may be built on.
type Value interface{ ~uint8 | ~int }

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

// Marshal is v's entry in names. A v that names has no entry for is an
// error naming pkg and kind.
func Marshal[T Value](pkg, kind string, names []string, v T) ([]byte, error) {
	i, ok := index(names, v)
	if !ok {
		return nil, fmt.Errorf("%s: %s %d has no name", pkg, kind, v)
	}
	return []byte(names[i]), nil
}

// Unmarshal sets *v to the index of b in names. A b that is not exactly one
// of names, case included, is an error naming pkg and kind, and leaves *v
// as it was.
func Unmarshal[T Value](pkg, kind string, names []string, b []byte, v *T) error {
	i := slices.Index(names, string(b))
	if i < 0 {
		return fmt.Errorf("%s: unknown %s %q", pkg, kind, b)
	}
	*v = T(i)
	return nil
}
