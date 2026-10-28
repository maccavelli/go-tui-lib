package pick_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/lineedit"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/pick"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// The keys as a legacy terminal sends them.
const (
	up     = "\x1b[A"
	down   = "\x1b[B"
	pgup   = "\x1b[5~"
	pgdown = "\x1b[6~"
	enter  = "\r"
	esc    = "\x1b"
)

// TestLayoutGolden: each list drawn on the test screen across the matrix,
// with an empty query, "son", a query nothing matches, and "son" through
// the index; each line within its width.
func TestLayoutGolden(t *testing.T) {
	for _, list := range []struct {
		name    string
		entries []pick.Entry
	}{{"models", models()}, {"sessions", sessions()}} {
		for _, q := range []struct {
			name string
			o    []pick.Option
		}{
			{"empty", nil},
			{"son", []pick.Option{pick.WithQuery("son")}},
			{"none", []pick.Option{pick.WithQuery("zzqx")}},
			{"index", []pick.Option{pick.WithQuery("son"), pick.WithIndexAbove(10)}},
		} {
			tuitest.Golden(t, list.name+"-"+q.name, tuitest.Matrix{Widths: []int{60, 100}}, func(c tuitest.Case) string {
				h := newHost(t, c.Width, list.entries, append([]pick.Option{pick.WithGlyphs(c.Glyphs())}, q.o...)...)
				text := h.scr.Text()
				tuitest.Fits(t, text, c.Width, ansi.WcWidth)
				return text
			})
		}
	}
}

// TestDetail: an entry's Detail is drawn at the right and never matched,
// and gives way when the Text would keep fewer than 8 cells.
func TestDetail(t *testing.T) {
	entries := []pick.Entry{{ID: "a", Text: "alpha", Detail: "beta"}}
	h := newHost(t, 30, entries)
	if got := h.lines()[1]; got != "> alpha                   beta" {
		t.Errorf("drawn %q; want the detail at the right", got)
	}
	h = newHost(t, 30, entries, pick.WithQuery("beta"))
	if got := h.lines()[1]; got != "" {
		t.Errorf("the query beta matched %q: the detail is not matched", got)
	}
	h = newHost(t, 12, []pick.Entry{{ID: "a", Text: "alphabetical", Detail: "beta"}})
	if got := h.lines()[1]; got != "> alphabeti~" {
		t.Errorf("at 12 columns: %q; want the text cut and the detail left out", got)
	}
}

// TestRankLikeMatcher: a query ranks the entries by the matcher's score,
// grade and then value, best first, ties in the order given, never by
// length as fuzzy's Rank breaks them
// (docs/decisions/0023-MADR-line-mode-components.md A4).
func TestRankLikeMatcher(t *testing.T) {
	entries := append([]pick.Entry{{ID: "long", Text: "qabzzzzzz"}, {ID: "short", Text: "qab"}}, models()...)
	m := fuzzy.NewMatcher("ab")
	long, _ := m.Score("qabzzzzzz")
	if short, _ := m.Score("qab"); long != short {
		t.Fatalf("the tie's two entries score %v and %v; the fixture needs a tie", long, short)
	}
	for _, q := range []string{"ab", "son", "gpt", "mini", "o", "lla"} {
		m := fuzzy.NewMatcher(q)
		type scored struct {
			id string
			s  fuzzy.Score
		}
		var all []scored
		for _, e := range entries {
			if s, ok := m.Score(e.Text); ok {
				all = append(all, scored{e.ID, s})
			}
		}
		slices.SortStableFunc(all, func(a, b scored) int {
			if a.s.Grade != b.s.Grade {
				return int(a.s.Grade) - int(b.s.Grade)
			}
			return b.s.Value - a.s.Value
		})
		var want []string
		for _, x := range all {
			want = append(want, x.id)
		}
		if got := order(t, entries, q); !slices.Equal(got, want) {
			t.Errorf("%q: ranked %q; want %q", q, got, want)
		}
	}
	if got := order(t, entries, "ab"); slices.Index(got, "long") > slices.Index(got, "short") {
		t.Errorf("a tie: %q; want long before short, the order given", got)
	}
}

// TestEmptyQueryOrder: an empty query keeps the entries in the order
// given, and so does a query typed and taken back.
func TestEmptyQueryOrder(t *testing.T) {
	entries := models()
	if got := order(t, entries, ""); !slices.Equal(got, ids(entries)) {
		t.Errorf("an empty query: %q", got)
	}
	h := newHost(t, 100, entries, pick.WithHeight(30)) // and the query row, within the screen's 40
	before := h.scr.Text()
	h.typeIn("son\x7f\x7f\x7f")
	if got := h.scr.Text(); got != before {
		t.Errorf("after son typed and deleted:\n%s\nwant:\n%s", got, before)
	}
}

