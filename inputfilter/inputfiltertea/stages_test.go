package inputfiltertea_test

import (
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
	"github.com/maccavelli/go-tui-lib/v2/internal/termevent"
	"github.com/maccavelli/go-tui-lib/v2/internal/termevent/termeventtest"
)

// TestFilterSplitReport: an SGR report Bubble Tea's reader gave up on
// after three bytes, its head an unknown event and the rest keys 60 ms
// apart, reaches the model as the click it was; a head with nothing after
// it is given back as the message it came as, once its window passes
// (docs/decisions/0009-PLAN-streaming-content-engine.md Step A1.3).
func TestFilterSplitReport(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newProg()
		stop := p.f.Attach(p.send)
		defer stop()
		p.give(termeventtest.Unknown("\x1b[<0;1"))
		for _, k := range []tea.KeyPressMsg{press('0'), press(';'), press('5'), {Code: 'm', Text: "M", Mod: tea.ModShift}} {
			time.Sleep(60 * time.Millisecond)
			p.give(k)
		}
		if len(p.got) != 1 || p.got[0] != (tea.MouseClickMsg{X: 9, Y: 4, Button: tea.MouseLeft}) {
			t.Fatalf("the model got %#v; want the click at (9,4)", p.got)
		}
		p.got = nil
		p.give(termeventtest.Unknown("\x1b[<0;1"))
		time.Sleep(500 * time.Millisecond)
		p.pump()
		if len(p.got) != 1 {
			t.Fatalf("after the window the model got %#v; want the head", p.got)
		}
		if ev, ok := termevent.Decode(p.got[0]); !ok || ev.Kind != termevent.Unknown || ev.Raw != "\x1b[<0;1" {
			t.Fatalf("the head came back as %#v; want the unknown event it was", p.got[0])
		}
	})
}

// TestFilterX10: the clicks the measurement found misread, as Bubble Tea
// decoded ConPTY's re-encoding, with the key their row became, reach the
// model as the clicks they were; a click at those cells with no key after
// it passes as it came.
func TestFilterX10(t *testing.T) {
	for _, c := range []struct {
		misread tea.MouseClickMsg
		col     int
	}{
		{tea.MouseClickMsg{X: 161, Y: 95, Button: tea.MouseLeft}, 96},
		{tea.MouseClickMsg{X: 161, Y: 119, Button: tea.MouseLeft}, 120},
		{tea.MouseClickMsg{X: 162, Y: 135, Button: tea.MouseLeft}, 200},
		{tea.MouseClickMsg{X: 162, Y: 158, Button: tea.MouseLeft}, 223},
	} {
		synctest.Test(t, func(t *testing.T) {
			p := newProg()
			stop := p.f.Attach(p.send)
			defer stop()
			p.give(c.misread)
			p.give(press('!'))
			want := tea.MouseClickMsg{X: c.col - 1, Y: 0, Button: tea.MouseLeft}
			if len(p.got) != 1 || p.got[0] != want {
				t.Fatalf("column %d: the model got %#v; want %#v", c.col, p.got, want)
			}
			p.got = nil
			p.give(c.misread)
			time.Sleep(time.Second)
			p.pump()
			if len(p.got) != 1 || p.got[0] != c.misread {
				t.Fatalf("column %d alone: the model got %#v; want it as it came", c.col, p.got)
			}
		})
	}
}

// TestFilterPastes: an empty bracketed paste reaches the model as a
// ClipboardImageRequestMsg, and as itself with WithoutClipboardImageRequest;
// a paste's CR and CR LF are LF, attached or not.
func TestFilterPastes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newProg()
		p.give(tea.PasteMsg{Content: ""})
		p.give(tea.PasteMsg{Content: "a\r\nb\rc"})
		if len(p.got) != 2 || p.got[0] != (inputfilter.ClipboardImageRequestMsg{}) || p.got[1] != (tea.PasteMsg{Content: "a\nb\nc"}) {
			t.Fatalf("the model got %#v; want the request, then the paste normalised", p.got)
		}
		q := newProg(inputfilter.WithoutClipboardImageRequest())
		q.give(tea.PasteMsg{Content: ""})
		if len(q.got) != 1 || q.got[0] != (tea.PasteMsg{Content: ""}) {
			t.Fatalf("WithoutClipboardImageRequest: the model got %#v; want the empty paste", q.got)
		}
	})
}

// TestFilterCapture: Capture holds keys while a click passes; Replay gives
// the typing up to the first Esc, through the drains.
func TestFilterCapture(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newProg()
		stop := p.f.Attach(p.send)
		defer stop()
		p.f.Capture()
		click := tea.MouseClickMsg{X: 1, Button: tea.MouseLeft}
		for _, m := range []tea.Msg{press('h'), click, press('i'), tea.KeyPressMsg{Code: tea.KeyEscape}, press('x')} {
			p.give(m)
		}
		if len(p.got) != 1 || p.got[0] != click {
			t.Fatalf("during Capture the model got %#v; want the click alone", p.got)
		}
		p.got = nil
		p.f.Replay()
		p.pump()
		if s := p.shown(); s != "key(h) key(i)" {
			t.Fatalf("Replay gave %q; want the typing before Esc", s)
		}
	})
}

// TestFilterQuarantine: until the deadline, keys, pastes, clicks and focus
// are dropped, and other messages pass; at it, input passes again.
func TestFilterQuarantine(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newProg()
		stop := p.f.Attach(p.send)
		defer stop()
		p.f.Quarantine(time.Now().Add(time.Second))
		for _, m := range []tea.Msg{press('y'), tea.PasteMsg{Content: "y"}, tea.MouseClickMsg{Button: tea.MouseLeft}, tea.FocusMsg{}, tea.WindowSizeMsg{Width: 80}} {
			p.give(m)
		}
		if len(p.got) != 1 || p.got[0] != (tea.WindowSizeMsg{Width: 80}) {
			t.Fatalf("during the quarantine the model got %#v; want the size alone", p.got)
		}
		p.got = nil
		time.Sleep(time.Second)
		p.give(press('n'))
		if s := p.shown(); s != "key(n)" {
			t.Fatalf("at the deadline the model got %q; want the key", s)
		}
	})
}
