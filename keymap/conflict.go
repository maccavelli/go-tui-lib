package keymap

import (
	"fmt"
	"slices"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/internal/enum"
)

// ConflictKind is what is wrong with a binding (MADR §4, amendment A6).
type ConflictKind uint8

// The kinds. Each but Shadow is an error New returns; Conflicts lists the
// Shadows.
const (
	Duplicate    ConflictKind = iota // one scope and layer bind the keys twice, with the same when
	Prefix                           // in one scope, one binding's keys start another's
	Reserved                         // a user binding starts with a key the program reserves
	NotSupported                     // a binding needs features the terminal has not reported
	Shadow                           // an inner scope, or a later layer, hides a binding
)

var conflictNames = enum.Names[ConflictKind]{
	Pkg: pkgName, Type: "ConflictKind",
	Tokens: []string{"duplicate", "prefix", "reserved", "not-supported", "shadow"},
}

// String is the kind's token.
func (c ConflictKind) String() string { return conflictNames.String(c) }

// MarshalText is the kind's token. A kind with no token is an error.
func (c ConflictKind) MarshalText() ([]byte, error) { return conflictNames.Marshal(c) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (c *ConflictKind) UnmarshalText(b []byte) error { return conflictNames.Unmarshal(b, c) }

// BindingConflict is a binding New refuses, or one it warns of.
type BindingConflict struct {
	Kind     ConflictKind
	Keys     Sequence      // the keys in conflict: for Prefix the shorter, for Reserved the reserved ones
	Scope    command.Scope // where they conflict
	Bindings []Binding     // the bindings involved, with their locations; the last is the one at fault
}

// Error names the binding at fault by its location, keys and scope, then
// what is wrong: "keybindings.json:12:5 (/3/key): ctrl+k: ...".
func (c BindingConflict) Error() string {
	if len(c.Bindings) == 0 {
		return fmt.Sprintf("keymap: %s %s", c.Kind, c.Keys)
	}
	last := c.Bindings[len(c.Bindings)-1]
	var other Binding
	if len(c.Bindings) > 1 {
		other = c.Bindings[0]
	}
	var what string
	switch c.Kind {
	case Duplicate:
		what = fmt.Sprintf("bound twice in one layer with the same when, also to %s at %s", other.Command, other.Location)
	case Prefix:
		what = fmt.Sprintf("one of %s and %s (%s at %s) starts the other, so the shorter could never wait for the longer",
			last.Keys, other.Keys, other.Command, other.Location)
	case Reserved:
		what = fmt.Sprintf("starts with %s, which the program reserves", c.Keys)
	case NotSupported:
		what = fmt.Sprintf("needs %s, which the terminal has not reported", requirement(c.Keys))
	case Shadow:
		what = fmt.Sprintf("hides %s's %s at %s", other.Command, other.Keys, other.Location)
	default:
		what = c.Kind.String()
	}
	return where(last) + ": " + what
}

// requirement is what a sequence needs: its features, and the Releases
// fact for a release or repeat stroke.
func requirement(q Sequence) string {
	s := needs(q).String()
	if f, _ := keyFacts(q); f != 0 {
		s += " and " + f.String()
	}
	return s
}

// check finds the conflicts among the bindings in effect: in each scope,
// Duplicate and Prefix errors and a later layer's Shadow; and a scope's
// Shadow of a global binding, since command.Global is on every path.
func (b *builder) check() {
	k := b.k
	seen := map[command.Scope]map[Stroke][]*entry{}
	for _, e := range k.bound {
		byStroke := seen[e.Scope]
		if byStroke == nil {
			byStroke = map[Stroke][]*entry{}
			seen[e.Scope] = byStroke
		}
		for _, o := range byStroke[e.Keys[0]] {
			pair := []Binding{o.Binding, e.Binding}
			switch {
			case !related(o.Keys, e.Keys):
			case !slices.Equal(o.Keys, e.Keys):
				short := min(len(o.Keys), len(e.Keys))
				b.errs = append(b.errs, BindingConflict{Kind: Prefix, Keys: e.Keys[:short], Scope: e.Scope, Bindings: pair})
			case o.layer == e.layer && o.When == e.When:
				b.errs = append(b.errs, BindingConflict{Kind: Duplicate, Keys: e.Keys, Scope: e.Scope, Bindings: pair})
			case o.layer != e.layer:
				k.conflicts = append(k.conflicts, BindingConflict{Kind: Shadow, Keys: e.Keys, Scope: e.Scope, Bindings: pair})
			}
		}
		byStroke[e.Keys[0]] = append(byStroke[e.Keys[0]], e)
	}
	global := seen[command.Global]
	for _, e := range k.bound {
		if e.Scope == command.Global {
			continue
		}
		for _, o := range global[e.Keys[0]] {
			if related(o.Keys, e.Keys) {
				k.conflicts = append(k.conflicts, BindingConflict{Kind: Shadow, Keys: e.Keys, Scope: e.Scope, Bindings: []Binding{o.Binding, e.Binding}})
			}
		}
	}
}
