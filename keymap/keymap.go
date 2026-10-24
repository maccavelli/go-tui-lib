package keymap

import (
	"cmp"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"iter"
	"runtime"
	"slices"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// Path is the focus path, outermost first: the scopes a key is resolved
// in, the innermost first. A path lists command.Global itself when global
// bindings apply.
type Path []command.Scope

// Layer is where a binding comes from. A later layer overrides an earlier
// one; the user's file is last.
type Layer uint8

// The layers. DefaultLayer is the zero value.
const (
	DefaultLayer Layer = iota // a package's or a program's defaults
	ProgramLayer              // a program's own bindings
	UserLayer                 // the user's file
)

var layerNames = enum.Names[Layer]{
	Pkg: pkgName, Type: "Layer",
	Tokens: []string{"default", "program", "user"},
}

// String is the layer's token.
func (l Layer) String() string { return layerNames.String(l) }

// MarshalText is the layer's token. A layer with no token is an error.
func (l Layer) MarshalText() ([]byte, error) { return layerNames.Marshal(l) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (l *Layer) UnmarshalText(b []byte) error { return layerNames.Unmarshal(b, l) }

// Binding binds a key sequence to a command in a scope.
type Binding struct {
	Keys     Sequence
	Command  command.ID
	Args     jsontext.Value // passed through to the command, as VS Code's args
	When     string         // a when expression; empty means always
	Scope    command.Scope  // command.Global, the zero value, or a pane, overlay or mode name
	Remove   bool           // VS Code's "-command": remove a binding an earlier one made
	Location Location       // where the binding came from, for errors
}

// Location is where a binding came from.
type Location struct {
	Layer     Layer
	File      string           // "" for code
	Pointer   jsontext.Pointer // RFC 6901, such as "/3/key"
	Line, Col int
}

// String is "file:line:col (pointer)", as much of it as is known, or the
// layer's name for a binding made in code.
func (l Location) String() string {
	s := l.File
	if s == "" {
		s = l.Layer.String() + " layer"
	}
	if l.Line > 0 {
		s += fmt.Sprintf(":%d:%d", l.Line, l.Col)
	}
	if l.Pointer != "" {
		s += " (" + string(l.Pointer) + ")"
	}
	return s
}

// Features are the keyboard enhancements the terminal reports, as the
// Kitty protocol's flags (MADR §5).
type Features uint8

// The features, each the Kitty flag of the same bit.
const (
	Disambiguate     Features = 1 << iota // tell ctrl+i from tab, shift+enter from enter
	EventTypes                            // report releases and repeats
	AlternateKeys                         // report the US-layout key, for Scan strokes
	AllKeysAsEscapes                      // report every key as an escape code
	AssociatedText                        // report the text a key types
)

// allFeatures is every feature this package names.
const allFeatures = Disambiguate | EventTypes | AlternateKeys | AllKeysAsEscapes | AssociatedText

var featureNames = enum.Bits[Features]{
	Pkg: pkgName, Type: "Features", None: "none",
	Tokens: []string{"disambiguate", "event-types", "alternate-keys", "all-keys-as-escapes", "associated-text"},
}

// String is the set's names joined with "|", or "none". Bits with no name
// come last, in hex.
func (f Features) String() string { return featureNames.String(f) }

// MarshalText is the set's names joined with "|", or "none". A bit with no
// name is an error.
func (f Features) MarshalText() ([]byte, error) { return featureNames.Marshal(f) }

// UnmarshalText reads what MarshalText wrote, the names in any order.
func (f *Features) UnmarshalText(b []byte) error { return featureNames.Unmarshal(b, f) }

// FeaturesFrom is the features in Kitty keyboard flags, as
// tea.KeyboardEnhancementsMsg.Flags and termcap's KeyboardCaps.Accepted
// give them. Flags this package does not name are dropped.
func FeaturesFrom(flags int) Features {
	var f Features
	for bit := Disambiguate; bit&allFeatures != 0; bit <<= 1 {
		if flags&int(bit) != 0 {
			f |= bit
		}
	}
	return f
}

// Default is a binding a package or program gives, with keys to bind
// instead where the terminal cannot deliver its own (MADR amendment A1).
// Its own keys need their features, and, like any alternative's, the
// Releases fact for a release or repeat stroke; a host's taking ctrl+q or
// ctrl+i bars those strokes.
type Default struct {
	Binding
	// Alternatives are tried in order when the binding's own keys cannot
	// be delivered: the first whose needs and facts hold is bound, and
	// with none, nothing is, and nothing is reported.
	Alternatives []Alternative
}

// Settings configure a keymap.
type Settings struct {
	Leader   string        // the sequence <leader> stands for; "" means none
	Timeout  time.Duration // how long a pending chord or leader waits; 0 means one second
	Features Features      // what the terminal reports; empty until it says
	Facts    KeyFacts      // what the terminal and its host deliver, as FactsFrom reads them
	// EscReplays makes esc end a pending sequence and be replayed with it,
	// as MADR §6 says. Unset, the default, esc clears the sequence and is
	// swallowed (amendment A10).
	EscReplays bool
	// BackspaceReplays makes backspace end a pending sequence and be
	// replayed with it. Unset, the default, backspace removes the
	// sequence's last stroke.
	BackspaceReplays bool
	Reserved         []string // sequences only the program may bind, such as "ctrl+c"
	// GOOS is the system the keys come from, as runtime.GOOS names it, for
	// Normalize's AltGr; "" means runtime.GOOS. A program serving a remote
	// client sets the client's.
	GOOS string
}

// Resolved is the binding a sequence resolves to.
type Resolved struct {
	Command command.ID
	Args    jsontext.Value
	Binding Binding // the binding that won, with its scope and location
}

// Keymap is a built set of bindings. It does not change after New, and is
// safe for concurrent use; Rebuild gives a new one.
type Keymap struct {
	settings  Settings
	defaults  []Default
	layers    [][]Binding
	bound     []*entry                              // the bindings in effect, in order
	scopes    map[command.Scope][]*entry            // each scope's, in resolution order
	index     map[command.Scope]map[Stroke][]*entry // the same, by first stroke
	conflicts []BindingConflict                     // the warnings
	leader    Sequence                              // Settings.Leader, parsed
}

// entry is a binding in effect: its keys expanded, a default's keys
// chosen, its when parsed, and the layer it came in.
type entry struct {
	Binding
	layer   int // 0 for the defaults, then each layer of New's in turn
	when    when.Expr
	hasWhen bool
	own     Sequence   // a default's own keys, expanded
	alts    []Sequence // a default's alternatives' keys, expanded
}

// goos is Settings.GOOS, or runtime.GOOS when it is empty.
func (k *Keymap) goos() string { return cmp.Or(k.settings.GOOS, runtime.GOOS) }

// holds reports whether the entry's when holds in ctx.
func (e *entry) holds(ctx when.Context) bool {
	if !e.hasWhen {
		return true
	}
	if ctx == nil {
		ctx = when.Map(nil)
	}
	return e.when.Eval(ctx)
}

// New builds a keymap from the defaults and then each layer in turn
// (amendment A6). It always returns a keymap, holding every binding that
// built, and every error found, joined: a bad when, a <leader> with no
// leader, a removal of a key not bound, and each Duplicate, Prefix,
// Reserved and NotSupported BindingConflict. A NotSupported binding binds
// when Rebuild gives its features. Shadows do not fail; Conflicts lists
// them.
func New(s Settings, defaults []Default, layers ...[]Binding) (*Keymap, error) {
	s.Reserved = slices.Clone(s.Reserved)
	k := &Keymap{settings: s, defaults: slices.Clone(defaults), layers: make([][]Binding, len(layers))}
	for i := range k.defaults {
		k.defaults[i].Binding = cloneBinding(k.defaults[i].Binding)
		k.defaults[i].Alternatives = slices.Clone(k.defaults[i].Alternatives)
		for j := range k.defaults[i].Alternatives {
			k.defaults[i].Alternatives[j].Keys = slices.Clone(k.defaults[i].Alternatives[j].Keys)
		}
	}
	for i, l := range layers {
		k.layers[i] = make([]Binding, len(l))
		for j, b := range l {
			k.layers[i][j] = cloneBinding(b)
		}
	}
	err := k.build()
	return k, err
}

func cloneBinding(b Binding) Binding {
	b.Keys = slices.Clone(b.Keys)
	b.Args = slices.Clone(b.Args)
	return b
}

// Rebuild is the keymap built again from the same defaults and layers with
// new features and facts, as when a tea.KeyboardEnhancementsMsg or a late
// probe reply arrives.
func (k *Keymap) Rebuild(f Features, facts KeyFacts) (*Keymap, error) {
	s := k.settings
	s.Features, s.Facts = f, facts
	return New(s, k.defaults, k.layers...)
}

// builder is New's state.
type builder struct {
	k        *Keymap
	leader   Sequence
	reserved []Sequence
	all      []*entry // every binding made, defaults left unbound included
	off      map[*entry]bool
	errs     []error
}

func (k *Keymap) build() error {
	b := &builder{k: k, off: map[*entry]bool{}}
	if k.settings.Leader != "" {
		q, err := ParseSequence(k.settings.Leader, nil)
		switch {
		case err != nil:
			b.errs = append(b.errs, fmt.Errorf("keymap: Settings.Leader: %w", err))
		case slices.ContainsFunc(q, func(s Stroke) bool { return s.Code == keyLeader }):
			b.errs = append(b.errs, fmt.Errorf("%w %q: Settings.Leader names %s", ErrNotation, k.settings.Leader, leaderName))
		default:
			b.leader, k.leader = q, q
		}
	}
	for _, r := range k.settings.Reserved {
		q, err := ParseSequence(r, b.leader)
		if err == nil {
			q, err = b.expand(q, r)
		}
		if err != nil {
			b.errs = append(b.errs, fmt.Errorf("keymap: Settings.Reserved: %w", err))
			continue
		}
		b.reserved = append(b.reserved, q)
	}
	for _, d := range k.defaults {
		b.add(d.Binding, 0, &d)
	}
	for i, l := range k.layers {
		for _, bd := range l {
			b.add(bd, i+1, nil)
		}
	}
	for _, e := range b.all {
		if !b.off[e] {
			k.bound = append(k.bound, e)
		}
	}
	b.check()
	k.scopes = map[command.Scope][]*entry{}
	k.index = map[command.Scope]map[Stroke][]*entry{}
	for _, e := range slices.Backward(k.bound) {
		k.scopes[e.Scope] = append(k.scopes[e.Scope], e)
		byStroke := k.index[e.Scope]
		if byStroke == nil {
			byStroke = map[Stroke][]*entry{}
			k.index[e.Scope] = byStroke
		}
		byStroke[e.Keys[0]] = append(byStroke[e.Keys[0]], e)
	}
	return errors.Join(b.errs...)
}

// expand replaces the leader placeholder with the leader.
func (b *builder) expand(q Sequence, at string) (Sequence, error) {
	if !slices.ContainsFunc(q, func(s Stroke) bool { return s.Code == keyLeader }) {
		return q, nil
	}
	if len(b.leader) == 0 {
		return nil, fmt.Errorf("%w %q: %s, and no leader is set", ErrNotation, at, leaderName)
	}
	var out Sequence
	for _, s := range q {
		if s.Code == keyLeader {
			out = append(out, b.leader...)
		} else {
			out = append(out, s)
		}
	}
	if len(out) > MaxStrokes {
		return nil, fmt.Errorf("%w %q: %d strokes with the leader, more than %d", ErrNotation, at, len(out), MaxStrokes)
	}
	return out, nil
}

// fail records err about bd, naming its location and keys.
func (b *builder) fail(bd Binding, err error) {
	b.errs = append(b.errs, fmt.Errorf("%s: %w", where(bd), err))
}

// where is a binding's location, keys and scope, as its errors begin.
func where(bd Binding) string {
	s := bd.Location.String()
	if len(bd.Keys) > 0 {
		s += ": " + bd.Keys.String()
	}
	if bd.Scope != command.Global {
		s += " in " + string(bd.Scope)
	}
	return s
}

// add makes one binding, or a default when d is not nil.
func (b *builder) add(bd Binding, layer int, d *Default) {
	keys, err := b.expand(bd.Keys, bd.Keys.String())
	if err != nil {
		b.fail(bd, err)
		return
	}
	bd.Keys = keys
	if len(bd.Keys) == 0 && !bd.Remove {
		b.fail(bd, fmt.Errorf("%s binds no keys", bd.Command))
		return
	}
	e := &entry{Binding: bd, layer: layer}
	if bd.When != "" {
		x, err := when.Parse(bd.When)
		if err != nil {
			b.fail(bd, fmt.Errorf("when: %w", err))
			return
		}
		e.when, e.hasWhen = x, true
	}
	if bd.Remove {
		b.remove(bd)
		return
	}
	switch {
	case d != nil:
		e.own = bd.Keys
		chosen := b.deliverable(bd.Keys, 0, 0)
		for i, a := range d.Alternatives {
			keys, err := b.expand(a.Keys, a.Keys.String())
			if err == nil && len(keys) == 0 {
				err = errors.New("binds no keys")
			}
			if err != nil {
				b.fail(bd, fmt.Errorf("alternative %d: %w", i+1, err))
				return
			}
			e.alts = append(e.alts, keys)
			if !chosen && b.deliverable(keys, a.Needs, a.Facts) {
				e.Keys, chosen = keys, true
			}
		}
		if !chosen {
			b.off[e] = true // no binding, and nothing to report
		}
	case !b.supported(bd.Keys):
		b.errs = append(b.errs, BindingConflict{Kind: NotSupported, Keys: bd.Keys, Scope: bd.Scope, Bindings: []Binding{bd}})
		return
	case bd.Location.Layer == UserLayer:
		for _, r := range b.reserved {
			if hasPrefix(bd.Keys, r) {
				b.errs = append(b.errs, BindingConflict{Kind: Reserved, Keys: r, Scope: bd.Scope, Bindings: []Binding{bd}})
				return
			}
		}
	}
	b.all = append(b.all, e)
}

// deliverable reports whether a default's keys can be delivered: their
// features and f are reported, their facts and facts hold, and no fact
// bars them.
func (b *builder) deliverable(q Sequence, f Features, facts KeyFacts) bool {
	s := b.k.settings
	f |= needs(q)
	need, bar := keyFacts(q)
	need |= facts
	return s.Features&f == f && s.Facts&need == need && s.Facts&bar == 0
}

// supported reports whether a binding's keys can be delivered: their
// features are reported and, for a release or repeat stroke, the
// Releases fact holds (amendment A1). A host's keys do not bar a user's
// own choice.
func (b *builder) supported(q Sequence) bool {
	s := b.k.settings
	f := needs(q)
	need, _ := keyFacts(q)
	return s.Features&f == f && s.Facts&need == need
}

// remove takes away what an earlier binding made: with keys, those keys of
// that command in that scope, and an error when none is bound; with none,
// every binding of the command in the scope. A when limits it to bindings
// with that when's text.
func (b *builder) remove(bd Binding) {
	found := false
	b.all = slices.DeleteFunc(b.all, func(e *entry) bool {
		if e.Scope != bd.Scope || e.Command != bd.Command || bd.When != "" && e.When != bd.When {
			return false
		}
		if len(bd.Keys) > 0 && !slices.Equal(e.Keys, bd.Keys) && !slices.Equal(e.own, bd.Keys) &&
			!slices.ContainsFunc(e.alts, func(q Sequence) bool { return slices.Equal(q, bd.Keys) }) {
			return false
		}
		found = true
		delete(b.off, e)
		return true
	})
	if !found && len(bd.Keys) > 0 {
		b.fail(bd, fmt.Errorf("removes %s from %s, which is not bound there", bd.Keys, bd.Command))
	}
}

// ambiguous are the strokes a legacy terminal cannot tell from another
// key, which need Disambiguate (MADR §5).
var ambiguous = func() map[Stroke]bool {
	m := map[Stroke]bool{}
	for _, s := range []string{"ctrl+i", "ctrl+m", "ctrl+[", "shift+enter", "alt+[", "alt+]"} {
		st, err := ParseStroke(s)
		if err != nil {
			panic(err)
		}
		m[st] = true
	}
	return m
}()

// needs is the features a sequence's strokes need: Disambiguate for the
// ambiguous ones, EventTypes for a release or repeat, and AlternateKeys for
// a Scan stroke.
func needs(q Sequence) Features {
	var f Features
	for _, s := range q {
		if s.Event != Press {
			f |= EventTypes
		}
		if s.Scan {
			f |= AlternateKeys
		}
		press := s
		press.Event = Press
		if ambiguous[press] {
			f |= Disambiguate
		}
	}
	return f
}

// hasPrefix reports whether q starts with p.
func hasPrefix(q, p Sequence) bool {
	return len(q) >= len(p) && slices.Equal(q[:len(p)], p)
}

// related reports whether one sequence starts with the other.
func related(a, b Sequence) bool { return hasPrefix(a, b) || hasPrefix(b, a) }

// Lookup resolves a sequence along a path. The innermost scope with a
// binding whose when holds in ctx and whose keys are the sequence, start
// it, or start with it decides (amendment A6): in it, the later layer and
// then the later binding wins, and outer scopes are not tried. ok is false
// when no binding has the sequence's keys, or when the deciding scope has
// only longer or shorter ones.
func (k *Keymap) Lookup(p Path, ctx when.Context, q Sequence) (r Resolved, ok bool) {
	if len(q) == 0 {
		return Resolved{}, false
	}
	for _, scope := range slices.Backward(p) {
		decided := false
		for _, e := range k.index[scope][q[0]] {
			if !e.holds(ctx) {
				continue
			}
			if slices.Equal(e.Keys, q) {
				return Resolved{Command: e.Command, Args: e.Args, Binding: e.Binding}, true
			}
			decided = decided || related(e.Keys, q)
		}
		if decided {
			return Resolved{}, false
		}
	}
	return Resolved{}, false
}

// Bindings is the sequences bound to id along a path, in resolution order,
// so the first is the primary. A sequence a binding with no when hides is
// left out: one earlier in its scope with the same keys, or one in an
// inner scope whose keys start it or start with it.
func (k *Keymap) Bindings(id command.ID, p Path) []Sequence {
	var out []Sequence
	for _, e := range k.shownEntries(id, p) {
		out = append(out, e.Keys)
	}
	return out
}

// Reachable is the commands a key can reach along a path with ctx: each
// with a binding in a scope on the path whose when holds, not hidden by
// another whose when holds, one in an inner scope whose keys start it or
// start with it, or one before it in its scope with the same keys
// (amendment A1). Each command comes once, innermost scope first and in
// resolution order.
func (k *Keymap) Reachable(p Path, ctx when.Context) iter.Seq[command.ID] {
	return func(yield func(command.ID) bool) {
		seen := map[command.ID]bool{}
		for i, scope := range slices.Backward(p) {
			for j, e := range k.scopes[scope] {
				if seen[e.Command] || !e.holds(ctx) || k.blocked(e, ctx, p[i+1:], k.scopes[scope][:j]) {
					continue
				}
				seen[e.Command] = true
				if !yield(e.Command) {
					return
				}
			}
		}
	}
}

// blocked reports whether a binding whose when holds in ctx hides e: one
// before it in its scope with its keys, or one in the inner scopes whose
// keys are related to e's.
func (k *Keymap) blocked(e *entry, ctx when.Context, inner Path, before []*entry) bool {
	for _, o := range before {
		if o.holds(ctx) && slices.Equal(o.Keys, e.Keys) {
			return true
		}
	}
	for _, scope := range inner {
		if scope == e.Scope {
			continue
		}
		for _, o := range k.index[scope][e.Keys[0]] {
			if o.holds(ctx) && related(o.Keys, e.Keys) {
				return true
			}
		}
	}
	return false
}

// Shown is the bindings Bindings gives the sequences of, for id along a
// path, the primary first: the bindings help shows, as keymaptea's Help
// and KeyBinding do (0020-MADR A1.2). Each is a copy.
func (k *Keymap) Shown(id command.ID, p Path) []Binding {
	var out []Binding
	for _, e := range k.shownEntries(id, p) {
		out = append(out, cloneBinding(e.Binding))
	}
	return out
}

// shownEntries is the entries Bindings gives the sequences of.
func (k *Keymap) shownEntries(id command.ID, p Path) []*entry {
	var out []*entry
	for i, scope := range slices.Backward(p) {
		for j, e := range k.scopes[scope] {
			if e.Command != id || slices.ContainsFunc(out, func(o *entry) bool { return slices.Equal(o.Keys, e.Keys) }) {
				continue
			}
			if !k.hidden(e, p[i+1:], k.scopes[scope][:j]) {
				out = append(out, e)
			}
		}
	}
	return out
}

// hidden reports whether a binding with no when hides e: one before it in
// its scope with its keys, or one in the inner scopes whose keys are
// related to e's.
func (k *Keymap) hidden(e *entry, inner Path, before []*entry) bool {
	for _, o := range before {
		if !o.hasWhen && slices.Equal(o.Keys, e.Keys) {
			return true
		}
	}
	for _, scope := range inner {
		if scope == e.Scope {
			continue
		}
		for _, o := range k.index[scope][e.Keys[0]] {
			if !o.hasWhen && related(o.Keys, e.Keys) {
				return true
			}
		}
	}
	return false
}

// Conflicts is the warnings New found: each Shadow.
func (k *Keymap) Conflicts() []BindingConflict { return slices.Clone(k.conflicts) }

// Rules is the bindings in effect, in the order New took them: the
// defaults, then each layer. Keys are expanded, and a default's are the
// ones it bound.
func (k *Keymap) Rules() iter.Seq[Binding] {
	return func(yield func(Binding) bool) {
		for _, e := range k.bound {
			if !yield(cloneBinding(e.Binding)) {
				return
			}
		}
	}
}
