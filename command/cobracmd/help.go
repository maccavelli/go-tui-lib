package cobracmd

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/maccavelli/go-tui-lib/command"
)

// minWidth is the narrowest help is wrapped to.
const minWidth = 40

// otherCategory heads the commands without a category.
const otherCategory = "Other"

// The verbs, as Cobra commands.
const (
	verbList     = "list"
	verbDescribe = "describe"
	verbSchema   = "schema"
)

// verbs are list, describe and schema, as command/cli has them.
func (b *builder) verbs() []*cobra.Command {
	verb := map[string]string{annotationNode: nodeVerb}
	var asJSON bool
	list := &cobra.Command{
		Use:         verbList,
		Short:       "List the commands you can run now",
		Long:        "Lists the commands you can run now, by category, or with --json their manifest.",
		Annotations: verb,
		Args: func(cc *cobra.Command, args []string) error {
			if len(args) > 0 {
				return b.usagef(cc, "list takes only --json")
			}
			return nil
		},
		RunE: func(cc *cobra.Command, _ []string) error { return b.list(cc, asJSON) },
	}
	addSwitch(list.Flags(), &asJSON, "json", "print the manifest of the commands as JSON")
	describe := &cobra.Command{
		Use:               verbDescribe + " <command>",
		Short:             "Describe a command as JSON",
		Long:              "Describes one command as JSON: its ID, arguments schema and danger.",
		Annotations:       verb,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: b.completeIDs,
		RunE:              func(cc *cobra.Command, args []string) error { return b.describe(cc, args) },
	}
	schema := &cobra.Command{
		Use:               verbSchema + " <command>",
		Short:             "Print a command's arguments schema",
		Long:              "Prints one command's arguments as a JSON Schema.",
		Annotations:       verb,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: b.completeIDs,
		RunE:              func(cc *cobra.Command, args []string) error { return b.schema(cc, args) },
	}
	return []*cobra.Command{list, describe, schema}
}

// helpCommand is the root's help command: the program's help, or one
// command's, by its words or its dotted ID.
func (b *builder) helpCommand() *cobra.Command {
	return &cobra.Command{
		Use:               helpName + " [command]",
		Short:             "Show the help of the program or of a command",
		Annotations:       map[string]string{annotationNode: nodeVerb},
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: completeSubcommands,
		RunE: func(cc *cobra.Command, args []string) error {
			root := cc.Root()
			if len(args) == 0 {
				root.HelpFunc()(root, nil)
				return nil
			}
			if c, _, ok := b.resolve(args); ok && strings.Contains(args[0], ".") {
				args = slices.Concat(strings.Split(string(c.ID), "."), args[1:])
			}
			target, rest, err := root.Find(args)
			if err != nil || len(rest) > 0 || target == root {
				return b.usagef(cc, "unknown command %q", strings.Join(args, " "))
			}
			target.HelpFunc()(target, nil)
			return nil
		},
	}
}

// list writes the commands the shell offers now, or with --json the
// manifest of them.
func (b *builder) list(cc *cobra.Command, asJSON bool) error {
	var cmds []command.Command
	for c := range b.r.Available(b.cfg.context, command.SurfaceCLI) {
		cmds = append(cmds, c)
	}
	if !asJSON {
		p := b.page()
		p.list(cmds)
		emit(cc.OutOrStdout(), p.String())
		return nil
	}
	m := b.r.Manifest()
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
	return b.printJSON(cc, m)
}

// one finds the single command a verb names, by its words or ID.
func (b *builder) one(cc *cobra.Command, verb string, args []string) (command.Command, error) {
	if len(args) == 0 {
		return command.Command{}, b.usagef(cc, "%s needs a command", verb)
	}
	c, rest, ok := b.resolve(args)
	if !ok || len(rest) > 0 {
		return command.Command{}, b.usagef(cc, "unknown command %q", strings.Join(args, " "))
	}
	return c, nil
}

