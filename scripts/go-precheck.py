#!/usr/bin/env python3
"""Go pre-add checks: format, lint, vet, tests, tidiness, vulnerabilities.

The single implementation of the pre-add rule in AGENTS.md, called from two
places so they cannot drift: `make pre-add-check`, and the machine-wide
agent gate that runs before every agent `git commit`
(~/.agents/hooks/lib/precommit-checks.sh), which prefers
scripts/go-precheck.sh whenever the repository ships one and Go files are
staged. That file is a shim that runs this one
(docs/decisions/0017-MADR-python-repository-scripts.md item 5).

Taken from go-selfupdate-lib's (then go-core-lib's) scripts/go-precheck.sh
under docs/decisions/0001-MADR-scaffold-charm-tui-library.md §4; go-selfupdate-lib
took it from go-llmprovider-sdk, which adapted magic-cli-remote's. Ported to
Python under docs/decisions/0017-PLAN-python-repository-scripts.md Phase 5.

golangci-lint runs with this repository's .golangci.yml instead of golint.
golint is archived, and CI already runs golangci-lint; a gate weaker than CI
is not a gate. golint's own checks live on as revive's exported,
package-comments and var-naming rules in .golangci.yml.

Every module is checked on its own, in its own directory
(docs/decisions/0010-MADR-nested-adapter-modules.md §4;
docs/decisions/0010-PLAN-nested-adapter-modules.md Phase 3). The modules are
the ones go.work lists (scripts/go-modules.py). For each module:
  1. gofmt on its files;
  2. with GOWORK=off, so a module builds from its own go.mod and published
     versions only: golangci-lint for linux, darwin and windows; go vet;
     go test; go mod tidy -diff; govulncheck;
  3. go test once more in workspace mode, against the tree's other modules;
  4. for a module other than the root: no replace directive, and a
     requirement of the root, if any, on a release version (vX.Y.Z). The
     go.mod text is `go mod edit -print`'s stdout alone (0017-PLAN D4).
gofmt's own failure fails the check: a file it cannot read or parse makes it
exit non-zero with nothing on its output, and that is not "formatted"
(docs/decisions/0015-MADR-precheck-gofmt-errors.md).

Then, once: scripts/go-modules.py --check, which fails when go.work and the
tracked go.mod files disagree; and, when no file list is given (the
`make release-check` path), scripts/go-apicheck.py, which fails on an
incompatible API change since each module's previous tag that
scripts/apicheck.allow does not list
(docs/decisions/0014-PLAN-api-policy-gates.md Step 4). A file list skips it:
the API belongs to the whole module, not to the files. And
scripts/go-examples.py, which builds and runs the framework examples under
testdata/frameworks and checks the guides' excerpts of them, on the
release-check path and whenever a file list names a file under
testdata/frameworks (Step 5, deviation D5).

A Go file under a testdata directory is gofmt'd, but its directory is not
given to go vet or go test: the go command ignores testdata, and the
framework examples there import modules no module of this repository
requires.

Usage:
  scripts/go-precheck.py [file.go ...]

With no arguments it checks every tracked Go file the work tree has, in
every module; with arguments, only those that exist, in the modules that own
them (non-Go arguments are ignored, so callers can pass a whole changed-file
list). A tracked file deleted but not yet staged is not a file to commit, so
neither path checks it. The lint, tidy and vulnerability steps are
module-scoped either way: golangci-lint analyses packages, not files, so
narrowing it to a file list would report different findings than `make lint`
and the two would drift.

Exit codes: 0 all clear · 1 a check failed · 2 a required tool is missing.

Env:
  GOLANGCI_LINT=<path>         golangci-lint binary (default: $(go env GOPATH)/bin)
  GO_PRECHECK_SKIP_VULN=1      skip govulncheck (offline work)
  GO_PRECHECK_SKIP_APICHECK=1  skip the API diff gate (offline work)
  GO_PRECHECK_SKIP_EXAMPLES=1  skip the framework examples (offline work)
"""