// TestIndex: above WithIndexAbove entries the index narrows them first,
// lossy as fuzzy documents it: a sparse match a full ranking finds is
// dropped, and those it keeps are in the full ranking's order.
func TestIndex(t *testing.T) {
	entries := append(models(), pick.Entry{ID: "sparse", Text: "s1o2n3n4e5t6"})
	full := order(t, entries, "sonnet")
	indexed := order(t, entries, "sonnet", pick.WithIndexAbove(10))
	if !slices.Contains(full, "sparse") {
		t.Fatalf("the full ranking %q does not find the sparse match", full)
	}
	if slices.Contains(indexed, "sparse") {
		t.Errorf("through the index: %q; want the sparse match dropped, as the index drops it", indexed)
	}
	var kept []string
	for _, id := range full {
		if slices.Contains(indexed, id) {
			kept = append(kept, id)
		}
	}
	if len(indexed) == 0 || !slices.Equal(kept, indexed) {
		t.Errorf("through the index: %q; want the full ranking's order, %q", indexed, kept)
	}
	if got := order(t, entries, "sonnet", pick.WithIndexAbove(0)); !slices.Equal(got, full) {
		t.Errorf("WithIndexAbove(0) keeps the default, a full ranking of 41: %q", got)
	}
}

// TestCount: the count is the selected row's place and the matches', "no
// results" for none, a "+" when the index yielded all it yields, and gives
// way to the query below 8 cells of it.
func TestCount(t *testing.T) {
	h := newHost(t, 40, models())
	if got := h.lines()[0]; got != ">                                1 of 40" {
		t.Errorf("drawn %q", got)
	}
	h.typeIn(down + down)
	if got := h.lines()[0]; !strings.HasSuffix(got, " 3 of 40") {
		t.Errorf("after two moves: %q", got)
	}
	h.typeIn("zzqx")
	if got := h.lines()[0]; got != "> zzqx                        no results" {
		t.Errorf("no match: %q", got)
	}
	var many []pick.Entry
	for i := range 2500 {
		many = append(many, pick.Entry{ID: fmt.Sprint(i), Text: fmt.Sprintf("x%04d", i)})
	}
	h = newHost(t, 40, many, pick.WithIndexAbove(10), pick.WithQuery("x"))
	if got := h.lines()[0]; !strings.HasSuffix(got, " 1 of 2000+") {
		t.Errorf("through a capped index: %q; want 1 of 2000+", got)
	}
	h = newHost(t, 40, many, pick.WithQuery("x"))
	if got := h.lines()[0]; !strings.HasSuffix(got, " 1 of 2500") {
		t.Errorf("ranked in full: %q; want 1 of 2500", got)
	}
	h = newHost(t, 16, models())
	if got := h.lines()[0]; got != ">" {
		t.Errorf("at 16 columns: %q; want the count left out", got)
	}
}

// TestMoves: up and ctrl+p, down and ctrl+n move a row, stopping at either
// end; pgup and pgdown move a page; the window keeps the selection drawn;
// a change of the query selects the first row again.
func TestMoves(t *testing.T) {
	h := newHost(t, 60, models(), pick.WithHeight(5))
	selected := func() string {
		for _, l := range h.lines()[1:] {
			if rest, ok := strings.CutPrefix(l, "> "); ok {
				return strings.Fields(rest)[0]
			}
		}
		return ""
	}
	steps := []struct{ keys, want string }{
		{up, "anthropic/claude-opus-5-5"},
		{down, "anthropic/claude-sonnet-5-5"},
		{"\x0e", "anthropic/claude-haiku-5-5"},
		{"\x10", "anthropic/claude-sonnet-5-5"},
		{pgdown, "openai/gpt-5"},
		{pgdown, "openai/o3"},
		{pgup, "openai/gpt-5"},
		{strings.Repeat(pgdown, 10), "local/ollama-person-7b"},
		{down, "local/ollama-person-7b"},
		{"m", order(t, models(), "m")[0]},
		{down + "i", order(t, models(), "mi")[0]},
	}
	for _, s := range steps {
		h.typeIn(s.keys)
		if got := selected(); got != s.want {
			t.Errorf("after %q: %q selected; want %q", s.keys, got, s.want)
		}
	}
}

