package cli

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/command"
)

// minWidth is the narrowest help is wrapped to.
const minWidth = 40

// otherCategory heads the commands without a category.
const otherCategory = "Other"

func (s shell) width() int { return max(s.cfg.width, minWidth) }

// registryHelp writes the registry's help: usage, the commands the shell
// offers by category, and the flags every command takes. Hidden commands
// are left out.
func (s shell) registryHelp(w *strings.Builder) {
	n := s.cfg.name
	fmt.Fprintf(w, "Usage:\n  %s <command> [arguments] [flags]\n", n)
	for _, verb := range []string{"list [--json]", "describe <command>", "schema <command>", "help [<command>]"} {
		fmt.Fprintf(w, "  %s %s\n", n, verb)
	}
	var cmds []command.Command
	for c := range s.r.All() {
		if offered(c) && !c.Hidden {
			cmds = append(cmds, c)
		}
	}
	s.listText(w, cmds)
	fmt.Fprintf(w, "\nFlags for every command:\n")
	s.columns(w, [][2]string{
		{"--args JSON", "the arguments as one JSON object"},
		{"--json", "print the result's value as JSON"},
		{"--yes", "run a destructive command without asking"},
		{"-h, --help", "show the command's help"},
	})
}

// listText writes cmds by category, categories in order and Other last,
// each as its words and its description or title.
func (s shell) listText(w *strings.Builder, cmds []command.Command) {
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
		fmt.Fprintf(w, "\n%s commands:\n", cat)
		s.columnsAt(w, byCat[cat], left)
	}
}

// leftWidth is the widest left column of rows.
func leftWidth(rows [][2]string) int {
	left := 0
	for _, r := range rows {
		left = max(left, ansi.StringWidth(r[0]))
	}
	return left
}

// commandHelp writes one command's help: usage, title and description, its
// positional arguments, its flags by group, and its danger. Hidden
// arguments are left out.
func (s shell) commandHelp(w *strings.Builder, c command.Command, props []*property) {
	var usage []string
	var args, flags [][2]string
	groups := map[string][][2]string{}
	var groupOrder []string
	for _, p := range props {
		if p.cli.Hidden {
			continue
		}
		if p.cli.Arg {
			name := placeholder(p)
			if p.is(tArray) {
				name += "..."
			}
			if !p.required {
				name = "[" + name + "]"
			}
			usage = append(usage, name)
			args = append(args, [2]string{placeholder(p), p.desc})
		}
		row := [2]string{flagName(p), describe(p)}
		if g := p.cli.Group; g != "" {
			if _, seen := groups[g]; !seen {
				groupOrder = append(groupOrder, g)
			}
			groups[g] = append(groups[g], row)
			continue
		}
		flags = append(flags, row)
	}
	line := strings.Join(append([]string{s.cfg.name, words(c.ID)}, usage...), " ")
	fmt.Fprintf(w, "Usage:\n  %s [flags]\n\n", line)
	fmt.Fprintf(w, "%s\n", s.wrap(c.Title, 0))
	if c.Description != "" && c.Description != c.Title {
		fmt.Fprintf(w, "%s\n", s.wrap(c.Description, 0))
	}
	if len(args) > 0 {
		fmt.Fprintf(w, "\nArguments:\n")
		s.columns(w, args)
	}
	if len(flags) > 0 {
		fmt.Fprintf(w, "\nFlags:\n")
		s.columns(w, flags)
	}
	for _, g := range groupOrder {
		fmt.Fprintf(w, "\n%s flags:\n", g)
		s.columns(w, groups[g])
	}
	fmt.Fprintf(w, "\nDanger: %s", c.Danger)
	if c.Danger == command.Destructive {
		fmt.Fprintf(w, "; it asks first, or needs --yes")
	}
	fmt.Fprintln(w)
}

// placeholder is an argument's value name: the schema's, an enum's
// choices, or one by its type.
func placeholder(p *property) string {
	switch {
	case p.cli.Placeholder != "":
		return p.cli.Placeholder
	case len(p.enum) > 0:
		parts := make([]string, len(p.enum))
		for i, e := range p.enum {
			parts[i] = fmt.Sprint(e)
		}
		return strings.Join(parts, "|")
	}
	switch p.typeOf() {
	case tInteger, tNumber:
		return "N"
	case tBoolean:
		return "BOOL"
	case tObject:
		return "JSON"
	case tArray:
		if itemType(p) == tObject {
			return "JSON"
		}
		return "VALUE"
	}
	return "TEXT"
}

// flagName is a property's flag as help shows it.
func flagName(p *property) string {
	f := "--" + p.name
	if p.cli.Short != "" {
		f = "-" + p.cli.Short + ", " + f
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

// columns writes rows as two columns indented two cells, the right one
// wrapped at the width with its later lines under its first.
func (s shell) columns(w *strings.Builder, rows [][2]string) { s.columnsAt(w, rows, leftWidth(rows)) }

// columnsAt is columns with the left column left cells wide, at most a
// third of the width.
func (s shell) columnsAt(w *strings.Builder, rows [][2]string, left int) {
	left = min(left, s.width()/3)
	for _, r := range rows {
		name := r[0]
		pad := left - ansi.StringWidth(name)
		indent := 2 + left + 2
		if pad < 0 {
			fmt.Fprintf(w, "  %s\n", name)
			name, pad = "", left
		}
		text := s.wrap(r[1], indent)
		fmt.Fprintf(w, "  %s%s  %s\n", name, strings.Repeat(" ", pad), text)
	}
}

// wrap wraps text to the width less indent, and indents the lines after
// the first by indent.
func (s shell) wrap(text string, indent int) string {
	lines := strings.Split(ansi.Wordwrap(text, max(s.width()-indent, 10), ""), "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.Repeat(" ", indent) + lines[i]
	}
	return strings.Join(lines, "\n")
}
