package termsvc

import "testing"

// TestDeprecatedOptionsStillWork: each old name does what its new name does,
// through v0.9.x (docs/decisions/0014-PLAN-canonicalization.md Step 4). It
// sits in the package, so that staticcheck does not report the deprecated
// names it uses on purpose.
func TestDeprecatedOptionsStillWork(t *testing.T) {
	for _, want := range []bool{false, true} {
		n := NewNotifier(WithGate(func(Notification) bool { return want }))
		if n.gate == nil || n.gate(Notification{}) != want {
			t.Errorf("WithGate(always %v): the filter is not set", want)
		}
	}
}
