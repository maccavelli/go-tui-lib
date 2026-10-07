// Package kongcmd runs a command.Registry's commands through Kong, as
// commands of their own parser or mounted beside a program's own grammar,
// with help drawn with the library's theme and glyphs and completion for
// bash, zsh, fish and PowerShell, so a Kong program offers the same
// commands in its TUI and from the shell
// (docs/decisions/0006-MADR-command-registry.md A1, A11).
//
// New reads the registry's commands offered on command.SurfaceCLI and
// builds a Kong grammar for each from its schema, Kong-native: a required
// property is a required flag, a positional one an argument, an enum
// Kong's enum. A dotted ID becomes nested commands, "prog workspace state
// get", and Run takes the dotted form too. --args takes the whole object
// as JSON, for the properties the grammar does not require; --json prints
// the result's value as JSON. The verbs list, describe, schema, help and
// completion are command/cli's, and completion's scripts ask the program,
// through "prog __complete <words>", which Run answers. A hidden command
// runs but is neither listed nor completed.
//
// Parser returns a parser of the registry's commands alone; Options
// returns the options that mount them into a program's own kong.New. Each
// sets kong.Name, an Exit that Run recovers, the help printer, and a check
// that fails kong.New when two commands share a name. Neither sets the
// writers: the caller passes kong.Writers, and Parser discards output
// until it does, so nothing reaches a standard stream. Parse with Run,
// which recovers the Exit; for the program's own command it returns the
// parsed context with handled false, bound to the registry and to the
// run's context.Context, for the program to run.
//
// A shell invocation is command.OriginCLI. A destructive command asks
// WithConfirm, or without it needs --yes. Run returns command/cli's exit
// codes, and never exits the process. No grammar has an env tag, which
// would read the process environment; Resolver fills flags from the
// config.* keys of the context given with WithContext instead.
package kongcmd

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/colorprofile"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/command/cli"
	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/when"
)

// Option configures New.
type Option func(*config)

type config struct {
	name    string
	confirm func(prompt string) bool
	context when.Context
	theme   theme.Theme
	glyphs  *glyph.Set
	width   int
}

// WithName is the program's name, in usage and help, and the parser's
// (default "app").
func WithName(name string) Option { return func(c *config) { c.name = name } }

// WithConfirm asks the user before a destructive command runs without
// --yes; it returns true to run it.
func WithConfirm(f func(prompt string) bool) Option { return func(c *config) { c.confirm = f } }

// WithContext is the context commands' When is evaluated in, and whose
// config.* keys Resolver reads (default empty).
func WithContext(c when.Context) Option { return func(cfg *config) { cfg.context = c } }

// WithTheme is the theme help is drawn with (default: no colour, ASCII
// glyphs).
func WithTheme(t theme.Theme) Option { return func(c *config) { c.theme = t } }

// WithGlyphs replaces the theme's glyphs.
func WithGlyphs(g glyph.Set) Option { return func(c *config) { c.glyphs = &g } }

// WithWidth is the width help wraps at, in cells (default 80).
func WithWidth(w int) Option { return func(c *config) { c.width = w } }

// Adapter is a registry's commands as Kong grammars.
type Adapter struct {
	cfg     config
	r       *command.Registry
	top     []*node
	entries map[command.ID]*entry
	order   []*entry // in ID order
}

// entry is a registry command and its grammar.
type entry struct {
	c      command.Command
	props  []*property
	typ    reflect.Type
	broken error // why it cannot run: a schema that cannot be read, or a property named like a shell flag
}

