package termcaptea

import (
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// TestColorProfileRoundTrip: every colorprofile value round-trips through
// termcap's, whose token is colorprofile's name in lower case; a value
// neither names is unknown on the other side.
func TestColorProfileRoundTrip(t *testing.T) {
	for p := colorprofile.Unknown; p <= colorprofile.TrueColor; p++ {
		c := FromColorProfile(p)
		if back := ToColorProfile(c); back != p {
			t.Errorf("%v: back as %v", p, back)
		}
		if c.String() != strings.ToLower(p.String()) {
			t.Errorf("%v: token %q, want %q", p, c.String(), strings.ToLower(p.String()))
		}
	}
	if FromColorProfile(colorprofile.TrueColor+1) != termcap.ProfileUnknown || ToColorProfile(termcap.ProfileTrueColor+1) != colorprofile.Unknown {
		t.Error("an unnamed profile is not unknown")
	}
}
