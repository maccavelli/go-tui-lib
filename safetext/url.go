package safetext

import (
	"iter"
	"net/url"
	"strings"
	"unicode/utf8"
)

// SafeURL reports whether u may be the target of a trusted OSC 8 link, and
// returns it unchanged when it may: an http, https or mailto URL, the
// scheme compared without case, that parses, has a host or an address, and
// holds no control, no space, no ESC and no ST, so that a link written
// with it cannot be broken out of.
func SafeURL(u string) (string, bool) {
	if !utf8.ValidString(u) {
		return "", false
	}
	for _, r := range u {
		if r <= 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f || bidi(r) || invisible(r) {
			return "", false
		}
	}
	p, err := url.Parse(u)
	if err != nil || p.Host == "" && p.Opaque == "" {
		return "", false
	}
	switch strings.ToLower(p.Scheme) {
	case "http", "https", "mailto":
		return u, true
	}
	return "", false
}

// URLs yields the byte span of each web URL in s, cleaned text, as start
// and end: each run that begins with http://, https:// or mailto:, the
// scheme in any case, and goes on to the next space or control, less the
// punctuation a sentence ends a URL with (. , ; : ! ? ' " and a closing
// bracket with no opening one inside the URL). A span SafeURL refuses is
// not yielded.
func URLs(s string) iter.Seq2[int, int] {
	return func(yield func(int, int) bool) {
		for i := 0; i < len(s); {
			j := nextScheme(s, i)
			if j < 0 {
				return
			}
			end := j
			for end < len(s) {
				r, n := utf8.DecodeRuneInString(s[end:])
				if r <= 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f {
					break
				}
				end += n
			}
			end = trimTail(s[j:end]) + j
			if _, ok := SafeURL(s[j:end]); ok {
				if !yield(j, end) {
					return
				}
			}
			i = max(end, j+1)
		}
	}
}

// nextScheme is where the next http://, https:// or mailto: begins in s
// from i, at a word's start, or -1.
func nextScheme(s string, i int) int {
	for ; i < len(s); i++ {
		if i > 0 && isWordByte(s[i-1]) {
			continue
		}
		rest := s[i:]
		for _, p := range [...]string{"https://", "http://", "mailto:"} {
			if len(rest) >= len(p) && strings.EqualFold(rest[:len(p)], p) {
				return i
			}
		}
	}
	return -1
}

func isWordByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}

// trimTail is the length of u less the punctuation that ends a sentence
// rather than the URL.
func trimTail(u string) int {
	n := len(u)
	for n > 0 {
		switch u[n-1] {
		case '.', ',', ';', ':', '!', '?', '\'', '"':
			n--
			continue
		case ')':
			if !strings.Contains(u[:n-1], "(") {
				n--
				continue
			}
		case ']':
			if !strings.Contains(u[:n-1], "[") {
				n--
				continue
			}
		}
		return n
	}
	return n
}
