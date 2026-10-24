package termcap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestFromEnvBrands(t *testing.T) {
	cases := []struct {
		name string
		env  Env
		want Brand
	}{
		{"Cursor's trace ID", Env{"CURSOR_TRACE_ID=x", "TERM_PROGRAM=vscode"}, BrandCursor},
		{"an editor fork over SSH", Env{"VSCODE_GIT_ASKPASS_MAIN=/home/u/.windsurf-server/askpass.js", "SSH_TTY=/dev/pts/0"}, BrandWindsurf},
		{"VS Code's askpass", Env{"VSCODE_GIT_ASKPASS_MAIN=/home/u/.vscode-server/askpass.js"}, BrandVSCode},
		{"Apple Terminal", Env{"TERM_PROGRAM=Apple_Terminal"}, BrandAppleTerminal},
		{"iTerm2", Env{"TERM_PROGRAM=iTerm.app"}, BrandITerm2},
		{"WezTerm", Env{"TERM_PROGRAM=WezTerm"}, BrandWezTerm},
		{"Ghostty", Env{"TERM_PROGRAM=ghostty"}, BrandGhostty},
		{"Warp", Env{"TERM_PROGRAM=WarpTerminal"}, BrandWarp},
		{"mintty", Env{"TERM_PROGRAM=mintty"}, BrandMintty},
		{"JetBrains, which also sets TERM_SESSION_ID", Env{"TERMINAL_EMULATOR=JetBrains-JediTerm", "TERM_SESSION_ID=1F2E3D4C-0000"}, BrandJetBrains},
		{"Apple Terminal's session ID", Env{"TERM_SESSION_ID=1F2E3D4C-5B6A-0000"}, BrandAppleTerminal},
		{"iTerm2's session ID", Env{"TERM_SESSION_ID=w0t1p0:1F2E3D4C"}, BrandITerm2},
		{"LC_TERMINAL over SSH", Env{"LC_TERMINAL=iTerm2", "SSH_CONNECTION=a 1 b 22", "TERM=xterm-256color"}, BrandITerm2},
		{"tmux's own TERM_PROGRAM is skipped", Env{"TERM_PROGRAM=tmux", "TMUX=x", "TERM=tmux-256color", "LC_TERMINAL=iTerm2"}, BrandITerm2},
		{"kitty by TERM", Env{"TERM=xterm-kitty"}, BrandKitty},
		{"kitty by its window", Env{"TERM=xterm-256color", "KITTY_WINDOW_ID=1"}, BrandKitty},
		{"Alacritty", Env{"TERM=alacritty"}, BrandAlacritty},
		{"foot", Env{"TERM=foot"}, BrandFoot},
		{"Terminator, which also sets VTE_VERSION", Env{"TERMINATOR_UUID=urn:uuid:x", "VTE_VERSION=7600", "TERM=xterm-256color"}, BrandTerminator},
		{"VTE", Env{"VTE_VERSION=7600", "TERM=xterm-256color"}, BrandVTE},
		{"Konsole", Env{"KONSOLE_VERSION=240202", "TERM=xterm-256color"}, BrandKonsole},
		{"Windows Terminal, last", Env{"WT_SESSION=x", "TERM_PROGRAM=WezTerm"}, BrandWezTerm},
		{"Windows Terminal", Env{"WT_SESSION=x"}, BrandWindowsTerminal},
		{"nothing", Env{"TERM=xterm-256color"}, BrandUnknown},
	}
	for _, c := range cases {
		if got := FromEnv(c.env, "linux"); got.Brand != c.want || got.EnvBrand != c.want {
			t.Errorf("%s: Brand %v, EnvBrand %v; want %v", c.name, got.Brand, got.EnvBrand, c.want)
		}
	}
}

