package termcap_test

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/termcap"
	"github.com/maccavelli/go-tui-lib/workspace"
)

// program embeds a prober beside a workspace. The workspace's own
// background query is turned off, so the prober's, behind its gates, is
// the only one (docs/decisions/0005-MADR-terminal-capabilities-and-services.md
// A2, Q8).
type program struct {
	probe *termcap.Prober
	ws    *workspace.Workspace
	caps  termcap.Caps
}

func (p *program) Init() tea.Cmd { return tea.Batch(p.probe.Init(), p.ws.Init()) }

func (p *program) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{p.probe.Update(msg), p.ws.Update(msg)}
	switch m := msg.(type) {
	case termcap.CapsMsg:
		p.caps = m.Caps
	case tea.KeyPressMsg:
		if m.String() == "ctrl+c" {
			// Quit through the prober, which resets mode 2031 first.
			return p, tea.Batch(append(cmds, p.probe.Quit())...)
		}
	}
	return p, tea.Batch(cmds...)
}

func (p *program) View() tea.View {
	v := p.ws.View()
	v.KeyboardEnhancements = termcap.KeyboardFlags(p.caps)
	v.ReportFocus = true // for termsvc's notifications
	return v
}

type pane struct{}

func (pane) Update(tea.Msg) (workspace.Pane, tea.Cmd) { return pane{}, nil }
func (pane) View(int, int) string                     { return "" }

// A program keeps a Prober beside its workspace, and quits through it.
// Split replies are dropped before they reach the program as keys.
func ExampleProber() {
	probe := termcap.New()
	ws := workspace.New(layout.Pane{ID: "main"}, map[layout.PaneID]workspace.Pane{"main": pane{}},
		workspace.WithoutBackgroundQuery())
	prog := tea.NewProgram(&program{probe: probe, ws: ws},
		tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			if probe.IsReplyFragment(msg) {
				return nil
			}
			return msg
		}))
	_ = prog // prog.Run() in a real program
}

// A doctor command writes the report to the writer it is given; a
// program's main may pass os.Stdout. Here it goes to a buffer, and its
// first lines are printed.
func ExampleReport() {
	var c termcap.Caps
	c.Terminal.Set("WezTerm 20240203", termcap.Queried)
	c.Brand.Set(termcap.BrandWezTerm, termcap.Queried)
	c.KittyKeyboard.Set(termcap.Supported, termcap.Queried)

	var b strings.Builder
	if err := termcap.Report(&b, c, termcap.WithReportWidth(80)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for _, l := range strings.Split(b.String(), "\n")[:3] {
		fmt.Println(l)
	}
	// Output:
	// complete               no
	// timed_out              no
	// terminal               WezTerm 20240203 (query)
}
