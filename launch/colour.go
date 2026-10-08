package launch

import (
	"slices"

	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// colourEnv is env with the two rules of
// docs/decisions/0013-MADR-cli-integration-helpers.md §5 applied, in the
// form colorprofile reads, where the last value of a variable wins:
//   - NO_COLOR set and not empty, whatever its value, becomes NO_COLOR=1,
//     since colorprofile honours only a value strconv.ParseBool accepts;
//   - FORCE_COLOR set, not empty, and neither "0" nor "false", adds
//     CLICOLOR_FORCE=1 when CLICOLOR_FORCE is unset. colorprofile does not
//     read FORCE_COLOR, and NO_COLOR still wins over it there.
func colourEnv(env termcap.Env) []string {
	out := slices.Clone([]string(env))
	if env.Getenv("NO_COLOR") != "" {
		out = append(out, "NO_COLOR=1")
	}
	if v := env.Getenv("FORCE_COLOR"); v != "" && v != "0" && v != "false" {
		if _, ok := env.LookupEnv("CLICOLOR_FORCE"); !ok {
			out = append(out, "CLICOLOR_FORCE=1")
		}
	}
	return out
}

// profile is the colour profile of a stream: NoTTY when it is not a
// terminal, else colorprofile.Env of the adjusted environment (A1.3). Env
// reads the environment and, on Windows with TERM unset, empty or dumb,
// the system's build number (A1.7). It opens no file and starts no
// process, where colorprofile.Detect would load the terminfo entry and,
// inside tmux, run tmux.
func profile(terminal bool, env termcap.Env) colorprofile.Profile {
	if !terminal {
		return colorprofile.NoTTY
	}
	return colorprofile.Env(colourEnv(env))
}
