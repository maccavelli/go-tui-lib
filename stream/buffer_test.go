package stream_test

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/frame"
	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// TestBufferWriters: 8 goroutines writing lines at once lose no byte, and
// each writer's lines keep their order.
func TestBufferWriters(t *testing.T) {
	b := stream.NewBuffer(nil)
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 200 {
				if _, err := fmt.Fprintf(b, "w%d-%03d\n", g, i); err != nil {
					t.Errorf("Write: %v", err)
				}
			}
		})
	}
	wg.Wait()
	d := stream.NewDoc(echo)
	if d.Pull(b, nil, time.Now()) {
		t.Fatal("Pull with Immediate left lines queued")
	}
	lines := strings.Split(strings.TrimSuffix(d.Source(), "\n"), "\n")
	if len(lines) != 8*200 {
		t.Fatalf("%d lines; want %d", len(lines), 8*200)
	}
	next := map[string]int{}
	for _, l := range lines {
		w, n, _ := strings.Cut(l, "-")
		if want := fmt.Sprintf("%03d", next[w]); n != want {
			t.Fatalf("writer %s's line %s came where %s was due", w, n, want)
		}
		next[w]++
	}
}

// TestBufferFrames: a partial line asks for no frame; the line it
// completes asks for one.
func TestBufferFrames(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := frame.NewScheduler()
		defer s.Close()
		b := stream.NewBuffer(s)
		b.Write([]byte("par"))
		synctest.Wait()
		if len(s.Frames()) != 0 {
			t.Fatal("a partial line asked for a frame")
		}
		b.Write([]byte("tial\n"))
		synctest.Wait()
		if len(s.Frames()) != 1 {
			t.Fatal("a completed line asked for no frame")
		}
	})
}

// TestBufferClose: Close queues the partial last line, and the Pull that
// takes the last of the stream calls Finish; a Write after Close returns
// ErrClosed.
func TestBufferClose(t *testing.T) {
	b := stream.NewBuffer(nil)
	b.Write([]byte("x\ny"))
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte("z")); !errors.Is(err, stream.ErrClosed) {
		t.Fatalf("Write after Close = %v; want ErrClosed", err)
	}
	d := stream.NewDoc(echo)
	d.Pull(b, nil, time.Now())
	if !d.Finished() || d.Source() != "x\ny" {
		t.Fatalf("after the last Pull: finished %v, source %q; want finished, x and y", d.Finished(), d.Source())
	}
}

// TestPullAsksNextFrame: a host that pulls on each frame and asks for
// none itself still drains a closed stream a line a frame, and finishes
// (docs/decisions/0009-PLAN-streaming-content-engine.md D17).
func TestPullAsksNextFrame(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := frame.NewScheduler()
		defer s.Close()
		b := stream.NewBuffer(s)
		d := stream.NewDoc(echo)
		b.Write([]byte("a\nb\nc\nd\ne\n"))
		b.Close()
		frames := 0
		for !d.Finished() {
			select {
			case m := <-s.Frames():
				frames++
				d.Pull(b, one, m.At)
			case <-time.After(5 * time.Second):
				t.Fatalf("no frame in 5 s after %d: source %q, not finished", frames, d.Source())
			}
		}
		if frames != 5 || d.Source() != "a\nb\nc\nd\ne\n" {
			t.Errorf("finished after %d frames with %q; want 5 frames, a to e", frames, d.Source())
		}
	})
}

// one is a Pacer that releases one line a frame.
var one = frame.PacerFunc(func(frame.Snapshot, time.Time) frame.Release { return frame.Release{Take: 1} })

// TestPullQueued: the partial line is hidden while complete lines are
// queued, and shown once none is; Pull reports the lines left.
func TestPullQueued(t *testing.T) {
	b := stream.NewBuffer(nil)
	b.Write([]byte("a\nb\nc\npart"))
	d := stream.NewDoc(echo)
	now := time.Now()
	for i, want := range []struct {
		more  bool
		frame string
	}{{true, "a"}, {true, "a\nb"}, {false, "a\nb\nc\npart"}} {
		v := d.Version()
		if more := d.Pull(b, one, now); more != want.more {
			t.Fatalf("Pull %d reported %v; want %v", i, more, want.more)
		}
		if f := d.Render(80); f != want.frame || d.Version() == v {
			t.Fatalf("after Pull %d the frame is %q, version %d after %d; want %q and a new version", i, f, d.Version(), v, want.frame)
		}
	}
}
