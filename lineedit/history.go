package lineedit

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// History is the lines a Composer recalls with up and down, and adds each
// submission to. It is x/term's History, so its and gobble's plug in
// (docs/decisions/0023-MADR-line-mode-components.md D6, A3.2).
type History interface {
	// Add will be called to add a new, most recent entry to the history.
	// It is allowed to drop any entry, including the entry being added
	// (e.g., if it's deemed an invalid entry), the least-recent entry
	// (e.g., to keep the history bounded), or any other entry.
	Add(entry string)

	// Len returns the number of entries in the history.
	Len() int

	// At returns an entry from the history. Index 0 is the most-recently
	// added entry and index Len()-1 is the least-recently added entry. If
	// index is < 0 or >= Len(), it panics.
	At(idx int) string
}

// DefaultHistorySize is the most entries a MemHistory keeps, as gobble's
// and liner's do.
const DefaultHistorySize = 1000

// MemHistory is a History in memory: an empty line and one equal to the
// most recent are dropped, as bash's ignoredups and zsh's HIST_IGNORE_DUPS
// drop them, and the oldest go past Max. WriteTo and ReadFrom are its file
// form, gobble's and fish's: one entry per line, a line break in an entry
// written `\n` and a backslash `\\` (A3 item 12). The file, its path and
// its mode are the program's. The zero MemHistory is ready to use. It is
// not safe for concurrent use.
type MemHistory struct {
	// Max is the most entries kept; zero or less keeps DefaultHistorySize.
	Max     int
	entries []string // oldest first
}

// Add appends entry, unless it is empty or equal to the most recent entry,
// and drops the oldest past Max.
func (h *MemHistory) Add(entry string) {
	if entry == "" || (len(h.entries) > 0 && h.entries[len(h.entries)-1] == entry) {
		return
	}
	h.entries = append(h.entries, entry)
	if n := h.max(); len(h.entries) > n {
		h.entries = append([]string(nil), h.entries[len(h.entries)-n:]...)
	}
}

func (h *MemHistory) max() int {
	if h.Max > 0 {
		return h.Max
	}
	return DefaultHistorySize
}

// Len is the number of entries.
func (h *MemHistory) Len() int { return len(h.entries) }

// At is entry idx, 0 the most recent. It panics when idx is out of range,
// as History allows.
func (h *MemHistory) At(idx int) string {
	if idx < 0 || idx >= len(h.entries) {
		panic(fmt.Sprintf("lineedit: history index %d out of range [0,%d)", idx, len(h.entries)))
	}
	return h.entries[len(h.entries)-1-idx]
}

// escaper writes an entry as one line of the file.
var escaper = strings.NewReplacer(`\`, `\\`, "\n", `\n`)

// WriteTo writes the entries, oldest first, one per line, a line break in
// one as `\n` and a backslash as `\\`.
func (h *MemHistory) WriteTo(w io.Writer) (int64, error) {
	var b strings.Builder
	for _, e := range h.entries {
		b.WriteString(escaper.Replace(e))
		b.WriteByte('\n')
	}
	n, err := io.WriteString(w, b.String())
	return int64(n), err
}

// ReadFrom reads entries as WriteTo writes them, oldest first, and adds
// each as Add does: an empty line is skipped, a line's \r ending dropped,
// and a backslash before anything but n and a backslash kept as it is.
func (h *MemHistory) ReadFrom(r io.Reader) (int64, error) {
	br := bufio.NewReader(r)
	var n int64
	for {
		line, err := br.ReadString('\n')
		n += int64(len(line))
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		h.Add(unescape(line))
		switch {
		case errors.Is(err, io.EOF):
			return n, nil
		case err != nil:
			return n, err
		}
	}
}

// unescape reads one line of the file as its entry.
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case '\\':
			b.WriteByte('\\')
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// Completer completes the line at the cursor, as x/term's
// AutoCompleteCallback does for tab: given the line and the cursor's byte
// offset, it returns the new line and cursor, and ok; with ok false the
// line is left as it is.
type Completer interface {
	Complete(line string, pos int) (newLine string, newPos int, ok bool)
}

// CompleterFunc is a function used as a Completer.
type CompleterFunc func(line string, pos int) (newLine string, newPos int, ok bool)

// Complete calls f.
func (f CompleterFunc) Complete(line string, pos int) (string, int, bool) { return f(line, pos) }
