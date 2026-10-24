package termcap

import "github.com/maccavelli/go-tui-lib/v2/internal/enum"

// ColorProfile is how many colours the terminal shows. Its values are
// colorprofile's, in the same order, so a Bubble Tea program converts with
// termcaptea.FromColorProfile and ToColorProfile; termcap itself detects
// none, and a host sets it with Probe.SetColorProfile.
type ColorProfile uint8

const (
	// ProfileUnknown means no profile is known. The zero value.
	ProfileUnknown ColorProfile = iota
	// ProfileNoTTY is output that is not a terminal.
	ProfileNoTTY
	// ProfileASCII is a terminal with no colour.
	ProfileASCII
	// ProfileANSI is 16 colours.
	ProfileANSI
	// ProfileANSI256 is 256 colours.
	ProfileANSI256
	// ProfileTrueColor is 24-bit colour.
	ProfileTrueColor
)

var colorProfileNames = enum.Names[ColorProfile]{
	Pkg: pkgName, Type: "ColorProfile", Unknown: ErrUnknownName,
	Tokens: []string{tokenUnknown, "notty", "ascii", "ansi", "ansi256", "truecolor"},
}

// colorProfileSpellings are colorprofile's own names, which a report of
// schema version 1 wrote, in ColorProfile's order.
var colorProfileSpellings = []string{"Unknown", "NoTTY", "Ascii", "ANSI", "ANSI256", "TrueColor"}

// String is the profile's name, as the report and JSON print it.
func (p ColorProfile) String() string { return colorProfileNames.String(p) }

// MarshalText is the profile's name.
func (p ColorProfile) MarshalText() ([]byte, error) { return colorProfileNames.Marshal(p) }

// UnmarshalText reads a name MarshalText wrote.
func (p *ColorProfile) UnmarshalText(b []byte) error { return colorProfileNames.Unmarshal(b, p) }

// parseProfile reads a profile's token, or its colorprofile spelling.
func parseProfile(s string) (ColorProfile, bool) {
	var p ColorProfile
	if p.UnmarshalText([]byte(s)) == nil {
		return p, true
	}
	for i, name := range colorProfileSpellings {
		if name == s {
			return ColorProfile(i), true
		}
	}
	return ProfileUnknown, false
}
