package enum

import (
	"fmt"
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

type side int

var sideNames = []string{"left", "right"}

// TestEnumInt: an enum over int, as theme's are, has the same forms, and a
// negative value has no name
// (docs/decisions/0014-PLAN-component-native-forms.md Step 10).
func TestEnumInt(t *testing.T) {
	for i, want := range sideNames {
		v := side(i)
		b, err := Marshal("theme", "Side", sideNames, v)
		var back side = 9
		if Name(sideNames, v) != want || string(b) != want || err != nil ||
			Unmarshal("theme", "Side", sideNames, b, &back) != nil || back != v {
			t.Errorf("%d: %q, %q, %v, back %d", i, Name(sideNames, v), b, err, back)
		}
	}
	for _, v := range []side{-1, 2, -100} {
		if got := Name(sideNames, v); got != fmt.Sprint(int(v)) {
			t.Errorf("Name(%d) = %q", v, got)
		}
		if _, err := Marshal("theme", "Side", sideNames, v); err == nil || err.Error() != fmt.Sprintf("theme: Side %d has no name", v) {
			t.Errorf("Marshal(%d): %v", v, err)
		}
	}
}

// TestNames: Names gives the tokens and the errors the functions give, and
// the stringer form "Type(N)" for a value with no token
// (docs/decisions/0014-PLAN-canonicalization.md Step 6).
func TestNames(t *testing.T) {
	m := Names[mux]{Pkg: "termcap", Type: "Mux", Tokens: muxNames}
	for i, want := range muxNames {
		v := mux(i)
		b, err := m.Marshal(v)
		back := mux(9)
		if m.String(v) != want || string(b) != want || err != nil || m.Unmarshal(b, &back) != nil || back != v {
			t.Errorf("%d: %q, %q, %v, back %d", i, m.String(v), b, err, back)
		}
	}
	if got := m.String(9); got != "Mux(9)" {
		t.Errorf("String(9) = %q, want \"Mux(9)\"", got)
	}
	if _, err := m.Marshal(9); err == nil || err.Error() != "termcap: Mux 9 has no name" {
		t.Errorf("Marshal(9): %v", err)
	}
	v := mux(3)
	if err := m.Unmarshal([]byte("Mux(9)"), &v); err == nil || err.Error() != `termcap: unknown Mux "Mux(9)"` || v != 3 {
		t.Errorf("Unmarshal(Mux(9)): %v, v %d", err, v)
	}

	s := Names[side]{Pkg: "theme", Type: "Side", Tokens: sideNames}
	if got := s.String(-1); got != "Side(-1)" {
		t.Errorf("String(-1) = %q, want \"Side(-1)\"", got)
	}
}

type surface uint8

var surfaces = Bits[surface]{Pkg: "command", Type: "Surface", None: "none", Tokens: []string{"key", "palette", "slash"}}

// TestBits: a bit set's text is its bits' tokens, lowest first, joined
// with "|"; None for the empty set; and a bit with no token in hex, which
// MarshalText refuses. UnmarshalText takes the tokens in any order, and
// nothing else (docs/decisions/0014-PLAN-canonicalization.md Step 6, D2).
func TestBits(t *testing.T) {
	for _, c := range []struct {
		v    surface
		text string
	}{{0, "none"}, {1, "key"}, {2, "palette"}, {5, "key|slash"}, {7, "key|palette|slash"}} {
		b, err := surfaces.Marshal(c.v)
		back := surface(0x40)
		if surfaces.String(c.v) != c.text || string(b) != c.text || err != nil ||
			surfaces.Unmarshal(b, &back) != nil || back != c.v {
			t.Errorf("%#x: %q, %q, %v, back %#x; want %q", c.v, surfaces.String(c.v), b, err, back, c.text)
		}
	}
	if got := surfaces.String(0x81); got != "key|0x80" {
		t.Errorf("String(0x81) = %q", got)
	}
	if got := surfaces.String(0x08); got != "0x8" {
		t.Errorf("String(0x08) = %q", got)
	}
	if _, err := surfaces.Marshal(0x89); err == nil || err.Error() != "command: Surface 0x88 has no name" {
		t.Errorf("Marshal(0x89): %v", err)
	}
	var v surface
	if err := surfaces.Unmarshal([]byte("slash|key"), &v); err != nil || v != 5 {
		t.Errorf("Unmarshal(slash|key) = %#x, %v", v, err)
	}
	for _, in := range []string{"", "Key", "key|", "|key", "key||slash", "key|none", "key|0x80", " key"} {
		v := surface(0x40)
		err := surfaces.Unmarshal([]byte(in), &v)
		if want := `command: unknown Surface "` + in + `"`; err == nil || err.Error() != want {
			t.Errorf("Unmarshal(%q): %v, want %q", in, err, want)
		}
		if v != 0x40 {
			t.Errorf("Unmarshal(%q) changed v to %#x", in, v)
		}
	}
}

// TestBitsEight: a set with eight tokens names every bit.
func TestBitsEight(t *testing.T) {
	eight := Bits[surface]{Pkg: "p", Type: "T", None: "none", Tokens: []string{"a", "b", "c", "d", "e", "f", "g", "h"}}
	b, err := eight.Marshal(0xff)
	if err != nil || string(b) != "a|b|c|d|e|f|g|h" {
		t.Errorf("Marshal(0xff) = %q, %v", b, err)
	}
}
