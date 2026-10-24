package keymap

import (
	"bytes"
	"cmp"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"iter"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// LoadError is an error at one place in a keybindings file.
type LoadError struct {
	File      string           // the name LoadVSCode was given
	Line, Col int              // from 1, the column in bytes, as go/token counts; 0 when not known
	Pointer   jsontext.Pointer // RFC 6901: the element or member at fault, such as "/3/key"
	Err       error
}

// Error is "keymap: file:line:col (pointer): err", as much of the place as
// is known.
func (e *LoadError) Error() string {
	var b strings.Builder
	b.WriteString("keymap: ")
	b.WriteString(e.File)
	if e.Line > 0 {
		if e.File != "" {
			b.WriteByte(':')
		}
		fmt.Fprintf(&b, "%d:%d", e.Line, e.Col)
	}
	if e.Pointer != "" {
		fmt.Fprintf(&b, " (%s)", e.Pointer)
	}
	if b.Len() > len("keymap: ") {
		b.WriteString(": ")
	}
	b.WriteString(e.Err.Error())
	return b.String()
}

// Unwrap is the error at that place: a notation error wrapping
// ErrNotation, a when.SyntaxError, a jsontext.SyntacticError, and others.
func (e *LoadError) Unwrap() error { return e.Err }

// LoadVSCode reads a keybindings file in VS Code's format (MADR §8): an
// array of {"key", "command", "when", "args"} objects, and this package's
// one extra member, "context", the binding's scope, "global" when absent.
// A "-" before the command removes a binding an earlier layer made: the
// keys given, or with no "key", every key of that command in that scope.
// Comments and trailing commas are accepted. <leader> stays a placeholder
// until New expands it.
//
// Each binding's Location is the user layer, the file, and the element's
// pointer, line and column. A binding with any error is left out, and the
// others are returned with every error, joined, each a *LoadError. An
// unknown member is an error, and so is a command ids rejects, when ids is
// not nil. A syntax error ends the file there. A file with no value, only
// space or comments, holds no bindings.
func LoadVSCode(r io.Reader, file string, ids func(command.ID) bool) ([]Binding, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, &LoadError{File: file, Err: err}
	}
	l := &loader{src: src, file: file, ids: ids}
	out := l.load()
	return out, errors.Join(l.errs...)
}

// loader is LoadVSCode's state.
type loader struct {
	src   []byte
	file  string
	ids   func(command.ID) bool
	errs  []error
	lines []int // each line's first byte offset, made on first use
}

// fail records err at a byte offset of the file.
func (l *loader) fail(off int, p jsontext.Pointer, err error) {
	line, col := l.position(off)
	l.errs = append(l.errs, &LoadError{File: l.file, Line: line, Col: col, Pointer: p, Err: err})
}

// syntax records a decoder's error, at the offset it names.
func (l *loader) syntax(err error) {
	if se, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		l.fail(int(se.ByteOffset), se.JSONPointer, err)
		return
	}
	l.fail(len(l.src), "", err)
}

// position is the line and column of a byte offset, from 1, the column in
// bytes.
func (l *loader) position(off int) (line, col int) {
	if l.lines == nil {
		l.lines = []int{0}
		for i, c := range l.src {
			if c == '\n' {
				l.lines = append(l.lines, i+1)
			}
		}
	}
	off = min(max(off, 0), len(l.src))
	line, _ = slices.BinarySearch(l.lines, off+1) // the lines starting at or before off
	return line, off - l.lines[line-1] + 1
}

