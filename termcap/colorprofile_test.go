package termcap

import (
	json "encoding/json/v2"
	"strings"
	"testing"
)

// TestColorProfileTokens: D4's tokens, in colorprofile's order.
func TestColorProfileTokens(t *testing.T) {
	var got []string
	for p := ProfileUnknown; p <= ProfileTrueColor; p++ {
		got = append(got, p.String())
	}
	if s := strings.Join(got, ","); s != "unknown,notty,ascii,ansi,ansi256,truecolor" {
		t.Errorf("tokens %s", s)
	}
	if len(colorProfileSpellings) != int(ProfileTrueColor)+1 {
		t.Errorf("%d spellings for %d profiles", len(colorProfileSpellings), ProfileTrueColor+1)
	}
}

// TestCapsJSONReadsOldProfile: a report of schema version 1 wrote the
// profile in colorprofile's spelling; it still reads, and is written back
// by its token.
func TestCapsJSONReadsOldProfile(t *testing.T) {
	for i, old := range colorProfileSpellings {
		var c Caps
		if err := json.Unmarshal([]byte(`{"profile":"`+old+`"}`), &c); err != nil || c.Profile != ColorProfile(i) {
			t.Errorf("%q: %v, %v; want %v", old, c.Profile, err, ColorProfile(i))
		}
		b, err := json.Marshal(c)
		if want := `"profile":"` + ColorProfile(i).String() + `"`; i > 0 && (err != nil || !strings.Contains(string(b), want)) {
			t.Errorf("%q writes back as %s, %v; want %s", old, b, err, want)
		}
	}
	var c Caps
	if err := json.Unmarshal([]byte(`{"profile":"ansi256"}`), &c); err != nil || c.Profile != ProfileANSI256 {
		t.Errorf("the token: %v, %v", c.Profile, err)
	}
}

// TestReportSchemaVersion: version 2, in the JSON report.
func TestReportSchemaVersion(t *testing.T) {
	if ReportSchemaVersion != 2 {
		t.Fatalf("ReportSchemaVersion = %d, want 2", ReportSchemaVersion)
	}
	var b strings.Builder
	if err := Report(&b, Caps{Profile: ProfileTrueColor}, WithReportJSON()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `"schema_version": 2`) || !strings.Contains(b.String(), `"profile": "truecolor"`) {
		t.Errorf("report:\n%s", b.String())
	}
}
