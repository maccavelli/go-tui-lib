#!/usr/bin/env python3
"""Tests for go-apicheck.py, on throwaway repositories in a temporary
directory, never in this one (docs/decisions/0014-PLAN-api-policy-gates.md
Step 4; ported from go-apicheck_test.sh under
docs/decisions/0017-PLAN-python-repository-scripts.md Phase 3).

The removal cases are the ones that matter: a gate that passed an unlisted
incompatible change, or compared a tag with itself, would be no gate at
all. apidiff is installed once, at the Makefile's pin, and every case uses
that binary.

APICHECK names the script under test (default go-apicheck.py beside this
file); a file not ending in .py is run as a program, so the shell script it
replaced can be tested too. An error inside a case is reported as that
case's FAIL, and the run goes on (0017-MADR item 3).
"""

import os
import re
import shutil
import subprocess
import sys
import tempfile
import traceback

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
APICHECK = os.environ.get("APICHECK") or os.path.join(ROOT, "scripts", "go-apicheck.py")
EXE = ".exe" if os.name == "nt" else ""

passed = 0
failed = 0


def check(name, want, got):
    global passed, failed
    if want == got:
        print(f"  ok   {name}", flush=True)
        passed += 1
    else:
        print(f"  FAIL {name}: want {want}, got {got}", flush=True)
        failed += 1


def case(title, fn, *args):
    """Run one case; an exception in it is its FAIL, and the run goes on."""
    global failed
    try:
        fn(*args)
    except Exception:  # noqa: BLE001 - every error is reported, by case
        last = traceback.format_exc().strip().splitlines()[-1]
        print(f"  FAIL {title}: the case stopped: {last}", flush=True)
        failed += 1


def write(path, text, mode="w"):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, mode, encoding="utf-8", newline="\n") as f:
        f.write(text)


def g(directory, *args):
    """git, as a throwaway identity."""
    subprocess.run(["git", "-c", "user.name=apicheck", "-c", "user.email=apicheck@example.invalid",
                    "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false", *args],
                   cwd=directory, check=True, stdout=subprocess.DEVNULL)


def repo(directory):
    """A repository whose root is module example.com/r, with package p
    exporting F and G, a go.work and an empty allow file, committed."""
    os.makedirs(os.path.join(directory, "p"), exist_ok=True)
    g(directory, "init", "-q")
    write(os.path.join(directory, "go.mod"), "module example.com/r\n\ngo 1.27.1\n")
    write(os.path.join(directory, "go.work"), "go 1.27.1\n\nuse .\n")
    write(os.path.join(directory, "p", "p.go"),
          "package p\n\n// F is kept.\nfunc F() {}\n\n// G is removed by some cases.\nfunc G() {}\n")
    write(os.path.join(directory, "allow"), "")
    g(directory, "add", "-A")
    g(directory, "commit", "-q", "-m", "base")


def build(work, name, body):
    """A small Go program, built natively, standing in for a shell script."""
    src = os.path.join(work, "src", name)
    write(os.path.join(src, "go.mod"), f"module example.com/{name}\n\ngo 1.27.1\n")
    write(os.path.join(src, "main.go"), body)
    exe = os.path.join(work, "bin", name + EXE)
    subprocess.run(["go", "build", "-o", exe, "."], cwd=src, check=True,
                   env={**os.environ, "GOWORK": "off", "CGO_ENABLED": "0"})
    return exe


class Runner:
    def __init__(self, apidiff):
        self.apidiff = apidiff
        self.out = ""

    def run(self, directory, apidiff=None, env=None):
        """The script's exit status, with stdout and stderr together in
        self.out, as the shell test's run read them."""
        cmd = [sys.executable, APICHECK] if APICHECK.endswith(".py") else [APICHECK]
        r = subprocess.run(cmd, cwd=directory, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False,
                           env={**os.environ, "APIDIFF": apidiff or self.apidiff,
                                "APICHECK_ALLOW": os.path.join(directory, "allow"), **(env or {})})
        self.out = r.stdout.decode("utf-8", "replace").replace("\r\n", "\n")
        return r.returncode

    def has(self, name, pattern):
        """Whether the last run's output has a line matching pattern."""
        global passed, failed
        if re.search(pattern, self.out, re.MULTILINE):
            print(f"  ok   {name}", flush=True)
            passed += 1
        else:
            print(f"  FAIL {name}: no line matches {pattern} in:", flush=True)
            for line in self.out.rstrip("\n").split("\n"):
                print(f"         {line}", flush=True)
            failed += 1


P_G = "package p\n\n// F is kept.\nfunc F() {}\n\n// G is removed by some cases.\nfunc G() {}\n"
P_NO_G = "package p\n\n// F is kept.\nfunc F() {}\n"


