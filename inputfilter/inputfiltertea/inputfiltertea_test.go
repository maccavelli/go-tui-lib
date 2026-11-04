package inputfiltertea_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
	"github.com/maccavelli/go-tui-lib/v2/inputfilter/inputfiltertea"
)

// prog is a program's event loop over a Filter: what the test gives, and
// what Attach's goroutine sends, passes through Filter, and what it
// returns is what the model gets. Its Send blocks until the loop reads,
// as a program's does, so a send on the calling goroutine would deadlock.
type prog struct {
	f    *inputfiltertea.Filter
	sent chan tea.Msg
	got  []tea.Msg
}

func newProg(o ...inputfilter.Option) *prog {
	return &prog{f: inputfiltertea.NewFilter(o...), sent: make(chan tea.Msg)}
}

func (p *prog) send(m tea.Msg) { p.sent <- m }

// handle runs m through the filter, as the event loop does.
func (p *prog) handle(m tea.Msg) {
	if out := p.f.Filter(nil, m); out != nil {
		p.got = append(p.got, out)
	}
}

// pump handles what Attach's goroutine sends until nothing more comes.
func (p *prog) pump() {
	for {
		synctest.Wait()
		select {
		case m := <-p.sent:
			p.handle(m)
		default:
			return
		}
	}
}

// give handles m, then pumps.
func (p *prog) give(m tea.Msg) {
	p.handle(m)
	p.pump()
}

// shown is the model's messages as text, and empties them.
func (p *prog) shown() string {
	s := ""
	for i, m := range p.got {
		if i > 0 {
			s += " "
		}
		switch m := m.(type) {
		case tea.PasteMsg:
			s += fmt.Sprintf("paste(%s)", m.Content)
		case tea.KeyPressMsg:
			s += "key(" + m.String() + ")"
		case tea.MouseWheelMsg:
			s += "wheel(" + m.String() + ")"
		case inputfiltertea.CoalescedWheelMsg:
			s += fmt.Sprintf("coalesced(%d,%d)", m.DX, m.DY)
		default:
			s += fmt.Sprintf("%T", m)
		}
	}
	p.got = nil
	return s
}

func press(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func linuxBurst() inputfilter.Option {
	return inputfilter.WithPasteBurst(inputfilter.PasteBurstConfig{GOOS: "linux"})
}

// TestFilterBurst: a burst arrives as one tea.PasteMsg once it has been
// idle, through Attach (docs/decisions/0009-MADR-streaming-content-engine.md
// §6).
func TestFilterBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newProg(linuxBurst())
		stop := p.f.Attach(p.send)
		defer stop()
		for _, r := range "abc" {
			p.give(press(r))
			time.Sleep(time.Millisecond)
		}
		if s := p.shown(); s != "" {
			t.Fatalf("during the burst the model got %q", s)
		}
		time.Sleep(7 * time.Millisecond)
		p.pump()
		if s := p.shown(); s != "paste(abc)" {
			t.Fatalf("the model got %q; want paste(abc)", s)
		}
	})
}

// TestFilterBracketed: a bracketed paste reaches the model as it came,
// and turns detection off, so fast keys pass at once.
func TestFilterBracketed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newProg(linuxBurst())
		stop := p.f.Attach(p.send)
		defer stop()
		p.give(tea.PasteStartMsg{})
		p.give(tea.PasteMsg{Content: "x"})
		p.give(tea.PasteEndMsg{})
		for _, r := range "abc" {
			p.give(press(r))
		}
		if s := p.shown(); s != "tea.PasteStartMsg paste(x) tea.PasteEndMsg key(a) key(b) key(c)" {
			t.Fatalf("the model got %q", s)
		}
	})
}

// TestFilterOrder: a burst ended by an arrow gives the paste and holds the
// arrow; a wheel event before the drain joins the end; the drains give the
// arrow, then the wheel event.
func TestFilterOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newProg(linuxBurst())
		stop := p.f.Attach(p.send)
		defer stop()
		for _, r := range "abc" {
			p.handle(press(r))
			time.Sleep(time.Millisecond)
		}
		p.handle(tea.KeyPressMsg{Code: tea.KeyUp})
		p.handle(tea.MouseWheelMsg{X: 1, Y: 2, Button: tea.MouseWheelDown})
		if s := p.shown(); s != "paste(abc)" {
			t.Fatalf("before the drains the model got %q; want the paste alone", s)
		}
		p.pump()
		if s := p.shown(); s != "key(up) wheel(wheeldown)" {
			t.Fatalf("the drains gave %q; want the arrow, then the wheel", s)
		}
	})
}

