package termcaptea

import (
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/internal/termevent/termeventtest"
	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// The Bubble Tea cases of termcap's identity tests, moved with the Prober
// (docs/decisions/0020-PLAN-charm-free-cores-glyphs-and-safetext.md Step 8).

func TestWithGOOS(t *testing.T) {
	win, _ := start(termcap.Env{"TERM=xterm-256color"}, termcap.WithGOOS("windows"))
	lin, _ := start(termcap.Env{"TERM=xterm-256color"}, termcap.WithGOOS("linux"))
	if b := win.Caps().Brand.Value; b != termcap.BrandWindowsTerminal {
		t.Errorf("WithGOOS(windows): brand %v, want windows-terminal", b)
	}
	if b := lin.Caps().Brand.Value; b != termcap.BrandUnknown {
		t.Errorf("WithGOOS(linux): brand %v, want unknown", b)
	}
	if !win.Caps().LegacyConsole.Value || lin.Caps().LegacyConsole.Value {
		t.Error("WithGOOS did not reach the legacy-console fact")
	}
}

func TestAppearanceChain(t *testing.T) {
	type step struct {
		name string
		env  termcap.Env
		hook *bool
		msg  tea.Msg
		want termcap.Fact[bool]
	}
	yes, no := true, false
	fgbgDark := termcap.Env{"TERM=xterm-kitty", "COLORFGBG=15;0"}
	steps := []step{
		{"COLORFGBG", fgbgDark, nil, nil, termcap.Fact[bool]{Value: true, Origin: termcap.Heuristic, Reason: termcap.ReasonColorFGBGGuess}},
		{"the desktop hook outranks COLORFGBG", fgbgDark, &no, nil, termcap.Fact[bool]{Value: false, Origin: termcap.Heuristic, Reason: termcap.ReasonDesktopAppearance}},
		{"the program's variable outranks both", append(termcap.Env{"APP_LOOK=dark"}, fgbgDark...), &no, nil, termcap.Fact[bool]{Value: true, Origin: termcap.Environment}},
		{"its LC_ form crosses SSH", append(termcap.Env{"LC_APP_LOOK=Light"}, fgbgDark...), &yes, nil, termcap.Fact[bool]{Value: false, Origin: termcap.Environment}},
		{"a reply outranks the variable", append(termcap.Env{"APP_LOOK=dark"}, fgbgDark...), nil, termeventtest.LightColorScheme(), termcap.Fact[bool]{Value: false, Origin: termcap.Queried}},
	}
	for _, s := range steps {
		o := []termcap.Option{termcap.WithAppearanceEnv("APP_LOOK")}
		if s.hook != nil {
			dark := *s.hook
			o = append(o, termcap.WithAppearanceHook(func() (bool, bool) { return dark, true }))
		}
		// The hook runs as a command; tea feeds its answer back.
		p, msgs := start(s.env, o...)
		feed(p, msgs...)
		if s.msg != nil {
			feed(p, s.msg)
		}
		if got := p.Caps().Dark; got != s.want {
			t.Errorf("%s: Dark %+v, want %+v", s.name, got, s.want)
		}
	}
	over, _ := start(termcap.Env{"APP_LOOK=dark"}, termcap.WithAppearanceEnv("APP_LOOK"),
		termcap.WithOverride(func(c *termcap.Caps) { c.Dark.Set(false, termcap.Override) }))
	feed(over, termeventtest.DarkColorScheme())
	if got := over.Caps().Dark; got != (termcap.Fact[bool]{Value: false, Origin: termcap.Override}) {
		t.Errorf("override: Dark %+v", got)
	}
}

func TestHooksAreNotCalledWhenNotSet(t *testing.T) {
	p, msgs := start(local)
	for _, m := range msgs {
		switch m.(type) {
		case hookMsg:
			t.Fatalf("a hook ran with none set: %T", m)
		}
	}
	_ = p
}

func TestConsoleHost(t *testing.T) {
	p := NewProber(termcap.WithGOOS("windows"), termcap.WithConsoleHost(func() (bool, bool) { return false, true }))
	feed(p, feed(p, tea.EnvMsg(termcap.Env{}))...)
	if got := p.Caps().LegacyConsole; got != (termcap.Fact[bool]{Value: false, Origin: termcap.Queried}) {
		t.Errorf("console host said ConPTY: %+v", got)
	}
	q := NewProber(termcap.WithGOOS("linux"), termcap.WithConsoleHost(func() (bool, bool) { t.Error("console host asked off Windows"); return true, true }))
	feed(q, feed(q, tea.EnvMsg(termcap.Env{}))...)
}

func TestPaletteAndForeground(t *testing.T) {
	p, _ := start(local)
	feed(p, tea.ForegroundColorMsg{Color: color.RGBA{R: 1, G: 2, B: 3, A: 0xff}})
	for i := range 15 {
		feed(p, termeventtest.UnknownOsc("\x1b]4;"+itoa(i)+";rgb:ffff/0000/8080\x1b\\"))
	}
	if c := p.Caps(); c.PaletteKnown || c.Palette[15] != nil {
		t.Fatalf("PaletteKnown after 15 of 16 replies: %v", c.PaletteKnown)
	}
	feed(p, termeventtest.UnknownOsc("\x1b]4;15;#102030\x07"))
	c := p.Caps()
	if !c.PaletteKnown {
		t.Fatal("PaletteKnown false after all 16 replies")
	}
	// termcap reads OSC 4 itself, at 16 bits, the precision the reply
	// carries (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md
	// A2.4); tea's foreground keeps tea's colour. The refusals are termcap's
	// TestPaletteRefusals.
	if c.Palette[0] != (color.RGBA64{R: 0xffff, G: 0, B: 0x8080, A: 0xffff}) || c.Palette[15] != (color.RGBA64{R: 0x1000, G: 0x2000, B: 0x3000, A: 0xffff}) {
		t.Errorf("palette %v %v", c.Palette[0], c.Palette[15])
	}
	if c.Foreground != (color.RGBA{R: 1, G: 2, B: 3, A: 0xff}) {
		t.Errorf("foreground %v", c.Foreground)
	}
}

func itoa(i int) string { return string(rune('0'+i/10)) + string(rune('0'+i%10)) }

func TestBrandFromXTVersion(t *testing.T) {
	p, _ := start(termcap.Env{"TERM=xterm-256color", "SSH_TTY=x"}, termcap.WithoutHeuristic())
	feed(p, tea.TerminalVersionMsg{Name: "WezTerm 20240203-110809-5046fc22"})
	c := p.Caps()
	if c.Brand != (termcap.Fact[termcap.Brand]{Value: termcap.BrandWezTerm, Origin: termcap.Queried}) || c.EnvBrand.Value != termcap.BrandUnknown {
		t.Errorf("Brand %+v, EnvBrand %+v", c.Brand, c.EnvBrand)
	}
}
