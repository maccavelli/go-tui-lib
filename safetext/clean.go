package safetext

import (
	"strings"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
)

// Clean returns s as the rules let it reach a terminal. When nothing needs
// to change, it returns s itself and allocates nothing; otherwise it reads
// s once and allocates its result once.
func Clean(s string, r Rules) string {
	if printableASCII(s) {
		return s // no rule removes, replaces or rewrites printable ASCII
	}
	var m machine
	m.init(&r, modeClean, s)
	run(&m, s)
	m.finish()
	return m.o.result()
}

// printableASCII reports whether s is printable ASCII alone.
func printableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

// IsClean reports whether Clean(s, r) would return s: whether s holds
// nothing the rules would remove, replace or rewrite. It allocates nothing
// and stops at the first byte Clean would change, so a validator that
// refuses text rather than cleaning it can ask cheaply.
func IsClean(s string, r Rules) bool {
	var m machine
	m.init(&r, modeCompare, s)
	run(&m, s)
	if !m.o.diverged {
		m.finish()
	}
	return !m.o.diverged && m.o.pos == len(s)
}

// The output modes of a machine.
const (
	// modeClean compares the output with the input, and copies from the
	// first byte that differs.
	modeClean = iota
	// modeCompare compares the output with the input, and stops at the
	// first byte that differs.
	modeCompare
	// modeStream appends the output to a buffer a Sanitizer writes out.
	modeStream
)

// output is where a machine's output goes.
type output struct {
	mode     int
	src      string          // modeClean and modeCompare: the input
	pos      int             // how much of src the output so far equals
	diverged bool            // the output no longer equals a prefix of src
	sb       strings.Builder // modeClean, once diverged
	buf      []byte          // modeStream
}

// str adds s to the output.
func (o *output) str(s string) {
	if o.mode == modeStream {
		o.buf = append(o.buf, s...)
		return
	}
	if o.diverged {
		if o.mode == modeClean {
			o.sb.WriteString(s)
		}
		return
	}
	if strings.HasPrefix(o.src[o.pos:], s) {
		o.pos += len(s)
		return
	}
	o.diverge()
	if o.mode == modeClean {
		o.sb.WriteString(s)
	}
}

// byt adds c to the output.
func (o *output) byt(c byte) {
	switch {
	case o.mode == modeStream:
		o.buf = append(o.buf, c)
	case !o.diverged && o.pos < len(o.src) && o.src[o.pos] == c:
		o.pos++
	default:
		if !o.diverged {
			o.diverge()
		}
		if o.mode == modeClean {
			o.sb.WriteByte(c)
		}
	}
}

// raw adds b, bytes held by the machine, to the output, without making a
// string of them.
func (o *output) raw(b []byte) {
	switch {
	case o.mode == modeStream:
		o.buf = append(o.buf, b...)
	case !o.diverged && len(o.src)-o.pos >= len(b) && string(b) == o.src[o.pos:o.pos+len(b)]:
		o.pos += len(b)
	default:
		if !o.diverged {
			o.diverge()
		}
		if o.mode == modeClean {
			o.sb.Write(b)
		}
	}
}

// escaped adds r, escaped, to the output.
func (o *output) escaped(r rune) {
	var tmp [16]byte
	o.str(string(appendEscaped(tmp[:0], r)))
}

// diverge starts the copy: the output so far is src[:pos].
func (o *output) diverge() {
	o.diverged = true
	if o.mode == modeClean {
		o.sb.Grow(len(o.src) + 16)
		o.sb.WriteString(o.src[:o.pos])
	}
}

// result is Clean's result: src itself, or its prefix, when the output
// never differed from it, else the copy.
func (o *output) result() string {
	if !o.diverged {
		return o.src[:o.pos]
	}
	return o.sb.String()
}

// The parser's states, after vt100.net's DEC parser as x/ansi v0.11.8's
// transition table has it (parser/transition_table.go), with 0009-MADR §2's
// rules where they differ: BEL ends every string, a string past MaxString
// is abandoned, and a non-ASCII character inside an SOS, PM or APC is part
// of it, as it is of an OSC or DCS.
const (
	stGround = iota
	stEscape
	stEscInter
	stCsiEntry
	stCsiParam
	stCsiInter
	stOsc
	stDcsEntry
	stDcsParam
	stDcsInter
	stDcsString
	stString // SOS, PM and APC
)

// sgrMax is how many bytes of a CSI's parameters a kept SGR may have; 32
// parameters of two digits each with their separators fit.
const sgrMax = 128

