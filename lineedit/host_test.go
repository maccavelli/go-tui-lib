package lineedit

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// host is a line-mode host on a test screen: it resizes the Composer to
// the screen, decodes what is typed and what the screen answers, feeds it
// in, and writes what the Composer asks.
type host struct {
	t       *testing.T
	scr     *inlinetest.Screen
	c       *Composer
	dec     termkey.Decoder
	now     time.Time
	replies []byte
	written []byte          // every byte the Composer asked to write
	events  []ComposerEvent // every event but the writes
}

func newHost(t *testing.T, columns int, o ...Option) *host {
	t.Helper()
	h := &host{t: t, scr: inlinetest.NewScreen(columns, 30, ansi.WcWidth), now: time.Unix(1, 0)}
	h.scr.Answer = func(reply string) { h.replies = append(h.replies, reply...) }
	h.c = NewComposer(append([]Option{WithGlyphs(glyph.ASCII())}, o...)...)
	h.c.Resize(columns, h.now)
	h.drain()
	return h
}

func (h *host) drain() {
	for e := h.c.NextEvent(); e.Kind != EditNone; e = h.c.NextEvent() {
		if e.Kind == EditWrite {
			h.written = append(h.written, e.Data...)
			_, _ = h.scr.Write([]byte(e.Data))
			continue
		}
		h.events = append(h.events, e)
	}
}

// typeIn decodes b and feeds each input, then what the screen answered.
func (h *host) typeIn(b string) {
	for _, in := range h.dec.Flush(h.dec.Decode([]byte(b), nil)) {
		h.c.HandleInput(in, h.now)
		h.drain()
	}
	if len(h.replies) > 0 {
		r := h.replies
		h.replies = nil
		h.typeIn(string(r))
	}
}

// kinds is the events' kinds, in order.
func (h *host) kinds() []ComposerEventKind {
	var out []ComposerEventKind
	for _, e := range h.events {
		out = append(out, e.Kind)
	}
	return out
}

// submitted is the texts submitted, in order.
func (h *host) submitted() []string {
	out := []string{}
	for _, e := range h.events {
		if e.Kind == EditSubmit {
			out = append(out, e.Text)
		}
	}
	return out
}

// eof reports whether the input ended.
func (h *host) eof() bool {
	for _, e := range h.events {
		if e.Kind == EditEOF {
			return true
		}
	}
	return false
}

// pos is the cursor, in runes from the text's start.
func (h *host) pos() int {
	return utf8.RuneCountInString(strings.Join(h.c.line.Clusters()[:h.c.line.Pos()], ""))
}

// at is the text with the cursor shown as |.
func (h *host) at() string {
	cl := h.c.line.Clusters()
	return strings.Join(cl[:h.c.line.Pos()], "") + "|" + strings.Join(cl[h.c.line.Pos():], "")
}

// fixedHistory is a History that Add leaves as it is, as gobble's parity
// test keeps x/term's and its own alike.
type fixedHistory []string

func (fixedHistory) Add(string)          {}
func (h fixedHistory) Len() int          { return len(h) }
func (h fixedHistory) At(idx int) string { return h[idx] }
