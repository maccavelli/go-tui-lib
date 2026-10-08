package launch

import (
	"errors"
	"io"
	"os"
	"runtime"
	"slices"
	"testing"

	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/termcap"
)

// ttyOpener is a controlling terminal for decideWith: it counts its opens, and
// returns in and out, or fails.
type ttyOpener struct {
	in, out *fake
	fail    bool
	opens   int
}

func (o *ttyOpener) open() (io.ReadCloser, io.WriteCloser, error) {
	o.opens++
	if o.fail {
		return nil, nil, errors.New("no controlling terminal")
	}
	return o.in, o.out, nil
}

// oneTTY is a controlling terminal that is one file, as on Unix.
func oneTTY() *ttyOpener {
	f := terminal(100, 50)
	return &ttyOpener{in: f, out: f}
}

func TestDecideRules(t *testing.T) {
	type want struct {
		reason Reason
		in, ui Target
	}
	plainIs := func(r Reason) want { return want{reason: r} }
	both := want{ReasonTerminal, TargetStream, TargetStream}
	for _, c := range []struct {
		name          string
		in, out, err  *fake
		env           []string
		cfg           Config
		ttyFails      bool
		want          want
		wantTTYOpened bool
	}{
		{name: "both terminals", in: terminal(80, 24), out: terminal(80, 24), env: base, want: both},
		{name: "plain requested", in: terminal(80, 24), out: terminal(80, 24), env: base,
			cfg: Config{Choice: ChoicePlain}, want: plainIs(ReasonRequestedPlain)},

		// Rule 2: the environment's vetoes, in order.
		{name: "no-input variable set", in: terminal(80, 24), out: terminal(80, 24), env: with("PROG_NO_TUI=1"),
			cfg: Config{NoInputEnv: "PROG_NO_TUI"}, want: plainIs(ReasonNoInputVariable)},
		{name: "no-input variable named, unset", in: terminal(80, 24), out: terminal(80, 24), env: base,
			cfg: Config{NoInputEnv: "PROG_NO_TUI"}, want: both},
		{name: "no-input variable set empty", in: terminal(80, 24), out: terminal(80, 24), env: with("PROG_NO_TUI="),
			cfg: Config{NoInputEnv: "PROG_NO_TUI"}, want: both},
		{name: "CI=true", in: terminal(80, 24), out: terminal(80, 24), env: with("CI=true"), want: plainIs(ReasonCI)},
		{name: "CI=1", in: terminal(80, 24), out: terminal(80, 24), env: with("CI=1"), want: plainIs(ReasonCI)},
		{name: "CI=false", in: terminal(80, 24), out: terminal(80, 24), env: with("CI=false"), want: both},
		{name: "CI=0", in: terminal(80, 24), out: terminal(80, 24), env: with("CI=0"), want: both},
		{name: "CI empty", in: terminal(80, 24), out: terminal(80, 24), env: with("CI="), want: both},
		{name: "the last CI wins", in: terminal(80, 24), out: terminal(80, 24), env: with("CI=true", "CI=false"), want: both},

		// The order, with two causes at once.
		{name: "plain before the vetoes", in: terminal(80, 24), out: terminal(80, 24), env: with("CI=true", "PROG_NO_TUI=1"),
			cfg: Config{Choice: ChoicePlain, NoInputEnv: "PROG_NO_TUI"}, want: plainIs(ReasonRequestedPlain)},
		{name: "no-input before CI", in: terminal(80, 24), out: terminal(80, 24), env: with("CI=true", "PROG_NO_TUI=1"),
			cfg: Config{NoInputEnv: "PROG_NO_TUI"}, want: plainIs(ReasonNoInputVariable)},
		{name: "CI before TERM=dumb", in: terminal(80, 24), out: terminal(80, 24), env: []string{"TERM=dumb", "CI=true"},
			want: plainIs(ReasonCI)},
		{name: "TERM=dumb before the input", in: pipe(), out: pipe(), env: []string{"TERM=dumb"},
			want: plainIs(ReasonDumbTerminal)},
		{name: "the input before the output", in: pipe(), out: pipe(), env: base, want: plainIs(ReasonInputNotTerminal)},

		// Rule 3, under each choice.
		{name: "TERM=dumb, auto", in: terminal(80, 24), out: terminal(80, 24), env: []string{"TERM=dumb"},
			want: plainIs(ReasonDumbTerminal)},
		{name: "TERM=dumb, tui", in: terminal(80, 24), out: terminal(80, 24), env: []string{"TERM=dumb"},
			cfg: Config{Choice: ChoiceTUI}, want: plainIs(ReasonDumbTerminal)},
		{name: "TERM=dumb, plain", in: terminal(80, 24), out: terminal(80, 24), env: []string{"TERM=dumb"},
			cfg: Config{Choice: ChoicePlain}, want: plainIs(ReasonRequestedPlain)},

		// ChoiceTUI skips the environment's vetoes, and nothing else.
		{name: "tui skips the vetoes", in: terminal(80, 24), out: terminal(80, 24), env: with("CI=true", "PROG_NO_TUI=1"),
			cfg: Config{Choice: ChoiceTUI, NoInputEnv: "PROG_NO_TUI"}, want: both},
		{name: "tui needs an input", in: pipe(), out: terminal(80, 24), env: base,
			cfg: Config{Choice: ChoiceTUI}, want: plainIs(ReasonInputNotTerminal)},

		// Rules 4 and 5.
		{name: "input not a terminal", in: pipe(), out: terminal(80, 24), env: base, want: plainIs(ReasonInputNotTerminal)},
		{name: "output not a terminal", in: terminal(80, 24), out: pipe(), err: terminal(80, 24), env: base,
			want: plainIs(ReasonOutputNotTerminal)},
		{name: "UIOnErr draws on Err", in: terminal(80, 24), out: pipe(), err: terminal(80, 24), env: base,
			cfg: Config{UIOnErr: true}, want: want{ReasonTerminal, TargetStream, TargetErr}},
		{name: "UIOnErr, Err not a terminal", in: terminal(80, 24), out: pipe(), err: pipe(), env: base,
			cfg: Config{UIOnErr: true}, want: plainIs(ReasonOutputNotTerminal)},
		{name: "OpenTTY reads the terminal", in: pipe(), out: terminal(80, 24), env: base,
			cfg: Config{OpenTTY: true}, want: want{ReasonTerminal, TargetTTY, TargetStream}, wantTTYOpened: true},
		{name: "OpenTTY draws on the terminal", in: terminal(80, 24), out: pipe(), err: pipe(), env: base,
			cfg: Config{OpenTTY: true}, want: want{ReasonTerminal, TargetStream, TargetTTY}, wantTTYOpened: true},
		{name: "OpenTTY for both", in: pipe(), out: pipe(), env: base,
			cfg: Config{OpenTTY: true}, want: want{ReasonTerminal, TargetTTY, TargetTTY}, wantTTYOpened: true},
		{name: "Err before the terminal", in: terminal(80, 24), out: pipe(), err: terminal(80, 24), env: base,
			cfg: Config{UIOnErr: true, OpenTTY: true}, want: want{ReasonTerminal, TargetStream, TargetErr}},
		{name: "OpenTTY fails, input", in: pipe(), out: terminal(80, 24), env: base,
			cfg: Config{OpenTTY: true}, ttyFails: true, want: plainIs(ReasonInputNotTerminal), wantTTYOpened: true},
		{name: "OpenTTY fails, output", in: terminal(80, 24), out: pipe(), err: pipe(), env: base,
			cfg: Config{OpenTTY: true}, ttyFails: true, want: plainIs(ReasonOutputNotTerminal), wantTTYOpened: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := oneTTY()
			o.fail = c.ttyFails
			var errStream io.Writer
			if c.err != nil {
				errStream = c.err
			}
			d := decideWith(streams(c.in, c.out, errStream, c.env...), c.cfg, "linux", o.open)
			got := want{d.Reason, d.In, d.UI}
			if got != c.want {
				t.Errorf("decideWith = %+v, want %+v", got, c.want)
			}
			if d.Interactive != (c.want.reason == ReasonTerminal) {
				t.Errorf("Interactive = %v with reason %s", d.Interactive, d.Reason)
			}
			if !d.Interactive && d.UIProfile != colorprofile.NoTTY {
				t.Errorf("a plain decision's UIProfile is %s, want NoTTY", d.UIProfile)
			}
			if (o.opens > 0) != c.wantTTYOpened {
				t.Errorf("the controlling terminal was opened %d times", o.opens)
			}
		})
	}
}