// maxSGRParams is the most parameters a kept SGR may have: x/ansi
// v0.11.8's parser, which Bubble Tea's renderer runs over every frame,
// holds 32, and panics on the 33rd (docs/decisions/0014-PLAN-hardening.md
// Step 9, D9).
const maxSGRParams = 32

// oscMax is the longest OSC 8 a machine keeps, in bytes: a link's target
// longer than this is removed with its sequence.
const oscMax = 2048

// machine is the parser and the rules it cleans by. Its zero value is not
// ready: init readies it, in place, since it is large.
type machine struct {
	// The rules.
	tabs, newlines, flatten, bidi, visible, escape, keepSGR bool
	control                                                 string // Visible's mark
	maxString                                               int
	schemes                                                 []string

	st     int
	brk    bool // the last output was Flatten's space
	cr     bool // a \r waits for the byte after it
	stTail bool // an ESC ended a string, so a '\' after it is its ST
	fromC1 bool // the bytes being parsed are a C1 code point's 7-bit form

	strLen   int  // the bytes the current string holds
	csiDirty bool // the current CSI cannot be a kept SGR
	csiN     int
	csi      [sgrMax]byte

	oscKeep    bool // the current OSC may be a kept OSC 8
	oscPending bool // an ESC ended it; ST decides
	oscN       int
	osc        [oscMax]byte

	pend  [4]byte // modeStream: the start of a character a Write cut
	pendN int

	o output
}

// init readies m, a zero machine, for r, writing in mode, with src the
// input in modeClean and modeCompare.
func (m *machine) init(r *Rules, mode int, src string) {
	m.control = r.Glyphs.Control
	if m.control == "" {
		m.control = glyph.Unicode().Control
	}
	m.maxString = r.MaxString
	if m.maxString <= 0 {
		m.maxString = DefaultMaxString
	}
	m.tabs, m.newlines, m.flatten, m.bidi = r.Tabs, r.Newlines, r.Flatten, r.Bidi
	m.visible, m.escape, m.keepSGR = r.Visible && !r.Escape, r.Escape, r.KeepSGR && !r.Escape
	m.schemes = r.LinkSchemes
	m.o.mode, m.o.src = mode, src
}

// run feeds s to m. In modeStream a character s cuts short at its end is
// kept for the next call; otherwise it is invalid UTF-8.
func run[T string | []byte](m *machine, s T) {
	for i := 0; i < len(s); {
		if m.o.diverged && m.o.mode == modeCompare {
			return
		}
		if c := s[i]; c < 0x80 {
			if m.st == stGround && !m.cr && c >= 0x20 && c < 0x7f {
				i = textRun(m, s, i, i+1)
				continue
			}
			if c == 0x1b && m.st == stGround && !m.escape {
				if n := wholeSequence(m, s, i); n > 0 {
					i += n
					continue
				}
			}
			if n := bulk(m, s, i); n > 0 {
				i += n
				continue
			}
			m.ascii(c)
			i++
			continue
		}
		r, n, ok, partial := decode(s[i:])
		switch {
		case partial && m.o.mode == modeStream:
			m.pendN = copy(m.pend[:], s[i:])
			return
		case ok && m.st == stGround && !m.cr && m.plain(r):
			i = textRun(m, s, i, i+n)
		case ok:
			char(m, r, s[i:i+n])
			i += n
		default:
			char(m, 0xfffd, "\ufffd")
			i += n
		}
	}
}

// textRun writes the run of text that passes as it is, printable ASCII
// and characters no rule removes, from s[i:], whose first piece ends at j,
// in one piece, and returns where the run ends.
func textRun[T string | []byte](m *machine, s T, i, j int) int {
	for j < len(s) {
		if b := s[j]; b >= 0x20 && b < 0x7f {
			j++
			continue
		} else if b < 0x80 {
			break
		}
		r, n, ok, _ := decode(s[j:])
		if !ok || !m.plain(r) {
			break
		}
		j += n
	}
	emitSpan(m, s, i, j)
	m.brk = false
	return j
}

// emitSpan adds s[i:j], printable text from the input, to the output. When
// the output so far is the input up to i, the span is the input's own
// bytes, so the output simply moves on: nothing to compare.
func emitSpan[T string | []byte](m *machine, s T, i, j int) {
	if m.o.mode != modeStream && !m.o.diverged && m.o.pos == i {
		m.o.pos = j
		return
	}
	emitRaw(m, s[i:j])
}

