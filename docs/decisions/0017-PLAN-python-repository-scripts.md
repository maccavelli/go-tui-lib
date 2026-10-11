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
     failing input;
   * each case reads the same stream the shell case read: stdout alone,
     or stdout and stderr together;
   * the Python test passes with an empty module cache (`GOMODCACHE` a new
     directory), as on CI, where `go` writes "go: downloading" lines to
     stderr (2026-10-10, D1).
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
* **2026-10-10, D2:** the module list comes from `go list`'s stdout
  alone; a warning on its stderr passes through to stderr, and a failure
  prints what `go` wrote, as before. A case pins it, and mutation P2-2
  reads stderr into the list again.

### Phase 3: `go-apicheck`

* **`scripts/go-apicheck.py`** replaces `go-apicheck.sh`: the base tag
  (the newest the HEAD does not contain), `apidiff` at the `Makefile`'s
  pin or `APIDIFF`, `scripts/apicheck.allow`'s entries and stale
  entries, and the lines "apicheck: <m>: against <tag>, N incompatible
  change(s)" and "apicheck: clean".
* **`scripts/go-apicheck_test.py`** replaces its test.
* **Callers:** `make apicheck`; `go-precheck.sh`.
* **Mutation P3-1:** a stale entry is not reported; the test.
* **2026-10-10, D3:** the module path comes from `go list -m`'s stdout
  alone, as D2's list does; a case pins it, and mutation P3-2 reads
  stderr into the path again.

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
* **2026-10-10, D4:** a nested module's `go.mod` text comes from `go mod
  edit -print`'s stdout alone, as D2's list and D3's path do; a case pins
  it, and mutation P5-2 reads stderr in again.

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

* **D1 (2026-10-10): case 6 read discovery's stderr.**
  * **Found:** CI run `38079124213`, on `ab0ed44`, failed its `fuzz` step:
    "FAIL the repository's packages with targets are all found", the
    "got" list holding the eight packages and, around them, "go:
    downloading" lines and module paths. The shell test's case 6 read only
    discovery's stdout (`"$FUZZ" -a -l ./... | sort`); the port ran it
    through `Runner.run`, which merges stderr into the output, as every
    other case's `run` did. On CI's empty module cache, `go list` writes
    "go: downloading …" to stderr. The agent's runs on macOS and the
    Windows test host had warm caches, so the comparison passed. With an
    empty `GOMODCACHE`, the agent reproduced it: "26 passed, 1 failed".
    The tag `v0.10.0` was not yet made.
  * **The owner's choice:** `Runner` keeps stdout apart from the merged
    output; cases 6 and 7 compare discovery's stdout, as the shell test
    did; and rule 3 gains two checks for every phase: each case reads the
    stream its shell case read, and the test passes with an empty module
    cache.
  * **The other:** the same fix without the cold-cache check.

Three other things found on the way, none changing the scope:

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

#### D1's fix and checks

* **The fix:** `Runner.run` takes `stdout_only`; cases 6 and 7 read
  discovery's list from stdout alone, and every other case reads stdout
  and stderr together, as each shell case did.
* **The probe** is the reproduction: with an empty `GOMODCACHE`, before
  the fix, "FAIL the repository's packages with targets are all found",
  "26 passed, 1 failed".
* **After the fix,** "27 passed, 0 failed" with a warm cache and with an
  empty one, on macOS and on the Windows test host (Python 3.14.7). The
  empty caches were cleaned with `go clean -modcache` and removed.

### Phase 2: `go-modules`

#### Deviations

* **D2 (2026-10-10): `go list`'s stderr was read as module directories.**
  * **Found,** porting `go-modules.sh`: it ran `dirs="$(… go list -m -f
    '{{.Dir}}' 2>&1)"`, reading stderr with stdout so that a failure could
    print `go`'s error. On success, a line `go` wrote to stderr, such as a
    warning, became a "directory": `git -C` fails on it, so it is kept as
    a directory outside the repository, printed in the list, and reported
    by `--check` as a module with no tracked `go.mod`. The agent ported it
    as it was, as rule 3 and the out-of-scope list require, and named it in
    the handoff.
  * **The owner's choice:** fix it in this phase. The list comes from
    `go list`'s stdout alone; on success, anything `go` wrote to stderr
    passes through to the script's stderr; on failure, the message and
    `go`'s output, indented, are as before.
  * **The other:** keep the port faithful, and leave it to a later record.

