// Package workspace hosts panes in a Bubble Tea program: it lays them out
// with the layout package, draws their chrome, routes keys and mouse events,
// moves focus, resizes, zooms and hides panes, opens overlays, and places
// the terminal cursor for the focused pane
// (docs/decisions/0002-MADR-multi-pane-workspace-layouts.md §3).
//
// The program owns the program. A Workspace is not a tea.Model: the
// program's Update calls Workspace.Update, and its View returns
// Workspace.View, or builds a tea.View from Render and Cursor. The program, never the workspace, sets the
// alternate screen, the mouse mode, focus reporting and keyboard
// enhancements, and ctrl+c is never bound here
// (docs/decisions/0001-MADR-scaffold-charm-tui-library.md §6, rules 1 and 2).
// Mouse handling is built in and stays inert until the program turns a mouse
// mode on.
//
// A pane needs only Update and View. Optional interfaces, found by type
// assertion, add a title, a badge, focus callbacks, a minimum size, a
// cursor, help bindings and change tracking.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package workspace

import (
	"iter"
	"maps"
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/internal/cells"
	"github.com/maccavelli/go-tui-lib/internal/limits"
	"github.com/maccavelli/go-tui-lib/layout"
	"github.com/maccavelli/go-tui-lib/theme"
)

// Pane is a component the workspace hosts. View renders it into exactly
// width × height cells; anything larger is clipped.
//
// The workspace redraws a frame only when something changed: its size,
// layout, state, focus or overlays, or a message delivered to a pane. A
// pane's view therefore changes through its Update, or, for a Changer, when
// Changed reports it. A pane changed some other way, such as by the program
// writing to a pointer pane directly, is drawn anew at the next of those
// (docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md §2).
type Pane interface {
	Update(msg tea.Msg) (Pane, tea.Cmd)
	View(width, height int) string
}

// Titled gives a pane a title for its chrome. Without it the pane's ID is
// shown.
type Titled interface{ Title() string }

// Badged gives a pane a badge in its chrome, such as an unread count. An
// empty badge is not drawn.
type Badged interface{ Badge() string }

// Focuser is told when a pane gains and loses the keyboard, before the
// pane's Update gets PaneFocusMsg or PaneBlurMsg. Its methods change the
// value the workspace holds, so only a pointer pane keeps the change. A
// pane implements Focuser or handles the messages, not both; a pane with
// value semantics, such as a bubbles model, handles the messages, or is
// hosted with Wrap.
type Focuser interface {
	Focus() tea.Cmd
	Blur()
}

// Focusable lets a pane refuse focus, such as a status footer.
type Focusable interface{ Focusable() bool }

// Sizer gives a pane a minimum content size. The workspace raises the
// pane's layout minimum to fit it, chrome included.
type Sizer interface{ MinSize() (width, height int) }

// Cursorer places the terminal cursor while the pane is focused, in the
// pane's own cells. A nil cursor hides it.
type Cursorer interface{ Cursor() *tea.Cursor }

// KeyMapper lists a pane's bindings, for a help footer.
type KeyMapper interface{ Keys() []key.Binding }

// Changer reports whether a pane's view changed since it was last drawn.
// A pane reporting false at the same size and focus is not drawn again.
type Changer interface{ Changed() bool }

// EscConsumer is an overlay pane that handles esc itself; without it, esc
// closes the overlay.
type EscConsumer interface{ ConsumesEsc() bool }

// SizeMsg tells a pane the size of its content area. It is sent through the
// pane's Update whenever that size changes.
type SizeMsg struct{ Width, Height int }

// Chrome is what the workspace draws around panes.
type Chrome uint8

const (
	// Borders draws a border with the title in its top edge around each
	// pane. Use layout.WithGap(0) with it, or the gaps stay blank.
	Borders Chrome = iota
	// Separators draws a title row at the top of each pane and separator
	// lines in the layout's gaps.
	Separators
	// None draws nothing: each pane gets its whole rectangle.
	None
)

// fallback is the size used before the first tea.WindowSizeMsg, so the
// first frame is not empty.
const fallbackWidth, fallbackHeight = 80, 24