// wholeSequence takes, at the ESC s[i], a CSI or an OSC that s holds whole
// and in the plainest form, in one step, and reports its length; 0 leaves
// it to the byte-at-a-time parser, which decides every other form. The two
// forms are a CSI of parameter bytes, then intermediate bytes, then its
// final byte; and an OSC of printable ASCII, no longer than MaxString,
// ended by BEL or ST. Each is kept or removed as the parser would decide.
func wholeSequence[T string | []byte](m *machine, s T, i int) int {
	if i+2 >= len(s) {
		return 0
	}
	switch s[i+1] {
	case '[':
		p := i + 2
		j := p
		for j < len(s) && s[j] >= 0x30 && s[j] <= 0x3f {
			j++
		}
		pe := j
		for j < len(s) && s[j] >= 0x20 && s[j] <= 0x2f {
			j++
		}
		if j >= len(s) || s[j] < 0x40 || s[j] > 0x7e {
			return 0
		}
		m.settleCR(0x1b)
		if m.keepSGR && s[j] == 'm' && pe == j && keptSGR(s[p:pe]) {
			emitSpan(m, s, i, j+1)
		} else {
			m.done()
		}
		return j + 1 - i
	case ']':
		j := i + 2
		for j < len(s) && s[j] >= 0x20 && s[j] < 0x7f {
			j++
		}
		if j >= len(s) || j-(i+2) > m.maxString {
			return 0
		}
		end := j + 1
		switch {
		case s[j] == 0x07:
		case s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\':
			end = j + 2
		default:
			return 0
		}
		m.settleCR(0x1b)
		body := s[i+2 : j]
		if m.keepSGR && len(body) <= oscMax && keepLink(body, m.schemes) {
			emitSpan(m, s, i, end)
		} else {
			m.done()
		}
		return end - i
	}
	return 0
}

// keptSGR reports whether params, a CSI's parameter bytes before its final
// m, make an SGR KeepSGR keeps: no prefix, no more than sgrMax bytes, and
// fewer than maxSGRParams separators.
func keptSGR[T string | []byte](params T) bool {
	if len(params) > sgrMax {
		return false
	}
	n := 0
	for k := range len(params) {
		switch c := params[k]; {
		case c > 0x3b:
			return false // a prefix, or one out of place
		case c == ';' || c == ':':
			n++
		}
	}
	return n < maxSGRParams
}

// bulk takes the run of bytes at s[i:] that only continue the current
// sequence, a CSI's parameters or a string's printable content, in one
// step, and reports how many it took; 0 leaves s[i] to the byte-at-a-time
// parser. It changes nothing the parser would not.
func bulk[T string | []byte](m *machine, s T, i int) int {
	switch m.st {
	case stCsiEntry, stCsiParam:
		j := i
		for j < len(s) && s[j] >= 0x30 && s[j] <= 0x3b {
			j++
		}
		n := j - i
		if n == 0 {
			return 0
		}
		m.st = stCsiParam
		if m.csiN+n <= len(m.csi) {
			m.csiN += copy(m.csi[m.csiN:], s[i:j])
		} else {
			m.csiDirty = true
		}
		return n
	case stOsc, stDcsString, stString:
		j := i
		for j < len(s) && s[j] >= 0x20 && s[j] < 0x7f {
			j++
		}
		n := min(j-i, m.maxString-m.strLen) // the byte past MaxString goes byte by byte
		if n <= 0 {
			return 0
		}
		if m.oscKeep {
			if m.oscN+n <= len(m.osc) {
				m.oscN += copy(m.osc[m.oscN:], s[i:i+n])
			} else {
				m.oscKeep = false
			}
		}
		m.strLen += n
		return n
	}
	return 0
}

// emitRaw adds the input s, printable text, to the output.
func emitRaw[T string | []byte](m *machine, s T) {
	if m.o.mode == modeStream {
		m.o.buf = append(m.o.buf, s...)
		return
	}
	m.o.str(string(s))
}

// plain reports whether r, from U+0080 up, passes as it is: not a C1
// control, a line or paragraph separator, a format character no rule
// keeps, or a bidirectional control the rules remove.
func (m *machine) plain(r rune) bool {
	switch {
	case r <= 0x9f:
		return false
	case r < 0x2000:
		return r != 0x00ad && (r != 0x061c || m.bidi)
	case r >= 0x2070 && r < 0xe0000:
		return r != 0xfeff // the common case: no rule touches this range but for U+FEFF
	}
	return r != 0x2028 && r != 0x2029 && !invisible(r) && (m.bidi || !bidi(r))
}

