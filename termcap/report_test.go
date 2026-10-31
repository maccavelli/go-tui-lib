package termcap

import (
	"errors"
	"reflect"
	"slices"
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
// checked without a change here. The view facts are its one exception:
// each is written only once set, and full() sets them all
// (docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md
// Step 2, D7).
func TestReportNamesEveryField(t *testing.T) {
	views := []string{"LinksSupport", "NotifyOSC9Support", "NotifyOSC777Support", "KeyboardRequest", "PointerSupport", "ProgressSupport"}
	ct := reflect.TypeFor[Caps]()
	for _, tc := range []struct {
		name  string
		c     Caps
		views bool
	}{{"zero", Caps{}, false}, {"full", full(), true}} {
		out := reportOf(t, tc.c, WithReportWidth(0))
		facts, _, _ := strings.Cut(out, "\n\nfindings\n") // the findings follow the facts
		lines := strings.Split(strings.TrimSuffix(facts, "\n"), "\n")
		var names []string
		for f := range ct.Fields() {
			if tc.views || !slices.Contains(views, f.Name) {
				names = append(names, fieldName(f))
			}
		}
		if len(lines) != len(names) {
			t.Fatalf("%s: %d lines for %d fields:\n%s", tc.name, len(lines), len(names), out)
		}
		for i, name := range names {
			if !strings.HasPrefix(lines[i], name+" ") {
				t.Errorf("%s: line %d is %q, want field %s", tc.name, i, lines[i], name)
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
		"profile                ansi256\n",
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

// TestReportKeepsTokensWhole is PLAN D12: the real-terminal check's run 3
// showed "terminal.windows-terminal-" and "guess" on two lines.
func TestReportKeepsTokensWhole(t *testing.T) {
	var c Caps
	c.Brand.SetReason(BrandWindowsTerminal, Heuristic, ReasonWindowsTerminalGuess)
	for _, w := range []int{60, 64, 70, 80} {
		out := reportOf(t, c, WithReportWidth(w))
		// The facts alone: the findings section names the token too.
		facts, _, _ := strings.Cut(out, "\n\nfindings\n")
		if !strings.Contains(facts, ReasonWindowsTerminalGuess) {
			t.Errorf("width %d: the token is broken in the facts:\n%s", w, facts)
		}
		for l := range strings.SplitSeq(strings.TrimSuffix(out, "\n"), "\n") {
			if len(l) > w || strings.HasSuffix(l, "-") && !strings.HasSuffix(l, " -") { // " -" is an empty value
				t.Errorf("width %d: a line of %d, broken at a hyphen or too long: %q", w, len(l), l)
			}
		}
	}
}

func TestWrapWords(t *testing.T) {
	cases := []struct {
		s     string
		limit int
		want  []string
	}{
		{"a bb ccc", 4, []string{"a bb", "ccc"}},
		{"#aa;#bb;#cc", 8, []string{"#aa;#bb;", "#cc"}},
		{"one terminal.windows-terminal-guess two", 33, []string{"one", "terminal.windows-terminal-guess", "two"}},
		{"abcdefghij", 4, []string{"abcd", "efgh", "ij"}},
		{"x  y", 1, []string{"x", "y"}},
		{"", 5, []string{""}},
	}
	for _, c := range cases {
		if got := wrapWords(c.s, c.limit); strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("wrapWords(%q, %d) = %q, want %q", c.s, c.limit, got, c.want)
		}
	}
}
