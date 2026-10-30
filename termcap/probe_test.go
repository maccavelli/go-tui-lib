package termcap

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// The batch, byte by byte, for a local terminal the heuristic allows.
const (
	kittyKeyboardQ = "\x1b[?u"
	modesQ         = "\x1b[?2031$p\x1b[?2048$p\x1b[?1004$p"
	dsr996Q        = "\x1b[?996n"
	osc11Q         = "\x1b]11;?\x07"
	xtversionQ     = "\x1b[>q"
	osc99Q         = "\x1b]99;i=termcap:p=?;\x07"
	kittyGraphicsQ = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\"
	da1Q           = "\x1b[c"
	da2Q           = "\x1b[>c"
	osc10Q         = "\x1b]10;?\x07"
	paletteQ       = "\x1b]4;0;?\x07\x1b]4;1;?\x07\x1b]4;2;?\x07\x1b]4;3;?\x07" +
		"\x1b]4;4;?\x07\x1b]4;5;?\x07\x1b]4;6;?\x07\x1b]4;7;?\x07" +
		"\x1b]4;8;?\x07\x1b]4;9;?\x07\x1b]4;10;?\x07\x1b]4;11;?\x07" +
		"\x1b]4;12;?\x07\x1b]4;13;?\x07\x1b]4;14;?\x07\x1b]4;15;?\x07"
	syncQ = "\x1b[?2026$p\x1b[?2027$p" // DECRQM 2026 and 2027, which tea sends in a TUI
)

var local = Env{"TERM=xterm-kitty"}

// events drains p.
func events(p *Probe) []ProbeEvent {
	var out []ProbeEvent
	for e := p.NextEvent(); e.Kind != ProbeNoEvent; e = p.NextEvent() {
		out = append(out, e)
	}
	return out
}

// written is every ProbeWrite among es, joined.
func written(es []ProbeEvent) string {
	var b strings.Builder
	for _, e := range es {
		if e.Kind == ProbeWrite {
			b.WriteString(e.Data)
		}
	}
	return b.String()
}

// kinds counts the events of kind k among es.
func kinds(es []ProbeEvent, k ProbeEventKind) int {
	n := 0
	for _, e := range es {
		if e.Kind == k {
			n++
		}
	}
	return n
}

// started is a probe started on env at t0, and what it queued.
func started(env Env, o ...Option) (*Probe, []ProbeEvent) {
	p := NewProbe(o...)
	p.Start(env, t0)
	return p, events(p)
}

// t0 is the time each test's probe starts.
var t0 = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

// reply passes each raw reply to p, parsed, and returns what p queued.
func reply(p *Probe, raws ...string) []ProbeEvent {
	for _, raw := range raws {
		p.HandleReply(ParseReply(raw), t0)
	}
	return events(p)
}

