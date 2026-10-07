package kongcmd

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/command/cli"
)

// minWidth is the narrowest help is wrapped to.
const minWidth = 40

// otherCategory heads the commands without a category.
const otherCategory = "Other"

// help is the parser's kong.Help: every node's help, drawn with the
// adapter's theme at its width, never Kong's own printer, which reads the
// width from $COLUMNS or the terminal (A1).
func (a *Adapter) help(_ kong.HelpOptions, kctx *kong.Context) error {
	return write(kctx.Stdout, a.helpText(kctx.Model, kctx.Selected()))
}

// helpText is n's help: a registry command's, a namespace's, the program's
// for Parser's root, or any other node's from Kong's model.
func (a *Adapter) helpText(app *kong.Application, n *kong.Node) string {
	p := a.page()
	if n == nil {
		n = app.Node
	}
	switch {
	case n.Tag != nil && n.Tag.Get(tagID) != "":
		p.command(n, a.entries[command.ID(n.Tag.Get(tagID))])
	case n.Tag != nil && n.Tag.Get(tagNS) != "":
		p.namespace(n, a.below(n.Tag.Get(tagNS)))
	case n.Type == kong.ApplicationNode && reflect.Indirect(app.Target).Type() == reflect.TypeFor[parserRoot]():
		p.root()
	default:
		p.other(n)
	}
	return p.String()
}

// helpVerb writes the program's help, or one command's by its words or
// its dotted ID.
func (a *Adapter) helpVerb(kctx *kong.Context, args []string) int {
	n := kctx.Model.Node
	if len(args) > 0 {
		if c, rest, ok := a.resolve(args); ok && len(rest) == 0 {
			args = strings.Split(string(c.ID), ".")
		}
		n = find(kctx.Model.Node, args)
		if n == nil {
			return a.usagef(kctx, "unknown command %q", strings.Join(args, " "))
		}
		if self := selfChild(n); self != nil {
			n = self
		}
	}
	if write(kctx.Stdout, a.helpText(kctx.Model, n)) != nil {
		return cli.ExitFailed
	}
	return cli.ExitOK
}

// find is the command node words name below n, by name or alias.
func find(n *kong.Node, words []string) *kong.Node {
	for _, w := range words {
		var next *kong.Node
		for _, c := range n.Children {
			if c.Name == w || slices.Contains(c.Aliases, w) {
				next = c
				break
			}
		}
		if next == nil {
			return nil
		}
		n = next
	}
	return n
}

// selfChild is n's own command, when n is also a parent.
func selfChild(n *kong.Node) *kong.Node {
	for _, c := range n.Children {
		if c.Tag != nil && c.Tag.Get(tagSelf) != "" {
			return c
		}
	}
	return nil
}

// below is the registry commands under the ID prefix, not hidden.
func (a *Adapter) below(prefix string) []command.Command {
	var out []command.Command
	for _, e := range a.order {
		if strings.HasPrefix(string(e.c.ID), prefix+".") && !e.c.Hidden {
			out = append(out, e.c)
		}
	}
	return out
}

// page is one help text, drawn with the adapter's theme at its width.
type page struct {
	strings.Builder
	a *Adapter
}

func (a *Adapter) page() *page { return &page{a: a} }

func (p *page) width() int { return max(p.a.cfg.width, minWidth) }

func (p *page) printf(format string, args ...any) { fmt.Fprintf(&p.Builder, format, args...) }

func (p *page) heading(s string) { p.WriteString(p.a.cfg.theme.Styles.Title.Render(s) + "\n") }

// path is n's words below the application, as a command line names it: a
// command that is also a parent is named by its parent's words.
func path(n *kong.Node) string {
	var out []string
	for ; n != nil && n.Type != kong.ApplicationNode; n = n.Parent {
		if n.Tag != nil && n.Tag.Get(tagSelf) != "" {
			continue
		}
		out = append(out, n.Name)
	}
	slices.Reverse(out)
	return strings.Join(out, " ")
}

// usage is the program's name and n's words.
func (p *page) usage(n *kong.Node) string {
	return strings.TrimSpace(p.a.cfg.name + " " + path(n))
}

