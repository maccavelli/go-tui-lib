package palette

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/pprof"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// fake is a provider whose Search is a function, and which records each
// search's query and context.
type fake struct {
	name   string
	search func(ctx context.Context, q Search, yield func(Hit) bool) error

	mu    sync.Mutex
	texts []string
	ctxs  []context.Context
	qs    []Search
	stops atomic.Int32
}

func (f *fake) Name() string { return f.name }

func (f *fake) Search(ctx context.Context, q Search, yield func(Hit) bool) error {
	f.mu.Lock()
	f.texts = append(f.texts, q.Text)
	f.ctxs = append(f.ctxs, ctx)
	f.qs = append(f.qs, q)
	f.mu.Unlock()
	if f.search == nil {
		return nil
	}
	return f.search(ctx, q, yield)
}

// searched is the queries f was asked, in order.
func (f *fake) searched() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.texts)
}

// The optional capabilities, each added to a fake by a type of its own.
type (
	prefixed struct {
		*fake
		prefix string
	}
	debounced struct {
		*fake
		d time.Duration
	}
	stopping   struct{ *fake }
	discovered struct {
		*fake
		recent []Hit
	}
	started struct {
		*fake
		n   atomic.Int32
		err error
	}
)

func (p prefixed) Prefix() string              { return p.prefix }
func (p debounced) Debounce() time.Duration    { return p.d }
func (p stopping) Stop()                       { p.stops.Add(1) }
func (p *started) Start(context.Context) error { p.n.Add(1); return p.err }

// walking is a provider whose Start runs until the picker closes.
type walking struct{ *fake }

func (walking) Start(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }

func (p discovered) Discover(_ context.Context, yield func(Hit) bool) error {
	for _, h := range p.recent {
		if !yield(h) {
			return nil
		}
	}
	return nil
}

// hit is a hit with an ID, which is also its text, and a score.
func hit(id string, g fuzzy.Grade, v int) Hit {
	return Hit{ID: id, Text: id, Score: fuzzy.Score{Grade: g, Value: v}}
}

// echo yields one hit, the query, and then waits for its cancellation.
func echo(ctx context.Context, q Search, yield func(Hit) bool) error {
	yield(hit(q.Text, fuzzy.Exact, 0))
	<-ctx.Done()
	return ctx.Err()
}

// yields yields hits, then returns.
func yields(hs ...Hit) func(context.Context, Search, func(Hit) bool) error {
	return func(_ context.Context, _ Search, yield func(Hit) bool) error {
		for _, h := range hs {
			if !yield(h) {
				return nil
			}
		}
		return nil
	}
}

// loop plays Bubble Tea's event loop for an engine, inside a synctest
// bubble: a command runs on a goroutine of its own, its message is queued,
// and settle hands the queue to update, one message at a time, until the
// bubble is idle.
type loop struct {
	t       *testing.T
	e       *engine
	mu      sync.Mutex
	queue   []tea.Msg
	batches []int // the hits in each hitsMsg update was given
	check   func()
}

func newLoop(t *testing.T, opts ...Option) *loop {
	return &loop{t: t, e: newEngine(newConfig(opts))}
}

func (l *loop) add(p Provider, a Availability) {
	l.t.Helper()
	if err := l.e.add(p, a); err != nil {
		l.t.Fatal(err)
	}
}

func (l *loop) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		if m := cmd(); m != nil {
			l.mu.Lock()
			l.queue = append(l.queue, m)
			l.mu.Unlock()
		}
	}()
}

func (l *loop) settle() {
	for {
		synctest.Wait()
		l.mu.Lock()
		q := l.queue
		l.queue = nil
		l.mu.Unlock()
		if len(q) == 0 {
			return
		}
		for _, m := range q {
			if b, ok := m.(tea.BatchMsg); ok {
				for _, c := range b {
					l.run(c)
				}
				continue
			}
			if h, ok := m.(hitsMsg); ok && h.gen == l.e.gen {
				n := 0
				for _, ev := range h.events {
					if !ev.end {
						n++
					}
				}
				l.batches = append(l.batches, n)
			}
			l.run(l.e.update(m))
			if l.check != nil {
				l.check()
			}
		}
	}
}

// query types text and settles.
func (l *loop) query(text string) {
	l.run(l.e.query(text))
	l.settle()
}

// advance moves the fake clock on by d and settles.
func (l *loop) advance(d time.Duration) {
	time.Sleep(d)
	l.settle()
}

// texts is the rows' texts, a failure's marked with a "!".
func (l *loop) texts() []string {
	var out []string
	for _, r := range l.e.rows() {
		if r.failed {
			out = append(out, "!"+r.hit.Text)
			continue
		}
		out = append(out, r.hit.Text)
	}
	return out
}

