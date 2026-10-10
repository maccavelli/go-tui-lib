package teamsg

import "testing"

type msg struct{ n int }

// TestCmd: the command delivers the message it was given, each time it runs.
func TestCmd(t *testing.T) {
	c := Cmd(msg{7})
	for range 2 {
		if got := c(); got != (msg{7}) {
			t.Errorf("Cmd(msg{7})() = %#v", got)
		}
	}
}