// print adds printable text to the output.
func (m *machine) print(s string) {
	m.o.str(s)
	m.brk = false
}

// removed takes r out of the text: escaped under Escape, else dropped.
func (m *machine) removed(r rune) {
	if m.escape {
		m.o.escaped(r)
		m.brk = false
	}
}

// space is Flatten's space for a break, once for each run.
func (m *machine) space() {
	if !m.brk {
		m.o.byt(' ')
		m.brk = true
	}
}

// settleCR decides a waiting \r: the \n after it makes \r\n one \n.
func (m *machine) settleCR(next rune) bool {
	if !m.cr {
		return false
	}
	m.cr = false
	if next == '\n' {
		m.print("\n")
		return true
	}
	m.removed('\r')
	return false
}

// text handles a control, a break or DEL met as text: in ground, or
// executed inside a CSI or an escape, as the DEC parser executes it.
func (m *machine) text(r rune) {
	if m.settleCR(r) {
		return
	}
	switch {
	case m.flatten && isBreak(r):
		m.space()
	case r == '\t' && m.tabs:
		m.print("\t")
	case (r == '\n' || r == 0x2028 || r == 0x2029) && m.newlines:
		m.print("\n")
	case r == '\r' && m.newlines:
		m.cr = true
	default:
		m.removed(r)
	}
}

// char handles a character from U+0080 up, or U+FFFD for invalid UTF-8,
// with raw its bytes as they came.
func char[T string | []byte](m *machine, r rune, raw T) {
	if r >= 0x80 && r <= 0x9f {
		if m.escape {
			m.settleCR(r)
			m.removed(r)
			return
		}
		// A C1 control is its 7-bit form (0009-MADR §2), and KeepSGR
		// keeps nothing it introduces or ends (0020-MADR A3.3).
		m.fromC1 = true
		m.ascii(0x1b)
		m.ascii(byte(r - 0x40))
		m.fromC1 = false
		return
	}
	switch m.st {
	case stOsc, stDcsString, stString:
		if m.content(len(raw), false) {
			return
		}
	case stGround:
	default:
		m.drop() // the DEC parser leaves a CSI or an escape for UTF-8 text
	}
	if m.settleCR(r) {
		return
	}
	switch {
	case r == 0x2028 || r == 0x2029:
		m.text(r)
	case invisible(r), bidi(r) && !m.bidi:
		m.removed(r)
	default:
		emitRaw(m, raw)
		m.brk = false
	}
}

// ascii handles one byte below 0x80.
func (m *machine) ascii(c byte) {
	if m.escape {
		if c < 0x20 || c == 0x7f {
			m.text(rune(c))
			return
		}
		m.settleCR(rune(c))
		m.print(string(rune(c)))
		return
	}
	if (c == 0x18 || c == 0x1a) && m.st != stGround {
		m.drop() // CAN and SUB abort any sequence
		m.text(rune(c))
		return
	}
	switch m.st {
	case stGround:
		switch {
		case c == 0x1b:
			m.settleCR(0x1b)
			m.begin(stEscape)
		case c < 0x20 || c == 0x7f:
			m.text(rune(c))
		default:
			m.settleCR(rune(c))
			m.o.byt(c)
			m.brk = false
		}
	case stEscape:
		m.escapeByte(c)
	case stEscInter:
		switch {
		case c == 0x1b:
			m.drop()
			m.begin(stEscape)
		case c < 0x20:
			m.text(rune(c))
		case c == 0x7f:
		case c < 0x30:
		default:
			m.done()
		}
	case stCsiEntry, stCsiParam, stCsiInter:
		m.csiByte(c)
	case stOsc:
		m.oscByte(c)
	case stDcsEntry, stDcsParam, stDcsInter:
		m.dcsByte(c)
	case stDcsString, stString:
		switch c {
		case 0x1b:
			m.done()
			m.st, m.stTail = stEscape, true
		case 0x07:
			m.done() // BEL ends every string (0009-MADR §2)
		default:
			if !m.content(1, false) {
				m.ascii(c)
			}
		}
	}
}

// begin starts a sequence in state st.
func (m *machine) begin(st int) {
	m.st, m.stTail = st, false
	m.strLen, m.csiN, m.csiDirty = 0, 0, false
	m.oscN, m.oscKeep, m.oscPending = 0, false, false
}