import os
import posixpath
import re
import shutil
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
NETWORK = re.compile(r"no such host|connection refused|timeout|dial tcp|proxy", re.IGNORECASE)
RELEASE = re.compile(r"^v[0-9]+\.[0-9]+\.[0-9]+$")


def err(line):
    print(line, file=sys.stderr, flush=True)


class Check:
    def __init__(self, root):
        self.root = root
        self.failed = 0

    def fail(self, code):
        # 2 (tool missing) outranks 1 (check failed).
        self.failed = max(self.failed, code)

    @staticmethod
    def show(title, output, lines=None):
        """Report a failed step, indented: output as `$(...)` kept it,
        its last lines only when lines is given."""
        err(f"{title}:")
        text = output.rstrip("\n").split("\n")
        for line in text[-lines:] if lines else text:
            err(f"  {line}")


def run(args, cwd=None, env=None, merged=True):
    """A command's status and output: stdout and stderr together, or with
    merged False, stdout and stderr apart."""
    try:
        r = subprocess.run(args, cwd=cwd, env={**os.environ, **(env or {})}, stdout=subprocess.PIPE,
                           stderr=subprocess.STDOUT if merged else subprocess.PIPE, check=False)
    except OSError as e:
        return (127, str(e)) if merged else (127, "", str(e))
    text = r.stdout.decode("utf-8", "replace")
    if merged:
        return r.returncode, text
    return r.returncode, text, r.stderr.decode("utf-8", "replace")


def need(tool):
    if shutil.which(tool):
        return True
    err(f"go-precheck: {tool} not found in PATH.")
    if tool == "govulncheck":
        err("  install: go install golang.org/x/vuln/cmd/govulncheck@v1.8.0")
    return False


def executable(path):
    """path, or on Windows path.exe, when it is an executable file, as Git
    Bash's -x test finds it; else None."""
    for p in (path, path + ".exe") if os.name == "nt" else (path,):
        if os.path.isfile(p) and os.access(p, os.X_OK):
            return p
    return None


def owner(f, modules):
    """The module directory that owns f, deepest first."""
    for m in modules:
        if m == ".":
            return "."
        if f.startswith(m + "/"):
            return m
    return "."


def packages(mfiles, m):
    """The packages the files belong to, relative to the module."""
    dirs = set()
    for f in mfiles:
        if "/testdata/" in "/" + f.removeprefix("./"):
            continue
        d = posixpath.dirname(f) or "."
        if m == ".":
            dirs.add(d)
        elif d == m:
            dirs.add(".")
        else:
            dirs.add(d.removeprefix(m + "/"))
    return [f"./{d}" for d in sorted(dirs)]


def vuln(c, cwd):
    """govulncheck over a module. It reports *called* vulnerabilities, so it
    is a property of the whole build rather than of the edited files.

    Exit status 3 is govulncheck's "vulnerabilities found" and always fails.
    Only another non-zero status whose output looks like a network failure is
    treated as an unreachable database: matching those words on a status-3
    run would let a finding whose trace names a `proxy` package or a
    `Timeout` function through."""
    rc, out = run(["govulncheck", "./..."], cwd=cwd, env={"GOWORK": "off"})
    if rc == 3:
        c.show("govulncheck: vulnerabilities found", out, 30)
        c.fail(1)
    elif rc != 0:
        if NETWORK.search(out):
            err("govulncheck: could not reach the vulnerability database; skipped.")
        else:
            c.show(f"govulncheck: failed (exit {rc})", out, 30)
            c.fail(1)


