package transcript

import (
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// Transcript is what a line-mode CLI has shown, kept to be printed again at
// another width (docs/decisions/0023-MADR-line-mode-components.md D9). It
// does no I/O and is used on one goroutine.
type Transcript struct {
	maxBytes       int
	maxRows        int
	settle         time.Duration
	method         ansi.Method
	keepScrollback bool
	sync           bool
	rules          safetext.Rules
	catalog        msgcat.Catalog
	notice         stream.Style

	pieces  []piece
	dropped bool // pieces went, past the bound

	width    int
	settling bool      // a resize waits to settle
	due      time.Time // when it has
	pending  bool      // it has, and the reprint waits for the host's idle
}

// piece is one thing shown: text as written, a function of the width, or a
// reply.
type piece struct {
	text  string
	drawn func(width int) string
	reply *stream.Printer
	size  int // text's and drawn's; a reply's is its source's, as it grows
}

// NewTranscript is a Transcript with nothing kept.
func NewTranscript(o ...Option) *Transcript {
	t := &Transcript{
		maxBytes: DefaultMaxBytes, maxRows: DefaultMaxRows, settle: DefaultSettle,
		sync: true, rules: safetext.Text(),
	}
	for _, opt := range o {
		opt.apply(t)
	}
	return t
}

// floor is what AddText keeps at the least: safetext's Text rules.
var floor = safetext.Text()

// AddText keeps s as it was written, cleaned by the rules and then the
// floor: output whose layout the program does not make again at another
// width.
func (t *Transcript) AddText(s string) {
	if s = safetext.Clean(safetext.Clean(s, t.rules), floor); s != "" {
		t.add(piece{text: s, size: len(s)})
	}
}

// AddDrawn keeps f, which draws a piece at a width, such as a help listing
// or a log record, and is called again at each reprint's width. Its text is
// the host's own drawing and is written as drawn. Its size, for the bound,
// is what it draws at the width it is added at.
func (t *Transcript) AddDrawn(f func(width int) string) {
	if f != nil {
		t.add(piece{drawn: f, size: len(f(t.width))})
	}
}

// AddReply keeps p, a reply, drawn again from its cleaned source by p.At at
// each reprint's width. Its size is its source's, as the reply grows.
func (t *Transcript) AddReply(p *stream.Printer) {
	if p != nil {
		t.add(piece{reply: p})
	}
}

func (t *Transcript) add(p piece) {
	t.pieces = append(t.pieces, p)
	t.trim()
}

// trim drops the oldest pieces while the rest are more than maxBytes, the
// newest always kept.
func (t *Transcript) trim() {
	total := 0
	for i := range t.pieces {
		total += t.pieces[i].bytes()
	}
	n := 0
	for ; total > t.maxBytes && n < len(t.pieces)-1; n++ {
		total -= t.pieces[n].bytes()
	}
	if n > 0 {
		t.pieces = slices.Delete(t.pieces, 0, n)
		t.dropped = true
	}
}

func (p *piece) bytes() int {
	if p.reply != nil {
		return len(p.reply.Source())
	}
	return p.size
}

func (p *piece) at(width int) string {
	switch {
	case p.reply != nil:
		return p.reply.At(width)
	case p.drawn != nil:
		return p.drawn(width)
	}
	return p.text
}

// Resize gives the terminal's width, in cells. The first sets it; a later
// one with another width, or any during a settle, starts the settle again,
// the reprint due WithSettle after it. A width of 0 or less changes
// nothing.
func (t *Transcript) Resize(width int, now time.Time) {
	switch {
	case width <= 0:
		return
	case t.width == 0:
		t.width = width
		return
	case width == t.width && !t.settling:
		return
	}
	t.width, t.settling, t.due = width, true, now.Add(t.settle)
}

// Timeout is when HandleTimeout is next due: the settle's end while a
// resize waits for it, or nothing.
func (t *Transcript) Timeout() (time.Time, bool) {
	if t.settling {
		return t.due, true
	}
	return time.Time{}, false
}

// HandleTimeout gives the reprint once the last resize has settled and the
// host's output is idle, at a line's start with nothing held; otherwise "".
// A reprint still due after the settle waits for the next call with idle
// true, so the host calls it whenever its output becomes idle
// (docs/decisions/0023-MADR-line-mode-components.md A8 item 4).
func (t *Transcript) HandleTimeout(now time.Time, idle bool) string {
	if t.settling && !now.Before(t.due) {
		t.settling, t.pending = false, true
	}
	if !t.pending || !idle {
		return ""
	}
	t.pending = false
	return t.reprint()
}

// reprint is SGR 0, then the clear and the pieces at the width, as one
// synchronized update, after the notice when anything was left out
// (D9, A3 item 15).
func (t *Transcript) reprint() string {
	t.trim()
	text, cut := t.render(t.width)
	var b strings.Builder
	b.WriteString("\x1b[0m")
	if t.sync {
		b.WriteString("\x1b[?2026h")
	}
	b.WriteString("\x1b[2J")
	if !t.keepScrollback {
		b.WriteString("\x1b[3J")
	}
	b.WriteString("\x1b[H")
	if cut {
		msg := msgcat.Format(t.catalog, MsgEarlierOutput, Messages()[MsgEarlierOutput])
		b.WriteString(t.notice.Apply(safetext.Clean(safetext.Clean(msg, t.rules), safetext.Line())))
		b.WriteString("\r\n")
	}
	b.WriteString(crlf(text))
	if t.sync {
		b.WriteString("\x1b[?2026l")
	}
	return b.String()
}

// render is the pieces drawn at width: the newest back to at most maxRows
// rows, the oldest of them cut by its first lines to fit, and whether
// anything earlier was left out, by the bound or by the rows. Only the
// pieces it shows are drawn.
func (t *Transcript) render(width int) (string, bool) {
	cut := t.dropped
	var drawn []string
	rows := 0
	for i := len(t.pieces) - 1; i >= 0; i-- {
		if rows >= t.maxRows {
			cut = true
			break
		}
		s := t.pieces[i].at(width)
		lines := strings.SplitAfter(s, "\n")
		n := 0
		for j, line := range slices.Backward(lines) {
			r := t.rowsOf(line, width)
			if rows+n+r > t.maxRows {
				s, cut = strings.Join(lines[j+1:], ""), true
				break
			}
			n += r
		}
		drawn = append(drawn, s)
		rows += n
	}
	var b strings.Builder
	for _, s := range slices.Backward(drawn) {
		b.WriteString(s)
	}
	return b.String(), cut
}

// rowsOf is the rows a line takes at width cells: one, or more when the
// terminal wraps it; an empty last piece of a split takes none.
func (t *Transcript) rowsOf(line string, width int) int {
	if line == "" {
		return 0
	}
	w := t.method.StringWidth(strings.TrimSuffix(line, "\n"))
	if width <= 0 || w <= width {
		return 1
	}
	return (w + width - 1) / width
}

// crlf is s with each line break written as CR LF, so it reads alike when
// the terminal is in raw mode.
func crlf(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\n", "\r\n")
}
