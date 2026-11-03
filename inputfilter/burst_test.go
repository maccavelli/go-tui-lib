package inputfilter_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// char is the key press that types s.
func char(s string) termkey.Input {
	return termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: []rune(s)[0], Text: s}}
}

// key is the key press of code with mods, which types nothing.
func key(code rune, mods termkey.Mod) termkey.Input {
	return termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: code, Mod: mods}}
}

func paste(s string) termkey.Input { return termkey.Input{Kind: termkey.InputPaste, Text: s} }

// show is ins as text: a paste as [text], a key as its keystroke.
func show(ins []termkey.Input) string {
	var parts []string
	for _, in := range ins {
		if in.Kind == termkey.InputPaste {
			parts = append(parts, fmt.Sprintf("[%s]", strings.ReplaceAll(in.Text, "\n", `\n`)))
			continue
		}
		parts = append(parts, in.Key.Keystroke())
	}
	return strings.Join(parts, " ")
}

// step is an input at a time after t0, or a tick when in is nil.
type step struct {
	at time.Duration
	in *termkey.Input
}

func in(at time.Duration, i termkey.Input) step { return step{at, &i} }
func tick(at time.Duration) step                { return step{at: at} }

// feed runs steps through p, handling a passed input itself, and gives
// every input that came out, in order.
func feed(p *inputfilter.PasteBurst, steps ...step) []termkey.Input {
	var out []termkey.Input
	for _, s := range steps {
		now := t0.Add(s.at)
		if s.in != nil && p.Key(*s.in, now) == inputfilter.Pass {
			out = append(out, *s.in)
			continue
		}
		for i, ok := p.Tick(now); ok; i, ok = p.Tick(now) {
			out = append(out, i)
		}
	}
	return out
}

const ms = time.Millisecond

