#!/usr/bin/env python3
"""Offline tests for check_records.py, on throwaway repositories in a
temporary directory, never in this one
(docs/decisions/0022-PLAN-align-docs-ci-and-disclosure-gates.md Phase 1).

Each case builds a small tree of records, an index and links, breaks one
thing, and reads the script's exit status and output. The last case runs
the script on this repository, which must be clean. CHECK_RECORDS names the
script under test (default check_records.py beside this file). An error
inside a case is reported as that case's FAIL, and the run goes on
(0017-MADR item 3).
"""

import os
import shutil
import subprocess
import sys
import tempfile
import traceback

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SCRIPT = os.environ.get("CHECK_RECORDS") or os.path.join(ROOT, "scripts", "check_records.py")

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


def write(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="\n") as f:
        f.write(text)


def git(directory, *args):
    subprocess.run(["git", "-C", directory, *args], check=True, stdout=subprocess.DEVNULL)


MADR = "docs/decisions/0001-MADR-first.md"
PLAN = "docs/decisions/0001-PLAN-first.md"
REPORT = "docs/reports/0002-REPORT-seen.md"

INDEX = """# Docs

## Records

| No. | Kind | Title | Status |
| :--- | :--- | :--- | :--- |
| 0001 | MADR | [First](decisions/0001-MADR-first.md) | accepted |
| 0001 | PLAN | [First, built](decisions/0001-PLAN-first.md) | complete |
| 0002 | REPORT | [Seen](reports/0002-REPORT-seen.md) | — |

## Guides

| I want to… | Read |
| :--- | :--- |
| see the plan | [PLAN](decisions/0001-PLAN-first.md#steps) |
"""


def tree(directory, extra=None, untracked=None):
    """A git repository with one MADR, its PLAN, a REPORT, an index and a
    guide; extra adds or replaces tracked files, untracked adds files git
    does not track. A value of None removes a file."""
    os.makedirs(directory)
    git(directory, "init", "-q")
    files = {
        MADR: "# First\n\nThe [plan](0001-PLAN-first.md#the-plan-step-1).\n",
        PLAN: "# First, built\n\n## Steps\n\n### The `plan`: step 1\n\nSee [guide](../guides/g.md#usage-1).\n",
        REPORT: "# Seen\n\nBy [0001](../decisions/0001-MADR-first.md).\n",
        "docs/README.md": INDEX,
        "docs/guides/g.md": "# G\n\n## Usage\n\n## Usage\n\nA [link in code](`x.md`) and `[x](gone.md)`.\n\n"
                            "```md\n[fenced](gone.md)\n```\n\n[web](https://example.com/x.md) [self](#usage)\n",
        "README.md": "# R\n\n[docs](docs/README.md)\n",
    }
    files.update(extra or {})
    for rel, text in files.items():
        if text is not None:
            write(os.path.join(directory, rel), text)
    git(directory, "add", "-A")
    for rel, text in (untracked or {}).items():
        write(os.path.join(directory, rel), text)
    return directory


def run(directory, *args):
    r = subprocess.run([sys.executable, SCRIPT, "--root", directory, *args], capture_output=True, check=False)
    out = (r.stdout + r.stderr).decode("utf-8", "replace").replace("\r\n", "\n")
    return r.returncode, out


def errors(out):
    return [line for line in out.split("\n") if line and not line.startswith("check-records:")]


def case1(work):
    # 1. A clean tree: exit 0, the counts, and no error. Links in code spans,
    #    fences and URLs are not links; a repeated heading's anchor is -1;
    #    a heading's code span and colon are dropped from its anchor.
    d = tree(os.path.join(work, "clean"))
    code, out = run(d)
    check("a clean tree exits 0", 0, code)
    check("a clean tree has no error", [], errors(out))
    check("the counts", True, "check-records: 3 record(s), 6 Markdown file(s), 0 error(s)" in out)


