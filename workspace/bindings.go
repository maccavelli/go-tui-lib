package workspace

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/command/commandtea"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// The scopes the workspace's bindings and KeyPath name
// (docs/decisions/0007-MADR-keymap-engine.md §9, A11): the layout's, and
// the one on the path while any overlay is open.
const (
	scopeWorkspace command.Scope = "workspace"
	scopeOverlay   command.Scope = "overlay"
)

// The workspace's commands that its code names more than once.
const (
	cmdResizeDirection command.ID = "workspace.resize.direction"
	cmdOverlayClose    command.ID = "workspace.overlay.close"
)

// paneScope and overlayScope are a pane's and an overlay's scopes on the
// path: "pane:<id>" and "overlay:<id>".
func paneScope(id string) command.Scope    { return command.Scope("pane:" + id) }
func overlayScope(id string) command.Scope { return command.Scope("overlay:" + id) }

// KeyContexter is a pane, or an overlay's content, that names scopes of its
// own for the key path, such as "kind:editor" or "mode:vim-normal".
type KeyContexter interface {
	KeyContexts() []command.Scope
}

// RequestMsg is a matched key's command that is not the workspace's own:
// the program passes Request to its registry's Dispatch. Request has
// Origin command.OriginKey and the workspace's WhenContext
// (docs/decisions/0007-MADR-keymap-engine.md A11).
type RequestMsg struct {
	Request command.Request
}

// CommandConsumer is a pane, or an overlay's content, that runs some
// commands itself, as a picker runs its moves. A matched command that is
// not the workspace's own is offered, before it becomes a RequestMsg, to
// whoever has the keyboard: the top modal overlay alone, when one is open;
// else the open overlays, top first, then the focused pane. The first whose
// ConsumesCommand reports true is sent CommandMsg at once, so its commands
// run in the order their keys were pressed
// (docs/decisions/0008-MADR-command-palette.md A3 item 1).
type CommandConsumer interface {
	ConsumesCommand(id command.ID) bool
}

// CommandMsg is a matched key's command, delivered to the CommandConsumer
// that takes it. Request is the one a RequestMsg would carry: Origin
// command.OriginKey and the workspace's WhenContext.
type CommandMsg struct {
	Request command.Request
}

// seq is a default's keys, which are fixed: an error is a bug.
func seq(s string) keymap.Sequence {
	q, err := keymap.ParseSequence(s, nil)
	if err != nil {
		panic(err)
	}
	return q
}

// def is a default binding of keys to id in scope, with args.
func def(scope command.Scope, keys keymap.Sequence, id command.ID, args string) keymap.Default {
	b := keymap.Binding{Keys: keys, Command: id, Scope: scope, Location: keymap.Location{Layer: keymap.DefaultLayer}}
	if args != "" {
		b.Args = jsontext.Value(args)
	}
	return keymap.Default{Binding: b}
}

// DefaultRules is the workspace's default bindings, the keys v0.1.4 set
// (docs/decisions/0007-MADR-keymap-engine.md §9, A4): in the workspace
// scope, alt+. and alt+, to move focus, alt+1 to alt+9 to focus a pane of
// the ring, alt+z to zoom, and alt+shift and an arrow to resize; and esc
// to close the top overlay, in the overlay scope. None is ctrl+c, which is
// the program's. Each command's keys are listed last first, so that the
// first, such as alt+1, is the primary a help shows.
//
// A terminal without an enhanced keyboard protocol sends alt+x as ESC and
// x. When x is one of [ ] O P _ ^ X or \, that is also how a control
// sequence starts, so the key cannot be told from the sequence. No default
// uses alt with one of them, and a rebinding should not either.
func DefaultRules() []keymap.Default {
	out := []keymap.Default{
		def(scopeWorkspace, seq("alt+."), "workspace.focus.next", ""),
		def(scopeWorkspace, seq("alt+,"), "workspace.focus.prev", ""),
	}
	for n := 9; n >= 1; n-- {
		out = append(out, def(scopeWorkspace, seq(fmt.Sprintf("alt+%d", n)), "workspace.focus.index", fmt.Sprintf(`{"index":%d}`, n)))
	}
	out = append(out, def(scopeWorkspace, seq("alt+z"), "workspace.zoom", ""))
	for _, d := range []string{"down", "up", "right", "left"} {
		out = append(out, def(scopeWorkspace, seq("alt+shift+"+d), cmdResizeDirection, `{"direction":"`+d+`"}`))
	}
	return append(out, def(scopeOverlay, seq("esc"), cmdOverlayClose, ""))
}

