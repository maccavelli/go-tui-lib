package termkey_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// notifyReply is termcap's test helper of that name: an OSC 99 reply n
// bytes long.
func notifyReply(n int) string {
	head, tail := "\x1b]99;i=termcap:p=?;a=", "\x1b\\"
	return head + strings.Repeat("x", n-len(head)-len(tail)) + tail
}

// TestRepliesReachParseReply holds MADR A2.4: a terminal's reply, read by
// the decoder whole or split at any byte, is one InputUnknown whose Raw
// termcap.ParseReply reads as it reads the reply. The rows are the raw
// strings of termcap's TestParseReply. Its "text" and "nothing" rows are
// left out: "x" is a key and "" no input, and neither is a reply.
func TestRepliesReachParseReply(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		n    int // the inputs the decoder gives, the reply first; 0 means 1
		max  int // the decoder's MaxSequence; 0 is its default
	}{
		{name: "DA1", raw: "\x1b[?62;22c"},
		{name: "DA1, 8-bit CSI", raw: "\x9b?62;22c"},
		{name: "DA1 with no parameters", raw: "\x1b[?c"},
		{name: "DA2", raw: "\x1b[>1;4000;39c"},
		{name: "XTVERSION, ST", raw: "\x1bP>|kitty(0.39.1)\x1b\\"},
		{name: "XTVERSION, 8-bit ST", raw: "\x1bP>|kitty(0.39.1)\x9c"},
		{name: "XTVERSION, BEL", raw: "\x1bP>|kitty(0.39.1)\a"},
		{name: "XTVERSION, bare ESC", raw: "\x1bP>|kitty(0.39.1)\x1b"},
		{name: "XTVERSION, 8-bit DCS", raw: "\x90>|kitty(0.39.1)\x1b\\"},
		{name: "DECRPM Pm 0", raw: "\x1b[?2026;0$y"},
		{name: "DECRPM Pm 1", raw: "\x1b[?2027;1$y"},
		{name: "DECRPM Pm 2", raw: "\x1b[?2031;2$y"},
		{name: "DECRPM Pm 3", raw: "\x1b[?2048;3$y"},
		{name: "DECRPM Pm 4", raw: "\x1b[?1004;4$y"},
		{name: "DECRPM Pm 5", raw: "\x1b[?1004;5$y"},
		{name: "OSC 11, BEL", raw: "\x1b]11;rgb:1e1e/1e1e/2e2e\a"},
		{name: "OSC 11, ST", raw: "\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\"},
		{name: "OSC 11, 8-bit ST", raw: "\x1b]11;rgb:1e1e/1e1e/2e2e\x9c"},
		{name: "OSC 11, bare ESC", raw: "\x1b]11;rgb:1e1e/1e1e/2e2e\x1b"},
		{name: "OSC 11, 8-bit OSC", raw: "\x9d11;rgb:1e1e/1e1e/2e2e\a"},
		{name: "OSC 10", raw: "\x1b]10;#cdd6f4\a"},
		{name: "OSC 4, index 0", raw: "\x1b]4;0;rgb:ffff/0000/8080\x1b\\"},
		{name: "OSC 4, index 15", raw: "\x1b]4;15;rgb:10/20/30\a"},
		{name: "an unreadable colour", raw: "\x1b]11;rgb:zz/00/00\a"},
		{name: "OSC 11's query", raw: "\x1b]11;?\a"},
		{name: "Kitty keyboard flags", raw: "\x1b[?31u"},
		{name: "DSR 997, dark", raw: "\x1b[?997;1n"},
		{name: "DSR 997, light", raw: "\x1b[?997;2n"},
		{name: "DSR 997, neither", raw: "\x1b[?997;3n"},
		{name: "Kitty graphics, ST", raw: "\x1b_Gi=31;OK\x1b\\"},
		{name: "Kitty graphics, 8-bit APC and ST", raw: "\x9fGi=31;OK\x9c"},
		{name: "Kitty graphics, an error", raw: "\x1b_Gi=31;EINVAL:bad format\x1b\\"},
		{name: "OSC 99", raw: "\x1b]99;i=termcap:p=?;a=focus\x1b\\"},
		{name: "a reply of 1 KiB", raw: notifyReply(1024)},
		// Under the decoder's 4 KiB bound, the reply comes whole, and
		// ParseReply refuses it by its own 1 KiB bound.
		{name: "a reply of 1 KiB + 1", raw: notifyReply(1025)},
		// Cut by a decoder bound of 1 KiB on the ESC of its ESC \, the cut
		// leaves the ESC out, so ParseReply reads it as it reads the whole
		// reply: as nothing.
		{name: "a reply of 1 KiB + 1, cut by a 1 KiB bound", raw: notifyReply(1025), max: 1024},
		{name: "a reply of 4 KiB + 1, cut by the default bound", raw: notifyReply(4<<10 + 1)},
		// Two replies are two inputs; the first is the reply ParseReply reads.
		{name: "two sequences, read as the first", raw: "\x1b[?62;22c\x1b[?1u", n: 2},
		{name: "33 parameters", raw: "\x1b[?" + strings.Repeat("1;", 32) + "1c"},
		{name: "32 parameters", raw: "\x1b[?" + strings.Repeat("1;", 31) + "1c"},
		{name: "a sequence it does not know", raw: "\x1b[0n"},
		{name: "cancelled by CAN", raw: "\x1b]11;rgb:0/0/0\x18"},
		// The ESC that cuts the reply is left out of it, and begins
		// ESC X, alt+shift+x.
		{name: "cut by an ESC before more", raw: "\x1b]11;rgb:0/0/0\x1bX", n: 2},
		{name: "unterminated", raw: "\x1b]11;rgb:0/0/0"},
	}
	for _, c := range cases {
		want := termcap.ParseReply(c.raw)
		want.Raw = ""
		n := max(c.n, 1)
		for i := range len(c.raw) {
			d := termkey.Decoder{MaxSequence: c.max}
			got := d.Decode([]byte(c.raw[:i]), nil)
			got = d.Flush(d.Decode([]byte(c.raw[i:]), got))
			if len(got) != n || got[0].Kind != termkey.InputUnknown {
				t.Errorf("%s, split at %d: %d inputs, the first %v; want %d, the first unknown", c.name, i, len(got), got[0].Kind, n)
				continue
			}
			if n == 1 && !got[0].Truncated && string(got[0].Raw) != c.raw {
				t.Errorf("%s, split at %d: Raw = %q, want the reply %q", c.name, i, got[0].Raw, c.raw)
			}
			reply := termcap.ParseReply(string(got[0].Raw))
			reply.Raw = ""
			if !reflect.DeepEqual(reply, want) {
				t.Errorf("%s, split at %d: ParseReply(Raw) = %+v, want %+v", c.name, i, reply, want)
			}
		}
	}
}
