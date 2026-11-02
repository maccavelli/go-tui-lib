package stream

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// The bounds Interpret applies where InterpretOptions leaves them at zero
// (docs/decisions/0009-MADR-streaming-content-engine.md A1, as grok's).
const (
	// DefaultInterpretRows is the most rows Interpret gives: past it, the
	// oldest go, so the output's tail is kept.
	DefaultInterpretRows = 50_000
	// DefaultInterpretCols is the most cells a row holds: past it, the row
	// is cut.
	DefaultInterpretCols = 8_192
)

const (
	// maxSGRParams is the most parameters an SGR may have to be read, as
	// safetext's KeepSGR keeps; one with more is dropped.
	maxSGRParams = 32
	// maxSubParams is the most parts a parameter's : form is read to.
	maxSubParams = 8
)

// InterpretOptions bounds Interpret. The zero value takes the defaults.
type InterpretOptions struct {
	Method  ansi.Method // how a cell is counted; the zero value is ansi.WcWidth
	MaxRows int         // 0 or less: DefaultInterpretRows
	MaxCols int         // 0 or less: DefaultInterpretCols
}

// InterpretRow is one row of interpreted output.
type InterpretRow struct {
	Styled string // the row with its SGR, reset at its end
	Plain  string // the row's text alone, for copy and search
}

// Interpret replays captured terminal output, such as a tool's, into the
// rows a terminal would show, where safetext would strip it
// (docs/decisions/0009-MADR-streaming-content-engine.md A1, A7 Q13): a
// progress bar rewritten by CR ends as its last state, and colour stays.
//
// It keeps SGR, in its ; and : forms, written again once per change of
// style; it moves the cursor by CR, BS, a tab to the next multiple of 8,
// LF to the next row's start, and CSI A, B, C, D and G; it erases by CSI K
// and J, an erased cell blank and unstyled. Every other sequence, an OSC
// among them, and every other control are dropped, read as safetext's
// grammar reads them, C1 code points included. Text is cleaned by
// safetext's zero Rules, so invalid UTF-8 is one U+FFFD for each maximal
// subpart. A cell is counted by o.Method: a cluster takes every cell its
// width measures, under ansi.WcWidth the sum of its runes' widths, and one
// written over in any cell is blanked whole. Past MaxCols a row is cut,
// and past MaxRows the oldest rows go. Trailing blank cells
// and trailing empty rows are left out.
//
// Each call starts afresh, so the result is a function of raw and o.
func Interpret(raw []byte, o InterpretOptions) []InterpretRow {
	in := &interpreter{method: o.Method, maxRows: o.MaxRows, maxCols: o.MaxCols}
	if in.maxRows <= 0 {
		in.maxRows = DefaultInterpretRows
	}
	if in.maxCols <= 0 {
		in.maxCols = DefaultInterpretCols
	}
	in.run(raw)
	return in.result()
}

// sgrColor is a foreground, background or underline colour: none, one of
// the 16 basic colours, one of the 256 indexed ones, or 24-bit RGB.
type sgrColor struct {
	kind uint8 // colorNone, colorBasic, colorIndex, colorRGB
	v    uint32
}

const (
	colorNone uint8 = iota
	colorBasic
	colorIndex
	colorRGB
)

// The attributes an SGR sets, as bits.
const (
	attrBold uint16 = 1 << iota
	attrFaint
	attrItalic
	attrBlink
	attrReverse
	attrConceal
	attrStrike
	attrOverline
)

// sgrStyle is the SGR state of a cell; its zero value is no style.
type sgrStyle struct {
	attrs      uint16
	underline  uint8 // 0 none; 1 single to 5 dashed, as SGR 4:n numbers them
	fg, bg, ul sgrColor
}

// sequence is the SGR that sets s from no style: SGR 0 and then s's own
// parameters, or a bare reset for the zero style.
func (s sgrStyle) sequence() string {
	if s == (sgrStyle{}) {
		return "\x1b[m"
	}
	var b strings.Builder
	b.WriteString("\x1b[0")
	for _, a := range [...]struct {
		bit  uint16
		code string
	}{
		{attrBold, "1"}, {attrFaint, "2"}, {attrItalic, "3"}, {attrBlink, "5"},
		{attrReverse, "7"}, {attrConceal, "8"}, {attrStrike, "9"}, {attrOverline, "53"},
	} {
		if s.attrs&a.bit != 0 {
			b.WriteString(";" + a.code)
		}
	}
	switch {
	case s.underline == 1:
		b.WriteString(";4")
	case s.underline > 1:
		b.WriteString(";4:" + strconv.Itoa(int(s.underline)))
	}
	writeColor(&b, s.fg, 30, 90, "38")
	writeColor(&b, s.bg, 40, 100, "48")
	writeColor(&b, s.ul, 0, 0, "58")
	b.WriteByte('m')
	return b.String()
}