// describe writes one command's description as JSON.
func (b *builder) describe(cc *cobra.Command, args []string) error {
	c, err := b.one(cc, verbDescribe, args)
	if err != nil {
		return err
	}
	for _, mc := range b.r.Manifest().Commands {
		if mc.ID == c.ID {
			return b.printJSON(cc, mc)
		}
	}
	return b.usagef(cc, "unknown command %q", c.ID)
}

// schema writes one command's argument schema.
func (b *builder) schema(cc *cobra.Command, args []string) error {
	c, err := b.one(cc, verbSchema, args)
	if err != nil {
		return err
	}
	schema := c.Args
	if len(schema) == 0 {
		schema = command.Schema(`{"type":"object"}`)
	}
	emit(cc.OutOrStdout(), string(schema)+"\n")
	return nil
}

// help is the help function of every command New and Mount build.
func (b *builder) help(cc *cobra.Command, _ []string) {
	emit(cc.OutOrStdout(), b.helpText(cc))
}

// helpText is cc's help: the program's for New's root, a namespace's, a
// registry command's, or that of any other command, such as Cobra's
// completion.
func (b *builder) helpText(cc *cobra.Command) string {
	p := b.page()
	switch e := b.entries[cc]; {
	case e != nil:
		p.command(cc, e)
	case cc.Annotations[annotationNode] == nodeRoot:
		p.root(cc, b.order)
	case cc.Annotations[annotationNode] == nodeNamespace:
		p.namespace(cc, b.below(cc))
	default:
		p.other(cc)
	}
	return p.String()
}

// below is the registry commands in the tree under cc, a namespace or a
// registry command, that are not hidden.
func (b *builder) below(cc *cobra.Command) []*entry {
	prefix := b.prefixes[cc] + "."
	var out []*entry
	for _, e := range b.order {
		if strings.HasPrefix(string(e.c.ID), prefix) && !e.c.Hidden {
			out = append(out, e)
		}
	}
	return out
}

// page is one help text, drawn with the builder's theme at its width.
type page struct {
	strings.Builder
	b *builder
}

func (b *builder) page() *page { return &page{b: b} }

func (p *page) width() int { return max(p.b.cfg.width, minWidth) }

func (p *page) printf(format string, a ...any) { fmt.Fprintf(&p.Builder, format, a...) }

func (p *page) heading(s string) { p.WriteString(p.b.cfg.theme.Styles.Title.Render(s) + "\n") }

// root writes the program's help: usage, the commands the shell offers by
// category, and the flags every command takes. Hidden commands are left
// out.
func (p *page) root(cc *cobra.Command, entries []*entry) {
	n := cc.Name()
	p.heading("Usage:")
	p.printf("  %s <command> [arguments] [flags]\n", n)
	for _, verb := range []string{"list [--json]", "describe <command>", "schema <command>", "help [<command>]", "completion bash|zsh|fish|powershell"} {
		p.printf("  %s %s\n", n, verb)
	}
	var cmds []command.Command
	for _, e := range entries {
		if !e.c.Hidden {
			cmds = append(cmds, e.c)
		}
	}
	p.list(cmds)
	p.WriteString("\n")
	p.heading("Flags for every command:")
	p.columns([][2]string{
		{"--args JSON", "the arguments as one JSON object"},
		{"--json", "print the result's value as JSON"},
		{"--yes", "run a destructive command without asking"},
		{"-h, --help", "show the command's help"},
	})
}

// namespace writes a namespace's help: its usage and the commands below
// it.
func (p *page) namespace(cc *cobra.Command, entries []*entry) {
	p.heading("Usage:")
	p.printf("  %s <command> [arguments] [flags]\n", cc.CommandPath())
	cmds := make([]command.Command, len(entries))
	for i, e := range entries {
		cmds[i] = e.c
	}
	p.list(cmds)
}

