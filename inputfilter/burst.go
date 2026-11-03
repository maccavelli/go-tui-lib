package inputfilter

import (
	"runtime"
	"strings"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// The burst thresholds PasteBurst applies where its config leaves them,
// codex's (codex tui/src/bottom_pane/paste_burst.rs, at 4b4a5cf;
// docs/decisions/0009-PLAN-streaming-content-engine.md D11).
const (
	defaultMinChars    = 3
	defaultGap         = 8 * time.Millisecond
	defaultIdle        = 8 * time.Millisecond
	defaultWindowsIdle = 60 * time.Millisecond
)

// goosWindows is Windows' name, as runtime.GOOS gives it.
const goosWindows = "windows"

// PasteBurstConfig is how a PasteBurst tells a paste from typing. A zero
// or negative field takes its default, and each of grok's PowerShell
// rules, the last four, is off at its zero.
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

	// FirstGap is how soon a run's second character must follow its
	// first: Gap when it is zero.
	FirstGap time.Duration
	// LinesOnly gives back a burst with no character after an Enter as
	// the keys it was, unless it is path-shaped, so that only a pasted
	// line break makes a paste.
	LinesOnly bool
	// PathMinChars is, with GOOS "windows", how long a burst that begins
	// with a path's anchor must be to be a paste under LinesOnly: a
	// dropped file's path. Zero turns the rule off.
	PathMinChars int
	// EnterCtrlJ reads an Enter followed by a Ctrl+J, which a raw-mode
	// terminal gives for a pasted CR LF, as one line break, and makes the
	// run a burst at once.
	EnterCtrlJ bool
}

// PowerShellPasteBurst is grok's rules for a terminal that gives pastes as
// keys, as PowerShell's does: the second key within 2 ms, the rest within
// 10 ms, a burst a paste only when it holds a line break or a dropped
// Windows path of 8 or more, and an Enter then a Ctrl+J a CR LF (grok
// xai-grok-pager/src/app/event_loop.rs:3436-3715).
func PowerShellPasteBurst() PasteBurstConfig {
	return PasteBurstConfig{
		MinChars:     3,
		Gap:          10 * time.Millisecond,
		Idle:         10 * time.Millisecond,
		FirstGap:     2 * time.Millisecond,
		LinesOnly:    true,
		PathMinChars: 8,
		EnterCtrlJ:   true,
	}
}

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
	minChars, pathMinChars int
	gap, firstGap, idle    time.Duration
	altGr                  bool // Ctrl+Alt with text is AltGr typing, as on Windows
	linesOnly, enterCtrlJ  bool

	run    []termkey.Input // the inputs taken since the run began
	active bool            // the run is a burst
	last   time.Time       // when the run's last input came
	ready  []termkey.Input // given by Tick, in order
	off    bool            // a bracketed paste was seen
}

// NewPasteBurst is a PasteBurst with c's thresholds.
func NewPasteBurst(c PasteBurstConfig) *PasteBurst {
	goos := c.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	windows := goos == goosWindows
	p := &PasteBurst{
		minChars:   defaultMinChars,
		gap:        defaultGap,
		idle:       defaultIdle,
		altGr:      windows,
		linesOnly:  c.LinesOnly,
		enterCtrlJ: c.EnterCtrlJ,
	}
	if windows {
		p.idle = defaultWindowsIdle
		p.pathMinChars = max(c.PathMinChars, 0)
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
	p.firstGap = p.gap
	if c.FirstGap > 0 {
		p.firstGap = c.FirstGap
	}
	return p
}

// Key gives the PasteBurst in at now, and reports what it did; on
// Release, Tick gives what is ready.
func (p *PasteBurst) Key(in termkey.Input, now time.Time) Action {
	p.expire(now)
	switch {
	case p.off:
	case in.Kind == termkey.InputPaste:
		p.off = true
		p.end()
	case p.joins(in) && (len(p.run) > 0 || p.typed(in)): // an Enter or a Tab starts no run
		p.run = append(p.run, in)
		p.last = now
		if len(p.run) >= p.minChars || p.enterCtrlJ && ctrlJ(in) {
			p.active = true
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
func (p *PasteBurst) taken() Action {
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
	case len(p.run) > 0:
		return p.last.Add(p.wait()), true
	}
	return time.Time{}, false
}

// wait is how long the run waits for its next input: the idle time for a
// burst, the first gap after its first character, and the gap after that.
func (p *PasteBurst) wait() time.Duration {
	switch {
	case p.active:
		return p.idle
	case len(p.run) == 1:
		return p.firstGap
	}
	return p.gap
}

// drop forgets the run and what is ready; detection stays as it was.
func (p *PasteBurst) drop() {
	p.run, p.active, p.ready = p.run[:0], false, nil
}

// expire readies the run when now has passed its wait.
func (p *PasteBurst) expire(now time.Time) {
	if len(p.run) > 0 && now.Sub(p.last) >= p.wait() {
		p.end()
	}
}

// end readies the run: as one paste when it is a burst that LinesOnly
// leaves one, and otherwise as the keys it was.
func (p *PasteBurst) end() {
	if len(p.run) == 0 {
		return
	}
	text, lines := p.text()
	if p.active && (!p.linesOnly || lines || p.pathShaped(text)) {
		p.ready = append(p.ready, termkey.Input{Kind: termkey.InputPaste, Text: text})
	} else {
		p.ready = append(p.ready, p.run...)
	}
	p.run, p.active = p.run[:0], false
}

// text is the run's text, and whether it has a line break LinesOnly
// counts: a character after an Enter, or an Enter and a Ctrl+J.
func (p *PasteBurst) text() (string, bool) {
	var b strings.Builder
	enter, lines := false, false
	for _, in := range p.run {
		switch {
		case ctrlJ(in):
			lines = true // a CR LF, its Enter already a newline
		case in.Key.Text != "":
			b.WriteString(in.Key.Text)
			lines = lines || enter
		case in.Key.Code == termkey.KeyEnter:
			b.WriteByte('\n')
			enter = true
		default:
			b.WriteByte('\t')
			lines = lines || enter
		}
	}
	return b.String(), lines
}

// pathShaped reports whether a burst's text is a dropped Windows path
// long enough for PathMinChars.
func (p *PasteBurst) pathShaped(text string) bool {
	return p.pathMinChars > 0 && len(p.run) >= p.pathMinChars && dropAnchor(text)
}

// joins reports whether in may join a run or a burst: a character typed,
// an Enter or a Tab, and with EnterCtrlJ a Ctrl+J after an Enter.
func (p *PasteBurst) joins(in termkey.Input) bool {
	switch {
	case p.typed(in):
		return true
	case p.enterCtrlJ && ctrlJ(in):
		return len(p.run) > 0 && p.run[len(p.run)-1].Key.Code == termkey.KeyEnter
	case in.Kind != termkey.InputKey || in.Key.Release || in.Key.Mod&^locks != 0:
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

// ctrlJ reports whether in is a Ctrl+J press, which a raw-mode terminal
// gives for a line feed.
func ctrlJ(in termkey.Input) bool {
	return in.Kind == termkey.InputKey && !in.Key.Release && in.Key.Code == 'j' && in.Key.Mod&^locks == termkey.ModCtrl
}

// locks is the lock keys' states, which a key's modifiers may carry.
const locks = termkey.ModCapsLock | termkey.ModNumLock | termkey.ModScrollLock
