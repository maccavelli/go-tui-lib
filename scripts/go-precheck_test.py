#!/usr/bin/env python3
"""Tests for go-precheck.py's gofmt step and its choice of files, and for
the shim the machine-wide gate runs, on throwaway repositories in a
temporary directory, never in this one
(docs/decisions/0015-PLAN-precheck-gofmt-errors.md Step 2; ported from
go-precheck_test.sh under
docs/decisions/0017-PLAN-python-repository-scripts.md Phase 5).

gofmt, go vet and go test are the real ones; golangci-lint is a stub that
passes, and the network gates (govulncheck, the API diff gate, the framework
examples) are skipped. Each case gets a repository of its own: a module with
one package and its test, and a copy of the scripts. Each run goes through
scripts/go-precheck.sh, as the machine-wide gate's does.

PRECHECK names the script under test (default go-precheck.py beside this
file, run through the shim); a file not ending in .py is copied as
scripts/go-precheck.sh itself, so the shell script it replaced can be tested
too. An error inside a case is reported as that case's FAIL, and the run
goes on (0017-MADR item 3).
"""

import os
import shutil
import subprocess
import sys
import tempfile
import traceback

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PRECHECK = os.environ.get("PRECHECK") or os.path.join(ROOT, "scripts", "go-precheck.py")
EXE = ".exe" if os.name == "nt" else ""

passed = 0
failed = 0
last_out = ""


def show():
    for line in last_out.rstrip("\n").split("\n"):
        print(f"         {line}", flush=True)


def check(name, want, got):
    global passed, failed
    if want == got:
        print(f"  ok   {name}", flush=True)
        passed += 1
    else:
        print(f"  FAIL {name}: want {want}, got {got}", flush=True)
        show()
        failed += 1


def has(name, text):
    global passed, failed
    if any(text in line for line in last_out.split("\n")):
        print(f"  ok   {name}", flush=True)
        passed += 1
    else:
        print(f"  FAIL {name}: no line holds {text} in:", flush=True)
        show()
        failed += 1


def hasnt(name, text):
    global passed, failed
    if any(text in line for line in last_out.split("\n")):
        print(f"  FAIL {name}: a line holds {text} in:", flush=True)
        show()
        failed += 1
    else:
        print(f"  ok   {name}", flush=True)
        passed += 1


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


def g(d, *args):
    """git, as a throwaway identity."""
    subprocess.run(["git", "-c", "user.name=precheck", "-c", "user.email=precheck@example.invalid",
                    "-c", "commit.gpgsign=false", *args], cwd=d, check=True, stdout=subprocess.DEVNULL)


def build(work, name, body):
    """A small Go program, built natively, standing in for a shell script."""
    src = os.path.join(work, "src", name)
    write(os.path.join(src, "go.mod"), f"module example.com/{name}\n\ngo 1.27.1\n")
    write(os.path.join(src, "main.go"), body)
    out_dir = os.path.join(work, "bin-" + name)
    exe = os.path.join(out_dir, ("go" if name == "warn-go" else name) + EXE)
    subprocess.run(["go", "build", "-o", exe, "."], cwd=src, check=True,
                   env={**os.environ, "GOWORK": "off", "CGO_ENABLED": "0"})
    return exe


def repo(d):
    """A repository holding module example.com/r with package p and its
    test, go.work, and the scripts, committed."""
    scripts = os.path.join(d, "scripts")
    os.makedirs(scripts)
    if PRECHECK.endswith(".py"):
        shutil.copy2(PRECHECK, os.path.join(scripts, "go-precheck.py"))
        shutil.copy2(os.path.join(ROOT, "scripts", "go-precheck.sh"), os.path.join(scripts, "go-precheck.sh"))
    else:
        shutil.copy2(PRECHECK, os.path.join(scripts, "go-precheck.sh"))
    shutil.copy2(os.path.join(ROOT, "scripts", "go-modules.py"), os.path.join(scripts, "go-modules.py"))
    g(d, "init", "-q")
    write(os.path.join(d, "go.mod"), "module example.com/r\n\ngo 1.27.1\n")
    write(os.path.join(d, "go.work"), "go 1.27.1\n\nuse .\n")
    write(os.path.join(d, "p", "p.go"), "package p\n\n// F is a function.\nfunc F() int { return 1 }\n")
    write(os.path.join(d, "p", "p_test.go"),
          'package p\n\nimport "testing"\n\nfunc TestF(t *testing.T) {\n\tif F() != 1 {\n\t\tt.Fatal(F())\n\t}\n}\n')
    g(d, "add", "-A")
    g(d, "commit", "-q", "-m", "base")
    return d


def git_bash():
    """The bash that runs the shim, as the machine-wide gate's own bash does:
    on Windows, Git's, found beside git itself, since `bash` there can name
    WSL's, which runs a Linux environment; elsewhere, bash on PATH."""
    if os.name != "nt":
        return "bash"
    exec_path = subprocess.run(["git", "--exec-path"], stdout=subprocess.PIPE, check=True).stdout.decode().strip()
    d = os.path.abspath(exec_path)
    while True:
        for rel in (("bin", "bash.exe"), ("usr", "bin", "bash.exe")):
            candidate = os.path.join(d, *rel)
            if os.path.isfile(candidate):
                return candidate
        parent = os.path.dirname(d)
        if parent == d:
            raise RuntimeError(f"no Git Bash bash.exe above {exec_path}")
        d = parent


