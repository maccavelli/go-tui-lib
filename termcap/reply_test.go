package termcap

import (
	"image/color"
	"slices"
	"strings"
	"testing"
)

// rgb64 is a 16-bit colour.
func rgb64(r, g, b uint16) color.Color { return color.RGBA64{R: r, G: g, B: b, A: 0xffff} }

// sameColor reports whether a and b are the same colour, or both nil.
func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// sameReply reports whether got is want, Raw aside: Params compare by
// value, and colours by their channels.
func sameReply(got, want Reply) bool {
	return got.Kind == want.Kind && slices.Equal(got.Params, want.Params) && got.Mode == want.Mode &&
		got.Setting == want.Setting && got.OSC == want.OSC && got.Index == want.Index &&
		sameColor(got.Color, want.Color) && got.Flags == want.Flags && got.Dark == want.Dark && got.Text == want.Text
}

// notifyReply is a reply to the OSC 99 query padded to n bytes.
func notifyReply(n int) string {
	head, tail := "\x1b]99;i=termcap:p=?;a=", "\x1b\\"
	return head + strings.Repeat("x", n-len(head)-len(tail)) + tail
}

func TestParseReply(t *testing.T) {
	bg := rgb64(0x1e1e, 0x1e1e, 0x2e2e)
	da1 := Reply{Kind: ReplyPrimaryAttributes, Params: []int{62, 22}}
	version := Reply{Kind: ReplyVersion, Text: "kitty(0.39.1)"}
	color11 := Reply{Kind: ReplyColor, OSC: 11, Color: bg}
	graphics := Reply{Kind: ReplyKittyGraphics, Text: "OK"}
	cases := []struct {
		name string
		raw  string
		want Reply
	}{
		{"DA1", "\x1b[?62;22c", da1},
		{"DA1, 8-bit CSI", "\x9b?62;22c", da1},
		{"DA1 with no parameters", "\x1b[?c", Reply{Kind: ReplyPrimaryAttributes, Params: []int{}}},
		{"DA2", "\x1b[>1;4000;39c", Reply{Kind: ReplySecondaryAttributes, Params: []int{1, 4000, 39}}},
		{"XTVERSION, ST", "\x1bP>|kitty(0.39.1)\x1b\\", version},
		{"XTVERSION, 8-bit ST", "\x1bP>|kitty(0.39.1)\x9c", version},
		{"XTVERSION, BEL", "\x1bP>|kitty(0.39.1)\a", version},
		{"XTVERSION, bare ESC", "\x1bP>|kitty(0.39.1)\x1b", version},
		{"XTVERSION, 8-bit DCS", "\x90>|kitty(0.39.1)\x1b\\", version},
		{"DECRPM Pm 0", "\x1b[?2026;0$y", Reply{Kind: ReplyMode, Mode: 2026, Setting: ModeNotRecognized}},
		{"DECRPM Pm 1", "\x1b[?2027;1$y", Reply{Kind: ReplyMode, Mode: 2027, Setting: ModeSet}},
		{"DECRPM Pm 2", "\x1b[?2031;2$y", Reply{Kind: ReplyMode, Mode: 2031, Setting: ModeReset}},
		{"DECRPM Pm 3", "\x1b[?2048;3$y", Reply{Kind: ReplyMode, Mode: 2048, Setting: ModePermanentlySet}},
		{"DECRPM Pm 4", "\x1b[?1004;4$y", Reply{Kind: ReplyMode, Mode: 1004, Setting: ModePermanentlyReset}},
		{"DECRPM Pm 5", "\x1b[?1004;5$y", Reply{}},
		{"OSC 11, BEL", "\x1b]11;rgb:1e1e/1e1e/2e2e\a", color11},
		{"OSC 11, ST", "\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\", color11},
		{"OSC 11, 8-bit ST", "\x1b]11;rgb:1e1e/1e1e/2e2e\x9c", color11},
		{"OSC 11, bare ESC", "\x1b]11;rgb:1e1e/1e1e/2e2e\x1b", color11},
		{"OSC 11, 8-bit OSC", "\x9d11;rgb:1e1e/1e1e/2e2e\a", color11},
		{"OSC 10", "\x1b]10;#cdd6f4\a", Reply{Kind: ReplyColor, OSC: 10, Color: rgb64(0xcd00, 0xd600, 0xf400)}},
		{"OSC 4, index 0", "\x1b]4;0;rgb:ffff/0000/8080\x1b\\", Reply{Kind: ReplyColor, OSC: 4, Index: 0, Color: rgb64(0xffff, 0, 0x8080)}},
		{"OSC 4, index 15", "\x1b]4;15;rgb:10/20/30\a", Reply{Kind: ReplyColor, OSC: 4, Index: 15, Color: rgb64(0x1010, 0x2020, 0x3030)}},
		{"an unreadable colour", "\x1b]11;rgb:zz/00/00\a", Reply{}},
		{"OSC 11's query", "\x1b]11;?\a", Reply{}},
		{"Kitty keyboard flags", "\x1b[?31u", Reply{Kind: ReplyKittyKeyboard, Flags: 31}},
		{"DSR 997, dark", "\x1b[?997;1n", Reply{Kind: ReplyColorScheme, Dark: true}},
		{"DSR 997, light", "\x1b[?997;2n", Reply{Kind: ReplyColorScheme}},
		{"DSR 997, neither", "\x1b[?997;3n", Reply{}},
		{"Kitty graphics, ST", "\x1b_Gi=31;OK\x1b\\", graphics},
		{"Kitty graphics, 8-bit APC and ST", "\x9fGi=31;OK\x9c", graphics},
		{"Kitty graphics, an error", "\x1b_Gi=31;EINVAL:bad format\x1b\\", Reply{Kind: ReplyKittyGraphics, Text: "EINVAL:bad format"}},
		{"OSC 99", "\x1b]99;i=termcap:p=?;a=focus\x1b\\", Reply{Kind: ReplyNotify, Text: "a=focus"}},
		{"a reply of 1 KiB", notifyReply(maxReply), Reply{Kind: ReplyNotify, Text: notifyReply(maxReply)[len("\x1b]99;i=termcap:p=?;") : maxReply-2]}},
		{"a reply of 1 KiB + 1", notifyReply(maxReply + 1), Reply{}},
		{"two sequences, read as the first", "\x1b[?62;22c\x1b[?1u", da1},
		{"33 parameters", "\x1b[?" + strings.Repeat("1;", 32) + "1c", Reply{}},
		{"32 parameters", "\x1b[?" + strings.Repeat("1;", 31) + "1c", Reply{Kind: ReplyPrimaryAttributes, Params: slices.Repeat([]int{1}, 32)}},
		{"a sequence it does not know", "\x1b[0n", Reply{}},
		{"cancelled by CAN", "\x1b]11;rgb:0/0/0\x18", Reply{}},
		{"cut by an ESC before more", "\x1b]11;rgb:0/0/0\x1bX", Reply{}},
		{"unterminated", "\x1b]11;rgb:0/0/0", Reply{}},
		{"text", "x", Reply{}},
		{"nothing", "", Reply{}},
	}
	for _, c := range cases {
		got := ParseReply(c.raw)
		if got.Raw != c.raw {
			t.Errorf("%s: Raw = %q, want the input %q", c.name, got.Raw, c.raw)
		}
		if !sameReply(got, c.want) {
			t.Errorf("%s: ParseReply(%q) = %+v, want %+v", c.name, c.raw, got, c.want)
		}
	}
}

func TestReplyKindsAndSettings(t *testing.T) {
	if s := strings.Join([]string{ReplyUnknown.String(), ReplyNotify.String(), ModeNotRecognized.String(), ModePermanentlyReset.String()}, ","); s != "unknown,notify,not-recognized,permanently-reset" {
		t.Errorf("names %s", s)
	}
	for s, want := range map[ModeSetting]bool{ModeNotRecognized: false, ModeSet: true, ModeReset: true, ModePermanentlySet: true, ModePermanentlyReset: false} {
		if s.supported() != want {
			t.Errorf("%v supported = %v, want %v", s, s.supported(), want)
		}
	}
}
