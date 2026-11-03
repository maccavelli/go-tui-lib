package inputfilter_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// feedFlusher gives ins to f 60 ms apart, and gives every input that came
// out, flushing long after the last.
func feedFlusher(f *inputfilter.Flusher, ins []termkey.Input) []termkey.Input {
	var out []termkey.Input
	now := t0
	for _, in := range ins {
		now = now.Add(60 * time.Millisecond)
		out = append(out, f.Input(in, now)...)
	}
	return append(out, f.Flush(now.Add(time.Hour)).Inputs...)
}

// TestFlusherStages: the Flusher joins a split report, repairs an X10
// report, drops a reply WithReplies knows and normalises a paste by
// default; each option turns its stage off.
func TestFlusherStages(t *testing.T) {
	split := decoded("\x1b[<0;1", "0;5M")
	reply := decoded("\x1b[?62", ";22c")
	pasted := []termkey.Input{paste("a\r\nb\rc")}
	for _, c := range []struct {
		name string
		o    []inputfilter.Option
		ins  []termkey.Input
		want string
	}{
		{"a split report", nil, split, `u("\x1b[<0;10;5M")`},
		{"WithoutReassembly", []inputfilter.Option{inputfilter.WithoutReassembly()}, split, shown(split)},
		{"a reply, no matcher", nil, reply, shown(reply)},
		{"WithReplies", []inputfilter.Option{inputfilter.WithReplies(replies)}, reply, ""},
		{"a paste", nil, pasted, "[a\nb\nc]"},
		{"WithoutPasteNormalization", []inputfilter.Option{inputfilter.WithoutPasteNormalization()}, pasted, "[a\r\nb\rc]"},
	} {
		f := inputfilter.NewFlusher(c.o...)
		if got := shown(feedFlusher(f, c.ins)); got != c.want {
			t.Errorf("%s: %s; want %s", c.name, got, c.want)
		}
		_ = f.Close()
	}
	x := decoded(conpty(x10(0, 120, 1)))
	for _, c := range []struct {
		name string
		o    []inputfilter.Option
		want string
	}{
		{"an X10 report", nil, shown([]termkey.Input{{Kind: termkey.InputUnknown, Raw: []byte(x10(0, 120, 1))}})},
		{"WithoutX10Repair", []inputfilter.Option{inputfilter.WithoutX10Repair()}, shown(x)},
	} {
		f := inputfilter.NewFlusher(c.o...)
		var out []termkey.Input
		for _, in := range x {
			out = append(out, f.Input(in, t0)...)
		}
		out = append(out, f.Flush(t0.Add(time.Hour)).Inputs...)
		if got := shown(out); got != c.want {
			t.Errorf("%s: %s; want %s", c.name, got, c.want)
		}
		_ = f.Close()
	}
}

// TestFlusherWindow: a head held falls due when its window passes, and
// Flush gives it back; WithReassemblyWindow sets it.
func TestFlusherWindow(t *testing.T) {
	for _, c := range []struct {
		o    []inputfilter.Option
		want time.Duration
	}{{nil, 500 * time.Millisecond}, {[]inputfilter.Option{inputfilter.WithReassemblyWindow(90 * time.Millisecond)}, 90 * time.Millisecond}} {
		synctest.Test(t, func(t *testing.T) {
			f := inputfilter.NewFlusher(c.o...)
			defer f.Close()
			start := time.Now()
			if out := f.Input(termkey.Input{Kind: termkey.InputUnknown, Raw: []byte("\x1b[<0;1")}, start); len(out) != 0 {
				t.Fatalf("the head gave %s; want it held", shown(out))
			}
			time.Sleep(c.want - time.Nanosecond)
			noFlush(t, f, "a nanosecond short of the window")
			time.Sleep(time.Nanosecond)
			at := mustFlush(t, f)
			if !at.Equal(start.Add(c.want)) {
				t.Fatalf("the head fell due %v on; want %v", at.Sub(start), c.want)
			}
			if got := shown(f.Flush(time.Now()).Inputs); got != `u("\x1b[<0;1")` {
				t.Fatalf("Flush gave %s; want the head", got)
			}
		})
	}
}

// TestFlusherCapture: Capture holds keys and pastes, and passes the
// terminal's reports; Replay gives the typing, as Typeahead keeps it, each
// paste normalised, and ends the capture.
func TestFlusherCapture(t *testing.T) {
	f := inputfilter.NewFlusher()
	defer f.Close()
	f.Capture()
	report := termkey.Input{Kind: termkey.InputUnknown, Raw: []byte("\x1b[<0;1;1M")}
	var passed []termkey.Input
	for _, in := range []termkey.Input{char("h"), report, key(termkey.KeyUp, 0), paste("a\rb"), char("i"), key(termkey.KeyEscape, 0), char("x")} {
		passed = append(passed, f.Input(in, t0)...)
	}
	if got := shown(passed); got != `u("\x1b[<0;1;1M")` {
		t.Fatalf("during Capture: %s; want the report alone", got)
	}
	if got := show(f.Replay()); got != `h [a\nb] i` {
		t.Fatalf("Replay gave %q; want the typing up to Esc", got)
	}
	if got := show(f.Input(char("y"), t0)); got != "y" {
		t.Fatalf("after Replay a key gave %q; want it", got)
	}
}

// TestFlusherQuarantine: until its deadline, keys, pastes, wheel events and
// motion are dropped, and what the stages held with them; after it, input
// passes.
func TestFlusherQuarantine(t *testing.T) {
	f := inputfilter.NewFlusher(inputfilter.WithPasteBurst(inputfilter.PasteBurstConfig{GOOS: "linux"}))
	defer f.Close()
	f.Input(char("a"), t0)
	f.Input(termkey.Input{Kind: termkey.InputUnknown, Raw: []byte("\x1b[<0;1")}, t0)
	f.Quarantine(t0.Add(time.Second))
	if fl := f.Flush(t0.Add(time.Hour)); len(fl.Inputs) != 0 {
		t.Fatalf("what the stages held came out: %s", show(fl.Inputs))
	}
	at := t0.Add(999 * time.Millisecond)
	if out := f.Input(char("b"), at); len(out) != 0 {
		t.Fatal("a key before the deadline passed")
	}
	if out := f.Input(paste("p"), at); len(out) != 0 {
		t.Fatal("a paste before the deadline passed")
	}
	if _, ok := f.Scroll(down(0, 0), at); ok {
		t.Fatal("a wheel event before the deadline passed")
	}
	if f.Motion(at) {
		t.Fatal("motion before the deadline passed")
	}
	if out := f.Input(key(termkey.KeyUp, 0), t0.Add(time.Second)); show(out) != "up" {
		t.Fatalf("at the deadline a key gave %q; want it", show(out))
	}
}

// TestClipboardImageRequest: an empty bracketed paste is the request,
// unless WithoutClipboardImageRequest; a paste with text never is.
func TestClipboardImageRequest(t *testing.T) {
	f := inputfilter.NewFlusher()
	g := inputfilter.NewFlusher(inputfilter.WithoutClipboardImageRequest())
	defer f.Close()
	defer g.Close()
	if !f.ClipboardImageRequest(paste("")) || f.ClipboardImageRequest(paste("x")) || f.ClipboardImageRequest(char("a")) {
		t.Fatal("ClipboardImageRequest: want only an empty paste")
	}
	if g.ClipboardImageRequest(paste("")) {
		t.Fatal("WithoutClipboardImageRequest still reports the request")
	}
}