#### What was built

* **`scripts/go-modules.py`,** executable, the standard library only: the
  list, `--check`, the messages and exit codes of `go-modules.sh`. Its
  stdout and stderr write `\n` alone, on Windows too: its callers read
  the list with `$(...)`, which would keep a `\r`. By D2, the list comes
  from `go list`'s stdout alone; what `go` writes to stderr passes
  through.
* **`scripts/go-modules_test.py`:** the shell test's 17 cases, by name,
  and two more:
  * case 9, two checks: a listing's bytes hold no `\r`;
  * case 10, four checks (D2): through a `go` that writes a warning to
    stderr and succeeds, the listing is `.` and `sub` alone, the warning
    reaches stderr, and `--check` passes.

  Case 8's other spelling of the directories, a bash wrapper in the shell
  test, is a small Go program the test builds, as Phase 1's recorder is;
  case 10 uses it too.
* **Callers:** `Makefile`'s `MODULES_CMD` is `$(PYTHON)
  ./scripts/go-modules.py`; CI's `modules` step runs the test, `--check`
  and the list with `python3`, and its `gates` step lists the modules so;
  `go-apicheck.sh` and `go-precheck.sh` run `"${PYTHON:-python3}"` on
  it; `go-precheck_test.sh` copies it into its throwaway repositories.
* **Docs:** `AGENTS.md` (the modules section, the pre-add text, and the
  scripts still in shell), `docs/architecture.md` and
  `docs/guides/releasing.md` name the Python files; so do the shell
  scripts' comments.
* **Removed:** `scripts/go-modules.sh` and `scripts/go-modules_test.sh`.
* **Committed in two parts:** the owner committed the port as `a2a6a17`
  while its Windows run was going, and D2, with its deviation entry, as
  `5832dc9`.

#### Checks

* **The comparison (rule 3),** before D2, on the same tree: the shell
  test, "17 passed, 0 failed"; the Python test, every shell case by name,
  against `go-modules.py` and against `go-modules.sh` (`MODULES`); and the
  two scripts, on nine inputs (the repository's list and `--check`; a
  module missing from `go.work`; a `go.work` entry with no `go.mod`; an
  untracked `go.mod`; no `go.work`, listed and checked; an unknown
  argument; outside a repository): the same stdout, stderr and exit code
  on each, the usage line's own script name aside.
* **After D2,** the same nine inputs give the same stdout, stderr and exit
  code as `go-modules.sh` from `HEAD~1`: D2 changes only a successful run
  in which `go` writes to stderr.
* **The probe for D2:** the Python test against `go-modules.sh` from
  `HEAD~1` fails case 10's three checks that read the list and stderr:
  "the warning is not listed", "the warning reaches stderr" and
  "--check with a warning passes", "20 passed, 3 failed".
* **The empty module cache (rule 3, D1):** "23 passed, 0 failed", on
  macOS and on the Windows test host.
* **Mutations,** on scratch copies, each killed:
  * **P2-1** (`--check` ignores a `go.mod` that `go.work` omits): "a
    module missing from go.work fails --check: want 1, got 0", "21 passed,
    2 failed".
  * **P2-2** (the list reads `go`'s stderr again, appended to stdout):
    the same three checks as the probe, "20 passed, 3 failed". A first
    version of P2-2 left `r.stderr` empty and crashed the script; it was
    replaced, since a crash is not the fault it stands for.
* **macOS:** `py_compile`; `go-modules_test.py`, "23 passed, 0 failed";
  `go-fuzz_test.py`, "27 passed"; the shell tests still in shell,
  `go-precheck_test.sh` ("12 passed"), `go-apicheck_test.sh` ("19
  passed") and `go-examples_test.sh` ("19 passed"), each running
  `go-modules.py`; `make release-check`, "200 file(s) clean"; `make test`;
  `make lint`, `make fuzz FUZZTIME=2s` ("8 packages ran clean"),
  `shellcheck`, `markdownlint-cli2` and `actionlint`, before D2. All
  clean.
* **The Windows test host,** go1.27.2 windows/amd64, Python 3.14.7, with
  D2 and the index synced: `go-modules.py`'s list, as bytes, is `.` and
  `\n`; `make pre-add-check`, `make lint`, `make vuln`, `make examples`
  and `make test`, each exit 0, through `MODULES_CMD`; `go test -count=2
  -shuffle=on ./...`, exit 0; `py_compile`, exit 0;
  `go-modules_test.py`, "23 passed, 0 failed", warm and with an empty
  module cache; `go-fuzz_test.py`, "27 passed"; `go-precheck_test.sh`,
  "12 passed"; `go-apicheck_test.sh`, "19 passed"; `go-examples_test.sh`,
  "19 passed"; `make fuzz FUZZTIME=3s`, "8 packages ran clean".
  * A run on the tree before D2 was stopped when D2 began, at the owner's
    word, as it would have to run again; its processes on the host were
    ended with `taskkill`, and its files removed, before this run.

### Phase 3: `go-apicheck`

#### Deviations

* **D3 (2026-10-10): `go list -m`'s stderr was read into the module
  path.**
  * **Found,** reading `go-apicheck.sh` before the port: it runs
    `mod="$(cd "$m" && GOWORK=off "$GO" list -m 2>&1)"` and passes `$mod`
    to `apidiff` as the module path. On success, a line `go` wrote to
    stderr, such as a warning, would join the path, the fault D2 fixed in
    `go-modules`. Its other `2>&1` reads print their output only when the
    command fails, and are kept.
  * **The owner's choice:** fix it in this phase, as D2: the path is `go
    list -m`'s stdout; on success, what `go` wrote to stderr passes
    through; on failure, the message names `go`'s output as before.
  * **The other:** port it faithfully, and leave it to a later record.

#### What was built

* **`scripts/go-apicheck.py`,** executable, the standard library only:
  the base tag, `apidiff` at the `Makefile`'s pin or `APIDIFF`, the allow
  file's entries, unlisted and stale lines, its messages and exit codes,
  and its environment variables, as `go-apicheck.sh` had them. It reads
  `git archive`'s output with `tarfile`, where the shell piped it to
  `tar`, and lists the modules with `go-modules.py` through the Python
  running it. By D3, the module path is `go list -m`'s stdout alone. Its
  output's lines end in `\n` alone.
* **`scripts/go-apicheck_test.py`:** the shell test's 19 cases, by name,
  and case 11 (D3), three checks: through a `go` that writes a warning to
  stderr and succeeds, an unchanged API passes, against `v0.1.0`, and the
  warning reaches the output. Case 10's failing `apidiff`, a shell script
  in the shell test, and case 11's `go` are small Go programs the test
  builds, so they run natively on every host.
* **Callers:** `make apicheck` runs `$(PYTHON) ./scripts/go-apicheck.py`;
  `go-precheck.sh` runs it with `"${PYTHON:-python3}"`; CI's `apicheck`
  step runs the test with `python3`.
* **Docs:** `AGENTS.md`, `docs/architecture.md`, the comments of
  `go-precheck.sh`, and `scripts/apicheck.allow`'s header name the Python
  file. The header's one changed word is the file name: a close-out that
  compares the emptied file with `v0.8.0`'s now compares with this
  header.
* **Removed:** `scripts/go-apicheck.sh` and `scripts/go-apicheck_test.sh`.

#### Checks

* **The comparison (rule 3),** on the same tree:
  * the shell test, "19 passed, 0 failed"; the Python test has every shell
    case by name, and adds case 11's three;
  * the two scripts, on ten inputs, with stdout and stderr compared apart:
    the repository, with `apidiff` installed at the pin ("clean"); a
    module never tagged; an unchanged API; an unlisted removal; a listed
    one; a stale entry; a failing `apidiff`; the skip variable; no allow
    file; outside a repository. The same stdout, stderr and exit code on
    each.
* **The probe for D3:** the Python test against `go-apicheck.sh` fails
  case 11: "an unchanged API passes with go warning: want 0, got 2", the
  shell script reporting "exporting v0.1.0's API failed", since the
  warning had joined the module path; "20 passed, 2 failed".
* **The empty module cache:** "22 passed, 0 failed", on macOS and on the
  Windows test host.
* **Mutations,** on scratch copies, each killed:
  * **P3-1** (a stale entry is not reported): "a stale entry fails: want
    1, got 0" and "the stale entry is named", "20 passed, 2 failed".
  * **P3-2** (the module path reads `go`'s stderr again): the probe's two
    failures, "20 passed, 2 failed".
* **macOS:** `py_compile`; `go-apicheck_test.py` ("22 passed"),
  `go-modules_test.py` ("23 passed") and `go-fuzz_test.py` ("27
  passed"); `go-precheck_test.sh` ("12 passed") and `go-examples_test.sh`
  ("19 passed"); `make apicheck`, "against v0.10.0, 0 incompatible
  change(s)", "clean"; `make release-check`, "200 file(s) clean", which
  runs the Python script through `go-precheck.sh`; `shellcheck`,
  `markdownlint-cli2` and `actionlint`. All clean.
* **The Windows test host,** go1.27.2 windows/amd64, Python 3.14.7, with
  the index synced: `make pre-add-check`, `make lint`, `make vuln`, `make
  examples` and `make test`, each exit 0; `go test -count=2 -shuffle=on
  ./...`, exit 0; `py_compile`, exit 0; `go-apicheck_test.py`, "22
  passed, 0 failed", warm and with an empty module cache;
  `go-modules_test.py`, "23 passed"; `make apicheck`, "clean";
  `go-fuzz_test.py`, "27 passed"; `go-precheck_test.sh`, "12 passed";
  `go-examples_test.sh`, "19 passed"; `make fuzz FUZZTIME=3s`, "8
  packages ran clean".

### Phase 4: `go-examples`

#### Deviations

No deviation. Its `2>&1` reads print `go`'s output only when a command
fails, and `GOEXE` is read from stdout alone, so the fault D2 and D3 fixed
is not here. A region's name is matched as a regular expression, as the
shell's `awk` matched it; every name the guides use is a plain word, so
the match is the same as a literal one.

#### What was built

* **`scripts/go-examples.py`,** executable, the standard library only:
  the temporary module, the build, vet and cases, the guide check, and
  `--check-guide` and `--update`, with the messages, environment
  variables and exit codes of `go-examples.sh`. It copies the tree with
  `shutil`, reads the guides and regions in Python where the shell used
  `awk`, and prints a differing excerpt with `difflib`'s unified diff
  where the shell used `diff -u`. Its usage line and its update hint name
  `go-examples.py`, the one change to its messages. Its output's lines end
  in `\n` alone.
* **`scripts/go-examples_test.py`:** the shell test's 19 cases, by name.
  Case 7's hint is checked against the name of the script under test.
* **Callers:** `make examples` runs `$(PYTHON) ./scripts/go-examples.py`;
  `go-precheck.sh` runs it with `"${PYTHON:-python3}"`; CI's `examples`
  step runs the test with `python3`.
* **Docs:** `AGENTS.md` (the pre-add text, and only `go-precheck` left in
  shell), `docs/architecture.md`, the comments of `go-precheck.sh`,
  `testdata/frameworks/cases.txt`, and the package comments of the four
  framework examples, outside their guide regions, name the Python file.
* **Removed:** `scripts/go-examples.sh` and `scripts/go-examples_test.sh`.

#### Checks

* **The comparison (rule 3),** on the same tree:
  * the shell test, "19 passed, 0 failed"; the Python test has every shell
    case by name, against `go-examples.py` and against `go-examples.sh`
    (`EXAMPLES`), "19 passed, 0 failed";
  * the two scripts, on fourteen inputs, with stdout and stderr compared
    apart and the script's own name normalized: the repository, every step
    and `--check-guide`; a passing tree; a wrong exit; the other stream;
    a one-byte excerpt difference, its diff included; a missing region; a
    from line with no block; a broken build, with `--check-guide` and
    without; the skip switch; an unknown argument; no `go.mod.tmpl`; and
    `--update`, on two copies of one tree, whose written `go.mod.tmpl` and
    `go.sum` are equal. The same stdout, stderr and exit code on each.
* **The empty module cache:** "19 passed, 0 failed", on macOS and on the
  Windows test host.
* **Mutation P4-1** (an excerpt that differs from its source passes),
  killed: "a one-byte excerpt difference fails: want 1, got 0" and "the
  difference is shown", "17 passed, 2 failed".
* **macOS:** `py_compile`; `go-examples_test.py` ("19 passed"),
  `go-apicheck_test.py` ("22 passed"), `go-modules_test.py` ("23 passed")
  and `go-fuzz_test.py` ("27 passed"); `go-precheck_test.sh` ("12
  passed"); `make examples`, "10 excerpt(s) checked", "clean"; `make
  pre-add-check` on the four framework examples, "4 file(s) clean", with
  the examples gate; `make release-check`, "200 file(s) clean"; `make
  lint`, `shellcheck`, `markdownlint-cli2` and `actionlint`. All clean.
* **The Windows test host,** go1.27.2 windows/amd64, Python 3.14.7, with
  the index synced: `make pre-add-check`, `make lint`, `make vuln` and
  `make examples`, each exit 0, `make examples` "clean"; `py_compile`,
  exit 0; `go-examples_test.py`, "19 passed, 0 failed", warm and with an
  empty module cache; `go-apicheck_test.py`, "22 passed";
  `go-modules_test.py`, "23 passed"; `go-fuzz_test.py`, "27 passed";
  `go-precheck_test.sh`, "12 passed"; `make fuzz FUZZTIME=3s`, "8
  packages ran clean". The run left out `make test` and a second shuffled
  `go test`, which `make pre-add-check` already covers, as the owner and
  the agent agreed after Phase 3's run.

### Phase 5: `go-precheck`, and the shim

#### Deviations

* **D4 (2026-10-10): `go mod edit -print`'s stderr was read into the
  `go.mod` text.**
  * **Found,** reading `go-precheck.sh` before the port: its requirements
    step, for a module other than the root, runs `printed="$(GOWORK=off go
    mod edit -print 2>&1)"` and searches `printed` for `replace` and
    `require` lines. On success, a line `go` wrote to stderr joins the
    text: the fault D2 and D3 fixed, with less reach, since a warning
    misleads the check only if it begins with the word `replace` or
    `require` and a space. Its
    other `2>&1` reads are printed only when the command fails, or, for
    `govulncheck`, need stderr to tell a network failure, and are kept.
  * **The owner's choice:** fix it in this phase, as D2 and D3: the text
    is the command's stdout; on success, what `go` wrote to stderr passes
    through; on failure, the step shows `go`'s output as before.
  * **The other:** port it faithfully, and leave it to a later record.

