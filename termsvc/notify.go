package termsvc

import (
	"context"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/internal/sanitize"
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
	// Auto picks from the terminal's capabilities, Caps.Notifications: OSC
	// 99 when it answered the OSC 99 query, else OSC 9 or OSC 777 by
	// brand, else the bell.
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
	// UnlessFocused sends unless the terminal is known to be focused: when
	// it is unfocused, and when focus is unknown, as on terminals that do
	// not report it (MADR A1, Q6).
	UnlessFocused
)

// SkipReason is why Notify sent nothing.
type SkipReason uint8

const (
	// NotSkipped means the notification was sent.
	NotSkipped SkipReason = iota
	// SkipDisabled means the policy is Never, or the protocol Off.
	SkipDisabled
	// SkipEmpty means nothing was left of the text after cleaning.
	SkipEmpty
	// SkipFocused means the terminal has focus.
	SkipFocused
	// SkipFocusUnknown means WhenUnfocused, and no focus report has arrived.
	SkipFocusUnknown
	// SkipGated means the WithGate function said no.
	SkipGated
	// SkipFailed means the backend returned an error, in Err.
	SkipFailed
)

var skipNames = []string{"", "disabled", "empty", "focused", "focus-unknown", "gated", "failed"}

// String is the reason's name, "" when nothing was skipped.
func (s SkipReason) String() string {
	if int(s) < len(skipNames) {
		return skipNames[s]
	}
	return strconv.Itoa(int(s))
}

// NotifyResultMsg reports what Notify did. Sent says the notification
// left: written to the terminal, which does not confirm it, or accepted by
// the backend.
type NotifyResultMsg struct {
	Notification Notification
	Sent         bool
	Skipped      SkipReason
	Err          error // the backend's error, with SkipFailed
}

