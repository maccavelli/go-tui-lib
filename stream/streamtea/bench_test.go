package streamtea_test

import (
	"fmt"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/stream/streamtea"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// BenchmarkView: a View of 24 rows once the rows are kept costs the same
// at 10,000 rows as at 100 (0009-PLAN Step 6: within 2×).
func BenchmarkView(b *testing.B) {
	for _, n := range []int{100, 10_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			d := stream.NewDoc(echo)
			d.Append(numbered(n))
			w := workspace.New(layout.Pane{ID: "doc"}, map[layout.PaneID]workspace.Pane{"doc": placeholder{}},
				workspace.WithTheme(asciiTheme), workspace.WithoutBackgroundQuery())
			p := streamtea.NewDocPane(w, d)
			p.Update(workspace.SizeMsg{Width: 80, Height: 24})
			p.View(80, 24)
			b.ReportAllocs()
			for b.Loop() {
				p.View(80, 24)
			}
		})
	}
}