#### What was built

* **`scripts/go-precheck.py`,** executable, the standard library only:
  the files, the modules and their owners, `gofmt` and its own failures,
  the three `golangci-lint` runs, `go vet`, `go test`, `go mod tidy
  -diff`, `govulncheck` and its network rule, workspace-mode tests, a
  nested module's requirements, `go-modules.py --check`, and the API diff
  gate and the examples, with the skip variables, messages and exit codes
  of `go-precheck.sh`. It finds `golangci-lint` as Git Bash's `-x` test
  did, with `.exe` on Windows. By D4, a nested module's `go.mod` text is
  `go mod edit -print`'s stdout alone. Its output's lines end in `\n`
  alone.
* **`scripts/go-precheck.sh`** is the shim: one `exec` of `python3` (or
  `PYTHON`) on `go-precheck.py` beside it, with `"$@"`. It holds no logic;
  `shellcheck` passes it.
* **`scripts/go-precheck_test.py`:** the shell test's twelve checks, by
  name, each run through the shim, as the machine-wide gate runs it; and:
  * case 7, three checks: through the shim and run directly, the same
    exit status and output, the file list reaching the check;
  * case 8 (D4), two checks: a nested module, through a `go` that writes
    `replace example.com/x => ./x` to stderr and succeeds, passes, and no
    replace directive is reported.

  Its `golangci-lint` stub and case 8's `go` are small Go programs it
  builds. It runs the shim without `BASH_ENV` and `ENV`: on the agent's
  host, `BASH_ENV` names a start-up file that puts the real `go`'s
  directory first on `PATH`, so case 8's `go` was never run, and the
  first probe against the shell script passed. With them removed, the
  probe fails as below.
