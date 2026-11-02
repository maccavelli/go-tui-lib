package frametea_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/frame"
	"github.com/maccavelli/go-tui-lib/v2/frame/frametea"
)

// TestWait: Wait's command returns the next frame as a frame.DueMsg, and
// nil once the Scheduler is closed.
func TestWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := frame.NewScheduler()
		s.Request()
		if m, ok := frametea.Wait(s)().(frame.DueMsg); !ok || m.Seq != 1 {
			t.Fatalf("Wait gave %#v; want the first frame", m)
		}
		_ = s.Close()
		if m := frametea.Wait(s)(); m != nil {
			t.Fatalf("Wait after Close gave %#v; want nil", m)
		}
	})
}

// model is a program that redraws once a frame: it asks the Scheduler for
// a frame when its content changes, and re-arms Wait on each.
type model struct {
	frames *frame.Scheduler
	lines  int
}

func (m model) Init() tea.Cmd { return frametea.Wait(m.frames) }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case frame.DueMsg:
		m.lines++ // draw what came since the last frame
		return m, frametea.Wait(m.frames)
	case tea.KeyPressMsg:
		m.frames.Request() // content changed: one frame, however many keys
	}
	return m, nil
}

func (m model) View() tea.View { return tea.NewView("") }

// A Bubble Tea program paced by a Scheduler: Init arms Wait, each
// frame.DueMsg re-arms it, and anything that changes the view asks for a
// frame with Request, from any goroutine.
func ExampleWait() {
	s := frame.NewScheduler()
	defer s.Close()
	_ = tea.NewProgram(model{frames: s})
}

// sent collects what Forward sends.
type sent struct {
	mu   sync.Mutex
	msgs []tea.Msg
}

func (s *sent) send(m tea.Msg) { s.mu.Lock(); s.msgs = append(s.msgs, m); s.mu.Unlock() }

func (s *sent) len() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.msgs) }

// TestForward: Forward sends each frame to send; stop ends it, may be
// called again, and a frame after it waits on Frames unread; with no stop,
// the Scheduler's Close ends it. Each bubble ends with no goroutine left
// (0009-PLAN-streaming-content-engine.md Step 3, D2).
func TestForward(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := frame.NewScheduler()
		defer s.Close()
		var got sent
		stop := frametea.Forward(s, got.send)
		s.Request()
		synctest.Wait()
		time.Sleep(time.Second / 60)
		s.Request()
		synctest.Wait()
		if got.len() != 2 {
			t.Fatalf("Forward sent %d frames; want 2", got.len())
		}
		stop()
		stop()
		time.Sleep(time.Second)
		s.Request()
		synctest.Wait()
		if got.len() != 2 || len(s.Frames()) != 1 {
			t.Fatalf("after stop: %d sent, %d waiting; want 2 sent and the new frame waiting", got.len(), len(s.Frames()))
		}
	})
	synctest.Test(t, func(t *testing.T) {
		s := frame.NewScheduler()
		var got sent
		_ = frametea.Forward(s, got.send)
		s.Request()
		synctest.Wait()
		_ = s.Close()
		if got.len() != 1 {
			t.Fatalf("Forward sent %d frames; want 1", got.len())
		}
	})
}

// TestTickCmd: None arms no timer, a nil command; Slow and Fast tick once,
// after Ticks' interval, as a frame.TickMsg with the demand and the time it
// fired (docs/decisions/0009-MADR-streaming-content-engine.md A1 §3).
func TestTickCmd(t *testing.T) {
	if cmd := frametea.TickCmd(frame.None, frame.Ticks{}); cmd != nil {
		t.Fatal("TickCmd(None) is a command; want nil")
	}
	for _, c := range []struct {
		d    frame.Demand
		t    frame.Ticks
		want time.Duration
	}{
		{frame.Slow, frame.Ticks{}, 83 * time.Millisecond},
		{frame.Fast, frame.Ticks{}, time.Second / 60},
		{frame.Slow, frame.Ticks{Slow: time.Second}, time.Second},
	} {
		synctest.Test(t, func(t *testing.T) {
			start := time.Now()
			m, ok := frametea.TickCmd(c.d, c.t)().(frame.TickMsg)
			if !ok || m.Demand != c.d || m.At.Sub(start) != c.want || time.Since(start) != c.want {
				t.Fatalf("TickCmd(%v) gave %#v after %v; want its tick after %v", c.d, m, time.Since(start), c.want)
			}
		})
	}
}

// TestCoalesceCmd: the command returns a frame.FlushMsg at the deadline a
// Coalescer's Push gave, and at once for one passed.
func TestCoalesceCmd(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var applied [][]int
		c := frame.NewCoalescer(func(b []int) { applied = append(applied, slices.Clone(b)) }, nil, 0)
		start := time.Now()
		due, ok := c.Push(1, start)
		c.Push(2, start)
		if !ok {
			t.Fatal("the first Push gave no deadline")
		}
		m, isFlush := frametea.CoalesceCmd(due)().(frame.FlushMsg)
		if !isFlush || !m.At.Equal(due) || time.Since(start) != 100*time.Millisecond {
			t.Fatalf("CoalesceCmd gave %#v after %v; want the deadline after 100ms", m, time.Since(start))
		}
		c.Flush()
		if len(applied) != 1 || !slices.Equal(applied[0], []int{1, 2}) {
			t.Fatalf("the flush at the deadline applied %v; want [[1 2]]", applied)
		}
		before := time.Now()
		if _, ok := frametea.CoalesceCmd(start)().(frame.FlushMsg); !ok || time.Since(before) != 0 {
			t.Fatal("a passed deadline did not flush at once")
		}
	})
}

// TestWaitUpgrade: the command returns the Upgrader's next result as a
// frame.UpgradeMsg, and nil once it is closed.
func TestWaitUpgrade(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		u := frame.NewUpgrader[string, int]()
		u.Submit("k", 3, func(context.Context) int { return 9 })
		m, ok := frametea.WaitUpgrade(u)().(frame.UpgradeMsg[string, int])
		if !ok || m.Key != "k" || m.Gen != 3 || m.Result != 9 {
			t.Fatalf("WaitUpgrade gave %#v; want {k 3 9}", m)
		}
		_ = u.Close()
		if m := frametea.WaitUpgrade(u)(); m != nil {
			t.Fatalf("WaitUpgrade after Close gave %#v; want nil", m)
		}
	})
}
