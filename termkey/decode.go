package termkey

import (
	"bytes"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// DefaultEscTimeout is how long a host waits, after the input goes quiet
// with Pending true, before it calls Flush: ultraviolet's 50 ms, so a
// line-mode CLI and a Bubble Tea program wait alike. tmux waits 10 ms and
// Vim 100 ms; the host chooses.
const DefaultEscTimeout = 50 * time.Millisecond

// The decoder's default bounds.
const (
	defaultMaxPaste    = 1 << 20
	defaultMaxSequence = 4 << 10
)

// Decoder reads a terminal's input bytes as keys, pastes and the sequences
// it does not read as keys. It is a state machine fed bytes, with no
// goroutine and no clock: it holds what it cannot decide yet, a lone ESC or
// a sequence cut by a read, and says so with Pending; the host calls Flush
// once the input has been quiet for a while, DefaultEscTimeout for one, and
// the decoder gives up what it holds. The zero value is ready to use. It
// is not safe for concurrent use.
//
// It reads UTF-8 text; C0 controls as ctrl and their letter; ESC before a
// key as alt and that key; the legacy CSI and SS3 keys, with xterm's
// modifier parameter; xterm's modifyOtherKeys; the Kitty keyboard
// protocol; and bracketed paste, as one input. Any other CSI, OSC, DCS, APC,
// PM or SOS is one InputUnknown with its bytes, so a terminal's reply never
// reads as keys. Windows' win32-input-mode is read as InputUnknown too.
type Decoder struct {
	// MaxPaste bounds a paste, in bytes: a longer one keeps its first
	// MaxPaste bytes and is Truncated. 0 means 1 MiB.
	MaxPaste int
	// MaxSequence bounds any other sequence, in bytes: a longer one is one
	// InputUnknown of its first MaxSequence bytes, Truncated, and the rest is
	// consumed up to its end. A string cut on an ESC leaves the ESC out, so
	// that no cut reply ends the way a whole one may, and termcap.ParseReply
	// never reads it as whole. 0 means 4 KiB, over ParseReply's 1 KiB.
	MaxSequence int

	buf       []byte // bytes not decoded yet: what a read cut, or a paste's possible end
	inPaste   bool   // between CSI 200 ~ and CSI 201 ~
	paste     []byte // the paste so far, at most MaxPaste bytes
	truncated bool   // the paste went over MaxPaste
	skip      byte   // the kind of an over-long sequence whose rest is consumed, or 0
	skipEsc   bool   // the skipped string's last byte was ESC
}

// The kinds of sequence skip consumes the rest of.
const (
	skipCSI    = 'c' // ends with a final byte, 0x40 to 0x7e
	skipSS3    = '3' // ends with the byte after its digits
	skipString = 's' // ends with BEL, ST, ESC \, CAN or SUB
)

// pasteEnd ends a bracketed paste.
const pasteEnd = "\x1b[201~"

// pasteStart is what token gives for CSI 200 ~, which begins a paste and
// is no input of its own.
var pasteStart = Input{Kind: InputPaste}

// The control bytes the decoder reads.
const (
	esc = 0x1b
	bel = 0x07
	can = 0x18
	sub = 0x1a
	st  = 0x9c
)

func (d *Decoder) maxPaste() int {
	if d.MaxPaste > 0 {
		return d.MaxPaste
	}
	return defaultMaxPaste
}

func (d *Decoder) maxSequence() int {
	if d.MaxSequence > 0 {
		return d.MaxSequence
	}
	return defaultMaxSequence
}

// Decode reads p, appends every input it completes to out, and returns
// out. A byte that ends no input yet is held, and Pending reports it; a
// later Decode or Flush reads on from it, so a stream split anywhere
// decodes as the whole.
func (d *Decoder) Decode(p []byte, out []Input) []Input {
	d.buf = append(d.buf, p...)
	return d.drain(out, false)
}

// Pending reports whether the decoder holds bytes it has not decided:
// a lone ESC, a sequence or a UTF-8 character a read cut, or a paste not
// yet ended. A host that sees it true calls Flush once the input has been
// quiet long enough.
func (d *Decoder) Pending() bool { return len(d.buf) > 0 || d.inPaste || d.skip != 0 }

// Flush gives up what the decoder holds, as the input going quiet says:
// a lone ESC as the escape key, ESC before a sequence's introducer as alt
// and that key, an incomplete sequence as InputUnknown with its bytes, and
// a paste not yet ended as the paste so far. It appends them to out, and
// returns out. Pending is false after it.
func (d *Decoder) Flush(out []Input) []Input {
	out = d.drain(out, true)
	if d.inPaste {
		out = append(out, d.endPaste())
	}
	d.buf, d.skip, d.skipEsc = nil, 0, false
	return out
}

// drain decodes buf until it holds only what it cannot decide; in final
// mode it decides everything.
func (d *Decoder) drain(out []Input, final bool) []Input {
	for len(d.buf) > 0 {
		n, more, ok := d.next(d.buf, out, final)
		if !ok {
			break
		}
		out = more
		d.buf = d.buf[n:]
	}
	if len(d.buf) == 0 {
		d.buf = nil
	} else {
		d.buf = append([]byte(nil), d.buf...) // let the read's array go
	}
	return out
}

// next decodes the start of b, appends what it makes to out, and returns
// the bytes it consumed; ok is false when b holds too little to decide and
// final is false.
func (d *Decoder) next(b []byte, out []Input, final bool) (n int, _ []Input, ok bool) {
	switch {
	case d.skip != 0:
		return d.skipRest(b), out, true
	case d.inPaste:
		return d.pasteBytes(b, out, final)
	}
	n, in, ok := d.token(b, final)
	switch {
	case !ok:
		return 0, out, false
	case in == &pasteStart:
		d.inPaste = true
	case in != nil:
		out = append(out, *in)
	}
	return n, out, true
}

// token decodes one input from the start of b, or gives pasteStart. Only
// tooLong changes the decoder's state.
func (d *Decoder) token(b []byte, final bool) (int, *Input, bool) {
	c := b[0]
	switch {
	case c == esc:
		return d.escape(b, final)
	case c == 0x9b: // CSI
		return d.csi(b, 1, final)
	case c == 0x8f: // SS3
		return d.ss3(b, 1, final)
	case c == 0x90 || c == 0x9d || c == 0x9f || c == 0x9e || c == 0x98: // DCS, OSC, APC, PM, SOS
		return d.str(b, 1, final)
	case c >= 0x80 && c <= 0x9f:
		// Another C1 control, which no UTF-8 character begins with, as
		// ctrl+alt and its 7-bit form's letter, as ultraviolet reads it.
		return 1, key(b[:1], KeyEvent{Code: rune(c) - 0x40, Mod: ModCtrl | ModAlt}), true
	case c < 0x20 || c == 0x7f || c == ' ':
		return 1, key(b[:1], control(c)), true
	case c < 0x7f:
		k := KeyEvent{Code: rune(c), Text: string(rune(c))}
		if unicode.IsUpper(k.Code) {
			k.Code, k.ShiftedCode, k.Mod = unicode.ToLower(k.Code), k.Code, ModShift
		}
		return 1, key(b[:1], k), true
	}
	if !utf8.FullRune(b) && !final {
		return 0, nil, false
	}
	r, size := utf8.DecodeRune(b)
	if r == utf8.RuneError && size == 1 {
		return 1, unknown(b[:1], false), true
	}
	return size, key(b[:size], KeyEvent{Code: r, Text: string(r)}), true
}

// escape decodes what an ESC begins: a CSI, SS3 or string sequence, or
// alt and the key after it, or the escape key alone.
func (d *Decoder) escape(b []byte, final bool) (int, *Input, bool) {
	if len(b) == 1 {
		if !final {
			return 0, nil, false
		}
		return 1, key(b, KeyEvent{Code: KeyEscape}), true
	}
	if introducer(b[1]) && len(b) == 2 {
		// ESC and an introducer: a sequence's start, or alt and the key,
		// as the input going quiet says.
		if !final {
			return 0, nil, false
		}
		return 2, key(b, altIntroducer(b[1])), true
	}
	switch b[1] {
	case '[':
		return d.csi(b, 2, final)
	case 'O':
		return d.ss3(b, 2, final)
	case 'P', ']', '_', '^', 'X':
		return d.str(b, 2, final)
	}
	n, in, ok := d.token(b[1:], final)
	if !ok {
		return 0, nil, false
	}
	if in != nil && in.Kind == InputKey {
		in.Key.Mod |= ModAlt
		in.Key.Text = ""
		in.Raw = append([]byte(nil), b[:n+1]...)
		return n + 1, in, true
	}
	// Not a key after the ESC: the escape key alone, and what follows is
	// decoded again on its own, so a skip it began is undone.
	d.skip, d.skipEsc = 0, false
	return 1, key(b[:1], KeyEvent{Code: KeyEscape}), true
}

// introducer reports whether c, after ESC, begins a sequence.
func introducer(c byte) bool { return strings.IndexByte("[OP]_^X", c) >= 0 }

// altIntroducer is ESC and an introducer read as alt and that key, as
// ultraviolet reads it: shift as well for O, P and X, which are upper case.
func altIntroducer(c byte) KeyEvent {
	switch c {
	case 'O', 'P', 'X':
		return KeyEvent{Code: unicode.ToLower(rune(c)), Mod: ModShift | ModAlt}
	}
	return KeyEvent{Code: rune(c), Mod: ModAlt}
}

// control is a C0 control, DEL or space as the key a terminal sends it for.
func control(c byte) KeyEvent {
	switch c {
	case 0x00:
		return KeyEvent{Code: KeySpace, Mod: ModCtrl}
	case 0x08:
		return KeyEvent{Code: 'h', Mod: ModCtrl}
	case 0x09:
		return KeyEvent{Code: KeyTab}
	case 0x0d:
		return KeyEvent{Code: KeyEnter}
	case esc:
		return KeyEvent{Code: KeyEscape}
	case 0x7f:
		return KeyEvent{Code: KeyBackspace}
	case ' ':
		return KeyEvent{Code: KeySpace, Text: " "}
	case 0x1c, 0x1d, 0x1e, 0x1f:
		return KeyEvent{Code: rune(c) + 0x40, Mod: ModCtrl}
	}
	return KeyEvent{Code: rune(c) + 0x60, Mod: ModCtrl} // 0x01 to 0x1a: ctrl and a letter
}

// key is k as an input, from raw.
func key(raw []byte, k KeyEvent) *Input {
	return &Input{Kind: InputKey, Key: k, Raw: append([]byte(nil), raw...)}
}

// unknown is raw as an input the decoder does not read as a key.
func unknown(raw []byte, truncated bool) *Input {
	return &Input{Kind: InputUnknown, Raw: append([]byte(nil), raw...), Truncated: truncated}
}

// tooLong gives the first MaxSequence bytes of an over-long sequence of
// kind, and consumes the rest of b into the skip that follows.
func (d *Decoder) tooLong(b []byte, kind byte) (int, *Input, bool) {
	limit := d.maxSequence()
	d.skip, d.skipEsc = kind, b[limit-1] == esc
	if d.skipEsc {
		return limit, unknown(b[:limit-1], true), true // an ESC that may begin ESC \
	}
	return limit, unknown(b[:limit], true), true
}

// skipRest consumes b up to the end of the sequence skip is in, and
// returns how much it consumed.
func (d *Decoder) skipRest(b []byte) int {
	for i, c := range b {
		switch d.skip {
		case skipCSI:
			if c >= 0x40 && c <= 0x7e {
				d.skip = 0
				return i + 1
			}
		case skipSS3:
			if c < '0' || c > '9' {
				d.skip = 0
				if c >= 0x21 && c <= 0x7e {
					return i + 1
				}
				return i
			}
		case skipString:
			if c == bel || c == st || c == can || c == sub || d.skipEsc && c == '\\' {
				d.skip, d.skipEsc = 0, false
				return i + 1
			}
			d.skipEsc = c == esc
		}
	}
	return len(b)
}

// param is one CSI parameter: its value, -1 when it is missing, and
// whether a ':' sub-parameter follows it.
type param struct {
	v    int
	more bool
}

// get is p's value, or def when it is missing.
func (p param) get(def int) int {
	if p.v < 0 {
		return def
	}
	return p.v
}

// maxParams is the most CSI parameters the decoder reads; a key has
// at most a few.
const maxParams = 32

// at is ps[i]'s value, or def when it is missing or beyond ps.
func at(ps []param, i, def int) int {
	if i >= len(ps) {
		return def
	}
	return ps[i].get(def)
}

// scanParams reads the CSI parameter bytes from b[i], within limit bytes
// of b: the parameters, at most maxParams of them; where they end; whether
// there were more; and whether they ran to limit, which cuts the sequence.
func scanParams(b []byte, i, limit int) (ps []param, end int, overflow, cut bool) {
	ps = []param{}
	cur, seen := param{v: -1}, false
	for ; i < len(b) && b[i] >= 0x30 && b[i] <= 0x3f; i++ {
		if i >= limit {
			return nil, i, false, true
		}
		seen = true
		switch c := b[i]; {
		case c >= '0' && c <= '9':
			cur.v = max(cur.v, 0)
			if cur.v < 1<<24 {
				cur.v = cur.v*10 + int(c-'0')
			}
		case c == ';' || c == ':':
			cur.more = c == ':'
			ps, overflow = addParam(ps, cur, overflow)
			cur = param{v: -1}
		}
	}
	if seen {
		ps, overflow = addParam(ps, cur, overflow)
	}
	return ps, i, overflow, false
}

// addParam adds p to ps, or reports overflow when ps is full.
func addParam(ps []param, p param, overflow bool) ([]param, bool) {
	if len(ps) < maxParams {
		return append(ps, p), overflow
	}
	return ps, true
}

// csi decodes a CSI whose parameters start at b[i].
func (d *Decoder) csi(b []byte, i int, final bool) (int, *Input, bool) {
	limit := d.maxSequence()
	var prefix, inter byte
	if i < len(b) && b[i] >= '<' && b[i] <= '?' {
		prefix = b[i]
		i++
	}
	ps, i, overflow, cut := scanParams(b, i, limit)
	if cut {
		return d.tooLong(b, skipCSI)
	}
	inters := 0
	for ; i < len(b) && b[i] >= 0x20 && b[i] <= 0x2f; i++ {
		if i >= limit {
			return d.tooLong(b, skipCSI)
		}
		inter = b[i]
		inters++
	}
	urxvt := prefix == 0 && inters == 1 && inter == '$' && !overflow
	if i >= len(b) {
		// b is at most limit bytes here, or the scan would have cut it; the
		// next byte decides, as it does for the stream read whole.
		switch {
		case !final:
			return 0, nil, false
		case urxvt:
			return urxvtShift(b, ps)
		}
		return len(b), unknown(b, false), true
	}
	fin := b[i]
	if fin < 0x40 || fin > 0x7e {
		if urxvt {
			return urxvtShift(b[:i], ps)
		}
		return i, unknown(b[:i], false), true // cut by a byte no CSI holds
	}
	if i >= limit {
		return d.tooLong(b, skipCSI)
	}
	i++
	if fin == 'M' && prefix == 0 && inter == 0 && len(ps) == 0 {
		// An X10 mouse report: three bytes follow.
		if i+3 > len(b) {
			if !final {
				return 0, nil, false
			}
			return len(b), unknown(b, false), true
		}
		return i + 3, unknown(b[:i+3], false), true
	}
	if prefix != 0 || inter != 0 || overflow {
		return i, unknown(b[:i], false), true
	}
	if fin == '~' && at(ps, 0, 0) == 200 {
		return i, &pasteStart, true
	}
	if k, ok := csiKey(fin, ps); ok {
		return i, key(b[:i], k), true
	}
	return i, unknown(b[:i], false), true
}

// urxvtShift reads urxvt's CSI n $, all of b, as shift and the key CSI n ~
// names, as ultraviolet does; a CSI n $ that names no key is unknown.
func urxvtShift(b []byte, ps []param) (int, *Input, bool) {
	if k, ok := csiKey('~', ps); ok {
		k.Mod |= ModShift
		return len(b), key(b, k), true
	}
	return len(b), unknown(b, false), true
}

// csiKey reads a CSI with final byte fin and parameters ps as a key, as
// ultraviolet's parseCsi does, or reports false.
func csiKey(fin byte, ps []param) (KeyEvent, bool) {
	switch fin {
	case 'R':
		if col := at(ps, 1, 1); len(ps) == 2 && ps[0].get(1) == 1 && col >= 1 && col-1 <= int(ModMeta|ModShift|ModAlt|ModCtrl) {
			// CSI 1 ; m R is a modified F3, and a cursor position at row 1
			// too, which ultraviolet reports both of; the key is ours.
			return KeyEvent{Code: KeyF3, Mod: legacyMod(col)}, true
		}
		if len(ps) != 0 {
			return KeyEvent{}, false
		}
		return csiLetter(fin, ps)
	case 'a', 'b', 'c', 'd', 'A', 'B', 'C', 'D', 'E', 'F', 'H', 'P', 'Q', 'S', 'Z':
		return csiLetter(fin, ps)
	case 'u':
		if len(ps) == 0 {
			return KeyEvent{}, false
		}
		return kitty(ps), true
	case '@', '^', '~':
		return csiTilde(fin, ps)
	}
	return KeyEvent{}, false
}

// csiLetter is a key a CSI's final letter names: CSI 1 ; m X with xterm's
// modifier parameter, and the Kitty event type after it.
func csiLetter(fin byte, ps []param) (KeyEvent, bool) {
	var k KeyEvent
	switch fin {
	case 'a', 'b', 'c', 'd':
		k = KeyEvent{Code: KeyUp + rune(fin-'a'), Mod: ModShift}
	case 'A', 'B', 'C', 'D':
		k = KeyEvent{Code: KeyUp + rune(fin-'A')}
	case 'E':
		k = KeyEvent{Code: KeyBegin}
	case 'F':
		k = KeyEvent{Code: KeyEnd}
	case 'H':
		k = KeyEvent{Code: KeyHome}
	case 'P', 'Q', 'R', 'S':
		k = KeyEvent{Code: KeyF1 + rune(fin-'P')}
	case 'Z':
		k = KeyEvent{Code: KeyTab, Mod: ModShift}
	}
	id := at(ps, 0, 1)
	if len(ps) > 2 && !ps[1].more || id != 1 {
		return KeyEvent{}, false
	}
	if len(ps) > 1 {
		k.Mod |= legacyMod(at(ps, 1, 1))
	}
	return kittyEvent(ps, k), true
}

// csiTilde is a key CSI n ; m ~ names, xterm's modifyOtherKeys
// CSI 27 ; m ; code ~, or the start of a bracketed paste, which csiKey
// leaves to csi; and urxvt's CSI n ^ and CSI n @, ctrl and ctrl+shift.
func csiTilde(fin byte, ps []param) (KeyEvent, bool) {
	if len(ps) == 0 {
		return KeyEvent{}, false
	}
	n := at(ps, 0, 0)
	if fin == '~' && n == 27 {
		if len(ps) != 3 {
			return KeyEvent{}, false
		}
		return modifyOtherKeys(ps), true
	}
	var k KeyEvent
	switch {
	case n == 1 || n == 7:
		k.Code = KeyHome
	case n == 2:
		k.Code = KeyInsert
	case n == 3:
		k.Code = KeyDelete
	case n == 4 || n == 8:
		k.Code = KeyEnd
	case n == 5:
		k.Code = KeyPgUp
	case n == 6:
		k.Code = KeyPgDown
	case n >= 11 && n <= 15:
		k.Code = KeyF1 + rune(n-11)
	case n >= 17 && n <= 21:
		k.Code = KeyF6 + rune(n-17)
	case n >= 23 && n <= 26:
		k.Code = KeyF11 + rune(n-23)
	case n == 28 || n == 29:
		k.Code = KeyF15 + rune(n-28)
	case n >= 31 && n <= 34:
		k.Code = KeyF17 + rune(n-31)
	default:
		return KeyEvent{}, false
	}
	if len(ps) > 1 && ps[1].v >= 0 {
		k.Mod |= legacyMod(ps[1].v)
	}
	switch fin {
	case '~':
		return kittyEvent(ps, k), true
	case '^':
		k.Mod |= ModCtrl
	case '@':
		k.Mod |= ModCtrl | ModShift
	}
	return k, true
}

// legacyMod is xterm's modifier parameter, 1 plus the modifier bits.
func legacyMod(m int) Mod {
	if m < 1 || m-1 > 0xffff {
		return 0
	}
	return Mod(m - 1) //nolint:gosec // within uint16, checked above
}

// kittyEvent applies the Kitty protocol's event type, the sub-parameter
// after the modifiers, to a legacy key: a repeat or a release.
func kittyEvent(ps []param, k KeyEvent) KeyEvent {
	if len(ps) > 2 && ps[1].more {
		switch at(ps, 2, 1) {
		case 2:
			k.IsRepeat = true
		case 3:
			k.Release = true
		}
	}
	return k
}

// modifyOtherKeys reads xterm's CSI 27 ; m ; code ~.
func modifyOtherKeys(ps []param) KeyEvent {
	mod := legacyMod(at(ps, 1, 1))
	r := rune(at(ps, 2, 1)) //nolint:gosec // bounded by the parser
	switch r {
	case 0x08, 0x7f:
		return KeyEvent{Code: KeyBackspace, Mod: mod}
	case 0x09:
		return KeyEvent{Code: KeyTab, Mod: mod}
	case 0x0d:
		return KeyEvent{Code: KeyEnter, Mod: mod}
	case esc:
		return KeyEvent{Code: KeyEscape, Mod: mod}
	}
	k := KeyEvent{Code: r, Mod: mod}
	if k.Mod <= ModShift {
		k.Text = string(r)
	}
	return k
}

// The Kitty protocol's modifier bits, which number super and meta the
// other way round from xterm's.
const (
	kittyShift = 1 << iota
	kittyAlt
	kittyCtrl
	kittySuper
	kittyHyper
	kittyMeta
	kittyCapsLock
	kittyNumLock
)

// fromKittyMod is the Kitty protocol's modifier bits as a Mod.
func fromKittyMod(mod int) Mod {
	var m Mod
	for _, b := range []struct {
		kitty int
		mod   Mod
	}{
		{kittyShift, ModShift}, {kittyAlt, ModAlt}, {kittyCtrl, ModCtrl}, {kittySuper, ModSuper},
		{kittyHyper, ModHyper}, {kittyMeta, ModMeta}, {kittyCapsLock, ModCapsLock}, {kittyNumLock, ModNumLock},
	} {
		if mod&b.kitty != 0 {
			m |= b.mod
		}
	}
	return m
}

// kitty reads the Kitty keyboard protocol's
// CSI code : shifted : base ; mods : event ; text u, as ultraviolet's
// parseKittyKeyboard does; xterm's formatOtherKeys CSI code ; m u reads the
// same.
func kitty(ps []param) KeyEvent {
	var k KeyEvent
	field, sub := 0, 0
	for _, p := range ps {
		switch field {
		case 0:
			switch sub {
			case 0:
				code := p.get(1)
				if known, ok := kittyKeys[code]; ok {
					k = known
				} else {
					r := rune(code) //nolint:gosec // bounded by the parser
					if !utf8.ValidRune(r) {
						r = utf8.RuneError
					}
					k.Code = r
				}
			case 2:
				// ultraviolet falls through here to set ShiftedCode to the
				// base code as well; the shifted code is the one before it.
				if b := rune(p.get(1)); unicode.IsPrint(b) { //nolint:gosec // bounded by the parser
					k.BaseCode = b
				}
			case 1:
				if s := rune(p.get(1)); unicode.IsPrint(s) { //nolint:gosec // bounded by the parser
					k.ShiftedCode = s
				}
			}
		case 1:
			switch sub {
			case 0:
				if mod := p.get(1); mod > 1 {
					k.Mod = fromKittyMod(mod - 1)
					if k.Mod > ModShift {
						k.Text = ""
					}
				}
			case 1:
				switch p.get(1) {
				case 2:
					k.IsRepeat = true
				case 3:
					k.Release = true
				}
			}
		case 2:
			if code := p.get(0); code != 0 {
				k.Text += string(rune(code)) //nolint:gosec // bounded by the parser
			}
		}
		sub++
		if !p.more {
			field, sub = field+1, 0
		}
	}
	kittyText(&k)
	return k
}

// kittyText fills in the text of a Kitty key that sent none, as
// ultraviolet does: the keypad's characters, and a printable key's rune in
// the case its modifiers ask for.
func kittyText(k *KeyEvent) {
	mod := k.Mod &^ ModNumLock
	printMod := mod <= ModShift || mod == ModCapsLock || mod == ModShift|ModCapsLock
	if k.Text == "" && printMod && k.Code >= KeyKpEqual && k.Code <= KeyKpSep {
		switch {
		case k.Code >= KeyKp0 && k.Code <= KeyKp9:
			k.Text = string('0' + k.Code - KeyKp0)
		case k.Code == KeyKpEqual:
			k.Text = "="
		case k.Code == KeyKpMultiply:
			k.Text = "*"
		case k.Code == KeyKpPlus:
			k.Text = "+"
		case k.Code == KeyKpMinus:
			k.Text = "-"
		case k.Code == KeyKpDecimal:
			k.Text = "."
		case k.Code == KeyKpDivide:
			k.Text = "/"
		case k.Code == KeyKpSep:
			k.Text = ","
		}
	}
	if k.Text == "" && printMod && unicode.IsPrint(k.Code) {
		switch {
		case mod == 0:
			k.Text = string(k.Code)
		case k.ShiftedCode != 0:
			k.Text = string(k.ShiftedCode)
		case mod.Contains(ModShift) || mod.Contains(ModCapsLock):
			k.Text = string(unicode.ToUpper(k.Code))
		default:
			k.Text = string(unicode.ToLower(k.Code))
		}
	}
}

// ss3 decodes an SS3 key whose digits start at b[i]: the application
// cursor and keypad keys, with urxvt's modifier digits.
func (d *Decoder) ss3(b []byte, i int, final bool) (int, *Input, bool) {
	mod, limit := 0, d.maxSequence()
	for ; i < len(b) && b[i] >= '0' && b[i] <= '9'; i++ {
		if i >= limit {
			return d.tooLong(b, skipSS3)
		}
		if mod < 1<<16 {
			mod = mod*10 + int(b[i]-'0')
		}
	}
	if i >= len(b) {
		if !final {
			return 0, nil, false
		}
		return len(b), unknown(b, false), true
	}
	gl := b[i]
	if gl < 0x21 || gl > 0x7e {
		return i, unknown(b[:i], false), true
	}
	if i >= limit {
		return d.tooLong(b, skipSS3)
	}
	i++
	var k KeyEvent
	switch {
	case gl >= 'a' && gl <= 'd':
		k = KeyEvent{Code: KeyUp + rune(gl-'a'), Mod: ModCtrl}
	case gl >= 'A' && gl <= 'D':
		k = KeyEvent{Code: KeyUp + rune(gl-'A')}
	case gl == 'E':
		k = KeyEvent{Code: KeyBegin}
	case gl == 'F':
		k = KeyEvent{Code: KeyEnd}
	case gl == 'H':
		k = KeyEvent{Code: KeyHome}
	case gl >= 'P' && gl <= 'S':
		k = KeyEvent{Code: KeyF1 + rune(gl-'P')}
	case gl == 'M':
		k = KeyEvent{Code: KeyKpEnter}
	case gl == 'X':
		k = KeyEvent{Code: KeyKpEqual}
	case gl >= 'j' && gl <= 'y':
		k = KeyEvent{Code: KeyKpMultiply + rune(gl-'j')}
	default:
		return i, unknown(b[:i], false), true
	}
	if mod > 0 {
		k.Mod |= legacyMod(mod)
	}
	return i, key(b[:i], k), true
}

// str decodes an OSC, DCS, APC, PM or SOS whose text starts at b[i], to its
// end: BEL, ST or ESC \; CAN or SUB, which cancel it; or an ESC before
// anything else, which cuts it there and begins what follows, as ECMA-48
// has it. ultraviolet consumes that ESC with the string; leaving it out
// keeps a cut reply from ending in the bare ESC termcap.ParseReply reads as
// an end. An ESC with nothing after it waits for the next byte, or for
// Flush, which ends the string with it.
func (d *Decoder) str(b []byte, i int, final bool) (int, *Input, bool) {
	start, limit := i, d.maxSequence()
	for ; i < len(b); i++ {
		if i >= limit {
			return d.tooLong(b, skipString)
		}
		switch b[i] {
		case bel, st, can, sub:
			return i + 1, unknown(b[:i+1], false), true
		case esc:
			switch {
			case i+1 >= len(b) && !final:
				return 0, nil, false // ESC \ may follow
			case i+1 < len(b) && b[i+1] == '\\':
				if i+2 > limit {
					return d.tooLong(b, skipString)
				}
				return i + 2, unknown(b[:i+2], false), true
			case i == start && start == 2:
				// ESC, an introducer, then ESC and no \: alt and the
				// introducer's key, as ultraviolet reads it, and the second
				// ESC begins what follows.
				return 2, key(b[:2], altIntroducer(b[1])), true
			case i+1 >= len(b):
				// A bare ESC as the input goes quiet ends the string, as old
				// urxvt ends its replies and termcap.ParseReply reads them.
				return i + 1, unknown(b[:i+1], false), true
			}
			return i, unknown(b[:i], false), true
		}
	}
	if !final {
		return 0, nil, false // at most limit bytes, or the scan would have cut it
	}
	return len(b), unknown(b, false), true
}

// pasteBytes takes b into the paste, up to its end, which ends the paste
// as one input. Without the end in b, the bytes that may begin it are held
// for the next Decode, unless final.
func (d *Decoder) pasteBytes(b []byte, out []Input, final bool) (int, []Input, bool) {
	if i := bytes.Index(b, []byte(pasteEnd)); i >= 0 {
		d.addPaste(b[:i])
		return i + len(pasteEnd), append(out, d.endPaste()), true
	}
	keep := 0
	if !final {
		for k := min(len(pasteEnd)-1, len(b)); k > 0; k-- {
			if bytes.HasSuffix(b, []byte(pasteEnd[:k])) {
				keep = k
				break
			}
		}
	}
	n := len(b) - keep
	if n == 0 {
		return 0, out, false
	}
	d.addPaste(b[:n])
	return n, out, true
}

// addPaste adds p to the paste, up to MaxPaste bytes.
func (d *Decoder) addPaste(p []byte) {
	if room := d.maxPaste() - len(d.paste); len(p) > room {
		p, d.truncated = p[:max(room, 0)], true
	}
	d.paste = append(d.paste, p...)
}

// endPaste ends the paste, and gives it.
func (d *Decoder) endPaste() Input {
	in := Input{Kind: InputPaste, Text: strings.ToValidUTF8(string(d.paste), ""), Truncated: d.truncated}
	d.inPaste, d.paste, d.truncated = false, nil, false
	return in
}
