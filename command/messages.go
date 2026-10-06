package command

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// ResultMsg follows every Dispatch: the request, what the command
// produced, its error, and how long it ran. Err is ErrUnknown,
// ErrUnavailable or ErrRefused under errors.Is when the command did not
// run.
type ResultMsg struct {
	Request  Request
	Result   Result
	Err      error
	Duration time.Duration
}

// PromptMsg carries a Prompt or Forward command's text, for the host to
// send to its agent.
type PromptMsg struct {
	Text    string
	Command Command
	Request Request
}

// ChangedMsg is Watch's message: the registry's commands changed, and
// Version is the version after the change.
type ChangedMsg struct {
	Version uint64
}

// ConflictMsg reports the slash names a load had to rename.
type ConflictMsg struct {
	Conflicts []Conflict
}

// LoopMsg is a Loop command that Run, called off the event loop, hands to
// the program (WithLoop). The host's Update returns msg.Run(), which runs
// the command on the loop and returns its Result.Cmd.
type LoopMsg struct{ run func() tea.Cmd }

// Run runs the command and returns its effect for the program.
func (m LoopMsg) Run() tea.Cmd {
	if m.run == nil {
		return nil
	}
	return m.run()
}

// QuitRequestMsg asks the program to quit. The program decides; the
// registry never quits it.
type QuitRequestMsg struct{}

// Conflict is a loaded command whose slash name was taken: it asked for
// Slash, which Holder has, and was given Renamed, or no slash name when
// Renamed is empty. When Err is set, the command was not loaded at all,
// and Err says why (docs/decisions/0006-MADR-command-registry.md A6).
type Conflict struct {
	ID      ID
	Slash   string
	Renamed string
	Holder  ID
	Err     error
}
