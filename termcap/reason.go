package termcap

// Reason tokens say why a fact is unsupported, unknown or gated. They are
// API: a token is added, never renamed, and the doctor's findings use the
// same strings (docs/decisions/0005-MADR-terminal-capabilities-and-services.md
// A1).
const (
	// ReasonKittyUnsupported: the terminal did not answer the Kitty
	// keyboard query, so no enhancement is asked for.
	ReasonKittyUnsupported = "keyboard.kitty-unsupported"
	// ReasonKittyUnknown: the Kitty keyboard query has no answer yet.
	ReasonKittyUnknown = "keyboard.kitty-unknown"
	// ReasonLeaksReleases: iTerm2 and Ghostty report releases of shortcuts
	// they consume themselves, so event types are not asked for.
	ReasonLeaksReleases = "terminal.leaks-releases"
	// ReasonAlacrittyRelease: Alacritty 0.14 and older send a duplicate
	// legacy release with event types on.
	ReasonAlacrittyRelease = "terminal.alacritty-duplicate-release"
	// ReasonTmuxExtendedKeys: tmux passes event types only with
	// extended-keys-format csi-u.
	ReasonTmuxExtendedKeys = "tmux.extended-keys-off"
	// ReasonMSYSNoKitty: mintty, MSYS2 and Git Bash want no Kitty
	// keyboard, which tea still asks for (A4).
	ReasonMSYSNoKitty = "keyboard.msys-no-kitty"
	// ReasonWSLDeadKeys: under WSL, keyboard enhancement breaks dead keys
	// in VS Code, and is not trusted where the terminal is unknown; tea
	// still asks for disambiguation (A4).
	ReasonWSLDeadKeys = "keyboard.wsl-dead-keys"

	// ReasonAppleNoOSC8: Apple Terminal's parser mishandles OSC 8.
	ReasonAppleNoOSC8 = "terminal.apple-no-osc8"
	// ReasonWarpNoOSC8: Warp shows no OSC 8 links.
	ReasonWarpNoOSC8 = "terminal.warp-no-osc8"
	// ReasonTmuxLinks: tmux passes hyperlinks from 3.4.
	ReasonTmuxLinks = "tmux.hyperlinks-before-3.4"
	// ReasonMuxNoLinks: screen and Zellij pass no hyperlinks.
	ReasonMuxNoLinks = "mux.hyperlinks-unsupported"
	// ReasonUnknownTerminal: nothing names the terminal, so a capability
	// that depends on it fails closed.
	ReasonUnknownTerminal = "terminal.unknown"
	// ReasonZellijNoForwarding: Zellij passes no notification sequence.
	ReasonZellijNoForwarding = "mux.zellij-no-passthrough"

	// ReasonColorFGBGGuess: light or dark is read from COLORFGBG, which is a
	// guess.
	ReasonColorFGBGGuess = "appearance.colorfgbg-guess"
	// ReasonDesktopAppearance: light or dark is the desktop's setting, from
	// the program's hook, not the terminal's.
	ReasonDesktopAppearance = "appearance.desktop"

	// ReasonLegacyConsoleGuess: on Windows with no terminal variables, the
	// console is taken for the legacy console host.
	ReasonLegacyConsoleGuess = "console.no-terminal-variables"
	// ReasonWindowsTerminalGuess: on Windows an unnamed terminal is taken
	// for Windows Terminal, whose default-terminal hand-off omits
	// WT_SESSION.
	ReasonWindowsTerminalGuess = "terminal.windows-terminal-guess"

	// ReasonJetBrainsPaints: JetBrains terminals paint a query as text, so
	// the prober sends none there, and every fact comes from the
	// environment.
	ReasonJetBrainsPaints = "terminal.jetbrains-paints-queries"
	// ReasonEditorTerminal: inside an editor's terminal, which answers for
	// the editor rather than the user's terminal, the gated queries are not
	// sent.
	ReasonEditorTerminal = "editor.terminal-gated"
	// ReasonReplyTooLong: a reply longer than 1 KiB was not parsed.
	ReasonReplyTooLong = "probe.reply-too-long"
)

// reasons is every token, for the tests that keep them unique and the
// doctor's findings.
var reasons = []string{
	ReasonKittyUnsupported, ReasonKittyUnknown, ReasonLeaksReleases, ReasonAlacrittyRelease,
	ReasonTmuxExtendedKeys, ReasonMSYSNoKitty, ReasonWSLDeadKeys,
	ReasonAppleNoOSC8, ReasonWarpNoOSC8, ReasonTmuxLinks, ReasonMuxNoLinks, ReasonUnknownTerminal, ReasonZellijNoForwarding,
	ReasonColorFGBGGuess, ReasonDesktopAppearance,
	ReasonLegacyConsoleGuess, ReasonWindowsTerminalGuess,
	ReasonJetBrainsPaints, ReasonEditorTerminal, ReasonReplyTooLong,
}
