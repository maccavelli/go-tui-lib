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
	"errors"

	"github.com/maccavelli/go-tui-lib/internal/enum"
)

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "termcap"

// ErrUnknownName is wrapped by the error for a name termcap does not know:
// from each enum's UnmarshalText, and from Caps.UnmarshalJSON for a colour
// profile. The errors' texts name the type and the name.
var ErrUnknownName = errors.New("termcap: unknown name")

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

var supportNames = enum.Names[Support]{
	Pkg: pkgName, Type: "Support", Unknown: ErrUnknownName,
	Tokens: []string{"unknown", "unsupported", "supported"},
}

// String is the support's name, as the report and JSON print it.
func (s Support) String() string { return supportNames.String(s) }

// MarshalText is the support's name.
func (s Support) MarshalText() ([]byte, error) { return supportNames.Marshal(s) }

// UnmarshalText reads a name MarshalText wrote.
func (s *Support) UnmarshalText(b []byte) error { return supportNames.Unmarshal(b, s) }

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

var originNames = enum.Names[Origin]{
	Pkg: pkgName, Type: "Origin", Unknown: ErrUnknownName,
	Tokens: []string{"not-queried", "heuristic", "env", "query", "override"}, // stable text
}

// String is the origin's name, as the report and JSON print it.
func (o Origin) String() string { return originNames.String(o) }

// MarshalText is the origin's name.
func (o Origin) MarshalText() ([]byte, error) { return originNames.Marshal(o) }

// UnmarshalText reads a name MarshalText wrote.
func (o *Origin) UnmarshalText(b []byte) error { return originNames.Unmarshal(b, o) }

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

var muxNames = enum.Names[Mux]{
	Pkg: pkgName, Type: "Mux", Unknown: ErrUnknownName,
	Tokens: []string{"none", "tmux", "screen", "zellij"},
}

// String is the multiplexer's name, as the report and JSON print it.
func (m Mux) String() string { return muxNames.String(m) }

// MarshalText is the multiplexer's name.
func (m Mux) MarshalText() ([]byte, error) { return muxNames.Marshal(m) }

// UnmarshalText reads a name MarshalText wrote.
func (m *Mux) UnmarshalText(b []byte) error { return muxNames.Unmarshal(b, m) }