// noLeaks fails when the goroutineleak profile reports a goroutine.
func noLeaks(t *testing.T) {
	t.Helper()
	p := pprof.Lookup("goroutineleak")
	if p == nil {
		t.Fatal("no goroutineleak profile")
	}
	var b strings.Builder
	if err := p.WriteTo(&b, 1); err != nil {
		t.Fatal(err)
	}
	if first, _, _ := strings.Cut(b.String(), "\n"); first != "goroutineleak profile: total 0" {
		t.Errorf("leaked goroutines:\n%s", b.String())
	}
}

// TestStaleHitsNeverAppear: a provider's hit for the last query, drained
// before the next keystroke and delivered after it, is dropped, and so is
// the search's end.
func TestStaleHitsNeverAppear(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLoop(t)
		l.add(&fake{name: "slow", search: echo}, Availability{})
		l.e.open(nil)
		l.query("a") // the drain holds "a" and waits out its frame
		l.query("b")
		l.advance(frame)
		if got := l.texts(); !slices.Equal(got, []string{"b"}) {
			t.Errorf("rows %q; want only b, the current query's", got)
		}
		l.e.close()
	})
}

// TestBatches: a batch holds at most 64 hits, and a hit that arrives alone
// is delivered within a frame.
func TestBatches(t *testing.T) {
	t.Run("64", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			l := newLoop(t, WithLimit(1000))
			var hs []Hit
			for i := range 200 {
				hs = append(hs, hit(fmt.Sprint(i), fuzzy.Subsequence, i))
			}
			l.add(&fake{name: "many", search: yields(hs...)}, Availability{})
			l.e.open(nil)
			l.query("x")
			if n := len(l.e.rows()); n != 200 {
				t.Errorf("%d rows; want 200", n)
			}
			if len(l.batches) < 4 || slices.Max(l.batches) > maxBatch {
				t.Errorf("batches %v; want at least 4, none over %d hits", l.batches, maxBatch)
			}
			l.e.close()
		})
	})
	t.Run("frame", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			l := newLoop(t)
			l.add(&fake{name: "trickle", search: func(_ context.Context, _ Search, yield func(Hit) bool) error {
				yield(hit("first", fuzzy.Exact, 0))
				time.Sleep(time.Second)
				yield(hit("second", fuzzy.Exact, 0))
				return nil
			}}, Availability{})
			l.e.open(nil)
			start := time.Now()
			l.query("x")
			l.advance(frame)
			if got := l.texts(); !slices.Equal(got, []string{"first"}) || time.Since(start) != frame {
				t.Errorf("after %v: rows %q; want first, after one frame", time.Since(start), got)
			}
			l.advance(time.Second)
			if got := l.texts(); !slices.Equal(got, []string{"first", "second"}) || l.e.pending != 0 {
				t.Errorf("rows %q, %d pending; want first and second, none pending", got, l.e.pending)
			}
			l.e.close()
		})
	})
}

// TestFailuresAreContained: an error, a panic and a failed Start each give
// one row naming the provider, the error reaches the logger and no row, and
// the provider beside them finishes.
func TestFailuresAreContained(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var log bytes.Buffer
		l := newLoop(t, WithLogger(slog.NewTextHandler(&log, nil)))
		l.add(&fake{name: "good", search: yields(hit("one", fuzzy.Exact, 0), hit("two", fuzzy.Exact, 0))}, Availability{})
		l.add(&fake{name: "bad", search: func(context.Context, Search, func(Hit) bool) error { return errors.New("disk on fire") }}, Availability{})
		l.add(&fake{name: "panicky", search: func(context.Context, Search, func(Hit) bool) error { panic("boom") }}, Availability{})
		l.add(&started{fake: &fake{name: "unstarted"}, err: errors.New("no index")}, Availability{})
		l.e.open(nil)
		l.query("x")
		l.advance(frame)
		got := l.texts()
		slices.Sort(got[2:])
		if want := []string{"one", "two", "!bad", "!panicky", "!unstarted"}; !slices.Equal(got, want) {
			t.Errorf("rows %q; want %q", got, want)
		}
		for _, r := range l.e.rows() {
			if s := fmt.Sprintf("%+v", r.hit); strings.Contains(s, "fire") || strings.Contains(s, "boom") || strings.Contains(s, "index") {
				t.Errorf("a row shows the error: %s", s)
			}
		}
		for _, want := range []string{"provider=bad error=\"disk on fire\"", "provider=panicky error=\"panicked: boom\"", "provider=unstarted error=\"no index\""} {
			if !strings.Contains(log.String(), want) {
				t.Errorf("the log lacks %s:\n%s", want, log.String())
			}
		}
		l.e.close()
	})
}

