// Package frame paces drawing: a Queue of items released a frame at a time
// by a Pacer, and a Scheduler that merges requests for a frame into one, at
// most a frame-rate's worth a second
// (docs/decisions/0009-MADR-streaming-content-engine.md §3, A6 items 8 and
// 10, A7).
//
// The package has two layers. The pure layer takes the time from its
// caller and reads no clock: a Queue, its Snapshot, and a Pacer that
// decides how many items a frame releases. Adaptive is codex's two gears:
// Smooth releases one item a frame, and CatchUp the whole queue, entered
// when the queue grows deep or old, left once it has stayed shallow and
// young a while. Immediate releases everything every frame. A Queue and a
// Pacer are used on one goroutine.
//
// The Scheduler is the driver: its Request and RequestIn are safe for
// concurrent use, so a stream's producers ask for a frame from their own
// goroutines, and the frames come on one channel of capacity one, which
// merges every request made while a frame is waiting. Between frames no
// goroutine runs: one is started when a frame falls due, and stops once it
// is sent. Close stops it and closes the channel. A DueMsg carries the
// time its frame fell due, which is the now a program gives its Pacer.
//
// A line-mode host selects on Frames beside its input; a Bubble Tea
// program uses frame/frametea, which turns the channel into commands.
//
// The package links no Charm module.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package frame
