package theme_test

import (
	"sync"
	"testing"

	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/theme"
)

// TestID: each New stamps a new, non-zero ID; a copy keeps it, changed or
// not; a literal Theme has 0
// (docs/decisions/0009-MADR-streaming-content-engine.md A7 item 4).
func TestID(t *testing.T) {
	a := theme.New(colorprofile.TrueColor, theme.Dark, glyph.Unicode())
	b := theme.New(colorprofile.TrueColor, theme.Dark, glyph.Unicode())
	if a.ID() == 0 || b.ID() == 0 || a.ID() == b.ID() {
		t.Errorf("two News: IDs %d and %d; want two different, neither 0", a.ID(), b.ID())
	}
	c := a
	c.Glyphs = glyph.ASCII()
	if c.ID() != a.ID() {
		t.Errorf("a changed copy: ID %d; want its original's %d", c.ID(), a.ID())
	}
	if id := (theme.Theme{Glyphs: glyph.ASCII()}).ID(); id != 0 {
		t.Errorf("a literal: ID %d; want 0", id)
	}
}

// TestIDConcurrent: News on many goroutines, under -race, stamp IDs that
// are all different.
func TestIDConcurrent(t *testing.T) {
	const goroutines, each = 8, 100
	var (
		mu   sync.Mutex
		seen = map[uint64]bool{}
		wg   sync.WaitGroup
	)
	for range goroutines {
		wg.Go(func() {
			for range each {
				id := theme.New(colorprofile.ASCII, theme.Unknown, glyph.ASCII()).ID()
				mu.Lock()
				seen[id] = true
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if len(seen) != goroutines*each || seen[0] {
		t.Errorf("%d News: %d distinct IDs, 0 among them %v; want %d, without 0", goroutines*each, len(seen), seen[0], goroutines*each)
	}
}