// list writes cmds by category, categories in order and Other last, each
// as its words and its description or title.
func (p *page) list(cmds []command.Command) {
	byCat := map[string][][2]string{}
	for _, c := range cmds {
		cat := c.Category
		if cat == "" {
			cat = otherCategory
		}
		desc := c.Description
		if desc == "" {
			desc = c.Title
		}
		byCat[cat] = append(byCat[cat], [2]string{words(c.ID), desc})
	}
	cats := make([]string, 0, len(byCat))
	for c := range byCat {
		if c != otherCategory {
			cats = append(cats, c)
		}
	}
	slices.Sort(cats)
	if _, ok := byCat[otherCategory]; ok {
		cats = append(cats, otherCategory)
	}
	left := 0
	for _, rows := range byCat {
		left = max(left, leftWidth(rows))
	}
	for _, cat := range cats {
		p.WriteString("\n")
		p.heading(cat + " commands:")
		p.columnsAt(byCat[cat], left)
	}
}

// command writes a registry command's help: usage, title and description,
// its positional arguments, its flags by group, its danger, and the
// commands below it. Hidden arguments are left out.
func (p *page) command(cc *cobra.Command, e *entry) {
	c := e.c
	var args, flags [][2]string
	groups := map[string][][2]string{}
	var groupOrder []string
	for _, prop := range e.props {
		if prop.cli.Hidden {
			continue
		}
		if prop.cli.Arg {
			args = append(args, [2]string{placeholder(prop), prop.desc})
		}
		row := [2]string{flagName(prop), describe(prop)}
		if g := prop.cli.Group; g != "" {
			if _, seen := groups[g]; !seen {
				groupOrder = append(groupOrder, g)
			}
			groups[g] = append(groups[g], row)
			continue
		}
		flags = append(flags, row)
	}
	line := strings.Join(append([]string{cc.CommandPath()}, usageArgs(e.props, p.b.cfg.theme.Glyphs.Ellipsis)...), " ")
	p.heading("Usage:")
	p.printf("  %s [flags]\n\n", line)
	p.printf("%s\n", p.wrap(c.Title, 0))
	if c.Description != "" && c.Description != c.Title {
		p.printf("%s\n", p.wrap(c.Description, 0))
	}
	if len(cc.Aliases) > 0 {
		p.printf("\nAliases: %s\n", strings.Join(cc.Aliases, ", "))
	}
	if len(args) > 0 {
		p.WriteString("\n")
		p.heading("Arguments:")
		p.columns(args)
	}
	if len(flags) > 0 {
		p.WriteString("\n")
		p.heading("Flags:")
		p.columns(flags)
	}
	for _, g := range groupOrder {
		p.WriteString("\n")
		p.heading(g + " flags:")
		p.columns(groups[g])
	}
	p.WriteString("\n")
	danger := "Danger: " + c.Danger.String()
	if c.Danger == command.Destructive {
		danger = p.b.cfg.theme.Styles.Warning.Render(danger + "; it asks first, or needs --yes")
	}
	p.WriteString(danger + "\n")
	if below := p.b.below(cc); len(below) > 0 {
		cmds := make([]command.Command, len(below))
		for i, e := range below {
			cmds[i] = e.c
		}
		p.list(cmds)
	}
}

// other writes the help of a command this package did not build from the
// registry, such as a verb or Cobra's completion: its usage, its
// description, its subcommands and its flags.
func (p *page) other(cc *cobra.Command) {
	p.heading("Usage:")
	p.printf("  %s\n", cc.UseLine())
	if cc.HasAvailableSubCommands() {
		p.printf("  %s <command>\n", cc.CommandPath())
	}
	desc := cc.Long
	if desc == "" {
		desc = cc.Short
	}
	if desc != "" {
		p.printf("\n%s\n", p.wrap(desc, 0))
	}
	var subs [][2]string
	for _, s := range cc.Commands() {
		if s.IsAvailableCommand() {
			subs = append(subs, [2]string{s.Name(), s.Short})
		}
	}
	if len(subs) > 0 {
		p.WriteString("\n")
		p.heading("Commands:")
		p.columns(subs)
	}
	var flags [][2]string
	cc.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		name := "--" + f.Name
		if f.Shorthand != "" {
			name = "-" + f.Shorthand + ", " + name
		}
		if t := f.Value.Type(); t != typeBool {
			name += " " + t
		}
		flags = append(flags, [2]string{name, f.Usage})
	})
	if len(flags) > 0 {
		p.WriteString("\n")
		p.heading("Flags:")
		p.columns(flags)
	}
}

