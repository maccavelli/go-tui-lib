package inputfilter

import (
	"sync"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// Option configures NewFlusher. It is opaque: only this package's
// functions make one.
type Option interface{ apply(*Flusher) }

// optionFunc is a function used as an Option.
type optionFunc func(*Flusher)

func (f optionFunc) apply(fl *Flusher) { f(fl) }

// WithInterval passes at most one wheel event and one motion sample an
// interval d: 16 ms by default, and zero or less keeps it.
func WithInterval(d time.Duration) Option {
	return optionFunc(func(f *Flusher) {
		if d > 0 {
			f.wheel.Interval, f.motion.Interval = d, d
		}
	})
}

// WithoutWheelCoalescing passes every wheel event as it comes.
func WithoutWheelCoalescing() Option {
	return optionFunc(func(f *Flusher) { f.noWheel = true })
}

// WithoutMotionThrottle passes every motion sample.
func WithoutMotionThrottle() Option {
	return optionFunc(func(f *Flusher) { f.noMotion = true })
}

// WithPasteBurst turns paste-burst detection on, with c's thresholds. It
// is off by default (docs/decisions/0009-MADR-streaming-content-engine.md
// Q3).
func WithPasteBurst(c PasteBurstConfig) Option {
	return optionFunc(func(f *Flusher) { f.burst = NewPasteBurst(c) })
}

// WithWheelProfile is how many wheel events the terminal sends for one
// notch: 1 by default, and zero or less keeps it.
func WithWheelProfile(eventsPerNotch int) Option {
	return optionFunc(func(f *Flusher) { f.wheel.EventsPerNotch = eventsPerNotch })
}

// WithoutReassembly leaves a report or a reply split past the reader's
// timeout as it came.
func WithoutReassembly() Option {
	return optionFunc(func(f *Flusher) { f.noReassembly = true })
}

// WithReassemblyWindow is how long a head waits for each next byte:
// 500 ms by default, and zero or less keeps it.
func WithReassemblyWindow(d time.Duration) Option {
	return optionFunc(func(f *Flusher) { f.reassembly.Window = d })
}

// WithReplies drops the replies m knows, which a terminal sends to a
// probe, when they come whole or rejoined. None is dropped by default.
func WithReplies(m ReplyMatcher) Option {
	return optionFunc(func(f *Flusher) { f.reassembly.Replies = m })
}

// WithoutX10Repair leaves an X10 report ConPTY re-encoded as it came.
func WithoutX10Repair() Option {
	return optionFunc(func(f *Flusher) { f.reassembly.NoX10 = true })
}

// WithoutPasteNormalization leaves a paste's CR and CR LF as they came.
func WithoutPasteNormalization() Option {
	return optionFunc(func(f *Flusher) { f.noNormalize = true })
}

// WithoutClipboardImageRequest leaves an empty bracketed paste an empty
// paste, which ClipboardImageRequest then does not report.
func WithoutClipboardImageRequest() Option {
	return optionFunc(func(f *Flusher) { f.noClipboard = true })
}

// Flush is what a Flusher held that has fallen due: inputs its stages
// released, in order, and a wheel sum.
type Flush struct {
	Inputs    []termkey.Input
	Scroll    Scroll
	HasScroll bool
}

// Flusher is the driver over a Wheel, a Throttle, and the input stages:
// a Reassembler, the typeahead, paste normalisation and, with
// WithPasteBurst, a PasteBurst, each one's output the next one's input.
// Its methods are safe for concurrent use.
//
// When something it holds falls due, a goroutine of the Flusher's sends
// the time on Flushes, and the caller takes what is due with Flush, on
// its own goroutine. Flushes has capacity one: a time waiting there
// unread serves every flush after it. Nothing leaves the stages but by a
// call, so a key given after a burst's idle end finds the burst still
// held, and comes after it. With no reader, a wheel sum held joins the
// next wheel event, as crush's filter does. Between deadlines no goroutine
// of the Flusher's runs.
type Flusher struct {
	noWheel, noMotion bool
	noReassembly      bool
	noNormalize       bool
	noClipboard       bool
	reassembly        ReassembleConfig
	flushes           chan time.Time
	moved             chan struct{} // the deadline changed, for the waiting goroutine
	stop              chan struct{} // closed by Close
	wg                sync.WaitGroup

	mu         sync.Mutex
	wheel      Wheel
	motion     Throttle
	reasm      *Reassembler // nil without reassembly
	burst      *PasteBurst  // nil when detection is off
	capturing  bool
	captured   []termkey.Input
	quarantine time.Time // input before it is dropped
	deadline   time.Time // when the next flush falls due
	pending    bool      // a flush is due
	running    bool      // the goroutine waits for the deadline
	closed     bool
}

// NewFlusher is a Flusher that sums the wheel and throttles motion, at
// 16 ms, joins split reports, repairs X10 reports and normalises pastes,
// and detects no paste bursts.
func NewFlusher(o ...Option) *Flusher {
	f := &Flusher{
		flushes: make(chan time.Time, 1),
		moved:   make(chan struct{}, 1),
		stop:    make(chan struct{}),
	}
	for _, opt := range o {
		opt.apply(f)
	}
	if !f.noReassembly {
		f.reasm = NewReassembler(f.reassembly)
	}
	return f
}

// quarantined reports whether input at now is dropped. It is called with
// mu held.
func (f *Flusher) quarantined(now time.Time) bool { return now.Before(f.quarantine) }

// Scroll gives s at now to the wheel sum, and gives the event to pass, and
// true, or false while it is held, or quarantined.
func (f *Flusher) Scroll(s Scroll, now time.Time) (Scroll, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case f.quarantined(now):
		return Scroll{}, false
	case f.noWheel:
		return s, true
	}
	out, ok := f.wheel.Add(s, now)
	f.arm()
	return out, ok
}

