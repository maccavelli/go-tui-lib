// Package termsvc offers terminal services, notifications, the clipboard,
// hyperlinks and shell-integration prompt marks, as Bubble Tea commands or
// strings built from what termcap learned about the terminal
// (docs/decisions/0005-MADR-terminal-capabilities-and-services.md §5).
//
// Nothing here writes to the terminal itself or starts a process. A
// sequence leaves as a tea.Cmd or a string the caller places, and a native
// clipboard or notifier is a hook the program supplies. Text from a
// program, which may be an agent's, is stripped of control bytes before it
// is encoded, so it cannot smuggle a sequence into the terminal.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package termsvc

import (
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// screenLimit is the longest string GNU screen passes on in one piece
// (its MAXSTR); ScreenPassthrough splits a longer sequence into chunks.
const screenLimit = 768

// Wrap returns seq as the multiplexer c names passes it to the outer
// terminal: a tmux passthrough, which needs tmux's allow-passthrough, or a
// screen passthrough. Outside a multiplexer it returns seq.
func Wrap(c termcap.Caps, seq string) string {
	switch c.Mux.Value {
	case termcap.Tmux:
		return ansi.TmuxPassthrough(seq)
	case termcap.Screen:
		return ansi.ScreenPassthrough(seq, screenLimit)
	}
	return seq
}

// ErrScheme is Link's error for a URL whose scheme it does not link.
var ErrScheme = errors.New("termsvc: link scheme not allowed")

// linkSchemes are the schemes Link accepts.
var linkSchemes = []string{"http", "https", "file", "mailto"}

// Link returns text as an OSC 8 hyperlink to rawURL, with params (such as
// "id=x") on the opening sequence. It refuses any scheme but http, https,
// file and mailto, and strips control bytes from the URL, the text and the
// params. Lip Gloss's Style.Hyperlink is the styling path; Link is the
// validating one.
func Link(rawURL, text string, params ...string) (string, error) {
	rawURL = strip(rawURL)
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if !slices.Contains(linkSchemes, strings.ToLower(u.Scheme)) {
		return "", ErrScheme
	}
	clean := make([]string, len(params))
	for i, p := range params {
		clean[i] = strip(p)
	}
	return ansi.SetHyperlink(rawURL, clean...) + strip(text) + ansi.ResetHyperlink(), nil
}

// PromptStart marks where a shell prompt starts (OSC 133 A). Prompt marks
// let a terminal jump between commands in inline output; on the alternate
// screen they mean nothing.
func PromptStart() string { return ansi.FinalTermPrompt() }

// CommandStart marks where the command line starts, after the prompt
// (OSC 133 B).
func CommandStart() string { return ansi.FinalTermCmdStart() }

// CommandExecuted marks where the command's output starts (OSC 133 C).
func CommandExecuted() string { return ansi.FinalTermCmdExecuted() }

// CommandFinished marks where the command's output ends, with its exit
// status (OSC 133 D).
func CommandFinished(exit int) string { return ansi.FinalTermCmdFinished(strconv.Itoa(exit)) }

// strip removes every control character, C0, DEL and C1, ESC among them,
// and every byte that is not UTF-8, which would include a C1 control in
// its 8-bit form; it turns tabs and line breaks into spaces.
func strip(s string) string {
	var b strings.Builder
	for s != "" {
		r, size := utf8.DecodeRuneInString(s)
		switch {
		case r == utf8.RuneError && size == 1:
		case r == '\t' || r == '\n' || r == '\r':
			b.WriteByte(' ')
		case r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f):
		default:
			b.WriteString(s[:size])
		}
		s = s[size:]
	}
	return b.String()
}

// lineBreaks is a run of line breaks with the blanks around it.
var lineBreaks = regexp.MustCompile(`[ \t]*[\r\n]+[ \t]*`)

// clean is s for a notification: escape sequences removed, each run of
// line breaks collapsed to one space, controls removed, blanks at either
// end trimmed, and cut to cells cells, by width, never in the middle of a
// character.
func clean(s string, cells int) string {
	s = lineBreaks.ReplaceAllString(ansi.Strip(s), " ")
	return ansi.Truncate(strings.TrimSpace(strip(s)), cells, "")
}
