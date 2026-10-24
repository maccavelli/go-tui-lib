// Package keymaptea is keymap's Bubble Tea form: a Dispatcher driven by
// tea's key messages, with tea.Tick for a pending sequence's timeout; the
// conversions between tea.Key and termkey.KeyEvent; and help and
// key.Binding for bubbles
// (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md D7,
// A1.2). keymap itself links no Bubble Tea.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package keymaptea

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// Update takes a message, the focus path and the when values, as a
// program's Update does. A tea.KeyPressMsg goes to d.Key, and a
// tea.KeyReleaseMsg to d.Key as a release; a keymap.TimeoutMsg goes to
// d.Expire; any other message gives a zero Outcome. When the Outcome
// waits, the command is the tea.Tick that brings its Timeout back.
func Update(d *keymap.Dispatcher, msg tea.Msg, p keymap.Path, ctx when.Context) (keymap.Outcome, tea.Cmd) {
	var o keymap.Outcome
	switch m := msg.(type) {
	case tea.KeyPressMsg:
		o = d.Key(FromTea(m.Key()), p, ctx)
	case tea.KeyReleaseMsg:
		k := FromTea(m.Key())
		k.Release = true
		o = d.Key(k, p, ctx)
	case keymap.TimeoutMsg:
		o = d.Expire(m)
	default:
		return keymap.Outcome{}, nil
	}
	if o.Wait <= 0 {
		return o, nil
	}
	timeout := o.Timeout
	return o, tea.Tick(o.Wait, func(time.Time) tea.Msg { return timeout })
}

// FromTea is k as termkey's key, field for field: the codes and the
// modifier bits have the same values. The result is a press; a release is
// its message's type in Bubble Tea, which Update reads.
func FromTea(k tea.Key) termkey.KeyEvent {
	return termkey.KeyEvent{
		Code:        k.Code,
		Mod:         termkey.Mod(k.Mod), //nolint:gosec // tea's modifier bits fit in 16
		Text:        k.Text,
		ShiftedCode: k.ShiftedCode,
		BaseCode:    k.BaseCode,
		IsRepeat:    k.IsRepeat,
	}
}

// ToTea is k as tea's key, field for field, so that a replayed key reaches
// a pane as the tea.KeyPressMsg it came in as. A release is not a field of
// tea.Key: the caller chooses the message.
func ToTea(k termkey.KeyEvent) tea.Key {
	return tea.Key{
		Code:        k.Code,
		Mod:         tea.KeyMod(k.Mod),
		Text:        k.Text,
		ShiftedCode: k.ShiftedCode,
		BaseCode:    k.BaseCode,
		IsRepeat:    k.IsRepeat,
	}
}
