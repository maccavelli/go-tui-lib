package stream

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// print feeds chunks to a Printer made with o, in turn, and flushes it.
func print(chunks []string, o ...Option) string {
	p := NewPrinter(o...)
	var b strings.Builder
	for _, c := range chunks {
		b.WriteString(p.Write(c))
	}
	b.WriteString(p.Flush())
	return b.String()
}

// visible is s with ESC shown as <ESC>, for a golden that stays readable.
func visible(s string) string { return strings.ReplaceAll(s, "\x1b", "<ESC>") }

// styleCases are gobble-cli's (its render/mdstyle_test.go at 5fb6928), the
// styles its term package writes, which DefaultMarkup's are.
var styleCases = []struct{ name, in, want string }{
	{"heading", "## Title\n", "\x1b[1m## Title\x1b[0m\n"},
	{"not a heading", "#hashtag\n", "#hashtag\n"},
	{"list marker", "- item\n", "\x1b[36m-\x1b[0m item\n"},
	{"ordered marker", "  12. item\n", "  \x1b[36m12.\x1b[0m item\n"},
	{"quote marker", "> said\n", "\x1b[36m>\x1b[0m said\n"},
	{"inline code", "run `go test` now\n", "run \x1b[2m`go test`\x1b[0m now\n"},
	{"unclosed inline code", "a `b\n", "a `b\n"},
	{"fence with label", "```go\nx := 1\n```\n", "\x1b[2m```\x1b[0m\x1b[36mgo\x1b[0m\nx := 1\n\x1b[2m```\x1b[0m\n"},
	{"code inside a fence is not styled", "```\n# not a heading\n- nor a list\n```\n",
		"\x1b[2m```\x1b[0m\n# not a heading\n- nor a list\n\x1b[2m```\x1b[0m\n"},
	{"no trailing newline", "# T", "\x1b[1m# T\x1b[0m"},
	{"bold", "a **b c** d\n", "a \x1b[1m**b c**\x1b[0m d\n"},
	{"bold underscores", "__b__\n", "\x1b[1m__b__\x1b[0m\n"},
	{"italic", "an *it* and _it_\n", "an \x1b[3m*it*\x1b[0m and \x1b[3m_it_\x1b[0m\n"},
	{"bold italic", "***both***\n", "\x1b[1;3m***both***\x1b[0m\n"},
	{"strikethrough", "~~gone~~ ~not~\n", "\x1b[2m~~gone~~\x1b[0m ~not~\n"},
	{"snake_case stays plain", "call snake_case_name now\n", "call snake_case_name now\n"},
	{"spaced stars stay plain", "2 * 3 * 4\n", "2 * 3 * 4\n"},
	{"unclosed", "**open and *half\n", "**open and *half\n"},
	{"escaped", "\\*not\\* *yes*\n", "\\*not\\* \x1b[3m*yes*\x1b[0m\n"},
	{"not nested", "**a *b* c**\n", "\x1b[1m**a *b* c**\x1b[0m\n"},
	{"beside a code span", "`a*b*` and *c*\n", "\x1b[2m`a*b*`\x1b[0m and \x1b[3m*c*\x1b[0m\n"},
	{"not across a code span", "*a `b` c*\n", "*a \x1b[2m`b`\x1b[0m c*\n"},
	{"a star list marker then emphasis", "* *it*\n", "\x1b[36m*\x1b[0m \x1b[3m*it*\x1b[0m\n"},
	{"a lone star marker is not an opener", "* item\n", "\x1b[36m*\x1b[0m item\n"},
	// Links, 0023-MADR A6 item 1.
	{"an inline link", "see [the docs](https://x.example/a) now\n", "see \x1b[4m[the docs]\x1b[0m\x1b[2m(https://x.example/a)\x1b[0m now\n"},
	{"an image", "![alt](i.png)\n", "\x1b[4m!\x1b[0m\x1b[4m[alt]\x1b[0m\x1b[2m(i.png)\x1b[0m\n"},
	{"a link's text holds no emphasis", "[*a*](u)\n", "\x1b[4m[*a*]\x1b[0m\x1b[2m(u)\x1b[0m\n"},
	{"an undefined reference is text", "[a][b] and [c]\n", "[a][b] and [c]\n"},
	{"brackets with a space before the parenthesis", "[a] (u)\n", "[a] (u)\n"},
}