// TestNoLeaks: a cancelled search and a closed picker leave no goroutine,
// the keystroke after a search cancels it, closing cancels a Start still
// running, and closing stops the provider that was used and not the one
// that was not.
func TestNoLeaks(t *testing.T) {
	var used, unused *fake
	synctest.Test(t, func(t *testing.T) {
		used, unused = &fake{name: "used", search: echo}, &fake{name: "unused", search: echo}
		l := newLoop(t)
		l.add(stopping{used}, Availability{})
		l.add(stopping{unused}, Availability{When: "false"})
		l.add(walking{&fake{name: "walker"}}, Availability{})
		l.e.open(nil)
		l.query("a")
		l.query("b")
		if err := used.ctxs[0].Err(); err == nil {
			t.Error("the search for a was not cancelled by the next keystroke")
		}
		l.e.close()
		synctest.Wait()
		if err := used.ctxs[1].Err(); err == nil {
			t.Error("the search for b was not cancelled by closing")
		}
	})
	if used.stops.Load() != 1 || unused.stops.Load() != 0 {
		t.Errorf("stopped %d and %d times; want the used provider once, the other never", used.stops.Load(), unused.stops.Load())
	}
	noLeaks(t)
}

// TestAvailability: a provider searches only where its When held at open,
// and is asked with the when values of the opening.
func TestAvailability(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		editor := &fake{name: "editor", search: yields(hit("e", fuzzy.Exact, 0))}
		everywhere := &fake{name: "everywhere", search: yields(hit("a", fuzzy.Exact, 0))}
		l := newLoop(t)
		l.add(editor, Availability{When: "pane == 'editor'"})
		l.add(everywhere, Availability{})
		if err := l.e.add(&fake{name: "broken"}, Availability{When: "pane =="}); err == nil {
			t.Error("a When that does not parse was registered")
		}
		l.e.open(when.Map{"pane": when.StringValue("editor")})
		l.query("x")
		l.e.open(when.Map{"pane": when.StringValue("shell")})
		l.query("y")
		if got := editor.searched(); !slices.Equal(got, []string{"x"}) {
			t.Errorf("editor searched %q; want only x, while the pane was the editor", got)
		}
		if got := everywhere.searched(); !slices.Equal(got, []string{"x", "y"}) {
			t.Errorf("everywhere searched %q; want x and y", got)
		}
		if v, _ := editor.qs[0].WhenContext.Value("pane"); v.String() != "editor" {
			t.Errorf("the search's when values gave pane %q; want editor", v)
		}
		l.e.close()
	})
}

// TestPrefixes: a query that starts with a prefix searches its provider
// alone, without the prefix, and "?" lists the prefixes.
func TestPrefixes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmds, files, plain := &fake{name: "commands"}, &fake{name: "files"}, &fake{name: "plain"}
		l := newLoop(t)
		l.add(prefixed{cmds, ">"}, Availability{})
		l.add(prefixed{files, "@"}, Availability{})
		l.add(plain, Availability{})
		l.e.open(nil)
		l.query(">ab")
		if q := cmds.qs[0]; q.Text != "ab" || q.Matcher.Query() != "ab" {
			t.Errorf("commands asked %q, matching %q; want ab", q.Text, q.Matcher.Query())
		}
		if len(files.searched())+len(plain.searched()) != 0 {
			t.Errorf("files searched %q and plain %q; want neither", files.searched(), plain.searched())
		}
		l.query("ab")
		for _, f := range []*fake{cmds, files, plain} {
			if got := f.searched(); got[len(got)-1] != "ab" {
				t.Errorf("%s searched %q; want ab last", f.name, got)
			}
		}
		l.query(listPrefixes)
		var got []string
		for _, r := range l.e.rows() {
			got = append(got, r.hit.Text+" "+r.hit.Group)
		}
		if want := []string{"> commands", "@ files"}; !slices.Equal(got, want) {
			t.Errorf("? lists %q; want %q", got, want)
		}
		l.e.close()
	})
}

// TestDebounce: a debounced provider searches after its wait, and the next
// keystroke within the wait cancels the search it would have run.
func TestDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := &fake{name: "files", search: yields(hit("f", fuzzy.Exact, 0))}
		l := newLoop(t)
		l.add(debounced{files, 30 * time.Millisecond}, Availability{})
		l.e.open(nil)
		l.query("a")
		l.advance(20 * time.Millisecond)
		l.query("ab")
		l.advance(20 * time.Millisecond)
		if got := files.searched(); len(got) != 0 {
			t.Errorf("40ms in, searched %q; want nothing, ab's wait not over", got)
		}
		l.advance(10 * time.Millisecond)
		if got := files.searched(); !slices.Equal(got, []string{"ab"}) {
			t.Errorf("searched %q; want ab once, a never", got)
		}
		l.advance(frame)
		if got := l.texts(); !slices.Equal(got, []string{"f"}) {
			t.Errorf("rows %q; want f", got)
		}
		l.e.close()
	})
}

