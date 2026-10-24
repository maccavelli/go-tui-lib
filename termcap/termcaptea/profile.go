package termcaptea

import (
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// FromColorProfile is colorprofile's profile as termcap's. A value
// colorprofile does not name is termcap.ProfileUnknown.
func FromColorProfile(p colorprofile.Profile) termcap.ColorProfile {
	if p > colorprofile.TrueColor {
		return termcap.ProfileUnknown
	}
	return termcap.ColorProfile(p)
}

// ToColorProfile is termcap's profile as colorprofile's. A value termcap
// does not name is colorprofile.Unknown.
func ToColorProfile(c termcap.ColorProfile) colorprofile.Profile {
	if c > termcap.ProfileTrueColor {
		return colorprofile.Unknown
	}
	return colorprofile.Profile(c)
}
