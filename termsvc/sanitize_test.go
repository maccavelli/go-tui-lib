package termsvc

import (
	"strconv"
	"testing"
	"time"
)

// termsvc's untrusted strings pass through sanitize
// (docs/decisions/0014-PLAN-hardening.md Step 2).

func TestActivityVendorSanitized(t *testing.T) {
	now := time.Now()
	r, err := ParseActivity("ev\x1b[31m\u202eil\u200b;activity;1;busy;"+strconv.FormatInt(now.UnixMilli(), 10), now)
	if err != nil || r.Vendor != "evil" {
		t.Errorf("ParseActivity vendor %q, %v; want \"evil\"", r.Vendor, err)
	}
	for _, c := range []rune{0x200b, 0x2028, 0xfeff, 0x2060, 0xe0041, 0x00ad, 0x202e} {
		got := SanitizeTitle("a" + string(c) + "b")
		want := "ab"
		if c == 0x2028 {
			want = "a b"
		}
		if got != want {
			t.Errorf("SanitizeTitle with U+%04X = %q, want %q", c, got, want)
		}
	}
	if !(LinkPolicy{}).Openable("https://example.com/a") || (LinkPolicy{}).Openable("https://example.com/\u202ea") {
		t.Error("Openable does not refuse a bidirectional control")
	}
}
