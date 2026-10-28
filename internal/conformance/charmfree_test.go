package conformance

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// cores is every package whose non-test import closure holds no Charm
// module but x/ansi, by directory relative to the repository root, sorted.
// A line-mode CLI that links no TUI framework may import any of them
// (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md D1 and
// D11). A package joins in the commit that makes it clean, and leaves only
// by a record.
var cores = []string{
	"ask",
	"command",
	"fuzzy",
	"glyph",
	"inline",
	"inline/inlinetest",
	"internal/enum",
	"internal/lineinput",
	"internal/sanitize",
	"keymap",
	"layout",
	"msgcat",
	"pick",
	"safetext",
	"termcap",
	"termkey",
	"termsvc",
	"when",
}

// charmGOOS is each operating system a core's closure is listed for: build
// constraints are read for one target at a time.
var charmGOOS = []string{"linux", "darwin", "windows"}

// charm reports whether p, an import path, is a Charm package no core may
// link: anything under charm.land/, and anything under
// github.com/charmbracelet/ except github.com/charmbracelet/x/ansi and the
// packages beneath it (0020-MADR D1).
func charm(p string) bool {
	const ansi = "github.com/charmbracelet/x/ansi"
	switch {
	case strings.HasPrefix(p, "charm.land/"):
		return true
	case p == ansi || strings.HasPrefix(p, ansi+"/"):
		return false
	}
	return strings.HasPrefix(p, "github.com/charmbracelet/")
}

func TestCharmPredicate(t *testing.T) {
	for p, want := range map[string]bool{
		"charm.land/bubbletea/v2":                    true,
		"charm.land/lipgloss/v2":                     true,
		"github.com/charmbracelet/colorprofile":      true,
		"github.com/charmbracelet/ultraviolet":       true,
		"github.com/charmbracelet/x/term":            true,
		"github.com/charmbracelet/x/ansix":           true,
		"github.com/charmbracelet/x/ansi":            false,
		"github.com/charmbracelet/x/ansi/parser":     false,
		"github.com/clipperhouse/uax29/v2/graphemes": false,
		"strings": false,
	} {
		if got := charm(p); got != want {
			t.Errorf("charm(%q) = %v, want %v", p, got, want)
		}
	}
}

// depGraph is one go list -deps run: each package's direct imports,
// whether it is in the standard library, and the packages go list could
// not load, with its error.
type depGraph struct {
	imports  map[string][]string
	standard map[string]bool
	broken   map[string]string
}

// listDeps runs go list -e -deps over pkgs, directories relative to root,
// in root with GOWORK=off, CGO_ENABLED=0 and goos, and returns the graph
// it prints: every package in the closure, test files left out.
func listDeps(t *testing.T, root, goos string, pkgs []string) depGraph {
	t.Helper()
	args := []string{"list", "-e", "-deps", "-f", "{{.ImportPath}}\t{{if .Error}}{{.Error}}{{end}}\t{{.Standard}}\t{{join .Imports \" \"}}"}
	for _, p := range pkgs {
		args = append(args, "./"+p)
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0", "GOOS="+goos)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := errors.AsType[*exec.ExitError](err); ok {
			t.Fatalf("go list for GOOS=%s: %v\n%s", goos, err, ee.Stderr)
		}
		t.Fatalf("go list for GOOS=%s: %v", goos, err)
	}
	g := depGraph{imports: map[string][]string{}, standard: map[string]bool{}, broken: map[string]string{}}
	for line := range strings.Lines(string(out)) {
		fields := strings.SplitN(strings.TrimRight(line, "\r\n"), "\t", 4)
		if len(fields) != 4 {
			t.Fatalf("go list for GOOS=%s printed %q", goos, line)
		}
		path, failure, standard, imports := fields[0], fields[1], fields[2], fields[3]
		if failure != "" {
			g.broken[path] = failure
			continue
		}
		g.imports[path] = strings.Fields(imports)
		g.standard[path] = standard == "true"
	}
	return g
}

// chains is, for each package from reaches that stop is true for, the
// shortest import chain to it, from first: a breadth-first walk over
// imports in lexical order, so that of two chains of one length the
// lexically first is kept. The walk does not go on through a package stop
// is true for, so each chain names the one a package stop is false for
// imports, not the ones beneath it.
func chains(g depGraph, from string, stop func(string) bool) [][]string {
	prev := map[string]string{from: ""}
	queue := []string{from}
	var hits []string
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if stop(p) {
			hits = append(hits, p)
			continue
		}
		imports := slices.Clone(g.imports[p])
		slices.Sort(imports)
		for _, next := range imports {
			if _, seen := prev[next]; !seen {
				prev[next] = p
				queue = append(queue, next)
			}
		}
	}
	slices.Sort(hits)
	var chains [][]string
	for _, h := range hits {
		var chain []string
		for p := h; p != ""; p = prev[p] {
			chain = append(chain, p)
		}
		slices.Reverse(chain)
		chains = append(chains, chain)
	}
	return chains
}

