package termcap_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/termcap"
	"github.com/maccavelli/go-tui-lib/termcap/termcaptest"
	"github.com/maccavelli/go-tui-lib/tuitest"
)

// app embeds a prober, and quits through its Quit.
type app struct{ p *termcap.Prober }

func (a app) Init() tea.Cmd { return a.p.Init() }

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := a.p.Update(msg)
	if _, ok := msg.(termcaptest.StopMsg); ok {
		return a, tea.Batch(cmd, a.p.Quit())
	}
	return a, cmd
}

func (a app) View() tea.View { return tea.NewView("") }

// TestReportGolden reports what each termcaptest profile yields, at 80 and
// 120 columns. The report is ASCII with no colour, so the matrix reduces to
// its widths (docs/decisions/0005-MADR-terminal-capabilities-and-services.md
// Confirmation).
func TestReportGolden(t *testing.T) {
	// The Kitty flags in CapsMsg depend on whether tea's first render, which
	// pushes them, reaches the terminal before the probe's own query, as on
	// a real terminal. The override pins them so the files are stable.
	pin := termcap.WithOverride(func(c *termcap.Caps) { c.KeyboardFlags = 0 })
	for _, p := range []termcaptest.Profile{
		termcaptest.Kitty(), termcaptest.XTerm(), termcaptest.Tmux(true), termcaptest.Tmux(false),
		termcaptest.DA1Only(), termcaptest.Silent(),
	} {
		c := termcaptest.Run(t, app{termcap.New(pin, termcap.WithTimeout(100*time.Millisecond))}, p)
		name := "report-" + strings.ReplaceAll(p.Name, " ", "-")
		for _, w := range []int{80, 120} {
			var b strings.Builder
			if err := termcap.Report(&b, c, termcap.WithReportWidth(w)); err != nil {
				t.Fatal(err)
			}
			tuitest.Text(t, name+"."+strconv.Itoa(w), b.String())
		}
	}
}
