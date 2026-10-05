package command

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/when"
)

// The registry's errors, wrapped with detail; test them with errors.Is
// (docs/decisions/0006-MADR-command-registry.md A4).
var (
	// ErrUnknown: no command has the request's ID.
	ErrUnknown = errors.New("command: unknown command")
	// ErrUnavailable: the command's When is false in the request's
	// context, or its Surfaces exclude the request's origin.
	ErrUnavailable = errors.New("command: not available")
	// ErrRefused: the policy or the gate refused the request, or it had no
	// origin.
	ErrRefused = errors.New("command: refused")
)

// Registry holds commands and runs them. Reads walk an immutable snapshot
// and take no lock; writes copy it and publish a new one. A Registry is
// safe for use from any goroutine.
type Registry struct {
	mu       sync.Mutex // serialises writers
	snap     atomic.Pointer[snapshot]
	policy   policy
	auditor  Auditor
	prefixer func(Source) string
	running  running
}

// snapshot is one published version of the registry. Nothing changes it
// once it is published.
type snapshot struct {
	version uint64
	entries []entry        // sorted by ID
	byID    map[ID]int     // index into entries
	bySlash map[string]int // slash names and aliases, index into entries
	changed chan struct{}  // closed when the next version is published
}

// entry is a command with its compiled When.
type entry struct {
	cmd  Command
	when when.Expr
}

// RegistryOption configures NewRegistry.
type RegistryOption func(*Registry)

// WithGate sets the gate the policy asks. Without one, a request the
// policy would ask about is refused.
func WithGate(g Gate) RegistryOption { return func(r *Registry) { r.policy.gate = g } }

// WithAuditor sets the auditor every request is recorded to.
func WithAuditor(a Auditor) RegistryOption { return func(r *Registry) { r.auditor = a } }

// WithPrefixer replaces the rule that names a loaded command's slash
// prefix when its name is taken.
func WithPrefixer(f func(Source) string) RegistryOption {
	return func(r *Registry) { r.prefixer = f }
}

// NewRegistry returns an empty registry at version 0.
func NewRegistry(o ...RegistryOption) *Registry {
	r := &Registry{}
	r.policy.always = map[alwaysKey]Decision{}
	r.running.runs = map[ID]map[uint64]context.CancelFunc{}
	for _, f := range o {
		f(r)
	}
	r.snap.Store(&snapshot{
		byID:    map[ID]int{},
		bySlash: map[string]int{},
		changed: make(chan struct{}),
	})
	return r
}

