// Package cobracmd runs a command.Registry's commands as a Cobra command
// tree, with flags from their schemas, completion, and help drawn with the
// library's theme and glyphs, so a Cobra program offers the same commands
// in its TUI and from the shell
// (docs/decisions/0006-MADR-command-registry.md A1, A10).
//
// New builds a root command holding every command the registry offers on
// command.SurfaceCLI; Mount grafts them into a program's own tree. A
// dotted ID becomes nested commands, "prog workspace state get", and Run
// takes the dotted form too, "prog workspace.state.get". Each argument is
// a flag, --delta 4, which an array repeats and an object takes as JSON;
// positional arguments come in order; --args takes the whole object as
// JSON; and --json prints the result's value as JSON. The verbs list,
// describe and schema describe the commands, as in command/cli, and help
// and completion are Cobra's, drawn by this package. A hidden command runs
// but is neither listed nor completed.
//
// A shell invocation is command.OriginCLI. A destructive command asks
// WithConfirm, or without it needs --yes. Run returns command/cli's exit
// codes, and never exits the process.
//
// Each command this package builds carries the annotations "command.id",
// "command.danger" and "command.surfaces", and "cobracmd.node", which
// names what it is: "root", "namespace", "command" or "verb".
//
// Three things Cobra does are the program's to know:
//
//   - Build the tree once per process. Cobra keeps flag completions in a
//     process-wide map with no delete, so every tree built adds entries
//     that are never freed.
//   - On Windows, Cobra exits a program started from Explorer unless the
//     program sets cobra.MousetrapHelpText to "" itself; this package sets
//     no Cobra variable.
//   - A failing shell completion request writes to os.Stderr inside
//     Cobra, which no setter reaches.
//   - Cobra's own completion command keeps the writer of the first run
//     that creates it, so a later run of the same tree writes its script
//     there. New's root has a completion command of its own that does
//     not; a program that mounts into its own root and runs it more than
//     once disables Cobra's, with CompletionOptions.DisableDefaultCmd.
package cobracmd

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/command/cli"
	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/theme"
	"github.com/maccavelli/go-tui-lib/when"
)

// Option configures New and Mount.
type Option func(*config)

type config struct {
	name    string
	confirm func(prompt string) bool
	context when.Context
	theme   theme.Theme
	glyphs  *glyph.Set
	width   int
}

// WithName is the root command's name, in usage and help (default "app").
func WithName(name string) Option { return func(c *config) { c.name = name } }

// WithConfirm asks the user before a destructive command runs without
// --yes; it returns true to run it.
func WithConfirm(f func(prompt string) bool) Option { return func(c *config) { c.confirm = f } }

// WithContext is the context commands' When is evaluated in (default
// empty).
func WithContext(c when.Context) Option { return func(cfg *config) { cfg.context = c } }

// WithTheme is the theme help is drawn with (default: no colour, ASCII
// glyphs).
func WithTheme(t theme.Theme) Option { return func(c *config) { c.theme = t } }

// WithGlyphs replaces the theme's glyphs.
func WithGlyphs(g glyph.Set) Option { return func(c *config) { c.glyphs = &g } }

// WithWidth is the width help wraps at, in cells (default 80).
func WithWidth(w int) Option { return func(c *config) { c.width = w } }

func configure(o []Option) config {
	cfg := config{name: "app", width: 80, theme: theme.New(colorprofile.NoTTY, theme.Unknown, glyph.ASCII())}
	for _, f := range o {
		f(&cfg)
	}
	if cfg.context == nil {
		cfg.context = when.Map{}
	}
	if cfg.glyphs != nil {
		cfg.theme.Glyphs = *cfg.glyphs
	}
	return cfg
}

// The annotations each command this package builds carries.
const (
	annotationID       = "command.id"
	annotationDanger   = "command.danger"
	annotationSurfaces = "command.surfaces"
	annotationNode     = "cobracmd.node"
)

// The kinds of node, the values of annotationNode.
const (
	nodeRoot      = "root"
	nodeNamespace = "namespace"
	nodeCommand   = "command"
	nodeVerb      = "verb"
)

// The names Cobra gives the commands it adds to a root.
const (
	helpName       = "help"
	completionName = "completion"
)

