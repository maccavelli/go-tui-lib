package termcaptea

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/termevent/termeventtest"
	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

func TestDA2IsSentBeforeDA1(t *testing.T) {
	_, msgs := start(local)
	b := raw(msgs)
	if i, j := strings.Index(b, da2Q), strings.LastIndex(b, da1Q); i < 0 || j < 0 || i > j || j != len(b)-len(da1Q) {
		t.Fatalf("DA2 at %d, DA1 at %d of %d: want DA2 before DA1, DA1 last", i, j, len(b))
	}
}

func TestAppleTerminalFingerprint(t *testing.T) {
	ssh := termcap.Env{"TERM=xterm-256color", "SSH_TTY=/dev/pts/1"}
	da1, da2 := termeventtest.DeviceAttributes(1, 2), termeventtest.SecondaryDeviceAttributes(1, 95, 0)
	for name, order := range map[string][]tea.Msg{"DA2 then DA1": {da2, da1}, "DA1 then DA2": {da1, da2}} {
		p, _ := start(ssh)
		feed(p, order...)
		if c := p.Caps(); c.Brand != (termcap.Fact[termcap.Brand]{Value: termcap.BrandAppleTerminal, Origin: termcap.Queried}) {
			t.Errorf("%s: Brand %+v, want Apple Terminal from the replies", name, c.Brand)
		}
	}
	for name, msgs := range map[string][]tea.Msg{
		"DA1 alone":       {da1},
		"DA2 alone":       {da2},
		"another DA1":     {termeventtest.DeviceAttributes(62, 22), da2},
		"another DA2":     {da1, termeventtest.SecondaryDeviceAttributes(1, 95, 1)},
		"a DA2 in pieces": {da1, termeventtest.SecondaryDeviceAttributes(1, 95)},
	} {
		p, _ := start(ssh)
		feed(p, msgs...)
		if b := p.Caps().Brand.Value; b == termcap.BrandAppleTerminal {
			t.Errorf("%s: read as Apple Terminal", name)
		}
	}
	p, _ := start(ssh)
	feed(p, da2)
	if got := p.Caps().SecondaryAttributes; len(got) != 3 || got[1] != 95 {
		t.Errorf("SecondaryAttributes = %v", got)
	}
}

// osc99Reply is a reply to the OSC 99 query padded to n bytes.
func osc99Reply(n int) string {
	head, tail := "\x1b]99;i=termcap:p=?;a=", "\x1b\\"
	return head + strings.Repeat("x", n-len(head)-len(tail)) + tail
}

func TestReplyCap(t *testing.T) {
	ok, _ := start(local)
	feed(ok, termeventtest.UnknownOsc(osc99Reply(1024)))
	if f := ok.Caps().DesktopNotify; f != (termcap.Fact[termcap.Support]{Value: termcap.Supported, Origin: termcap.Queried}) {
		t.Errorf("a 1024-byte reply: %+v, want parsed", f)
	}
	long, _ := start(local)
	feed(long, termeventtest.UnknownOsc(osc99Reply(1025)), termeventtest.DeviceAttributes(62))
	if f := long.Caps().DesktopNotify; f != (termcap.Fact[termcap.Support]{Value: termcap.Unknown, Origin: termcap.Queried, Reason: termcap.ReasonReplyTooLong}) {
		t.Errorf("a 1025-byte reply: %+v, want unparsed with the reason, and not marked silent", f)
	}
	parsed := false
	added := termcap.Query{Name: "x", Seq: fixed("\x1b[?5n"), Parse: func(r termcap.Reply, _ *termcap.Caps) bool { parsed = parsed || r.Raw != ""; return r.Raw != "" }}
	p, _ := start(local, termcap.WithQuery(added))
	feed(p, termeventtest.UnknownCsi("\x1b["+strings.Repeat("1", 1030)+"n"))
	if parsed {
		t.Error("an added query was given a reply over 1 KiB")
	}
}

