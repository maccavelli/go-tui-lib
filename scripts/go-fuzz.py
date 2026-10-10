#!/usr/bin/env python3
"""Fuzz every fuzz target in one package, for a fixed time each.

Taken from go-selfupdate-lib (then go-core-lib) under
docs/decisions/0002-PLAN-multi-pane-workspace-layouts.md Step 5
(go-selfupdate-lib docs/decisions/0004-PLAN-h2-fuzzing-and-manifest-differential.md
Step 4), and ported from go-fuzz.sh to Python under
docs/decisions/0017-PLAN-python-repository-scripts.md Phase 1. go test -fuzz
takes exactly one fuzz target per run, so the targets run one after another.

Usage: go-fuzz.py [-t FUZZTIME] [-z MINIMIZETIME] [-m MIN] PKG
       go-fuzz.py -a [-l] [-t FUZZTIME] [-z MINIMIZETIME] [-m MIN] PATTERN...
  -t FUZZTIME      time per target, as go test -fuzztime takes it
                   (default 20s)
  -z MINIMIZETIME  the cap on minimizing one input, as go test
                   -fuzzminimizetime takes it (default 5s). Go's own
                   default, 60s, lets the minimizer pause fuzzing for most
                   of a short run (deviation D2 of go-selfupdate-lib
                   docs/decisions/0004-PLAN-h2-fuzzing-and-manifest-differential.md)
  -m MIN           the fewest targets accepted (default 4, at least 1): a
                   loop that finds nothing must not pass
  -a               discover: PATTERN... are package patterns, as go list
                   takes them; every package whose test files declare a
                   func Fuzz... is fuzzed, MIN applying to each, and the
                   others are skipped. Finding no package is an error
                   (docs/decisions/0014-PLAN-hardening.md Step 9, H9)
  -l               with -a, print the packages found, one import path per
                   line, and fuzz nothing

GO names the go command (default go); the tests inject a recorder through
it.

Exit 0 when every target ran clean; on a fuzz failure, go test's status; 1
when too few targets or packages are found, or they cannot be listed; 2 on
a usage error. Go leaves a failing input in PKG/testdata/fuzz/<Name>/.
"""

import os
import re
import subprocess
import sys

USAGE = (
    "usage: go-fuzz.py [-t FUZZTIME] [-z MINIMIZETIME] [-m MIN] PKG\n"
    "       go-fuzz.py -a [-l] [-t FUZZTIME] [-z MINIMIZETIME] [-m MIN] PATTERN..."
)

# A test file that declares a fuzz target has a line starting so.
FUZZ_FUNC = re.compile(rb"^func Fuzz", re.MULTILINE)


class Usage(Exception):
    """A usage error: exit 2."""


class Fail(Exception):
    """A failure with its exit status, its message already printed."""

    def __init__(self, code):
        super().__init__(code)
        self.code = code


def say(line):
    """Print a progress line, flushed before any command writes."""
    print(line, flush=True)


def err(line):
    print(line, file=sys.stderr, flush=True)


def parse(argv):
    """Parse argv as getopts ':t:z:m:al' does: options first, each taking
    its value from the rest of its word or the next word, and flags that
    take none grouped in one word."""
    opts = {"t": "20s", "z": "5s", "m": "4", "a": False, "l": False}
    i = 0
    while i < len(argv):
        word = argv[i]
        if word == "--":
            i += 1
            break
        if not word.startswith("-") or word == "-":
            break
        j = 1
        while j < len(word):
            c = word[j]
            if c in "tzm":
                value = word[j + 1:]
                if not value:
                    i += 1
                    if i >= len(argv):
                        raise Usage()
                    value = argv[i]
                opts[c] = value
                break
            if c in "al":
                opts[c] = True
                j += 1
                continue
            raise Usage()
        i += 1
    rest = argv[i:]
    m = opts["m"]
    if not m or not m.isascii() or not m.isdigit() or m == "0":
        raise Usage()
    opts["m"] = int(m)
    if not opts["a"]:
        if opts["l"] or len(rest) != 1:
            raise Usage()
    elif not rest:
        raise Usage()
    return opts, rest


def go_cmd():
    return os.environ.get("GO") or "go"


def output(args):
    """The command's standard output and its status; standard error passes
    through. A command that cannot start has status 127, as in a shell."""
    try:
        r = subprocess.run(args, stdout=subprocess.PIPE, check=False)
    except OSError:
        return "", 127
    return r.stdout.decode("utf-8", "replace"), r.returncode


def fuzz_pkg(opts, pkg, directory=None):
    """Fuzz every target in pkg, one after another; directory is where its
    corpus is, pkg itself unless given."""
    directory = directory or pkg
    go = go_cmd()
    listing, rc = output([go, "test", "-list", "^Fuzz", pkg])
    if rc != 0:
        err(f"go-fuzz: cannot list the fuzz targets in {pkg}")
        raise Fail(1)
    targets = [line for line in listing.splitlines() if line.startswith("Fuzz")]
    if len(targets) < opts["m"]:
        err(f"go-fuzz: found {len(targets)} fuzz targets in {pkg}, want at least {opts['m']}")
        raise Fail(1)
    for target in targets:
        say(f"go-fuzz: {target} for {opts['t']} (minimizing at most {opts['z']})")
        try:
            rc = subprocess.run([go, "test", "-run", "^$", "-fuzz", f"^{target}$",
                                 "-fuzztime", opts["t"], "-fuzzminimizetime", opts["z"], pkg],
                                check=False).returncode
        except OSError:
            rc = 127
        if rc != 0:
            err(f"go-fuzz: {target} failed (exit {rc}); Go wrote the failing input under "
                f"{directory}/testdata/fuzz/{target}/")
            raise Fail(rc)
    say(f"go-fuzz: {len(targets)} fuzz targets ran clean in {pkg}")


def discover(patterns):
    """Each package with a fuzz target: its import path and directory, from
    one go list run whose status is taken before anything reads it."""
    fmt = ('{{.ImportPath}}{{"\\t"}}{{.Dir}}{{range .TestGoFiles}}{{"\\t"}}{{.}}{{end}}'
           '{{range .XTestGoFiles}}{{"\\t"}}{{.}}{{end}}')
    listing, rc = output([go_cmd(), "list", "-f", fmt, *patterns])
    if rc != 0:
        err(f"go-fuzz: cannot list the packages in {' '.join(patterns)}")
        raise Fail(1)
    found = []
    for line in listing.splitlines():
        fields = line.split("\t")
        if len(fields) < 3:  # no test files
            continue
        path, directory = fields[0], fields[1]
        for name in fields[2:]:
            with open(os.path.join(directory, name), "rb") as f:
                if FUZZ_FUNC.search(f.read()):
                    found.append((path, directory))
                    break
    return found


def main(argv):
    try:
        opts, rest = parse(argv)
    except Usage:
        err(USAGE)
        return 2
    try:
        if not opts["a"]:
            fuzz_pkg(opts, rest[0])
            return 0
        found = discover(rest)
        if not found:
            err(f"go-fuzz: found no package with fuzz targets in {' '.join(rest)}")
            return 1
        if opts["l"]:
            for path, _ in found:
                say(path)
            return 0
        for path, directory in found:
            fuzz_pkg(opts, path, directory)
        say(f"go-fuzz: {len(found)} packages ran clean")
        return 0
    except Fail as f:
        return f.code


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
