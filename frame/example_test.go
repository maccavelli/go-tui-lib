package frame_test

import (
	"fmt"

	"github.com/maccavelli/go-tui-lib/v2/frame"
	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// answer writes a reply into b from a goroutine of its own, as a network
// reader does, and closes b at the reply's end.
func answer(b *stream.Buffer) {
	for _, c := range []string{"# Pacing\n\nEach line ", "is released ", "a frame at a time.\n\n", "Then the reply ends.\n"} {
		_, _ = b.Write([]byte(c))
	}
	_ = b.Close()
}

// A line-mode host pacing a reply that streams in from a goroutine, with no
// Bubble Tea (docs/guides/streaming-content.md): the Buffer asks the
// Scheduler for a frame as each line completes; on each frame the host
// pulls the lines its pacer releases, and draws.
func ExampleScheduler() {
	// guide:scheduler
	s := frame.NewScheduler()
	defer s.Close()
	b := stream.NewBuffer(s) // a completed line asks s for a frame
	d := stream.NewDoc(nil)
	pacer := frame.Adaptive()
	go answer(b)
	for !d.Finished() {
		due := <-s.Frames()      // a host selects on Frames beside its input and its timers
		d.Pull(b, pacer, due.At) // the lines due; another frame is asked for while more wait
		_ = d.Render(40)         // draw the frame
	}
	// guide:end
	fmt.Print(d.Render(40))
	// Output:
	// # Pacing
	//
	// Each line is released a frame at a time.
	//
	// Then the reply ends.
}