* **Callers:** `make pre-add-check` and `make release-check` run
  `$(PYTHON) ./scripts/go-precheck.py`; CI's precheck step runs the test
  with `python3`; the machine-wide gate runs the shim, unchanged.
* **Docs:** `AGENTS.md` (the pre-add text, and the scripts section: every
  script is Python, and the shim is the one shell file),
  `docs/architecture.md` (the tree, with the shim, and the pre-add text),
  `docs/guides/releasing.md` and the `Makefile`'s comment name the Python
  file.
* **Removed:** `scripts/go-precheck_test.sh`. `scripts/` holds no other
  shell file than the shim.

#### Checks

* **The comparison (rule 3),** on the same tree, the shell pair from a
  scratch clone of `67597d4`:
  * the shell test there, "12 passed, 0 failed"; the Python test has every
    shell check by name, and adds cases 7 and 8;
  * the two scripts, on ten inputs, run as the gate runs them, with stdout
    and stderr compared apart: the repository with two files listed, and
    with none (every gate, the API diff gate and the examples among them,
    200 files); a clean tree; an unformatted file; a testdata file that
    does not parse; a tracked file deleted; a list with no Go file;
    `golangci-lint` missing (exit 2); a nested module with a replace
    directive; and one requiring the root at a pseudo-version. The same
    stdout, stderr and exit code on each.