def case2(work):
    # 2. Names: a Markdown file in a record directory that is not a record,
    #    a record's kind in lowercase, and a lookalike elsewhere.
    d = tree(os.path.join(work, "names"), {
        "docs/decisions/notes.md": "# n\n",
        "docs/decisions/0003-madr-low.md": "# l\n",
        "docs/guides/12-PLAN-short.md": "# s\n",
    })
    code, out = run(d)
    check("misnamed records exit 1", 1, code)
    for f in ("docs/decisions/notes.md", "docs/decisions/0003-madr-low.md", "docs/guides/12-PLAN-short.md"):
        check(f"{f} is misnamed", True, any(e.startswith(f + ": misnamed record") for e in errors(out)))


def case3(work):
    # 3. Placement and pairs: a PLAN under docs/reports, a second MADR of a
    #    number, a PLAN with no MADR, a GATES with no PLAN. Two PLANs of one
    #    number are allowed.
    d = tree(os.path.join(work, "pairs"), {
        "docs/reports/0001-PLAN-misplaced.md": "# m\n",
        "docs/decisions/0001-MADR-again.md": "# a\n",
        "docs/decisions/0004-PLAN-orphan.md": "# o\n",
        "docs/reports/0005-GATES-alone.md": "# g\n",
        "docs/decisions/0001-PLAN-second.md": "# s\n",
        "docs/README.md": INDEX.replace("| 0002 |", "| 0001 | MADR | [A](decisions/0001-MADR-again.md) | x |\n"
                                        "| 0001 | PLAN | [M](reports/0001-PLAN-misplaced.md) | x |\n"
                                        "| 0001 | PLAN | [S](decisions/0001-PLAN-second.md) | x |\n"
                                        "| 0004 | PLAN | [O](decisions/0004-PLAN-orphan.md) | x |\n"
                                        "| 0005 | GATES | [G](reports/0005-GATES-alone.md) | x |\n| 0002 |"),
    })
    code, out = run(d)
    e = errors(out)
    check("pair errors exit 1", 1, code)
    check("a PLAN under docs/reports", True, "docs/reports/0001-PLAN-misplaced.md: a PLAN belongs in docs/decisions/" in e)
    check("two MADRs of 0001", True, any(x.startswith("0001: claimed by 2 MADRs") for x in e))
    check("a PLAN with no MADR", True, "0004: a PLAN with no MADR: docs/decisions/0004-PLAN-orphan.md" in e)
    check("a GATES with no PLAN", True, "0005: a GATES with no PLAN: docs/reports/0005-GATES-alone.md" in e)
    check("two PLANs of a number are allowed", 4, len(e))


def case4(work):
    # 4. The index: a record with no row; a row whose number is not its
    #    file's; a second row for one record. A row outside "## Records"
    #    does not count as the record's row.
    d = tree(os.path.join(work, "index"), {
        "docs/decisions/0006-MADR-unlisted.md": "# u\n",
        "docs/README.md": INDEX.replace("| 0002 | REPORT |", "| 0003 | REPORT |")
        + "| 0001 | MADR | [again](decisions/0001-MADR-first.md) | x |\n",
    })
    code, out = run(d)
    e = errors(out)
    check("index errors exit 1", 1, code)
    check("a record with no row", True, "docs/README.md: no row for docs/decisions/0006-MADR-unlisted.md" in e)
    check("a row's number differs", True,
          "docs/README.md:9: the row says 0003 REPORT, and links reports/0002-REPORT-seen.md" in e)
    check("the misnumbered row leaves its record unlisted", True,
          "docs/README.md: no row for docs/reports/0002-REPORT-seen.md" in e)
    d2 = tree(os.path.join(work, "index2"), {
        "docs/README.md": INDEX.replace("| 0002 | REPORT | [Seen](reports/0002-REPORT-seen.md) | — |\n",
                                        "| 0002 | REPORT | [Seen](reports/0002-REPORT-seen.md) | — |\n"
                                        "| 0002 | REPORT | [Seen](reports/0002-REPORT-seen.md) | — |\n"),
    })
    code, out = run(d2)
    check("a second row", ["docs/README.md:10: a second row for docs/reports/0002-REPORT-seen.md; the first is line 9"],
          errors(out))
    d3 = tree(os.path.join(work, "index3"), {
        "docs/README.md": INDEX.replace("| 0002 | REPORT | [Seen](reports/0002-REPORT-seen.md) | — |\n", "")
        + "| 0002 | REPORT | [Seen](reports/0002-REPORT-seen.md) | — |\n",
    })
    code, out = run(d3)
    check("a row under another heading is not the record's", ["docs/README.md: no row for docs/reports/0002-REPORT-seen.md"],
          errors(out))
    d4 = tree(os.path.join(work, "index4"), {"docs/README.md": None, "README.md": "# R\n"})
    code, out = run(d4)
    check("no index", ["docs/README.md: missing, and every record needs a row in it"], errors(out))


