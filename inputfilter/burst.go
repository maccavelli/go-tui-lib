package inputfilter

import (
	"runtime"
	"strings"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// pkgName is the package's name, as its enums' errors give it.
const pkgName = "inputfilter"

// The burst thresholds PasteBurst applies where its config leaves them,
// codex's (codex tui/src/bottom_pane/paste_burst.rs, at 4b4a5cf;
// docs/decisions/0009-PLAN-streaming-content-engine.md D11).
const (
	defaultMinChars    = 3
	defaultGap         = 8 * time.Millisecond
	defaultIdle        = 8 * time.Millisecond
	defaultWindowsIdle = 60 * time.Millisecond
)

// PasteBurstConfig is how a PasteBurst tells a paste from typing. A zero
// or negative field takes its default.
type PasteBurstConfig struct {
	// MinChars is how many characters, each following the last within
	// Gap, make a burst: 3 by default. The characters before them are held
	// until it is known.
	MinChars int
	// Gap is how soon a character must follow the last to join a run:
	// 8 ms by default.
	Gap time.Duration
	// Idle is how long a burst waits for its next character before it
	// ends: 8 ms by default, and 60 ms when GOOS is "windows", where
	// pastes come slower.
	Idle time.Duration
	// GOOS is the platform whose timing applies, as runtime.GOOS names
	// it: the one the program was built for when it is empty.
	GOOS string
}

// BurstAction is what a PasteBurst did with an input.
type BurstAction uint8

const (
	// Pass is an input the PasteBurst did not take, with nothing ready
	// before it: the caller handles it. The zero value.
	Pass BurstAction = iota
	// Hold is an input the PasteBurst took, with nothing ready yet.
	Hold
	// Release is an input the PasteBurst took, with inputs ready, in
	// order, which Tick gives until it reports false.
	Release
)

var burstActionNames = enum.Names[BurstAction]{
	Pkg: pkgName, Type: "BurstAction",
	Tokens: []string{"pass", "hold", "release"},
}

// String is the action's token.
func (a BurstAction) String() string { return burstActionNames.String(a) }

// MarshalText is the action's token. An action with no token is an error.
func (a BurstAction) MarshalText() ([]byte, error) { return burstActionNames.Marshal(a) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (a *BurstAction) UnmarshalText(b []byte) error { return burstActionNames.Unmarshal(b, a) }

// PasteBurst tells a paste from typing on a terminal that does not
// bracket its pastes, by its speed, as codex's does
// (docs/decisions/0009-MADR-streaming-content-engine.md §6). Characters
// that follow one another within the gap are held; at MinChars they are a
// burst, which takes every character after it, an Enter as a newline and
// a Tab as a tab, until the burst has been idle, and is then given as one
// termkey.InputPaste. Characters held that do not become a burst are given
// back as the keys they were, once the gap has passed. Any other input
// ends what is held first. A bracketed paste turns detection off for
// good. A PasteBurst reads no clock, and is used on one goroutine.
type PasteBurst struct {
	minChars  int
	gap, idle time.Duration
	altGr     bool // Ctrl+Alt with text is AltGr typing, as on Windows

	held   []termkey.Input // a run of fewer than minChars, not yet a burst
	burst  strings.Builder // the burst's text
	active bool            // a burst is taking characters
	last   time.Time       // when the run's or burst's last character came
	ready  []termkey.Input // given by Tick, in order
	off    bool            // a bracketed paste was seen
}

// NewPasteBurst is a PasteBurst with c's thresholds.
func NewPasteBurst(c PasteBurstConfig) *PasteBurst {
	goos := c.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	p := &PasteBurst{
		minChars: defaultMinChars,
		gap:      defaultGap,
		idle:     defaultIdle,
		altGr:    goos == "windows",
	}
	if goos == "windows" {
		p.idle = defaultWindowsIdle
	}
	if c.MinChars > 0 {
		p.minChars = c.MinChars
	}
	if c.Gap > 0 {
		p.gap = c.Gap
	}
	if c.Idle > 0 {
		p.idle = c.Idle
	}
	return p
}

// Key gives the PasteBurst in at now, and reports what it did; on
// Release, Tick gives what is ready.
func (p *PasteBurst) Key(in termkey.Input, now time.Time) BurstAction {
	p.expire(now)
	switch {
	case p.off:
	case in.Kind == termkey.InputPaste:
		p.off = true
		p.end()
	case p.active && p.joins(in):
		p.burst.WriteString(text(in))
		p.last = now
		return p.taken()
	case p.joins(in) && (len(p.held) > 0 || p.typed(in)): // an Enter or a Tab starts no run
		p.held = append(p.held, in)
		p.last = now
		if len(p.held) >= p.minChars {
			for _, h := range p.held {
				p.burst.WriteString(text(h))
			}
			p.held, p.active = p.held[:0], true
		}
		return p.taken()
	default:
		p.end()
	}
	if len(p.ready) == 0 {
		return Pass
	}
	p.ready = append(p.ready, in)
	return Release
}

// taken is the action for an input the PasteBurst took.
func (p *PasteBurst) taken() BurstAction {
	if len(p.ready) > 0 {
		return Release
	}
	return Hold
}

// Tick gives the next input ready at now, oldest first: what Key
// released, a run that the gap has ended, given back as its keys, and a
// burst that has been idle, as one termkey.InputPaste.
func (p *PasteBurst) Tick(now time.Time) (termkey.Input, bool) {
	p.expire(now)
	if len(p.ready) == 0 {
		return termkey.Input{}, false
	}
	in := p.ready[0]
	p.ready = p.ready[1:]
	if len(p.ready) == 0 {
		p.ready = nil
	}
	return in, true
}

// Due is when Tick next has an input: the zero time when one is ready
// already, and false when nothing is held.
func (p *PasteBurst) Due() (time.Time, bool) {
	switch {
	case len(p.ready) > 0:
		return time.Time{}, true
	case p.active:
		return p.last.Add(p.idle), true
	case len(p.held) > 0:
		return p.last.Add(p.gap), true
	}
	return time.Time{}, false
}

// expire readies what now has ended: a burst idle for its idle time, or a
// run the gap has passed.
func (p *PasteBurst) expire(now time.Time) {
	switch {
	case p.active && now.Sub(p.last) >= p.idle:
		p.end()
	case len(p.held) > 0 && now.Sub(p.last) >= p.gap:
		p.end()
	}
}

// end readies what is held: the burst as one paste, or the run's keys.
func (p *PasteBurst) end() {
	if p.active {
		p.ready = append(p.ready, termkey.Input{Kind: termkey.InputPaste, Text: p.burst.String()})
		p.burst.Reset()
		p.active = false
	}
	p.ready = append(p.ready, p.held...)
	p.held = p.held[:0]
}

// joins reports whether in may join a run or a burst: a character typed,
// an Enter or a Tab.
func (p *PasteBurst) joins(in termkey.Input) bool {
	if p.typed(in) {
		return true
	}
	if in.Kind != termkey.InputKey || in.Key.Release || in.Key.Mod&^locks != 0 {
		return false
	}
	return in.Key.Code == termkey.KeyEnter || in.Key.Code == termkey.KeyTab
}

// typed reports whether in is a character typed: a key press with text,
// shifted at most, or on Windows with AltGr's Ctrl and Alt.
func (p *PasteBurst) typed(in termkey.Input) bool {
	if in.Kind != termkey.InputKey || in.Key.Release || in.Key.Text == "" {
		return false
	}
	mods := in.Key.Mod &^ (locks | termkey.ModShift)
	return mods == 0 || p.altGr && mods == termkey.ModCtrl|termkey.ModAlt
}

// locks is the lock keys' states, which a key's modifiers may carry.
const locks = termkey.ModCapsLock | termkey.ModNumLock | termkey.ModScrollLock

// text is what in adds to a burst.
func text(in termkey.Input) string {
	switch {
	case in.Key.Text != "":
		return in.Key.Text
	case in.Key.Code == termkey.KeyEnter:
		return "\n"
	}
	return "\t"
}
