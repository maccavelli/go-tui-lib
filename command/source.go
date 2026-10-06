package command

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ReplaceSource swaps src's whole set of commands for cmds in one version:
// ACP sends its full list in each update, and MCP says only that its list
// changed. Each command's Source becomes src, and its ID must lie in src's
// namespace: user.…, project.…, mcp.<server>.…, acp.<agent>.… or
// plugin.<name>.…, the name as an ID segment.
//
// Built-ins win slash names. A command whose slash name or alias is taken
// is renamed <prefix>:<name>, "user:review" or "mcp:github:issue", or
// loses the name when that is taken too. A command that cannot be loaded,
// because it is malformed or its ID is taken, is left out. Each rename and
// refusal is a Conflict, returned and sent through Watch as ConflictMsg
// (docs/decisions/0006-MADR-command-registry.md §6, A6).
func (r *Registry) ReplaceSource(src Source, cmds []Command) []Conflict {
	ns, err := namespace(src)
	if err != nil {
		conflicts := make([]Conflict, len(cmds))
		for i, c := range cmds {
			conflicts[i] = Conflict{ID: c.ID, Slash: c.Slash, Err: err}
		}
		return conflicts
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.snap.Load()
	d := r.draftOf(old, func(e *entry) bool { return !e.builtin && e.cmd.Source == src })
	for _, c := range cmds {
		c.Source = src
		d.addLoaded(c, ns)
	}
	r.publish(old, d)
	return slices.Clone(d.conflicts)
}

// namespace is the ID prefix src's commands live under.
func namespace(src Source) (string, error) {
	switch src.Kind {
	case User, Project:
		return src.Kind.String(), nil
	case MCP, ACP, Plugin:
		if seg := segment(src.Name); seg != "" {
			return src.Kind.String() + "." + seg, nil
		}
		return "", fmt.Errorf("command: a %s source needs a name", src.Kind)
	}
	return "", errors.New("command: built-in commands are added with Register, not ReplaceSource")
}

// addLoaded adds c, a loaded command in namespace ns, renaming its taken
// slash names, or reports why it cannot.
func (d *draft) addLoaded(c Command, ns string) {
	refuse := func(err error) {
		d.conflicts = append(d.conflicts, Conflict{ID: c.ID, Slash: c.Slash, Err: err})
	}
	if !strings.HasPrefix(string(c.ID), ns+".") {
		refuse(fmt.Errorf("command: %s is outside %s's namespace, %s.*", c.ID, c.Source, ns))
		return
	}
	e, err := compile(c)
	if err != nil {
		refuse(err)
		return
	}
	if i, ok := d.ids[c.ID]; ok {
		refuse(fmt.Errorf("command: %s is taken by %s's command", c.ID, d.entries[i].cmd.Source))
		return
	}
	for _, n := range slashNames(&e.cmd) {
		if err := validSlash(c.ID, n); err != nil {
			refuse(err)
			return
		}
	}
	at := len(d.entries)
	d.entries = append(d.entries, e)
	d.ids[c.ID] = at
	lc := &d.entries[at].cmd
	if lc.Slash != "" {
		lc.Slash = d.claim(lc, lc.Slash, at)
	}
	aliases := make([]string, 0, len(lc.Aliases))
	for _, a := range lc.Aliases {
		if got := d.claim(lc, a, at); got != "" {
			aliases = append(aliases, got)
		}
	}
	lc.Aliases = aliases
}

// claim gives the loaded command c at i the slash name n, or its renamed
// form when n is taken, reporting the rename; "" when both are taken.
func (d *draft) claim(c *Command, n string, i int) string {
	holder, taken := d.slashes[n]
	switch {
	case !taken:
		d.slashes[n] = i
		return n
	case holder == i:
		return "" // the command names it twice
	}
	renamed := d.rename(c.Source, n, i)
	d.conflicts = append(d.conflicts, Conflict{ID: c.ID, Slash: n, Renamed: renamed, Holder: d.entries[holder].cmd.ID})
	return renamed
}

// segment maps name to an ID segment: lowercased, each run of characters
// outside [a-z0-9] one "-", the ends trimmed. It is "" when nothing is
// left (docs/decisions/0006-MADR-command-registry.md A6).
func segment(name string) string {
	var b strings.Builder
	gap := false
	for _, c := range strings.ToLower(name) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			if gap && b.Len() > 0 {
				b.WriteByte('-')
			}
			gap = false
			b.WriteRune(c)
			continue
		}
		gap = true
	}
	return b.String()
}
