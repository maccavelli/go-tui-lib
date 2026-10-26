package msgcat

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

const (
	idCount MessageID = "test.count"
	idNone  MessageID = "test.none"
)

// TestFormat: the default for a nil catalogue and for an ID the catalogue
// keeps; the catalogue's text otherwise; arguments placed by fmt, a
// translation free to reorder them; and both Table and CatalogFunc are
// catalogues.
func TestFormat(t *testing.T) {
	const def = "%d of %d"
	for name, c := range map[string]Catalog{"nil": nil, "nil Table": Table(nil), "other ID": Table{idNone: "x"}} {
		if got := Format(c, idCount, def, 1, 3); got != "1 of 3" {
			t.Errorf("%s: %q; want the default, 1 of 3", name, got)
		}
	}
	tr := Table{idCount: "%[2]d entries, number %[1]d"}
	if got := Format(tr, idCount, def, 1, 3); got != "3 entries, number 1" {
		t.Errorf("reordered: %q", got)
	}
	fn := CatalogFunc(func(id MessageID) (string, bool) { return strings.ToUpper(string(id)), id == idNone })
	if got := Format(fn, idNone, "none"); got != "TEST.NONE" {
		t.Errorf("CatalogFunc: %q", got)
	}
	if got := Format(fn, idCount, def, 2, 5); got != "2 of 5" {
		t.Errorf("CatalogFunc keeping the default: %q", got)
	}
	if got := Format(Table{idNone: "100%%"}, idNone, "none"); got != "100%" {
		t.Errorf("a literal percent: %q", got)
	}
}

// TestCheck: a translation whose verbs match its default's passes, in any
// order of arguments; one that drops a verb, adds one, or formats an
// argument with another verb is reported by its ID; a message the
// catalogue keeps the default of is not checked.
func TestCheck(t *testing.T) {
	defaults := Table{
		"t.a": "%d of %d",
		"t.b": "%s failed",
		"t.c": "no results",
		"t.d": "%-8s|%*d",
	}
	good := Table{
		"t.a": "%[2]d total, at %[1]d",
		"t.b": "échec de %s",
		"t.d": "%[1]-8s|%[2]*[3]d",
	}
	if err := Check(defaults, good); err != nil {
		t.Fatalf("matching translations: %v", err)
	}
	bad := Table{
		"t.a": "%d",           // drops one
		"t.b": "%s failed %d", // adds one
		"t.c": "nothing for %s",
		"t.d": "%-8s|%*s", // another verb
	}
	err := Check(defaults, bad)
	if err == nil {
		t.Fatal("mismatched translations passed")
	}
	for _, id := range []string{"t.a", "t.b", "t.c", "t.d"} {
		if !strings.Contains(err.Error(), id+":") {
			t.Errorf("%s not reported: %v", id, err)
		}
	}
	if err := Check(defaults, Table{}); err != nil {
		t.Errorf("an empty catalogue: %v", err)
	}
}

// TestVerbs: the arguments a format reads, by number and verb, as fmt
// numbers them.
func TestVerbs(t *testing.T) {
	for _, c := range []struct {
		in   string
		want string
	}{
		{"plain", "none"},
		{"100%% done", "none"},
		{"%d of %d", "%[1]d %[2]d"},
		{"%[2]s then %[1]q", "%[1]q %[2]s"},
		{"%-08.3f", "%[1]f"},
		{"%*d", "%[1]* %[2]d"},
		{"%[3]*.[2]*[1]f", "%[1]f %[2]* %[3]*"},
		{"%[x", "%[1]?"},
	} {
		if got := show(verbs(c.in)); got != c.want {
			t.Errorf("verbs(%q) = %s; want %s", c.in, got, c.want)
		}
	}
	// fmt agrees on how many arguments each integer format reads: given as
	// many integers as its verbs name, it prints no error marker, and given
	// one fewer, it does.
	for _, f := range []string{"%d of %d", "%[2]d then %[1]d", "%*d", "%[3]*.[2]*[1]d"} {
		n := 0
		for _, v := range verbs(f) {
			n = max(n, v.arg)
		}
		if s := fmt.Sprintf(f, slices.Repeat([]any{3}, n)...); strings.Contains(s, "%!") {
			t.Errorf("%q with %d arguments: %q", f, n, s)
		}
		if s := fmt.Sprintf(f, slices.Repeat([]any{3}, n-1)...); !strings.Contains(s, "%!") {
			t.Errorf("%q with %d arguments printed no marker: %q", f, n-1, s)
		}
	}
}

// ExampleFormat: a program's catalogue rewords one message and reorders
// its arguments; another message keeps its default.
func ExampleFormat() {
	c := Table{"palette.count": "%[2]d results, at %[1]d"}
	fmt.Println(Format(c, "palette.count", "%d of %d", 3, 40))
	fmt.Println(Format(c, "palette.no-results", "no results"))
	// Output:
	// 40 results, at 3
	// no results
}
