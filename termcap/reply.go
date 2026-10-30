package termcap

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
)

// Reply is one terminal reply. ParseReply always sets Raw; a host that has
// only a decoded form, as Bubble Tea gives, sets the fields of its Kind and
// may leave Raw empty. Only the fields its Kind names are set. Fields are
// added, never changed, so build a Reply with a keyed literal.
type Reply struct {
	Kind    ReplyKind
	Raw     string      // the reply's bytes, as ParseReply read them
	Params  []int       // ReplyPrimaryAttributes, ReplySecondaryAttributes
	Mode    int         // ReplyMode: the mode DECRPM reports
	Setting ModeSetting // ReplyMode
	OSC     int         // ReplyColor: 4, 10 or 11
	Index   int         // ReplyColor with OSC 4: the palette index
	Color   color.Color // ReplyColor
	Flags   int         // ReplyKittyKeyboard: the flags the terminal holds
	Dark    bool        // ReplyColorScheme: DSR 997 said dark
	Text    string      // ReplyVersion's name; ReplyKittyGraphics's "OK" or error; ReplyNotify's keys
	Row     int         // ReplyCursorPosition: the cursor's row, from 1
	Col     int         // ReplyCursorPosition: the cursor's column, from 1
}

// ReplyKind is what a Reply answers. A host ignores a kind it does not
// know: a later version names more replies, and Raw still holds the bytes.
type ReplyKind uint8

const (
	// ReplyUnknown is a reply this version does not read, or one over
	// 1 KiB. Raw holds it. The zero value.
	ReplyUnknown ReplyKind = iota
	// ReplyPrimaryAttributes is DA1, CSI ? Ps ; … c, in Params.
	ReplyPrimaryAttributes
	// ReplySecondaryAttributes is DA2, CSI > Pp ; Pv ; Pc c, in Params.
	ReplySecondaryAttributes
	// ReplyVersion is XTVERSION's DCS > | text ST, in Text.
	ReplyVersion
	// ReplyMode is DECRPM, CSI ? Ps ; Pm $ y, in Mode and Setting.
	ReplyMode
	// ReplyColor is OSC 10, 11 or 4 answering a colour query, in OSC, Index
	// and Color.
	ReplyColor
	// ReplyKittyKeyboard is CSI ? flags u, in Flags.
	ReplyKittyKeyboard
	// ReplyColorScheme is DSR 997, CSI ? 997 ; 1 n or ; 2 n, in Dark.
	ReplyColorScheme
	// ReplyKittyGraphics is the Kitty graphics reply, APC G … ; text ST,
	// with "OK" or the error in Text.
	ReplyKittyGraphics
	// ReplyNotify is OSC 99's reply to a p=? query, with its keys in Text.
	ReplyNotify
	// ReplyCursorPosition is the cursor position report, CSI row ; col R,
	// answering DSR 6 (CSI 6 n), or DECXCPR's CSI ? row ; col R, in Row and
	// Col, each from 1 (docs/decisions/0023-MADR-line-mode-components.md
	// D3). A decoder may read the report on the first row as a modified F3,
	// CSI 1 ; mod R; only a host that asked for the report knows which it
	// is.
	ReplyCursorPosition
)

var replyKindNames = enum.Names[ReplyKind]{
	Pkg: pkgName, Type: "ReplyKind", Unknown: ErrUnknownName,
	Tokens: []string{
		tokenUnknown, "primary-attributes", "secondary-attributes", "version", "mode", "color",
		tokenKittyKeyboard, tokenColorScheme, tokenKittyGraphics, "notify", "cursor-position",
	},
}

// String is the kind's name.
func (k ReplyKind) String() string { return replyKindNames.String(k) }

// MarshalText is the kind's name.
func (k ReplyKind) MarshalText() ([]byte, error) { return replyKindNames.Marshal(k) }

// UnmarshalText reads a name MarshalText wrote.
func (k *ReplyKind) UnmarshalText(b []byte) error { return replyKindNames.Unmarshal(b, k) }

