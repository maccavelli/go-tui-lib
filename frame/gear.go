package frame

import "github.com/maccavelli/go-tui-lib/v2/internal/enum"

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "frame"

// Gear is how a Pacer releases a queue: an item a frame, or the whole of
// it.
type Gear uint8

const (
	// Smooth releases one item a frame. The zero value.
	Smooth Gear = iota
	// CatchUp releases the whole queue each frame.
	CatchUp
)

var gearNames = enum.Names[Gear]{
	Pkg: pkgName, Type: "Gear",
	Tokens: []string{"smooth", "catch-up"},
}

// String is the gear's token.
func (g Gear) String() string { return gearNames.String(g) }

// MarshalText is the gear's token. A gear with no token is an error.
func (g Gear) MarshalText() ([]byte, error) { return gearNames.Marshal(g) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (g *Gear) UnmarshalText(b []byte) error { return gearNames.Unmarshal(b, g) }
