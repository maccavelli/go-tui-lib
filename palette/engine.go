package palette

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// The drain's bounds: a batch holds at most maxBatch hits, or what arrived
// within one frame of the first (0008-MADR §4).
const (
	maxBatch = 64
	frame    = 16 * time.Millisecond
)

// listPrefixes is the query that lists the providers' prefixes.
const listPrefixes = "?"

// source is one registered provider, with what the engine reads of it once,
// at registration, so that nothing but Search, Discover, Start and Stop runs
// a provider's code after that.
type source struct {
	p        Provider
	name     string
	when     when.Expr
	hasWhen  bool
	prefix   string
	debounce time.Duration
}

// session is one opening of the picker: its context, which closing cancels,
// each source's start, and the searches still running.
type session struct {
	ctx    context.Context
	cancel context.CancelFunc
	whenC  when.Context
	active []bool   // by source: its availability held at open
	starts []*start // by source
	wg     sync.WaitGroup
}

// start is a source's one Start in a session. Its first search begins it,
// on the event loop, and every search waits for done, or for its own
// cancellation.
type start struct {
	begun bool          // the source was used in the session
	done  chan struct{} // closed when Start has returned
	err   error         // Start's, read once done is closed
}

// event is what a search sends: a hit, or its end and failure.
type event struct {
	src int
	hit Hit
	end bool
	err error
}

// hitsMsg is one drained batch of a generation's events.
type hitsMsg struct {
	gen    uint64
	events []event
}

// startMsg ends a source's debounce in a generation.
type startMsg struct {
	gen uint64
	src int
}

// entry is a kept hit: its source and its arrival, which breaks ties.
type entry struct {
	hit Hit
	src int
	seq int
}

// row is one line of the list: a hit, or a provider's failure, whose hit
// names the provider and holds nothing else.
type row struct {
	hit    Hit
	src    int
	failed bool
}

// engine runs the providers for a picker (0008-MADR §4). Every method runs
// on the event loop; only the searches, and the drains that read them, run
// off it.
type engine struct {
	cfg     config
	sources []*source
	s       *session // nil while closed

	gen     uint64
	cancel  context.CancelFunc
	ctx     context.Context
	ch      chan event
	search  Search
	pending int // the generation's sources that have not ended
	prefix  bool

	entries  []entry
	failures []int // sources, in the order they failed
	seq      int
}

func newEngine(cfg config) *engine { return &engine{cfg: cfg} }

// add registers p, available where a.When holds.
func (e *engine) add(p Provider, a Availability) error {
	src := &source{p: p, name: p.Name()}
	if a.When != "" {
		x, err := when.Parse(a.When)
		if err != nil {
			return fmt.Errorf("palette: provider %q: %w", src.name, err)
		}
		src.when, src.hasWhen = x, true
	}
	if pr, ok := p.(Prefixed); ok {
		src.prefix = pr.Prefix()
	}
	if d, ok := p.(Debouncer); ok {
		src.debounce = d.Debounce()
	}
	e.sources = append(e.sources, src)
	if e.s != nil {
		e.s.active = append(e.s.active, src.available(e.s.whenC))
		e.s.starts = append(e.s.starts, &start{})
	}
	return nil
}

func (src *source) available(c when.Context) bool { return !src.hasWhen || src.when.Eval(c) }

