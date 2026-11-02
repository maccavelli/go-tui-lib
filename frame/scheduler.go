package frame

import (
	"sync"
	"time"
)

// DueMsg is a frame that fell due.
type DueMsg struct {
	At  time.Time // when the frame fell due
	Seq uint64    // one more for each frame delivered, from 1
}

// The frame rate a Scheduler applies, as Bubble Tea's renderer bounds it
// (charm.land/bubbletea/v2 renderer.go: 60 by default, at most 120).
const (
	defaultFPS = 60
	maxFPS     = 120
)

// Option configures NewScheduler. It is opaque: only this package's
// functions make one.
type Option interface{ apply(*Scheduler) }

// optionFunc is a function used as an Option.
type optionFunc func(*Scheduler)

func (f optionFunc) apply(s *Scheduler) { f(s) }

// WithFPS delivers at most n frames a second: 60 by default; n over 120 is
// 120, and zero or less keeps 60.
func WithFPS(n int) Option {
	return optionFunc(func(s *Scheduler) {
		if n > 0 {
			s.fps = min(n, maxFPS)
		}
	})
}

// Scheduler merges requests for a frame. Its methods are safe for
// concurrent use. Frames come on Frames, a channel of capacity one: a
// request made while a frame waits there unread is served by that frame,
// and frames come no closer than a frame-rate's interval apart. Between
// frames no goroutine of the Scheduler's runs; one is started when a frame
// falls due later than now, and stops once it is sent.
type Scheduler struct {
	fps    int
	frames chan DueMsg
	moved  chan struct{} // the deadline moved earlier, for the waiting goroutine
	stop   chan struct{} // closed by Close
	wg     sync.WaitGroup

	mu       sync.Mutex
	deadline time.Time // when the pending frame falls due
	pending  bool      // a frame is due
	running  bool      // the goroutine waits for the deadline
	last     time.Time // when the last frame was sent
	seq      uint64
	closed   bool
}

// NewScheduler is a Scheduler with no frame due.
func NewScheduler(o ...Option) *Scheduler {
	s := &Scheduler{
		fps:    defaultFPS,
		frames: make(chan DueMsg, 1),
		moved:  make(chan struct{}, 1),
		stop:   make(chan struct{}),
	}
	for _, opt := range o {
		opt.apply(s)
	}
	return s
}

// interval is the least time between two frames.
func (s *Scheduler) interval() time.Duration { return time.Second / time.Duration(s.fps) }

// Request asks for a frame as soon as the frame rate allows.
func (s *Scheduler) Request() { s.RequestIn(0) }

// RequestIn asks for a frame d from now, or as soon after as the frame
// rate allows; the earliest deadline asked for wins. After Close it does
// nothing.
func (s *Scheduler) RequestIn(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	due := time.Now().Add(max(d, 0))
	if !s.last.IsZero() {
		if earliest := s.last.Add(s.interval()); due.Before(earliest) {
			due = earliest
		}
	}
	if s.pending && !due.Before(s.deadline) {
		return
	}
	s.deadline, s.pending = due, true
	if s.running {
		select {
		case s.moved <- struct{}{}:
		default:
		}
		return
	}
	s.running = true
	s.wg.Go(s.wait)
}

// wait is the goroutine that sends the pending frame once it falls due,
// and then stops.
func (s *Scheduler) wait() {
	for {
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return
		}
		d := time.Until(s.deadline)
		if d <= 0 {
			s.send()
			s.pending, s.running = false, false
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
		t := time.NewTimer(d)
		select {
		case <-t.C:
		case <-s.moved:
			t.Stop()
		case <-s.stop:
			t.Stop()
			return
		}
	}
}

// send puts the due frame on Frames, unless one waits there unread, which
// serves this request too. It is called with mu held.
func (s *Scheduler) send() {
	select {
	case s.frames <- DueMsg{At: s.deadline, Seq: s.seq + 1}:
		s.seq++
		s.last = time.Now()
	default:
	}
}

// Frames is the channel the frames come on. Close closes it.
func (s *Scheduler) Frames() <-chan DueMsg { return s.frames }

// Close stops the Scheduler: a frame pending is not sent, the goroutine
// waiting for it stops, and Frames is closed once it has. A second Close
// returns nil, as the first does.
func (s *Scheduler) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()
	close(s.stop)
	s.wg.Wait()
	close(s.frames)
	return nil
}