// Workspace hosts panes. Its zero value is not usable; call New.
//
// A Workspace is not safe for concurrent use. Bubble Tea calls a program's
// Update and View on one goroutine, and the program calls the workspace
// from there; the commands it returns run elsewhere, and reach it only as
// messages to Update.
type Workspace struct {
	root     layout.Node
	panes    map[layout.PaneID]Pane
	state    layout.State
	theme    theme.Theme
	keys     KeyMap
	chrome   Chrome
	chromes  map[layout.PaneID]Chrome
	border   theme.BorderStyle
	ring     []layout.PaneID
	mouse    bool
	width    int
	height   int
	plan     layout.Plan
	err      error
	focus    layout.PaneID
	overlays []Overlay
	sizes    map[layout.PaneID]SizeMsg
	osizes   map[string]SizeMsg // each open overlay's last content size
	cache    map[slotKey]cachedView
	ids      []layout.PaneID   // every pane's ID, sorted, for Broadcast
	frame    *cells.Frame      // the reused frame buffer
	method   ansi.Method       // how the frame and its views measure
	pinned   bool              // WithWidthMethod fixed method
	themeGen uint64            // raised on every theme change
	follow   bool              // rebuild the theme on profile and background messages
	builder  ThemeBuilder      // how to rebuild it; nil is theme.New with the glyphs in use
	gbuilder GlyphThemeBuilder // how to rebuild it, given the glyphs; wins over builder
	themed   bool              // WithTheme gave the first theme
	glyphs   glyph.Set         // the starting glyphs, for the first theme
	profile  colorprofile.Profile
	bg       theme.Background // the background the terminal last reported
	pinnedBg theme.Background // SetBackground's choice; Unknown follows bg
	noQuery  bool             // WithoutBackgroundQuery
	regions  []region         // the last frame's regions, top first, for hit testing
	last     string           // the last frame
	dirty    bool             // something changed since the last frame
	drag     *drag
}

// viewKind tells a pane's cached view from an overlay's: their IDs are
// separate namespaces, so one ID may name both.
type viewKind uint8

const (
	paneView viewKind = iota
	overlayView
)

// slotKey names a cache slot: whose view it holds, and whether it was drawn
// focused. A pane or an overlay has at most two slots, so the cache holds at
// most 2 × (panes + overlays) views
// (docs/decisions/0014-PLAN-hardening.md Step 3, finding H1).
type slotKey struct {
	kind    viewKind
	id      string
	focused bool
}

// cachedView is a slot's view, and everything it was drawn under
// (docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md §2 and
// amendment A1, Q5). A hit needs every field to equal the request; a miss
// replaces the slot.
type cachedView struct {
	width, height int
	method        ansi.Method
	themeGen      uint64
	view          string
}

type drag struct {
	sep  layout.Separator
	last int
}

// Option configures a Workspace.
type Option func(*Workspace)

// ThemeBuilder makes the theme for a colour profile and a background. It
// chooses its own glyphs.
type ThemeBuilder func(p colorprofile.Profile, bg theme.Background) theme.Theme

// GlyphThemeBuilder makes the theme for a colour profile, a background and
// the glyphs in use: WithGlyphs's at first, then the theme's own.
type GlyphThemeBuilder func(p colorprofile.Profile, bg theme.Background, g glyph.Set) theme.Theme

// WithTheme fixes the theme: profile and background messages do not change
// it, though they still reach the panes. Without it, the workspace follows
// the terminal: it starts with ANSI256, an unknown background and Unicode
// glyphs, or what WithProfile, WithBackground and WithGlyphs say, and
// rebuilds the theme on each tea.ColorProfileMsg and tea.BackgroundColorMsg
// (docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md §4).
//
// WithTheme's theme is the first theme, whatever the options' order, and
// its profile, background and glyphs are the starting ones: WithProfile,
// WithBackground and WithGlyphs are then ignored. A WithThemeBuilder or
// WithGlyphThemeBuilder after it makes the workspace follow from there
// (docs/decisions/0014-PLAN-component-native-forms.md D5).
func WithTheme(t theme.Theme) Option {
	return func(w *Workspace) { w.theme, w.follow, w.themed = t, false, true }
}

