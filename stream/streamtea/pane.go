package streamtea

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/frame"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/theme"
	"github.com/maccavelli/go-tui-lib/v2/workspace"
)

// DocPane is a stream.Doc in a workspace pane. NewDocPane builds one. Like
// the workspace, it is used on the program's event loop only.
//
// It draws the Doc's rows at the pane's width, less one column on the
// right, where the theme's ScrollUp and ScrollDown glyphs mark that rows
// continue above or below. A row wider than its column is cut, measured
// with the workspace's width method, and ends with the theme's Ellipsis. It
// gives the Doc its theme's ID as its revision, so a Renderer or a
// Highlighter that draws with the theme draws again when it changes.
type DocPane struct {
	w             *workspace.Workspace
	d             *stream.Doc
	cfg           config
	width, height int  // the pane's, from SizeMsg
	top           int  // the first row shown, while not following
	follow        bool // the last rows are shown, however many come
	cache         rows
	drawn         drawn
	literal       uint64 // the last revision given for a theme of ID 0
}

// literalRevs is the revisions a DocPane gives its Doc under a theme of ID
// 0, one each draw, from where no theme's ID reaches: theme.New counts from
// 1.
const literalRevs = 1 << 63

// rowsKey is what a pane's rows were made under.
type rowsKey struct {
	version uint64
	width   int
	method  ansi.Method
	theme   uint64
}

// rows is the Doc's rows, fitted to their column, under key.
type rows struct {
	key  rowsKey
	rows []string
}

// drawn is what the last View drew, which Changed compares with now.
type drawn struct {
	ok                 bool
	version            uint64
	width, height, top int
	follow             bool
}

// wheelMsg is a mouse message that carries a wheel's notches, as
// inputfiltertea.CoalescedWheelMsg does.
type wheelMsg interface {
	tea.MouseMsg
	WheelDelta() (dx, dy int)
}

// NewDocPane is a pane over d in w, which it reads at each draw for its
// theme and width method; a program builds w first, then puts the pane in
// it with Workspace.SetPane.
func NewDocPane(w *workspace.Workspace, d *stream.Doc, o ...Option) *DocPane {
	p := &DocPane{w: w, d: d, cfg: newConfig(o)}
	p.follow = !p.cfg.noFollow
	return p
}

// Update takes the pane's size; on a frame.DueMsg, its Buffer's lines, as
// its pacer releases them; its commands; and the wheel, a row a notch,
// from a tea.MouseWheelMsg or any mouse message with WheelDelta.
func (p *DocPane) Update(msg tea.Msg) (workspace.Pane, tea.Cmd) {
	switch m := msg.(type) {
	case workspace.SizeMsg:
		p.width, p.height = m.Width, m.Height
	case frame.DueMsg:
		if p.cfg.buf != nil {
			p.d.Pull(p.cfg.buf, p.cfg.pacer, m.At)
		}
	case workspace.CommandMsg:
		p.command(m.Request.ID)
	case tea.MouseWheelMsg:
		switch m.Button {
		case tea.MouseWheelUp:
			p.scroll(-1)
		case tea.MouseWheelDown:
			p.scroll(1)
		}
	case wheelMsg:
		_, dy := m.WheelDelta()
		p.scroll(dy)
	}
	return p, nil
}

// command runs one of the pane's own commands.
func (p *DocPane) command(id command.ID) {
	switch id {
	case ScrollUpID:
		p.scroll(-1)
	case ScrollDownID:
		p.scroll(1)
	case PageUpID:
		p.scroll(-p.height)
	case PageDownID:
		p.scroll(p.height)
	case TopID:
		p.scroll(-p.lines())
	case BottomID:
		p.scroll(p.lines())
	}
}

// lines is how many rows the Doc has at the pane's width.
func (p *DocPane) lines() int { return len(p.rows(p.width)) }

// scroll moves the rows shown by k, within the document; at its end the
// pane follows the tail again, unless WithoutFollow.
func (p *DocPane) scroll(k int) {
	if k == 0 {
		return
	}
	n := p.lines()
	bottom := max(n-p.height, 0)
	top := min(max(p.first(n, p.height)+k, 0), bottom)
	p.follow = top == bottom && !p.cfg.noFollow
	p.top = top
	if p.follow {
		p.top = 0 // unused while following, so that a scroll at the end changes nothing
	}
}

// first is the first row shown of n, height at a time.
func (p *DocPane) first(n, height int) int {
	bottom := max(n-height, 0)
	if p.follow {
		return bottom
	}
	return min(p.top, bottom)
}

