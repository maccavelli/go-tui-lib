#!/usr/bin/env python3
"""Offline tests for check_disclosure.py, on throwaway repositories in a
temporary directory, never in this one
(docs/decisions/0022-PLAN-align-docs-ci-and-disclosure-gates.md Phase 1).

Every real-looking path is built from pieces at run time, so this file
passes the check itself. The last case runs the script on this
repository, which must be clean. CHECK_DISCLOSURE names the script under
test (default check_disclosure.py beside this file). An error inside a case
is reported as that case's FAIL, and the run goes on (0017-MADR item 3).
"""

import os
import shutil
import subprocess
import sys
import tempfile
import traceback

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SCRIPT = os.environ.get("CHECK_DISCLOSURE") or os.path.join(ROOT, "scripts", "check_disclosure.py")

# An account no placeholder rule allows, and that the output must never
# show.
ACCT = "zq" + "xjv"
BS = "\\"

passed = 0
failed = 0


def check(name, want, got):
    global passed, failed
    if want == got:
        print(f"  ok   {name}", flush=True)
        passed += 1
    else:
        print(f"  FAIL {name}: want {want!r}, got {got!r}", flush=True)
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


def git(directory, *args):
    subprocess.run(["git", "-C", directory, *args], check=True, stdout=subprocess.DEVNULL)


def tree(directory, files, untracked=None):
    """A git repository with files (name to str or bytes) tracked, and
    untracked written but not added."""
    os.makedirs(directory)
    git(directory, "init", "-q")
    for group, add in ((files, True), (untracked or {}, False)):
        for rel, data in group.items():
            path = os.path.join(directory, rel)
            os.makedirs(os.path.dirname(path), exist_ok=True)
            with open(path, "wb") as f:
                f.write(data if isinstance(data, bytes) else data.encode("utf-8"))
        if add and group:
            git(directory, "add", "-A")
    return directory


def run(directory):
    r = subprocess.run([sys.executable, SCRIPT, "--root", directory], capture_output=True, check=False)
    out = (r.stdout + r.stderr).decode("utf-8", "replace").replace("\r\n", "\n")
    return r.returncode, out


def findings(out):
    return [line for line in out.split("\n") if line and not line.startswith("check-disclosure:")]


def case1(work):
    # 1. Each form, on its own line, by file, line and category; the output
    #    never holds the account.
    text = "\n".join([
        "clean line",
        "/Us" + "ers/" + ACCT + "/go",
        "in /ho" + "me/" + ACCT + "/src",
        "C:" + BS + "Users" + BS + ACCT + BS + "AppData",
        "c:/Users/" + ACCT + "/x",
        "D:" + BS * 2 + "Users" + BS * 2 + ACCT + BS * 2 + "x",
        "",
    ])
    d = tree(os.path.join(work, "forms"), {"a.txt": text})
    code, out = run(d)
    check("findings exit 1", 1, code)
    check("each form, by line and category", [
        "a.txt:2: macos-home", "a.txt:3: linux-home", "a.txt:4: windows-home",
        "a.txt:5: windows-home", "a.txt:6: windows-home",
    ], findings(out))
    check("the output does not hold the account", False, ACCT in out)
    check("the summary", True, "check-disclosure: 1 tracked file(s), 5 finding(s)" in out)


def case2(work):
    # 2. Placeholders on each form: angle brackets, JSON-escaped brackets,
    #    $ and % variables, the generic names, and dots or an ellipsis.
    names = ["<user>", "\\u003cuser\\u003e", "$USER", "%USERNAME%", "user", "u", "alice", "Bob", "carol", "dave",
             "...", "\u2026"]
    lines = []
    for n in names:
        lines += ["/Us" + "ers/" + n + "/x", "/ho" + "me/" + n + "/x", "C:" + BS + "Users" + BS + n + BS + "x",
                  "C:" + BS * 2 + "Users" + BS + n]
    d = tree(os.path.join(work, "placeholders"), {"p.txt": "\n".join(lines) + "\n"})
    code, out = run(d)
    check("placeholders pass", (0, []), (code, findings(out)))


def case3(work):
    # 3. Not a home path: inside a longer path, in a URL's path, or after a
    #    dotted name.
    text = "\n".join([
        "/tmp/ho" + "me/" + ACCT + "/x",
        "https://example.com/Us" + "ers/" + ACCT,
        "a.b/ho" + "me/" + ACCT,
        "x-/Us" + "ers/" + ACCT,
        "",
    ])
    d = tree(os.path.join(work, "not-home"), {"n.txt": text})
    code, out = run(d)
    check("paths that are not home paths pass", (0, []), (code, findings(out)))


def case4(work):
    # 4. Unreadable and binary: text that is not UTF-8 is refused, a file
    #    with a NUL byte is skipped, and a tracked file that is gone is
    #    refused.
    d = tree(os.path.join(work, "bytes"), {
        "latin.txt": b"caf\xe9\n",
        "bin.dat": b"\x00/Us" + b"ers/" + ACCT.encode() + b"/x",
        "gone.txt": "x\n",
    })
    os.remove(os.path.join(d, "gone.txt"))
    code, out = run(d)
    check("unreadable files fail", (1, ["gone.txt:0: unreadable", "latin.txt:0: unreadable"]),
          (code, sorted(findings(out))))


def case5(work):
    # 5. Only tracked files: an untracked file's home path is not read.
    d = tree(os.path.join(work, "untracked"), {"t.txt": "x\n"},
             untracked={"u.txt": "/ho" + "me/" + ACCT + "/x\n"})
    code, out = run(d)
    check("an untracked file is not read", (0, []), (code, findings(out)))


def case6(work):
    # 6. Outside a repository the script exits 2.
    bare = os.path.join(work, "bare")
    os.makedirs(bare)
    code, _ = run(bare)
    check("git fails: exit 2", 2, code)


def case7(_work):
    # 7. This repository is clean.
    code, out = run(ROOT)
    check("this repository is clean", (0, []), (code, findings(out)))


def main():
    work = tempfile.mkdtemp(prefix="check-disclosure-test-")
    try:
        for title, fn in [
            ("1 forms", case1), ("2 placeholders", case2), ("3 not home paths", case3),
            ("4 unreadable", case4), ("5 untracked", case5), ("6 no repository", case6),
            ("7 this repository", case7),
        ]:
            print(title, flush=True)
            case(title, fn, work)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    print(f"check_disclosure_test: {passed} passed, {failed} failed")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
