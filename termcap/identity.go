package termcap

import (
	"regexp"
	"strings"
)

// goosWindows is runtime.GOOS on Windows.
const goosWindows = "windows"

// Brand is the terminal emulator.
type Brand uint8

// Brands. BrandUnknown is the zero value.
const (
	BrandUnknown Brand = iota
	BrandAppleTerminal
	BrandITerm2
	BrandKitty
	BrandGhostty
	BrandWezTerm
	BrandAlacritty
	BrandFoot
	BrandRio
	BrandContour
	BrandVTE
	BrandKonsole
	BrandTerminator
	BrandWindowsTerminal
	BrandVSCode
	BrandCursor
	BrandWindsurf
	BrandZed
	BrandJetBrains
	BrandWarp
	BrandMintty
	BrandXTerm
)

var brandNames = []string{
	"unknown", "apple-terminal", "iterm2", "kitty", "ghostty", "wezterm", "alacritty", "foot",
	"rio", "contour", "vte", "konsole", "terminator", "windows-terminal", "vscode", "cursor",
	"windsurf", "zed", "jetbrains", "warp", "mintty", "xterm",
}

// String is the brand's name, as the report and JSON print it.
func (b Brand) String() string { return name(brandNames, b) }

// MarshalText is the brand's name.
func (b Brand) MarshalText() ([]byte, error) { return marshal(brandNames, "Brand", b) }

// UnmarshalText reads a name MarshalText wrote.
func (b *Brand) UnmarshalText(t []byte) error { return unmarshal(brandNames, "Brand", t, b) }

// Editor is an editor whose embedded terminal the program runs in. Its
// terminal answers for the editor, not for the user's terminal.
type Editor uint8

// Editors. EditorNone is the zero value.
const (
	EditorNone Editor = iota
	EditorNeovim
	EditorVim
	EditorEmacs
)

var editorNames = []string{"none", "neovim", "vim", "emacs"}

// String is the editor's name, as the report and JSON print it.
func (e Editor) String() string { return name(editorNames, e) }

// MarshalText is the editor's name.
func (e Editor) MarshalText() ([]byte, error) { return marshal(editorNames, "Editor", e) }

// UnmarshalText reads a name MarshalText wrote.
func (e *Editor) UnmarshalText(t []byte) error { return unmarshal(editorNames, "Editor", t, e) }

// Platform is a compatibility layer the program runs under.
type Platform uint8

// Platforms. PlatformNative is the zero value.
const (
	PlatformNative Platform = iota
	PlatformMSYS            // MSYS2, Git Bash: MSYSTEM is set
	PlatformWSL             // the Windows Subsystem for Linux
)

var platformNames = []string{"native", "msys", "wsl"}

// String is the platform's name, as the report and JSON print it.
func (p Platform) String() string { return name(platformNames, p) }

// MarshalText is the platform's name.
func (p Platform) MarshalText() ([]byte, error) { return marshal(platformNames, "Platform", p) }

// UnmarshalText reads a name MarshalText wrote.
func (p *Platform) UnmarshalText(t []byte) error {
	return unmarshal(platformNames, "Platform", t, p)
}

// Identity is what the environment says about the terminal.
type Identity struct {
	// Brand is EnvBrand refined: on Windows an unknown brand is Windows
	// Terminal, whose default-terminal hand-off omits WT_SESSION. A
	// decision that must not trust the guess reads EnvBrand.
	Brand, EnvBrand Brand
	// Version is the terminal's version, kept only when the variable that
	// named the brand also gave it.
	Version      string
	Mux          Mux
	Editor       Editor
	Platform     Platform
	Remote       bool
	Term         string // TERM
	TermFeatures string // TERM_FEATURES
}

// itermSession is iTerm2's TERM_SESSION_ID; Apple Terminal's is a bare
// UUID.
var itermSession = regexp.MustCompile(`^w\d+t\d+p\d+:`)

// FromEnv is the identity env gives on goos, the caller's runtime.GOOS. It
// is a pure function, so a test can fake tmux, SSH or Windows.
//
// The order encodes known traps: an editor fork's markers survive SSH and
// tmux, so they come first; JetBrains sets TERM_SESSION_ID and would read
// as Apple Terminal, so TERMINAL_EMULATOR comes before it; LC_TERMINAL
// crosses SSH; Terminator sets VTE_VERSION too; and WT_SESSION is last.
func FromEnv(env Env, goos string) Identity {
	id := Identity{Term: env.Getenv("TERM"), TermFeatures: env.Getenv("TERM_FEATURES")}
	id.EnvBrand, id.Version = envBrand(env)
	id.Brand = id.EnvBrand
	if id.Brand == BrandUnknown && goos == goosWindows {
		id.Brand = BrandWindowsTerminal
	}

	switch {
	case env.Getenv("TMUX") != "":
		id.Mux = Tmux
	case env.Getenv("STY") != "":
		id.Mux = Screen
	case env.Getenv("ZELLIJ") != "":
		id.Mux = Zellij
	}

	switch {
	case env.Getenv("NVIM") != "":
		id.Editor = EditorNeovim
	case env.Getenv("VIM_TERMINAL") != "":
		id.Editor = EditorVim
	case env.Getenv("INSIDE_EMACS") != "":
		id.Editor = EditorEmacs
	}

	switch {
	case env.Getenv("MSYSTEM") != "":
		id.Platform = PlatformMSYS
	case env.Getenv("WSL_DISTRO_NAME") != "" || env.Getenv("WSL_INTEROP") != "":
		id.Platform = PlatformWSL
	}

	_, tty := env.LookupEnv("SSH_TTY")
	_, conn := env.LookupEnv("SSH_CONNECTION")
	id.Remote = tty || conn
	return id
}

