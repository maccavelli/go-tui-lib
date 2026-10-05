package termcap

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func reportOf(t *testing.T, c Caps, o ...ReportOption) string {
	t.Helper()
	var b strings.Builder
	if err := Report(&b, c, o...); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestReportNamesEveryField walks Caps itself, so a field added later is
// checked without a change here.
func TestReportNamesEveryField(t *testing.T) {
	for _, c := range []Caps{{}, full()} {
		out := reportOf(t, c, WithReportWidth(0))
		facts, _, _ := strings.Cut(out, "\n\nfindings\n") // the findings follow the facts
		lines := strings.Split(strings.TrimSuffix(facts, "\n"), "\n")
		ct := reflect.TypeFor[Caps]()
		if len(lines) != ct.NumField() {
			t.Fatalf("%d lines for %d fields:\n%s", len(lines), ct.NumField(), out)
		}
		for i := range ct.NumField() {
			name := fieldName(ct.Field(i))
			if !strings.HasPrefix(lines[i], name+" ") {
				t.Errorf("line %d is %q, want field %s", i, lines[i], ct.Field(i).Name)
			}
		}
	}
}

func TestReportIsASCII(t *testing.T) {
	c := full()
	c.Terminal.Set("evil\x1b]52;c;cGF5bG9hZA==\x07 終端", Override)
	out := reportOf(t, c, WithReportWidth(0))
	for i := range len(out) {
		if b := out[i]; b != '\n' && (b < 0x20 || b > 0x7e) {
			t.Fatalf("byte %#x at %d is not printable ASCII:\n%q", b, i, out)
		}
	}
	if !strings.Contains(out, "evil?]52;c;cGF5bG9hZA==? ??????") {
		t.Errorf("the terminal's name was not made printable:\n%s", out)
	}
}

func TestReportValuesAndOrigins(t *testing.T) {
	out := reportOf(t, full(), WithReportWidth(0))
	for _, want := range []string{
		"complete               yes\n",
		"terminal               WezTerm 20240203 (query)\n",
		"mux                    tmux (env)\n",
		"attributes             62;4;22\n",
		"desktop_notify         unsupported (override)\n",
		"kitty_graphics         unknown (heuristic)\n",
		"dark                   yes (query)\n",
		"background             #1e1f29\n",
		"profile                ANSI256\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
}

func TestReportSaysWhyAFactWasNotQueried(t *testing.T) {
	out := reportOf(t, Caps{}, WithReportWidth(0))
	for _, want := range []string{
		"terminal               unknown (not-queried)\n",
		"mux                    unknown (not-queried)\n",
		"dark                   unknown (not-queried)\n",
		"sync_output            unknown (not-queried: tea asks only outside SSH and Apple Terminal, and not every terminal answers)\n",
		"desktop_notify         unknown (not-queried: asked only behind the heuristic)\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
	timedOut := reportOf(t, Caps{TimedOut: true}, WithReportWidth(0))
	for _, want := range []string{
		"kitty_keyboard         unknown (not-queried: no reply before the timeout)\n",
		"desktop_notify         unknown (not-queried: asked only behind the heuristic; or no reply before the timeout)\n",
	} {
		if !strings.Contains(timedOut, want) {
			t.Errorf("timed-out report lacks %q:\n%s", want, timedOut)
		}
	}
}

func TestReportWrapsUnderTheValueColumn(t *testing.T) {
	out := reportOf(t, Caps{TimedOut: true}, WithReportWidth(60))
	for l := range strings.SplitSeq(strings.TrimSuffix(out, "\n"), "\n") {
		if len(l) > 60 {
			t.Errorf("line of %d cells: %q", len(l), l)
		}
	}
	if !strings.Contains(out, "asked only\n"+strings.Repeat(" ", nameWidth+1)+"behind the heuristic; or no reply\n") {
		t.Errorf("no continuation under the value column:\n%s", out)
	}
}

type failingWriter struct{}

var errWrite = errors.New("write failed")

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

func TestReportReturnsTheWriteError(t *testing.T) {
	if err := Report(failingWriter{}, Caps{}); !errors.Is(err, errWrite) {
		t.Fatalf("Report = %v, want the writer's error", err)
	}
}
