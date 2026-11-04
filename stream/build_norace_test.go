//go:build !race

package stream_test

// raceBuild reports a build with the race detector; this build has none
// (docs/decisions/0025-PLAN-charm-free-stream-allocations.md deviation D3).
const raceBuild = false
