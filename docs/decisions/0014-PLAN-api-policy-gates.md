---
status: in-progress
date: 2026-10-07
associated-madr: "0014-MADR-native-integration-api.md"
---
# Implement W0: the API diff gate, conformance bans, the collision check, the glossary, the framework-example harness and the conventions

Associated MADR: [0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md),
workstream W0. This is the first of the record's PLANs. The others are:

* [0013-PLAN-cli-integration-helpers.md](0013-PLAN-cli-integration-helpers.md),
  for W1;
* [0014-PLAN-component-native-forms.md](0014-PLAN-component-native-forms.md),
  for W2;
* [0014-PLAN-hardening.md](0014-PLAN-hardening.md), for W3;
* [0014-PLAN-canonicalization.md](0014-PLAN-canonicalization.md), for W4.

## Goal

Before any API changes, the repository can tell when one goes wrong:

* **The API diff gate.** An incompatible change against the previous tag
  fails `make apicheck`, `make release-check` and CI, unless it is listed.
* **Conformance.**
  * It asserts every package was read.
  * It refuses reading the environment, exiting and spawning in library
    code.
* **The collision check.** Two public packages cannot export the same type
  name unless the glossary says the clash is deliberate.
* **The framework examples.** The guide's tier-1 examples for stdlib
  `flag`, cobra, kong and urfave/cli v3 compile and run in a temporary
  module, and the guide's excerpts match them.
* **The conventions.** `AGENTS.md` states:
  * deprecation, options, enums, errors, hooks, constructors and the
    environment;
  * stability lines;
  * the toolchain floor.

No exported API changes in this PLAN.

## Scope

### Facts this PLAN starts from (2026-10-07)

| Fact | Where it was read |
| :--- | :--- |
| `golang.org/x/exp@latest` is `v0.0.0-20261007180756-3d68b386da03`; `apidiff -m -w` exports a module's API, and `-incompatible old new` compares them, in about 1 s and 0.3 s | a probe in a scratch copy |
| `apidiff` exits 0 even when it reports incompatible changes; exit 1 means a load or write error, and 2 bad usage | `cmd/apidiff/main.go:58,63,68,271` at that version |
| its lines are stable: `- ./when.ListValue: removed`, `- package github.com/maccavelli/go-tui-lib/command/cli: removed`; internal packages are skipped with "Ignoring internal package" on stderr | probes: a planted removal; `v0.5.0` against `v0.6.0` |
| in workspace mode a nested module's packages leak into the root's export; with `GOWORK=off` they do not | probe with a planted `stream/probe` module |
| `gorelease` exits 0 on an incompatible change in `v0` | probe |
| the previous tag is `git tag --merged HEAD --no-contains HEAD -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname \| head -1`: `v0.6.0` on `HEAD`, `v0.5.0` with `v0.6.0` checked out | probe |
| CI's checkout fetches no tags (no `fetch-depth`) | `.github/workflows/ci.yml:96` |
| `GOPROXY=off` breaks `go run tool@version`; a `file://` proxy over the module cache works | probe |
| the Makefile loops modules through `scripts/go-modules.sh` (`Makefile:24-33`); tool pins live at `Makefile:10-12`; `help` pads names to 12 characters (`:114`) | `Makefile` |
| `scripts/go-precheck.sh` has a per-module subshell (234-277), a once-only check (282-286), a summary line (289), and the skip switch `GO_PRECHECK_SKIP_VULN` (49, 112-113) | `scripts/go-precheck.sh` |
| CI's `gates` job runs fuzz, vet/gofmt/tidy/modernize/lint, govulncheck and shellcheck/markdownlint/actionlint in that order (`ci.yml:92-173`); script tests run as steps (`:38`, `:110`) | `.github/workflows/ci.yml` |
| conformance finds modules (`modules()` 234-257), scans with `build.Default.ImportDir` (`scanModule()` 261-317), discards the `*types.Package` (`typeCheck()` 177-197, at 193), checks `forbidden()` (71-109) and `check()` (142-173); the must-read list is a constant (346); plants are type-checked in memory (356-433) | `internal/conformance/conformance_test.go` |
| a prototype of the bans passed its plants and found exactly one use in the tree: `tuitest/tuitest.go:127`, `os.Getenv(updateEnv)` | a probe in a scratch copy |
| exported type names declared in more than one public package: `Context` (layout, when), `Kind` (command, when), `Option` (command, termcap, theme, workspace), `Origin` (command, termcap), `Pane` (layout, workspace) | a scratch AST scan |
| no glossary exists; no package states its stability | search; the package doc comments, line 1 of each package's main file |
| `go.mod:3` and `go.work:1` say `go 1.27.1`, with no `toolchain` line; go1.27.1 is the newest patch | `go.mod`, `go.work`, the toolchain list |
| the commands guide's CLI examples (flag, cobra, kong) are in "Run commands from your own CLI" | `docs/guides/commands.md:297-375` |
| `go-modules.sh` (79) and conformance's `skipDir` (230-232) skip `testdata`; golangci-lint skips it by default | the scripts; golangci-lint's defaults |
| depguard refuses cobra, pflag and kong in `$all` files | `.golangci.yml:55-60` |

### Preconditions

* **0011 is complete.** Its Step 3 edits `docs/guides/commands.md`, which
  this PLAN's Step 5 edits too.

### In scope

