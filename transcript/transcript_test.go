package transcript_test

import (
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/transcript"
)

// The reprint's framing, as D9 and A3 item 15 order it.
const (
	head = "\x1b[0m\x1b[?2026h\x1b[2J\x1b[3J\x1b[H"
	tail = "\x1b[?2026l"
)

var t0 = time.Unix(1, 0)

// reprint resizes t from 80 columns to width, lets the resize settle, and
// gives the reprint with the host idle.
func reprint(t *testing.T, tr *transcript.Transcript, width int) string {
	t.Helper()
	tr.Resize(80, t0)
	if width == 80 {
		tr.Resize(81, t0)
	}
	tr.Resize(width, t0)
	out := tr.HandleTimeout(t0.Add(transcript.DefaultSettle), true)
	if out == "" {
		t.Fatal("no reprint once the resize settled with the host idle")
	}
	return out
}

// body is a default reprint's text, its framing checked and taken off, and
// its CR LF read as LF.
func body(t *testing.T, out string) string {
	t.Helper()
	if !strings.HasPrefix(out, head) || !strings.HasSuffix(out, tail) {
		t.Fatalf("reprint %q: want %q first and %q last", out, head, tail)
	}
	return strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(out, head), tail), "\r\n", "\n")
}

const notice = "(earlier output is not shown again)\n"

// TestRendersAtAWidth: each piece drawn at the reprint's width: text as it
// was written, a reply by its Printer at the width, a function of the width
// by that function (gobble-cli's TestTranscriptRendersAtAWidth).
func TestRendersAtAWidth(t *testing.T) {
	tr := transcript.NewTranscript()
	tr.AddText("tool output that is wider than twenty columns, kept as written\n")
	p := stream.NewPrinter(stream.WithWidth(80))
	p.Write("a reply long enough to be wrapped at twenty columns\n")
	p.Flush()
	tr.AddReply(p)
	tr.AddDrawn(func(width int) string { return strings.Repeat("-", width) + "\n" })
	want := "tool output that is wider than twenty columns, kept as written\n" +
		"a reply long enough\nto be wrapped at\ntwenty columns\n" + strings.Repeat("-", 20) + "\n"
	if got := body(t, reprint(t, tr, 20)); got != want {
		t.Fatalf("at 20: %q; want %q", got, want)
	}
	if got := body(t, reprint(t, tr, 40)); !strings.HasSuffix(got, strings.Repeat("-", 40)+"\n") {
		t.Fatalf("at 40, the function not drawn at 40: %q", got)
	}
}

// TestNewestRows: a reprint draws at most WithMaxRows rows, the newest,
// after the notice (gobble-cli's TestTranscriptRenderKeepsTheNewestRows).
func TestNewestRows(t *testing.T) {
	tr := transcript.NewTranscript(transcript.WithMaxRows(10))
	for i := range 30 {
		tr.AddText(strings.Repeat("x", i%7+1) + "\n")
	}
	tr.AddText("last one\n")
	want := notice + "x\nxx\nxxx\nxxxx\nxxxxx\nxxxxxx\nxxxxxxx\nx\nxx\nlast one\n"
	if got := body(t, reprint(t, tr, 80)); got != want {
		t.Fatalf("%q; want the notice and the last 10 rows, %q", got, want)
	}
}

// TestCutPiece: the oldest piece a reprint shows is cut by its first lines
// to fit the rows.
func TestCutPiece(t *testing.T) {
	tr := transcript.NewTranscript(transcript.WithMaxRows(3))
	tr.AddText("one\ntwo\nthree\nfour\n")
	tr.AddText("five\n")
	if got := body(t, reprint(t, tr, 80)); got != notice+"three\nfour\nfive\n" {
		t.Fatalf("%q", got)
	}
}

// TestWrappedRows: a line wider than the width counts as the rows the
// terminal wraps it into, measured by the method.
func TestWrappedRows(t *testing.T) {
	tr := transcript.NewTranscript(transcript.WithMaxRows(4))
	tr.AddText("old\n")
	tr.AddText(strings.Repeat("世", 15) + "\n") // 30 cells: three rows at 10
	tr.AddText("new\n")
	if got, want := body(t, reprint(t, tr, 10)), notice+strings.Repeat("世", 15)+"\nnew\n"; got != want {
		t.Fatalf("%q; want %q: the wide line's three rows and new's fill four, and old is left out", got, want)
	}
}

