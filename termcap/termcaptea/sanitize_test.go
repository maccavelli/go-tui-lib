package termcaptea

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// The terminal's name, from tea's XTVERSION message, passes through
// sanitize.Line (docs/decisions/0014-PLAN-hardening.md Step 2). The
// environment's cases stay in termcap.

func TestXTVersionSanitized(t *testing.T) {
	p, _ := start(termcap.Env{"TERM=xterm-256color"}, termcap.WithoutHeuristic())
	feed(p, tea.TerminalVersionMsg{Name: "evil\x1b[2J\u202eterm\n1.0\u200b"})
	if got := p.Caps().Terminal; got != (termcap.Fact[string]{Value: "evilterm 1.0", Origin: termcap.Queried}) {
		t.Errorf("Terminal = %+v", got)
	}
}
