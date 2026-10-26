package conformance

import (
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// inventoryHeader is the header of each of docs/guides/defaults.md's
// tables that TestDefaultsInventory reads.
const inventoryHeader = "| Behaviour | Default | Control | Status |"

var (
	// controlRef is a control named in backticks: a package directory, an
	// exported name, and optionally a method or a field of it, as
	// `workspace.WithoutMouse` or `command.Registry.Remove`.
	controlRef = regexp.MustCompile("`([a-z][a-z0-9]*(?:/[a-z][a-z0-9]*)*)\\.([A-Z][A-Za-z0-9]*)(?:\\.([A-Z][A-Za-z0-9]*))?`")
	// planRef and recordRef name a record by its file name.
	planRef   = regexp.MustCompile(`\d{4}-PLAN-[a-z0-9-]+\.md`)
	recordRef = regexp.MustCompile(`\d{4}-(?:MADR|PLAN)-[a-z0-9-]+\.md`)
)

// inventoryRow is one row of an inventory table, with the line it is on.
type inventoryRow struct {
	line                       int
	behaviour, control, status string
}

// inventoryRows is every row of path's inventory tables.
func inventoryRows(t *testing.T, path string) []inventoryRow {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rows []inventoryRow
	in := false
	for i, line := range strings.Split(string(b), "\n") {
		switch {
		case strings.TrimSpace(line) == inventoryHeader:
			in = true
		case !in:
		case !strings.HasPrefix(line, "|"):
			in = false
		case strings.HasPrefix(line, "| :"):
		default:
			cells := strings.Split(strings.Trim(line, "|"), "|")
			if len(cells) != 4 {
				t.Errorf("%s:%d: a row of %d cells; want 4", path, i+1, len(cells))
				continue
			}
			rows = append(rows, inventoryRow{
				line: i + 1, behaviour: strings.TrimSpace(cells[0]),
				control: strings.TrimSpace(cells[2]), status: strings.TrimSpace(cells[3]),
			})
		}
	}
	return rows
}

// resolver finds a control in the module's type-checked packages.
type resolver struct {
	mod  string
	imp  types.Importer
	pkgs map[string]*types.Package
}

// resolve reports why ref, a controlRef match, names nothing exported, or
// "" when it does.
func (r *resolver) resolve(m []string) string {
	dir, name, member := m[1], m[2], m[3]
	pkg, ok := r.pkgs[dir]
	if !ok {
		p, err := r.imp.Import(r.mod + "/" + dir)
		if err != nil {
			return "no package " + dir + ": " + err.Error()
		}
		pkg, r.pkgs[dir] = p, p
	}
	obj := pkg.Scope().Lookup(name)
	if obj == nil || !obj.Exported() {
		return dir + " exports no " + name
	}
	if member == "" {
		return ""
	}
	if _, isType := obj.(*types.TypeName); !isType {
		return dir + "." + name + " is not a type, so it has no " + member
	}
	if o, _, _ := types.LookupFieldOrMethod(obj.Type(), true, pkg, member); o == nil || !o.Exported() {
		return dir + "." + name + " has no exported field or method " + member
	}
	return ""
}

// TestDefaultsInventory holds docs/guides/defaults.md to the code
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D3 and A1): in
// every inventory table, a built row's controls each name an exported
// identifier, method or field that exists; a planned row names a PLAN that
// exists, and is not built already; a floor row's controls, the lower
// layer a program may use, exist; a decided row cites a record that exists,
// or AGENTS.md. Any other status fails.
func TestDefaultsInventory(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "docs", "guides", "defaults.md")
	rows := inventoryRows(t, path)
	if len(rows) == 0 {
		t.Fatalf("%s has no inventory rows under %q", path, inventoryHeader)
	}
	r := &resolver{mod: modulePath(t, root), imp: sourceImporter(token.NewFileSet()), pkgs: map[string]*types.Package{}}
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(root, "docs", "decisions", name))
		return err == nil
	}
	for _, row := range rows {
		where := "docs/guides/defaults.md:" + strconv.Itoa(row.line) + " (" + row.behaviour + ")"
		refs := controlRef.FindAllStringSubmatch(row.control, -1)
		var missing []string
		for _, m := range refs {
			if why := r.resolve(m); why != "" {
				missing = append(missing, why)
			}
		}
		status := strings.ToLower(row.status)
		switch {
		case strings.HasPrefix(status, "built"):
			if len(refs) == 0 {
				t.Errorf("%s: built, and names no control", where)
			}
			for _, why := range missing {
				t.Errorf("%s: built, but %s", where, why)
			}
		case strings.HasPrefix(status, "planned"):
			plan := planRef.FindString(row.status)
			switch {
			case plan == "":
				t.Errorf("%s: planned, and names no PLAN", where)
			case !exists(plan):
				t.Errorf("%s: planned by %s, which does not exist", where, plan)
			}
			if len(refs) > 0 && len(missing) == 0 {
				t.Errorf("%s: planned, but every control it names exists: mark it built", where)
			}
		case strings.HasPrefix(status, "floor"):
			for _, why := range missing {
				t.Errorf("%s: floor, but its lower layer is missing: %s", where, why)
			}
		case strings.HasPrefix(status, "decided"):
			rec := recordRef.FindString(row.status)
			switch {
			case rec == "" && !strings.Contains(row.status, "AGENTS.md"):
				t.Errorf("%s: decided, and cites no record", where)
			case rec != "" && !exists(rec):
				t.Errorf("%s: decided by %s, which does not exist", where, rec)
			}
		default:
			t.Errorf("%s: status %q; want built, planned, floor or decided", where, row.status)
		}
	}
}
