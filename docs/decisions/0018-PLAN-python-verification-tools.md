---
status: in-progress
date: 2026-10-14
associated-madr: "0018-MADR-python-verification-tools.md"
---
# Implement the Python verification tools

Associated MADR: [0018-MADR-python-verification-tools.md](0018-MADR-python-verification-tools.md)

## Goal

Build `tools/verify`, so that every gate, mutation proof, remote-host run
and identifier scan a PLAN step needs runs from the repository, in Python,
with one gate definition on every host, and no host or account named in
any committed file.

Done means every item under Verification holds, and
[0008-PLAN-command-palette.md](0008-PLAN-command-palette.md) Step 3's
gates have run through it on all three hosts.

## Scope

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0018-*`, `docs/README.md`, `AGENTS.md` | accept the records; AGENTS.md's "Tools" section |
| 2 | `tools/__init__.py`, `tools/verify/{__init__,__main__,config,proc,tree,gates}.py`, `tools/verify/verify.example.json`, `tools/verify/tests/` | the package, the config, the runner, the tree listing, `gates` |
| 3 | `tools/verify/mutate.py`, its tests | `mutate` |
| 4 | `tools/verify/remote.py`, its tests | `remote`, for a `posix` and a `windows` host |
| 5 | `tools/verify/scan.py`, `tools/verify/names.py`, their tests | `scan` and `names` |
| 6 | `.github/workflows/ci.yml`, `docs/` | the checks in CI; the docs; the close-out |
| 7 (A2) | `tools/verify/gates.py`, its tests, `docs/decisions/0008-PLAN-command-palette.md` | flushed and timed output; the default list without repeats |

The standard library only; nothing is added to `go.mod`, and no Python
package is installed.

### Out of scope

* `scripts/` and its callers: `make`, CI's existing steps, the commit
  gate.
* A Python linter beyond `py_compile`, which needs its own record (0017
  item 6).
* The scratch probes that test an upstream library once, such as the
  Bubble Tea ordering probe and the x/ansi `Cut` probe. They stay probes.
* Any Go code.
* `git push` and tags, which are the owner's.

## Prerequisites

* The owner has approved this pair.
* [0008-PLAN-command-palette.md](0008-PLAN-command-palette.md) is paused
  at Step 3, whose code and tests pass and whose gates are not run yet. It
  resumes with Step 2 of this PLAN's `gates`, and Step 4's `remote`.
* The local config exists on the owner's machine before Step 4 runs on a
  host. The agent writes it there, outside the repository, from the host
  names the owner has given, and never commits it.

## Rules for every step

1. **No shell scripts.** Every file is Python, run with `python3`. Shell
   is used for one command at a time, at most with one pipe. Searching and
   editing use the native tools, not `grep`, `sed`, `awk` or `cut`.
2. **0017's conventions:** Python 3.12 or later; the standard library
   only; `subprocess` with argument lists, never `shell=True`; no `eval`.
3. **Tests:** `python3 -m unittest discover -s tools -t .` passes; every
   test module names its cases; a test never touches a real host, using
   fakes for `ssh` and `scp`.
4. **Mutation proofs.** Each step names mutations of its key invariants,
   each applied to a scratch copy, each making a named test fail. Until
   Step 3 builds `mutate`, they run through the scratch `mutate.py`, which
   is Python.
5. **Identifiers.** No file under `tools/` names a host, an account or a
   real path. The identifier scan runs before each handoff: the scratch
   pipeline until Step 5, then `scan`.
6. **Other checks per step:** `markdownlint-cli2`, `actionlint` (Step 6),
   `python3 -m py_compile` over `tools/`, and the existing
   `scripts/*_test.py`.
7. **Commit.** One commit per step, by the owner, after the agent stops
   with the step's evidence in the execution record.

## Implementation Steps

### Step 1: records

* The owner's approval goes into the MADR, which becomes `accepted`; this
  PLAN becomes `in-progress`; `docs/README.md` gains both rows.
* AGENTS.md gains "Tools": `tools/verify` and its commands; that it is
  run by a person or an agent, never by `make`, CI or the commit gate;
  where the local config lives, and that it is never committed.
* [0008-PLAN-command-palette.md](0008-PLAN-command-palette.md) gains a
  dated note in its Step 3 record: the gates wait for this PLAN's Steps 2
  and 4.
* Records and AGENTS.md only.

### Step 2: the package, the runner and `gates`

* `config.py`: the config's path (`--config`, else
  `$XDG_CONFIG_HOME/go-tui-lib/verify.json`, else
  `~/.config/go-tui-lib/verify.json`); a dataclass per host (`name`,
  `kind`, `ssh`, `path_prepend`, `python`) and the identifier patterns,
  each compiled; a clear error for a missing or malformed file.
* `proc.py`: `run(args, *, cwd, env, out) -> Result`, with the output to a
  file and the exit status kept before anything reads the output.
* `tree.py`: the files `git ls-files --cached --others --exclude-standard`
  lists that exist, so a deleted file is not copied; a copy of them to a
  directory; a tar of them.
* `gates.py`: `Gate(name, args, env, needs_history, quote)`; the default
  list, the scratch `gates.sh`'s in its order, with CI's other checks
  (`shellcheck scripts/*.sh`, `markdownlint-cli2`, `actionlint`, the
  `scripts/*_test.py`) and the `tools` tests; `--only` and `--skip` by
  name; `--no-history` skips the gates that need git history and says so.
* `python3 -m tools.verify gates --label L FILE…` prints `name: exit=N`
  per gate and its quoted lines, keeps each output in a directory under
  the scratch root it is given (`--out`), and exits non-zero when any gate
  failed.
* **Tests** (`tools/verify/tests/`):
  * `tree`: in a throwaway repository, a tracked file deleted and not
    staged is not listed, an ignored file is not, an untracked one is;
  * `proc`: the exit status of a failing command is kept, and its output
    is in the file;
  * `gates`: with a fake gate list of `python3 -c` commands, one failing,
    the runner reports each by name, exits non-zero, and runs the rest
    after the failure; `--no-history` skips and names the history gates;
  * `config`: a missing file, a malformed one, an unknown host kind.
* **Acceptance:** `gates` on 0008 Step 3's files, on macOS: each gate's
  exit status is recorded.
* **Mutations:**
  * the runner stops at the first failing gate;
  * `tree` lists a deleted file;
  * `--no-history` runs `make apicheck` anyway.

### Step 3: `mutate`

* `python3 -m tools.verify mutate FILE.json`: the scratch `mutate.py`'s
  contract (each mutation's `name`, `file`, `old`, `new`, `test`, and
  optional `extra` and `env`), each applied once to a fresh copy from
  `tree`, the anchor asserted to occur exactly once, the named `go test`
  run, and `KILLED` or `SURVIVED` printed with the failing lines.
* **Tests:** a throwaway Go module with one test: a mutation that breaks
  it is killed; one that leaves it passing survives and the command exits
  non-zero; an anchor found twice, or not at all, is an error naming it.
* **Mutations:** an anchor found twice is applied anyway; a survivor exits
  zero.

### Step 4: `remote`

* `python3 -m tools.verify remote HOST [--label L] -- GATES-ARGS…`:
  * `posix` host: a tar of the tree and, for history, a git bundle of
    `main` and its tags, copied with `scp`; through `ssh`, with an
    argument list: `python3 -c` with a short, fixed bootstrap from
    `remote.py` that makes a temporary directory, extracts the tar, makes
    the repository from the bundle and the copied files, runs `python3 -m
    tools.verify gates …` there, and removes the directory, whatever
    happened;
  * `windows` host: the same, under Git for Windows' `python3`, with the
    `PATH` the config gives;
  * the output redacted: home paths and the config's identifier patterns
    become placeholders.
* **Tests,** with fake `ssh` and `scp` (Python programs put first on
  `PATH` that record their argument lists and run the bootstrap locally):
  * the bootstrap removes its directory after a failing gate;
  * no argument list passes a shell string;
  * redaction replaces a home path and a planted host name;
  * a host missing from the config is an error naming the config's path.
* **Acceptance:** 0008 Step 3's gates through `remote` on the Linux and
  the Windows host.
* **Mutations:** the bootstrap leaves its directory on failure; redaction
  misses a Windows-style home path.

### Step 5: `scan` and `names`

* `scan`: the diff against `HEAD`, staged and not, and the untracked
  files; each line matching a pattern reported as `file:line`, never the
  match; exit 1 on any.
* `names`: the scratch `names.py`, with the planned names from a JSON
  file the record's author writes, and the shared names read from
  `internal/conformance`'s `sharedNames`.
* **Tests:** a throwaway repository with a planted identifier in a staged
  file, an unstaged one and an untracked one, each found, and the output
  never holds the identifier; `names` on a planted package.
* **Mutations:** `scan` skips untracked files; `scan` prints the match.

### Step 6: CI, docs and close-out

* CI's lint step runs `python3 -m py_compile` over `tools/` and `python3
  -m unittest discover -s tools -t .`.
* `docs/README.md` rows; `docs/architecture.md` notes `tools/` beside
  `scripts/`.
* This PLAN becomes `complete` after CI is green on the pushed tree.

### Step 7 (A2): live output, timed gates, no repeats

*Added 2026-10-14, for MADR amendment A2, which the owner approved the
same day ("approved"); this PLAN is `in-progress` until Step 7 is done.*

* **`gates.py`:**
  * every line the runner emits is flushed;
  * each gate is timed with `time.monotonic`, and reported as `name:
    exit=N (Ns)`; the summary ends with the total;
  * the default list as A2 item 3 gives it, `modernize` (`make
    modernize`) in place of `lint`; the removed gates stay defined, and
    `--only` runs them;
  * `Gate` gains `without_history`, the environment a gate runs with when
    there is no history, in place of being skipped; `release-check`'s is
    `{"GO_PRECHECK_SKIP_APICHECK": "1"}`.
* **Tests:**
  * a gate's line carries its time, and the summary the total;
  * the runner's lines reach a pipe as they are emitted, not at exit: a
    child process runs two gates, the first printing at once and the
    second sleeping, and the parent reads the first gate's line before
    the second ends;
  * the default list holds each A2 item 3 gate and none of the removed
    ones; every removed gate is still selectable with `--only`;
  * with `--no-history`, `release-check` runs with
    `GO_PRECHECK_SKIP_APICHECK=1`, and a gate with no `without_history`
    is still skipped.
* **Mutations:** the flush is removed; the time is left out; `lint` is
  back in the default list; `release-check` is skipped without history.
* **0008-PLAN, Rule 3,** gains a dated note: its checks are met by the
  default list, where `release-check` covers `pre-add-check`, `make
  lint`'s golangci-lint, `go mod tidy -diff`, workspace mode, the modules
  check, `make apicheck` and `make examples`, and `modernize` covers the
  rest of `make lint`.
* **Acceptance:** 0008 Step 5's gates run with the new list on all three
  hosts, each gate timed, and the Windows run's lines arrive as its gates
  end; the record gives each host's total and its slowest gates.
* **Checks:** `tools-tests`, `markdownlint` and `scan`.

## Verification

* `python3 -m unittest discover -s tools -t .` passes on macOS, the Linux
  host and the Windows host.
* Every step's mutations are killed.
* `gates`, locally and through `remote`, has run 0008 Step 3's gates on
  all three hosts.
* `scan` reports nothing in the repository's diff, and no file under
  `tools/` names a host, an account or a real path.
* CI is green on the pushed tree, with the new checks.

## Rollout and Rollback

* **Rollout.** The owner commits and pushes each step. From Step 2, the
  agent's local gates go through `tools/verify`; from Step 4, its remote
  ones.
* **Rollback.** `tools/` is called by nothing in `make`, CI or the commit
  gate until Step 6, so a step is reverted by its own commit. After Step 6,
  removing the CI lines removes the checks.

## Execution Record

### Step 1: records

*2026-10-12.* The owner approved the pair ("approved to proceed").

* **The MADR** is `accepted`, with the approval in its outcome; **this
  PLAN** is `in-progress`; `docs/README.md`'s rows say so.
* **AGENTS.md** gains "Tools": `tools/verify`, its commands, that nothing
  in `make`, CI or the commit gate runs it, and that its config lives
  outside the repository and is never committed.
* **[0008-PLAN-command-palette.md](0008-PLAN-command-palette.md)** gains a
  Step 3 entry: its gates wait for this PLAN's Steps 2 and 4.
* **Records and AGENTS.md only.** The tree also holds 0008 Step 3's
  unfinished work (`fuzzy/`, `internal/conformance/enums_test.go`, its
  record and the glossary), which is not this step's and is committed
  apart.

### Step 2: the package, the runner and `gates`

*2026-10-12.* The owner committed Step 1 as `07b01e5` ("committed,
proceed"), and 0008 Step 3's code as `79de568` before it.

* **Files:** `tools/__init__.py`; `tools/verify/__init__.py`,
  `__main__.py`, `config.py`, `proc.py`, `tree.py`, `gates.py` and
  `verify.example.json`; `tools/verify/tests/__init__.py`, `support.py`,
  `test_config.py`, `test_proc.py`, `test_tree.py` and `test_gates.py`.
* **`gates`:** 25 gates, in the scratch `gates.sh`'s order, then CI's other
  checks, the five `scripts/*_test.py` and the tools' own tests
  (`python3 -m tools.verify gates --list`). `--only`, `--skip`,
  `--no-history`, `--label`, `--out`; one `name: exit=N` line per gate,
  its quoted lines, or its last 15 lines when it fails; a summary line;
  exit 1 when any failed, 2 for an unknown gate name.
* **Correction to the MADR (item 2), recorded there:** `make examples`
  needs a git root, not tags, so only `apicheck` and `release-check` are
  marked `needs_history`; `test_history` pins it.
* **Tests:** `python3 -m unittest discover -s tools -t .`: "Ran 25 tests",
  OK. `python3 -m compileall -q tools`: 0.
* **Mutations,** through the scratch `mutate.py`, which gained a `cmd` key
  for a non-Go command (Python, scratch only, until Step 3):

  | Mutation | Killed by |
  | :--- | :--- |
  | the runner stops at the first failing gate | `test_a_failure_does_not_stop_the_run`: "Lists differ: [0, 3] != [0, 3, 0]" |
  | `tree` lists a deleted file | `test_tracked_untracked_ignored_and_deleted`: "['.gitignore', 'gone.go', 'kept.go', 'sub/new.py'] != ['.gitignore', 'kept.go', 'sub/new.py']" |
  | `--no-history` runs the history gates anyway | `test_history_gates_are_skipped_without_history`: "0 is not None" |
  | `apicheck` is not marked as needing history | `test_history`: "Items in the second set but not the first" |

* **Acceptance:** `python3 -m tools.verify gates --label p8s3` over 0008
  Step 3's 14 files, on macOS: "gates: 25 passed, 0 failed, 0 skipped",
  with `pre-add-check` "14 file(s) clean", `apicheck` "against v0.12.0, 0
  incompatible change(s)", `examples` clean, and `release-check` "240
  file(s) clean".
  * **Against `gates.sh`:** the MADR's Confirmation asks for the same exit
    statuses as the scratch `gates.sh` on the same tree. That comparison
    is not available: `gates.sh` last ran on Step 3's code before its lint
    fixes (`pre-add-check`, `lint` and `release-check` failed there, every
    other gate 0), and it is not run again. Every gate `gates.sh` ran is
    in this list, under the same command.
* **Identifier scan** of `tools/`, as one `git grep --untracked`: nothing.

### Step 3: `mutate`

*2026-10-12.* The owner committed Step 2 as `babf00c` ("committed,
proceed").

* **Files:** `tools/verify/mutate.py`; `__main__.py` gains `mutate`;
  `tools/verify/tests/test_mutate.py`.
* **`mutate FILE.json [--work DIR] [--keep]`:** the scratch `mutate.py`'s
  format, with its `cmd` key: `name`, `file`, `old`, `new`, then `test`
  (after `go test -count=1`) or `cmd`, and optional `extra` and `env`. Each
  mutation gets a fresh copy from `tree.copy`; every anchor must occur
  exactly once, or the mutation is an `ERROR` naming the file and the
  count; `KILLED` or `SURVIVED` with the exit status and up to 8 failing
  lines; a summary line; exit 0 only when every mutation is killed, 2 for
  a file that does not parse.
* **Tests:** "Ran 33 tests", OK, among them a real `go test` in a
  throwaway module (skipped where `go` is missing), and an `env` case that
  survives only if the variable reaches the test.
* **Mutations,** through `tools.verify mutate` itself, each on a copy:

  | Mutation | Killed by |
  | :--- | :--- |
  | an anchor found twice is applied anyway | `test_killed_survived_and_errors`: "['KILLED', 'SURVIVED', 'SURVIVED', 'ERROR'] != ['KILLED', 'SURVIVED', 'ERROR', 'ERROR']" |
  | a survivor exits zero | `test_exit_status`: "0 != 1" |
  | the verdict is inverted | `test_exit_status`, `test_a_go_test`, `test_env_reaches_the_test`, `test_extra_edits` |

* **Acceptance:** 0008 Step 3's nine mutations, on its committed code
  (`79de568`), one anchor moved with its lint fixes: "mutate: 9 killed, 0
  survived, 0 errors" (0008-PLAN's Step 3 entry quotes each).
* **Checks** through `tools.verify gates --only`: `tools-tests`,
  `markdownlint` and the five `script-test-*`, "7 passed, 0 failed";
  `compileall` 0; the identifier scan of `tools/`: nothing.

### Step 4: `remote`

*2026-10-12.* The owner committed Step 3 as `a3a654e` ("committed,
proceed.").

* **Files:** `tools/verify/remote.py`, `tools/verify/_bootstrap.py`;
  `__main__.py` gains `remote`, and splits its arguments at `--` before
  argparse (an `argparse.REMAINDER` positional swallowed `--config`, which
  a test caught); `proc.py` gains `resolve`; tests
  `tools/verify/tests/test_remote.py`, `tests/fakes/{__init__,ssh,scp}.py`,
  and a case in `test_proc.py`.
* **`remote HOST [--label L] [--config PATH] -- GATES-ARGS…`:** a tar of
  the tree and a git bundle of `HEAD` and the tags, uploaded with `scp`;
  then `ssh HOST python3 -` with `_bootstrap.py`'s source and its
  parameters on stdin, so the remote command is two fixed words and no
  shell string carries anything. The bootstrap unpacks into a new
  temporary directory, makes the repository from the bundle with the
  copied files as its index, runs `python3 -m tools.verify gates` there,
  and removes the directory, the outputs and the uploads whatever
  happened. Output comes back line by line, home paths and the config's
  identifiers redacted, ssh's own notices dropped.
* **The local config,** written outside the repository from the host
  names the owner gave: two hosts, `posix` and `windows`, each with an
  empty `path_prepend`, since a plain `ssh` session there already has
  `python3` 3.14.7, Go, `make`, golangci-lint and govulncheck on its
  `PATH` (a one-off probe, fed to each host's `python3` on stdin).
* **Tests:** "Ran 44 tests", OK. The fakes are Python programs run by
  `sys.executable`, so the tests run on Windows too: a round trip to a
  fake host lists the gates; after a passing and a failing run the fake
  home and temporary directory are empty; the `ssh` call is exactly
  `-o BatchMode=yes -T far-host python3 -`; the bootstrap's copy has
  `HEAD`, the tags and the untracked file in its index; a corrupt bundle
  leaves nothing behind; redaction of POSIX, macOS and both Windows path
  styles, identifiers and noise.
* **Mutations,** through `tools.verify mutate`, each killed:

  | Mutation | Killed by |
  | :--- | :--- |
  | the bootstrap leaves its work directory | `test_nothing_is_left_on_the_host`, for a passing and a failing run |
  | redaction misses a Windows-style home path | `test_homes_identifiers_and_noise`: `'in C:\\Users\\carol\\…' != 'in C:\\Users\\<user>\\…'` |
  | a label is not checked | `test_a_bad_label`: "RemoteError not raised" |
  | `tree.copy` drops the file modes (D3) | `test_copy_keeps_paths_and_modes`: "420 != 493" |

* **Acceptance:** 0008 Step 3's gates through `remote`, over its 14 files,
  with nothing skipped: on the Linux host "gates: 25 passed, 0 failed, 0
  skipped", and on the Windows host the same; each with `apicheck`
  "against v0.12.0, 0 incompatible change(s)", `examples` clean and
  `release-check` "240 file(s) clean". The output named no account: the
  work directories came back as `C:\Users\<user>\…`.
* **Checks** through `tools.verify gates --only`: `tools-tests`,
  `markdownlint`, `shellcheck` and the five `script-test-*`, "8 passed, 0
  failed"; `compileall` 0. The identifier scan of `tools/`, the docs and
  AGENTS.md for the account, host and organisation names: nothing. A scan
  for home-path shapes matches only the redaction test's made-up names,
  which are there to be redacted.

#### Deviations

* **D1 (2026-10-12): a stray `.git` on the Linux host.** The first Linux
  run failed 7 gates (`examples`, `release-check` and five
  `script-test-*`) with "error obtaining VCS status: exit status 128":
  every `go build` in a temporary directory found an empty
  `/data/cache/tmp/.git` (created 2026-08-25, not by this work), and git
  failed there. The Windows host passed the same gates. *Resolution, the
  owner's:* the agent removed it with `rmdir`, which removes only an empty
  directory. The rerun passed every gate.
* **D2 (2026-10-12): gate tools missing on the hosts.** `actionlint` was
  on neither host, and `markdownlint-cli2` not on the Linux host ("npx
  canceled due to missing packages"). *Resolution, the owner's:* install
  them, over a skip list or a per-run `--skip`. The agent installed
  `actionlint` v1.7.12, CI's version, on both with `go install`, and
  `markdownlint-cli2@0.23.3` on the Linux host with `npm install -g`,
  the macOS host's version; CI pins 0.23.2. The first acceptance runs had
  skipped `actionlint`; the reruns skipped nothing.
* **D3 (2026-10-12): a test of Step 2's that held on POSIX only.** On
  Windows, `test_copy_keeps_paths_and_modes` failed ("0 is not true"): it
  asserted an executable bit, which NTFS has not got. It now asserts the
  copy's permission bits equal the source's, which keeps the executable
  bit on POSIX and holds on Windows; the mutation above shows it still
  fails when modes are dropped.
* **D4 (2026-10-12): commands are resolved on their own `PATH`.** On
  Windows, `npx` is `npx.CMD`, which a bare name in an argument list does
  not reach. `proc.run` resolves the first argument with `shutil.which`
  against the environment the gate runs with; a test pins it. The MADR's
  A1 records this and the transport's shape.

### Step 5: `scan` and `names`

*2026-10-12.* The owner committed Step 4 as `ee43407` ("committed,
proceed.").

* **Files:** `tools/verify/scan.py`, `tools/verify/names.py`;
  `__main__.py` gains both; `tests/test_scan.py`, `tests/test_names.py`;
  `tests/support.py` gains `commit` (no hooks, no signing, a throwaway
  identity), which `test_remote.py` now uses.
* **`scan [--base REV] [--config PATH]`:** every line the work tree adds
  against `--base` (default `HEAD`; the empty tree before a first commit),
  staged or not, from `git diff --unified=0`, and every line of each
  untracked, unignored text file. A hit prints as `file:line: matches
  identifier pattern N`, never the text; exit 1 on any hit, 0 when clean,
  2 with no patterns or a bad base. `--base origin/main` scans what a push
  would send.
* **`names FILE.json`:** a JSON object of package directory to planned
  type names; each is checked against the exported type names of every
  public package (tests, `internal`, `testdata` and hidden directories
  left out, as the conformance scan does), and allowed only when
  `sharedNames`, read from `internal/conformance`, names it with `*` or
  with exactly the packages that would share it.
* **Tests:** "Ran 58 tests", OK: a planted identifier in a staged, an
  unstaged and an untracked file, each found and never printed; an
  unchanged committed line and an ignored file not reported; `--base`
  reaching back past a commit; binary files skipped; a repository with no
  commit; for `names`, grouped and generic declarations, lower-case,
  internal, test and `testdata` types left out, the three ways
  `sharedNames` allows a name, and the repository's own `sharedNames`
  parsed.
* **Mutations,** through `tools.verify mutate`, each killed:

  | Mutation | Killed by |
  | :--- | :--- |
  | `scan` skips untracked files | `test_staged_unstaged_and_untracked` and `test_exit_status_and_config` ("0 != 1") |
  | `scan` prints the match | `test_staged_unstaged_and_untracked`, `test_a_base_reaches_back`, `test_everything_is_new` |
  | `names` counts internal packages | `test_public_exported_types`: "'Hidden' unexpectedly found" |
  | `names` ignores `sharedNames` | `test_clashes_and_allowances` |

* **Acceptance:** `scan` on this step's change: "scan: clean (5
  pattern(s))". `names` on 0008's planned `palette` types, with
  `RegistryOption` added as a known positive: "names:
  palette.RegistryOption: also in command", the one clash, which MADR A3
  item 7 had already steered `palette` away from; the 22 others clean.
* **Checks** through `tools.verify gates --only`: `tools-tests`,
  `markdownlint`, `shellcheck` and the five `script-test-*`, "8 passed, 0
  failed"; `compileall` 0. From this step on, the identifier scan is
  `python3 -m tools.verify scan`.

### Step 6: CI, docs and close-out

*2026-10-12.* The owner committed Step 5 as `5b08f32` ("committed,
proceed").

* **CI:** the `gates` job gains a `verification tools` step, after the
  scripts' checks: `python3 --version`, `python3 -m compileall -q tools`
  (the recursive form of `py_compile`), and `python3 -m unittest discover
  -s tools -t .`. Its runner's `python3` is Ubuntu 24.04's 3.12, the floor
  0017 set.
* **Seen to fail,** on copies through `tools.verify mutate`: a syntax
  error in `tools/verify/scan.py` fails the `compileall` line ("SyntaxError:
  expected ':'"); a runner that runs a history gate under `--no-history`
  fails the `unittest` line ("0 is not None").
* **Python 3.12,** locally, before CI: `compileall` 0 and "Ran 58 tests",
  OK.
* **Docs:** `docs/architecture.md`'s tree gains `tools/verify/`, and its
  CI list the step; `docs/README.md`'s "I want to…" table gains "verify a
  PLAN step", and 0018-MADR's row reads "accepted; A1 accepted; built".
* **Checks** through `tools.verify gates --only`: `tools-tests`,
  `markdownlint`, `actionlint`, `shellcheck` and the five `script-test-*`,
  "9 passed, 0 failed"; `tools.verify scan`: "clean (5 pattern(s))".
* **Close-out:** this PLAN becomes `complete` once CI is green on the
  pushed tree, which is the owner's to push.

#### The push, CI and the close-out

* The owner committed this step as `c0d1488`. Before the push, the agent
  ran the disclosure guard over the nine outgoing commits,
  `d9930f8..c0d1488`: exit 0, no output; and `python3 -m tools.verify scan
  --base origin/main`: "clean (5 pattern(s))".
* The owner pushed `main`. CI run `38289280504` on `c0d1488` concluded
  `success`, every job: `modules`, `gates`, and `test` on
  `ubuntu-24.04`, `macos-15` and `windows-2025`. In `gates`, the
  `verification tools` step printed "Python 3.12.3", compiled `tools/`,
  and "Ran 58 tests", OK.
* Every item under Verification holds: the tests pass on macOS, the Linux
  host (through `remote`, Step 4) and the Windows host, and on CI's 3.12;
  every step's mutations are killed; 0008 Step 3's gates ran through the
  tools on all three hosts; `scan` is clean; CI is green.
* **This PLAN is `complete`,** and its `docs/README.md` row says so.

### Step 7 (A2): live output, timed gates, no repeats

*2026-10-15.* The owner committed the amendment as `e6ee706` and approved
it ("approved", then "proceed").

* **`proc.say`** prints each line with `flush=True`; `gates.run`,
  `mutate.run` and `remote.run` emit through it by default, and the
  `gates` command's header line uses it.
* **`gates.py`:** each gate is timed with `time.monotonic` and reported as
  `name: exit=N (Ns)`; the summary ends `in Ts`. `Gate` gains `default`
  and `without_history`. `select` takes the default gates when `--only`
  is empty. The default list is A2 item 3's; `pre-add-check`, `lint`,
  `workspace-mode`, `tidy`, `modules`, `apicheck` and `examples` are
  defined with `default=False`, and `--list` marks them "(only when
  named)". `release-check`'s `without_history` is
  `{"GO_PRECHECK_SKIP_APICHECK": "1"}`; `--no-history`'s help says so.
* **Tests** (`tools/verify/tests/test_gates.py`, 58 to 62 tests):
  `test_each_gate_is_timed`, `test_without_history_a_gate_may_run_differently`,
  `TestLiveOutput`, and `TestDefaultGates.test_the_default_list`;
  `test_select` and `test_history` cover `default` and
  `without_history`; the existing line checks take the timed format.
  A gate's time is matched as `(\d+s)` except in the timed test, which
  compares it with the measured seconds, so a loaded host cannot round a
  quick gate to `(1s)`.
* **Found while proving the live-output test.** Its first form read the
  first line and then asked `poll()` whether the child still ran. With
  the flush removed the mutation survived: a child that flushes at exit
  is still seen as running between the flush and its exit. A probe on a
  scratch script measured the wait from the first line to EOF instead:
  0.00s without the flush, 2.01s with it, against a two-second sleep. The
  test now asserts that wait is over 1.5s against a three-second gate.
* **Mutations** through `tools.verify mutate`, after the change above,
  7 killed, 0 survived:
  * the flush removed: "0.005475166020914912 not greater than 1.5 : the
    first line came only when the run ended";
  * the time left out of the line: "Regex didn't match: '^broken: exit=3
    \\(\\d+s\\)$' not found in 'broken: exit=3'", and three more;
  * the time not measured: "0.0 not greater than or equal to 1.2";
  * `lint` back in the default list, and `select` ignoring `default`:
    `test_the_default_list`, "Items in the first set but not the
    second"; `test_select`, "['a', 'b', 'c'] != ['a', 'b']";
  * `release-check` skipped without history: "None !=
    {'GO_PRECHECK_SKIP_APICHECK': '1'}";
  * the runner ignoring `without_history`: "5 != 0".
* **0008-PLAN, Rule 3,** has the dated note.
* **Acceptance.** 0008 Step 5 has no code yet, so its gates ran on this
  step's tree: the default list Step 5 will run, with this step's files,
  none of them Go, so `gofmt` is skipped. Each host: "18 passed, 0
  failed, 1 skipped".

  | Host | Total | Slowest gates |
  | :--- | ---: | :--- |
  | macOS | 310s | race 84s, shuffle 69s, release-check 65s, lc-all-c 35s |
  | Linux | 860s | race 310s, shuffle 179s, release-check 153s, lc-all-c 81s |
  | Windows | 1618s | release-check 1131s, race 171s, shuffle 156s, lc-all-c 75s |

  macOS's 310s compares with 386s for 0008 Step 4's run under the old
  list. On Windows the lines arrived as the gates ended: read mid-run,
  the log held the first five gates' lines while `race` ran.
  `release-check` took 70% of the Windows run; which of its parts is
  slow there was not measured, as the gate's output stays on the host.
* **Checks:** the remote runs uploaded the tests before their time
  assertions were loosened, so `tools-tests` ran again on both hosts:
  "Ran 62 tests", OK, on Linux (9s) and Windows (19s). Locally, through
  `tools.verify gates --only`, `markdownlint` and `tools-tests`: "2
  passed, 0 failed"; under Python 3.12, CI's floor, "Ran 62 tests", OK;
  `tools.verify scan`: "clean (5 pattern(s))".
* **Close-out:** this PLAN becomes `complete` once CI is green on the
  pushed tree, which is the owner's to push.
