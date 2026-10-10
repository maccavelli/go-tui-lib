#!/usr/bin/env python3
"""Builds and runs the framework examples, and checks the guides' excerpts
of them (docs/decisions/0014-MADR-native-integration-api.md W0.4;
docs/decisions/0014-PLAN-api-policy-gates.md Step 5). Ported from
go-examples.sh under docs/decisions/0017-PLAN-python-repository-scripts.md
Phase 4.

The examples live under testdata/frameworks, which the go command,
scripts/go-modules.py and golangci-lint all skip. They are programs built on
the standard flag package, Cobra, Kong and urfave/cli, and they build in a
temporary module outside the repository's modules:
  1. testdata/frameworks is copied into a temporary directory, with go.mod
     written from go.mod.tmpl, @ROOT@ set to the repository root, and the
     committed go.sum beside it.
  2. With GOWORK=off: go build ./... and go vet ./....
  3. Each line of cases.txt runs one program, as
       <framework> <args...> => <exit> stdout|stderr <substring>
     and fails unless the exit status matches and the stream named holds
     the substring.
  4. The guide check: every fenced Go block in docs/guides/*.md whose
     nearest non-blank line above is
       <!-- from: testdata/frameworks/<file>#<name> -->
     must equal the lines between "// guide:<name>" and "// guide:end" in
     that file. The region's blank lines at either end are dropped (gofmt
     puts one before a top-level marker), its common leading tabs are
     removed, and each remaining leading tab is written as four spaces,
     since the guides indent with spaces (deviation D4); then the two are
     compared line for line.

Usage:
  scripts/go-examples.py                 steps 1 to 4
  scripts/go-examples.py --check-guide   step 4 only: no build, no network
  scripts/go-examples.py --update        tidy the temporary module, and
                                         write go.mod.tmpl and go.sum back
                                         (deviation D7); read the diff

Env:
  EXAMPLES_ROOT=<dir>          the tree to read (default: the repository);
                               the tests point it at a throwaway one
  GO_PRECHECK_SKIP_EXAMPLES=1  skip (offline work)
  GO                           the go command (default go)

Exit codes: 0 clean or skipped · 1 a failed case or excerpt · 2 a build,
usage or tool failure.
"""

import difflib
import glob
import os
import re
import shutil
import subprocess
import sys
import tempfile

FROM_LINE = re.compile(r"^<!-- from: ([^ ]+) -->[ \t\n\r\f\v]*$")
BLANK = re.compile(r"^[ \t]*$")
GO_FENCE = re.compile(r"^```go[ \t\n\r\f\v]*$")
END_FENCE = re.compile(r"^```[ \t\n\r\f\v]*$")
END_MARKER = re.compile(r"^[ \t]*// guide:end$")
REPLACE = re.compile(r"^replace github\.com/maccavelli/go-tui-lib => ")


class Stop(Exception):
    """Stop with this exit status, the message already printed."""

    def __init__(self, code):
        super().__init__(code)
        self.code = code


def out(line):
    print(line, flush=True)


def err(line):
    print(line, file=sys.stderr, flush=True)


def lines_of(path):
    """A file's lines, each without its newline and one trailing \\r, as
    `read -r` and awk's sub(/\\r$/, "") read them; a last line with no
    newline is kept."""
    with open(path, encoding="utf-8", errors="replace", newline="") as f:
        text = f.read()
    lines = text.split("\n")
    if lines and lines[-1] == "":
        lines.pop()
    return [line.removesuffix("\r") for line in lines]


def run(args, cwd=None, env=None):
    """A command's status, and its stdout and stderr together."""
    try:
        r = subprocess.run(args, cwd=cwd, env={**os.environ, **(env or {})}, stdout=subprocess.PIPE,
                           stderr=subprocess.STDOUT, check=False)
    except OSError as e:
        return 127, str(e)
    return r.returncode, r.stdout.decode("utf-8", "replace")


def indented(text):
    for line in text.rstrip("\n").split("\n"):
        err(f"  {line}")


def module(root, src, work):
    """The temporary module, at work/m. The go command on Windows needs a
    path with forward slashes in a replace directive."""
    m = os.path.join(work, "m")
    try:
        shutil.copytree(src, m)
        with open(os.path.join(src, "go.mod.tmpl"), encoding="utf-8", newline="") as f:
            tmpl = f.read().rstrip("\n")
    except OSError:
        raise Stop(2)
    with open(os.path.join(m, "go.mod"), "w", encoding="utf-8", newline="") as f:
        f.write(tmpl.replace("@ROOT@", root.replace("\\", "/")) + "\n")
    os.remove(os.path.join(m, "go.mod.tmpl"))
    return m


