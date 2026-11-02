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

// Flush is what a Flusher held that has fallen due: inputs a paste burst
// released, in order, and a wheel sum.
type Flush struct {
	Inputs    []termkey.Input
	Scroll    Scroll
	HasScroll bool
}

// Flusher is the driver over a Wheel, a Throttle and, with
// WithPasteBurst, a PasteBurst. Its methods are safe for concurrent use.
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
	flushes           chan time.Time
	moved             chan struct{} // the deadline changed, for the waiting goroutine
	stop              chan struct{} // closed by Close
	wg                sync.WaitGroup

	mu       sync.Mutex
	wheel    Wheel
	motion   Throttle
	burst    *PasteBurst // nil when detection is off
	deadline time.Time   // when the next flush falls due
	pending  bool        // a flush is due
	running  bool        // the goroutine waits for the deadline
	closed   bool
}

// NewFlusher is a Flusher that sums the wheel and throttles motion, at
// 16 ms, and detects no paste bursts.
func NewFlusher(o ...Option) *Flusher {
	f := &Flusher{
		flushes: make(chan time.Time, 1),
		moved:   make(chan struct{}, 1),
		stop:    make(chan struct{}),
	}
	for _, opt := range o {
		opt.apply(f)
	}
	return f
}

// Scroll gives s at now to the wheel sum, and gives the event to pass, and
// true, or false while it is held.
func (f *Flusher) Scroll(s Scroll, now time.Time) (Scroll, bool) {
	if f.noWheel {
		return s, true
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out, ok := f.wheel.Add(s, now)
	f.arm()
	return out, ok
}

// Motion reports whether a motion sample at now passes.
func (f *Flusher) Motion(now time.Time) bool {
	if f.noMotion {
		return true
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.motion.Allow(now)
}

// Input gives in at now to the paste burst, and gives what is to be
// handled now, in order: in alone with detection off, nothing while it is
// held, and what is ready otherwise, which may include in.
func (f *Flusher) Input(in termkey.Input, now time.Time) []termkey.Input {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.burst == nil {
		return []termkey.Input{in}
	}
	var out []termkey.Input
	switch f.burst.Key(in, now) {
	case Pass:
		out = []termkey.Input{in}
	case Release:
		out = f.ready(now)
	}
	f.arm()
	return out
}

// Flush takes what has fallen due by now: the inputs a paste burst
// releases, and a wheel sum held for its interval.
func (f *Flusher) Flush(now time.Time) Flush {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out Flush
	if due, ok := f.wheel.Due(); ok && !now.Before(due) {
		out.Scroll, out.HasScroll = f.wheel.Residual()
	}
	if f.burst != nil {
		out.Inputs = f.ready(now)
	}
	f.arm()
	return out
}

// ready is what the paste burst gives at now. It is called with mu held.
func (f *Flusher) ready(now time.Time) []termkey.Input {
	var out []termkey.Input
	for in, ok := f.burst.Tick(now); ok; in, ok = f.burst.Tick(now) {
		out = append(out, in)
	}
	return out
}

// Flushes is the channel the times of due flushes come on. Close closes
// it.
func (f *Flusher) Flushes() <-chan time.Time { return f.flushes }

// due is the earliest time a stage's flush falls due. It is called with
// mu held.
func (f *Flusher) due() (time.Time, bool) {
	t, ok := f.wheel.Due()
	if f.burst != nil {
		if b, bok := f.burst.Due(); bok && (!ok || b.Before(t)) {
			t, ok = b, true
		}
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
