package command

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// LoadDir's limits and the expansion cap (docs/decisions/0014-PLAN-hardening.md
// Step 5, finding H4). TestLoadDirLimits and TestExpandCap use only names
// that were there before the limits, so they also run on the code before.

// many is a MapFS with n command files under dir.
func many(dir string, n int) fstest.MapFS {
	m := fstest.MapFS{}
	for i := range n {
		m[dir+"/c"+strconv.Itoa(i)+".md"] = &fstest.MapFile{Data: []byte("x")}
	}
	return m
}

// deep is a MapFS with one command file d directories below the root.
func deep(d int) fstest.MapFS {
	return fstest.MapFS{strings.Repeat("d/", d) + "c.md": {Data: []byte("x")}}
}

// lying is an fs.FS whose files say they are one byte long, whatever they
// hold, so only a limited read sees their size.
type lying struct{ fstest.MapFS }

func (l lying) Open(name string) (fs.File, error) {
	f, err := l.MapFS.Open(name)
	if err != nil {
		return nil, err
	}
	return lyingFile{f}, nil
}

type lyingFile struct{ fs.File }

func (f lyingFile) Stat() (fs.FileInfo, error) {
	fi, err := f.File.Stat()
	if err != nil {
		return nil, err
	}
	return lyingInfo{fi}, nil
}

type lyingInfo struct{ fs.FileInfo }

func (lyingInfo) Size() int64 { return 1 }

func TestLoadDirLimits(t *testing.T) {
	big := strings.Repeat("a", 256<<10)
	for _, tc := range []struct {
		name     string
		fsys     fs.FS
		loaded   int
		errs     int
		errorHas string
	}{
		{"exactly 256 KiB", fstest.MapFS{"ok.md": {Data: []byte(big)}}, 1, 0, ""},
		{"one byte over 256 KiB", fstest.MapFS{"big.md": {Data: []byte(big + "a")}}, 0, 1, "over the size limit"},
		{"8 MiB", fstest.MapFS{"huge.md": {Data: []byte(strings.Repeat("a", 8<<20))}}, 0, 1, "over the size limit"},
		{"8 MiB that says it is 1 byte", lying{fstest.MapFS{"liar.md": {Data: []byte(strings.Repeat("a", 8<<20))}}}, 0, 1, "over the size limit"},
		{"exactly 1,000 files", many("m", 1000), 1000, 0, ""},
		{"5,000 files", many("m", 5000), 1000, 1, "more command files than the limit"},
		{"8 levels down", deep(8), 1, 0, ""},
		{"9 levels down", deep(9), 0, 1, "nested deeper than the limit"},
		{"64 levels down", deep(64), 0, 1, "nested deeper than the limit"},
	} {
		start := time.Now()
		cmds, errs := LoadDir(tc.fsys, userSrc)
		if len(cmds) != tc.loaded || len(errs) != tc.errs {
			t.Errorf("%s: %d loaded, %d errors %v; want %d and %d", tc.name, len(cmds), len(errs), errs, tc.loaded, tc.errs)
			continue
		}
		if tc.errorHas != "" && !strings.Contains(errs[0].Error(), tc.errorHas) {
			t.Errorf("%s: %v; want %q", tc.name, errs[0], tc.errorHas)
		}
		if d := time.Since(start); d > 10*time.Second {
			t.Errorf("%s took %v", tc.name, d)
		}
	}
}

func TestLoadDirWithOptions(t *testing.T) {
	_, errs := LoadDirWith(fstest.MapFS{"c.md": {Data: []byte("12345")}}, userSrc, WithMaxFileBytes(4))
	if len(errs) != 1 || !errors.Is(errs[0], ErrFileTooLarge) || !strings.HasPrefix(errs[0].Error(), "command: c.md: ") {
		t.Errorf("WithMaxFileBytes(4): %v", errs)
	}
	cmds, errs := LoadDirWith(many("m", 10), userSrc, WithMaxFiles(3))
	if len(cmds) != 3 || len(errs) != 1 || !errors.Is(errs[0], ErrTooManyFiles) {
		t.Errorf("WithMaxFiles(3): %d loaded, %v", len(cmds), errs)
	}
	cmds, errs = LoadDirWith(deep(2), userSrc, WithMaxDepth(1))
	if len(cmds) != 0 || len(errs) != 1 || !errors.Is(errs[0], ErrTooDeep) {
		t.Errorf("WithMaxDepth(1): %d loaded, %v", len(cmds), errs)
	}
	cmds, errs = LoadDirWith(deep(9), userSrc, WithMaxDepth(9), WithMaxFiles(0), WithMaxFileBytes(-1))
	if len(cmds) != 1 || len(errs) != 0 {
		t.Errorf("WithMaxDepth(9), and values below 1 keeping the defaults: %d loaded, %v", len(cmds), errs)
	}
}

func TestExpandCap(t *testing.T) {
	// Sixteen $A of 64 KiB each is exactly 1 MiB; one byte more is over.
	cmds, errs := LoadDir(fstest.MapFS{"x.md": {Data: []byte(strings.Repeat("$A", 16))}}, userSrc)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	r := NewRegistry()
	r.ReplaceSource(userSrc, cmds)
	run := func(n int) (Result, error) {
		args, _ := json.Marshal(map[string]string{"A": strings.Repeat("a", n)})
		return r.Run(context.Background(), Request{ID: "user.x", Args: args, Origin: OriginCLI})
	}
	if res, err := run(64 << 10); err != nil || len(res.Text) != 1<<20 {
		t.Errorf("exactly 1 MiB: %d bytes, %v", len(res.Text), err)
	}
	res, err := run(64<<10 + 1)
	if _, ok := errors.AsType[*ArgError](err); !ok || !strings.Contains(err.Error(), "over 1048576 bytes") || res.Text != "" {
		t.Errorf("one byte over 1 MiB: %d bytes, %v; want an *ArgError naming the limit", len(res.Text), err)
	}
}