// TestApply: a Style's SGR parameters in order, the lowest colour alone;
// the zero Style and empty text are left as they are.
func TestApply(t *testing.T) {
	for _, c := range []struct {
		s        Style
		in, want string
	}{
		{0, "x", "x"},
		{Bold, "", ""},
		{Bold | Underline, "x", "\x1b[1;4mx\x1b[0m"},
		{Dim | Italic, "x", "\x1b[2;3mx\x1b[0m"},
		{Red | Cyan, "x", "\x1b[31mx\x1b[0m"},
		{Bold | White, "x", "\x1b[1;37mx\x1b[0m"},
	} {
		if got := c.s.Apply(c.in); got != c.want {
			t.Errorf("%v.Apply(%q) = %q; want %q", c.s, c.in, got, c.want)
		}
	}
}

// TestStyleCases: each of gobble's styling cases, and the links'.
func TestStyleCases(t *testing.T) {
	for _, c := range styleCases {
		if got := print([]string{c.in}, WithMarkup(DefaultMarkup())); got != c.want {
			t.Errorf("%s: %q gives %q; want %q", c.name, c.in, got, c.want)
		}
	}
}

// TestStyleGolden: gobble's styling input, printed with DefaultMarkup, its
// table laid out.
func TestStyleGolden(t *testing.T) {
	in, err := os.ReadFile("testdata/style.input.txt")
	if err != nil {
		t.Fatal(err)
	}
	tuitest.Text(t, "style", visible(print([]string{string(in)}, WithMarkup(DefaultMarkup()))))
}

// TestZeroMarkup: the zero Markup and no width give the input byte for
// byte under WithoutTableLayout, and without it but for a table's layout
// (0023-MADR A7); every Markup keeps every byte once its escape sequences
// are taken out.
func TestZeroMarkup(t *testing.T) {
	in, err := os.ReadFile("testdata/style.input.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got := print([]string{string(in)}, WithoutTableLayout()); got != string(in) {
		t.Errorf("the zero Markup without table layout changed the text:\n%q", got)
	}
	table := "| a | b |\n|---|---|\n| 1 | 2 |\n"
	if got := print([]string{table}); got != "| a   | b   |\n| --- | --- |\n| 1   | 2   |\n" {
		t.Errorf("a table not laid out by default: %q", got)
	}
	if got := print([]string{table}, WithoutTableLayout(), WithMarkup(DefaultMarkup())); got != table {
		t.Errorf("a table's rows not as they came: %q", got)
	}
	if got := print([]string{"| a `b` |\n|---|\n"}, WithoutTableLayout(), WithMarkup(DefaultMarkup())); got != "| a \x1b[2m`b`\x1b[0m |\n|---|\n" {
		t.Errorf("a table's rows as they came, not styled: %q", got)
	}
	plain := print([]string{string(in)})
	if got := ansi.Strip(print([]string{string(in)}, WithMarkup(DefaultMarkup()))); got != plain {
		t.Errorf("styled less its escapes is not the plain text:\n%q\nwant\n%q", got, plain)
	}
}

