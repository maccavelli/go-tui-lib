// Package inputfiltertea is inputfilter's Bubble Tea form
// (docs/decisions/0009-MADR-streaming-content-engine.md §6, A6 item 8;
// docs/decisions/0009-PLAN-streaming-content-engine.md D10): a Filter whose
// Filter method is given to tea.WithFilter, and CoalescedWheelMsg, many
// wheel notches as one message.
//
//	f := inputfiltertea.NewFilter(inputfilter.WithPasteBurst(inputfilter.PasteBurstConfig{}))
//	p := tea.NewProgram(m, tea.WithFilter(f.Filter))
//	stop := f.Attach(p.Send)
//	defer stop()
//
// The Filter converts Bubble Tea's messages to inputfilter's types and
// back, and holds no logic of inputfilter's. A wheel sum of one notch
// reaches the model as the tea.MouseWheelMsg it was, and of more as a
// CoalescedWheelMsg; a paste burst as a tea.PasteMsg, so a pane that reads
// bracketed pastes reads it with no change.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package inputfiltertea

import (
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
	"github.com/maccavelli/go-tui-lib/v2/keymap/keymaptea"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// CoalescedWheelMsg is many wheel notches as one message: where the
// pointer is, and the notches along each axis, DY positive down and DX
// positive right. It is a tea.MouseMsg. Pointer is not an embedded
// tea.Mouse, since a field named Mouse and the method Mouse cannot both
// exist.
type CoalescedWheelMsg struct {
	Pointer tea.Mouse
	DX, DY  int
}

// Mouse is the pointer, with the wheel button of the sum's direction.
func (m CoalescedWheelMsg) Mouse() tea.Mouse { return m.Pointer }

// String is the pointer's, as tea.MouseWheelMsg's is.
func (m CoalescedWheelMsg) String() string { return m.Pointer.String() }

// WithMouse is m with the pointer t, so that a workspace gives it to a
// pane in the pane's cells.
func (m CoalescedWheelMsg) WithMouse(t tea.Mouse) tea.Msg {
	m.Pointer = t
	return m
}

// WheelDelta is the notches along each axis, so that a pane reads any
// coalesced wheel message without importing this package.
func (m CoalescedWheelMsg) WheelDelta() (dx, dy int) { return m.DX, m.DY }

// drainMsg asks the Filter for the next message it holds.
type drainMsg struct{}

// flushMsg is a flush the Flusher said fell due, to be taken on the event
// loop.
type flushMsg struct{}

// Filter is passed to tea.WithFilter. Its Filter method runs on the event
// loop; Attach may be called from any goroutine.
//
// When it must give more than one message, such as the characters a burst
// held and the key that ended it, it returns the first and holds the
// rest; a private message, sent through the program's Send by Attach's
// goroutine, releases each in turn, and while any is held, new input
// joins the end, so nothing overtakes it. Paste bursts are detected only
// while the Filter is attached, since a held character needs Attach to be
// given back.
type Filter struct {
	f   *inputfilter.Flusher
	now func() time.Time

	mu           sync.Mutex
	held         []tea.Msg
	wake         chan struct{} // a drain is wanted
	attached     bool
	attachedOnce bool
}

// NewFilter is a Filter over an inputfilter.Flusher built with o.
func NewFilter(o ...inputfilter.Option) *Filter {
	return &Filter{
		f:    inputfilter.NewFlusher(o...),
		now:  time.Now,
		wake: make(chan struct{}, 1),
	}
}

// Filter is the tea.WithFilter function.
func (fl *Filter) Filter(_ tea.Model, msg tea.Msg) tea.Msg {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	now := fl.now()
	var out []tea.Msg
	switch m := msg.(type) {
	case drainMsg:
		return fl.drain()
	case flushMsg:
		out = append(out, flushed(fl.f.Flush(now))...)
	case tea.MouseWheelMsg:
		if s, ok := fl.f.Scroll(scroll(m), now); ok {
			out = append(out, wheel(s))
		}
	case tea.MouseMotionMsg:
		if fl.f.Motion(now) {
			out = append(out, m)
		}
	case tea.KeyPressMsg, tea.KeyReleaseMsg, tea.PasteMsg:
		if !fl.attached {
			out = append(out, msg)
			break
		}
		for _, in := range fl.f.Input(input(m), now) {
			out = append(out, message(in))
		}
	default:
		out = append(out, msg)
	}
	return fl.release(out)
}

// release returns the first of out and holds the rest, or, while messages
// are held, holds all of out after them. It is called with mu held.
func (fl *Filter) release(out []tea.Msg) tea.Msg {
	if len(fl.held) > 0 {
		fl.held = append(fl.held, out...)
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	if len(out) > 1 {
		fl.held = append(fl.held, out[1:]...)
		fl.want()
	}
	return out[0]
}

// drain returns the next message held, and asks for another drain while
// more are. It is called with mu held.
func (fl *Filter) drain() tea.Msg {
	if len(fl.held) == 0 {
		return nil
	}
	m := fl.held[0]
	fl.held = fl.held[1:]
	if len(fl.held) > 0 {
		fl.want()
	} else {
		fl.held = nil
	}
	return m
}

// want asks Attach's goroutine for a drain. It is called with mu held.
func (fl *Filter) want() {
	select {
	case fl.wake <- struct{}{}:
	default:
	}
}

// Attach gives the Filter the program's Send, from a goroutine of its own:
// each drain of a held message, and each flush the Flusher says fell due,
// a wheel sum held or a paste burst ended. stop ends the goroutine, closes
// the Flusher and waits, and may be called more than once. It is for the
// program's end, once its Run has returned: what the Filter holds then, a
// character held a few milliseconds or a message waiting for its drain,
// ends with the program, and after it the Filter passes every message as
// it comes. Without Attach, a wheel sum held joins the next wheel event,
// and no paste burst is detected. A second Attach panics.
func (fl *Filter) Attach(send func(tea.Msg)) (stop func()) {
	fl.mu.Lock()
	if fl.attachedOnce {
		fl.mu.Unlock()
		panic("inputfiltertea: Attach called twice")
	}
	fl.attached, fl.attachedOnce = true, true
	fl.mu.Unlock()
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			select {
			case <-fl.wake:
				send(drainMsg{})
			case _, ok := <-fl.f.Flushes():
				if !ok {
					return
				}
				send(flushMsg{})
			case <-done:
				return
			}
		}
	})
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			wg.Wait()
			fl.f.Close() //nolint:errcheck // a Flusher's Close always returns nil
			fl.mu.Lock()
			fl.attached, fl.held = false, nil
			fl.mu.Unlock()
		})
		wg.Wait()
	}
}