def cases_a(r, work):
    a = os.path.join(work, "a")
    repo(a)
    pfile = os.path.join(a, "p", "p.go")

    # 1. No tag: the module was never released, and is skipped.
    check("no previous tag passes", 0, r.run(a))
    r.has("no previous tag is said", "no previous tag, skipped")

    # 2. Tagged, unchanged: clean. A tag on HEAD is never the base (case 7),
    # so each tag here is followed by a commit.
    g(a, "tag", "v0.1.0")
    g(a, "commit", "-q", "--allow-empty", "-m", "next")
    check("an unchanged API passes", 0, r.run(a))
    r.has("against the tag", "against v0.1.0, 0 incompatible")

    # 3. An addition is compatible.
    write(pfile, "\n// H is added.\nfunc H() {}\n", mode="a")
    check("an addition passes", 0, r.run(a))

    # 4. A removal fails, naming the change.
    write(pfile, P_NO_G)
    check("a removal fails", 1, r.run(a))
    r.has("the removal is named", "p.G: removed")

    # 5. The same removal, listed, passes.
    line = next((l.lstrip() for l in r.out.split("\n") if re.search("p.G: removed", l)), "")
    write(os.path.join(a, "allow"), f"# a test\n{line}\n")
    check("a listed removal passes", 0, r.run(a))

    # 6. A listed line apidiff no longer prints is stale, and fails.
    g(a, "checkout", "-q", "--", "p/p.go")
    check("a stale entry fails", 1, r.run(a))
    r.has("the stale entry is named", "stale")
    write(os.path.join(a, "allow"), "")

    # 7. On a tagged commit, the base is the tag before it.
    write(pfile, P_NO_G)
    g(a, "commit", "-q", "-am", "remove G")
    g(a, "tag", "v0.2.0")
    check("a tag on HEAD compares with the tag before", 1, r.run(a))
    r.has("against the earlier tag", "against v0.1.0")


def cases_b(r, work):
    # 8. A package moved into a nested module is gone from the root. With
    # GOWORK=off the root's export no longer holds it; in workspace mode it
    # would, and the removal would pass unseen.
    b = os.path.join(work, "b")
    repo(b)
    write(os.path.join(b, "sub", "s.go"), "package sub\n\n// S is exported.\nfunc S() {}\n")
    g(b, "add", "-A")
    g(b, "commit", "-q", "-m", "sub")
    g(b, "tag", "v0.1.0")
    write(os.path.join(b, "sub", "go.mod"), "module example.com/r/sub\n\ngo 1.27.1\n")
    write(os.path.join(b, "go.work"), "go 1.27.1\n\nuse (\n\t.\n\t./sub\n)\n")
    g(b, "add", "-A")
    g(b, "commit", "-q", "-m", "sub is a module")
    check("a package moved out of the root fails", 1, r.run(b))
    r.has("the moved package is named", "sub.*removed")
    r.has("the new module has no tag yet", "sub: no previous tag, skipped")

    # 9. A nested module is compared with its own prefixed tag.
    g(b, "tag", "sub/v0.1.0")
    g(b, "tag", "v0.2.0")
    g(b, "commit", "-q", "--allow-empty", "-m", "next")
    write(os.path.join(b, "sub", "s.go"), "package sub\n\n// T replaces S.\nfunc T() {}\n")
    check("a nested module's removal fails", 1, r.run(b))
    r.has("it is compared with sub/v0.1.0", "sub: against sub/v0.1.0, 1 incompatible")
    r.has("it is listed under the module", "^  sub\t.*S: removed")


def case10(r, work):
    # 10. A failing apidiff is a tool failure.
    fail = build(work, "fail", "package main\n\nimport \"os\"\n\nfunc main() { os.Exit(1) }\n")
    check("a failing apidiff exits 2", 2, r.run(os.path.join(work, "a"), apidiff=fail))


WARNER = """package main

import (
	"errors"
	"os"
	"os/exec"
)

func main() {
	cmd := exec.Command(os.Getenv("WARN_REAL"), os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	_, _ = os.Stderr.WriteString("go: warning: a note from go\\n")
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		os.Exit(exit.ExitCode())
	}
}
"""


def case11(r, work):
    # 11. A line go writes to stderr while it succeeds, such as a warning,
    #     is not part of the module path: an unchanged API still passes
    #     (docs/decisions/0017-PLAN-python-repository-scripts.md D3).
    c = os.path.join(work, "c")
    repo(c)
    g(c, "tag", "v0.1.0")
    g(c, "commit", "-q", "--allow-empty", "-m", "next")
    warner = build(work, "warn-go", WARNER)
    env = {"GO": warner, "WARN_REAL": shutil.which("go")}
    check("an unchanged API passes with go warning", 0, r.run(c, env=env))
    r.has("it is compared with v0.1.0", "against v0.1.0, 0 incompatible")
    r.has("the warning reaches the output", "go: warning: a note from go")


def main():
    work = tempfile.mkdtemp()
    os.environ.pop("GOWORK", None)
    try:
        # apidiff is installed as go-apicheck.py installs it, through the
        # configured proxy. The cases then run with GOPROXY=off: the
        # throwaway modules require nothing, so the go command must never
        # need the network (docs/decisions/0014-PLAN-api-policy-gates.md,
        # deviation D9).
        with open(os.path.join(ROOT, "Makefile"), encoding="utf-8") as f:
            version = re.search(r"^APIDIFF_VERSION[ \t]*\?=[ \t]*(.*)$", f.read(), re.MULTILINE).group(1)
        subprocess.run(["go", "install", f"golang.org/x/exp/cmd/apidiff@{version}"], check=True,
                       env={**os.environ, "GOBIN": os.path.join(work, "bin"), "GOWORK": "off"})
        apidiff = os.path.join(work, "bin", "apidiff")
        if os.access(apidiff + ".exe", os.X_OK):
            apidiff += ".exe"
        os.environ["GOPROXY"] = "off"
        r = Runner(apidiff)
        case("1-7. the root module", cases_a, r, work)
        case("8-9. a nested module", cases_b, r, work)
        case("10. a failing apidiff", case10, r, work)
        case("11. a warning from go", case11, r, work)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    print(f"go-apicheck_test: {passed} passed, {failed} failed")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