// writeColor writes c's parameters: a basic colour as its code from base,
// or from bright for the upper eight, where there is a basic form; an
// indexed or RGB one, or a basic one with no basic form, by ext.
func writeColor(b *strings.Builder, c sgrColor, base, bright int, ext string) {
	switch {
	case c.kind == colorBasic && base > 0 && c.v < 8:
		b.WriteString(";" + strconv.Itoa(base+int(c.v)))
	case c.kind == colorBasic && base > 0:
		b.WriteString(";" + strconv.Itoa(bright+int(c.v)-8))
	case c.kind == colorBasic, c.kind == colorIndex:
		b.WriteString(";" + ext + ";5;" + strconv.Itoa(int(c.v)))
	case c.kind == colorRGB:
		b.WriteString(";" + ext + ";2;" + strconv.Itoa(int(c.v>>16)) + ";" +
			strconv.Itoa(int(c.v>>8&0xff)) + ";" + strconv.Itoa(int(c.v&0xff)))
	}
}

// gridCell is one cell of the grid: a grapheme cluster, its width, and its
// style. A cluster w cells wide takes w cells: every cell after its first
// is a continuation, with no text and width 0. Under ansi.WcWidth a
// cluster measures the sum of its runes' widths, so it may take more than
// two (0009-PLAN Step A1.1, D3).
type gridCell struct {
	text  string
	width int
	style sgrStyle
}

// emptyCell is an erased or padding cell.
var emptyCell = gridCell{text: " ", width: 1}

// interpreter is Interpret's state, for one call.
type interpreter struct {
	method           ansi.Method
	maxRows, maxCols int

	grid     [][]gridCell
	row, col int
	style    sgrStyle
	pending  []byte // printable bytes not yet placed
}

// run reads raw, as safetext's grammar reads it: ESC and the C1 code
// points open sequences, the other C0 controls act or are dropped, and the
// rest is text.
func (in *interpreter) run(raw []byte) {
	for i := 0; i < len(raw); {
		b := raw[i]
		switch {
		case b == 0x1b:
			in.flush()
			i = in.escape(raw, i+1)
		case b < 0x20 || b == 0x7f:
			in.flush()
			in.control(b)
			i++
		case b == 0xc2 && i+1 < len(raw) && raw[i+1] >= 0x80 && raw[i+1] <= 0x9f:
			in.flush()
			i = in.c1(raw, i+2, raw[i+1])
		default:
			in.pending = append(in.pending, b)
			i++
		}
	}
	in.flush()
}

// escape reads the sequence an ESC at raw[i-1] opens, and returns where
// the text after it starts.
func (in *interpreter) escape(raw []byte, i int) int {
	if i >= len(raw) {
		return i
	}
	switch b := raw[i]; {
	case b == '[':
		return in.csi(raw, i+1)
	case b == ']' || b == 'P' || b == 'X' || b == '^' || b == '_':
		return skipString(raw, i+1)
	case b == 0x1b:
		return i // a new sequence
	case b >= 0x20 && b <= 0x2f:
		for i < len(raw) && raw[i] >= 0x20 && raw[i] <= 0x2f {
			i++
		}
		if i < len(raw) && raw[i] >= 0x30 && raw[i] <= 0x7e {
			i++
		}
		return i
	default:
		return i + 1 // dropped with the ESC
	}
}

// c1 reads the sequence a C1 code point c opens, as its 7-bit form does,
// and returns where the text after it starts; one that opens none is
// dropped.
func (in *interpreter) c1(raw []byte, i int, c byte) int {
	switch c {
	case 0x9b:
		return in.csi(raw, i)
	case 0x9d, 0x90, 0x98, 0x9e, 0x9f:
		return skipString(raw, i)
	}
	return i
}

// skipString passes over an OSC, DCS, SOS, PM or APC string, to its BEL or
// ST, or to where CAN or SUB aborts it; one over safetext.DefaultMaxString
// bytes is abandoned there, and the bytes after are text.
func skipString(raw []byte, i int) int {
	start := i
	for i < len(raw) {
		switch b := raw[i]; {
		case i-start > safetext.DefaultMaxString:
			return i
		case b == 0x07, b == 0x18, b == 0x1a:
			return i + 1
		case b == 0x1b && i+1 < len(raw) && raw[i+1] == '\\':
			return i + 2
		case b == 0x1b:
			return i // a new sequence
		case b == 0xc2 && i+1 < len(raw) && raw[i+1] == 0x9c:
			return i + 2
		}
		i++
	}
	return i
}