func (l *loader) load() []Binding {
	text, at, err := blank(l.src)
	if err != nil {
		l.fail(at, "", err)
		return nil
	}
	// Duplicate names are allowed here and refused per binding, so that one
	// does not end the file.
	d := jsontext.NewDecoder(bytes.NewReader(text), jsontext.AllowDuplicateNames(true))
	if d.PeekKind() != '[' {
		v, err := d.ReadValue()
		switch {
		case errors.Is(err, io.EOF):
		case err != nil:
			l.syntax(err)
		default:
			l.fail(int(d.InputOffset())-len(v), "", fmt.Errorf("a keybindings file is an array, not %s", kindName(v.Kind())))
		}
		return nil
	}
	if _, err := d.ReadToken(); err != nil {
		l.syntax(err)
		return nil
	}
	var out []Binding
	for i := 0; d.PeekKind() != ']'; i++ {
		v, err := d.ReadValue()
		if err != nil {
			l.syntax(err)
			return out
		}
		if b, ok := l.binding(v, int(d.InputOffset())-len(v), i); ok {
			out = append(out, b)
		}
	}
	if _, err := d.ReadToken(); err != nil {
		l.syntax(err)
		return out
	}
	if _, err := d.ReadToken(); !errors.Is(err, io.EOF) {
		l.syntax(err)
	}
	return out
}

// member is one member of a binding, with its place.
type member struct {
	v   jsontext.Value
	off int // the value's byte offset in the file
	p   jsontext.Pointer
}

// The members a binding may have.
const (
	memberKey     = "key"
	memberCommand = "command"
	memberWhen    = "when"
	memberArgs    = "args"
	memberContext = "context"
)

// fields are the members a binding may have, in the order they are read.
var fields = []string{memberKey, memberCommand, memberWhen, memberArgs, memberContext}

// binding reads the i-th element, v, at byte offset base.
func (l *loader) binding(v jsontext.Value, base, i int) (Binding, bool) {
	at := jsontext.Pointer("/" + strconv.Itoa(i))
	if v.Kind() != '{' {
		l.fail(base, at, fmt.Errorf("a binding is an object, not %s", kindName(v.Kind())))
		return Binding{}, false
	}
	n := len(l.errs)
	line, col := l.position(base)
	b := Binding{Location: Location{Layer: UserLayer, File: l.file, Pointer: at, Line: line, Col: col}}
	d := jsontext.NewDecoder(bytes.NewReader(v), jsontext.AllowDuplicateNames(true))
	ms := map[string]member{}
	if _, err := d.ReadToken(); err != nil { // v is one valid object, so not met
		l.fail(base, at, err)
		return Binding{}, false
	}
	for d.PeekKind() != '}' {
		// A value read is valid only until the next read.
		raw, err := d.ReadValue()
		name, nameAt, p := unquote(raw), base+int(d.InputOffset())-len(raw), at+d.StackPointer()
		val, err2 := d.ReadValue()
		if err = cmp.Or(err, err2); err != nil {
			l.fail(base, at, err)
			return Binding{}, false
		}
		m := member{v: slices.Clone(val), off: base + int(d.InputOffset()) - len(val), p: p}
		switch _, seen := ms[name]; {
		case !slices.Contains(fields, name):
			l.fail(nameAt, p, fmt.Errorf("the unknown member %q; a binding has %s", name, strings.Join(fields, ", ")))
		case seen:
			l.fail(nameAt, p, fmt.Errorf("the member %q twice", name))
		default:
			ms[name] = m
		}
	}
	l.command(&b, ms, base, at)
	if m, ok := ms[memberKey]; ok {
		if s, ok := l.text(m, memberKey); ok {
			q, err := ParseSequence(s, nil)
			if err != nil {
				l.fail(m.off, m.p, err)
			}
			b.Keys = q
		}
	} else if _, ok := ms[memberCommand]; ok && !b.Remove {
		l.fail(base, at, errors.New(`the binding has no "key"`))
	}
	if m, ok := ms[memberWhen]; ok {
		if s, ok := l.text(m, memberWhen); ok && s != "" {
			if _, err := when.Parse(s); err != nil {
				l.fail(m.off, m.p, err)
			}
			b.When = s
		}
	}
	if m, ok := ms[memberArgs]; ok {
		// The file was read allowing duplicate names; args may not have them.
		_, err := jsontext.NewDecoder(bytes.NewReader(m.v)).ReadValue()
		if se, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
			l.fail(m.off+int(se.ByteOffset), m.p+se.JSONPointer, err)
		}
		b.Args = m.v
		if err := b.Args.Compact(); err != nil { // m.v is one valid value, so not met
			l.fail(m.off, m.p, err)
		}
	}
	if m, ok := ms[memberContext]; ok {
		if s, ok := l.text(m, memberContext); ok {
			switch s {
			case "":
				l.fail(m.off, m.p, errors.New(`an empty "context"; the global scope is "global"`))
			case command.Global.String():
			default:
				b.Scope = command.Scope(s)
			}
		}
	}
	return b, len(l.errs) == n
}