// WithThemeBuilder makes the workspace follow the terminal, building its
// theme with b for the starting profile and background, and again on each
// profile and background message. The default builder is theme.New with
// the glyphs in use; a program with its own palettes passes one that adds
// them, such as with theme.WithPaletteFor. b chooses its own glyphs, so
// WithGlyphs does not reach it; a GlyphThemeBuilder is given them.
func WithThemeBuilder(b ThemeBuilder) Option {
	return func(w *Workspace) { w.builder, w.follow = b, true }
}

// WithGlyphThemeBuilder is WithThemeBuilder for a builder that is given the
// glyphs in use. It wins over a WithThemeBuilder.
func WithGlyphThemeBuilder(b GlyphThemeBuilder) Option {
	return func(w *Workspace) { w.gbuilder, w.follow = b, true }
}

// WithGlyphs sets the starting glyphs (default Unicode), for the default
// builder and a GlyphThemeBuilder, so a program's own command line can pass
// launch's choice of glyph tier.
func WithGlyphs(g glyph.Set) Option { return func(w *Workspace) { w.glyphs = g } }

// WithProfile sets the starting colour profile (default ANSI256). A
// tea.ColorProfileMsg replaces it.
func WithProfile(p colorprofile.Profile) Option { return func(w *Workspace) { w.profile = p } }

// WithBackground sets the starting background (default Unknown). A
// tea.BackgroundColorMsg replaces it; SetBackground is the way to pin one.
func WithBackground(bg theme.Background) Option { return func(w *Workspace) { w.bg = bg } }

// WithSize sets the size the workspace lays out at before its first
// tea.WindowSizeMsg (default 80 by 24), such as a plain render's. It is
// clamped as a WindowSizeMsg is.
func WithSize(width, height int) Option {
	return func(w *Workspace) { w.width, w.height = clampSize(width, height) }
}

// The largest window the workspace draws
// (docs/decisions/0014-PLAN-hardening.md Step 4, finding H3). A larger
// tea.WindowSizeMsg, or WithSize, is clamped: each side to MaxSide, then
// the height until the area is at most MaxCells, keeping the width, and
// the frame draws from the top left. The constants change only by a
// record.
const (
	// MaxSide is the most cells the window has on either side.
	MaxSide = limits.MaxSide
	// MaxCells is the most cells the window has in all: 524,288, about
	// 56 MiB of frame.
	MaxCells = limits.MaxCells
)

// clampSize is the size w × h is drawn at.
func clampSize(w, h int) (int, int) { return limits.Clamp(w, h) }

// WithoutBackgroundQuery leaves tea.RequestBackgroundColor out of Init, for
// a program that runs its own probe or must send nothing unasked. By default
// Init asks, so a following theme learns whether the background is light or
// dark.
func WithoutBackgroundQuery() Option { return func(w *Workspace) { w.noQuery = true } }

// WithKeyMap replaces the workspace's bindings.
func WithKeyMap(k KeyMap) Option { return func(w *Workspace) { w.keys = k } }

// WithChrome sets what is drawn around panes (default Borders).
func WithChrome(c Chrome) Option { return func(w *Workspace) { w.chrome = c } }

// WithPaneChrome overrides the chrome for one pane, such as None for a
// one-row status footer.
func WithPaneChrome(id layout.PaneID, c Chrome) Option {
	return func(w *Workspace) {
		if w.chromes == nil {
			w.chromes = map[layout.PaneID]Chrome{}
		}
		w.chromes[id] = c
	}
}

// WithBorder sets the border style for Borders chrome and overlays (default
// rounded).
func WithBorder(s theme.BorderStyle) Option { return func(w *Workspace) { w.border = s } }

// WithFocusRing sets the order focus moves in (default: tree order).
func WithFocusRing(ids ...layout.PaneID) Option {
	return func(w *Workspace) { w.ring = slices.Clone(ids) }
}

// WithState restores a saved layout state.
func WithState(s layout.State) Option { return func(w *Workspace) { w.state = s } }