// open begins a session with the when values c, which decide the
// providers' availability until the session closes.
func (e *engine) open(c when.Context) {
	e.close()
	if c == nil {
		c = when.Map{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &session{ctx: ctx, cancel: cancel, whenC: c}
	for _, src := range e.sources {
		s.active = append(s.active, src.available(c))
		s.starts = append(s.starts, &start{})
	}
	e.s = s
}

// close ends the session: its searches are cancelled, their late hits are
// dropped, and once they have returned each provider used in it is
// stopped.
func (e *engine) close() {
	s := e.s
	if s == nil {
		return
	}
	e.s = nil
	e.reset()
	s.cancel()
	go func() {
		s.wg.Wait()
		for i, st := range s.starts {
			if st.begun {
				e.stop(e.sources[i])
			}
		}
	}()
}

// stop calls src's Stop, if it has one; a panic there is logged.
func (e *engine) stop(src *source) {
	st, ok := src.p.(Stopper)
	if !ok {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			e.cfg.logger.Error("palette: provider failed", "provider", src.name, "error", fmt.Errorf("stop panicked: %v", r))
		}
	}()
	st.Stop()
}

// reset starts a new generation with nothing in it: the last one's
// searches are cancelled and its late messages dropped.
func (e *engine) reset() {
	if e.cancel != nil {
		e.cancel()
	}
	e.gen++
	e.cancel, e.ctx, e.ch = nil, nil, nil
	e.search, e.pending, e.prefix = Search{}, 0, false
	e.entries, e.failures, e.seq = e.entries[:0], nil, 0
}

// query starts the search for text, and returns the commands that run it.
func (e *engine) query(text string) tea.Cmd {
	e.reset()
	if e.s == nil {
		return nil
	}
	if text == listPrefixes {
		e.prefix = true
		return nil
	}
	targets := e.targets(&text)
	e.ctx, e.cancel = context.WithCancel(e.s.ctx)
	e.ch = make(chan event, maxBatch)
	e.search = Search{Text: text, WhenContext: e.s.whenC}
	if text != "" {
		e.search.Matcher = fuzzy.NewMatcher(text)
	}
	var cmds []tea.Cmd
	for _, i := range targets {
		src := e.sources[i]
		if _, ok := src.p.(Discoverer); text == "" && !ok {
			continue
		}
		e.pending++
		if src.debounce > 0 {
			gen := e.gen
			cmds = append(cmds, tea.Tick(src.debounce, func(time.Time) tea.Msg { return startMsg{gen: gen, src: i} }))
			continue
		}
		e.launch(i)
	}
	if e.pending == 0 {
		return nil
	}
	return tea.Batch(append(cmds, e.drain())...)
}

// targets is the sources a query searches: the one whose prefix starts it,
// with the prefix taken off text, or else every available source.
func (e *engine) targets(text *string) []int {
	var all []int
	for i, src := range e.sources {
		if !e.s.active[i] {
			continue
		}
		if src.prefix != "" && strings.HasPrefix(*text, src.prefix) {
			*text = (*text)[len(src.prefix):]
			return []int{i}
		}
		all = append(all, i)
	}
	return all
}

// launch starts source i's search in the current generation, and its
// Start first if it is the session's first.
func (e *engine) launch(i int) {
	s, ctx, ch, q, src := e.s, e.ctx, e.ch, e.search, e.sources[i]
	st := s.starts[i]
	if !st.begun {
		st.begun, st.done = true, make(chan struct{})
		s.wg.Go(func() {
			defer close(st.done)
			st.err = src.start(s.ctx)
		})
	}
	s.wg.Go(func() {
		send := func(ev event) bool {
			select {
			case ch <- ev:
				return true
			case <-ctx.Done():
				return false
			}
		}
		err := src.run(ctx, st, q, func(h Hit) bool { return send(event{src: i, hit: h}) })
		send(event{src: i, end: true, err: err})
	})
}

// start runs the source's Start, if it has one, with the session's
// context. A panic is its error.
func (src *source) start(life context.Context) (err error) {
	sp, ok := src.p.(Starter)
	if !ok {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("start panicked: %v", r)
		}
	}()
	return sp.Start(life)
}

// run waits for the source's start, then searches, or discovers for an
// empty query. A panic is its error.
func (src *source) run(ctx context.Context, st *start, q Search, yield func(Hit) bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panicked: %v", r)
		}
	}()
	select {
	case <-st.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	if st.err != nil {
		return st.err
	}
	if q.Text != "" {
		return src.p.Search(ctx, q, yield)
	}
	if d, ok := src.p.(Discoverer); ok {
		return d.Discover(ctx, yield)
	}
	return nil
}

