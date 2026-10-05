package termsvc

import (
	"net/url"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// Display is how a link is shown.
type Display uint8

const (
	// LabelOnly shows the label as an OSC 8 link.
	LabelOnly Display = iota
	// LabelAndURL shows the label and the URL beside it, as text a
	// terminal can still detect, where OSC 8 links do not work.
	LabelAndURL
)

// LinkDisplay is how links should be shown on c's terminal: the label only
// where OSC 8 links work, and the label with its URL on Apple Terminal,
// Warp, an unknown terminal, inside screen or Zellij, and inside tmux
// before 3.4 (termcap.Caps.Links).
func LinkDisplay(c termcap.Caps) Display {
	if c.Links().OSC8.Value == termcap.Supported {
		return LabelOnly
	}
	return LabelAndURL
}

// LinkPolicy decides whether a click may open a link.
type LinkPolicy struct {
	// Schemes are the schemes that open, compared without case. Empty means
	// http and https.
	Schemes []string
}

// Openable reports whether rawURL may be opened: it holds no control
// character and its scheme is one the policy names.
func (p LinkPolicy) Openable(rawURL string) bool {
	if strip(rawURL) != rawURL || strings.ContainsAny(rawURL, " ") {
		return false
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" {
		return false
	}
	schemes := p.Schemes
	if len(schemes) == 0 {
		schemes = []string{"http", "https"}
	}
	return slices.ContainsFunc(schemes, func(s string) bool { return strings.EqualFold(s, u.Scheme) })
}

// OpenURLMsg asks the program to open URL. The library never starts a
// process; the program opens it.
type OpenURLMsg struct{ URL string }

// Open returns the command that delivers OpenURLMsg for rawURL, or nil
// when the policy refuses it.
func (p LinkPolicy) Open(rawURL string) tea.Cmd {
	if !p.Openable(rawURL) {
		return nil
	}
	return func() tea.Msg { return OpenURLMsg{URL: rawURL} }
}
