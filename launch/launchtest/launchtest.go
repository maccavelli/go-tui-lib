// Package launchtest gives a program's own tests fake streams for launch, so
// a CLI's --tui path can be tested end to end with no real terminal
// (docs/decisions/0013-MADR-cli-integration-helpers.md A1.4).
//
// A Terminal says it is a terminal, has a size, takes keystrokes with Type,
// and keeps everything written to it. A Pipe is a stream that is not a
// terminal. Streams and Env build launch's Streams from them:
//
//	term := launchtest.NewTerminal(80, 24)
//	s := launchtest.Streams(term, term, launchtest.NewPipe(""), "TERM=xterm-256color")
//	d := launch.Decide(s, launch.Config{Choice: launch.ChoiceTUI})
//	term.Type("q")
//	final, err := launch.Run(ctx, s, d, model)
//	term.Close()
//
// Nothing here touches the process's streams, environment or terminal.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package launchtest

import (
	"bytes"
	"io"
	"strings"
	"sync"

	"github.com/maccavelli/go-tui-lib/launch"
	"github.com/maccavelli/go-tui-lib/termcap"
)

// Terminal is a fake terminal stream: a reader and a writer that say they
// are a terminal, with a size. It is safe for use from several goroutines.
type Terminal struct {
	mu     sync.Mutex
	ready  *sync.Cond // signalled when input arrives or the terminal closes
	in     []byte
	out    bytes.Buffer
	w, h   int
	closed bool
}

// NewTerminal is a terminal of width × height cells, with no input yet.
func NewTerminal(width, height int) *Terminal {
	t := &Terminal{w: width, h: height}
	t.ready = sync.NewCond(&t.mu)
	return t
}

// Read reads keystrokes Type queued. It waits while there are none, as a
// terminal does, and returns io.EOF once the terminal is closed and every
// keystroke has been read.
func (t *Terminal) Read(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for len(t.in) == 0 && !t.closed {
		t.ready.Wait()
	}
	if len(t.in) == 0 {
		return 0, io.EOF
	}
	n := copy(p, t.in)
	t.in = t.in[n:]
	return n, nil
}

// Write keeps p, for Output.
func (t *Terminal) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.out.Write(p)
}

// IsTerminal is true: launch's terminal test takes a stream's own answer.
func (t *Terminal) IsTerminal() bool { return true }

// Size is the terminal's size in cells.
func (t *Terminal) Size() (width, height int) { return t.w, t.h }

// Type queues keys as the raw bytes a terminal sends, for Read: "q" is the
// key q, and "\r" is enter.
func (t *Terminal) Type(keys string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.in = append(t.in, keys...)
	t.ready.Broadcast()
}

// Close ends the input: Read returns io.EOF once the queued keystrokes are
// read. It is safe to call more than once.
func (t *Terminal) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	t.ready.Broadcast()
	return nil
}

// Output is everything written so far.
func (t *Terminal) Output() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.out.String()
}

// Pipe is a stream that is not a terminal: input to read, as a redirected
// standard input gives it, and a writer that keeps what is written, as a
// redirected standard output takes it.
type Pipe struct {
	mu  sync.Mutex
	in  *strings.Reader
	out bytes.Buffer
}

// NewPipe is a pipe whose reader yields input, then io.EOF.
func NewPipe(input string) *Pipe { return &Pipe{in: strings.NewReader(input)} }

// Read reads the pipe's input.
func (p *Pipe) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.in.Read(b)
}

// Write keeps b, for Output.
func (p *Pipe) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.out.Write(b)
}

// Output is everything written so far.
func (p *Pipe) Output() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.out.String()
}

// Streams is in, out and err, with an environment of env's KEY=value
// strings.
func Streams(in io.Reader, out, err io.Writer, env ...string) launch.Streams {
	return launch.Streams{In: in, Out: out, Err: err, Env: Env(env...)}
}

// Env is an environment of KEY=value strings, for Streams or a Decide
// test.
func Env(kv ...string) termcap.Env { return termcap.Env(kv) }
