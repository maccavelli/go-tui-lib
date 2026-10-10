---
status: accepted
date: 2026-10-10
decision-makers: the repository owner
consulted: the agent that executes the PLAN
informed: contributors, through AGENTS.md
---
# Write the repository's scripts in Python, standard library only, and port the ten shell scripts

## Context and Problem Statement

Every script under `scripts/` is bash: five tools and five tests, about
1,700 lines (`go-apicheck`, `go-examples`, `go-fuzz`, `go-modules` and
`go-precheck`, each with a `_test.sh`). `make`, CI and the machine-wide
agent commit gate run them, and CI checks them with `shellcheck`. They
follow the scaffold this repository copied from go-core-lib
([0001-MADR-scaffold-charm-tui-library.md](0001-MADR-scaffold-charm-tui-library.md)).

The owner's global rules say to use Python, with `python3` and the
standard library, for loops, conditionals, string surgery and structured
data, and to add the Python ignore rules when such a script lives in a
non-Python repository. The scripts are mostly that kind of logic, written
as shell pipelines.

The cost showed on 2026-10-10. `go-fuzz_test.sh`'s case 6 compares fuzz
discovery with its own listing, built as `git ls-files --cached … | xargs
grep -l '^func Fuzz' | …`. While a tracked test file is deleted and not
staged, `ls-files` still lists it, `grep` cannot open it, and the script
exits 1 under `set -e` without a FAIL line. It happened in
[0014-PLAN-canonicalization.md](0014-PLAN-canonicalization.md) Steps 4 and
9. The agent wrote case 6 in
[0014-PLAN-hardening.md](0014-PLAN-hardening.md) Step 9, in shell, by the
repository's convention, without raising the conflict with the rule.

Which language should the repository's scripts be written in, and how is
the case-6 fault fixed?

## Decision Drivers

* **The owner's rule:** Python, `python3` and the standard library, for
  logic of this kind.
* **The fault class:** a shell pipeline under `set -e` can stop with no
  case named. A test harness must report every failure as a FAIL line
  naming its case.
* **Contracts outside the scripts stay whole:**
  * the machine-wide commit gate runs `./scripts/go-precheck.sh <files>`
    when that file is executable, and otherwise `make pre-add-check` over
    every file;
  * `make` targets, CI steps and records quote the scripts' output lines,
    such as "go-precheck: N file(s) clean" and "apicheck: clean".
* **No new dependency:** the standard library only, so no `pip`, no
  virtual environment and no lockfile.
* **Every host runs them:** macOS, CI's Ubuntu runner, which runs every
  script step, and the Windows test host, through Git Bash and `make`.
* **The org's coding rules:** no `subprocess(..., shell=True)`, and no
  `eval`.

## Considered Options

* Port every script to Python, standard library only, one pair at a time,
  `go-fuzz` first with the case-6 fix
* Port `go-fuzz_test` alone, and leave the rest in shell
* Patch case 6 in shell now, and port later
* Keep shell, and record the repository's scripts as an exception to the
  rule
* Rewrite the scripts in Go

## Decision Outcome

Chosen option: "Port every script to Python, standard library only, one
pair at a time, `go-fuzz` first with the case-6 fix", because the
owner's rule asks for Python for this logic, the case-6 fault is the kind
the rule prevents, and a pair at a time keeps each change checkable
against the script it replaces. The owner chose it on 2026-10-10, over
the two smaller routes.

1. **Language.** Each script is a Python 3 program using only the
   standard library, with `#!/usr/bin/env python3`. The floor is 3.12,
   Ubuntu 24.04's `python3`, the oldest of the hosts (macOS and the
   Windows test host have 3.14.7). `subprocess` runs commands from
   argument lists, never with `shell=True`.
2. **Behaviour is kept.** A ported script takes the same arguments and
   environment variables, exits with the same codes, and prints the same
   lines that `make`, CI and the records read. A ported test keeps the
   shell test's cases, by name and count, and its summary line, such as
   "go-fuzz_test: N passed, M failed".
