package workspace

import (
	"slices"
	"strings"
	"testing"
)

// introducers are the bytes that, after ESC, open a control sequence in
// legacy key encoding, where alt+x is ESC x: CSI, OSC, SS3, DCS, APC, PM,
// SOS and ST.
var introducers = []string{"[", "]", "O", "P", "_", "^", "X", `\`, "shift+o", "shift+p", "shift+x"}

func TestDefaultKeysAvoidSequenceIntroducers(t *testing.T) {
	for _, b := range DefaultKeyMap().Bindings() {
		for _, k := range b.Keys() {
			rest, ok := strings.CutPrefix(k, "alt+")
			if !ok {
				continue
			}
			if slices.Contains(introducers, rest) {
				t.Errorf("default %q is alt and a sequence introducer, which a legacy terminal cannot tell from the sequence", k)
			}
		}
	}
}
