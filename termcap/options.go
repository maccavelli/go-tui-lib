package termcap

import (
	"slices"
	"time"
)

// Option configures a Probe, and through it termcaptea's Prober and
// EnvCaps. It is opaque: only this package's functions make one.
type Option interface{ apply(*Probe) }

// optionFunc is a function used as an Option.
type optionFunc func(*Probe)

func (f optionFunc) apply(p *Probe) { f(p) }

// WithTimeout sets how long the probe waits for its sentinel. The default
// is DefaultTimeout.
func WithTimeout(d time.Duration) Option { return optionFunc(func(p *Probe) { p.timeout = d }) }

// WithQuery adds a query to the batch, after the built-in queries and
// before the sentinel.
func WithQuery(q Query) Option { return optionFunc(func(p *Probe) { p.added = append(p.added, q) }) }

// WithoutHeuristic sends the gated queries to every terminal, even those
// the heuristic would spare: Apple Terminal, and SSH peers the environment
// does not name.
func WithoutHeuristic() Option { return optionFunc(func(p *Probe) { p.ungated = true }) }

// WithOverride sets facts the program knows better, from a flag or a file.
// f runs on the facts the probe starts from, and again on every Caps it
// hands out. f should set each fact with origin Override, so that no reply
// replaces it.
func WithOverride(f func(*Caps)) Option {
	return optionFunc(func(p *Probe) { p.overrides = append(p.overrides, f) })
}

// WithoutColorSchemeUpdates never subscribes to mode 2031, so the terminal
// sends no DSR 997 reports after the probe, and Restore is always empty.
func WithoutColorSchemeUpdates() Option { return optionFunc(func(p *Probe) { p.noScheme = true }) }

// WithoutBackgroundRequest leaves the OSC 11 background query out of the
// batch, and out of the follow-up to each DSR 997 report.
func WithoutBackgroundRequest() Option { return optionFunc(func(p *Probe) { p.noBackground = true }) }

// WithDisabled sends nothing: Start writes no query and ends the probe at
// once, with only the environment's facts, and the JetBrains reasons when
// the environment names a JetBrains terminal. EnvCaps gives the same facts
// without a host. A program run without input must not probe: the
// replies would reach the shell.
func WithDisabled() Option { return optionFunc(func(p *Probe) { p.disabled = true }) }

// WithGOOS reads the terminal's identity for goos instead of
// runtime.GOOS: for a test, or for a wish server, whose own operating
// system is not the SSH client's.
func WithGOOS(goos string) Option { return optionFunc(func(p *Probe) { p.goos = goos }) }

// WithAppearanceEnv names an environment variable the program sets to
// "dark" or "light", such as MYAPP_APPEARANCE. It is also read with an LC_
// prefix, because a default sshd forwards LC_* variables. It outranks
// COLORFGBG and the desktop hook, and a terminal's reply outranks it.
func WithAppearanceEnv(name string) Option {
	return optionFunc(func(p *Probe) { p.appearanceEnv = name })
}

// WithAppearanceHook asks the desktop whether it is dark, through f: the
// macOS appearance, the XDG portal or the Windows registry, which need a
// process or an OS binding the library does not take. The probe hands f
// to the host once, on Start, as a ProbeRunHook event; f reports ok false
// when it cannot tell. Its answer outranks COLORFGBG only.
func WithAppearanceHook(f func() (dark, ok bool)) Option {
	return optionFunc(func(p *Probe) { p.appearanceHook = f })
}

// WithConsoleHost asks, on Windows, whether the console is the classic
// console host rather than ConPTY, through f, which a later Windows record
// supplies. The probe hands f to the host once, on Start, as a
// ProbeRunHook event, and only when GOOS is windows; f reports ok false
// when it cannot tell.
func WithConsoleHost(f func() (classic, ok bool)) Option {
	return optionFunc(func(p *Probe) { p.consoleHost = f })
}

// WithoutModeQuery leaves DECRQM for the modes out of the batch: the host
// asks for them itself and passes their reports to HandleReply, which
// reads them as it reads the probe's own. termcaptea passes 2026 and 2027,
// which Bubble Tea asks for.
func WithoutModeQuery(modes ...int) Option {
	return optionFunc(func(p *Probe) {
		for _, m := range modes {
			if !slices.Contains(p.hostModes, m) {
				p.hostModes = append(p.hostModes, m)
			}
		}
	})
}
