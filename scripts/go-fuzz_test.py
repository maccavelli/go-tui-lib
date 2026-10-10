#!/usr/bin/env python3
"""Offline tests for go-fuzz.py, on throwaway modules in a temporary
directory (go-selfupdate-lib
docs/decisions/0004-PLAN-h2-fuzzing-and-manifest-differential.md Step 4;
here under docs/decisions/0002-PLAN-multi-pane-workspace-layouts.md Step 5,
and ported from go-fuzz_test.sh under
docs/decisions/0017-PLAN-python-repository-scripts.md Phase 1).

The failing-target case is the one that matters: a fuzz gate that reported
success while a target failed would be no gate at all.

FUZZ names the script under test (default go-fuzz.py beside this file); a
file not ending in .py is run as a program, so the shell script it replaced
can be tested too. An error inside a case is reported as that case's FAIL,
and the run goes on (0017-MADR item 3).
"""

import os
import re
import shutil
import subprocess
import sys
import tempfile
import traceback

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
FUZZ = os.environ.get("FUZZ") or os.path.join(ROOT, "scripts", "go-fuzz.py")
FUZZ_FUNC = re.compile(rb"^func Fuzz", re.MULTILINE)

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


def module(directory, name, body):
    """A module whose package has one test file."""
    write(os.path.join(directory, "go.mod"), f"module example.com/{name}\n\ngo 1.27.1\n")
    write(os.path.join(directory, "fz_test.go"), f'package {name}\n\nimport "testing"\n\n{body}\n')


class Runner:
    def __init__(self, work):
        self.work = work
        self.out = ""

    def run(self, directory, *args, env=None):
        """The script's exit status, with its output in self.out."""
        cmd = [sys.executable, FUZZ] if FUZZ.endswith(".py") else [FUZZ]
        r = subprocess.run(cmd + list(args), cwd=directory, stdout=subprocess.PIPE,
                           stderr=subprocess.STDOUT, env={**os.environ, **(env or {})}, check=False)
        # Python on Windows writes \r\n to a pipe; the lines are read as \n.
        self.out = r.stdout.decode("utf-8", "replace").replace("\r\n", "\n")
        return r.returncode

    def count(self, pattern):
        return sum(1 for line in self.out.splitlines() if re.search(pattern, line))


def recorder(work):
    """A go command that records its arguments and runs the real go: a Go
    program, so that it runs natively on every host, where the shell test
    used a bash script."""
    src = os.path.join(work, "recorder")
    write(os.path.join(src, "go.mod"), "module example.com/recorder\n\ngo 1.27.1\n")
    write(os.path.join(src, "main.go"), """package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

func main() {
	f, err := os.OpenFile(os.Getenv("GO_REC_LOG"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		os.Exit(3)
	}
	_, _ = f.WriteString(strings.Join(os.Args[1:], " ") + "\\n")
	_ = f.Close()
	cmd := exec.Command(os.Getenv("GO_REC_REAL"), os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		os.Exit(1)
	}
}
""")
    exe = os.path.join(work, "bin", "go" + (".exe" if os.name == "nt" else ""))
    subprocess.run(["go", "build", "-o", exe, "."], cwd=src, check=True,
                   env={**os.environ, "GOWORK": "off", "CGO_ENABLED": "0"})
    return exe


def fuzz_packages(root, mod):
    """The packages whose tracked or untracked, not ignored, test files
    declare a fuzz target, by import path: the listing discovery is checked
    against. A tracked file the work tree no longer has, deleted and not
    staged, is skipped, where the shell test's grep stopped on it."""
    files = subprocess.run(["git", "ls-files", "--cached", "--others", "--exclude-standard", "*_test.go"],
                           cwd=root, stdout=subprocess.PIPE, check=True).stdout.decode().splitlines()
    dirs = set()
    for name in files:
        if "/testdata/" in f"/{name}":
            continue
        path = os.path.join(root, name)
        if not os.path.isfile(path):
            continue
        with open(path, "rb") as f:
            if FUZZ_FUNC.search(f.read()):
                dirs.add(os.path.dirname(name) or ".")
    return sorted(f"{mod}/{d}" for d in dirs)


def go_list_m(directory):
    return subprocess.run(["go", "list", "-m"], cwd=directory, stdout=subprocess.PIPE, check=True,
                          env={**os.environ, "GOWORK": "off"}).stdout.decode().strip()


