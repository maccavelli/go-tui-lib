package ask_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/ask"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// host is a line-mode host on a test screen: it resizes the Asker to the
// screen, decodes what is typed and what the screen answers, feeds it in,
// and writes what the Asker asks, until it is done.
type host struct {
	t       *testing.T
	scr     *inlinetest.Screen
	a       *ask.Asker
	dec     termkey.Decoder
	now     time.Time
	replies []byte
	resp    *ask.Response
	dones   int
}

func newHost(t *testing.T, columns int, q ask.Question, o ...ask.Option) *host {
	t.Helper()
	h := &host{t: t, scr: inlinetest.NewScreen(columns, 40, ansi.WcWidth), now: time.Unix(1, 0)}
	h.scr.Answer = func(reply string) { h.replies = append(h.replies, reply...) }
	h.a = ask.NewAsker(q, append([]ask.Option{ask.WithGlyphs(glyph.ASCII())}, o...)...)
	h.a.Resize(columns, h.now)
	h.drain()
	return h
}

func (h *host) drain() {
	for e := h.a.NextEvent(); e.Kind != ask.AskNone; e = h.a.NextEvent() {
		switch e.Kind {
		case ask.AskWrite:
			_, _ = h.scr.Write([]byte(e.Data))
		case ask.AskDone:
			r := e.Response
			h.resp = &r
			h.dones++
		}
	}
}

// typeIn decodes b and feeds each input, then what the screen answered.
func (h *host) typeIn(b string) {
	for _, in := range h.dec.Flush(h.dec.Decode([]byte(b), nil)) {
		h.a.HandleInput(in, h.now)
		h.drain()
	}
	if len(h.replies) > 0 {
		r := h.replies
		h.replies = nil
		h.typeIn(string(r))
	}
}

// last is the last row of the screen with text on it.
func (h *host) last() string {
	lines := h.scr.Lines()
	for _, line := range slices.Backward(lines) {
		if line != "" {
			return line
		}
	}
	return ""
}

func yesNo() ask.Question {
	return ask.Question{
		Title:  "Go on?",
		Rows:   []ask.Row{{Answers: []ask.Answer{{ID: "yes", Key: "y", Label: "yes", Echo: "going on"}, {ID: "no", Key: "n", Label: "no"}}}},
		Cancel: ask.Answer{ID: "cancel", Key: "c", Label: "cancel"},
	}
}

// TestAnswer: an answer's key answers at once, the question and the echo
// left in the scrollback, the cursor below them.
func TestAnswer(t *testing.T) {
	h := newHost(t, 40, yesNo())
	if got := h.scr.Text(); got != "Go on?\n  [y] yes  [n] no   [c] cancel\n>\n" {
		t.Fatalf("drawn: %q", got)
	}
	h.typeIn("y")
	if h.resp == nil || h.resp.Answer.ID != "yes" || h.resp.Canceled {
		t.Fatalf("response %+v; want yes", h.resp)
	}
	if got := h.scr.Text(); got != "Go on?\n  [y] yes  [n] no   [c] cancel\n> going on\n" {
		t.Errorf("left: %q; want the echo after the prompt", got)
	}
	if h.scr.Row != 3 || h.scr.Col != 0 {
		t.Errorf("cursor at %d,%d; want 3,0, below the question", h.scr.Row, h.scr.Col)
	}
	h.typeIn("n")
	h.a.Cancel()
	h.drain()
	if h.dones != 1 {
		t.Errorf("%d done events; want one, then nothing", h.dones)
	}
}

// TestEchoLabel: an answer with no Echo is left as its label.
func TestEchoLabel(t *testing.T) {
	h := newHost(t, 40, yesNo())
	h.typeIn("n")
	if got := h.scr.Lines()[2]; got != "> no" {
		t.Errorf("left %q; want the label", got)
	}
}

// TestUnknownKey: a key no answer shows, enter, ctrl+x, a release and a
// paste are ignored.
func TestUnknownKey(t *testing.T) {
	h := newHost(t, 40, yesNo())
	before := h.scr.Text()
	h.typeIn("x\r\x18\x1b[200~y\x1b[201~Y")
	if h.resp != nil {
		t.Fatalf("answered %+v by keys no answer shows", h.resp)
	}
	if got := h.scr.Text(); got != before {
		t.Errorf("the screen changed: %q", got)
	}
}

