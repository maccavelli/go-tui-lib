package launch

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestRestorerFunc: a RestorerFunc is a Restorer that calls its function,
// and Run writes what it returns, as it writes any Restorer's
// (docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md Step 5).
func TestRestorerFunc(t *testing.T) {
	calls := 0
	f := RestorerFunc(func() string { calls++; return "<FUNC>" })
	if got := f.Restore(); got != "<FUNC>" || calls != 1 {
		t.Fatalf("Restore = %q after %d calls; want <FUNC> after 1", got, calls)
	}
	countSaves(t)
	old := newProgram
	// A program that fails before Init: Bubble Tea refuses a nil model.
	newProgram = func(_ tea.Model, o ...tea.ProgramOption) *tea.Program { return old(nil, o...) }
	t.Cleanup(func() { newProgram = old })
	out := terminal(80, 24)
	runTM(t, t.Context(), Streams{In: blocking(t), Out: out}, interactive(), tm{keys: "given"}, WithRestorer(f))
	if !strings.HasSuffix(out.String(), "<FUNC>") || calls != 2 {
		t.Errorf("Run wrote %q after %d calls; want <FUNC> last, after 2", out.String(), calls)
	}
}
