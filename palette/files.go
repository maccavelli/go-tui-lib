package palette

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/fuzzy"
)

// The file provider's defaults (0008-MADR §5, A3 items 7 and 10): a walk
// of more paths than DefaultIndexAbove is searched through a fuzzy.Index,
// and a walk stops at DefaultMaxFiles.
const (
	DefaultIndexAbove = 5000
	DefaultMaxFiles   = 200_000
)

// filesDebounce is how long the file provider waits after a keystroke, so
// a burst of keys searches the files once.
const filesDebounce = 30 * time.Millisecond

// Walker yields the paths a file provider searches, slash-separated and
// relative to its root, until ctx is cancelled or yield returns false.
type Walker interface {
	Walk(ctx context.Context, yield func(path string) bool) error
}

// WalkerFunc is a function used as a Walker.
type WalkerFunc func(ctx context.Context, yield func(path string) bool) error

// Walk calls f.
func (f WalkerFunc) Walk(ctx context.Context, yield func(path string) bool) error {
	return f(ctx, yield)
}

// PathFilter leaves paths out of a walk: a directory it skips is not
// entered. Ignore rules, such as a .gitignore's, come through it.
type PathFilter interface {
	Skip(path string, d fs.DirEntry) bool
}

// PathFilterFunc is a function used as a PathFilter.
type PathFilterFunc func(path string, d fs.DirEntry) bool

// Skip calls f.
func (f PathFilterFunc) Skip(p string, d fs.DirEntry) bool { return f(p, d) }

// FSWalker walks fsys's files, such as os.DirFS(root)'s, leaving out what
// skip skips; a nil skip skips nothing. The caller gives the fs.FS, so the
// palette opens nothing itself. A directory it cannot read is passed over.
func FSWalker(fsys fs.FS, skip PathFilter) Walker {
	return WalkerFunc(func(ctx context.Context, yield func(string) bool) error {
		err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
			if cerr := ctx.Err(); cerr != nil {
				return cerr
			}
			switch {
			case err != nil && p == ".":
				return err
			case err != nil:
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			case p == ".":
				return nil
			case skip != nil && skip.Skip(p, d):
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			case d.IsDir():
				return nil
			}
			if !yield(p) {
				return fs.SkipAll
			}
			return nil
		})
		if errors.Is(err, fs.SkipAll) {
			return nil
		}
		return err
	})
}

// FileOption configures Files.
type FileOption interface{ applyFile(*fileConfig) }

type fileFunc func(*fileConfig)

func (f fileFunc) applyFile(c *fileConfig) { f(c) }

type fileConfig struct {
	indexAbove, maxFiles int
}

// WithIndexAbove searches a walk of more than n paths through a
// fuzzy.Index, and a smaller one in full; below 1 keeps
// DefaultIndexAbove.
func WithIndexAbove(n int) FileOption {
	return fileFunc(func(c *fileConfig) {
		if n > 0 {
			c.indexAbove = n
		}
	})
}

// WithMaxFiles stops a walk at n paths, and shows one muted row saying so;
// below 1 keeps DefaultMaxFiles.
func WithMaxFiles(n int) FileOption {
	return fileFunc(func(c *fileConfig) {
		if n > 0 {
			c.maxFiles = n
		}
	})
}

// Files is a provider of the paths walk yields (0008-MADR §5, A3 item 7).
// Each time the picker opens it walks them once, in Start, off the event
// loop, and indexes them when there are more than WithIndexAbove. It owns
// the prefix "@", waits 30 ms after a keystroke, and ranks a path by
// crush's tiers on its last element: the element, or its stem, equal to the
// query; the element starting with it; a whole element of the path equal to
// it; then the rest, each tier by the matcher's score. A path hit has no
// Run: SelectedMsg carries it to the program.
func Files(walk Walker, opts ...FileOption) Provider {
	c := fileConfig{indexAbove: DefaultIndexAbove, maxFiles: DefaultMaxFiles}
	for _, o := range opts {
		o.applyFile(&c)
	}
	return &files{walk: walk, cfg: c}
}

type files struct {
	walk Walker
	cfg  fileConfig
	at   atomic.Pointer[walked] // the last opening's walk
}

// walked is one walk: its paths, their index when there are enough, and
// whether it stopped at the limit.
type walked struct {
	paths   []string
	ix      *fuzzy.Index
	stopped bool
}

func (*files) Name() string            { return "files" }
func (*files) Prefix() string          { return "@" }
func (*files) Debounce() time.Duration { return filesDebounce }

// Start walks the paths, stopping at the limit, and indexes them.
func (f *files) Start(ctx context.Context) error {
	w := &walked{}
	err := f.walk.Walk(ctx, func(p string) bool {
		if len(w.paths) == f.cfg.maxFiles {
			w.stopped = true
			return false
		}
		w.paths = append(w.paths, p)
		return true
	})
	if err != nil {
		return err
	}
	if len(w.paths) > f.cfg.indexAbove {
		w.ix = fuzzy.NewIndex(w.paths)
	}
	f.at.Store(w)
	return nil
}

func (f *files) Search(ctx context.Context, q Search, yield func(Hit) bool) error {
	w := f.at.Load()
	if w == nil {
		return nil
	}
	try := func(p string) bool {
		sc, ok := tiered(q.Matcher, p)
		return !ok || yield(Hit{ID: "file:" + p, Text: p, Score: sc})
	}
	n := 0
	live := func() bool { // checks the context every 256 paths
		n++
		return n%256 != 0 || ctx.Err() == nil
	}
	if w.ix != nil {
		for i := range w.ix.Candidates(q.Matcher) {
			if !live() {
				return ctx.Err()
			}
			if !try(w.ix.At(i)) {
				return nil
			}
		}
	} else {
		for _, p := range w.paths {
			if !live() {
				return ctx.Err()
			}
			if !try(p) {
				return nil
			}
		}
	}
	if w.stopped {
		yield(Hit{
			ID:       "files:limit",
			Text:     fmt.Sprintf("only the first %d files are searched", f.cfg.maxFiles),
			Disabled: "the file limit",
			Score:    fuzzy.Score{Grade: fuzzy.Subsequence, Value: math.MinInt},
		})
	}
	return ctx.Err()
}

// tiered is m's score of p, its grade set by crush's tiers on the last path
// element (0008-MADR §5), and whether m matched p at all.
func tiered(m *fuzzy.Matcher, p string) (fuzzy.Score, bool) {
	sc, ok := m.Score(p)
	if !ok {
		return sc, false
	}
	q := m.Query()
	base := path.Base(p)
	switch {
	case strings.EqualFold(base, q) || strings.EqualFold(strings.TrimSuffix(base, path.Ext(base)), q):
		sc.Grade = fuzzy.Exact
	case strings.HasPrefix(strings.ToLower(base), strings.ToLower(q)):
		sc.Grade = fuzzy.Prefix
	case hasElement(p, q):
		sc.Grade = fuzzy.Boundary
	default:
		sc.Grade = fuzzy.Subsequence
	}
	return sc, true
}

// hasElement reports whether an element of the slash-separated p is q,
// under folding.
func hasElement(p, q string) bool {
	for e := range strings.SplitSeq(p, "/") {
		if strings.EqualFold(e, q) {
			return true
		}
	}
	return false
}
