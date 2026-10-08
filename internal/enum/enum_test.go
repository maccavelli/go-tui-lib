package enum

import (
	"testing"
)

type mux uint8

var muxNames = []string{"none", "tmux", "screen", "zellij"}

// TestEnumNames round-trips every name, and checks that the error texts are
// termcap's, byte for byte, as they were before termcap moved onto this
// package.
func TestEnumNames(t *testing.T) {
	for i, want := range muxNames {
		v := mux(i)
		if got := Name(muxNames, v); got != want {
			t.Errorf("Name(%d) = %q, want %q", i, got, want)
		}
		b, err := Marshal("termcap", "Mux", muxNames, v)
		if err != nil || string(b) != want {
			t.Errorf("Marshal(%d) = %q, %v; want %q", i, b, err, want)
		}
		var back mux
		if err := Unmarshal("termcap", "Mux", muxNames, b, &back); err != nil || back != v {
			t.Errorf("Unmarshal(%q) = %d, %v; want %d", b, back, err, v)
		}
	}

	if got := Name(muxNames, mux(9)); got != "9" {
		t.Errorf("Name(9) = %q, want \"9\"", got)
	}
	if _, err := Marshal("termcap", "Mux", muxNames, mux(9)); err == nil || err.Error() != "termcap: Mux 9 has no name" {
		t.Errorf("Marshal(9): %v", err)
	}

	// Only the exact name is accepted, and a refused one leaves v alone.
	for _, in := range []string{"TMUX", "Tmux", " tmux", "", "9"} {
		v := mux(3)
		err := Unmarshal("termcap", "Mux", muxNames, []byte(in), &v)
		if want := `termcap: unknown Mux "` + in + `"`; err == nil || err.Error() != want {
			t.Errorf("Unmarshal(%q): %v, want %q", in, err, want)
		}
		if v != 3 {
			t.Errorf("Unmarshal(%q) changed v to %d", in, v)
		}
	}
}
