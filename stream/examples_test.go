package stream

// The specifications' examples the Printer is tested on, their Markdown
// only, by number (0023-MADR A6 item 3), written by a script from the
// specifications read on 2026-10-28.

// setextExamples are CommonMark 0.31.2's, section 4.3, Setext headings
// (spec.commonmark.org/0.31.2/spec.json).
var setextExamples = []struct {
	n  int
	md string
}{
	{80, "Foo *bar*\n=========\n\nFoo *bar*\n---------\n"},
	{81, "Foo *bar\nbaz*\n====\n"},
	{82, "  Foo *bar\nbaz*\t\n====\n"},
	{83, "Foo\n-------------------------\n\nFoo\n=\n"},
	{84, "   Foo\n---\n\n  Foo\n-----\n\n  Foo\n  ===\n"},
	{85, "    Foo\n    ---\n\n    Foo\n---\n"},
	{86, "Foo\n   ----      \n"},
	{87, "Foo\n    ---\n"},
	{88, "Foo\n= =\n\nFoo\n--- -\n"},
	{89, "Foo  \n-----\n"},
	{90, "Foo\\\n----\n"},
	{91, "`Foo\n----\n`\n\n<a title=\"a lot\n---\nof dashes\"/>\n"},
	{92, "> Foo\n---\n"},
	{93, "> foo\nbar\n===\n"},
	{94, "- Foo\n---\n"},
	{95, "Foo\nBar\n---\n"},
	{96, "---\nFoo\n---\nBar\n---\nBaz\n"},
	{97, "\n====\n"},
	{98, "---\n---\n"},
	{99, "- foo\n-----\n"},
	{100, "    foo\n---\n"},
	{101, "> foo\n-----\n"},
	{102, "\\> foo\n------\n"},
	{103, "Foo\n\nbar\n---\nbaz\n"},
	{104, "Foo\nbar\n\n---\n\nbaz\n"},
	{105, "Foo\nbar\n* * *\nbaz\n"},
	{106, "Foo\nbar\n\\---\nbaz\n"},
}

// tableExamples are GFM 0.29-gfm's, section 4.10, Tables (extension)
// (cmark-gfm 0.29.0.gfm.13, test/spec.txt).
var tableExamples = []struct {
	n  int
	md string
}{
	{198, "| foo | bar |\n| --- | --- |\n| baz | bim |\n"},
	{199, "| abc | defghi |\n:-: | -----------:\nbar | baz\n"},
	{200, "| f\\|oo  |\n| ------ |\n| b `\\|` az |\n| b **\\|** im |\n"},
	{201, "| abc | def |\n| --- | --- |\n| bar | baz |\n> bar\n"},
	{202, "| abc | def |\n| --- | --- |\n| bar | baz |\nbar\n\nbar\n"},
	{203, "| abc | def |\n| --- |\n| bar |\n"},
	{204, "| abc | def |\n| --- | --- |\n| bar |\n| bar | baz | boo |\n"},
	{205, "| abc | def |\n| --- | --- |\n"},
}
