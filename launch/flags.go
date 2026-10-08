package launch

import "flag"

// Flags is the shared flag set: --mode, a Choice, and --tui, which asks for
// the TUI. Its tags serve the frameworks that read a struct: Kong embeds it
// (embed:""), go-flags groups it, ff adds it with AddStruct, and go-arg
// embeds it. TUI is a plain bool because ff refuses a *bool, and because
// Kong takes a bare --tui only into a bool. No flag means no TUI, so there
// is no --no-tui; Kong's --tui=false cancels an earlier --tui
// (docs/decisions/0013-MADR-cli-integration-helpers.md A1.10).
type Flags struct {
	Mode Choice `name:"mode" long:"mode" ff:"long=mode" default:"auto" help:"auto, tui or plain" description:"auto, tui or plain"`
	TUI  bool   `name:"tui" long:"tui" ff:"long=tui" help:"start the TUI" description:"start the TUI"`
}

// RegisterFlags adds -mode and -tui to fs, for the standard flag package.
// Cobra takes them through pflag's AddGoFlagSet, which keeps -tui a bare
// boolean, and ff through NewFlagSetFrom.
func (f *Flags) RegisterFlags(fs *flag.FlagSet) {
	fs.Var(&f.Mode, "mode", "auto, tui or plain")
	fs.BoolVar(&f.TUI, "tui", f.TUI, "start the TUI")
}

// Resolve is the choice the flags make: ChoiceTUI when TUI is set, and Mode
// otherwise.
func (f Flags) Resolve() Choice {
	if f.TUI {
		return ChoiceTUI
	}
	return f.Mode
}