3. **A test harness never stops silently.** An exception in a case is a
   FAIL line naming the case, and the run goes on to the next.
4. **The case-6 fix.** The listing reads only test files that exist in
   the work tree, parsed in Python. A new case pins it: a throwaway
   repository with a tracked fuzz test deleted and not staged.
5. **`scripts/go-precheck.sh` stays, as a shim** that `exec`s
   `python3` on `go-precheck.py` with its arguments, so the machine-wide
   gate keeps its contract. It holds no logic, so the rule does not reach
   it. `shellcheck` keeps checking it.
6. **The checks.** `python3 -m py_compile scripts/*.py` joins CI's lint
   step. A Python linter such as ruff is a new tool, and needs its own
   record.
7. **Ignore rules.** `.gitignore` already ignores `*.py[cod]`; it gains
   `__pycache__/`.
8. **Order.** `go-fuzz` first, then `go-modules`, `go-apicheck`,
   `go-examples` and `go-precheck`, each pair in one commit with its
   callers. W4's release `v0.10.0`
   ([0014-PLAN-canonicalization.md](0014-PLAN-canonicalization.md) Step
   10) follows the first pair.

### Consequences

* Good, because the scripts follow the owner's rule, and logic is written
  in a language with exceptions, data structures and tests.
* Good, because the case-6 fault is fixed and pinned, and a harness can
  no longer stop without naming the case.
* Good, because callers see no change: the arguments, outputs, exit codes
  and the gate's entry point stay.
* Neutral, because the shim is the one shell file left, and it holds no
  logic.
* Bad, because porting about 1,700 lines takes five commits, each to be
  checked against the script it replaces.
* Bad, because the repository leaves the go-core-lib scaffold's shell
  convention, which go-selfupdate-lib still follows.
* Bad, because `shellcheck` no longer covers the logic, and nothing yet
  lints Python beyond its syntax.

### Confirmation

* Each pair's ported test passes every case the shell test had, by name,
  on the same tree, before the shell pair is deleted.
* Case 6's new case fails on the shell test's listing, and passes on the
  Python one.
* `make`, CI, the Windows test host and the machine-wide gate, through the
  shim, run the ported scripts with the outputs the records quote.
* The PLAN's mutations are killed.

## Pros and Cons of the Options

### Port every script to Python, a pair at a time

* Good, because the whole repository follows the rule.
* Good, because each pair is checked against the script it replaces.
* Bad, because it is the largest change.

### Port `go-fuzz_test` alone

* Good, because it fixes the fault in the rule's language, in one commit.
* Bad, because the repository's scripts are then in two languages with no
  plan for the rest.

### Patch case 6 in shell now, port later

* Good, because the fix is a few lines.
* Bad, because it adds shell logic against the rule, to be ported again.

### Keep shell, and record an exception

* Good, because nothing is ported.
* Bad, because it writes the rule's opposite into the repository, and the
  fault class stays.

### Rewrite the scripts in Go

* Good, because Go is the repository's language, and its tests would run
  under `go test`.
* Bad, because the rule names Python.
* Bad, because tools inside the module join `./...`, and every gate,
  `govulncheck` and the conformance scan would scan them.

## More Information

* The machine-wide commit gate is outside this repository. It runs
  `./scripts/go-precheck.sh "${GO_FILES[@]}"` when the file is
  executable, and otherwise `make pre-add-check`.
* The case-6 failure is in
  [0014-PLAN-canonicalization.md](0014-PLAN-canonicalization.md)'s Step 9
  record: "grep: command/rename_test.go: No such file or directory", no
  FAIL line, exit 1; and "25 passed, 0 failed" on a clone with the
  deletions staged.
* The implementation is
  [0017-PLAN-python-repository-scripts.md](0017-PLAN-python-repository-scripts.md).