// New builds the grammars of every command r offers on the CLI surface.
func New(r *command.Registry, o ...Option) (*Adapter, error) {
	if r == nil {
		return nil, errors.New("kongcmd: New needs a registry")
	}
	a := &Adapter{r: r, entries: map[command.ID]*entry{}}
	a.cfg = config{name: "app", width: 80, theme: theme.New(colorprofile.NoTTY, theme.Unknown, glyph.ASCII())}
	for _, f := range o {
		f(&a.cfg)
	}
	if a.cfg.context == nil {
		a.cfg.context = when.Map{}
	}
	if a.cfg.glyphs != nil {
		a.cfg.theme.Glyphs = *a.cfg.glyphs
	}
	root := &node{}
	for c := range r.All() {
		if !offered(c) {
			continue
		}
		e := &entry{c: c}
		props, err := properties(c.Args)
		if err != nil {
			e.broken = fmt.Errorf("its schema cannot be read: %w", err)
		}
		e.props = props
		e.typ, err = commandType(props)
		if e.broken == nil {
			e.broken = err
		}
		a.entries[c.ID] = e
		a.order = append(a.order, e)
		n, prefix := root, ""
		for seg := range c.ID.Segments() {
			if prefix != "" {
				prefix += "."
			}
			prefix += seg
			n = n.child(seg, prefix)
		}
		n.cmd = e
	}
	a.top = root.children
	for _, n := range a.top {
		if _, err := nodeType(n); err != nil {
			return nil, err
		}
	}
	return a, nil
}

// offered reports whether c is one the shell lists and runs.
func offered(c command.Command) bool {
	return c.Surfaces == 0 || c.Surfaces&command.SurfaceCLI != 0
}

// exitCode is the sentinel the adapter's Exit panics with, and Run
// recovers.
type exitCode int

// Options returns the options that mount the registry's commands, and the
// verbs, into a program's own kong.New: a kong.DynamicCommand for each
// top-level word, kong.Name, an Exit that Run recovers, the help printer,
// the groups, the registry bound for the program's own commands, and a
// check that fails kong.New on a name two commands share. Pass
// kong.Writers too, and parse with Run.
func (a *Adapter) Options() []kong.Option {
	opts := []kong.Option{
		kong.Name(a.cfg.name),
		kong.Exit(func(code int) { panic(exitCode(code)) }),
		kong.Help(a.help),
		kong.Bind(a.r),
		kong.PostBuild(checkClashes),
	}
	var groups []kong.Group
	seen := map[string]bool{}
	taken := map[string]bool{}
	for _, n := range a.top {
		taken[n.word] = true
	}
	for _, n := range a.top {
		t, err := nodeType(n)
		if err != nil {
			continue // New refused it
		}
		g := n.group()
		if g != "" && !seen[g] {
			seen[g] = true
			groups = append(groups, kong.Group{Key: g, Title: g + " commands"})
		}
		kv := commandTags(n, taken)
		var tags []string
		for i := 0; i+1 < len(kv); i += 2 {
			if kv[i] == keyName || kv[i] == keyHelp || kv[i] == keyGroup {
				continue
			}
			tags = append(tags, kv[i]+":"+quote(kv[i+1]))
		}
		help := "The " + n.word + " commands"
		if n.cmd != nil {
			help = n.cmd.c.Title
		}
		opts = append(opts, kong.DynamicCommand(n.word, strings.ReplaceAll(help, "$", "$$"), g, reflect.New(t).Interface(), tags...))
	}
	for _, v := range verbs {
		opts = append(opts, kong.DynamicCommand(v.name, v.help, "", reflect.New(verbType(v.name)).Interface(), tagVerb+":"+quote(v.name)))
	}
	if len(groups) > 0 {
		opts = append(opts, kong.ExplicitGroups(groups))
	}
	return opts
}

// parserRoot is the grammar of Parser's own root, which holds nothing but
// the registry's commands and the verbs.
type parserRoot struct{}

// Parser returns a parser of the registry's commands alone. Its writers
// discard until o gives kong.Writers.
func (a *Adapter) Parser(o ...kong.Option) (*kong.Kong, error) {
	opts := append([]kong.Option{kong.Writers(io.Discard, io.Discard)}, a.Options()...)
	return kong.New(&parserRoot{}, append(opts, o...)...)
}

