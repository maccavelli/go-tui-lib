// Command xtermparity writes lineedit's parity reference: what
// golang.org/x/term v0.47.0's Terminal makes of each of gobble-cli's parity
// cases (its internal/cli/editor/buffer_test.go:144-204, at 5fb6928), run
// live, as JSON on standard output
// (docs/decisions/0023-MADR-line-mode-components.md A5 item 1). The go
// command ignores testdata, so this is built with its own module file, from
// the repository's root:
//
//	GOWORK=off GOFLAGS=-mod=mod go run -modfile=lineedit/testdata/xtermparity/xterm.mod ./lineedit/testdata/xtermparity > lineedit/testdata/xterm-parity.json
//
// lineedit's TestKeysMatchXterm compares the Composer with the file.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	xterm "golang.org/x/term"
)

// fixedHistory is a History that Add leaves as it is, so the Terminal,
// which adds each line it returns, sees the same entries throughout.
// Index 0 is the most recent.
type fixedHistory []string

func (fixedHistory) Add(string)          {}
func (h fixedHistory) Len() int          { return len(h) }
func (h fixedHistory) At(idx int) string { return h[idx] }

// history is the parity cases' history, gobble's.
var history = fixedHistory{"newest entry", "middle", "oldest one"}

// outcome is what a run of keys leaves: the lines it ended, each marked
// when it was pasted, whether ctrl+d (or ctrl+c) ended the input, and the
// line being typed with its cursor, in runes; pos -1 when the input ended
// and the line cannot be read.
type outcome struct {
	Lines  []string `json:"lines"`
	Pasted []bool   `json:"pasted"`
	EOF    bool     `json:"eof"`
	Text   string   `json:"text"`
	Pos    int      `json:"pos"`
}

// parityCase is one case and its reference.
type parityCase struct {
	Name string  `json:"name"`
	In   string  `json:"in"`
	Rows bool    `json:"rows"` // up or down, compared on one row only
	Want outcome `json:"want"`
}

// probeKey is sent after a run: it reaches AutoCompleteCallback with the
// line and the cursor, and changes nothing.
const probeKey = 0x1c

// probeConn feeds a Terminal its reads in turn, then io.EOF, and discards
// what it draws. The probe comes in a read of its own: x/term drops the
// rest of a read after ctrl+d or ctrl+c ends the input.
type probeConn struct{ reads [][]byte }

func (c *probeConn) Read(p []byte) (int, error) {
	if len(c.reads) == 0 {
		return 0, io.EOF
	}
	n := copy(p, c.reads[0])
	if c.reads[0] = c.reads[0][n:]; len(c.reads[0]) == 0 {
		c.reads = c.reads[1:]
	}
	return n, nil
}

func (c *probeConn) Write(p []byte) (int, error) { return len(p), nil }

// run is what x/term's Terminal makes of in.
func run(in []byte) (outcome, error) {
	term := xterm.NewTerminal(&probeConn{reads: [][]byte{slices.Clone(in), {probeKey}}}, "> ")
	term.History = history
	out := outcome{Lines: []string{}, Pasted: []bool{}}
	probed := false
	term.AutoCompleteCallback = func(line string, pos int, key rune) (string, int, bool) {
		if key == probeKey {
			out.Text, out.Pos, probed = line, utf8.RuneCountInString(line[:pos]), true
		}
		return "", 0, false
	}
	for range 64 {
		line, err := term.ReadLine()
		switch {
		case err == nil, errors.Is(err, xterm.ErrPasteIndicator):
			out.Lines = append(out.Lines, line)
			out.Pasted = append(out.Pasted, err != nil)
		case errors.Is(err, io.EOF) && probed:
			return out, nil
		case errors.Is(err, io.EOF):
			out.EOF, out.Pos = true, -1
			return out, nil
		default:
			return out, err
		}
	}
	return out, errors.New("no end to the run")
}

const (
	up, down, left, right = "\x1b[A", "\x1b[B", "\x1b[D", "\x1b[C"
	home, end, del        = "\x1b[H", "\x1b[F", "\x1b[3~"
	altLeft, altRight     = "\x1b[1;3D", "\x1b[1;3C"
	pasteOn, pasteOff     = "\x1b[200~", "\x1b[201~"
)

func times(s string, n int) string { return strings.Repeat(s, n) }

