package launch_test

import (
	"testing"

	"github.com/maccavelli/go-tui-lib/launch"
	"github.com/maccavelli/go-tui-lib/launch/launchtest"
)

// Decide clamps its size as workspace clamps a window
// (docs/decisions/0014-PLAN-hardening.md Step 4, finding H3).
func TestDecideClampsSize(t *testing.T) {
	huge := launchtest.NewTerminal(1<<30, 1<<30)
	d := launch.Decide(launchtest.Streams(huge, huge, huge), launch.Config{})
	if !d.Interactive || d.Width != 4096 || d.Height != 128 {
		t.Errorf("a 2^30 × 2^30 terminal: interactive %v, %d × %d; want 4096 × 128", d.Interactive, d.Width, d.Height)
	}
	pipe := launchtest.NewPipe("")
	d = launch.Decide(launchtest.Streams(pipe, pipe, pipe, "COLUMNS=99999999"), launch.Config{})
	if d.Interactive || d.Width != 4096 || d.Height != 0 {
		t.Errorf("COLUMNS=99999999: interactive %v, %d × %d; want 4096 × 0", d.Interactive, d.Width, d.Height)
	}
	small := launchtest.NewTerminal(120, 40)
	if d := launch.Decide(launchtest.Streams(small, small, small), launch.Config{}); d.Width != 120 || d.Height != 40 {
		t.Errorf("a 120 × 40 terminal: %d × %d", d.Width, d.Height)
	}
}