// TestCancel: the cancel key, ctrl+c and the host's Cancel each cancel,
// the cancel answer's label left.
func TestCancel(t *testing.T) {
	for name, act := range map[string]func(h *host){
		"the cancel key": func(h *host) { h.typeIn("c") },
		"ctrl+c":         func(h *host) { h.typeIn("\x03") },
		"Cancel":         func(h *host) { h.a.Cancel(); h.drain() },
		"ctrl+c in a reason": func(h *host) {
			h.typeIn("f")
			h.typeIn("abc\x03")
		},
	} {
		q := yesNo()
		q.Rows = append(q.Rows, ask.Row{Answers: []ask.Answer{{ID: "why", Key: "f", Label: "say why", Reason: true}}})
		h := newHost(t, 40, q)
		act(h)
		if h.resp == nil || !h.resp.Canceled || h.resp.Answer.ID != "cancel" {
			t.Errorf("%s: response %+v; want canceled", name, h.resp)
			continue
		}
		if got := h.last(); got != "> cancel" {
			t.Errorf("%s: left %q", name, got)
		}
	}
	q := yesNo()
	q.Cancel = ask.Answer{}
	h := newHost(t, 40, q)
	h.typeIn("c")
	if h.resp != nil {
		t.Errorf("with no cancel answer, c answered %+v", h.resp)
	}
	h.a.Cancel()
	h.drain()
	if h.resp == nil || !h.resp.Canceled || h.last() != "> cancel" {
		t.Errorf("Cancel with no cancel answer: %+v, %q; want canceled, the catalogue's word", h.resp, h.last())
	}
}

// TestShiftKey: an answer keyed by the text it types, "N", or by its name,
// "shift+n", answers to shift+n; plain n does not.
func TestShiftKey(t *testing.T) {
	for _, key := range []string{"N", "shift+n"} {
		q := ask.Question{Title: "Run?", Rows: []ask.Row{{Answers: []ask.Answer{{ID: "never", Key: key, Label: "never"}}}}}
		h := newHost(t, 40, q)
		h.typeIn("n")
		if h.resp != nil {
			t.Errorf("key %q: plain n answered", key)
		}
		h.a.HandleInput(termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: 'n', Mod: termkey.ModShift, Text: "N"}}, h.now)
		h.drain()
		if h.resp == nil || h.resp.Answer.ID != "never" {
			t.Errorf("key %q: shift+n gives %+v", key, h.resp)
		}
	}
}

// reasonQuestion is gobble's "reject and say why".
func reasonQuestion() ask.Question {
	return ask.Question{
		Title:  "Run make?",
		Rows:   []ask.Row{{Answers: []ask.Answer{{ID: "y", Key: "y", Label: "allow"}, {ID: "why", Key: "f", Label: "reject and say why", Echo: "reject", Reason: true}}}},
		Cancel: ask.Answer{ID: "cancel", Key: "c", Label: "cancel"},
	}
}

// TestReason: an answer with Reason reads a line, echoed as it is typed,
// edited by its keys, a paste into it, and enter gives it trimmed, left
// after the answer's echo.
func TestReason(t *testing.T) {
	h := newHost(t, 60, reasonQuestion())
	h.typeIn("f")
	if h.resp != nil {
		t.Fatalf("answered before the reason: %+v", h.resp)
	}
	h.typeIn("not  now")
	if got := h.scr.Lines()[2]; got != "> reject and say why: not  now" {
		t.Errorf("while typed: %q", got)
	}
	h.typeIn("\x7f\x7f\x7f\x1b[200~tomorrow\x1b[201~ ")
	h.typeIn("\r")
	if h.resp == nil || h.resp.Answer.ID != "why" || h.resp.Reason != "not  tomorrow" || h.resp.Canceled {
		t.Fatalf("response %+v; want why, \"not  tomorrow\"", h.resp)
	}
	if got := h.scr.Lines()[2]; got != "> reject: not tomorrow" {
		t.Errorf("left %q; want the echo and the reason", got)
	}
}

// TestReasonLimit: a reason is bounded at DefaultReasonLimit bytes, or
// WithReasonLimit's, a key past it dropped.
func TestReasonLimit(t *testing.T) {
	for _, c := range []struct {
		o    []ask.Option
		want int
	}{
		{nil, ask.DefaultReasonLimit},
		{[]ask.Option{ask.WithReasonLimit(0)}, ask.DefaultReasonLimit},
		{[]ask.Option{ask.WithReasonLimit(5)}, 5},
	} {
		h := newHost(t, 60, reasonQuestion(), c.o...)
		h.typeIn("f")
		h.typeIn(strings.Repeat("x", 3000))
		h.typeIn("\r")
		if h.resp == nil || len(h.resp.Reason) != c.want {
			t.Errorf("%v: reason of %d bytes; want %d", c.o, len(h.resp.Reason), c.want)
		}
	}
}

// TestReasonView: a reason longer than its row shows the part around the
// cursor, the row within the width.
func TestReasonView(t *testing.T) {
	h := newHost(t, 40, reasonQuestion())
	h.typeIn("f" + strings.Repeat("abcdefghij", 5))
	rows, row, col := h.a.View(40)
	if w := ansi.WcWidth.StringWidth(rows[row]); w > 40 || col >= 40 {
		t.Errorf("row %q is %d cells, the cursor at %d, in 40", rows[row], w, col)
	}
	if !strings.HasSuffix(rows[row], "hij") || h.scr.Col != col {
		t.Errorf("row %q, cursor %d on the screen at %d; want the newest text and the cursor after it", rows[row], col, h.scr.Col)
	}
}

