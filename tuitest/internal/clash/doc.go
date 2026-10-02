// Package clash is a consumer of tuitest that defines its own -update flag,
// as many Go test packages do. Its test proves that importing tuitest does
// not panic with "flag redefined: update"
// (docs/decisions/0002-PLAN-harden-workspace-v0-1-1.md, Step 2).
package clash