CLEAN = '''
func FuzzA(f *testing.F) {
	f.Add([]byte("a"))
	f.Fuzz(func(t *testing.T, b []byte) { _ = len(b) })
}

func FuzzB(f *testing.F) {
	f.Add("b")
	f.Fuzz(func(t *testing.T, s string) { _ = len(s) })
}'''

BOOM = '''
func FuzzBoom(f *testing.F) {
	f.Add([]byte("a"))
	f.Fuzz(func(t *testing.T, b []byte) {
		if string(b) != "a" {
			t.Fatalf("boom on %q", b)
		}
	})
}'''


def case1(r, work):
    # 1. Two clean targets, two required: both are fuzzed, and it passes.
    clean = os.path.join(work, "clean")
    check("clean targets pass", 0, r.run(clean, "-t", "1s", "-m", "2", "./"))
    check("both targets were fuzzed", 2, r.count(r"^go-fuzz: Fuzz[AB] for 1s "))


def case1b(r, work):
    # 1b. -z reaches go test as -fuzzminimizetime, through a recording GO.
    log = os.path.join(work, "go-args")
    env = {"GO": recorder(work), "GO_REC_LOG": log, "GO_REC_REAL": shutil.which("go")}
    check("a recorded run passes", 0, r.run(os.path.join(work, "clean"), "-t", "1s", "-z", "3s", "-m", "2", "./", env=env))
    with open(log, encoding="utf-8") as f:
        check("-z reaches each fuzz run", 2, sum(1 for line in f if "-fuzzminimizetime 3s" in line))


def case2(r, work):
    # 2. Fewer targets than required: refused, naming the count.
    check("too few targets is refused", 1, r.run(os.path.join(work, "clean"), "-t", "1s", "-m", "3", "./"))
    check("the refusal names the count", 1, r.count("found 2 fuzz targets"))


def case3(r, work):
    # 3. THE CASE THAT MATTERS. A target that fails on any input but its
    #    seed: the fuzzer finds one at once, so the script must fail and Go
    #    must leave the input behind.
    boom = os.path.join(work, "boom")
    module(boom, "boom", BOOM)
    rc = r.run(boom, "-t", "5s", "-m", "1", "./")
    check("a failing target fails the run", "nonzero", "nonzero" if rc != 0 else rc)
    corpus = os.path.join(boom, "testdata", "fuzz", "FuzzBoom")
    saved = sum(len(f) for _, _, f in os.walk(corpus)) if os.path.isdir(corpus) else 0
    check("the failing input is saved", 1, saved)


def case4(r, work):
    # 4. Usage errors.
    clean = os.path.join(work, "clean")
    check("no package is a usage error", 2, r.run(clean))
    check("a non-numeric minimum is a usage error", 2, r.run(clean, "-m", "x", "./"))
    check("a zero minimum is a usage error", 2, r.run(clean, "-m", "0", "./"))
    check("an unknown flag is a usage error", 2, r.run(clean, "-z", "./"))
    check("-l without -a is a usage error", 2, r.run(clean, "-l", "./"))
    check("-a with no pattern is a usage error", 2, r.run(clean, "-a"))