// WithBindings sets the keymap the workspace's keys go through, in place of
// one built from DefaultRules. A program builds it with DefaultRules as
// its defaults, its own bindings, and the user's file
// (docs/decisions/0007-MADR-keymap-engine.md §9).
func WithBindings(k *keymap.Keymap) Option {
	return optionFunc(func(w *Workspace) { w.km = k })
}

// SetBindings replaces the keymap, as after keymap.Rebuild when the
// terminal reports its keyboard, and cancels a pending sequence
// (docs/decisions/0007-MADR-keymap-engine.md A11). A nil k is DefaultRules
// again.
func (w *Workspace) SetBindings(k *keymap.Keymap) {
	if k == nil {
		k = builtinKeymap()
	}
	w.km = k
	w.disp.SetKeymap(k)
}

// builtinKeymap is the keymap of DefaultRules. They are fixed, and build
// clean: an error is a bug.
func builtinKeymap() *keymap.Keymap {
	k, err := keymap.New(keymap.Settings{}, DefaultRules())
	if err != nil {
		panic(err)
	}
	return k
}

// KeyPath is the scopes a key is resolved in, outermost first
// (docs/decisions/0007-MADR-keymap-engine.md §9, A11): global;
// "workspace"; the focused pane, as "pane:<id>", and the scopes it names
// as a KeyContexter; and, while any overlay is open, "overlay" and each
// open overlay as "overlay:<id>", bottom first. While a modal overlay is
// open, it is global, "overlay", that overlay and its own scopes only, so
// the layout's bindings do not reach through it. The modes PushMode added
// come last.
func (w *Workspace) KeyPath() keymap.Path {
	p := keymap.Path{command.Global}
	if o := w.modal(); o != nil {
		p = append(p, scopeOverlay, overlayScope(o.ID))
		p = append(p, keyContexts(o.Pane)...)
	} else {
		p = append(p, scopeWorkspace)
		if pane, ok := w.panes[w.focus]; ok {
			p = append(p, paneScope(string(w.focus)))
			p = append(p, keyContexts(pane)...)
		}
		if len(w.overlays) > 0 {
			p = append(p, scopeOverlay)
			for _, o := range w.overlays {
				p = append(p, overlayScope(o.ID))
			}
		}
	}
	for _, m := range w.modes {
		p = append(p, m.scope)
	}
	return p
}

// keyContexts is p's own scopes, when it is a KeyContexter.
func keyContexts(p Pane) []command.Scope {
	if k, ok := p.(KeyContexter); ok {
		return slices.Clone(k.KeyContexts())
	}
	return nil
}

// mode is a scope PushMode added.
type mode struct {
	id    uint64
	scope command.Scope
}

// PushMode adds scope s to the end of the key path, as a dialog adds
// "mode:modal", until pop is called. pop removes that mode, and only that
// one, however many were pushed since; a second pop does nothing.
func (w *Workspace) PushMode(s command.Scope) (pop func()) {
	w.modeSeq++
	id := w.modeSeq
	w.modes = append(w.modes, mode{id: id, scope: s})
	return func() {
		w.modes = slices.DeleteFunc(w.modes, func(m mode) bool { return m.id == id })
	}
}

