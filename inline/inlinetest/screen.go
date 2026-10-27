// Package inlinetest is a terminal's screen for tests of what a line-mode
// component draws: the bytes a Region returns are written to a Screen, and
// a test checks the rows and the cursor it ends with, not the bytes
// (docs/decisions/0023-MADR-line-mode-components.md D3). It is gobble-cli's
// termtest.Screen, measured by x/ansi, by grapheme cluster.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package inlinetest

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/inline"
)

// Screen is a terminal's screen, enough of one to check what a component
// draws: a grid of cells, each one grapheme cluster, a wide one taking two
// cells, measured by the screen's width method; \r; a line feed, scrolling
// at the bottom into the scrollback; the cursor moves up, down, forward,
// back and to a place (CSI A, B, C, D, H); erasing below, the whole screen
// and the scrollback (CSI J, 2 J, 3 J), and to the end of the line or the
// whole line (CSI K, 2 K); hiding and showing the cursor (DECTCEM);
// synchronized output (mode 2026); and the cursor report (CSI 6 n), which
// Answer is given. Other sequences, SGR among them, show nothing. A cluster
// written in the last column leaves a wrap pending, as terminals do: the
// next one goes to the next row's start, and \r, a line feed or a move
// cancels it.
//
// A line feed keeps the column, as in raw mode, where a component draws.
// Cooked makes it also return to the row's start, as a terminal's output
// processing does outside raw mode.
type Screen struct {
	// Row and Col are the cursor's, from 0.
	Row, Col int
	// Cooked makes a line feed a new line.
	Cooked bool
	// Hidden is the cursor hidden, by DECTCEM.
	Hidden bool
	// Synchronized is mode 2026 set.
	Synchronized bool
	// Answer, when set, is given the reply to a cursor report request,
	// CSI row ; col R, each from 1.
	Answer func(reply string)

	method        ansi.Method
	width, height int
	cells         [][]string // each cell a cluster; "" is the second cell of a wide one
	pending       bool
	scrolled      []string
	rest          string // the start of a sequence a Write cut short
	parser        *ansi.Parser
}

// NewScreen returns a blank screen of columns by rows that measures with
// m, the cursor at the top left.
func NewScreen(columns, rows int, m ansi.Method) *Screen {
	s := &Screen{method: m, width: max(columns, 1), height: max(rows, 1), parser: ansi.NewParser()}
	for range s.height {
		s.cells = append(s.cells, s.blank())
	}
	return s
}

func (s *Screen) blank() []string {
	r := make([]string, s.width)
	for i := range r {
		r[i] = " "
	}
	return r
}

// Size is the screen's columns and rows.
func (s *Screen) Size() (columns, rows int) { return s.width, s.height }

// Write draws p. A sequence cut short at its end waits for the next Write.
func (s *Screen) Write(p []byte) (int, error) {
	data := s.rest + string(p)
	s.rest = ""
	var state byte
	for data != "" {
		seq, w, n, next := s.method.DecodeSequenceInString(data, state, s.parser)
		if n == 0 {
			break
		}
		if next != 0 && n == len(data) {
			s.rest = data // cut short: wait for the rest
			break
		}
		state = next
		data = data[n:]
		s.apply(seq, w)
	}
	return len(p), nil
}

// apply draws one sequence or cluster.
func (s *Screen) apply(seq string, w int) {
	switch {
	case seq == "\r":
		s.Col, s.pending = 0, false
	case seq == "\n":
		s.pending = false
		if s.Cooked {
			s.Col = 0
		}
		s.down()
	case ansi.HasCsiPrefix(seq):
		s.csi(ansi.Cmd(s.parser.Command()), s.parser.Params())
	case seq[0] == ansi.ESC, seq[0] < 0x20, seq[0] == 0x7f:
	case w > 0:
		s.put(seq, w)
	}
}

// csi applies one control sequence.
func (s *Screen) csi(c ansi.Cmd, ps ansi.Params) {
	// param is the i-th parameter, 1 when it is missing or 0.
	param := func(i int) int {
		if p, _, ok := ps.Param(i, 1); ok && p > 0 {
			return p
		}
		return 1
	}
	if c.Prefix() == '?' {
		on := c.Final() == 'h'
		if c.Final() != 'h' && c.Final() != 'l' {
			return
		}
		for i := range ps {
			switch ps[i].Param(0) {
			case 25:
				s.Hidden = !on
			case 2026:
				s.Synchronized = on
			}
		}
		return
	}
	if c.Prefix() != 0 || c.Intermediate() != 0 {
		return
	}
	n, _, _ := ps.Param(0, 0)
	switch c.Final() {
	case 'A':
		s.pending, s.Row = false, max(s.Row-param(0), 0)
	case 'B':
		s.pending, s.Row = false, min(s.Row+param(0), s.height-1)
	case 'C':
		s.pending, s.Col = false, min(s.Col+param(0), s.width-1)
	case 'D':
		s.pending, s.Col = false, max(s.Col-param(0), 0)
	case 'H', 'f':
		s.pending = false
		s.Row, s.Col = min(param(0), s.height)-1, min(param(1), s.width)-1
	case 'J':
		switch n {
		case 0:
			s.eraseRight()
			for r := s.Row + 1; r < s.height; r++ {
				s.cells[r] = s.blank()
			}
		case 2:
			for r := range s.cells {
				s.cells[r] = s.blank()
			}
		case 3:
			s.scrolled = nil
		}
	case 'K':
		switch n {
		case 0:
			s.eraseRight()
		case 2:
			s.cells[s.Row] = s.blank()
		}
	case 'n':
		if n == 6 && s.Answer != nil {
			s.Answer("\x1b[" + strconv.Itoa(s.Row+1) + ";" + strconv.Itoa(s.Col+1) + "R")
		}
	}
}

