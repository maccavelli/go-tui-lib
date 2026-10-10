package termcap

import (
	"reflect"

	"github.com/maccavelli/go-tui-lib/internal/enum"
)

// Disposition is how a finding should be read.
type Disposition uint8

const (
	// Recommendation is advice: the terminal works, with less.
	Recommendation Disposition = iota
	// Issue is something wrong that the user can fix.
	Issue
)

var dispositionNames = enum.Names[Disposition]{
	Pkg: pkgName, Type: "Disposition",
	Tokens: []string{"recommendation", "issue"},
}

// String is the disposition's name, as the report and JSON print it.
func (d Disposition) String() string { return dispositionNames.String(d) }

// MarshalText is the disposition's name.
func (d Disposition) MarshalText() ([]byte, error) {
	return dispositionNames.Marshal(d)
}

// UnmarshalText reads a name MarshalText wrote.
func (d *Disposition) UnmarshalText(t []byte) error {
	return dispositionNames.Unmarshal(t, d)
}

// Finding is one thing the doctor tells the user: a reason token, what it
// means, and what would change it. Fix is text, never an action.
type Finding struct {
	ID          string      `json:"id"` // a reason token
	Disposition Disposition `json:"disposition"`
	Message     string      `json:"message"`
	Fix         string      `json:"fix"`
}

// noFix is the fix where nothing needs doing.
const noFix = "None needed."

// finding is a Finding's text, by its token.
type finding struct {
	d            Disposition
	message, fix string
}

// findingText says what each reason token means, and what would change it.
var findingText = map[string]finding{
	ReasonKittyUnsupported: {Recommendation,
		"The terminal did not answer the Kitty keyboard query, so keys such as shift+enter may not be told apart.",
		"Use a terminal with the Kitty keyboard protocol, such as kitty, Ghostty, WezTerm, foot or iTerm2."},
	ReasonKittyUnknown: {Issue,
		"The Kitty keyboard query had no answer before the probe ended.",
		"Run again; if it persists, give the prober a longer timeout."},
	ReasonLeaksReleases: {Recommendation,
		"This terminal reports releases of shortcuts it handles itself, so key releases are not asked for.",
		"None needed: bindings that wait for a release fall back to presses."},
	ReasonAlacrittyRelease: {Recommendation,
		"Alacritty 0.14 or older sends a duplicate release when event types are on, so they are not asked for.",
		"Upgrade Alacritty past 0.14."},
	ReasonTmuxExtendedKeys: {Issue,
		"tmux does not pass key event types.",
		"Set extended-keys on and extended-keys-format csi-u in tmux.conf, and pass TmuxQuery's output to the prober."},
	ReasonMSYSNoKitty: {Recommendation,
		"mintty and MSYS2 want no Kitty keyboard protocol, which Bubble Tea still asks for.",
		"None in this library; a later terminal-mode record can turn it off."},
	ReasonWSLDeadKeys: {Recommendation,
		"Under WSL, keyboard enhancement breaks dead keys in VS Code, and is not trusted in an unnamed terminal.",
		"None in this library; a later terminal-mode record can turn it off."},
	ReasonAppleNoOSC8: {Recommendation,
		"Apple Terminal mishandles OSC 8 links, so links are shown with their URL.",
		"Use a terminal that supports OSC 8 links."},
	ReasonWarpNoOSC8: {Recommendation,
		"Warp shows no OSC 8 links, so links are shown with their URL.",
		"Use a terminal that supports OSC 8 links."},
	ReasonTmuxLinks: {Issue,
		"tmux before 3.4 drops hyperlinks, so links are shown with their URL.",
		"Upgrade tmux to 3.4 or later, and pass TmuxQuery's output to the prober."},
	ReasonMuxNoLinks: {Recommendation,
		"This multiplexer passes no hyperlinks, so links are shown with their URL.",
		"Run outside the multiplexer, or in tmux 3.4 or later."},
	ReasonUnknownTerminal: {Recommendation,
		"Nothing names the terminal, so capabilities that depend on it are off.",
		"Set TERM_PROGRAM, or let LC_TERMINAL through SSH."},
	ReasonZellijNoForwarding: {Recommendation,
		"Zellij passes no notification sequence, so notifications ring the bell.",
		noFix},
	ReasonColorFGBGGuess: {Recommendation,
		"Light or dark is guessed from COLORFGBG.",
		"Use a terminal that answers DSR 996 or OSC 11, or set the program's appearance variable."},
	ReasonDesktopAppearance: {Recommendation,
		"Light or dark is the desktop's setting, not the terminal's.",
		noFix},
	ReasonLegacyConsoleGuess: {Issue,
		"On Windows nothing names the terminal, so it is taken for the legacy console host, which lacks most capabilities.",
		"Run in Windows Terminal."},
	ReasonConPTYAnswers: {Recommendation,
		"Over SSH into Windows, ConPTY answers the queries itself, so these facts describe ConPTY, not your terminal.",
		"None in this library; run the program on the machine with the terminal to see the terminal's own capabilities."},
	ReasonWindowsTerminalGuess: {Recommendation,
		"The terminal is taken for Windows Terminal, whose hand-off omits WT_SESSION.",
		noFix},
	ReasonJetBrainsPaints: {Recommendation,
		"JetBrains terminals show queries as text, so none is sent, and capabilities are unknown.",
		"None needed; run in another terminal for every capability."},
	ReasonEditorTerminal: {Recommendation,
		"Inside an editor's terminal, the queries that describe the outer terminal are not sent.",
		"None needed; run outside the editor for every capability."},
	ReasonReplyTooLong: {Issue,
		"A reply longer than 1 KiB was ignored.",
		"Report the terminal and its version."},
}

// Findings is a finding for each reason token in c, its facts and its
// views, in field order, each token once.
func Findings(c Caps) []Finding {
	var out []Finding
	seen := map[string]bool{}
	add := func(token string) {
		if token == "" || seen[token] {
			return
		}
		seen[token] = true
		f, ok := findingText[token]
		if !ok {
			f = finding{Recommendation, "No description.", "None."}
		}
		out = append(out, Finding{ID: token, Disposition: f.d, Message: f.message, Fix: f.fix})
	}
	v := reflect.ValueOf(c)
	for _, f := range v.Fields() {
		if f.Kind() == reflect.Struct && f.NumField() == 3 && f.Field(2).Kind() == reflect.String {
			add(f.Field(2).String())
		}
	}
	add(c.Keyboard().Kitty.Reason)
	add(c.Links().OSC8.Reason)
	n := c.Notifications()
	add(n.OSC777.Reason)
	add(n.OSC9.Reason)
	return out
}
