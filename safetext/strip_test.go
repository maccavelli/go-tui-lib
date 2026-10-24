package safetext_test

import (
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// TestAgainstStrip is 0009-PLAN Step 2's differential: over generated
// well-formed CSI, OSC and DCS sequences mixed with valid UTF-8 text, Clean
// under Strict equals x/ansi's Strip with every C0 control but tab and
// line feed removed. The test file alone imports x/ansi; safetext does not.
func TestAgainstStrip(t *testing.T) {
	rng := rand.New(rand.NewPCG(20, 26)) // fixed: the same inputs on every run
	words := []string{"plain", "日本語", "😀", "é", " ", "\t", "\n", "\r\n", "x", "Ω"}
	piece := func() string {
		switch rng.IntN(6) {
		case 0:
			var p []string
			for range rng.IntN(4) {
				p = append(p, strconv.Itoa(rng.IntN(300)))
			}
			return "\x1b[" + strings.Join(p, ";") + string(rune('@'+rng.IntN(63)))
		case 1:
			term := "\x07"
			if rng.IntN(2) == 0 {
				term = "\x1b\\"
			}
			return "\x1b]" + strconv.Itoa(rng.IntN(120)) + ";payload " + strconv.Itoa(rng.IntN(1e6)) + term
		case 2:
			return "\x1bP" + strconv.Itoa(rng.IntN(9)) + "$q data\x1b\\"
		default:
			return words[rng.IntN(len(words))]
		}
	}
	for range 2000 {
		var b strings.Builder
		for range 1 + rng.IntN(12) {
			b.WriteString(piece())
		}
		in := b.String()
		want := strings.Map(func(r rune) rune {
			if r < 0x20 && r != '\t' && r != '\n' {
				return -1
			}
			return r
		}, ansi.Strip(in))
		if got := safetext.Clean(in, safetext.Strict()); got != want {
			t.Fatalf("Clean(%q, Strict) = %q; Strip with the controls removed gives %q", in, got, want)
		}
	}
}