// usageArgs is the positional arguments of a usage line: each one's value
// name, the ellipsis after one that repeats, and brackets round one that
// is optional.
func usageArgs(props []*property, ellipsis string) []string {
	var out []string
	for _, p := range props {
		if p.cli.Hidden || !p.cli.Arg {
			continue
		}
		name := placeholder(p)
		if p.is(tArray) {
			name += ellipsis
		}
		if !p.required {
			name = "[" + name + "]"
		}
		out = append(out, name)
	}
	return out
}

// leftWidth is the widest left column of rows.
func leftWidth(rows [][2]string) int {
	left := 0
	for _, r := range rows {
		left = max(left, ansi.StringWidth(r[0]))
	}
	return left
}

// placeholder is an argument's value name: the schema's, an enum's
// choices, or one by its type.
func placeholder(p *property) string {
	switch {
	case p.cli.Placeholder != "":
		return p.cli.Placeholder
	case len(p.enum) > 0:
		return strings.Join(choices(p), "|")
	}
	switch p.typeOf() {
	case tInteger, tNumber:
		return "N"
	case tBoolean:
		return "BOOL"
	case tObject:
		return valueJSON
	case tArray:
		if itemType(p) == tObject {
			return valueJSON
		}
		return "VALUE"
	}
	return "TEXT"
}

// flagName is a property's flag as help shows it.
func flagName(p *property) string {
	f := "--" + p.name
	if s := p.cli.Short; len(s) == 1 && s != "h" && s[0] < 0x80 {
		f = "-" + s + ", " + f
	}
	if !p.is(tBoolean) {
		f += " " + placeholder(p)
	}
	return f
}

// describe is a flag's description and its notes.
func describe(p *property) string {
	var notes []string
	if p.required {
		notes = append(notes, "required")
	}
	if p.hasDefault {
		notes = append(notes, "default "+fmt.Sprint(p.def))
	}
	if p.min != nil || p.max != nil {
		notes = append(notes, bounds(p))
	}
	if p.is(tArray) {
		notes = append(notes, "repeat for more")
	}
	if p.is(tObject) {
		notes = append(notes, "JSON")
	}
	d := p.desc
	if len(notes) > 0 {
		d = strings.TrimSpace(d + " (" + strings.Join(notes, "; ") + ")")
	}
	return d
}

func bounds(p *property) string {
	num := func(f *float64) string { return strconv.FormatFloat(*f, 'f', -1, 64) }
	switch {
	case p.min != nil && p.max != nil:
		return num(p.min) + " to " + num(p.max)
	case p.min != nil:
		return "at least " + num(p.min)
	}
	return "at most " + num(p.max)
}

// columns writes rows as two columns indented two cells, the left one in
// the accent style, the right one wrapped at the width with its later
// lines under its first.
func (p *page) columns(rows [][2]string) { p.columnsAt(rows, leftWidth(rows)) }

// columnsAt is columns with the left column left cells wide, at most a
// third of the width.
func (p *page) columnsAt(rows [][2]string, left int) {
	left = min(left, p.width()/3)
	accent := p.b.cfg.theme.Styles.Accent
	for _, r := range rows {
		name := r[0]
		pad := left - ansi.StringWidth(name)
		indent := 2 + left + 2
		if pad < 0 {
			p.printf("  %s\n", accent.Render(name))
			name, pad = "", left
		}
		if name != "" {
			name = accent.Render(name)
		}
		text := p.wrap(r[1], indent)
		p.printf("  %s%s  %s\n", name, strings.Repeat(" ", pad), text)
	}
}

// wrap wraps text to the width less indent, and indents the lines after
// the first by indent.
func (p *page) wrap(text string, indent int) string {
	lines := strings.Split(ansi.Wordwrap(text, max(p.width()-indent, 10), ""), "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.Repeat(" ", indent) + lines[i]
	}
	return strings.Join(lines, "\n")
}
