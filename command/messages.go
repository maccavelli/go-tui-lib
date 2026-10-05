package command

import "time"

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

// QuitRequestMsg asks the program to quit. The program decides; the
// registry never quits it.
type QuitRequestMsg struct{}

// Conflict is a loaded command whose slash name was taken: it asked for
// Slash, which Holder has, and was given Renamed.
type Conflict struct {
	ID      ID
	Slash   string
	Renamed string
	Holder  ID
}