// drain waits for the generation's first event, then takes what else
// arrives within a frame, up to maxBatch hits, or until every search
// running has ended. It gives nothing when the generation is cancelled
// first.
func (e *engine) drain() tea.Cmd {
	gen, ctx, ch, pending := e.gen, e.ctx, e.ch, e.pending
	return func() tea.Msg {
		m := hitsMsg{gen: gen}
		hits, ends := 0, 0
		take := func(ev event) {
			m.events = append(m.events, ev)
			if ev.end {
				ends++
			} else {
				hits++
			}
		}
		select {
		case ev := <-ch:
			take(ev)
		case <-ctx.Done():
			return nil
		}
		t := time.NewTimer(frame)
		defer t.Stop()
		for hits < maxBatch && ends < pending {
			select {
			case ev := <-ch:
				take(ev)
			case <-t.C:
				return m
			case <-ctx.Done():
				return m
			}
		}
		return m
	}
}

// update takes the engine's own messages, and returns the next drain while
// the generation has sources running.
func (e *engine) update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case startMsg:
		if m.gen == e.gen {
			e.launch(m.src)
		}
	case hitsMsg:
		if m.gen != e.gen {
			return nil
		}
		for _, ev := range m.events {
			e.take(ev)
		}
		if e.pending > 0 {
			return e.drain()
		}
	}
	return nil
}

// take merges one event of the current generation.
func (e *engine) take(ev event) {
	src := e.sources[ev.src]
	if ev.end {
		e.pending--
		if ev.err != nil {
			e.cfg.logger.Error("palette: provider failed", "provider", src.name, "error", ev.err)
			e.failures = append(e.failures, ev.src)
		}
		return
	}
	h := ev.hit
	if h.Group == "" {
		h.Group = src.name
	}
	e.seq++
	e.insert(entry{hit: h, src: ev.src, seq: e.seq})
}

// insert keeps n in order, unless the limit is full of better hits; a hit
// with the ID of a kept one replaces it only with a better score.
func (e *engine) insert(n entry) {
	if n.hit.ID != "" {
		if i := slices.IndexFunc(e.entries, func(x entry) bool { return x.hit.ID == n.hit.ID }); i >= 0 {
			if !better(n.hit.Score, e.entries[i].hit.Score) {
				return
			}
			e.entries = slices.Delete(e.entries, i, i+1)
		}
	}
	i, _ := slices.BinarySearchFunc(e.entries, n, func(x, n entry) int {
		if e.before(x, n) {
			return -1
		}
		return 1
	})
	if i >= e.cfg.limit {
		return
	}
	e.entries = slices.Insert(e.entries, i, n)
	if len(e.entries) > e.cfg.limit {
		e.entries = e.entries[:e.cfg.limit]
	}
}

// before orders kept hits: by provider first under ByProvider, then by
// score, then by arrival.
func (e *engine) before(a, b entry) bool {
	if e.cfg.grouping == ByProvider && a.src != b.src {
		return a.src < b.src
	}
	if a.hit.Score != b.hit.Score {
		return better(a.hit.Score, b.hit.Score)
	}
	return a.seq < b.seq
}

// better reports whether a is a better score than b: a lower grade, or a
// higher value within one.
func better(a, b fuzzy.Score) bool {
	if a.Grade != b.Grade {
		return a.Grade < b.Grade
	}
	return a.Value > b.Value
}

// rows is the list: the prefixes for "?", or else the kept hits, best
// first, then a row for each provider that failed.
func (e *engine) rows() []row {
	var out []row
	if e.prefix {
		for i, src := range e.sources {
			if src.prefix != "" && e.s != nil && e.s.active[i] {
				out = append(out, row{hit: Hit{ID: listPrefixes + src.prefix, Text: src.prefix, Group: src.name}, src: i})
			}
		}
		return out
	}
	for _, x := range e.entries {
		out = append(out, row{hit: x.hit, src: x.src})
	}
	for _, i := range e.failures {
		out = append(out, row{hit: Hit{Text: e.sources[i].name, Group: e.sources[i].name}, src: i, failed: true})
	}
	return out
}
