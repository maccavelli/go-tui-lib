package lineedit

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/internal/lineinput"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// The keys as a legacy terminal sends them, and the Kitty protocol's
// shift+enter.
const (
	up, down, left, right = "\x1b[A", "\x1b[B", "\x1b[D", "\x1b[C"
	enter                 = "\r"
	altEnter              = "\x1b\r"
	shiftEnter            = "\x1b[13;2u"
	pasteOn, pasteOff     = "\x1b[200~", "\x1b[201~"
)

// TestIDs: the IDs the Composer gives a Line are the Line's own commands.
func TestIDs(t *testing.T) {
	want := []command.ID{
		CharLeftID, CharRightID, WordLeftID, WordRightID, LineStartID, LineEndID,
		DeleteBackID, DeleteForwardID, DeleteOrEOFID, DeleteWordBackID, DeleteWordForwardID,
		DeleteSpaceWordBackID, DeleteToLineEndID, DeleteToLineStartID, TransposeID,
	}
	if got := lineinput.Commands(); !slices.Equal(got, want) {
		t.Errorf("lineinput's commands %q; want lineedit's %q", got, want)
	}
}

// TestDefaultRules: the default bindings, in ComposerScope, and no ctrl+c.
func TestDefaultRules(t *testing.T) {
	got := map[string]command.ID{}
	for _, d := range DefaultRules() {
		if d.Scope != ComposerScope {
			t.Errorf("%s in %q; want %q", d.Keys, d.Scope, ComposerScope)
		}
		got[d.Keys.String()] = d.Command
	}
	for keys, id := range map[string]command.ID{
		"alt+f": WordRightID, "alt+b": WordLeftID, "alt+d": DeleteWordForwardID, "alt+backspace": DeleteWordBackID,
		"ctrl+w": DeleteSpaceWordBackID, "alt+enter": NewlineID, "shift+enter": NewlineID, "enter": SubmitID,
		"ctrl+j": SubmitID, "ctrl+l": ClearID, "ctrl+x ctrl+e": ExternalID, "ctrl+d": DeleteOrEOFID, "tab": CompleteID,
		"up": UpID, "ctrl+p": UpID, "down": DownID, "ctrl+n": DownID,
	} {
		if got[keys] != id {
			t.Errorf("%s binds %q; want %q", keys, got[keys], id)
		}
	}
	if _, ok := got["ctrl+c"]; ok {
		t.Error("ctrl+c is bound; it is the program's")
	}
	if len(got) != 35 {
		t.Errorf("%d bindings; want 35", len(got))
	}
}

// TestWords: alt+f stops at a word's end and alt+b at its start, alt+d and
// alt+backspace kill to there, a word being letters and digits by a
// cluster's first rune (0023-MADR A3 item 9, A5 item 6): can't is two
// words, 32.3 two, a CJK run one, an accented word one, composed or not.
func TestWords(t *testing.T) {
	for _, c := range []struct{ name, text, keys, want string }{
		{"can't, alt+f", "can't stop", "\x01\x1bf", "can|'t stop"},
		{"can't, alt+f twice", "can't stop", "\x01\x1bf\x1bf", "can't| stop"},
		{"32.3, alt+f", "32.3 kg", "\x01\x1bf", "32|.3 kg"},
		{"a CJK run, alt+f", "世界世界 ok", "\x01\x1bf", "世界世界| ok"},
		{"café composed, alt+b", "a café", "\x1bb", "a |café"},
		{"cafe\u0301 decomposed, alt+b", "a cafe\u0301", "\x1bb", "a |cafe\u0301"},
		{"alt+right and ctrl+right are alt+f", "one two", "\x01\x1b[1;3C\x1b[1;5C", "one two|"},
		{"alt+left and ctrl+left are alt+b", "one two three", "\x1b[1;3D\x1b[1;5D", "one |two three"},
		{"alt+d", "one two", "\x01\x1bd", "| two"},
		{"alt+d between words", "one  two", "\x01\x1bf\x1bd", "one|"},
		{"alt+backspace", "path/to/file.txt", "\x1b\x7f", "path/to/file.|"},
		{"a ZWJ sequence is one step", "a👨\u200d👩\u200d👧b", left + "\x7f", "a|b"},
	} {
		h := newHost(t, 80)
		h.typeIn(c.text + c.keys)
		if got := h.at(); got != c.want {
			t.Errorf("%s: %q; want %q", c.name, got, c.want)
		}
	}
}