// TestPasteBurst: codex's cases, as a filter reads them
// (docs/decisions/0009-MADR-streaming-content-engine.md §6;
// docs/decisions/0009-PLAN-streaming-content-engine.md D11).
func TestPasteBurst(t *testing.T) {
	linux := inputfilter.PasteBurstConfig{GOOS: "linux"}
	for _, c := range []struct {
		name  string
		cfg   inputfilter.PasteBurstConfig
		steps []step
		want  string
	}{
		{"a 3-character burst", linux,
			[]step{in(0, char("a")), in(ms, char("b")), in(2*ms, char("c")), tick(9 * ms), tick(10 * ms)},
			"[abc]"},
		{"idle not yet over", linux,
			[]step{in(0, char("a")), in(ms, char("b")), in(2*ms, char("c")), tick(9*ms + 999*time.Microsecond)},
			""},
		{"two held, given back as keys", linux,
			[]step{in(0, char("a")), in(ms, char("b")), tick(8 * ms), tick(9 * ms)},
			"a b"},
		{"a held character, given back after the gap", linux,
			[]step{in(0, char("a")), tick(7 * ms), tick(8 * ms)},
			"a"},
		{"Enter inside a burst", linux,
			[]step{in(0, char("a")), in(ms, char("b")), in(2*ms, char("c")), in(3*ms, key(termkey.KeyEnter, 0)), in(4*ms, char("d")), tick(20 * ms)},
			`[abc\nd]`},
		{"Tab inside a burst", linux,
			[]step{in(0, char("a")), in(ms, char("b")), in(2*ms, char("c")), in(3*ms, key(termkey.KeyTab, 0)), tick(20 * ms)},
			"[abc\t]"},
		{"Enter inside a run, which becomes a burst", linux,
			[]step{in(0, char("a")), in(ms, key(termkey.KeyEnter, 0)), in(2*ms, char("b")), tick(20 * ms)},
			`[a\nb]`},
		{"Enter alone is a key", linux,
			[]step{in(0, key(termkey.KeyEnter, 0))},
			"enter"},
		{"a character after the idle end starts again", linux,
			[]step{in(0, char("a")), in(ms, char("b")), in(2*ms, char("c")), in(20*ms, char("x")), tick(40 * ms)},
			"[abc] x"},
		{"a burst ended by an arrow", linux,
			[]step{in(0, char("a")), in(ms, char("b")), in(2*ms, char("c")), in(3*ms, key(termkey.KeyUp, 0))},
			"[abc] up"},
		{"a run ended by an arrow", linux,
			[]step{in(0, char("a")), in(ms, key(termkey.KeyUp, 0))},
			"a up"},
		{"a run ended by ctrl+a", linux,
			[]step{in(0, char("a")), in(ms, termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: 'a', Text: "a", Mod: termkey.ModCtrl}})},
			"a ctrl+a"},
		{"Windows' longer idle", inputfilter.PasteBurstConfig{GOOS: "windows"},
			[]step{in(0, char("a")), in(ms, char("b")), in(2*ms, char("c")), tick(10 * ms), in(30*ms, char("d")), tick(89 * ms), tick(90 * ms)},
			"[abcd]"},
		{"AltGr on Windows types", inputfilter.PasteBurstConfig{GOOS: "windows"},
			[]step{in(0, char("a")), in(ms, termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: '@', Text: "@", Mod: termkey.ModCtrl | termkey.ModAlt}}), in(2*ms, char("b")), tick(70 * ms)},
			"[a@b]"},
		{"Ctrl+Alt elsewhere does not", linux,
			[]step{in(0, char("a")), in(ms, termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: '@', Text: "@", Mod: termkey.ModCtrl | termkey.ModAlt}})},
			"a ctrl+alt+@"},
		{"shifted and capitals join", linux,
			[]step{in(0, char("A")), in(ms, termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: 'b', Text: "B", Mod: termkey.ModShift | termkey.ModCapsLock}}), in(2*ms, char("c")), tick(20 * ms)},
			"[ABc]"},
		{"typing 50 ms apart passes through", linux,
			[]step{in(0, char("h")), in(50*ms, char("i")), tick(100 * ms), in(150*ms, char("!")), tick(200 * ms)},
			"h i !"},
		{"MinChars, Gap and Idle set", inputfilter.PasteBurstConfig{GOOS: "linux", MinChars: 2, Gap: 20 * ms, Idle: 30 * ms},
			[]step{in(0, char("a")), in(15*ms, char("b")), tick(44 * ms), tick(45 * ms)},
			"[ab]"},
		{"a bracketed paste turns detection off", linux,
			[]step{in(0, paste("x")), in(ms, char("a")), in(2*ms, char("b")), in(3*ms, char("c"))},
			"[x] a b c"},
		{"a bracketed paste ends a run first", linux,
			[]step{in(0, char("a")), in(ms, paste("x")), in(2*ms, char("b"))},
			"a [x] b"},
	} {
		got := show(feed(inputfilter.NewPasteBurst(c.cfg), c.steps...))
		if got != c.want {
			t.Errorf("%s: %q; want %q", c.name, got, c.want)
		}
	}
}