def run(d, *files, env=None, direct=False):
    """The precheck's exit status, through the shim, or with direct the
    Python file itself, with stdout and stderr together kept."""
    global last_out
    cmd = ([sys.executable, "scripts/go-precheck.py"] if direct else [git_bash(), "scripts/go-precheck.sh"])
    # Without BASH_ENV or ENV: a host's start-up file could put its own
    # directories first on PATH, ahead of a go a case puts there.
    base = {k: v for k, v in os.environ.items() if k not in ("BASH_ENV", "ENV")}
    r = subprocess.run(cmd + list(files), cwd=d, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                       stderr=subprocess.STDOUT, check=False, env={**base, **(env or {})})
    last_out = r.stdout.decode("utf-8", "replace").replace("\r\n", "\n")
    return r.returncode


BROKEN = "package main\n\nfunc main( {\n"


def case1(work):
    # 1. A clean tree.
    a = repo(os.path.join(work, "a"))
    check("a clean tree passes", 0, run(a))
    has("it says clean", "file(s) clean")


def case2(work):
    # 2. An unformatted file fails, named under gofmt.
    b = repo(os.path.join(work, "b"))
    write(os.path.join(b, "p", "p.go"), "package p\n\n// F is a function.\nfunc  F() int { return 1 }\n")
    check("an unformatted file fails", 1, run(b, "p/p.go"))
    has("it is named", "these files are not formatted")


def case3_4(work):
    # 3. A file under testdata that does not parse fails at gofmt, in a
    #    file list: go vet and go test never see testdata, so only gofmt can.
    c = repo(os.path.join(work, "c"))
    write(os.path.join(c, "testdata", "x", "broken.go"), BROKEN)
    check("a testdata file that does not parse fails", 1, run(c, "testdata/x/broken.go"))
    has("gofmt's failure is named", "gofmt: failed")
    has("gofmt's message is shown", "expected ')'")
    # 4. The same file, tracked, fails with no list.
    g(c, "add", "-A")
    g(c, "commit", "-q", "-m", "broken")
    check("a tracked file that does not parse fails, no list", 1, run(c))
    has("gofmt's failure is named, no list", "gofmt: failed")


def case5(work):
    # 5. A tracked test file deleted from the work tree is not checked.
    d = repo(os.path.join(work, "d"))
    os.remove(os.path.join(d, "p", "p_test.go"))
    check("a deleted tracked file is skipped, no list", 0, run(d))
    hasnt("gofmt is not given it", "lstat")


def case6(work):
    # 6. A file list naming a missing file and a good one.
    e = repo(os.path.join(work, "e"))
    check("a missing file in a list is ignored", 0, run(e, "p/gone.go", "p/p.go"))


def case7(work):
    # 7. The shim passes its arguments, output and exit status through: the
    #    same as running the Python file
    #    (docs/decisions/0017-PLAN-python-repository-scripts.md Phase 5).
    f = repo(os.path.join(work, "f"))
    write(os.path.join(f, "p", "p.go"), "package p\n\n// F is a function.\nfunc  F() int { return 1 }\n")
    write(os.path.join(f, "testdata", "x", "broken.go"), BROKEN)
    args = ("testdata/x/broken.go",)
    rc_shim, out_shim = run(f, *args), last_out
    rc_direct = run(f, *args, direct=True)
    check("the shim's exit status is the Python file's", rc_direct, rc_shim)
    check("the shim's output is the Python file's", last_out, out_shim)
    check("the shim passes the file list", 1, rc_shim)


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
	_, _ = os.Stderr.WriteString("replace example.com/x => ./x\\n")
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		os.Exit(exit.ExitCode())
	}
}
"""


def case8(work):
    # 8. A nested module's go.mod is read from go's stdout alone: a line go
    #    writes to stderr, even one that reads as a replace directive, is not
    #    part of it (docs/decisions/0017-PLAN-python-repository-scripts.md D4).
    h = repo(os.path.join(work, "h"))
    write(os.path.join(h, "sub", "go.mod"), "module example.com/r/sub\n\ngo 1.27.1\n")
    write(os.path.join(h, "sub", "s.go"), "package sub\n\n// S is a function.\nfunc S() int { return 2 }\n")
    write(os.path.join(h, "go.work"), "go 1.27.1\n\nuse (\n\t.\n\t./sub\n)\n")
    g(h, "add", "-A")
    g(h, "commit", "-q", "-m", "sub")
    warner = build(work, "warn-go", WARNER)
    env = {"WARN_REAL": shutil.which("go"), "PATH": os.path.dirname(warner) + os.pathsep + os.environ["PATH"]}
    check("a nested module passes through a go that warns", 0, run(h, env=env))
    hasnt("the warning is not taken for a replace directive", "a replace directive")


def main():
    work = tempfile.mkdtemp()
    os.environ.pop("GOWORK", None)
    try:
        lint = build(work, "golangci-lint", "package main\n\nfunc main() {}\n")
        os.environ.update({"GOLANGCI_LINT": lint, "GO_PRECHECK_SKIP_VULN": "1", "GO_PRECHECK_SKIP_APICHECK": "1",
                           "GO_PRECHECK_SKIP_EXAMPLES": "1", "PYTHON": sys.executable})
        case("1. a clean tree", case1, work)
        case("2. an unformatted file", case2, work)
        case("3-4. a testdata file that does not parse", case3_4, work)
        case("5. a deleted tracked file", case5, work)
        case("6. a missing file in a list", case6, work)
        if PRECHECK.endswith(".py"):
            case("7. the shim", case7, work)
        case("8. a warning from go", case8, work)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    print(f"go-precheck_test: {passed} passed, {failed} failed")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