// WithoutMouse turns mouse handling off. It is on by default, and does
// nothing until the program sets a mouse mode.
func WithoutMouse() Option { return func(w *Workspace) { w.mouse = false } }

// WithMouse turns mouse handling on or off.
//
// Deprecated: use WithoutMouse, since handling is on by default (0014-MADR W4).
func WithMouse(on bool) Option { return func(w *Workspace) { w.mouse = on } }

// WithWidthMethod fixes how the workspace measures text, and stops it
// following the terminal's mode 2027 report. By default it measures with
// ansi.WcWidth, as Bubble Tea's renderer starts, and switches to
// ansi.GraphemeWidth when the terminal reports grapheme clustering, as
// Bubble Tea does (docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md
// §3). WithWidthMethod(ansi.GraphemeWidth) restores v0.1's measurement.
func WithWidthMethod(m ansi.Method) Option {
	return func(w *Workspace) { w.method, w.pinned = m, true }
}

// WithFocus sets the pane focused first (default: the first of the ring).
func WithFocus(id layout.PaneID) Option { return func(w *Workspace) { w.focus = id } }

// New hosts panes in root. Panes the tree never places are kept, and can be
// shown by a later SetLayout.
func New(root layout.Node, panes map[layout.PaneID]Pane, opts ...Option) *Workspace {
	w := &Workspace{
		root:   root,
		panes:  make(map[layout.PaneID]Pane, len(panes)),
		keys:   DefaultKeyMap(),
		border: theme.BorderRounded,
		mouse:  true,
		width:  fallbackWidth,
		height: fallbackHeight,
		sizes:  map[layout.PaneID]SizeMsg{},
		osizes: map[string]SizeMsg{},
		cache:  map[slotKey]cachedView{},
		method: ansi.WcWidth,
		dirty:  true,
		follow: true,

		profile: colorprofile.ANSI256,
		bg:      theme.Unknown,
		glyphs:  glyph.Unicode(),
	}
	maps.Copy(w.panes, panes)
	w.ids = slices.Sorted(maps.Keys(w.panes))
	for _, o := range opts {
		o(w)
	}
	// The first theme is built once every option has applied
	// (docs/decisions/0014-PLAN-component-native-forms.md Step 6).
	if w.themed {
		w.profile, w.bg, w.glyphs = w.theme.Profile, w.theme.Background, w.theme.Glyphs
	} else {
		w.theme = w.build(w.glyphs)
	}
	w.solve()
	if !w.focusable(w.focus) {
		w.focus = ""
		if r := w.focusRing(); len(r) > 0 {
			w.focus = r[0]
		}
	}
	return w
}

// Init returns the commands a program runs first, from its own Init: each
// placed pane's first SizeMsg, and PaneFocusMsg to the pane with the
// keyboard.
func (w *Workspace) Init() tea.Cmd {
	cmds := []tea.Cmd{w.resolve(), w.gain(w.focusTarget())}
	if !w.noQuery {
		cmds = append(cmds, tea.RequestBackgroundColor)
	}
	return tea.Batch(cmds...)
}

// SetTheme replaces the theme. Every cached view misses, since its key holds
// the theme's generation, and the next frame is drawn anew. A following
// workspace rebuilds over it at the next profile or background message.
func (w *Workspace) SetTheme(t theme.Theme) tea.Cmd {
	w.theme = t
	w.themeGen++
	w.dirty = true
	return nil
}

// rebuildTheme makes the theme again for the current profile and
// background, when the workspace follows the terminal.
func (w *Workspace) rebuildTheme() {
	if !w.follow {
		return
	}
	w.SetTheme(w.build(w.theme.Glyphs))
}

// build makes a theme for the current profile and background with the
// builder in use: the GlyphThemeBuilder, given g; else the ThemeBuilder;
// else theme.New with g.
func (w *Workspace) build(g glyph.Set) theme.Theme {
	bg := w.background()
	switch {
	case w.gbuilder != nil:
		return w.gbuilder(w.profile, bg, g)
	case w.builder != nil:
		return w.builder(w.profile, bg)
	}
	return theme.New(w.profile, bg, g)
}

