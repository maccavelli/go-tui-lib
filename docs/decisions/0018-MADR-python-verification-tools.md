---
status: accepted
date: 2026-10-12
decision-makers: the repository owner
consulted: the agent that executes the PLANs
informed: contributors, through AGENTS.md
---
# Keep the agent's verification tooling in the repository, as one modular Python package under `tools/`

## Context and Problem Statement

On 2026-10-12, during
[0008-PLAN-command-palette.md](0008-PLAN-command-palette.md) Step 3, the
owner said:

> why are you continuing to write bash scripts. that is specifically
> prohibited by global agent rules. 1. no bash scripting. 2. use native
> tools when possible, NOT bash grep/sed/awk/cut, write out all tooling in
> python and save to the appropriate tools/ subdir in the repo. for any
> re-usable scripts written, re-write them all in idiomatic, modular
> python, and ensure they are in the appropriate dir in the repo.

Since [0017-MADR-python-repository-scripts.md](0017-MADR-python-repository-scripts.md),
the scripts that `make`, CI and the commit gate run are Python. The
tooling the agent uses to verify each PLAN step is not. It lives in the
session's scratch directory, outside the repository, and most of it is
bash:

| Scratch tool | Language | What it does |
| :--- | :--- | :--- |
| `gates.sh` | bash | every local gate of a step: gofmt, `make pre-add-check FILES=…`, `make lint`, the cross `go vet` for `freebsd/amd64`, `openbsd/amd64` and `linux/386`, `-race`, `-shuffle=on -count=2`, `LC_ALL=C`, workspace mode, `go mod tidy -diff`, `make vuln`, `scripts/go-modules.py --check`, `make apicheck`, `make examples`, `make release-check` |
| `mutate.py` | Python | each mutation on a scratch copy of the tree; the named test must fail |
| `<host>.py` | Python | copies the tree to the idle Linux host, runs a script there, redacts the output |
| `<host>_gates.sh`, `<host>_s*.sh` | bash | the gates on that host, with no git history |
| `winprobe_exist.sh` | bash | copies the tree to the Windows test host, runs a script there, redacts the output |
| `make_remote_*.sh` | bash | the gates on that host, with history from a git bundle, one copy per step |
| `names.py` | Python | checks a record's planned type names against every public package's |
| shell `grep` pipelines | bash | the identifier scan of a step's diff |

The cost showed in the same session:

* **The tooling is lost or rebuilt.** On 2026-10-12, `winprobe_exist.sh`
  was gone from the scratch directory between two steps, and the Windows
  run failed with "No such file or directory" until it was rewritten from
  the transcript.
* **Each step forks a script.** `make_remote_p8s2.sh`,
  `make_remote_p8s2b.sh` and `make_remote_p8s3.sh` differ only in a label
  and a test list, edited with `sed`.
* **Shell fails quietly.** The Linux script ran `make` in a copy with no
  `.git`, so `pre-add-check`, `lint`, `tidy-check` and `vuln` failed with
  "Stopping at filesystem boundary", and the script printed no reason.
* **Nobody else can rerun the evidence.** The PLANs quote these tools'
  output, but a reviewer cannot run them.

Where should this tooling live, in what form, and how does it keep the
hosts' names, which must never be committed, out of the repository?

## Decision Drivers

* **The owner's rules:** no bash scripting; Python for logic; native
  tools over `grep`, `sed`, `awk` and `cut`; reusable tooling in the
  repository's `tools/` directory.
* **0017's conventions hold:** Python 3.12 or later, the standard library
  only, commands from argument lists and never `shell=True`, and a test
  that names every failing case.
* **Nothing committed names a host, an account or a real path** (AGENTS.md,
  "Identifiers"). The two test hosts and the identifier patterns are the
  owner's, and stay on the owner's machine.
* **Every host runs the same gates.** One definition of the gates, run on
  macOS, the Linux host and the Windows host, instead of three scripts
  that drift.
* **The contracts of `scripts/` stay:** `make`, CI and the commit gate
  call `scripts/`, and nothing there changes.

## Considered Options

* **A. A Python package, `tools/verify`, run as `python3 -m tools.verify <command>`,** with the hosts and identifier patterns read from a local config file outside the repository
* **B. Python files added to `scripts/`,** beside the ones `make` and CI run
* **C. Separate Python scripts under `tools/`,** one file per tool, no shared package
* **D. Keep the tooling in the session's scratch directory, rewritten in Python**

