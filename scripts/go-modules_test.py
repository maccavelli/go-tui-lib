#!/usr/bin/env python3
"""Offline tests for go-modules.py, on throwaway repositories in a temporary
directory, never in this one
(docs/decisions/0010-PLAN-nested-adapter-modules.md Phase 3; ported from
go-modules_test.sh under docs/decisions/0017-PLAN-python-repository-scripts.md
Phase 2).

The --check failures are the cases that matter: a check that passed with a
module missing from go.work would let a gate skip that module.

MODULES names the script under test (default go-modules.py beside this
file); a file not ending in .py is run as a program, so the shell script it
replaced can be tested too. An error inside a case is reported as that
case's FAIL, and the run goes on (0017-MADR item 3).
"""

import os
import shutil
import subprocess
import sys
import tempfile
import traceback

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MODULES = os.environ.get("MODULES") or os.path.join(ROOT, "scripts", "go-modules.py")

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


def write(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="\n") as f:
        f.write(text)


def git(directory, *args):
    subprocess.run(["git", "-C", directory, *args], check=True, stdout=subprocess.DEVNULL)


def repo(directory):
    """A git repository whose root is a module, with a go.work."""
    os.makedirs(directory, exist_ok=True)
    git(directory, "init", "-q")
    write(os.path.join(directory, "go.mod"), "module example.com/root\n\ngo 1.27.1\n")
    write(os.path.join(directory, "root.go"), "package root\n")
    write(os.path.join(directory, "go.work"), "go 1.27.1\n\nuse .\n")


def nested(directory, sub):
    """A module at directory/sub."""
    write(os.path.join(directory, sub, "go.mod"), f"module example.com/root/{sub}\n\ngo 1.27.1\n")
    write(os.path.join(directory, sub, "sub.go"), f"package {os.path.basename(sub)}\n")


class Runner:
    def __init__(self):
        self.out = ""
        self.raw = b""

    def run(self, directory, *args, go="go"):
        """The script's exit status, with its stdout and stderr together in
        self.out, as the shell test's run read them, and stdout's bytes in
        self.raw. go is the go command the script runs."""
        cmd = [sys.executable, MODULES] if MODULES.endswith(".py") else [MODULES]
        r = subprocess.run(cmd + list(args), cwd=directory, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                           env={**os.environ, "GO": go, "GOWORK": "off"}, check=False)
        self.raw = r.stdout
        self.out = (r.stdout + r.stderr).decode("utf-8", "replace").replace("\r\n", "\n")
        return r.returncode

    def sorted_lines(self):
        return ",".join(sorted(line for line in self.out.split("\n") if line))

    def count(self, text):
        return sum(1 for line in self.out.split("\n") if text in line)


def two_repo(work):
    two = os.path.join(work, "two")
    repo(two)
    nested(two, "sub")
    write(os.path.join(two, "go.work"), "go 1.27.1\n\nuse (\n\t.\n\t./sub\n)\n")
    git(two, "add", "-A")
    return two


def case1(r, work):
    # 1. A root and one nested module, both in go.work: two lines, and
    #    --check passes. GOWORK=off in the caller does not hide the nested
    #    module.
    two = os.path.join(work, "two")
    check("listing exits 0", 0, r.run(two))
    check("listing prints the root and sub", ".,sub", r.sorted_lines())
    check("--check passes", 0, r.run(two, "--check"))


def case2(r, work):
    # 2. A nested go.mod missing from go.work: --check fails and names it.
    d = os.path.join(work, "missing")
    repo(d)
    nested(d, "sub")
    git(d, "add", "-A")
    check("a module missing from go.work fails --check", 1, r.run(d, "--check"))
    check("the failure names it", 1, r.count("sub/go.mod is tracked, but go.work does not list sub"))


def case3(r, work):
    # 3. A go.work entry with no go.mod: --check fails.
    d = os.path.join(work, "ghost")
    repo(d)
    os.makedirs(os.path.join(d, "ghost"))
    write(os.path.join(d, "go.work"), "go 1.27.1\n\nuse (\n\t.\n\t./ghost\n)\n")
    git(d, "add", "-A")
    check("a go.work entry with no go.mod fails --check", 1, r.run(d, "--check"))
    check("the failure names it", 1, r.count("cannot load module ghost listed in go.work"))


def case4(r, work):
    # 4. A go.mod that is not tracked: --check fails, because the module
    #    would not reach a commit.
    d = os.path.join(work, "untracked")
    repo(d)
    nested(d, "sub")
    write(os.path.join(d, "go.work"), "go 1.27.1\n\nuse (\n\t.\n\t./sub\n)\n")
    git(d, "add", "go.mod", "go.work", "root.go")
    check("an untracked module in go.work fails --check", 1, r.run(d, "--check"))
    check("the failure names it", 1, r.count("go.work lists sub, which has no tracked go.mod"))