// Err returns the error of the last layout, if any. The workspace keeps the
// previous plan when a layout fails.
func (w *Workspace) Err() error { return w.err }

// State returns the layout state, for the program to persist.
func (w *Workspace) State() layout.State { return w.state }

// WidthMethod is how the workspace measures text now, so a pane can
// measure the same way.
func (w *Workspace) WidthMethod() ansi.Method { return w.method }

// setMethod changes how the workspace measures. Every cached view misses,
// since its key holds the method, and the next frame is drawn anew.
func (w *Workspace) setMethod(m ansi.Method) {
	if m != w.method {
		w.method = m
		w.dirty = true
	}
}

// followMode switches to grapheme widths on the terminal's mode 2027 report,
// under the rule Bubble Tea's renderer follows (bubbletea v2.0.10
// tea.go:802-805), unless WithWidthMethod pinned the method.
func (w *Workspace) followMode(m tea.ModeReportMsg) {
	if w.pinned || m.Mode != ansi.ModeUnicodeCore {
		return
	}
	switch m.Value {
	case ansi.ModeReset, ansi.ModeSet, ansi.ModePermanentlySet:
		w.setMethod(ansi.GraphemeWidth)
	}
}

// Focused returns the focused pane's ID.
func (w *Workspace) Focused() layout.PaneID { return w.focus }

// Plan returns the current layout plan.
func (w *Workspace) Plan() layout.Plan { return w.plan }

// Pane returns the pane with id.
func (w *Workspace) Pane(id layout.PaneID) (Pane, bool) {
	p, ok := w.panes[id]
	return p, ok
}

// SetLayout replaces the layout tree.
func (w *Workspace) SetLayout(root layout.Node) tea.Cmd {
	w.root = root
	return w.resolve()
}

// SetPane adds or replaces a pane. A replaced pane's cached view is dropped,
// and the new pane is told its size.
func (w *Workspace) SetPane(id layout.PaneID, p Pane) tea.Cmd {
	if _, ok := w.panes[id]; !ok {
		i, _ := slices.BinarySearch(w.ids, id)
		w.ids = slices.Insert(w.ids, i, id)
	}
	w.panes[id] = p
	delete(w.sizes, id)
	w.forget(paneView, string(id))
	return w.resolve()
}

// forget drops the cached views of the pane or overlay id: its two slots.
func (w *Workspace) forget(kind viewKind, id string) {
	delete(w.cache, slotKey{kind, id, false})
	delete(w.cache, slotKey{kind, id, true})
}

// SetState replaces the layout state.
func (w *Workspace) SetState(s layout.State) tea.Cmd {
	w.state = s
	return w.resolve()
}

// Update handles a message: sizes, keys, mouse events, targeted messages,
// and anything else, which reaches every pane. A tea.WindowSizeMsg is
// clamped to MaxSide and MaxCells.
func (w *Workspace) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		w.width, w.height = clampSize(m.Width, m.Height)
		return w.resolve()
	case targeted:
		return w.Send(m.id, m.msg)
	case tea.KeyPressMsg:
		return w.key(m)
	case tea.PasteMsg, tea.PasteStartMsg, tea.PasteEndMsg:
		if o := w.modal(); o != nil {
			return w.updateOverlay(len(w.overlays)-1, msg)
		}
		return w.Send(w.focus, msg)
	case tea.MouseMsg:
		if !w.mouse {
			return nil
		}
		return w.mouseEvent(m)
	case tea.ModeReportMsg:
		w.followMode(m) // and the panes still get it
	case tea.ColorProfileMsg:
		w.profile = m.Profile
		w.rebuildTheme()
	case tea.BackgroundColorMsg:
		if m.Color != nil {
			w.bg = theme.FromDark(m.IsDark())
			if w.pinnedBg == theme.Unknown {
				w.rebuildTheme()
			}
		}
	}
	return w.Broadcast(msg)
}

type targeted struct {
	id  layout.PaneID
	msg tea.Msg
}