## Decision Outcome

Chosen option: "A", because it is the owner's `tools/` directory, gives
the gates one definition that every host runs, keeps the hosts' names out
of the repository, and leaves `scripts/` and its callers as they are. The
owner approved it on 2026-10-12 ("approved to proceed").

1. **Layout.** `tools/__init__.py` makes `tools` a package; `tools/verify`
   holds the modules, run as `python3 -m tools.verify <command>` from the
   repository root:

   ```text
   tools/verify/
     __main__.py   argparse: gates, mutate, remote, scan, names
     config.py     the local config: hosts, identifier patterns
     proc.py       run an argument list, output to a file, the exit status kept
     tree.py       the working tree's files: tracked and untracked, not ignored, present
     gates.py      the gates, as data, and their runner
     mutate.py     mutation proofs on scratch copies
     remote.py     copy the tree to a host, run tools.verify there, redact
     scan.py       the identifier scan of a diff
     names.py      a record's planned type names against the public packages'
     tests/        unittest, one module per module
   ```

2. **The gates are data.** Each gate is a name, an argument list, an
   environment and the lines worth quoting from its output. A step names
   its Go files; the runner writes each gate's full output to a file and
   prints one line per gate, `name: exit=N`, with the quoted lines. A gate
   that needs git history, such as `make apicheck`, ~~`make examples`~~ and
   `make release-check`, says so, and the runner skips it, saying it did,
   in a copy with none. *Corrected 2026-10-12, in the PLAN's Step 2:*
   `make examples` needs a git root (`scripts/go-examples.py:348`), which
   a copy with an index has, and no tags; only `apicheck` and
   `release-check`, which runs it, read the tags.
3. **One gate definition on every host.** `remote` copies the tree (the
   files `tree.py` lists), and on the Windows host a git bundle of `main`
   and its tags, to a temporary directory on the host. It starts `python3
   -m tools.verify gates …` there through `ssh`, with an argument list,
   removes the copy, and redacts account names and home paths from what
   it prints. No shell script is sent to a host.
4. **The local config.** `$XDG_CONFIG_HOME/go-tui-lib/verify.json`
   (default `~/.config/go-tui-lib/verify.json`), or `--config PATH`,
   names each host, its kind (`posix` or `windows`), its `ssh`
   destination and the directories to put first on its `PATH`, and the
   identifier patterns `scan` looks for. It is never committed.
   `tools/verify/verify.example.json` shows its shape, with placeholders
   only. With no config, `remote` and `scan` stop and say where the file
   goes.
5. **`scan` replaces the `grep` pipelines.** It reads the staged and
   unstaged diff and the untracked files, and reports each line that
   matches a configured pattern by file and line, never the match itself.
6. **The scratch directory keeps only one-off probes,** such as a Go
   program that tests an upstream library, and run outputs. A probe that
   is run a second time moves to `tools/verify`.
7. **Checks.** `python3 -m unittest discover -s tools` and `python3 -m
   py_compile` over `tools/` join CI's lint step, beside `scripts/`'s.
   Each command's tests run it against fakes: a throwaway repository, a
   fake `ssh` and `scp` that record their argument lists, and patterns
   seen to match.
8. **AGENTS.md** gains a "Tools" section: what `tools/verify` is, that it
   is the agent's and the maintainers' and never `make`'s or CI's, and
   where its config lives.

### Consequences

* Good, because the tooling follows the owner's rules and survives the
  session, and a reviewer can rerun a PLAN's evidence.
* Good, because one gate list runs on every host, so a gate added once
  runs everywhere.
* Good, because the hosts' names, and the identifier patterns, stay on the
  owner's machine.
* Neutral, because `scripts/` and `tools/` are both Python: `scripts/` is
  what `make`, CI and the commit gate run, and `tools/` is what a person
  or an agent runs to verify a change.
* Bad, because it is new code to own, with its own tests, and the hosts
  need `python3` 3.12 or later on the `PATH` that `ssh` gives.
* Bad, because until it is built, a step's remote gates cannot run
  without the bash scripts the owner has ruled out.

### Confirmation

* Each command's unittest module passes, and each mutation the PLAN names
  makes a named test fail.
* `gates` on macOS reports the same exit statuses as the scratch
  `gates.sh` did on the same tree, at
  [0008-PLAN-command-palette.md](0008-PLAN-command-palette.md) Step 3.
