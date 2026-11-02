package frame_test

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/maccavelli/go-tui-lib/v2/frame"
)

type upgrader = frame.Upgrader[string, string]

// mustResult fails the test unless u's next result, once the bubble has
// blocked, is gen's.
func mustResult(t *testing.T, u *upgrader, key string, gen uint64, result string) {
	t.Helper()
	synctest.Wait()
	select {
	case m := <-u.Results():
		if m.Key != key || m.Gen != gen || m.Result != result {
			t.Fatalf("result %+v; want {%s %d %s}", m, key, gen, result)
		}
	default:
		t.Fatalf("no result; want {%s %d %s}", key, gen, result)
	}
}

// noResult fails the test when a result waits once the bubble has blocked.
func noResult(t *testing.T, u *upgrader, why string) {
	t.Helper()
	synctest.Wait()
	select {
	case m := <-u.Results():
		t.Fatalf("%s: result %+v", why, m)
	default:
	}
}

// blocked is a job that returns "cancelled" once its context is, and
// counts the jobs cancelled.
func blocked(cancelled *atomic.Int32) func(context.Context) string {
	return func(ctx context.Context) string {
		<-ctx.Done()
		cancelled.Add(1)
		return "cancelled"
	}
}

// held is a job that returns r once release is closed, whatever its
// context.
func held(release chan struct{}, r string) func(context.Context) string {
	return func(context.Context) string {
		<-release
		return r
	}
}

func done(r string) func(context.Context) string {
	return func(context.Context) string { return r }
}

// TestUpgraderReplaces: a newer job for a key cancels the older one's
// context, and only its result comes; another key's job is its own
// (docs/decisions/0009-MADR-streaming-content-engine.md A1 §3).
func TestUpgraderReplaces(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		u := frame.NewUpgrader[string, string]()
		defer u.Close()
		var cancelled atomic.Int32
		u.Submit("a", 1, blocked(&cancelled))
		u.Submit("b", 1, done("b1"))
		synctest.Wait()
		u.Submit("a", 2, done("a2"))
		synctest.Wait()
		if cancelled.Load() != 1 {
			t.Fatalf("%d jobs cancelled; want the replaced one", cancelled.Load())
		}
		got := map[string]frame.UpgradeMsg[string, string]{}
		for range 2 {
			m := <-u.Results()
			got[m.Key] = m
		}
		if got["a"].Gen != 2 || got["a"].Result != "a2" || got["b"].Result != "b1" {
			t.Fatalf("results %+v; want a at 2 and b at 1", got)
		}
		noResult(t, u, "the replaced job")
	})
}

// TestUpgraderStale: a replaced job that ignores its context and returns
// has its result dropped.
func TestUpgraderStale(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		u := frame.NewUpgrader[string, string]()
		defer u.Close()
		first, second := make(chan struct{}), make(chan struct{})
		u.Submit("a", 1, held(first, "stale"))
		u.Submit("a", 2, held(second, "fresh"))
		close(first)
		noResult(t, u, "the replaced job returned")
		close(second)
		mustResult(t, u, "a", 2, "fresh")
	})
}

// TestUpgraderGenerations: a job submitted at a lower generation than the
// key's running job is not started; at the same one, it replaces it; and
// a key is forgotten once its job ends, so that a lower generation runs
// after it (docs/decisions/0009-PLAN-streaming-content-engine.md D8).
func TestUpgraderGenerations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		u := frame.NewUpgrader[string, string]()
		defer u.Close()
		release := make(chan struct{})
		var ran atomic.Int32
		u.Submit("a", 5, held(release, "five"))
		u.Submit("a", 3, func(context.Context) string { ran.Add(1); return "three" })
		close(release)
		mustResult(t, u, "a", 5, "five")
		synctest.Wait() // the job ends once its result is read
		if ran.Load() != 0 {
			t.Fatal("the job at a lower generation ran")
		}
		var cancelled atomic.Int32
		u.Submit("a", 3, blocked(&cancelled))
		u.Submit("a", 3, done("again"))
		mustResult(t, u, "a", 3, "again")
		if cancelled.Load() != 1 {
			t.Fatal("the same generation did not replace the running job")
		}
	})
}

// TestUpgraderClose: Close cancels every job, waits for each, and closes
// Results; a Submit after it runs nothing; a second Close returns nil. The
// bubble ends with no goroutine left.
func TestUpgraderClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		u := frame.NewUpgrader[string, string]()
		var cancelled atomic.Int32
		u.Submit("a", 1, blocked(&cancelled))
		u.Submit("b", 1, blocked(&cancelled))
		u.Submit("c", 1, done("unread"))
		synctest.Wait()
		if err := u.Close(); err != nil {
			t.Fatalf("Close = %v", err)
		}
		if cancelled.Load() != 2 {
			t.Fatalf("%d jobs cancelled by Close; want 2", cancelled.Load())
		}
		if m, ok := <-u.Results(); ok {
			t.Fatalf("a result after Close: %+v", m)
		}
		var ran atomic.Int32
		u.Submit("d", 1, func(context.Context) string { ran.Add(1); return "" })
		synctest.Wait()
		if ran.Load() != 0 {
			t.Fatal("a job submitted after Close ran")
		}
		if err := u.Close(); err != nil {
			t.Fatalf("a second Close = %v", err)
		}
	})
}
