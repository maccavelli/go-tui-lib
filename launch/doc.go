// Package launch starts go-tui-lib's TUI from a program's own command line,
// on the program's own streams, and only where it can run
// (docs/decisions/0013-MADR-cli-integration-helpers.md, as amended by A1).
//
// A program binds Flags, or a Choice of its own, to its CLI framework: the
// standard flag package, Cobra and pflag, Kong, urfave/cli and others each
// take them natively. It builds Streams from its own standard streams and
// environment, with FromSource for a Cobra command. Decide then says whether
// the TUI can start, where it reads and draws, and why, as a Decision whose
// Reason is a stable token.
//
// For a program whose default is its own CLI mode, --tui asks for the TUI,
// and a TUI that cannot start falls back to that CLI mode:
//
//	var f launch.Flags
//	f.RegisterFlags(fs)
//	// … parse …
//	if f.Resolve() == launch.ChoiceTUI {
//		d := launch.Decide(s, launch.Config{Choice: launch.ChoiceTUI})
//		if !d.Interactive {
//			fmt.Fprintf(s.Err, "Warning: --tui unavailable (%s); using the CLI\n", d.Reason)
//			// … run the CLI mode …
//		}
//	}
//
// launch never reads the process's environment or its standard streams:
// every stream and variable comes from the Streams a program passes, so a
// test and an SSH session each pass their own. It never installs a signal
// handler, never sets the alternate screen, and never starts a process: the
// colour profile comes from the environment and the terminal test, and on
// Windows with no TERM, from the system's build number.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package launch