* `remote` runs the gates on the Linux and the Windows host, and its
  output names no host, account or home path.
* `scan` finds a planted identifier in a throwaway repository, and nothing
  in the repository's own diff.
* No file under `tools/` names a host, an account or a real path.

## Pros and Cons of the Options

### A. A package under `tools/`, with a local config

* Good, because it is where the owner asked for it.
* Good, because modules share the runner, the tree listing and the
  redaction, so nothing is copied between tools.
* Good, because the config keeps private names out of the repository.
* Bad, because it is the most code of the options.

### B. Python files in `scripts/`

* Good, because `scripts/` is already the repository's Python.
* Bad, because `scripts/` is what `make`, CI and the commit gate call, and
  0017 holds its files to their callers' contracts.
* Bad, because it is not the directory the owner named.

### C. Separate scripts under `tools/`

* Good, because each is one file to read.
* Bad, because the runner, the tree listing and the redaction are copied
  into each, and drift, as the scratch scripts did.

### D. Python in the scratch directory

* Good, because nothing is committed.
* Bad, because the owner asked for the repository, and scratch files were
  already lost once.

## Amendments

### A1 (2026-10-12): how `remote` reaches a host, as built

*Status: accepted (2026-10-12),* recording what
[0018-PLAN-python-verification-tools.md](0018-PLAN-python-verification-tools.md)
Step 4 built and its deviations D1 to D4. Item 3 stands; these are its
details.

1. **Both kinds of host get the bundle.** Item 3 gives the Windows host a
   git bundle; every host gets one, so `apicheck` and `release-check` run
   everywhere, and no gate is skipped for want of history.
2. **The bootstrap goes on stdin.** The remote command is `python3 -`, two
   fixed words; `_bootstrap.py`'s source and the run's parameters follow
   on stdin. No argument reaches a remote shell to be parsed.
3. **Commands are found on their own `PATH`** (`proc.resolve`): on
   Windows a bare name reaches only a `.exe`, and `npx` is `npx.CMD`.
4. **The hosts carry the gates' tools.** A host without one fails that
   gate with 127. On 2026-10-12 the owner chose to install them over
   skipping, so both hosts have `actionlint` and the Linux host
   `markdownlint-cli2` (D2); and a stray empty `.git` in the Linux host's
   temporary directory, which broke every `go build` there, was removed
   (D1).

### A2 (2026-10-14): live output, timed gates, and no gate run twice

*Status: accepted (2026-10-14),* the owner's "approved". At the owner's
request, after
[0008-PLAN-command-palette.md](0008-PLAN-command-palette.md) Step 4's
Windows run showed nothing for its whole length. Implemented by
[0018-PLAN-python-verification-tools.md](0018-PLAN-python-verification-tools.md)
Step 7.

**Found.**

1. **Remote output comes all at once, at the end.** `gates` prints with
   Python's default buffering, which holds whole blocks while stdout is a
   pipe, as it is under `ssh`; the bootstrap's own line flushes, the gates'
   do not. A run that is half done looks stuck.
