package termcap_test

import (
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
	"github.com/maccavelli/go-tui-lib/v2/termcap/termcaptest"
)

// split is each escape sequence of a write, as a terminal reads them:
// termcaptest's own reader, through a terminal that answers nothing.
func split(data string) []string {
	r := termcaptest.NewTerminal(termcaptest.Silent())
	if _, err := r.Write([]byte(data)); err != nil {
		panic(err)
	}
	return r.Sequences()
}

// drive runs p against term with no program: each write's sequences are
// answered by term.Reply, each reply is parsed and handed back, in order,
// and a probe nothing ends is timed out at its deadline. It returns the
// Caps of ProbeDone.
func drive(t *testing.T, p *termcap.Probe, term *termcaptest.Terminal, env termcap.Env) termcap.Caps {
	t.Helper()
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	p.Start(env, now)
	var replies []string
	var caps termcap.Caps
	done := 0
	for {
		for e := p.NextEvent(); e.Kind != termcap.ProbeNoEvent; e = p.NextEvent() {
			switch e.Kind {
			case termcap.ProbeWrite:
				for _, seq := range split(e.Data) {
					if r := term.Reply(seq); r != "" {
						replies = append(replies, r)
					}
				}
			case termcap.ProbeDone:
				caps = e.Caps
				done++
			}
		}
		if len(replies) > 0 {
			p.HandleReply(termcap.ParseReply(replies[0]), now)
			replies = replies[1:]
			continue
		}
		if deadline, ok := p.Timeout(); ok {
			p.HandleTimeout(deadline)
			continue
		}
		break
	}
	if done != 1 {
		t.Fatalf("%d ProbeDone", done)
	}
	return caps
}

type fact = termcap.Fact[termcap.Support]

var (
	yes = fact{Value: termcap.Supported, Origin: termcap.Queried}
	no  = fact{Value: termcap.Unsupported, Origin: termcap.Queried}
)

// TestProbeTranscripts: a Probe driven with no program, through each
// termcaptest profile's own replies to its batch, ends with the facts
// termcaptest's TestEachProfile pins for that profile through a Prober.
func TestProbeTranscripts(t *testing.T) {
	cases := []struct {
		p          termcaptest.Profile
		terminal   string
		mux        termcap.Mux
		dark       bool
		keyboard   fact
		scheme     fact
		resize     fact
		focus      fact
		notify     fact
		graphics   fact
		syncOutput fact
	}{
		{termcaptest.Kitty(), "kitty(0.39.1)", termcap.NoMux, true, yes, yes, yes, yes, yes, yes, yes},
		{termcaptest.XTerm(), "xterm-256color", termcap.NoMux, false, no, no, no, yes, no, no, no},
		{termcaptest.Tmux(true), "tmux 3.4", termcap.Tmux, true, no, no, no, yes, yes, yes, no},
		{termcaptest.Tmux(false), "tmux 3.4", termcap.Tmux, true, no, no, no, yes, no, no, no},
		{termcaptest.DA1Only(), "xterm-256color", termcap.NoMux, false, no, no, no, no, no, no, fact{}},
	}
	for _, c := range cases {
		got := drive(t, termcap.NewProbe(termcap.WithGOOS("linux")), termcaptest.NewTerminal(c.p), termcap.Env(c.p.Env))
		if !got.Complete || got.TimedOut {
			t.Errorf("%s: Complete %v, TimedOut %v; want the sentinel", c.p.Name, got.Complete, got.TimedOut)
		}
		if got.Terminal.Value != c.terminal || got.Mux.Value != c.mux {
			t.Errorf("%s: Terminal %+v, Mux %+v; want %q, %v", c.p.Name, got.Terminal, got.Mux, c.terminal, c.mux)
		}
		if c.p.Background != nil && got.Dark.Value != c.dark {
			t.Errorf("%s: Dark = %+v, want %v", c.p.Name, got.Dark, c.dark)
		}
		for name, f := range map[string][2]fact{
			"KittyKeyboard":      {got.KittyKeyboard, c.keyboard},
			"ColorSchemeReports": {got.ColorSchemeReports, c.scheme},
			"InBandResize":       {got.InBandResize, c.resize},
			"FocusEvents":        {got.FocusEvents, c.focus},
			"DesktopNotify":      {got.DesktopNotify, c.notify},
			"KittyGraphics":      {got.KittyGraphics, c.graphics},
			"SyncOutput":         {got.SyncOutput, c.syncOutput},
		} {
			if f[0] != f[1] {
				t.Errorf("%s: %s = %+v, want %+v", c.p.Name, name, f[0], f[1])
			}
		}
	}

	apple := termcaptest.AppleTerminalSSH()
	if got := drive(t, termcap.NewProbe(termcap.WithGOOS("linux")), termcaptest.NewTerminal(apple), termcap.Env(apple.Env)); got.Brand != (termcap.Fact[termcap.Brand]{Value: termcap.BrandAppleTerminal, Origin: termcap.Queried}) || !got.Remote.Value {
		t.Errorf("Apple Terminal over SSH: Brand %+v, Remote %+v", got.Brand, got.Remote)
	}
	jb := termcaptest.JetBrains()
	if got := drive(t, termcap.NewProbe(termcap.WithoutHeuristic()), termcaptest.NewTerminal(jb), termcap.Env(jb.Env)); got.Complete || got.KittyKeyboard.Reason != termcap.ReasonJetBrainsPaints {
		t.Errorf("JetBrains: Complete %v, KittyKeyboard %+v", got.Complete, got.KittyKeyboard)
	}
	silent := termcaptest.Silent()
	if got := drive(t, termcap.NewProbe(), termcaptest.NewTerminal(silent), termcap.Env(silent.Env)); !got.TimedOut || got.Complete || got.KittyKeyboard != (fact{}) {
		t.Errorf("silent: TimedOut %v, Complete %v, KittyKeyboard %+v", got.TimedOut, got.Complete, got.KittyKeyboard)
	}
}