// Register adds cmds, all or none. It is an error for an ID to be
// malformed or taken, for a slash name or alias to be taken or malformed,
// for Danger to be undeclared, for an Action or Prompt to have no handler,
// and for When not to parse.
func (r *Registry) Register(cmds ...Command) error {
	if len(cmds) == 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.snap.Load()
	entries := slices.Clone(old.entries)
	ids := keySet(old.byID)
	slashes := keySet(old.bySlash)
	var errs []error
	for _, c := range cmds {
		e, err := compile(c)
		if err == nil {
			err = claim(c, ids, slashes)
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		entries = append(entries, e)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	r.publish(old, entries)
	return nil
}

// keySet copies m's keys into a set.
func keySet[K comparable](m map[K]int) map[K]bool {
	s := make(map[K]bool, len(m))
	for k := range m {
		s[k] = true
	}
	return s
}

// compile checks c on its own and compiles its When.
func compile(c Command) (entry, error) {
	if err := c.ID.Valid(); err != nil {
		return entry{}, err
	}
	if c.Danger == 0 || c.Danger > Destructive {
		return entry{}, fmt.Errorf("command: %s: danger is %s; declare ReadOnly, UI, Mutating or Destructive", c.ID, c.Danger)
	}
	if c.Kind > Forward {
		return entry{}, fmt.Errorf("command: %s: unknown kind %s", c.ID, c.Kind)
	}
	if c.Handler == nil && c.Kind != Forward {
		return entry{}, fmt.Errorf("command: %s: an %s command needs a handler", c.ID, c.Kind)
	}
	// The snapshot keeps its own copies, so a caller's later change to a
	// slice or map it passed does not reach a published snapshot.
	c.Aliases = slices.Clone(c.Aliases)
	c.Args = slices.Clone(c.Args)
	c.Output = slices.Clone(c.Output)
	c.Meta = maps.Clone(c.Meta)
	var e entry
	if c.When != "" {
		x, err := when.Parse(c.When)
		if err != nil {
			return entry{}, fmt.Errorf("command: %s: when: %w", c.ID, err)
		}
		e.when = x
	}
	e.cmd = c
	return e, nil
}

// claim takes c's ID and slash names in ids and slashes, or says which is
// taken or malformed.
func claim(c Command, ids map[ID]bool, slashes map[string]bool) error {
	if ids[c.ID] {
		return fmt.Errorf("command: %s is registered already", c.ID)
	}
	names := c.Aliases
	if c.Slash != "" {
		names = append([]string{c.Slash}, c.Aliases...)
	}
	for _, n := range names {
		if n == "" || strings.HasPrefix(n, "/") || strings.ContainsFunc(n, isSpaceOrControl) {
			return fmt.Errorf("command: %s: slash name %q is empty, starts with /, or holds a space or control character", c.ID, n)
		}
		if slashes[n] {
			return fmt.Errorf("command: %s: slash name %q is taken", c.ID, n)
		}
	}
	ids[c.ID] = true
	for _, n := range names {
		slashes[n] = true
	}
	return nil
}

func isSpaceOrControl(c rune) bool { return c <= ' ' || c == 0x7f || (c >= 0x80 && c < 0xa0) }

// Remove takes the commands with ids out. IDs that are not registered are
// ignored; when none is, the version does not change.
func (r *Registry) Remove(ids ...ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.snap.Load()
	entries := slices.DeleteFunc(slices.Clone(old.entries), func(e entry) bool {
		return slices.Contains(ids, e.cmd.ID)
	})
	if len(entries) == len(old.entries) {
		return
	}
	r.publish(old, entries)
}

// publish builds and stores the snapshot after old, holding entries, and
// wakes Watch. The caller holds r.mu, and entries is its own copy.
func (r *Registry) publish(old *snapshot, entries []entry) {
	slices.SortFunc(entries, func(a, b entry) int { return strings.Compare(string(a.cmd.ID), string(b.cmd.ID)) })
	s := &snapshot{
		version: old.version + 1,
		entries: entries,
		byID:    make(map[ID]int, len(entries)),
		bySlash: make(map[string]int, len(entries)),
		changed: make(chan struct{}),
	}
	for i, e := range entries {
		s.byID[e.cmd.ID] = i
		if e.cmd.Slash != "" {
			s.bySlash[e.cmd.Slash] = i
		}
		for _, a := range e.cmd.Aliases {
			s.bySlash[a] = i
		}
	}
	r.snap.Store(s)
	close(old.changed)
}

// Lookup is the command with id.
func (r *Registry) Lookup(id ID) (Command, bool) {
	s := r.snap.Load()
	i, ok := s.byID[id]
	if !ok {
		return Command{}, false
	}
	return s.entries[i].cmd, true
}

// Slash is the command whose slash name or alias is name, given with or
// without its leading "/".
func (r *Registry) Slash(name string) (Command, bool) {
	s := r.snap.Load()
	i, ok := s.bySlash[strings.TrimPrefix(name, "/")]
	if !ok {
		return Command{}, false
	}
	return s.entries[i].cmd, true
}

// All is every command, hidden ones too, in ID order, from one snapshot.
func (r *Registry) All() iter.Seq[Command] {
	s := r.snap.Load()
	return func(yield func(Command) bool) {
		for _, e := range s.entries {
			if !yield(e.cmd) {
				return
			}
		}
	}
}

// Available is every command that is not hidden, whose When holds in c,
// and that is offered on a surface in s (any surface when s is zero), in
// ID order, from one snapshot.
func (r *Registry) Available(c when.Context, s Surface) iter.Seq[Command] {
	snap := r.snap.Load()
	return func(yield func(Command) bool) {
		for i := range snap.entries {
			e := &snap.entries[i]
			if e.cmd.Hidden || (s != 0 && e.cmd.surfaces()&s == 0) || !e.holds(c) {
				continue
			}
			if !yield(e.cmd) {
				return
			}
		}
	}
}

// holds reports whether e's When is true in c; an empty When always is.
func (e *entry) holds(c when.Context) bool {
	return e.cmd.When == "" || e.when.Eval(c)
}

// Version counts the changes since NewRegistry. It never falls.
func (r *Registry) Version() uint64 { return r.snap.Load().version }

// Watch returns a command that waits for the registry's next change and
// then returns ChangedMsg. The host issues it again after each ChangedMsg,
// as with any subscription.
func (r *Registry) Watch() tea.Cmd {
	ch := r.snap.Load().changed
	return func() tea.Msg {
		<-ch
		return ChangedMsg{Version: r.Version()}
	}
}