// ModeSetting is DECRPM's Pm: what the terminal says of a mode.
type ModeSetting uint8

const (
	// ModeNotRecognized is Pm 0: the terminal does not know the mode. The
	// zero value.
	ModeNotRecognized ModeSetting = iota
	// ModeSet is Pm 1.
	ModeSet
	// ModeReset is Pm 2.
	ModeReset
	// ModePermanentlySet is Pm 3.
	ModePermanentlySet
	// ModePermanentlyReset is Pm 4.
	ModePermanentlyReset
)

var modeSettingNames = enum.Names[ModeSetting]{
	Pkg: pkgName, Type: "ModeSetting", Unknown: ErrUnknownName,
	Tokens: []string{"not-recognized", "set", "reset", "permanently-set", "permanently-reset"},
}

// String is the setting's name.
func (s ModeSetting) String() string { return modeSettingNames.String(s) }

// MarshalText is the setting's name.
func (s ModeSetting) MarshalText() ([]byte, error) { return modeSettingNames.Marshal(s) }

// UnmarshalText reads a name MarshalText wrote.
func (s *ModeSetting) UnmarshalText(b []byte) error { return modeSettingNames.Unmarshal(b, s) }

// supported reports whether a mode with setting s can be used: a mode the
// terminal does not recognise, or holds permanently off, cannot.
func (s ModeSetting) supported() bool { return s != ModeNotRecognized && s != ModePermanentlyReset }

// maxReply is the longest reply the probe parses.
const maxReply = 1024

// ParseReply reads one complete reply, as a decoder such as termkey's
// returns it. It never fails: a sequence it does not know, or one over
// 1 KiB, is ReplyUnknown. A string holding more than one sequence is read
// as its first, and Raw is always raw.
//
// It reads the 7-bit introducers and the 8-bit ones, 0x9B CSI, 0x9D OSC,
// 0x90 DCS and 0x9F APC, and ends a string at BEL, ESC \, 0x9C, or a bare
// ESC at the end, as old urxvt sends. A reply with sanitize.MaxSeparators
// parameter separators or more, on any path to them, is ReplyUnknown:
// x/ansi's parser panics past its parameter buffer
// (docs/decisions/0023-MADR-line-mode-components.md A10).
func ParseReply(raw string) Reply {
	r := Reply{Raw: raw}
	if len(raw) > maxReply || sanitize.Separators(raw) >= sanitize.MaxSeparators {
		return r
	}
	p := ansi.NewParser()
	p.SetParamsSize(sanitize.MaxSeparators + 1)
	p.SetDataSize(maxReply)
	seq, _, _, state := ansi.DecodeSequence(raw, 0, p)
	data := string(p.Data())
	if stringSeq(seq) {
		switch {
		case state == ansi.StringState && strings.HasSuffix(raw, "\a"):
			// BEL ends a DCS or APC as it ends an OSC; x/ansi reads it as
			// data, to the end of raw.
			data = strings.TrimSuffix(data, "\a")
		case state != ansi.NormalState:
			return r // unterminated
		case terminated(seq):
		case seq == raw[:len(raw)-1] && raw[len(raw)-1] == ansi.ESC:
			// urxvt's bare ESC: x/ansi ends the string before it.
		default:
			return r // cut short by CAN, SUB, or an ESC before more bytes
		}
	} else if state != ansi.NormalState {
		return r
	}
	switch {
	case ansi.HasCsiPrefix(seq):
		csiReply(&r, ansi.Cmd(p.Command()), p.Params())
	case ansi.HasOscPrefix(seq):
		oscReply(&r, data)
	case ansi.HasDcsPrefix(seq):
		if c := ansi.Cmd(p.Command()); c.Prefix() == '>' && c.Final() == '|' && c.Intermediate() == 0 {
			r.Kind, r.Text = ReplyVersion, data
		}
	case ansi.HasApcPrefix(seq):
		if text, ok := strings.CutPrefix(data, "G"); ok {
			_, payload, _ := strings.Cut(text, ";")
			r.Kind, r.Text = ReplyKittyGraphics, payload
		}
	}
	return r
}