// own is the workspace's own commands, which a matched key runs in place
// of a registry, made on first use.
func (w *Workspace) own() map[command.ID]command.Command {
	if w.commands == nil {
		w.commands = map[command.ID]command.Command{}
		for _, c := range Commands(w) {
			w.commands[c.ID] = c
		}
	}
	return w.commands
}

// key takes a key press or release through the dispatcher along KeyPath:
// a matched command runs; a key the dispatcher does not handle, and the
// keys a sequence replays, go to the top modal overlay or the focused
// pane. ok is false for a release the dispatcher does not handle, which
// Update broadcasts as before.
func (w *Workspace) key(msg tea.Msg) (cmd tea.Cmd, ok bool) {
	_, release := msg.(tea.KeyReleaseMsg)
	ctx := w.WhenContext()
	o, tick := w.disp.Update(msg, w.KeyPath(), ctx)
	cmds := []tea.Cmd{tick}
	for _, r := range o.Replay {
		cmds = append(cmds, w.deliver(r))
	}
	switch {
	case o.Command != "":
		cmds = append(cmds, w.run(o, msg, ctx))
	case !o.Handled && release:
		return tea.Batch(cmds...), len(o.Replay) > 0
	case !o.Handled:
		cmds = append(cmds, w.deliver(msg))
	}
	return tea.Batch(cmds...), true
}

// deliver gives an unmatched key to whoever has the keyboard: the top modal
// overlay, or the focused pane.
func (w *Workspace) deliver(msg tea.Msg) tea.Cmd {
	if w.modal() != nil {
		return w.updateOverlay(len(w.overlays)-1, msg)
	}
	return w.Send(w.focus, msg)
}

// run runs a matched command: the workspace's own through its handler,
// with an error as a command.ResultMsg; any other through the
// CommandConsumer that takes it, or else as a RequestMsg for the program.
// workspace.overlay.close's key goes to a top overlay that is an
// EscConsumer consuming esc, instead.
func (w *Workspace) run(o keymap.Outcome, msg tea.Msg, ctx when.Context) tea.Cmd {
	if n := len(w.overlays); o.Command == cmdOverlayClose && n > 0 {
		if c, ok := w.overlays[n-1].Pane.(EscConsumer); ok && c.ConsumesEsc() {
			return w.updateOverlay(n-1, msg)
		}
	}
	req := command.Request{ID: o.Command, Args: o.Args, Origin: command.OriginKey, WhenContext: ctx}
	c, ok := w.own()[o.Command]
	if !ok {
		if cmd, taken := w.offer(req); taken {
			return cmd
		}
		return func() tea.Msg { return RequestMsg{Request: req} }
	}
	res, err := c.Handler.Run(context.Background(), &command.Invocation{
		Command: c, Args: o.Args, Origin: command.OriginKey, WhenContext: ctx,
	})
	if err != nil {
		return func() tea.Msg { return command.ResultMsg{Request: req, Err: err} }
	}
	return commandtea.Cmd(res.Cmd)
}

// offer sends req as a CommandMsg to the first CommandConsumer that takes
// it, in CommandConsumer's order, and reports whether one did.
func (w *Workspace) offer(req command.Request) (tea.Cmd, bool) {
	msg := CommandMsg{Request: req}
	if w.modal() != nil {
		n := len(w.overlays) - 1
		if consumes(w.overlays[n].Pane, req.ID) {
			return w.updateOverlay(n, msg), true
		}
		return nil, false
	}
	for i, o := range slices.Backward(w.overlays) {
		if consumes(o.Pane, req.ID) {
			return w.updateOverlay(i, msg), true
		}
	}
	if p, ok := w.panes[w.focus]; ok && consumes(p, req.ID) {
		return w.Send(w.focus, msg), true
	}
	return nil, false
}

// consumes reports whether p is a CommandConsumer that takes id.
func consumes(p Pane, id command.ID) bool {
	c, ok := p.(CommandConsumer)
	return ok && c.ConsumesCommand(id)
}