// TestCtrlW: ctrl+w deletes back to white space, as Readline's
// unix-word-rubout and x/term's do.
func TestCtrlW(t *testing.T) {
	h := newHost(t, 80)
	h.typeIn("cat path/to/file.txt  \x17")
	if got := h.at(); got != "cat |" {
		t.Errorf("%q; want the path and the spaces after it gone", got)
	}
}

// TestNewline: alt+enter, shift+enter where the terminal reports it, and a
// trailing backslash before enter each insert a line break, drawn after
// the continuation prompt; enter submits the text with them.
func TestNewline(t *testing.T) {
	k, err := keymap.New(keymap.Settings{Features: keymap.Disambiguate, Facts: keymap.ShiftEnter}, DefaultRules())
	if err != nil {
		t.Fatal(err)
	}
	h := newHost(t, 40, WithKeymap(k))
	h.typeIn("one" + altEnter + "two" + shiftEnter + "three\\" + enter + "four")
	if got := h.c.Text(); got != "one\ntwo\nthree\nfour" {
		t.Fatalf("text %q", got)
	}
	if got := h.scr.Lines()[:4]; !slices.Equal(got, []string{"> one", "~ two", "~ three", "~ four"}) {
		t.Errorf("drawn %q", got)
	}
	h.typeIn(enter)
	if got := h.submitted(); !slices.Equal(got, []string{"one\ntwo\nthree\nfour"}) {
		t.Errorf("submitted %q", got)
	}
	if got := h.scr.Lines()[:5]; !slices.Equal(got, []string{"> one", "~ two", "~ three", "~ four", ">"}) {
		t.Errorf("after the submit: %q; want it in the scrollback and a prompt below", got)
	}
}

// TestPaste: a paste is inserted as text, cleaned with its line breaks and
// tabs kept, a paste's end marker inside its text removed, and no
// completion run on it; under PasteSubmit it submits the text at once.
func TestPaste(t *testing.T) {
	completed := 0
	x := CompleterFunc(func(line string, pos int) (string, int, bool) { completed++; return "", 0, false })
	h := newHost(t, 60, WithCompleter(x))
	h.typeIn("say " + pasteOn + "a\tb\r\nc\x1b[31m\x07d" + pasteOff)
	if got := h.c.Text(); got != "say a\tb\ncd" || len(h.submitted()) != 0 || completed != 0 {
		t.Errorf("PasteInsert: %q, submitted %q, %d completions; want the text kept, nothing submitted", got, h.submitted(), completed)
	}
	h.c.HandleInput(termkey.Input{Kind: termkey.InputPaste, Text: "x\x1b[201~y"}, h.now)
	h.drain()
	if got := h.c.Text(); !strings.HasSuffix(got, "xy") || strings.Contains(got, "\x1b") {
		t.Errorf("an end marker inside the text: %q; want it removed", got)
	}
	s := newHost(t, 60, WithPaste(PasteSubmit))
	s.typeIn("say " + pasteOn + "one\ntwo" + pasteOff)
	if got := s.submitted(); !slices.Equal(got, []string{"say one\ntwo"}) || s.c.Text() != "" {
		t.Errorf("PasteSubmit: submitted %q, left %q; want the whole text submitted", got, s.c.Text())
	}
}