def requirements(c, m, root_path):
    """A module other than the root has no replace directive, and requires
    the root, if at all, at a release version."""
    rc, printed, warn = run(["go", "mod", "edit", "-print"], cwd=m, env={"GOWORK": "off"}, merged=False)
    if rc != 0:
        c.show(f"go mod edit ({m})", printed + warn)
        c.fail(1)
        return
    sys.stderr.write(warn)
    lines = printed.rstrip("\n").split("\n")
    if any(re.match(r"^replace[ \t\n\r\f\v]", line) for line in lines):
        shown, r = [], False
        for line in lines:
            if line.startswith("replace ("):
                r = True
            if r or line.startswith("replace "):
                shown.append(line)
            if r and line.startswith(")"):
                r = False
        c.show(f"go.mod ({m}): a replace directive (a module builds from published versions only)",
               "\n".join(shown))
        c.fail(1)
    versions, inreq = [], False
    for line in lines:
        fields = line.split()
        if line.startswith("require ("):
            inreq = True
            continue
        if inreq and line.startswith(")"):
            inreq = False
            continue
        if inreq and fields and fields[0] == root_path:
            versions.append(fields[1] if len(fields) > 1 else "")
        if len(fields) > 1 and fields[0] == "require" and fields[1] == root_path:
            versions.append(fields[2] if len(fields) > 2 else "")
    ver = "\n".join(versions)
    if ver and not any(RELEASE.match(v) for v in versions):
        err(f"go.mod ({m}): requires {root_path} at {ver}, which is not a release version (vX.Y.Z).")
        c.fail(1)


def check_module(c, m, mfiles, pkgs, golangci, have_vuln):
    """Steps 1 to 4 for module m."""
    # 1. gofmt. Its exit status counts as well as its list: a file it cannot
    # read or parse makes it fail with nothing on its output.
    rc, unformatted, gofmt_err = run(["gofmt", "-l", *mfiles], merged=False)
    if rc != 0:
        c.show(f"gofmt: failed (exit {rc})", gofmt_err)
        c.fail(1)
    unformatted = unformatted.rstrip("\n")
    if unformatted:
        err("gofmt: these files are not formatted (run 'gofmt -w <file>'):")
        for line in unformatted.split("\n"):
            err(f"  {line}")
        c.fail(1)

    off = {"GOWORK": "off"}
    # 2. golangci-lint, with this repository's configuration: the same
    # command `make lint` runs, so a commit cannot pass a weaker check than
    # CI applies. It runs once per target the code builds for, with cgo off,
    # because a host-only run never sees a *_windows.go or *_unix.go file of
    # another OS (go-selfupdate-lib
    # docs/decisions/0002-MADR-rehome-selfupdate-from-mcplib.md §5).
    if golangci:
        for goos in ("linux", "darwin", "windows"):
            rc, out = run([golangci, "run", "-c", os.path.join(c.root, ".golangci.yml"), "./..."], cwd=m,
                          env={**off, "CGO_ENABLED": "0", "GOOS": goos})
            if rc != 0:
                c.show(f"golangci-lint ({m}, GOOS={goos})", out, 40)
                c.fail(1)

    # go vet and go test, over the packages the files belong to. AGENTS.md
    # requires both on the touched packages. Files under testdata alone
    # leave no package.
    if pkgs:
        rc, out = run(["go", "vet", *pkgs], cwd=m, env=off)
        if rc != 0:
            c.show(f"go vet ({m})", out)
            c.fail(1)
        rc, out = run(["go", "test", *pkgs], cwd=m, env=off)
        if rc != 0:
            c.show(f"go test ({m})", out, 40)
            c.fail(1)
    rc, out = run(["go", "mod", "tidy", "-diff"], cwd=m, env=off)
    if rc != 0:
        c.show(f"go mod tidy -diff ({m})", out, 40)
        c.fail(1)
    if have_vuln:
        vuln(c, m)

    # 3. go test in workspace mode, against the tree's other modules.
    if pkgs:
        rc, out = run(["go", "test", *pkgs], cwd=m, env={"GOWORK": os.path.join(c.root, "go.work")})
        if rc != 0:
            c.show(f"go test, workspace mode ({m})", out, 40)
            c.fail(1)


