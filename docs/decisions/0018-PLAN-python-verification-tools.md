---
status: in-progress
date: 2026-10-12
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
