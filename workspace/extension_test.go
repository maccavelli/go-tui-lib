package workspace

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter/inputfiltertea"
)

// ownMouse is a program's own mouse message, which local translates by
// its WithMouse.
type ownMouse struct{ m tea.Mouse }

func (o ownMouse) Mouse() tea.Mouse              { return o.m }
func (o ownMouse) String() string                { return "own" }
func (o ownMouse) WithMouse(t tea.Mouse) tea.Msg { return ownMouse{m: t} }

// bareMouse is a mouse message with no WithMouse, which local drops.
type bareMouse struct{ m tea.Mouse }

func (o bareMouse) Mouse() tea.Mouse { return o.m }
func (o bareMouse) String() string   { return "bare" }

// TestMouseExtensionPoint: a CoalescedWheelMsg over a pane arrives in the
// pane's cells, with its notches; so does a program's own message with
// WithMouse; one without it is dropped
// (docs/decisions/0009-MADR-streaming-content-engine.md §6).
func TestMouseExtensionPoint(t *testing.T) {
	r := newRig(t, 120, 40)
	x, y := r.at("logs", 3, 2)
	r.w.Update(inputfiltertea.CoalescedWheelMsg{Pointer: tea.Mouse{X: x, Y: y, Button: tea.MouseWheelDown}, DY: 7})
	got := filter[inputfiltertea.CoalescedWheelMsg](r.logs.msgs)
	if len(got) != 1 || got[0].Pointer.X != 3 || got[0].Pointer.Y != 2 || got[0].DY != 7 {
		t.Fatalf("logs got %v; want one coalesced wheel at (3,2) with 7 notches", got)
	}
	if len(filter[inputfiltertea.CoalescedWheelMsg](r.side.msgs)) != 0 {
		t.Fatal("the coalesced wheel reached a pane not under the pointer")
	}
	r.w.Update(ownMouse{m: tea.Mouse{X: x, Y: y}})
	if own := filter[ownMouse](r.logs.msgs); len(own) != 1 || own[0].m.X != 3 || own[0].m.Y != 2 {
		t.Fatalf("logs got %v; want the program's own message at (3,2)", own)
	}
	r.w.Update(bareMouse{m: tea.Mouse{X: x, Y: y}})
	if bare := filter[bareMouse](r.logs.msgs); len(bare) != 0 {
		t.Fatalf("a mouse message with no WithMouse reached the pane: %v", bare)
	}
}
