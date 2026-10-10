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

	"github.com/maccavelli/go-tui-lib/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/when"
)

// The registry's errors, wrapped with detail; test them with errors.Is
// (docs/decisions/0006-MADR-command-registry.md A4). Each has an
// ExitCode() int, the status a CLI exits with when the error ends it
// (docs/decisions/0014-MADR-native-integration-api.md A1), which
// launch.ExitCode honours through any wrapping.
var (
	// ErrUnknown: no command has the request's ID. Its status is 2.
	ErrUnknown error = &codedError{"command: unknown command", 2}
	// ErrUnavailable: the command's When is false in the request's
	// context, or its Surfaces exclude the request's origin. Its status
	// is 1.
	ErrUnavailable error = &codedError{"command: not available", 1}
	// ErrRefused: the policy or the gate refused the request, or it had no
	// origin. Its status is 3.
	ErrRefused error = &codedError{"command: refused", 3}
	// ErrPanicked: a handler or a gate panicked. A *PanicError wraps it.
	// Its status is 2.
	ErrPanicked error = &codedError{"command: panicked", 2}
)

// codedError is a sentinel with an exit status. It is compared by
// identity, as errors.New's are.
type codedError struct {
	msg  string
	code int
}

func (e *codedError) Error() string { return e.msg }

// ExitCode is the status a CLI exits with.
func (e *codedError) ExitCode() int { return e.code }

// Registry holds commands and runs them. Reads walk an immutable snapshot
// and take no lock; writes copy it and publish a new one. A Registry is
// safe for use from any goroutine.
type Registry struct {
	mu       sync.Mutex // serialises writers, and Attach and detach
	snap     atomic.Pointer[snapshot]
	policy   policy
	auditor  Auditor
	prefixer func(Source) string
	loop     atomic.Pointer[loopState] // the program's loop, or nil
	running  running
	maxArg   int // the largest request's arguments, and slash or command line, in bytes
}

// DefaultMaxArgBytes is the largest request's arguments a registry takes,
// in bytes, unless WithMaxArgBytes sets another
// (docs/decisions/0014-PLAN-hardening.md Step 6, finding H5).
const DefaultMaxArgBytes = 1 << 20

// WithMaxArgBytes sets the largest request's arguments the registry takes,
// in bytes: Request.Args and Request.Raw, a slash line's tail, and the
// words ParseArgs reads, each on its own. Larger is an *ArgError before
// anything parses it. A value below 1 keeps DefaultMaxArgBytes.
func WithMaxArgBytes(n int) RegistryOption {
	return registryOptionFunc(func(r *Registry) {
		if n > 0 {
			r.maxArg = n
		}
	})
}

// tooLarge is the error for n bytes of arguments, or nil when they fit.
func (r *Registry) tooLarge(n int) error {
	if n <= r.maxArg {
		return nil
	}
	return &ArgError{Reason: fmt.Sprintf("arguments are %d bytes, over %d", n, r.maxArg)}
}

// snapshot is one published version of the registry. Nothing changes it
// once it is published, but next, which is set once, before changed is
// closed.
type snapshot struct {
	version   uint64
	entries   []entry        // sorted by ID
	byID      map[ID]int     // index into entries
	bySlash   map[string]int // slash names and aliases, index into entries
	changed   chan struct{}  // closed when the next version is published
	next      *snapshot      // the next version, once changed is closed
	conflicts []Conflict     // what the write that made this version renamed or refused
}

// entry is a command with its compiled When and arguments schema.
type entry struct {
	cmd     Command
	when    when.Expr
	args    *rule // nil when the command has no schema
	builtin bool  // added by Register, not loaded by ReplaceSource
}

// loopState is one attachment of a program's loop: its send, and a channel
// closed when it is detached. WithLoop's done is nil: it is never
// detached.
type loopState struct {
	send func(tea.Msg)
	done chan struct{}
}

// RegistryOption configures NewRegistry. It is opaque: only this package's
// functions make one.
type RegistryOption interface{ apply(*Registry) }

// registryOptionFunc is a function used as a RegistryOption.
type registryOptionFunc func(*Registry)

func (f registryOptionFunc) apply(r *Registry) { f(r) }

// WithGate sets the gate the policy asks. Without one, a request the
// policy would ask about is refused.
func WithGate(g Gate) RegistryOption {
	return registryOptionFunc(func(r *Registry) { r.policy.gate = g })
}

// WithAuditor sets the auditor every request is recorded to.
func WithAuditor(a Auditor) RegistryOption {
	return registryOptionFunc(func(r *Registry) { r.auditor = a })
}

