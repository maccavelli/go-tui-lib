package termcap

import (
	"reflect"
	"strings"
	"testing"
)

// querySeq is each built-in query's bytes on local, by name, in the batch's
// order (docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md
// Step 1).
var querySeq = []struct{ name, seq string }{
	{"kitty-keyboard", kittyKeyboardQ},
	{"mode-2031", "\x1b[?2031$p"},
	{"mode-2048", "\x1b[?2048$p"},
	{"mode-1004", "\x1b[?1004$p"},
	{"dsr-996", dsr996Q},
	{"background", osc11Q},
	{"da2", da2Q},
	{"xtversion", xtversionQ},
	{"notify-query", osc99Q},
	{"kitty-graphics", kittyGraphicsQ},
	{"foreground", osc10Q},
	{"palette", paletteQ},
	{"mode-2026", "\x1b[?2026$p"},
	{"mode-2027", "\x1b[?2027$p"},
}

// TestQueryNames: every built-in query has its token, in the batch's
// order, and queryNames, the tokens WithoutQuery takes, is that table.
func TestQueryNames(t *testing.T) {
	var got []string
	for _, q := range NewProbe(WithGOOS("linux")).batch(local) {
		got = append(got, q.Name)
	}
	var want []string
	for _, q := range querySeq {
		want = append(want, q.name)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the batch's names\n %q\nwant\n %q", got, want)
	}
	if !reflect.DeepEqual(queryNames, want) {
		t.Errorf("queryNames\n %q\nwant\n %q", queryNames, want)
	}
}

// TestWithoutQuery: each name leaves its own query out, and the rest of
// the batch is today's, byte for byte, DA1 last.
func TestWithoutQuery(t *testing.T) {
	for _, drop := range querySeq {
		var want strings.Builder
		for _, q := range querySeq {
			if q.name != drop.name {
				want.WriteString(q.seq)
			}
		}
		want.WriteString(da1Q)
		_, es := started(local, WithGOOS("linux"), WithoutQuery(drop.name))
		if got := written(es); got != want.String() {
			t.Errorf("without %s: wrote\n %q\nwant\n %q", drop.name, got, want.String())
		}
	}
	_, es := started(local, WithGOOS("linux"), WithoutQuery("da2", "palette", "mode-2026"))
	if got := written(es); strings.Contains(got, da2Q) || strings.Contains(got, paletteQ) ||
		strings.Contains(got, "\x1b[?2026$p") || !strings.HasSuffix(got, "\x1b[?2027$p"+da1Q) {
		t.Errorf("without three: %q", got)
	}
}

// TestWithoutQueryReplies: a query left out is not offered its reply, and
// is not unsupported at the sentinel; leaving out background also leaves
// out the background query that follows a DSR 997 report.
func TestWithoutQueryReplies(t *testing.T) {
	p, _ := started(local, WithGOOS("linux"), WithoutQuery("kitty-keyboard", "background"))
	es := reply(p, "\x1b[?1u", "\x1b[?62c")
	if c := p.Caps(); c.KittyKeyboard != (Fact[Support]{}) {
		t.Errorf("KittyKeyboard %+v, want untouched: its query was not sent", c.KittyKeyboard)
	}
	es = append(es, reply(p, "\x1b[?997;1n")...)
	if got := written(es); strings.Contains(got, osc11Q) {
		t.Errorf("after DSR 997, wrote %q: the background query was left out", got)
	}
	q, _ := started(local, WithGOOS("linux"))
	if got := written(reply(q, "\x1b[?62c", "\x1b[?997;1n")); !strings.Contains(got, osc11Q) {
		t.Errorf("by default, after DSR 997, wrote %q, want the background query", got)
	}
}

// TestWithoutQueryAdded: a WithQuery's Name names it.
func TestWithoutQueryAdded(t *testing.T) {
	added := Query{Name: "dsr-5", Seq: fixed("\x1b[5n"), Parse: func(Reply, *Caps) bool { return false }}
	_, es := started(local, WithGOOS("linux"), WithQuery(added))
	if !strings.Contains(written(es), "\x1b[5n") {
		t.Fatalf("the added query is not sent: %q", written(es))
	}
	_, es = started(local, WithGOOS("linux"), WithQuery(added), WithoutQuery("dsr-5"))
	if got := written(es); strings.Contains(got, "\x1b[5n") || !strings.HasSuffix(got, da1Q) {
		t.Errorf("without the added query: %q", got)
	}
}

// TestWithoutQueryUnknownName: a name no query has, DA1's own among them,
// sends nothing and ends the probe at once, with ReasonUnknownQuery on
// each query fact; the hooks still run.
func TestWithoutQueryUnknownName(t *testing.T) {
	dark := func() (bool, bool) { return true, true }
	for _, name := range []string{"da1", "nope", "", "Kitty-Keyboard"} {
		p, es := started(local, WithGOOS("linux"), WithoutQuery("background", name), WithAppearanceHook(dark))
		if written(es) != "" || kinds(es, ProbeDone) != 1 || kinds(es, ProbeRunHook) != 1 || p.Awaiting() {
			t.Errorf("%q: %+v, awaiting %v; want nothing written, the hook, and done", name, es, p.Awaiting())
			continue
		}
		c := p.Caps()
		for _, f := range queryFacts(&c) {
			if *f != (Fact[Support]{Value: Unknown, Origin: NotQueried, Reason: ReasonUnknownQuery}) {
				t.Errorf("%q: a query fact is %+v, want unknown with ReasonUnknownQuery", name, *f)
			}
		}
	}
	// "" names no query, even an added query that has no name.
	unnamed := Query{Seq: fixed("\x1b[5n"), Parse: func(Reply, *Caps) bool { return false }}
	if _, es := started(local, WithGOOS("linux"), WithQuery(unnamed), WithoutQuery("")); written(es) != "" {
		t.Errorf(`WithoutQuery("") beside an unnamed query wrote %q, want nothing`, written(es))
	}
}