// To wraps msg so that Update delivers it to pane id only.
func To(id layout.PaneID, msg tea.Msg) tea.Msg { return targeted{id: id, msg: msg} }

// Send delivers msg to pane id now, and returns its command. When no pane
// has that ID, it tries an open overlay with it, as SendOverlay does.
func (w *Workspace) Send(id layout.PaneID, msg tea.Msg) tea.Cmd {
	p, ok := w.panes[id]
	if !ok {
		return w.SendOverlay(string(id), msg)
	}
	next, cmd := p.Update(msg)
	if next != nil {
		w.panes[id] = next
		p = next
	}
	w.touched(p)
	return cmd
}

// touched marks the frame dirty after a message reached p, unless p is a
// Changer, which Render asks instead.
func (w *Workspace) touched(p Pane) {
	if _, ok := p.(Changer); !ok {
		w.dirty = true
	}
}

// Broadcast delivers msg to every pane and overlay, and batches their
// commands.
func (w *Workspace) Broadcast(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for _, id := range w.ids {
		cmds = append(cmds, w.Send(id, msg))
	}
	for i := range w.overlays {
		cmds = append(cmds, w.updateOverlay(i, msg))
	}
	return tea.Batch(cmds...)
}

func (w *Workspace) key(m tea.KeyPressMsg) tea.Cmd {
	if n := len(w.overlays); n > 0 {
		top := w.overlays[n-1]
		if key.Matches(m, w.keys.Close) {
			if c, ok := top.Pane.(EscConsumer); !ok || !c.ConsumesEsc() {
				return w.Pop()
			}
			return w.updateOverlay(n-1, m)
		}
		if top.Modal {
			return w.updateOverlay(n-1, m)
		}
	}
	switch {
	case key.Matches(m, w.keys.FocusNext):
		return w.cycle(1)
	case key.Matches(m, w.keys.FocusPrev):
		return w.cycle(-1)
	case key.Matches(m, w.keys.Zoom):
		return w.Zoom(w.focus)
	case key.Matches(m, w.keys.ResizeLeft):
		return w.resizeFocused(layout.Horizontal, -1)
	case key.Matches(m, w.keys.ResizeRight):
		return w.resizeFocused(layout.Horizontal, 1)
	case key.Matches(m, w.keys.ResizeUp):
		return w.resizeFocused(layout.Vertical, -1)
	case key.Matches(m, w.keys.ResizeDown):
		return w.resizeFocused(layout.Vertical, 1)
	}
	for i := range w.keys.FocusPane {
		if key.Matches(m, w.keys.FocusPane[i]) {
			if r := w.focusRing(); i < len(r) {
				return w.Focus(r[i])
			}
			return nil
		}
	}
	return w.Send(w.focus, m)
}

// Focus moves focus to pane id, if it is visible and focusable. The pane
// that had the keyboard is sent PaneBlurMsg, and id PaneFocusMsg. While a
// modal overlay is open it keeps the keyboard, and id is told when the
// overlay closes.
func (w *Workspace) Focus(id layout.PaneID) tea.Cmd {
	if id == w.focus || !w.focusable(id) {
		return nil
	}
	was := w.focusTarget()
	w.focus = id
	w.dirty = true
	return w.moveFocus(was)
}

// FocusNext moves focus to the next pane of the ring.
func (w *Workspace) FocusNext() tea.Cmd { return w.cycle(1) }

// FocusPrev moves focus to the previous pane of the ring.
func (w *Workspace) FocusPrev() tea.Cmd { return w.cycle(-1) }

func (w *Workspace) cycle(step int) tea.Cmd {
	r := w.focusRing()
	if len(r) == 0 {
		return nil
	}
	i := slices.Index(r, w.focus)
	if i < 0 {
		return w.Focus(r[0])
	}
	return w.Focus(r[(i+step+len(r))%len(r)])
}

func (w *Workspace) focusRing() []layout.PaneID {
	var out []layout.PaneID
	if len(w.ring) > 0 {
		for _, id := range w.ring {
			if w.focusable(id) {
				out = append(out, id)
			}
		}
		return out
	}
	for id := range w.plan.All() {
		if w.focusable(id) {
			out = append(out, id)
		}
	}
	return out
}

