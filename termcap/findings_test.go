package termcap

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestEveryReasonHasAFinding walks the reason tokens: each has a finding's
// text, and the text names no token that is not one.
func TestEveryReasonHasAFinding(t *testing.T) {
	for _, r := range reasons {
		f, ok := findingText[r]
		if !ok {
			t.Errorf("token %q has no finding", r)
			continue
		}
		if f.message == "" || f.fix == "" {
			t.Errorf("token %q: empty message or fix", r)
		}
	}
	for token := range findingText {
		if !strings.Contains(strings.Join(reasons, " "), token) {
			t.Errorf("a finding for %q, which is not a token", token)
		}
	}
}

func TestFindings(t *testing.T) {
	var c Caps
	c.setEnv(Env{"TERM=xterm-256color", "TMUX=x", "COLORFGBG=15;0"}, "linux", "")
	c.KittyKeyboard.Set(Supported, Queried)
	c.Brand.Set(BrandKitty, Queried)
	got := Findings(c)
	ids := make([]string, len(got))
	for i, f := range got {
		ids[i] = f.ID
	}
	want := []string{ReasonColorFGBGGuess, ReasonTmuxExtendedKeys, ReasonTmuxLinks}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("findings %v, want %v", ids, want)
	}
	if got[1].Disposition != Issue || !strings.Contains(got[1].Fix, "extended-keys-format csi-u") {
		t.Errorf("the tmux finding: %+v", got[1])
	}
	for _, f := range Findings(full()) {
		if strings.Count(strings.Join(ids, ","), f.ID) > 1 {
			t.Errorf("token %q found twice", f.ID)
		}
	}
}

// TestFindingsOverSSHIntoWindows is the real-terminal check's run 3: WezTerm
// on macOS, over SSH, into MSYS2 on Windows (PLAN Step 7, D11).
func TestFindingsOverSSHIntoWindows(t *testing.T) {
	var c Caps
	c.setEnv(Env{"TERM=xterm-256color", "MSYSTEM=UCRT64", "SSH_CONNECTION=a 1 b 22", "SSH_TTY=/dev/pty0"}, "windows", "")
	if c.Brand.Value != BrandUnknown || c.LegacyConsole.Value {
		t.Errorf("Brand %+v, LegacyConsole %+v; want no Windows guess over SSH", c.Brand, c.LegacyConsole)
	}
	var ids []string
	for _, f := range Findings(c) {
		ids = append(ids, f.ID)
		if f.ID == ReasonLegacyConsoleGuess || f.ID == ReasonWindowsTerminalGuess {
			t.Errorf("finding %q over SSH", f.ID)
		}
	}
	if !strings.Contains(strings.Join(ids, " "), ReasonConPTYAnswers) {
		t.Errorf("findings %v lack %q", ids, ReasonConPTYAnswers)
	}
}

func TestReportJSON(t *testing.T) {
	var c Caps
	c.setEnv(Env{}, "windows", "")
	var b strings.Builder
	if err := Report(&b, c, WithReportJSON(), WithReportWidth(20)); err != nil {
		t.Fatal(err)
	}
	var got struct {
		SchemaVersion int       `json:"schema_version"`
		Caps          Caps      `json:"caps"`
		Findings      []Finding `json:"findings"`
	}
	if err := json.Unmarshal([]byte(b.String()), &got); err != nil {
		t.Fatalf("%v\n%s", err, b.String())
	}
	if got.SchemaVersion != ReportSchemaVersion || got.Caps.LegacyConsole != c.LegacyConsole {
		t.Errorf("schema %d, caps %+v", got.SchemaVersion, got.Caps.LegacyConsole)
	}
	if len(got.Findings) == 0 || got.Findings[0].ID != ReasonWindowsTerminalGuess {
		t.Errorf("findings %+v", got.Findings)
	}
	if !strings.Contains(b.String(), `"disposition": "recommendation"`) {
		t.Errorf("dispositions not written by name:\n%s", b.String())
	}
	b.Reset()
	if err := Report(&b, full(), WithReportJSON()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `"findings": [`) {
		t.Errorf("findings missing:\n%s", b.String())
	}
}

func TestReportFindingsSection(t *testing.T) {
	var c Caps
	c.setEnv(Env{}, "windows", "")
	out := reportOf(t, c, WithReportWidth(0))
	if !strings.Contains(out, "\n\nfindings\nrecommendation         terminal.windows-terminal-guess: ") {
		t.Errorf("no findings section:\n%s", out)
	}
	var none Caps
	none.Brand.Set(BrandKitty, Queried)
	none.KittyKeyboard.Set(Supported, Queried)
	if out := reportOf(t, none, WithReportWidth(0)); strings.Contains(out, "findings") {
		t.Errorf("a findings section with nothing to say:\n%s", out)
	}
}
