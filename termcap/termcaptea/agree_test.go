package termcaptea_test

import (
	"image/color"
	"reflect"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
	"github.com/maccavelli/go-tui-lib/v2/termcap/termcaptea"
	"github.com/maccavelli/go-tui-lib/v2/termcap/termcaptest"
)

// eight is c at 8 bits per channel, the precision of the colours tea's
// decoded messages carry; nil stays nil.
func eight(c color.Color) color.Color {
	if c == nil {
		return nil
	}
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// comparable is c with its colours at 8 bits and its profile dropped: only
// tea detects the profile.
func comparable(c termcap.Caps) termcap.Caps {
	c.Profile = termcap.ProfileUnknown
	c.Background, c.Foreground = eight(c.Background), eight(c.Foreground)
	for i := range c.Palette {
		c.Palette[i] = eight(c.Palette[i])
	}
	return c
}

// TestCoreAndAdapterAgree is the differential of MADR A2.5: for each
// termcaptest profile, the Caps a Prober delivers in a real tea.Program,
// and the Caps of a termcap.Probe given the same replies with no program,
// are equal but for Profile. The Probe is fed the replies a fresh
// Terminal gives, through Reply, to every sequence the program wrote, in
// its order: the same bytes, decoded once by tea and ultraviolet and once
// by termcap.ParseReply. So the Kitty flags tea's first render pushes,
// which may reach the terminal before or after the probe's own query, are
// the same on both sides (0020-PLAN, deviation of 2026-10-24).
func TestCoreAndAdapterAgree(t *testing.T) {
	opts := []termcap.Option{termcap.WithGOOS("linux"), termcap.WithTimeout(100 * time.Millisecond)}
	for _, p := range []termcaptest.Profile{
		termcaptest.Kitty(), termcaptest.XTerm(), termcaptest.Tmux(true), termcaptest.Tmux(false),
		termcaptest.DA1Only(), termcaptest.AppleTerminalSSH(), termcaptest.JetBrains(), termcaptest.Silent(),
	} {
		term := termcaptest.NewTerminal(p)
		tui := term.Run(t, app{termcaptea.NewProber(opts...)})

		// The same options, and the modes tea asks for, as the Prober's own.
		core := termcap.NewProbe(append(opts, termcap.WithoutModeQuery(2026, 2027))...)
		now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
		core.Start(termcap.Env(p.Env), now)
		var caps termcap.Caps
		done := false
		take := func() {
			for e := core.NextEvent(); e.Kind != termcap.ProbeNoEvent; e = core.NextEvent() {
				if e.Kind == termcap.ProbeDone {
					caps, done = e.Caps, true
				}
			}
		}
		take()
		fresh := termcaptest.NewTerminal(p)
		for _, seq := range term.Sequences() {
			if done {
				break
			}
			if r := fresh.Reply(seq); r != "" {
				core.HandleReply(termcap.ParseReply(r), now)
				take()
			}
		}
		if deadline, ok := core.Timeout(); !done && ok {
			core.HandleTimeout(deadline)
			take()
		}
		if !done {
			t.Fatalf("%s: the core never ended", p.Name)
		}
		if got, want := comparable(caps), comparable(tui); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: the core and the Prober disagree:\ncore %+v\n tui %+v", p.Name, got, want)
		}
	}
}

// TestDarkMatchesBubbleTea: the core's reading of an OSC 11 colour as dark
// is tea's BackgroundColorMsg.IsDark, colour by colour.
func TestDarkMatchesBubbleTea(t *testing.T) {
	for _, c := range []color.RGBA{
		{0, 0, 0, 0xff}, {0xff, 0xff, 0xff, 0xff}, {0x7f, 0x7f, 0x7f, 0xff}, {0x80, 0x80, 0x80, 0xff},
		{0xfe, 0, 0, 0xff}, {0xff, 0, 0, 0xff}, {0xff, 0, 0xff, 0xff}, {0x1e, 0x1e, 0x2e, 0xff}, {0x7f, 0x80, 0, 0xff},
		{0, 0xff, 0, 0xff}, {0x40, 0xbf, 0x10, 0xff}, {0x80, 0x7f, 0x7f, 0xff},
	} {
		p := termcap.NewProbe()
		p.Start(termcap.Env{"TERM=xterm-kitty"}, time.Time{})
		p.HandleReply(termcap.Reply{Kind: termcap.ReplyColor, OSC: 11, Color: c}, time.Time{})
		if got, want := p.Caps().Dark.Value, (tea.BackgroundColorMsg{Color: c}).IsDark(); got != want {
			t.Errorf("%v: dark %v, tea says %v", c, got, want)
		}
	}
}
