package termcaptea_test

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
	"github.com/maccavelli/go-tui-lib/v2/termcap/termcaptea"
	"github.com/maccavelli/go-tui-lib/v2/termcap/termcaptest"
)

// EnvCaps is what a disabled Prober holds after the first tea.EnvMsg
// (docs/decisions/0014-PLAN-component-native-forms.md Step 8). The facts
// themselves are termcap's TestEnvCapsMatchesDisabledProbe.

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
				p := termcaptea.NewProber(append(append([]termcap.Option{}, opts...), termcap.WithoutQueries(), termcap.WithGOOS(goos))...)
				p.Update(tea.EnvMsg(env))
				want := p.Caps()
				got := termcap.EnvCaps(env, goos, opts...)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("%s, %s, %s:\n got %+v\nwant %+v", pr.Name, goos, name, got, want)
				}
			}
		}
	}
}
