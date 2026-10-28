package stream

import "strings"

// The block scanner reads one complete line at a time, as CommonMark 0.31.2
// reads a line's start, to the depth the Printer needs: it tells a fence's
// opening and closing lines, indented code, ATX headings, setext
// underlines, thematic breaks, list items, block quotes, HTML blocks,
// reference definitions and table rows from a paragraph's lines. It is
// unexported until 0009's Doc is built on it
// (docs/decisions/0023-MADR-line-mode-components.md A6 item 2).

// indentOf is the columns of s's leading spaces and tabs, a tab to the
// next multiple of 4, and the bytes they take.
func indentOf(s string) (cols, n int) {
	for n < len(s) {
		switch s[n] {
		case ' ':
			cols++
		case '\t':
			cols += 4 - cols%4
		default:
			return cols, n
		}
		n++
	}
	return cols, n
}

// blank reports a line of nothing but spaces and tabs.
func blank(s string) bool { return strings.TrimLeft(s, " \t") == "" }

// fenceRun is the run of three or more backticks or tildes that opens a
// fence on line, after at most three spaces of indentation, or "": a
// backtick fence's info string holds no backtick (CommonMark §4.5).
func fenceRun(line string) string {
	cols, n := indentOf(line)
	t := line[n:]
	if cols > 3 || t == "" || (t[0] != '`' && t[0] != '~') {
		return ""
	}
	run := t[:len(t)-len(strings.TrimLeft(t, t[:1]))]
	if len(run) < 3 || (run[0] == '`' && strings.Contains(t[len(run):], "`")) {
		return ""
	}
	return run
}

// closesFence reports whether line closes the fence opened by run: the
// same character, at least as many, after at most three spaces, and
// nothing but spaces after it (CommonMark §4.5).
func closesFence(line, run string) bool {
	cols, n := indentOf(line)
	t := strings.TrimRight(line[n:], " \t")
	return cols <= 3 && len(t) >= len(run) && strings.Trim(t, run[:1]) == ""
}

// indentedCode reports a line indented four columns or more, which is
// code unless it continues a paragraph (§4.4).
func indentedCode(line string) bool {
	cols, _ := indentOf(line)
	return cols >= 4 && !blank(line)
}

// atxHeading reports a line that is an ATX heading: at most three spaces,
// one to six #, then a space, a tab or the line's end (§4.2).
func atxHeading(line string) bool {
	cols, n := indentOf(line)
	if cols > 3 {
		return false
	}
	t := line[n:]
	h := len(t) - len(strings.TrimLeft(t, "#"))
	return h >= 1 && h <= 6 && (len(t) == h || t[h] == ' ' || t[h] == '\t')
}

// setextUnderline reports a line that underlines the paragraph line before
// it: at most three spaces, a run of = or of -, then only spaces (§4.3).
func setextUnderline(line string) bool {
	cols, n := indentOf(line)
	if cols > 3 {
		return false
	}
	t := strings.TrimRight(line[n:], " \t")
	return t != "" && (strings.Trim(t, "=") == "" || strings.Trim(t, "-") == "")
}

// thematicBreak reports a line of three or more -, * or _, alike, with
// spaces or tabs between them, after at most three spaces (§4.1).
func thematicBreak(line string) bool {
	cols, n := indentOf(line)
	if cols > 3 {
		return false
	}
	var c byte
	count := 0
	for _, r := range []byte(line[n:]) {
		switch {
		case r == ' ' || r == '\t':
		case c == 0 && (r == '-' || r == '*' || r == '_'):
			c, count = r, 1
		case r == c:
			count++
		default:
			return false
		}
	}
	return count >= 3
}

// markerLen is the length of the list marker (-, *, +, 1. or 1)) or the
// quote marker (>) s starts with, or 0 (gobble-cli's render/mdstyle.go).
func markerLen(s string) int {
	switch {
	case s == "":
		return 0
	case s[0] == '>':
		return 1
	case (s[0] == '-' || s[0] == '*' || s[0] == '+') && (len(s) == 1 || s[1] == ' ' || s[1] == '\t'):
		return 1
	}
	d := len(s) - len(strings.TrimLeft(s, "0123456789"))
	if d >= 1 && d <= 9 && len(s) > d && (s[d] == '.' || s[d] == ')') && (len(s) == d+1 || s[d+1] == ' ' || s[d+1] == '\t') {
		return d + 1
	}
	return 0
}

