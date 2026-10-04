package termcap

import (
	"slices"
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

// setEnv fills the facts the environment gives, with origin Environment:
// the terminal's name, the multiplexer, and whether the session is remote.
// A stronger fact already held stands.
func (c *Caps) setEnv(e Env) {
	if v := e.Getenv("TERM_PROGRAM"); v != "" {
		c.Terminal.Set(v, Environment)
	} else if v := e.Getenv("TERM"); v != "" {
		c.Terminal.Set(v, Environment)
	}

	mux := NoMux
	switch {
	case e.Getenv("TMUX") != "":
		mux = Tmux
	case e.Getenv("STY") != "":
		mux = Screen
	case e.Getenv("ZELLIJ") != "":
		mux = Zellij
	}
	c.Mux.Set(mux, Environment)

	_, tty := e.LookupEnv("SSH_TTY")
	_, conn := e.LookupEnv("SSH_CONNECTION")
	c.Remote.Set(tty || conn, Environment)
}