// userKeys is a keymap of pick's and lineedit's defaults, as WithKeymap's
// documentation asks, and the user's layer.
func userKeys(t *testing.T, user ...keymap.Binding) *keymap.Keymap {
	t.Helper()
	k, err := keymap.New(keymap.Settings{}, append(pick.DefaultRules(), lineedit.DefaultRules()...), user)
	if err != nil {
		t.Fatalf("keymap.New: %v", err)
	}
	return k
}

// userBinding is a user-layer binding of keys to id in scope, or its
// removal.
func userBinding(t *testing.T, scope command.Scope, keys string, id command.ID, remove bool) keymap.Binding {
	t.Helper()
	q, err := keymap.ParseSequence(keys, nil)
	if err != nil {
		t.Fatal(err)
	}
	return keymap.Binding{Keys: q, Command: id, Scope: scope, Remove: remove, Location: keymap.Location{Layer: keymap.UserLayer}}
}

// TestKeymap: WithKeymap's keymap decides the keys, in ChooserScope alone:
// a user's binding moves, a removed one types, a binding in another scope
// does not reach the Chooser, and a sequence waits, its keys typed into the
// query when it matches nothing or its wait ends.
func TestKeymap(t *testing.T) {
	k := userKeys(t,
		userBinding(t, pick.ChooserScope, "ctrl+t", pick.SelectNextID, false),
		userBinding(t, pick.ChooserScope, "down", pick.SelectNextID, true),
		userBinding(t, command.Global, "ctrl+a", "other.thing", false),
		userBinding(t, pick.ChooserScope, "g g", pick.SelectNextID, false),
	)
	h := newHost(t, 60, models(), pick.WithKeymap(k))
	h.typeIn("\x14")
	if got := h.lines()[0]; !strings.HasSuffix(got, " 2 of 40") {
		t.Errorf("ctrl+t, the user's: %q; want the selection moved", got)
	}
	h.typeIn(down)
	if got := h.lines()[0]; !strings.HasSuffix(got, " 2 of 40") {
		t.Errorf("down, removed: %q; want nothing moved", got)
	}
	h.typeIn("gg")
	if got := h.lines()[0]; !strings.HasSuffix(got, " 3 of 40") {
		t.Errorf("g g, the user's sequence: %q; want the selection moved", got)
	}
	h.typeIn("gx")
	if got := h.lines()[0]; !strings.HasPrefix(got, "> gx ") {
		t.Errorf("g then x: %q; want both typed into the query", got)
	}
	h.typeIn("\x01g")
	due, ok := h.c.Timeout()
	if !ok || !due.After(h.now) {
		t.Fatalf("a pending g: Timeout = %v, %v; want the sequence's wait", due, ok)
	}
	h.c.HandleTimeout(due)
	h.drain()
	if got := h.lines()[0]; !strings.HasPrefix(got, "> ggx ") {
		t.Errorf("ctrl+a, then g and its wait ended: %q; want ctrl+a to the query's start and g typed there", got)
	}
	if _, ok := h.c.Timeout(); ok {
		t.Error("Timeout still due after the wait ended")
	}
	// The query's editing keys are the keymap's too, in lineedit's scope
	// after pick's (0023-MADR A5 item 4).
	q := userKeys(t, userBinding(t, lineedit.ComposerScope, "ctrl+o", lineedit.DeleteToLineStartID, false))
	h = newHost(t, 60, models(), pick.WithKeymap(q))
	h.typeIn("son\x0f")
	if got := h.lines()[0]; !strings.HasSuffix(got, " 1 of 40") {
		t.Errorf("ctrl+o, the user's lineedit command: %q; want the query emptied", got)
	}
}

// TestCancel: esc, ctrl+g, ctrl+c and the host's Cancel each cancel, the
// prompt and the catalogue's word left; nothing is taken after.
func TestCancel(t *testing.T) {
	for name, act := range map[string]func(h *host){
		"esc":    func(h *host) { h.typeIn(esc) },
		"ctrl+g": func(h *host) { h.typeIn("\x07") },
		"ctrl+c": func(h *host) { h.typeIn("\x03") },
		"Cancel": func(h *host) { h.c.Cancel(); h.drain() },
	} {
		h := newHost(t, 40, models(), pick.WithQuery("son"))
		act(h)
		if h.done == nil || !h.done.Canceled || h.done.Entry != (pick.Entry{}) {
			t.Errorf("%s: %+v; want canceled", name, h.done)
			continue
		}
		if got := h.scr.Text(); got != "> cancel\n" {
			t.Errorf("%s: left %q; want the prompt and the word", name, got)
		}
		h.typeIn("x" + enter)
		h.c.Cancel()
		h.c.Resize(30, h.now)
		h.c.HandleTimeout(h.now.Add(time.Hour))
		h.drain()
		if h.dones != 1 || h.scr.Text() != "> cancel\n" {
			t.Errorf("%s: %d done events, %q; want one, and nothing drawn after", name, h.dones, h.scr.Text())
		}
	}
}

