package when

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// ctx is the context most tests evaluate against.
var ctx = Map{
	"a":     BoolValue(true),
	"f":     BoolValue(false),
	"n":     NumberValue(3),
	"z":     NumberValue(0),
	"s":     StringValue("hello world"),
	"e":     StringValue(""),
	"l":     ListValue([]string{"x", "y"}),
	"none":  ListValue(nil),
	"item":  StringValue("y"),
	"other": StringValue("q"),
}

func eval(t *testing.T, src string, c Context) bool {
	t.Helper()
	e, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	return e.Eval(c)
}

// operators is every operator of MADR §8, alone, with its result on ctx.
var operators = []struct {
	src  string
	want bool
}{
	{"a", true}, {"f", false}, {"z", false}, {"e", false}, {"l", true}, {"none", false},
	{"true", true}, {"false", false}, {"(a)", true},
	{"!a", false}, {"!f", true}, {"!!a", true},
	{"a && f", false}, {"a && a", true}, {"a || f", true}, {"f || f", false},
	{"n == 3", true}, {"n == 3.0", true}, {"n != 3", false}, {"n != 4", true},
	{"n < 4", true}, {"n < 3", false}, {"n <= 3", true}, {"n > 3", false}, {"n > 2", true}, {"n >= 3", true},
	{"s == 'hello world'", true}, {"s != 'hello world'", false},
	{"s =~ /^hel/", true}, {"s =~ /HELLO/i", true}, {"s =~ /^x/", false},
	{"item in l", true}, {"other in l", false}, {"item not in l", false}, {"other not in l", true},
}

func TestOperators(t *testing.T) {
	for _, c := range operators {
		if got := eval(t, c.src, ctx); got != c.want {
			t.Errorf("%q = %v, want %v", c.src, got, c.want)
		}
	}
}

func TestPrecedence(t *testing.T) {
	// !foo && bar is (!foo) && bar.
	if !eval(t, "!foo && bar", Map{"foo": BoolValue(false), "bar": BoolValue(true)}) {
		t.Error("!foo && bar with foo false, bar true: want true")
	}
	if eval(t, "!foo && bar", Map{"foo": BoolValue(true), "bar": BoolValue(false)}) {
		t.Error("!foo && bar read as !(foo && bar)")
	}
	// foo || bar && baz is foo || (bar && baz).
	if !eval(t, "foo || bar && baz", Map{"foo": BoolValue(true)}) {
		t.Error("foo || bar && baz read as (foo || bar) && baz")
	}
	for src, want := range map[string]string{
		"!foo && bar":         "!foo && bar",
		"foo || bar && baz":   "foo || bar && baz",
		"(foo || bar) && baz": "(foo || bar) && baz",
		"!(foo && bar)":       "!(foo && bar)",
		"a || (b || c)":       "a || (b || c)",
		"((a))":               "a",
	} {
		if got := MustParse(src).String(); got != want {
			t.Errorf("%q prints %q, want %q", src, got, want)
		}
	}
}

func TestUnsetKeys(t *testing.T) {
	for src, want := range map[string]bool{
		"x": false, "!x": true, "x == 'a'": false, "x != 'a'": true,
		"x < 1": false, "x >= 0": false, "x =~ /.*/": false,
	} {
		if got := eval(t, src, Map{}); got != want {
			t.Errorf("%q with x unset = %v, want %v", src, got, want)
		}
	}
	if eval(t, "x == ''", Map{}) {
		t.Error("an unset key equals ''")
	}
}

func TestIn(t *testing.T) {
	for src, want := range map[string]bool{
		"item in l": true, "item not in l": false,
		"item in missing": false, "item not in missing": true,
		"missing in l": false, "missing not in l": true,
		"l in l": false, "n in l": false,
	} {
		if got := eval(t, src, ctx); got != want {
			t.Errorf("%q = %v, want %v", src, got, want)
		}
	}
}

func TestTerms(t *testing.T) {
	c := Map{
		"s": StringValue("a b"), "v": StringValue("foo.bar"), "q": StringValue("it's"),
		"n": NumberValue(1.5), "m": NumberValue(-2), "b": BoolValue(true), "t": StringValue("true"),
	}
	for _, src := range []string{
		"s == 'a b'", "v == foo.bar", "v == 'foo.bar'", `q == 'it\'s'`,
		"n == 1.5", "m == -2", "b == true", "!(b == false)", "t == true", "t == 'true'",
	} {
		if !eval(t, src, c) {
			t.Errorf("%q is false", src)
		}
	}
}

