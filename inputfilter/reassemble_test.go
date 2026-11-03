package inputfilter_test

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// decoded is what termkey's Decoder gives for parts, flushed between
// them, as a line-mode host does when a gap passes its timeout; this is
// the measurement's termkey path (docs/decisions/0009-PLAN-streaming-content-engine.md
// Step A1.3).
func decoded(parts ...string) []termkey.Input {
	var d termkey.Decoder
	var ins []termkey.Input
	for _, p := range parts {
		ins = d.Decode([]byte(p), ins)
		if d.Pending() {
			ins = d.Flush(ins)
		}
	}
	return ins
}

// joinAll gives ins to r 60 ms apart, past Bubble Tea's 50 ms join, and
// gives every input that came out, ticking long after the last.
func joinAll(r *inputfilter.Reassembler, ins []termkey.Input) []termkey.Input {
	return joinApart(r, ins, 60*time.Millisecond)
}

// joinApart is joinAll with the inputs gap apart.
func joinApart(r *inputfilter.Reassembler, ins []termkey.Input, gap time.Duration) []termkey.Input {
	var out []termkey.Input
	now := t0
	for _, in := range ins {
		now = now.Add(gap)
		if r.Key(in, now) == inputfilter.Pass {
			out = append(out, in)
			continue
		}
		for i, ok := r.Tick(now); ok; i, ok = r.Tick(now) {
			out = append(out, i)
		}
	}
	for i, ok := r.Tick(now.Add(time.Hour)); ok; i, ok = r.Tick(now.Add(time.Hour)) {
		out = append(out, i)
	}
	return out
}

// shown is ins as text: an unknown as u(raw), a key as its keystroke.
func shown(ins []termkey.Input) string {
	var parts []string
	for _, in := range ins {
		switch in.Kind {
		case termkey.InputUnknown:
			parts = append(parts, fmt.Sprintf("u(%q)", in.Raw))
		case termkey.InputPaste:
			parts = append(parts, fmt.Sprintf("[%s]", in.Text))
		default:
			parts = append(parts, in.Key.Keystroke())
		}
	}
	return strings.Join(parts, " ")
}

// replies is a matcher that knows the replies these tests split.
var replies = inputfilter.ReplyMatcherFunc(func(raw string) bool {
	return strings.HasPrefix(raw, "\x1b[?") || strings.HasPrefix(raw, "\x1b]")
})

// TestReassembleSplits: a report or a reply split past the reader's
// timeout, in each two-way split whose head is three bytes or more, is
// one input again, or none for a reply the matcher knows; a split after
// one or two bytes is left as it came, and a reply the matcher does not
// know is given back as it came
// (docs/decisions/0009-PLAN-streaming-content-engine.md D14).
func TestReassembleSplits(t *testing.T) {
	for _, c := range []struct {
		name, seq string
		reply     bool
	}{
		{"an SGR report", "\x1b[<0;10;5M", false},
		{"a DA1 reply", "\x1b[?62;22c", true},
		{"a kitty keyboard reply", "\x1b[?1u", true},
		{"an OSC 11 reply", "\x1b]11;rgb:0000/0000/0000\x07", true},
	} {
		for k := 1; k < len(c.seq); k++ {
			ins := decoded(c.seq[:k], c.seq[k:])
			name := fmt.Sprintf("%s split %q|%q", c.name, c.seq[:k], c.seq[k:])
			got := shown(joinAll(inputfilter.NewReassembler(inputfilter.ReassembleConfig{Replies: replies}), ins))
			want := ""
			switch {
			case k < 3:
				want = shown(ins)
			case !c.reply:
				want = fmt.Sprintf("u(%q)", c.seq)
			}
			if got != want {
				t.Errorf("%s: %s; want %s", name, got, want)
			}
			if k >= 3 && c.reply {
				nomatch := shown(joinAll(inputfilter.NewReassembler(inputfilter.ReassembleConfig{}), ins))
				if nomatch != shown(ins) {
					t.Errorf("%s with no matcher: %s; want it as it came, %s", name, nomatch, shown(ins))
				}
			}
		}
	}
}

// conpty is s as ConPTY relays it: each byte of 0x80 and over the UTF-8 of
// its value.
func conpty(s string) string {
	var b strings.Builder
	for i := range len(s) {
		if s[i] < 0x80 {
			b.WriteByte(s[i])
		} else {
			b.Write(utf8.AppendRune(nil, rune(s[i])))
		}
	}
	return b.String()
}

