// Package frametea gives a Bubble Tea program frame's frames
// (docs/decisions/0009-MADR-streaming-content-engine.md §3, A6 item 8): Wait
// is a command that waits for the Scheduler's next frame and returns it as
// a frame.DueMsg, and Forward pushes every frame to the program's Send.
// TickCmd ticks an animation at its frame.Demand, CoalesceCmd delivers a
// frame.Coalescer's deadline, and WaitUpgrade an Upgrader's next result.
//
// A program re-arms Wait on each frame:
//
//	case frame.DueMsg:
//		return m, tea.Batch(m.pane.Update(msg), frametea.Wait(m.frames))
//
// The package holds conversions, and no logic of frame's.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package frametea

import (
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/frame"
)

// Wait is a command that waits for s's next frame, and returns it as a
// frame.DueMsg; once s is closed, it returns nil. It runs on Bubble Tea's
// command goroutine, as every command does.
func Wait(s *frame.Scheduler) tea.Cmd {
	return func() tea.Msg {
		m, ok := <-s.Frames()
		if !ok {
			return nil
		}
		return m
	}
}

// Forward sends each of s's frames to send, the program's Send, from a
// goroutine of its own, until stop is called or s is closed. stop waits
// for the goroutine, and may be called more than once. A send that blocks,
// as a program's Send does until the program reads, holds stop until it
// returns.
func Forward(s *frame.Scheduler, send func(tea.Msg)) (stop func()) {
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			select {
			case m, ok := <-s.Frames():
				if !ok {
					return
				}
				send(m)
			case <-done:
				return
			}
		}
	})
	var once sync.Once
	return func() {
		once.Do(func() { close(done) })
		wg.Wait()
	}
}

// TickCmd is a command that ticks once, after t's interval for d, as a
// frame.TickMsg; for frame.None it is nil, so that a program with nothing
// animating arms no timer. A program re-arms it on each TickMsg, with the
// demand of what it drew.
func TickCmd(d frame.Demand, t frame.Ticks) tea.Cmd {
	iv := t.Interval(d)
	if iv == 0 {
		return nil
	}
	return tea.Tick(iv, func(at time.Time) tea.Msg { return frame.TickMsg{At: at, Demand: d} })
}

// CoalesceCmd is a command that returns a frame.FlushMsg at due, the
// deadline a Coalescer's Push gave, on which the program calls Flush.
func CoalesceCmd(due time.Time) tea.Cmd {
	return tea.Tick(time.Until(due), func(time.Time) tea.Msg { return frame.FlushMsg{At: due} })
}

// WaitUpgrade is a command that waits for u's next result, and returns it
// as a frame.UpgradeMsg; once u is closed, it returns nil. A program
// re-arms it on each result.
func WaitUpgrade[K comparable, R any](u *frame.Upgrader[K, R]) tea.Cmd {
	return func() tea.Msg {
		m, ok := <-u.Results()
		if !ok {
			return nil
		}
		return m
	}
}