// scroll is m as an inputfilter.Scroll.
func scroll(m tea.MouseWheelMsg) inputfilter.Scroll {
	s := inputfilter.Scroll{X: m.X, Y: m.Y, Mod: termkey.Mod(m.Mod)} //nolint:gosec // tea's modifier bits fit in 16
	switch m.Button {
	case tea.MouseWheelUp:
		s.DY = -1
	case tea.MouseWheelDown:
		s.DY = 1
	case tea.MouseWheelLeft:
		s.DX = -1
	case tea.MouseWheelRight:
		s.DX = 1
	}
	return s
}

// wheel is s as a message: one notch as the tea.MouseWheelMsg it was, and
// more as a CoalescedWheelMsg.
func wheel(s inputfilter.Scroll) tea.Msg {
	p := tea.Mouse{X: s.X, Y: s.Y, Mod: tea.KeyMod(s.Mod)}
	switch {
	case s.DY < 0:
		p.Button = tea.MouseWheelUp
	case s.DY > 0:
		p.Button = tea.MouseWheelDown
	case s.DX < 0:
		p.Button = tea.MouseWheelLeft
	case s.DX > 0:
		p.Button = tea.MouseWheelRight
	}
	if abs(s.DX)+abs(s.DY) == 1 {
		return tea.MouseWheelMsg(p)
	}
	return CoalescedWheelMsg{Pointer: p, DX: s.DX, DY: s.DY}
}

func abs(n int) int { return max(n, -n) }

// flushed is f as messages: its inputs, then its wheel sum.
func flushed(f inputfilter.Flush) []tea.Msg {
	var out []tea.Msg
	for _, in := range f.Inputs {
		out = append(out, message(in))
	}
	if f.HasScroll {
		out = append(out, wheel(f.Scroll))
	}
	return out
}

// input is a key or paste message as a termkey.Input.
func input(msg tea.Msg) termkey.Input {
	switch m := msg.(type) {
	case tea.KeyPressMsg:
		return termkey.Input{Kind: termkey.InputKey, Key: keymaptea.FromTea(m.Key())}
	case tea.KeyReleaseMsg:
		k := keymaptea.FromTea(m.Key())
		k.Release = true
		return termkey.Input{Kind: termkey.InputKey, Key: k}
	case tea.PasteMsg:
		return termkey.Input{Kind: termkey.InputPaste, Text: m.Content}
	}
	return termkey.Input{}
}

// message is in as the message it came as: a paste, a release or a press.
func message(in termkey.Input) tea.Msg {
	switch {
	case in.Kind == termkey.InputPaste:
		return tea.PasteMsg{Content: in.Text}
	case in.Key.Release:
		return tea.KeyReleaseMsg(keymaptea.ToTea(in.Key))
	}
	return tea.KeyPressMsg(keymaptea.ToTea(in.Key))
}