def update(go, root, src, work):
    m = module(root, src, work)
    rc, text = run([go, "mod", "tidy"], cwd=m, env={"GOWORK": "off"})
    if rc != 0:
        err("examples: go mod tidy failed:")
        indented(text)
        raise Stop(2)
    # go.mod goes back as the template: the replace line names @ROOT@ again.
    lines = ["replace github.com/maccavelli/go-tui-lib => @ROOT@" if REPLACE.match(line) else line
             for line in lines_of(os.path.join(m, "go.mod"))]
    with open(os.path.join(src, "go.mod.tmpl"), "w", encoding="utf-8", newline="") as f:
        f.write("".join(line + "\n" for line in lines))
    shutil.copyfile(os.path.join(m, "go.sum"), os.path.join(src, "go.sum"))
    out("examples: wrote testdata/frameworks/go.mod.tmpl and go.sum; read the diff before committing.")


def stream_lines(data):
    text = data.decode("utf-8", "replace")
    lines = text.split("\n")
    if lines and lines[-1] == "":
        lines.pop()
    return lines


def cases(go, root, src, work):
    """Steps 1 to 3; the status they give."""
    m = module(root, src, work)
    hint = "  If go.mod.tmpl or go.sum is out of date: scripts/go-examples.py --update"
    # The build writes each program into work/bin: a plain `go build ./...`
    # of one main package would write it into the module.
    bin_dir = os.path.join(work, "bin")
    os.makedirs(bin_dir)
    for step, args in (("build", ["-o", bin_dir + "/", "./..."]), ("vet", ["./..."])):
        rc, text = run([go, step, *args], cwd=m, env={"GOWORK": "off"})
        if rc != 0:
            err(f"examples: go {step} failed:")
            indented(text)
            err(hint)
            raise Stop(2)
    try:
        exe = subprocess.run([go, "env", "GOEXE"], env={**os.environ, "GOWORK": "off"},
                             stdout=subprocess.PIPE, check=False).stdout.decode().strip()
    except OSError:
        exe = ""

    status = 0
    n = 0
    for line in lines_of(os.path.join(src, "cases.txt")):
        if line == "" or line.startswith("#"):
            continue
        at = line.find(" => ")
        if at < 0:
            err(f'examples: cases.txt: no " => " in: {line}')
            raise Stop(2)
        words = line[:at].split()
        right = line[at + 4:].strip(" \t\n")
        want, stream, substr = (right.split(None, 2) + ["", "", ""])[:3]
        fw = words[0] if words else ""
        program = os.path.join(bin_dir, fw + exe)
        if not os.access(program, os.X_OK):
            err(f"examples: cases.txt names {fw}, which is not a program under testdata/frameworks.")
            raise Stop(2)
        if stream not in ("stdout", "stderr"):
            err(f'examples: cases.txt: the stream is stdout or stderr, not "{stream}", in: {line}')
            raise Stop(2)
        r = subprocess.run([program, *words[1:]], cwd=work, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                           stderr=subprocess.PIPE, check=False)
        streams = {"stdout": stream_lines(r.stdout), "stderr": stream_lines(r.stderr)}
        n += 1
        if str(r.returncode) != want:
            err(f"examples: {line}: exit {r.returncode}, want {want}")
            for name in ("stdout", "stderr"):
                for text in streams[name]:
                    err(f"  {name}: {text}")
            status = 1
        elif not any(substr in text for text in streams[stream]):
            err(f'examples: {line}: {stream} does not hold "{substr}"')
            for text in streams[stream]:
                err(f"  {stream}: {text}")
            status = 1
    if n == 0:
        err("examples: cases.txt holds no case.")
        raise Stop(2)
    out(f"examples: {n} case(s) run")
    return status


def blocks(guides):
    """Each <!-- from: --> line of the guides, in order: (guide, line, ref,
    block), the block None for a from line no Go block follows."""
    found = []
    for guide in guides:
        pending = None
        inblock = False
        body = None
        for i, line in enumerate(lines_of(guide), start=1):
            if inblock:
                if END_FENCE.match(line):
                    inblock = False
                else:
                    body.append(line)
                continue
            m = FROM_LINE.match(line)
            if m:
                if pending:
                    found.append((pending[0], pending[1], pending[2], None))
                pending = (guide, i, m.group(1))
                continue
            if pending and BLANK.match(line):
                continue
            if pending and GO_FENCE.match(line):
                body = []
                found.append((pending[0], pending[1], pending[2], body))
                pending = None
                inblock = True
                continue
            if pending:
                found.append((pending[0], pending[1], pending[2], None))
                pending = None
        if pending:
            found.append((pending[0], pending[1], pending[2], None))
    return found


