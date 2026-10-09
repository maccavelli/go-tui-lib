package layout

import (
	"encoding/json"
	"errors"
	"testing"
)

// FuzzState: a State that unmarshals marshals and unmarshals again to the
// same JSON, and solves a fixed tree, at two sizes, with every pane and
// separator in its area and none overlapping
// (docs/decisions/0014-PLAN-hardening.md Step 9, finding H9).
func FuzzState(f *testing.F) {
	tree := SidebarRightBottom("main", "side", "bottom")
	areas := []Rect{{W: 120, H: 40}, {W: 30, H: 8}}
	p, err := Solve(tree, areas[0], State{})
	if err != nil {
		f.Fatal(err)
	}
	for _, s := range p.Separators {
		if s.Resizable {
			f.Add([]byte(`{"version":1,"resize":{"` + s.ID + `":5}}`))
			f.Add([]byte(`{"version":1,"resize":{"` + s.ID + `":-9223372036854775808}}`))
		}
	}
	for _, s := range []string{
		`{}`, `{"version":1}`, `{"version":2}`, `{"version":1,"hidden":["side","bottom"]}`,
		`{"version":1,"zoom":"main"}`, `{"version":1,"zoom":"nope","hidden":["main"]}`,
		`{"resize":{}}`, `{"hidden":null}`, `null`, `[]`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var s State
		if json.Unmarshal(data, &s) != nil {
			return
		}
		out, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("%q unmarshals, and does not marshal: %v", data, err)
		}
		var back State
		if err := json.Unmarshal(out, &back); err != nil {
			t.Fatalf("%q marshals to %q, which does not unmarshal: %v", data, out, err)
		}
		if again, _ := json.Marshal(back); string(again) != string(out) {
			t.Fatalf("%q round-trips to %q, then %q", data, out, again)
		}
		for _, area := range areas {
			p, err := Solve(tree, area, s)
			if err != nil {
				if errors.Is(err, ErrStateVersion) {
					t.Fatalf("%q unmarshals with a version Solve refuses: %v", data, err)
				}
				t.Fatalf("%q at %dx%d: %v", data, area.W, area.H, err)
			}
			check(t, string(data), p, area, false)
		}
	})
}
