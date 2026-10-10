package termcap

import (
	"encoding"
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/tuitest"
)

// TestCapsJSONStable pins the JSON report as golden files, so a change of
// encoder shows as a diff: full(), whose terminal name carries the
// characters JSON v1 escaped, and the zero Caps
// (docs/decisions/0014-PLAN-canonicalization.md Step 7).
func TestCapsJSONStable(t *testing.T) {
	c := full()
	c.Terminal.Value = "WezTerm <20240203> & co"
	for name, c := range map[string]Caps{"report-json-full": c, "report-json-zero": {}} {
		var b strings.Builder
		if err := Report(&b, c, WithReportJSON()); err != nil {
			t.Fatal(err)
		}
		tuitest.Text(t, name, b.String())
	}
}

// TestCapsJSONReadsStrictly pins what JSON v2 changed in reading Caps: a
// duplicate name and invalid UTF-8 are refused, and a name matches only in
// its own case (docs/decisions/0014-PLAN-canonicalization.md Step 7).
func TestCapsJSONReadsStrictly(t *testing.T) {
	for _, s := range []string{
		`{"complete":true,"complete":false}`,
		`{"terminal":{"value":"wez` + "\xff" + `term"}}`,
	} {
		var c Caps
		if err := c.UnmarshalJSON([]byte(s)); err == nil {
			t.Errorf("%q was read without an error: %+v", s, c)
		}
	}
	var c Caps
	if err := c.UnmarshalJSON([]byte(`{"Complete":true,"MUX":{"value":"tmux"}}`)); err != nil || c.Complete || c.Mux.Value != NoMux {
		t.Errorf("names in another case were read: %v, %+v", err, c)
	}
}

// TestErrUnknownName: every name termcap reads that it does not know is an
// error that wraps ErrUnknownName, with its text unchanged; another error
// does not (docs/decisions/0014-PLAN-canonicalization.md Step 7, D3).
func TestErrUnknownName(t *testing.T) {
	for name, u := range map[string]encoding.TextUnmarshaler{
		"Support": new(Support), "Origin": new(Origin), "Mux": new(Mux), "Disposition": new(Disposition),
		"Brand": new(Brand), "Editor": new(Editor), "Platform": new(Platform),
	} {
		err := u.UnmarshalText([]byte("bogus"))
		if want := `termcap: unknown ` + name + ` "bogus"`; !errors.Is(err, ErrUnknownName) || err.Error() != want {
			t.Errorf("%s: %v, want %q wrapping ErrUnknownName", name, err, want)
		}
	}
	for src, text := range map[string]string{
		`{"mux":{"value":"bogus"}}`: `termcap: unknown Mux "bogus"`,
		`{"profile":"Sixteen"}`:     `termcap: unknown colour profile "Sixteen"`,
	} {
		var c Caps
		err := c.UnmarshalJSON([]byte(src))
		if !errors.Is(err, ErrUnknownName) || !strings.Contains(err.Error(), text) {
			t.Errorf("%s: %v, want %q wrapping ErrUnknownName", src, err, text)
		}
	}
	var c Caps
	if err := c.UnmarshalJSON([]byte(`{"background":"red"}`)); err == nil || errors.Is(err, ErrUnknownName) {
		t.Errorf("a bad colour: %v, want an error that is not ErrUnknownName", err)
	}
}

// TestJSONWritesInvalidUTF8: a value with invalid UTF-8, which only a
// program's own values can hold, is written as U+FFFD rather than failing
// the report; and ParseTmux cleans its fields
// (docs/decisions/0014-PLAN-canonicalization.md Step 7, D3).
func TestJSONWritesInvalidUTF8(t *testing.T) {
	var c Caps
	c.Terminal.Set("wez\xffterm", Override)
	b, err := c.MarshalJSON()
	if err != nil || !strings.Contains(string(b), "wez\ufffdterm") {
		t.Errorf("MarshalJSON = %s, %v; want U+FFFD for the invalid byte", b, err)
	}
	var r strings.Builder
	if err := Report(&r, c, WithReportJSON()); err != nil || !strings.Contains(r.String(), "wez\ufffdterm") {
		t.Errorf("Report = %v:\n%s", err, r.String())
	}

	got := ParseTmux("3.4\xff\x1b[31m\tcsi-u\u202e\t1\tRGB,\x07clip\tfocus\n")
	want := TmuxFacts{Known: true, Version: "3.4", ExtendedKeysFormat: "csi-u", Mouse: true,
		TermFeatures: []string{"RGB", "clip"}, ClientFlags: []string{"focus"}}
	if !tmuxEqual(got, want) {
		t.Errorf("ParseTmux kept untrusted bytes: %+q", got)
	}
}