// TestCoresAreCharmFree fails when a core's non-test import closure, for
// any of charmGOOS, holds a Charm package other than x/ansi, naming the
// core, the package and the shortest chain to it; and when cores lists a
// package go list does not find, so a rename cannot drop a core unseen
// (0020-MADR D11).
func TestCoresAreCharmFree(t *testing.T) {
	root := repoRoot(t)
	mod := modulePath(t, root)
	short := func(p string) string { return strings.TrimPrefix(p, mod+"/") }
	for _, goos := range charmGOOS {
		t.Run(goos, func(t *testing.T) {
			g := listDeps(t, root, goos, cores)
			for _, core := range cores {
				path := mod + "/" + core
				if _, ok := g.imports[path]; !ok {
					t.Errorf("cores lists %s, which go list does not find (%s)", core, g.broken["./"+core])
					continue
				}
				for _, chain := range chains(g, path, charm) {
					names := make([]string, len(chain))
					for i, p := range chain {
						names[i] = short(p)
					}
					t.Errorf("%s (GOOS=%s) links %s: %s", core, goos, chain[len(chain)-1], strings.Join(names, " → "))
				}
			}
		})
	}
}

// narrowCore is a core whose record allows a narrower closure than the
// cores' rule: the standard library, the core, and the packages of this
// module it names, by directory relative to the repository root.
type narrowCore struct {
	core   string
	allow  []string
	record string
}

// narrow is every core with a narrower rule
// (docs/decisions/0021-MADR-termkey-closure-gate.md N1). A core joins by
// the record that states its rule, with a depguard rule beside it that
// allows the same, as termkeystd does termkey's (N4) and safetextstd
// safetext's (A1).
var narrow = []narrowCore{
	{core: "termkey", allow: []string{"internal/enum"}, record: "0020-MADR D7, 0021-MADR N1"},
	{core: "safetext", allow: []string{"glyph", "internal/enum"}, record: "0020-MADR D9, 0021-MADR A1"},
	{core: "msgcat", record: "0024-MADR D9"},
}

// strayFrom reports, for g, whether a package is one a narrow core may not
// link: not in the standard library, and not one of allowed.
func strayFrom(g depGraph, allowed map[string]bool) func(string) bool {
	return func(p string) bool { return !g.standard[p] && !allowed[p] }
}

// closure is every package from reaches in g, from included.
func closure(g depGraph, from string) map[string]bool {
	seen := map[string]bool{from: true}
	for queue := []string{from}; len(queue) > 0; queue = queue[1:] {
		for _, next := range g.imports[queue[0]] {
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return seen
}

// TestNarrowClosures fails when a narrow core's non-test closure, for any
// of charmGOOS, holds a package its record does not allow, naming the core,
// the package and the shortest chain to it; when narrow lists a package
// cores does not, or go list does not find; and when it allows a package
// the closure does not hold (0021-MADR N2).
func TestNarrowClosures(t *testing.T) {
	root := repoRoot(t)
	mod := modulePath(t, root)
	short := func(p string) string { return strings.TrimPrefix(p, mod+"/") }
	dirs := make([]string, 0, len(narrow))
	for _, n := range narrow {
		if !slices.Contains(cores, n.core) {
			t.Errorf("narrow lists %s, which cores does not", n.core)
		}
		dirs = append(dirs, n.core)
	}
	for _, goos := range charmGOOS {
		t.Run(goos, func(t *testing.T) {
			g := listDeps(t, root, goos, dirs)
			for _, n := range narrow {
				path := mod + "/" + n.core
				if _, ok := g.imports[path]; !ok {
					t.Errorf("narrow lists %s, which go list does not find (%s)", n.core, g.broken["./"+n.core])
					continue
				}
				allowed := map[string]bool{path: true}
				for _, a := range n.allow {
					allowed[mod+"/"+a] = true
				}
				for _, chain := range chains(g, path, strayFrom(g, allowed)) {
					names := make([]string, len(chain))
					for i, p := range chain {
						names[i] = short(p)
					}
					t.Errorf("%s (GOOS=%s) links %s, which %s does not allow: %s",
						n.core, goos, chain[len(chain)-1], n.record, strings.Join(names, " → "))
				}
				held := closure(g, path)
				for _, a := range n.allow {
					if !held[mod+"/"+a] {
						t.Errorf("narrow allows %s for %s, which its closure does not hold (GOOS=%s)", a, n.core, goos)
					}
				}
			}
		})
	}
}

// TestNarrowChains: over a graph of a core, a standard package, an allowed
// one and two stray ones, the walk with the narrow predicate gives the
// chain to each stray package and stops there, and passes through the
// standard and the allowed packages.
func TestNarrowChains(t *testing.T) {
	g := depGraph{
		imports: map[string][]string{
			"m/core": {"fmt", "m/ok", "m/bad"},
			"fmt":    {"io"},
			"io":     nil,
			"m/ok":   {"x/mod"},
			"m/bad":  {"m/deep"},
			"m/deep": nil,
			"x/mod":  nil,
		},
		standard: map[string]bool{"fmt": true, "io": true},
	}
	got := chains(g, "m/core", strayFrom(g, map[string]bool{"m/core": true, "m/ok": true}))
	want := [][]string{{"m/core", "m/bad"}, {"m/core", "m/ok", "x/mod"}}
	if !slices.EqualFunc(got, want, slices.Equal[[]string]) {
		t.Errorf("chains = %q, want %q", got, want)
	}
	if held := closure(g, "m/core"); len(held) != 7 || !held["m/deep"] {
		t.Errorf("closure = %v, want all seven packages", held)
	}
}
