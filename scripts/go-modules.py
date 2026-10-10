#!/usr/bin/env python3
"""Lists this repository's Go modules, and checks go.work against the tree
(docs/decisions/0010-MADR-nested-adapter-modules.md §2 and §4;
docs/decisions/0010-PLAN-nested-adapter-modules.md Phase 3). Ported from
go-modules.sh under docs/decisions/0017-PLAN-python-repository-scripts.md
Phase 2.

Usage:
  scripts/go-modules.py           print each module directory, relative to
                                  the root ("." for the root), one per line
  scripts/go-modules.py --check   compare the modules go.work lists with the
                                  directories of the tracked go.mod files

The list comes from `go list -m` in workspace mode, with GOWORK pointing at
the root's go.work whatever the caller's environment says, so a caller that
runs its gates with GOWORK=off still gets every module.

--check fails, naming each difference, when go.work lists a module that has
no tracked go.mod, or a tracked go.mod is missing from go.work. A go.mod the
go command ignores is skipped: one under testdata, or under a directory
whose name starts with "." or "_".

GO names the go command (default go); the tests inject one through it, as
scripts/go-fuzz_test.py does for go-fuzz.py.

The output's lines end in \\n alone, on Windows too, since shell callers
read it with $(...), which would keep a \\r.

Exit codes: 0 ok · 1 check failed · 2 usage or tool error.
"""

import os
import posixpath
import re
import subprocess
import sys

# A go.mod the go command ignores: under testdata, or a directory whose name
# starts with "." or "_".
IGNORED = re.compile(r"(^|/)(testdata|[._][^/]*)/")


def git(*args, quiet=False):
    """git's stdout without its final newline, or None when it fails. Its
    stderr passes through unless quiet, as in the shell script."""
    try:
        r = subprocess.run(["git", *args], stdout=subprocess.PIPE,
                           stderr=subprocess.DEVNULL if quiet else None, check=False)
    except OSError:
        return None
    if r.returncode != 0:
        return None
    return r.stdout.decode("utf-8", "replace").rstrip("\n")


def main(argv):
    for stream in (sys.stdout, sys.stderr):
        stream.reconfigure(newline="\n")
    go = os.environ.get("GO") or "go"
    root = git("rev-parse", "--show-toplevel")
    if root is None:
        return 2
    os.chdir(root)

    mode = "list"
    first = argv[0] if argv else ""
    if first == "--check":
        mode = "check"
    elif first != "":
        print(f"usage: {sys.argv[0]} [--check]", file=sys.stderr)
        return 2
    failed = 1 if mode == "check" else 2

    if not os.path.isfile("go.work"):
        print("go-modules: no go.work at the repository root.", file=sys.stderr)
        return failed

    # The modules go.work lists, relative to the root. git turns each
    # directory into a path relative to the repository, whatever form the go
    # command gives it: on Windows `go list` prints C:\Users\..., where git's
    # own top level is C:/Users/... (0010-PLAN deviation D1). The list is
    # go's stdout alone: what it writes to stderr, such as a warning, passes
    # through, and is never taken for a directory (0017-PLAN D2).
    try:
        r = subprocess.run([go, "list", "-m", "-f", "{{.Dir}}"], stdout=subprocess.PIPE,
                           stderr=subprocess.PIPE, env={**os.environ, "GOWORK": f"{root}/go.work"},
                           check=False)
        out, err, rc = r.stdout.decode("utf-8", "replace"), r.stderr.decode("utf-8", "replace"), r.returncode
    except OSError as e:
        out, err, rc = "", str(e), 127
    if rc != 0:
        print("go-modules: go list -m failed in workspace mode:", file=sys.stderr)
        for line in (out + err).rstrip("\n").split("\n"):
            print(f"  {line}", file=sys.stderr)
        return failed
    sys.stderr.write(err)
    listed = []
    for d in out.splitlines():
        if not d:
            continue
        if git("-C", d, "rev-parse", "--show-toplevel", quiet=True) == root:
            prefix = (git("-C", d, "rev-parse", "--show-prefix") or "").rstrip("/")
            listed.append(prefix or ".")
        else:
            listed.append(d)  # outside the repository; --check reports it

    if mode == "list":
        print("\n".join(listed))
        return 0

    # The directories of the tracked go.mod files the go command would read.
    files = git("ls-files", "--", "go.mod", ":(glob)**/go.mod") or ""
    tracked = [posixpath.dirname(f) or "." for f in files.split("\n") if f and not IGNORED.search(f)]

    status = 0
    for d in sorted(set(listed) - set(tracked)):
        print(f"go-modules: go.work lists {d}, which has no tracked go.mod.", file=sys.stderr)
        status = 1
    for d in sorted(set(tracked) - set(listed)):
        print(f"go-modules: {d}/go.mod is tracked, but go.work does not list {d} (run 'go work use {d}').",
              file=sys.stderr)
        status = 1
    return status


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