// TestDrawsOnlyWhatItShows: a reprint calls only the functions of the
// pieces it shows (gobble-cli's TestTranscriptRenderDrawsOnlyWhatItShows).
func TestDrawsOnlyWhatItShows(t *testing.T) {
	tr := transcript.NewTranscript(transcript.WithMaxRows(10))
	drawn := 0
	for range 100 {
		tr.AddDrawn(func(int) string {
			drawn++
			return "row\n"
		})
	}
	drawn = 0
	reprint(t, tr, 60)
	if drawn != 10 {
		t.Fatalf("%d pieces drawn for 10 rows; want 10", drawn)
	}
}

// TestBound: past WithMaxBytes the oldest pieces go, the newest always
// kept, and the reprint says so (gobble-cli's TestTranscriptBound).
func TestBound(t *testing.T) {
	tr := transcript.NewTranscript(transcript.WithMaxBytes(100))
	tr.AddText("the very first\n")
	for range 12 {
		tr.AddText(strings.Repeat("y", 9) + "\n")
	}
	got := body(t, reprint(t, tr, 80))
	if strings.Contains(got, "the very first") || !strings.HasPrefix(got, notice) {
		t.Fatalf("the oldest piece kept past 100 bytes, or not said to be gone: %q", got)
	}
	if n := strings.Count(got, "yyyyyyyyy\n"); n != 10 {
		t.Fatalf("%d rows of 10 bytes kept in 100; want 10", n)
	}
	one := transcript.NewTranscript(transcript.WithMaxBytes(10))
	one.AddText(strings.Repeat("z", 50) + "\n")
	if got := body(t, reprint(t, one, 80)); got != strings.Repeat("z", 50)+"\n" {
		t.Fatalf("the newest piece, over the bound alone, not kept: %q", got)
	}
}

// TestReplyGrows: a reply added before it ends is drawn whole at the
// reprint, and its source counts against the bound as it grows.
func TestReplyGrows(t *testing.T) {
	tr := transcript.NewTranscript(transcript.WithMaxBytes(30))
	tr.AddText("before\n")
	p := stream.NewPrinter()
	tr.AddReply(p)
	p.Write("first line\n")
	if got := body(t, reprint(t, tr, 80)); got != "before\nfirst line\n" {
		t.Fatalf("%q", got)
	}
	p.Write("a second line, longer\n")
	if got := body(t, reprint(t, tr, 80)); got != notice+"first line\na second line, longer\n" {
		t.Fatalf("the grown reply not counted against the bound: %q", got)
	}
}

// TestReprintOrder: SGR 0 first, then mode 2026 set, CSI 2 J before CSI
// 3 J, the cursor home, the text and mode 2026 reset (A3 item 15).
func TestReprintOrder(t *testing.T) {
	tr := transcript.NewTranscript()
	tr.AddText("x\n")
	if got := reprint(t, tr, 60); got != head+"x\r\n"+tail {
		t.Fatalf("%q", got)
	}
}

// TestWithoutScrollbackClear: CSI 2 J is written, and CSI 3 J is not (D13).
func TestWithoutScrollbackClear(t *testing.T) {
	tr := transcript.NewTranscript(transcript.WithoutScrollbackClear())
	tr.AddText("x\n")
	if got := reprint(t, tr, 60); got != "\x1b[0m\x1b[?2026h\x1b[2J\x1b[H"+"x\r\n"+tail {
		t.Fatalf("%q", got)
	}
}

// TestWithSynchronized: off, the reprint sets no mode 2026 (A8 item 3).
func TestWithSynchronized(t *testing.T) {
	tr := transcript.NewTranscript(transcript.WithSynchronized(false))
	tr.AddText("x\n")
	if got := reprint(t, tr, 60); got != "\x1b[0m\x1b[2J\x1b[3J\x1b[H"+"x\r\n" {
		t.Fatalf("%q", got)
	}
}

