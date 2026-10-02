package layout

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
)

// StateVersion is the version State is written with.
const StateVersion = 1

// ErrStateVersion is returned for a State of a version this package does not
// know.
var ErrStateVersion = errors.New("layout: unknown state version")

// State is what a user changed about a layout. It is plain data: a program
// stores it as JSON and hands it back to Solve, which re-applies it and
// re-clamps it to the bounds at every size. The zero State changes nothing.
type State struct {
	// Version is StateVersion, or 0 for a State that was never written.
	Version int `json:"version"`
	// Resize moves a named split's separator: a positive delta gives cells
	// from the child after the separator to the child before it. A host
	// stores the delta Plan.Resize reports as applied, not the one it asked
	// for.
	Resize map[string]int `json:"resize,omitempty"`
	// Hidden lists the panes the user has hidden. WithHidden keeps it
	// sorted, so its JSON is stable.
	Hidden []PaneID `json:"hidden,omitempty"`
	// Zoom, when set, gives that pane the whole area.
	Zoom PaneID `json:"zoom,omitempty"`
}

func (s State) valid() error {
	if s.Version != 0 && s.Version != StateVersion {
		return fmt.Errorf("%w: %d", ErrStateVersion, s.Version)
	}
	return nil
}

// UnmarshalJSON refuses a version it does not know.
func (s *State) UnmarshalJSON(b []byte) error {
	type plain State
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	if err := State(p).valid(); err != nil {
		return err
	}
	*s = State(p)
	return nil
}

// IsHidden reports whether the user hid pane id.
func (s State) IsHidden(id PaneID) bool {
	return slices.Contains(s.Hidden, id)
}

// WithHidden returns a copy of s with pane id hidden or shown.
func (s State) WithHidden(id PaneID, hidden bool) State {
	out := s.clone()
	out.Hidden = slices.DeleteFunc(out.Hidden, func(h PaneID) bool { return h == id })
	if hidden {
		out.Hidden = append(out.Hidden, id)
	}
	slices.Sort(out.Hidden)
	return out
}

// WithResize returns a copy of s with delta added to separator sep's
// resize.
func (s State) WithResize(sep string, delta int) State {
	out := s.clone()
	if out.Resize == nil {
		out.Resize = map[string]int{}
	}
	out.Resize[sep] += delta
	if out.Resize[sep] == 0 {
		delete(out.Resize, sep)
	}
	if len(out.Resize) == 0 {
		out.Resize = nil // as JSON decoding leaves it
	}
	return out
}

// WithZoom returns a copy of s with pane id zoomed, or with no zoom when id
// is empty.
func (s State) WithZoom(id PaneID) State {
	out := s.clone()
	out.Zoom = id
	return out
}

func (s State) clone() State {
	out := s
	out.Version = StateVersion
	out.Hidden = slices.Clone(s.Hidden)
	if s.Resize != nil {
		out.Resize = make(map[string]int, len(s.Resize))
		maps.Copy(out.Resize, s.Resize)
	}
	return out
}