// New builds a root command holding every command r offers on the CLI
// surface, with the verbs list, describe, schema and help.
func New(r *command.Registry, o ...Option) (*cobra.Command, error) {
	if r == nil {
		return nil, errors.New("cobracmd: New needs a registry")
	}
	b := newBuilder(r, o)
	root := &cobra.Command{
		Use:           b.cfg.name,
		Short:         "Run " + b.cfg.name + "'s commands",
		Args:          cobra.ArbitraryArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		Annotations:   map[string]string{annotationNode: nodeRoot},
		RunE:          b.namespaceRun,
	}
	root.SetHelpFunc(b.help)
	root.SetFlagErrorFunc(b.flagError)
	root.SetHelpCommand(b.helpCommand())
	root.CompletionOptions.DisableDefaultCmd = true
	if err := b.mount(root); err != nil {
		return nil, err
	}
	root.AddCommand(b.completionCommand())
	return root, nil
}

// Mount grafts every command r offers on the CLI surface, and the verbs
// list, describe and schema, into parent, beside the program's own
// commands. It changes nothing and returns an error when parent already
// has a command by a name or alias one of them would take.
func Mount(parent *cobra.Command, r *command.Registry, o ...Option) error {
	if parent == nil || r == nil {
		return errors.New("cobracmd: Mount needs a parent and a registry")
	}
	return newBuilder(r, o).mount(parent)
}

// Run executes root against args, a program's arguments without its name,
// with the caller's streams, and returns command/cli's exit code. A dotted
// ID is taken as its words. It never calls os.Exit, and before it runs it
// clears the flags of the commands New and Mount built, which Cobra keeps
// between runs. An error from a command the program built itself is
// cli.ExitFailed.
func Run(ctx context.Context, root *cobra.Command, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	out, errOut := &recorder{w: stdout}, &recorder{w: stderr}
	if args == nil {
		args = []string{} // Cobra reads os.Args for nil
	}
	clearFlags(root)
	root.SetArgs(dotted(root, args))
	root.SetIn(stdin)
	root.SetOut(out)
	root.SetErr(errOut)
	root.SilenceErrors = true
	root.SilenceUsage = true
	_, err := root.ExecuteContextC(ctx)
	code := cli.ExitOK
	if err != nil {
		if ee, ok := errors.AsType[*exitError](err); ok {
			code = ee.code
		} else {
			emit(errOut, root.Name()+": "+err.Error()+"\n")
			code = cli.ExitFailed
		}
	}
	if out.err != nil || errOut.err != nil {
		code = cli.ExitFailed
	}
	return code
}

// exitError ends a run with an exit code; its message is already written.
type exitError struct{ code int }

func (e *exitError) Error() string { return "exit status " + strconv.Itoa(e.code) }

// recorder passes writes through to w, and keeps the first error, because
// Cobra's help and completion functions return none.
type recorder struct {
	w   io.Writer
	err error
}

func (r *recorder) Write(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	n, err := r.w.Write(p)
	if err != nil {
		r.err = err
	}
	return n, err
}

// emit writes s to w. Run's writers record a failure, which Run returns as
// cli.ExitFailed; a help function has no error to return it through.
func emit(w io.Writer, s string) {
	if _, err := io.WriteString(w, s); err != nil {
		return
	}
}

// walk calls f on c and every command below it.
func walk(c *cobra.Command, f func(*cobra.Command)) {
	f(c)
	for _, sub := range c.Commands() {
		walk(sub, f)
	}
}

// clearFlags clears the flags of every command New or Mount built under
// root, so that a run sees none of the last run's.
func clearFlags(root *cobra.Command) {
	walk(root, func(c *cobra.Command) {
		if c.Annotations[annotationNode] == "" {
			return
		}
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if v, ok := f.Value.(resetter); ok {
				v.reset()
			} else if f.Changed {
				if err := f.Value.Set(f.DefValue); err != nil {
					return
				}
			}
			f.Changed = false
		})
	})
}

