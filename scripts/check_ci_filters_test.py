#!/usr/bin/env python3
"""Offline tests for check_ci_filters.py, on copies of this repository's
tracked files in a temporary directory, never on the tree itself
(docs/decisions/0022-PLAN-align-docs-ci-and-disclosure-gates.md Phase 2).

The tracked files are copied once into a git repository; each case copies
that, breaks one part of the contract, and requires that part's message.
The first case requires the copy to be clean, the last runs the script on
this repository. The YAML reader and the path globs have cases of their
own. CHECK_CI_FILTERS names the script under test (default
check_ci_filters.py beside this file). An error inside a case is reported
as that case's FAIL, and the run goes on (0017-MADR item 3).
"""

import importlib.util
import os
import shutil
import subprocess
import sys
import tempfile
import traceback

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SCRIPT = os.environ.get("CHECK_CI_FILTERS") or os.path.join(ROOT, "scripts", "check_ci_filters.py")
CI = ".github/workflows/ci.yml"
DOCS = ".github/workflows/docs.yml"
IGNORE = "['*.md', 'docs/**/*.md', '.claude/**/*.md', '.grok/**/*.md', '.opencode/**/*.md']"
DOCS_PATHS = ("['*.md', 'docs/**/*.md', '.claude/**/*.md', '.grok/**/*.md', '.opencode/**/*.md', "
              "'.markdownlint-cli2.jsonc', '.github/workflows/docs.yml']")
DOC_LINE = "make markdownlint check-records docs-check disclosure-check"

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


def base(work):
    """A git repository holding a copy of this repository's tracked files."""
    names = subprocess.run(["git", "-C", ROOT, "ls-files", "-z"], capture_output=True, check=True).stdout
    d = os.path.join(work, "base")
    for rel in (n for n in names.decode("utf-8").split("\0") if n):
        src = os.path.join(ROOT, rel)
        if not os.path.isfile(src):
            continue
        dst = os.path.join(d, rel)
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        shutil.copyfile(src, dst)
    git(d, "init", "-q")
    git(d, "add", "-A")
    return d


class Tree:
    """A copy of the base repository, to break."""

    n = 0

    def __init__(self, work, src):
        Tree.n += 1
        self.dir = os.path.join(work, f"t{Tree.n}")
        shutil.copytree(src, self.dir, symlinks=True)

    def edit(self, rel, old, new, count=1):
        path = os.path.join(self.dir, rel)
        with open(path, encoding="utf-8", newline="") as f:
            text = f.read()
        if text.count(old) != count:
            raise AssertionError(f"{rel}: {old!r} occurs {text.count(old)} time(s), not {count}")
        with open(path, "w", encoding="utf-8", newline="") as f:
            f.write(text.replace(old, new))
        return self

    def add(self, rel, text):
        path = os.path.join(self.dir, rel)
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, "w", encoding="utf-8", newline="\n") as f:
            f.write(text)
        git(self.dir, "add", rel)
        return self

    def run(self):
        return run(self.dir)


def run(directory):
    r = subprocess.run([sys.executable, SCRIPT, "--root", directory], capture_output=True, check=False)
    out = (r.stdout + r.stderr).decode("utf-8", "replace").replace("\r\n", "\n")
    problems = [line.removeprefix("check-ci-filters: ") for line in out.split("\n")
                if line.startswith("check-ci-filters: ") and not line.endswith("problem(s)")]
    return r.returncode, problems


def has(problems, prefix):
    return any(p.startswith(prefix) for p in problems)


def case1(work, b):
    # 1. The copy of this repository is clean.
    check("the copy is clean", (0, []), Tree(work, b).run())


def case2(work, b):
    # 2. filters: CI's pull_request list loses a pattern; CI gains a paths
    #    filter; CI loses its v* tags, its branch, or pull_request.
    code, p = Tree(work, b).edit(CI, "  pull_request:\n    paths-ignore: " + IGNORE,
                                  "  pull_request:\n    paths-ignore: ['*.md']").run()
    check("lists differ: exit 1", 1, code)
    check("lists differ", True, has(p, "filters: CI's push and pull_request paths-ignore differ"))
    _, p = Tree(work, b).edit(CI, "  pull_request:\n", "  pull_request:\n    paths: ['go.mod']\n").run()
    check("a paths filter", True, has(p, "filters: CI has a paths filter"))
    _, p = Tree(work, b).edit(CI, "    tags: ['v*']\n", "").run()
    check("tags lost", True, "filters: CI lost its trigger: push tags v*" in p)
    _, p = Tree(work, b).edit(CI, "    branches: [main]\n", "    branches: [release]\n").run()
    check("the branch lost", True, "filters: CI lost its trigger: push branches main" in p)
    _, p = Tree(work, b).edit(CI, "  pull_request:\n    paths-ignore: " + IGNORE + "\n", "").run()
    check("pull_request lost", True, "filters: CI lost its trigger: pull_request" in p)


