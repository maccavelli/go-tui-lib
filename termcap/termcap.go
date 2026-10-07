// Package termcap holds what a program knows about its terminal, and where
// each fact came from (docs/decisions/0005-MADR-terminal-capabilities-and-services.md).
//
// A Caps is a set of facts. Each is a Fact: a value and its Origin, so a
// report can say why the program believes it. A fact is replaced only by one
// of equal or stronger origin, and an override always wins. The environment
// comes from tea.EnvMsg as an Env, never from the process: under wish it is
// the SSH session's.
//
// A Prober learns the facts from one batch of queries that ends with DA1,
// and delivers them as a CapsMsg.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package termcap

import (
	"fmt"
	"slices"
)

// Support is what is known about one capability.
type Support uint8

const (
	// Unknown means nothing answered: not queried, or no reply before the
	// probe ended.
	Unknown Support = iota
	// Unsupported means the terminal said no, or answered a later query and
	// not this one.
	Unsupported
	// Supported means the terminal said yes.
	Supported
)

var supportNames = []string{"unknown", "unsupported", "supported"}

// String is the support's name, as the report and JSON print it.
func (s Support) String() string { return name(supportNames, s) }

// MarshalText is the support's name.
func (s Support) MarshalText() ([]byte, error) { return marshal(supportNames, "Support", s) }

// UnmarshalText reads a name MarshalText wrote.
func (s *Support) UnmarshalText(b []byte) error { return unmarshal(supportNames, "Support", b, s) }

// Origin is where a fact came from. The constants are in order of strength,
// weakest first.
type Origin uint8

const (
	// NotQueried means nothing was asked or read. The zero value.
	NotQueried Origin = iota
	// Heuristic is a guess from what else is known, such as a terminal's
	// name.
	Heuristic
	// Environment is an environment variable, from tea.EnvMsg.
	Environment
	// Queried is the terminal's reply to a query, or its silence before
	// the probe's sentinel answered.
	Queried
	// Override is the program's own setting, from a flag or a file. It
	// always wins.
	Override
)

var originNames = []string{"not-queried", "heuristic", "env", "query", "override"} // stable text

// String is the origin's name, as the report and JSON print it.
func (o Origin) String() string { return name(originNames, o) }

// MarshalText is the origin's name.
func (o Origin) MarshalText() ([]byte, error) { return marshal(originNames, "Origin", o) }

// UnmarshalText reads a name MarshalText wrote.
func (o *Origin) UnmarshalText(b []byte) error { return unmarshal(originNames, "Origin", b, o) }

// Fact is a value with its provenance, so a report can say why. Reason is
// a reason token, one of the Reason constants, that says why a fact is
// unsupported, unknown or gated; "" when there is nothing to explain.
type Fact[T any] struct {
	Value  T      `json:"value,omitzero"`
	Origin Origin `json:"origin,omitzero"`
	Reason string `json:"reason,omitzero"`
}

// Set replaces the fact with v when o is at least as strong as the fact's
// origin, and reports whether it did. An Override is replaced only by
// another Override. A replaced fact has no reason.
func (f *Fact[T]) Set(v T, o Origin) bool { return f.SetReason(v, o, "") }

// SetReason is Set with a reason token.
func (f *Fact[T]) SetReason(v T, o Origin, reason string) bool {
	if o < f.Origin {
		return false
	}
	f.Value, f.Origin, f.Reason = v, o, reason
	return true
}

// Mux is the terminal multiplexer the program runs inside.
type Mux uint8

const (
	// NoMux means no multiplexer.
	NoMux Mux = iota
	// Tmux is tmux.
	Tmux
	// Screen is GNU screen.
	Screen
	// Zellij is Zellij.
	Zellij
)

var muxNames = []string{"none", "tmux", "screen", "zellij"}

// String is the multiplexer's name, as the report and JSON print it.
func (m Mux) String() string { return name(muxNames, m) }

// MarshalText is the multiplexer's name.
func (m Mux) MarshalText() ([]byte, error) { return marshal(muxNames, "Mux", m) }

// UnmarshalText reads a name MarshalText wrote.
func (m *Mux) UnmarshalText(b []byte) error { return unmarshal(muxNames, "Mux", b, m) }

// name is v's entry in names, or its number when names has none.
func name[T ~uint8](names []string, v T) string {
	if int(v) < len(names) {
		return names[v]
	}
	return fmt.Sprintf("%d", v)
}

func marshal[T ~uint8](names []string, kind string, v T) ([]byte, error) {
	if int(v) >= len(names) {
		return nil, fmt.Errorf("termcap: %s %d has no name", kind, v)
	}
	return []byte(names[v]), nil
}

func unmarshal[T ~uint8](names []string, kind string, b []byte, v *T) error {
	i := slices.Index(names, string(b))
	if i < 0 {
		return fmt.Errorf("termcap: unknown %s %q", kind, b)
	}
	*v = T(i)
	return nil
}
