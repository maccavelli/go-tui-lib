package termcap

// EnvCaps is the facts the environment gives on goos, without a program or
// a probe: exactly what a prober built with WithDisabled holds after the
// first tea.EnvMsg, so a program's own command line can decide from them
// (docs/decisions/0014-MADR-native-integration-api.md W2).
//
// opts are the prober's options. WithAppearanceEnv and WithOverride apply.
// The options that only shape a probe, WithTimeout, WithQuery,
// WithoutHeuristic, WithoutColorSchemeUpdates and WithoutBackgroundRequest,
// change nothing. The hooks, WithAppearanceHook and WithConsoleHost, are
// not run: they are commands, and EnvCaps runs none. goos wins over
// WithGOOS. Profile stays unknown, since tea's colour profile is not
// known here; a JetBrains terminal's query facts carry
// ReasonJetBrainsPaints, as a disabled prober's do.
func EnvCaps(env Env, goos string, opts ...Option) Caps {
	o := make([]Option, 0, len(opts)+2)
	o = append(append(o, opts...), WithDisabled(), WithGOOS(goos))
	p := New(o...)
	p.start(env) // its command, the hooks and the CapsMsg, is not run
	return p.Caps()
}