// dotted replaces the first word of args that is a registry command's
// dotted ID with the ID's words, when the words before it lead to where
// that command is mounted.
func dotted(root *cobra.Command, args []string) []string {
	mounts := map[string][]string{}
	walk(root, func(c *cobra.Command) {
		id := c.Annotations[annotationID]
		if !strings.Contains(id, ".") {
			return
		}
		path := strings.Fields(c.CommandPath())[1:]
		mounts[id] = path[:len(path)-strings.Count(id, ".")-1]
	})
	for i, a := range args {
		if strings.HasPrefix(a, "-") {
			break
		}
		if before, ok := mounts[a]; ok && slices.Equal(args[:i], before) {
			return slices.Concat(args[:i], strings.Split(a, "."), args[i+1:])
		}
	}
	return args
}

// builder builds one tree for one registry.
type builder struct {
	cfg      config
	r        *command.Registry
	entries  map[*cobra.Command]*entry
	order    []*entry                  // in ID order
	prefixes map[*cobra.Command]string // each node's ID, or its namespace's
}

// entry is a registry command in the tree.
type entry struct {
	c     command.Command
	props []*property
}

func newBuilder(r *command.Registry, o []Option) *builder {
	return &builder{cfg: configure(o), r: r, entries: map[*cobra.Command]*entry{}, prefixes: map[*cobra.Command]string{}}
}

// node is one word of the tree: a registry command, a namespace of
// commands, or both.
type node struct {
	word     string
	cmd      *command.Command
	children []*node
}

func (n *node) child(word string) *node {
	for _, c := range n.children {
		if c.word == word {
			return c
		}
	}
	c := &node{word: word}
	n.children = append(n.children, c)
	return c
}

// offered reports whether c is one the shell lists and runs.
func offered(c command.Command) bool {
	return c.Surfaces == 0 || c.Surfaces&command.SurfaceCLI != 0
}

// tree is the registry's CLI commands as a tree of words, in ID order.
func (b *builder) tree() *node {
	top := &node{}
	for c := range b.r.All() {
		if !offered(c) {
			continue
		}
		n := top
		for seg := range c.ID.Segments() {
			n = n.child(seg)
		}
		n.cmd = &c
	}
	return top
}

// mount adds the registry's commands and the verbs to parent, after
// checking that none of their names is taken there.
func (b *builder) mount(parent *cobra.Command) error {
	top := b.tree()
	var kids []*cobra.Command
	kids = append(kids, b.verbs()...)
	for _, n := range top.children {
		c, err := b.build(n, n.word)
		if err != nil {
			return err
		}
		kids = append(kids, c)
	}
	taken := map[string]bool{}
	if !parent.HasParent() {
		taken[helpName], taken[completionName] = true, true
	}
	for _, c := range parent.Commands() {
		taken[c.Name()] = true
		for _, a := range c.Aliases {
			taken[a] = true
		}
	}
	var clashes []string
	for _, c := range kids {
		if taken[c.Name()] {
			clashes = append(clashes, strconv.Quote(c.Name()))
		}
		taken[c.Name()] = true
	}
	if len(clashes) > 0 {
		return fmt.Errorf("cobracmd: %s already has a command named %s", parent.CommandPath(), strings.Join(clashes, ", "))
	}
	sortAliases(kids, taken)
	addGroups(parent, kids)
	if parent.Annotations[annotationNode] == "" {
		for _, c := range kids {
			c.SetHelpFunc(b.help)
			c.SetFlagErrorFunc(b.flagError)
		}
	}
	parent.AddCommand(kids...)
	return nil
}

// sortAliases keeps the aliases of kids that name nothing else among them,
// nor anything in taken.
func sortAliases(kids []*cobra.Command, taken map[string]bool) {
	for _, c := range kids {
		var keep []string
		for _, a := range c.Aliases {
			if a == "" || a == c.Name() || taken[a] || strings.ContainsAny(a, " \t.") {
				continue
			}
			taken[a] = true
			keep = append(keep, a)
		}
		c.Aliases = keep
	}
}

// addGroups adds to parent the group of each of kids that it does not
// have yet, before the kids themselves.
func addGroups(parent *cobra.Command, kids []*cobra.Command) {
	for _, c := range kids {
		if id := c.GroupID; id != "" && !parent.ContainsGroup(id) {
			parent.AddGroup(&cobra.Group{ID: id, Title: id + " commands:"})
		}
	}
}