// envBrand is the brand the environment names, in FromEnv's order, and the
// version when the same variable's sibling gives one.
func envBrand(env Env) (Brand, string) {
	// 1. Editor forks, whose markers survive SSH and tmux.
	if env.Getenv("CURSOR_TRACE_ID") != "" {
		return BrandCursor, ""
	}
	if askpass := strings.ToLower(env.Getenv("VSCODE_GIT_ASKPASS_MAIN")); askpass != "" {
		switch {
		case strings.Contains(askpass, "windsurf"):
			return BrandWindsurf, ""
		case strings.Contains(askpass, "cursor"):
			return BrandCursor, ""
		}
		return BrandVSCode, ""
	}

	// 2. TERM_PROGRAM, whose version is TERM_PROGRAM_VERSION. A
	// multiplexer sets it to its own name, which names no terminal.
	version := env.Getenv("TERM_PROGRAM_VERSION")
	switch env.Getenv("TERM_PROGRAM") {
	case "Apple_Terminal":
		return BrandAppleTerminal, version
	case "iTerm.app":
		return BrandITerm2, version
	case "WezTerm":
		return BrandWezTerm, version
	case "ghostty":
		return BrandGhostty, version
	case "vscode":
		return BrandVSCode, version
	case "WarpTerminal":
		return BrandWarp, version
	case "mintty":
		return BrandMintty, version
	case "zed":
		return BrandZed, version
	case "rio":
		return BrandRio, version
	}

	// 3. JetBrains, before TERM_SESSION_ID.
	if env.Getenv("TERMINAL_EMULATOR") == "JetBrains-JediTerm" {
		return BrandJetBrains, ""
	}
	if sid := env.Getenv("TERM_SESSION_ID"); sid != "" {
		if itermSession.MatchString(sid) {
			return BrandITerm2, ""
		}
		return BrandAppleTerminal, ""
	}

	// 4. LC_TERMINAL, which a default sshd forwards.
	if env.Getenv("LC_TERMINAL") == "iTerm2" {
		return BrandITerm2, env.Getenv("LC_TERMINAL_VERSION")
	}

	// 5. TERM, and the variables its terminals set beside it.
	term := env.Getenv("TERM")
	for _, t := range []struct {
		brand    Brand
		variable string
	}{
		{BrandKitty, "KITTY_WINDOW_ID"},
		{BrandGhostty, "GHOSTTY_RESOURCES_DIR"},
		{BrandWezTerm, "WEZTERM_PANE"},
		{BrandAlacritty, "ALACRITTY_WINDOW_ID"},
		{BrandFoot, ""},
		{BrandRio, ""},
		{BrandContour, ""},
	} {
		if strings.Contains(term, t.brand.String()) || t.variable != "" && env.Getenv(t.variable) != "" {
			return t.brand, ""
		}
	}

	// 6. Terminator before VTE, which it is built on; then Konsole.
	if env.Getenv("TERMINATOR_UUID") != "" {
		return BrandTerminator, ""
	}
	if v := env.Getenv("VTE_VERSION"); v != "" {
		return BrandVTE, v
	}
	if v := env.Getenv("KONSOLE_VERSION"); v != "" {
		return BrandKonsole, v
	}

	// 7. WT_SESSION last.
	if env.Getenv("WT_SESSION") != "" {
		return BrandWindowsTerminal, ""
	}
	return BrandUnknown, ""
}

// brandFromVersion is the brand an XTVERSION reply names, which over SSH
// is often the only name the terminal gives.
func brandFromVersion(v string) Brand {
	v = strings.ToLower(v)
	for _, b := range []Brand{
		BrandKitty, BrandGhostty, BrandWezTerm, BrandITerm2, BrandFoot, BrandContour,
		BrandKonsole, BrandXTerm, BrandAlacritty,
	} {
		if strings.HasPrefix(v, b.String()) {
			return b
		}
	}
	return BrandUnknown
}
