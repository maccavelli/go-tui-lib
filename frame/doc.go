// Package frame paces drawing: a Queue of items released a frame at a time
// by a Pacer, and a Scheduler that merges requests for a frame into one, at
// most a frame-rate's worth a second
// (docs/decisions/0009-MADR-streaming-content-engine.md §3, A1, A6 items 8
// and 10, A7).
//
// The package has two layers. The pure layer takes the time from its
// caller and reads no clock: a Queue, its Snapshot, and a Pacer that
// decides how many items a frame releases. Adaptive is codex's two gears:
// Smooth releases one item a frame, and CatchUp the whole queue, entered
// when the queue grows deep or old, left once it has stayed shallow and
// young a while. Immediate releases everything every frame. A Coalescer
// batches a stream's deltas for the next frame, and applies a lifecycle
// event between them in order. A Demand is how often a view needs ticks to
// animate, and Ticks each one's interval. A Queue, a Pacer and a Coalescer
// are used on one goroutine.
//
// The Scheduler is the driver: its Request and RequestIn are safe for
// concurrent use, so a stream's producers ask for a frame from their own
// goroutines, and the frames come on one channel of capacity one, which
// merges every request made while a frame is waiting. Between frames no
// goroutine runs: one is started when a frame falls due, and stops once it
// is sent. Close stops it and closes the channel. A DueMsg carries the
// time its frame fell due, which is the now a program gives its Pacer.
//
// Two more drivers sit beside it. An Output wraps the program's terminal
// output, so that a terminal slow to read holds the program no longer than
// a wait, and reports a stall. An Upgrader runs costly renders off the
// event loop, a newer job for a key replacing the older.
//
// A line-mode host selects on Frames beside its input; a Bubble Tea
// program uses frame/frametea, which turns the channel into commands.
//
// The package links no Charm module.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package frame
