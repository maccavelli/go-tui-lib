// Package termsvctea turns termsvc's services into Bubble Tea commands
// (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md D3, D5
// and A1.2): a notification, a copy, opening a link and the activity
// beacon's tick, each as a tea.Cmd that delivers termsvc's message, and
// Observe, which gives a Notifier the capabilities and focus a program's
// messages carry.
//
// Each command decides on the caller's goroutine, through termsvc's
// Prepare and PrepareCopy, and calls a backend or clipboard the program
// supplies only inside the command, off the program's loop.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package termsvctea

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/teamsg"
	"github.com/maccavelli/go-tui-lib/v2/termcap"
	"github.com/maccavelli/go-tui-lib/v2/termsvc"
)

// Observe gives n what msg tells it: the terminal's capabilities from
// termcap.CapsMsg, and its focus from tea.FocusMsg and tea.BlurMsg, which
// need the program's View.ReportFocus. A program passes it every message.
func Observe(n *termsvc.Notifier, msg tea.Msg) {
	switch m := msg.(type) {
	case termcap.CapsMsg:
		n.SetCaps(m.Caps)
	case tea.FocusMsg:
		n.SetFocus(true)
	case tea.BlurMsg:
		n.SetFocus(false)
	}
}

// Notify returns the command that sends x through n, and delivers
// termsvc.NotifyResultMsg to say what it did. It is NotifyContext under
// context.Background().
func Notify(n *termsvc.Notifier, x termsvc.Notification) tea.Cmd {
	return NotifyContext(context.Background(), n, x)
}

// NotifyContext is Notify, with a backend called under ctx, bounded by
// termsvc.WithBackendTimeout. The notification is decided now, by
// n.Prepare; the command writes its bytes, then delivers the result, and
// calls a backend only when it runs.
func NotifyContext(ctx context.Context, n *termsvc.Notifier, x termsvc.Notification) tea.Cmd {
	seq, deliver := n.Prepare(ctx, x)
	result := func() tea.Msg { return deliver() }
	if seq == "" {
		return result
	}
	return tea.Sequence(tea.Raw(seq), result)
}

// Copy writes text to the clipboard, and delivers termsvc.CopiedMsg. With
// termsvc.WithClipboard it goes through the program's clipboard; otherwise
// through OSC 52, and inside tmux also wrapped for passthrough. A payload
// over termsvc.MaxCopyBytes is not sent, and fails. It is CopyContext under
// context.Background().
func Copy(c termcap.Caps, text string, o ...termsvc.CopyOption) tea.Cmd {
	return CopyContext(context.Background(), c, text, o...)
}

// CopyContext is Copy, with the clipboard called under ctx, bounded by
// termsvc.WithCopyTimeout. The route is decided now, by
// termsvc.PrepareCopy; OSC 52 goes out through tea's own clipboard
// command.
func CopyContext(ctx context.Context, c termcap.Caps, text string, o ...termsvc.CopyOption) tea.Cmd {
	route, deliver := termsvc.PrepareCopy(ctx, c, text, o...)
	result := func() tea.Msg { return deliver() }
	switch route {
	case termsvc.RouteOSC52:
		return tea.Sequence(tea.SetClipboard(text), result)
	case termsvc.RouteOSC52Tmux:
		return tea.Sequence(
			tea.Batch(tea.SetClipboard(text), tea.Raw(termsvc.Wrap(c, ansi.SetSystemClipboard(text)))),
			result)
	}
	return result
}

// Open returns the command that delivers termsvc.OpenURLMsg for rawURL, or
// nil when p refuses it.
func Open(p termsvc.LinkPolicy, rawURL string) tea.Cmd {
	if !p.Openable(rawURL) {
		return nil
	}
	return teamsg.Cmd(termsvc.OpenURLMsg{URL: rawURL})
}

// ActivityBeacon is the beacon's timer: it delivers termsvc.ActivityTickMsg
// after termsvc.ActivityInterval. A program that opts in writes
// termsvc.Activity on each tick and returns ActivityBeacon again.
func ActivityBeacon() tea.Cmd {
	return tea.Tick(termsvc.ActivityInterval, func(t time.Time) tea.Msg { return termsvc.ActivityTickMsg{At: t} })
}
