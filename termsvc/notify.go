package termsvc

import (
	"context"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// Notification is a desktop notification.
type Notification struct {
	Title, Body string
	ID          string  // groups updates to one notification (OSC 99)
	Urgency     Urgency // OSC 99 only
}

// Urgency is how urgent a notification is.
type Urgency uint8

const (
	// Normal is the default urgency.
	Normal Urgency = iota
	// Low is below normal.
	Low
	// Critical is above normal.
	Critical
)

// Protocol is how a notification reaches the terminal.
type Protocol uint8

const (
	// Auto picks from the terminal's capabilities: OSC 99 when it answered
	// the OSC 99 query, else OSC 777 or OSC 9 on terminals known to show
	// them, else the bell.
	Auto Protocol = iota
	// OSC99 is the Kitty desktop notifications protocol.
	OSC99
	// OSC777 is rxvt's notify extension.
	OSC777
	// OSC9 is iTerm2's notification sequence.
	OSC9
	// Bell rings the bell, and shows no text.
	Bell
	// Off sends nothing.
	Off
)

// Policy is when a notification is sent.
type Policy uint8

const (
	// WhenUnfocused sends only while the terminal is known to be unfocused.
	// Focus comes from tea.FocusMsg and tea.BlurMsg, which need the
	// program's View.ReportFocus; with neither seen, nothing is sent.
	WhenUnfocused Policy = iota
	// Always sends whatever the focus.
	Always
	// Never sends nothing.
	Never
)

// Backend is a notifier the program supplies, such as a desktop API. The
// library never starts a process itself.
type Backend interface {
	Notify(ctx context.Context, x Notification) error
}

// NotifyErrorMsg reports a Backend's error.
type NotifyErrorMsg struct {
	Notification Notification
	Err          error
}

// NotifyOption configures a Notifier.
type NotifyOption func(*Notifier)

// WithProtocol sets the protocol. The default is Auto.
func WithProtocol(p Protocol) NotifyOption { return func(n *Notifier) { n.protocol = p } }

// WithPolicy sets when notifications are sent. The default is
// WhenUnfocused.
func WithPolicy(p Policy) NotifyOption { return func(n *Notifier) { n.policy = p } }

// WithBackend sends notifications through b instead of the terminal.
func WithBackend(b Backend) NotifyOption { return func(n *Notifier) { n.backend = b } }

// Notifier sends notifications that suit the terminal. A program passes it
// every message, as it does a termcap.Prober, so it learns the terminal's
// capabilities and focus.
type Notifier struct {
	protocol Protocol
	policy   Policy
	backend  Backend

	caps       termcap.Caps
	focused    bool
	focusKnown bool
	next       int // numbers notifications without an ID
}

// NewNotifier returns a Notifier.
func NewNotifier(o ...NotifyOption) *Notifier {
	n := &Notifier{}
	for _, f := range o {
		f(n)
	}
	return n
}

// Update observes termcap.CapsMsg, tea.FocusMsg and tea.BlurMsg. It sends
// nothing.
func (n *Notifier) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case termcap.CapsMsg:
		n.caps = m.Caps
	case tea.FocusMsg:
		n.focused, n.focusKnown = true, true
	case tea.BlurMsg:
		n.focused, n.focusKnown = false, true
	}
	return nil
}

// Notify returns the command that sends x, or nil when the policy, the
// protocol or an empty notification says no.
func (n *Notifier) Notify(x Notification) tea.Cmd {
	switch n.policy {
	case Never:
		return nil
	case WhenUnfocused:
		if !n.focusKnown || n.focused {
			return nil
		}
	}
	x.Title, x.Body = strip(x.Title), strip(x.Body)
	if x.Title == "" && x.Body == "" {
		return nil
	}
	if n.backend != nil {
		b := n.backend
		return func() tea.Msg {
			if err := b.Notify(context.Background(), x); err != nil {
				return NotifyErrorMsg{Notification: x, Err: err}
			}
			return nil
		}
	}
	seq := n.encode(x)
	if seq == "" {
		return nil
	}
	return tea.Raw(seq)
}

// chosen is the protocol Notify uses now: the one set, or for Auto, the
// one the terminal's capabilities pick.
func (n *Notifier) chosen() Protocol {
	if n.protocol != Auto {
		return n.protocol
	}
	if n.caps.DesktopNotify.Value == termcap.Supported {
		return OSC99
	}
	term := strings.ToLower(n.caps.Terminal.Value)
	for _, name := range []string{"ghostty", "foot", "vte"} {
		if strings.Contains(term, name) {
			return OSC777
		}
	}
	for _, name := range []string{"iterm", "wezterm", "warp"} {
		if strings.Contains(term, name) {
			return OSC9
		}
	}
	return Bell
}

// encode is x's sequence in the protocol Notify uses, wrapped for the
// multiplexer; the bell is not wrapped, since a multiplexer passes it on.
func (n *Notifier) encode(x Notification) string {
	var seq string
	switch n.chosen() {
	case OSC99:
		seq = n.osc99(x)
	case OSC777:
		// Fields are separated by ';', so the text must not hold one.
		seq = ansi.URxvtExt("notify", strings.ReplaceAll(x.Title, ";", ","), strings.ReplaceAll(x.Body, ";", ","))
	case OSC9:
		seq = ansi.Notify(osc9Text(x))
	case Bell:
		return string(rune(ansi.BEL))
	default:
		return ""
	}
	return Wrap(n.caps, seq)
}

// osc99 is x in the Kitty protocol: the title, then the body, as chunks of
// one notification tied by its identifier.
func (n *Notifier) osc99(x Notification) string {
	id := osc99ID(x.ID)
	if id == "" {
		n.next++
		id = "n" + strconv.Itoa(n.next)
	}
	meta := []string{"i=" + id}
	if x.Urgency != Normal {
		meta = append(meta, "u="+map[Urgency]string{Low: "0", Critical: "2"}[x.Urgency])
	}
	if x.Body == "" {
		return ansi.DesktopNotification(x.Title, meta...)
	}
	if x.Title == "" {
		return ansi.DesktopNotification(x.Body, append(meta, "p=body")...)
	}
	return ansi.DesktopNotification(x.Title, append(meta, "d=0")...) +
		ansi.DesktopNotification(x.Body, "i="+id, "p=body")
}

// osc99ID keeps the characters the Kitty protocol allows in an identifier.
func osc99ID(id string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', strings.ContainsRune("-_+.", r):
			return r
		}
		return -1
	}, id)
}

// osc9Text is x as one OSC 9 message. A message that starts with a number
// and ';' would read as an OSC 9 sub-command, such as 9;4 progress, so it
// gets a leading space.
func osc9Text(x Notification) string {
	s := x.Title
	if x.Body != "" {
		if s != "" {
			s += ": "
		}
		s += x.Body
	}
	if i := strings.IndexByte(s, ';'); i > 0 {
		if _, err := strconv.Atoi(s[:i]); err == nil {
			s = " " + s
		}
	}
	return s
}
