package stream_test

import (
	"bytes"
	"errors"
	"io"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// printed is what a Printer made with o gives for in, written whole and
// flushed.
func printed(in string, o ...stream.Option) string {
	p := stream.NewPrinter(o...)
	return p.Write(in) + p.Flush()
}

// written is what a Writer made with o writes for r, copied by io.Copy
// and closed.
func written(t *testing.T, r io.Reader, o ...stream.Option) string {
	t.Helper()
	var b bytes.Buffer
	w := stream.NewWriter(&b, o...)
	if _, err := io.Copy(w, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return b.String()
}

// TestWriterCopy: io.Copy of a reply into a Writer, read a byte at a time,
// half a read at a time, or whole, writes what a Printer prints for it,
// byte for byte, at each width, with and without styling
// (docs/decisions/0009-MADR-streaming-content-engine.md A8).
func TestWriterCopy(t *testing.T) {
	reply := string(readReply(t))
	readers := map[string]func(string) io.Reader{
		"one byte": func(s string) io.Reader { return iotest.OneByteReader(strings.NewReader(s)) },
		"half":     func(s string) io.Reader { return iotest.HalfReader(strings.NewReader(s)) },
		"whole":    func(s string) io.Reader { return bytes.NewReader([]byte(s)) },
	}
	for name, reader := range readers {
		for _, width := range []int{0, 60, 100} {
			for _, markup := range []stream.Markup{{}, stream.DefaultMarkup()} {
				o := []stream.Option{stream.WithWidth(width), stream.WithMarkup(markup)}
				if got, want := written(t, reader(reply), o...), printed(reply, o...); got != want {
					t.Errorf("%s, width %d, styled %v: the Writer's output differs from the Printer's\n got %q\nwant %q",
						name, width, markup != (stream.Markup{}), got[:min(len(got), 200)], want[:min(len(want), 200)])
				}
			}
		}
	}
}

// TestWriterDefaults: with no option, a Writer has the Printer's defaults,
// a table laid out; with WithoutTableLayout, its output is the input
// cleaned by safetext.Text, byte for byte (0009-PLAN Step A8.1, D4).
func TestWriterDefaults(t *testing.T) {
	in := "# Title\x1b[2J\n\n| a | bb |\n|---|---|\n| 1 | 2 |\n\nThe end."
	if got, want := written(t, strings.NewReader(in)), printed(in); got != want {
		t.Errorf("no option: %q; want the Printer's, %q", got, want)
	}
	if got, want := written(t, strings.NewReader(in), stream.WithoutTableLayout()), safetext.Clean(in, safetext.Text()); got != want {
		t.Errorf("WithoutTableLayout: %q; want the input cleaned, %q", got, want)
	}
}

// TestWriterString: WriteString writes what Write does.
func TestWriterString(t *testing.T) {
	var a, b bytes.Buffer
	wa, wb := stream.NewWriter(&a), stream.NewWriter(&b)
	for _, s := range []string{"a first line\nand ", "a second\n\n", "- a list\n"} {
		if _, err := wa.Write([]byte(s)); err != nil {
			t.Fatal(err)
		}
		if n, err := wb.WriteString(s); err != nil || n != len(s) {
			t.Fatalf("WriteString(%q) = %d, %v; want %d, nil", s, n, err, len(s))
		}
	}
	_, _ = wa.Close(), wb.Close()
	if a.String() != b.String() {
		t.Fatalf("Write gave %q, WriteString %q", a.String(), b.String())
	}
}

// TestCRLF: under WithCRLF, every line a Printer or a Writer gives, and
// every row of a wrapped one, ends with CR LF, and no LF stands alone.
func TestCRLF(t *testing.T) {
	in := "A paragraph long enough to be wrapped at twenty columns.\n\n- an item\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\n```\ncode\n```\nend\n"
	at := stream.NewPrinter(stream.WithCRLF(), stream.WithWidth(80))
	at.Write(in)
	at.Flush()
	for name, out := range map[string]string{
		"Printer":    printed(in, stream.WithCRLF(), stream.WithWidth(20)),
		"Writer":     written(t, strings.NewReader(in), stream.WithCRLF(), stream.WithWidth(20)),
		"Printer.At": at.At(20),
	} {
		if strings.Count(out, "\n") < 8 || strings.Count(out, "\n") != strings.Count(out, "\r\n") || strings.Count(out, "\r") != strings.Count(out, "\r\n") {
			t.Errorf("%s: %q; want every line ending CR LF, and no lone CR or LF", name, out)
		}
		if lf := printed(in, stream.WithWidth(20)); strings.ReplaceAll(out, "\r\n", "\n") != lf {
			t.Errorf("%s: %q is not the LF output with CR LF for LF, %q", name, out, lf)
		}
	}
}

// failAfter takes n bytes, then fails every write.
type failAfter struct {
	n    int
	took int
}

var errFull = errors.New("full")

func (f *failAfter) Write(p []byte) (int, error) {
	if f.took+len(p) > f.n {
		return 0, errFull
	}
	f.took += len(p)
	return len(p), nil
}

// TestWriterError: the destination's error is sticky. The Write whose own
// write fails returns len(p) and the error; every later Write returns 0 and
// it, and Close returns it; a second Close returns nil.
func TestWriterError(t *testing.T) {
	w := stream.NewWriter(&failAfter{n: 10})
	if n, err := w.Write([]byte("short\n\n")); n != 7 || err != nil {
		t.Fatalf("a write within the bound = %d, %v; want 7, nil", n, err)
	}
	if n, err := w.Write([]byte("a longer line, past the bound\n\n")); n != 31 || !errors.Is(err, errFull) {
		t.Fatalf("the failing write = %d, %v; want 31 and the destination's error", n, err)
	}
	if n, err := w.Write([]byte("more\n")); n != 0 || !errors.Is(err, errFull) {
		t.Fatalf("a write after the failure = %d, %v; want 0 and the same error", n, err)
	}
	if err := w.Close(); !errors.Is(err, errFull) {
		t.Fatalf("Close = %v; want the destination's error", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("a second Close = %v; want nil", err)
	}
}

// shortWriter takes one byte less than each write, and reports no error.
type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return max(len(p)-1, 0), nil }

// TestWriterShortWrite: a destination that takes less than it is given,
// with no error, is io.ErrShortWrite.
func TestWriterShortWrite(t *testing.T) {
	w := stream.NewWriter(shortWriter{})
	if _, err := w.Write([]byte("a line\n\n")); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("a short write = %v; want io.ErrShortWrite", err)
	}
}

// closeCounter counts its Close calls.
type closeCounter struct {
	bytes.Buffer
	closes int
}

func (c *closeCounter) Close() error { c.closes++; return nil }

// TestWriterClose: Close writes what the Printer holds, leaves the
// destination open, and a Write after it returns ErrClosed.
func TestWriterClose(t *testing.T) {
	dst := &closeCounter{}
	w := stream.NewWriter(dst)
	if _, err := w.Write([]byte("a paragraph's last line, held")); err != nil {
		t.Fatal(err)
	}
	if dst.Len() != 0 {
		t.Fatalf("the held line was written before Close: %q", dst.String())
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if got := dst.String(); got != "a paragraph's last line, held" {
		t.Fatalf("after Close: %q; want the held line", got)
	}
	if dst.closes != 0 {
		t.Fatalf("Close closed the destination %d times", dst.closes)
	}
	if n, err := w.Write([]byte("x")); n != 0 || !errors.Is(err, stream.ErrClosed) {
		t.Fatalf("Write after Close = %d, %v; want 0, ErrClosed", n, err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("a second Close = %v; want nil", err)
	}
}

// FuzzWriter: any split of any input, written to a Writer, writes what a
// Printer prints for the whole.
func FuzzWriter(f *testing.F) {
	f.Add("# Title\n\nA paragraph\nwith two lines.\n\n| a | b |\n|---|---|\n| 1 | 2 |\n", uint64(1), uint8(40))
	f.Add("```go\nfunc f() {}\n```\n- a\n  - b\n> quote\x1b[31m red\n", uint64(7), uint8(0))
	f.Fuzz(func(t *testing.T, in string, seed uint64, width uint8) {
		o := []stream.Option{stream.WithWidth(int(width % 120)), stream.WithMarkup(stream.DefaultMarkup())}
		var b bytes.Buffer
		w := stream.NewWriter(&b, o...)
		r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
		for rest := in; rest != ""; {
			n := min(r.IntN(16)+1, len(rest))
			if _, err := w.WriteString(rest[:n]); err != nil {
				t.Fatal(err)
			}
			rest = rest[n:]
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if want := printed(in, o...); b.String() != want {
			t.Fatalf("split writes gave %q; the whole gives %q", b.String(), want)
		}
	})
}

// BenchmarkWriter: the benchmark reply, copied into a Writer in 64-byte
// reads, as BenchmarkReplyRender streams it.
func BenchmarkWriter(b *testing.B) {
	reply := readReply(b)
	b.ReportAllocs()
	for b.Loop() {
		w := stream.NewWriter(io.Discard, stream.WithMarkup(stream.DefaultMarkup()), stream.WithWidth(100))
		_, _ = io.CopyBuffer(w, iotest.HalfReader(bytes.NewReader(reply)), make([]byte, 64))
		_ = w.Close()
	}
}

// A reply copied to a destination as it streams (0009-MADR A8): a Writer
// is an io.Writer, so io.Copy feeds it; each line reaches the destination
// once it is stable, and Close writes the rest. Here the destination is
// the example's output; a program's might be a file, a pipe or a log.
func ExampleWriter() {
	reply := strings.NewReader("# Title\n\nA paragraph long enough to wrap at thirty-two columns.\n\n| a | b |\n|---|---|\n| 1 | 22 |\n")
	w := stream.NewWriter(os.Stdout, stream.WithWidth(32))
	_, _ = io.Copy(w, reply)
	_ = w.Close()
	// Output:
	// # Title
	//
	// A paragraph long enough to wrap
	// at thirty-two columns.
	//
	// | a   | b   |
	// | --- | --- |
	// | 1   | 22  |
}
