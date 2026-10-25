package palette

import (
	"bytes"
	"cmp"
	"context"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
	"github.com/maccavelli/go-tui-lib/v2/when"
)

// The recents' defaults: a use counts half as much after DefaultHalfLife,
// and the best DefaultMaxEntries are kept (0008-MADR §6).
const (
	DefaultHalfLife   = 7 * 24 * time.Hour
	DefaultMaxEntries = 500
)

// recentsVersion is the version WriteTo writes and ReadFrom reads.
const recentsVersion = 1

// The limits ReadFrom holds a stored file to (0008-MADR A3 item 10).
const (
	maxRecentsBytes = 1 << 20
	maxRecentID     = 1024
)

// maxRecentBoost bounds what the recents add to a typed query's hit: one
// consecutive-match bonus, the bound fuzzy's RankContext keeps, so that a
// recent hit wins a near-tie within its grade and never passes a clearly
// better match (0008-MADR §2). Each use adds 1, as its score decays.
const maxRecentBoost = 4

// For an empty query, recentFirst lifts a recent hit above every hit
// discovered, and the recents are ordered among themselves by their
// scores, in thousandths; maxOrder keeps the sum an int on any platform.
const (
	recentFirst = 1 << 29
	orderScale  = 1000
	maxOrder    = 1 << 28
)

// ErrRecentsVersion is returned by ReadFrom for recents of a version this
// package does not know, wrapped with that version.
var ErrRecentsVersion = errors.New("palette: unknown recents version")

// Frecency is a store of how recently and how often hits were used, by
// their IDs (0008-MADR §6, A3 items 5, 7 and 10). Each use adds 1 to a
// hit's score, which halves every half-life; the score is worked out when
// it is read, so no timer runs. It keeps IDs only, never a hit's arguments.
// It is safe for use from more than one goroutine.
type Frecency struct {
	halfLife time.Duration
	max      int

	mu      sync.Mutex
	entries map[string]frecent
}

// frecent is one ID's score as of at.
type frecent struct {
	score float64
	at    time.Time
}

// FrecencyOption configures a Frecency.
type FrecencyOption interface{ applyFrecency(*Frecency) }

type frecencyFunc func(*Frecency)

func (f frecencyFunc) applyFrecency(c *Frecency) { f(c) }

// WithHalfLife is how long a use takes to count half as much; below one
// nanosecond keeps DefaultHalfLife.
func WithHalfLife(d time.Duration) FrecencyOption {
	return frecencyFunc(func(f *Frecency) {
		if d > 0 {
			f.halfLife = d
		}
	})
}

// WithMaxEntries keeps the best n IDs, dropping the lowest scored past it;
// below 1 keeps DefaultMaxEntries.
func WithMaxEntries(n int) FrecencyOption {
	return frecencyFunc(func(f *Frecency) {
		if n > 0 {
			f.max = n
		}
	})
}

// NewFrecency returns an empty store.
func NewFrecency(opts ...FrecencyOption) *Frecency {
	f := &Frecency{halfLife: DefaultHalfLife, max: DefaultMaxEntries, entries: map[string]frecent{}}
	for _, o := range opts {
		o.applyFrecency(f)
	}
	return f
}

// Record counts one use of id now. An empty ID, or one longer than 1024
// bytes, is not kept.
func (f *Frecency) Record(id string) {
	if id == "" || len(id) > maxRecentID {
		return
	}
	now := time.Now().UTC()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries[id] = frecent{score: f.decayed(f.entries[id], now) + 1, at: now}
	f.trim(now)
}

// Score is id's score now: each use's 1, halved for every half-life since
// it, summed; 0 for an ID never used.
func (f *Frecency) Score(id string) float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.entries[id]
	if !ok {
		return 0
	}
	return f.decayed(e, time.Now())
}

// decayed is e's score at now. A time before e's, as a clock set back
// gives, decays nothing.
func (f *Frecency) decayed(e frecent, now time.Time) float64 {
	if e.score == 0 {
		return 0
	}
	elapsed := max(now.Sub(e.at), 0)
	return e.score * math.Exp2(-float64(elapsed)/float64(f.halfLife))
}

// trim drops the lowest scored entries at now past the store's maximum,
// the greater ID first among equals.
func (f *Frecency) trim(now time.Time) {
	if len(f.entries) <= f.max {
		return
	}
	for _, id := range f.ranked(now)[f.max:] {
		delete(f.entries, id)
	}
}

// ranked is the IDs, best scored at now first, then by ID.
func (f *Frecency) ranked(now time.Time) []string {
	ids := make([]string, 0, len(f.entries))
	for id := range f.entries {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b string) int {
		if c := cmp.Compare(f.decayed(f.entries[b], now), f.decayed(f.entries[a], now)); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})
	return ids
}

// recentsFile is the stored form:
// {"version":1,"entries":[{"id":…,"score":…,"at":…}]}.
type recentsFile struct {
	Version int            `json:"version"`
	Entries []recentsEntry `json:"entries"`
}

type recentsEntry struct {
	ID    string    `json:"id"`
	Score float64   `json:"score"`
	At    time.Time `json:"at"`
}

