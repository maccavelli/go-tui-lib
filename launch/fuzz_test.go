package launch

import (
	"math"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/internal/limits"
)

// FuzzDecideEnv: Decide, over a fuzzed environment and configuration with
// streams that are not terminals, never panics; it stays plain unless it
// may open the controlling terminal, and then reads and draws on it; and
// its size is within the limits
// (docs/decisions/0014-PLAN-hardening.md Step 9, finding H9). The
// environment is env's lines; flags picks the choice, the options, the
// operating system and whether the terminal opens.
func FuzzDecideEnv(f *testing.F) {
	for _, c := range []struct {
		env    string
		flags  byte
		tw, th int
	}{
		{strings.Join(base, "\n"), 0, 80, 24},
		{strings.Join(base, "\n"), 16, 100, 50},
		{strings.Join(with("CI=true"), "\n"), 16, 100, 50},
		{strings.Join(with("PROG_NO_TUI=1"), "\n"), 16 | 32, 100, 50},
		{"TERM=dumb", 1 | 16, 100, 50},
		{strings.Join(with("COLUMNS=99999999999"), "\n"), 2, 0, 0},
		{strings.Join(base, "\n"), 1 | 16, math.MaxInt, math.MaxInt}, // fits either int size (D10)
		{strings.Join(base, "\n"), 1 | 16, -5, -7},
		{strings.Join(base, "\n"), 1 | 16 | 64, 80, 24},
		{strings.Join(base, "\n"), 1 | 8 | 16, 80, 24},
		{"=\n\x00\nTERM", 255, 3, 3},
	} {
		f.Add(c.env, c.flags, c.tw, c.th)
	}
	f.Fuzz(func(t *testing.T, env string, flags byte, tw, th int) {
		cfg := Config{
			Choice:  Choice(flags % 4),
			UIOnErr: flags&4 != 0,
			OpenTTY: flags&16 != 0,
		}
		if flags&32 != 0 {
			cfg.NoInputEnv = "PROG_NO_TUI"
		}
		goos := "linux"
		if flags&64 != 0 {
			goos = "windows"
		}
		tty := terminal(tw, th)
		o := &ttyOpener{in: tty, out: tty, fail: flags&8 != 0}
		d := decideWith(streams(pipe(), pipe(), pipe(), strings.Split(env, "\n")...), cfg, goos, o.open)

		mayOpen := cfg.OpenTTY && goos != "windows"
		if !mayOpen && (d.Interactive || o.opens != 0) {
			t.Fatalf("%+v on %s: interactive %v, %d opens, without the controlling terminal", cfg, goos, d.Interactive, o.opens)
		}
		if d.Interactive && (d.In != TargetTTY || d.UI != TargetTTY) {
			t.Fatalf("%+v: interactive, reading %v and drawing on %v, not the controlling terminal", cfg, d.In, d.UI)
		}
		if !d.Interactive && (d.In != TargetNone || d.UI != TargetNone) {
			t.Fatalf("%+v: plain, reading %v and drawing on %v", cfg, d.In, d.UI)
		}
		if o.opens > 1 {
			t.Fatalf("%+v: the controlling terminal opened %d times", cfg, o.opens)
		}
		if d.Width < 0 || d.Height < 0 || d.Width > limits.MaxSide || d.Width*d.Height > limits.MaxCells {
			t.Fatalf("%+v, terminal %d x %d: size %d x %d", cfg, tw, th, d.Width, d.Height)
		}
	})
}