// container reports a line that starts a list item or a block quote,
// after at most three spaces.
func container(line string) bool {
	cols, n := indentOf(line)
	return cols <= 3 && markerLen(line[n:]) > 0
}

// htmlEnd is the end condition of the HTML block kinds 1 to 5, by kind:
// the line that holds it ends the block. Kinds 6 and 7 end at a blank
// line (§4.6).
var htmlEnd = [...][]string{
	1: {"</script>", "</pre>", "</style>", "</textarea>"},
	2: {"-->"},
	3: {"?>"},
	4: {">"},
	5: {"]]>"},
}

// htmlBlockNames are the tags that start an HTML block of kind 6 (§4.6).
var htmlBlockNames = strings.Fields(`address article aside base basefont blockquote body caption center col
colgroup dd details dialog dir div dl dt fieldset figcaption figure footer form frame frameset h1 h2 h3
h4 h5 h6 head header hr html iframe legend li link main menu menuitem nav noframes ol optgroup option p
param search section summary table tbody td tfoot th thead title tr track ul`)

// htmlStart is the kind of HTML block line starts, 1 to 7, or 0, after at
// most three spaces (CommonMark §4.6). Kind 7, a whole tag alone on its
// line, cannot interrupt a paragraph, so cont rules it out.
func htmlStart(line string, cont bool) int {
	cols, n := indentOf(line)
	t := line[n:]
	if cols > 3 || len(t) < 2 || t[0] != '<' {
		return 0
	}
	lower := strings.ToLower(t)
	for _, name := range []string{"script", "pre", "style", "textarea"} {
		if rest, ok := strings.CutPrefix(lower, "<"+name); ok && (rest == "" || rest[0] == ' ' || rest[0] == '\t' || rest[0] == '>') {
			return 1
		}
	}
	switch {
	case strings.HasPrefix(t, "<!--"):
		return 2
	case strings.HasPrefix(t, "<?"):
		return 3
	case strings.HasPrefix(t, "<![CDATA["):
		return 5
	case len(t) > 2 && t[1] == '!' && letter(t[2]):
		return 4
	}
	name := strings.TrimPrefix(lower[1:], "/")
	for _, b := range htmlBlockNames {
		if rest, ok := strings.CutPrefix(name, b); ok && (rest == "" || rest[0] == ' ' || rest[0] == '\t' || rest[0] == '>' || strings.HasPrefix(rest, "/>")) {
			return 6
		}
	}
	if !cont && wholeTag(strings.TrimRight(t, " \t")) {
		return 7
	}
	return 0
}

// wholeTag reports s, one open or closing tag and nothing else: a name of
// letters, digits and hyphens, then attributes, quoted values among them,
// to a > (CommonMark §6.6, simplified to the tag's shape).
func wholeTag(s string) bool {
	if len(s) < 3 || s[0] != '<' || s[len(s)-1] != '>' {
		return false
	}
	body := strings.TrimSuffix(strings.TrimSuffix(s[1:len(s)-1], "/"), " ")
	body = strings.TrimPrefix(body, "/")
	if body == "" || !letter(body[0]) {
		return false
	}
	var quote byte
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '<' || c == '>':
			return false
		}
	}
	return quote == 0
}

// letter reports an ASCII letter, which a tag's name starts with.
func letter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// htmlEnds reports whether line ends an HTML block of kind k.
func htmlEnds(k int, line string) bool {
	if k >= 6 {
		return blank(line)
	}
	lower := strings.ToLower(line)
	for _, e := range htmlEnd[k] {
		if strings.Contains(lower, e) {
			return true
		}
	}
	return false
}

// definition reads a link reference definition, [label]: destination,
// after at most three spaces (§4.7), and gives its label.
func definition(line string) (lbl string, ok bool) {
	cols, n := indentOf(line)
	t := line[n:]
	if cols > 3 || !strings.HasPrefix(t, "[") {
		return "", false
	}
	end := closingBracket(t, 0)
	if end < 0 || !strings.HasPrefix(t[end+1:], ":") || strings.TrimSpace(t[end+2:]) == "" {
		return "", false
	}
	return label(t[1:end]), true
}

// startsBlock reports a line that begins a block other than a paragraph,
// and so ends a table (GFM §4.10) or a paragraph's run of lines.
func startsBlock(line string) bool {
	return fenceRun(line) != "" || atxHeading(line) || thematicBreak(line) || container(line) || htmlStart(line, true) > 0
}