func TestJetBrainsSendsNothing(t *testing.T) {
	jb := termcap.Env{"TERMINAL_EMULATOR=JetBrains-JediTerm", "TERM_SESSION_ID=1F2E3D4C", "TERM=xterm-256color"}
	p, msgs := start(jb, termcap.WithoutHeuristic())
	if b := raw(msgs); b != "" {
		t.Fatalf("sent %q under JetBrains", b)
	}
	c := capsOf(t, msgs)
	if c.Complete || c.TimedOut {
		t.Errorf("Complete %v, TimedOut %v; want neither, nothing was asked", c.Complete, c.TimedOut)
	}
	// The support facts the batch's queries set (termcap's queryFacts).
	for _, f := range []termcap.Fact[termcap.Support]{
		c.KittyKeyboard, c.ColorSchemeReports, c.InBandResize, c.FocusEvents, c.DesktopNotify, c.KittyGraphics, c.Sixel,
	} {
		if f != (termcap.Fact[termcap.Support]{Value: termcap.Unknown, Origin: termcap.NotQueried, Reason: termcap.ReasonJetBrainsPaints}) {
			t.Errorf("a query fact is %+v, want unknown with the JetBrains reason", f)
		}
	}
	if c.Brand.Value != termcap.BrandJetBrains {
		t.Errorf("Brand %v", c.Brand.Value)
	}
	if b := raw(feed(p, tea.ModeReportMsg{})); b != "" {
		t.Errorf("wrote %q later", b)
	}
}

func TestEditorTerminalSkipsTheGatedQueries(t *testing.T) {
	for _, env := range []termcap.Env{
		{"TERM=xterm-kitty", "NVIM=/tmp/nvim.sock"},
		{"TERM=xterm-kitty", "VIM_TERMINAL=900"},
		{"TERM=xterm-kitty", "INSIDE_EMACS=29.1,vterm"},
	} {
		p, msgs := start(env)
		b := raw(msgs)
		for _, q := range []string{xtversionQ, osc99Q, kittyGraphicsQ, osc10Q} {
			if strings.Contains(b, q) {
				t.Errorf("%v: sent the gated %q", env, q)
			}
		}
		if !strings.HasPrefix(b, kittyKeyboardQ+modesQ+dsr996Q) || !strings.HasSuffix(b, da1Q) {
			t.Errorf("%v: the safe set is missing: %q", env, b)
		}
		if f := p.Caps().DesktopNotify; f.Reason != termcap.ReasonEditorTerminal {
			t.Errorf("%v: DesktopNotify %+v, want the editor reason", env, f)
		}
	}
	_, msgs := start(termcap.Env{"TERM=xterm-kitty", "NVIM=x"}, termcap.WithoutHeuristic())
	if !strings.Contains(raw(msgs), osc99Q) {
		t.Error("WithoutHeuristic did not send the gated queries in an editor")
	}
}

func key(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: rune(text[0]), Text: text}
}

