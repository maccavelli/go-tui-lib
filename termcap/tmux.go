package termcap

import (
	"strconv"
	"strings"
)

// TmuxFacts is what tmux says about itself, from one run of TmuxQuery.
type TmuxFacts struct {
	Known              bool     `json:"known,omitzero"`                // the output held every field
	Version            string   `json:"version,omitzero"`              // tmux's version, such as "3.4"
	ExtendedKeysFormat string   `json:"extended_keys_format,omitzero"` // "csi-u" or "xterm"
	Mouse              bool     `json:"mouse,omitzero"`                // the mouse option is on
	TermFeatures       []string `json:"term_features,omitzero"`        // client_termfeatures
	ClientFlags        []string `json:"client_flags,omitzero"`         // client_flags, such as "control-mode"
}

// String is the facts on one line, for the report.
func (t TmuxFacts) String() string {
	if !t.Known {
		return "-"
	}
	return "version " + t.Version + ", extended-keys-format " + t.ExtendedKeysFormat +
		", mouse " + strconv.FormatBool(t.Mouse) +
		", features " + strings.Join(t.TermFeatures, ",") +
		", flags " + strings.Join(t.ClientFlags, ",")
}

// tmuxFormat is the fields TmuxQuery asks for, tab-separated, in the order
// ParseTmux reads them.
const tmuxFormat = "#{version}\t#{extended-keys-format}\t#{mouse}\t#{client_termfeatures}\t#{client_flags}"

// TmuxQuery is the argv of one tmux command that prints what ParseTmux
// reads. The library never runs it: the program does, inside the tmux the
// environment names, and passes the facts to Prober.SetTmux
// (docs/decisions/0005-MADR-terminal-capabilities-and-services.md A1).
func TmuxQuery() []string { return []string{"tmux", "display-message", "-p", tmuxFormat} }

// ParseTmux reads TmuxQuery's output. Known is false unless every field is
// there; the fields it found are kept either way.
func ParseTmux(out string) TmuxFacts {
	fields := strings.Split(strings.TrimRight(out, "\r\n"), "\t")
	var t TmuxFacts
	field := func(i int) string {
		if i < len(fields) {
			return strings.TrimSpace(fields[i])
		}
		return ""
	}
	t.Version = field(0)
	t.ExtendedKeysFormat = field(1)
	t.Mouse = field(2) == "1" || field(2) == "on"
	t.TermFeatures = list(field(3))
	t.ClientFlags = list(field(4))
	t.Known = len(fields) == 5 && t.Version != ""
	return t
}

// list splits a comma-separated tmux list, with no empty entries.
func list(s string) []string {
	var out []string
	for f := range strings.SplitSeq(s, ",") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// atLeast reports whether the dotted version v, such as "3.4" or "3.3a", is
// at least major.minor. A version it cannot read is not.
func atLeast(v string, major, minor int) bool {
	ma, mi, _ := strings.Cut(v, ".")
	m, err := strconv.Atoi(ma)
	if err != nil {
		return false
	}
	n, err := strconv.Atoi(strings.TrimRightFunc(mi, func(r rune) bool { return r < '0' || r > '9' }))
	if err != nil {
		n = 0 // "3" alone is 3.0
	}
	return m > major || m == major && n >= minor
}
