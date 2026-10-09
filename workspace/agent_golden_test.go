package workspace_test

import (
	"testing"

	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/tuitest"
)

// TestAgentSessionGolden renders the agent session in the four arrangements
// the owner named, across the 0001 §6 matrix at 80, 120 and 200 columns. At
// 80 the sidebar folds under the session; at 120 and 200 it is beside it.
func TestAgentSessionGolden(t *testing.T) {
	for _, s := range []struct {
		name string
		left bool
		span layout.Span
	}{
		{"agent-right-bottom-full", false, layout.FullWidth},
		{"agent-left-bottom-full", true, layout.FullWidth},
		{"agent-right-bottom-main", false, layout.UnderMain},
		{"agent-left-bottom-main", true, layout.UnderMain},
	} {
		tuitest.Golden(t, s.name, tuitest.Matrix{Widths: []int{80, 120, 200}}, func(c tuitest.Case) string {
			th := theme.New(c.Profile(), theme.Unknown, c.Glyphs())
			return session(agentRoot(s.left, s.span), th, c.Width, 30).Render()
		})
	}
}