func TestFromEnvRefinesOnWindowsOnly(t *testing.T) {
	win := FromEnv(Env{}, "windows")
	if win.Brand != BrandWindowsTerminal || win.EnvBrand != BrandUnknown {
		t.Errorf("Windows, no variables: Brand %v, EnvBrand %v; want windows-terminal, unknown", win.Brand, win.EnvBrand)
	}
	if lin := FromEnv(Env{}, "linux"); lin.Brand != BrandUnknown {
		t.Errorf("Linux, no variables: Brand %v, want unknown", lin.Brand)
	}
	// Over SSH the terminal is the client's: no guess (A5).
	if ssh := FromEnv(Env{"SSH_CONNECTION=a 1 b 22"}, "windows"); ssh.Brand != BrandUnknown || !ssh.Remote {
		t.Errorf("Windows over SSH: Brand %v, Remote %v; want unknown, remote", ssh.Brand, ssh.Remote)
	}
	var c Caps
	c.setEnv(Env{}, "windows", "")
	if c.Brand != (Fact[Brand]{Value: BrandWindowsTerminal, Origin: Heuristic, Reason: ReasonWindowsTerminalGuess}) {
		t.Errorf("refined Brand fact %+v", c.Brand)
	}
	if c.EnvBrand != (Fact[Brand]{Value: BrandUnknown, Origin: Environment}) {
		t.Errorf("EnvBrand fact %+v, want the raw unknown", c.EnvBrand)
	}
}

func TestFromEnvVersionNeedsItsBrand(t *testing.T) {
	if v := FromEnv(Env{"TERM_PROGRAM=WezTerm", "TERM_PROGRAM_VERSION=20240203"}, "linux").Version; v != "20240203" {
		t.Errorf("WezTerm's own version: %q", v)
	}
	if v := FromEnv(Env{"TERM_PROGRAM=tmux", "TERM_PROGRAM_VERSION=3.4", "TERM=xterm-kitty"}, "linux").Version; v != "" {
		t.Errorf("tmux's version kept for kitty: %q", v)
	}
	if v := FromEnv(Env{"VTE_VERSION=7600"}, "linux").Version; v != "7600" {
		t.Errorf("VTE's version: %q", v)
	}
}

func TestFromEnvOtherFacts(t *testing.T) {
	id := FromEnv(Env{"NVIM=/tmp/nvim.sock", "TMUX=x", "SSH_TTY=/dev/pts/1", "MSYSTEM=MINGW64", "TERM=xterm", "TERM_FEATURES=T2:Sc"}, "linux")
	if id.Editor != EditorNeovim || id.Mux != Tmux || !id.Remote || id.Platform != PlatformMSYS || id.Term != "xterm" || id.TermFeatures != "T2:Sc" {
		t.Errorf("identity %+v", id)
	}
	for env, want := range map[string]Editor{"VIM_TERMINAL=900": EditorVim, "INSIDE_EMACS=29.1,vterm": EditorEmacs} {
		if got := FromEnv(Env{env}, "linux").Editor; got != want {
			t.Errorf("%s: editor %v, want %v", env, got, want)
		}
	}
	if got := FromEnv(Env{"WSL_DISTRO_NAME=Ubuntu"}, "linux").Platform; got != PlatformWSL {
		t.Errorf("WSL: platform %v", got)
	}
}

func TestLegacyConsole(t *testing.T) {
	cases := []struct {
		env  Env
		goos string
		want Fact[bool]
	}{
		{Env{}, "windows", Fact[bool]{Value: true, Origin: Heuristic, Reason: ReasonLegacyConsoleGuess}},
		{Env{"WT_SESSION=x"}, "windows", Fact[bool]{Value: false, Origin: Heuristic}},
		{Env{"WEZTERM_PANE=0"}, "windows", Fact[bool]{Value: false, Origin: Heuristic}},
		{Env{}, "linux", Fact[bool]{Value: false, Origin: Environment}},
		{Env{"SSH_TTY=/dev/pty0"}, "windows", Fact[bool]{Value: false, Origin: Heuristic, Reason: ReasonConPTYAnswers}},
		{Env{"SSH_CONNECTION=a 1 b 22", "WT_SESSION=x"}, "windows", Fact[bool]{Value: false, Origin: Heuristic, Reason: ReasonConPTYAnswers}},
	}
	for _, c := range cases {
		var caps Caps
		caps.setEnv(c.env, c.goos, "")
		if caps.LegacyConsole != c.want {
			t.Errorf("%v on %s: %+v, want %+v", c.env, c.goos, caps.LegacyConsole, c.want)
		}
	}
}