// wrapCases are gobble-cli's (its render/wrap_test.go at 5fb6928); a table
// is laid out, where gobble's Wrap passed one as it came.
var wrapCases = []struct {
	name, in string
	width    int
	want     string
}{
	{"prose breaks at spaces", "The quick brown fox jumps over the lazy dog.\n", 20,
		"The quick brown fox\njumps over the lazy\ndog.\n"},
	{"a trailing comma stays with its word", "aaaa bbbb cccc dddd, eeee\n", 20, "aaaa bbbb cccc dddd,\neeee\n"},
	{"a word wider than the row is whole on its own row", "see https://example.com/a/very/long/path ok\n", 20,
		"see\nhttps://example.com/a/very/long/path\nok\n"},
	{"wide characters take two columns", "漢字漢字 漢字漢字 漢字漢字\n", 18, "漢字漢字 漢字漢字\n漢字漢字\n"},
	{"a list item continues under its text", "- alpha beta gamma delta epsilon\n", 20,
		"- alpha beta gamma\n  delta epsilon\n"},
	{"an ordered item continues under its text", "12. alpha beta gamma delta\n", 20,
		"12. alpha beta gamma\n    delta\n"},
	{"a nested item keeps its indent", "  * alpha beta gamma delta\n", 20, "  * alpha beta gamma\n    delta\n"},
	{"a quote continues with its marker", "> alpha beta gamma delta epsilon\n", 20,
		"> alpha beta gamma\n> delta epsilon\n"},
	{"a list in a quote", "> - alpha beta gamma delta\n", 20, "> - alpha beta gamma\n>   delta\n"},
	{"a lone dash does not start a row", "aaaa bbbb cccc dddd - eeee\n", 20, "aaaa bbbb cccc\ndddd - eeee\n"},
	{"a number and a dot do not start a row", "aaaa bbbb cccc dddd 1990. eeee\n", 20,
		"aaaa bbbb cccc\ndddd 1990. eeee\n"},
	{"hashes do not start a row", "aaaa bbbb cccc dddd ## eeee\n", 20, "aaaa bbbb cccc\ndddd ## eeee\n"},
	{"a quote marker does not start a row", "aaaa bbbb cccc dddd >x eeee\n", 20, "aaaa bbbb cccc\ndddd >x eeee\n"},
	{"a pipe does not start a row", "aaaa bbbb cccc dddd |x| eeee\n", 20, "aaaa bbbb cccc\ndddd |x| eeee\n"},
	{"a fence does not start a row", "aaaa bbbb cccc dddd ```x eeee\n", 20, "aaaa bbbb cccc\ndddd ```x eeee\n"},
	{"a rule does not start a row", "aaaa bbbb cccc dddd *** eeee\n", 20, "aaaa bbbb cccc\ndddd *** eeee\n"},
	{"fenced code passes through", "```go\nfunc main() { fmt.Println(\"a long line of code\") }\n```\nalpha beta gamma delta\n", 20,
		"```go\nfunc main() { fmt.Println(\"a long line of code\") }\n```\nalpha beta gamma\ndelta\n"},
	{"a tilde fence, longer run", "~~~~\nlong code line here that exceeds\n~~~\nstill code inside here ok\n~~~~\n", 20,
		"~~~~\nlong code line here that exceeds\n~~~\nstill code inside here ok\n~~~~\n"},
	{"backticks closed on their line are inline code, not a fence", "```a``` then some words that wrap here\n", 20,
		"```a``` then some\nwords that wrap here\n"},
	{"an unclosed fence passes through", "```\nlong code line here that exceeds the width", 20,
		"```\nlong code line here that exceeds the width"},
	{"indented code passes through", "    indented code line that is long enough\n", 20,
		"    indented code line that is long enough\n"},
	{"a tab-indented line passes through", "\tindented code line that is long enough\n", 20,
		"\tindented code line that is long enough\n"},
	{"a table is laid out and not wrapped", "| a | b |\n|---|---|\n| long cell value here | other long cell |\n", 20,
		"| a                    | b               |\n| -------------------- | --------------- |\n| long cell value here | other long cell |\n"},
	{"a heading passes through", "## A heading that is longer than twenty\n", 20, "## A heading that is longer than twenty\n"},
	{"blank lines and line ends are kept", "one two\n\nthree four\nfive\n", 20, "one two\n\nthree four\nfive\n"},
	{"a hard line end is not joined", "alpha beta gamma delta\nepsilon\n", 20, "alpha beta gamma\ndelta\nepsilon\n"},
	{"no final line end", "alpha beta gamma delta", 20, "alpha beta gamma\ndelta"},
	{"spaces between words are kept", "a  b   c\n", 20, "a  b   c\n"},
	{"a row exactly the width", "aaaaaaaaa bbbbbbbbbb\n", 20, "aaaaaaaaa bbbbbbbbbb\n"},
	{"no width, no wrapping", "The quick brown fox jumps over the lazy dog.\n", 0,
		"The quick brown fox jumps over the lazy dog.\n"},
}