// stringSeq reports whether seq is an OSC, DCS or APC, which the decoder
// reads as strings.
func stringSeq(seq string) bool {
	return ansi.HasOscPrefix(seq) || ansi.HasDcsPrefix(seq) || ansi.HasApcPrefix(seq)
}

// terminated reports whether the string seq ends with BEL, 0x9C or ESC \.
func terminated(seq string) bool {
	return strings.HasSuffix(seq, "\a") || strings.HasSuffix(seq, "\x9c") || strings.HasSuffix(seq, "\x1b\\")
}

// csiReply reads the CSI replies: DA1, DA2, DECRPM, the Kitty keyboard
// flags, DSR 997 and the cursor position report.
func csiReply(r *Reply, c ansi.Cmd, ps ansi.Params) {
	prefix, inter, final := c.Prefix(), c.Intermediate(), c.Final()
	values := make([]int, len(ps))
	for i, p := range ps {
		values[i] = p.Param(0)
	}
	switch {
	case prefix == '?' && inter == 0 && final == 'c':
		r.Kind, r.Params = ReplyPrimaryAttributes, values
	case prefix == '>' && inter == 0 && final == 'c':
		r.Kind, r.Params = ReplySecondaryAttributes, values
	case prefix == '?' && inter == '$' && final == 'y' && len(values) == 2:
		if values[1] >= 0 && values[1] <= int(ModePermanentlyReset) {
			r.Kind, r.Mode, r.Setting = ReplyMode, values[0], ModeSetting(values[1]) //nolint:gosec // 0 to 4, checked above
		}
	case prefix == '?' && inter == 0 && final == 'u' && len(values) <= 1:
		r.Kind = ReplyKittyKeyboard
		if len(values) == 1 {
			r.Flags = values[0]
		}
	case (prefix == 0 || prefix == '?') && inter == 0 && final == 'R' && len(values) == 2 && values[0] >= 1 && values[1] >= 1:
		r.Kind, r.Row, r.Col = ReplyCursorPosition, values[0], values[1]
	case prefix == '?' && inter == 0 && final == 'n' && len(values) == 2 && values[0] == 997:
		switch values[1] {
		case 1:
			r.Kind, r.Dark = ReplyColorScheme, true
		case 2:
			r.Kind = ReplyColorScheme
		}
	}
}

// oscReply reads the OSC replies: a colour, OSC 10, 11 or 4, and OSC 99's
// keys. data is the OSC's text, without its introducer or terminator.
func oscReply(r *Reply, data string) {
	num, rest, ok := strings.Cut(data, ";")
	if !ok {
		return
	}
	switch num {
	case "10", "11":
		if c, ok := xParseColor(rest); ok {
			r.Kind, r.OSC, r.Color = ReplyColor, 10, c
			if num == "11" {
				r.OSC = 11
			}
		}
	case "4":
		idx, spec, ok := strings.Cut(rest, ";")
		i, err := strconv.Atoi(idx)
		if !ok || err != nil || i < 0 {
			return
		}
		if c, ok := xParseColor(spec); ok {
			r.Kind, r.OSC, r.Index, r.Color = ReplyColor, 4, i, c
		}
	case "99":
		if _, keys, ok := strings.Cut(rest, ";"); ok {
			r.Kind, r.Text = ReplyNotify, keys
		}
	}
}

// oscData is the text of the OSC raw holds, without its introducer or
// terminator, or "" and false when raw is not one.
func oscData(raw string) (string, bool) {
	p := ansi.NewParser()
	p.SetDataSize(maxReply)
	seq, _, _, state := ansi.DecodeSequence(raw, 0, p)
	if state != ansi.NormalState || !ansi.HasOscPrefix(seq) {
		return "", false
	}
	return string(p.Data()), true
}