// keyboard is the keyboard view of a terminal with the given brand,
// platform and multiplexer, whose Kitty support is k.
func keyboard(brand Brand, platform Platform, mux Mux, k Support, tmux TmuxFacts) KeyboardCaps {
	var c Caps
	c.Brand.Set(brand, Environment)
	c.EnvBrand.Set(brand, Environment)
	c.Platform.Set(platform, Environment)
	c.Mux.Set(mux, Environment)
	c.KittyKeyboard.Set(k, Queried)
	c.Tmux = tmux
	return c.Keyboard()
}

func TestKeyboardRequest(t *testing.T) {
	csiU := TmuxFacts{Known: true, Version: "3.4", ExtendedKeysFormat: "csi-u"}
	cases := []struct {
		name     string
		k        KeyboardCaps
		releases bool
		reason   string
	}{
		{"kitty", keyboard(BrandKitty, PlatformNative, NoMux, Supported, TmuxFacts{}), true, ""},
		{"WezTerm", keyboard(BrandWezTerm, PlatformNative, NoMux, Supported, TmuxFacts{}), true, ""},
		{"iTerm2", keyboard(BrandITerm2, PlatformNative, NoMux, Supported, TmuxFacts{}), false, ReasonLeaksReleases},
		{"Ghostty", keyboard(BrandGhostty, PlatformNative, NoMux, Supported, TmuxFacts{}), false, ReasonLeaksReleases},
		{"Alacritty", keyboard(BrandAlacritty, PlatformNative, NoMux, Supported, TmuxFacts{}), false, ReasonAlacrittyRelease},
		{"tmux, csi-u", keyboard(BrandKitty, PlatformNative, Tmux, Supported, csiU), true, ""},
		{"tmux, xterm format", keyboard(BrandKitty, PlatformNative, Tmux, Supported, TmuxFacts{Known: true, ExtendedKeysFormat: "xterm"}), false, ReasonTmuxExtendedKeys},
		{"tmux, not asked", keyboard(BrandKitty, PlatformNative, Tmux, Supported, TmuxFacts{}), false, ReasonTmuxExtendedKeys},
		{"mintty", keyboard(BrandMintty, PlatformNative, NoMux, Supported, TmuxFacts{}), false, ReasonMSYSNoKitty},
		{"MSYS2", keyboard(BrandWindowsTerminal, PlatformMSYS, NoMux, Supported, TmuxFacts{}), false, ReasonMSYSNoKitty},
		{"WSL in VS Code", keyboard(BrandVSCode, PlatformWSL, NoMux, Supported, TmuxFacts{}), false, ReasonWSLDeadKeys},
		{"WSL, terminal unknown", keyboard(BrandUnknown, PlatformWSL, NoMux, Supported, TmuxFacts{}), false, ReasonWSLDeadKeys},
		{"WSL in WezTerm", keyboard(BrandWezTerm, PlatformWSL, NoMux, Supported, TmuxFacts{}), true, ""},
		{"no Kitty protocol", keyboard(BrandKitty, PlatformNative, NoMux, Unsupported, TmuxFacts{}), false, ReasonKittyUnsupported},
	}
	for _, c := range cases {
		want := 0
		if c.releases {
			want = ansi.KittyReportEventTypes
		}
		if c.k.Request != want || c.k.Kitty.Reason != c.reason {
			t.Errorf("%s: Request %d, reason %q; want %d, %q", c.name, c.k.Request, c.k.Kitty.Reason, want, c.reason)
		}
	}
}

func TestReleasesReported(t *testing.T) {
	for flags, want := range map[int]bool{0: false, 1: false, 3: true, ansi.KittyReportEventTypes: true} {
		if got := (Caps{KeyboardFlags: flags}).ReleasesReported(); got != want {
			t.Errorf("flags %d: ReleasesReported %v, want %v", flags, got, want)
		}
	}
}