// command reads the binding's command, and a "-" before it.
func (l *loader) command(b *Binding, ms map[string]member, base int, at jsontext.Pointer) {
	m, ok := ms[memberCommand]
	if !ok {
		l.fail(base, at, errors.New(`the binding has no "command"`))
		return
	}
	s, ok := l.text(m, memberCommand)
	if !ok {
		return
	}
	rest, remove := strings.CutPrefix(s, "-")
	b.Command, b.Remove = command.ID(rest), remove
	if err := b.Command.Valid(); err != nil {
		l.fail(m.off, m.p, err)
		return
	}
	if l.ids != nil && !l.ids(b.Command) {
		l.fail(m.off, m.p, fmt.Errorf("the unknown command %q", rest))
	}
}

// text is a member's string, or an error when it is not one.
func (l *loader) text(m member, name string) (string, bool) {
	if m.v.Kind() != '"' {
		l.fail(m.off, m.p, fmt.Errorf("%q is %s, not a string", name, kindName(m.v.Kind())))
		return "", false
	}
	return unquote(m.v), true
}

// unquote is the text of v, a string the decoder read. Such a string
// cannot fail to unquote; if it did, its raw text would be returned.
func unquote(v jsontext.Value) string {
	s, err := jsontext.AppendUnquote(nil, v)
	if err != nil {
		return string(v)
	}
	return string(s)
}

// kindName is a JSON value's kind, in words.
func kindName(k jsontext.Kind) string {
	switch k {
	case 'n':
		return "null"
	case 't', 'f':
		return "a boolean"
	case '"':
		return "a string"
	case '0':
		return "a number"
	case '{':
		return "an object"
	case '[':
		return "an array"
	}
	return "nothing"
}

// vscodeBinding is one element of a keybindings file, as ExportVSCode
// writes it.
type vscodeBinding struct {
	Key     string         `json:"key,omitzero"`
	Command string         `json:"command"`
	When    string         `json:"when,omitzero"`
	Args    jsontext.Value `json:"args,omitzero"`
	Context string         `json:"context,omitzero"`
}

// ExportVSCode writes bindings in the format LoadVSCode reads, the keys in
// VS Code's spelling and indented with tabs, as VS Code writes the file.
// LoadVSCode reads back the same bindings, but for their Location and the
// space in their Args, and a scope named "global", which reads back as
// command.Global. A <leader> placeholder is written as "<leader>".
func ExportVSCode(w io.Writer, bindings iter.Seq[Binding]) error {
	rows := []vscodeBinding{}
	for b := range bindings {
		r := vscodeBinding{Key: b.Keys.VSCode(), Command: string(b.Command), When: b.When, Args: b.Args}
		if b.Remove {
			r.Command = "-" + r.Command
		}
		if b.Scope != command.Global {
			r.Context = string(b.Scope)
		}
		rows = append(rows, r)
	}
	if err := json.MarshalWrite(w, rows, jsontext.WithIndent("\t")); err != nil {
		return fmt.Errorf("keymap: ExportVSCode: %w", err)
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("keymap: ExportVSCode: %w", err)
	}
	return nil
}