// column is the width the Doc is drawn at in a pane width cells wide:
// all of it but the marks' column, when there is room for both.
func column(width int) int {
	if width >= 2 {
		return width - 1
	}
	return width
}

// rows is the Doc's rows at a pane width wide, each fitted to its column
// with the workspace's width method and the theme's Ellipsis, kept while
// the Doc, the width, the method and the theme are the same. A theme of ID
// 0, built as a literal, is never served from the cache.
func (p *DocPane) rows(width int) []string {
	if width <= 0 {
		return nil
	}
	t := p.w.Theme()
	p.revise(t.ID())
	key := rowsKey{version: p.d.Version(), width: width, method: p.w.WidthMethod(), theme: t.ID()}
	if key.theme != 0 && p.cache.key == key {
		return p.cache.rows
	}
	cw := column(width)
	var out []string
	// A finished Doc's frame is its Renderer's, which ends with a line
	// break: that ends the last row and begins none (0009-PLAN D21).
	if src := strings.TrimSuffix(p.d.Render(cw), "\n"); src != "" {
		for l := range strings.SplitSeq(src, "\n") {
			out = append(out, fit(l, cw, key.method, t.Glyphs.Ellipsis))
		}
	}
	p.cache = rows{key: key, rows: out}
	return out
}

// revise gives the Doc its revision, the theme's ID, so that a Renderer or
// a Highlighter that draws with the theme is drawn again when it changes;
// under a theme of ID 0, a new revision each time, so that a literal theme
// draws right at the cost of every cache
// (docs/decisions/0009-MADR-streaming-content-engine.md A7 item 4;
// docs/decisions/0009-PLAN-streaming-content-engine.md D19).
func (p *DocPane) revise(id uint64) {
	if id == 0 {
		p.literal++
		id = literalRevs | p.literal
	}
	p.d.SetRevision(id)
}

// fit is s cut to cells cells measured with m, ending with tail when it is
// cut, and padded to them.
func fit(s string, cells int, m ansi.Method, tail string) string {
	s = sanitize.Truncate(s, cells, m, tail)
	return s + strings.Repeat(" ", max(cells-sanitize.StringWidth(m, s), 0))
}

// View is the rows shown at the pane's size, with their marks.
func (p *DocPane) View(width, height int) string {
	all := p.rows(width) // first, since a new revision raises the Doc's Version
	p.drawn = drawn{ok: true, version: p.d.Version(), width: width, height: height, top: p.top, follow: p.follow}
	if width <= 0 || height <= 0 {
		return ""
	}
	top := p.first(len(all), height)
	shown := all[top:min(top+height, len(all))]
	if column(width) == width || len(shown) == 0 {
		return strings.Join(shown, "\n")
	}
	t := p.w.Theme()
	m := p.w.WidthMethod()
	var b strings.Builder
	for i, r := range shown {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(r)
		switch {
		case i == len(shown)-1 && top+len(shown) < len(all):
			b.WriteString(mark(t, t.Glyphs.ScrollDown, m))
		case i == 0 && top > 0:
			b.WriteString(mark(t, t.Glyphs.ScrollUp, m))
		default:
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// mark is glyph g in one cell, measured with m, in the theme's Muted
// style.
func mark(t theme.Theme, g string, m ansi.Method) string {
	return t.Styles.Muted.Render(fit(g, 1, m, ""))
}

// Changed reports whether the view differs from the last one drawn: a new
// Version of the Doc, a new size, or a scroll. A message that changes none
// of them, a key or a frame with nothing to pull among them, leaves it
// false.
func (p *DocPane) Changed() bool {
	d := p.drawn
	return !d.ok || d.version != p.d.Version() || d.width != p.width || d.height != p.height ||
		d.top != p.top || d.follow != p.follow
}

// Title is the pane's title, WithTitle's.
func (p *DocPane) Title() string { return p.cfg.title }

// KeyContexts names the pane's key scope, "kind:docpane", which
// DefaultRules binds its keys in.
func (p *DocPane) KeyContexts() []command.Scope { return []command.Scope{DocPaneScope} }

// ConsumesCommand takes the pane's own commands, so that they run in the
// order their keys were pressed.
func (p *DocPane) ConsumesCommand(id command.ID) bool {
	switch id {
	case ScrollUpID, ScrollDownID, PageUpID, PageDownID, TopID, BottomID:
		return true
	}
	return false
}
