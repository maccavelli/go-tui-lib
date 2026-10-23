package safetext

import "strconv"

// isBreak reports whether r is a line break or a tab: what Flatten turns
// into a space, and what Tabs and Newlines decide.
func isBreak(r rune) bool {
	return r == '\r' || r == '\n' || r == '\t' || r == 0x2028 || r == 0x2029
}

// invisible reports whether r is a format character no rule keeps: U+200B,
// U+FEFF, U+2060 to U+2064, U+00AD and the tag characters
// (0020-MADR D9 S1, as internal/sanitize has removed them since v0.8.0).
// ZWJ, ZWNJ and the variation selectors are not among them.
func invisible(r rune) bool {
	switch {
	case r == 0x200b, r == 0xfeff, r == 0x00ad:
		return true
	case r >= 0x2060 && r <= 0x2064:
		return true
	case r >= 0xe0000 && r <= 0xe007f:
		return true
	}
	return false
}

// bidi reports whether r is in the Bidi_Control set of Unicode's
// PropList.txt, the twelve code points of Trojan Source (CVE-2021-42574).
func bidi(r rune) bool {
	switch {
	case r == 0x061c, r == 0x200e, r == 0x200f:
		return true
	case r >= 0x202a && r <= 0x202e:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	}
	return false
}

// appendEscaped appends r as strconv.QuoteToASCII writes it, without the
// quotes: \x1b, \n, \u202e, \U000e0041.
func appendEscaped(dst []byte, r rune) []byte {
	var tmp [16]byte
	q := strconv.AppendQuoteRuneToASCII(tmp[:0], r)
	return append(dst, q[1:len(q)-1]...)
}