* **The probe for D4:** the Python test against `go-precheck.sh` fails
  case 8: "a nested module passes through a go that warns: want 0, got
  1", the shell reporting "go.mod (sub): a replace directive …" with the
  warning's line; "12 passed, 2 failed" (case 7 does not apply to it).
* **The machine-wide gate,** `~/.agents/hooks/lib/precommit-checks.sh`,
  on a scratch copy of the tree: with `when/error.go` staged unformatted,
  exit 1, "Go pre-commit check failed (scripts/go-precheck.sh)", "gofmt:
  these files are not formatted", naming the file; with a formatted change
  to it staged, exit 0.
* **The empty module cache:** "17 passed, 0 failed", on macOS and on the
  Windows test host.
* **Mutations,** on scratch copies, each killed:
  * **P5-1** (the shim drops its arguments): case 3's three checks and
    "the shim's output is the Python file's", "13 passed, 4 failed".
  * **P5-2** (the `go.mod` text reads `go`'s stderr again): the probe's
    two failures, "15 passed, 2 failed".
* **macOS:** `py_compile`; the five tests, `go-precheck_test.py` ("17
  passed"), `go-examples_test.py` ("19"), `go-apicheck_test.py` ("22"),
  `go-modules_test.py` ("23") and `go-fuzz_test.py` ("27"); `make
  pre-add-check FILES=when/error.go`, "1 file(s) clean"; `make
  release-check`, "200 file(s) clean"; `make lint`, `shellcheck` (the
  shim), `markdownlint-cli2` and `actionlint`. All clean.
* **The Windows test host,** go1.27.2 windows/amd64, Python 3.14.7, with
  the index synced, trimmed as in Phase 4: `make pre-add-check`, `make
  lint`, `make vuln` and `make examples`, each exit 0;
  `py_compile`, exit 0; `go-precheck_test.py`, "17 passed, 0 failed",
  warm and with an empty module cache, through the shim under Git Bash
  (after the fix below);
  `go-examples_test.py`, "19 passed"; `go-apicheck_test.py`, "22
  passed"; `go-modules_test.py`, "23 passed"; `go-fuzz_test.py`, "27
  passed"; the shim on one file, exit 0, "1 file(s) clean"; `make fuzz
  FUZZTIME=3s`, "8 packages ran clean".
  * **The first run failed `go-precheck_test.py`,** "2 passed, 15
    failed": every run through the shim gave exit 2, "go-precheck: gofmt
    not found in PATH", while the Python file run directly passed. The
    test ran `bash` from a native Windows Python, which found WSL's
    `bash.exe` before Git's: the shim ran in WSL's Linux, whose `python3`
    has no `gofmt` (its `PATH` read `/mnt/c/...`). The machine-wide gate is
    a Git Bash script, and runs the shim with Git's bash. The test now
    finds Git's `bash.exe` above `git --exec-path` on Windows, and `bash`
    elsewhere; on the rerun, "17 passed, 0 failed", warm and with an empty
    module cache. The mutations ran again after the change, with the same
    results.
  * A stray command of the agent's copied the tree to the host under a
    label `x` and failed at its second copy, leaving `tree-x.tgz` in the
    home directory; the agent removed it.