// TestWrap: each of gobble's wrapping cases.
func TestWrap(t *testing.T) {
	for _, c := range wrapCases {
		if got := print([]string{c.in}, WithWidth(c.width)); got != c.want {
			t.Errorf("%s:\n%q\nwant\n%q", c.name, got, c.want)
		}
	}
}

// TestAnySplit: any split of a reply gives what the whole reply gives:
// every split in two, and random splits into many, plain and styled
// (gobble-cli's TestWrapAnySplit).
func TestAnySplit(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	in, err := os.ReadFile("testdata/style.input.txt")
	if err != nil {
		t.Fatal(err)
	}
	inputs := []string{string(in)}
	for _, c := range wrapCases {
		inputs = append(inputs, c.in)
	}
	for _, e := range setextExamples {
		inputs = append(inputs, e.md)
	}
	for _, e := range tableExamples {
		inputs = append(inputs, e.md)
	}
	for _, o := range [][]Option{{WithWidth(20)}, {WithWidth(20), WithMarkup(DefaultMarkup())}} {
		for _, s := range inputs {
			want := print([]string{s}, o...)
			for c := range s {
				if c == 0 || !utf8.RuneStart(s[c]) {
					continue
				}
				if got := print([]string{s[:c], s[c:]}, o...); got != want {
					t.Fatalf("%q split at %d: %q; want %q", s, c, got, want)
				}
			}
			for range 20 {
				var chunks []string
				for rest := s; rest != ""; {
					n := min(1+r.IntN(5), len(rest))
					chunks = append(chunks, rest[:n])
					rest = rest[n:]
				}
				if got := print(chunks, o...); got != want {
					t.Fatalf("%q in chunks %q: %q; want %q", s, chunks, got, want)
				}
			}
		}
	}
}

// TestSetext: every setext heading example of CommonMark 0.31.2, printed
// with DefaultMarkup: a heading's last line and its underline in the
// Heading style, an earlier line printed as the paragraph it was (0023-MADR
// A6 item 4), and a lazy continuation never a heading's text.
func TestSetext(t *testing.T) {
	var b strings.Builder
	for _, e := range setextExamples {
		fmt.Fprintf(&b, "## example %d\n%s--\n%s\n", e.n, e.md, visible(print([]string{e.md}, WithMarkup(DefaultMarkup()))))
	}
	tuitest.Text(t, "setext", b.String())
}

// TestTables: every table example of GFM 0.29-gfm, printed with
// DefaultMarkup at 40 columns.
func TestTables(t *testing.T) {
	var b strings.Builder
	for _, e := range tableExamples {
		fmt.Fprintf(&b, "## example %d\n%s--\n%s\n", e.n, e.md, visible(print([]string{e.md}, WithMarkup(DefaultMarkup()), WithWidth(40))))
	}
	tuitest.Text(t, "tables", b.String())
}

// TestTableLayout: gobble-cli's table golden (its render/table_test.go at
// 5fb6928): columns as wide as their widest cell in cells, aligned as the
// delimiter row says, a short row padded, an escaped pipe kept in its cell.
func TestTableLayout(t *testing.T) {
	in := "Before.\n\n" +
		"| Name | Lang | Count |\n" +
		"|:---|:---:|---:|\n" +
		"| gobble | Go | 1 |\n" +
		"| 世界 | 日本語 | 12345 |\n" +
		"| esc \\| pipe | x |\n" +
		"\nAfter.\n" +
		"| not | a table |\n" +
		"no delimiter row\n" +
		"| last | table |\n|---|---|\n| no | newline |"
	got := print([]string{in})
	tuitest.Text(t, "layout", got)
	if !strings.HasSuffix(got, "\n| no   | newline |") { // the golden's rows do not show a last line break
		t.Errorf("a table that ends the reply with no line break: %q", got[max(0, len(got)-40):])
	}
}