// WithPrefixer replaces the rule that names a loaded command's slash
// prefix when its name is taken.
func WithPrefixer(f func(Source) string) RegistryOption {
	return registryOptionFunc(func(r *Registry) { r.prefixer = f })
}

// WithLoop gives the registry the program's event loop: send is the
// program's Send. Run, called off the loop, then hands a Loop command to
// the loop as a LoopMsg and waits for it, so an agent's tool call never
// touches the program's state from its own goroutine
// (docs/decisions/0006-MADR-command-registry.md A7). With WithLoop set,
// Update must use Dispatch, not Run, or it waits for itself.
//
// WithLoop is a permanent Attach, for a registry whose program runs for its
// whole life.
func WithLoop(send func(tea.Msg)) RegistryOption {
	return registryOptionFunc(func(r *Registry) { r.loop.Store(&loopState{send: send}) })
}

// NewRegistry returns a registry at version 0 holding only its own
// commands: command.list, command.describe and app.quit.
func NewRegistry(o ...RegistryOption) *Registry {
	r := &Registry{maxArg: DefaultMaxArgBytes}
	r.policy.always = map[alwaysKey]Verdict{}
	r.running.runs = map[ID]map[uint64]context.CancelFunc{}
	for _, f := range o {
		f.apply(r)
	}
	d := r.draftOf(&snapshot{}, nil)
	for _, c := range r.builtins() {
		if err := d.addBuiltin(c); err != nil {
			panic(err) // the registry's own commands are fixed; this is a bug
		}
	}
	r.snap.Store(newSnapshot(0, d.entries, nil))
	return r
}

