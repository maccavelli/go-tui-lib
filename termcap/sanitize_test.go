package termcap

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The terminal's name, from its reply and from the environment, passes
// through sanitize.Line (docs/decisions/0014-PLAN-hardening.md Step 2).

func TestXTVersionSanitized(t *testing.T) {
	p, _ := start(Env{"TERM=xterm-256color"}, WithoutHeuristic())
	feed(p, tea.TerminalVersionMsg{Name: "evil\x1b[2J\u202eterm\n1.0\u200b"})
	if got := p.Caps().Terminal; got != (Fact[string]{Value: "evilterm 1.0", Origin: Queried}) {
		t.Errorf("Terminal = %+v", got)
	}

	env := Env{"TERM_PROGRAM=x\x1b]0;title\x07y\u202e", "TERM_PROGRAM_VERSION=1\u2066.2\x07", "TERM=xterm\u200b"}
	if got := EnvCaps(env, "linux").Terminal.Value; got != "xy" {
		t.Errorf("the environment's terminal = %q, want \"xy\"", got)
	}
	if id := FromEnv(env, "linux"); id.Term != "xterm" {
		t.Errorf("FromEnv Term = %q", id.Term)
	}
	ghostty := Env{"TERM_PROGRAM=ghostty", "TERM_PROGRAM_VERSION=1.2\x1b[31m\u202e"}
	if id := FromEnv(ghostty, "linux"); id.Version != "1.2" {
		t.Errorf("FromEnv Version = %q, want \"1.2\"", id.Version)
	}
}