// TestPowerShellPasteBurst: grok's cases (grok
// xai-grok-pager/src/app/event_loop.rs:3587-3715): a single-line run is
// given back as its keys; a run with a character after an Enter is a
// paste; an Enter then a Ctrl+J is one line break, and a paste; a dropped
// Windows path of 8 or more is a paste, and of 7 is not; and the second
// key must come within 2 ms.
func TestPowerShellPasteBurst(t *testing.T) {
	win := inputfilter.PowerShellPasteBurst()
	win.GOOS = "windows"
	ctrlJ := termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: 'j', Mod: termkey.ModCtrl}}
	typed := func(s string) []step {
		var out []step
		for i, r := range s {
			out = append(out, in(time.Duration(i)*ms, char(string(r))))
		}
		return append(out, tick(time.Second))
	}
	for _, c := range []struct {
		name  string
		steps []step
		want  string
	}{
		{"a single line given back", typed("abcd"), "a b c d"},
		{"a character after an Enter", []step{
			in(0, char("a")), in(ms, char("b")), in(2*ms, key(termkey.KeyEnter, 0)), in(3*ms, char("c")), tick(time.Second),
		}, `[ab\nc]`},
		{"an Enter alone at the end", []step{
			in(0, char("a")), in(ms, char("b")), in(2*ms, key(termkey.KeyEnter, 0)), tick(time.Second),
		}, "a b enter"},
		{"Enter then Ctrl+J", []step{
			in(0, char("a")), in(ms, key(termkey.KeyEnter, 0)), in(2*ms, ctrlJ), tick(time.Second),
		}, `[a\n]`},
		{"a Windows path of 8", typed(`C:\x.png`), `[C:\x.png]`},
		{"a Windows path of 7", typed(`C:\x.pn`), `C : \ x . p n`},
		{"the second key past 2 ms", []step{
			in(0, char("a")), in(3*ms, char("b")), in(4*ms, key(termkey.KeyEnter, 0)), in(5*ms, char("c")), tick(time.Second),
		}, `a [b\nc]`},
	} {
		got := show(feed(inputfilter.NewPasteBurst(win), c.steps...))
		if got != c.want {
			t.Errorf("%s: %q; want %q", c.name, got, c.want)
		}
	}
	first := inputfilter.PasteBurstConfig{GOOS: "linux", FirstGap: 2 * ms}
	if got := show(feed(inputfilter.NewPasteBurst(first), in(0, char("a")), in(3*ms, char("b")), in(4*ms, char("c")), in(5*ms, char("d")), tick(time.Second))); got != "a [bcd]" {
		t.Errorf("FirstGap alone: %q; want the first given back and the rest a paste", got)
	}
}

// A line-mode host on a terminal that does not bracket pastes: three
// characters a millisecond apart are a burst, given as one paste once it
// has been idle; a key typed alone is given back after the gap.
func ExamplePasteBurst() {
	p := inputfilter.NewPasteBurst(inputfilter.PasteBurstConfig{GOOS: "linux"})
	start := time.Unix(0, 0)
	for i, r := range "abc" {
		p.Key(char(string(r)), start.Add(time.Duration(i)*time.Millisecond))
	}
	p.Key(char("x"), start.Add(50*time.Millisecond))
	for in, ok := p.Tick(start.Add(time.Second)); ok; in, ok = p.Tick(start.Add(time.Second)) {
		fmt.Printf("%s %q\n", in.Kind, in.Text+in.Key.Text)
	}
	// Output:
	// paste "abc"
	// key "x"
}

// TestPasteBurstDue: Due is the gap after a run's last character, the idle
// time after a burst's, and false with nothing held.
func TestPasteBurstDue(t *testing.T) {
	p := inputfilter.NewPasteBurst(inputfilter.PasteBurstConfig{GOOS: "windows"})
	if _, ok := p.Due(); ok {
		t.Fatal("Due with nothing held")
	}
	p.Key(char("a"), t0)
	if due, _ := p.Due(); !due.Equal(t0.Add(8 * ms)) {
		t.Fatalf("a run's Due %v; want the gap", due.Sub(t0))
	}
	p.Key(char("b"), t0.Add(ms))
	p.Key(char("c"), t0.Add(2*ms))
	if due, _ := p.Due(); !due.Equal(t0.Add(62 * ms)) {
		t.Fatalf("a burst's Due %v; want Windows' idle after the last", due.Sub(t0))
	}
	if a := p.Key(key(termkey.KeyUp, 0), t0.Add(3*ms)); a != inputfilter.Release {
		t.Fatalf("an arrow after a burst: %v; want release", a)
	}
	if due, ok := p.Due(); !ok || !due.IsZero() {
		t.Fatalf("Due with inputs ready: %v, %v; want the zero time", due, ok)
	}
}