def case3(work, b):
    # 3. docs: Docs' push paths lose a pattern; both lose it.
    code, p = Tree(work, b).edit(DOCS, "  push:\n    branches: [main]\n    paths: " + DOCS_PATHS,
                                  "  push:\n    branches: [main]\n    paths: ['*.md']").run()
    check("Docs' lists differ", True, has(p, "docs: Docs' push and pull_request paths differ"))
    _, p = Tree(work, b).edit(DOCS, "'.grok/**/*.md', ", "", count=2).run()
    check("Docs misses an ignored pattern", True, has(p, "docs: Docs' paths are not CI's ignored paths"))
    check("its message names it", True, any("missing ['.grok/**/*.md']" in x for x in p))
    check("and its files", True, "tracked: CI ignores .grok/rules/madr-plan-before-mutating-work.md, "
                                 "and Docs does not run on it" in p)


def case4(work, b):
    # 4. tracked: an ignored pattern that matches nothing.
    t = Tree(work, b).edit(CI, IGNORE, IGNORE[:-1] + ", 'nothing/**/*.md']", count=2)
    t.edit(DOCS, "'.markdownlint-cli2.jsonc'", "'nothing/**/*.md', '.markdownlint-cli2.jsonc'", count=2)
    check("a pattern with no file", ["tracked: CI ignores 'nothing/**/*.md', which matches no tracked file"], t.run()[1])


def case5(work, b):
    # 5. readers: CI ignores the command fixtures under testdata.
    t = Tree(work, b).edit(CI, IGNORE, IGNORE[:-1] + ", 'command/testdata/**/*.md']", count=2)
    t.edit(DOCS, "'.markdownlint-cli2.jsonc'", "'command/testdata/**/*.md', '.markdownlint-cli2.jsonc'", count=2)
    _, p = t.run()
    check("testdata ignored", True,
          "readers: CI ignores command/testdata/commands/notes.md, under a testdata directory go test reads" in p)


def case6(work, b):
    # 6. readers: Docs no longer runs docs-check; its recipe drops a reader.
    _, p = Tree(work, b).edit(DOCS, DOC_LINE, "make markdownlint check-records disclosure-check").run()
    check("Docs without docs-check", True, any(
        x.startswith("readers: CI ignores docs/guides/defaults.md, docs/decisions/*.md, and Docs does not run "
                      "make docs-check") for x in p))
    check("and the gate's message", True, "gates: Docs does not run make docs-check" in p)
    _, p = Tree(work, b).edit("Makefile", "-run '^TestDefaultsInventory$$' ", "").run()
    check("the recipe drops the inventory", ["readers: make docs-check does not run TestDefaultsInventory, "
                                             "the reader in internal/conformance/defaults_test.go"], p)
    _, p = Tree(work, b).edit("Makefile", "./scripts/go-examples.py --check-guide", "./scripts/go-examples.py").run()
    check("the recipe drops the guide check", ["readers: make docs-check does not run --check-guide, "
                                               "the reader in scripts/go-examples.py"], p)


def case7(work, b):
    # 7. readers: a Go file names an ignored document no reader covers, as
    #    a string or by filepath.Join's literal parts; a NOT_READS line
    #    whose literal is gone.
    t = Tree(work, b).add("x/x_test.go", 'package x\n\nvar a = "../docs/glossary.md"\n'
                          'var b = filepath.Join(root, "docs", "architecture.md")\n')
    _, p = t.run()
    check("a string names an ignored file", True,
          "readers: CI ignores docs/glossary.md, which x/x_test.go names, and no reader of it runs in Docs" in p)
    check("a Join names one", True,
          "readers: CI ignores docs/architecture.md, which x/x_test.go names, and no reader of it runs in Docs" in p)
    _, p = Tree(work, b).edit("palette/providers_test.go", '"README.md":', '"READ.txt":').run()
    check("a stale NOT_READS line", ["readers: NOT_READS lists palette/providers_test.go naming README.md, "
                                     "which it no longer does"], p)


