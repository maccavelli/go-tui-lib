package termcap

import "time"

// EnvCaps is the facts the environment gives on goos, without a program or
// a probe: exactly what a Probe built with WithoutQueries holds once
// started, so a program's own command line can decide from them
// (docs/decisions/0014-MADR-native-integration-api.md W2).
//
// opts are the probe's options. WithAppearanceEnv and WithOverride apply.
// The options that only shape a probe, WithTimeout, WithQuery,
// WithoutHeuristic, WithoutColorSchemeUpdates, WithoutBackgroundRequest
// and WithoutModeQuery, change nothing. The hooks, WithAppearanceHook and
// WithConsoleHost, are not run: a started probe hands them to its host,
// and EnvCaps runs none. goos wins over WithGOOS. Profile stays unknown,
// since no host detected it; a JetBrains terminal's query facts carry
// ReasonJetBrainsPaints, as a disabled probe's do.
func EnvCaps(env Env, goos string, opts ...Option) Caps {
	o := make([]Option, 0, len(opts)+2)
	o = append(append(o, opts...), WithoutQueries(), WithGOOS(goos))
	p := NewProbe(o...)
	p.Start(env, time.Time{}) // its events, the hooks and ProbeDone, are not taken
	return p.Caps()
}
