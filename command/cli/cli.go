// Package cli runs a command.Registry's commands as shell subcommands, with
// flags from their schemas and JSON output, so a program offers the same
// commands in its TUI and from the shell
// (docs/decisions/0006-MADR-command-registry.md §10, A9).
//
// A command's ID becomes words, "prog workspace state get", or stays
// dotted, "prog workspace.state.get". Each argument is a flag, --delta 4,
// which an array repeats and an object takes as JSON; positional arguments
// come in order; --args takes the whole object as JSON; and --json prints
// the result's value as JSON. The verbs list, describe, schema and help
// describe the commands.
//
// A shell invocation is command.OriginCLI. A destructive command asks
// WithConfirm, or without it needs --yes. Only commands offered on
// command.SurfaceCLI are listed or run, and Result.Cmd is ignored: there
// is no event loop. Run writes only to the writers it is given and never
// exits the process.
package cli

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/when"
)

// The exit codes Run returns.
const (
	ExitOK      = 0 // the command ran
	ExitFailed  = 1 // the command failed, or is not available now
	ExitUsage   = 2 // an unknown command or flag, or a bad argument
	ExitRefused = 3 // a destructive command was not confirmed
)

// Option configures Run.
type Option func(*config)

type config struct {
	name    string
	confirm func(prompt string) bool
	context when.Context
	width   int
}

// WithName is the program's name in usage and help (default "app").
func WithName(name string) Option { return func(c *config) { c.name = name } }

// WithConfirm asks the user before a destructive command runs without
// --yes; it returns true to run it.
func WithConfirm(f func(prompt string) bool) Option { return func(c *config) { c.confirm = f } }

// WithContext is the context commands' When is evaluated in (default
// empty).
func WithContext(c when.Context) Option { return func(cfg *config) { cfg.context = c } }

// WithWidth is the width help wraps at, in cells (default 80).
func WithWidth(w int) Option { return func(c *config) { c.width = w } }

// Run runs args, a program's arguments without its name, against r, and
// returns the exit code. It writes results to stdout and errors and help
// for a usage error to stderr, each once, when the command has finished;
// a write that fails makes the exit code ExitFailed.
func Run(ctx context.Context, r *command.Registry, args []string, stdout, stderr io.Writer, o ...Option) int {
	cfg := config{name: "app", width: 80}
	for _, f := range o {
		f(&cfg)
	}
	if cfg.context == nil {
		cfg.context = when.Map{}
	}
	s := shell{cfg: cfg, r: r, out: &strings.Builder{}, err: &strings.Builder{}}
	code := s.dispatch(ctx, args)
	if _, err := io.WriteString(stdout, s.out.String()); err != nil {
		code = ExitFailed
	}
	if _, err := io.WriteString(stderr, s.err.String()); err != nil {
		code = ExitFailed
	}
	return code
}

// shell is one Run: its output is gathered, and written when it ends.
type shell struct {
	cfg      config
	r        *command.Registry
	out, err *strings.Builder
}

// dispatch runs one command line: a verb, or a command.
func (s shell) dispatch(ctx context.Context, args []string) int {
	if len(args) == 0 {
		s.registryHelp(s.err)
		return ExitUsage
	}
	switch args[0] {
	case "help", "-h", "--help", "-help":
		return s.help(args[1:])
	case "list":
		return s.list(args[1:])
	case "describe":
		return s.describe(args[1:])
	case "schema":
		return s.schema(args[1:])
	}
	return s.run(ctx, args)
}

func (s shell) usagef(format string, a ...any) int {
	fmt.Fprintf(s.err, "%s: "+format+"\n", append([]any{s.cfg.name}, a...)...)
	fmt.Fprintf(s.err, "Run '%s help' for the commands.\n", s.cfg.name)
	return ExitUsage
}

// offered reports whether c is one the shell lists and runs.
func offered(c command.Command) bool {
	return c.Surfaces == 0 || c.Surfaces&command.SurfaceCLI != 0
}

// resolve finds the command args name: the longest run of leading words,
// joined with dots, that is an ID offered on the CLI surface. The rest are
// its arguments.
func (s shell) resolve(args []string) (command.Command, []string, bool) {
	words := 0
	for words < len(args) && !strings.HasPrefix(args[words], "-") {
		words++
	}
	for n := words; n > 0; n-- {
		if c, ok := s.r.Lookup(command.ID(strings.Join(args[:n], "."))); ok && offered(c) {
			return c, args[n:], true
		}
	}
	return command.Command{}, nil, false
}

func (s shell) run(ctx context.Context, args []string) int {
	c, rest, ok := s.resolve(args)
	if !ok {
		return s.usagef("unknown command %q", strings.Join(args, " "))
	}
	props, err := properties(c.Args)
	if err != nil {
		return s.usagef("%s: its schema cannot be read: %v", c.ID, err)
	}
	p, err := parse(props, rest)
	if err != nil {
		return s.argError(c, err)
	}
	if p.help {
		s.commandHelp(s.out, c, props)
		return ExitOK
	}
	raw, err := p.request()
	if err != nil {
		return s.argError(c, err)
	}
	res, err := s.r.Run(ctx, command.Request{
		ID: c.ID, Args: raw, Origin: command.OriginCLI, Caller: s.cfg.name, Context: s.cfg.context,
		Gate: confirmGate{yes: p.yes, confirm: s.cfg.confirm},
	})
	if err != nil {
		return s.failed(c, err)
	}
	return s.print(res, p.asJSON)
}

