package frame_test

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/frame"
)

// terminal is a writer that blocks while held, and records each write and
// the most writes ever in flight at once.
type terminal struct {
	mu       sync.Mutex
	writes   []string
	inflight int
	most     int
	held     chan struct{} // while open, a write blocks; nil holds none
}

func (w *terminal) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.inflight++
	w.most = max(w.most, w.inflight)
	held := w.held
	w.mu.Unlock()
	if held != nil {
		<-held
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.inflight--
	w.writes = append(w.writes, string(p))
	return len(p), nil
}

func (w *terminal) got() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.writes)
}

// release lets the held write, and every write after it, complete.
func (w *terminal) release() {
	w.mu.Lock()
	defer w.mu.Unlock()
	close(w.held)
	w.held = nil
}

// mostInFlight is the most writes that were ever in flight at once.
func (w *terminal) mostInFlight() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.most
}

// timedWrite writes s to o, and fails the test unless it returns len(s)
// and no error after exactly want.
func timedWrite(t *testing.T, o *frame.Output, s string, want time.Duration) {
	t.Helper()
	start := time.Now()
	n, err := o.Write([]byte(s))
	if n != len(s) || err != nil {
		t.Fatalf("Write(%q) = %d, %v", s, n, err)
	}
	if took := time.Since(start); took != want {
		t.Fatalf("Write(%q) took %v; want %v", s, took, want)
	}
}

// mustEvent fails the test unless want waits on o's channel once the
// bubble has blocked.
func mustEvent(t *testing.T, o *frame.Output, want frame.OutputEvent) {
	t.Helper()
	synctest.Wait()
	select {
	case e := <-o.Events():
		if e != want {
			t.Fatalf("event %v; want %v", e, want)
		}
	default:
		t.Fatalf("no event; want %v", want)
	}
}

// noEvent fails the test when an event waits on o's channel once the
// bubble has blocked.
func noEvent(t *testing.T, o *frame.Output, why string) {
	t.Helper()
	synctest.Wait()
	select {
	case e := <-o.Events():
		t.Fatalf("%s: event %v", why, e)
	default:
	}
}

// TestOutputKeepsUp: on a terminal that keeps up, a Write returns once its
// bytes are written, with no time passing, and each is its own write; the
// bubble ends with no goroutine left (0009-PLAN-streaming-content-engine.md
// D9).
func TestOutputKeepsUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &terminal{}
		o := frame.NewOutput(w)
		for _, s := range []string{"a", "bc", "d"} {
			timedWrite(t, o, s, 0)
			if got := w.got(); got[len(got)-1] != s {
				t.Fatalf("Write(%q) returned before it was written: %q", s, got)
			}
		}
		noEvent(t, o, "a terminal that keeps up")
		if err := o.Close(); err != nil {
			t.Fatalf("Close = %v", err)
		}
	})
}

// TestOutputStalled: on a terminal that stops reading, the first Write
// returns after the write wait and later ones at once; what is written
// meanwhile is one write, after the first, never two at once, and every
// byte is written in order; one Stalled comes 5 s after the write began,
// and one Resumed when it completes (docs/decisions/0009-MADR-streaming-content-engine.md
// A7 item 10, docs/decisions/0009-PLAN-streaming-content-engine.md D9).
func TestOutputStalled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &terminal{held: make(chan struct{})}
		o := frame.NewOutput(w)
		timedWrite(t, o, "first", 100*time.Millisecond)
		var given bytes.Buffer
		given.WriteString("first")
		for _, s := range []string{"b", "c", "d"} {
			timedWrite(t, o, s, 0)
			given.WriteString(s)
		}
		time.Sleep(5*time.Second - 100*time.Millisecond - time.Nanosecond)
		noEvent(t, o, "a nanosecond short of the stall time")
		time.Sleep(time.Nanosecond)
		mustEvent(t, o, frame.Stalled)
		time.Sleep(time.Minute)
		noEvent(t, o, "a minute into the stall")
		w.release()
		mustEvent(t, o, frame.Resumed)
		if got := w.got(); !slices.Equal(got, []string{"first", "bcd"}) {
			t.Fatalf("writes %q; want the first, then the rest as one", got)
		}
		if most := w.mostInFlight(); most != 1 {
			t.Fatalf("%d writes in flight at once; want 1", most)
		}
		timedWrite(t, o, "e", 0)
		given.WriteString("e")
		_ = o.Close()
		if got := strings.Join(w.got(), ""); got != given.String() {
			t.Fatalf("wrote %q; want %q", got, given.String())
		}
	})
}