// TestQueryEdit: the query takes typed text, its editing keys and a paste,
// cleaned, bounded at 1024 bytes; WithQuery starts it.
func TestQueryEdit(t *testing.T) {
	h := newHost(t, 60, models(), pick.WithQuery("gpt"))
	if got, n := h.lines()[0], len(order(t, models(), "gpt")); !strings.HasPrefix(got, "> gpt ") || !strings.HasSuffix(got, fmt.Sprintf(" 1 of %d", n)) {
		t.Errorf("WithQuery: %q; want gpt and its %d matches", got, n)
	}
	h.typeIn("\x7f\x7f\x7f\x1b[200~so\nn\x1b[201~")
	if got := h.lines()[0]; !strings.HasPrefix(got, "> so n ") {
		t.Errorf("a paste with a line break: %q; want it one line", got)
	}
	h.typeIn("\x15")
	if got := h.lines()[0]; !strings.HasSuffix(got, " 1 of 40") || strings.HasPrefix(got, "> so") {
		t.Errorf("ctrl+u: %q; want the query empty", got)
	}
	h.typeIn("\x1b[200~" + strings.Repeat("x", 1100) + "\x1b[201~")
	h.typeIn(strings.Repeat("\x7f", 1024))
	if got := h.lines()[0]; !strings.HasSuffix(got, " 1 of 40") {
		t.Errorf("1100 bytes pasted, 1024 deleted: %q; want the query empty, bounded at 1024", got)
	}
}

// TestAccept: enter chooses the selected entry, the caller's, and leaves
// the prompt and its text, the cursor below.
func TestAccept(t *testing.T) {
	h := newHost(t, 60, models(), pick.WithPrompt("model"))
	h.typeIn("son" + down + enter)
	want := order(t, models(), "son")[1]
	if h.done == nil || h.done.Canceled || h.done.Entry.ID != want {
		t.Fatalf("chose %+v; want %s, the second ranked", h.done, want)
	}
	if got := h.scr.Text(); got != "model "+want+"\n" {
		t.Errorf("left %q; want the prompt and the entry's text", got)
	}
	if h.scr.Row != 1 || h.scr.Col != 0 {
		t.Errorf("cursor at %d,%d; want 1,0, below the line", h.scr.Row, h.scr.Col)
	}
	long := []pick.Entry{{ID: "l", Text: strings.Repeat("abcdefghij", 8)}}
	h = newHost(t, 30, long)
	h.typeIn(enter)
	if got := h.lines()[0]; got != "> abcdefghijabcdefghijabcdefg~" {
		t.Errorf("a long text: left %q; want one line, cut", got)
	}
}

// TestAcceptNoMatch: enter with nothing matched does nothing.
func TestAcceptNoMatch(t *testing.T) {
	h := newHost(t, 60, models(), pick.WithQuery("zzqx"))
	h.typeIn(enter)
	if h.done != nil {
		t.Fatalf("enter with no match: %+v", h.done)
	}
	h.typeIn("\x15" + enter)
	if h.done == nil || h.done.Entry.ID != models()[0].ID {
		t.Errorf("after the query was cleared: %+v; want the first entry", h.done)
	}
}

// TestCleaned: the entries' text is drawn as one line of display text,
// and the event carries the entry as the caller gave it.
func TestCleaned(t *testing.T) {
	e := pick.Entry{ID: "x", Text: "rm\x1b]0;pwned\x07 -rf\n/\x1b[2J", Detail: "\x1b[31mnow"}
	h := newHost(t, 40, []pick.Entry{e}, pick.WithPrompt("\x1b[1mrun"))
	if got := h.lines()[1]; got != "> rm -rf /"+strings.Repeat(" ", 27)+"now" {
		t.Errorf("drawn %q", got)
	}
	if got := h.lines()[0]; !strings.HasPrefix(got, "run ") {
		t.Errorf("the prompt: %q", got)
	}
	h.typeIn(enter)
	if h.done == nil || h.done.Entry != e {
		t.Errorf("chose %+v; want the entry as given", h.done)
	}
	if got := h.lines()[0]; got != "run rm -rf /" {
		t.Errorf("left %q", got)
	}
}

