package lineedit

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// parityCase is one of gobble's parity cases and x/term v0.47.0's outcome,
// as lineedit/testdata/xtermparity writes them (0023-MADR A5 item 1).
type parityCase struct {
	Name string `json:"name"`
	In   string `json:"in"`
	Rows bool   `json:"rows"`
	Want struct {
		Lines []string `json:"lines"`
		EOF   bool     `json:"eof"`
		Text  string   `json:"text"`
		Pos   int      `json:"pos"`
	} `json:"want"`
}

// notParity is the cases this record decides otherwise, by name, and why.
var notParity = map[string]string{
	"a paste of two lines":        "D6: a paste is inserted as text, where x/term returns it a line at a time",
	"a paste then Enter":          "D6: as above",
	"a paste of three lines":      "D6: as above",
	"a paste into a typed line":   "D6: as above",
	"control keys inside a paste": "A3 item 11: a paste is cleaned, its controls but tab and line break removed",
	"Ctrl+C":                      "D6: ctrl+c is EditInterrupt, where x/term ends the input",
	"Alt+Right twice":             "A5 item 6: a forward word motion stops at the word's end, as Readline's",
	"Alt+Right from a space":      "A5 item 6: as above",
}

// TestKeysMatchXterm: every key of gobble's parity table does in the
// Composer what x/term's Terminal does with it, on one row at 200 columns
// and, but for the cases that move between rows, on rows of 12; the cases
// this record decides otherwise are left out by name.
func TestKeysMatchXterm(t *testing.T) {
	b, err := os.ReadFile("testdata/xterm-parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []parityCase
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 55 {
		t.Fatalf("%d cases; want gobble's 55", len(cases))
	}
	seen := map[string]bool{}
	compared := 0
	for _, c := range cases {
		seen[c.Name] = true
		if _, ok := notParity[c.Name]; ok {
			continue
		}
		compared++
		for _, columns := range []int{200, 12} {
			if columns == 12 && c.Rows {
				continue
			}
			h := newHost(t, columns, WithPrompt("> "), WithHistory(fixedHistory{"newest entry", "middle", "oldest one"}))
			h.typeIn(c.In)
			got := h.submitted()
			if !slices.Equal(got, c.Want.Lines) || h.eof() != c.Want.EOF {
				t.Errorf("%s at %d: submitted %q, eof %v; want %q, %v", c.Name, columns, got, h.eof(), c.Want.Lines, c.Want.EOF)
				continue
			}
			if c.Want.Pos >= 0 && (h.c.Text() != c.Want.Text || h.pos() != c.Want.Pos) {
				t.Errorf("%s at %d: %q, cursor %d; want %q, %d", c.Name, columns, h.c.Text(), h.pos(), c.Want.Text, c.Want.Pos)
			}
		}
	}
	for name := range notParity {
		if !seen[name] {
			t.Errorf("notParity names %q, which the reference does not hold", name)
		}
	}
	if compared != 47 {
		t.Errorf("%d cases compared; want 47", compared)
	}
}
