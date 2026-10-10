---
status: proposed
date: 2026-10-10
associated-madr: "0017-MADR-python-repository-scripts.md"
---
# Implement Python repository scripts: port the ten shell scripts, `go-fuzz` first with the case-6 fix

Associated MADR: [0017-MADR-python-repository-scripts.md](0017-MADR-python-repository-scripts.md)

## Goal

Every script under `scripts/` is a Python 3.12+ program using only the
standard library, with the arguments, outputs and exit codes its callers
use today. `go-fuzz_test`'s case 6 no longer stops silently while a
tracked test file is deleted and not staged. `scripts/go-precheck.sh`
stays as a shim for the machine-wide commit gate.

## Scope

### Facts this PLAN starts from (2026-10-10)

| Fact | Where |
| :--- | :--- |
| Ten shell scripts, 1,726 lines: `go-apicheck` (150) and its test (146), `go-examples` (279, 151), `go-fuzz` (138, 138), `go-modules` (94, 129), `go-precheck` (369, 132) | `scripts/*.sh` |
| `make` runs `go-modules.sh` (`MODULES_CMD`), `go-fuzz.sh` (`fuzz`), `go-apicheck.sh` (`apicheck`), `go-examples.sh` (`examples`) and `go-precheck.sh` (`pre-add-check`, `release-check`) | `Makefile:28`, `:101`, `:106`, `:112`, `:119`, `:123` |
| CI runs every script step on `ubuntu-24.04`: the five tests, `go-modules.sh` and `--check`, and `shellcheck scripts/*.sh` | `.github/workflows/ci.yml:38-40`, `:115`, `:130`, `:159`, `:167`, `:173`, `:194` |
| The scripts call each other: `go-precheck.sh` runs `go-modules.sh`, `go-apicheck.sh` and `go-examples.sh`; `go-apicheck.sh` runs `go-modules.sh` | the scripts |
| The machine-wide commit gate runs `./scripts/go-precheck.sh <files>` when it is executable, else `make pre-add-check` | outside the repository |
| `AGENTS.md`, `docs/architecture.md` and `docs/guides/releasing.md` name the scripts | `git grep scripts/go-` |
| `python3` is 3.14.7 on macOS and on the Windows test host; Ubuntu 24.04's is 3.12 | `python3 --version`; the runner image |
| `.gitignore` ignores `*.py[cod]`, not `__pycache__/` | `.gitignore:18` |
| Case 6's fault: exit 1, "grep: …: No such file or directory", no FAIL line, while a tracked test file is deleted and not staged | [0014-PLAN-canonicalization.md](0014-PLAN-canonicalization.md) Step 9 |

### In scope

| Phase | Delivers |
| :--- | :--- |
| 0 | these records, approved |
| 1 | `go-fuzz.py` and `go-fuzz_test.py`, with the case-6 fix; `py_compile` in CI; `__pycache__/`; `AGENTS.md`'s scripts rule |
| 2 | `go-modules.py` and its test |
| 3 | `go-apicheck.py` and its test |
| 4 | `go-examples.py` and its test |
| 5 | `go-precheck.py` and its test; `go-precheck.sh` as the shim |
| 6 | close-out: the documents, this PLAN `complete` |

[0014-PLAN-canonicalization.md](0014-PLAN-canonicalization.md) Step 10,
the release `v0.10.0`, runs after Phase 1 is committed, as its D4
records.

### Out of scope

* **A Python linter** (MADR item 6): its own record.
* **The machine-wide gate's own code,** outside this repository: the shim
  keeps its contract.
* **go-selfupdate-lib's scripts.**
* **New behaviour** in any script beyond the case-6 fix and the harness
  rule: a change found while porting is a deviation.

## Implementation Steps

### Rules

1. **A phase starts when the previous one is committed.** The agent
   commits only when the owner asks in that turn, with `git commit
   --no-edit`. The owner pushes and tags.
2. **One pair a phase,** with its callers (`Makefile`, CI, the other
   scripts, the docs that name it), in one commit. The shell pair is
   deleted in that commit, after the comparison in rule 3.
3. **The comparison,** on the same tree, before the shell pair goes:
   * the Python test passes every case the shell test has, by name, and
     its summary line has the same counts, plus any case this PLAN adds;
   * the Python tool, run as its callers run it, prints the lines they
     read, byte for byte, and exits with the same code, on a passing and a
     failing input.
4. **Each script:** `#!/usr/bin/env python3`, executable, the standard
   library only, `subprocess.run` with an argument list, never
   `shell=True`; a `main()` returning the exit code; argument handling
   that accepts what the shell script accepted.
5. **Each test harness** runs its cases on throwaway directories under a
   temporary directory, as the shell tests do; an exception in a case is a
   FAIL line naming the case, and the run continues.
6. **The checks of each phase:** `python3 -m py_compile scripts/*.py`;
   the pair's test; `make lint`, `make release-check`; CI's other gates
   (`shellcheck` on the shell scripts left, `markdownlint-cli2`,
   `actionlint`); the Windows test host, with its index synced to the
   tree; CI green after the push.
7. **Mutations** run on scratch copies, and each must be killed by the
   phase's test.
8. **Anything unplanned stops the phase.**

### Phase 0: records

These two records; `docs/README.md` rows; a dated D4 in
[0014-PLAN-canonicalization.md](0014-PLAN-canonicalization.md), ordering
its Step 10 after Phase 1; and a dated note in
[0014-PLAN-hardening.md](0014-PLAN-hardening.md)'s Step 9 record, that
case 6's fault is fixed here. The owner approves.

### Phase 1: `go-fuzz`, with the case-6 fix

