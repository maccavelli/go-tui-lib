package when

import (
	"strings"
	"testing"
)

// FuzzParse: Parse never panics; a parsed expression's canonical form
// parses to the same form; and evaluation never panics, and gives the same
// result for the expression and its canonical form, on a context whose keys
// and values come from the input.
func FuzzParse(f *testing.F) {
	for _, c := range operators {
		f.Add(c.src)
	}
	for _, s := range []string{`q == 'it\'s'`, "a && (b || !c)", "s =~ /x\\/y/im", "n >= -1.25", "((((a))))"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		e, err := Parse(src)
		if err != nil {
			return
		}
		first := e.String()
		again, err := Parse(first)
		if err != nil {
			t.Fatalf("%q prints %q, which does not parse: %v", src, first, err)
		}
		if again.String() != first {
			t.Fatalf("%q prints %q, then %q", src, first, again.String())
		}
		m := Map{}
		for i, k := range []string{"a", "b", "s", "n", "l"} {
			switch i % 4 {
			case 0:
				m[k] = BoolValue(len(src)%2 == 0)
			case 1:
				m[k] = NumberValue(float64(len(src)))
			case 2:
				m[k] = StringValue(src)
			default:
				m[k] = ListValue(strings.Fields(src))
			}
		}
		for k := range e.Keys() {
			if _, ok := m[k]; !ok {
				m[k] = StringValue(k)
			}
		}
		if e.Eval(m) != again.Eval(m) || e.Eval(Map{}) != again.Eval(Map{}) {
			t.Fatalf("%q prints %q, which evaluates differently", src, first)
		}
	})
}