// TestView: View is what is drawn, the cursor after the prompt, and does
// not change the Asker.
func TestView(t *testing.T) {
	h := newHost(t, 40, yesNo())
	rows, row, col := h.a.View(40)
	if strings.Join(rows, "\n")+"\n" != h.scr.Text() || row != 2 || col != 2 || h.scr.Row != 2 || h.scr.Col != 2 {
		t.Errorf("View = %q, %d, %d; the screen %q, the cursor at %d,%d", rows, row, col, h.scr.Text(), h.scr.Row, h.scr.Col)
	}
	if again, _, _ := h.a.View(40); strings.Join(again, "\n") != strings.Join(rows, "\n") || h.a.NextEvent().Kind != ask.AskNone {
		t.Error("View changed the Asker")
	}
	if narrow, _, _ := h.a.View(20); len(narrow) <= len(rows) {
		t.Errorf("at 20 columns: %q; want it wrapped", narrow)
	}
}

// TestResize: a resize asks where the cursor is, and redraws at the new
// width once the report comes.
func TestResize(t *testing.T) {
	h := newHost(t, 40, yesNo())
	h.scr.Resize(20, inline.ReflowDown)
	h.a.Resize(20, h.now)
	h.drain()
	if len(h.replies) == 0 {
		t.Fatal("the resize asked nothing of the screen")
	}
	h.typeIn("")
	want := ask.Layout(yesNo(), 20, ansi.WcWidth, glyph.ASCII())
	if got := h.scr.Lines()[:len(want)]; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("after the resize: %q; want %q", got, want)
	}
}

// TestWithRegion: the Asker draws in the host's region, with its options.
func TestWithRegion(t *testing.T) {
	r := inline.NewRegion(inline.WithoutCursorQuery())
	h := newHost(t, 40, yesNo(), ask.WithRegion(r))
	h.scr.Resize(30, inline.ReflowDown)
	h.a.Resize(30, h.now)
	h.drain()
	if len(h.replies) != 0 {
		t.Errorf("the host's region, made WithoutCursorQuery, asked for the cursor: %q", h.replies)
	}
	want := ask.Layout(yesNo(), 30, ansi.WcWidth, glyph.ASCII())
	if got := h.scr.Lines()[:len(want)]; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("redrawn with no report: %q; want %q", got, want)
	}
}

// TestCatalog: the Asker's own words come from the program's catalogue,
// and its defaults' verbs are the defaults'.
func TestCatalog(t *testing.T) {
	cat := msgcat.Table{ask.MsgReasonPrompt: "pourquoi (%s) ? ", ask.MsgReasonEcho: "%[2]s — %[1]s"}
	h := newHost(t, 100, reasonQuestion(), ask.WithCatalog(cat))
	h.typeIn("fnon")
	if got := h.scr.Lines()[2]; got != "> pourquoi (reject and say why) ? non" {
		t.Errorf("while typed: %q", got)
	}
	h.typeIn("\r")
	if got := h.scr.Lines()[2]; got != "> non — reject" {
		t.Errorf("left %q", got)
	}
	if err := msgcat.Check(ask.Messages(), cat); err != nil {
		t.Errorf("Check: %v", err)
	}
}

// TestRules: WithRules cleans more than the default, and never less than
// safetext's Line rules.
func TestRules(t *testing.T) {
	q := ask.Question{Title: "a\nb\x1b[1mc?"}
	for _, c := range []struct {
		name string
		o    []ask.Option
		want string
	}{
		{"default", nil, "a bc?"},
		{"stricter", []ask.Option{ask.WithRules(safetext.Rules{})}, "abc?"},
		{"weaker", []ask.Option{ask.WithRules(safetext.Rules{Newlines: true, KeepSGR: true})}, "a bc?"},
	} {
		h := newHost(t, 40, q, c.o...)
		if got := h.scr.Lines()[0]; got != c.want {
			t.Errorf("%s: %q; want %q", c.name, got, c.want)
		}
	}
	// The lead before a reason, the label in the catalogue's text, is drawn
	// as it is: the floor holds there under weaker rules too.
	r := ask.Question{Title: "Run?", Rows: []ask.Row{{Answers: []ask.Answer{{Key: "f", Label: "say\x1b[31m why\n", Reason: true}}}}}
	cat := msgcat.Table{ask.MsgReasonPrompt: "\x1b]0;pwned\x07%s: "}
	h := newHost(t, 80, r, ask.WithRules(safetext.Rules{Newlines: true, KeepSGR: true}), ask.WithCatalog(cat))
	h.typeIn("f")
	rows, row, _ := h.a.View(80)
	if got := rows[row]; got != "> say why : " || strings.ContainsAny(got, "\x1b\n\a") {
		t.Errorf("the lead under weaker rules: %q; want it cleaned to the floor", got)
	}
}

// TestRow1Report: a resize's cursor report on the first row is taken as
// the report, not as a key.
func TestRow1Report(t *testing.T) {
	q := ask.Question{Title: "Go?", Rows: []ask.Row{{Answers: []ask.Answer{{ID: "f3", Key: "ctrl+f3", Label: "f3"}}}}}
	h := newHost(t, 40, q)
	h.a.Resize(30, h.now)
	h.drain()
	h.replies = nil
	h.typeIn("\x1b[1;5R")
	if h.resp != nil {
		t.Errorf("the awaited report answered as a key: %+v", h.resp)
	}
}