// TestRules: WithRules cleans more than the default, and never less than
// safetext's Line rules.
func TestRules(t *testing.T) {
	entries := []pick.Entry{{ID: "x", Text: "a\nb\x1b[31mc"}}
	for _, c := range []struct {
		name string
		o    []pick.Option
		want string
	}{
		{"default", nil, "> a bc"},
		{"stricter", []pick.Option{pick.WithRules(safetext.Rules{})}, "> abc"},
		{"weaker", []pick.Option{pick.WithRules(safetext.Rules{Newlines: true, KeepSGR: true})}, "> a bc"},
	} {
		h := newHost(t, 40, entries, c.o...)
		if got := h.lines()[1]; got != c.want || strings.Contains(string(h.written), "\x1b[31m") {
			t.Errorf("%s: %q, writing %q; want %q", c.name, got, h.written, c.want)
		}
	}
}

// TestResize: a resize asks where the cursor is, and redraws at the new
// width once the report comes.
func TestResize(t *testing.T) {
	h := newHost(t, 60, models())
	h.scr.Resize(40, inline.ReflowDown)
	h.c.Resize(40, h.now)
	h.drain()
	if len(h.replies) == 0 {
		t.Fatal("the resize asked nothing of the screen")
	}
	h.typeIn("")
	want := newHost(t, 40, models()).lines()
	if got := h.lines()[:len(want)]; !slices.Equal(got, want) {
		t.Errorf("after the resize:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestWithRegion: the Chooser draws in the host's region, with its
// options.
func TestWithRegion(t *testing.T) {
	r := inline.NewRegion(inline.WithoutCursorQuery())
	h := newHost(t, 60, models(), pick.WithRegion(r))
	h.scr.Resize(40, inline.ReflowDown)
	h.c.Resize(40, h.now)
	h.drain()
	if len(h.replies) != 0 {
		t.Errorf("the host's region, made WithoutCursorQuery, asked for the cursor: %q", h.replies)
	}
	want := newHost(t, 40, models()).lines()
	if got := h.lines()[:len(want)]; !slices.Equal(got, want) {
		t.Errorf("redrawn with no report:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestRow1Report: a resize's cursor report on the first row is taken as
// the report, not as a key, though a binding names that key.
func TestRow1Report(t *testing.T) {
	k := userKeys(t, userBinding(t, pick.ChooserScope, "ctrl+f3", pick.AcceptID, false))
	h := newHost(t, 40, models(), pick.WithKeymap(k))
	h.c.Resize(30, h.now)
	h.drain()
	h.replies = nil
	h.typeIn("\x1b[1;5R")
	if h.done != nil {
		t.Errorf("the awaited report reached the keymap as a key: %+v", h.done)
	}
	h.typeIn("\x1b[1;5R")
	if h.done == nil {
		t.Error("ctrl+f3, with no report awaited, did not reach the keymap")
	}
}

// TestCatalog: the Chooser's own words come from the program's catalogue.
func TestCatalog(t *testing.T) {
	cat := msgcat.Table{pick.MsgCount: "%[1]d sur %[2]d", pick.MsgNoResults: "aucun", pick.MsgCanceled: "annulé", pick.MsgCountMore: "%[1]d sur %[2]d et plus"}
	h := newHost(t, 40, models(), pick.WithCatalog(cat))
	if got := h.lines()[0]; !strings.HasSuffix(got, " 1 sur 40") {
		t.Errorf("the count: %q", got)
	}
	h.typeIn("zzqx")
	if got := h.lines()[0]; !strings.HasSuffix(got, " aucun") {
		t.Errorf("no match: %q", got)
	}
	h.typeIn(esc)
	if got := h.lines()[0]; got != "> annulé" {
		t.Errorf("left %q", got)
	}
	if err := msgcat.Check(pick.Messages(), cat); err != nil {
		t.Errorf("Check: %v", err)
	}
}

// TestDefaultRules: the default bindings are the ones the documentation
// names, in ChooserScope.
func TestDefaultRules(t *testing.T) {
	var got []string
	for _, d := range pick.DefaultRules() {
		if d.Scope != pick.ChooserScope {
			t.Errorf("%s in %q; want %q", d.Keys, d.Scope, pick.ChooserScope)
		}
		got = append(got, d.Keys.String()+"="+string(d.Command))
	}
	want := []string{
		"up=pick.select.prev", "ctrl+p=pick.select.prev", "down=pick.select.next", "ctrl+n=pick.select.next",
		"pgup=pick.page.prev", "pgdown=pick.page.next", "enter=pick.accept", "esc=pick.cancel", "ctrl+g=pick.cancel",
	}
	if !slices.Equal(got, want) {
		t.Errorf("DefaultRules = %q; want %q", got, want)
	}
}
