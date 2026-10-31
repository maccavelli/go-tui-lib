package when

import (
	"reflect"
	"testing"
)

// TestContextFunc: a ContextFunc is a Context that asks its function, as
// an expression and Layered see it
// (docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md Step 5).
func TestContextFunc(t *testing.T) {
	var asked []string
	c := ContextFunc(func(key string) (Value, bool) {
		asked = append(asked, key)
		if key == "mode" {
			return StringValue("insert"), true
		}
		return Value{}, false
	})
	if v, ok := c.Value("mode"); !ok || !reflect.DeepEqual(v, StringValue("insert")) {
		t.Errorf("Value(mode) = %v, %v; want insert, true", v, ok)
	}
	if _, ok := c.Value("other"); ok {
		t.Error("Value(other) is set")
	}
	if !MustParse("mode == 'insert'").Eval(c) || MustParse("mode == 'normal'").Eval(c) {
		t.Error("an expression did not read the function's value")
	}
	if v, ok := Layered(Map{}, c).Value("mode"); !ok || !reflect.DeepEqual(v, StringValue("insert")) {
		t.Errorf("Layered reached %v, %v; want insert, true", v, ok)
	}
	if len(asked) == 0 || asked[0] != "mode" {
		t.Errorf("the function was asked %q", asked)
	}
}
