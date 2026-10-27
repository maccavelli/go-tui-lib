package inline_test

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/inline"
)

// blocker is an input whose reads each wait for what the test feeds, and
// that counts its reads and the most of them in flight at once.
type blocker struct {
	mu                    sync.Mutex
	reads, inFlight, most int
	feed                  chan []byte
}

func (b *blocker) Read(p []byte) (int, error) {
	b.mu.Lock()
	b.reads++
	b.inFlight++
	b.most = max(b.most, b.inFlight)
	b.mu.Unlock()
	c, ok := <-b.feed
	b.mu.Lock()
	b.inFlight--
	b.mu.Unlock()
	if !ok {
		return 0, io.EOF
	}
	return copy(p, c), nil
}

func (b *blocker) counts() (reads, most int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.reads, b.most
}

// settled waits up to a second for b's reads to be n, then returns them.
func (b *blocker) settled(n int) int {
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		if got, _ := b.counts(); got == n {
			return got
		}
	}
	got, _ := b.counts()
	return got
}

// steady reports whether b's reads stay n for 100 ms.
func (b *blocker) steady(n int) bool {
	for end := time.Now().Add(100 * time.Millisecond); time.Now().Before(end); time.Sleep(time.Millisecond) {
		if got, _ := b.counts(); got != n {
			return false
		}
	}
	return true
}

// TestReader: nothing is read until asked; a wait that ends leaves its read
// in flight for the next Read, and no second read starts; what a read gave
// is given out before another is asked for; the input's error comes after
// its bytes, and again after that.
func TestReader(t *testing.T) {
	in := &blocker{feed: make(chan []byte)}
	r := inline.NewReader(in)
	if !in.steady(0) {
		t.Fatal("NewReader read before it was asked")
	}
	buf := make([]byte, 8)
	for range 2 {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		n, err := r.Read(ctx, buf)
		cancel()
		if n != 0 || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("a Read whose wait ended: %d, %v", n, err)
		}
	}
	if got := in.settled(1); got != 1 {
		t.Fatalf("%d reads of the input; want the one asked for", got)
	}
	in.feed <- []byte("abc")
	small := make([]byte, 2)
	if n, err := r.Read(t.Context(), small); err != nil || string(small[:n]) != "ab" {
		t.Fatalf("Read = %q, %v; want ab", small[:n], err)
	}
	if n, err := r.Read(t.Context(), small); err != nil || string(small[:n]) != "c" {
		t.Fatalf("Read = %q, %v; want c", small[:n], err)
	}
	if !in.steady(1) {
		t.Fatal("a read of the input started before a Read asked for it")
	}
	done := make(chan error, 1)
	go func() {
		_, err := r.Read(context.Background(), buf)
		done <- err
	}()
	if got := in.settled(2); got != 2 {
		t.Fatalf("%d reads; want a second, asked for", got)
	}
	close(in.feed)
	if err := <-done; !errors.Is(err, io.EOF) {
		t.Fatalf("Read at the end of the input: %v", err)
	}
	if _, err := r.Read(t.Context(), buf); !errors.Is(err, io.EOF) {
		t.Fatalf("Read after the end: %v", err)
	}
	if reads, most := in.counts(); reads != 2 || most != 1 {
		t.Errorf("%d reads, %d at once at most; want 2, and 1", reads, most)
	}
}

// TestNoClose: no file of inline calls Close: the input is never closed,
// since closing console input while a read is pending hangs the process
// (gobble-cli's TestNoCloseOnStdin).
func TestNoClose(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Close" {
					t.Errorf("%s calls Close: the input is never closed", name)
				}
			}
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("no file scanned")
	}
}