// TestWithoutGatedQueries: the gated queries stay off where the heuristic
// would send them, an SSH peer it names included, and under
// WithoutHeuristic; an editor's reason is not given for it.
func TestWithoutGatedQueries(t *testing.T) {
	safe := kittyKeyboardQ + modesQ + dsr996Q + osc11Q + da2Q
	sshKitty := Env{"TERM=xterm-kitty", "SSH_TTY=/dev/pts/1"}
	for _, c := range []struct {
		name string
		env  Env
		o    []Option
		want string
	}{
		{"local", local, nil, safe + syncQ + da1Q},
		{"an SSH peer the heuristic names", sshKitty, nil, safe + syncQ + da1Q},
		{"with WithoutHeuristic", Env{"TERM=xterm-256color", "SSH_TTY=/dev/pts/1"}, []Option{WithoutHeuristic()}, safe + da1Q},
	} {
		_, es := started(c.env, append([]Option{WithGOOS("linux"), WithoutGatedQueries()}, c.o...)...)
		if got := written(es); got != c.want {
			t.Errorf("%s: wrote\n %q\nwant\n %q", c.name, got, c.want)
		}
	}
	_, es := started(sshKitty, WithGOOS("linux"))
	if got := written(es); !strings.Contains(got, xtversionQ) {
		t.Errorf("by default, an SSH peer the heuristic names is sent the gated queries: %q", got)
	}
	p, _ := started(Env{"TERM=xterm-256color", "NVIM=/tmp/nvim.sock"}, WithGOOS("linux"), WithoutGatedQueries())
	if c := p.Caps(); c.DesktopNotify.Reason != "" || c.KittyGraphics.Reason != "" {
		t.Errorf("an editor's reason under WithoutGatedQueries: %+v, %+v", c.DesktopNotify, c.KittyGraphics)
	}
}

// TestWithoutQueriesIsWithDisabled: the deprecated name and its successor
// give the same probe.
func TestWithoutQueriesIsWithDisabled(t *testing.T) {
	for _, env := range []Env{local, {"TERMINAL_EMULATOR=JetBrains-JediTerm"}} {
		a, ae := started(env, WithGOOS("linux"), WithoutQueries())
		b, be := started(env, WithGOOS("linux"), WithDisabled())
		if written(ae) != "" || !reflect.DeepEqual(ae, be) || !reflect.DeepEqual(a.Caps(), b.Caps()) {
			t.Errorf("%v: WithoutQueries %+v, WithDisabled %+v", env, ae, be)
		}
	}
}

// TestProbeReadsEnvForItsGOOS: the probe's reads of its Env fold case by
// its GOOS, not the host's: a Windows probe on any host finds ssh_tty and
// term_program, and a Linux probe on any host does not
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md A2).
func TestProbeReadsEnvForItsGOOS(t *testing.T) {
	env := Env{"term=xterm-256color", "ssh_tty=/dev/pts/1", "term_program=WezTerm"}
	win, es := started(env, WithGOOS("windows"))
	if c := win.Caps(); !c.Remote.Value || c.EnvBrand.Value != BrandWezTerm {
		t.Errorf("windows: Remote %+v, EnvBrand %+v; want remote, WezTerm", c.Remote, c.EnvBrand)
	}
	if got := written(es); strings.Contains(got, xtversionQ) {
		t.Errorf("windows: an unknown SSH peer was sent the gated queries: %q", got)
	}
	lin, es := started(env, WithGOOS("linux"))
	if c := lin.Caps(); c.Remote.Value || c.EnvBrand.Value != BrandUnknown {
		t.Errorf("linux: Remote %+v, EnvBrand %+v; want local, unknown", c.Remote, c.EnvBrand)
	}
	if got := written(es); !strings.Contains(got, xtversionQ) {
		t.Errorf("linux: a local terminal was not sent the gated queries: %q", got)
	}
	if !gatedAllowed(Env{"term_program=Apple_Terminal"}, "linux") || gatedAllowed(Env{"term_program=Apple_Terminal"}, "windows") {
		t.Error("gatedAllowed does not read term_program by its goos")
	}
	if querySynchronizedOutput(Env{"term=xterm", "ssh_tty=x"}, "windows") || !querySynchronizedOutput(Env{"term=xterm", "ssh_tty=x"}, "linux") {
		t.Error("querySynchronizedOutput does not read ssh_tty by its goos")
	}
	if FromEnv(Env{"tmux=x"}, "windows").Mux != Tmux || FromEnv(Env{"tmux=x"}, "linux").Mux != NoMux {
		t.Error("FromEnv does not read tmux by its goos")
	}
}
