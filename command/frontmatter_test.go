package command

import (
	"context"
	"testing"
	"testing/fstest"
)

func TestExpand(t *testing.T) {
	body := "Focus on $FOCUS. $ARGUMENTS | $lower $$ $(ls) !{ls} ${FOCUS} $MISSING $FOCUS"
	got, err := expand(body, map[string]string{"FOCUS": "$ARGUMENTS"}, "the tail")
	want := "Focus on $ARGUMENTS. the tail | $lower $$ $(ls) !{ls} ${FOCUS} $MISSING $ARGUMENTS"
	if got != want || err != nil {
		t.Errorf("expand:\n got %q, %v\nwant %q", got, err, want)
	}
	// Through the registry: a loaded file's arguments and slash tail.
	cmds, errs := LoadDir(fstest.MapFS{"r.md": {Data: []byte("Check $WHAT then !{rm -rf /} and $(id): $ARGUMENTS")}}, userSrc)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	r := NewRegistry()
	r.ReplaceSource(userSrc, cmds)
	req, err := r.ParseSlash("/r tests")
	if err != nil {
		t.Fatal(err)
	}
	req.Raw = "and more"
	res, err := r.Run(context.Background(), req)
	if err != nil || res.Text != "Check tests then !{rm -rf /} and $(id): and more" {
		t.Errorf("Run: %q, %v", res.Text, err)
	}
	if got := placeholders("$B $A $B $ARGUMENTS $C_1"); len(got) != 3 || got[0] != "B" || got[1] != "A" || got[2] != "C_1" {
		t.Errorf("placeholders = %v", got)
	}
}

// FuzzFrontMatter: parsing never panics, and front matter that parses
// re-serialises to front matter that parses to the same keys, values and
// body.
func FuzzFrontMatter(f *testing.F) {
	for _, s := range []string{
		"---\ntitle: x\ndescription: \"a: b\"\naliases: a, b\ndanger: ui\nwhen: a && !b\nhidden: true\narg.X: x\n---\n$X",
		"plain", "---\n---\n---\nbody", "---\ntitle: 'q'\n---\n", "---\r\ntitle: x\r\n---\r\nbody\r\n",
		"---\ndescription: \"\"x\"\"\n---\n", "---\nslash: a:b\n---\n",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		fm, body, err := parseFrontMatter(src)
		if err != nil {
			return
		}
		again := fm.String() + body
		fm2, body2, err := parseFrontMatter(again)
		if err != nil {
			t.Fatalf("%q re-serialised as %q, which does not parse: %v", src, again, err)
		}
		if body2 != body || fm2.present != fm.present || len(fm2.fields) != len(fm.fields) {
			t.Fatalf("%q re-serialised as %q: %+v %q, then %+v %q", src, again, fm, body, fm2, body2)
		}
		for i, x := range fm.fields {
			if y := fm2.fields[i]; y.key != x.key || y.value != x.value {
				t.Fatalf("%q re-serialised as %q: field %d %+v, then %+v", src, again, i, x, y)
			}
		}
	})
}
