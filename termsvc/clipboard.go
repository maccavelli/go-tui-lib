package termsvc

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// MaxCopyBytes is the largest payload Copy sends. A terminal route may
// truncate or refuse a longer OSC 52 without saying so.
const MaxCopyBytes = 100_000

// Status is what is known about a copy.
type Status uint8

const (
	// Unconfirmed means it was written to the terminal, which never confirms OSC 52.
	Unconfirmed Status = iota
	// Confirmed means the program's clipboard accepted it.
	Confirmed
	// Failed means it was too long, or the program's clipboard returned an error.
	Failed
)

// Route is a way to the clipboard.
type Route uint8

const (
	// RouteNone means nothing was sent.
	RouteNone Route = iota
	// RouteBackend means the program's own clipboard, such as a desktop API.
	RouteBackend
	// RouteTmuxBuffer means tmux's buffer, through the program's run of
	// TmuxLoadBuffer.
	RouteTmuxBuffer
	// RouteOSC52 means OSC 52, through tea.
	RouteOSC52
	// RouteOSC52Tmux means OSC 52 through tea, and again wrapped for tmux's
	// passthrough, for a tmux whose set-clipboard is off.
	RouteOSC52Tmux
)

// CopiedMsg reports a copy.
type CopiedMsg struct {
	Status Status
	Route  Route
	Err    error // the clipboard's error, with Failed
}

// Clipboard is a clipboard the program supplies. The library never starts
// a process itself.
type Clipboard interface {
	Copy(ctx context.Context, text string) error
}

// CopyOption configures Copy.
type CopyOption func(*copyConfig)

type copyConfig struct{ clipboard Clipboard }

// WithClipboard copies through b, which confirms or fails, instead of the
// terminal.
func WithClipboard(b Clipboard) CopyOption { return func(c *copyConfig) { c.clipboard = b } }

// Copy writes text to the clipboard, and delivers CopiedMsg. With
// WithClipboard it goes through the program's clipboard; otherwise through
// OSC 52, and inside tmux also wrapped for passthrough. A payload over
// MaxCopyBytes is not sent, and fails.
func Copy(c termcap.Caps, text string, o ...CopyOption) tea.Cmd {
	var cfg copyConfig
	for _, f := range o {
		f(&cfg)
	}
	if len(text) > MaxCopyBytes {
		return copied(CopiedMsg{Status: Failed})
	}
	if b := cfg.clipboard; b != nil {
		return func() tea.Msg {
			if err := b.Copy(context.Background(), text); err != nil {
				return CopiedMsg{Status: Failed, Route: RouteBackend, Err: err}
			}
			return CopiedMsg{Status: Confirmed, Route: RouteBackend}
		}
	}
	if c.Mux.Value == termcap.Tmux {
		return tea.Sequence(
			tea.Batch(tea.SetClipboard(text), tea.Raw(Wrap(c, ansi.SetSystemClipboard(text)))),
			copied(CopiedMsg{Status: Unconfirmed, Route: RouteOSC52Tmux}))
	}
	return tea.Sequence(tea.SetClipboard(text), copied(CopiedMsg{Status: Unconfirmed, Route: RouteOSC52}))
}

func copied(m CopiedMsg) tea.Cmd { return func() tea.Msg { return m } }

// CopyPlan is the routes to try, in order: the program's clipboard, tmux's
// buffer inside tmux, OSC 52, and OSC 52 wrapped for tmux inside tmux. A
// program skips a route it has no means for.
func CopyPlan(c termcap.Caps) []Route {
	if c.Mux.Value == termcap.Tmux {
		return []Route{RouteBackend, RouteTmuxBuffer, RouteOSC52, RouteOSC52Tmux}
	}
	return []Route{RouteBackend, RouteOSC52}
}

// TmuxLoadBuffer is the argv that puts its standard input into tmux's
// buffer and the system clipboard, for RouteTmuxBuffer. The program runs
// it with the text on standard input.
func TmuxLoadBuffer() []string { return []string{"tmux", "load-buffer", "-w", "-"} }

// pngToStdout is the PowerShell that writes the clipboard's image to
// standard output as PNG, or nothing.
const pngToStdout = "Add-Type -AssemblyName System.Windows.Forms,System.Drawing;" +
	"$i=[System.Windows.Forms.Clipboard]::GetImage();" +
	"if($i){$m=New-Object System.IO.MemoryStream;" +
	"$i.Save($m,[System.Drawing.Imaging.ImageFormat]::Png);" +
	"$o=[Console]::OpenStandardOutput();$o.Write($m.ToArray(),0,$m.Length)}"

// ImageReadCommands is the argv of each command that reads an image from
// the clipboard on goos, in the order to try: osascript on macOS, which
// prints the PNG as «data PNGf<hex>»; PowerShell on Windows; on Linux,
// wl-paste under Wayland, then xclip, then PowerShell through WSL's
// interop. Each other command writes PNG bytes. The program runs them; the
// library never does.
func ImageReadCommands(goos string, wayland bool) [][]string {
	powershell := func(exe string) []string {
		return []string{exe, "-NoProfile", "-NonInteractive", "-Command", pngToStdout}
	}
	switch goos {
	case "darwin":
		return [][]string{{"osascript", "-e", "the clipboard as «class PNGf»"}}
	case "windows":
		return [][]string{powershell("powershell")}
	}
	var cmds [][]string
	if wayland {
		cmds = append(cmds, []string{"wl-paste", "--no-newline", "--type", "image/png"})
	}
	return append(cmds,
		[]string{"xclip", "-selection", "clipboard", "-target", "image/png", "-out"},
		powershell("powershell.exe"))
}
