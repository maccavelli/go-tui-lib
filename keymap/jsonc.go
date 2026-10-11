package keymap

import "errors"

// bom is the UTF-8 byte order mark, which an editor may write first.
const bom = "\xef\xbb\xbf"

// errComment is an unterminated block comment.
var errComment = errors.New("a /* comment with no */")

// blank is src with JSONC's extras replaced by spaces, as JSON reads them:
// line and block comments, a trailing comma before "]" or "}", and a
// leading byte order mark. Line breaks inside a comment are kept, so every
// byte keeps its offset, and its line and column (MADR §8). Strings are
// copied unchanged, so a "//" in a string is not a comment. An
// unterminated block comment is an error at the offset where it begins.
func blank(src []byte) ([]byte, int, error) {
	out := []byte(string(src))
	space := func(from, to int) {
		for i := from; i < to; i++ {
			if out[i] != '\n' && out[i] != '\r' {
				out[i] = ' '
			}
		}
	}
	if len(out) >= len(bom) && string(out[:len(bom)]) == bom {
		space(0, len(bom))
	}
	comma := -1 // the last comma, while only space and comments follow it
	for i := 0; i < len(out); i++ {
		switch c := out[i]; {
		case c == '"':
			comma = -1
			for i++; i < len(out) && out[i] != '"'; i++ {
				if out[i] == '\\' {
					i++
				}
			}
		case c == '/' && i+1 < len(out) && out[i+1] == '/':
			end := i
			for end < len(out) && out[end] != '\n' && out[end] != '\r' {
				end++
			}
			space(i, end)
			i = end - 1
		case c == '/' && i+1 < len(out) && out[i+1] == '*':
			end := i + 2
			for end+1 < len(out) && (out[end] != '*' || out[end+1] != '/') {
				end++
			}
			if end+1 >= len(out) {
				return nil, i, errComment
			}
			space(i, end+2)
			i = end + 1
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
		case c == ',':
			comma = i
		case (c == ']' || c == '}') && comma >= 0:
			out[comma] = ' '
			comma = -1
		default:
			comma = -1
		}
	}
	return out, 0, nil
}
