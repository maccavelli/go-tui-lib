// Package termcaptea drives termcap's probe from a Bubble Tea program
// (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md D3, D4
// and A2).
//
// A Prober is a termcap.Probe's host: it starts the probe on the first
// tea.EnvMsg, turns the replies Bubble Tea and ultraviolet decode into
// termcap.Reply values, writes what the probe asks for, runs its hooks as
// commands, and delivers its facts as a termcap.CapsMsg. Every option is a
// termcap.Option. KeyboardEnhancements turns termcap's keyboard view into
// what a View asks for, and FromColorProfile and ToColorProfile convert
// colorprofile's profile to termcap's and back.
//
// A program that does not run Bubble Tea drives a termcap.Probe itself,
// and needs no part of this package.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package termcaptea
