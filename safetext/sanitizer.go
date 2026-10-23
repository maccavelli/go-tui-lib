package safetext

import (
	"errors"
	"io"
)

// ErrClosed is Write's error after Close.
var ErrClosed = errors.New("safetext: write after Close")

// Sanitizer is Clean for a stream: writing a and then b through one, then
// closing it, writes what Clean(a+b) gives. A sequence or a character split
// across writes is held until it is whole; Close drops a sequence that
// never ended, and never writes it. A Sanitizer is not safe for concurrent
// use.
type Sanitizer struct {
	w      io.Writer
	m      machine
	closed bool
}

// NewSanitizer returns a Sanitizer that writes to w what the rules let
// through.
func NewSanitizer(w io.Writer, r Rules) *Sanitizer {
	s := &Sanitizer{w: w}
	s.m.init(&r, modeStream, "")
	return s
}

// Write cleans p and writes what survives to the underlying writer. It
// reports len(p) unless the underlying writer fails.
func (s *Sanitizer) Write(p []byte) (int, error) {
	if s.closed {
		return 0, ErrClosed
	}
	written := len(p)
	m := &s.m
	// Complete the character the last Write cut short, a byte at a time.
	for len(p) > 0 && m.pendN > 0 {
		m.pend[m.pendN] = p[0]
		m.pendN++
		p = p[1:]
		r, n, ok, partial := decode(m.pend[:m.pendN])
		if partial {
			continue
		}
		var held [4]byte
		k := copy(held[:], m.pend[:m.pendN])
		m.pendN = 0
		if ok {
			char(m, r, held[:n])
		} else {
			char(m, 0xfffd, "\ufffd")
		}
		run(m, held[n:k])
	}
	if m.pendN == 0 {
		run(m, p)
	}
	return written, s.flush()
}

// Close ends the stream: a character cut short is invalid UTF-8, a lone \r
// is removed, and a sequence never ended is dropped. It writes what is
// left. It does not close the underlying writer.
func (s *Sanitizer) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	s.m.finish()
	return s.flush()
}

// flush writes what the machine has made so far.
func (s *Sanitizer) flush() error {
	if len(s.m.o.buf) == 0 {
		return nil
	}
	_, err := s.w.Write(s.m.o.buf)
	s.m.o.buf = s.m.o.buf[:0]
	return err
}
