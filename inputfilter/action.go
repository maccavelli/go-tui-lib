package inputfilter

import "github.com/maccavelli/go-tui-lib/v2/internal/enum"

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "inputfilter"

// Action is what a stage did with an input: the PasteBurst and the
// Reassembler each give one from Key
// (docs/decisions/0009-PLAN-streaming-content-engine.md D13).
type Action uint8

const (
	// Pass is an input the stage did not take, with nothing ready before
	// it: the caller handles it. The zero value.
	Pass Action = iota
	// Hold is an input the stage took, with nothing ready yet.
	Hold
	// Release is an input the stage took, with inputs ready, in order,
	// which Tick gives until it reports false.
	Release
)

var actionNames = enum.Names[Action]{
	Pkg: pkgName, Type: "Action",
	Tokens: []string{"pass", "hold", "release"},
}

// String is the action's token.
func (a Action) String() string { return actionNames.String(a) }

// MarshalText is the action's token. An action with no token is an error.
func (a Action) MarshalText() ([]byte, error) { return actionNames.Marshal(a) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (a *Action) UnmarshalText(b []byte) error { return actionNames.Unmarshal(b, a) }