def case8(work, b):
    # 8. gates: CI runs markdownlint-cli2 itself; the Makefile pins it twice;
    #    CI drops the disclosure check.
    _, p = Tree(work, b).edit(CI, "          python3 -m py_compile scripts/*.py\n",
                              "          python3 -m py_compile scripts/*.py\n"
                              "          npx --yes markdownlint-cli2@0.23.2\n").run()
    check("CI runs markdownlint itself", ["gates: CI runs markdownlint-cli2 itself, not through make markdownlint"], p)
    _, p = Tree(work, b).edit(
        "Makefile", "MARKDOWNLINT ?= npx --yes markdownlint-cli2@0.23.2\n",
        "MARKDOWNLINT ?= npx --yes markdownlint-cli2@0.23.2\n# markdownlint-cli2@0.23.1 was older\n").run()
    check("the pin twice", ["gates: the Makefile names markdownlint-cli2@ 2 time(s), not once"], p)
    _, p = Tree(work, b).edit(CI, DOC_LINE, "make markdownlint check-records docs-check").run()
    check("CI drops the disclosure check", ["gates: CI does not run make disclosure-check"], p)


def case9(work, b):
    # 9. pins: Docs checks out at another pin; Docs' permissions widened.
    _, p = Tree(work, b).edit(DOCS, "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1",
                              "actions/checkout@0000000000000000000000000000000000000000").run()
    check("another checkout pin", True, any(x.startswith("pins: Docs uses actions/checkout@0000") for x in p))
    _, p = Tree(work, b).edit(DOCS, "  contents: read\n", "  contents: write\n").run()
    check("Docs may write", ["pins: Docs' permissions are not contents: read"], p)


def case10(work, b):
    # 10. Outside the YAML subset, or no repository: exit 2.
    for title, old, new in [
        ("an anchor", "    runs-on: ubuntu-24.04\n    timeout-minutes", "    runs-on: &os ubuntu-24.04\n    timeout-minutes"),
        ("a duplicate key", "    timeout-minutes: 10\n", "    timeout-minutes: 10\n    timeout-minutes: 11\n"),
        ("a flow mapping", "permissions:\n  contents: read\n", "permissions: {contents: read}\n"),
        ("a tab in indentation", "    timeout-minutes: 10\n", "\ttimeout-minutes: 10\n"),
        ("a document marker", "name: Docs\n", "---\nname: Docs\n"),
    ]:
        code, _ = Tree(work, b).edit(DOCS, old, new).run()
        check(f"{title} is refused", 2, code)
    bare = os.path.join(work, "bare")
    os.makedirs(bare)
    check("git fails: exit 2", 2, run(bare)[0])


def case11(_work, _b):
    # 11. The reader's values, and the globs.
    spec = importlib.util.spec_from_file_location("check_ci_filters", SCRIPT)
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    got = m.load("on:\n  push:\n    paths: ['a b', \"c\", d]  # note\nx: |\n  one\n  two\ny:\n  - k: v\n    l: w\n  - z\n")
    check("the reader", {"on": {"push": {"paths": ["a b", "c", "d"]}}, "x": "one\ntwo\n",
                         "y": [{"k": "v", "l": "w"}, "z"]}, got)
    for pattern, path, want in [
        ("*.md", "README.md", True), ("*.md", "docs/a.md", False), ("docs/**/*.md", "docs/a.md", True),
        ("docs/**/*.md", "docs/a/b/c.md", True), ("**/x", "x", True), ("**/x", "a/b/x", True),
        (".claude/**/*.md", ".claude/rules/r.md", True), ("a?c", "abc", True), ("a?c", "a/c", False),
    ]:
        check(f"{pattern} on {path}", want, bool(m.glob_re(pattern).match(path)))


def case12(_work, _b):
    # 12. This repository holds to the contract.
    check("this repository", (0, []), run(ROOT))


def main():
    work = tempfile.mkdtemp(prefix="check-ci-filters-test-")
    try:
        b = base(work)
        for title, fn in [
            ("1 clean", case1), ("2 filters", case2), ("3 docs", case3), ("4 tracked", case4),
            ("5 testdata", case5), ("6 readers", case6), ("7 named files", case7), ("8 gates", case8),
            ("9 pins", case9), ("10 refused", case10), ("11 reader and globs", case11),
            ("12 this repository", case12),
        ]:
            print(title, flush=True)
            case(title, fn, work, b)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    print(f"check_ci_filters_test: {passed} passed, {failed} failed")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
