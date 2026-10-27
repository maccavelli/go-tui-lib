package inline

import (
	"context"
	"io"
	"sync"
)

// Reader reads the host's input only when asked, and never closes it
// (docs/decisions/0023-MADR-line-mode-components.md D3). Its one goroutine
// starts at the first Read and makes one read of the input for each Read
// that finds nothing kept, with a buffer as large as that Read's, and none
// ahead: so a read starts in the terminal mode of whoever asked, raw for a
// component and cooked for the program's own prompt. On Windows a console
// read keeps the mode it started in, and closing console input while a
// read is pending hangs the process (gobble-cli's
// 0027-MADR-native-line-editor-word-wrap.md D5, as amended).
//
// The program shares one Reader between its components and its own
// prompts, so no second reader takes their keys. Read is safe from any
// goroutine; one Read waits at a time.
type Reader struct {
	in    io.Reader
	start sync.Once
	asks  chan int      // a read of this many bytes is asked for
	got   chan chunk    // what the read gave
	turn  chan struct{} // held by the Read that waits

	kept  []byte // read and not yet given
	err   error  // the input's error, given once kept is empty
	asked bool   // a read was asked for and has not been taken
}

// chunk is one read of the input.
type chunk struct {
	b   []byte
	err error
}

// NewReader returns a Reader of in, which reads nothing until asked.
func NewReader(in io.Reader) *Reader {
	return &Reader{in: in, asks: make(chan int, 1), got: make(chan chunk, 1), turn: make(chan struct{}, 1)}
}

// Read reads what was kept, or asks for one read of the input and waits
// for it until ctx ends. ctx ends the wait, not the read: a read asked for
// and not taken by then is taken by the next Read. The input's error is
// returned once the bytes before it are read, and by every Read after.
func (r *Reader) Read(ctx context.Context, p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	select {
	case r.turn <- struct{}{}:
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	defer func() { <-r.turn }()
	if len(r.kept) == 0 && r.err == nil {
		if !r.asked {
			r.start.Do(func() { go r.loop() })
			r.asks <- len(p)
			r.asked = true
		}
		select {
		case c := <-r.got:
			r.kept, r.err, r.asked = c.b, c.err, false
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	if len(r.kept) == 0 {
		return 0, r.err
	}
	n := copy(p, r.kept)
	r.kept = r.kept[n:]
	return n, nil
}

// loop makes each read asked for, until the input fails. A read that
// gives nothing and no error is made again.
func (r *Reader) loop() {
	for size := range r.asks {
		b := make([]byte, size)
		n, err := r.in.Read(b)
		for n == 0 && err == nil {
			n, err = r.in.Read(b)
		}
		r.got <- chunk{b: b[:n], err: err}
		if err != nil {
			return
		}
	}
}
