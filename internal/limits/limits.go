// Package limits holds the largest window the library draws, for workspace
// and launch alike (docs/decisions/0014-PLAN-hardening.md Step 4, finding
// H3). workspace exports the two constants; launch reads them here, so it
// does not import workspace.
//
// Stability: internal.
package limits

// MaxSide is the most cells a window has on either side.
const MaxSide = 4096

// MaxCells is the most cells a window has in all: 524,288, about 56 MiB of
// frame. An 8K display with a 6 × 12 pixel font is about 1280 × 360, or
// 460,800 cells.
const MaxCells = 1 << 19

// Clamp is the size w × h is drawn at: each side put in [0, MaxSide], then
// the height cut until the area is at most MaxCells. The width is kept, so
// lines stay whole, and the frame draws from the top left.
func Clamp(w, h int) (int, int) {
	w = min(max(w, 0), MaxSide)
	h = min(max(h, 0), MaxSide)
	if w > 0 && w*h > MaxCells {
		h = MaxCells / w
	}
	return w, h
}
