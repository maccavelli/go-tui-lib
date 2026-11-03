package stream

import (
	"strings"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
)

// BlockKind is what a line is to the block it is in.
type BlockKind uint8

// The kinds a Scanner gives a line.
const (
	// BlockBlank is a blank line outside a fence and an HTML block that a
	// blank line does not end. The zero value.
	BlockBlank BlockKind = iota
	// BlockParagraph is a paragraph's line, which the next line can make a
	// setext heading's text or a table's header.
	BlockParagraph
	// BlockContinuation is a list item's or a quote's text, continued
	// lazily on a line of its own (CommonMark §5.1).
	BlockContinuation
	// BlockHeading is an ATX heading.
	BlockHeading
	// BlockSetext is a setext heading's underline: the line before it is
	// the heading's text.
	BlockSetext
	// BlockBreak is a thematic break.
	BlockBreak
	// BlockFence is a fence's opening line.
	BlockFence
	// BlockCode is a line inside a fence.
	BlockCode
	// BlockFenceEnd is a fence's closing line.
	BlockFenceEnd
	// BlockIndented is indented code.
	BlockIndented
	// BlockHTML is an HTML block's line.
	BlockHTML
	// BlockList is a list item's first line.
	BlockList
	// BlockQuote is a block quote's line.
	BlockQuote
	// BlockTable is a table's delimiter row: the line before it is the
	// table's header.
	BlockTable
	// BlockTableRow is a table's row after its delimiter row.
	BlockTableRow
	// BlockDefinition is a link reference or footnote definition.
	BlockDefinition
)

var blockKindNames = enum.Names[BlockKind]{
	Pkg: "stream", Type: "BlockKind",
	Tokens: []string{
		"blank", "paragraph", "continuation", "heading", "setext", "break", "fence", "code",
		"fence-end", "indented", "html", "list", "quote", "table", "table-row", "definition",
	},
}

// String is the kind's token.
func (k BlockKind) String() string { return blockKindNames.String(k) }

