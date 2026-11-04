//go:build race

package stream_test

// raceBuild reports a build with the race detector, which allocates more
// than the build that ships, so an allocation budget takes its own
// ceiling there (docs/decisions/0025-PLAN-charm-free-stream-allocations.md
// deviation D3).
const raceBuild = true