// TestNotTables: lines that are not a table print as they came (gobble-cli's
// TestTablesLeaveOtherTextAlone).
func TestNotTables(t *testing.T) {
	for _, in := range []string{"", "plain\n", "| a |\n| b |\n", "|a|b|\n|-|x|\n"} {
		if got := print([]string{in}); got != in {
			t.Errorf("%q gives %q", in, got)
		}
	}
}

// TestHoldOneLine: a paragraph's line is printed once the next line has
// come, and not before; the reply's end prints the last.
func TestHoldOneLine(t *testing.T) {
	p := NewPrinter()
	steps := []struct{ in, want string }{
		{"Foo\n", ""},
		{"Bar\n", "Foo\n"},
		{"===\n", "Bar\n===\n"},
		{"tail", ""},
	}
	for _, s := range steps {
		if got := p.Write(s.in); got != s.want {
			t.Fatalf("Write(%q) = %q; want %q", s.in, got, s.want)
		}
	}
	if got := p.Flush(); got != "tail" {
		t.Errorf("Flush() = %q; want tail", got)
	}
}

// TestTableHeld: a table's rows are held until a blank line ends it, then
// laid out at once.
func TestTableHeld(t *testing.T) {
	p := NewPrinter()
	for _, in := range []string{"| a | b |\n", "|---|---|\n", "| 1 | 2 |\n"} {
		if got := p.Write(in); got != "" {
			t.Fatalf("Write(%q) printed %q before the table ended", in, got)
		}
	}
	if got := p.Write("\n"); got != "| a   | b   |\n| --- | --- |\n| 1   | 2   |\n\n" {
		t.Errorf("the table at its end: %q", got)
	}
}

// TestFenceLines: a fence's lines print as each completes.
func TestFenceLines(t *testing.T) {
	p := NewPrinter()
	for _, s := range []struct{ in, want string }{
		{"```go\n", "```go\n"},
		{"x := 1\n", "x := 1\n"},
		{"y", ""},
		{" := 2\n", "y := 2\n"},
		{"```\n", "```\n"},
	} {
		if got := p.Write(s.in); got != s.want {
			t.Fatalf("Write(%q) = %q; want %q", s.in, got, s.want)
		}
	}
}

// TestInlineOpen: a delimiter still open when its line prints is printed
// as it is, and no printed line is styled again by a later one (0023-MADR
// A3 item 14).
func TestInlineOpen(t *testing.T) {
	p := NewPrinter(WithMarkup(DefaultMarkup()))
	first := p.Write("an **open\nrun of bold**\n")
	rest := p.Flush()
	if first != "an **open\n" || rest != "run of bold**\n" {
		t.Errorf("printed %q, then %q; want each line as written", first, rest)
	}
}

// TestReferenceLater: a reference link is styled only when its definition
// was printed before it (0023-MADR A6 item 1).
func TestReferenceLater(t *testing.T) {
	got := print([]string{"see [x][r] first\n\n[r]: https://x.example\n\nsee [x][r] then, and [r] too\n"}, WithMarkup(DefaultMarkup()))
	want := "see [x][r] first\n\n\x1b[4m[r]\x1b[0m:\x1b[2m https://x.example\x1b[0m\n\nsee \x1b[4m[x]\x1b[0m\x1b[2m[r]\x1b[0m then, and \x1b[4m[r]\x1b[0m too\n"
	if got != want {
		t.Errorf("%q\nwant\n%q", got, want)
	}
}

