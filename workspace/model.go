package workspace

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Bubble is the shape of a bubbles component, and of any model built like
// one: Update returns the changed model by value, and View returns a string.
type Bubble[M any] interface {
	Update(msg tea.Msg) (M, tea.Cmd)
	View() string
}

// Model hosts a Bubble as a pane. Wrap builds it. The workspace's SizeMsg,
// PaneFocusMsg and PaneBlurMsg become calls on the hosted model, found by
// type assertion or given by an option; every other message reaches the
// model's own Update.
type Model[M Bubble[M]] struct {
	// M is the hosted model. A program may read and set it between updates.
	M M

	onSize  func(m *M, width, height int)
	onFocus func(m *M) tea.Cmd
	onBlur  func(m *M)
	cursor  func(m M) *tea.Cursor
	keys    func(m M) []key.Binding
}

// WrapOption overrides one of the methods Wrap finds by type assertion.
type WrapOption[M Bubble[M]] func(*Model[M])

// Wrap hosts m as a pane. For each of these, Wrap uses the model's own
// method when it has one, unless an option replaces it:
//
//   - SizeMsg calls SetSize(width, height), or SetWidth and SetHeight, on
//     *M, as bubbles list, viewport, textinput and textarea define them;
//   - PaneFocusMsg calls Focus() tea.Cmd, and PaneBlurMsg calls Blur(), on
//     *M, as bubbles textinput and textarea define them;
//   - Cursor returns the model's Cursor() *tea.Cursor, in the pane's cells.
//     bubbles textinput and textarea draw a virtual cursor by default and
//     return none; call SetVirtualCursor(false) to use the terminal's;
//   - Keys returns the model's Keys() []key.Binding.
func Wrap[M Bubble[M]](m M, opts ...WrapOption[M]) *Model[M] {
	p := &Model[M]{M: m}
	for _, o := range opts {
		o(p)
	}
	return p
}

// OnSize replaces how the model is told its size.
func OnSize[M Bubble[M]](f func(m *M, width, height int)) WrapOption[M] {
	return func(p *Model[M]) { p.onSize = f }
}

// OnFocus replaces how the model is told it has the keyboard.
func OnFocus[M Bubble[M]](f func(m *M) tea.Cmd) WrapOption[M] {
	return func(p *Model[M]) { p.onFocus = f }
}

// OnBlur replaces how the model is told it lost the keyboard.
func OnBlur[M Bubble[M]](f func(m *M)) WrapOption[M] {
	return func(p *Model[M]) { p.onBlur = f }
}

// WithCursor replaces where the model's cursor is, in the pane's cells.
func WithCursor[M Bubble[M]](f func(m M) *tea.Cursor) WrapOption[M] {
	return func(p *Model[M]) { p.cursor = f }
}

// WithKeys gives the model's bindings, for a help footer.
func WithKeys[M Bubble[M]](f func(m M) []key.Binding) WrapOption[M] {
	return func(p *Model[M]) { p.keys = f }
}

// Update handles the workspace's size and focus messages, and passes every
// other message to the hosted model.
func (p *Model[M]) Update(msg tea.Msg) (Pane, tea.Cmd) {
	switch m := msg.(type) {
	case SizeMsg:
		p.setSize(m.Width, m.Height)
		return p, nil
	case PaneFocusMsg:
		cmd := p.focus()
		return p, cmd
	case PaneBlurMsg:
		p.blur()
		return p, nil
	}
	next, cmd := p.M.Update(msg)
	p.M = next
	return p, cmd
}

// View is the hosted model's view. The workspace clips it to the pane,
// measuring with its own width method, which the model cannot know.
func (p *Model[M]) View(int, int) string {
	return p.M.View()
}

// Cursor is the hosted model's cursor, in the pane's own cells, or nil.
func (p *Model[M]) Cursor() *tea.Cursor {
	if p.cursor != nil {
		return p.cursor(p.M)
	}
	if c, ok := any(p.M).(interface{ Cursor() *tea.Cursor }); ok {
		return c.Cursor()
	}
	if c, ok := any(&p.M).(interface{ Cursor() *tea.Cursor }); ok {
		return c.Cursor()
	}
	return nil
}

// Keys is the hosted model's bindings, or nil.
func (p *Model[M]) Keys() []key.Binding {
	if p.keys != nil {
		return p.keys(p.M)
	}
	if k, ok := any(p.M).(interface{ Keys() []key.Binding }); ok {
		return k.Keys()
	}
	return nil
}

func (p *Model[M]) setSize(width, height int) {
	if p.onSize != nil {
		p.onSize(&p.M, width, height)
		return
	}
	if s, ok := any(&p.M).(interface{ SetSize(width, height int) }); ok {
		s.SetSize(width, height)
		return
	}
	if s, ok := any(&p.M).(interface{ SetWidth(width int) }); ok {
		s.SetWidth(width)
	}
	if s, ok := any(&p.M).(interface{ SetHeight(height int) }); ok {
		s.SetHeight(height)
	}
}

func (p *Model[M]) focus() tea.Cmd {
	if p.onFocus != nil {
		return p.onFocus(&p.M)
	}
	if f, ok := any(&p.M).(interface{ Focus() tea.Cmd }); ok {
		return f.Focus()
	}
	return nil
}

func (p *Model[M]) blur() {
	if p.onBlur != nil {
		p.onBlur(&p.M)
		return
	}
	if b, ok := any(&p.M).(interface{ Blur() }); ok {
		b.Blur()
	}
}
