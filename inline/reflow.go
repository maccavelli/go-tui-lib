package inline

import "github.com/maccavelli/go-tui-lib/v2/internal/enum"

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "inline"

// Reflow is what a terminal does to its rows when its window narrows, the
// three ways gobble-cli measured (its 0030-PLAN-resize-and-performance.md
// P1). A Region reads which from the cursor report when it can, and takes
// WithReflow's kind when it cannot.
type Reflow uint8

const (
	// ReflowUnknown is a terminal not known: redrawn as ReflowDown. The
	// zero value.
	ReflowUnknown Reflow = iota
	// ReflowDown re-wraps each row at the new width and moves the cursor
	// down with its cell: WezTerm, kitty, Terminal.app and the Windows
	// pseudo-console.
	ReflowDown
	// ReflowKeepRow re-wraps each row and keeps the cursor on its screen
	// row, pushing the rows above it out of the top: iTerm2 and VTE.
	ReflowKeepRow
	// ReflowCut keeps each row on one row, cut at the new width: xterm.
	ReflowCut
)

var reflowNames = enum.Names[Reflow]{
	Pkg: pkgName, Type: "Reflow",
	Tokens: []string{"unknown", "down", "keep-row", "cut"},
}

// String is the kind's token.
func (k Reflow) String() string { return reflowNames.String(k) }

// MarshalText is the kind's token. A kind with no token is an error.
func (k Reflow) MarshalText() ([]byte, error) { return reflowNames.Marshal(k) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (k *Reflow) UnmarshalText(b []byte) error { return reflowNames.Unmarshal(b, k) }
