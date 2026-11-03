// Package inputfiltertea is inputfilter's Bubble Tea form
// (docs/decisions/0009-MADR-streaming-content-engine.md §6, A6 item 8;
// docs/decisions/0009-PLAN-streaming-content-engine.md D10, Step A1.3): a
// Filter whose Filter method is given to tea.WithFilter, and
// CoalescedWheelMsg, many wheel notches as one message.
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
// bracketed pastes reads it with no change; a report the Reassembler
// joined, or an X10 report it repaired, as Bubble Tea's own mouse, focus
// or blur message; and an empty bracketed paste as an
// inputfilter.ClipboardImageRequestMsg.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package inputfiltertea

import (
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
	"github.com/maccavelli/go-tui-lib/v2/internal/termevent"
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

// maxOriginals bounds the messages a Filter keeps to give back as they
// came; a head is given back, joined or dropped long before so many more.
const maxOriginals = 64

// original is a message the Filter gave the Flusher as an InputUnknown,
// to give back if the Flusher gives that input back as it came.
type original struct {
	raw string
	msg tea.Msg
}

// Filter is passed to tea.WithFilter. Its Filter method runs on the event
// loop; the others may be called from any goroutine.
//
// When it must give more than one message, such as the characters a burst
// held and the key that ended it, it returns the first and holds the
// rest; a private message, sent through the program's Send by Attach's
// goroutine, releases each in turn, and while any is held, new input
// joins the end, so nothing overtakes it. Keys and a terminal's split
// reports reach the Flusher's stages only while the Filter is attached,
// since what a stage holds needs Attach to be given back; pastes reach
// them always.
type Filter struct {
	f   *inputfilter.Flusher
	now func() time.Time

	mu           sync.Mutex
	held         []tea.Msg
	originals    []original
	quarantine   time.Time
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
	if now.Before(fl.quarantine) && isInput(msg) {
		return nil
	}
	if m, ok := msg.(tea.MouseMsg); ok && fl.attached && x10Suspect(m.Mouse()) {
		raw := x10Raw(m)
		fl.keep(raw, msg)
		return fl.release(fl.messages(fl.f.Input(termkey.Input{Kind: termkey.InputUnknown, Raw: []byte(raw)}, now)))
	}
	var out []tea.Msg
	switch m := msg.(type) {
	case drainMsg:
		return fl.drain()
	case flushMsg:
		f := fl.f.Flush(now)
		out = fl.messages(f.Inputs)
		if f.HasScroll {
			out = append(out, wheel(f.Scroll))
		}
	case tea.MouseWheelMsg:
		if s, ok := fl.f.Scroll(scroll(m), now); ok {
			out = append(out, wheel(s))
		}
	case tea.MouseMotionMsg:
		if fl.f.Motion(now) {
			out = append(out, m)
		}
	case tea.PasteMsg:
		out = fl.messages(fl.f.Input(input(m), now))
	case tea.KeyPressMsg, tea.KeyReleaseMsg:
		if !fl.attached {
			out = append(out, msg)
			break
		}
		out = fl.messages(fl.f.Input(input(m), now))
	default:
		ev, ok := termevent.Decode(msg)
		if !ok || ev.Kind != termevent.Unknown || !fl.attached {
			out = append(out, msg)
			break
		}
		fl.keep(ev.Raw, msg)
		out = fl.messages(fl.f.Input(termkey.Input{Kind: termkey.InputUnknown, Raw: []byte(ev.Raw)}, now))
	}
	return fl.release(out)
}

// keep remembers msg, which the Filter gives the Flusher as raw. It is
// called with mu held.
func (fl *Filter) keep(raw string, msg tea.Msg) {
	if len(fl.originals) == maxOriginals {
		fl.originals = fl.originals[1:]
	}
	fl.originals = append(fl.originals, original{raw, msg})
}

// messages is ins as messages. It is called with mu held.
func (fl *Filter) messages(ins []termkey.Input) []tea.Msg {
	var out []tea.Msg
	for _, in := range ins {
		if m := fl.message(in); m != nil {
			out = append(out, m)
		}
	}
	return out
}

// message is in as a message: an InputUnknown as the message it came as,
// or as the Report it is; a paste, a request for the clipboard's image or
// a paste; a key, a release or a press. It is called with mu held.
func (fl *Filter) message(in termkey.Input) tea.Msg {
	switch {
	case in.Kind == termkey.InputUnknown:
		raw := string(in.Raw)
		for i, o := range fl.originals {
			if o.raw == raw {
				fl.originals = append(fl.originals[:i], fl.originals[i+1:]...)
				return o.msg
			}
		}
		if r, ok := inputfilter.ParseReport(raw); ok {
			return report(r)
		}
		return nil
	case in.Kind == termkey.InputPaste && fl.f.ClipboardImageRequest(in):
		return inputfilter.ClipboardImageRequestMsg{}
	case in.Kind == termkey.InputPaste:
		return tea.PasteMsg{Content: in.Text}
	case in.Key.Release:
		return tea.KeyReleaseMsg(keymaptea.ToTea(in.Key))
	}
	return tea.KeyPressMsg(keymaptea.ToTea(in.Key))
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

// Capture holds the keys and pastes that come after it until Replay, as
// the Flusher's Capture does. It needs Attach, through which Replay gives
// them.
func (fl *Filter) Capture() { fl.f.Capture() }

// Replay ends Capture, and gives the typing it held, as the Flusher's
// Replay keeps it, after any message the Filter holds, through Attach's
// drains.
func (fl *Filter) Replay() {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	msgs := fl.messages(fl.f.Replay())
	if len(msgs) == 0 {
		return
	}
	fl.held = append(fl.held, msgs...)
	fl.want()
}

// Quarantine drops all input until until, as the Flusher's Quarantine
// does: keys, pastes, mouse and focus messages, those that come and those
// the Filter holds. Other messages pass.
func (fl *Filter) Quarantine(until time.Time) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.f.Quarantine(until)
	fl.quarantine = until
	kept := fl.held[:0]
	for _, m := range fl.held {
		if !isInput(m) {
			kept = append(kept, m)
		}
	}
	fl.held = kept
}

// isInput reports whether msg is the user's input or the terminal's:
// a key, a paste, a mouse or focus message, or a sequence nothing decoded.
func isInput(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.KeyPressMsg, tea.KeyReleaseMsg, tea.PasteMsg, tea.PasteStartMsg, tea.PasteEndMsg,
		tea.MouseMsg, tea.FocusMsg, tea.BlurMsg, inputfilter.ClipboardImageRequestMsg:
		return true
	}
	ev, ok := termevent.Decode(msg)
	return ok && ev.Kind == termevent.Unknown
}

