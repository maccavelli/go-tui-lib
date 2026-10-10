#!/usr/bin/env python3
"""Fails on an incompatible change to a module's exported API since the
module's previous release tag
(docs/decisions/0014-MADR-native-integration-api.md W0.2;
docs/decisions/0014-PLAN-api-policy-gates.md Step 4). Ported from
go-apicheck.sh under docs/decisions/0017-PLAN-python-repository-scripts.md
Phase 3.

For each module scripts/go-modules.py lists, with GOWORK=off:
  1. The base is the newest <prefix>vX.Y.Z tag merged into HEAD that does
     not contain HEAD, so a run on a tagged commit compares with the tag
     before it. The prefix is empty for the root and "<dir>/" for a nested
     module. A module with no such tag has never been released, and is
     skipped.
  2. apidiff exports the base's API, from `git archive <tag>`, and the
     tree's, each in module mode.
  3. apidiff compares the two, printing incompatible changes only. It exits
     0 even when it prints one, so its output is judged line by line
     against scripts/apicheck.allow: a line the file does not list fails,
     and so does a listed line apidiff no longer prints, which is stale.

scripts/apicheck.allow holds "<module dir><TAB><apidiff line>" entries, and
"#" comments. Each block names the PLAN that allows its changes; a
release's close-out removes them.

Usage: scripts/go-apicheck.py

Env:
  APIDIFF_VERSION              the golang.org/x/exp version apidiff runs at
                               (default: the Makefile's pin)
  APIDIFF=<path>               an apidiff binary to use instead (tests)
  APICHECK_ALLOW=<path>        the allow file (default scripts/apicheck.allow)
  GO_PRECHECK_SKIP_APICHECK=1  skip (offline work)
  GO                           the go command (default go)

The module path is `go list -m`'s stdout alone; what go writes to stderr
passes through (0017-PLAN D3).

Exit codes: 0 clean or skipped · 1 an unlisted or stale line · 2 a tool or
git failure.
"""

import os
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
VERSION_LINE = re.compile(r"^APIDIFF_VERSION[ \t]*\?=[ \t]*(.*)$", re.MULTILINE)


class Stop(Exception):
    """Stop with this exit status, the message already printed."""

    def __init__(self, code):
        super().__init__(code)
        self.code = code


def out(line):
    print(line, flush=True)


def err(line):
    print(line, file=sys.stderr, flush=True)


def indented(text):
    """Text as the shell script printed it through sed 's/^/  /'."""
    for line in text.rstrip("\n").split("\n"):
        err(f"  {line}")


def run(args, cwd=None, env=None, merged=False):
    """A command's status and output: stdout and stderr together when
    merged, else stdout, with stderr passing through. A command that cannot
    start has status 127 and its error as output."""
    try:
        r = subprocess.run(args, cwd=cwd, env={**os.environ, **(env or {})}, stdout=subprocess.PIPE,
                           stderr=subprocess.STDOUT if merged else None, check=False)
    except OSError as e:
        return 127, str(e)
    return r.returncode, r.stdout.decode("utf-8", "replace")


def apidiff_binary(work, go):
    """apidiff, at the Makefile's pin unless a binary is given."""
    given = os.environ.get("APIDIFF")
    if given:
        return given
    version = os.environ.get("APIDIFF_VERSION")
    if not version:
        with open(os.path.join(HERE, "..", "Makefile"), encoding="utf-8") as f:
            m = VERSION_LINE.search(f.read())
        version = m.group(1) if m else ""
    if not version:
        err("apicheck: APIDIFF_VERSION is not set, and the Makefile names none.")
        raise Stop(2)
    rc, text = run([go, "install", f"golang.org/x/exp/cmd/apidiff@{version}"],
                   env={"GOBIN": os.path.join(work, "bin"), "GOWORK": "off"}, merged=True)
    if rc != 0:
        err(f"apicheck: cannot install apidiff@{version}:")
        indented(text)
        raise Stop(2)
    binary = os.path.join(work, "bin", "apidiff")
    if os.access(binary + ".exe", os.X_OK):
        binary += ".exe"
    return binary


def extract(tag, module, base):
    """The tag's tree, or a nested module's part of it, under base."""
    args = ["git", "archive", tag] + ([] if module == "." else ["--", module])
    try:
        proc = subprocess.Popen(args, stdout=subprocess.PIPE)
    except OSError:
        raise Stop(2)
    try:
        with tarfile.open(fileobj=proc.stdout, mode="r|") as archive:
            archive.extractall(base, filter="fully_trusted")
    except tarfile.TarError:
        proc.kill()
        proc.wait()
        raise Stop(2)
    if proc.wait() != 0:
        raise Stop(2)