def case5(r, work):
    # 5. A go.mod under testdata, or under a "_" or "." directory, is not a
    #    module the go command builds: --check skips it.
    d = os.path.join(work, "ignored")
    repo(d)
    nested(d, "testdata/fixture")
    nested(d, "_scratch")
    git(d, "add", "-A")
    check("ignored go.mod files pass --check", 0, r.run(d, "--check"))


def case6(r, work):
    # 6. No go.work: --check fails, and listing is a usage error.
    d = os.path.join(work, "nowork")
    repo(d)
    os.remove(os.path.join(d, "go.work"))
    git(d, "add", "-A")
    check("no go.work fails --check", 1, r.run(d, "--check"))
    check("no go.work fails listing", 2, r.run(d))


def case7(r, work):
    # 7. An unknown argument is a usage error.
    check("an unknown argument exits 2", 2, r.run(os.path.join(work, "two"), "--bogus"))


def speller(work, top, link):
    """A go command that runs the real go and reports top as link at the
    start of each line of its stdout: a Go program, so that it runs
    natively on every host, where the shell test used a bash script."""
    src = os.path.join(work, "speller")
    write(os.path.join(src, "go.mod"), "module example.com/speller\n\ngo 1.27.1\n")
    write(os.path.join(src, "main.go"), """package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
)

func main() {
	var out bytes.Buffer
	cmd := exec.Command(os.Getenv("SPELL_REAL"), os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, &out, os.Stderr
	err := cmd.Run()
	top, link := os.Getenv("SPELL_TOP"), os.Getenv("SPELL_LINK")
	for _, line := range strings.SplitAfter(out.String(), "\\n") {
		if strings.HasPrefix(line, top) {
			line = link + strings.TrimPrefix(line, top)
		}
		_, _ = os.Stdout.WriteString(line)
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		os.Exit(exit.ExitCode())
	}
}
""")
    exe = os.path.join(work, "bin", "spell-go" + (".exe" if os.name == "nt" else ""))
    subprocess.run(["go", "build", "-o", exe, "."], cwd=src, check=True,
                   env={**os.environ, "GOWORK": "off", "CGO_ENABLED": "0"})
    os.environ.update({"SPELL_REAL": shutil.which("go"), "SPELL_TOP": top, "SPELL_LINK": link})
    return exe


def go_dirs(go, top):
    r = subprocess.run([go, "list", "-m", "-f", "{{.Dir}}"], stdout=subprocess.PIPE, check=True,
                       env={**os.environ, "GOWORK": f"{top}/go.work"}, cwd=top)
    return r.stdout.decode().replace("\r\n", "\n").split("\n")


def case8(r, work):
    # 8. The go command spells the module directories differently from git,
    #    as on Windows, where go prints C:\Users\... and git C:/Users/...
    #    (docs/decisions/0010-PLAN-nested-adapter-modules.md deviation D1).
    #    On a host where go's spelling is already git's, a go that reports
    #    them through a symlink to the repository stands in for it. Either
    #    way the list and --check must not change.
    two = os.path.join(work, "two")
    top = subprocess.run(["git", "-C", two, "rev-parse", "--show-toplevel"], stdout=subprocess.PIPE,
                         check=True).stdout.decode().strip()
    spell = "go"
    if go_dirs("go", top)[0] == top:
        link = os.path.join(work, "two-link")
        os.symlink(top, link)
        spell = speller(work, top, link)
    check("go's spelling differs from git's", 0, sum(1 for d in go_dirs(spell, top) if d == top))
    check("listing under another spelling exits 0", 0, r.run(two, go=spell))
    check("listing under another spelling prints the root and sub", ".,sub", r.sorted_lines())
    check("--check under another spelling passes", 0, r.run(two, "--check", go=spell))


def case9(r, work):
    # 9. The listing's lines end in \n alone, on Windows too, for callers
    #    that read it with $(...) (docs/decisions/0017-PLAN-python-repository-scripts.md
    #    Phase 2).
    check("listing exits 0 again", 0, r.run(os.path.join(work, "two")))
    check("the listing has no \\r", 0, r.raw.count(b"\r"))


def main():
    work = tempfile.mkdtemp()
    try:
        r = Runner()
        two_repo(work)
        case("1. a root and a nested module", case1, r, work)
        case("2. a module missing from go.work", case2, r, work)
        case("3. a go.work entry with no go.mod", case3, r, work)
        case("4. an untracked go.mod", case4, r, work)
        case("5. ignored go.mod files", case5, r, work)
        case("6. no go.work", case6, r, work)
        case("7. an unknown argument", case7, r, work)
        case("8. another spelling of the directories", case8, r, work)
        case("9. line endings", case9, r, work)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    print(f"go-modules_test: {passed} passed, {failed} failed")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
