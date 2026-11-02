package frame

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
)

// The times an Output applies where its options leave them: grok's stall
// (docs/reports/0003-REPORT-agent-tui-ecosystem-research.md §8.17), and the
// wait of docs/decisions/0009-PLAN-streaming-content-engine.md D9.
const (
	defaultStallAfter = 5 * time.Second
	defaultWriteWait  = 100 * time.Millisecond
)

// ErrClosed is the error of a Write to a closed Output.
var ErrClosed = errors.New("frame: output closed")

// OutputEvent is a change in whether the terminal takes an Output's
// writes.
type OutputEvent uint8

const (
	// Stalled is a write that has not completed in the stall time. The
	// zero value.
	Stalled OutputEvent = iota
	// Resumed is the stalled write completed.
	Resumed
)

var outputEventNames = enum.Names[OutputEvent]{
	Pkg: pkgName, Type: "OutputEvent",
	Tokens: []string{"stalled", "resumed"},
}

// String is the event's token.
func (e OutputEvent) String() string { return outputEventNames.String(e) }

// MarshalText is the event's token. An event with no token is an error.
func (e OutputEvent) MarshalText() ([]byte, error) { return outputEventNames.Marshal(e) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (e *OutputEvent) UnmarshalText(b []byte) error { return outputEventNames.Unmarshal(b, e) }

// OutputOption configures NewOutput. It is opaque: only this package's
// functions make one.
type OutputOption interface{ applyOutput(*Output) }

// outputFunc is a function used as an OutputOption.
type outputFunc func(*Output)

func (f outputFunc) applyOutput(o *Output) { f(o) }

// WithStallAfter reports a stall when a write has not completed in d: 5 s
// by default, and zero or less keeps it.
func WithStallAfter(d time.Duration) OutputOption {
	return outputFunc(func(o *Output) {
		if d > 0 {
			o.stallAfter = d
		}
	})
}

// WithWriteWait is how long a Write waits for its bytes to be written
// before it returns with them still pending: 100 ms by default, and zero
// or less keeps it.
func WithWriteWait(d time.Duration) OutputOption {
	return outputFunc(func(o *Output) {
		if d > 0 {
			o.writeWait = d
		}
	})
}

// Output wraps a program's terminal output, the writer it gives
// tea.WithOutput, so that a terminal slow to read holds the program no
// longer than a wait (docs/decisions/0009-MADR-streaming-content-engine.md
// A1 §3, A7 item 10; docs/decisions/0009-PLAN-streaming-content-engine.md
// D9).
//
// A goroutine of Output's writes, one write at a time. A Write copies its
// bytes, and returns once they are written or the write wait has passed,
// whichever is first; on a terminal that keeps up, every Write is written
// before it returns, so Bubble Tea's last write lands before it restores
// the terminal. Once a wait has run out, later Writes return at once, until
// a write completes. What is written while a write is in flight is appended to
// one pending buffer, written whole once that write completes: many draws
// become one write, and no byte is dropped, since Bubble Tea's renderer
// writes only the changes from the screen it drew last.
//
// A write that has not completed in the stall time sends Stalled on
// Events, and Resumed once it does. Output has Fd and Read, forwarded to
// what it wraps, so that it is a term.File, as Bubble Tea looks for one to
// find the terminal's size, its resizes and its colours.
//
// Between writes no goroutine of Output's runs. Its methods are safe for
// concurrent use.
type Output struct {
	w          io.Writer
	stallAfter time.Duration
	writeWait  time.Duration
	events     chan OutputEvent
	wg         sync.WaitGroup

	mu       sync.Mutex
	pending  *batch // written after the write in flight
	spare    []byte // the last write's buffer, reused
	running  bool   // the goroutine is writing
	slow     bool   // a Write's wait ran out, and no write has completed since
	inflight uint64 // the write in flight's number; 0 between writes
	writes   uint64 // the writes started
	stalled  bool   // Stalled was sent for the write in flight
	err      error  // the first failed write's, returned from then on
	closed   bool
}

// batch is the bytes of one write, and how it ended: err is set before
// done is closed.
type batch struct {
	buf  []byte
	done chan struct{}
	err  error
}

// NewOutput is an Output that writes to w.
func NewOutput(w io.Writer, o ...OutputOption) *Output {
	out := &Output{
		w:          w,
		stallAfter: defaultStallAfter,
		writeWait:  defaultWriteWait,
		events:     make(chan OutputEvent, 1),
	}
	for _, opt := range o {
		opt.applyOutput(out)
	}
	return out
}

// Write queues p to be written, and waits for it up to the write wait,
// unless an earlier Write's wait ran out with no write completed since. It
// returns len(p), or once a write has failed, 0 and that write's error,
// and after Close, ErrClosed.
func (o *Output) Write(p []byte) (int, error) {
	o.mu.Lock()
	switch {
	case o.closed:
		o.mu.Unlock()
		return 0, ErrClosed
	case o.err != nil:
		err := o.err
		o.mu.Unlock()
		return 0, err
	case len(p) == 0:
		o.mu.Unlock()
		return 0, nil
	}
	if o.pending == nil {
		o.pending = &batch{buf: o.spare, done: make(chan struct{})}
		o.spare = nil
	}
	b := o.pending
	b.buf = append(b.buf, p...)
	if !o.running {
		o.running = true
		o.wg.Go(o.run)
	}
	slow := o.slow
	o.mu.Unlock()
	if slow {
		return len(p), nil
	}
	t := time.NewTimer(o.writeWait)
	defer t.Stop()
	select {
	case <-b.done:
		if b.err != nil {
			return 0, b.err
		}
	case <-t.C:
		o.mu.Lock()
		select {
		case <-b.done:
		default:
			o.slow = true
		}
		o.mu.Unlock()
	}
	return len(p), nil
}

// run is the goroutine that writes the pending buffer, again while more
// is pending, and then stops.
func (o *Output) run() {
	o.mu.Lock()
	defer o.mu.Unlock()
	for o.pending != nil {
		b := o.pending
		o.pending = nil
		o.writes++
		id := o.writes
		o.inflight = id
		o.mu.Unlock()
		stall := time.AfterFunc(o.stallAfter, func() { o.stall(id) })
		n, err := o.w.Write(b.buf)
		stall.Stop()
		if err == nil && n < len(b.buf) {
			err = io.ErrShortWrite
		}
		o.mu.Lock()
		o.inflight, o.slow = 0, false
		if o.stalled {
			o.stalled = false
			o.emit(Resumed)
		}
		b.err = err
		close(b.done)
		o.spare = b.buf[:0]
		if err != nil {
			o.err = err
			if p := o.pending; p != nil {
				p.err = err
				close(p.done)
				o.pending = nil
			}
		}
	}
	o.running = false
}

// stall sends Stalled when write id is still in flight, once.
func (o *Output) stall(id uint64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.inflight == id && !o.stalled {
		o.stalled = true
		o.emit(Stalled)
	}
}

// emit sends e on Events. It is called with mu held. Events alternate, so
// an event still unread there is e's opposite, and is taken back instead:
// what the channel holds is always the current state, and a send never
// blocks.
func (o *Output) emit(e OutputEvent) {
	select {
	case <-o.events:
	default:
		o.events <- e
	}
}

// Events is the channel Stalled and Resumed come on, of capacity one. An
// event not yet received when its opposite comes is withdrawn, so a
// program that reads late reads the current state. Close closes it.
func (o *Output) Events() <-chan OutputEvent { return o.events }

// Fd is the file descriptor of what Output wraps, when that has one, so
// that Bubble Tea finds the terminal through it; otherwise it is
// ^uintptr(0), which names no file.
func (o *Output) Fd() uintptr {
	if f, ok := o.w.(interface{ Fd() uintptr }); ok {
		return f.Fd()
	}
	return ^uintptr(0)
}

// Read reads from what Output wraps, when that is an io.Reader, as a
// term.File does; otherwise it returns errors.ErrUnsupported. Bubble Tea
// never reads its output.
func (o *Output) Read(p []byte) (int, error) {
	if r, ok := o.w.(io.Reader); ok {
		return r.Read(p)
	}
	return 0, errors.ErrUnsupported
}

// Close writes what is pending, waits for the write in flight, closes
// Events, and returns the first failed write's error. It does not close
// what Output wraps. A second Close returns what the first did. A program
// closes its Output after its Bubble Tea program's Run returns.
func (o *Output) Close() error {
	o.mu.Lock()
	first := !o.closed
	o.closed = true
	o.mu.Unlock()
	o.wg.Wait()
	if first {
		close(o.events)
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.err
}
