package workspace

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/tuitest"
)

// Titles, badges and views pass through sanitize
// (docs/decisions/0014-PLAN-hardening.md Step 2).

// evilPane has a title with a line break and a badge with controls, an
// escape sequence and a bidirectional override.
func evilPane() *fake {
	return &fake{id: "evil", title: "evil\nline", badge: "b\r\x1b[2J\u202e", body: "body \u202etext\x1b"}
}

func TestTitleWithLineBreakKeepsBorder(t *testing.T) {
	tuitest.Golden(t, "evil-title", tuitest.Matrix{Widths: []int{30, 60}}, func(c tuitest.Case) string {
		var out []string
		for _, chrome := range []Chrome{Borders, Separators} {
			w := New(layout.Pane{ID: "evil"}, map[layout.PaneID]Pane{"evil": evilPane()},
				WithTheme(caseTheme(c)), WithChrome(chrome))
			w.Update(tea.WindowSizeMsg{Width: c.Width, Height: 4})
			frame := w.Render()
			lines := strings.Split(frame, "\n")
			if len(lines) != 4 {
				t.Errorf("%s: %d lines, want 4", c.Name(), len(lines))
			}
			for i, l := range lines {
				// The frame drops a line's trailing blanks, so only a
				// bordered line is exactly the width; none is wider.
				if got := ansi.StringWidth(l); got > c.Width || chrome == Borders && got != c.Width {
					t.Errorf("%s, line %d: %d cells, want %d", c.Name(), i, got, c.Width)
				}
				if strings.ContainsRune(ansi.Strip(l), '\u202e') {
					t.Errorf("%s, line %d holds U+202E", c.Name(), i)
				}
			}
			if chrome == Borders && !strings.HasSuffix(ansi.Strip(lines[0]), caseTheme(c).Border(w.border).TopRight) {
				t.Errorf("%s: the top border is broken: %q", c.Name(), ansi.Strip(lines[0]))
			}
			out = append(out, frame)
		}
		return strings.Join(out, "\n")
	})
	plain := New(layout.Pane{ID: "evil"}, map[layout.PaneID]Pane{"evil": evilPane()}, WithTheme(asciiTheme)).RenderPlain(40)
	if want := "evil line [b ]\nbody text\n\n"; plain != want {
		t.Errorf("RenderPlain = %q, want %q", plain, want)
	}
}

func TestClipSanitizes(t *testing.T) {
	for _, tc := range []struct {
		in   string
		w, h int
		want string
	}{
		{"\x1b", 3, 1, "   "},
		{"a\x1b[1mb\x1b[m", 4, 1, "a\x1b[1mb\x1b[m  "},
		{"a\u202eb\x07c", 3, 1, "abc"},
		{"a\tb", 3, 1, "a b"},
		{"x\x1b[31", 2, 1, "x "},
	} {
		got := clip(ansi.WcWidth, tc.in, tc.w, tc.h)
		if got != tc.want {
			t.Errorf("clip(%q, %d, %d) = %q, want %q", tc.in, tc.w, tc.h, got, tc.want)
		}
		for l := range strings.SplitSeq(got, "\n") {
			if ansi.StringWidth(l) != tc.w {
				t.Errorf("clip(%q): a line %d cells wide, want %d", tc.in, ansi.StringWidth(l), tc.w)
			}
		}
	}
}