// csi reads a control sequence's parameters, intermediates and final byte
// from raw[i:], acts on the ones Interpret keeps, and returns where the
// text after it starts. ESC starts a new sequence, CAN and SUB abort it, a
// C0 control inside it acts as it would outside, and a byte that cannot be
// in it ends it unread.
func (in *interpreter) csi(raw []byte, i int) int {
	start := i
	for i < len(raw) && raw[i] >= 0x30 && raw[i] <= 0x3f {
		i++
	}
	params := raw[start:i]
	inter := false
	for i < len(raw) {
		switch b := raw[i]; {
		case b >= 0x20 && b <= 0x2f:
			inter = true
		case b >= 0x40 && b <= 0x7e:
			if !inter {
				in.dispatch(params, b)
			}
			return i + 1
		case b == 0x1b:
			return i
		case b == 0x18 || b == 0x1a:
			return i + 1
		case b < 0x20:
			in.control(b)
		default:
			return i
		}
		i++
	}
	return i
}

// control acts on a C0 control: LF to the next row's start, CR to column
// 0, BS a cell back, HT to the next multiple of 8. Every other one is
// dropped.
func (in *interpreter) control(b byte) {
	switch b {
	case '\n':
		in.moveRow(in.row + 1)
		in.col = 0
	case '\r':
		in.col = 0
	case '\b':
		in.col = max(in.col-1, 0)
	case '\t':
		in.col = min((in.col/8+1)*8, in.maxCols)
	}
}

// dispatch acts on a control sequence with parameters params and final
// byte f, of those Interpret keeps; a private one, whose parameters begin
// with <, =, > or ?, is dropped.
func (in *interpreter) dispatch(params []byte, f byte) {
	if len(params) > 0 && params[0] >= '<' {
		return
	}
	groups, ok := parseParams(params)
	if !ok {
		return
	}
	count := 1 // the first parameter, as a count: none or 0 is 1
	if len(groups) > 0 && groups[0][0] > 0 {
		count = groups[0][0]
	}
	switch f {
	case 'm':
		if len(groups) <= maxSGRParams {
			in.sgr(groups)
		}
	case 'K':
		in.eraseLine(firstParam(groups))
	case 'J':
		in.eraseScreen(firstParam(groups))
	case 'A':
		in.moveRow(max(in.row-count, 0))
	case 'B':
		in.moveRow(in.row + count)
	case 'C':
		in.col = min(in.col+count, in.maxCols)
	case 'D':
		in.col = max(in.col-count, 0)
	case 'G':
		in.col = min(count-1, in.maxCols)
	}
}

// firstParam is the first parameter, or 0.
func firstParam(groups [][]int) int {
	if len(groups) == 0 || groups[0][0] < 0 {
		return 0
	}
	return groups[0][0]
}

// parseParams splits a sequence's parameters at ; into groups, and each
// group at : into its parts, at most maxSubParams, each a number, or -1
// when empty. A number over 65535 is 65535. It stops past maxSGRParams+1
// groups, more than any sequence it keeps reads, and reports false when a
// byte is neither a digit nor a separator.
func parseParams(p []byte) ([][]int, bool) {
	if len(p) == 0 {
		return nil, true
	}
	groups := [][]int{{-1}}
	for _, b := range p {
		g := groups[len(groups)-1]
		switch {
		case b == ';':
			if len(groups) > maxSGRParams {
				return append(groups, []int{-1}), true
			}
			groups = append(groups, []int{-1})
		case b == ':':
			if len(g) < maxSubParams {
				groups[len(groups)-1] = append(g, -1)
			}
		case b >= '0' && b <= '9':
			v := &g[len(g)-1]
			*v = min(max(*v, 0)*10+int(b-'0'), 65535)
		default:
			return nil, false
		}
	}
	return groups, true
}

// attrCodes is what each SGR code that sets or clears attributes does.
var attrCodes = map[int]struct{ set, clear uint16 }{
	1: {set: attrBold}, 2: {set: attrFaint}, 3: {set: attrItalic},
	5: {set: attrBlink}, 6: {set: attrBlink}, 7: {set: attrReverse},
	8: {set: attrConceal}, 9: {set: attrStrike}, 53: {set: attrOverline},
	22: {clear: attrBold | attrFaint}, 23: {clear: attrItalic}, 25: {clear: attrBlink},
	27: {clear: attrReverse}, 28: {clear: attrConceal}, 29: {clear: attrStrike},
	55: {clear: attrOverline},
}

