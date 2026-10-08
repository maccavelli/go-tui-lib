package launch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/tuitest"
	"github.com/maccavelli/go-tui-lib/workspace"
)

// text is a pane that shows its lines.
type text struct{ lines []string }

func (p text) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return p, nil }
func (p text) View(_, _ int) string                     { return strings.Join(p.lines, "\n") }

// board is the program around a workspace.
type board struct{ ws *workspace.Workspace }

func (b board) Init() tea.Cmd                           { return b.ws.Init() }
func (b board) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return b, b.ws.Update(msg) }
func (b board) View() tea.View                          { return tea.NewView(b.ws.Render()) }

// TestFrameGolden draws a three-pane workspace once, as a plain status
// board. The tree is fixed, not a responsive preset, so every case shows the
// three panes. The workspace starts with a TrueColor theme and follows the
// profile Frame sends, so a no-colour case is plain only if Frame sends it.
func TestFrameGolden(t *testing.T) {
	tuitest.Golden(t, "frame-workspace", tuitest.Matrix{Widths: []int{60, 100}}, func(c tuitest.Case) string {
		p := colorprofile.ASCII
		if c.Color {
			p = colorprofile.TrueColor
		}
		g := glyph.For(c.UTF8)
		build := func(p colorprofile.Profile, bg theme.Background) theme.Theme { return theme.New(p, bg, g) }
		root := layout.Split{Axis: layout.Vertical, Children: []layout.Child{
			{Node: layout.Split{Axis: layout.Horizontal, Children: []layout.Child{
				{Node: layout.Pane{ID: "main"}},
				{Node: layout.Pane{ID: "side"}, Size: layout.Fixed(18)},
			}}},
			{Node: layout.Pane{ID: "bottom"}, Size: layout.Fixed(3)},
		}}
		ws := workspace.New(
			root,
			map[layout.PaneID]workspace.Pane{
				"main":   text{[]string{"build 1832 passed", "deploy to staging: done"}},
				"side":   text{[]string{"3 agents", "0 failing"}},
				"bottom": text{[]string{"last run 12:04"}},
			},
			workspace.WithTheme(theme.New(colorprofile.TrueColor, theme.Dark, g)),
			workspace.WithThemeBuilder(build),
		)
		return Frame(board{ws}, c.Width, 12, p)
	})
}

// commands is a model that records whether Init ran, or any command it
// returned did.
type commands struct{ init, ran *bool }

func (m commands) Init() tea.Cmd {
	*m.init = true
	return func() tea.Msg { *m.ran = true; return nil }
}

func (m commands) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, func() tea.Msg { *m.ran = true; return nil }
}

func (m commands) View() tea.View { return tea.NewView("drawn") }

func TestFrameRunsNoCommand(t *testing.T) {
	var init, ran bool
	if got := Frame(commands{&init, &ran}, 20, 2, colorprofile.ASCII); got != "drawn" {
		t.Errorf("Frame = %q", got)
	}
	if init || ran {
		t.Errorf("Frame called Init (%v) or ran a command (%v)", init, ran)
	}
}

// exitCoder is a program's own error with a status.
type exitCoder struct{}

func (exitCoder) Error() string { return "custom" }
func (exitCoder) ExitCode() int { return 42 }

func TestExitCode(t *testing.T) {
	for _, c := range []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"ErrNotStarted", ErrNotStarted, 2},
		{"tea.ErrInterrupted", tea.ErrInterrupted, 130},
		{"context.DeadlineExceeded", context.DeadlineExceeded, 124},
		{"context.Canceled", context.Canceled, 130},
		{"tea.ErrProgramPanic", tea.ErrProgramPanic, 2},
		{"another error", errors.New("x"), 1},
		{"ExitError", &ExitError{Code: 7}, 7},
		{"a type with ExitCode", exitCoder{}, 42},
	} {
		if got := ExitCode(c.err); got != c.want {
			t.Errorf("%s: ExitCode = %d, want %d", c.name, got, c.want)
		}
		if c.err == nil {
			continue
		}
		if got := ExitCode(fmt.Errorf("wrapped: %w", c.err)); got != c.want {
			t.Errorf("%s, wrapped: ExitCode = %d, want %d", c.name, got, c.want)
		}
	}

	// As Run returns them.
	crash := fmt.Errorf("%w: %w", ErrCrashed, fmt.Errorf("%w: %w", tea.ErrProgramKilled, tea.ErrProgramPanic))
	if got := ExitCode(crash); got != 2 {
		t.Errorf("a crash from a panic: %d, want 2", got)
	}
	if got := ExitCode(fmt.Errorf("%w: %w", tea.ErrProgramKilled, context.Canceled)); got != 130 {
		t.Errorf("a cancel: %d, want 130", got)
	}

	// An ExitError inside ErrCrashed, around a panic: the ExitError wins.
	inside := fmt.Errorf("%w: %w", ErrCrashed, &ExitError{Code: 7, Err: tea.ErrProgramPanic})
	if got := ExitCode(inside); got != 7 {
		t.Errorf("an ExitError inside a crash: %d, want 7", got)
	}
}

func TestExitError(t *testing.T) {
	cause := errors.New("cause")
	e := &ExitError{Code: 3, Err: cause}
	if e.Error() != "cause" || !errors.Is(e, cause) || e.ExitCode() != 3 {
		t.Errorf("ExitError{3, cause} = %q, Is %v, %d", e.Error(), errors.Is(e, cause), e.ExitCode())
	}
	if got := (&ExitError{Code: 4}).Error(); got != "exit status 4" {
		t.Errorf("ExitError{4} = %q", got)
	}
	if (&ExitError{Code: 4}).Unwrap() != nil {
		t.Error("an ExitError with no Err unwraps to something")
	}
}
