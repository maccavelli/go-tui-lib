package termcap

import (
	"runtime"
	"testing"
)

// Env's lookups on Windows and elsewhere
// (docs/decisions/0014-PLAN-component-native-forms.md Step 8).

var lookupEnv = Env{"Path=C:\\bin", "TERM=xterm", "term=dumb-later", "NO_COLOR=", "EMPTY", "=C:=C:\\x"}

func TestLookupFoldsOnWindows(t *testing.T) {
	for _, tc := range []struct {
		key, want string
		ok        bool
	}{
		{"PATH", "C:\\bin", true},
		{"path", "C:\\bin", true},
		{"Term", "dumb-later", true}, // the last of either case wins
		{"no_color", "", true},
		{"EMPTY", "", false},
		{"MISSING", "", false},
		{"PAT", "", false},
	} {
		if v, ok := lookupEnv.lookup(tc.key, "windows"); v != tc.want || ok != tc.ok {
			t.Errorf("windows %q = %q, %v; want %q, %v", tc.key, v, ok, tc.want, tc.ok)
		}
	}
}

func TestLookupExactElsewhere(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "freebsd"} {
		for _, tc := range []struct {
			key, want string
			ok        bool
		}{
			{"Path", "C:\\bin", true},
			{"PATH", "", false},
			{"TERM", "xterm", true},
			{"term", "dumb-later", true},
			{"NO_COLOR", "", true},
			{"no_color", "", false},
		} {
			if v, ok := lookupEnv.lookup(tc.key, goos); v != tc.want || ok != tc.ok {
				t.Errorf("%s %q = %q, %v; want %q, %v", goos, tc.key, v, ok, tc.want, tc.ok)
			}
		}
	}
	// LookupEnv and Getenv follow the running GOOS.
	want := "C:\\bin"
	if runtime.GOOS != "windows" {
		want = ""
	}
	if got := lookupEnv.Getenv("PATH"); got != want {
		t.Errorf("Getenv(PATH) on %s = %q, want %q", runtime.GOOS, got, want)
	}
}