// TestSettle: resizes within the settle reprint once, WithSettle after the
// last; the reprint waits for the host's idle, Timeout giving no deadline
// meanwhile, and comes at its next idle call (D9, A8 item 4).
func TestSettle(t *testing.T) {
	tr := transcript.NewTranscript(transcript.WithSettle(100 * time.Millisecond))
	tr.AddText("x\n")
	if tr.Resize(80, t0); tr.HandleTimeout(t0.Add(time.Second), true) != "" {
		t.Fatal("the first Resize, which only sets the width, reprinted")
	}
	if _, ok := tr.Timeout(); ok {
		t.Fatal("a deadline after the first Resize")
	}
	if tr.Resize(80, t0); func() bool { _, ok := tr.Timeout(); return ok }() {
		t.Fatal("a deadline after a Resize to the same width")
	}
	tr.Resize(70, t0)
	tr.Resize(60, t0.Add(50*time.Millisecond))
	tr.Resize(50, t0.Add(90*time.Millisecond))
	if d, ok := tr.Timeout(); !ok || !d.Equal(t0.Add(190*time.Millisecond)) {
		t.Fatalf("Timeout = %v, %v; want the last resize's settle, %v", d, ok, t0.Add(190*time.Millisecond))
	}
	if got := tr.HandleTimeout(t0.Add(180*time.Millisecond), true); got != "" {
		t.Fatalf("reprinted before the last resize settled: %q", got)
	}
	if got := tr.HandleTimeout(t0.Add(190*time.Millisecond), false); got != "" {
		t.Fatalf("reprinted while the host's output was not idle: %q", got)
	}
	if _, ok := tr.Timeout(); ok {
		t.Fatal("a deadline, passed, while the reprint waits for idle")
	}
	if got := tr.HandleTimeout(t0.Add(time.Second), true); got != head+"x\r\n"+tail {
		t.Fatalf("at the next idle call: %q", got)
	}
	if got := tr.HandleTimeout(t0.Add(2*time.Second), true); got != "" {
		t.Fatalf("a second reprint for one settled resize: %q", got)
	}
}

// TestNoNotice: nothing left out, no notice.
func TestNoNotice(t *testing.T) {
	tr := transcript.NewTranscript(transcript.WithMaxRows(2))
	tr.AddText("one\ntwo\n")
	if got := body(t, reprint(t, tr, 80)); got != "one\ntwo\n" {
		t.Fatalf("%q", got)
	}
}

// TestNotice: the notice is the catalogue's, cleaned to one line, its line
// break a space, in WithNoticeStyle's style.
func TestNotice(t *testing.T) {
	cat := msgcat.Table{transcript.MsgEarlierOutput: "(plus \x1b]0;t\x07tôt\n)"}
	tr := transcript.NewTranscript(transcript.WithMaxRows(1), transcript.WithCatalog(cat), transcript.WithNoticeStyle(stream.Dim))
	tr.AddText("a\nb\n")
	if got := body(t, reprint(t, tr, 80)); got != "\x1b[2m(plus tôt )\x1b[0m\nb\n" {
		t.Fatalf("%q", got)
	}
	if err := msgcat.Check(transcript.Messages(), cat); err != nil {
		t.Fatal(err)
	}
}

// TestRules: AddText's text is cleaned by the rules, and never below the
// floor, safetext's Text; AddDrawn's is the host's, written as drawn.
func TestRules(t *testing.T) {
	tr := transcript.NewTranscript(transcript.WithRules(safetext.Rules{Newlines: true, KeepSGR: true}))
	tr.AddText("a\tb \x1b[31mred\x1b[0m\x1b]0;title\x07\n")
	tr.AddDrawn(func(int) string { return "\x1b[2mdim\x1b[0m\n" })
	if got := body(t, reprint(t, tr, 80)); got != "ab red\n\x1b[2mdim\x1b[0m\n" {
		t.Fatalf("%q", got)
	}
}

// TestCRLF: each line break is CR LF, one already so not doubled, so the
// reprint reads alike in raw mode.
func TestCRLF(t *testing.T) {
	tr := transcript.NewTranscript()
	tr.AddDrawn(func(int) string { return "a\r\nb\nc" })
	if got := reprint(t, tr, 60); got != head+"a\r\nb\r\nc"+tail {
		t.Fatalf("%q", got)
	}
}
