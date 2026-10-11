package keymap

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// FuzzLoadVSCode: LoadVSCode does not panic on any input; each error is a
// *LoadError at a line and column inside the input; what it loads,
// ExportVSCode writes and LoadVSCode reads back unchanged; and the
// schema's key pattern accepts any input ParseSequence reads.
func FuzzLoadVSCode(f *testing.F) {
	for _, s := range []string{
		userFile,
		"[",
		"// c\n[{\"key\": \"a\", /* x */ \"command\": \"app.x\",},]",
		`[{"key": "a", "command": "app.x", "key": "b", "args": {"a": [1, {"b": null}]}}]`,
		"[\n\t/* open",
		"\xef\xbb\xbf[{\"command\": \"-app.x\", \"context\": \"pane:a\"}]",
		"ctrl+k <leader> a:release",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		bs, err := LoadVSCode(strings.NewReader(src), "f.json", nil)
		lines := strings.Split(src, "\n")
		for _, e := range unjoin(err) {
			var le *LoadError
			if !errors.As(e, &le) {
				t.Fatalf("an error is %T, not a *LoadError: %v", e, e)
			}
			if le.Line < 1 || le.Line > len(lines) || le.Col < 1 || le.Col > len(lines[le.Line-1])+1 {
				t.Fatalf("%v: %d:%d is outside the input", le, le.Line, le.Col)
			}
		}
		var b bytes.Buffer
		if err := ExportVSCode(&b, slices.Values(bs)); err != nil {
			t.Fatalf("ExportVSCode: %v", err)
		}
		got, err := LoadVSCode(&b, "f.json", nil)
		if err != nil || !reflect.DeepEqual(placeless(got), placeless(bs)) {
			t.Fatalf("read back %+v, %v; want %+v", got, err, bs)
		}
		if _, err := ParseSequence(src, nil); err == nil && !keyRE.MatchString(src) {
			t.Fatalf("the pattern refuses the key %q", src)
		}
	})
}
