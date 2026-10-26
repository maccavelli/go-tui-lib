package msgcat

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// MessageID names one message: its package's name, a dot, and the
// message's own name, such as "palette.no-results".
type MessageID string

// Catalog gives a message's text in the program's words. It reports false
// to keep the package's default.
type Catalog interface {
	Message(id MessageID) (string, bool)
}

// CatalogFunc is a function used as a Catalog.
type CatalogFunc func(id MessageID) (string, bool)

// Message calls f.
func (f CatalogFunc) Message(id MessageID) (string, bool) { return f(id) }

// Table is a Catalog read from a map, as a program loads one from a file.
// A nil Table keeps every default.
type Table map[MessageID]string

// Message is t's text for id, if it has one.
func (t Table) Message(id MessageID) (string, bool) {
	s, ok := t[id]
	return s, ok
}

// Format is c's text for id, or def when c is nil or keeps the default,
// with args placed by fmt.Sprintf. The text is a format either way, so a
// literal percent sign is "%%".
func Format(c Catalog, id MessageID, def string, args ...any) string {
	text := def
	if c != nil {
		if s, ok := c.Message(id); ok {
			text = s
		}
	}
	return fmt.Sprintf(text, args...)
}

// Check reports each message of defaults whose text in c has other verbs
// than its default: an argument dropped, added, or formatted by another
// verb. A translation may reorder its arguments with explicit indexes. A
// message c keeps the default of is not checked. The errors are joined, in
// the order of the IDs.
func Check(defaults Table, c Catalog) error {
	var errs []error
	for _, id := range slices.Sorted(maps.Keys(defaults)) {
		text, ok := c.Message(id)
		if !ok {
			continue
		}
		want, got := verbs(defaults[id]), verbs(text)
		if !slices.Equal(want, got) {
			errs = append(errs, fmt.Errorf("msgcat: %s: verbs %s, where the default has %s", id, show(got), show(want)))
		}
	}
	return errors.Join(errs...)
}

// verb is one argument a format reads: its number, from 1, and the verb
// letter that formats it; a width or precision read from an argument is
// the verb '*'.
type verb struct {
	arg  int
	kind rune
}

// verbs is the arguments s reads as a format, sorted by argument and
// verb, as fmt numbers them: in order, or by an explicit [n] index, which
// sets the next argument's number.
func verbs(s string) []verb {
	var out []verb
	next := 1
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		if rs[i] != '%' {
			continue
		}
		i++
		if i < len(rs) && rs[i] == '%' {
			continue
		}
		// One directive: flags, an index, a width, a precision, the verb.
		for ; i < len(rs); i++ {
			r := rs[i]
			if r == '[' {
				end := slices.Index(rs[i:], ']')
				if end < 0 {
					out = append(out, verb{arg: next, kind: '?'})
					break
				}
				var n int
				if _, err := fmt.Sscanf(string(rs[i+1:i+end]), "%d", &n); err == nil {
					next = n
				}
				i += end
				continue
			}
			if r == '*' {
				out = append(out, verb{arg: next, kind: '*'})
				next++
				continue
			}
			if strings.ContainsRune("+-# .0123456789", r) {
				continue
			}
			out = append(out, verb{arg: next, kind: r})
			next++
			break
		}
	}
	slices.SortFunc(out, func(a, b verb) int {
		if a.arg != b.arg {
			return a.arg - b.arg
		}
		return int(a.kind - b.kind)
	})
	return out
}

// show is vs as fmt would write them: "%[1]d %[2]s".
func show(vs []verb) string {
	if len(vs) == 0 {
		return "none"
	}
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = fmt.Sprintf("%%[%d]%c", v.arg, v.kind)
	}
	return strings.Join(parts, " ")
}
