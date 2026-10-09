package termcap_test

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/termcap"
	"github.com/maccavelli/go-tui-lib/termcap/termcaptest"
)

// EnvCaps (docs/decisions/0014-PLAN-component-native-forms.md Step 8).

func TestEnvCapsMatchesDisabledProber(t *testing.T) {
	override := termcap.WithOverride(func(c *termcap.Caps) { c.Terminal.Set("forced", termcap.Override) })
	optionSets := map[string][]termcap.Option{
		"no options":              nil,
		"appearance and override": {termcap.WithAppearanceEnv("APP_THEME"), override},
		"probe-only options":      {termcap.WithoutHeuristic(), termcap.WithoutColorSchemeUpdates(), termcap.WithGOOS("plan9")},
	}
	profiles := []termcaptest.Profile{
		termcaptest.Kitty(), termcaptest.XTerm(), termcaptest.Tmux(true), termcaptest.DA1Only(),
		termcaptest.AppleTerminalSSH(), termcaptest.JetBrains(), termcaptest.Silent(),
	}
	for _, pr := range profiles {
		env := termcap.Env(append(append([]string{}, pr.Env...), "LC_APP_THEME=dark"))
		for _, goos := range []string{"linux", "darwin", "windows"} {
			for name, opts := range optionSets {
				p := termcap.New(append(append([]termcap.Option{}, opts...), termcap.WithDisabled(), termcap.WithGOOS(goos))...)
				p.Update(tea.EnvMsg(env))
				want := p.Caps()
				got := termcap.EnvCaps(env, goos, opts...)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("%s, %s, %s:\n got %+v\nwant %+v", pr.Name, goos, name, got, want)
				}
				if got.Profile != 0 {
					t.Errorf("%s: Profile = %v, want unknown", pr.Name, got.Profile)
				}
			}
		}
	}

	// The facts themselves, not only the match.
	c := termcap.EnvCaps(termcap.Env(termcaptest.JetBrains().Env), "linux")
	for name, f := range map[string]termcap.Fact[termcap.Support]{
		"KittyKeyboard": c.KittyKeyboard, "ColorSchemeReports": c.ColorSchemeReports,
		"InBandResize": c.InBandResize, "FocusEvents": c.FocusEvents, "DesktopNotify": c.DesktopNotify,
		"KittyGraphics": c.KittyGraphics, "Sixel": c.Sixel,
	} {
		if f != (termcap.Fact[termcap.Support]{Value: termcap.Unknown, Origin: termcap.NotQueried, Reason: termcap.ReasonJetBrainsPaints}) {
			t.Errorf("JetBrains %s = %+v; want unknown, not queried, ReasonJetBrainsPaints", name, f)
		}
	}
	if c.EnvBrand.Value != termcap.BrandJetBrains {
		t.Errorf("JetBrains EnvBrand = %v", c.EnvBrand)
	}
	o := termcap.EnvCaps(termcap.Env{"TERM=xterm", "LC_APP_THEME=dark"}, "linux", termcap.WithAppearanceEnv("APP_THEME"), override)
	if !o.Dark.Value || o.Dark.Origin != termcap.Environment || o.Terminal.Value != "forced" {
		t.Errorf("the appearance variable or the override did not apply: %+v, %+v", o.Dark, o.Terminal)
	}
	if k := termcap.EnvCaps(termcap.Env(termcaptest.Kitty().Env), "linux"); k.KittyKeyboard != (termcap.Fact[termcap.Support]{}) {
		t.Errorf("Kitty, unprobed: KittyKeyboard = %+v; want no fact", k.KittyKeyboard)
	}
}