// done ends the current sequence, which is removed: Visible's mark in its
// place.
func (m *machine) done() {
	if m.visible {
		m.print(m.control)
	}
	m.st = stGround
}

// drop is done for a sequence cut short: by CAN or SUB, an ESC, UTF-8 text,
// MaxString or the end of the input. An ESC that ended a string and is cut
// short is that string's terminator, already counted, unless it ended an
// OSC whose fate waited for an ST.
func (m *machine) drop() {
	switch {
	case m.oscPending:
		m.oscPending, m.stTail = false, false
	case m.stTail:
		m.stTail = false
		m.st = stGround
		return
	}
	m.done()
}

// escapeByte handles c after ESC.
func (m *machine) escapeByte(c byte) {
	if m.stTail {
		if c == '\\' {
			m.stTail = false
			m.st = stGround // the ST of the string that ESC ended
			if m.oscPending {
				m.oscPending = false
				m.oscFinish("\x1b\\")
			}
			return
		}
		// Not an ST: the ESC that ended the string begins a new sequence,
		// which c continues. An OSC that waited for its ST is removed.
		m.stTail = false
		if m.oscPending {
			m.oscPending = false
			if m.visible {
				m.print(m.control)
			}
		}
	}
	switch {
	case c == 0x1b:
		m.drop()
		m.begin(stEscape)
	case c < 0x20:
		m.text(rune(c))
	case c == 0x7f:
	case c < 0x30:
		m.st = stEscInter
	case c == '[':
		m.begin(stCsiEntry)
		m.csiDirty = m.fromC1
	case c == ']':
		m.begin(stOsc)
		m.oscKeep = m.keepSGR && !m.fromC1
	case c == 'P':
		m.begin(stDcsEntry)
	case c == 'X' || c == '^' || c == '_':
		m.begin(stString)
	default:
		m.done()
	}
}

// csiByte handles c inside a CSI.
func (m *machine) csiByte(c byte) {
	switch {
	case c == 0x1b:
		m.drop()
		m.begin(stEscape)
	case c < 0x20:
		m.csiDirty = true
		m.text(rune(c))
	case c == 0x7f:
		m.csiDirty = true
	case c >= 0x40:
		m.dispatchCSI(c)
	case m.st == stCsiInter:
		if c >= 0x30 {
			m.done() // x/ansi: a parameter after an intermediate ends the CSI
			return
		}
	case c < 0x30:
		m.csiDirty = true
		m.st = stCsiInter
	case c >= 0x3c:
		m.csiDirty = true // a prefix, or a prefix out of place
		m.st = stCsiParam
	default:
		m.st = stCsiParam
		if m.csiN < len(m.csi) {
			m.csi[m.csiN] = c
			m.csiN++
		} else {
			m.csiDirty = true
		}
	}
}

// dispatchCSI ends a CSI with final byte c: kept when it is an SGR KeepSGR
// keeps, else removed.
func (m *machine) dispatchCSI(c byte) {
	if c != 'm' || !m.keepSGR || m.csiDirty {
		m.done()
		return
	}
	params := m.csi[:m.csiN]
	if !keptSGR(params) {
		m.done()
		return
	}
	m.st = stGround
	m.o.str("\x1b[")
	m.o.raw(params)
	m.o.byt('m')
}

// oscByte handles c inside an OSC.
func (m *machine) oscByte(c byte) {
	switch {
	case c == 0x1b:
		m.st, m.stTail = stEscape, true
		if m.oscKeep {
			m.oscPending = true
			return
		}
		if m.visible {
			m.print(m.control)
		}
	case c == 0x07:
		if m.oscKeep {
			m.st = stGround
			m.oscFinish("\a")
			return
		}
		m.done()
	case c < 0x20:
		m.oscKeep = false // ignored inside an OSC; not a printable link
	default:
		if !m.content(1, c == 0x7f) {
			m.ascii(c)
			return
		}
		if m.oscKeep {
			m.osc[m.oscN] = c
			m.oscN++
		}
	}
}

// content counts n more bytes of the current string. When they would run
// it past MaxString it abandons the string, so a lost terminator hides at
// most MaxString bytes, and reports false: those bytes are then text.
// notASCII marks a byte that no kept OSC 8 may hold.
func (m *machine) content(n int, notASCII bool) bool {
	if m.strLen+n > m.maxString {
		m.drop()
		return false
	}
	if notASCII || n > 1 || m.oscN+n > len(m.osc) {
		m.oscKeep = false
	}
	m.strLen += n
	return true
}