def main(argv):
    for stream in (sys.stdout, sys.stderr):
        stream.reconfigure(newline="\n")
    rc, root, _ = run(["git", "rev-parse", "--show-toplevel"], merged=False)
    if rc != 0:
        return 1
    root = root.rstrip("\n")
    os.chdir(root)
    c = Check(root)

    # Whether the framework examples run: on a whole-tree run, or when a
    # file under testdata/frameworks is given.
    examples_run = not argv or any(f.removeprefix("./").startswith("testdata/frameworks/") for f in argv)

    # Collect the Go files to check.
    if argv:
        files = [f for f in argv if f.endswith(".go") and os.path.isfile(f)]
    else:
        _, listed, _ = run(["git", "ls-files", "*.go"], merged=False)
        files = [f for f in listed.split("\n") if f and os.path.isfile(f)]

    # Nothing to check is not an error: a docs-only change, or a tree with
    # no Go code yet. Return before any tool runs.
    if not files:
        print("go-precheck: no Go files to check.", flush=True)
        return 0

    if not need("gofmt") or not need("go"):
        return 2
    golangci_path = os.environ.get("GOLANGCI_LINT") or ""
    if not golangci_path:
        _, gopath, _ = run(["go", "env", "GOPATH"], merged=False)
        golangci_path = gopath.strip() + "/bin/golangci-lint"
    golangci = executable(golangci_path)
    if not golangci:
        err(f"go-precheck: golangci-lint not found at {golangci_path}.")
        err("  install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0")
        c.fail(2)
    have_vuln = True
    if os.environ.get("GO_PRECHECK_SKIP_VULN", "0") == "1":
        err("govulncheck: skipped (GO_PRECHECK_SKIP_VULN=1)")
        have_vuln = False
    elif not need("govulncheck"):
        c.fail(2)
        have_vuln = False

    # The modules, longest directory first, so that a file belongs to the
    # deepest module containing it.
    rc, module_list, _ = run([sys.executable, os.path.join(HERE, "go-modules.py")], merged=False)
    if rc != 0:
        err("go-precheck: could not list the modules (scripts/go-modules.py).")
        return 1
    in_order = [m for m in module_list.split("\n") if m]
    deepest = sorted(in_order, key=len, reverse=True)
    _, root_path, _ = run(["go", "list", "-m"], env={"GOWORK": "off"}, merged=False)
    root_path = root_path.rstrip("\n")

    checked_files = checked_modules = 0
    # The root, then the other modules, in go.work's order.
    for m in in_order:
        mfiles = [f for f in files if owner(f, deepest) == m]
        if not mfiles:
            continue
        checked_files += len(mfiles)
        checked_modules += 1
        err(f"go-precheck: module {m} ({len(mfiles)} file(s))")
        pkgs = packages(mfiles, m) if argv else ["./..."]
        check_module(c, m, mfiles, pkgs, golangci, have_vuln)
        # 4. A nested module's requirements.
        if m != ".":
            requirements(c, m, root_path)

    # Once: go.work lists exactly the tree's modules.
    rc, out = run([sys.executable, os.path.join(HERE, "go-modules.py"), "--check"])
    if rc != 0:
        c.show("go.work", out)
        c.fail(1)

    # Once, for the whole tree: no unlisted incompatible API change.
    apicheck = ""
    if not argv:
        if os.environ.get("GO_PRECHECK_SKIP_APICHECK", "0") == "1":
            err("apicheck: skipped (GO_PRECHECK_SKIP_APICHECK=1)")
        else:
            rc, out = run([sys.executable, os.path.join(HERE, "go-apicheck.py")])
            if rc != 0:
                c.show("apicheck", out)
                c.fail(rc)
            else:
                apicheck = ", apicheck"

    # Once: the framework examples build, run as their cases say, and match
    # the guides' excerpts.
    examples = ""
    if examples_run:
        if os.environ.get("GO_PRECHECK_SKIP_EXAMPLES", "0") == "1":
            err("examples: skipped (GO_PRECHECK_SKIP_EXAMPLES=1)")
        else:
            rc, out = run([sys.executable, os.path.join(HERE, "go-examples.py")])
            if rc != 0:
                c.show("examples", out)
                c.fail(rc)
            else:
                examples = ", examples"

    if c.failed == 0:
        print(f"go-precheck: {checked_files} file(s) clean in {checked_modules} module(s) (gofmt, golangci-lint, "
              f"go vet, go test, go mod tidy, govulncheck{apicheck}{examples}).", flush=True)
    return c.failed


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
