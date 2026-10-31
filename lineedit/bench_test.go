package lineedit_test

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/lineedit"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// The keys the benchmarks type.
var (
	keyX     = termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: 'x', Text: "x"}}
	keyBack  = termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: termkey.KeyBackspace}}
	keyHome  = termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: 'a', Mod: termkey.ModCtrl}}
	keyUp    = termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: termkey.KeyUp}}
	keyDown  = termkey.Input{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: termkey.KeyDown}}
	typeBack = []termkey.Input{keyX, keyBack}
)

// keyComposer is issue #13's Composer: a prompt of "> " in a region 24 rows
// tall, 80 columns wide, holding text, its events drained.
func keyComposer(text string) *lineedit.Composer {
	c := lineedit.NewComposer(lineedit.WithRegion(inline.NewRegion(inline.WithHeight(24))), lineedit.WithPrompt("> "))
	now := time.Unix(1, 0)
	c.Resize(80, now)
	c.SetText(text)
	drain(c)
	return c
}

func drain(c *lineedit.Composer) {
	for e := c.NextEvent(); e.Kind != lineedit.EditNone; e = c.NextEvent() {
	}
}

// keyCase is one of the benchmarks: its Composer, made ready, and the keys
// it takes in turn.
type keyCase struct {
	name string
	make func() *lineedit.Composer
	keys []termkey.Input
}

var keyCases = []keyCase{
	// The issue's: a key typed at the end of three rows and taken back.
	{"KeyDraw", func() *lineedit.Composer { return keyComposer(strings.Repeat("words fill the prompt ", 10)) }, typeBack},
	// A key typed at the start of twenty rows, every row after it moved.
	{"KeyStart", func() *lineedit.Composer {
		c := keyComposer(strings.Repeat("words fill the prompt ", 70))
		c.HandleInput(keyHome, time.Unix(1, 0))
		drain(c)
		return c
	}, typeBack},
	// A key at the end of a text near DefaultLimit.
	{"KeyLong", func() *lineedit.Composer { return keyComposer(strings.Repeat("words fill the prompt ", 180)) }, typeBack},
	// Up and down within three rows.
	{"UpDown", func() *lineedit.Composer { return keyComposer(strings.Repeat("words fill the prompt ", 10)) }, []termkey.Input{keyUp, keyDown}},
}

// benchKeys types c.keys in turn, each a HandleInput and its events
// drained, as a host does.
func benchKeys(b *testing.B, kc keyCase) {
	c, now := kc.make(), time.Unix(1, 0)
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		c.HandleInput(kc.keys[i%len(kc.keys)], now)
		drain(c)
		i++
	}
}

// BenchmarkKeyDraw is issue #13's benchmark: a key typed at the end of a
// three-row text, 80 columns wide and 24 rows tall, then taken back with
// backspace, in turn.
func BenchmarkKeyDraw(b *testing.B) { benchKeys(b, keyCases[0]) }

// BenchmarkKeyStart: a key typed at the start of twenty rows, and taken back.
func BenchmarkKeyStart(b *testing.B) { benchKeys(b, keyCases[1]) }

// BenchmarkKeyLong: a key at the end of a text near DefaultLimit.
func BenchmarkKeyLong(b *testing.B) { benchKeys(b, keyCases[2]) }

// BenchmarkUpDown: up and down within three rows.
func BenchmarkUpDown(b *testing.B) { benchKeys(b, keyCases[3]) }

// A key's budget, gobble-cli's editor's BenchmarkKeyDraw at 6c5eb02: 8
// allocations and 808 bytes a key
// (docs/decisions/0023-MADR-line-mode-components.md A12 item 7).
const (
	keyAllocs = 8
	keyBytes  = 808
)

// TestKeyBudget: each benchmark's key allocates no more, in count or in
// bytes, than gobble-cli's editor's.
func TestKeyBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("lays out texts near DefaultLimit")
	}
	for _, kc := range keyCases {
		c, now := kc.make(), time.Unix(1, 0)
		i := 0
		allocs, bytes := perRun(100, func() {
			c.HandleInput(kc.keys[i%len(kc.keys)], now)
			drain(c)
			i++
		})
		if allocs > keyAllocs || bytes > keyBytes {
			t.Errorf("%s: %.1f allocations and %.0f bytes a key; want at most %d and %d", kc.name, allocs, bytes, keyAllocs, keyBytes)
		}
	}
}

// perRun is f's allocations and bytes allocated a call, over runs calls
// after one to warm it, counted as testing.AllocsPerRun counts them.
func perRun(runs int, f func()) (allocs, bytes float64) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	f()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range runs {
		f()
	}
	runtime.ReadMemStats(&after)
	return float64(after.Mallocs-before.Mallocs) / float64(runs), float64(after.TotalAlloc-before.TotalAlloc) / float64(runs)
}