func TestColorFGBG(t *testing.T) {
	cases := []struct {
		v        string
		dark, ok bool
	}{
		{"0;15", false, true},
		{"15;0", true, true},
		{"15;default;0", true, true},
		{"7;8", true, true},
		{"0;7", false, true},
		{"default;default", false, false},
		{"15;16", false, false},
		{"garbage", false, false},
		{"", false, false},
	}
	for _, c := range cases {
		if dark, ok := colorFGBG(c.v); dark != c.dark || ok != c.ok {
			t.Errorf("COLORFGBG=%q: %v, %v; want %v, %v", c.v, dark, ok, c.dark, c.ok)
		}
	}
}

func TestParseTmux(t *testing.T) {
	got := ParseTmux("3.4\tcsi-u\t1\tRGB,clipboard,extkeys\tfocus,control-mode\n")
	want := TmuxFacts{Known: true, Version: "3.4", ExtendedKeysFormat: "csi-u", Mouse: true,
		TermFeatures: []string{"RGB", "clipboard", "extkeys"}, ClientFlags: []string{"focus", "control-mode"}}
	if !tmuxEqual(got, want) {
		t.Errorf("real-shaped output: %+v", got)
	}
	if got := ParseTmux(""); got.Known || got.Version != "" {
		t.Errorf("empty output: %+v", got)
	}
	if got := ParseTmux("3.3a\txterm\t0"); got.Known || got.Version != "3.3a" || got.ExtendedKeysFormat != "xterm" || got.Mouse {
		t.Errorf("missing fields: %+v", got)
	}
	if q := TmuxQuery(); q[0] != "tmux" || q[1] != "display-message" || q[2] != "-p" || strings.Count(q[3], "\t") != 4 {
		t.Errorf("TmuxQuery = %q", q)
	}
	p := NewProbe()
	p.Start(Env{"TMUX=x", "TERM=xterm-kitty"}, time.Time{})
	p.SetTmux(want)
	if !tmuxEqual(p.Caps().Tmux, want) {
		t.Errorf("SetTmux: %+v", p.Caps().Tmux)
	}
}

func tmuxEqual(a, b TmuxFacts) bool {
	return a.Known == b.Known && a.Version == b.Version && a.ExtendedKeysFormat == b.ExtendedKeysFormat &&
		a.Mouse == b.Mouse && slices.Equal(a.TermFeatures, b.TermFeatures) && slices.Equal(a.ClientFlags, b.ClientFlags)
}

func TestAtLeast(t *testing.T) {
	for v, want := range map[string]bool{"3.4": true, "3.5a": true, "4.0": true, "3.3a": false, "3.3": false, "2.9": false, "next-3.5": false, "": false} {
		if got := atLeast(v, 3, 4); got != want {
			t.Errorf("atLeast(%q, 3.4) = %v", v, got)
		}
	}
}

func TestLinks(t *testing.T) {
	brand := func(b Brand, mux Mux, version string) LinkCaps {
		var c Caps
		c.Brand.Set(b, Environment)
		c.Mux.Set(mux, Environment)
		c.Tmux.Version = version
		return c.Links()
	}
	cases := []struct {
		name   string
		l      LinkCaps
		want   Support
		reason string
	}{
		{"kitty", brand(BrandKitty, NoMux, ""), Supported, ""},
		{"Windows Terminal", brand(BrandWindowsTerminal, NoMux, ""), Supported, ""},
		{"Apple Terminal", brand(BrandAppleTerminal, NoMux, ""), Unsupported, ReasonAppleNoOSC8},
		{"Warp", brand(BrandWarp, NoMux, ""), Unsupported, ReasonWarpNoOSC8},
		{"unknown", brand(BrandUnknown, NoMux, ""), Unknown, ReasonUnknownTerminal},
		{"tmux 3.3", brand(BrandKitty, Tmux, "3.3"), Unsupported, ReasonTmuxLinks},
		{"tmux 3.4", brand(BrandKitty, Tmux, "3.4"), Supported, ""},
		{"tmux, version unknown", brand(BrandKitty, Tmux, ""), Unsupported, ReasonTmuxLinks},
	}
	for _, c := range cases {
		if c.l.OSC8.Value != c.want || c.l.OSC8.Reason != c.reason {
			t.Errorf("%s: %+v, want %v %q", c.name, c.l.OSC8, c.want, c.reason)
		}
	}
	var c Caps
	c.Mux.Set(Tmux, Environment)
	c.Terminal.Set("tmux 3.4", Queried)
	c.Brand.Set(BrandKitty, Environment)
	if c.Links().OSC8.Value != Supported {
		t.Error("tmux's version from XTVERSION was not read")
	}
}

