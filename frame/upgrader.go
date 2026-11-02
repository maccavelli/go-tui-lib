package frame

import (
	"context"
	"sync"
)

// UpgradeMsg is an upgraded render: the key and generation it was
// submitted with, and its result.
type UpgradeMsg[K comparable, R any] struct {
	Key    K
	Gen    uint64
	Result R
}

// Upgrader runs costly renders off the event loop, one job a key, so that
// a cheap render is shown until the costly one comes
// (docs/decisions/0009-MADR-streaming-content-engine.md A1 §3). Each job
// runs on a goroutine of its own, with a context cancelled when a newer
// job replaces it and on Close. A key is remembered while its job runs,
// until its result has been read or dropped
// (docs/decisions/0009-PLAN-streaming-content-engine.md D8). Its methods
// are safe for concurrent use.
type Upgrader[K comparable, R any] struct {
	results chan UpgradeMsg[K, R]
	wg      sync.WaitGroup

	mu     sync.Mutex
	jobs   map[K]*upgradeJob
	closed bool
}

// upgradeJob is a key's running job.
type upgradeJob struct {
	gen    uint64
	cancel context.CancelFunc
}

// NewUpgrader is an Upgrader with no job running.
func NewUpgrader[K comparable, R any]() *Upgrader[K, R] {
	return &Upgrader[K, R]{
		results: make(chan UpgradeMsg[K, R]),
		jobs:    make(map[K]*upgradeJob),
	}
}

// Submit runs fn for key at generation gen, on a goroutine of its own. A
// job running for key at a higher generation keeps running, and this one
// is not started; otherwise the running job's context is cancelled, and
// its result dropped. fn returns when its context is cancelled. After
// Close, Submit does nothing.
func (u *Upgrader[K, R]) Submit(key K, gen uint64, fn func(context.Context) R) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.closed {
		return
	}
	if old, ok := u.jobs[key]; ok {
		if gen < old.gen {
			return
		}
		old.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	j := &upgradeJob{gen: gen, cancel: cancel}
	u.jobs[key] = j
	u.wg.Go(func() { u.run(ctx, key, j, fn) })
}

// run is a job's goroutine: it sends fn's result unless the job was
// replaced or the Upgrader closed, and forgets the key once it is done.
func (u *Upgrader[K, R]) run(ctx context.Context, key K, j *upgradeJob, fn func(context.Context) R) {
	defer j.cancel()
	r := fn(ctx)
	if ctx.Err() == nil {
		select {
		case u.results <- UpgradeMsg[K, R]{Key: key, Gen: j.gen, Result: r}:
		case <-ctx.Done():
		}
	}
	u.mu.Lock()
	if u.jobs[key] == j {
		delete(u.jobs, key)
	}
	u.mu.Unlock()
}

// Results is the channel results come on, unbuffered: a job waits there
// until its result is read, or it is replaced. A job replaced at the
// moment its result is read may still deliver it; its Gen says which it
// is. Close closes the channel.
func (u *Upgrader[K, R]) Results() <-chan UpgradeMsg[K, R] { return u.results }

// Close cancels every job, waits for each to return, and closes Results.
// A second Close returns nil, as the first does.
func (u *Upgrader[K, R]) Close() error {
	u.mu.Lock()
	if u.closed {
		u.mu.Unlock()
		return nil
	}
	u.closed = true
	for _, j := range u.jobs {
		j.cancel()
	}
	u.mu.Unlock()
	u.wg.Wait()
	close(u.results)
	return nil
}