def region(path, name):
    """The region name of the file, trimmed and dedented, with each leading
    tab as four spaces; or 3 with no start marker, 4 with no end marker."""
    start = re.compile(r"^[ \t]*// guide:" + name + "$")
    on = found = ended = False
    lines = []
    for line in lines_of(path):
        if not on and start.search(line):
            on = found = True
            continue
        if on and END_MARKER.match(line):
            ended = True
            break
        if on:
            lines.append(line)
    if not found:
        return 3
    if not ended:
        return 4
    tabs = [len(line) - len(line.lstrip("\t")) for line in lines if not BLANK.match(line)]
    least = min(tabs) if tabs else 0
    first, last = 0, len(lines) - 1
    while first <= last and BLANK.match(lines[first]):
        first += 1
    while last >= first and BLANK.match(lines[last]):
        last -= 1
    result = []
    for line in lines[first:last + 1]:
        if BLANK.match(line):
            result.append("")
            continue
        line = line[least:]
        lead = len(line) - len(line.lstrip("\t"))
        result.append("    " * lead + line[lead:])
    return result


def guide_check(root):
    """Step 4; the status it gives."""
    status = 0
    excerpts = 0
    guides = sorted(glob.glob(os.path.join(root, "docs", "guides", "*.md")))
    for guide, gline, ref, block in blocks(guides):
        where = f"docs/guides/{os.path.basename(guide)}:{gline}"
        if block is None:
            err(f"examples: {where}: <!-- from: {ref} --> is not followed by a Go block.")
            status = 1
            continue
        file, _, name = ref.partition("#")
        if file == ref or not name:
            err(f"examples: {where}: {ref} names no region (<file>#<name>).")
            status = 1
            continue
        if not file.startswith("testdata/frameworks/"):
            err(f"examples: {where}: {ref} is not under testdata/frameworks.")
            status = 1
            continue
        if not os.path.isfile(os.path.join(root, file)):
            err(f"examples: {where}: {file} does not exist.")
            status = 1
            continue
        got = region(os.path.join(root, file), name)
        if got == 3:
            err(f'examples: {where}: {file} has no region "// guide:{name}".')
            status = 1
            continue
        if got == 4:
            err(f'examples: {where}: {file}\'s region {name} has no "// guide:end".')
            status = 1
            continue
        excerpts += 1
        if got != block:
            err(f"examples: {where}: the excerpt differs from {ref} (- the program, + the guide):")
            for line in list(difflib.unified_diff(got, block, lineterm=""))[2:]:
                err(f"  {line}")
            status = 1
    out(f"examples: {excerpts} excerpt(s) checked")
    return status


def main(argv):
    for stream in (sys.stdout, sys.stderr):
        stream.reconfigure(newline="\n")
    go = os.environ.get("GO") or "go"
    first = argv[0] if argv else ""
    modes = {"": "all", "--check-guide": "guide", "--update": "update"}
    if first not in modes:
        err("usage: scripts/go-examples.py [--check-guide | --update]")
        return 2
    mode = modes[first]
    if os.environ.get("GO_PRECHECK_SKIP_EXAMPLES", "0") == "1":
        err("examples: skipped (GO_PRECHECK_SKIP_EXAMPLES=1)")
        return 0

    given = os.environ.get("EXAMPLES_ROOT")
    if given:
        if not os.path.isdir(given):
            return 2
        root = os.path.abspath(given)
    else:
        try:
            r = subprocess.run(["git", "rev-parse", "--show-toplevel"], stdout=subprocess.PIPE, check=False)
        except OSError:
            return 2
        if r.returncode != 0:
            return 2
        root = r.stdout.decode().rstrip("\n")
    src = os.path.join(root, "testdata", "frameworks")
    if not os.path.isfile(os.path.join(src, "go.mod.tmpl")):
        err(f"examples: no {src}/go.mod.tmpl.")
        return 2

    work = tempfile.mkdtemp()
    try:
        if mode == "update":
            update(go, root, src, work)
            return 0
        status = cases(go, root, src, work) if mode == "all" else 0
        status = guide_check(root) or status
        if status == 0:
            out("examples: clean")
        return status
    except Stop as s:
        return s.code
    finally:
        shutil.rmtree(work, ignore_errors=True)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