// Panes yields the panes of the focus ring, the panes focus moves through,
// in its order: WithFocusRing's, or else the tree's
// (docs/decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md §6).
func (w *Workspace) Panes() iter.Seq2[layout.PaneID, Pane] {
	return func(yield func(layout.PaneID, Pane) bool) {
		for _, id := range w.focusRing() {
			if !yield(id, w.panes[id]) {
				return
			}
		}
	}
}

// PaneAs returns pane id as T, or, when no pane has that ID, the open
// overlay id's pane, as Send finds them. It returns false when neither is
// there or it is not a T. It is a generic method, which Go 1.27 allows.
func (w *Workspace) PaneAs[T Pane](id layout.PaneID) (T, bool) {
	p, ok := w.panes[id]
	if !ok {
		i := w.overlayIndex(string(id))
		if i < 0 {
			var zero T
			return zero, false
		}
		p = w.overlays[i].Pane
	}
	t, ok := p.(T)
	return t, ok
}

func (w *Workspace) focusable(id layout.PaneID) bool {
	p, ok := w.panes[id]
	if !ok {
		return false
	}
	if _, placed := w.plan.Panes[id]; !placed {
		return false
	}
	f, ok := p.(Focusable)
	return !ok || f.Focusable()
}

// Zoom gives pane id the whole area, or restores the layout when id is
// already zoomed or empty.
func (w *Workspace) Zoom(id layout.PaneID) tea.Cmd {
	if id == "" || w.state.Zoom == id {
		w.state = w.state.WithZoom("")
	} else {
		w.state = w.state.WithZoom(id)
	}
	return w.resolve()
}

// Toggle hides pane id, or shows it if hidden.
func (w *Workspace) Toggle(id layout.PaneID) tea.Cmd {
	w.state = w.state.WithHidden(id, !w.state.IsHidden(id))
	return w.resolve()
}

// Resize moves separator sep by delta cells from where it is shown (see
// layout.State.Resize), as far as the panes' bounds allow. It stores the
// delta the layout applied, not the one asked for, so a held key or a drag
// past a limit moves back at once. It does nothing when the separator does
// not move: at a limit, or when it is not a Resizable separator of the
// current plan, since the layout applies nothing to one. A window resize
// never changes what Resize stored, so the layout comes back when the
// window grows.
func (w *Workspace) Resize(sep string, delta int) tea.Cmd {
	shown := w.plan.Resize[sep]
	want := w.state.WithResize(sep, shown+delta-w.state.Resize[sep])
	plan, err := layout.Solve(w.withMinimums(w.root), layout.Rect{W: w.width, H: w.height}, want)
	if err != nil {
		return nil
	}
	applied := plan.Resize[sep]
	if applied == shown {
		return nil
	}
	w.state = w.state.WithResize(sep, applied-w.state.Resize[sep])
	return w.resolve()
}

// resizeFocused moves the Resizable separator on the focused pane's
// trailing edge along axis, or its leading edge when it has none.
func (w *Workspace) resizeFocused(axis layout.Axis, delta int) tea.Cmd {
	r, ok := w.plan.Panes[w.focus]
	if !ok {
		return nil
	}
	var lead, trail *layout.Separator
	for i := range w.plan.Separators {
		s := &w.plan.Separators[i]
		if s.Axis != axis || !s.Resizable {
			continue
		}
		if axis == layout.Horizontal && s.Rect.Y <= r.Y && s.Rect.Y+s.Rect.H >= r.Y+r.H {
			switch {
			case s.Rect.X == r.X+r.W:
				trail = s
			case s.Rect.X+s.Rect.W == r.X:
				lead = s
			}
		}
		if axis == layout.Vertical && s.Rect.X <= r.X && s.Rect.X+s.Rect.W >= r.X+r.W {
			switch {
			case s.Rect.Y == r.Y+r.H:
				trail = s
			case s.Rect.Y+s.Rect.H == r.Y:
				lead = s
			}
		}
	}
	sep := trail
	if sep == nil {
		sep = lead
	}
	if sep == nil {
		return nil
	}
	return w.Resize(sep.ID, delta)
}