func TestIsReplyFragment(t *testing.T) {
	// The spike's shape: DA1 split after "\x1b[?62;", then "4" and "c".
	p, _ := start(local)
	for i, c := range []struct {
		msg  tea.Msg
		want bool
	}{
		{termeventtest.Unknown("\x1b[?62;"), true},
		{key("4"), true},
		{key("c"), true},
		{key("x"), false},
	} {
		if got := p.IsReplyFragment(c.msg); got != c.want {
			t.Errorf("CSI step %d (%v): %v, want %v", i, c.msg, got, c.want)
		}
	}

	// An OSC reply split before BEL, which arrives as ctrl+g.
	q, _ := start(local)
	for i, c := range []struct {
		msg  tea.Msg
		want bool
	}{
		{termeventtest.Unknown("\x1b]11;rgb:1e1e"), true},
		{key("/"), true},
		{tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, true},
		{key("a"), false},
	} {
		if got := q.IsReplyFragment(c.msg); got != c.want {
			t.Errorf("OSC step %d: %v, want %v", i, got, c.want)
		}
	}

	// An OSC reply ending in ST, ESC then backslash.
	s, _ := start(local)
	for i, m := range []tea.Msg{termeventtest.Unknown("\x1b]99;i=termcap"), tea.KeyPressMsg{Code: tea.KeyEscape}, key("\\")} {
		if !s.IsReplyFragment(m) {
			t.Errorf("ST step %d: not a fragment", i)
		}
	}
	if s.IsReplyFragment(key("z")) {
		t.Error("a key after ST is a fragment")
	}

	// alt+[ is a key, not a reply.
	r, _ := start(local)
	if r.IsReplyFragment(tea.KeyPressMsg{Code: '[', Mod: tea.ModAlt}) {
		t.Error("alt+[ is a fragment")
	}
	// A complete unknown sequence is a reply, not a fragment.
	if r.IsReplyFragment(termeventtest.Unknown("\x1b[?5n")) {
		t.Error("a complete sequence is a fragment")
	}
	// Before the batch, and once everything is answered, nothing is awaited.
	idle := NewProber()
	if idle.IsReplyFragment(termeventtest.Unknown("\x1b[?62;")) {
		t.Error("a fragment before the batch was sent")
	}
	// Every query the batch sent there answered, then DA1.
	done, _ := start(termcap.Env{"TERM=xterm", "SSH_TTY=x"})
	feed(done,
		tea.KeyboardEnhancementsMsg{Flags: 1},
		tea.ModeReportMsg{Mode: ansi.ModeLightDark, Value: ansi.ModeReset},
		tea.ModeReportMsg{Mode: ansi.ModeInBandResize, Value: ansi.ModeReset},
		tea.ModeReportMsg{Mode: ansi.ModeFocusEvent, Value: ansi.ModeReset},
		termeventtest.DarkColorScheme(),
		tea.BackgroundColorMsg{Color: color.Black},
		termeventtest.SecondaryDeviceAttributes(1, 2, 3),
		termeventtest.DeviceAttributes(62))
	if done.IsReplyFragment(termeventtest.Unknown("\x1b[?62;")) {
		t.Error("a fragment with nothing awaited")
	}
}

func TestIsReplyFragmentStopsAt1KiB(t *testing.T) {
	p, _ := start(local)
	p.IsReplyFragment(termeventtest.Unknown("\x1b]4;1;"))
	n := 0
	for p.IsReplyFragment(key("x")) {
		n++
		if n > 2*maxReply {
			break
		}
	}
	if n+len("\x1b]4;1;") != maxReply {
		t.Fatalf("a fragment ran to %d bytes, want %d", n+len("\x1b]4;1;"), maxReply)
	}
}

// TestCapsMsgWaitsForTheColourProfile: tea sends tea.ColorProfileMsg
// unordered with tea.EnvMsg, so CapsMsg waits for it; the deadline does
// not (MADR A3, PLAN D10).
func TestCapsMsgWaitsForTheColourProfile(t *testing.T) {
	p := NewProber()
	feed(p, tea.EnvMsg(local))
	if n := count[termcap.CapsMsg](feed(p, termeventtest.DeviceAttributes(62))); n != 0 {
		t.Fatalf("%d CapsMsg before the colour profile", n)
	}
	c := capsOf(t, feed(p, tea.ColorProfileMsg{Profile: colorprofile.ANSI256}))
	if !c.Complete || c.Profile != termcap.ProfileANSI256 {
		t.Errorf("Complete %v, Profile %v; want the sentinel's CapsMsg with the profile", c.Complete, c.Profile)
	}

	jb := NewProber()
	feed(jb, tea.EnvMsg(termcap.Env{"TERMINAL_EMULATOR=JetBrains-JediTerm"}))
	if c := capsOf(t, feed(jb, tea.ColorProfileMsg{Profile: colorprofile.TrueColor})); c.Profile != termcap.ProfileTrueColor {
		t.Errorf("JetBrains: Profile %v", c.Profile)
	}

	late := NewProber()
	feed(late, tea.EnvMsg(local))
	if c := capsOf(t, feed(late, timeoutMsg{late})); !c.TimedOut {
		t.Error("the deadline waited for the colour profile")
	}
}
