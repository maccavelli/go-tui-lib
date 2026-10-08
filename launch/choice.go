package launch

import "github.com/maccavelli/go-tui-lib/internal/enum"

// Choice is the program's request, bound to its own flag: auto, tui or
// plain. Its methods make it a flag value in each framework
// docs/reports/0014-REPORT-api-assessment-and-integration-research.md §3.2
// measured:
//   - String and Set: the standard flag package, pflag, urfave/cli and ff;
//   - Type: pflag and Cobra, which print "mode" in help;
//   - Get: urfave/cli v3, which requires it;
//   - MarshalText and UnmarshalText: Kong, go-arg, flag.TextVar and
//     urfave/cli's TextFlag;
//   - MarshalFlag and UnmarshalFlag: go-flags.
//
// The tokens are "auto", "tui" and "plain", exactly.
type Choice uint8

// The choices. ChoiceAuto is the zero value.
const (
	// ChoiceAuto decides from the streams and the environment.
	ChoiceAuto Choice = iota
	// ChoiceTUI asks for the TUI, as --tui does: the environment's vetoes
	// (Config.NoInputEnv and CI) are skipped, but TERM=dumb and a missing
	// terminal still give a plain decision.
	ChoiceTUI
	// ChoicePlain never starts the TUI.
	ChoicePlain
)

var choiceNames = []string{"auto", "tui", "plain"}

// String is the choice's token.
func (c Choice) String() string { return enum.Name(choiceNames, c) }

// Set reads a token, for a flag.Value.
func (c *Choice) Set(s string) error { return c.UnmarshalText([]byte(s)) }

// Type is the value's name in pflag's help: "mode".
func (c Choice) Type() string { return "mode" }

// Get is the choice itself, for flag.Getter and urfave/cli.
func (c Choice) Get() any { return c }

// MarshalText is the choice's token. A choice with no token is an error.
func (c Choice) MarshalText() ([]byte, error) {
	return enum.Marshal("launch", "Choice", choiceNames, c)
}

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (c *Choice) UnmarshalText(b []byte) error {
	return enum.Unmarshal("launch", "Choice", choiceNames, b, c)
}

// MarshalFlag is the choice's token, for go-flags.
func (c Choice) MarshalFlag() (string, error) {
	b, err := c.MarshalText()
	return string(b), err
}

// UnmarshalFlag reads a token, for go-flags.
func (c *Choice) UnmarshalFlag(s string) error { return c.UnmarshalText([]byte(s)) }