// TestDuplicates: two hits with one ID keep the better score, whichever
// came first, across providers too.
func TestDuplicates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLoop(t)
		l.add(&fake{name: "one", search: yields(hit("x", fuzzy.Subsequence, 1), hit("y", fuzzy.Exact, 5), hit("z", fuzzy.Prefix, 1))}, Availability{})
		l.e.open(nil)
		l.query("q")
		l.add(&fake{name: "two", search: yields(hit("x", fuzzy.Exact, 5), hit("y", fuzzy.Subsequence, 1), hit("z", fuzzy.Prefix, 9))}, Availability{})
		l.query("q")
		var got []string
		for _, r := range l.e.rows() {
			got = append(got, fmt.Sprintf("%s:%v/%d", r.hit.ID, r.hit.Score.Grade, r.hit.Score.Value))
		}
		slices.Sort(got)
		if want := []string{"x:exact/5", "y:exact/5", "z:prefix/9"}; !slices.Equal(got, want) {
			t.Errorf("rows %q; want %q", got, want)
		}
		l.e.close()
	})
}

// TestLimit: the kept hits never number more than the limit, and they are
// the best ones.
func TestLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLoop(t, WithLimit(5))
		var hs []Hit
		for i := range 150 {
			hs = append(hs, hit(fmt.Sprint(i), fuzzy.Subsequence, (i*37)%150))
		}
		l.add(&fake{name: "many", search: yields(hs...)}, Availability{})
		l.check = func() {
			if n := len(l.e.entries); n > 5 {
				t.Fatalf("%d hits kept; the limit is 5", n)
			}
		}
		l.e.open(nil)
		l.query("q")
		var got []int
		for _, r := range l.e.rows() {
			got = append(got, r.hit.Score.Value)
		}
		if want := []int{149, 148, 147, 146, 145}; !slices.Equal(got, want) {
			t.Errorf("kept values %v; want %v", got, want)
		}
		l.e.close()
	})
}

// TestGrouping: ByScore orders every hit by score, ByProvider by
// registration first.
func TestGrouping(t *testing.T) {
	for _, c := range []struct {
		g    Grouping
		want []string
	}{
		{ByScore, []string{"b1", "a1", "b2", "a2"}},
		{ByProvider, []string{"a1", "a2", "b1", "b2"}},
	} {
		synctest.Test(t, func(t *testing.T) {
			l := newLoop(t, WithGrouping(c.g))
			l.add(&fake{name: "a", search: yields(hit("a2", fuzzy.Subsequence, 0), hit("a1", fuzzy.Prefix, 0))}, Availability{})
			l.add(&fake{name: "b", search: yields(hit("b1", fuzzy.Exact, 0), hit("b2", fuzzy.Boundary, 0))}, Availability{})
			l.e.open(nil)
			l.query("q")
			if got := l.texts(); !slices.Equal(got, c.want) {
				t.Errorf("%v: rows %q; want %q", c.g, got, c.want)
			}
			for _, r := range l.e.rows() {
				if r.hit.Group != l.e.sources[r.src].name {
					t.Errorf("%s's group %q; want its provider's name", r.hit.ID, r.hit.Group)
				}
			}
			l.e.close()
		})
	}
}

// TestStartAndDiscover: Start runs once per opening, before the first
// search; an empty query asks the discoverers alone.
func TestStartAndDiscover(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st := &started{fake: &fake{name: "walker", search: yields(hit("w", fuzzy.Exact, 0))}}
		other := &fake{name: "other"}
		l := newLoop(t)
		l.add(discovered{&fake{name: "recents"}, []Hit{hit("recent", fuzzy.Exact, 0)}}, Availability{})
		l.add(st, Availability{})
		l.add(other, Availability{})
		l.e.open(nil)
		l.query("")
		if got := l.texts(); !slices.Equal(got, []string{"recent"}) || len(other.searched()) != 0 {
			t.Errorf("an empty query: rows %q, other searched %q; want recent alone", got, other.searched())
		}
		l.query("a")
		l.query("ab")
		if n := st.n.Load(); n != 1 {
			t.Errorf("started %d times in one opening; want once", n)
		}
		l.e.open(nil)
		l.query("abc")
		if n := st.n.Load(); n != 2 {
			t.Errorf("started %d times in two openings; want twice", n)
		}
		l.e.close()
	})
}
