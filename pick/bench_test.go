package pick_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/pick"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// BenchmarkKeystroke is one typed key and its drawing, the query changing
// between "so" and "son": at 5000 entries, DefaultIndexAbove, ranked in
// full, and at 100,000, through the index.
func BenchmarkKeystroke(b *testing.B) {
	for _, n := range []int{pick.DefaultIndexAbove, 100_000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			entries := make([]pick.Entry, n)
			for i := range entries {
				entries[i] = pick.Entry{ID: fmt.Sprint(i), Text: fmt.Sprintf("provider-%02d/model-%06d-%s", i%37, i, []string{"sonnet", "haiku", "flash", "mini"}[i%4])}
			}
			now := time.Unix(1, 0)
			c := pick.NewChooser(entries, pick.WithQuery("so"))
			c.Resize(100, now)
			drain := func() {
				for e := c.NextEvent(); e.Kind != pick.PickNone; e = c.NextEvent() {
				}
			}
			drain()
			keys := []termkey.Input{
				{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: 'n', Text: "n"}},
				{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: termkey.KeyBackspace}},
			}
			i := 0
			for b.Loop() {
				c.HandleInput(keys[i%2], now)
				drain()
				i++
			}
		})
	}
}