// underlines is SGR 4:n's style for each n it reads.
var underlines = [...]uint8{0, 1, 2, 3, 4, 5}

// sgr applies an SGR's parameter groups to the current style.
func (in *interpreter) sgr(groups [][]int) {
	if len(groups) == 0 {
		in.style = sgrStyle{}
		return
	}
	s := &in.style
	for i := 0; i < len(groups); i++ {
		g := groups[i]
		code := max(g[0], 0)
		if a, ok := attrCodes[code]; ok {
			s.attrs = s.attrs&^a.clear | a.set
			continue
		}
		switch {
		case code == 0:
			*s = sgrStyle{}
		case code == 4:
			s.underline = 1
			if len(g) > 1 && g[1] >= 0 && g[1] < len(underlines) {
				s.underline = underlines[g[1]]
			}
		case code == 21:
			s.underline = 2
		case code == 24:
			s.underline = 0
		case code >= 30 && code <= 37:
			s.fg = sgrColor{colorBasic, uint32(code - 30)}
		case code == 39:
			s.fg = sgrColor{}
		case code >= 40 && code <= 47:
			s.bg = sgrColor{colorBasic, uint32(code - 40)}
		case code == 49:
			s.bg = sgrColor{}
		case code == 59:
			s.ul = sgrColor{}
		case code >= 90 && code <= 97:
			s.fg = sgrColor{colorBasic, uint32(code - 90 + 8)}
		case code >= 100 && code <= 107:
			s.bg = sgrColor{colorBasic, uint32(code - 100 + 8)}
		case code == 38 || code == 48 || code == 58:
			var c sgrColor
			var ok bool
			if len(g) > 1 {
				c, ok = extColor(g[1:])
			} else {
				var used int
				c, used, ok = extColorSemi(groups[i+1:])
				i += used
			}
			if !ok {
				continue
			}
			switch code {
			case 38:
				s.fg = c
			case 48:
				s.bg = c
			default:
				s.ul = c
			}
		}
	}
}

// extColor reads an extended colour in its : form, the parts after 38, 48
// or 58: 5 and an index, or 2, an optional colour space, and R, G and B.
func extColor(p []int) (sgrColor, bool) {
	switch {
	case len(p) >= 2 && p[0] == 5:
		return indexColor(p[1])
	case len(p) >= 5 && p[0] == 2:
		return rgbColor(p[2], p[3], p[4])
	case len(p) == 4 && p[0] == 2:
		return rgbColor(p[1], p[2], p[3])
	}
	return sgrColor{}, false
}

// extColorSemi reads an extended colour in its ; form from the groups
// after 38, 48 or 58, and how many groups it took.
func extColorSemi(groups [][]int) (sgrColor, int, bool) {
	at := func(i int) int {
		if i < len(groups) {
			return groups[i][0]
		}
		return -1
	}
	switch at(0) {
	case 5:
		c, ok := indexColor(at(1))
		return c, min(2, len(groups)), ok
	case 2:
		c, ok := rgbColor(at(1), at(2), at(3))
		return c, min(4, len(groups)), ok
	}
	return sgrColor{}, min(1, len(groups)), false
}

// indexColor is the indexed colour v, when v is one.
func indexColor(v int) (sgrColor, bool) {
	if v < 0 || v > 255 {
		return sgrColor{}, false
	}
	return sgrColor{colorIndex, uint32(v)}, true
}

// rgbColor is the 24-bit colour of r, g and b, an empty part read as 0.
func rgbColor(r, g, b int) (sgrColor, bool) {
	if r > 255 || g > 255 || b > 255 {
		return sgrColor{}, false
	}
	return sgrColor{colorRGB, uint32(max(r, 0))<<16 | uint32(max(g, 0))<<8 | uint32(max(b, 0))}, true
}

// flush places the pending printable bytes, cleaned, a grapheme cluster at
// a time.
func (in *interpreter) flush() {
	if len(in.pending) == 0 {
		return
	}
	s := safetext.Clean(string(in.pending), safetext.Rules{})
	in.pending = in.pending[:0]
	var state byte
	for s != "" {
		cluster, w, n, next := in.method.DecodeSequenceInString(s, state, nil)
		if n <= 0 {
			return
		}
		in.place(cluster, w)
		s, state = s[n:], next
	}
}