// solve lays the tree out for the current size, keeping the last plan on
// error.
func (w *Workspace) solve() {
	plan, err := layout.Solve(w.withMinimums(w.root), layout.Rect{W: w.width, H: w.height}, w.state)
	w.err = err
	if err == nil {
		w.plan = plan
	}
}

// resolve re-solves the layout, tells each pane and each open overlay whose
// content size changed, and moves focus off a pane that is no longer shown.
func (w *Workspace) resolve() tea.Cmd {
	w.dirty = true
	w.solve()
	var cmds []tea.Cmd
	for id, r := range w.plan.All() {
		in := w.content(id, r)
		sz := SizeMsg{Width: in.W, Height: in.H}
		if old, ok := w.sizes[id]; ok && old == sz {
			continue
		}
		w.sizes[id] = sz
		cmds = append(cmds, w.Send(id, sz))
	}
	for i, o := range w.overlays {
		sz := w.overlayContent(o)
		if old, ok := w.osizes[o.ID]; ok && old == sz {
			continue
		}
		w.osizes[o.ID] = sz
		cmds = append(cmds, w.updateOverlay(i, sz))
	}
	if !w.focusable(w.focus) {
		if r := w.focusRing(); len(r) > 0 {
			was := w.focusTarget()
			w.focus = r[0]
			cmds = append(cmds, w.moveFocus(was))
		}
	}
	return tea.Batch(cmds...)
}

// chromeOf is pane id's chrome: its override, or the workspace's.
func (w *Workspace) chromeOf(id layout.PaneID) Chrome {
	if c, ok := w.chromes[id]; ok {
		return c
	}
	return w.chrome
}

// content is pane id's content area in r, under its chrome.
func (w *Workspace) content(id layout.PaneID, r layout.Rect) layout.Rect {
	switch w.chromeOf(id) {
	case Borders:
		return insetBorder(r)
	case Separators:
		if r.H < 1 {
			return layout.Rect{X: r.X, Y: r.Y}
		}
		return layout.Rect{X: r.X, Y: r.Y + 1, W: r.W, H: r.H - 1}
	}
	return r
}

// insetBorder is r less a one-cell border, or an empty rectangle when r
// has no room inside one.
func insetBorder(r layout.Rect) layout.Rect {
	if r.W < 2 || r.H < 2 {
		return layout.Rect{X: r.X, Y: r.Y}
	}
	return layout.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
}

// chromeSize is what pane id's chrome adds to its content, in cells.
func (w *Workspace) chromeSize(id layout.PaneID) (int, int) {
	switch w.chromeOf(id) {
	case Borders:
		return 2, 2
	case Separators:
		return 0, 1
	}
	return 0, 0
}

// withMinimums raises the layout minimum of every Sizer pane that is a direct
// child of a built-in split. Other nodes are kept as they are.
func (w *Workspace) withMinimums(n layout.Node) layout.Node {
	switch v := n.(type) {
	case layout.Split:
		out := v
		out.Children = make([]layout.Child, len(v.Children))
		for i, c := range v.Children {
			c.Node = w.withMinimums(c.Node)
			if p, ok := c.Node.(layout.Pane); ok {
				if s, ok := w.panes[p.ID].(Sizer); ok {
					mw, mh := s.MinSize()
					cw, ch := w.chromeSize(p.ID)
					need := mw + cw
					if v.Axis == layout.Vertical {
						need = mh + ch
					}
					c.Size.Min = max(c.Size.Min, need)
					if c.Size.Max > 0 && c.Size.Max < c.Size.Min {
						c.Size.Max = c.Size.Min
					}
				}
			}
			out.Children[i] = c
		}
		return out
	case layout.Responsive:
		out := layout.Responsive{Else: w.withMinimums(v.Else)}
		for _, r := range v.Rules {
			out.Rules = append(out.Rules, layout.Rule{When: r.When, Use: w.withMinimums(r.Use)})
		}
		return out
	}
	return n
}