// TestDecideOpenTTYNotOnWindows: on Windows, OpenTTY is ignored, and the
// controlling terminal is never opened (0013-MADR A1.9).
func TestDecideOpenTTYNotOnWindows(t *testing.T) {
	for _, c := range []struct {
		name    string
		in, out *fake
		want    Reason
	}{
		{"input", pipe(), terminal(80, 24), ReasonInputNotTerminal},
		{"output", terminal(80, 24), pipe(), ReasonOutputNotTerminal},
	} {
		o := oneTTY()
		d := decideWith(streams(c.in, c.out, nil, base...), Config{Choice: ChoiceTUI, OpenTTY: true}, "windows", o.open)
		if d.Interactive || d.Reason != c.want || o.opens != 0 {
			t.Errorf("%s on windows: %v, %s, opened %d times; want plain, %s, never opened", c.name, d.Interactive, d.Reason, o.opens, c.want)
		}
		// The same decision elsewhere opens the terminal.
		o = oneTTY()
		if d := decideWith(streams(c.in, c.out, nil, base...), Config{Choice: ChoiceTUI, OpenTTY: true}, "linux", o.open); !d.Interactive || o.opens != 1 {
			t.Errorf("%s on linux: %v, opened %d times; want interactive through the terminal", c.name, d.Interactive, o.opens)
		}
	}
}