// root writes the help of Parser's root: usage, the commands the shell
// offers by category, and the flags every command takes. Hidden commands
// are left out.
func (p *page) root() {
	n := p.a.cfg.name
	p.heading("Usage:")
	p.printf("  %s <command> [arguments] [flags]\n", n)
	for _, verb := range []string{"list [--json]", "describe <command>", "schema <command>", "help [<command>]", "completion bash|zsh|fish|powershell"} {
		p.printf("  %s %s\n", n, verb)
	}
	var cmds []command.Command
	for _, e := range p.a.order {
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
func (p *page) namespace(n *kong.Node, cmds []command.Command) {
	p.heading("Usage:")
	p.printf("  %s <command> [arguments] [flags]\n", p.usage(n))
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
func (p *page) command(n *kong.Node, e *entry) {
	c := e.c
	var args, flags [][2]string
	groups := map[string][][2]string{}
	var groupOrder []string
	for _, prop := range e.props {
		if prop.cli.Hidden || slices.Contains(reserved, prop.name) {
			continue
		}
		if prop.cli.Arg {
			args = append(args, [2]string{placeholder(prop), describe(prop)})
			continue
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
	line := strings.Join(append([]string{p.usage(n)}, usageArgs(e.props, p.a.cfg.theme.Glyphs.Ellipsis)...), " ")
	p.heading("Usage:")
	p.printf("  %s [flags]\n\n", line)
	p.printf("%s\n", p.wrap(c.Title, 0))
	if c.Description != "" && c.Description != c.Title {
		p.printf("%s\n", p.wrap(c.Description, 0))
	}
	if aliases := n.Aliases; len(aliases) > 0 {
		p.printf("\nAliases: %s\n", strings.Join(aliases, ", "))
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
		danger = p.a.cfg.theme.Styles.Warning.Render(danger + "; it asks first, or needs --yes")
	}
	p.WriteString(danger + "\n")
	if below := p.a.below(string(c.ID)); len(below) > 0 {
		p.list(below)
	}
}

// other writes the help of a node the adapter did not build from the
// registry, such as a verb or a program's own command, from Kong's model.
func (p *page) other(n *kong.Node) {
	var args []string
	for _, a := range n.Positional {
		name := "<" + a.Name + ">"
		if !a.Required {
			name = "[" + name + "]"
		}
		args = append(args, name)
	}
	p.heading("Usage:")
	p.printf("  %s\n", strings.Join(append([]string{p.usage(n)}, append(args, "[flags]")...), " "))
	var subs [][2]string
	for _, c := range n.Children {
		if !c.Hidden {
			subs = append(subs, [2]string{c.Name, c.Help})
		}
	}
	if len(subs) > 0 {
		p.printf("  %s <command>\n", p.usage(n))
	}
	if help := n.Help; help != "" {
		p.printf("\n%s\n", p.wrap(help, 0))
	}
	if len(subs) > 0 {
		p.WriteString("\n")
		p.heading("Commands:")
		p.columns(subs)
	}
	var argRows [][2]string
	for _, a := range n.Positional {
		argRows = append(argRows, [2]string{a.Name, valueHelp(a)})
	}
	if len(argRows) > 0 {
		p.WriteString("\n")
		p.heading("Arguments:")
		p.columns(argRows)
	}
	var flags [][2]string
	for _, f := range n.Flags {
		if f.Hidden {
			continue
		}
		name := "--" + f.Name
		if f.Short != 0 {
			name = "-" + string(f.Short) + ", " + name
		}
		if !f.IsBool() {
			name += " " + f.FormatPlaceHolder()
		}
		flags = append(flags, [2]string{name, valueHelp(f.Value)})
	}
	if len(flags) > 0 {
		p.WriteString("\n")
		p.heading("Flags:")
		p.columns(flags)
	}
}

// valueHelp is a Kong value's help, with its choices and default.
func valueHelp(v *kong.Value) string {
	var notes []string
	if v.Enum != "" {
		notes = append(notes, "one of "+strings.Join(v.EnumSlice(), ", "))
	}
	if v.HasDefault {
		notes = append(notes, "default "+v.Default)
	}
	if len(notes) == 0 {
		return v.Help
	}
	return strings.TrimSpace(v.Help + " (" + strings.Join(notes, "; ") + ")")
}

// usageArgs is the positional arguments of a usage line: each one's value
// name, the ellipsis after one that repeats, and brackets round one that
// is optional.
func usageArgs(props []*property, ellipsis string) []string {
	var out []string
	for _, p := range props {
		if p.cli.Hidden || !p.cli.Arg || slices.Contains(reserved, p.name) {
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

// valueJSON is the value name of a JSON value.
const valueJSON = "JSON"

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

// describe is a property's description and its notes.
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
		notes = append(notes, valueJSON)
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
	accent := p.a.cfg.theme.Styles.Accent
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
		p.printf("  %s%s  %s\n", name, strings.Repeat(" ", pad), p.wrap(r[1], indent))
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
