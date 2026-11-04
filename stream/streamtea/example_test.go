package streamtea_test

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/v2/frame"
	"github.com/maccavelli/go-tui-lib/v2/frame/frametea"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/layout"
	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/stream/streamtea"
	"github.com/maccavelli/go-tui-lib/v2/theme"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// guide:docpane
// model is a program whose workspace shows an answer as it streams in.
type model struct {
	w      *workspace.Workspace
	frames *frame.Scheduler
}

func (m model) Init() tea.Cmd { return tea.Batch(m.w.Init(), frametea.Wait(m.frames)) }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.w.Update(msg) // a frame.DueMsg reaches the DocPane, which pulls its Buffer
	if _, ok := msg.(frame.DueMsg); ok {
		cmd = tea.Batch(cmd, frametea.Wait(m.frames)) // the one line: wait for the next frame
	}
	return m, cmd
}

func (m model) View() tea.View { return tea.NewView(m.w.Render()) }

// answer writes the answer into the Buffer from a goroutine of its own, as
// a network reader does, and closes it at the end.
func answer(b *stream.Buffer, chunks []string) {
	for _, c := range chunks {
		_, _ = b.Write([]byte(c))
	}
	_ = b.Close()
}

// guide:end

// An answer streamed from a goroutine through a Buffer into a DocPane in a
// workspace (docs/guides/streaming-content.md). The loop below does what
// Bubble Tea's does with the program's commands: it runs Wait, and gives
// its frame.DueMsg to Update.
func ExampleDocPane() {
	s := frame.NewScheduler()
	defer s.Close()
	b := stream.NewBuffer(s)
	d := stream.NewDoc(nil)
	km, _ := keymap.New(keymap.Settings{}, append(workspace.DefaultRules(), streamtea.DefaultRules()...))
	w := workspace.New(layout.Pane{ID: "answer"}, map[layout.PaneID]workspace.Pane{"answer": placeholder{}},
		workspace.WithBindings(km), workspace.WithChrome(workspace.None),
		workspace.WithTheme(theme.New(colorprofile.ASCII, theme.Unknown, glyph.ASCII())), workspace.WithoutBackgroundQuery())
	w.SetPane("answer", streamtea.NewDocPane(w, d, streamtea.WithBuffer(b)))
	m := model{w: w, frames: s}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 5})

	go answer(b, []string{"# An answer\n\nIt streams in ", "from a goroutine, a line a frame, ", "and the pane follows its tail.\n"})
	for !d.Finished() {
		m.Update(frametea.Wait(s)())
	}
	for l := range strings.SplitSeq(m.View().Content, "\n") {
		fmt.Println(strings.TrimRight(l, " "))
	}
	// Output:
	// # An answer
	//
	// It streams in from a goroutine, a line
	// a frame, and the pane follows its tail.
}
