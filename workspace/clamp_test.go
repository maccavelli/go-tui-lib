package workspace

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
)

// The window clamp (docs/decisions/0014-PLAN-hardening.md Step 4, finding
// H3). The sizes are written as numbers, not as MaxSide and MaxCells, so
// the test also runs against the code before the clamp.

func TestWindowClamp(t *testing.T) {
	for _, tc := range []struct{ w, h, wantW, wantH int }{
		{1 << 30, 1 << 30, 4096, 128},
		{4096, 4096, 4096, 128},
		{1000, 600, 1000, 524},
		{512, 1024, 512, 1024},
		{100, 0, 100, 0},
		{-5, -7, 0, 0},
		{5000, 10, 4096, 10},
	} {
		w := New(layout.Pane{ID: "a"}, map[layout.PaneID]Pane{"a": &fake{id: "a"}}, WithTheme(asciiTheme))
		w.Update(tea.WindowSizeMsg{Width: tc.w, Height: tc.h})
		if w.width != tc.wantW || w.height != tc.wantH {
			t.Errorf("WindowSizeMsg %d × %d gives %d × %d, want %d × %d", tc.w, tc.h, w.width, w.height, tc.wantW, tc.wantH)
		}
		o := New(layout.Pane{ID: "a"}, map[layout.PaneID]Pane{"a": &fake{id: "a"}}, WithSize(tc.w, tc.h))
		if o.width != tc.wantW || o.height != tc.wantH {
			t.Errorf("WithSize(%d, %d) gives %d × %d, want %d × %d", tc.w, tc.h, o.width, o.height, tc.wantW, tc.wantH)
		}
	}
	// A pane is never told a size past the clamp.
	p := &fake{id: "a"}
	w := New(layout.Pane{ID: "a"}, map[layout.PaneID]Pane{"a": p}, WithTheme(asciiTheme), WithChrome(None))
	w.Update(tea.WindowSizeMsg{Width: 1 << 30, Height: 1 << 30})
	for _, m := range p.msgs {
		if s, ok := m.(SizeMsg); ok && (s.Width > 4096 || s.Width*s.Height > 1<<19) {
			t.Errorf("the pane was told %d × %d", s.Width, s.Height)
		}
	}
	// RenderPlain lays out, and asks for plain views, at a clamped width.
	pw := New(layout.Pane{ID: "a"}, map[layout.PaneID]Pane{"a": widthPane{&fake{id: "a"}}}, WithTheme(asciiTheme))
	if got := pw.RenderPlain(1 << 30); !strings.Contains(got, "plain at 4096\n") {
		t.Errorf("RenderPlain(2^30) = %q, want a view asked for at 4096", got)
	}
}

// widthPane says the width its plain view was asked for.
type widthPane struct{ *fake }

func (widthPane) PlainView(width int) string { return "plain at " + strconv.Itoa(width) }

func TestClampConstants(t *testing.T) {
	if MaxSide != 4096 || MaxCells != 1<<19 {
		t.Errorf("MaxSide %d, MaxCells %d", MaxSide, MaxCells)
	}
}

func TestRenderAtClamp(t *testing.T) {
	if testing.Short() {
		t.Skip("draws a 4096 × 128 frame")
	}
	w := New(layout.SidebarRight("main", "side"), map[layout.PaneID]Pane{
		"main": &fake{id: "main", body: "hello"}, "side": &fake{id: "side"},
	}, WithTheme(asciiTheme))
	w.Update(tea.WindowSizeMsg{Width: MaxSide, Height: MaxSide})
	var out string
	allocs := testing.AllocsPerRun(1, func() {
		w.dirty = true
		out = w.Render()
	})
	lines := strings.Split(out, "\n")
	if len(lines) != MaxCells/MaxSide {
		t.Errorf("%d lines, want %d", len(lines), MaxCells/MaxSide)
	}
	for i, l := range lines {
		if ansi.StringWidth(l) > MaxSide {
			t.Fatalf("line %d is %d cells wide", i, ansi.StringWidth(l))
		}
	}
	t.Logf("a frame at the clamp, %d × %d: %.0f allocations", w.width, w.height, allocs)
}