// Motion reports whether a motion sample at now passes.
func (f *Flusher) Motion(now time.Time) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case f.quarantined(now):
		return false
	case f.noMotion:
		return true
	}
	return f.motion.Allow(now)
}

// Input gives in at now to the stages, and gives what is to be handled
// now, in order: nothing while it is held, captured or quarantined, and
// what the stages release otherwise, which may include in.
func (f *Flusher) Input(in termkey.Input, now time.Time) []termkey.Input {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.quarantined(now) {
		return nil
	}
	var out []termkey.Input
	if f.reasm == nil {
		out = f.after(in, now, out)
	} else {
		switch f.reasm.Key(in, now) {
		case Pass:
			out = f.after(in, now, out)
		case Release:
			for r, ok := f.reasm.Tick(now); ok; r, ok = f.reasm.Tick(now) {
				out = f.after(r, now, out)
			}
		}
	}
	f.arm()
	return out
}

// after gives in to the stages after the Reassembler, and appends what
// they release to out. It is called with mu held.
func (f *Flusher) after(in termkey.Input, now time.Time, out []termkey.Input) []termkey.Input {
	if f.capturing && (in.Kind == termkey.InputKey || in.Kind == termkey.InputPaste) {
		if len(f.captured) < maxCapture {
			f.captured = append(f.captured, in)
		}
		return out
	}
	if in.Kind == termkey.InputPaste && !f.noNormalize {
		in.Text = NormalizePaste(in.Text)
	}
	if f.burst == nil {
		return append(out, in)
	}
	switch f.burst.Key(in, now) {
	case Pass:
		out = append(out, in)
	case Release:
		out = f.ready(now, out)
	}
	return out
}

// Flush takes what has fallen due by now: the inputs the stages release,
// and a wheel sum held for its interval.
func (f *Flusher) Flush(now time.Time) Flush {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out Flush
	if due, ok := f.wheel.Due(); ok && !now.Before(due) {
		out.Scroll, out.HasScroll = f.wheel.Residual()
	}
	if f.reasm != nil {
		for r, ok := f.reasm.Tick(now); ok; r, ok = f.reasm.Tick(now) {
			out.Inputs = f.after(r, now, out.Inputs)
		}
	}
	if f.burst != nil {
		out.Inputs = f.ready(now, out.Inputs)
	}
	f.arm()
	return out
}