// WriteTo writes the store as versioned JSON, best scored first, and
// returns the bytes written.
func (f *Frecency) WriteTo(w io.Writer) (int64, error) {
	f.mu.Lock()
	doc := recentsFile{Version: recentsVersion, Entries: []recentsEntry{}}
	for _, id := range f.ranked(time.Now()) {
		e := f.entries[id]
		doc.Entries = append(doc.Entries, recentsEntry{ID: id, Score: e.score, At: e.at})
	}
	f.mu.Unlock()
	b, err := json.Marshal(doc)
	if err != nil {
		return 0, fmt.Errorf("palette: recents: %w", err)
	}
	n, err := w.Write(append(b, '\n'))
	return int64(n), err
}

// ReadFrom replaces the store with what WriteTo wrote, and returns the
// bytes read. It reads at most 1 MiB, and refuses an unknown member, an
// unknown version (ErrRecentsVersion), an empty or repeated ID, one longer
// than 1024 bytes, and a score that is not above 0. Past the store's
// maximum, the best scored are kept. On an error the store is unchanged.
func (f *Frecency) ReadFrom(r io.Reader) (int64, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxRecentsBytes+1))
	n := int64(len(b))
	if err != nil {
		return n, fmt.Errorf("palette: recents: %w", err)
	}
	if len(b) > maxRecentsBytes {
		return n, errors.New("palette: recents: over 1 MiB")
	}
	var doc recentsFile
	if err := json.UnmarshalRead(bytes.NewReader(b), &doc, json.RejectUnknownMembers(true)); err != nil {
		return n, fmt.Errorf("palette: recents: %w", err)
	}
	if doc.Version != recentsVersion {
		return n, fmt.Errorf("%w: %d", ErrRecentsVersion, doc.Version)
	}
	entries := make(map[string]frecent, len(doc.Entries))
	for _, e := range doc.Entries {
		switch _, seen := entries[e.ID]; {
		case e.ID == "":
			return n, errors.New("palette: recents: an entry has no id")
		case len(e.ID) > maxRecentID:
			return n, fmt.Errorf("palette: recents: an id of %d bytes, over %d", len(e.ID), maxRecentID)
		case seen:
			return n, fmt.Errorf("palette: recents: id %q twice", e.ID)
		case !(e.Score > 0):
			return n, fmt.Errorf("palette: recents: id %q has score %v", e.ID, e.Score)
		}
		entries[e.ID] = frecent{score: e.Score, at: e.At}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = entries
	f.trim(time.Now())
	return n, nil
}

// boost is what score adds to a typed query's hit: the score, rounded, and
// at most maxRecentBoost.
func boost(score float64) int { return int(math.Round(min(score, maxRecentBoost))) }

// order is a recent hit's place among the recents for an empty query.
func order(score float64) int { return int(min(math.Round(score*orderScale), maxOrder)) }

// Recents wraps inner so that its recently and often used hits come first
// for an empty query, the most used first, and win a near-tie within their
// grade for a typed one, by their scores in f (0008-MADR §2, §5): a recent
// hit gains at most one consecutive-match bonus, so it never passes a
// clearly better match, nor a better grade. The program records the hits a picker runs in the same store,
// with WithRecents. Recents has inner's name, prefix, debounce, start and
// stop; it discovers nothing when inner does not.
func Recents(f *Frecency, inner Provider) Provider { return &recents{f: f, inner: inner} }

type recents struct {
	f     *Frecency
	inner Provider
}

func (r *recents) Name() string { return r.inner.Name() }

func (r *recents) Prefix() string {
	if p, ok := r.inner.(Prefixed); ok {
		return p.Prefix()
	}
	return ""
}

func (r *recents) Debounce() time.Duration {
	if d, ok := r.inner.(Debouncer); ok {
		return d.Debounce()
	}
	return 0
}

func (r *recents) Start(ctx context.Context) error {
	if s, ok := r.inner.(Starter); ok {
		return s.Start(ctx)
	}
	return nil
}

func (r *recents) Stop() {
	if s, ok := r.inner.(Stopper); ok {
		s.Stop()
	}
}

func (r *recents) opened(c when.Context) {
	if op, ok := r.inner.(opener); ok {
		op.opened(c)
	}
}

func (r *recents) score(id string) float64 {
	if r.f == nil || id == "" {
		return 0
	}
	return r.f.Score(id)
}

func (r *recents) Search(ctx context.Context, s Search, yield func(Hit) bool) error {
	return r.inner.Search(ctx, s, func(h Hit) bool {
		h.Score.Value += boost(r.score(h.ID))
		return yield(h)
	})
}

func (r *recents) Discover(ctx context.Context, yield func(Hit) bool) error {
	d, ok := r.inner.(Discoverer)
	if !ok {
		return nil
	}
	return d.Discover(ctx, func(h Hit) bool {
		if sc := r.score(h.ID); sc > 0 {
			h.Score = fuzzy.Score{Grade: fuzzy.Exact, Value: recentFirst + order(sc)}
		}
		return yield(h)
	})
}