// Run parses args, a program's arguments without its name, with p, runs
// the selected registry command or verb, and returns its exit code with
// handled true; it also answers "__complete". When the program's own
// command is selected, it returns the parsed context with handled false,
// bound to the registry and to ctx, for the program to run. A dotted ID
// is taken as its words. Run never calls os.Exit; Kong's Exit, after
// --help, ends the parse with exit code 0.
func (a *Adapter) Run(ctx context.Context, p *kong.Kong, args []string) (kctx *kong.Context, code int, handled bool) {
	if args == nil {
		args = []string{}
	}
	args = a.dotted(args)
	if len(args) > 0 && args[0] == completeCommand {
		return nil, a.complete(p, args[1:]), true
	}
	defer func() {
		if r := recover(); r != nil {
			c, ok := r.(exitCode)
			if !ok {
				panic(r)
			}
			kctx, code, handled = nil, int(c), true
		}
	}()
	kctx, err := p.Parse(args)
	if err != nil {
		if errors.Is(err, errWrite) {
			return kctx, cli.ExitFailed, true
		}
		return kctx, a.usage(p.Stderr, err), true
	}
	kctx.BindTo(ctx, (*context.Context)(nil))
	sel := kctx.Selected()
	if sel == nil {
		return kctx, cli.ExitOK, false
	}
	if id := sel.Tag.Get(tagID); id != "" {
		return kctx, a.runCommand(ctx, kctx, a.entries[command.ID(id)]), true
	}
	if v := sel.Tag.Get(tagVerb); v != "" {
		return kctx, a.runVerb(kctx, v), true
	}
	return kctx, cli.ExitOK, false
}

// dotted replaces the first word of args that is not a flag with the
// words of the registry command whose dotted ID it is.
func (a *Adapter) dotted(args []string) []string {
	for i, w := range args {
		if strings.HasPrefix(w, "-") {
			continue
		}
		if _, ok := a.entries[command.ID(w)]; ok && strings.Contains(w, ".") {
			return slices.Concat(args[:i], strings.Split(w, "."), args[i+1:])
		}
		break
	}
	return args
}

// usage reports a command line Kong could not parse, and returns
// cli.ExitUsage.
func (a *Adapter) usage(w io.Writer, err error) int {
	if write(w, fmt.Sprintf("%s: %v\nRun '%s --help' for the commands.\n", a.cfg.name, err, a.cfg.name)) != nil {
		return cli.ExitFailed
	}
	return cli.ExitUsage
}

// errWrite marks a write that failed.
var errWrite = errors.New("kongcmd: a write failed")

// write writes s to w, and returns errWrite when it fails.
func write(w io.Writer, s string) error {
	if _, err := io.WriteString(w, s); err != nil {
		return fmt.Errorf("%w: %w", errWrite, err)
	}
	return nil
}

// runCommand runs the registry command e from the parsed context.
func (a *Adapter) runCommand(ctx context.Context, kctx *kong.Context, e *entry) int {
	if e.broken != nil {
		return a.argError(kctx.Stderr, e, e.broken)
	}
	obj, shell, err := a.values(kctx, e)
	if err != nil {
		return a.argError(kctx.Stderr, e, err)
	}
	raw, err := request(shell.raw, obj)
	if err != nil {
		return a.argError(kctx.Stderr, e, err)
	}
	res, err := a.r.Run(ctx, command.Request{
		ID: e.c.ID, Args: raw, Origin: command.OriginCLI, Caller: a.cfg.name, Context: a.cfg.context,
		Gate: confirmGate{yes: shell.yes, confirm: a.cfg.confirm},
	})
	if err != nil {
		return a.failed(kctx.Stderr, e, err)
	}
	return a.print(kctx, res, shell.asJSON)
}

// shellFlags are the shell's own flags of one run.
type shellFlags struct {
	raw         string
	asJSON, yes bool
}

// values reads the arguments the command line gave, and the flags a
// resolver filled, from the parse's path: Kong's defaults are left to the
// registry, which applies the schema's.
func (a *Adapter) values(kctx *kong.Context, e *entry) (map[string]any, shellFlags, error) {
	obj := map[string]any{}
	var shell shellFlags
	byName := map[string]*property{}
	for _, p := range e.props {
		byName[p.name] = p
	}
	for _, el := range kctx.Path {
		var name string
		switch {
		case el.Flag != nil:
			name = el.Flag.Name
		case el.Positional != nil:
			name = el.Positional.Name
		default:
			continue
		}
		v := kctx.Value(el)
		switch name {
		case "args":
			shell.raw = v.String()
			continue
		case "json":
			shell.asJSON = v.Bool()
			continue
		case "yes":
			shell.yes = v.Bool()
			continue
		}
		p := byName[name]
		if p == nil {
			continue
		}
		x, err := jsonValue(p, v)
		if err != nil {
			return nil, shell, &command.ArgError{Path: "/" + p.name, Reason: err.Error()}
		}
		obj[p.name] = x
	}
	return obj, shell, nil
}

