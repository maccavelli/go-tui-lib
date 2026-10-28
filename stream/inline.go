package stream

import "strings"

// inline styles one line's text, its block marker already taken off:
// code spans in Markup's Code style, backticks kept; then, in the text
// between them, links and emphasis, gobble-cli's styling (render/mdstyle.go)
// with links added (0023-MADR A6 item 1). Every byte of the line is kept,
// and a construct still open at the line's end is left as it is, so no
// line is styled again by a later one (A3 item 14).
func inline(m Markup, line string, defs map[string]bool) string {
	var b strings.Builder
	var text strings.Builder // text outside code spans, not yet styled
	for line != "" {
		i := strings.IndexByte(line, '`')
		if i < 0 {
			text.WriteString(line)
			break
		}
		text.WriteString(line[:i])
		rest := line[i:]
		run := rest[:len(rest)-len(strings.TrimLeft(rest, "`"))]
		end := closingRun(rest[len(run):], len(run))
		if end < 0 {
			text.WriteString(run)
			line = rest[len(run):]
			continue
		}
		b.WriteString(links(m, text.String(), defs))
		text.Reset()
		span := rest[:len(run)+end+len(run)]
		b.WriteString(m.Code.Apply(span))
		line = rest[len(span):]
	}
	b.WriteString(links(m, text.String(), defs))
	return b.String()
}

// closingRun is the offset in s of a backtick run of exactly n, or -1.
func closingRun(s string, n int) int {
	for off := 0; off < len(s); {
		i := strings.IndexByte(s[off:], '`')
		if i < 0 {
			return -1
		}
		start := off + i
		end := start
		for end < len(s) && s[end] == '`' {
			end++
		}
		if end-start == n {
			return start
		}
		off = end
	}
	return -1
}

// links styles the links in s, which holds no code span, and the emphasis
// between them: an inline link [text](dest), an image ![alt](src), a full
// reference [text][label] and a collapsed [label][] or shortcut [label]
// reference, the last three only when the label's definition was printed
// before. The text is in the Link style, its brackets kept; the
// destination or label in the URL style. A link's text holds no emphasis,
// so that one style's reset never ends another's early.
func links(m Markup, s string, defs map[string]bool) string {
	if !strings.Contains(s, "[") {
		return emphasis(m, s)
	}
	var b strings.Builder
	var text strings.Builder // text between links, not yet styled
	for i := 0; i < len(s); {
		if s[i] == '\\' && i+1 < len(s) {
			text.WriteString(s[i : i+2])
			i += 2
			continue
		}
		if s[i] != '[' {
			text.WriteByte(s[i])
			i++
			continue
		}
		bang := i > 0 && s[i-1] == '!' && (i < 2 || s[i-2] != '\\')
		link, after, ok := linkAt(m, s, i, defs)
		if !ok {
			text.WriteByte(s[i])
			i++
			continue
		}
		pending := text.String()
		if bang {
			pending = pending[:len(pending)-1]
			link = m.Link.Apply("!") + link
		}
		b.WriteString(emphasis(m, pending))
		text.Reset()
		b.WriteString(link)
		i = after
	}
	b.WriteString(emphasis(m, text.String()))
	return b.String()
}

// linkAt reads a link whose text's bracket is at s[i], and gives it
// styled, with the offset after it; ok is false for no link.
func linkAt(m Markup, s string, i int, defs map[string]bool) (styled string, after int, ok bool) {
	shut := closingBracket(s, i)
	if shut < 0 {
		return "", 0, false
	}
	text := s[i : shut+1]
	rest := s[shut+1:]
	switch {
	case strings.HasPrefix(rest, "("):
		if end := closingParen(rest); end > 0 {
			return m.Link.Apply(text) + m.URL.Apply(rest[:end+1]), shut + 1 + end + 1, true
		}
	case strings.HasPrefix(rest, "[]"):
		if defs[label(text[1:len(text)-1])] {
			return m.Link.Apply(text) + m.URL.Apply("[]"), shut + 3, true
		}
		return "", 0, false
	case strings.HasPrefix(rest, "["):
		if end := closingBracket(rest, 0); end > 0 && defs[label(rest[1:end])] {
			return m.Link.Apply(text) + m.URL.Apply(rest[:end+1]), shut + 1 + end + 1, true
		}
	}
	if defs[label(text[1:len(text)-1])] && !strings.HasPrefix(rest, "(") && !strings.HasPrefix(rest, "[") {
		return m.Link.Apply(text), shut + 1, true
	}
	return "", 0, false
}