// oscFinish ends an OSC that may be a kept OSC 8, with its terminator.
func (m *machine) oscFinish(term string) {
	body := m.osc[:m.oscN]
	if !m.oscKeep || m.fromC1 || !keepLink(body, m.schemes) {
		if m.visible {
			m.print(m.control)
		}
		return
	}
	m.o.str("\x1b]")
	m.o.raw(body)
	m.o.str(term)
}

// keepLink reports whether body, an OSC's printable-ASCII content, is an
// OSC 8 hyperlink KeepSGR and LinkSchemes keep: "8;", parameters, ";" and
// a target.
func keepLink[T string | []byte](body T, schemes []string) bool {
	if len(body) < 2 || body[0] != '8' || body[1] != ';' {
		return false
	}
	semi := -1
	for k := 2; k < len(body); k++ {
		if body[k] == ';' {
			semi = k
			break
		}
	}
	if semi < 0 {
		return false
	}
	target := body[semi+1:]
	if len(target) == 0 || len(schemes) == 0 {
		return true // a link's end, or any link
	}
	return linkAllowed(target, schemes)
}

// linkAllowed reports whether target, printable ASCII, has a scheme of
// schemes, compared without case, and a host or an opaque part, as RFC 3986
// reads them: scheme ":" then either "//" and an authority whose host is
// not empty, or a part that does not begin with "/", as mailto's address.
func linkAllowed[T string | []byte](target T, schemes []string) bool {
	colon := -1
	for k := range len(target) {
		c := target[k]
		alpha := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		if k == 0 && !alpha {
			return false
		}
		if c == ':' {
			colon = k
			break
		}
		digit := c >= '0' && c <= '9'
		if !alpha && !digit && c != '+' && c != '-' && c != '.' {
			return false
		}
	}
	if colon <= 0 {
		return false
	}
	scheme, rest := target[:colon], target[colon+1:]
	known := false
	for _, s := range schemes {
		if equalFoldASCII(scheme, s) {
			known = true
			break
		}
	}
	if !known {
		return false
	}
	if len(rest) < 2 || rest[0] != '/' || rest[1] != '/' {
		return len(rest) > 0 && rest[0] != '/' // an opaque part
	}
	auth := rest[2:]
	for k := range len(auth) {
		if c := auth[k]; c == '/' || c == '?' || c == '#' {
			auth = auth[:k]
			break
		}
	}
	for k := len(auth) - 1; k >= 0; k-- {
		if auth[k] == '@' {
			auth = auth[k+1:] // the userinfo is not the host
			break
		}
	}
	if len(auth) > 0 && auth[0] == '[' {
		return len(auth) > 2 // an IPv6 literal
	}
	for k := range len(auth) {
		if auth[k] == ':' {
			auth = auth[:k] // the port is not the host
			break
		}
	}
	return len(auth) > 0
}

// equalFoldASCII reports whether b and s are equal, ASCII letters compared
// without case.
func equalFoldASCII[T string | []byte](b T, s string) bool {
	if len(b) != len(s) {
		return false
	}
	for i := range len(b) {
		x, y := b[i], s[i]
		if x >= 'A' && x <= 'Z' {
			x += 'a' - 'A'
		}
		if y >= 'A' && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}

// dcsByte handles c in a DCS before its string.
func (m *machine) dcsByte(c byte) {
	switch {
	case c == 0x1b && m.st == stDcsEntry:
		m.st = stDcsString // x/ansi: passed through, as tmux and screen do
		m.content(1, false)
	case c == 0x1b:
		m.drop()
		m.begin(stEscape)
	case m.st == stDcsEntry && c >= 0x08 && c <= 0x0d:
		m.st = stDcsString
		m.content(1, false)
	case c < 0x20 || c == 0x7f:
	case c < 0x30:
		m.st = stDcsInter
	case m.st == stDcsInter, c >= 0x40:
		m.st = stDcsString
		m.content(1, false)
	default:
		m.st = stDcsParam
	}
}

// finish ends the input: a waiting \r is lone, a character cut short is
// invalid UTF-8, and a sequence not ended is removed.
func (m *machine) finish() {
	if m.pendN > 0 {
		pend := m.pend[:m.pendN]
		m.pendN = 0
		for len(pend) > 0 {
			_, n, _, _ := decode(pend)
			char(m, 0xfffd, "\ufffd")
			pend = pend[n:]
		}
	}
	if m.cr {
		m.cr = false
		m.removed('\r')
	}
	if m.st != stGround {
		m.drop()
	}
}
