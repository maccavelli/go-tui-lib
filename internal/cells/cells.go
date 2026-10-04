// Package cells is a reusable cell buffer that draws strings into
// rectangles with a chosen width method. It is the only package that imports
// github.com/charmbracelet/ultraviolet, which has no tagged release: keeping
// every use here means an upstream change is one package's fix
// (docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md §1). A
// depguard rule refuses the import anywhere else.
//
// The drawing is lipgloss's own: lipgloss's Canvas is a uv.ScreenBuffer, and
// a Layer draws with uv.NewStyledString(s).Draw. A Frame is the same buffer
// without the layer and compositor around it, and with a width method the
// caller sets, where lipgloss's canvas fixes grapheme widths.
package cells

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/layout"
)

// Frame is a reusable cell buffer with a width method. Its zero value is not
// usable; call NewFrame.
type Frame struct {
	scr uv.ScreenBuffer
}

// NewFrame returns a w × h frame that measures with m.
func NewFrame(w, h int, m ansi.Method) *Frame {
	f := &Frame{scr: uv.NewScreenBuffer(max(w, 0), max(h, 0))}
	f.scr.Method = m
	return f
}

// Resize makes the frame w × h. It does nothing when the size is unchanged,
// so a caller may call it before every frame.
func (f *Frame) Resize(w, h int) {
	w, h = max(w, 0), max(h, 0)
	if f.scr.Width() == w && f.scr.Height() == h {
		return
	}
	f.scr.Resize(w, h)
}

// Width is the frame's width in cells.
func (f *Frame) Width() int { return f.scr.Width() }

// Height is the frame's height in cells.
func (f *Frame) Height() int { return f.scr.Height() }

// SetMethod sets how the frame measures the strings drawn after it.
func (f *Frame) SetMethod(m ansi.Method) { f.scr.Method = m }

// Method is how the frame measures what is drawn on it.
func (f *Frame) Method() ansi.Method { return f.scr.Method }

// Clear empties every cell.
func (f *Frame) Clear() { f.scr.Clear() }

// Draw clears r, then prints s into it, clipped to it. s may carry ANSI
// styles and newlines. Cells outside r are untouched.
func (f *Frame) Draw(s string, r layout.Rect) {
	uv.NewStyledString(s).Draw(f.scr, uv.Rect(r.X, r.Y, r.W, r.H))
}

// Render is the frame as a styled string, one line per row, with trailing
// spaces trimmed.
func (f *Frame) Render() string { return uv.TrimSpace(f.scr.Render()) }