// cases are gobble's 55, as its buffer_test.go:144-204 lists them.
var cases = []parityCase{
	{Name: "typing", In: "abc"},
	{Name: "typing in the middle", In: "hello world" + times(left, 6) + "big "},
	{Name: "left at the start stays", In: "ab" + times(left, 5)},
	{Name: "right at the end stays", In: "ab" + left + times(right, 4)},
	{Name: "Ctrl+B and Ctrl+F", In: "abcd\x02\x02\x02\x06"},
	{Name: "Backspace", In: "hello" + left + left + "\x7f"},
	{Name: "Backspace at the start", In: "ab" + home + "\x7f"},
	{Name: "Ctrl+H", In: "hello\x08\x08"},
	{Name: "Home", In: "hello world" + left + home},
	{Name: "End", In: "hello world" + home + right + end},
	{Name: "Ctrl+A", In: "hello world" + left + "\x01"},
	{Name: "Ctrl+E", In: "hello world" + "\x01\x06\x05"},
	{Name: "Alt+Left twice", In: "one two  three" + altLeft + altLeft},
	{Name: "Alt+Left over spaces", In: "one two   " + altLeft},
	{Name: "Alt+Left at the start", In: "one" + home + altLeft},
	{Name: "Alt+Right twice", In: "one two  three" + home + altRight + altRight},
	{Name: "Alt+Right at the end", In: "one two" + altRight},
	{Name: "Alt+Right from a space", In: "one   two" + home + times(right, 4) + altRight},
	{Name: "Delete in the middle", In: "hello" + left + left + del},
	{Name: "Delete at the end", In: "hello" + del},
	{Name: "Ctrl+D in the middle", In: "hello" + home + "\x04"},
	{Name: "Ctrl+D at the end of a line", In: "hello\x04"},
	{Name: "Ctrl+K", In: "hello world" + times(left, 5) + "\x0b"},
	{Name: "Ctrl+U", In: "hello world" + times(left, 5) + "\x15"},
	{Name: "Ctrl+W", In: "one two three\x17"},
	{Name: "Ctrl+W over spaces", In: "one two   \x17"},
	{Name: "Ctrl+W in the middle", In: "one two three" + times(left, 3) + "\x17"},
	{Name: "Ctrl+T in the middle", In: "abcd" + left + left + "\x14"},
	{Name: "Ctrl+T at the end", In: "abcd\x14"},
	{Name: "Ctrl+T at the start", In: "abcd" + home + "\x14"},
	{Name: "Ctrl+L", In: "hello" + left + "\x0c"},
	{Name: "Tab without completion", In: "ab\t"},
	{Name: "unknown sequences", In: "ab\x1b[Z\x1b[15~\x1b[1;2A"},
	{Name: "wide and accented", In: "héllo 世界" + left + "x"},
	{Name: "Enter in the middle", In: "hello world" + times(left, 3) + "\r" + "next"},
	{Name: "CR LF is one Enter", In: "ab\r\ncd"},
	{Name: "LF is Enter", In: "ab\ncd"},
	{Name: "Ctrl+D on an empty line", In: "\x04"},
	{Name: "Ctrl+D after a line", In: "ab\r\x04"},
	{Name: "Ctrl+C", In: "ab\x03"},
	{Name: "a paste of two lines", In: pasteOn + "one\ntwo" + pasteOff},
	{Name: "a paste then Enter", In: pasteOn + "one\ntwo" + pasteOff + "\r"},
	{Name: "a paste of three lines", In: pasteOn + "one\ntwo\nthree" + pasteOff + "\r"},
	{Name: "a paste into a typed line", In: "say " + pasteOn + "one\ntwo" + pasteOff + "\r"},
	{Name: "control keys inside a paste", In: pasteOn + "a\tb\x03c\x04d\x7f" + pasteOff},
	{Name: "the line limit", In: times("a", 4100)},
	{Name: "Up recalls", In: up, Rows: true},
	{Name: "Up twice", In: up + up, Rows: true},
	{Name: "Up past the oldest", In: times(up, 5), Rows: true},
	{Name: "Up then Down", In: "draft" + up + up + down, Rows: true},
	{Name: "Down back to the draft", In: "draft" + up + down, Rows: true},
	{Name: "Down with nothing recalled", In: "draft" + down, Rows: true},
	{Name: "Ctrl+P and Ctrl+N", In: "draft\x10\x10\x0e", Rows: true},
	{Name: "Up then edit", In: up + "\x7f\x7fX", Rows: true},
	{Name: "Up after Enter", In: "a\r" + up, Rows: true},
}

func main() {
	out := make([]parityCase, 0, len(cases))
	for _, c := range cases {
		want, err := run([]byte(c.In))
		if err != nil {
			fmt.Fprintf(os.Stderr, "xtermparity: %s: %v\n", c.Name, err)
			os.Exit(1)
		}
		c.Want = want
		out = append(out, c)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "\t")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "xtermparity: %v\n", err)
		os.Exit(1)
	}
}