// closingBracket is the offset of the ] that closes the [ at s[i], with no
// [ between them, a backslash escaping either; or -1.
func closingBracket(s string, i int) int {
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case '[':
			return -1
		case ']':
			if j == i+1 {
				return -1 // an empty text is no link
			}
			return j
		}
	}
	return -1
}

// closingParen is the offset of the ) that closes the ( s starts with,
// with no space or line break before it and parentheses balanced; or -1.
func closingParen(s string) int {
	depth := 0
	for j := 0; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case ' ', '\t':
			return -1
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

// label is a link label as CommonMark matches it: case folded, its white
// space runs one space, trimmed.
func label(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// emphasisStyle is the style of a delimiter run of n c's, and whether the
// run is one: three * or _ strong emphasis, two strong, one emphasis, and
// ~~ struck (gobble-cli's 0008-PLAN P8).
func emphasisStyle(m Markup, c byte, n int) (Style, bool) {
	switch {
	case c == '~':
		return m.Strike, n == 2
	case n == 1:
		return m.Emphasis, true
	case n == 2:
		return m.Strong, true
	case n == 3:
		return m.StrongEmphasis, true
	}
	return 0, false
}

// emphasis styles emphasis in s, which holds no code span and no link.
// Markers are kept and styled with the text. An opener is followed by a
// non-space, its closer, the same run, is preceded by one, and _ opens and
// closes only at a word boundary, so snake_case stays plain. Escaped and
// unclosed runs stay as they are, and spans are not nested: a nested reset
// would end the outer style early.
func emphasis(m Markup, s string) string {
	if !strings.ContainsAny(s, "*_~") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			b.WriteString(s[i : i+2])
			i += 2
			continue
		}
		if c != '*' && c != '_' && c != '~' {
			b.WriteByte(c)
			i++
			continue
		}
		n := runLen(s, i)
		style, known := emphasisStyle(m, c, n)
		if end := closer(s, i, n); known && opens(s, i, n) && end > 0 {
			b.WriteString(style.Apply(s[i : end+n]))
			i = end + n
			continue
		}
		b.WriteString(s[i : i+n])
		i += n
	}
	return b.String()
}

// runLen is the length of the run of s[i] starting at i.
func runLen(s string, i int) int {
	n := 1
	for i+n < len(s) && s[i+n] == s[i] {
		n++
	}
	return n
}

// opens reports whether the run s[i:i+n] can open emphasis.
func opens(s string, i, n int) bool {
	if i+n >= len(s) || isSpace(s[i+n]) {
		return false
	}
	return s[i] != '_' || i == 0 || !isWord(s[i-1])
}

// closer finds the run that closes s[i:i+n]: the same character, exactly n
// long, after a non-space, and for _ before a word boundary. It returns the
// closer's offset, or -1.
func closer(s string, i, n int) int {
	c := s[i]
	for j := i + n + 1; j < len(s); {
		if s[j] == '\\' {
			j += 2
			continue
		}
		if s[j] != c {
			j++
			continue
		}
		k := runLen(s, j)
		if k == n && !isSpace(s[j-1]) && (c != '_' || j+k == len(s) || !isWord(s[j+k])) {
			return j
		}
		j += k
	}
	return -1
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' }

// isWord reports a letter or digit, ASCII or the lead byte of a multibyte
// rune, which is taken as a letter.
func isWord(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c >= 0x80
}