// TestMemHistory: an empty entry and one equal to the newest are dropped,
// the oldest go past Max, At is newest first and panics out of range, and
// the file form escapes a line break and a backslash and reads back.
func TestMemHistory(t *testing.T) {
	h := &MemHistory{Max: 3}
	for _, e := range []string{"a", "", "a", "b", "a", "two\nlines", `back\slash`} {
		h.Add(e)
	}
	var got []string
	for i := range h.Len() {
		got = append(got, h.At(i))
	}
	if want := []string{`back\slash`, "two\nlines", "a"}; !slices.Equal(got, want) {
		t.Errorf("entries %q; want %q", got, want)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("At(3) of 3 did not panic")
			}
		}()
		h.At(3)
	}()
	var b bytes.Buffer
	if _, err := h.WriteTo(&b); err != nil {
		t.Fatal(err)
	}
	if want := "a\ntwo\\nlines\nback\\\\slash\n"; b.String() != want {
		t.Errorf("file %q; want %q", b.String(), want)
	}
	back := &MemHistory{}
	if _, err := back.ReadFrom(strings.NewReader(b.String() + "\r\nkept\\x\r\n")); err != nil {
		t.Fatal(err)
	}
	if back.Len() != 4 || back.At(0) != `kept\x` || back.At(1) != `back\slash` || back.At(2) != "two\nlines" {
		t.Errorf("read back %d entries, newest %q, %q, %q", back.Len(), back.At(0), back.At(1), back.At(2))
	}
	var zero MemHistory
	for i := range DefaultHistorySize + 5 {
		zero.Add(fmt.Sprint(i))
	}
	if zero.Len() != DefaultHistorySize || zero.At(zero.Len()-1) != "5" {
		t.Errorf("the zero MemHistory kept %d, oldest %q; want %d, from 5", zero.Len(), zero.At(zero.Len()-1), DefaultHistorySize)
	}
}

// TestHistoryRows: up and down move by row first, and recall history from
// the first and the last row only (gobble's 0027-MADR D4, Q1).
func TestHistoryRows(t *testing.T) {
	const fox = "the quick brown fox jumps"
	// At 12 columns, fox is "> the quick", "  brown fox" and "  jumps".
	for _, c := range []struct {
		name string
		hist fixedHistory
		in   string
		want string
	}{
		{"up from the last row", nil, fox + up, "the quick brown| fox jumps"},
		{"up twice", nil, fox + up + up, "the q|uick brown fox jumps"},
		{"up from the first row recalls", fixedHistory{"old"}, fox + up + up + up, "old|"},
		{"down from a middle row moves", fixedHistory{"old"}, fox + up + up + down, "the quick brown| fox jumps"},
		{"down from the last row with nothing recalled", nil, fox + down, fox + "|"},
		{"down after a recall gives the draft back", fixedHistory{"old"}, "x" + up + down, "x|"},
		{"a recalled entry is edited by row", fixedHistory{fox}, "x" + up + up, "the quick brown| fox jumps"},
	} {
		h := newHost(t, 12, WithHistory(c.hist))
		h.typeIn(c.in)
		if got := h.at(); got != c.want {
			t.Errorf("%s: %q; want %q", c.name, got, c.want)
		}
	}
	m := &MemHistory{}
	h := newHost(t, 40, WithHistory(m))
	h.typeIn("first" + enter + "second" + enter + "second" + enter + up + up)
	if got := h.c.Text(); got != "first" || m.Len() != 2 {
		t.Errorf("after three submits: %d entries, up twice gives %q; want 2, first", m.Len(), got)
	}
}

// TestLayout: whole words to the width, a row's break taking its space, a
// word wider than its row broken at the width, continuation rows indented
// by the prompt, wide clusters at their width, and a cursor past a full
// row on a row of its own.
func TestLayout(t *testing.T) {
	for _, c := range []struct {
		name, text string
		width      int
		want       []string
	}{
		{"whole words", "the quick brown fox jumps", 12, []string{"> the quick", "  brown fox", "  jumps"}},
		{"a word wider than a row", "see /home/<user>/a-long-path now", 12, []string{"> see", "  /home/<use", "  r>/a-long-", "  path now"}},
		{"wide clusters", "世界世界世界 ab", 10, []string{"> 世界世界", "  世界 ab"}},
		{"a ZWJ sequence's width", "a👨\u200d👩 b", 7, []string{"> a👨\u200d👩", "  b"}},
		{"a full row", "abcde", 7, []string{"> abcde", "  "}},
		{"a pasted tab drawn as a space", pasteOn + "a\tb" + pasteOff, 10, []string{"> a b"}},
	} {
		h := newHost(t, c.width)
		h.typeIn(c.text)
		rows, _, _ := h.c.View(c.width)
		if !slices.Equal(rows, c.want) {
			t.Errorf("%s at %d: %q; want %q", c.name, c.width, rows, c.want)
		}
		for _, r := range rows {
			if w := ansi.WcWidth.StringWidth(r); w > c.width {
				t.Errorf("%s: %q is %d cells, in %d", c.name, r, w, c.width)
			}
		}
	}
}