def case5(work):
    # 5. Links: a missing file, a missing anchor, a link out of the
    #    repository, and a broken link in the root README.md and in an
    #    agent-rule file. A link in a testdata file is not checked.
    d = tree(os.path.join(work, "links"), {
        "docs/guides/g.md": "# G\n\n## Usage\n\n[gone](gone.md) [anchor](../README.md#nowhere) [up](../../../x.md)\n",
        "README.md": "# R\n\n[x](docs/x.md)\n",
        ".claude/rules/r.md": "# r\n\n[y](../../y.md)\n",
        "command/testdata/c.md": "# c\n\n[z](z.md)\n",
        PLAN: "# First, built\n\n## Steps\n\n### The `plan`: step 1\n\nSee [guide](../guides/g.md#usage-1).\n",
    })
    code, out = run(d)
    e = errors(out)
    check("link errors exit 1", 1, code)
    check("a missing file", True, "docs/guides/g.md:5: broken link: gone.md" in e)
    check("a missing anchor", True, "docs/guides/g.md:5: broken anchor: ../README.md#nowhere" in e)
    check("a link out of the repository", True, "docs/guides/g.md:5: broken link: ../../../x.md" in e)
    check("the root README", True, "README.md:3: broken link: docs/x.md" in e)
    check("an agent-rule file", True, ".claude/rules/r.md:3: broken link: ../../y.md" in e)
    check("an anchor of a heading that is no longer repeated", True,
          "docs/decisions/0001-PLAN-first.md:7: broken anchor: ../guides/g.md#usage-1" in e)
    check("a testdata file is not checked", False, any(x.startswith("command/testdata/") for x in e))
    check("six errors", 6, len(e))


def case6(work):
    # 6. Only tracked files: an untracked record needs no row, and an
    #    untracked file's broken link is not read.
    d = tree(os.path.join(work, "untracked"), untracked={
        "docs/decisions/0007-MADR-draft.md": "# d\n\n[gone](gone.md)\n",
    })
    code, out = run(d)
    check("untracked files are not checked", (0, []), (code, errors(out)))


def case7(work):
    # 7. --next is one more than the highest number, a gap kept; outside a
    #    repository the script exits 2.
    d = tree(os.path.join(work, "next"), {"docs/decisions/0009-MADR-late.md": "# l\n"})
    code, out = run(d, "--next")
    check("--next after 0009", (0, "0010"), (code, out.strip()))
    bare = os.path.join(work, "bare")
    os.makedirs(bare)
    code, out = run(bare)
    check("git fails: exit 2", 2, code)


def case8(_work):
    # 8. This repository is clean.
    code, out = run(ROOT)
    check("this repository is clean", (0, []), (code, errors(out)))


def main():
    work = tempfile.mkdtemp(prefix="check-records-test-")
    try:
        for title, fn in [
            ("1 clean", case1), ("2 names", case2), ("3 pairs", case3), ("4 index", case4),
            ("5 links", case5), ("6 untracked", case6), ("7 next", case7), ("8 this repository", case8),
        ]:
            print(title, flush=True)
            case(title, fn, work)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    print(f"check_records_test: {passed} passed, {failed} failed")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
