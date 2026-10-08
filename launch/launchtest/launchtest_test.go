package launchtest

import (
	"errors"
	"io"
	"testing"
	"time"
)

func TestTerminalReadWaitsForKeys(t *testing.T) {
	term := NewTerminal(80, 24)
	got := make(chan string, 1)
	go func() {
		b := make([]byte, 8)
		n, err := term.Read(b)
		if err != nil {
			t.Error(err)
		}
		got <- string(b[:n])
	}()
	select {
	case s := <-got:
		t.Fatalf("Read returned %q before any key was typed", s)
	case <-time.After(50 * time.Millisecond):
	}
	term.Type("q")
	select {
	case s := <-got:
		if s != "q" {
			t.Errorf("Read = %q, want \"q\"", s)
		}
	case <-time.After(time.Second):
		t.Fatal("Read did not return after Type")
	}
}

func TestTerminalCloseEndsInput(t *testing.T) {
	term := NewTerminal(80, 24)
	term.Type("ab")
	if err := term.Close(); err != nil {
		t.Fatal(err)
	}
	if err := term.Close(); err != nil {
		t.Fatal("a second Close failed:", err)
	}
	b, err := io.ReadAll(term)
	if err != nil || string(b) != "ab" {
		t.Errorf("after Close: %q, %v; want the queued keys, then EOF", b, err)
	}
	if n, err := term.Read(make([]byte, 1)); n != 0 || !errors.Is(err, io.EOF) {
		t.Errorf("Read after the keys = %d, %v; want io.EOF", n, err)
	}
}

func TestTerminalFacts(t *testing.T) {
	term := NewTerminal(132, 43)
	if !term.IsTerminal() {
		t.Error("a Terminal is not a terminal")
	}
	if w, h := term.Size(); w != 132 || h != 43 {
		t.Errorf("Size = %d×%d", w, h)
	}
	_, _ = term.Write([]byte("one "))
	_, _ = term.Write([]byte("two"))
	if got := term.Output(); got != "one two" {
		t.Errorf("Output = %q", got)
	}
}

func TestPipe(t *testing.T) {
	p := NewPipe("input")
	if _, ok := any(p).(interface{ IsTerminal() bool }); ok {
		t.Error("a Pipe claims to answer whether it is a terminal")
	}
	b, err := io.ReadAll(p)
	if err != nil || string(b) != "input" {
		t.Errorf("ReadAll = %q, %v", b, err)
	}
	_, _ = p.Write([]byte("out"))
	if p.Output() != "out" {
		t.Errorf("Output = %q", p.Output())
	}
}

func TestStreamsAndEnv(t *testing.T) {
	term, pipe := NewTerminal(80, 24), NewPipe("")
	s := Streams(term, term, pipe, "TERM=xterm", "TERM=dumb")
	if s.In != term || s.Out != term || s.Err != pipe {
		t.Error("Streams did not take its three streams")
	}
	if got := s.Env.Getenv("TERM"); got != "dumb" {
		t.Errorf("TERM = %q; want the last value, as in Bubble Tea", got)
	}
	if got := Env("A=1").Getenv("A"); got != "1" {
		t.Errorf("Env: A = %q", got)
	}
}
