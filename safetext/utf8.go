package safetext

import "unicode/utf8"

// decode reads the rune that s, which is not empty, begins with. ok is
// false when s does not begin with well-formed UTF-8, and n is then the
// length of its maximal subpart (the Unicode Standard, §3.9.6): the
// longest prefix of a well-formed sequence of Table 3-7 that s begins
// with, or 1 when no well-formed sequence begins there. A maximal subpart
// becomes one U+FFFD. partial is true when that prefix runs to the end of
// s, so more bytes could still complete it.
func decode[T string | []byte](s T) (r rune, n int, ok, partial bool) {
	c := s[0]
	if c < utf8.RuneSelf {
		return rune(c), 1, true, false
	}
	// Well-formed UTF-8, the common case, by the standard library's
	// table-driven decoder; it reports (RuneError, 1) for anything else,
	// and a U+FFFD that is well-formed as (RuneError, 3).
	switch v := any(s).(type) {
	case string:
		r, n = utf8.DecodeRuneInString(v)
	case []byte:
		r, n = utf8.DecodeRune(v)
	}
	if r != utf8.RuneError || n > 1 {
		return r, n, true, false
	}
	need := seqLen(c)
	if need == 1 {
		return utf8.RuneError, 1, false, false // no well-formed sequence starts with c
	}
	n = subpart(s, need)
	return utf8.RuneError, n, false, n == len(s) && n < need
}

// subpart is how many bytes of the need-byte sequence s begins with are as
// Table 3-7 allows: the lead byte, then each byte after it that is in its
// place's range, stopping at the first that is not.
func subpart[T string | []byte](s T, need int) int {
	lo, hi := second(s[0])
	n := 1
	for n < need && n < len(s) {
		c := s[n]
		if n == 1 && (c < lo || c > hi) || n > 1 && (c < 0x80 || c > 0xbf) {
			break
		}
		n++
	}
	return n
}

// second is the range Table 3-7 allows for the byte after lead, a byte that
// begins a multi-byte sequence.
func second(lead byte) (lo, hi byte) {
	switch lead {
	case 0xe0:
		return 0xa0, 0xbf
	case 0xed:
		return 0x80, 0x9f
	case 0xf0:
		return 0x90, 0xbf
	case 0xf4:
		return 0x80, 0x8f
	}
	return 0x80, 0xbf
}

// seqLen is the length of the well-formed sequence lead begins: 2 for
// C2..DF, 3 for E0..EF, 4 for F0..F4, and 1 for a byte that begins none.
func seqLen(lead byte) int {
	switch {
	case lead >= 0xc2 && lead <= 0xdf:
		return 2
	case lead >= 0xe0 && lead <= 0xef:
		return 3
	case lead >= 0xf0 && lead <= 0xf4:
		return 4
	}
	return 1
}
