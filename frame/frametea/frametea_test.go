package frametea_test

import (
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
