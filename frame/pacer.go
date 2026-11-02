package frame

import "time"

// Pacer decides how many of a queue's items a frame releases, from the
// queue's Snapshot at now.
type Pacer interface {
	Pace(s Snapshot, now time.Time) Release
}

// PacerFunc is a function used as a Pacer.
type PacerFunc func(s Snapshot, now time.Time) Release

// Pace calls f.
func (f PacerFunc) Pace(s Snapshot, now time.Time) Release { return f(s, now) }

// Release is what a Pacer gives a frame: how many items to take, and the
// gear it is in.
type Release struct {
	Take int
	Gear Gear
}

// Immediate is a Pacer that releases the whole queue every frame, in
// CatchUp.
func Immediate() Pacer {
	return PacerFunc(func(s Snapshot, _ time.Time) Release { return Release{Take: s.Depth, Gear: CatchUp} })
}

// The thresholds Adaptive applies where its options leave them, codex's
// (docs/decisions/0009-MADR-streaming-content-engine.md §3).
const (
	defaultCatchUpDepth = 8
	defaultCatchUpAge   = 120 * time.Millisecond
	defaultSmoothDepth  = 2
	defaultSmoothAge    = 40 * time.Millisecond
	defaultHold         = 250 * time.Millisecond
	defaultSevereDepth  = 64
	defaultSevereAge    = 300 * time.Millisecond
)

// AdaptiveOption configures Adaptive. It is opaque: only this package's
// functions make one.
type AdaptiveOption interface{ applyAdaptive(*adaptive) }

// adaptiveFunc is a function used as an AdaptiveOption.
type adaptiveFunc func(*adaptive)

func (f adaptiveFunc) applyAdaptive(a *adaptive) { f(a) }

// intOption is an AdaptiveOption that sets the field field picks to n,
// when n is positive.
func intOption(n int, field func(*adaptive) *int) AdaptiveOption {
	return adaptiveFunc(func(a *adaptive) {
		if n > 0 {
			*field(a) = n
		}
	})
}

// durationOption is an AdaptiveOption that sets the field field picks to
// d, when d is positive.
func durationOption(d time.Duration, field func(*adaptive) *time.Duration) AdaptiveOption {
	return adaptiveFunc(func(a *adaptive) {
		if d > 0 {
			*field(a) = d
		}
	})
}

// WithCatchUpDepth enters CatchUp at n items queued: 8 by default, and
// zero or less keeps it.
func WithCatchUpDepth(n int) AdaptiveOption {
	return intOption(n, func(a *adaptive) *int { return &a.catchUpDepth })
}

// WithCatchUpAge enters CatchUp when the oldest item has waited d: 120 ms
// by default, and zero or less keeps it.
func WithCatchUpAge(d time.Duration) AdaptiveOption {
	return durationOption(d, func(a *adaptive) *time.Duration { return &a.catchUpAge })
}

// WithSmoothDepth leaves CatchUp only while at most n items are queued: 2
// by default, and zero or less keeps it.
func WithSmoothDepth(n int) AdaptiveOption {
	return intOption(n, func(a *adaptive) *int { return &a.smoothDepth })
}

// WithSmoothAge leaves CatchUp only while the oldest item has waited at
// most d: 40 ms by default, and zero or less keeps it.
func WithSmoothAge(d time.Duration) AdaptiveOption {
	return durationOption(d, func(a *adaptive) *time.Duration { return &a.smoothAge })
}

// WithHold is how long the queue must stay shallow and young before
// CatchUp is left, and how long after it is left before it may be entered
// again: 250 ms by default, and zero or less keeps it.
func WithHold(d time.Duration) AdaptiveOption {
	return durationOption(d, func(a *adaptive) *time.Duration { return &a.hold })
}

// WithSevereDepth enters CatchUp within the hold after leaving it, at n
// items queued: 64 by default, and zero or less keeps it.
func WithSevereDepth(n int) AdaptiveOption {
	return intOption(n, func(a *adaptive) *int { return &a.severeDepth })
}

// WithSevereAge enters CatchUp within the hold after leaving it, when the
// oldest item has waited d: 300 ms by default, and zero or less keeps it.
func WithSevereAge(d time.Duration) AdaptiveOption {
	return durationOption(d, func(a *adaptive) *time.Duration { return &a.severeAge })
}

// adaptive is Adaptive's state.
type adaptive struct {
	catchUpDepth, smoothDepth, severeDepth int
	catchUpAge, smoothAge, severeAge, hold time.Duration

	gear      Gear
	calmSince time.Time // in CatchUp, since when the queue has been shallow and young; zero when it is not
	leftAt    time.Time // when CatchUp was last left; zero before
}

// Adaptive is a Pacer in two gears, codex's
// (docs/decisions/0009-MADR-streaming-content-engine.md §3): Smooth
// releases one item a frame; CatchUp releases the whole queue. It enters
// CatchUp when the queue is 8 deep or its oldest item 120 ms old, and
// leaves once at most 2 items and 40 ms have held for 250 ms; after
// leaving, it enters again within 250 ms only for a severe backlog, 64
// items or 300 ms. An empty queue puts it back in Smooth. Every number is
// an option. An Adaptive keeps its state between frames, so each queue has
// its own, used on one goroutine.
func Adaptive(o ...AdaptiveOption) Pacer {
	a := &adaptive{
		catchUpDepth: defaultCatchUpDepth, catchUpAge: defaultCatchUpAge,
		smoothDepth: defaultSmoothDepth, smoothAge: defaultSmoothAge,
		hold:        defaultHold,
		severeDepth: defaultSevereDepth, severeAge: defaultSevereAge,
	}
	for _, opt := range o {
		opt.applyAdaptive(a)
	}
	return a
}

// Pace moves between the gears as Adaptive says, and releases by the gear.
func (a *adaptive) Pace(s Snapshot, now time.Time) Release {
	if s.Depth == 0 {
		a.gear, a.calmSince = Smooth, time.Time{}
		return Release{Gear: Smooth}
	}
	switch a.gear {
	case Smooth:
		deep := s.Depth >= a.catchUpDepth || s.Oldest >= a.catchUpAge
		severe := s.Depth >= a.severeDepth || s.Oldest >= a.severeAge
		held := !a.leftAt.IsZero() && now.Sub(a.leftAt) < a.hold
		if deep && (!held || severe) {
			a.gear, a.calmSince = CatchUp, time.Time{}
		}
	case CatchUp:
		calm := s.Depth <= a.smoothDepth && s.Oldest <= a.smoothAge
		switch {
		case !calm:
			a.calmSince = time.Time{}
		case a.calmSince.IsZero():
			a.calmSince = now
		case now.Sub(a.calmSince) >= a.hold:
			a.gear, a.leftAt, a.calmSince = Smooth, now, time.Time{}
		}
	}
	if a.gear == CatchUp {
		return Release{Take: s.Depth, Gear: CatchUp}
	}
	return Release{Take: 1, Gear: Smooth}
}
