package when

import "testing"

// benchSrc is a five-clause expression, as a command's when might be.
const benchSrc = "workspace.focusedPane == transcript && !workspace.modal && workspace.width >= 80 && " +
	"mode in modes || workspace.overlay =~ /^picker/"

func BenchmarkParse(b *testing.B) {
	for b.Loop() {
		if _, err := Parse(benchSrc); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEval(b *testing.B) {
	e := MustParse(benchSrc)
	c := Map{
		"workspace.focusedPane": StringValue("transcript"),
		"workspace.modal":       BoolValue(false),
		"workspace.width":       NumberValue(120),
		"mode":                  StringValue("insert"),
		"modes":                 ListValue([]string{"normal", "insert"}),
		"workspace.overlay":     StringValue(""),
	}
	for b.Loop() {
		e.Eval(c)
	}
}
