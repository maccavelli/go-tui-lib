package launch

import tea "charm.land/bubbletea/v2"

// state is what Run learns from the model while the TUI runs: whether Init
// ran, and the model after the last Update that returned. Init, Update and
// View run on the goroutine that called Run, and Run reads state after
// Bubble Tea has returned, so it needs no lock.
type state[M tea.Model] struct {
	started bool
	last    M
}

// wrapped forwards to the program's model, and records state. Bubble Tea
// asserts no optional interface on a model, so it sees no difference.
type wrapped[M tea.Model] struct {
	inner M
	st    *state[M]
}

// Init records that the model started, then starts it.
func (w wrapped[M]) Init() tea.Cmd {
	w.st.started = true
	return w.inner.Init()
}

// Update keeps an M result as the last good model, and wraps it. A result
// that is not an M is returned as it is; Run reports it at the end.
func (w wrapped[M]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := w.inner.Update(msg)
	m, ok := next.(M)
	if !ok {
		return next, cmd
	}
	w.st.last = m
	return wrapped[M]{inner: m, st: w.st}, cmd
}

// View is the model's view.
func (w wrapped[M]) View() tea.View { return w.inner.View() }

// unwrap is final as an M: the model inside Run's wrapper, or final itself.
func unwrap[M tea.Model](final tea.Model) (M, bool) {
	if w, ok := final.(wrapped[M]); ok {
		return w.inner, true
	}
	m, ok := final.(M)
	return m, ok
}