// Attach gives the Filter the program's Send, from a goroutine of its own:
// each drain of a held message, and each flush the Flusher says fell due,
// a wheel sum held or what a stage ended. stop ends the goroutine, closes
// the Flusher and waits, and may be called more than once. It is for the
// program's end, once its Run has returned: what the Filter holds then, a
// character held a few milliseconds or a message waiting for its drain,
// ends with the program, and after it the Filter passes every message as
// it comes. Without Attach, a wheel sum held joins the next wheel event,
// and only pastes reach the stages. A second Attach panics.
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

// x10Suspect reports whether m is how Bubble Tea decodes an X10 report
// ConPTY re-encoded: a column whose byte, plus 33, is a UTF-8 lead byte,
// 0xC2 or 0xC3, and a row whose byte is a continuation, 0x80 to 0xBF
// (docs/decisions/0009-PLAN-streaming-content-engine.md Step A1.3's
// measurement).
func x10Suspect(m tea.Mouse) bool {
	cx, cy := m.X+33, m.Y+33
	return (cx == 0xc2 || cx == 0xc3) && cy >= 0x80 && cy <= 0xbf
}

// x10Raw is the X10 report Bubble Tea read msg from: CSI M, the button's
// byte as x/ansi encodes it, and the column's and row's bytes.
func x10Raw(msg tea.MouseMsg) string {
	m := msg.Mouse()
	_, motion := msg.(tea.MouseMotionMsg)
	cb := ansi.EncodeMouseButton(m.Button, motion,
		m.Mod&tea.ModShift != 0, m.Mod&tea.ModAlt != 0, m.Mod&tea.ModCtrl != 0)
	return "\x1b[M" + string([]byte{cb + 32, byte(m.X + 33), byte(m.Y + 33)}) //nolint:gosec // x10Suspect bounds both
}

// report is r as Bubble Tea's message of its kind.
func report(r inputfilter.Report) tea.Msg {
	m := tea.Mouse{X: r.X, Y: r.Y, Button: tea.MouseButton(r.Button), Mod: tea.KeyMod(r.Mod)}
	switch r.Kind {
	case inputfilter.MouseRelease:
		return tea.MouseReleaseMsg(m)
	case inputfilter.MouseMotion:
		return tea.MouseMotionMsg(m)
	case inputfilter.MouseWheel:
		return tea.MouseWheelMsg(m)
	case inputfilter.FocusIn:
		return tea.FocusMsg{}
	case inputfilter.FocusOut:
		return tea.BlurMsg{}
	}
	return tea.MouseClickMsg(m)
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
