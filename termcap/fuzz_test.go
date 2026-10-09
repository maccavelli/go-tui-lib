package termcap

import (
	"encoding"
	"reflect"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/internal/termevent/termeventtest"
)

// Fuzz targets (docs/decisions/0014-PLAN-hardening.md Step 9, finding H9).
// The seeds are the existing tests' inputs and the replies termcaptest's
// profiles give (termcaptest.go, answerCSI and answerString), written out
// here: termcaptest imports this package.

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
		q := Query{Name: "fuzz", Seq: fixed("\x1b[?5n"), Parse: func(r Reply, _ *Caps) bool {
			if len(r.Raw) > maxReply {
				over = len(r.Raw)
			}
			return false
		}}
		p, _ := start(local, WithQuery(q))
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

// FuzzParseTmux: ParseTmux never panics; Known means five tab-separated
// fields and a version; and no list holds an empty entry.
func FuzzParseTmux(f *testing.F) {
	for _, s := range []string{
		"3.4\tcsi-u\t1\tRGB,extkeys,focus,mouse\tattached,focused,UTF-8\n",
		"3.3a\txterm\toff\t\t\r\n",
		"3.4\tcsi-u\t1\tRGB",
		"\tcsi-u\t1\t\t",
		"3.4\tcsi-u\ton\t, ,a,,b\tcontrol-mode\t\n",
		"",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, out string) {
		got := ParseTmux(out)
		if got.Known {
			fields := strings.Split(strings.TrimRight(out, "\r\n"), "\t")
			if len(fields) != 5 || got.Version == "" {
				t.Fatalf("%q is Known with %d fields and version %q", out, len(fields), got.Version)
			}
		}
		for _, l := range [][]string{got.TermFeatures, got.ClientFlags} {
			for _, e := range l {
				if e == "" || strings.TrimSpace(e) != e {
					t.Fatalf("%q gives the list entry %q", out, e)
				}
			}
		}
	})
}

// textEnum is a termcap enum, through its pointer.
type textEnum interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}

// FuzzEnumText: for each of termcap's enums, a name UnmarshalText takes is
// the name MarshalText gives back, and reads back to the same value.
func FuzzEnumText(f *testing.F) {
	for _, s := range []string{"supported", "unknown", "queried", "tmux", "kitty", "vscode", "wsl", "issue", "", "Supported", "7"} {
		f.Add([]byte(s))
	}
	fresh := []func() textEnum{
		func() textEnum { return new(Support) }, func() textEnum { return new(Origin) },
		func() textEnum { return new(Mux) }, func() textEnum { return new(Brand) },
		func() textEnum { return new(Editor) }, func() textEnum { return new(Platform) },
		func() textEnum { return new(Disposition) },
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		for _, mk := range fresh {
			v := mk()
			if v.UnmarshalText(b) != nil {
				continue
			}
			out, err := v.MarshalText()
			if err != nil || string(out) != string(b) {
				t.Fatalf("%T: %q reads, and writes back as %q, %v", v, b, out, err)
			}
			back := mk()
			if err := back.UnmarshalText(out); err != nil || !reflect.DeepEqual(back, v) {
				t.Fatalf("%T: %q reads back as %v, %v; want %v", v, out, back, err, v)
			}
		}
	})
}

// FuzzColorFGBG: when colorFGBG reads a value, its last field is a colour
// from 0 to 15, and dark follows Vim's rule: 0-6 and 8 are dark. When it
// does not, dark is false.
func FuzzColorFGBG(f *testing.F) {
	for _, s := range []string{"15;0", "0;15", "7;default", "0;7", "12;8", "1;2;3", "default;default", "", ";", "0;16", "0;-1", "0;+7"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, v string) {
		dark, ok := colorFGBG(v)
		if !ok {
			if dark {
				t.Fatalf("%q: not read, but dark", v)
			}
			return
		}
		n, err := strconv.Atoi(v[strings.LastIndexByte(v, ';')+1:])
		if err != nil || n < 0 || n > 15 {
			t.Fatalf("%q: read, but its background is %d, %v", v, n, err)
		}
		if want := n <= 6 || n == 8; dark != want {
			t.Fatalf("%q: dark %v, want %v for colour %d", v, dark, want, n)
		}
	})
}