// TestProbeBatch: the first ProbeWrite after Start is today's batch, byte
// for byte, with DECRQM 2026 and 2027 before DA1 where Bubble Tea's rule
// sends them, and without them under WithoutModeQuery(2026, 2027), as
// termcaptea builds it.
func TestProbeBatch(t *testing.T) {
	tmux := func(seq string) string {
		return "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	safe := kittyKeyboardQ + modesQ + dsr996Q + osc11Q + da2Q
	gated := xtversionQ + osc99Q + kittyGraphicsQ + osc10Q + paletteQ
	inTmux := Env{"TERM=tmux-256color", "TMUX=/tmp/tmux-1/default,1,0"}
	ssh := Env{"TERM=xterm-256color", "SSH_TTY=/dev/pts/1"}
	apple := Env{"TERM=xterm-256color", "TERM_PROGRAM=Apple_Terminal"}
	tea := WithoutModeQuery(2026, 2027)
	cases := []struct {
		name string
		env  Env
		o    []Option
		want string
	}{
		{"local", local, nil, safe + gated + syncQ + da1Q},
		{"local, without 2026 and 2027", local, []Option{tea}, safe + gated + da1Q},
		{"inside tmux", inTmux, nil,
			safe + xtversionQ + tmux(osc99Q) + tmux(kittyGraphicsQ) + osc10Q + paletteQ + syncQ + da1Q},
		{"inside tmux, without 2026 and 2027", inTmux, []Option{tea},
			safe + xtversionQ + tmux(osc99Q) + tmux(kittyGraphicsQ) + osc10Q + paletteQ + da1Q},
		{"unknown SSH peer", ssh, nil, safe + da1Q},
		{"unknown SSH peer, without 2026 and 2027", ssh, []Option{tea}, safe + da1Q},
		{"unknown SSH peer, no heuristic", ssh, []Option{WithoutHeuristic()}, safe + gated + da1Q},
		{"Apple Terminal", apple, nil, safe + da1Q},
		{"Apple Terminal, without 2026 and 2027", apple, []Option{tea}, safe + da1Q},
		{"no background query", local, []Option{WithoutBackgroundRequest(), tea},
			kittyKeyboardQ + modesQ + dsr996Q + da2Q + gated + da1Q},
		{"without 2031", local, []Option{WithoutModeQuery(2031), tea},
			kittyKeyboardQ + "\x1b[?2048$p\x1b[?1004$p" + dsr996Q + osc11Q + da2Q + gated + da1Q},
	}
	for _, c := range cases {
		_, es := started(c.env, c.o...)
		if len(es) == 0 || es[0].Kind != ProbeWrite || es[0].Data != c.want {
			t.Errorf("%s: first event %+v\nwant the batch\n %q", c.name, es, c.want)
		}
		if n := kinds(es, ProbeWrite); n != 1 {
			t.Errorf("%s: %d writes on Start, want 1", c.name, n)
		}
	}
}

// TestProbeReadsHostModes: a mode WithoutModeQuery names, and 2026 and
// 2027, are read from a report whoever asked; another mode the batch did
// not ask for is not.
func TestProbeReadsHostModes(t *testing.T) {
	p, _ := started(local, WithoutModeQuery(2026, 2027, 2031))
	reply(p, "\x1b[?2026;2$y", "\x1b[?2027;0$y", "\x1b[?2031;1$y")
	c := p.Caps()
	yes, no := Fact[Support]{Value: Supported, Origin: Queried}, Fact[Support]{Value: Unsupported, Origin: Queried}
	if c.SyncOutput != yes || c.GraphemeWidth != no || c.ColorSchemeReports != yes {
		t.Errorf("SyncOutput %+v, GraphemeWidth %+v, ColorSchemeReports %+v", c.SyncOutput, c.GraphemeWidth, c.ColorSchemeReports)
	}
	q, _ := started(local, WithoutQueries())
	reply(q, "\x1b[?2031;1$y")
	if c := q.Caps(); c.ColorSchemeReports != (Fact[Support]{}) {
		t.Errorf("a report nobody asked for set ColorSchemeReports %+v", c.ColorSchemeReports)
	}
}

// TestProbeTimeout: with no reply, HandleTimeout before the deadline does
// nothing; at it, it ends the probe once, with TimedOut; a second does
// nothing.
func TestProbeTimeout(t *testing.T) {
	p, _ := started(local, WithTimeout(time.Second))
	deadline, ok := p.Timeout()
	if !ok || !deadline.Equal(t0.Add(time.Second)) {
		t.Fatalf("Timeout() = %v, %v; want %v", deadline, ok, t0.Add(time.Second))
	}
	p.HandleTimeout(deadline.Add(-time.Nanosecond))
	if es := events(p); len(es) != 0 {
		t.Fatalf("before the deadline: %+v", es)
	}
	p.HandleTimeout(deadline)
	es := events(p)
	if len(es) != 1 || es[0].Kind != ProbeDone || !es[0].Caps.TimedOut || es[0].Caps.Complete {
		t.Fatalf("at the deadline: %+v, want one ProbeDone, timed out", es)
	}
	if es[0].Caps.KittyKeyboard != (Fact[Support]{}) {
		t.Errorf("an unanswered fact is %+v, want unknown", es[0].Caps.KittyKeyboard)
	}
	if _, ok := p.Timeout(); ok {
		t.Error("Timeout after ProbeDone")
	}
	p.HandleTimeout(deadline.Add(time.Hour))
	reply(p, "\x1b[?62c")
	if es := events(p); len(es) != 0 {
		t.Errorf("after the end: %+v", es)
	}
	if _, ok := NewProbe(WithoutQueries()).Timeout(); ok {
		t.Error("a disabled probe has a deadline")
	}
}

// TestProbeSentinel: DA1 alone marks every query sent unsupported and ends
// the probe; a reply after it still counts.
func TestProbeSentinel(t *testing.T) {
	p, _ := started(local, WithoutModeQuery(2026, 2027))
	es := reply(p, "\x1b[?62;22c")
	if kinds(es, ProbeDone) != 1 {
		t.Fatalf("DA1 queued %+v, want ProbeDone", es)
	}
	c := p.Caps()
	no := Fact[Support]{Value: Unsupported, Origin: Queried}
	for name, f := range map[string]Fact[Support]{
		"KittyKeyboard": c.KittyKeyboard, "ColorSchemeReports": c.ColorSchemeReports,
		"InBandResize": c.InBandResize, "FocusEvents": c.FocusEvents,
		"DesktopNotify": c.DesktopNotify, "KittyGraphics": c.KittyGraphics, "Sixel": c.Sixel,
	} {
		if f != no {
			t.Errorf("%s = %+v after DA1 alone, want unsupported from a query", name, f)
		}
	}
	if c.SyncOutput.Origin != NotQueried {
		t.Errorf("SyncOutput %+v, which only tea asked for", c.SyncOutput)
	}
	reply(p, "\x1b]99;i=termcap:p=?;a=focus\a")
	if c := p.Caps(); c.DesktopNotify != (Fact[Support]{Value: Supported, Origin: Queried}) {
		t.Errorf("DesktopNotify = %+v after a late reply, want supported", c.DesktopNotify)
	}
	// The probe's own 2026 and 2027 carry no silence, as tea's do not.
	q, _ := started(local)
	reply(q, "\x1b[?62c")
	if c := q.Caps(); c.SyncOutput.Origin != NotQueried || c.GraphemeWidth.Origin != NotQueried {
		t.Errorf("unanswered 2026 and 2027 read as %+v, %+v", c.SyncOutput, c.GraphemeWidth)
	}
}

// TestProbeDuplicateDA1: a second DA1, as Windows Terminal sends under SSH
// (MADR A2.1), changes nothing.
func TestProbeDuplicateDA1(t *testing.T) {
	p, _ := started(local)
	first := reply(p, "\x1b[?62;22c")
	before := p.Caps()
	second := reply(p, "\x1b[?62;22c")
	if kinds(first, ProbeDone) != 1 || len(second) != 0 {
		t.Fatalf("first DA1 %+v, second %+v; want one ProbeDone, then nothing", first, second)
	}
	if !reflect.DeepEqual(before, p.Caps()) {
		t.Errorf("a second DA1 changed Caps:\n%+v\n%+v", before, p.Caps())
	}
}

// TestProbeHooks: a hook's ProbeRunHook comes only when an option sets
// it, the console host's only on Windows; HandleHook sets the facts.
func TestProbeHooks(t *testing.T) {
	hook := func(v bool) func() (bool, bool) { return func() (bool, bool) { return v, true } }
	if _, es := started(local); kinds(es, ProbeRunHook) != 0 {
		t.Errorf("a hook with none set: %+v", es)
	}
	_, es := started(local, WithGOOS("linux"), WithAppearanceHook(hook(true)), WithConsoleHost(hook(true)))
	if kinds(es, ProbeRunHook) != 1 || es[1].Hook != HookAppearance {
		t.Errorf("Linux: %+v, want the appearance hook only", es)
	}
	p, es := started(Env{}, WithGOOS("windows"), WithConsoleHost(hook(false)))
	if kinds(es, ProbeRunHook) != 1 || es[1].Hook != HookConsoleHost || es[1].Run == nil {
		t.Fatalf("Windows: %+v, want the console host", es)
	}
	v, ok := es[1].Run()
	p.HandleHook(es[1].Hook, v, ok)
	if got := p.Caps().LegacyConsole; got != (Fact[bool]{Value: false, Origin: Queried}) {
		t.Errorf("console host said ConPTY: %+v", got)
	}
	d, _ := started(Env{"TERM=xterm-kitty", "COLORFGBG=0;15"}, WithAppearanceHook(hook(true)))
	d.HandleHook(HookAppearance, true, true)
	if got := d.Caps().Dark; got != (Fact[bool]{Value: true, Origin: Heuristic, Reason: ReasonDesktopAppearance}) {
		t.Errorf("appearance hook: Dark %+v", got)
	}
	d.HandleHook(HookAppearance, false, false)
	if got := d.Caps().Dark; !got.Value {
		t.Errorf("a hook that cannot tell changed Dark: %+v", got)
	}
	// A disabled probe hands its hooks over too, and ends.
	_, es = started(local, WithoutQueries(), WithAppearanceHook(hook(true)))
	if kinds(es, ProbeRunHook) != 1 || kinds(es, ProbeDone) != 1 || written(es) != "" {
		t.Errorf("disabled: %+v", es)
	}
}

// TestProbeNoEvent: a fresh probe, and one drained, give the zero event.
func TestProbeNoEvent(t *testing.T) {
	if e := NewProbe().NextEvent(); !reflect.DeepEqual(e, ProbeEvent{}) {
		t.Errorf("a fresh probe: %+v", e)
	}
	p, _ := started(local)
	if e := p.NextEvent(); !reflect.DeepEqual(e, ProbeEvent{}) {
		t.Errorf("a drained probe: %+v", e)
	}
}

// TestProbeColorScheme: a DSR 997 after the probe queues ProbeColorScheme,
// then the background query unless declined; and mode 2031 is set once
// the terminal supports it, and reset by Restore.
func TestProbeColorScheme(t *testing.T) {
	for _, o := range [][]Option{nil, {WithoutBackgroundRequest()}} {
		p, _ := started(local, o...)
		reply(p, "\x1b[?62c")
		es := reply(p, "\x1b[?997;2n")
		want := []ProbeEvent{{Kind: ProbeColorScheme}, {Kind: ProbeWrite, Data: osc11Q}}
		if len(o) > 0 {
			want = want[:1]
		}
		if !reflect.DeepEqual(es, want) {
			t.Errorf("options %d: %+v, want %+v", len(o), es, want)
		}
		if c := p.Caps(); c.Dark != (Fact[bool]{Value: false, Origin: Queried}) {
			t.Errorf("Dark = %+v, want light from DSR 997", c.Dark)
		}
	}
	p, _ := started(local)
	if es := reply(p, "\x1b[?997;1n"); kinds(es, ProbeColorScheme) != 0 {
		t.Errorf("ProbeColorScheme for the probe's own DSR 996 reply: %+v", es)
	}
	es := reply(p, "\x1b[?2031;2$y", "\x1b[?2031;2$y")
	if written(es) != "\x1b[?2031h" || p.Restore() != "\x1b[?2031l" {
		t.Errorf("mode 2031: wrote %q, Restore %q", written(es), p.Restore())
	}
	q, _ := started(local, WithoutColorSchemeUpdates())
	if es := reply(q, "\x1b[?2031;2$y"); written(es) != "" || q.Restore() != "" {
		t.Errorf("declined: wrote %q, Restore %q", written(es), q.Restore())
	}
}

// TestProbeBackground: OSC 11 sets Background and, with no DSR 997, Dark,
// by Bubble Tea's own lightness rule.
func TestProbeBackground(t *testing.T) {
	for raw, dark := range map[string]bool{
		"\x1b]11;rgb:0000/0000/0000\a": true, "\x1b]11;rgb:ffff/ffff/ffff\a": false,
		"\x1b]11;rgb:7f/7f/7f\a": true, "\x1b]11;rgb:80/80/80\a": false, "\x1b]11;rgb:fe/00/00\a": true, "\x1b]11;rgb:ff/00/00\a": false,
	} {
		p, _ := started(local)
		reply(p, raw)
		if c := p.Caps(); c.Background == nil || c.Dark != (Fact[bool]{Value: dark, Origin: Queried}) {
			t.Errorf("%q: Background %v, Dark %+v; want dark %v", raw, c.Background, c.Dark, dark)
		}
	}
}

// TestProbeReplyTooLong: a reply over 1 KiB is not parsed; the query it
// would answer gets the reason, and is not marked silent.
func TestProbeReplyTooLong(t *testing.T) {
	long, _ := started(local)
	reply(long, notifyReply(maxReply+1), "\x1b[?62c")
	if f := long.Caps().DesktopNotify; f != (Fact[Support]{Value: Unknown, Origin: Queried, Reason: ReasonReplyTooLong}) {
		t.Errorf("a 1025-byte reply: %+v, want unparsed with the reason", f)
	}
	eight, _ := started(local)
	reply(eight, "\x9d"+notifyReply(maxReply + 2)[2:]) // 1025 bytes, the introducer one byte
	if f := eight.Caps().DesktopNotify; f.Reason != ReasonReplyTooLong {
		t.Errorf("an 8-bit OSC over 1 KiB: %+v", f)
	}
	parsed := false
	added := Query{Name: "x", Seq: fixed("\x1b[?5n"), Parse: func(r Reply, _ *Caps) bool { parsed = parsed || r.Raw != ""; return r.Raw != "" }}
	p, _ := started(local, WithQuery(added))
	reply(p, "\x1b["+strings.Repeat("1", 1030)+"n")
	if parsed {
		t.Error("an added query was given a reply over 1 KiB")
	}
}

// TestProbeJetBrains: nothing is sent; the probe ends at once, with the
// reason on each query fact.
func TestProbeJetBrains(t *testing.T) {
	p, es := started(Env{"TERMINAL_EMULATOR=JetBrains-JediTerm", "TERM=xterm-256color"}, WithoutHeuristic())
	if written(es) != "" || kinds(es, ProbeDone) != 1 || p.Awaiting() {
		t.Fatalf("JetBrains: %+v, awaiting %v", es, p.Awaiting())
	}
	c := p.Caps()
	for _, f := range queryFacts(&c) {
		if *f != (Fact[Support]{Value: Unknown, Origin: NotQueried, Reason: ReasonJetBrainsPaints}) {
			t.Errorf("a query fact is %+v, want unknown with the JetBrains reason", *f)
		}
	}
}

// TestProbeAwaiting: before the batch, nothing; after it, until DA1 and
// every query are answered.
func TestProbeAwaiting(t *testing.T) {
	p := NewProbe()
	if p.Awaiting() {
		t.Error("awaiting before Start")
	}
	q, _ := started(Env{"TERM=xterm-256color", "SSH_TTY=x"}, WithoutModeQuery(2026, 2027))
	if !q.Awaiting() {
		t.Error("not awaiting after the batch")
	}
	reply(q, "\x1b[?1u", "\x1b[?2031;2$y", "\x1b[?2048;2$y", "\x1b[?1004;2$y", "\x1b[?997;1n",
		"\x1b]11;rgb:0/0/0\a", "\x1b[>1;2;3c", "\x1b[?62c")
	if q.Awaiting() {
		t.Error("awaiting with every query answered")
	}
}

func TestGatedAllowed(t *testing.T) {
	cases := []struct {
		env  Env
		want bool
	}{
		{Env{"TERM=xterm-256color"}, true},
		{Env{}, true},
		{Env{"TERM_PROGRAM=Apple_Terminal", "TERM=xterm-256color"}, false},
		{Env{"TERM=xterm-256color", "SSH_TTY=/dev/pts/0"}, false},
		{Env{"TERM=xterm-256color", "SSH_CONNECTION=a 1 b 22"}, false},
		{Env{"TERM=xterm-kitty", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=xterm-ghostty", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=wezterm", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=alacritty", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=foot", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=xterm-256color", "LC_TERMINAL=iTerm2", "SSH_TTY=/dev/pts/0"}, true},
		{Env{"TERM=tmux-256color", "TMUX=x", "SSH_TTY=/dev/pts/0"}, false},
	}
	for _, c := range cases {
		if got := gatedAllowed(c.env, "linux"); got != c.want {
			t.Errorf("gatedAllowed(%v) = %v, want %v", c.env, got, c.want)
		}
	}
}

// TestQuerySynchronizedOutput: Bubble Tea's rule, case by case.
func TestQuerySynchronizedOutput(t *testing.T) {
	for _, c := range []struct {
		env  Env
		want bool
	}{
		{Env{"TERM=xterm-256color"}, true},
		{Env{"TERM=xterm-256color", "SSH_TTY=x"}, false},
		{Env{"TERM=xterm-kitty", "SSH_TTY=x"}, true},
		{Env{"TERM=xterm-256color", "SSH_TTY=x", "WT_SESSION=x"}, true},
		{Env{"TERM=xterm-256color", "TERM_PROGRAM=Apple_Terminal"}, false},
		{Env{"TERM=xterm-256color", "TERM_PROGRAM=WezTerm"}, true},
		{Env{"TERM=xterm-256color", "TERM_PROGRAM=WezTerm", "SSH_TTY=x"}, false},
	} {
		if got := querySynchronizedOutput(c.env, "linux"); got != c.want {
			t.Errorf("%v: %v, want %v", c.env, got, c.want)
		}
	}
}