// TestReassembleX10: the X10 reports the measurement found misread,
// columns 96, 120, 200 and 223 as ConPTY relays them, are repaired; under
// 96, and with NoX10, they pass as they came.
func TestReassembleX10(t *testing.T) {
	for _, col := range []int{96, 120, 200, 223} {
		ins := decoded(conpty(x10(0, col, 1)))
		out := joinApart(inputfilter.NewReassembler(inputfilter.ReassembleConfig{}), ins, 0) // one read
		if len(out) != 1 {
			t.Fatalf("column %d: %s; want one report", col, shown(out))
		}
		r, ok := inputfilter.ParseReport(string(out[0].Raw))
		if !ok || r.X != col-1 || r.Y != 0 || r.Kind != inputfilter.MousePress {
			t.Errorf("column %d: %+v, %v; want a press at X %d", col, r, ok, col-1)
		}
		if got := shown(joinApart(inputfilter.NewReassembler(inputfilter.ReassembleConfig{NoX10: true}), ins, 0)); got != shown(ins) {
			t.Errorf("column %d with NoX10: %s; want %s", col, got, shown(ins))
		}
		if got := shown(joinApart(inputfilter.NewReassembler(inputfilter.ReassembleConfig{}), ins, 6*time.Millisecond)); got != shown(ins) {
			t.Errorf("column %d, its key 6 ms after: %s; want it as it came, not one read", col, got)
		}
	}
	ins := decoded(conpty(x10(0, 95, 1)))
	if got := shown(joinAll(inputfilter.NewReassembler(inputfilter.ReassembleConfig{}), ins)); got != shown(ins) {
		t.Errorf("column 95: %s; want it as it came", got)
	}
}

// TestReassembleEnds: a head with nothing after it is given back when its
// window passes, and not before; a bare Esc, or any key that is not the
// sequence's next byte, gives it back at once, before itself; a sequence
// past 1 KiB is given back; a whole reply the matcher knows is dropped.
func TestReassembleEnds(t *testing.T) {
	head := termkey.Input{Kind: termkey.InputUnknown, Raw: []byte("\x1b[<0;1")}
	for _, c := range []struct {
		window, want time.Duration
	}{{0, 500 * time.Millisecond}, {80 * time.Millisecond, 80 * time.Millisecond}} {
		r := inputfilter.NewReassembler(inputfilter.ReassembleConfig{Window: c.window})
		if a := r.Key(head, t0); a != inputfilter.Hold {
			t.Fatalf("the head: %v; want hold", a)
		}
		if due, _ := r.Due(); !due.Equal(t0.Add(c.want)) {
			t.Fatalf("Due %v; want %v", due.Sub(t0), c.want)
		}
		if _, ok := r.Tick(t0.Add(c.want - time.Nanosecond)); ok {
			t.Fatal("the head given back before its window passed")
		}
		if in, ok := r.Tick(t0.Add(c.want)); !ok || string(in.Raw) != "\x1b[<0;1" {
			t.Fatalf("at the window: %v, %v; want the head", shown([]termkey.Input{in}), ok)
		}
	}
	for _, k := range []termkey.Input{key(termkey.KeyEscape, 0), key(termkey.KeyUp, 0)} {
		r := inputfilter.NewReassembler(inputfilter.ReassembleConfig{})
		r.Key(head, t0)
		if got := shown(joinAll(r, []termkey.Input{k})); got != fmt.Sprintf("u(%q) %s", "\x1b[<0;1", k.Key.Keystroke()) {
			t.Errorf("%s after a head: %s; want the head, then it", k.Key.Keystroke(), got)
		}
	}
	long := []termkey.Input{{Kind: termkey.InputUnknown, Raw: []byte("\x1b[?1")}}
	for range 1100 {
		long = append(long, char("1"))
	}
	if out := joinAll(inputfilter.NewReassembler(inputfilter.ReassembleConfig{}), long); len(out) != len(long) {
		t.Errorf("a sequence past 1 KiB gave %d inputs; want the %d it came as", len(out), len(long))
	}
	whole := termkey.Input{Kind: termkey.InputUnknown, Raw: []byte("\x1b[?62;22c")}
	if out := joinAll(inputfilter.NewReassembler(inputfilter.ReassembleConfig{Replies: replies}), []termkey.Input{whole}); len(out) != 0 {
		t.Errorf("a whole reply the matcher knows: %s; want it dropped", shown(out))
	}
}
