package termcap

import (
	"testing"
	"time"
)

// TestDeprecatedOptionsStillWork: each old name does what its new name does,
// through v0.9.x (docs/decisions/0014-PLAN-canonicalization.md Step 4). It
// sits in the package, so that staticcheck does not report the deprecated
// names it uses on purpose.
func TestDeprecatedOptionsStillWork(t *testing.T) {
	old := New(WithTimeout(3*time.Second), WithGOOS("plan9"))
	if old.timeout != 3*time.Second || old.goos != "plan9" {
		t.Errorf("New with options: timeout %v, goos %q", old.timeout, old.goos)
	}
	if New().timeout != NewProber().timeout || New().goos != NewProber().goos {
		t.Error("New() and NewProber() differ")
	}
}
