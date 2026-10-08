// Package enum gives an exported enum over uint8 its text forms from one
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

// Name is v's entry in names, or its number when names has none.
func Name[T ~uint8](names []string, v T) string {
	if int(v) < len(names) {
		return names[v]
	}
	return strconv.Itoa(int(v))
}

// Marshal is v's entry in names. A v that names has no entry for is an
// error naming pkg and kind.
func Marshal[T ~uint8](pkg, kind string, names []string, v T) ([]byte, error) {
	if int(v) >= len(names) {
		return nil, fmt.Errorf("%s: %s %d has no name", pkg, kind, v)
	}
	return []byte(names[v]), nil
}

// Unmarshal sets *v to the index of b in names. A b that is not exactly one
// of names, case included, is an error naming pkg and kind, and leaves *v
// as it was.
func Unmarshal[T ~uint8](pkg, kind string, names []string, b []byte, v *T) error {
	i := slices.Index(names, string(b))
	if i < 0 {
		return fmt.Errorf("%s: unknown %s %q", pkg, kind, b)
	}
	*v = T(i)
	return nil
}
