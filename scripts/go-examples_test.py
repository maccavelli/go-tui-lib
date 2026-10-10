#!/usr/bin/env python3
"""Tests for go-examples.py, on throwaway trees in a temporary directory,
never this one (docs/decisions/0014-PLAN-api-policy-gates.md Step 5; ported
from go-examples_test.sh under
docs/decisions/0017-PLAN-python-repository-scripts.md Phase 4).

Each tree holds one program with no requirement, so the tests need no
network: it prints its argument to stdout and exits with status 3.

EXAMPLES names the script under test (default go-examples.py beside this
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
EXAMPLES = os.environ.get("EXAMPLES") or os.path.join(ROOT, "scripts", "go-examples.py")
SCRIPT = os.path.basename(EXAMPLES)

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
    """Whether a line of the last run's output holds text."""
    global passed, failed
    if any(text in line for line in last_out.split("\n")):
        print(f"  ok   {name}", flush=True)
        passed += 1
    else:
        print(f"  FAIL {name}: no line holds {text} in:", flush=True)
        show()
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


def read(path):
    with open(path, encoding="utf-8") as f:
        return f.read()


MAIN = """// Command echo prints its argument.
package main

import (
	"fmt"
	"os"
)

func main() {
	// guide:echo
	if len(os.Args) > 1 {
		fmt.Println(os.Args[1])
	}
	// guide:end
	os.Exit(3)
}
"""

GUIDE = """# A guide

<!-- from: testdata/frameworks/echo/main.go#echo -->

```go
if len(os.Args) > 1 {
    fmt.Println(os.Args[1])
}
```
"""


def tree(d):
    """A tree with one program, echo, one passing case, and a guide whose
    Go block is the program's region."""
    fw = os.path.join(d, "testdata", "frameworks")
    write(os.path.join(fw, "go.mod.tmpl"),
          "module example.com/frameworks\n\ngo 1.27.1\n\nreplace github.com/maccavelli/go-tui-lib => @ROOT@\n")
    write(os.path.join(fw, "go.sum"), "")
    write(os.path.join(fw, "echo", "main.go"), MAIN)
    write(os.path.join(fw, "cases.txt"), "echo hello => 3 stdout hello\n")
    write(os.path.join(d, "docs", "guides", "g.md"), GUIDE)
    return d


def run(d, *args, env=None):
    """The script's exit status, with stdout and stderr together kept for
    has() and check(), as the shell test's run kept them."""
    global last_out
    cmd = [sys.executable, EXAMPLES] if EXAMPLES.endswith(".py") else [EXAMPLES]
    r = subprocess.run(cmd + list(args), stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False,
                       env={**os.environ, "EXAMPLES_ROOT": d, **(env or {})})
    last_out = r.stdout.decode("utf-8", "replace").replace("\r\n", "\n")
    return r.returncode


def case1(work):
    # 1. A passing case, and an excerpt equal to its region once dedented
    #    and with tabs as four spaces.
    a = tree(os.path.join(work, "a"))
    check("a passing tree passes", 0, run(a))
    has("the case ran", "1 case(s) run")
    has("the excerpt was checked", "1 excerpt(s) checked")


def case2(work):
    # 2. A wrong exit status fails.
    b = tree(os.path.join(work, "b"))
    write(os.path.join(b, "testdata", "frameworks", "cases.txt"), "echo hello => 0 stdout hello\n")
    check("a wrong exit fails", 1, run(b))
    has("the exit is named", "exit 3, want 0")


def case3(work):
    # 3. The substring in the wrong stream fails.
    c = tree(os.path.join(work, "c"))
    write(os.path.join(c, "testdata", "frameworks", "cases.txt"), "echo hello => 3 stderr hello\n")
    check("a substring in the other stream fails", 1, run(c))
    has("the stream is named", 'stderr does not hold "hello"')


def case4(work):
    # 4. An excerpt one byte away from its region fails, in --check-guide
    #    too.
    d = tree(os.path.join(work, "d"))
    g = os.path.join(d, "docs", "guides", "g.md")
    write(g, read(g).replace("Println", "Printl"))
    check("a one-byte excerpt difference fails", 1, run(d, "--check-guide"))
    has("the difference is shown", "+    fmt.Printl(os.Args[1])")


def case5(work):
    # 5. A region the program does not have fails.
    e = tree(os.path.join(work, "e"))
    g = os.path.join(e, "docs", "guides", "g.md")
    write(g, read(g).replace("#echo", "#missing"))
    check("a missing region fails", 1, run(e, "--check-guide"))
    has("the region is named", 'has no region "// guide:missing"')


def case6(work):
    # 6. A from line with no Go block after it fails.
    f = tree(os.path.join(work, "f"))
    g = os.path.join(f, "docs", "guides", "g.md")
    write(g, read(g) + "\n<!-- from: testdata/frameworks/echo/main.go#echo -->\n\nNo block.\n")
    check("a from line without a block fails", 1, run(f, "--check-guide"))
    has("it is named", "is not followed by a Go block")


def case7(work):
    # 7. --check-guide neither builds nor runs: a program that does not
    #    compile passes it, and fails the full run with exit 2.
    g = tree(os.path.join(work, "g"))
    write(os.path.join(g, "testdata", "frameworks", "echo", "broken.go"),
          "package main\n\nfunc main() { undefined() }\n")
    check("--check-guide does not build", 0, run(g, "--check-guide"))
    check("a build failure exits 2", 2, run(g))
    has("the update is suggested", f"scripts/{SCRIPT} --update")


def case8(work):
    # 8. --update writes go.mod.tmpl and go.sum back, with @ROOT@ kept.
    h = tree(os.path.join(work, "h"))
    check("--update passes", 0, run(h, "--update"))
    tmpl = read(os.path.join(h, "testdata", "frameworks", "go.mod.tmpl"))
    kept = any(line.endswith("=> @ROOT@") for line in tmpl.split("\n")) and work not in tmpl
    check("--update keeps @ROOT@", "yes", "yes" if kept else "no")


def case9(work):
    # 9. The skip switch.
    check("the skip switch passes", 0, run(os.path.join(work, "g"), env={"GO_PRECHECK_SKIP_EXAMPLES": "1"}))


def main():
    work = tempfile.mkdtemp()
    for name in ("GOWORK", "GO_PRECHECK_SKIP_EXAMPLES"):
        os.environ.pop(name, None)
    try:
        case("1. a passing tree", case1, work)
        case("2. a wrong exit", case2, work)
        case("3. the other stream", case3, work)
        case("4. a one-byte excerpt difference", case4, work)
        case("5. a missing region", case5, work)
        case("6. a from line without a block", case6, work)
        case("7. --check-guide and a broken build", case7, work)
        case("8. --update", case8, work)
        case("9. the skip switch", case9, work)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    print(f"go-examples_test: {passed} passed, {failed} failed")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
