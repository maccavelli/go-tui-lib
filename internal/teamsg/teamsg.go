// Package teamsg holds the one tea.Cmd that delivers a message, so that no
// package writes it again
// (docs/decisions/0014-PLAN-canonicalization.md Step 6).
//
// Stability: internal.
package teamsg

import tea "charm.land/bubbletea/v2"

// Cmd is the command that delivers m.
func Cmd[M any](m M) tea.Cmd { return func() tea.Msg { return m } }
