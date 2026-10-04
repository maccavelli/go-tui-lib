package termcap

import (
	"slices"
	"strconv"
	"strings"
)

// Env is a program's environment, as KEY=value strings. It is converted from
// tea.EnvMsg, Env(msg), and never read from the process: under wish,
// tea.EnvMsg is the SSH session's environment, not the server's.
type Env []string

// LookupEnv is the value of key, and whether it is set. When key appears
// more than once, the last wins, as in tea's own environment.
func (e Env) LookupEnv(key string) (string, bool) {
	for _, kv := range slices.Backward(e) {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			return v, true
		}
	}
	return "", false
}

// Getenv is the value of key, or "" when it is not set.
func (e Env) Getenv(key string) string {
	v, _ := e.LookupEnv(key)
	return v
}

// setEnv fills the facts the environment gives on goos, with origin
// Environment unless the fact is a guess: the terminal's name and brand,
// the multiplexer, the editor, the platform, whether the session is remote,
// whether this is the legacy Windows console, and light or dark from
// COLORFGBG and from the variable appearanceEnv names. A stronger fact
// already held stands.
func (c *Caps) setEnv(e Env, goos, appearanceEnv string) {
	if v := e.Getenv("TERM_PROGRAM"); v != "" {
		c.Terminal.Set(v, Environment)
	} else if v := e.Getenv("TERM"); v != "" {
		c.Terminal.Set(v, Environment)
	}

	id := FromEnv(e, goos)
	c.Mux.Set(id.Mux, Environment)
	c.Remote.Set(id.Remote, Environment)
	c.Editor.Set(id.Editor, Environment)
	c.Platform.Set(id.Platform, Environment)
	c.EnvBrand.Set(id.EnvBrand, Environment)
	if id.Brand != id.EnvBrand {
		c.Brand.SetReason(id.Brand, Heuristic, ReasonWindowsTerminalGuess)
	} else {
		c.Brand.Set(id.Brand, Environment)
	}

	// Bare cmd.exe sets no terminal variable, so on Windows a terminal the
	// environment does not name may be the classic console host, until a
	// WithConsoleHost hook can ask.
	if goos == goosWindows && id.EnvBrand == BrandUnknown {
		c.LegacyConsole.SetReason(true, Heuristic, ReasonLegacyConsoleGuess)
	} else if goos == goosWindows {
		c.LegacyConsole.Set(false, Heuristic)
	} else {
		c.LegacyConsole.Set(false, Environment)
	}

	// The appearance chain's environment links, weakest first: COLORFGBG,
	// then the program's own variable, also read with an LC_ prefix,
	// because a default sshd forwards LC_*.
	if dark, ok := colorFGBG(e.Getenv("COLORFGBG")); ok {
		c.Dark.SetReason(dark, Heuristic, ReasonColorFGBGGuess)
	}
	if appearanceEnv != "" {
		for _, k := range []string{"LC_" + appearanceEnv, appearanceEnv} {
			switch strings.ToLower(e.Getenv(k)) {
			case "dark":
				c.Dark.Set(true, Environment)
			case "light":
				c.Dark.Set(false, Environment)
			}
		}
	}
}

// colorFGBG reads COLORFGBG ("fg;bg", or "fg;x;bg") with Vim's heuristic:
// a background of 0-6 or 8 is dark, 7 or 9-15 light. "default", or
// anything else, is unknown.
func colorFGBG(v string) (dark, ok bool) {
	if v == "" {
		return false, false
	}
	bg := v[strings.LastIndexByte(v, ';')+1:]
	n, err := strconv.Atoi(bg)
	if err != nil || n < 0 || n > 15 {
		return false, false
	}
	return n <= 6 || n == 8, true
}
