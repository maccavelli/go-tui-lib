package launch

import (
	"io"
	"runtime"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/internal/enum"
	"github.com/maccavelli/go-tui-lib/internal/limits"
)

// Config is how a program lets the TUI reach the terminal.
type Config struct {
	// Choice is the program's request, from its flag.
	Choice Choice
	// NoInputEnv names the program's own variable that forbids the TUI
	// when it is set and not empty, as gh's GH_PROMPT_DISABLED does. Empty
	// names none.
	NoInputEnv string
	// UIOnErr lets the TUI draw on Err when Out is not a terminal, so that
	// x=$(prog pick) captures only the result.
	UIOnErr bool
	// OpenTTY lets the TUI read from and draw on the controlling terminal
	// when In or Out is not one. It is not supported on Windows, where
	// Decide ignores it: a read of the console handle launch would open
	// cannot be cancelled there, so it would take the CLI's next line
	// (docs/decisions/0013-MADR-cli-integration-helpers.md A1.9).
	OpenTTY bool
}

// Target is where the TUI reads or draws.
type Target uint8

// The targets. TargetNone is the zero value: a plain decision has no
// target.
const (
	// TargetNone is no stream: the decision is plain.
	TargetNone Target = iota
	// TargetStream is the program's own stream: In for reading, Out for
	// drawing.
	TargetStream
	// TargetErr is Err, for drawing, which Config.UIOnErr allows.
	TargetErr
	// TargetTTY is the controlling terminal, which Config.OpenTTY allows.
	TargetTTY
)

var targetNames = []string{"none", "stream", "err", "tty"}

// String is the target's token: "none", "stream", "err" or "tty".
func (t Target) String() string { return enum.Name(targetNames, t) }

// MarshalText is the target's token. A target with no token is an error.
func (t Target) MarshalText() ([]byte, error) {
	return enum.Marshal("launch", "Target", targetNames, t)
}

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (t *Target) UnmarshalText(b []byte) error {
	return enum.Unmarshal("launch", "Target", targetNames, b, t)
}

// Reason is why a decision fell as it did, as a stable token in termcap's
// "area.thing" style, for a log line or a warning.
type Reason string

// The reasons, in the order Decide's rules give them.
const (
	// ReasonRequestedPlain: Config.Choice is ChoicePlain.
	ReasonRequestedPlain Reason = "launch.requested-plain"
	// ReasonNoInputVariable: the variable Config.NoInputEnv names is set
	// and not empty.
	ReasonNoInputVariable Reason = "launch.no-input-variable"
	// ReasonCI: CI is set, not empty, and neither "false" nor "0".
	ReasonCI Reason = "launch.ci"
	// ReasonDumbTerminal: TERM is dumb, which has no cursor movement.
	ReasonDumbTerminal Reason = "launch.dumb-terminal"
	// ReasonInputNotTerminal: In is not a terminal, and no controlling
	// terminal was allowed or could be opened.
	ReasonInputNotTerminal Reason = "launch.input-not-terminal"
	// ReasonOutputNotTerminal: Out is not a terminal, and neither Err nor
	// the controlling terminal was allowed or available.
	ReasonOutputNotTerminal Reason = "launch.output-not-terminal"
	// ReasonTerminal: the decision is interactive.
	ReasonTerminal Reason = "launch.terminal"
)

// Decision is what Decide found: whether the TUI can start, where it reads
// and draws, and the colour, glyphs and size for it or for plain output.
type Decision struct {
	// Interactive is true when the TUI can start.
	Interactive bool
	// Reason is why.
	Reason Reason
	// In and UI are where the TUI reads and draws. Both are TargetNone
	// when the decision is plain.
	In, UI Target
	// Profile is the colour profile for Out: the TUI's result, or the
	// program's plain output.
	Profile colorprofile.Profile
	// UIProfile is the colour profile for the stream the TUI draws on, and
	// NoTTY when the decision is plain.
	UIProfile colorprofile.Profile
	// Glyphs is the glyph tier the locale allows.
	Glyphs glyph.Tier
	// Width and Height are in cells: the size of the stream the TUI draws
	// on, or, for a plain decision, Out's size when it is a terminal, else
	// COLUMNS when that is a positive integer, with Height 0. Each is 0
	// when unknown; the program picks its own fallback. Both are clamped
	// as workspace clamps a window: to workspace.MaxSide, then to
	// workspace.MaxCells.
	Width, Height int
}

