package sanitize

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// FuzzStringWidth: StringWidth is m.StringWidth, for any string, under both
// methods (0023-MADR A12).
func FuzzStringWidth(f *testing.F) {
	for _, s := range []string{"", "plain words ~", "e\u0301", "\x1b[1mbold\x1b[m", "世界", "\t\r\n", "\x7f", "a\x80"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
			if got, want := StringWidth(m, s), m.StringWidth(s); got != want {
				t.Fatalf("%v %q: %d, want %d", m, s, got, want)
			}
		}
	})
}
