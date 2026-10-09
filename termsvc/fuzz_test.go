package termsvc

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/internal/sanitize"
)

// FuzzSanitizers: no character Line drops survives ParseActivity's vendor,
// Link's URL, parameter and text, or SanitizeTitle, outside Link's own OSC
// 8 wrapper; and a title is at most TitleRunes runes
// (docs/decisions/0014-PLAN-hardening.md Step 9, finding H9).
func FuzzSanitizers(f *testing.F) {
	now := time.UnixMilli(1_700_000_000_000)
	ms := strconv.FormatInt(now.UnixMilli(), 10)
	for _, c := range [][4]string{
		{"kilo;activity;1;waiting;" + ms, "https://example.com", "site", "id=x"},
		{"ev\x1b[31m\u202eil\u200b;activity;1;busy;" + ms, "https://example.com/\x1b]8;;evil\x07x", "te\x1bxt\x9b", "id=\x07a"},
		{"kilo;activity;1;sleeping;" + ms, "javascript:alert(1)", "a\u2028b\u2029c", "a:b"},
		{"\x00;activity;1;done;" + ms, "file:///tmp/a\tb", "\ufeff\u2060\u00ad", ""},
		{"kilo;activity;2;busy;" + ms, "mailto:a@b", strings.Repeat("x", 300), "id=\u202e"},
		{"k\xffi;activity;1;retry;" + ms, "HTTP://EXAMPLE.COM", "\U000e0041tag", "\x1b\\"},
	} {
		f.Add(c[0], c[1], c[2], c[3])
	}
	f.Fuzz(func(t *testing.T, payload, rawURL, text, param string) {
		if r, err := ParseActivity(payload, now); err == nil && sanitize.HasControl(r.Vendor) {
			t.Fatalf("ParseActivity(%q): vendor %q holds what Line drops", payload, r.Vendor)
		}
		if title := SanitizeTitle(text); sanitize.HasControl(title) || len([]rune(title)) > TitleRunes {
			t.Fatalf("SanitizeTitle(%q) = %q: what Line drops, or over %d runes", text, title, TitleRunes)
		}
		out, err := Link(rawURL, text, param)
		if err != nil {
			return
		}
		reset := ansi.ResetHyperlink()
		inner, ok := strings.CutPrefix(out, "\x1b]8;")
		inner, ok2 := strings.CutSuffix(inner, reset)
		if !ok || !ok2 {
			t.Fatalf("Link(%q, %q, %q) = %q: not one OSC 8 link", rawURL, text, param, out)
		}
		// The opening sequence ends at its BEL or ST: the URL and the
		// parameter, cleaned, hold neither.
		end := strings.IndexAny(inner, "\x07\x1b")
		if end < 0 {
			t.Fatalf("Link(%q, %q, %q) = %q: the opening sequence does not end", rawURL, text, param, out)
		}
		head, body := inner[:end], inner[end+1:]
		if inner[end] == '\x1b' {
			body = strings.TrimPrefix(body, "\\")
		}
		if sanitize.HasControl(head) || sanitize.HasControl(body) {
			t.Fatalf("Link(%q, %q, %q) = %q: %q or %q holds what Line drops", rawURL, text, param, out, head, body)
		}
	})
}