// Decide decides whether the TUI can start on s, under c. It reads s's
// environment and the terminal state of its streams, and writes nothing.
// When c.OpenTTY allows it and a stream needs it, it opens the controlling
// terminal to learn whether there is one, and its size, and closes it at
// once; one that does not open, or does not close cleanly, is not used.
// The rules, in order:
//  1. ChoicePlain: plain.
//  2. Unless ChoiceTUI, the environment can veto: Config.NoInputEnv's
//     variable set and not empty, then CI set, not empty, and neither
//     "false" nor "0".
//  3. TERM=dumb: plain, whatever the choice.
//  4. Input: In if it is a terminal, else the controlling terminal if
//     allowed and it opens, else plain.
//  5. Drawing: Out if it is a terminal, else Err if UIOnErr allows it and
//     it is a terminal, else the controlling terminal if allowed and it
//     opens, else plain.
//  6. Otherwise: interactive.
func Decide(s Streams, c Config) Decision { return decideWith(s, c, runtime.GOOS, openTTY) }

// opener opens the controlling terminal: its input and its output, which
// may be one file.
type opener func() (in io.ReadCloser, out io.WriteCloser, err error)

// openTTY is Bubble Tea's: /dev/tty on Unix, CONIN$ and CONOUT$ on Windows.
func openTTY() (io.ReadCloser, io.WriteCloser, error) {
	in, out, err := tea.OpenTTY()
	if err != nil {
		return nil, nil, err
	}
	return in, out, nil
}

// tty is the controlling terminal, probed at most once, and only when a
// target needs it.
type tty struct {
	allowed, tried, ok bool
	open               opener
	w, h               int
}

// get probes the terminal on first use, and reports whether it can be used.
func (t *tty) get() bool {
	if t.allowed && !t.tried {
		t.tried = true
		t.ok = t.probe()
	}
	return t.ok
}

// probe opens the terminal, reads its size, and closes each distinct file it
// opened, once. A terminal that does not open, or does not close cleanly,
// is not used.
func (t *tty) probe() bool {
	in, out, err := t.open()
	if err != nil {
		return false
	}
	t.w, t.h = size(out)
	errIn := in.Close()
	var errOut error
	if any(out) != any(in) {
		errOut = out.Close()
	}
	return errIn == nil && errOut == nil
}

// decideWith is Decide for goos, with open as the controlling terminal.
func decideWith(s Streams, c Config, goos string, open opener) Decision {
	outTerm := isTerminal(s.Out)
	d := Decision{Profile: profile(outTerm, s.Env), Glyphs: tier(s.Env, goos)}
	plain := func(r Reason) Decision {
		d.Interactive, d.Reason = false, r
		d.In, d.UI = TargetNone, TargetNone
		d.UIProfile = colorprofile.NoTTY
		if outTerm {
			d.Width, d.Height = size(s.Out)
		} else {
			d.Width, d.Height = columns(s), 0
		}
		d.Width, d.Height = limits.Clamp(d.Width, d.Height)
		return d
	}

	if c.Choice == ChoicePlain {
		return plain(ReasonRequestedPlain)
	}
	if c.Choice != ChoiceTUI {
		if c.NoInputEnv != "" && s.Env.Getenv(c.NoInputEnv) != "" {
			return plain(ReasonNoInputVariable)
		}
		if v := s.Env.Getenv("CI"); v != "" && v != "false" && v != "0" {
			return plain(ReasonCI)
		}
	}
	if s.Env.Getenv("TERM") == "dumb" {
		return plain(ReasonDumbTerminal)
	}

	t := &tty{allowed: c.OpenTTY && goos != "windows", open: open}
	switch {
	case isTerminal(s.In):
		d.In = TargetStream
	case t.get():
		d.In = TargetTTY
	default:
		return plain(ReasonInputNotTerminal)
	}
	switch {
	case outTerm:
		d.UI = TargetStream
		d.Width, d.Height = size(s.Out)
	case c.UIOnErr && isTerminal(s.Err):
		d.UI = TargetErr
		d.Width, d.Height = size(s.Err)
	case t.get():
		d.UI = TargetTTY
		d.Width, d.Height = t.w, t.h
	default:
		return plain(ReasonOutputNotTerminal)
	}
	d.Interactive, d.Reason = true, ReasonTerminal
	d.UIProfile = profile(true, s.Env)
	d.Width, d.Height = limits.Clamp(d.Width, d.Height)
	return d
}

// columns is COLUMNS when it is a positive integer, and 0 otherwise.
func columns(s Streams) int {
	if n, err := strconv.Atoi(s.Env.Getenv("COLUMNS")); err == nil && n > 0 {
		return n
	}
	return 0
}
