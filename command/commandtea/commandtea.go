// Package commandtea runs command's registry in a Bubble Tea program
// (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md D3, D6
// and A1.2). A command's effect is a command.Effect, which needs no Bubble
// Tea; Cmd turns one into a tea.Cmd, and Effect a tea.Cmd into one, so a
// handler can return what a workspace method gives. Dispatch, Watch,
// WatchContext, RunLoop, Attach and WithLoop are the registry's forms for
// a program's Update and its Send.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package commandtea

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
)

// Effect is c as a command.Effect, for a handler's Result.Cmd. A nil c is
// a nil Effect.
func Effect(c tea.Cmd) command.Effect {
	if c == nil {
		return nil
	}
	return func() any { return c() }
}

// Cmd is e as a tea.Cmd. A nil e is a nil tea.Cmd. When e's value is an
// []command.Effect, Cmd runs it as tea.Batch of each converted, at once,
// so the program gets their messages as one tea.BatchMsg, as from a
// tea.Batch it ran; any other value is the message.
func Cmd(e command.Effect) tea.Cmd {
	if e == nil {
		return nil
	}
	return func() tea.Msg {
		v := e()
		batch, ok := v.([]command.Effect)
		if !ok {
			return v
		}
		cmds := make([]tea.Cmd, len(batch))
		for i, b := range batch {
			cmds[i] = Cmd(b)
		}
		if c := tea.Batch(cmds...); c != nil {
			return c()
		}
		return nil
	}
}

// Dispatch is r.Dispatch(ctx, req) as a tea.Cmd, for a program's Update.
func Dispatch(ctx context.Context, r *command.Registry, req command.Request) tea.Cmd {
	return Cmd(r.Dispatch(ctx, req))
}

// Watch returns a command that waits for r's next change, and then
// delivers command.ChangedMsg, with a command.ConflictMsg when that change
// renamed or refused a command. The program issues it again after each
// ChangedMsg. It waits for ever; WatchContext stops waiting.
func Watch(r *command.Registry) tea.Cmd { return WatchContext(context.Background(), r) }

// WatchContext is Watch, and returns a nil message, which Bubble Tea
// ignores, once ctx ends: a program that stops listening cancels ctx, and
// leaves no goroutine waiting.
func WatchContext(ctx context.Context, r *command.Registry) tea.Cmd {
	return Cmd(r.Watch(ctx))
}

// RunLoop runs a command.LoopMsg on the program's loop: the host's Update
// returns RunLoop(msg), and the command's Result.Cmd is the tea.Cmd it
// gives.
func RunLoop(m command.LoopMsg) tea.Cmd { return Cmd(m.Run()) }

// Attach is r.Attach with a program's Send: a Loop command started off the
// loop while attached runs on the program's loop, through a
// command.LoopMsg, until detach is called.
func Attach(r *command.Registry, send func(tea.Msg)) (detach func()) {
	return r.Attach(func(m any) { send(m) })
}

// WithLoop is command.WithLoop with a program's Send.
func WithLoop(send func(tea.Msg)) command.RegistryOption {
	return command.WithLoop(func(m any) { send(m) })
}
