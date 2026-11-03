package stream

import (
	"bytes"
	"sync"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/frame"
)

// Buffer takes a stream's chunks from any goroutine, such as a network
// reader's, and queues each whole line with the time it came, for a Doc to
// Pull on the goroutine that owns it
// (docs/decisions/0009-MADR-streaming-content-engine.md §4, A7 item 9).
// Each Write is taken whole, so a writer's lines keep their order. A
// completed line asks its Scheduler for a frame. Its methods are safe for
// concurrent use.
type Buffer struct {
	s *frame.Scheduler

	mu      sync.Mutex
	q       frame.Queue[string]
	pending []byte // the partial last line
	closed  bool
}

// NewBuffer is a Buffer that asks s for a frame when a line completes,
// and nothing when s is nil.
func NewBuffer(s *frame.Scheduler) *Buffer { return &Buffer{s: s} }

// Write queues p's complete lines, and holds its partial last line for
// the next. After Close it returns 0 and ErrClosed.
func (b *Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return 0, ErrClosed
	}
	now := time.Now()
	b.pending = append(b.pending, p...)
	lines := 0
	for {
		i := bytes.IndexByte(b.pending, '\n')
		if i < 0 {
			break
		}
		b.q.Push(string(b.pending[:i+1]), now)
		b.pending = b.pending[i+1:]
		lines++
	}
	if len(b.pending) == 0 {
		b.pending = nil
	}
	b.mu.Unlock()
	if lines > 0 && b.s != nil {
		b.s.Request()
	}
	return len(p), nil
}

// Close ends the stream: its partial last line is queued, and the Doc that
// Pulls the last of it then calls Finish. A second Close returns nil.
func (b *Buffer) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	if len(b.pending) > 0 {
		b.q.Push(string(b.pending), time.Now())
		b.pending = nil
	}
	b.mu.Unlock()
	if b.s != nil {
		b.s.Request()
	}
	return nil
}

// take gives the lines p releases at now, the partial line when no
// complete one is left queued, and whether lines are still queued and the
// stream has ended.
func (b *Buffer) take(p frame.Pacer, now time.Time) (lines []string, partial string, more, closed bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for l := range b.q.Drain(p, now) {
		lines = append(lines, l)
	}
	more = b.q.Len() > 0
	if !more && len(b.pending) > 0 {
		partial, b.pending = string(b.pending), nil
	}
	return lines, partial, more, b.closed
}

// Pull appends what b's queue releases to d by p at now, Immediate's
// whole queue when p is nil, and the partial line once no complete one is
// queued, which a frame then shows; once b is closed and empty, it calls
// Finish. It reports whether lines are still queued, so that the caller
// asks for another frame.
func (d *Doc) Pull(b *Buffer, p frame.Pacer, now time.Time) bool {
	if p == nil {
		p = frame.Immediate()
	}
	lines, partial, more, closed := b.take(p, now)
	for _, l := range lines {
		d.Append(l)
	}
	d.Append(partial)
	if closed && !more {
		d.Finish()
	}
	return more
}