| Step | Paths | Delivers |
| :--- | :--- | :--- |
| 1 | this PLAN, `docs/README.md` | approval |
| 2 | `internal/conformance/conformance_test.go` | the must-read list from `go list`; the bans, with an allowlist |
| 3 | `internal/conformance/`; `docs/glossary.md` | the collision check and its allowlist; the glossary |
| 4 | `scripts/go-apicheck.sh`, `scripts/go-apicheck_test.sh`, `scripts/apicheck.allow`; `Makefile`; `scripts/go-precheck.sh`; `.github/workflows/ci.yml` | the API diff gate |
| 5 | `testdata/frameworks/`; `scripts/go-examples.sh`, `scripts/go-examples_test.sh`; `Makefile`; `scripts/go-precheck.sh`; `.github/workflows/ci.yml`; `docs/guides/commands.md` | the framework-example harness, and the guide's excerpts tied to it |
| 6 | `AGENTS.md`; each package's documentation; `docs/architecture.md`; `docs/README.md`; `README.md` | the conventions, stability lines, toolchain floor and documents |
| 7 | this PLAN, the MADR, `docs/README.md` | close-out |

### Out of scope

* **Any exported API change,** and fixing the 22 enums that lack text
  forms (W2, W4). This PLAN only writes the rule.
* **Renaming the five colliding names.** The glossary records them as
  deliberate. Renaming any of them is a W4 decision that needs the owner.
* **A release.** These gates ship inside `v0.7.0` with W1.

## Implementation Steps

### Rules

1. **A step starts when the previous one is committed.** The agent
   commits on `main` only when the owner asks in that turn, with `git
   commit --no-edit`. The owner pushes.
2. **Checks.**
   * **For a step that changes Go, scripts, CI or configuration:**
     * `gofmt -l`;
     * `make pre-add-check`;
     * `make lint`, all three targets;
     * with `GOWORK=off`: `go test -race -count=1 ./...`,
       `-shuffle=on -count=2` and `LC_ALL=C`;
     * the tests in workspace mode;
     * `go mod tidy -diff`, `make vuln` and
       `scripts/go-modules.sh --check`;
     * `make release-check`;
     * shellcheck v0.11.0 on changed scripts;
     * actionlint v1.7.12 on changed workflows;
     * the Windows test host on a copy of the tree.
   * **For every step:**
     * markdownlint and the link checker;
     * the citation checker;
     * the identifier scan of the diff.
3. **Mutations** run on a scratch copy, or in a throwaway repository under
   a temporary directory for the script tests. Each must fail its named
   check.
4. **Anything unplanned stops the step,** recorded as a dated deviation.

### Step 1: records

* **This PLAN, approved by the owner.**
* **`docs/README.md`:** a row for each 0014 PLAN.

**Done when** the owner approves.

### Step 2: conformance reads every package, and bans the environment, exit and spawn

**The must-read list.**

* The constant (`conformance_test.go:346`) is replaced. For each module
  from `modules()`, the test runs
  `GOWORK=off go list -f '{{if .GoFiles}}{{.Dir}}{{end}}' ./...` in the
  module's directory and maps each directory to a root-relative path.
* The test fails if any of those paths is missing from what the scan
  read, or if the list is empty.
* The test is a test, so running `go list` is not library code.

**The bans,** added to `forbidden()` and `check()`. Each rule's text is
the message the scan prints:

| Use | Rule |
| :--- | :--- |
| `os.Getenv`, `os.LookupEnv`, `os.Environ`, `os.ExpandEnv` | "reads the environment with os.X" |
| `syscall.Getenv` | "reads the environment with syscall.Getenv" |
| `os.Exit` | "calls os.Exit" |
| `os.StartProcess`; `syscall.Exec`, `syscall.ForkExec`, `syscall.StartProcess` | "starts a process with X" |
| any package-level object of `os/exec` | "uses os/exec.X" |
| an import of `os/exec`, blank imports included (an `*ast.ImportSpec` case in `check()`) | "imports os/exec" |
| `tea.Exec`, `tea.ExecProcess` (Bubble Tea v2's path) | "hands the terminal to a process with tea.X" |

The last row enforces the owner's rule that the library spawns no process
([0003-REPORT](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
§11.5). A program may still call these itself.

**The allowlist** is a table in the test, keyed by root-relative file and
rule, never by line:

| File | Rule | Why |
| :--- | :--- | :--- |
| `tuitest/tuitest.go` | reads the environment with os.Getenv | `TUITEST_UPDATE`, the golden update switch; `tuitest` runs only under `go test` |

An allowlist entry that matches nothing fails the test, so a stale entry
cannot linger.

**Plants,** in `TestScanFindsEachRule`'s table (356-411). Each is one case
with its expected rule counts:

* each environment read, including through an alias and as a function
  value;
* `os.Exit`;
* `os.StartProcess`;
* `exec.CommandContext` with an `*exec.Cmd` variable;
* `import _ "os/exec"`;
* `syscall.ForkExec`;
* `tea.ExecProcess`.

**Mutations:**

| Name | Change, on a scratch copy | Must fail |
| :--- | :--- | :--- |
| S2-1 | `glyph/glyph.go` calls `os.Getenv("X")` | `TestNoPackageOwnsTheTerminal` |
| S2-2 | `when` imports `_ "os/exec"` | `TestNoPackageOwnsTheTerminal` |
| S2-3 | the scan skips the `when` directory | the must-read assertion |
| S2-4 | the `tuitest` allowlist entry is removed | `TestNoPackageOwnsTheTerminal` on `tuitest/tuitest.go` |
| S2-5 | an allowlist entry for a file with no such use is added | the stale-entry assertion |
| S2-6 | the `os.Exit` case is removed from `forbidden()` | `TestScanFindsEachRule` |

**Done when** Rule 2's checks are clean and S2-1 to S2-6 fail as named.

### Step 3: the collision check and the glossary

**The check:** `TestNoTypeNameMeansTwoThings`, in `internal/conformance`.

* `typeCheck` returns the `*types.Package` it now discards (193).
* For every public package of every module (no `internal` path element,
  no test files), the test collects the exported `*types.TypeName`
  objects of the package scope, aliases included.
* It fails on any name that appears in two or more packages, unless the
  name and its exact package set are in the allowlist. A new package with
  an allowed name still fails.
* The allowlist is a table in the test. It mirrors the glossary's
  "deliberate" entries:

  | Name | Packages | Glossary line |
  | :--- | :--- | :--- |
  | `Option` | any package | each package's option type (MADR W0.3) |
  | `Context` | `layout`, `when` | a layout's breakpoints and facts; the values a when-clause reads |
  | `Kind` | `command`, `when` | a command's kind; a when-value's type |
  | `Origin` | `command`, `termcap` | where a request came from; where a fact came from |
  | `Pane` | `layout`, `workspace` | a layout node; a pane component |

* 0013's PLAN adds `Decision` (`command`, `launch`) when `launch` lands,
  dated for removal in `v0.10.0` (0013-MADR A1.6).
* A stale allowlist entry fails the test.

**`docs/glossary.md`.** One line per exported concept whose name is
collision-prone. The deliberate entries are the five above, each with:

* the meaning in each package;
* the rule "qualified by package, never added to".

Then the names reserved by accepted records:

* `launch`: `Choice`, `Config`, `Target`, `Reason`, `Decision`,
  `Streams`, `Flags`, `Restorer`, `ExitError`;
* `command.Verdict` (W4);
* `glyph.Tier` (W1).

Then the names 0007–0009 must not take: `Context`, `Origin`, `Conflict`,
`Policy`. W4's amendments of those records pick their names from here.
The file ends with the rule: a new exported name is checked against this
glossary in the record that introduces it.

**Mutations:**

| Name | Change, on a scratch copy | Must fail |
| :--- | :--- | :--- |
| S3-1 | `theme` exports `type Pane struct{}` | `TestNoTypeNameMeansTwoThings` |
| S3-2 | the `Context` entry is removed | `TestNoTypeNameMeansTwoThings` |
| S3-3 | an entry for a name no package exports is added | the stale-entry assertion |
| S3-4 | `typeCheck` returns nil | the test's must-check-something assertion |

**Done when** Rule 2's checks are clean, S3-1 to S3-4 fail, and the
glossary is linked (Step 6).

### Step 4: the API diff gate

**`scripts/go-apicheck.sh`,** written in the style of `go-fuzz.sh` and
`go-modules.sh`.

* **The tool.** `APIDIFF_VERSION` comes from the Makefile; the default is
  `v0.0.0-20261007180756-3d68b386da03`. The command is `go run
  golang.org/x/exp/cmd/apidiff@$APIDIFF_VERSION`. It is never added to
  `go.mod`.
* **For each module** from `scripts/go-modules.sh`, in its directory with
  `GOWORK=off`:
  1. **The tag prefix:** `v` for the root, `<dir>/v` for a nested module.
  2. **The base:** `git tag --merged HEAD --no-contains HEAD -l
     '<prefix>[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -1`. If there
     is none, print `apicheck: <module>: no previous tag, skipped` and go
     on.
  3. **Export the base:** `git archive <tag> <dir>` into a temporary
     directory, then `apidiff -m -w old.api <module path>` there.
  4. **Export the tree:** `apidiff -m -w new.api <module path>`.
  5. **Compare:** `apidiff -m -incompatible old.api new.api`. A non-zero
     exit is a failure (a load or usage error).
  6. **Judge the output,** each stdout line against `scripts/apicheck.allow`:
     * a line not listed is printed, and the script fails;
     * a listed line that apidiff no longer prints is stale, is printed,
       and the script fails.
* **`scripts/apicheck.allow`** holds comment lines (`#`) and entries
  `<module dir><TAB><apidiff line>`.
  * Each block starts with `# <PLAN filename> <step>`, naming the PLAN
    that allows the change.
  * It is empty today.
  * A release's close-out removes its entries. The base moves to the new
    tag, which turns them stale and so forces the clean-up.
* **The skip switch** is `GO_PRECHECK_SKIP_APICHECK=1`, for offline work,
  like `GO_PRECHECK_SKIP_VULN`.
* **Exit codes:** 0 clean or skipped; 1 an unlisted or stale line; 2 a
  tool or git failure.

**`scripts/go-apicheck_test.sh`,** on throwaway repositories in a
temporary directory, with `GOPROXY` pointed at the module cache:

* no previous tag gives a skip;
* an unchanged API is clean;
* an addition is clean;
* a removal fails;
* the same removal, listed, is clean;
* a stale listed line fails;
* a tag on `HEAD` compares against the tag before it;
* a nested module uses its own prefix;
* a failing `apidiff` gives exit 2.

**Wiring:**

* **`Makefile`:**
  * `APIDIFF_VERSION ?= v0.0.0-20261007180756-3d68b386da03` beside the
    tool pins (10-12);
  * `apicheck` in `.PHONY`;
  * `apicheck: ## Fails on an incompatible API change since the previous
    tag, per module`, which runs `./scripts/go-apicheck.sh`.
* **`scripts/go-precheck.sh`:**
  * in the no-file-list branch (230-232, the `release-check` path), after
    `go-modules.sh --check`, run `./scripts/go-apicheck.sh` unless
    skipped;
  * the summary (289) names it.
* **`.github/workflows/ci.yml`:**
  * `gates`' checkout (96) gets `fetch-depth: 0`;
  * after govulncheck (150), a step `apicheck` runs
    `./scripts/go-apicheck_test.sh` then `make apicheck`;
  * the comment cites this PLAN.

**Mutations:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| S4-1 | the script judges `apidiff`'s exit code, not its output | the removal case in `go-apicheck_test.sh` |
| S4-2 | the stale-line check is removed | its test case |
| S4-3 | `--no-contains HEAD` is dropped | the tag-on-`HEAD` case |
| S4-4 | `GOWORK=off` is dropped | the nested-module case |
| S4-5 | on a scratch copy of the real tree, `when.ListValue` is unexported | `make apicheck` |

**Done when**:

* Rule 2's checks are clean;
* the mutations fail as named;
* `make apicheck` on the real tree is clean against `v0.6.0`;
* CI shows the new step green on the push.

### Step 5: the framework-example harness

**The programs,** under `testdata/frameworks/`, which the go command,
`go-modules.sh` and golangci-lint all skip:

* **`go.mod.tmpl`,** the module `example.com/frameworks`. It requires the
  root by `replace` to `@ROOT@`, and pins:
  * cobra v1.10.2 (with pflag v1.0.10);
  * kong v1.16.1;
  * urfave/cli/v3 v3.14.0.

  **`go.sum`** is committed beside it.
* **`flag/main.go`, `cobra/main.go`, `kong/main.go`, `urfave/main.go`:**
  the commands guide's "Run commands from your own CLI" programs, each
  complete and runnable, on a registry holding a `session.save`
  Destructive command. urfave/cli v3 is new to the guide, as tier 1 asks.
* **Region markers.** Each program marks the code the guide shows with
  `// guide:<name>` and `// guide:end`.
* **`cases.txt`.** One case per line: `<framework> <args…> => <exit> <stdout
  substring>`. For each framework:
  * `save notes` refused, exit 1;
  * `save --yes notes` saved, exit 0;
  * a bad flag, exit 2 for flag and kong, 1 for cobra and urfave,
    recorded as measured in this step.

**`scripts/go-examples.sh`:**

1. Copy `testdata/frameworks` into a temporary directory, and write
   `go.mod` from the template with `@ROOT@` set to the repository root.
2. With `GOWORK=off`, `go build ./...` and `go vet ./...`.
3. Run every case and compare.
4. `--check-guide`: every fenced Go block in `docs/guides/*.md` preceded
   by `<!-- from: testdata/frameworks/<file>#<name> -->` must equal that
   region, byte for byte, after removing the markers.

The skip switch is `GO_PRECHECK_SKIP_EXAMPLES=1`, for offline work.
Exit codes: 0, 1 a failed case or excerpt, 2 a build or tool failure.

**`scripts/go-examples_test.sh`,** in a temporary directory:

* a passing case;
* a case with a wrong exit;
* an excerpt that differs by one byte;
* a missing region.

**Wiring:**

* **`Makefile`:** `examples: ## Builds and runs the framework examples
  in a temporary module`. The name is eight characters, inside `help`'s
  12.
* **`go-precheck.sh`:** the release-check branch runs it unless skipped.
* **CI `gates`:** a step after `apicheck` running the test script, then
  `make examples`.
* **`docs/guides/commands.md`:**
  * each framework block gains its `<!-- from: … -->` line;
  * a urfave/cli v3 block is added;
  * the text says the examples are compiled and run on every release.
* **`AGENTS.md`, Dependencies,** gains a sentence. The framework examples
  under `testdata/frameworks` build in a temporary module outside the
  repository's modules. That module may require the tier-1 frameworks
  that 0014-MADR names; the library's modules never do.

**Mutations:**

| Name | Change | Must fail |
| :--- | :--- | :--- |
| S5-1 | the cobra program saves without `--yes` | its case |
| S5-2 | one byte of the guide's kong excerpt changes | `--check-guide` |
| S5-3 | the excerpt's `from` line names a missing region | `--check-guide` |
| S5-4 | the script ignores the exit status | the wrong-exit test |

**Done when** Rule 2's checks are clean, the mutations fail, and CI's
`examples` step is green.

### Step 6: conventions, stability lines, toolchain floor, documents

* **`AGENTS.md`,** a new section "API conventions" after "TUI
  conventions". It holds:
  1. **Deprecation (MADR W0.1):**
     * one minor release with the old name as a wrapper, alias or
       constant, marked `// Deprecated: use X`;
     * a renamed field keeps both fields, read new-then-old;
     * release notes list both.
  2. **The API diff gate:** `make apicheck` and `scripts/apicheck.allow`,
     whose entries are cleaned at release.
  3. **Names:** the glossary, and the collision check.
  4. **Options:** `With…`, `Without…`, `On…`, over opaque option types for
     new code. The existing ones change in W4.
  5. **Enums:** `String`, `MarshalText`, `UnmarshalText` through
     `internal/enum`.
  6. **Errors:** `Err…` sentinels, typed errors with `Unwrap`, and
     `ExitCode() int` on an error that ends a CLI.
  7. **Hooks:** an interface with a `…Func` adapter.
  8. **Constructors:** `NewX`, and `MustNewX` for programming errors.
  9. **The environment:** `termcap.Env` only. Library code never reads
     the process's environment, never exits and never spawns
     (conformance enforces it).
  10. **Stability lines.**
  11. **The toolchain floor:** it moves only by a record, to the newest
      patch of a supported release.
* **Stability lines.** Each package's documentation ends with
  `// Stability: stable. Exported names change only through the
  deprecation policy in AGENTS.md, "API conventions".`, placed before
  the `package` clause in:
  * `command/command.go`, `glyph/glyph.go`, `layout/layout.go`;
  * `termcap/termcap.go`, `termcap/termcaptest/termcaptest.go`;
  * `termsvc/termsvc.go`, `theme/theme.go`, `tuitest/tuitest.go`;
  * `when/when.go`, `workspace/workspace.go`.

  Internal packages get `// Stability: internal.`
* **`docs/architecture.md`:**
  * "Tooling": the targets `apicheck` and `examples`, the scripts, the CI
    steps;
  * "Tree": `docs/glossary.md`, `testdata/frameworks/`;
  * "What is not here": the `make apicheck` line, which said it waits for
    `v1`, goes.
* **`docs/README.md`:** "I want to…" rows for the glossary, the API
  conventions and the framework examples.
* **`README.md`:** the "I want to…" table gains the glossary.

**Mutation S6-1:** a scratch copy drops the stability line from
`when/when.go`. A test, `TestEveryPackageStatesStability` in
`internal/conformance`, reads each package's doc comment, and must fail.

**Done when** Rule 2's checks are clean, S6-1 fails, and every link
resolves.

### Step 7: close-out

* **Verification,** item by item.
* **This PLAN `complete`.** No release; these gates ship in `v0.7.0`.

## Verification

* **Conformance:**
  * the must-read list comes from `go list`, and the bans and their
    allowlist hold (S2-1 to S2-6);
  * no package uses the environment, `os.Exit` or a process, except the
    listed `tuitest` switch.
* **The collision check** holds, with five deliberate names and the
  glossary (S3-1 to S3-4).
* **`make apicheck`** is clean against `v0.6.0`; its test script passes;
  S4-1 to S4-5 fail as named; CI runs it with tags.
* **The four tier-1 framework programs** build and pass their cases, and
  the guide's excerpts match them (S5-1 to S5-4).
* **`AGENTS.md` states the conventions.** Every package states its
  stability (S6-1).
* **Rule 2's checks are clean at every step,** on macOS and the Windows
  test host, and CI is green.

## Rollout and Rollback

* **Rollout:**
  * one commit per step;
  * no tag;
  * nothing a consumer imports changes.
* **Rollback:** each step is one commit to revert. Reverting Step 4 or 5
  also removes its CI step.

## Execution Record

The owner approved execution on 2026-10-07 ("Commit to main and proceed",
after 0011's close-out was committed as `5c6f4bc`). Step 1 is done: this
PLAN was committed in `46f4c73`, and its row is in `docs/README.md`. The
precondition holds: 0011 is complete.

### Step 2: conformance reads every package, and bans the environment, exit and spawn

#### Deviations

* **D1 (2026-10-07): the `syscall.ForkExec` plant runs off Windows only.**
  * **Found:** `syscall.ForkExec` does not exist on Windows (`GOOS=windows
    go doc syscall ForkExec` fails; `Exec`, `StartProcess` and `Getenv`
    exist). The conformance test runs on the Windows CI leg, where the
    planted source would not type-check. The ban on the name is
    unaffected.
  * **Asked,** with two options:
    * plant `syscall.StartProcess` everywhere, and keep the `ForkExec`
      plant where it exists;
    * plant `StartProcess` only.
  * **The owner chose** the first, the recommendation. A plant case
    carries a GOOS it skips on, and the `ForkExec` case skips `windows`.
  * **Files:** none beyond this step's.

#### What was built

All in `internal/conformance/conformance_test.go`:

* **The must-read list comes from `go list`.**
  * `mustRead` runs `GOWORK=off go list -f '{{if .GoFiles}}{{.Dir}}{{end}}'
    ./...` in each module.
  * The test fails on any package it lists that the scan did not read, and
    on an empty list.
  * It now asserts 14 packages where the constant named 6.
* **The bans,** in `forbidden`:
  * `os.Getenv`, `LookupEnv`, `Environ` and `ExpandEnv`, and
    `syscall.Getenv`;
  * `os.Exit`;
  * `os.StartProcess`, and `syscall.Exec`, `ForkExec` and `StartProcess`;
  * any package-level object of `os/exec`;
  * `tea.Exec` and `tea.ExecProcess`;
  * and, in `check`, any import of `os/exec`, blank imports included.
* **The allowlist,** keyed by root-relative file and rule:
  * `tuitest/tuitest.go` reads the environment with `os.Getenv`;
  * an entry that matches nothing fails the test.
* **Ten plants:**
  * the four environment reads;
  * a read through an alias and as a function value;
  * `os.Exit`;
  * `os.StartProcess`;
  * `os/exec` with an `*exec.Cmd`;
  * a blank `os/exec`;
  * `syscall.StartProcess` with `syscall.Getenv`;
  * `syscall.ForkExec` (D1);
  * `tea.ExecProcess`.
* **The package documentation** names the new rules.
* **`make lint`'s modernize step** turned the new `errors.As` into
  `errors.AsType`.

#### Checks

* **Mutations, on scratch copies.** All killed, and run again after the
  modernize change:
  * S2-1, glyph calls `os.Getenv`: `glyph/glyph.go:12 reads the
    environment with os.Getenv`;
  * S2-2, a blank `os/exec` in `when`: `when/when.go:34 imports os/exec`;
  * S2-3, the scan skips `when`: "the scan did not read when";
  * S2-4, the `tuitest` entry removed: `tuitest/tuitest.go:127 reads the
    environment with os.Getenv`;
  * S2-5, a stale entry: `allowed lists glyph/glyph.go "calls os.Exit",
    which the scan no longer finds`;
  * S2-6, the `os.Exit` rule removed: the plant reports "found 0 times,
    want 1".
* **Lint and the pre-add check:**
  * `make pre-add-check FILES=…`: "1 file(s) clean";
  * `make lint`: exit 0, three "0 issues.", after the modernize change.
    Before it, `make modernize` failed on `errors.As`;
  * `make release-check`: "124 file(s) clean in 1 module(s)";
  * `make vuln`: "No vulnerabilities found."
* **The tests,** with `GOWORK=off`: `-race -count=1`, `-shuffle=on
  -count=2` and `LC_ALL=C` each gave 15 `ok`, and so did workspace mode.
* **The Windows test host,** on a copy of the tree, go1.27.1
  windows/amd64:
  * `make pre-add-check` "124 file(s) clean";
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` 15 `ok`;
  * the conformance tests pass, with `syscall.ForkExec` skipped.
* **The identifier scan of the diff:** no match.

### Step 3: the collision check and the glossary

The owner approved it on 2026-10-07 ("proceed"), after Step 2 was
committed as `7e8f09c` and pushed.

No deviation. The PLAN's table described `layout.Context` as "a layout's
breakpoints and facts". Its documentation says it "carries the state and
collects the plan while a tree is arranged", and the glossary uses that
(`go doc ./layout Context`).

#### What was built

* **`internal/conformance/conformance_test.go`:**
  * `typeCheck` returns the `*types.Package` it used to discard;
  * `scanModule` also returns every exported `*types.TypeName` of each
    public package (no `internal` path element), aliases included;
  * **`TestNoTypeNameMeansTwoThings`** fails on any name two or more
    public packages export, unless `sharedNames` holds the name and its
    exact, sorted packages;
  * `"*"` allows `Option` in any package;
  * an entry fewer than two packages export is stale, and fails;
  * a scan that collects no name fails.
* **`sharedNames`** holds `Option` (any), `Context` (layout, when),
  `Kind` (command, when), `Origin` (command, termcap) and `Pane` (layout,
  workspace). The test passes, so these are exactly today's clashes.
* **`docs/glossary.md`** has four parts:
  * the five deliberate names, with their meaning in each package from
    `go doc`;
  * the names the accepted records reserve:
    * `launch`'s ten names;
    * `glyph.Tier`;
    * `command`'s `Verdict`, `Format`, `Param`, `PanicError` and
      `LoadOption`;
    * `workspace`'s `PlainViewer` and `GlyphThemeBuilder`;
  * the names 0007–0009 must not take: `Context`, `Origin`, `Conflict`,
    `Policy`;
  * the rule for a new name.

  Step 6 links it.

#### Checks

* **Mutations, on scratch copies.** All killed:
  * S3-1, `theme` exports `Pane`: "Pane is exported by [layout theme
    workspace], and sharedNames allows [layout workspace]";
  * S3-2, the `Context` entry removed: "Context is exported by [layout
    when]: one name, one meaning";
  * S3-3, a stale `Widget` entry: "sharedNames lists Widget [layout
    theme], which fewer than two public packages export";
  * S3-4, `typeCheck` returns no package: "the scan collected no
    exported type name".
    * Its first form did not compile ("declared and not used: pkg"), so
      it did not reach the assertion.
    * It was rewritten to compile, and run again.
* **Lint and the pre-add check:**
  * `make lint`: three "0 issues.";
  * `make pre-add-check FILES=…`: "1 file(s) clean";
  * `make release-check`: "124 file(s) clean".
* **The tests,** with `GOWORK=off`: `-race`, `-shuffle=on -count=2` and
  `LC_ALL=C` each gave 15 `ok`, and so did workspace mode.
* **`docs/glossary.md`:** markdownlint 0 issues; the link check 0 broken.
* **The Windows test host,** go1.27.1 windows/amd64:
  * `make pre-add-check` "124 file(s) clean";
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` 15 `ok`;
  * the three conformance tests pass, with `syscall.ForkExec` skipped.
* **The identifier scan of the diff:** no match.

### Step 4: the API diff gate

The owner approved it on 2026-10-07 ("proceed, i committed"), after Step 3
was committed as `475d6ac`.

#### Deviations

* **D2 (2026-10-07): apidiff is installed once per run, not run with `go
  run` per call.**
  * **Found:** the PLAN's tool line is `go run
    golang.org/x/exp/cmd/apidiff@$APIDIFF_VERSION`. The script calls
    apidiff three times per module, and the tests need to inject a binary
    of their own (the failing-apidiff case).
  * **Built:** `GOBIN=<temp> GOWORK=off go install
    golang.org/x/exp/cmd/apidiff@$APIDIFF_VERSION` once, into the run's
    temporary directory, which the run removes. `APIDIFF=<path>` replaces
    that binary. The version is the same Makefile pin, and nothing is
    added to `go.mod`.
  * **Asked;** the owner chose this form, the recommendation.
* **D3 (2026-10-07): the test's `GOPROXY` reads the module cache first,
  then the configured proxy.**
  * **Found:** the PLAN says the test points `GOPROXY` at the module
    cache. The throwaway modules need no module. apidiff does, and on a
    CI runner with a cold cache a cache-only proxy cannot install it.
  * **Built:** `GOPROXY=file://<GOMODCACHE>/cache/download,<go env
    GOPROXY>`. A warm cache needs no network, and a cold one fetches
    apidiff once. On Windows the cache path goes through `cygpath -m`.
  * **Asked;** the owner chose this form, the recommendation.

#### What was built

* **`scripts/go-apicheck.sh`**, as the PLAN describes, with D2:
  * the tag prefix is empty for the root and `<dir>/` for a nested
    module. The pattern is `<prefix>v[0-9]*.[0-9]*.[0-9]*`;
  * each module's base is extracted into a numbered directory (`base1`,
    …), not one named after the module. The first form, `base-.`, ended
    in a dot. Windows strips that dot from a path, so apidiff's `chdir`
    failed there ("The system cannot find the file specified"), although
    Git Bash's `mkdir` and `cd` accepted it. The Windows run found this,
    and it was fixed within this step;
  * `GO` names the go command, as in `go-modules.sh`.
* **`scripts/apicheck.allow`:** the header comment only.
* **`scripts/go-apicheck_test.sh`:**
  * apidiff is installed once and passed to every case (D3);
  * it has the PLAN's nine cases and one more, 19 assertions in all;
  * the extra case moves a package from the root into a new nested
    module. With `GOWORK=off` that is a removal from the root, and the
    new module has no tag yet;
  * a tag on `HEAD` is never the base, so each case that tags then adds a
    commit.
* **`Makefile`:** the `APIDIFF_VERSION` pin, `apicheck` in `.PHONY`, and
  the `apicheck` target. `make help` lists it.
* **`scripts/go-precheck.sh`:**
  * with no file list, after `go-modules.sh --check`, it runs
    `go-apicheck.sh` unless `GO_PRECHECK_SKIP_APICHECK=1`;
  * a failure keeps the script's exit status;
  * the summary ends with `apicheck` when it ran;
  * the header and the Env list name it.
* **`.github/workflows/ci.yml`:** the `gates` checkout has `fetch-depth:
  0`. A step `apicheck` after govulncheck runs the test, then `make
  apicheck`.

#### Checks

* **Mutations.** S4-1 to S4-4 ran on copies of the script in the
  scratchpad, with the test pointed at each through `APICHECK`. S4-5 ran on
  a scratch clone. All were killed, and all were run again after the base
  directory fix:
  * S4-1, the diff output not read: the removal, stale, tag-on-`HEAD` and
    both nested cases fail ("a removal fails: want 1, got 0"), 10 of 19;
  * S4-2, the stale check removed: "a stale entry fails: want 1, got 0";
  * S4-3, `--no-contains HEAD` dropped: "a tag on HEAD compares with the
    tag before: want 1, got 0";
  * S4-4, `GOWORK=off` dropped from the module loop: both nested-module
    cases exit 2, "found no packages for module example.com/r" (the base
    holds `go.work`);
  * S4-5, `when.ListValue` unexported in the clone: `make apicheck` fails
    with ". - ./when.ListValue: removed". The unmutated clone is clean.
* **The precheck wiring,** on the same mutated clone:
  * `GO_PRECHECK_SKIP_VULN=1 ./scripts/go-precheck.sh` exits 1 and names
    the change;
  * with `GO_PRECHECK_SKIP_APICHECK=1` it exits 0 and says it skipped.
* **On the tree:**
  * `make apicheck`: "against v0.6.0, 0 incompatible change(s)", "clean";
  * `./scripts/go-apicheck_test.sh`: "19 passed, 0 failed";
  * shellcheck 0.11.0 on `scripts/*.sh` and actionlint v1.7.12: clean;
  * `make release-check`, run again after the fix: "124 file(s) clean
    in 1 module(s) (…, govulncheck, apicheck)";
  * `make lint`: three "0 issues.";
  * `make vuln`: "No vulnerabilities found.";
  * with `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C` each
    gave 15 `ok`, and so did workspace mode.
* **The Windows test host,** go1.27.1 windows/amd64:
  * the copy's history came from a bundle of `main` and its 15 tags, so
    the gate had `v0.6.0` to compare with;
  * the first run found the `base-.` defect above. Every tagged test
    case exited 2, and so did `make apicheck`;
  * after the fix, `go-apicheck_test.sh` gave "19 passed, 0 failed";
  * `make apicheck`: "against v0.6.0, 0 incompatible change(s)";
  * `make pre-add-check`: "124 file(s) clean ... apicheck";
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` gave 15 `ok`.
* **The identifier scan of the diff:** no match.

### Step 5: the framework-example harness

The owner approved it on 2026-10-07 ("approved proceed"), after Step 4 was
committed as `3d3fa98`.

#### Deviations

Four were found before anything was written. Each was asked, and the
owner chose the recommendation each time.

* **D4 (2026-10-07): an excerpt is compared after dedenting, with leading
  tabs as four spaces.**
  * **Found:** the PLAN says an excerpt equals its region "byte for byte,
    after removing the markers". The programs are gofmt'd, so they
    indent with tabs, and the regions sit inside function bodies. The
    guide's Go blocks indent with four spaces, and markdownlint's MD010
    (on; `.markdownlint-cli2.jsonc` does not override it) forbids hard
    tabs in code blocks.
  * **Chosen:** `--check-guide` removes the region's common leading tabs,
    writes each remaining leading tab as four spaces, then compares byte
    for byte. MD010 stays on.
  * **Not chosen:** tabs in the guide with MD010 off for code blocks,
    which loosens a lint rule to fit the check.
* **D5 (2026-10-07): the precheck formats a `testdata` Go file, and runs
  the examples gate for one under `testdata/frameworks`.**
  * **Found:** with a file list, `go-precheck.sh` passes each file's
    directory to `go vet` and `go test`. In a scratch clone,
    `testdata/frameworks/cobra/main.go` failed both with "no required
    module provides package github.com/spf13/cobra". The machine-wide
    agent gate runs the same check on each commit, so it would refuse
    the commit. No Go file is tracked under `testdata` today.
  * **Chosen:** a file with a `testdata` path element is gofmt'd and left
    out of the vet and test packages, since the go command ignores
    `testdata`. When a file under `testdata/frameworks` is given, the
    precheck also runs `go-examples.sh`, unless
    `GO_PRECHECK_SKIP_EXAMPLES=1`.
  * **Not chosen:** gofmt only, which would leave the examples to the
    release check and CI.
* **D6 (2026-10-07): a case names the stream it matches.**
  * **Found:** the PLAN's case matches a stdout substring. A refusal is an
    error, written to stderr, so the refused case could test only its
    exit status.
  * **Chosen:** `<framework> <args…> => <exit> stdout|stderr <substring>`.
  * **Not chosen:** matching stdout and stderr together, which would pass
    a result written to the wrong stream.
* **D7 (2026-10-07): `go-examples.sh --update` refreshes `go.sum`.**
  * **Found:** the committed `testdata/frameworks/go.sum` holds the
    root's requirements too, through the `replace`. A change to the
    root's `go.mod` (0013 adds `golang.org/x/term`) breaks the examples'
    build with a missing `go.sum` entry.
  * **Chosen:** `--update` runs `go mod tidy` in the temporary module and
    writes `go.sum` back, as `-tuitest.update` rewrites a golden file.
    `go.mod.tmpl` is written back too, with `@ROOT@` restored, since its
    indirect requirements change with `go.sum`. The diff is read before
    it is committed. The build failure names the command.
  * **Not chosen:** a manual copy, described in the failure message.

#### What was built

* **`testdata/frameworks/`:**
  * `go.mod.tmpl`, module `example.com/frameworks`, written by `go mod
    tidy`. It requires cobra v1.10.2, kong v1.16.1 and urfave/cli/v3
    v3.14.0; pflag v1.0.10 comes in through cobra, as an indirect
    requirement. The root is replaced by `@ROOT@`. Each pin is that
    module's latest release (`go list -m -versions`);
  * `go.sum`, beside it;
  * `flag/`, `cobra/`, `kong/` and `urfave/`, each a complete program:
    * the guide's `yesGate`, `runCLI` and `saveArgs`;
    * a registry holding `session.save`, `Destructive`, offered on
      `SurfaceCLI`, whose handler prints `saved <name>`;
    * a `main` around the guide's lines;
  * regions: `flag/main.go` holds `runcli` and `flag`; the others hold
    `cobra`, `kong` and `urfave`;
  * `cases.txt`, with three cases for each framework (D6).
* **The exit statuses measured** for a bad flag (`save --bogus notes`):
  * `flag` 2 (`flag.ExitOnError`);
  * `kong` 80, Kong's own status for a parse error, where the PLAN
    expected 2;
  * `cobra` 1 and `urfave` 1, from the programs' `os.Exit(1)` on an
    error.

  The PLAN asked for them to be recorded as measured. A refused save
  exits 1 in all four, with "refused: session.save" on stderr. `--yes`
  exits 0 with "saved notes" on stdout.
* **`scripts/go-examples.sh`:**
  * the PLAN's four steps, with D4, D6 and D7. Steps 1 to 4 are the
    default, and `--check-guide` runs step 4 alone;
  * a region's blank lines at either end are dropped. gofmt puts a blank
    line before a top-level `// guide:end`, and the PLAN's "byte for
    byte" did not foresee it. This was found by gofmt in this step;
  * the build writes the programs into a temporary directory. A plain
    `go build ./...` of a single main package writes its binary into
    the module, where it collided with the program's directory. The new
    test found this in this step;
  * it also fails on a `from` line that no Go block follows, a reference
    outside `testdata/frameworks`, a missing file, a region with no
    `// guide:end`, a case naming no program, and a stream other than
    stdout or stderr.
* **`scripts/go-examples_test.sh`:** the PLAN's four cases and five more,
  19 assertions in all, on throwaway trees whose one program needs no
  module, so no network:
  * the substring in the other stream;
  * a `from` line with no block;
  * `--check-guide` neither builds nor runs, and a build failure exits 2
    and names `--update`;
  * `--update` keeps `@ROOT@`;
  * the skip switch.
* **Wiring:**
  * `Makefile`: `examples` in `.PHONY`, and the target;
  * `go-precheck.sh`, as D5 describes: the examples run on the
    release-check path, or when a file list names a file under
    `testdata/frameworks`, unless skipped. The summary ends with
    `examples`;
  * CI `gates`: a step `examples` after `apicheck`;
  * `docs/guides/commands.md`:
    * the five framework blocks carry their `from` lines;
    * a urfave/cli v3 block was added;
    * the opening names urfave/cli;
    * the last bullet says the examples are compiled and run before
      every release, and what the cases check;
  * `AGENTS.md`, Dependencies: the sentence the PLAN gives.

#### Checks

* **Mutations, on scratch clones and copies.** All killed:
  * S5-1, `cobra` saves without `--yes` (`yesGate(true)`): "cobra save
    notes => 1 stderr refused: session.save: exit 0, want 1";
  * S5-2, a space in the guide's `kong.Parse(&cli)`: `--check-guide`
    exits 1 with the diff, "-kctx := kong.Parse(&cli)" against
    "+kctx := kong.Parse(&cli )";
  * S5-3, the kong `from` line names `#kong-cli`: `has no region
    "// guide:kong-cli"`;
  * S5-4, the exit status ignored: "a wrong exit fails: want 1, got 0".
* **D5's precheck wiring,** on the scratch clones:
  * before the change, `go-precheck.sh testdata/frameworks/cobra/main.go`
    failed in vet and test ("no required module provides package
    github.com/spf13/cobra");
  * after it, the same command exits 0, ending "(…, examples)";
  * with S5-1 applied, it exits 1 and shows the failed case.
* **On the tree:**
  * `make examples`: "12 case(s) run", "5 excerpt(s) checked", "clean";
  * `./scripts/go-examples_test.sh`: "19 passed, 0 failed";
  * shellcheck 0.11.0 on `scripts/*.sh`, actionlint v1.7.12,
    markdownlint and the relative-link check: clean;
  * `make release-check`: "124 file(s) clean in 1 module(s) (…,
    apicheck, examples)". The four programs are untracked until they are
    committed, so this count leaves them out. The clone's precheck
    covered them;
  * `make lint`: three "0 issues.";
  * `make vuln`: "No vulnerabilities found.";
  * `go mod tidy -diff`: clean;
  * with `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    each gave 15 `ok`, and so did workspace mode.
* **The Windows test host,** go1.27.1 windows/amd64, with the history
  from a bundle of `main` and its tags:
  * `go-examples_test.sh` gave "19 passed, 0 failed";
  * `make examples` was clean, so the `cygpath` path in the `replace`
    works;
  * `make pre-add-check FILES=testdata/frameworks/cobra/main.go` gave "1
    file(s) clean … examples";
  * `make pre-add-check` gave "124 file(s) clean … apicheck, examples";
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` gave 15 `ok`.
* **The identifier scan of the diff and the new files:** no match.
* **Not yet seen:** CI's `examples` step, which runs on the push.
