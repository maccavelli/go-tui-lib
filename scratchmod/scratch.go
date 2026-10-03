// Package scratchmod is a throwaway module that proves CI builds a matrix
// entry for a nested module (docs/decisions/0010-PLAN-nested-adapter-modules.md
// Phase 5). It is never merged.
package scratchmod

import "github.com/maccavelli/go-tui-lib/glyph"

// Bullet returns the ASCII bullet.
func Bullet() string { return glyph.ASCII().Bullet }
