package ask_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/ask"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// permission is gobble-cli's tool call question, the owner's example of
// its 0029-MADR-permission-prompt-layout.md.
func permission() ask.Question {
	scopes := func(verb, first string) []ask.Answer {
		var out []ask.Answer
		for i, s := range []string{"once", "session", "project", "everywhere"} {
			echo := verb + " for this " + s
			switch s {
			case "once":
				echo = verb + " once"
			case "everywhere":
				echo = verb + " everywhere"
			}
			out = append(out, ask.Answer{ID: verb + "-" + s, Key: string(rune(first[0] + byte(i))), Label: s, Echo: echo})
		}
		return out
	}
	return ask.Question{
		Title: "Run echo '=== staged ==='; git diff --cached --stat; echo '=== log ==='; git log --oneline -5?",
		After: []string{"Patterns: echo *, git diff *, git log *"},
		Rows: []ask.Row{
			{Label: "Allow", Answers: scopes("allow", "1")},
			{Label: "Reject", Answers: scopes("reject", "5")},
			{Answers: []ask.Answer{{ID: "why", Key: "f", Label: "reject and say why", Echo: "reject", Reason: true}}},
		},
		Cancel: ask.Answer{ID: "cancel", Key: "c", Label: "cancel"},
	}
}

// others are gobble-cli's other questions (its 0029-MADR D2), their paths
// placeholders.
func others() map[string]ask.Question {
	cancel := ask.Answer{ID: "cancel", Key: "c", Label: "cancel"}
	yn := func(yes, no string) []ask.Row {
		return []ask.Row{{Answers: []ask.Answer{{ID: "y", Key: "y", Label: yes}, {ID: "n", Key: "n", Label: no}}}}
	}
	return map[string]ask.Question{
		"trust": {
			Title: "Trust /home/<user>/src/project?",
			Rows: []ask.Row{{Answers: []ask.Answer{
				{ID: "trust", Key: "1", Label: "Trust"},
				{ID: "parent", Key: "2", Label: "Trust the parent folder (/home/<user>/src)"},
				{ID: "no", Key: "3", Label: "Do not trust"},
			}}},
			Cancel: cancel,
		},
		"planmode": {Title: "Leave plan mode and carry out the plan?", Rows: yn("start implementing", "keep planning"), Cancel: cancel},
		"grant": {
			Notes:  []string{"(only once is offered: the grant is for this turn)"},
			Title:  "Grant network access for this session: fetch the release notes of the three modules this change bumps?",
			Rows:   yn("allow once", "deny"),
			Cancel: cancel,
		},
		"toolcall": {
			Notes:  []string{"(the same call was asked for 3 times in a row)"},
			Title:  "Write docs/guides/line-mode.md?",
			Rows:   []ask.Row{{Answers: []ask.Answer{{ID: "y", Key: "y", Label: "allow once"}, {ID: "a", Key: "a", Label: "always"}, {ID: "n", Key: "n", Label: "deny"}, {ID: "N", Key: "N", Label: "never"}}}},
			Cancel: cancel,
		},
	}
}

// TestSketch: the permission question at 66 columns is gobble's sketch,
// row for row.
func TestSketch(t *testing.T) {
	want := []string{
		"Run echo '=== staged ==='; git diff --cached --stat; echo '=== log",
		"  ==='; git log --oneline -5?",
		"Patterns: echo *, git diff *, git log *",
		"  Allow   [1] once  [2] session  [3] project  [4] everywhere",
		"  Reject  [5] once  [6] session  [7] project  [8] everywhere",
		"  [f] reject and say why   [c] cancel",
		">",
	}
	got := ask.Layout(permission(), 66, ansi.WcWidth, glyph.ASCII())
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("at 66 columns:\n%s\nwant gobble's sketch:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestLayoutGolden: each question drawn on the test screen, as an Asker
// draws it, across the matrix; the permission question at gobble's 177
// too.
func TestLayoutGolden(t *testing.T) {
	render := func(q ask.Question) func(tuitest.Case) string {
		return func(c tuitest.Case) string {
			h := newHost(t, c.Width, q, ask.WithGlyphs(c.Glyphs()))
			text := h.scr.Text()
			tuitest.Fits(t, text, c.Width, ansi.WcWidth)
			return text
		}
	}
	tuitest.Golden(t, "permission", tuitest.Matrix{Widths: []int{60, 100, 177}}, render(permission()))
	for name, q := range others() {
		tuitest.Golden(t, name, tuitest.Matrix{Widths: []int{60, 100}}, render(q))
	}
}

// TestLayoutBreaks: a row of answers too long breaks between answers, its
// continuation under its first answer; a word wider than the width is
// broken at it, onto rows of its own; an answer wider than any row is on a
// row of its own; no row is wider than the width.
func TestLayoutBreaks(t *testing.T) {
	q := ask.Question{
		Title: "Read /home/<user>/a-very-long-path-that-is-wider-than-the-row/file.txt now?",
		Rows: []ask.Row{
			{Label: "Allow", Answers: []ask.Answer{{Key: "1", Label: "once"}, {Key: "2", Label: "for this session"}, {Key: "3", Label: "for this project"}}},
			{Label: "Other", Answers: []ask.Answer{{Key: "4", Label: "a label far wider than the whole row it is on"}}},
		},
		Cancel: ask.Answer{Key: "c", Label: "cancel"},
	}
	want := []string{
		"Read",
		"  /home/<user>/a-very-long-pat",
		"  h-that-is-wider-than-the-row",
		"  /file.txt now?",
		"  Allow  [1] once",
		"         [2] for this session",
		"         [3] for this project",
		"  Other",
		"         [4] a label far wider",
		"          than the whole row i",
		"         t is on   [c] cancel",
		">",
	}
	got := ask.Layout(q, 30, ansi.WcWidth, glyph.ASCII())
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("at 30 columns:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, row := range got {
		if w := ansi.WcWidth.StringWidth(row); w > 30 {
			t.Errorf("%q is %d cells, in 30", row, w)
		}
	}
}

// TestLayoutCleaned: escape sequences and line breaks in the caller's
// words are not drawn.
func TestLayoutCleaned(t *testing.T) {
	q := ask.Question{
		Title: "Run \x1b]0;pwned\x07rm\x1b[2J -rf\nx?",
		Rows:  []ask.Row{{Label: "\x1b[31mAllow", Answers: []ask.Answer{{Key: "1\x1b[A", Label: "once\r\n"}}}},
	}
	got := strings.Join(ask.Layout(q, 60, ansi.WcWidth, glyph.ASCII()), "\n")
	if want := "Run rm -rf x?\n  Allow  [1] once\n>"; got != want {
		t.Errorf("cleaned: %q; want %q", got, want)
	}
}

// TestLayoutWide: a wide cluster is measured as the method measures it,
// and never split.
func TestLayoutWide(t *testing.T) {
	q := ask.Question{Title: "世界世界世界 👨‍👩‍👧 done?"}
	for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		for _, row := range ask.Layout(q, 7, m, glyph.ASCII()) {
			if w := m.StringWidth(row); w > 7 {
				t.Errorf("method %d: %q is %d cells, in 7", m, row, w)
			}
			if strings.Contains(row, "\u200d") && !strings.Contains(row, "👨\u200d👩\u200d👧") {
				t.Errorf("method %d: %q splits the ZWJ sequence", m, row)
			}
		}
	}
}