// ready appends what the paste burst gives at now to out. It is called
// with mu held.
func (f *Flusher) ready(now time.Time, out []termkey.Input) []termkey.Input {
	for in, ok := f.burst.Tick(now); ok; in, ok = f.burst.Tick(now) {
		out = append(out, in)
	}
	return out
}

// Capture holds the keys and pastes given after it, as a program does
// while it starts or loads, until Replay
// (docs/decisions/0009-MADR-streaming-content-engine.md A1 §6). It holds
// at most 4096, which bounds untrusted input. The terminal's reports pass.
func (f *Flusher) Capture() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.capturing, f.captured = true, nil
}

// Replay ends Capture, and gives the typing it held, as Typeahead keeps
// it, each paste normalised.
func (f *Flusher) Replay() []termkey.Input {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := Typeahead(f.captured)
	f.capturing, f.captured = false, nil
	if !f.noNormalize {
		for i, in := range out {
			if in.Kind == termkey.InputPaste {
				out[i].Text = NormalizePaste(in.Text)
			}
		}
	}
	return out
}

// Quarantine drops all input until until, as a program does before a
// screen that asks for a security decision, so that keys typed before it
// cannot answer it (codex tui/src/tui/input_boundary.rs:25-64). What the
// stages hold is dropped with it.
func (f *Flusher) Quarantine(until time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.quarantine = until
	f.wheel.sum = Scroll{}
	if f.reasm != nil {
		f.reasm.drop()
	}
	if f.burst != nil {
		f.burst.drop()
	}
	f.captured = nil
	f.arm()
}

// ClipboardImageRequest reports whether in, an input the Flusher gave, is
// a request for the clipboard's image: an empty bracketed paste, unless
// WithoutClipboardImageRequest built the Flusher
// (docs/decisions/0009-PLAN-streaming-content-engine.md D15).
func (f *Flusher) ClipboardImageRequest(in termkey.Input) bool {
	return !f.noClipboard && in.Kind == termkey.InputPaste && in.Text == ""
}

// Flushes is the channel the times of due flushes come on. Close closes
// it.
func (f *Flusher) Flushes() <-chan time.Time { return f.flushes }

// due is the earliest time a stage's flush falls due. It is called with
// mu held.
func (f *Flusher) due() (time.Time, bool) {
	t, ok := f.wheel.Due()
	earliest := func(d time.Time, dok bool) {
		if dok && (!ok || d.Before(t)) {
			t, ok = d, true
		}
	}
	if f.reasm != nil {
		earliest(f.reasm.Due())
	}
	if f.burst != nil {
		earliest(f.burst.Due())
	}
	return t, ok
}

// arm sets the deadline from the stages, and starts the goroutine that
// waits for it, or tells the one waiting that it moved. It is called with
// mu held.
func (f *Flusher) arm() {
	if f.closed {
		return
	}
	due, ok := f.due()
	if ok == f.pending && due.Equal(f.deadline) {
		return
	}
	f.deadline, f.pending = due, ok
	if f.running {
		select {
		case f.moved <- struct{}{}:
		default:
		}
		return
	}
	if ok {
		f.running = true
		f.wg.Go(f.wait)
	}
}

// wait is the goroutine that sends the deadline once it falls due, and
// then stops; it stops too when nothing is due any longer.
func (f *Flusher) wait() {
	for {
		f.mu.Lock()
		if f.closed || !f.pending {
			f.running = false
			f.mu.Unlock()
			return
		}
		d := time.Until(f.deadline)
		if d <= 0 {
			select {
			case f.flushes <- f.deadline:
			default: // a time unread serves this flush
			}
			f.pending, f.running = false, false
			f.mu.Unlock()
			return
		}
		f.mu.Unlock()
		t := time.NewTimer(d)
		select {
		case <-t.C:
		case <-f.moved:
			t.Stop()
		case <-f.stop:
			t.Stop()
			return
		}
	}
}

// Close stops the Flusher's goroutine and closes Flushes. What the stages
// hold stays, and Flush still takes it. A second Close returns nil, as the
// first does.
func (f *Flusher) Close() error {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return nil
	}
	f.closed = true
	f.mu.Unlock()
	close(f.stop)
	f.wg.Wait()
	close(f.flushes)
	return nil
}
