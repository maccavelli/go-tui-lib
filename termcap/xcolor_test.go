package termcap

import "testing"

// TestXParseColor: XParseColor's grammar, scaled for rgb: and as the most
// significant bits for # (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md
// A2.4). It replaces TestParseXColor, which pinned the 8-bit reader.
func TestXParseColor(t *testing.T) {
	cases := []struct {
		spec    string
		r, g, b uint16
	}{
		{"rgb:f/0/8", 0xffff, 0, 0x8888},
		{"rgb:ff/a5/0", 0xffff, 0xa5a5, 0},
		{"rgb:fff/000/888", 0xffff, 0, 0x8888},
		{"rgb:ffff/0000/8080", 0xffff, 0, 0x8080},
		{"#3a7", 0x3000, 0xa000, 0x7000},
		{"#ff8000", 0xff00, 0x8000, 0},
		{"#fff000888", 0xfff0, 0, 0x8880},
		{"#ffff00008080", 0xffff, 0, 0x8080},
		{"rgba:ff/ff/ff/80", 0xffff, 0xffff, 0xffff},
	}
	for _, c := range cases {
		got, ok := xParseColor(c.spec)
		if !ok || !sameColor(got, rgb64(c.r, c.g, c.b)) {
			t.Errorf("xParseColor(%q) = %v, %v; want #%04x%04x%04x", c.spec, got, ok, c.r, c.g, c.b)
		}
	}
	for _, bad := range []string{
		"rgb:fffff/0/0", "rgb:g/0/0", "#12345", "cmyk:0/0/0/0", "rgb:1/2", "rgb://", "rgba:1/2/3",
		"#", "#ggg", "rgb:+f/0/0", "rgb:0x/0/0", "", "?",
	} {
		if c, ok := xParseColor(bad); ok || c != nil {
			t.Errorf("xParseColor(%q) = %v, %v; want no colour", bad, c, ok)
		}
	}
}