func (s shell) argError(c command.Command, err error) int {
	fmt.Fprintf(s.err, "%s: %s: %v\n", s.cfg.name, c.ID, err)
	fmt.Fprintf(s.err, "Run '%s %s --help' for its arguments.\n", s.cfg.name, words(c.ID))
	return ExitUsage
}

// failed maps a run's error to an exit code (docs/decisions/0006-MADR-command-registry.md A9).
func (s shell) failed(c command.Command, err error) int {
	fmt.Fprintf(s.err, "%s: %v\n", s.cfg.name, err)
	var argErr *command.ArgError
	isArg := errors.As(err, &argErr)
	switch {
	case isArg, errors.Is(err, command.ErrUnknown):
		fmt.Fprintf(s.err, "Run '%s %s --help' for its arguments.\n", s.cfg.name, words(c.ID))
		return ExitUsage
	case errors.Is(err, command.ErrRefused):
		if c.Danger == command.Destructive {
			fmt.Fprintf(s.err, "%s is destructive; pass --yes to run it.\n", words(c.ID))
		}
		return ExitRefused
	}
	return ExitFailed
}

// print writes a result: its value as JSON with --json, null when there is
// none, or else its text.
func (s shell) print(res command.Result, asJSON bool) int {
	if asJSON {
		b, err := json.Marshal(res.Value, json.Deterministic(true))
		if err != nil {
			fmt.Fprintf(s.err, "%s: the result is not JSON: %v\n", s.cfg.name, err)
			return ExitFailed
		}
		fmt.Fprintf(s.out, "%s\n", b)
		return ExitOK
	}
	if res.Text != "" {
		fmt.Fprint(s.out, res.Text)
		if !strings.HasSuffix(res.Text, "\n") {
			fmt.Fprintln(s.out)
		}
	}
	return ExitOK
}

// confirmGate is the shell's per-request gate: --yes allows, WithConfirm
// asks, and with neither a destructive command is refused.
type confirmGate struct {
	yes     bool
	confirm func(prompt string) bool
}

func (g confirmGate) Decide(_ context.Context, inv *command.Invocation) (command.Decision, error) {
	switch {
	case g.yes:
		return command.AllowOnce, nil
	case g.confirm != nil && g.confirm(fmt.Sprintf("Run %s? It is %s.", words(inv.Command.ID), inv.Command.Danger)):
		return command.AllowOnce, nil
	}
	return command.RejectOnce, nil
}

// words is an ID as the shell's words.
func words(id command.ID) string { return strings.ReplaceAll(string(id), ".", " ") }

// list writes the commands the shell offers, or with --json the manifest
// of them.
func (s shell) list(args []string) int {
	asJSON := len(args) == 1 && args[0] == "--json"
	if len(args) > 0 && !asJSON {
		return s.usagef("list takes only --json")
	}
	var cmds []command.Command
	for c := range s.r.Available(s.cfg.context, command.SurfaceCLI) {
		cmds = append(cmds, c)
	}
	if !asJSON {
		s.listText(s.out, cmds)
		return ExitOK
	}
	m := s.r.Manifest()
	keep := map[command.ID]bool{}
	for _, c := range cmds {
		keep[c.ID] = true
	}
	var listed []command.ManifestCommand
	for _, mc := range m.Commands {
		if keep[mc.ID] {
			listed = append(listed, mc)
		}
	}
	m.Commands = listed
	return s.printJSON(m)
}

func (s shell) printJSON(v any) int {
	b, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		fmt.Fprintf(s.err, "%s: %v\n", s.cfg.name, err)
		return ExitFailed
	}
	fmt.Fprintf(s.out, "%s\n", b)
	return ExitOK
}

// one finds the single command a verb names, by ID or words.
func (s shell) one(verb string, args []string) (command.Command, bool, int) {
	if len(args) == 0 {
		return command.Command{}, false, s.usagef("%s needs a command", verb)
	}
	c, rest, ok := s.resolve(args)
	if !ok || len(rest) > 0 {
		return command.Command{}, false, s.usagef("unknown command %q", strings.Join(args, " "))
	}
	return c, true, ExitOK
}

// describe writes one command's description as JSON.
func (s shell) describe(args []string) int {
	c, ok, code := s.one("describe", args)
	if !ok {
		return code
	}
	for _, mc := range s.r.Manifest().Commands {
		if mc.ID == c.ID {
			return s.printJSON(mc)
		}
	}
	return s.usagef("unknown command %q", c.ID)
}

// schema writes one command's argument schema.
func (s shell) schema(args []string) int {
	c, ok, code := s.one("schema", args)
	if !ok {
		return code
	}
	schema := c.Args
	if len(schema) == 0 {
		schema = command.Schema(`{"type":"object"}`)
	}
	fmt.Fprintf(s.out, "%s\n", schema)
	return ExitOK
}

// help writes the registry's help, or one command's.
func (s shell) help(args []string) int {
	if len(args) == 0 {
		s.registryHelp(s.out)
		return ExitOK
	}
	c, ok, code := s.one("help", args)
	if !ok {
		return code
	}
	props, err := properties(c.Args)
	if err != nil {
		return s.usagef("%s: its schema cannot be read: %v", c.ID, err)
	}
	s.commandHelp(s.out, c, props)
	return ExitOK
}