// Register adds built-in commands, all or none. It is an error for an ID
// to be malformed or taken, for a slash name or alias to be malformed or
// held by another built-in, for Danger to be undeclared, for an Action or
// Prompt to have no handler, and for When not to parse. A slash name a
// loaded command holds goes to the built-in, and the loaded command is
// renamed as ReplaceSource renames, reported through Watch
// (docs/decisions/0006-MADR-command-registry.md A6).
func (r *Registry) Register(cmds ...Command) error {
	if len(cmds) == 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.snap.Load()
	d := r.draftOf(old, nil)
	var errs []error
	for _, c := range cmds {
		if err := d.addBuiltin(c); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	r.publish(old, d)
	return nil
}

// compile checks c on its own and compiles its When and its arguments
// schema.
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
	if len(c.Args) > 0 {
		r, err := compileRule(c.Args)
		if err != nil {
			return entry{}, fmt.Errorf("command: %s: args schema: %w", c.ID, err)
		}
		e.args = r
	}
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

// slashNames is c's slash name and aliases.
func slashNames(c *Command) []string {
	if c.Slash == "" {
		return c.Aliases
	}
	return append([]string{c.Slash}, c.Aliases...)
}

// validSlash says why n cannot be a slash name, or nil.
func validSlash(id ID, n string) error {
	if n == "" || strings.HasPrefix(n, "/") || strings.ContainsFunc(n, isSpaceOrControl) {
		return fmt.Errorf("command: %s: slash name %q is empty, starts with /, or holds a space or control character", id, n)
	}
	return nil
}

// isSpaceOrControl reports whether c is a space, or a character
// sanitize.Line would change: a control, a bidirectional or invisible
// character (docs/decisions/0014-PLAN-hardening.md Step 2).
func isSpaceOrControl(c rune) bool { return c == ' ' || sanitize.HasControl(string(c)) }

// draft is a write in progress: a copy of a snapshot's entries, indexed,
// and the conflicts the write has met.
type draft struct {
	entries   []entry
	ids       map[ID]int
	slashes   map[string]int
	conflicts []Conflict
	prefix    func(Source) string
}

// draftOf copies s's entries, leaving out those drop reports.
func (r *Registry) draftOf(s *snapshot, drop func(*entry) bool) *draft {
	d := &draft{ids: map[ID]int{}, slashes: map[string]int{}, prefix: r.prefix}
	for i := range s.entries {
		if drop == nil || !drop(&s.entries[i]) {
			d.entries = append(d.entries, s.entries[i])
		}
	}
	for i := range d.entries {
		d.ids[d.entries[i].cmd.ID] = i
		for _, n := range slashNames(&d.entries[i].cmd) {
			d.slashes[n] = i
		}
	}
	return d
}

// addBuiltin adds c, taking any slash name a loaded command holds.
func (d *draft) addBuiltin(c Command) error {
	e, err := compile(c)
	if err != nil {
		return err
	}
	e.builtin = true
	if _, ok := d.ids[c.ID]; ok {
		return fmt.Errorf("command: %s is registered already", c.ID)
	}
	names := slashNames(&e.cmd)
	for _, n := range names {
		if err := validSlash(c.ID, n); err != nil {
			return err
		}
		if i, ok := d.slashes[n]; ok && d.entries[i].builtin {
			return fmt.Errorf("command: %s: slash name %q is taken by %s", c.ID, n, d.entries[i].cmd.ID)
		}
	}
	at := len(d.entries)
	d.entries = append(d.entries, e)
	d.ids[c.ID] = at
	for _, n := range names {
		if i, ok := d.slashes[n]; ok && i != at {
			d.displace(i, n, c.ID)
		}
		d.slashes[n] = at
	}
	return nil
}

// displace moves the loaded command at i off the slash name n, which
// holder takes: to its prefix and n, or to none when that is taken too.
func (d *draft) displace(i int, n string, holder ID) {
	c := &d.entries[i].cmd
	renamed := d.rename(c.Source, n, i)
	if c.Slash == n {
		c.Slash = renamed
	}
	aliases := make([]string, 0, len(c.Aliases))
	for _, a := range c.Aliases {
		switch {
		case a != n:
			aliases = append(aliases, a)
		case renamed != "":
			aliases = append(aliases, renamed)
		}
	}
	c.Aliases = aliases
	d.conflicts = append(d.conflicts, Conflict{ID: c.ID, Slash: n, Renamed: renamed, Holder: holder})
}

// rename claims src's prefixed form of n for the entry at i, or returns ""
// when that is taken too.
func (d *draft) rename(src Source, n string, i int) string {
	renamed := d.prefix(src) + ":" + n
	if _, taken := d.slashes[renamed]; taken {
		return ""
	}
	d.slashes[renamed] = i
	return renamed
}

// Remove takes the commands with ids out. IDs that are not registered are
// ignored; when none is, the version does not change.
func (r *Registry) Remove(ids ...ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.snap.Load()
	d := r.draftOf(old, func(e *entry) bool { return slices.Contains(ids, e.cmd.ID) })
	if len(d.entries) == len(old.entries) {
		return
	}
	r.publish(old, d)
}

// publish builds and stores the snapshot after old from d, and wakes
// Watch. The caller holds r.mu.
func (r *Registry) publish(old *snapshot, d *draft) {
	s := newSnapshot(old.version+1, d.entries, d.conflicts)
	old.next = s
	r.snap.Store(s)
	close(old.changed)
}

// newSnapshot sorts entries, which it then owns, and indexes them.
func newSnapshot(version uint64, entries []entry, conflicts []Conflict) *snapshot {
	slices.SortFunc(entries, func(a, b entry) int { return strings.Compare(string(a.cmd.ID), string(b.cmd.ID)) })
	s := &snapshot{
		version:   version,
		entries:   entries,
		byID:      make(map[ID]int, len(entries)),
		bySlash:   make(map[string]int, len(entries)),
		changed:   make(chan struct{}),
		conflicts: conflicts,
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
	return s
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
// then returns ChangedMsg, with a ConflictMsg when that change renamed or
// refused a command (docs/decisions/0006-MADR-command-registry.md A6). The
// host issues it again after each ChangedMsg, as with any subscription.
// It waits for ever; WatchContext stops waiting.
func (r *Registry) Watch() tea.Cmd { return r.WatchContext(context.Background()) }

// WatchContext is Watch, and returns a nil message, which Bubble Tea
// ignores, once ctx ends: a program that stops listening cancels ctx, and
// leaves no goroutine waiting.
func (r *Registry) WatchContext(ctx context.Context) tea.Cmd {
	s := r.snap.Load()
	return func() tea.Msg {
		select {
		case <-s.changed:
		case <-ctx.Done():
			return nil
		}
		next := s.next
		changed := ChangedMsg{Version: next.version}
		if len(next.conflicts) == 0 {
			return changed
		}
		return tea.BatchMsg{msgCmd(changed), msgCmd(ConflictMsg{Conflicts: slices.Clone(next.conflicts)})}
	}
}

// prefix is the prefix a loaded command from src takes on a slash clash:
// the WithPrefixer function's, or the source's kind and its name as an ID
// segment, "mcp:github".
func (r *Registry) prefix(src Source) string {
	if r.prefixer != nil {
		return r.prefixer(src)
	}
	if src.Name == "" {
		return src.Kind.String()
	}
	return src.Kind.String() + ":" + segment(src.Name)
}
