package safetext_test

import (
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// TestSafeURL: web and mail links pass; any other scheme, a URL holding a
// control, a space, ESC, BEL or ST, and one with no host do not.
func TestSafeURL(t *testing.T) {
	for u, want := range map[string]bool{
		"https://example.org/a?b=c#d": true,
		"HTTP://example.org":          true,
		"mailto:a@example.org":        true,
		"javascript:alert(1)":         false,
		"file:///etc/passwd":          false,
		"https://example.org/\x1b]":   false,
		"https://example.org/\a":      false,
		"https://example.org/ a":      false,
		"https://example.org/\x1b\\":  false,
		"https://example.org/\u009c":  false,
		"https://ex\u202eample.org":   false,
		"https://":                    false,
		"not a url":                   false,
	} {
		got, ok := safetext.SafeURL(u)
		if ok != want || ok && got != u {
			t.Errorf("SafeURL(%q) = %q, %v; want %v", u, got, ok, want)
		}
	}
}

// TestURLs: each span slices the cleaned text into its URL, less the
// punctuation that ends a sentence.
func TestURLs(t *testing.T) {
	text := safetext.Clean("See https://example.org/a, (and http://x.io/b). Mail mailto:me@x.io! Not ftp://x or xhttps://y.", safetext.Line())
	want := []string{"https://example.org/a", "http://x.io/b", "mailto:me@x.io"}
	var got []string
	for i, j := range safetext.URLs(text) {
		got = append(got, text[i:j])
	}
	if len(got) != len(want) {
		t.Fatalf("URLs = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("URL %d = %q, want %q", i, got[i], want[i])
		}
	}
	for range safetext.URLs("https://example.org https://example.com") {
		break // stopping early must not panic
	}
}