// Backend is a notifier the program supplies, such as a desktop API. The
// library never starts a process itself.
type Backend interface {
	Notify(ctx context.Context, x Notification) error
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

// WithNotifyFilter lets f decide, last, whether a notification is sent,
// such as only for a completed turn. The policy for that belongs to the
// program.
func WithNotifyFilter(f func(Notification) bool) NotifyOption {
	return func(n *Notifier) { n.gate = f }
}

// WithGate is WithNotifyFilter.
//
// Deprecated: use WithNotifyFilter; a gate is command's permission concept
// (0014-MADR W4).
func WithGate(f func(Notification) bool) NotifyOption { return WithNotifyFilter(f) }

// defaultBackendTimeout bounds a backend's Notify and a clipboard's Copy.
const defaultBackendTimeout = 5 * time.Second

// WithBackendTimeout bounds each call of the backend: its context ends d
// after the call starts, or when the caller's context ends, if sooner.
// The default is 5 s; d of 0 or less leaves only the caller's context.
func WithBackendTimeout(d time.Duration) NotifyOption {
	return func(n *Notifier) { n.timeout = d }
}

// Text limits, in cells.
const (
	TitleCells = 80
	BodyCells  = 240
)

// Notifier sends notifications that suit the terminal. A program passes it
// every message, as it does a termcap.Prober, so it learns the terminal's
// capabilities and focus.
//
// A program's own command line, which runs no Bubble Tea program, never
// sees focus: it passes WithPolicy(Always), gives the notifier
// termcap.EnvCaps through Update(termcap.CapsMsg{Caps: …}), and writes
// Sequence's bytes to the terminal itself
// (docs/decisions/0014-MADR-native-integration-api.md W2).
type Notifier struct {
	protocol Protocol
	policy   Policy
	backend  Backend
	gate     func(Notification) bool
	timeout  time.Duration

	caps       termcap.Caps
	focused    bool
	focusKnown bool
	next       int // numbers notifications without an ID
}

// NewNotifier returns a Notifier.
func NewNotifier(o ...NotifyOption) *Notifier {
	n := &Notifier{timeout: defaultBackendTimeout}
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

// Notify returns the command that sends x, and delivers NotifyResultMsg
// to say what it did. The title and body are cleaned first: escape
// sequences removed, line breaks collapsed to a space, controls removed,
// and cut to TitleCells and BodyCells cells. It is NotifyContext under
// context.Background().
func (n *Notifier) Notify(x Notification) tea.Cmd {
	return n.NotifyContext(context.Background(), x)
}

// NotifyContext is Notify, with a backend called under ctx, bounded by
// WithBackendTimeout. A backend that fails, its deadline included, gives
// SkipFailed with its error in Err.
func (n *Notifier) NotifyContext(ctx context.Context, x Notification) tea.Cmd {
	x.Title, x.Body = clean(x.Title, TitleCells), clean(x.Body, BodyCells)
	if n.backend != nil {
		if skip := n.skip(x, false); skip != NotSkipped {
			return result(NotifyResultMsg{Notification: x, Skipped: skip})
		}
		b, d := n.backend, n.timeout
		return func() tea.Msg {
			ctx, cancel := bounded(ctx, d)
			defer cancel()
			if err := b.Notify(ctx, x); err != nil {
				return NotifyResultMsg{Notification: x, Skipped: SkipFailed, Err: err}
			}
			return NotifyResultMsg{Notification: x, Sent: true}
		}
	}
	seq, skip := n.bytesFor(x)
	if skip != NotSkipped {
		return result(NotifyResultMsg{Notification: x, Skipped: skip})
	}
	return tea.Sequence(tea.Raw(seq), result(NotifyResultMsg{Notification: x, Sent: true}))
}

// Sequence is the bytes Notify would write to the terminal for x, cleaned
// and encoded as Notify does, with the same numbering, or "" and why it
// would send nothing. It is for a program that writes to the terminal
// itself, such as its own command line. It ignores WithBackend: the bytes
// are the terminal's.
func (n *Notifier) Sequence(x Notification) (string, SkipReason) {
	x.Title, x.Body = clean(x.Title, TitleCells), clean(x.Body, BodyCells)
	return n.bytesFor(x)
}

// bytesFor is Sequence for a cleaned x.
func (n *Notifier) bytesFor(x Notification) (string, SkipReason) {
	if skip := n.skip(x, true); skip != NotSkipped {
		return "", skip
	}
	return n.encode(x), NotSkipped
}

// bounded is ctx, ending d from now as well when d is positive.
func bounded(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d > 0 {
		return context.WithTimeout(ctx, d)
	}
	return context.WithCancel(ctx)
}

func result(m NotifyResultMsg) tea.Cmd { return func() tea.Msg { return m } }

// skip is why x is not to be sent, or NotSkipped: to the terminal, or
// else through the backend.
func (n *Notifier) skip(x Notification, terminal bool) SkipReason {
	switch {
	case n.policy == Never || terminal && n.chosen() == Off:
		return SkipDisabled
	case x.Title == "" && x.Body == "":
		return SkipEmpty
	case n.policy == WhenUnfocused && !n.focusKnown:
		return SkipFocusUnknown
	case (n.policy == WhenUnfocused || n.policy == UnlessFocused) && n.focusKnown && n.focused:
		return SkipFocused
	case n.gate != nil && !n.gate(x):
		return SkipGated
	}
	return NotSkipped
}

// chosen is the protocol Notify uses now: the one set, or for Auto, the
// one the terminal's capabilities pick.
func (n *Notifier) chosen() Protocol {
	if n.protocol != Auto {
		return n.protocol
	}
	v := n.caps.Notifications()
	switch termcap.Supported {
	case v.OSC99.Value:
		return OSC99
	case v.OSC9.Value:
		return OSC9
	case v.OSC777.Value:
		return OSC777
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
	return sanitize.Token(id, func(r rune) bool {
		return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_+.", r)
	})
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
