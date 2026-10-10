---
status: in-progress
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
| `.gitignore` ignores `*.py[cod]`, not `__pycache__/` (2026-10-10, Phase 1: wrong; it ignores both, `.gitignore:17-18`) | `.gitignore:18` |
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
* **`.gitignore`:** `__pycache__/`. (2026-10-10: already there; nothing
  to add.)
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

### Phase 0: records

* The owner chose the route on 2026-10-10, over porting `go-fuzz_test`
  alone or patching case 6 in shell, and to run W4's Step 10 after
  Phase 1.
* The records were committed as `3e9d79c`, after W4's Step 9 as
  `4cb8b72`; the owner approved this PLAN ("approved to proceed").

### Phase 1: `go-fuzz`, with the case-6 fix

#### Deviations

No deviation. Three things found on the way, none changing the scope:

* **The facts table was wrong about `.gitignore`:** it already ignores
  `__pycache__/`. The agent's search for it matched only `*.py[cod]`.
  Nothing was added.
* **The recorder.** The shell test's case 1b put a bash script in `GO`.
  A native Windows Python cannot run a shell script as a program, so the
  Python test builds its recorder as a small Go program in its temporary
  directory, which runs natively on every host. The case and its checks
  are the same.
* **Line endings.** On the Windows test host, the first run failed one
  case, "its failure names the directory the input is in: want yes, got
  no": Python on Windows writes `\r\n` to a pipe, so the path the test
  read ended in `\r`. The harness reads the script's output with `\r\n`
  as `\n`. The shell test had run under Git Bash, which writes `\n`.

#### What was built

* **`scripts/go-fuzz.py`,** executable, the standard library only. It
  parses its arguments as `getopts ':t:z:m:al'` did, runs `go` from
  argument lists (`GO` overrides it), and prints the same `go-fuzz:`
  lines with the same exit codes. Its usage text names `go-fuzz.py`.
* **`scripts/go-fuzz_test.py`:** the shell test's 25 cases, by name, and
  a case 7 of two checks: a throwaway repository whose tracked test files
  `b/b_test.go` (a fuzz target) and `a/plain_test.go` are deleted and not
  staged, where the listing names only `example.com/gone/a`, and
  discovery agrees. Case 6's listing is `fuzz_packages`: `git ls-files`,
  the files that exist, and `^func Fuzz` found in Python. Each case runs
  under `case()`, which turns an exception into a FAIL line naming the
  case. `FUZZ` names the script under test, and a file not ending in
  `.py` runs as a program, so the shell script can be tested too.
* **Callers:** `Makefile`'s `PYTHON ?= python3`, and `make fuzz` runs
  `$(PYTHON) scripts/go-fuzz.py`; CI's `fuzz` step runs `python3
  scripts/go-fuzz_test.py`; CI's lint step prints `python3 --version` and
  runs `python3 -m py_compile scripts/*.py`.
* **Docs:** `AGENTS.md` gains "Scripts": Python 3.12 or later, the
  standard library only, argument lists, the harness rule, `PYTHON`, and
  which scripts are still shell. `docs/architecture.md`'s tree and `make
  fuzz` line, and `scripts/go-modules.sh`'s comment, name the Python
  files.
* **Removed:** `scripts/go-fuzz.sh` and `scripts/go-fuzz_test.sh`.

#### Checks

* **The comparison (rule 3),** on the same tree:
  * the shell test, "25 passed, 0 failed"; the Python test, "27 passed,
    0 failed"; every shell case's name is in the Python test, which adds
    case 7's two;
  * the Python test against `go-fuzz.sh` (`FUZZ`): "27 passed, 0 failed";
  * the two tools, run alike on eight inputs (the repository's
    discovery list; one package's targets; too few targets; discovery
    finding none; a failing target found by discovery; three usage
    errors): the same exit codes and `go-fuzz:` lines on each, such as
    "go-fuzz: FuzzBoom failed (exit 1); Go wrote the failing input under
    …/boom/testdata/fuzz/FuzzBoom/"; and `-a -l ./...` byte for byte, 8
    packages.
* **The probe,** on a scratch clone with `when/when_test.go`, a tracked
  test file with no fuzz target, deleted and not staged: `go-fuzz_test.sh`
  exited 1 with no FAIL line ("grep: when/when_test.go: No such file or
  directory"); `go-fuzz_test.py`, "27 passed, 0 failed".
* **Mutations,** on scratch copies, each killed:
  * **P1-1** (the listing keeps files that do not exist): "FAIL 7. a
    deleted, unstaged test file: the case stopped: FileNotFoundError", "25
    passed, 1 failed".
  * **P1-2** (case 2 raises): "FAIL 2. too few targets: the case
    stopped: RuntimeError: injected", and the run went on, "25 passed, 1
    failed", exit 1.
  * **P1-3** (`go-fuzz.py -a` drops one package): seven FAILs, among them
    "the repository's packages with targets are all found", "20 passed, 7
    failed".

  They ran again after the line-ending change, with the same results.
* **macOS:** `python3 -m py_compile scripts/*.py`; `go-fuzz_test.py`, "27
  passed, 0 failed"; `make fuzz FUZZTIME=3s`, "go-fuzz: 8 packages ran
  clean"; `make release-check`, "200 file(s) clean"; `make lint`;
  `shellcheck scripts/*.sh`; `markdownlint-cli2`; `actionlint`;
  `go-precheck_test.sh`, "12 passed"; `go-modules_test.sh`, "17 passed".
  All clean.
* **The Windows test host,** go1.27.2 windows/amd64, Python 3.14.7, with
  the index synced: `make pre-add-check`, `make lint`, `make vuln` and
  `make examples`, each exit 0; `go test -count=2 -shuffle=on ./...`,
  exit 0; `py_compile`, exit 0; `go-fuzz_test.py`, "27 passed, 0 failed";
  `make fuzz FUZZTIME=3s`, "8 packages ran clean". (The harness's "the
  step's tests" count reads `go test` lines, and printed 0 for a phase
  with no Go test.)
  * Before that run, a rerun the agent started without first copying its
    bundle was stopped; its remote process ran on, and a second run under
    the same label failed at once on the busy directory. The agent
    removed both runs' files on the host, and ran again under a new
    label; the stopped run's directory was gone when it finished.