// jsonValue is a field's value as JSON: an object's text decoded, an enum
// the adapter checks checked.
func jsonValue(p *property, v reflect.Value) (any, error) {
	one := func(t string, v reflect.Value) (any, error) {
		if t == tObject {
			var x any
			if err := json.Unmarshal([]byte(v.String()), &x); err != nil {
				return nil, fmt.Errorf("%q is not JSON: %w", v.String(), err)
			}
			return x, nil
		}
		return v.Interface(), nil
	}
	if p.is(tArray) {
		out := make([]any, v.Len())
		for i := range v.Len() {
			x, err := one(itemType(p), v.Index(i))
			if err != nil {
				return nil, err
			}
			out[i] = x
		}
		return out, nil
	}
	x, err := one(p.typeOf(), v)
	if err != nil {
		return nil, err
	}
	if len(p.enum) > 0 && !p.kongEnum && !slices.Contains(choices(p), fmt.Sprint(x)) {
		return nil, fmt.Errorf("%q is not one of %s", fmt.Sprint(x), strings.Join(choices(p), ", "))
	}
	return x, nil
}

// request is the JSON arguments: --args, then each value given over it.
func request(raw string, obj map[string]any) ([]byte, error) {
	out := map[string]any{}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return nil, &command.ArgError{Reason: "--args is not a JSON object: " + err.Error()}
		}
	}
	maps.Copy(out, obj)
	return json.Marshal(out, json.Deterministic(true))
}

// argError reports a bad argument of e, and returns cli.ExitUsage.
func (a *Adapter) argError(w io.Writer, e *entry, err error) int {
	if write(w, fmt.Sprintf("%s: %s: %v\nRun '%s %s --help' for its arguments.\n", a.cfg.name, e.c.ID, err, a.cfg.name, words(e.c.ID))) != nil {
		return cli.ExitFailed
	}
	return cli.ExitUsage
}

// failed maps a run's error to an exit code
// (docs/decisions/0006-MADR-command-registry.md A9).
func (a *Adapter) failed(w io.Writer, e *entry, err error) int {
	msg := fmt.Sprintf("%s: %v\n", a.cfg.name, err)
	code := cli.ExitFailed
	var argErr *command.ArgError
	switch {
	case errors.As(err, &argErr), errors.Is(err, command.ErrUnknown):
		msg += fmt.Sprintf("Run '%s %s --help' for its arguments.\n", a.cfg.name, words(e.c.ID))
		code = cli.ExitUsage
	case errors.Is(err, command.ErrRefused):
		if e.c.Danger == command.Destructive {
			msg += words(e.c.ID) + " is destructive; pass --yes to run it.\n"
		}
		code = cli.ExitRefused
	}
	if write(w, msg) != nil {
		return cli.ExitFailed
	}
	return code
}

// print writes a result: its value as JSON with --json, null when there is
// none, or else its text.
func (a *Adapter) print(kctx *kong.Context, res command.Result, asJSON bool) int {
	if asJSON {
		return a.printJSON(kctx, res.Value)
	}
	if res.Text == "" {
		return cli.ExitOK
	}
	text := res.Text
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if write(kctx.Stdout, text) != nil {
		return cli.ExitFailed
	}
	return cli.ExitOK
}

func (a *Adapter) printJSON(kctx *kong.Context, v any) int {
	out, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		if write(kctx.Stderr, fmt.Sprintf("%s: the result is not JSON: %v\n", a.cfg.name, err)) != nil {
			return cli.ExitFailed
		}
		return cli.ExitFailed
	}
	if write(kctx.Stdout, string(out)+"\n") != nil {
		return cli.ExitFailed
	}
	return cli.ExitOK
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