// TestOutputWithdraws: an event not yet read when its opposite comes is
// taken back, so a program that reads late reads the current state.
func TestOutputWithdraws(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &terminal{held: make(chan struct{})}
		o := frame.NewOutput(w, frame.WithStallAfter(time.Second), frame.WithWriteWait(time.Millisecond))
		timedWrite(t, o, "x", time.Millisecond)
		time.Sleep(time.Second)
		synctest.Wait()
		if len(o.Events()) != 1 {
			t.Fatalf("%d events waiting at the stall; want Stalled", len(o.Events()))
		}
		w.release()
		noEvent(t, o, "a stall over before it was read")
		_ = o.Close()
		if _, ok := <-o.Events(); ok {
			t.Fatal("Events not closed by Close")
		}
	})
}

// failing is a writer whose first write fails, or writes short, and whose
// later writes succeed, so that only a sticky error fails them.
type failing struct {
	short bool
	calls int
}

var errTerminal = errors.New("terminal gone")

func (f *failing) Write(p []byte) (int, error) {
	f.calls++
	switch {
	case f.calls > 1:
		return len(p), nil
	case f.short:
		return len(p) - 1, nil
	}
	return 0, errTerminal
}

// TestOutputErrors: a failed write's error is the next Write's, with 0,
// and Close's; a short write is io.ErrShortWrite; an empty Write writes
// nothing; a Write after Close is ErrClosed, and a second Close returns
// what the first did.
func TestOutputErrors(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		for _, c := range []struct {
			w    *failing
			want error
		}{
			{&failing{}, errTerminal},
			{&failing{short: true}, io.ErrShortWrite},
		} {
			o := frame.NewOutput(c.w)
			if n, err := o.Write(nil); n != 0 || err != nil {
				t.Fatalf("an empty Write = %d, %v", n, err)
			}
			if n, err := o.Write([]byte("ab")); n != 0 || !errors.Is(err, c.want) {
				t.Fatalf("the failing Write = %d, %v; want 0, %v", n, err, c.want)
			}
			if n, err := o.Write([]byte("c")); n != 0 || !errors.Is(err, c.want) {
				t.Fatalf("the next Write = %d, %v; want 0, %v", n, err, c.want)
			}
			if err := o.Close(); !errors.Is(err, c.want) {
				t.Fatalf("Close = %v; want %v", err, c.want)
			}
			if err := o.Close(); !errors.Is(err, c.want) {
				t.Fatalf("a second Close = %v; want %v", err, c.want)
			}
			if _, err := o.Write([]byte("d")); !errors.Is(err, frame.ErrClosed) {
				t.Fatalf("a Write after Close = %v; want ErrClosed", err)
			}
		}
	})
}

// file is a writer with a file descriptor that reads, as a terminal's
// *os.File does.
type file struct {
	bytes.Buffer
	fd uintptr
}

func (f *file) Fd() uintptr { return f.fd }

// TestOutputFile: an Output is a term.File, x/term's interface, which
// Bubble Tea looks for; its Fd and Read are what it wraps', and with
// nothing to forward, ^uintptr(0) and errors.ErrUnsupported
// (docs/decisions/0009-PLAN-streaming-content-engine.md D9).
func TestOutputFile(t *testing.T) {
	var _ interface {
		io.ReadWriteCloser
		Fd() uintptr
	} = (*frame.Output)(nil)
	f := &file{fd: 42}
	f.WriteString("in")
	o := frame.NewOutput(f)
	if fd := o.Fd(); fd != 42 {
		t.Errorf("Fd = %d; want the file's, 42", fd)
	}
	buf := make([]byte, 4)
	if n, err := o.Read(buf); n != 2 || err != nil || string(buf[:n]) != "in" {
		t.Errorf("Read = %d, %v, %q; want the file's", n, err, buf[:n])
	}
	_ = o.Close()
	o = frame.NewOutput(&terminal{})
	if fd := o.Fd(); fd != ^uintptr(0) {
		t.Errorf("Fd with no file = %#x; want ^uintptr(0)", fd)
	}
	if _, err := o.Read(buf); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Read with no reader = %v; want ErrUnsupported", err)
	}
	_ = o.Close()
}