// build makes the Cobra command for n, whose ID or namespace is prefix,
// and the commands below it.
func (b *builder) build(n *node, prefix string) (*cobra.Command, error) {
	var c *cobra.Command
	if n.cmd == nil {
		c = &cobra.Command{
			Use:         n.word,
			Short:       "The " + n.word + " commands",
			Args:        cobra.ArbitraryArgs,
			RunE:        b.namespaceRun,
			Annotations: map[string]string{annotationNode: nodeNamespace},
		}
	} else {
		var err error
		if c, err = b.command(n.word, *n.cmd); err != nil {
			return nil, err
		}
	}
	var kids []*cobra.Command
	for _, k := range n.children {
		kc, err := b.build(k, prefix+"."+k.word)
		if err != nil {
			return nil, err
		}
		kids = append(kids, kc)
	}
	taken := map[string]bool{}
	for _, k := range kids {
		taken[k.Name()] = true
	}
	sortAliases(kids, taken)
	addGroups(c, kids)
	c.AddCommand(kids...)
	b.prefixes[c] = prefix
	if n.cmd == nil {
		c.Hidden = !slices.ContainsFunc(kids, func(k *cobra.Command) bool { return !k.Hidden })
		c.GroupID = sharedGroup(kids)
	}
	return c, nil
}

// sharedGroup is the group every visible one of kids is in, or none.
func sharedGroup(kids []*cobra.Command) string {
	group, seen := "", false
	for _, k := range kids {
		if k.Hidden {
			continue
		}
		if seen && k.GroupID != group {
			return ""
		}
		group, seen = k.GroupID, true
	}
	return group
}

// command makes the Cobra command for the registry command c.
func (b *builder) command(word string, c command.Command) (*cobra.Command, error) {
	e := &entry{c: c}
	st := &state{obj: map[string]any{}}
	props, broken := properties(c.Args)
	if broken != nil {
		broken = fmt.Errorf("its schema cannot be read: %w", broken)
	}
	e.props = props
	cc := &cobra.Command{
		Use:     strings.Join(append([]string{word}, usageArgs(props, "...")...), " "),
		Short:   c.Title,
		Aliases: slices.Clone(c.Aliases),
		Hidden:  c.Hidden,
		GroupID: c.Category,
		Annotations: map[string]string{
			annotationNode:     nodeCommand,
			annotationID:       string(c.ID),
			annotationDanger:   c.Danger.String(),
			annotationSurfaces: surfaces(c),
		},
	}
	if c.Description != c.Title {
		cc.Long = c.Description
	}
	if broken == nil {
		broken = addFlags(cc, props, st)
	}
	pos := positionals(props)
	// A broken command's arguments are not checked: RunE reports why it
	// cannot run.
	cc.Args = func(cc *cobra.Command, args []string) error {
		if broken == nil {
			if _, err := placeArgs(pos, st.obj, args); err != nil {
				return b.argError(cc, c, err)
			}
		}
		return nil
	}
	cc.RunE = func(cc *cobra.Command, args []string) error {
		if broken != nil {
			return b.argError(cc, c, broken)
		}
		return b.run(cc, c, st, pos, args)
	}
	cc.SetFlagErrorFunc(func(cc *cobra.Command, err error) error {
		if st.last != nil {
			return b.argError(cc, c, st.last)
		}
		return b.argError(cc, c, err)
	})
	cc.ValidArgsFunction = completeArgs(pos, st)
	for _, p := range props {
		if len(p.enum) > 0 && cc.Flags().Lookup(p.name) != nil {
			if err := cc.RegisterFlagCompletionFunc(p.name, cobra.FixedCompletions(choices(p), cobra.ShellCompDirectiveNoFileComp)); err != nil {
				return nil, fmt.Errorf("cobracmd: %s: %w", c.ID, err)
			}
		}
	}
	b.entries[cc] = e
	b.order = append(b.order, e)
	return cc, nil
}

// surfaces is c's surfaces for its annotation.
func surfaces(c command.Command) string {
	if c.Surfaces == 0 {
		return command.AllSurfaces.String()
	}
	return c.Surfaces.String()
}

