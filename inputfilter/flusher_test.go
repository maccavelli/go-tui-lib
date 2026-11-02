package inputfilter_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// mustFlush is the time waiting on f's channel once the bubble has
// blocked, or fails the test.
func mustFlush(t *testing.T, f *inputfilter.Flusher) time.Time {
	t.Helper()
	synctest.Wait()
	select {
	case at, ok := <-f.Flushes():
		if !ok {
			t.Fatal("Flushes is closed")
		}
		return at
	default:
		t.Fatal("no flush fell due")
	}
	return time.Time{}
}

func noFlush(t *testing.T, f *inputfilter.Flusher, why string) {
	t.Helper()
	synctest.Wait()
	select {
	case at := <-f.Flushes():
		t.Fatalf("%s: a flush at %v", why, at)
	default:
	}
}

// TestFlusherResidual: the wheel sum held after the last notch falls due
// an interval after the first passed, and Flush takes it; nothing more
// falls due after.
func TestFlusherResidual(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := inputfilter.NewFlusher()
		defer f.Close()
		start := time.Now()
		passed := 0
		for range 100 {
			if s, ok := f.Scroll(down(1, 1), time.Now()); ok {
				passed += s.DY
			}
			time.Sleep(150 * time.Microsecond)
		}
		time.Sleep(time.Until(start.Add(16*time.Millisecond)) - time.Nanosecond)
		noFlush(t, f, "a nanosecond short of the interval")
		time.Sleep(time.Nanosecond)
		at := mustFlush(t, f)
		if !at.Equal(start.Add(16 * time.Millisecond)) {
			t.Fatalf("the flush fell due %v after the first notch; want 16 ms", at.Sub(start))
		}
		fl := f.Flush(time.Now())
		if passed != 1 || !fl.HasScroll || fl.Scroll.DY != 99 || len(fl.Inputs) != 0 {
			t.Fatalf("passed %d, flushed %+v; want 1, then 99", passed, fl)
		}
		time.Sleep(time.Second)
		noFlush(t, f, "after the residual was taken")
	})
}

// TestFlusherBurst: a burst's idle end falls due, and Flush takes it as
// one paste; a key given after the idle end, before Flush, comes after
// the paste it followed (docs/decisions/0009-PLAN-streaming-content-engine.md
// D10a).
func TestFlusherBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := inputfilter.NewFlusher(inputfilter.WithPasteBurst(inputfilter.PasteBurstConfig{GOOS: "linux"}))
		defer f.Close()
		for _, s := range []string{"a", "b", "c"} {
			if out := f.Input(char(s), time.Now()); len(out) != 0 {
				t.Fatalf("%s gave %s; want it held", s, show(out))
			}
			time.Sleep(time.Millisecond)
		}
		time.Sleep(7 * time.Millisecond) // the idle time after "c"
		mustFlush(t, f)
		if out := f.Input(key(termkey.KeyUp, 0), time.Now()); show(out) != "[abc] up" {
			t.Fatalf("an arrow after the idle end gave %q; want the paste, then the arrow", show(out))
		}
		if fl := f.Flush(time.Now()); len(fl.Inputs) != 0 {
			t.Fatalf("the flush after gave %s; want nothing left", show(fl.Inputs))
		}
		for _, s := range []string{"x", "y", "z"} {
			f.Input(char(s), time.Now())
			time.Sleep(time.Millisecond)
		}
		time.Sleep(7 * time.Millisecond)
		mustFlush(t, f)
		if fl := f.Flush(time.Now()); show(fl.Inputs) != "[xyz]" {
			t.Fatalf("Flush gave %q; want [xyz]", show(fl.Inputs))
		}
	})
}

// TestFlusherUnread: with no reader, a time waits on Flushes and serves
// what falls due after it; the sum held joins the next wheel event, as
// crush's filter does.
func TestFlusherUnread(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := inputfilter.NewFlusher()
		defer f.Close()
		for range 3 {
			f.Scroll(down(0, 0), time.Now())
		}
		time.Sleep(time.Second)
		if s, ok := f.Scroll(down(0, 0), time.Now()); !ok || s.DY != 3 {
			t.Fatalf("the next notch gave %+v, %v; want the 2 held and itself", s, ok)
		}
		if len(f.Flushes()) != 1 {
			t.Fatalf("%d times waiting; want the one unread", len(f.Flushes()))
		}
	})
}

// TestFlusherOptions: WithoutWheelCoalescing and WithoutMotionThrottle
// pass everything, WithInterval sets both intervals, and with no paste
// burst Input gives each input back at once; motion is one sample an
// interval by default.
func TestFlusherOptions(t *testing.T) {
	f := inputfilter.NewFlusher(inputfilter.WithoutWheelCoalescing(), inputfilter.WithoutMotionThrottle())
	for i := range 5 {
		now := t0.Add(time.Duration(i) * time.Microsecond)
		if s, ok := f.Scroll(down(0, 0), now); !ok || s.DY != 1 {
			t.Fatalf("WithoutWheelCoalescing held notch %d", i)
		}
		if !f.Motion(now) {
			t.Fatalf("WithoutMotionThrottle dropped sample %d", i)
		}
		if out := f.Input(char("a"), now); show(out) != "a" {
			t.Fatalf("Input with no paste burst gave %q", show(out))
		}
	}
	_ = f.Close()
	f = inputfilter.NewFlusher(inputfilter.WithInterval(40 * time.Millisecond))
	defer f.Close()
	if !f.Motion(t0) || f.Motion(t0.Add(39*time.Millisecond)) || !f.Motion(t0.Add(40*time.Millisecond)) {
		t.Fatal("WithInterval(40ms) did not set motion's interval")
	}
	f.Scroll(down(0, 0), t0)
	if _, ok := f.Scroll(down(0, 0), t0.Add(39*time.Millisecond)); ok {
		t.Fatal("WithInterval(40ms) did not set the wheel's interval")
	}
	d := inputfilter.NewFlusher(inputfilter.WithInterval(0))
	defer d.Close()
	if !d.Motion(t0) || d.Motion(t0.Add(15*time.Millisecond)) || !d.Motion(t0.Add(16*time.Millisecond)) {
		t.Fatal("motion is not one sample every 16 ms by default")
	}
}

// TestFlusherClose: Close stops the goroutine waiting for a deadline at
// once, closes Flushes, and a second Close returns nil; what the stages
// hold stays for Flush. The bubble ends with no goroutine left.
func TestFlusherClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := inputfilter.NewFlusher(inputfilter.WithInterval(time.Hour))
		f.Scroll(down(0, 0), time.Now())
		f.Scroll(down(0, 0), time.Now())
		synctest.Wait()
		before := time.Now()
		if err := f.Close(); err != nil {
			t.Fatalf("Close = %v", err)
		}
		if waited := time.Since(before); waited != 0 {
			t.Fatalf("Close waited %v for the deadline; want it stopped at once", waited)
		}
		if _, ok := <-f.Flushes(); ok {
			t.Fatal("Flushes not closed by Close")
		}
		if err := f.Close(); err != nil {
			t.Fatalf("a second Close = %v", err)
		}
		if fl := f.Flush(time.Now().Add(time.Hour)); !fl.HasScroll || fl.Scroll.DY != 1 {
			t.Fatalf("Flush after Close gave %+v; want the sum held", fl)
		}
		f.Scroll(down(0, 0), time.Now().Add(2*time.Hour))
		f.Scroll(down(0, 0), time.Now().Add(2*time.Hour))
	})
}