// eraseRight blanks the cursor's row from its column, and the first cell
// of a wide cluster the column splits.
func (s *Screen) eraseRight() {
	row := s.cells[s.Row]
	if s.Col > 0 && row[s.Col] == "" {
		row[s.Col-1] = " "
	}
	for c := s.Col; c < s.width; c++ {
		row[c] = " "
	}
}

// down moves the cursor a row down, scrolling the top row into the
// scrollback at the bottom.
func (s *Screen) down() {
	if s.Row == s.height-1 {
		s.scrolled = append(s.scrolled, s.line(s.cells[0]))
		s.cells = append(s.cells[1:], s.blank())
		return
	}
	s.Row++
}

// put writes cluster, w cells wide, at the cursor, taking its pending wrap
// first. A cluster wider than the screen is not drawn.
func (s *Screen) put(cluster string, w int) {
	if w > s.width {
		return
	}
	if s.pending || s.Col+w > s.width {
		s.Col, s.pending = 0, false
		s.down()
	}
	row := s.cells[s.Row]
	if row[s.Col] == "" && s.Col > 0 {
		row[s.Col-1] = " " // the first cell of a wide cluster it splits
	}
	if end := s.Col + w; end < s.width && row[end] == "" {
		row[end] = " " // the second cell of a wide cluster it covers the first of
	}
	row[s.Col] = cluster
	for k := 1; k < w; k++ {
		row[s.Col+k] = ""
	}
	if s.Col+w >= s.width {
		s.Col, s.pending = s.width-1, true
		return
	}
	s.Col += w
}

// cluster is one cell's cluster and its width, for a re-wrap.
type cluster struct {
	s string
	w int
}

// Resize makes the screen columns wide, its rows changed as how says, the
// three ways gobble-cli measured terminals narrow
// (0030-PLAN-resize-and-performance.md P1): ReflowDown re-wraps each row and
// moves the cursor down with its cell; ReflowKeepRow re-wraps each row and
// keeps the cursor on its screen row, pushing the rows above it out of the
// top; ReflowCut keeps each row on one row, cut at the new width.
// ReflowUnknown is taken as ReflowDown. Each row is taken as a line of its
// own, and a wide cluster is kept whole on its row; the cursor moves as it
// would among cells one wide.
func (s *Screen) Resize(columns int, how inline.Reflow) {
	columns = max(columns, 1)
	text := make([][]cluster, len(s.cells))
	for i, row := range s.cells {
		for c := range row {
			if row[c] == "" {
				continue
			}
			w := 1
			if c+1 < len(row) && row[c+1] == "" {
				w = 2
			}
			text[i] = append(text[i], cluster{row[c], w})
		}
		for len(text[i]) > 0 && text[i][len(text[i])-1].s == " " {
			text[i] = text[i][:len(text[i])-1]
		}
	}
	row, col := s.Row, s.Col
	s.width, s.pending = columns, false
	if how == inline.ReflowCut {
		for i := range s.cells {
			s.cells[i] = s.lay(text[i])
		}
		s.Col = min(col, columns-1)
		return
	}
	var rows [][]cluster
	at := 0
	for i, line := range text {
		if i == row {
			at = len(rows) + col/columns
		}
		var cur []cluster
		used := 0
		for _, c := range line {
			if used+c.w > columns {
				rows = append(rows, cur)
				cur, used = nil, 0
			}
			cur = append(cur, c)
			used += c.w
		}
		rows = append(rows, cur)
	}
	start := max(at-(s.height-1), 0)
	if how == inline.ReflowKeepRow {
		start = max(at-row, 0)
	}
	for _, r := range rows[:min(start, len(rows))] {
		s.scrolled = append(s.scrolled, s.line(s.lay(r)))
	}
	for i := range s.cells {
		s.cells[i] = s.blank()
		if start+i < len(rows) {
			s.cells[i] = s.lay(rows[start+i])
		}
	}
	s.Row, s.Col = at-start, col%columns
}

// lay is clusters on a blank row, those past its end left out.
func (s *Screen) lay(cs []cluster) []string {
	row := s.blank()
	c := 0
	for _, cl := range cs {
		if c+cl.w > s.width {
			break
		}
		row[c] = cl.s
		for k := 1; k < cl.w; k++ {
			row[c+k] = ""
		}
		c += cl.w
	}
	return row
}

// line is a row's text, trailing spaces dropped.
func (s *Screen) line(row []string) string {
	return strings.TrimRight(strings.Join(row, ""), " ")
}

// Lines are the screen's rows, trailing spaces dropped.
func (s *Screen) Lines() []string {
	out := make([]string, s.height)
	for i, row := range s.cells {
		out[i] = s.line(row)
	}
	return out
}

// Scrollback is the rows scrolled off the top, oldest first, trailing
// spaces dropped, since the last CSI 3 J.
func (s *Screen) Scrollback() []string { return append([]string(nil), s.scrolled...) }

// Text is the screen's text form, for a golden file: the scrollback, then
// the screen's rows, each ended by a line feed, the blank rows below the
// last written one left out.
func (s *Screen) Text() string {
	all := append(s.Scrollback(), s.Lines()...)
	for len(all) > 0 && all[len(all)-1] == "" {
		all = all[:len(all)-1]
	}
	var b strings.Builder
	for _, l := range all {
		b.WriteString(l + "\n")
	}
	return b.String()
}