def case5(r, work):
    # 5. Discovery (docs/decisions/0014-PLAN-hardening.md Step 9): a module
    #    with a target in a package's own tests (a), one in an external test
    #    package (d), tests without a target (b) and no tests at all (c).
    disc = os.path.join(work, "disc")
    write(os.path.join(disc, "go.mod"), "module example.com/disc\n\ngo 1.27.1\n")
    for p in "abcd":
        write(os.path.join(disc, p, f"{p}.go"), f"package {p}\n")
    write(os.path.join(disc, "a", "a_test.go"),
          'package a\n\nimport "testing"\n\nfunc FuzzA(f *testing.F) {\n\tf.Add(1)\n\tf.Fuzz(func(t *testing.T, n int) {})\n}\n')
    write(os.path.join(disc, "b", "b_test.go"), 'package b\n\nimport "testing"\n\nfunc TestB(t *testing.T) {}\n')
    write(os.path.join(disc, "d", "d_test.go"),
          'package d_test\n\nimport "testing"\n\nfunc FuzzD(f *testing.F) {\n\tf.Add(1)\n\tf.Fuzz(func(t *testing.T, n int) {})\n}\n')
    check("discovery lists the packages with targets", 0, r.run(disc, "-a", "-l", "./..."))
    check("the list is a and d", "example.com/disc/a example.com/disc/d", " ".join(r.out.split()))
    check("discovery fuzzes them", 0, r.run(disc, "-a", "-t", "1s", "-m", "1", "./..."))
    check("each target is fuzzed once", 2, r.count(r"^go-fuzz: Fuzz[AD] for 1s "))
    check("a package with no target is skipped", 0, r.count(r"example\.com/disc/[bc]"))
    check("the run names two packages", 1, r.count(r"^go-fuzz: 2 packages ran clean$"))
    check("no package with a target is refused", 1, r.run(disc, "-a", "-l", "./b/...", "./c/..."))
    boom = os.path.join(work, "boom")  # case 3's module, its corpus removed
    shutil.rmtree(os.path.join(boom, "testdata"), ignore_errors=True)
    rc = r.run(boom, "-a", "-t", "5s", "-m", "1", "./...")
    check("a failing target found by discovery fails the run", "nonzero", "nonzero" if rc != 0 else rc)
    m = re.findall(r"Go wrote the failing input under (.*)$", r.out, re.MULTILINE)
    check("its failure names the directory the input is in", "yes", "yes" if m and os.path.isdir(m[-1]) else "no")


def case6(r):
    # 6. The repository's own fuzz targets: discovery finds exactly the
    #    packages whose tracked test files declare one, so a package cannot
    #    be left out of make fuzz.
    mod = go_list_m(ROOT)
    want = " ".join(fuzz_packages(ROOT, mod))
    rc = r.run(ROOT, "-a", "-l", "./...", env={"GOWORK": "off"})
    got = " ".join(sorted(r.out.split())) if rc == 0 else "discovery failed"
    check("the repository's packages with targets are all found", want, got)
    check("the repository has fuzz targets in at least 8 packages", "yes", "yes" if len(got.split()) >= 8 else "no")


def case7(r, work):
    # 7. The listing case 6 compares with skips a tracked test file that is
    #    deleted and not staged, where the shell test stopped with no FAIL
    #    line (docs/decisions/0017-PLAN-python-repository-scripts.md Phase 1).
    repo = os.path.join(work, "gone")
    write(os.path.join(repo, "go.mod"), "module example.com/gone\n\ngo 1.27.1\n")
    for p in "ab":
        write(os.path.join(repo, p, f"{p}.go"), f"package {p}\n")
        write(os.path.join(repo, p, f"{p}_test.go"),
              f'package {p}\n\nimport "testing"\n\nfunc Fuzz{p.upper()}(f *testing.F) {{\n\tf.Add(1)\n\tf.Fuzz(func(t *testing.T, n int) {{}})\n}}\n')
    write(os.path.join(repo, "a", "plain_test.go"), 'package a\n\nimport "testing"\n\nfunc TestPlain(t *testing.T) {}\n')
    git = ["git", "-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "commit.gpgsign=false"]
    subprocess.run(["git", "init", "-q"], cwd=repo, check=True)
    subprocess.run(git + ["add", "-A"], cwd=repo, check=True)
    subprocess.run(git + ["commit", "-q", "--no-verify", "-m", "t"], cwd=repo, check=True)
    os.remove(os.path.join(repo, "b", "b_test.go"))
    os.remove(os.path.join(repo, "a", "plain_test.go"))
    check("the listing skips tracked test files deleted and not staged", "example.com/gone/a",
          " ".join(fuzz_packages(repo, "example.com/gone")))
    rc = r.run(repo, "-a", "-l", "./...", env={"GOWORK": "off"})
    check("discovery agrees with it", "example.com/gone/a", " ".join(r.out.split()) if rc == 0 else "discovery failed")


def main():
    work = tempfile.mkdtemp()
    try:
        r = Runner(work)
        module(os.path.join(work, "clean"), "clean", CLEAN)
        case("1. clean targets", case1, r, work)
        case("1b. -z through a recorded go", case1b, r, work)
        case("2. too few targets", case2, r, work)
        case("3. a failing target", case3, r, work)
        case("4. usage errors", case4, r, work)
        case("5. discovery", case5, r, work)
        case("6. the repository's targets", case6, r)
        case("7. a deleted, unstaged test file", case7, r, work)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    print(f"go-fuzz_test: {passed} passed, {failed} failed")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
