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
	"fuzzy",
	"glyph",
	"internal/enum",
	"internal/sanitize",
	"layout",
	"safetext",
	"termcap",
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

// depGraph is one go list -deps run: each package's direct imports, and
// the packages go list could not load, with its error.
type depGraph struct {
	imports map[string][]string
	broken  map[string]string
}

// listDeps runs go list -e -deps over pkgs, directories relative to root,
// in root with GOWORK=off, CGO_ENABLED=0 and goos, and returns the graph
// it prints: every package in the closure, test files left out.
func listDeps(t *testing.T, root, goos string, pkgs []string) depGraph {
	t.Helper()
	args := []string{"list", "-e", "-deps", "-f", "{{.ImportPath}}\t{{if .Error}}{{.Error}}{{end}}\t{{join .Imports \" \"}}"}
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
	g := depGraph{imports: map[string][]string{}, broken: map[string]string{}}
	for line := range strings.Lines(string(out)) {
		fields := strings.SplitN(strings.TrimRight(line, "\r\n"), "\t", 3)
		if len(fields) != 3 {
			t.Fatalf("go list for GOOS=%s printed %q", goos, line)
		}
		path, failure, imports := fields[0], fields[1], fields[2]
		if failure != "" {
			g.broken[path] = failure
			continue
		}
		g.imports[path] = strings.Fields(imports)
	}
	return g
}

// charmChains is, for each Charm package from reaches, the shortest import
// chain to it, from first: a breadth-first walk over imports in lexical
// order, so that of two chains of one length the lexically first is kept.
// The walk does not go on through a Charm package, so each chain names the
// Charm package a non-Charm one imports, not the ones beneath it.
func charmChains(g depGraph, from string) [][]string {
	prev := map[string]string{from: ""}
	queue := []string{from}
	var hits []string
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if charm(p) {
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
				for _, chain := range charmChains(g, path) {
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