func TestDecideOpensOnlyWhenNeeded(t *testing.T) {
	for _, c := range []struct {
		name     string
		in, out  *fake
		err      *fake
		cfg      Config
		env      []string
		distinct bool // in and out are two files, as on Windows
		fails    bool
		badClose bool // the file opens, and does not close cleanly
		opens    int
		reason   Reason
	}{
		{name: "both terminals", in: terminal(80, 24), out: terminal(80, 24), opens: 0},
		{name: "plain requested", in: pipe(), out: pipe(), cfg: Config{Choice: ChoicePlain}, opens: 0},
		{name: "TERM=dumb", in: pipe(), out: pipe(), env: []string{"TERM=dumb"}, opens: 0},
		{name: "Err serves", in: terminal(80, 24), out: pipe(), err: terminal(80, 24), cfg: Config{UIOnErr: true}, opens: 0},
		{name: "input", in: pipe(), out: terminal(80, 24), opens: 1},
		{name: "output", in: terminal(80, 24), out: pipe(), err: pipe(), opens: 1},
		{name: "both, one file", in: pipe(), out: pipe(), opens: 1},
		{name: "both, two files", in: pipe(), out: pipe(), distinct: true, opens: 1},
		{name: "fails", in: pipe(), out: pipe(), fails: true, opens: 1, reason: ReasonInputNotTerminal},
		{name: "does not close cleanly", in: pipe(), out: pipe(), badClose: true, opens: 1, reason: ReasonInputNotTerminal},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := oneTTY()
			if c.distinct {
				o.out = terminal(100, 50)
			}
			o.fail = c.fails
			if c.badClose {
				o.in.closeErr = errors.New("close failed")
			}
			cfg := c.cfg
			cfg.OpenTTY = true
			env := c.env
			if env == nil {
				env = base
			}
			var errStream io.Writer
			if c.err != nil {
				errStream = c.err
			}
			d := decideWith(streams(c.in, c.out, errStream, env...), cfg, "linux", o.open)
			if c.reason != "" && d.Reason != c.reason {
				t.Errorf("reason %s, want %s", d.Reason, c.reason)
			}
			if o.opens != c.opens {
				t.Fatalf("opened %d times, want %d", o.opens, c.opens)
			}
			// Each distinct file it opened is closed exactly once.
			wantIn, wantOut := 0, 0
			if c.opens == 1 && !c.fails {
				wantIn, wantOut = 1, 1
			}
			if c.distinct {
				if o.in.closes() != wantIn || o.out.closes() != wantOut {
					t.Errorf("closes: in %d, out %d; want %d and %d", o.in.closes(), o.out.closes(), wantIn, wantOut)
				}
			} else if o.in.closes() != wantIn {
				t.Errorf("the one file was closed %d times, want %d", o.in.closes(), wantIn)
			}
		})
	}
}