// TestLayoutMarks: a mark or a selector with no base at the start of a
// line joins the space of the prompt drawn before it, and is measured
// where it lands: no row is wider than the width as the terminal draws it,
// and each cursor column is where the drawn text puts it (0023-MADR A5
// item 7).
func TestLayoutMarks(t *testing.T) {
	for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		for width := 4; width <= 24; width++ {
			l := lineinput.NewRunes(0, m)
			l.Insert("\ufe0f0 first\n\u0301second line here")
			cl := l.Clusters()
			got := layout(m, width, "> ", "~ ", cl)
			for _, row := range got.rows {
				if w := m.StringWidth(row); w > width {
					t.Errorf("method %d, width %d: %q is %d cells", m, width, row, w)
				}
			}
			if want := m.StringWidth("> " + cl[0]); got.places[1] != (place{0, want}) && want < width {
				t.Errorf("method %d, width %d: after the selector the cursor is at %v; want column %d, where it is drawn", m, width, got.places[1], want)
			}
		}
	}
}

// TestLayoutGolden: the Composer drawn on the test screen across the
// matrix: an empty prompt, a long text in whole words, three lines, and a
// header above a recalled entry.
func TestLayoutGolden(t *testing.T) {
	long := "Explain why the resize redraw in the inline region lands two rows low on xterm, and propose the smallest fix with a test that fails first."
	for _, s := range []struct {
		name string
		keys string
		o    []Option
	}{
		{"empty", "", nil},
		{"long", long, nil},
		{"lines", "first line" + altEnter + "second, a little longer than the first" + altEnter + "third", nil},
		{"header", up, []Option{WithHeader(func() string { return "model: claude-sonnet-5-5  \x1b[31mctx 12%\n" }), WithHistory(fixedHistory{long})}},
	} {
		tuitest.Golden(t, s.name, tuitest.Matrix{Widths: []int{60, 100}}, func(c tuitest.Case) string {
			h := newHost(t, c.Width, append([]Option{WithGlyphs(c.Glyphs())}, s.o...)...)
			h.typeIn(s.keys)
			text := h.scr.Text()
			tuitest.Fits(t, text, c.Width, ansi.WcWidth)
			return text
		})
	}
}

// TestExternal: ctrl+x ctrl+e sends EditExternal with the text only
// WithExternalEditor; SetText puts the edited text back and draws it.
func TestExternal(t *testing.T) {
	off := newHost(t, 40)
	off.typeIn("draft\x18\x05")
	if slices.Contains(off.kinds(), EditExternal) {
		t.Error("EditExternal sent without WithExternalEditor")
	}
	h := newHost(t, 40, WithExternalEditor())
	h.typeIn("draft\x18\x05")
	if len(h.events) != 1 || h.events[0].Kind != EditExternal || h.events[0].Text != "draft" {
		t.Fatalf("events %+v; want one EditExternal with the draft", h.events)
	}
	h.c.SetText("edited\ntwo\x1b[2J")
	h.drain()
	if got := h.c.Text(); got != "edited\ntwo" {
		t.Errorf("SetText: %q; want it cleaned, its line break kept", got)
	}
	if got := h.scr.Lines()[:2]; !slices.Equal(got, []string{"> edited", "~ two"}) {
		t.Errorf("drawn %q", got)
	}
}

// TestClear: ctrl+l clears the screen and draws the Composer at its top
// only WithClearScreen; the text stays.
func TestClear(t *testing.T) {
	for _, on := range []bool{false, true} {
		var o []Option
		if on {
			o = append(o, WithClearScreen())
		}
		h := newHost(t, 40, o...)
		_, _ = h.scr.Write([]byte("\x1b[5;1H")) // the prompt five rows down
		h.c = NewComposer(append([]Option{WithGlyphs(glyph.ASCII())}, o...)...)
		h.c.Resize(40, h.now)
		h.drain()
		h.written = nil
		h.typeIn("abc\x0c")
		cleared := bytes.Contains(h.written, []byte(clearScreen))
		if cleared != on || h.c.Text() != "abc" {
			t.Errorf("WithClearScreen %v: cleared %v, text %q", on, cleared, h.c.Text())
		}
		if on && (h.scr.Lines()[0] != "> abc" || h.scr.Row != 0) {
			t.Errorf("after the clear: %q, cursor row %d; want the prompt at the top", h.scr.Lines()[0], h.scr.Row)
		}
	}
}

