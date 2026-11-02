package stream

import (
	"errors"
	"io"
)

// ErrClosed is a Writer's error for a write after Close.
var ErrClosed = errors.New("stream: write after Close")

// Writer is an io.WriteCloser and io.StringWriter over a Printer: what is
// written to it is printed, and each line, once stable, is written to its
// destination (docs/decisions/0009-MADR-streaming-content-engine.md A8).
// So a reply reaches any io.Writer through the standard library's own
// composition: io.Copy from a reply's body, io.MultiWriter to a terminal
// and a log, bufio, a pipe.
//
// Its options are the Printer's, with its defaults: the input cleaned by
// safetext.Text, unstyled, unwrapped, a table laid out in aligned columns,
// each line ending with LF. The output is the same for any split of the
// input. A Writer is used on one goroutine, as a Printer is.
type Writer struct {
	p      *Printer
	dst    io.Writer
	err    error // the destination's first error, returned from then on
	closed bool
}

// NewWriter is a Writer to dst, its Printer made with o.
func NewWriter(dst io.Writer, o ...Option) *Writer {
	return &Writer{p: NewPrinter(o...), dst: dst}
}

// Write prints b, and writes what is now stable to the destination in one
// write. It returns len(b) once b is the Printer's, with the destination's
// error if that write failed; after a failure, 0 and that error; after
// Close, 0 and ErrClosed.
func (w *Writer) Write(b []byte) (int, error) { return w.WriteString(string(b)) }

// WriteString is Write for a string.
func (w *Writer) WriteString(s string) (int, error) {
	switch {
	case w.closed:
		return 0, ErrClosed
	case w.err != nil:
		return 0, w.err
	}
	w.put(w.p.Write(s))
	return len(s), w.err
}

// Close writes what the Printer still holds, ending the reply, and leaves
// the destination open, since it is the program's. It returns the
// destination's error, if any; a second Close returns nil.
func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	if w.err == nil {
		w.put(w.p.Flush())
	}
	return w.err
}

// put writes s to the destination, keeping its first error; a short write
// with no error is io.ErrShortWrite.
func (w *Writer) put(s string) {
	if s == "" {
		return
	}
	n, err := io.WriteString(w.dst, s)
	if err == nil && n < len(s) {
		err = io.ErrShortWrite
	}
	w.err = err
}
