package termcap

import "github.com/maccavelli/go-tui-lib/v2/msgcat"

// The report's words, by message ID: each finding's message and fix,
// by its reason token, and the notes beside a fact never queried
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D9, A4).
// Findings gives them in English; Report gives them in WithReportCatalog's.
const (
	// MsgKittyUnsupported is the finding keyboard.kitty-unsupported's message.
	MsgKittyUnsupported msgcat.MessageID = "termcap.keyboard.kitty-unsupported"
	// MsgKittyUnsupportedFix is its fix.
	MsgKittyUnsupportedFix msgcat.MessageID = "termcap.keyboard.kitty-unsupported.fix"
	// MsgKittyUnknown is the finding keyboard.kitty-unknown's message.
	MsgKittyUnknown msgcat.MessageID = "termcap.keyboard.kitty-unknown"
	// MsgKittyUnknownFix is its fix.
	MsgKittyUnknownFix msgcat.MessageID = "termcap.keyboard.kitty-unknown.fix"
	// MsgLeaksReleases is the finding terminal.leaks-releases's message.
	MsgLeaksReleases msgcat.MessageID = "termcap.terminal.leaks-releases"
	// MsgLeaksReleasesFix is its fix.
	MsgLeaksReleasesFix msgcat.MessageID = "termcap.terminal.leaks-releases.fix"
	// MsgAlacrittyRelease is the finding terminal.alacritty-duplicate-release's message.
	MsgAlacrittyRelease msgcat.MessageID = "termcap.terminal.alacritty-duplicate-release"
	// MsgAlacrittyReleaseFix is its fix.
	MsgAlacrittyReleaseFix msgcat.MessageID = "termcap.terminal.alacritty-duplicate-release.fix"
	// MsgTmuxExtendedKeys is the finding tmux.extended-keys-off's message.
	MsgTmuxExtendedKeys msgcat.MessageID = "termcap.tmux.extended-keys-off"
	// MsgTmuxExtendedKeysFix is its fix.
	MsgTmuxExtendedKeysFix msgcat.MessageID = "termcap.tmux.extended-keys-off.fix"
	// MsgMSYSNoKitty is the finding keyboard.msys-no-kitty's message.
	MsgMSYSNoKitty msgcat.MessageID = "termcap.keyboard.msys-no-kitty"
	// MsgMSYSNoKittyFix is its fix.
	MsgMSYSNoKittyFix msgcat.MessageID = "termcap.keyboard.msys-no-kitty.fix"
	// MsgWSLDeadKeys is the finding keyboard.wsl-dead-keys's message.
	MsgWSLDeadKeys msgcat.MessageID = "termcap.keyboard.wsl-dead-keys"
	// MsgWSLDeadKeysFix is its fix.
	MsgWSLDeadKeysFix msgcat.MessageID = "termcap.keyboard.wsl-dead-keys.fix"
	// MsgAppleNoOSC8 is the finding terminal.apple-no-osc8's message.
	MsgAppleNoOSC8 msgcat.MessageID = "termcap.terminal.apple-no-osc8"
	// MsgAppleNoOSC8Fix is its fix.
	MsgAppleNoOSC8Fix msgcat.MessageID = "termcap.terminal.apple-no-osc8.fix"
	// MsgWarpNoOSC8 is the finding terminal.warp-no-osc8's message.
	MsgWarpNoOSC8 msgcat.MessageID = "termcap.terminal.warp-no-osc8"
	// MsgWarpNoOSC8Fix is its fix.
	MsgWarpNoOSC8Fix msgcat.MessageID = "termcap.terminal.warp-no-osc8.fix"
	// MsgTmuxLinks is the finding tmux.hyperlinks-before-3.4's message.
	MsgTmuxLinks msgcat.MessageID = "termcap.tmux.hyperlinks-before-3.4"
	// MsgTmuxLinksFix is its fix.
	MsgTmuxLinksFix msgcat.MessageID = "termcap.tmux.hyperlinks-before-3.4.fix"
	// MsgMuxNoLinks is the finding mux.hyperlinks-unsupported's message.
	MsgMuxNoLinks msgcat.MessageID = "termcap.mux.hyperlinks-unsupported"
	// MsgMuxNoLinksFix is its fix.
	MsgMuxNoLinksFix msgcat.MessageID = "termcap.mux.hyperlinks-unsupported.fix"
	// MsgUnknownTerminal is the finding terminal.unknown's message.
	MsgUnknownTerminal msgcat.MessageID = "termcap.terminal.unknown"
	// MsgUnknownTerminalFix is its fix.
	MsgUnknownTerminalFix msgcat.MessageID = "termcap.terminal.unknown.fix"
	// MsgZellijNoForwarding is the finding mux.zellij-no-passthrough's message.
	MsgZellijNoForwarding msgcat.MessageID = "termcap.mux.zellij-no-passthrough"
	// MsgZellijNoForwardingFix is its fix.
	MsgZellijNoForwardingFix msgcat.MessageID = "termcap.mux.zellij-no-passthrough.fix"
	// MsgColorFGBGGuess is the finding appearance.colorfgbg-guess's message.
	MsgColorFGBGGuess msgcat.MessageID = "termcap.appearance.colorfgbg-guess"
	// MsgColorFGBGGuessFix is its fix.
	MsgColorFGBGGuessFix msgcat.MessageID = "termcap.appearance.colorfgbg-guess.fix"
	// MsgDesktopAppearance is the finding appearance.desktop's message.
	MsgDesktopAppearance msgcat.MessageID = "termcap.appearance.desktop"
	// MsgDesktopAppearanceFix is its fix.
	MsgDesktopAppearanceFix msgcat.MessageID = "termcap.appearance.desktop.fix"
	// MsgLegacyConsoleGuess is the finding console.no-terminal-variables's message.
	MsgLegacyConsoleGuess msgcat.MessageID = "termcap.console.no-terminal-variables"
	// MsgLegacyConsoleGuessFix is its fix.
	MsgLegacyConsoleGuessFix msgcat.MessageID = "termcap.console.no-terminal-variables.fix"
	// MsgConPTYAnswers is the finding terminal.conpty-answers's message.
	MsgConPTYAnswers msgcat.MessageID = "termcap.terminal.conpty-answers"
	// MsgConPTYAnswersFix is its fix.
	MsgConPTYAnswersFix msgcat.MessageID = "termcap.terminal.conpty-answers.fix"
	// MsgWindowsTerminalGuess is the finding terminal.windows-terminal-guess's message.
	MsgWindowsTerminalGuess msgcat.MessageID = "termcap.terminal.windows-terminal-guess"
	// MsgWindowsTerminalGuessFix is its fix.
	MsgWindowsTerminalGuessFix msgcat.MessageID = "termcap.terminal.windows-terminal-guess.fix"
	// MsgJetBrainsPaints is the finding terminal.jetbrains-paints-queries's message.
	MsgJetBrainsPaints msgcat.MessageID = "termcap.terminal.jetbrains-paints-queries"
	// MsgJetBrainsPaintsFix is its fix.
	MsgJetBrainsPaintsFix msgcat.MessageID = "termcap.terminal.jetbrains-paints-queries.fix"
	// MsgEditorTerminal is the finding editor.terminal-gated's message.
	MsgEditorTerminal msgcat.MessageID = "termcap.editor.terminal-gated"
	// MsgEditorTerminalFix is its fix.
	MsgEditorTerminalFix msgcat.MessageID = "termcap.editor.terminal-gated.fix"
	// MsgReplyTooLong is the finding probe.reply-too-long's message.
	MsgReplyTooLong msgcat.MessageID = "termcap.probe.reply-too-long"
	// MsgReplyTooLongFix is its fix.
	MsgReplyTooLongFix msgcat.MessageID = "termcap.probe.reply-too-long.fix"
	// MsgUnknownQuery is the finding probe.unknown-query's message.
	MsgUnknownQuery msgcat.MessageID = "termcap.probe.unknown-query"
	// MsgUnknownQueryFix is its fix.
	MsgUnknownQueryFix msgcat.MessageID = "termcap.probe.unknown-query.fix"
	// MsgNoDescription is the message of a reason token with no finding.
	MsgNoDescription msgcat.MessageID = "termcap.no-description"
	// MsgNoDescriptionFix is its fix.
	MsgNoDescriptionFix msgcat.MessageID = "termcap.no-description.fix"
	// MsgNoteTeaAsks is the note beside sync_output and grapheme_width when
	// no one asked for them.
	MsgNoteTeaAsks msgcat.MessageID = "termcap.note-tea-asks"
	// MsgNoteHeuristic is the note beside desktop_notify and kitty_graphics
	// when no one asked for them.
	MsgNoteHeuristic msgcat.MessageID = "termcap.note-heuristic"
	// MsgNoteTimeout is the note beside a fact never queried when the
	// probe timed out.
	MsgNoteTimeout msgcat.MessageID = "termcap.note-timeout"
	// MsgNoteOrTimeout is added to a note when the probe timed out: the
	// note.
	MsgNoteOrTimeout msgcat.MessageID = "termcap.note-or-timeout"
)