func TestNoSpacesNeeded(t *testing.T) {
	for src, want := range map[string]string{
		"a==1&&b!=2":     "a == 1 && b != 2",
		"!a||b<=3":       "!a || b <= 3",
		"s=~/x/&&n>1":    "s =~ /x/ && n > 1",
		"( a ) && ( b )": "a && b",
		"k   ==   'v w'": "k == 'v w'",
	} {
		if got := MustParse(src).String(); got != want {
			t.Errorf("%q prints %q, want %q", src, got, want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, src := range []string{
		"", "a &", "a |", "a = 1", "a ==", "'s'", "(a", "a)", "a &&", "a not b",
		"a in", "a in 'l'", "1abc", "a == (b)", "true == a", "a =~ x", "a =~ /x", "'open",
	} {
		if _, err := Parse(src); err == nil {
			t.Errorf("Parse(%q) succeeded", src)
		}
	}
}

func TestLayered(t *testing.T) {
	top := Map{"k": StringValue("top")}
	bottom := Map{"k": StringValue("bottom"), "o": StringValue("only")}
	c := Layered(top, nil, bottom)
	if v, ok := c.Value("k"); !ok || v.String() != "top" {
		t.Errorf("k = %v, %v; want the first context's", v, ok)
	}
	if v, ok := c.Value("o"); !ok || v.String() != "only" {
		t.Errorf("o = %v, %v; want the second context's", v, ok)
	}
	if _, ok := c.Value("missing"); ok {
		t.Error("a key no context holds is set")
	}
}

func TestCheck(t *testing.T) {
	known := Keys{"n": KindNumber, "s": KindString, "l": KindList, "b": KindBool}
	cases := []struct {
		src  string
		want []string
	}{
		{"n > 2 && s == 'a' && s in l && b && s =~ /x/", nil},
		{"typo", []string{`unknown key "typo"`}},
		{"typo && typo", []string{`unknown key "typo"`}},
		{"n == 'abc'", []string{`key "n" is a number, compared with the string 'abc'`}},
		{"s < 3", []string{`key "s" is a string, compared as a number`}},
		{"n < abc", []string{`not a number`}},
		{"s in n", []string{`key "n" is a number, not a list`}},
		{"l == 'x'", []string{`key "l" is a list`}},
		{"l =~ /x/", []string{`key "l" is a list, matched`}},
	}
	for _, c := range cases {
		errs := Check(MustParse(c.src), known)
		if len(errs) != len(c.want) {
			t.Errorf("%q: %d errors %v, want %d", c.src, len(errs), errs, len(c.want))
			continue
		}
		for i, w := range c.want {
			if !strings.Contains(errs[i].Error(), w) {
				t.Errorf("%q: error %q lacks %q", c.src, errs[i], w)
			}
		}
	}
}

func TestLimits(t *testing.T) {
	if _, err := Parse(strings.Repeat("a", MaxSource+1)); err == nil {
		t.Error("a source of MaxSource+1 bytes parsed")
	}
	if _, err := Parse(strings.Repeat("a", MaxSource)); err != nil {
		t.Errorf("a source of MaxSource bytes: %v", err)
	}
	// Sources within MaxSource whose canonical forms are longer: spaces
	// around operators, and a bareword's backslashes doubled in quotes.
	for _, src := range []string{
		strings.Repeat("a&&", (MaxSource-1)/3) + "a",
		"a==" + strings.Repeat(`\`, MaxSource-3),
	} {
		if len(src) > MaxSource {
			t.Fatalf("a test source of %d bytes", len(src))
		}
		if _, err := Parse(src); err == nil || !strings.Contains(err.Error(), "canonical form") {
			t.Errorf("a source of %d bytes printing longer than MaxSource: %v", len(src), err)
		}
	}
	nest := func(n int) string { return strings.Repeat("(", n) + "a" + strings.Repeat(")", n) }
	if _, err := Parse(nest(MaxDepth)); err != nil {
		t.Errorf("%d nested parentheses: %v", MaxDepth, err)
	}
	if _, err := Parse(nest(MaxDepth + 1)); err == nil {
		t.Errorf("%d nested parentheses parsed", MaxDepth+1)
	}
	if _, err := Parse(strings.Repeat("!", MaxDepth+1) + "a"); err == nil {
		t.Errorf("%d negations parsed", MaxDepth+1)
	}
}

func TestRegexIsRE2(t *testing.T) {
	// A pattern that backtracks exponentially in other engines.
	c := Map{"s": StringValue(strings.Repeat("a", 100_000) + "!")}
	began := time.Now()
	if eval(t, "s =~ /(a+)+$/", c) {
		t.Error("matched")
	}
	if d := time.Since(began); d > 2*time.Second {
		t.Errorf("took %v", d)
	}
	for _, src := range []string{"s =~ /(/", "s =~ /a/x", "s =~ /a/ig"} {
		if _, err := Parse(src); err == nil {
			t.Errorf("Parse(%q) succeeded", src)
		}
	}
	if !eval(t, `s =~ /a\/b/`, Map{"s": StringValue("a/b")}) {
		t.Error("an escaped slash in a regex")
	}
}

func TestStringRoundTrip(t *testing.T) {
	srcs := []string{
		`q == 'it\'s \\ ok'`, "a && (b || !c) && d =~ /x\\/y/im", "n >= -1.25 || s not in l",
		"!(a && b)", "(a || b) && c", "!(a || b) || c && !d",
	}
	for _, c := range operators {
		srcs = append(srcs, c.src)
	}
	for _, src := range srcs {
		e := MustParse(src)
		first := e.String()
		again, err := Parse(first)
		if err != nil {
			t.Errorf("%q prints %q, which does not parse: %v", src, first, err)
			continue
		}
		if again.String() != first {
			t.Errorf("%q prints %q, then %q", src, first, again.String())
		}
		// The same text could mean something else: compare the two on
		// every assignment of true and false to the keys.
		keys := slices.Collect(e.Keys())
		for mask := range 1 << len(keys) {
			m := Map{}
			for i, k := range keys {
				m[k] = BoolValue(mask&(1<<i) != 0)
			}
			if e.Eval(m) != again.Eval(m) {
				t.Errorf("%q prints %q, which differs on %v", src, first, m)
				break
			}
		}
	}
}

func TestKeysInOrder(t *testing.T) {
	got := slices.Collect(MustParse("b && a || b == 1 || c in l").Keys())
	if !slices.Equal(got, []string{"b", "a", "c", "l"}) {
		t.Errorf("Keys = %v", got)
	}
}

func TestTypedKeys(t *testing.T) {
	m := Map{}
	NewKey[bool]("b", "").Set(m, true)
	NewKey[int]("i", "").Set(m, 7)
	NewKey[float64]("f", "").Set(m, 1.5)
	NewKey[string]("s", "").Set(m, "v")
	NewKey[[]string]("l", "").Set(m, []string{"x"})
	if v, ok := NewKey[bool]("b", "").Get(m); !ok || !v {
		t.Error("bool")
	}
	if v, ok := NewKey[int]("i", "").Get(m); !ok || v != 7 {
		t.Error("int")
	}
	if v, ok := NewKey[float64]("f", "").Get(m); !ok || v != 1.5 {
		t.Error("float64")
	}
	if v, ok := NewKey[string]("s", "").Get(m); !ok || v != "v" {
		t.Error("string")
	}
	if v, ok := NewKey[[]string]("l", "").Get(m); !ok || !slices.Equal(v, []string{"x"}) {
		t.Error("list")
	}
	if _, ok := NewKey[string]("i", "").Get(m); ok {
		t.Error("a number read as a string")
	}
	if _, ok := NewKey[int]("f", "").Get(m); ok {
		t.Error("1.5 read as an int")
	}
	for k, want := range map[Kind]Kind{
		NewKey[bool]("", "").Kind(): KindBool, NewKey[int]("", "").Kind(): KindNumber,
		NewKey[string]("", "").Kind(): KindString, NewKey[[]string]("", "").Kind(): KindList,
	} {
		if k != want {
			t.Errorf("Kind %d, want %d", k, want)
		}
	}
}

func TestZeroExpr(t *testing.T) {
	var e Expr
	if e.Eval(ctx) || e.String() != "" || len(slices.Collect(e.Keys())) != 0 {
		t.Error("the zero Expr is not false and empty")
	}
	if !MustParse("x != 'a'").Eval(nil) {
		t.Error("a nil context is not empty")
	}
}