// TestInterrupt: ctrl+c sends EditInterrupt with the text, which stays;
// the host decides what it means.
func TestInterrupt(t *testing.T) {
	h := newHost(t, 40)
	h.typeIn("half\x03")
	if len(h.events) != 1 || h.events[0].Kind != EditInterrupt || h.events[0].Text != "half" || h.c.Text() != "half" {
		t.Errorf("events %+v, text %q", h.events, h.c.Text())
	}
}

// TestEOF: ctrl+d on an empty line and the host's Cancel end the input,
// the prompt left in the scrollback; nothing is taken after.
func TestEOF(t *testing.T) {
	for name, act := range map[string]func(h *host){
		"ctrl+d": func(h *host) { h.typeIn("\x04") },
		"Cancel": func(h *host) { h.c.Cancel(); h.drain() },
	} {
		h := newHost(t, 40)
		act(h)
		h.typeIn("more" + enter)
		h.c.Cancel()
		h.drain()
		if !slices.Equal(h.kinds(), []ComposerEventKind{EditEOF}) || h.scr.Text() != ">\n" {
			t.Errorf("%s: events %v, screen %q; want one EditEOF and the prompt left", name, h.kinds(), h.scr.Text())
		}
	}
	h := newHost(t, 40)
	h.typeIn("ab" + left + "\x04")
	if h.eof() || h.c.Text() != "a" {
		t.Errorf("ctrl+d on a line: eof %v, text %q; want a deletion", h.eof(), h.c.Text())
	}
}

// TestCompleter: tab gives the Completer the line and the cursor's byte
// offset, and takes its line and cursor, cleaned.
func TestCompleter(t *testing.T) {
	var gotLine string
	var gotPos int
	x := CompleterFunc(func(line string, pos int) (string, int, bool) {
		gotLine, gotPos = line, pos
		return "héllo \x1b[1mworld", len("héllo"), true
	})
	h := newHost(t, 40, WithCompleter(x))
	h.typeIn("héx" + left + "\t")
	if gotLine != "héx" || gotPos != len("hé") {
		t.Errorf("the Completer got %q, %d; want héx, %d", gotLine, gotPos, len("hé"))
	}
	if got := h.at(); got != "héllo| world" {
		t.Errorf("after the completion: %q", got)
	}
}

// TestKeymap: WithKeymap's keymap decides the keys: a user's binding
// rebinds one, a removed one types nothing, and a binding in another scope
// does not reach the Composer.
func TestKeymap(t *testing.T) {
	user := func(scope command.Scope, keys string, id command.ID, remove bool) keymap.Binding {
		q, err := keymap.ParseSequence(keys, nil)
		if err != nil {
			t.Fatal(err)
		}
		return keymap.Binding{Keys: q, Command: id, Scope: scope, Remove: remove, Location: keymap.Location{Layer: keymap.UserLayer}}
	}
	k, err := keymap.New(keymap.Settings{}, DefaultRules(), []keymap.Binding{
		user(ComposerScope, "ctrl+o", SubmitID, false),
		user(ComposerScope, "ctrl+u", DeleteToLineStartID, true),
		user(command.Global, "ctrl+a", "other.thing", false),
	})
	if err != nil {
		t.Fatal(err)
	}
	h := newHost(t, 40, WithKeymap(k))
	h.typeIn("abc\x15\x01X\x0f")
	if got := h.submitted(); !slices.Equal(got, []string{"Xabc"}) {
		t.Errorf("submitted %q; want ctrl+u to do nothing, ctrl+a the line's start, ctrl+o to submit", got)
	}
}

