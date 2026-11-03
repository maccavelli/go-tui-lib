package inputfilter

import (
	"time"

	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// defaultInterval is how often a Wheel and a Throttle pass an event: a
// frame at 60 fps, crush's
// (docs/reports/0003-REPORT-agent-tui-ecosystem-research.md §3.1).
const defaultInterval = 16 * time.Millisecond

// Scroll is a wheel event, or the sum of several: where the pointer is,
// the notches along each axis, and the modifiers held. DY is positive
// down and DX positive right.
type Scroll struct {
	X, Y   int
	DX, DY int
	Mod    termkey.Mod
}

// Wheel sums Scroll events. A change of direction along an axis starts
// the sum again, as crush's filter does. The first event after a quiet
// interval passes at once; later ones pass at most one an interval, each
// carrying the sum of those held since. The pointer and modifiers are the
// latest event's. Its zero value is ready, and it is used on one
// goroutine.
type Wheel struct {
	// Interval is the least time between two events passed: 16 ms when it
	// is zero or less.
	Interval time.Duration
	// EventsPerNotch is how many events a terminal sends for one notch,
	// which 0005's Caps tell (docs/decisions/0009-MADR-streaming-content-engine.md
	// A1 §6): 1 when it is zero or less. Whole notches pass; the events
	// left over carry into the next in the same direction.
	EventsPerNotch int

	sum  Scroll    // the events held, not yet notches passed
	last time.Time // when the last event passed
}

// per is how many events make a notch.
func (w *Wheel) per() int { return max(w.EventsPerNotch, 1) }

// take gives the whole notches the sum holds, and leaves the events over
// in it.
func (w *Wheel) take() (Scroll, bool) {
	n := w.per()
	out := w.sum
	out.DX, out.DY = w.sum.DX/n, w.sum.DY/n
	if out.DX == 0 && out.DY == 0 {
		return Scroll{}, false
	}
	w.sum.DX -= out.DX * n
	w.sum.DY -= out.DY * n
	return out, true
}

// interval is the Wheel's interval.
func (w *Wheel) interval() time.Duration {
	if w.Interval > 0 {
		return w.Interval
	}
	return defaultInterval
}

// Add adds s at now, and gives the event to pass, carrying the sum, and
// true; or false while the interval since the last one has not passed,
// the sum held.
func (w *Wheel) Add(s Scroll, now time.Time) (Scroll, bool) {
	if opposite(w.sum.DX, s.DX) || opposite(w.sum.DY, s.DY) {
		w.sum = Scroll{}
	}
	w.sum = Scroll{X: s.X, Y: s.Y, DX: w.sum.DX + s.DX, DY: w.sum.DY + s.DY, Mod: s.Mod}
	if !w.last.IsZero() && now.Sub(w.last) < w.interval() {
		return Scroll{}, false
	}
	out, ok := w.take()
	if ok {
		w.last = now
	}
	return out, ok
}

// opposite reports whether a and b point opposite ways.
func opposite(a, b int) bool { return a < 0 && b > 0 || a > 0 && b < 0 }

// Residual takes the whole notches held, and reports whether there were
// any.
func (w *Wheel) Residual() (Scroll, bool) { return w.take() }

// Due is when the notches held may pass, an interval after the last event
// passed, and false when no whole notch is held.
func (w *Wheel) Due() (time.Time, bool) {
	n := w.per()
	if w.sum.DX/n == 0 && w.sum.DY/n == 0 {
		return time.Time{}, false
	}
	return w.last.Add(w.interval()), true
}

// Throttle passes one sample an interval, as mouse motion is thinned. Its
// zero value is ready, and it is used on one goroutine.
type Throttle struct {
	// Interval is the least time between two samples passed: 16 ms when it
	// is zero or less.
	Interval time.Duration

	last time.Time
}

// Allow reports whether a sample at now passes: the first does, and then
// one at least an interval after the last that did.
func (t *Throttle) Allow(now time.Time) bool {
	iv := t.Interval
	if iv <= 0 {
		iv = defaultInterval
	}
	if !t.last.IsZero() && now.Sub(t.last) < iv {
		return false
	}
	t.last = now
	return true
}
