package workspace_test

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/workspace"
)

// The panes of an agent session, as a program like pi-go would write them.
// They use the public API only. They are examples, not the standard panes,
// which are later records.

// line is a streamed transcript line.
type line string

// transcript shows the conversation, newest at the bottom, and keeps the
// cursor at the end of the prompt line.
type transcript struct {
	lines  []string
	prompt string
	height int
}

func (p *transcript) Update(msg tea.Msg) (workspace.Pane, tea.Cmd) {
	switch m := msg.(type) {
	case workspace.SizeMsg:
		p.height = m.Height
	case line:
		p.lines = append(p.lines, string(m))
	case tea.KeyPressMsg:
		if m.Text != "" {
			p.prompt += m.Text
		}
	}
	return p, nil
}

func (p *transcript) View(w, h int) string {
	body := tail(p.lines, h-1)
	for len(body) < h-1 {
		body = append([]string{""}, body...)
	}
	return strings.Join(append(body, "> "+p.prompt), "\n")
}

func (p *transcript) Title() string { return "Session" }

// Cursor sits after the prompt, on the pane's last row.
func (p *transcript) Cursor() *tea.Cursor {
	return tea.NewCursor(2+len(p.prompt), p.height-1)
}

// metrics is a key/value sidebar.
type metrics struct{ rows [][2]string }

func (p *metrics) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return p, nil }
func (p *metrics) Title() string                            { return "Metrics" }
func (p *metrics) View(w, h int) string {
	out := make([]string, 0, len(p.rows))
	for _, r := range p.rows {
		out = append(out, fmt.Sprintf("%-9s %s", r[0], r[1]))
	}
	return strings.Join(out, "\n")
}

// logs tails log lines, with an unread count while it is not focused.
type logs struct {
	lines   []string
	unread  int
	focused bool
}

func (p *logs) Update(msg tea.Msg) (workspace.Pane, tea.Cmd) {
	if m, ok := msg.(logLine); ok {
		p.lines = append(p.lines, string(m))
		if !p.focused {
			p.unread++
		}
	}
	return p, nil
}

func (p *logs) View(w, h int) string { return strings.Join(tail(p.lines, h), "\n") }
func (p *logs) Title() string        { return "Logs" }
func (p *logs) Focus() tea.Cmd       { p.focused, p.unread = true, 0; return nil }
func (p *logs) Blur()                { p.focused = false }
func (p *logs) Badge() string {
	if p.unread == 0 {
		return ""
	}
	return fmt.Sprint(p.unread)
}

type logLine string

// footer is the status line. It takes no focus and has no chrome.
type footer struct{ text string }

func (p *footer) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return p, nil }
func (p *footer) View(int, int) string                     { return p.text }
func (p *footer) Focusable() bool                          { return false }

func tail(s []string, n int) []string {
	if n <= 0 {
		return nil
	}
	if len(s) > n {
		return s[len(s)-n:]
	}
	return append([]string(nil), s...)
}

// session builds the agent session on root with theme th, sized w × h,
// and streams some activity into it.
func session(root layout.Node, th theme.Theme, w, h int) *workspace.Workspace {
	ws := workspace.New(root, map[layout.PaneID]workspace.Pane{
		"session": &transcript{},
		"metrics": &metrics{rows: [][2]string{
			{"model", "claude"}, {"tokens", "12,480"}, {"context", "18%"}, {"cost", "$0.042"}, {"latency", "1.2s"},
		}},
		"logs":   &logs{},
		"footer": &footer{text: "claude | mode ask | ctx 18% | $0.042"},
	}, workspace.WithTheme(th), workspace.WithPaneChrome("footer", workspace.None))
	ws.Init()
	ws.Update(tea.WindowSizeMsg{Width: w, Height: h})
	for _, l := range []string{"you: list the open files", "agent: reading the workspace", "agent: 3 files are open"} {
		ws.Update(workspace.To("session", line(l)))
	}
	for _, l := range []string{"INFO session started", "INFO tool read_file", "WARN slow response 1.2s"} {
		ws.Update(workspace.To("logs", logLine(l)))
	}
	for _, r := range "why?" {
		ws.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return ws
}

// agentRoot is the preset: sidebar on the left or right, and the bottom pane
// across the width or under the session only.
func agentRoot(left bool, span layout.Span) layout.Node {
	opts := []layout.PresetOption{layout.Footer("footer", 1), layout.Gap(0), layout.BottomSpan(span)}
	if left {
		return layout.SidebarLeftBottom("session", "metrics", "logs", opts...)
	}
	return layout.SidebarRightBottom("session", "metrics", "logs", opts...)
}

// trimmed removes trailing blanks from each line, as example output needs.
func trimmed(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

// An agent session: the transcript in the main pane, metrics in a right
// sidebar, logs in a bottom pane and a status footer. The logs pane counts
// unread lines until it is focused.
func ExampleWorkspace_agentSession() {
	th := theme.New(colorprofile.NoTTY, theme.Unknown, glyph.ASCII())
	ws := session(agentRoot(false, layout.FullWidth), th, 104, 20)
	fmt.Println(trimmed(ws.Render()))
	c := ws.Cursor()
	fmt.Printf("cursor at %d,%d; focus %s\n", c.X, c.Y, ws.Focused())
	// Output:
	// +- > Session -----------------------------------------------------------++- Metrics -------------------+
	// |                                                                       ||model     claude             |
	// |                                                                       ||tokens    12,480             |
	// |                                                                       ||context   18%                |
	// |                                                                       ||cost      $0.042             |
	// |                                                                       ||latency   1.2s               |
	// |                                                                       ||                             |
	// |                                                                       ||                             |
	// |                                                                       ||                             |
	// |you: list the open files                                               ||                             |
	// |agent: reading the workspace                                           ||                             |
	// |agent: 3 files are open                                                ||                             |
	// |> why?                                                                 ||                             |
	// +-----------------------------------------------------------------------++-----------------------------+
	// +- Logs [3] -------------------------------------------------------------------------------------------+
	// |INFO session started                                                                                  |
	// |INFO tool read_file                                                                                   |
	// |WARN slow response 1.2s                                                                               |
	// +------------------------------------------------------------------------------------------------------+
	// claude | mode ask | ctx 18% | $0.042
	// cursor at 7,12; focus session
}