// run runs a registry command from its parsed command line.
func (b *builder) run(cc *cobra.Command, c command.Command, st *state, pos []*property, args []string) error {
	obj, err := placeArgs(pos, st.obj, args)
	if err != nil {
		return b.argError(cc, c, err)
	}
	raw, err := request(st.raw, obj)
	if err != nil {
		return b.argError(cc, c, err)
	}
	res, err := b.r.Run(cc.Context(), command.Request{
		ID: c.ID, Args: raw, Origin: command.OriginCLI, Caller: b.cfg.name, Context: b.cfg.context,
		Gate: confirmGate{yes: st.yes, confirm: b.cfg.confirm},
	})
	if err != nil {
		return b.failed(cc, c, err)
	}
	return b.print(cc, res, st.asJSON)
}

// usagef reports a usage error, and ends the run with cli.ExitUsage.
func (b *builder) usagef(cc *cobra.Command, format string, a ...any) error {
	name := cc.Root().Name()
	emit(cc.ErrOrStderr(), fmt.Sprintf("%s: "+format+"\n", append([]any{name}, a...)...)+
		fmt.Sprintf("Run '%s help' for the commands.\n", name))
	return &exitError{cli.ExitUsage}
}

// argError reports a bad argument of c, and ends the run with
// cli.ExitUsage.
func (b *builder) argError(cc *cobra.Command, c command.Command, err error) error {
	emit(cc.ErrOrStderr(), fmt.Sprintf("%s: %s: %v\nRun '%s --help' for its arguments.\n", cc.Root().Name(), c.ID, err, cc.CommandPath()))
	return &exitError{cli.ExitUsage}
}

// flagError is the flag error of a command that is not a registry
// command's.
func (b *builder) flagError(cc *cobra.Command, err error) error {
	return b.usagef(cc, "%v", err)
}

// failed maps a run's error to an exit code
// (docs/decisions/0006-MADR-command-registry.md A9).
func (b *builder) failed(cc *cobra.Command, c command.Command, err error) error {
	msg := fmt.Sprintf("%s: %v\n", cc.Root().Name(), err)
	var argErr *command.ArgError
	code := cli.ExitFailed
	switch {
	case errors.As(err, &argErr), errors.Is(err, command.ErrUnknown):
		msg += fmt.Sprintf("Run '%s --help' for its arguments.\n", cc.CommandPath())
		code = cli.ExitUsage
	case errors.Is(err, command.ErrRefused):
		if c.Danger == command.Destructive {
			msg += words(c.ID) + " is destructive; pass --yes to run it.\n"
		}
		code = cli.ExitRefused
	}
	emit(cc.ErrOrStderr(), msg)
	return &exitError{code}
}

// print writes a result: its value as JSON with --json, null when there is
// none, or else its text.
func (b *builder) print(cc *cobra.Command, res command.Result, asJSON bool) error {
	if asJSON {
		return b.printJSON(cc, res.Value)
	}
	if res.Text != "" {
		text := res.Text
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		if _, err := io.WriteString(cc.OutOrStdout(), text); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) printJSON(cc *cobra.Command, v any) error {
	out, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		emit(cc.ErrOrStderr(), fmt.Sprintf("%s: the result is not JSON: %v\n", cc.Root().Name(), err))
		return &exitError{cli.ExitFailed}
	}
	_, err = io.WriteString(cc.OutOrStdout(), string(out)+"\n")
	return err
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

// pathWords is cc's words below the root.
func pathWords(cc *cobra.Command) []string { return strings.Fields(cc.CommandPath())[1:] }

// namespaceRun is the root's and a namespace's run: with no words, its
// help on stderr and a usage error; with words, an unknown command.
func (b *builder) namespaceRun(cc *cobra.Command, args []string) error {
	if len(args) == 0 {
		emit(cc.ErrOrStderr(), b.helpText(cc))
		return &exitError{cli.ExitUsage}
	}
	return b.usagef(cc, "unknown command %q", strings.Join(append(pathWords(cc), args...), " "))
}

// resolve finds the registry command args name, in the tree, by its words
// or its dotted ID: the longest run of leading words, joined with dots,
// that is an ID offered on the CLI surface. The rest are its arguments.
func (b *builder) resolve(args []string) (command.Command, []string, bool) {
	words := 0
	for words < len(args) && !strings.HasPrefix(args[words], "-") {
		words++
	}
	for n := words; n > 0; n-- {
		if c, ok := b.r.Lookup(command.ID(strings.Join(args[:n], "."))); ok && offered(c) {
			return c, args[n:], true
		}
	}
	return command.Command{}, nil, false
}
