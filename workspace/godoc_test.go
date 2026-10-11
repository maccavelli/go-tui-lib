package workspace

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// removedNames are the names v0.12.0 removed
// (docs/decisions/0007-MADR-keymap-engine.md A4, item 2): a word of its
// own, not help.KeyMap, which bubbles defines, nor KeyMapper.
var removedNames = regexp.MustCompile(`(^|[^.\w])(KeyMap|DefaultKeyMap|WithKeyMap)\b`)

// TestDocNamesNoKeyMap: go doc ./workspace names no KeyMap, DefaultKeyMap
// or WithKeyMap.
func TestDocNamesNoKeyMap(t *testing.T) {
	gobin, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("go is not on the path: %v", err)
	}
	out, err := exec.Command(gobin, "doc", "-all", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go doc: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "func DefaultRules() []keymap.Default") {
		t.Fatalf("go doc printed no DefaultRules, so the check would read nothing:\n%s", out)
	}
	for i, line := range strings.Split(string(out), "\n") {
		if removedNames.MatchString(line) {
			t.Errorf("go doc, line %d, names a removed name: %q", i+1, line)
		}
	}
}