// TestFilterWheel: 100 notches within 16 ms give the first at once as the
// tea.MouseWheelMsg it was, and the other 99 as one CoalescedWheelMsg
// through Attach; without Attach, the sum held joins the next notch.
func TestFilterWheel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newProg()
		stop := p.f.Attach(p.send)
		defer stop()
		start := time.Now()
		for range 100 {
			p.give(tea.MouseWheelMsg{X: 3, Y: 4, Button: tea.MouseWheelDown})
			time.Sleep(150 * time.Microsecond)
		}
		if s := p.shown(); s != "wheel(wheeldown)" {
			t.Fatalf("within the interval the model got %q; want the first notch", s)
		}
		time.Sleep(time.Until(start.Add(16 * time.Millisecond)))
		p.pump()
		if s := p.shown(); s != "coalesced(0,99)" {
			t.Fatalf("the flush gave %q; want the other 99", s)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		p := newProg()
		for range 3 {
			p.give(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
		}
		time.Sleep(time.Second)
		p.give(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
		if s := p.shown(); s != "wheel(wheelup) coalesced(0,-3)" {
			t.Fatalf("without Attach the model got %q; want the first, then the rest with the next", s)
		}
	})
}

// TestFilterPasses: clicks, releases, keys, other messages pass as they
// came; motion is one sample an interval; with no paste burst, keys pass
// while attached too.
func TestFilterPasses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newProg()
		stop := p.f.Attach(p.send)
		defer stop()
		in := []tea.Msg{
			tea.MouseClickMsg{X: 1, Button: tea.MouseLeft},
			tea.MouseReleaseMsg{X: 1, Button: tea.MouseLeft},
			press('a'),
			tea.KeyReleaseMsg{Code: 'a', Text: "a"},
			tea.WindowSizeMsg{Width: 80, Height: 24},
		}
		for _, m := range in {
			p.give(m)
		}
		for i, m := range in {
			if p.got[i] != m {
				t.Fatalf("message %d reached the model as %#v; want %#v", i, p.got[i], m)
			}
		}
		p.got = nil
		p.give(tea.MouseMotionMsg{X: 1})
		p.give(tea.MouseMotionMsg{X: 2})
		time.Sleep(16 * time.Millisecond)
		p.give(tea.MouseMotionMsg{X: 3})
		if len(p.got) != 2 || p.got[0] != (tea.MouseMotionMsg{X: 1}) || p.got[1] != (tea.MouseMotionMsg{X: 3}) {
			t.Fatalf("motion reached the model as %v; want the first and the one an interval later", p.got)
		}
	})
}

// TestFilterAttach: before Attach, no burst is detected; after stop, what
// was held ends with it and every message passes as it comes, and no
// goroutine is left; a second stop is a no-op, and a second Attach panics.
func TestFilterAttach(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newProg(linuxBurst())
		for _, r := range "abc" {
			p.give(press(r))
		}
		if s := p.shown(); s != "key(a) key(b) key(c)" {
			t.Fatalf("before Attach the model got %q; want the keys", s)
		}
		stop := p.f.Attach(p.send)
		p.give(press('x'))
		stop()
		stop()
		p.give(tea.WindowSizeMsg{})
		for _, r := range "abc" {
			p.give(press(r))
		}
		if s := p.shown(); s != "tea.WindowSizeMsg key(a) key(b) key(c)" {
			t.Fatalf("after stop the model got %q; want each message as it came", s)
		}
		defer func() {
			if recover() == nil {
				t.Fatal("a second Attach did not panic")
			}
		}()
		p.f.Attach(p.send)
	})
}

// TestCoalescedWheelMsg: it is a tea.MouseMsg; Mouse and String are its
// pointer's; WithMouse moves the pointer and keeps the notches; and
// WheelDelta gives them.
func TestCoalescedWheelMsg(t *testing.T) {
	m := inputfiltertea.CoalescedWheelMsg{Pointer: tea.Mouse{X: 5, Y: 6, Button: tea.MouseWheelDown}, DY: 4}
	var mm tea.MouseMsg = m
	if mm.Mouse() != m.Pointer || mm.String() != m.Pointer.String() {
		t.Fatalf("Mouse %v, String %q; want the pointer's", mm.Mouse(), mm.String())
	}
	moved, ok := m.WithMouse(tea.Mouse{X: 1, Y: 2, Button: tea.MouseWheelDown}).(inputfiltertea.CoalescedWheelMsg)
	if !ok || moved.Pointer.X != 1 || moved.Pointer.Y != 2 || moved.DY != 4 {
		t.Fatalf("WithMouse gave %#v", moved)
	}
	if dx, dy := m.WheelDelta(); dx != 0 || dy != 4 {
		t.Fatalf("WheelDelta = %d, %d", dx, dy)
	}
}

// A Bubble Tea program filtered: the wheel summed and motion thinned
// before the event loop queues them, and paste bursts detected on a
// terminal that does not bracket pastes, once the Filter is attached to
// the program's Send.
func ExampleFilter() {
	// guide:filter
	f := inputfiltertea.NewFilter(inputfilter.WithPasteBurst(inputfilter.PasteBurstConfig{}))
	p := tea.NewProgram(nil, tea.WithFilter(f.Filter))
	stop := f.Attach(p.Send) // the timed flushes and the drains reach the program through its Send
	defer stop()
	// guide:end
}
