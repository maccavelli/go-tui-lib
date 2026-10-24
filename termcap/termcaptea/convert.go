package termcaptea

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/termevent"
	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// reply is msg as the probe reads it, when msg is a reply: the messages
// tea decodes itself, the replies ultraviolet decodes and tea passes on,
// and through termcap.ParseReply any sequence neither decoded. A host that
// decodes the same reply twice in two ways must agree with ParseReply;
// TestCoreAndAdapterAgree holds the two together.
func reply(msg tea.Msg) (termcap.Reply, bool) {
	switch m := msg.(type) {
	case tea.BackgroundColorMsg:
		return termcap.Reply{Kind: termcap.ReplyColor, OSC: 11, Color: m.Color}, true
	case tea.ForegroundColorMsg:
		return termcap.Reply{Kind: termcap.ReplyColor, OSC: 10, Color: m.Color}, true
	case tea.KeyboardEnhancementsMsg:
		return termcap.Reply{Kind: termcap.ReplyKittyKeyboard, Flags: m.Flags}, true
	case tea.ModeReportMsg:
		// The probe asks about DEC modes, CSI ? Ps $ p, only.
		if d, ok := m.Mode.(ansi.DECMode); ok {
			return termcap.Reply{Kind: termcap.ReplyMode, Mode: int(d), Setting: termcap.ModeSetting(m.Value)}, true
		}
		return termcap.Reply{}, false
	case tea.TerminalVersionMsg:
		return termcap.Reply{Kind: termcap.ReplyVersion, Text: m.Name}, true
	}
	ev, ok := termevent.Decode(msg)
	if !ok {
		return termcap.Reply{}, false
	}
	switch ev.Kind {
	case termevent.DeviceAttributes:
		return termcap.Reply{Kind: termcap.ReplyPrimaryAttributes, Params: ev.Attrs}, true
	case termevent.SecondaryDeviceAttributes:
		return termcap.Reply{Kind: termcap.ReplySecondaryAttributes, Params: ev.Attrs}, true
	case termevent.ColorScheme:
		return termcap.Reply{Kind: termcap.ReplyColorScheme, Dark: ev.Dark}, true
	case termevent.KittyGraphics:
		return termcap.Reply{Kind: termcap.ReplyKittyGraphics, Text: ev.Raw}, true
	case termevent.Unknown:
		return termcap.ParseReply(ev.Raw), true
	}
	return termcap.Reply{}, false
}