* **`scripts/go-fuzz.py`** replaces `go-fuzz.sh`: `-a` (discover), `-l`
  (list), `-t FUZZTIME`, `-z MINIMIZETIME`, `-m MIN` and the package or
  patterns; the lines "go-fuzz: N fuzz targets ran clean in <pkg>",
  "go-fuzz: N packages ran clean", and on a failure "go-fuzz: <target>
  failed (exit N); Go wrote the failing input under <dir>"; the same exit
  codes.
* **`scripts/go-fuzz_test.py`** replaces `go-fuzz_test.sh`, with its 25
  cases and "go-fuzz_test: N passed, M failed".
  * **Case 6's listing** reads `git ls-files --cached --others
    --exclude-standard '*_test.go'`, keeps the files that exist, and
    finds `^func Fuzz` in Python.
  * **A new case** builds a throwaway repository with two packages'
    fuzz tests, deletes one tracked file without staging it, and checks
    that the listing names only the package that still has a target.
* **Callers:** `Makefile`'s `fuzz` target and CI's `fuzz` step run the
  Python files; a `PYTHON ?= python3` variable in the `Makefile`.
* **CI:** `python3 -m py_compile scripts/*.py` in the lint step.
* **`.gitignore`:** `__pycache__/`.
* **`AGENTS.md`:** scripts are Python, standard library only
  (0017-MADR); the shell scripts left are listed until their phase.
* **Probe:** on a scratch copy with a tracked fuzz test deleted and not
  staged, `go-fuzz_test.sh` exits 1 with no FAIL line, and
  `go-fuzz_test.py` passes every case.
* **Mutations:**
  * P1-1: the listing keeps files that do not exist; the new case.
  * P1-2: a case raises; the run reports it as FAIL by name, goes on, and
    exits 1.
  * P1-3: `go-fuzz.py -a` drops one package; case 6.

### Phase 2: `go-modules`

* **`scripts/go-modules.py`** replaces `go-modules.sh`: the module list,
  one per line, and `--check`, with its messages and exit codes.
* **`scripts/go-modules_test.py`** replaces its test, with every case.
* **Callers:** `Makefile`'s `MODULES_CMD`; CI's steps that list modules;
  `go-apicheck.sh` and `go-precheck.sh`, still shell, call the Python
  file.
* **Mutation P2-1:** `--check` ignores a `go.mod` that `go.work` omits;
  the test.

### Phase 3: `go-apicheck`

* **`scripts/go-apicheck.py`** replaces `go-apicheck.sh`: the base tag
  (the newest the HEAD does not contain), `apidiff` at the `Makefile`'s
  pin or `APIDIFF`, `scripts/apicheck.allow`'s entries and stale
  entries, and the lines "apicheck: <m>: against <tag>, N incompatible
  change(s)" and "apicheck: clean".
* **`scripts/go-apicheck_test.py`** replaces its test.
* **Callers:** `make apicheck`; `go-precheck.sh`.
* **Mutation P3-1:** a stale entry is not reported; the test.

### Phase 4: `go-examples`

* **`scripts/go-examples.py`** replaces `go-examples.sh`: the temporary
  module, the framework examples' builds and cases, and the guide's
  excerpts; "examples: N excerpt(s) checked" and "examples: clean".
* **`scripts/go-examples_test.py`** replaces its test.
* **Callers:** `make examples`; `go-precheck.sh`.
* **Mutation P4-1:** an excerpt that differs from its source passes; the
  test.

### Phase 5: `go-precheck`, and the shim

* **`scripts/go-precheck.py`** replaces the logic of `go-precheck.sh`:
  the modules a file list belongs to, `gofmt` and its own failures
  (0015-MADR), the three `golangci-lint` runs, `go vet`, `go test`,
  `go mod tidy -diff`, `govulncheck`, workspace-mode tests, the nested
  module's checks, `go-modules`' check, and apicheck and examples with no
  list; the skip variables; "go-precheck: N file(s) clean in M
  module(s) (…)".
* **`scripts/go-precheck.sh`** becomes the shim: `exec python3
  "$(dirname "$0")/go-precheck.py" "$@"`, executable.
* **`scripts/go-precheck_test.py`** replaces its test, with its 12
  cases, and a case that the shim passes its arguments and exit code
  through.
* **Callers:** `make pre-add-check` and `release-check` call the shim or
  the Python file; the machine-wide gate calls the shim unchanged.
* **Check:** on a scratch clone with a staged Go file, the machine-wide
  gate runs the check through the shim and reports a gofmt failure that
  the scratch file carries.
* **Mutation P5-1:** the shim drops its arguments; the test.

### Phase 6: close-out

`docs/architecture.md`'s tooling and tree, `docs/guides/releasing.md`,
and `AGENTS.md`'s pre-add text name the Python files; `AGENTS.md`'s
scripts rule loses the list of shell scripts left; this PLAN
`complete`, and the MADR's index row notes it built.

## Verification

* Each phase's comparison (rule 3) holds, before its shell pair is
  deleted.
* The case-6 probe fails on the shell test and passes on the Python one.
* Every mutation is killed.
* `make release-check`, `make fuzz`, `make apicheck`, `make examples` and
  the five tests pass on macOS and on the Windows test host, and CI is
  green, after each phase.
* After Phase 5, `scripts/` holds no shell file but the shim, and the
  machine-wide gate's run through it is recorded.

## Rollout and Rollback

Each phase is one commit; reverting it restores that shell pair and its
callers, since the other scripts call it by path and are unchanged by the
revert. The shim keeps the machine-wide gate's entry point through every
phase. No tag is needed: the scripts are not part of the module's API.

## Execution Record

Not started.