// TestWrapCJK: a run of CJK text with no space is one word, left whole on
// a row of its own (0023-MADR A3 item 16, UAX #14 §3.1).
func TestWrapCJK(t *testing.T) {
	got := print([]string{"see 漢字漢字漢字漢字漢字漢字 now\n"}, WithWidth(10))
	if want := "see\n漢字漢字漢字漢字漢字漢字\nnow\n"; got != want {
		t.Errorf("%q; want %q", got, want)
	}
}

// TestRules: WithRules cleans more than the default, and never less than
// safetext's Text rules; a sequence split across writes is cleaned whole.
func TestRules(t *testing.T) {
	if got := print([]string{"a\x1b]0;x\x07b \x1b[31", "mred\x1b[0m\n"}, WithRules(safetext.Rules{Newlines: true, KeepSGR: true})); got != "ab red\n" {
		t.Errorf("under weaker rules: %q; want the floor's", got)
	}
	if got := print([]string{"a\tb\n"}, WithRules(safetext.Rules{Newlines: true})); got != "ab\n" {
		t.Errorf("under stricter rules: %q; want the tab removed", got)
	}
}

// TestAt: whenever the Printer is idle, At at its width gives what it has
// printed so far, for every input fed a byte at a time, plain and styled,
// with and without table layout; after a Flush, At gives the whole output;
// and At leaves the Printer as it was (0023-MADR A8 item 2).
func TestAt(t *testing.T) {
	in, err := os.ReadFile("testdata/style.input.txt")
	if err != nil {
		t.Fatal(err)
	}
	inputs := []string{string(in), "see [x][r]\n\n[r]: https://x.example\n\nthen [x][r]\n", "```go\nx := 1\n"}
	for _, c := range wrapCases {
		inputs = append(inputs, c.in)
	}
	for _, e := range setextExamples {
		inputs = append(inputs, e.md)
	}
	for _, e := range tableExamples {
		inputs = append(inputs, e.md)
	}
	for _, o := range [][]Option{{WithWidth(20)}, {WithWidth(20), WithMarkup(DefaultMarkup())}, {WithWidth(12), WithoutTableLayout()}} {
		for _, s := range inputs {
			p := NewPrinter(o...)
			var out strings.Builder
			for i := range len(s) {
				out.WriteString(p.Write(s[i : i+1]))
				if p.Idle() {
					if got := p.At(p.width); got != out.String() {
						t.Fatalf("%q after %d bytes, idle: At gives %q; printed %q", s, i+1, got, out.String())
					}
				}
			}
			out.WriteString(p.Flush())
			if got := p.At(p.width); got != out.String() {
				t.Fatalf("%q flushed: At gives %q; printed %q", s, got, out.String())
			}
		}
	}
	p := NewPrinter(WithWidth(10))
	first := p.Write("one two three four\n\n")
	_ = p.At(40)
	if rest := p.Write("five six seven\n") + p.Flush(); first+rest != "one two\nthree four\n\nfive six\nseven\n" {
		t.Errorf("At changed the Printer: %q", first+rest)
	}
	if got := p.At(40); got != "one two three four\n\nfive six seven\n" {
		t.Errorf("At(40) = %q", got)
	}
}

// TestSource: Source is the cleaned source, across a Flush; Idle is true
// only at a line's start with nothing held.
func TestSource(t *testing.T) {
	p := NewPrinter()
	if !p.Idle() {
		t.Fatal("a new Printer is not idle")
	}
	p.Write("one\x1b[2J\ntw")
	if p.Idle() {
		t.Error("idle with a line held and a partial line")
	}
	p.Flush()
	p.Write("three\n")
	if got := p.Source(); got != "one\ntwthree\n" {
		t.Errorf("Source = %q", got)
	}
}

