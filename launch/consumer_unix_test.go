//go:build unix

package launch_test

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/launch"
	"github.com/maccavelli/go-tui-lib/launch/launchtest"
)

// tickQuit is a model that quits on a timer.
type tickQuit struct{ model }

type tickMsg struct{}

func (m tickQuit) Init() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m tickQuit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tickMsg); ok {
		return m, tea.Quit
	}
	return m, nil
}

// TestRunSignalsStayTheProgramsUnix: Run installs no signal handler, so a
// SIGINT reaches the program's own notification and not Bubble Tea, and
// the TUI runs on to its own end.
func TestRunSignalsStayTheProgramsUnix(t *testing.T) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	got := within(t, 5*time.Second, func() error {
		_, err := launch.Run(t.Context(), launch.Streams{In: idle(t), Out: launchtest.NewTerminal(80, 24)}, interactive(), tickQuit{},
			launch.OnStart(func(*tea.Program) {
				go func() {
					time.Sleep(50 * time.Millisecond)
					if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
						t.Error(err)
					}
				}()
			}))
		return err
	})
	if got != nil {
		t.Errorf("Run = %v; want the model's own quit, nil", got)
	}
	select {
	case <-sig:
	case <-time.After(time.Second):
		t.Error("the program's own notification did not get the SIGINT")
	}
}
