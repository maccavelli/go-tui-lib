package launch

import (
	"bytes"
	"io"
	"sync"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// fake is a stream that says whether it is a terminal and what size it is,
// and counts its closes. It stands in for launchtest's terminals until
// docs/decisions/0013-PLAN-cli-integration-helpers.md Step 6 adds them; no
// test here touches a real terminal or the process's streams.
type fake struct {
	mu       sync.Mutex
	buf      bytes.Buffer
	term     bool
	w, h     int
	closed   int
	closeErr error // what Close returns
}

func (f *fake) IsTerminal() bool { return f.term }
func (f *fake) Size() (int, int) { return f.w, f.h }
func (f *fake) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.Read(p)
}

func (f *fake) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.Write(p)
}

func (f *fake) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed++
	return f.closeErr
}

// String is everything written to f, and not yet read.
func (f *fake) String() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.String()
}

func (f *fake) closes() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

// terminal is a fake terminal of w by h cells; pipe is a fake that is not a
// terminal.
func terminal(w, h int) *fake { return &fake{term: true, w: w, h: h} }
func pipe() *fake             { return &fake{} }

// streams is In, Out and Err, with env.
func streams(in io.Reader, out, err io.Writer, env ...string) Streams {
	return Streams{In: in, Out: out, Err: err, Env: termcap.Env(env)}
}

// base is the environment most tests start from: a 256-colour terminal in
// a UTF-8 locale, set explicitly, so no host's own matters.
var base = []string{"TERM=xterm-256color", "LANG=en_US.UTF-8"}

// with is base plus kv.
func with(kv ...string) []string { return append(append([]string{}, base...), kv...) }