// FuzzPrinter: for any reply, the Printer does not panic; a split in two at
// any byte gives what the whole reply gives; DefaultMarkup's output less its
// escapes is the zero Markup's; and, when every line is prose, each word on
// a row but its first ends within the width or would open a block there.
func FuzzPrinter(f *testing.F) {
	in, err := os.ReadFile("testdata/style.input.txt")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(string(in), uint16(40), uint8(20))
	for _, c := range wrapCases {
		f.Add(c.in, uint16(len(c.in)/2), uint8(c.width))
	}
	for _, e := range setextExamples {
		f.Add(e.md, uint16(3), uint8(8))
	}
	for _, e := range tableExamples {
		f.Add(e.md, uint16(7), uint8(8))
	}
	f.Fuzz(func(t *testing.T, s string, cut uint16, width uint8) {
		w := int(width % 48)
		p := NewPrinter(WithWidth(w))
		plain := p.Write(s) + p.Flush()
		c := int(cut) % (len(s) + 1)
		if got := print([]string{s[:c], s[c:]}, WithWidth(w)); got != plain {
			t.Fatalf("split at %d: %q; whole: %q", c, got, plain)
		}
		if got := ansi.Strip(print([]string{s}, WithWidth(w), WithMarkup(DefaultMarkup()))); got != plain {
			t.Fatalf("styled less its escapes: %q; plain: %q", got, plain)
		}
		if got := print([]string{s}, WithoutTableLayout()); got != p.Source() {
			t.Fatalf("the zero Markup, no width and no layout: %q; the cleaned source: %q", got, p.Source())
		}
		if w == 0 || !allProse(p.Source()) {
			return
		}
		for row := range strings.SplitSeq(plain, "\n") {
			at, first := prosePrefix(row), true
			for word := range strings.SplitSeq(row[at:], " ") {
				at += len(word)
				if word != "" {
					if end := ansi.WcWidth.StringWidth(row[:at]); !first && end > w && !opensBlock(word) {
						t.Fatalf("width %d: row %q: %q ends at %d", w, row, word, end)
					}
					first = false
				}
				at++ // the space after it
			}
		}
	})
}

// allProse reports a cleaned source whose every line is wrapped as prose:
// none passes through as code, a heading, HTML, a table, a definition or a
// rule, and none holds a tab, which the row's columns count differently.
func allProse(src string) bool {
	if strings.ContainsAny(src, "\t|`~<") {
		return false
	}
	for l := range strings.SplitSeq(src, "\n") {
		if _, def := definition(l); def || indentedCode(l) || atxHeading(l) || setextUnderline(l) || thematicBreak(l) {
			return false
		}
	}
	return true
}

// TestHTML: an HTML block's lines print as they come, unstyled and
// unwrapped, to its end: a blank line, or its own end condition.
func TestHTML(t *testing.T) {
	got := print([]string{"<div class=\"x\">\n**not styled** and a long line here\n</div>\n\n<!-- a\n**not** -->\n*yes*\n"}, WithMarkup(DefaultMarkup()), WithWidth(10))
	want := "<div class=\"x\">\n**not styled** and a long line here\n</div>\n\n<!-- a\n**not** -->\n\x1b[3m*yes*\x1b[0m\n"
	if got != want {
		t.Errorf("%q\nwant\n%q", got, want)
	}
	// A block tag interrupts a paragraph; any other whole tag, kind 7, does
	// not, and the paragraph goes on (CommonMark §4.6).
	got = print([]string{"text\n<div>\n*a*\n\ntext\n<span>\n*a*\n"}, WithMarkup(DefaultMarkup()))
	want = "text\n<div>\n*a*\n\ntext\n<span>\n\x1b[3m*a*\x1b[0m\n"
	if got != want {
		t.Errorf("%q\nwant\n%q", got, want)
	}
}

// TestParagraphContinues: a line indented four columns after a paragraph's
// line continues it, and is not code: an underline after it makes it a
// heading (CommonMark §4.4).
func TestParagraphContinues(t *testing.T) {
	got := print([]string{"Foo\n    bar\n===\n"}, WithMarkup(DefaultMarkup()))
	if want := "Foo\n\x1b[1m    bar\x1b[0m\n\x1b[1m===\x1b[0m\n"; got != want {
		t.Errorf("%q; want %q", got, want)
	}
}