// runVerb runs one of the shell's verbs.
func (a *Adapter) runVerb(kctx *kong.Context, verb string) int {
	sel := kctx.Selected()
	var target reflect.Value
	if sel.Target.IsValid() {
		target = reflect.Indirect(sel.Target)
	}
	field := func(name string) reflect.Value {
		if !target.IsValid() {
			return reflect.Value{}
		}
		return target.FieldByName(name)
	}
	switch verb {
	case verbList:
		return a.list(kctx, field("JSON").Bool())
	case verbDescribe, verbSchema, verbHelp:
		var words []string
		if f := field("Command"); f.IsValid() {
			if w, ok := f.Interface().([]string); ok {
				words = w
			}
		}
		switch verb {
		case verbDescribe:
			return a.describe(kctx, words)
		case verbSchema:
			return a.schema(kctx, words)
		}
		return a.helpVerb(kctx, words)
	case verbCompletion:
		return a.script(kctx, field("Shell").String())
	}
	return cli.ExitOK
}

// resolve finds the registry command args name, by its words or its
// dotted ID: the longest run of leading words, joined with dots, that is
// an ID offered on the CLI surface. The rest are its arguments.
func (a *Adapter) resolve(args []string) (command.Command, []string, bool) {
	for n := len(args); n > 0; n-- {
		if c, ok := a.r.Lookup(command.ID(strings.Join(args[:n], "."))); ok && offered(c) {
			return c, args[n:], true
		}
	}
	return command.Command{}, nil, false
}

// usagef reports a usage error of a verb, and returns cli.ExitUsage.
func (a *Adapter) usagef(kctx *kong.Context, format string, args ...any) int {
	if write(kctx.Stderr, fmt.Sprintf("%s: "+format+"\nRun '%s --help' for the commands.\n", append(append([]any{a.cfg.name}, args...), a.cfg.name)...)) != nil {
		return cli.ExitFailed
	}
	return cli.ExitUsage
}

// one finds the single command a verb names.
func (a *Adapter) one(kctx *kong.Context, verb string, args []string) (command.Command, int, bool) {
	if len(args) == 0 {
		return command.Command{}, a.usagef(kctx, "%s needs a command", verb), false
	}
	c, rest, ok := a.resolve(args)
	if !ok || len(rest) > 0 {
		return command.Command{}, a.usagef(kctx, "unknown command %q", strings.Join(args, " ")), false
	}
	return c, cli.ExitOK, true
}

// list writes the commands the shell offers now, or with --json the
// manifest of them.
func (a *Adapter) list(kctx *kong.Context, asJSON bool) int {
	var cmds []command.Command
	for c := range a.r.Available(a.cfg.context, command.SurfaceCLI) {
		cmds = append(cmds, c)
	}
	if !asJSON {
		p := a.page()
		p.list(cmds)
		if write(kctx.Stdout, p.String()) != nil {
			return cli.ExitFailed
		}
		return cli.ExitOK
	}
	m := a.r.Manifest()
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
	return a.printJSON(kctx, m)
}

// describe writes one command's description as JSON.
func (a *Adapter) describe(kctx *kong.Context, args []string) int {
	c, code, ok := a.one(kctx, verbDescribe, args)
	if !ok {
		return code
	}
	for _, mc := range a.r.Manifest().Commands {
		if mc.ID == c.ID {
			return a.printJSON(kctx, mc)
		}
	}
	return a.usagef(kctx, "unknown command %q", c.ID)
}

// schema writes one command's arguments schema.
func (a *Adapter) schema(kctx *kong.Context, args []string) int {
	c, code, ok := a.one(kctx, verbSchema, args)
	if !ok {
		return code
	}
	schema := c.Args
	if len(schema) == 0 {
		schema = command.Schema(`{"type":"object"}`)
	}
	if write(kctx.Stdout, string(schema)+"\n") != nil {
		return cli.ExitFailed
	}
	return cli.ExitOK
}
