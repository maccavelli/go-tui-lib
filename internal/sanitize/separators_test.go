package sanitize

import "testing"

// TestSeparators: the parameter separators x/ansi's parser reads for the
// sequence at a string's start, on each path to its parameters: ESC, any
// intermediates, then [ or P; 0x9B; 0x90
// (docs/decisions/0023-MADR-line-mode-components.md A10). A sequence with
// no parameters, a string's, and text have none.
func TestSeparators(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int
	}{
		{"\x1b[1;2;3m", 2},
		{"\x1b[?1;2c", 1},
		{"\x1b[1:2:3m", 2},
		{"\x1b [1;2m", 1},
		{"\x1b !\"[;;;0m", 3},
		{"\x9b;;m", 2},
		{"\x1bP1;2q\x1b\\", 1},
		{"\x1b P;;;0q", 3},
		{"\x90>;|v\x9c", 1},
		{"\x1b[m", 0},
		{"\x1b[", 0},
		{"\x1b [", 0},
		{"\x1b]4;1;rgb:0/0/0\a", 0},
		{"\x1b(B", 0},
		{"\x1b\x1b[;;m", 0},
		{"a;b;c", 0},
		{"", 0},
	} {
		if got := Separators(c.in); got != c.want {
			t.Errorf("Separators(%q) = %d; want %d", c.in, got, c.want)
		}
	}
}