def check_module(m, go, apidiff, work, i, printed):
    prefix = "" if m == "." else f"{m}/"
    rc, tags = run(["git", "tag", "--merged", "HEAD", "--no-contains", "HEAD", "-l",
                    f"{prefix}v[0-9]*.[0-9]*.[0-9]*", "--sort=-v:refname"])
    tag = tags.split("\n")[0] if rc == 0 else ""
    if not tag:
        out(f"apicheck: {m}: no previous tag, skipped")
        return
    try:
        r = subprocess.run([go, "list", "-m"], cwd=m, env={**os.environ, "GOWORK": "off"},
                           stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
        mod, warn, rc = r.stdout.decode("utf-8", "replace"), r.stderr.decode("utf-8", "replace"), r.returncode
    except OSError as e:
        mod, warn, rc = "", str(e), 127
    if rc != 0:
        err(f"apicheck: {m}: go list -m failed: {(mod + warn).rstrip(chr(10))}")
        raise Stop(2)
    sys.stderr.write(warn)
    mod = mod.rstrip("\n")

    # Numbered, not named after the module: "base-." would end in a dot,
    # which Windows strips from a path, so a native program could not enter
    # it.
    base = os.path.join(work, f"base{i}")
    os.makedirs(base)
    extract(tag, m, base)
    old_api, new_api = os.path.join(work, "old.api"), os.path.join(work, "new.api")
    for what, cwd, api in ((f"exporting {tag}'s API", os.path.join(base, m), old_api),
                           ("exporting the tree's API", m, new_api)):
        rc, text = run([apidiff, "-m", "-w", api, mod], cwd=cwd, env={"GOWORK": "off"}, merged=True)
        if rc != 0:
            err(f"apicheck: {m}: {what} failed:")
            indented(text)
            raise Stop(2)
    try:
        r = subprocess.run([apidiff, "-m", "-incompatible", old_api, new_api], stdout=subprocess.PIPE,
                           stderr=subprocess.PIPE, check=False)
        diff, diff_err, rc = r.stdout.decode("utf-8", "replace"), r.stderr.decode("utf-8", "replace"), r.returncode
    except OSError as e:
        diff, diff_err, rc = "", str(e), 127
    if rc != 0:
        err(f"apicheck: {m}: apidiff failed:")
        for line in diff_err.splitlines():
            err(f"  {line}")
        raise Stop(2)
    n = 0
    for line in diff.split("\n"):
        line = line.removesuffix("\r")
        if not line:
            continue
        printed.add(f"{m}\t{line}")
        n += 1
    out(f"apicheck: {m}: against {tag}, {n} incompatible change(s)")


def c_sorted(lines):
    """Sorted as LC_ALL=C sort sorts: by bytes."""
    return sorted(lines, key=lambda s: s.encode("utf-8"))


def main():
    for stream in (sys.stdout, sys.stderr):
        stream.reconfigure(newline="\n")
    go = os.environ.get("GO") or "go"
    if os.environ.get("GO_PRECHECK_SKIP_APICHECK", "0") == "1":
        err("apicheck: skipped (GO_PRECHECK_SKIP_APICHECK=1)")
        return 0
    rc, root = run(["git", "rev-parse", "--show-toplevel"])
    if rc != 0:
        return 2
    root = root.rstrip("\n")
    os.chdir(root)

    allow = os.environ.get("APICHECK_ALLOW") or os.path.join(root, "scripts", "apicheck.allow")
    if not os.path.isfile(allow):
        err(f"apicheck: no allow file at {allow}.")
        return 2

    work = tempfile.mkdtemp()
    try:
        apidiff = apidiff_binary(work, go)

        # The allow file's entries, without comments or blank lines.
        with open(allow, "rb") as f:
            text = f.read().decode("utf-8", "replace")
        allowed = {line.replace("\r", "") for line in text.split("\n")
                   if line and not line.startswith("#") and line.strip()}

        rc, modules = run([sys.executable, os.path.join(HERE, "go-modules.py")])
        if rc != 0:
            err("apicheck: scripts/go-modules.py failed.")
            return 2

        printed = set()
        for i, m in enumerate((m for m in modules.split("\n") if m), start=1):
            check_module(m, go, apidiff, work, i, printed)

        status = 0
        unlisted = c_sorted(printed - allowed)
        stale = c_sorted(allowed - printed)
        if unlisted:
            err("apicheck: incompatible changes that scripts/apicheck.allow does not list:")
            for line in unlisted:
                err(f"  {line}")
            status = 1
        if stale:
            err("apicheck: scripts/apicheck.allow lists changes apidiff no longer reports (stale):")
            for line in stale:
                err(f"  {line}")
            status = 1
        if status == 0:
            out("apicheck: clean")
        return status
    except Stop as s:
        return s.code
    finally:
        shutil.rmtree(work, ignore_errors=True)


if __name__ == "__main__":
    sys.exit(main())
