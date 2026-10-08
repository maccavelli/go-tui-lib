package launch

import (
	"io"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// Streams are the program's own standard streams and environment. main
// passes os.Stdin, os.Stdout, os.Stderr and os.Environ(); a Cobra command
// passes its own, through FromSource; an SSH session passes the session's.
// launch never reads a process-wide stream or variable itself.
type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
	Env termcap.Env
}

// StreamSource is what a *cobra.Command has: its input, output and error
// streams, each the process's unless the program set its own.
type StreamSource interface {
	InOrStdin() io.Reader
	OutOrStdout() io.Writer
	ErrOrStderr() io.Writer
}

// FromSource is src's streams, with env.
func FromSource(src StreamSource, env termcap.Env) Streams {
	return Streams{In: src.InOrStdin(), Out: src.OutOrStdout(), Err: src.ErrOrStderr(), Env: env}
}