// noFix is the fix where nothing needs doing.
const noFix = "None needed."

// Messages is the report's words in English, its defaults, by ID: where a
// program starts its translation.
func Messages() msgcat.Table {
	return msgcat.Table{
		MsgKittyUnsupported:        "The terminal did not answer the Kitty keyboard query, so keys such as shift+enter may not be told apart.",
		MsgKittyUnsupportedFix:     "Use a terminal with the Kitty keyboard protocol, such as kitty, Ghostty, WezTerm, foot or iTerm2.",
		MsgKittyUnknown:            "The Kitty keyboard query had no answer before the probe ended.",
		MsgKittyUnknownFix:         "Run again; if it persists, give the prober a longer timeout.",
		MsgLeaksReleases:           "This terminal reports releases of shortcuts it handles itself, so key releases are not asked for.",
		MsgLeaksReleasesFix:        "None needed: bindings that wait for a release fall back to presses.",
		MsgAlacrittyRelease:        "Alacritty 0.14 or older sends a duplicate release when event types are on, so they are not asked for.",
		MsgAlacrittyReleaseFix:     "Upgrade Alacritty past 0.14.",
		MsgTmuxExtendedKeys:        "tmux does not pass key event types.",
		MsgTmuxExtendedKeysFix:     "Set extended-keys on and extended-keys-format csi-u in tmux.conf, and pass TmuxQuery's output to the prober.",
		MsgMSYSNoKitty:             "mintty and MSYS2 want no Kitty keyboard protocol, which Bubble Tea still asks for.",
		MsgMSYSNoKittyFix:          "None in this library; a later terminal-mode record can turn it off.",
		MsgWSLDeadKeys:             "Under WSL, keyboard enhancement breaks dead keys in VS Code, and is not trusted in an unnamed terminal.",
		MsgWSLDeadKeysFix:          "None in this library; a later terminal-mode record can turn it off.",
		MsgAppleNoOSC8:             "Apple Terminal mishandles OSC 8 links, so links are shown with their URL.",
		MsgAppleNoOSC8Fix:          "Use a terminal that supports OSC 8 links.",
		MsgWarpNoOSC8:              "Warp shows no OSC 8 links, so links are shown with their URL.",
		MsgWarpNoOSC8Fix:           "Use a terminal that supports OSC 8 links.",
		MsgTmuxLinks:               "tmux before 3.4 drops hyperlinks, so links are shown with their URL.",
		MsgTmuxLinksFix:            "Upgrade tmux to 3.4 or later, and pass TmuxQuery's output to the prober.",
		MsgMuxNoLinks:              "This multiplexer passes no hyperlinks, so links are shown with their URL.",
		MsgMuxNoLinksFix:           "Run outside the multiplexer, or in tmux 3.4 or later.",
		MsgUnknownTerminal:         "Nothing names the terminal, so capabilities that depend on it are off.",
		MsgUnknownTerminalFix:      "Set TERM_PROGRAM, or let LC_TERMINAL through SSH.",
		MsgZellijNoForwarding:      "Zellij passes no notification sequence, so notifications ring the bell.",
		MsgZellijNoForwardingFix:   noFix,
		MsgColorFGBGGuess:          "Light or dark is guessed from COLORFGBG.",
		MsgColorFGBGGuessFix:       "Use a terminal that answers DSR 996 or OSC 11, or set the program's appearance variable.",
		MsgDesktopAppearance:       "Light or dark is the desktop's setting, not the terminal's.",
		MsgDesktopAppearanceFix:    noFix,
		MsgLegacyConsoleGuess:      "On Windows nothing names the terminal, so it is taken for the legacy console host, which lacks most capabilities.",
		MsgLegacyConsoleGuessFix:   "Run in Windows Terminal.",
		MsgConPTYAnswers:           "Over SSH into Windows, ConPTY answers the queries itself, so these facts describe ConPTY, not your terminal.",
		MsgConPTYAnswersFix:        "None in this library; run the program on the machine with the terminal to see the terminal's own capabilities.",
		MsgWindowsTerminalGuess:    "The terminal is taken for Windows Terminal, whose hand-off omits WT_SESSION.",
		MsgWindowsTerminalGuessFix: noFix,
		MsgJetBrainsPaints:         "JetBrains terminals show queries as text, so none is sent, and capabilities are unknown.",
		MsgJetBrainsPaintsFix:      "None needed; run in another terminal for every capability.",
		MsgEditorTerminal:          "Inside an editor's terminal, the queries that describe the outer terminal are not sent.",
		MsgEditorTerminalFix:       "None needed; run outside the editor for every capability.",
		MsgReplyTooLong:            "A reply longer than 1 KiB was ignored.",
		MsgReplyTooLongFix:         "Report the terminal and its version.",
		MsgUnknownQuery:            "The program left out a query the probe does not have, so no query was sent.",
		MsgUnknownQueryFix:         "Name only the probe's queries in WithoutQuery; its documentation lists them.",
		MsgNoDescription:           "No description.",
		MsgNoDescriptionFix:        "None.",
		MsgNoteTeaAsks:             "tea asks only outside SSH and Apple Terminal, and not every terminal answers",
		MsgNoteHeuristic:           "asked only behind the heuristic",
		MsgNoteTimeout:             "no reply before the timeout",
		MsgNoteOrTimeout:           "%s; or no reply before the timeout",
	}
}

// say is c's text for id, or the default.
func say(c msgcat.Catalog, id msgcat.MessageID, args ...any) string {
	return msgcat.Format(c, id, Messages()[id], args...)
}