func TestIsTerminalDevNull(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if isTerminal(f) {
		t.Errorf("%s is a terminal", os.DevNull)
	}
	if w, h := size(f); w != 0 || h != 0 {
		t.Errorf("size(%s) = %d, %d; want 0, 0", os.DevNull, w, h)
	}
	if isTerminal(nil) || isTerminal(pipe()) || !isTerminal(terminal(1, 1)) {
		t.Error("isTerminal does not take a stream's own answer")
	}
}

func TestColourEnv(t *testing.T) {
	term256 := "TERM=xterm-256color"
	// With TERM=dumb, colorprofile asks Windows for its build number unless
	// ConEmuANSI=ON, which it answers from the environment with TrueColor
	// (docs/decisions/0013-MADR-cli-integration-helpers.md A1.7). The forced
	// cases set it, so each OS has one exact answer.
	dumb := []string{"TERM=dumb", "ConEmuANSI=ON"}
	forced := colorprofile.ANSI
	if runtime.GOOS == "windows" {
		forced = colorprofile.TrueColor
	}
	for _, c := range []struct {
		name string
		env  []string
		want colorprofile.Profile
	}{
		{"no NO_COLOR", []string{term256}, colorprofile.ANSI256},
		{"NO_COLOR=1", []string{term256, "NO_COLOR=1"}, colorprofile.ASCII},
		{"NO_COLOR=yes", []string{term256, "NO_COLOR=yes"}, colorprofile.ASCII},
		{"NO_COLOR=x", []string{term256, "NO_COLOR=x"}, colorprofile.ASCII},
		{"NO_COLOR empty", []string{term256, "NO_COLOR="}, colorprofile.ANSI256},
		{"FORCE_COLOR=1", append(dumb, "FORCE_COLOR=1"), forced},
		{"FORCE_COLOR=true", append(dumb, "FORCE_COLOR=true"), forced},
		{"FORCE_COLOR=0", append(dumb, "FORCE_COLOR=0"), colorprofile.NoTTY},
		{"FORCE_COLOR=false", append(dumb, "FORCE_COLOR=false"), colorprofile.NoTTY},
		{"FORCE_COLOR empty", append(dumb, "FORCE_COLOR="), colorprofile.NoTTY},
		{"NO_COLOR wins over FORCE_COLOR", []string{term256, "NO_COLOR=yes", "FORCE_COLOR=1"}, colorprofile.ASCII},
		{"an existing CLICOLOR_FORCE stands", append(dumb, "CLICOLOR_FORCE=0", "FORCE_COLOR=1"), colorprofile.NoTTY},
	} {
		if got := profile(true, termcap.Env(c.env)); got != c.want {
			t.Errorf("%s: profile = %s, want %s", c.name, got, c.want)
		}
	}
	if got := profile(false, termcap.Env{term256, "FORCE_COLOR=1"}); got != colorprofile.NoTTY {
		t.Errorf("a stream that is not a terminal: profile = %s, want NoTTY", got)
	}

	// colourEnv leaves the input alone, and appends what it rewrites.
	in := termcap.Env{"NO_COLOR=yes", "FORCE_COLOR=1"}
	got := colourEnv(in)
	if want := []string{"NO_COLOR=yes", "FORCE_COLOR=1", "NO_COLOR=1", "CLICOLOR_FORCE=1"}; !slices.Equal(got, want) {
		t.Errorf("colourEnv = %q, want %q", got, want)
	}
	if len(in) != 2 {
		t.Error("colourEnv changed its input")
	}

	// Through Decide, on a fake terminal with no descriptor: the profile
	// comes from the environment alone. colorprofile.Detect would say NoTTY
	// here, since the stream has no Fd, and would read terminfo and run tmux
	// on a real one.
	d := decideWith(streams(terminal(80, 24), terminal(80, 24), nil, term256, "LANG=C.UTF-8"), Config{}, "linux", nil)
	if d.Profile != colorprofile.ANSI256 || d.UIProfile != colorprofile.ANSI256 {
		t.Errorf("Decide's profiles = %s, %s; want ANSI256 from TERM alone", d.Profile, d.UIProfile)
	}
	d = decideWith(streams(terminal(80, 24), pipe(), nil, term256), Config{}, "linux", nil)
	if d.Profile != colorprofile.NoTTY {
		t.Errorf("a redirected Out's profile = %s, want NoTTY", d.Profile)
	}
}