2. **The default list runs some checks twice.** `make release-check` runs
   `scripts/go-precheck.py` over every module and file: gofmt,
   golangci-lint for three targets, `go vet`, `go test`, `go mod tidy
   -diff`, govulncheck, `go test` again in workspace mode,
   `scripts/go-modules.py --check`, the API diff and the examples. So the
   gates `pre-add-check` (on the step's files, a subset), `lint`'s
   golangci-lint, `tidy`, `workspace-mode`, `modules`, `apicheck` and
   `examples` repeat it. Two parts of the separate gates do not repeat:
   * `make lint` runs `make modernize` (`go fix -diff` for three
     targets) first, which the precheck does not;
   * `make vuln` fails when govulncheck cannot reach its database, where
     the precheck prints "skipped" and goes on.
   * The precheck with no file list formats only tracked files, so the
     step's new, untracked files need the `gofmt` gate.
3. **What the repeats cost.** On 0008 Step 4's macOS run, from each
   output file's last write: 386 s in all, of which the repeated gates
   took 68 s (18%): `workspace-mode` 37, `pre-add-check` 16, `apicheck`
   6, `lint` 5, `examples` 4. Most of the time is the test suite under
   different flags, each its own check: `race` 89 s, `shuffle` 70,
   `release-check` 63, `lc-all-c` 35, and the scripts' and tools' tests 59.
   The Windows host's times are not known: `remote` removes its outputs.

**Decided.**

1. **Every line flushes**, so `remote` shows each gate as it ends.
2. **Each gate is timed:** `name: exit=N (Ns)`, and the summary gives the
   total, so the next record can say where a host's time goes.
3. **The default list runs each check once:** `gofmt`, `modernize`, the
   three cross `go vet`, `race`, `shuffle`, `lc-all-c`, `vuln`,
   `release-check`, `shellcheck`, `markdownlint`, `actionlint`, the
   `script-test-*` and `tools-tests`. `pre-add-check`, `lint`, `tidy`,
   `workspace-mode`, `modules`, `apicheck` and `examples` leave the default
   list, and stay runnable by name with `--only`.
4. **Without history, `release-check` still runs,** with
   `GO_PRECHECK_SKIP_APICHECK=1`, in place of being skipped; it is then
   the one gate that lints, so a copy with no tags is not left unlinted.
5. **What is not changed:** each step still runs every check the PLANs'
   rules name, on every host; only the repeats go. The larger savings, the
   scripts' and tools' tests only when a step touches them, or fewer test
   runs, would drop checks, and are the owner's to ask for.

### A3 (2026-10-25): a failing gate names what failed

*Status: accepted (2026-10-25),* the owner's "proceed". Asked for by the owner in
[0008-PLAN-command-palette.md](0008-PLAN-command-palette.md) Step 7,
deviation D3 ("Record it, fix the tool (Recommended)"). Implemented by
[0018-PLAN-python-verification-tools.md](0018-PLAN-python-verification-tools.md)
Step 8.

**Found.**

1. **A failing gate quotes only its last 15 lines** (`gates.py`, `TAIL`).
   `go test ./...` prints a failing package's `--- FAIL` and `FAIL` lines
   where the package ends, and every package after it follows. On
   2026-10-25 the Windows host's `lc-all-c` gate exited 1, and its 15
   lines were 14 packages' `ok` lines and a bare `FAIL`: they said the
   failing package sorts before `layout`, and not which it was.
2. **The full output was gone.** The bootstrap removes the gates'
   outputs after a run (item 3), so the quoted lines are all a remote
   run leaves. Three reruns passed, so the failure could not be read
   again.

**Considered.**

* **A. Quote the failure lines too:** every line of a failing gate's
  output that marks a failure, up to a limit, before the tail. Small, in
  `gates.py` alone, and the same on every host.
* **B. Bring a failing gate's whole output back:** the bootstrap prints
  it, or `remote` copies it back with `scp` before the directory is
  removed. It keeps everything, but changes the bootstrap, `remote` and
  where a run's files land locally, and a long output reaches the
  terminal or a second local directory.
* **C. Keep the outputs on the host.** It leaves files on a host after
  every run, which item 3 rules out.

**Decided:** "A", because the lines that name a failure are what a record
needs, and they come back through the run's existing output on every
host.

1. **A failing gate quotes its failure lines, then its tail.** A line is
   a failure line when it matches one of `FAIL_LINES`:
   * `^\s*--- FAIL`, a failing test or subtest;
   * `^FAIL\b`, a failing package, or `go test`'s last line;
   * `^panic:`, and `^\s+\S+_test\.go:\d+:`, a test's own message;
   * `^\s*\S+\.go:\d+:\d+:`, a compiler's, vet's or a linter's position.
2. **Up to `FAIL_QUOTE` (40) failure lines are quoted,** in their order,
   leaving out those the tail holds; more give one line, `… N more
   failure lines`. Then come the last 15 lines, as now.
3. **A gate that passes quotes as now,** by its `quote` patterns only.
4. **What is not changed:** the bootstrap still removes every output,
   and nothing new reaches a host.

## More Information

* The scratch tools named above are the ones
  [0007-PLAN-keymap-engine.md](0007-PLAN-keymap-engine.md) and
  [0008-PLAN-command-palette.md](0008-PLAN-command-palette.md) Steps 2 and
  3 used. Their execution records quote their output.
* [0017-MADR-python-repository-scripts.md](0017-MADR-python-repository-scripts.md):
  the Python conventions this record keeps.
* The implementation is
  [0018-PLAN-python-verification-tools.md](0018-PLAN-python-verification-tools.md).
