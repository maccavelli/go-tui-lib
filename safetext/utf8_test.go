package safetext

import (
	"strings"
	"testing"
)

// TestUnicodeTables holds Clean to the four worked examples of the Unicode
// Standard 18.0, §3.9.6 "U+FFFD Substitution of Maximal Subparts", each
// quoted from the specification: one U+FFFD for each maximal subpart
// (0020-MADR D9 S2).
func TestUnicodeTables(t *testing.T) {
	const f = "\ufffd"
	for _, c := range []struct {
		table string
		in    string
		want  string
	}{
		// Table 3-8, non-shortest form: C0 AF E0 80 BF F0 81 82 41.
		{"3-8", "\xc0\xaf\xe0\x80\xbf\xf0\x81\x82\x41", strings.Repeat(f, 8) + "A"},
		// Table 3-9, surrogates: ED A0 80 ED BF BF ED AF 41.
		{"3-9", "\xed\xa0\x80\xed\xbf\xbf\xed\xaf\x41", strings.Repeat(f, 8) + "A"},
		// Table 3-10, other ill-formed: F4 91 92 93 FF 41 80 BF 42.
		{"3-10", "\xf4\x91\x92\x93\xff\x41\x80\xbf\x42", strings.Repeat(f, 5) + "A" + f + f + "B"},
		// Table 3-11, truncated: E1 80 E2 F0 91 92 F1 BF 41.
		{"3-11", "\xe1\x80\xe2\xf0\x91\x92\xf1\xbf\x41", strings.Repeat(f, 4) + "A"},
		// 0020-MADR D9 S2's two: a truncated E2 82 is one, C0 AF two.
		{"E2 82", "\xe2\x82", f},
		{"C0 AF", "\xc0\xaf", f + f},
	} {
		if got := Clean(c.in, Text()); got != c.want {
			t.Errorf("Table %s: Clean(%q) = %q, want %q", c.table, c.in, got, c.want)
		}
	}
}

// refRanges is Table 3-7, written out plainly: each well-formed sequence's
// byte ranges, by lead byte.
var refRanges = [][][2]byte{
	{{0x00, 0x7f}},
	{{0xc2, 0xdf}, {0x80, 0xbf}},
	{{0xe0, 0xe0}, {0xa0, 0xbf}, {0x80, 0xbf}},
	{{0xe1, 0xec}, {0x80, 0xbf}, {0x80, 0xbf}},
	{{0xed, 0xed}, {0x80, 0x9f}, {0x80, 0xbf}},
	{{0xee, 0xef}, {0x80, 0xbf}, {0x80, 0xbf}},
	{{0xf0, 0xf0}, {0x90, 0xbf}, {0x80, 0xbf}, {0x80, 0xbf}},
	{{0xf1, 0xf3}, {0x80, 0xbf}, {0x80, 0xbf}, {0x80, 0xbf}},
	{{0xf4, 0xf4}, {0x80, 0x8f}, {0x80, 0xbf}, {0x80, 0xbf}},
}

// refSubpart is the reference's length of what s begins with, and whether
// it is well-formed: the longest prefix of a row of refRanges, or one byte.
func refSubpart(s string) (int, bool) {
	for _, row := range refRanges {
		if s[0] < row[0][0] || s[0] > row[0][1] {
			continue
		}
		n := 1
		for n < len(row) && n < len(s) && s[n] >= row[n][0] && s[n] <= row[n][1] {
			n++
		}
		return n, n == len(row)
	}
	return 1, false
}

// TestMaximalSubparts compares decode with the reference over every string
// of one, two and three bytes from 0x80 to 0xFF, and over every lead byte
// followed by every second byte and ASCII.
func TestMaximalSubparts(t *testing.T) {
	var inputs []string
	for a := 0x80; a <= 0xff; a++ {
		inputs = append(inputs, string([]byte{byte(a)}))
		for b := 0x80; b <= 0xff; b++ {
			inputs = append(inputs, string([]byte{byte(a), byte(b)}))
			for c := 0x80; c <= 0xff; c++ {
				inputs = append(inputs, string([]byte{byte(a), byte(b), byte(c)}))
			}
		}
	}
	for lead := 0xc0; lead <= 0xff; lead++ {
		for b := 0x00; b <= 0xff; b++ {
			inputs = append(inputs, string([]byte{byte(lead), byte(b), 'x'}))
		}
	}
	bad := 0
	for _, s := range inputs {
		for rest := s; rest != ""; {
			_, n, ok, _ := decode(rest)
			wn, wok := refSubpart(rest)
			if n != wn || ok != wok {
				if bad++; bad <= 10 {
					t.Errorf("decode(% x) = %d bytes, well-formed %v; the reference says %d, %v", rest, n, ok, wn, wok)
				}
				break
			}
			rest = rest[n:]
		}
	}
	if bad > 10 {
		t.Errorf("… and %d more", bad-10)
	}
}

// TestDecodePartial: a well-formed prefix at the end is partial, so a
// Sanitizer waits for the rest; a broken one is not.
func TestDecodePartial(t *testing.T) {
	for in, want := range map[string]bool{
		"\xe2":         true,
		"\xe2\x82":     true,
		"\xf0\x9f\x98": true,
		"\xe2\x41":     false,
		"\xc0":         false,
		"\xff":         false,
	} {
		if _, _, _, partial := decode(in); partial != want {
			t.Errorf("decode(% x): partial %v, want %v", in, partial, want)
		}
	}
}