// TestRules: WithRules cleans more than the default, and never less than
// safetext's Line rules; a paste never less than its floor.
func TestRules(t *testing.T) {
	weak := WithRules(safetext.Rules{Newlines: true, KeepSGR: true})
	h := newHost(t, 40, weak, WithPrompt("\x1b[31m$ "))
	h.c.HandleInput(termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: 'x', Text: "x\x1b[1m\n"}}, h.now)
	h.c.HandleInput(termkey.Input{Kind: termkey.InputPaste, Text: "p\x1b[32mq\nr"}, h.now)
	h.drain()
	if got := h.c.Text(); got != "x pq\nr" || bytes.Contains(h.written, []byte("\x1b[31m")) || bytes.Contains(h.written, []byte("\x1b[32m")) {
		t.Errorf("under weaker rules: %q, writing %q; want the floors held", got, h.written)
	}
	strict := newHost(t, 40, WithRules(safetext.Rules{}))
	strict.typeIn(pasteOn + "a\tb" + pasteOff)
	if got := strict.c.Text(); got != "a\tb" {
		t.Errorf("a paste under the strictest rules: %q; want its tab kept", got)
	}
}

// TestResize: a resize asks where the cursor is, and redraws at the new
// width once the report comes.
func TestResize(t *testing.T) {
	h := newHost(t, 40)
	h.typeIn("the quick brown fox jumps over the lazy dog")
	h.scr.Resize(20, inline.ReflowDown)
	h.c.Resize(20, h.now)
	h.drain()
	if len(h.replies) == 0 {
		t.Fatal("the resize asked nothing of the screen")
	}
	h.typeIn("")
	want, _, _ := h.c.View(20)
	if got := h.scr.Lines()[:len(want)]; !slices.Equal(got, trimRows(want)) {
		t.Errorf("after the resize: %q; want %q", got, want)
	}
}

// TestWithRegion: the Composer draws in the host's region, with its
// options.
func TestWithRegion(t *testing.T) {
	r := inline.NewRegion(inline.WithoutCursorQuery())
	h := newHost(t, 40, WithRegion(r))
	h.typeIn("abc")
	h.scr.Resize(30, inline.ReflowDown)
	h.c.Resize(30, h.now)
	h.drain()
	if len(h.replies) != 0 {
		t.Errorf("the host's region, made WithoutCursorQuery, asked for the cursor: %q", h.replies)
	}
	if got := h.scr.Lines()[0]; got != "> abc" {
		t.Errorf("redrawn with no report: %q", got)
	}
}

// TestRow1Report: a resize's cursor report on the first row is taken as
// the report, not as a key, though a binding names that key.
func TestRow1Report(t *testing.T) {
	k, err := keymap.New(keymap.Settings{}, append(DefaultRules(), keymap.Default{
		Keys: seq("ctrl+f3"), Command: SubmitID, Scope: ComposerScope,
	}))
	if err != nil {
		t.Fatal(err)
	}
	h := newHost(t, 40, WithKeymap(k))
	h.typeIn("x")
	h.c.Resize(30, h.now)
	h.drain()
	h.replies = nil
	h.typeIn("\x1b[1;5R")
	if len(h.submitted()) != 0 {
		t.Errorf("the awaited report reached the keymap as a key: %q", h.submitted())
	}
	h.typeIn("\x1b[1;5R")
	if len(h.submitted()) != 1 {
		t.Error("ctrl+f3, with no report awaited, did not reach the keymap")
	}
}

// TestKeyDrawBudget: a key's drawing writes at most the text it changed
// and 40 bytes besides, gobble's 0030-MADR D5: a key at a long text's end,
// in its middle, and one that only moves the cursor.
func TestKeyDrawBudget(t *testing.T) {
	text := strings.Repeat("words fill the prompt ", 10)
	for _, c := range []struct{ name, setup, key string }{
		{"a key at the end", text, "x"},
		{"a key in the middle", text + strings.Repeat(left, len(text)-40), "x"},
		{"the cursor moved", text, left},
	} {
		h := newHost(t, 80)
		h.typeIn(c.setup)
		before, _, _ := h.c.View(80)
		h.written = nil
		h.typeIn(c.key)
		after, _, _ := h.c.View(80)
		changed := 0
		for i, row := range after {
			was := ""
			if i < len(before) {
				was = before[i]
			}
			n := 0
			for n < len(was) && n < len(row) && was[n] == row[n] {
				n++
			}
			changed += len(row) - n
		}
		if n := len(h.written); n > changed+40 {
			t.Errorf("%s: %d bytes for %d changed, more than %d: %q", c.name, n, changed, changed+40, h.written)
		}
	}
}

// trimRows is rows as the screen's Lines give them, trailing spaces
// dropped.
func trimRows(rows []string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = strings.TrimRight(r, " ")
	}
	return out
}
