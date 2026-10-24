package termcaptea

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/internal/termevent/termeventtest"
	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// The Prober's fuzz target, moved from termcap with it
// (docs/decisions/0014-PLAN-hardening.md Step 9, finding H9). The seeds
// are the existing tests' inputs and the replies termcaptest's profiles
// give (termcaptest.go, answerCSI and answerString).

// replyKinds is how many kinds of reply replyMsg builds.
const replyKinds = 10

// replyMsg is the reply of kind k, carrying s.
func replyMsg(k byte, s string) tea.Msg {
	switch k % replyKinds {
	case 0:
		return termeventtest.UnknownOsc(s)
	case 1:
		return termeventtest.UnknownCsi(s)
	case 2:
		return termeventtest.UnknownDcs(s)
	case 3:
		return termeventtest.UnknownApc(s)
	case 4:
		return termeventtest.Unknown(s)
	case 5:
		return tea.TerminalVersionMsg{Name: s}
	case 6:
		return termeventtest.DeviceAttributes(attrs(s)...)
	case 7:
		return termeventtest.SecondaryDeviceAttributes(attrs(s)...)
	case 8:
		return termeventtest.KittyGraphics(s)
	}
	return termeventtest.DarkColorScheme()
}

// attrs is s's bytes as device attributes.
func attrs(s string) []int {
	out := make([]int, len(s))
	for i := range len(s) {
		out[i] = int(s[i])
	}
	return out
}

// FuzzProberReplies: a prober that has seen its environment takes any
// replies without a panic; a query is never given a raw reply over
// maxReply; and Caps.Terminal holds nothing Line would change. The replies
// are payload's parts, split at NUL, each of the kind kinds names in turn.
func FuzzProberReplies(f *testing.F) {
	for _, c := range []struct {
		kinds   []byte
		payload string
	}{
		{[]byte{0}, "\x1b]99;i=termcap:p=?;a=focus,report:o=always,unfocused,invisible:u=0,1,2\x1b\\"},
		{[]byte{0}, osc99Reply(1025)},
		{[]byte{0}, osc99Reply(1024)},
		{[]byte{5}, "kitty(0.39.1)"},
		{[]byte{5}, "tmux 3.4"},
		{[]byte{5}, "evil\x1b]0;title\x07\u202e"},
		{[]byte{6}, "\x01\x02"},
		{[]byte{7, 6}, "\x01\x5f\x00\x00\x01\x02"},
		{[]byte{8}, "OK"},
		{[]byte{1}, "\x1b[?997;1n"},
		{[]byte{1}, "\x1b[?2026;2$y"},
		{[]byte{1}, "\x1b[?" + strings.Repeat("1", 1030) + "n"},
		{[]byte{2}, "\x1bP>|kitty(0.39.1)\x1b\\"},
		{[]byte{0, 9, 6}, "\x1b]11;rgb:0000/0000/0000\x1b\\\x00\x00>\x16"},
		{[]byte{4}, "\x1b]4;1;"},
	} {
		f.Add(c.kinds, c.payload)
	}
	f.Fuzz(func(t *testing.T, kinds []byte, payload string) {
		over := 0
		q := termcap.Query{Name: "fuzz", Seq: fixed("\x1b[?5n"), Parse: func(r termcap.Reply, _ *termcap.Caps) bool {
			if len(r.Raw) > maxReply {
				over = len(r.Raw)
			}
			return false
		}}
		p, _ := start(local, termcap.WithQuery(q))
		for i, part := range strings.Split(payload, "\x00") {
			k := byte(0)
			if len(kinds) > 0 {
				k = kinds[i%len(kinds)]
			}
			feed(p, replyMsg(k, part))
		}
		if over > 0 {
			t.Fatalf("a query was given a raw reply of %d bytes, over %d", over, maxReply)
		}
		if v := p.Caps().Terminal.Value; sanitize.Line(v) != v {
			t.Fatalf("Caps.Terminal is %q, which Line changes", v)
		}
	})
}