// MarshalText is the kind's token. A kind with no token is an error.
func (k BlockKind) MarshalText() ([]byte, error) { return blockKindNames.Marshal(k) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (k *BlockKind) UnmarshalText(b []byte) error { return blockKindNames.Unmarshal(b, k) }

// Block is what a Scanner tells of one line.
type Block struct {
	// Kind is what the line is.
	Kind BlockKind
	// Cut reports that a cut falls before this line: blank lines came
	// before it, outside a fence and an HTML block of kinds 1 to 5, and it
	// continues no open list item and no indented code
	// (docs/decisions/0009-MADR-streaming-content-engine.md §4). It is
	// told on the line after the blank ones, since only that line decides
	// it.
	Cut bool
	// Split reports that a relaxed cut may fall before this line, where a
	// block can be split in two of its kind: inside a paragraph, a fence or
	// a quote, or before a list item at its list's own indent. A table and
	// an HTML block are never split.
	Split bool
	// Fence is, for a line inside a fence, the fence's opening line, which
	// a segment that begins inside it opens with again; "" otherwise.
	Fence string
	// Label is a definition's label.
	Label string
}

// listItem is an open list item: its marker's kind, a bullet's character
// or an ordered marker's delimiter, the columns before its marker, and
// the columns its content begins at.
type listItem struct {
	marker          byte
	indent, content int
}

// Scanner reads a Markdown document one complete line at a time, as
// CommonMark 0.31.2 reads a line's start, and tells each line's Block: the
// state a cut depends on, the open fence, HTML block, table, paragraph,
// list item and indented code, is kept from line to line, so each line is
// read once and the work is the line's
// (docs/decisions/0009-MADR-streaming-content-engine.md §4, A6 item 5;
// docs/decisions/0023-MADR-line-mode-components.md A6 item 2). The
// Printer and the Doc are built on it. Its zero value is a document's
// start, and it is used on one goroutine.
type Scanner struct {
	fence     string // the open fence's marker run, or ""
	fenceLine string // the open fence's opening line
	html      int    // the open HTML block's kind, 1 to 7, or 0
	table     bool   // a table is open
	paragraph bool   // the last line was a paragraph's
	prev      string // the last line, while it was a paragraph's
	lazy      bool   // a paragraph's line may continue the list item's or quote's text

	started  bool      // a line not blank has come
	blanks   bool      // blank lines came since the last line not blank
	last     BlockKind // the last line's kind
	list     listItem
	inList   bool // a list item is open
	indented bool // the last line not blank was indented code
}

// Scan reads line, one complete line without its line break, and tells
// its Block.
func (s *Scanner) Scan(line string) Block {
	kind, label := s.kind(line)
	b := Block{Kind: kind, Label: label}
	if kind == BlockCode {
		b.Fence = s.fenceLine
	}
	if kind == BlockBlank {
		s.blanks = s.blanks || s.started
		s.last = kind
		return b
	}
	b.Split = s.split(line, kind)
	if s.blanks {
		s.blanks = false
		if !s.continues(line, kind) {
			b.Cut, b.Split = true, false
			s.inList, s.indented = false, false
		}
	}
	switch {
	case kind == BlockList:
		s.list, s.inList = item(line), true
	case s.inList && kind != BlockContinuation && indent(line) < s.list.content:
		s.inList = false
	}
	s.indented = kind == BlockIndented
	s.started, s.last = true, kind
	return b
}

// kind is line's kind, and a definition's label, and keeps the state the
// next line's kind depends on.
func (s *Scanner) kind(l string) (BlockKind, string) {
	if s.fence != "" {
		if closesFence(l, s.fence) {
			s.fence, s.fenceLine = "", ""
			return BlockFenceEnd, ""
		}
		return BlockCode, ""
	}
	if s.html > 0 {
		end := htmlEnds(s.html, l)
		if !end || s.html < 6 {
			if end {
				s.html = 0
			}
			return BlockHTML, ""
		}
		s.html = 0 // the blank line is the block's end, and a blank line itself
	}
	if s.table {
		if !blank(l) && !startsBlock(l) {
			return BlockTableRow, ""
		}
		s.table = false
	}
	if s.paragraph {
		switch {
		case setextUnderline(l):
			s.paragraph = false
			return BlockSetext, ""
		case tableStart(s.prev, l):
			s.paragraph, s.table = false, true
			return BlockTable, ""
		}
	}
	cont, lazy := s.paragraph, s.lazy
	s.paragraph, s.lazy = false, false
	if lazy && !blank(l) && !startsBlock(l) && !indentedCode(l) {
		s.lazy = true // a lazy continuation, never a setext heading's text (CommonMark §4.3)
		return BlockContinuation, ""
	}
	switch {
	case blank(l):
		return BlockBlank, ""
	case fenceRun(l) != "":
		s.fence, s.fenceLine = fenceRun(l), l
		return BlockFence, ""
	case indentedCode(l) && !cont:
		return BlockIndented, ""
	case atxHeading(l):
		return BlockHeading, ""
	case thematicBreak(l):
		return BlockBreak, ""
	case htmlStart(l, cont) > 0:
		k := htmlStart(l, cont)
		if !htmlEnds(k, l) || k >= 6 {
			s.html = k
		}
		return BlockHTML, ""
	case container(l):
		s.lazy = !blank(strings.TrimLeft(containerText(l), ">"))
		if _, n := indentOf(l); l[n] == '>' {
			return BlockQuote, ""
		}
		return BlockList, ""
	}
	if lbl, ok := definition(l); ok && !cont {
		return BlockDefinition, lbl
	}
	s.paragraph, s.prev = true, l
	return BlockParagraph, ""
}

// split reports whether a relaxed cut may fall before line, of kind: a
// paragraph's line after another, a fence's line after its opening or
// another, a quote's line after another, or a list item of the open
// item's list at its indent.
func (s *Scanner) split(line string, kind BlockKind) bool {
	switch kind {
	case BlockParagraph:
		return s.last == BlockParagraph
	case BlockCode:
		return s.last == BlockFence || s.last == BlockCode
	case BlockQuote:
		return s.last == BlockQuote
	case BlockList:
		it := item(line)
		return s.inList && it.marker == s.list.marker && it.indent == s.list.indent
	}
	return false
}

// continues reports whether line, of kind, after blank lines, continues
// what was open before them: indented code, or a list item, by a marker
// of its list or by indentation to its content.
func (s *Scanner) continues(line string, kind BlockKind) bool {
	switch {
	case s.indented && kind == BlockIndented:
		return true
	case !s.inList:
		return false
	case indent(line) >= s.list.content:
		return true
	}
	return kind == BlockList && item(line).marker == s.list.marker && indent(line) < s.list.content
}

// indent is the columns before line's first character.
func indent(line string) int {
	cols, _ := indentOf(line)
	return cols
}

// item is the list item line opens: its marker's kind, and the columns
// before its marker and before its content.
func item(line string) listItem {
	cols, n := indentOf(line)
	body := line[n:]
	k := markerLen(body)
	it := listItem{marker: body[k-1], indent: cols, content: cols + k + 1} // a bullet, or an ordered marker's delimiter
	if rest := body[k:]; rest != "" {
		pad, _ := indentOf(rest)
		if pad >= 1 && pad <= 4 {
			it.content = cols + k + pad
		}
	}
	return it
}
