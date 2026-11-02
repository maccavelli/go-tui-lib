package frame_test

import (
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/frame"
)

// mustFrame is the frame waiting on s's channel, once every goroutine of
// the bubble has blocked; the test fails when none waits.
func mustFrame(t *testing.T, s *frame.Scheduler) frame.DueMsg {
	t.Helper()
	synctest.Wait()
	select {
	case m, ok := <-s.Frames():
		if !ok {
			t.Fatal("Frames is closed")
		}
		return m
	default:
		t.Fatal("no frame came")
	}
	return frame.DueMsg{}
}

// noFrame fails the test when a frame waits on s's channel.
func noFrame(t *testing.T, s *frame.Scheduler, why string) {
	t.Helper()
	synctest.Wait()
	select {
	case m := <-s.Frames():
		t.Fatalf("%s: a frame came, %+v", why, m)
	default:
	}
}

// TestSchedulerMerges: 1,000 requests from 10 goroutines within one frame
// give one frame, a frame's interval after the last
// (docs/decisions/0009-MADR-streaming-content-engine.md §3).
func TestSchedulerMerges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := frame.NewScheduler()
		defer s.Close()
		s.Request()
		first := mustFrame(t, s)
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				for range 100 {
					s.Request()
				}
			})
		}
		wg.Wait()
		noFrame(t, s, "within the frame's interval")
		time.Sleep(time.Second / 60)
		second := mustFrame(t, s)
		if second.Seq != 2 || second.At.Sub(first.At) != time.Second/60 {
			t.Fatalf("the merged frame is %+v after %+v; want Seq 2, a 60th of a second later", second, first)
		}
		time.Sleep(time.Second)
		noFrame(t, s, "after the merged frame")
	})
}

// TestSchedulerRate: frames come no closer than a second over the frame
// rate: 60 by default, WithFPS's n, at most 120, and 60 for zero or less.
func TestSchedulerRate(t *testing.T) {
	for _, c := range []struct {
		name string
		o    []frame.Option
		fps  int
	}{
		{"default", nil, 60},
		{"120", []frame.Option{frame.WithFPS(120)}, 120},
		{"30", []frame.Option{frame.WithFPS(30)}, 30},
		{"500 is 120", []frame.Option{frame.WithFPS(500)}, 120},
		{"0 is 60", []frame.Option{frame.WithFPS(0)}, 60},
		{"-3 is 60", []frame.Option{frame.WithFPS(-3)}, 60},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s := frame.NewScheduler(c.o...)
				defer s.Close()
				interval := time.Second / time.Duration(c.fps)
				s.Request()
				a := mustFrame(t, s)
				s.Request()
				noFrame(t, s, "at once after a frame")
				time.Sleep(interval - time.Nanosecond)
				noFrame(t, s, "a nanosecond short of the interval")
				time.Sleep(time.Nanosecond)
				if b := mustFrame(t, s); b.At.Sub(a.At) != interval {
					t.Fatalf("frames %v apart; want %v", b.At.Sub(a.At), interval)
				}
			})
		})
	}
}

// TestRequestIn: a frame asked for d from now comes then; the earliest
// deadline asked for wins, whichever was asked first; a Request after it
// comes at once.
func TestRequestIn(t *testing.T) {
	t.Run("alone", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := frame.NewScheduler()
			defer s.Close()
			start := time.Now()
			s.RequestIn(50 * time.Millisecond)
			time.Sleep(49 * time.Millisecond)
			noFrame(t, s, "before its deadline")
			time.Sleep(time.Millisecond)
			if m := mustFrame(t, s); !m.At.Equal(start.Add(50 * time.Millisecond)) {
				t.Fatalf("At %v; want 50 ms after the start", m.At.Sub(start))
			}
		})
	})
	t.Run("a Request after it", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := frame.NewScheduler()
			defer s.Close()
			start := time.Now()
			s.RequestIn(50 * time.Millisecond)
			s.Request()
			if m := mustFrame(t, s); !m.At.Equal(start) {
				t.Fatalf("At %v after the start; want at once", m.At.Sub(start))
			}
		})
	})
	for _, c := range []struct {
		name   string
		first  time.Duration
		second time.Duration
	}{
		{"the earlier first", 50 * time.Millisecond, 80 * time.Millisecond},
		{"the earlier second", 80 * time.Millisecond, 50 * time.Millisecond},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s := frame.NewScheduler()
				defer s.Close()
				start := time.Now()
				s.RequestIn(c.first)
				synctest.Wait() // the goroutine waits for the first deadline
				s.RequestIn(c.second)
				time.Sleep(50 * time.Millisecond)
				if m := mustFrame(t, s); !m.At.Equal(start.Add(50 * time.Millisecond)) {
					t.Fatalf("At %v after the start; want 50 ms", m.At.Sub(start))
				}
				time.Sleep(time.Second)
				noFrame(t, s, "the later deadline")
			})
		})
	}
	t.Run("a negative d is now", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := frame.NewScheduler()
			defer s.Close()
			s.RequestIn(-time.Second)
			mustFrame(t, s)
		})
	})
}

// TestSchedulerSeq: Seq rises by one each frame, from 1, and At is when
// the frame fell due.
func TestSchedulerSeq(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := frame.NewScheduler()
		defer s.Close()
		start := time.Now()
		for i := range uint64(3) {
			s.Request()
			m := mustFrame(t, s)
			if want := start.Add(time.Duration(i) * (time.Second / 60)); m.Seq != i+1 || !m.At.Equal(want) {
				t.Fatalf("frame %d is %+v; want Seq %d at %v", i, m, i+1, want.Sub(start))
			}
			time.Sleep(time.Second / 60)
		}
	})
}

// TestSchedulerUnread: a request made while a frame waits unread is served
// by that frame.
func TestSchedulerUnread(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := frame.NewScheduler()
		defer s.Close()
		s.Request()
		synctest.Wait()
		time.Sleep(time.Second)
		s.Request()
		time.Sleep(time.Second)
		if m := mustFrame(t, s); m.Seq != 1 {
			t.Fatalf("the frame read is %+v; want the first, unread one", m)
		}
		noFrame(t, s, "the request made while the first waited")
	})
}

// TestSchedulerClose: Close stops a pending frame and the goroutine that
// waits for it, closes Frames after any frame waiting there, and a second
// Close returns nil; a request after it does nothing. The bubble ends with
// no goroutine left.
func TestSchedulerClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := frame.NewScheduler()
		s.RequestIn(time.Hour)
		synctest.Wait()
		before := time.Now()
		if err := s.Close(); err != nil {
			t.Fatalf("Close = %v", err)
		}
		if waited := time.Since(before); waited != 0 {
			t.Fatalf("Close waited %v for the pending frame's deadline; want it stopped at once", waited)
		}
		if m, ok := <-s.Frames(); ok {
			t.Fatalf("a frame after Close: %+v", m)
		}
		if err := s.Close(); err != nil {
			t.Fatalf("a second Close = %v", err)
		}
		s.Request()
		s.RequestIn(time.Millisecond)
	})
	synctest.Test(t, func(t *testing.T) {
		s := frame.NewScheduler()
		s.Request()
		synctest.Wait()
		_ = s.Close()
		if m, ok := <-s.Frames(); !ok || m.Seq != 1 {
			t.Fatalf("the frame waiting at Close: %+v, %v; want it", m, ok)
		}
		if _, ok := <-s.Frames(); ok {
			t.Fatal("Frames not closed after the waiting frame")
		}
	})
}
