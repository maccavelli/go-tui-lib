package inputfilter_test

import (
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

var t0 = time.Unix(1_000, 0)

func down(x, y int) inputfilter.Scroll { return inputfilter.Scroll{X: x, Y: y, DY: 1} }
func up(x, y int) inputfilter.Scroll   { return inputfilter.Scroll{X: x, Y: y, DY: -1} }

// TestWheelSum: 100 notches within 16 ms give the first at once and the
// other 99 as the residual, so that none is lost, its deadline 16 ms after
// the first (docs/decisions/0009-MADR-streaming-content-engine.md §6).
func TestWheelSum(t *testing.T) {
	var w inputfilter.Wheel
	passed := 0
	for i := range 100 {
		if s, ok := w.Add(down(5, 7), t0.Add(time.Duration(i)*150*time.Microsecond)); ok {
			passed += s.DY
			if i != 0 {
				t.Fatalf("notch %d passed; want only the first", i)
			}
		}
	}
	if due, ok := w.Due(); !ok || !due.Equal(t0.Add(16*time.Millisecond)) {
		t.Fatalf("Due = %v, %v; want 16 ms after the first", due.Sub(t0), ok)
	}
	r, ok := w.Residual()
	if !ok || r.DY != 99 || r.X != 5 || r.Y != 7 || passed != 1 {
		t.Fatalf("passed %d, residual %+v, %v; want 1 and 99 at (5,7)", passed, r, ok)
	}
	if _, ok := w.Residual(); ok {
		t.Fatal("a second Residual gave a sum")
	}
	if _, ok := w.Due(); ok {
		t.Fatal("Due with nothing held")
	}
}

// TestWheelInterval: notches 1 ms apart pass one every 16 ms, each with
// the sum since the last, the pointer and modifiers the latest; a notch
// after a quiet interval passes at once; WithInterval's field sets it.
func TestWheelInterval(t *testing.T) {
	for _, c := range []struct {
		interval, want time.Duration
	}{{0, 16 * time.Millisecond}, {-time.Second, 16 * time.Millisecond}, {40 * time.Millisecond, 40 * time.Millisecond}} {
		w := inputfilter.Wheel{Interval: c.interval}
		var at []time.Duration
		total := 0
		for i := range 100 {
			now := t0.Add(time.Duration(i) * time.Millisecond)
			s := inputfilter.Scroll{X: i, DY: 1, Mod: termkey.ModShift}
			if out, ok := w.Add(s, now); ok {
				at = append(at, now.Sub(t0))
				total += out.DY
				if out.X != i || out.Mod != termkey.ModShift {
					t.Fatalf("interval %v: passed %+v; want the latest pointer", c.interval, out)
				}
			}
		}
		for i, d := range at {
			if d != time.Duration(i)*c.want {
				t.Fatalf("interval %v: passed at %v; want one every %v", c.interval, at, c.want)
			}
		}
		r, _ := w.Residual()
		if total+r.DY != 100 {
			t.Fatalf("interval %v: %d passed and %d held; want 100", c.interval, total, r.DY)
		}
	}
}

// TestWheelReverses: a notch the other way starts the sum again, along
// either axis; a notch on the other axis joins it.
func TestWheelReverses(t *testing.T) {
	var w inputfilter.Wheel
	w.Add(down(0, 0), t0)
	w.Add(down(0, 0), t0.Add(time.Millisecond))
	w.Add(down(0, 0), t0.Add(2*time.Millisecond))
	w.Add(up(0, 0), t0.Add(3*time.Millisecond))
	if r, _ := w.Residual(); r.DY != -1 {
		t.Fatalf("down, down, up held %+v; want DY -1", r)
	}
	w.Add(inputfilter.Scroll{DX: 1}, t0.Add(4*time.Millisecond))
	w.Add(inputfilter.Scroll{DX: -1}, t0.Add(5*time.Millisecond))
	if r, _ := w.Residual(); r.DX != -1 {
		t.Fatalf("right, left held %+v; want DX -1", r)
	}
	w.Add(inputfilter.Scroll{DX: 1}, t0.Add(6*time.Millisecond))
	w.Add(down(0, 0), t0.Add(7*time.Millisecond))
	if r, _ := w.Residual(); r.DX != 1 || r.DY != 1 {
		t.Fatalf("right, down held %+v; want both", r)
	}
}

// TestWheelProfile: with 3 events a notch, 3 events are one notch, the
// events over carry into the next, a change of direction drops them, and
// Residual gives whole notches only (docs/decisions/0009-MADR-streaming-content-engine.md
// A1 §6).
func TestWheelProfile(t *testing.T) {
	w := inputfilter.Wheel{EventsPerNotch: 3}
	var passed []int
	for i := range 7 {
		if s, ok := w.Add(down(0, 0), t0.Add(time.Duration(i)*time.Second)); ok {
			passed = append(passed, s.DY)
		}
	}
	if len(passed) != 2 || passed[0] != 1 || passed[1] != 1 {
		t.Fatalf("7 events a second apart passed %v; want two notches", passed)
	}
	if _, ok := w.Residual(); ok {
		t.Fatal("one event over gave a Residual; want none")
	}
	if _, ok := w.Due(); ok {
		t.Fatal("one event over is due; want nothing due")
	}
	carry := inputfilter.Wheel{EventsPerNotch: 3}
	notches := 0
	for _, at := range []time.Duration{0, 1, 2, 3, 4, 5, 6, 100, 200} {
		if s, ok := carry.Add(down(0, 0), t0.Add(at*time.Millisecond)); ok {
			notches += s.DY
		}
	}
	if notches != 3 {
		t.Fatalf("9 events, 4 of them held within an interval: %d notches; want 3, the 2 over carried", notches)
	}
	w.Add(down(0, 0), t0.Add(10*time.Second))
	w.Add(up(0, 0), t0.Add(11*time.Second))
	w.Add(up(0, 0), t0.Add(12*time.Second))
	if s, ok := w.Add(up(0, 0), t0.Add(13*time.Second)); !ok || s.DY != -1 {
		t.Fatalf("after a reversal, three up gave %+v, %v; want one notch up, the downs dropped", s, ok)
	}
	f := inputfilter.NewFlusher(inputfilter.WithWheelProfile(3), inputfilter.WithoutWheelCoalescing())
	defer f.Close()
	if s, ok := f.Scroll(down(0, 0), t0); !ok || s.DY != 1 {
		t.Fatal("WithoutWheelCoalescing with a profile: want each event as it came")
	}
	g := inputfilter.NewFlusher(inputfilter.WithWheelProfile(3))
	defer g.Close()
	n := 0
	for i := range 6 {
		if s, ok := g.Scroll(down(0, 0), t0.Add(time.Duration(i)*time.Second)); ok {
			n += s.DY
		}
	}
	if n != 2 {
		t.Fatalf("WithWheelProfile(3): 6 events gave %d notches; want 2", n)
	}
}

// TestThrottle: one sample an interval passes, 16 ms by default.
func TestThrottle(t *testing.T) {
	for _, c := range []struct {
		interval, want time.Duration
	}{{0, 16 * time.Millisecond}, {5 * time.Millisecond, 5 * time.Millisecond}} {
		th := inputfilter.Throttle{Interval: c.interval}
		var at []time.Duration
		for i := range 40 {
			now := t0.Add(time.Duration(i) * time.Millisecond)
			if th.Allow(now) {
				at = append(at, now.Sub(t0))
			}
		}
		for i, d := range at {
			if d != time.Duration(i)*c.want {
				t.Fatalf("interval %v: passed at %v; want one every %v", c.interval, at, c.want)
			}
		}
	}
}