// place writes a cluster w cells wide at the cursor, and moves the cursor
// past it. A cluster of no width joins the cluster before it; one that
// does not fit before MaxCols is cut.
func (in *interpreter) place(cluster string, w int) {
	in.moveRow(in.row)
	line := in.grid[in.row]
	if w == 0 {
		if p := min(in.col, len(line)) - 1; p >= 0 {
			line[head(line, p)].text += cluster
		}
		return
	}
	if in.col+w > in.maxCols {
		in.col = in.maxCols
		return
	}
	for len(line) < in.col+w {
		line = append(line, emptyCell)
	}
	for i := in.col; i < in.col+w; i++ {
		clearCluster(line, i)
	}
	line[in.col] = gridCell{text: cluster, width: w, style: in.style}
	for i := in.col + 1; i < in.col+w; i++ {
		line[i] = gridCell{style: in.style}
	}
	in.grid[in.row] = line
	in.col += w
}

// head is the first cell of the cluster that takes cell i.
func head(line []gridCell, i int) int {
	for i > 0 && line[i].width == 0 && line[i].text == "" {
		i--
	}
	return i
}

// clearCluster blanks every cell of the cluster that takes cell i, before
// a cell of it is written over or erased.
func clearCluster(line []gridCell, i int) {
	h := head(line, i)
	if h == i && line[i].width <= 1 {
		return
	}
	end := min(h+max(line[h].width, 1), len(line))
	for j := h; j < end; j++ {
		line[j] = emptyCell
	}
}

// moveRow puts the cursor on row r, adding rows to reach it. Past twice
// MaxRows, the grid keeps its newest MaxRows rows, so dropping the oldest
// costs a copy for every MaxRows rows; result gives the newest MaxRows.
func (in *interpreter) moveRow(r int) {
	for len(in.grid) <= r {
		in.grid = append(in.grid, nil)
	}
	if over := len(in.grid) - in.maxRows; over > in.maxRows {
		in.grid = append(in.grid[:0], in.grid[over:]...)
		r -= over
	}
	in.row = r
}

// eraseLine is CSI K: 0 from the cursor to the row's end, 1 from its start
// to the cursor, 2 all of it.
func (in *interpreter) eraseLine(mode int) {
	in.moveRow(in.row)
	line := in.grid[in.row]
	switch mode {
	case 0:
		if in.col < len(line) {
			clearCluster(line, in.col)
			line = line[:in.col]
		}
	case 1:
		end := min(in.col, len(line)-1)
		if end >= 0 {
			clearCluster(line, end)
		}
		for i := 0; i <= end; i++ {
			line[i] = emptyCell
		}
	case 2:
		line = nil
	}
	in.grid[in.row] = line
}

// eraseScreen is CSI J: 0 from the cursor to the end, 1 from the start to
// the cursor, 2 all of it; the rows above the cursor stay, empty.
func (in *interpreter) eraseScreen(mode int) {
	in.moveRow(in.row)
	switch mode {
	case 0:
		in.eraseLine(0)
		in.grid = in.grid[:in.row+1]
	case 1:
		for r := range in.row {
			in.grid[r] = nil
		}
		in.eraseLine(1)
	case 2:
		for r := range in.grid {
			in.grid[r] = nil
		}
	}
}

// result is the grid as rows, each without its trailing blank cells, with
// the trailing empty rows left out, and then the newest MaxRows of them.
func (in *interpreter) result() []InterpretRow {
	rows := make([]InterpretRow, 0, len(in.grid))
	var styled, plain strings.Builder
	for _, line := range in.grid {
		end := len(line)
		for end > 0 && line[end-1] == emptyCell {
			end--
		}
		styled.Reset()
		plain.Reset()
		var cur sgrStyle
		for _, c := range line[:end] {
			if c.width == 0 && c.text == "" {
				continue // a wide cluster's second cell
			}
			if c.style != cur {
				styled.WriteString(c.style.sequence())
				cur = c.style
			}
			styled.WriteString(c.text)
			plain.WriteString(c.text)
		}
		if cur != (sgrStyle{}) {
			styled.WriteString("\x1b[m")
		}
		rows = append(rows, InterpretRow{Styled: styled.String(), Plain: plain.String()})
	}
	for len(rows) > 0 && rows[len(rows)-1] == (InterpretRow{}) {
		rows = rows[:len(rows)-1]
	}
	if len(rows) > in.maxRows {
		rows = rows[len(rows)-in.maxRows:]
	}
	return rows
}