func TestNotifications(t *testing.T) {
	view := func(b Brand, mux Mux) NotifyCaps {
		var c Caps
		c.Brand.Set(b, Environment)
		c.Mux.Set(mux, Environment)
		c.DesktopNotify.Set(Supported, Queried)
		c.FocusEvents.Set(Supported, Queried)
		return c.Notifications()
	}
	cases := []struct {
		name         string
		n            NotifyCaps
		osc777, osc9 Support
	}{
		{"iTerm2", view(BrandITerm2, NoMux), Unsupported, Supported},
		{"WezTerm", view(BrandWezTerm, NoMux), Unsupported, Supported},
		{"Ghostty", view(BrandGhostty, NoMux), Supported, Unsupported},
		{"foot", view(BrandFoot, NoMux), Supported, Unsupported},
		{"VTE", view(BrandVTE, NoMux), Supported, Unsupported},
		{"kitty", view(BrandKitty, NoMux), Unsupported, Unsupported},
		{"unknown", view(BrandUnknown, NoMux), Unknown, Unknown},
		{"Zellij", view(BrandITerm2, Zellij), Unsupported, Unsupported},
	}
	for _, c := range cases {
		if c.n.OSC777.Value != c.osc777 || c.n.OSC9.Value != c.osc9 {
			t.Errorf("%s: OSC 777 %v, OSC 9 %v; want %v, %v", c.name, c.n.OSC777.Value, c.n.OSC9.Value, c.osc777, c.osc9)
		}
		if c.n.OSC99.Value != Supported || c.n.Focus.Value != Supported {
			t.Errorf("%s: OSC 99 or focus not carried over: %+v", c.name, c.n)
		}
	}
	if z := view(BrandITerm2, Zellij); z.OSC9.Reason != ReasonZellijNoForwarding {
		t.Errorf("Zellij reason %q", z.OSC9.Reason)
	}
}

// TestPaletteRefusals: an OSC 4 reply out of the palette's range, or
// whose index or colour does not read, sets no colour.
func TestPaletteRefusals(t *testing.T) {
	for _, bad := range []string{"\x1b]4;16;rgb:0/0/0\x07", "\x1b]4;x;rgb:0/0/0\x07", "\x1b]4;1;hsl:1/2/3\x07", "\x1b]4;1;rgb:12345/0/0\x07"} {
		var caps Caps
		if parsePalette(ParseReply(bad), &caps) {
			t.Errorf("parsed %q", bad)
		}
	}
}

// TestReasonTokens checks that every Reason constant is in reasons, and
// every token is unique and dotted.
func TestReasonTokens(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "reason.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var consts []string
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, s := range g.Specs {
			for _, n := range s.(*ast.ValueSpec).Names {
				if strings.HasPrefix(n.Name, "Reason") {
					consts = append(consts, n.Name)
				}
			}
		}
	}
	if len(consts) != len(reasons) {
		t.Errorf("%d Reason constants, %d in reasons", len(consts), len(reasons))
	}
	seen := map[string]bool{}
	for _, r := range reasons {
		if seen[r] {
			t.Errorf("token %q is used twice", r)
		}
		seen[r] = true
		if !strings.Contains(r, ".") || strings.ContainsAny(r, " _") {
			t.Errorf("token %q is not dotted kebab case", r)
		}
	}
}

func TestReportShowsReasons(t *testing.T) {
	var c Caps
	c.setEnv(Env{}, "windows", "")
	out := reportOf(t, c, WithReportWidth(0))
	for _, want := range []string{
		"brand                  windows-terminal (heuristic: terminal.windows-terminal-guess)\n",
		"legacy_console         yes (heuristic: console.no-terminal-variables)\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
}