func TestTierLocale(t *testing.T) {
	for _, c := range []struct {
		env  []string
		goos string
		want glyph.Tier
	}{
		{[]string{"LANG=en_US.UTF-8"}, "linux", glyph.TierUnicode},
		{[]string{"LANG=en_US.utf8"}, "linux", glyph.TierUnicode},
		{[]string{"LANG=C.UTF-8"}, "darwin", glyph.TierUnicode},
		{[]string{"LANG=en_US.ISO-8859-1"}, "linux", glyph.TierASCII},
		// LC_ALL, then LC_CTYPE, then LANG: the first set decides.
		{[]string{"LC_ALL=C", "LANG=en_US.UTF-8"}, "linux", glyph.TierASCII},
		{[]string{"LC_CTYPE=en_US.UTF-8", "LANG=C"}, "linux", glyph.TierUnicode},
		{[]string{"LC_ALL=", "LC_CTYPE=C", "LANG=en_US.UTF-8"}, "linux", glyph.TierASCII},
		{[]string{"LC_ALL=en_US.UTF-8", "LC_CTYPE=C"}, "linux", glyph.TierUnicode},
		// None set.
		{nil, "linux", glyph.TierASCII},
		{nil, "darwin", glyph.TierASCII},
		{nil, "windows", glyph.TierUnicode},
		{[]string{"LANG=C"}, "windows", glyph.TierASCII},
	} {
		if got := tier(termcap.Env(c.env), c.goos); got != c.want {
			t.Errorf("tier(%q, %s) = %s, want %s", c.env, c.goos, got, c.want)
		}
	}
}

func TestDecideSize(t *testing.T) {
	for _, c := range []struct {
		name         string
		in, out, err *fake
		env          []string
		cfg          Config
		w, h         int
	}{
		{name: "interactive, Out", in: terminal(1, 1), out: terminal(120, 40), env: base, w: 120, h: 40},
		{name: "interactive, Err", in: terminal(1, 1), out: pipe(), err: terminal(90, 30), env: base,
			cfg: Config{UIOnErr: true}, w: 90, h: 30},
		{name: "interactive, the terminal", in: terminal(1, 1), out: pipe(), env: base,
			cfg: Config{OpenTTY: true}, w: 100, h: 50},
		{name: "plain, Out a terminal", in: terminal(1, 1), out: terminal(132, 43), env: base,
			cfg: Config{Choice: ChoicePlain}, w: 132, h: 43},
		{name: "plain, COLUMNS=100", in: pipe(), out: pipe(), env: with("COLUMNS=100"), w: 100},
		{name: "plain, COLUMNS=0", in: pipe(), out: pipe(), env: with("COLUMNS=0")},
		{name: "plain, COLUMNS=-1", in: pipe(), out: pipe(), env: with("COLUMNS=-1")},
		{name: "plain, COLUMNS=x", in: pipe(), out: pipe(), env: with("COLUMNS=x")},
		{name: "plain, no COLUMNS", in: pipe(), out: pipe(), env: base},
	} {
		var errStream io.Writer
		if c.err != nil {
			errStream = c.err
		}
		d := decideWith(streams(c.in, c.out, errStream, c.env...), c.cfg, "linux", oneTTY().open)
		if d.Width != c.w || d.Height != c.h {
			t.Errorf("%s: size %d×%d, want %d×%d", c.name, d.Width, d.Height, c.w, c.h)
		}
	}
}

func TestDecideGlyphs(t *testing.T) {
	d := Decide(streams(pipe(), pipe(), nil, "LANG=en_US.UTF-8"), Config{})
	if d.Glyphs != glyph.TierUnicode {
		t.Errorf("Decide's Glyphs = %s, want unicode", d.Glyphs)
	}
	if d.Interactive || d.Reason != ReasonInputNotTerminal {
		t.Errorf("Decide on two pipes = %v, %s", d.Interactive, d.Reason)
	}
	// The operating system reaches the tier: with no locale set, Windows
	// is Unicode and Linux is ASCII.
	for goos, want := range map[string]glyph.Tier{"windows": glyph.TierUnicode, "linux": glyph.TierASCII} {
		d := decideWith(streams(terminal(80, 24), terminal(80, 24), nil, "TERM=xterm"), Config{}, goos, nil)
		if d.Glyphs != want {
			t.Errorf("%s with no locale: Glyphs = %s, want %s", goos, d.Glyphs, want)
		}
	}
}
