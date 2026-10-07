---
status: complete
date: 2026-10-07
associated-madr: "0012-MADR-bring-your-own-cli.md"
---
# Retire the CLI front ends: retract the adapters, remove `command/cli` and A12's check, release `v0.6.0`

Associated MADR: [0012-MADR-bring-your-own-cli.md](0012-MADR-bring-your-own-cli.md)

## Goal

go-tui-lib is a TUI layer for a Go program that brings its own CLI.

* `command/cobracmd` and `command/kongcmd` are retracted at `v0.1.1` and
  gone from `main`.
* The root's `v0.6.0` has no `command/cli` and no A12 check.
* depguard refuses every CLI framework.
* The documents say what the library is, and show a program's own CLI
  running the registry's commands.

## Scope

### Facts this PLAN starts from (2026-10-07)

| Fact | Where it was read |
| :--- | :--- |
| the root is at `v0.5.0`; the adapters at `command/cobracmd/v0.1.0` (requiring root `v0.4.0`) and `command/kongcmd/v0.1.0` (requiring `v0.5.0`) | `git tag`; the two `go.mod` files |
| only the adapters and `command/cli`'s own tests import `command/cli` | `grep -rln 'command/cli"' --include=*.go` |
| A12's code is `command/kongname.go`, the call and documentation in `command/args.go`, and `command/args_test.go` | `git diff --stat v0.4.0 v0.5.0` |
| the tags A12 added: `workspace/commands.go` (`zoomArg.Pane` `optional:""`, `stateArg.State` `required:""`), `command/decode_test.go` (four `required:""`), `command/export_test.go` (one `optional:""`) | 0006-PLAN Step 11a |
| `.golangci.yml` confines Cobra and pflag to `command/cobracmd/**`, and Kong to `command/kongcmd/**` | `.golangci.yml`, the `cobra` and `kong` depguard rules |
| `go.work` lists `.`, `./command/cobracmd` and `./command/kongcmd` | `go.work` |
| the 0011 pair is written and not yet committed | `git status` |

### In scope

| Step | Paths | Delivers | Released in |
| :--- | :--- | :--- | :--- |
| 1 | `docs/decisions/0012-*`, 0006-MADR A13, 0010-MADR A3, a 0011-PLAN revision, `docs/README.md` | the records | — |
| 2 | `command/cobracmd/go.mod`, `command/kongcmd/go.mod`, each package's documentation | the retracting versions | `command/cobracmd/v0.1.1`, `command/kongcmd/v0.1.1` |
| 3 | `command/cobracmd/`, `command/kongcmd/`, `go.work`, `.golangci.yml` | the modules removed; depguard refuses CLI frameworks everywhere | — |
| 4 | `command/cli/`, `command/kongname.go`, `command/args.go`, `command/args_test.go`, `workspace/commands.go`, `command/decode_test.go`, `command/export_test.go`, comments naming `command/cli` | the root's API change | `v0.6.0` |
| 5 | `README.md`, `AGENTS.md`, `docs/README.md`, `docs/architecture.md`, `docs/guides/commands.md`, `docs/guides/releasing.md` | the documents | `v0.6.0` |
| 6 | this PLAN's record | the release and its smoke test | `v0.6.0` |
| 7 | this PLAN, `docs/README.md` | close-out | — |

### Out of scope

* Integration helpers, such as launching the TUI from a CLI subcommand,
  or interactive versus plain output: a later record (MADR, Decision
  Outcome item 7).
* The rest of 0011's findings: 0011 resumes after this PLAN, re-scoped.
* `stream/glamourmd`, still planned under 0010.
* Deleting or moving a tag: never.

## Implementation Steps

### Rules

1. A step starts when the previous one is committed. The owner commits,
   pushes and tags; the agent stages nothing.
2. **Checks** of a step that changes Go: `gofmt -l`; `make
   pre-add-check FILES="…"` (in a scratch clone with the step staged,
   where the step adds or removes a module); `make lint` (three targets,
   every module); per module with `GOWORK=off`, `go test -race`,
   `-shuffle=on -count=2` and `LC_ALL=C`; the tests in workspace mode;
   `go mod tidy -diff`; `make vuln`; `scripts/go-modules.sh --check`;
   `make release-check`; the Windows test host on a copy of the tree. Of
   every step: markdownlint and the link checker on changed Markdown, the
   citation checker on changed records, and the identifier scan of the
   diff.
3. Each mutation runs on a scratch copy, and must fail its named test.
4. Anything this PLAN does not say stops the step for the owner.

### Step 1: records

* This pair.
* **0006-MADR A13** points to 0012 for what it supersedes. §10's
  `command/cli` paragraph and the A1, A10, A11 and A12 headings get an
  in-place note, *(Superseded by 0012-MADR.)*.
* **0010-MADR A3** says the same for §1's two command adapters, §5's
  Cobra, pflag and Kong, and §6's Cobra and Kong rules.
* **A 0011-PLAN revision** pauses that PLAN until this one is complete.
  It also names the findings this one makes moot: F18's example stays
  valid and is kept; F20's front-end section, and A12's text in F1 and
  F20, are dropped.
* **`docs/README.md`:** the rows of 0006, 0010, 0011 and 0012.

**Done when** the owner approves this PLAN.

### Step 2: the retracting versions

* **`command/cobracmd/go.mod`:**
  `// Deprecated: go-tui-lib no longer ships a CLI front end; a program's
  own CLI calls command.Registry.Run (see
  docs/decisions/0012-MADR-bring-your-own-cli.md).` above `module`, and
  `retract [v0.1.0, v0.1.1] // go-tui-lib no longer ships a Cobra front
  end; see 0012-MADR.`
* **`command/kongcmd/go.mod`:** the same, naming Kong.
* **Each package's documentation** gains a `Deprecated:` paragraph, which
  `go doc` and editors show. (*D1: not added. The `go.mod` deprecation and
  retraction carry the notice.*)
* **Checks:** Rule 2, in both modules. `go mod tidy -diff` must stay
  clean, and the gates must accept a `retract` directive.
* **The release:**
  1. The owner commits, runs the disclosure guard through the agent, and
     pushes. With CI green, they tag `command/cobracmd/v0.1.1` and
     `command/kongcmd/v0.1.1` (annotated, `-m` with the tag's name) on
     that commit, and push both tags.
  2. The agent then runs these from a scratch module outside the
     repository, and records what the go command shows:
     * `GOWORK=off go list -m -versions` and `-retracted -versions` of
       each module;
     * `go get <module>@latest`;
     * `go list -m -u` of a consumer pinned to `v0.1.0`.

**Done when** both tags exist and the go command reports each module
deprecated and each version retracted.

### Step 3: the modules removed

* `git rm -r command/cobracmd command/kongcmd`, by the owner from the
  agent's prepared tree. The agent removes the directories in the tree;
  the owner stages the deletions. `go work edit -dropuse` for both.
* **`.golangci.yml`:** the `cobra` and `kong` one-module rules become
  entries of the rule that refuses fang. Their message names this record:
  "go-tui-lib imports no CLI framework; a program brings its own (0012)".
  The `glamour` rule is unchanged.
* **`AGENTS.md`'s Dependencies:** the sentence that keeps Cobra, pflag and
  Kong in their modules becomes a refusal, citing 0012-MADR. (The module
  table changes in Step 5.)
* **Checks:** Rule 2. `scripts/go-modules.sh --check` must pass with
  `go.work` listing `.` alone, and `internal/conformance` must find no
  nested module.
* **Mutation S3-1:** in a scratch copy, a root package importing Cobra,
  and a package under a recreated `command/cobracmd/` importing it. Both
  must fail lint on depguard; with the new entry removed, neither may.

**Done when** the checks are clean and S3-1 is killed.

### Step 4: the root's API change

* **`command/cli/` is deleted:** `cli.go`, `flags.go`, `help.go`, its
  tests, example and goldens.
* **A12's check is deleted:** `command/kongname.go`, the call and the
  documentation paragraph in `command/args.go`, and
  `command/args_test.go`.
* **The tags A12 added are removed:** from `workspace/commands.go` and
  the two `command` test files.
* **Comments that name `command/cli`** or Kong agreement are reworded for
  the program's own CLI, among them the `command` package documentation
  and `SurfaceCLI`'s and `OriginCLI`'s. Found with `grep -rn
  'command/cli\|cli\.Run\|Kong' --include=*.go`.
* **A test, `TestNewTakesJSONRule`** in `command/args_test.go` (new): a
  flag required by its `json` tag without `required:""`, an `omitzero`
  positional without `optional:""`, and a `json` name Kong would spell
  otherwise are each accepted by `New`. Their schemas say what the
  `json` rule says.
* **Exported names removed:** package `command/cli`. No other exported
  name is added or removed (`go doc -short ./command` before and after).
* **Checks:** Rule 2; `make fuzz FUZZTIME=20s`; every earlier 0006
  mutation set whose anchors are in a changed file runs again; the
  goldens are unchanged (`git diff --stat -- '*.golden'` shows only
  `command/cli`'s, deleted).
* **Mutations:**
  * S4-1: `New` calls a Kong check again, which must fail
    `TestNewTakesJSONRule`;
  * S4-2: the `json` rule ignores `omitzero`, which must fail
    `TestNewTakesJSONRule` or the `command` schema tests.
* **Release notes for `v0.6.0`** go in the execution record.

**Done when** the checks are clean, the mutations are killed and the API
is as listed.

### Step 5: the documents

* **`README.md`:** go-tui-lib is a TUI layer a program stacks on its own
  Go CLI. The "Commands" bullet says a program's CLI runs the registry's
  commands. The adapters bullet goes. The current release is `v0.6.0`.
* **`AGENTS.md`:** the module table lists the root and the planned
  `stream/glamourmd`. The Dependencies section refuses CLI frameworks
  (Step 3).
* **`docs/README.md`:** the "I want to…" rows for the shell and for
  Cobra or Kong point to the guide's new section.
* **`docs/architecture.md`:**
  * the packages: ten public, with `command/cli` gone from the graph, the
    table, the imports and the tree;
  * the Modules section: the root, and `stream/glamourmd` planned;
  * "What is not here": the integration helpers, as a later record.
* **`docs/guides/commands.md`:**
  * "Run commands from the shell" becomes "Run commands from your own
    CLI": a handler calls `Registry.Run` with `OriginCLI`, shown for the
    standard `flag` package, Cobra and Kong. Each example is compiled in
    a scratch module outside the repository, which may import those
    frameworks; the library does not.
  * A12's paragraph goes.
  * "Four packages" becomes three.
  * The workspace commands table's note that none is on the CLI surface
    stays.
* **`docs/guides/releasing.md`:**
  * its examples use `stream/glamourmd` where they used
    `command/kongcmd`;
  * a new section, "Retire a module", gives Step 2's procedure: a last
    version with `// Deprecated:` and `retract`, then the removal, and
    tags never deleted.

**Done when** the checks are clean, every example compiles, and a grep
of the living documents for `cobracmd`, `kongcmd`, `command/cli` and A12
finds only historical or retirement mentions.

### Step 6: the release

1. The owner commits Steps 4 and 5, or has committed them. They run the
   disclosure guard through the agent, push, and with CI green tag
   `v0.6.0` (annotated, `-m "v0.6.0"`).
2. The agent runs the consumer smoke test against `v0.6.0` and records
   it. The test program:
   * imports `command`, `workspace` and `when`;
   * registers `workspace.Commands`;
   * registers a command whose argument struct has a `json`-required
     flag and no Kong tags;
   * runs it from a standard-library `flag` handler through
     `Registry.Run` with `OriginCLI`.

   A second check confirms `go get …/command/cli@v0.6.0` fails, because
   the package is gone.

**Done when** the tag exists and the smoke test builds and runs.

### Step 7: close-out

Verification, item by item; this PLAN `complete`, with its row; and the
0011 PLAN's revision that resumes it.

## Verification

* `command/cobracmd/v0.1.1` and `command/kongcmd/v0.1.1` exist. The go
  command reports both modules deprecated and every version retracted,
  as recorded from the proxy.
* `main` holds no `command/cobracmd`, `command/kongcmd` or `command/cli`.
  `go.work` lists `.` alone, and `scripts/go-modules.sh --check` passes.
* depguard refuses Cobra, pflag and Kong in every package (S3-1).
* `v0.6.0`'s exported API is `v0.5.0`'s less package `command/cli`.
  `New[A]` takes the `json` rule (`TestNewTakesJSONRule`, S4-1, S4-2).
* Rule 3's checks are clean at every step, on macOS and the Windows test
  host.
* The commands guide's examples for `flag`, Cobra and Kong compile.
* The consumer smoke test builds and runs against `v0.6.0`.
* The identifier scan finds nothing, and CI is green after each push.

## Rollout and Rollback

* **Rollout:**
  1. The retracting versions are tagged first, while the modules still
     exist on `main`.
  2. Then the directories leave `main`.
  3. Then the root's `v0.6.0`.
* **Rollback:** before a push, each step is one commit to revert. After
  the adapter tags, a retraction cannot be undone by deleting a tag. A
  later version without the `retract` directive would un-retract, which
  this record does not plan. After `v0.6.0`, `command/cli` and A12's
  check come back only in a later release, by a new record.

## Execution Record

### Step 2: the retracting versions

#### Deviations

* **D1 (2026-10-07): no package-doc `Deprecated:` paragraph.**
  * **Found:** with the paragraph in `cobracmd`'s package
    documentation, `make lint` failed in all three targets with
    `command/cobracmd/docs/docs_test.go:13:2: SA1019:
    github.com/maccavelli/go-tui-lib/command/cobracmd is deprecated: …
    (staticcheck)`. The `docs` package's test imports `cobracmd`.
  * **Asked**, with options: the `go.mod` deprecation alone, or the
    package paragraphs as well with `docs_test.go` rewritten to build its
    own Cobra tree. A `//nolint` or a lint exclusion was not offered: it
    would be a workaround.
  * **The owner chose** the `go.mod` deprecation alone, the
    recommendation. `go get`, `go list -m -u` and gopls report a
    module's `// Deprecated:` line and its retraction; the modules leave
    `main` in Step 3.
  * **Files:** the three package paragraphs added in this step were
    removed again. No test changed.

#### What was built

* **`command/cobracmd/go.mod` and `command/kongcmd/go.mod`:**
  * a `// Deprecated:` comment above `module`: "go-tui-lib no longer
    ships a CLI front end. A program's own CLI calls
    command.Registry.Run with command.OriginCLI instead; see
    docs/decisions/0012-MADR-bring-your-own-cli.md.";
  * `retract [v0.1.0, v0.1.1]`, with the reason as a comment ("go-tui-lib
    no longer ships a Cobra front end (0012-MADR)", or Kong).
* No Go file changed (D1).

#### Checks (Rule 2)

* `GOWORK=off go mod tidy -diff`: silent in both modules, the `retract`
  directive kept.
* `make lint`: exit 0, nine "0 issues." (three targets, three modules),
  after D1.
* `make vuln`: exit 0, "No vulnerabilities found." in each module.
* `make release-check`: exit 0, "go-precheck: 147 file(s) clean in 3
  module(s)". Its nested-module check, no `replace` and a release
  version of the root, accepts the `retract` directive.
* **Per module, with `GOWORK=off`:** `go test -race`, `-shuffle=on
  -count=2` and `LC_ALL=C`. The root's 16 packages, `command/cobracmd`'s
  2 and `command/kongcmd`'s 1 were all `ok`. The tests in workspace mode
  were all `ok`.
* `scripts/go-modules.sh --check`: exit 0.
* **Windows test host, on a copy of the tree, go1.27.1 windows/amd64:**
  * `make pre-add-check` exit 0 ("147 file(s) clean in 3 module(s)");
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` in each module exit
    0: 16, 2 and 1 `ok`.
* **This PLAN:** markdownlint, through a renamed copy, finds only the
  asterisk lists the records use; the link check finds 0 broken; the
  citation checker resolves 247 of 247.
* **The identifier scan of the diff:** no match.

#### The release

Left to the owner:

1. Commit this step.
2. Run the disclosure guard through the agent, and push.
3. With CI green, tag `command/cobracmd/v0.1.1` and
   `command/kongcmd/v0.1.1` on that commit (annotated, `-m` with the
   tag's name), and push both tags.

The agent then checks the retraction through the proxy and records it
here. The step is done when the go command reports each module
deprecated and each version retracted.

**After the release (2026-10-07).**

* **Commit:** at the owner's request in that turn ("Commit to main"),
  the agent committed this step on `main` as `bde54d0`, with `git commit
  --no-edit`.
* **Disclosure guard:** run over the outgoing commits, exit 0 and no
  finding.
* **Push and tags:** at the owner's request ("You can to main, then push
  the tags"), the agent pushed `main` (`11d8758..bde54d0`) and waited
  for CI on `bde54d0`, which succeeded in all 11 jobs. It then tagged
  `command/cobracmd/v0.1.1` and `command/kongcmd/v0.1.1`, annotated, on
  `bde54d0`, with each tag's name as its message, and pushed both.

From a scratch module outside the repository, with no `go.work` and
`GOPROXY` at its default:

* **`go list -m -versions`** of each module lists no version.
  **`go list -m -retracted -versions`** lists `v0.1.0 v0.1.1` for each.
* **A consumer pinned to `v0.1.0`** of both gets, from `go list -m -u`:
  `…/command/cobracmd v0.1.0 (retracted) (deprecated)` and
  `…/command/kongcmd v0.1.0 (retracted) (deprecated)`. With `-json`,
  `Retracted` carries the reason, "go-tui-lib no longer ships a Kong
  front end (0012-MADR).", and `Deprecated` the `go.mod` comment.
* **`go get …/command/cobracmd@latest`** failed: "module
  github.com/maccavelli/go-tui-lib@latest found (v0.5.0), but does not
  contain package github.com/maccavelli/go-tui-lib/command/cobracmd".
* **`go get …/command/kongcmd@latest`** resolved a pseudo-version of
  `main`'s head instead, `v0.0.0-20261007070813-bde54d0b41f4`.
  * **Why:** with every release retracted, the go command falls back to
    the default branch's head, and `bde54d0` still holds the module.
  * **Fix:** Step 3 removes the directory from `main`, after which
    `@latest` has no commit with the module to resolve. Step 3 checks
    this again.

Both modules are reported deprecated and every release retracted, so
Step 2 is done.

### Step 3: the modules removed

No deviation.

#### What was built

* **The two directories** were removed from the tree, 154 tracked files,
  and `go work edit -dropuse` dropped both from `go.work`, which now
  lists `.` alone. The deletions are for the owner to stage, or the
  agent when asked.
* **`.golangci.yml`:**
  * `github.com/spf13/cobra`, `github.com/spf13/pflag` and
    `github.com/alecthomas/kong` joined the `forbidden` rule, beside fang,
    each with "go-tui-lib imports no CLI framework; a program brings its
    own (docs/decisions/0012-MADR-bring-your-own-cli.md)";
  * the one-module `cobra` and `kong` rules were removed;
  * the comment above the remaining one-module rule, glamour's, now says
    "a nested module's dependency".
* **`AGENTS.md`'s Dependencies:** a paragraph refuses the three in every
  module, citing 0012-MADR. The nested-module sentence names glamour
  alone. The module table changes in Step 5.

#### Checks (Rule 2)

* **The root, with `GOWORK=off`:** `go test -race`, `-shuffle=on
  -count=2` and `LC_ALL=C` each gave 16 packages `ok`. The tests in
  workspace mode gave 16 `ok`. `go mod tidy -diff` was silent.
* **`make lint`:** exit 0, "0 issues." for the three targets. **`make
  vuln`:** exit 0.
* **`internal/conformance`:** `TestNoPackageOwnsTheTerminal/.` passes,
  and it finds no nested module.
* **A scratch clone with the step staged** (`rsync --delete`, so the
  deletions are mirrored; 154 `D` and 4 `M`):
  `scripts/go-modules.sh --check` exit 0; `make pre-add-check` exit 0
  ("no Go files to check", since the step deletes Go files and changes
  none); `make release-check` exit 0 ("go-precheck: 130 file(s) clean in
  1 module(s)"). In the tree, before staging, the check reports the two
  `go.mod` files as tracked and not in `go.work`, as expected.
* **The Windows test host,** on a copy of the files that exist,
  go1.27.1 windows/amd64:
  * `make pre-add-check` exit 0 ("130 file(s) clean in 1 module(s)");
  * `make lint` and `make vuln` exit 0;
  * `GOWORK=off go test -count=3 -shuffle=on ./...` exit 0, 16 `ok`.

  The first attempt sent `git ls-files`'s list, which still names the
  deleted files, and tar stopped on them. The transfer now sends only
  files that exist.
* **`AGENTS.md`:** markdownlint 0 issues; the link check 0 broken. **The
  identifier scan** of the diff: no match.

#### Mutation (Rule 3)

| Mutation | Check | Result |
| :--- | :--- | :--- |
| S3-1: the three `forbidden` entries removed | golangci-lint on a scratch copy with a root package, and a package under a recreated `command/cobracmd/`, each importing Cobra and Kong | real config: 4 depguard findings, among them `command/cobracmd/leak/leak.go:5:2: import 'github.com/spf13/cobra' is not allowed from list 'forbidden'` under the formerly exempt path; the entries removed: 0 findings. Killed |

#### Left to do after the push

Once this step is on `main`, the agent checks `go get …@latest` of both
modules from a scratch module again. Step 2 found that `kongcmd@latest`
fell back to a pseudo-version of `main`'s head.

**After the push (2026-10-07).**

* **Commit and push:** at the owner's request in that turn ("Commit and
  push"), the agent committed this step on `main` as `7fe52e6` (`git add
  -A`, `git commit --no-edit`). It ran the disclosure guard (exit 0, no
  finding) and pushed `bde54d0..7fe52e6`.
* **CI** on `7fe52e6`: run 37591447004, success in all 5 jobs (the test matrix now holds the root module alone, where it held three).
* **`go list -m <module>@latest`**, from a scratch module:
  * **With `GOPROXY=direct`:** "no matching versions for query
    \"latest\"" for both modules. `main` no longer holds them, and every
    release is retracted, so a fresh resolution finds nothing.
  * **Through `proxy.golang.org`:** both modules still answer with the
    cached pseudo-version of the commit that held them,
    `v0.0.0-20261007070813-bde54d0b41f4`. The proxy caches `@latest`
    answers, so this is the cache's, not the source's.
  * **What stays:** that pseudo-version stays fetchable, as every version
    the proxy has seen does. Its `go.mod` carries the `// Deprecated:`
    comment, so the go command reports it deprecated.
  * **Next check:** the close-out (Step 7) checks the proxy's `@latest`
    again and records what it answers then.

Step 3 is done.

### Step 4: the root's API change

No deviation.

#### What was built

* **`command/cli/` deleted:** `cli.go`, `flags.go`, `help.go`,
  `cli_test.go`, `example_test.go` and its six goldens, 12 files.
* **A12's check deleted:** `command/kongname.go`; in `command/args.go`
  the call and the documentation paragraph, and the `reflect` import it
  needed.
* **`command/args_test.go`** now holds `TestNewTakesJSONRule` alone.
  Three structs are each accepted by `New`, and each schema says what
  the `json` tag says:
  * a flag the `json` tag makes required, with no other tag;
  * an `omitzero` positional with no `optional:""`;
  * a `json` name Kong would spell otherwise (`max_items`).
* **The tags A12 added, removed:** `workspace/commands.go`
  (`zoomArg.Pane`'s `optional:""`, `stateArg.State`'s `required:""`),
  `command/decode_test.go` (four `required:""`) and
  `command/export_test.go` (one `optional:""`). No `required:""` or
  `optional:""` tag remains in the tree.
* **Comments reworded:**
  * `SurfaceCLI` and `OriginCLI` are "the program's own command line",
    the first citing 0012-MADR;
  * `Request.Gate`'s comment says how "a program's own command line
    approves one command, from its --yes flag or a prompt".

  The package documentation's "the shell", and `schema.go`'s "aligned
  with Kong's" tag vocabulary, still hold, and stay.
* **Exported names:** package `command/cli` removed. `go doc -short
  ./command` and `./workspace` are identical before and after.

#### Checks (Rule 2)

* `gofmt -l command workspace`: silent.
* **The root, with `GOWORK=off`:** `go test -race`, `-shuffle=on
  -count=2` and `LC_ALL=C` each gave 15 packages `ok`, one fewer, since
  `command/cli` is gone. The tests in workspace mode gave 15 `ok`. `go
  mod tidy -diff` was silent.
* **`make lint`:** exit 0, "0 issues." for the three targets. **`make
  vuln`:** exit 0.
* **`make fuzz FUZZTIME=20s`:** exit 0, "ran clean" in `./layout`,
  `./when` and `./command`.
* **The goldens:** none changed but `command/cli`'s six, deleted.
* **A scratch clone with the step staged:** `scripts/go-modules.sh
  --check` exit 0; `make pre-add-check` exit 0 ("7 file(s) clean in 1
  module(s)"); `make release-check` exit 0 ("124 file(s) clean in 1
  module(s)").
* **The Windows test host,** on a copy of the files that exist:
  go1.27.1 windows/amd64, `make pre-add-check` exit 0 ("124 file(s) clean
  in 1 module(s)"), `make lint` and `make vuln` exit 0, and `GOWORK=off
  go test -count=3 -shuffle=on ./...` exit 0, 15 `ok`.
* **The identifier scan** of the diff: no match.

#### Mutations (Rule 3)

| Mutation | Test | Failing line |
| :--- | :--- | :--- |
| S4-1: `New` checks Kong agreement again | `TestNewTakesJSONRule` | `a flag required by its json tag: New refused it: command: t.cmd: Name: Kong does not require it` |
| S4-2: the `json` rule ignores `omitzero` | `TestNewTakesJSONRule` | `a positional with omitzero: "pane" required true, want false` |

`command/command.go`, `command/args.go` and `workspace/commands.go` hold
anchors of 0006's Steps 3, 4 and 7, whose sets ran again against this
tree, all killed: Step 3's 11, Step 4's 13 and its layers' 2, and Step
7's 11. Step 8's set anchored in `command/cli`, which is gone; it is
retired with the package. Step 11a's and Steps 10's and 11's sets
anchored in code that 0012 removed, and are retired with it.

#### Release notes for `v0.6.0`

`v0.6.0` makes go-tui-lib a TUI layer a program stacks on its own Go
CLI (0012-MADR).

* **Removed: `command/cli`.** A program's own command line, whether
  `flag`, Cobra, Kong or another, runs a registry command by calling
  `Registry.Run` with `command.OriginCLI`. The commands guide shows how.
* **Changed: `command.New[A]` no longer checks agreement with Kong**
  (`v0.5.0`'s 0006-MADR A12). The `json` tag alone decides a property's
  name and whether it is required, as `SchemaOf` does. A `required:""`,
  `optional:""` or `name:""` tag added for `v0.5.0` is now ignored, and
  may be removed.
* **Retired: `command/cobracmd` and `command/kongcmd`.** They are
  deprecated and retracted at `v0.1.1`, and no longer in the repository.
* **Unchanged:** every other exported name. No module is added or
  removed from the root's `go.mod`.

### Step 5: the documents

No deviation.

#### What was written

* **`README.md`:**
  * the opening says what the library is: a TUI layer for Go
    command-line programs, stacked on the program's own CLI, citing this
    record;
  * the current release is `v0.6.0`;
  * the Commands bullet says a program's own CLI runs the commands
    through `Registry.Run`;
  * the adapters bullet became "No CLI front end, since `v0.6.0`";
  * the "I want to…" row reads "your own CLI".
* **`AGENTS.md`:** the module table lists the root and the planned
  `stream/glamourmd`.
* **`docs/README.md`:** "run the same commands from my own CLI (`flag`,
  Cobra, Kong)" and "use the command registry from Cobra or Kong" point
  to the guide's new section. A row points to this record ("know why
  go-tui-lib ships no CLI front end"). 0010's row speaks of "an adapter
  such as glamour".
* **`docs/architecture.md`:**
  * the opening and "What it is": ten packages, and "No CLI front end";
  * `command/cli` is gone from the package graph, the table, the imports
    note and the tree, and the two adapters from the tree;
  * the Modules section: the two front ends were built, released and
    retired, and `stream/glamourmd` is planned;
  * Dependencies: Cobra, pflag and Kong refused; glamour alone kept to
    one module;
  * "What is not here": a CLI front end, by design, and the integration
    helpers as a later record.
* **`docs/guides/commands.md`:**
  * the title and opening name the program's own command line; three
    packages; amendments A1 to A13, and this record;
  * A12's paragraph is removed;
  * "your CLI's `--yes`" replaces "the shell's", and `Run` is "for your
    CLI";
  * the workspace's commands are "offered everywhere but your CLI
    (`SurfaceCLI`)";
  * "Run commands from the shell" became "Run commands from your own
    CLI": a `yesGate` and a `runCLI` helper calling `Registry.Run` with
    `OriginCLI`, then a handler each for the standard `flag` package,
    Cobra and Kong, the last using the registry's argument struct as its
    grammar. Notes follow on the policy, the CLI surface, a struct
    shared with Kong, and the frameworks being the program's.
* **`docs/guides/releasing.md`:**
  * its examples use `stream/glamourmd` where they used
    `command/kongcmd`;
  * the depguard step names the `glamour` rule alone;
  * the two `git tag -a` lines carry `-m` with the tag's name;
  * a new section, "Retire a module", gives Step 2's procedure, and what
    the go command and the proxy show afterwards.
* **Findings of [0011-PLAN-docs-accuracy-after-v0-5-0.md](0011-PLAN-docs-accuracy-after-v0-5-0.md)
  resolved here,** on lines this step rewrote: F2 (README's opening),
  F19 (the amendment range, now A1 to A13), F26 (`-m` on `git tag -a`),
  and the moot F4, F9's adapters table, F15 and F20. 0011 re-checks them
  when it resumes.

#### Checks

* **The guide's examples, verbatim.** The four Go blocks of "Run commands
  from your own CLI" were extracted from the guide into a scratch module
  outside the repository. Its `go.mod` replaces the root with the
  working tree, and requires Cobra and Kong, which the library does not.
  * The blocks were wrapped in function bodies.
  * `go vet` passed: exit 0.
  * A `Destructive` `session.save` was then run through each framework,
    without and with `--yes`:
    * `flag`, Cobra and Kong without `--yes`: exit 1, refused by the gate
      ("command: refused: session.save: the gate said reject_once" in an
      earlier scratch run);
    * with `--yes`: exit 0, `saved notes`.
* **markdownlint** on the six changed documents: 0 issues. **The
  relative-link check:** 0 broken.
* **A grep of the living documents** for `cobracmd`, `kongcmd`,
  `command/cli` and A12 finds:
  * README's and the architecture note's retirement notes;
  * the index rows' record titles, which name what they decided.

  Nothing else.
* **The identifier scan** of the diff: no match.
* No Go file, `go.mod` or golden changed in this step.

#### The release

`v0.6.0` follows in Step 6: the owner commits this step, the agent runs
the disclosure guard, and with CI green the tag is made on that commit.

### Step 6: the release

* **Commit and push:** at the owner's request ("commit and push"), the
  agent committed Step 5 as `6bf8fad`. It ran the disclosure guard over
  `60ed13e` (Step 4) and `6bf8fad` (exit 0, no finding), and pushed
  `7fe52e6..6bf8fad`.
* **CI** on `6bf8fad`: success in all 5 jobs.
* **The tag:** the owner tagged `v0.6.0` (annotated, message `v0.6.0`)
  on `6bf8fad` and pushed it; `git ls-remote` shows the tag and
  `v0.6.0^{}` at `6bf8fad`.

**The consumer smoke test,** from a scratch module outside the
repository, with no `go.work` and `GOPROXY` at its default:

* **The program:**
  * imports `command`, `workspace`, `when`, `layout`, `theme` and
    `glyph`;
  * registers `workspace.Commands` of a workspace;
  * registers `session.save`: `Destructive`, with `When` set, and an
    argument struct whose one flag the `json` tag makes required and
    which has no Kong tag, which `v0.5.0`'s `New` refused;
  * runs it from its own `flag` handler through `Registry.Run`, with
    `OriginCLI` and a per-request gate that `--yes` opens.
* **The build:** `go mod init`, `go get
  github.com/maccavelli/go-tui-lib@v0.6.0` ("go: downloading
  github.com/maccavelli/go-tui-lib v0.6.0"), `go mod tidy`, `go build`
  and `go vet ./...`, each exit 0. `go.mod` requires
  `github.com/maccavelli/go-tui-lib v0.6.0`.
* **The run:**
  * without `-yes`: `command: refused: session.save: the gate said
    reject_once`, exit 1;
  * with `-yes`: `saved notes (16 commands)`, exit 0. That is the
    workspace's 12 (`layout.use` needs `WithLayouts`), the registry's
    own 3, and the program's.
* **`go get github.com/maccavelli/go-tui-lib/command/cli@v0.6.0`:** "go:
  module github.com/maccavelli/go-tui-lib@v0.6.0 found, but does not
  contain package github.com/maccavelli/go-tui-lib/command/cli".
* **`go list -m all` and `go.sum`** of the consumer name no Cobra, pflag
  or Kong.

The tag exists and the smoke test builds and runs, so Step 6 is done.

### Step 7: close-out

Verification, item by item, on 2026-10-07, against `6bf8fad` (`v0.6.0`).
Every gate was run again rather than read from the steps' records.

* **The retracted adapters.**
  * `command/cobracmd/v0.1.1` and `command/kongcmd/v0.1.1` exist (Step 2).
  * Through the proxy, `go list -m -versions` lists no version of either
    module, and `go list -m -retracted -versions` lists `v0.1.0 v0.1.1`
    for each.
  * A consumer pinned to `v0.1.0` gets `(retracted) (deprecated)` from
    `go list -m -u`.
  * With `GOPROXY=direct`, `@latest` finds "no matching versions".
  * **Observed, outside the repository's reach:** the proxy still answers
    `@latest` with the pseudo-version of `bde54d0`, the last commit that
    held the modules. Its `go.mod` carries the `// Deprecated:` comment,
    so the go command reports it deprecated.
* **`main` holds no `command/cobracmd`, `command/kongcmd` or
  `command/cli`, and `go.work` lists `.` alone.** `git ls-files` of the
  three directories finds 0 files, and `scripts/go-modules.sh --check`
  exits 0.
* **depguard refuses Cobra, pflag and Kong in every package.** S3-1, run
  again at the close, was killed: four depguard findings with the real
  config, in a root package and under a recreated `command/cobracmd/`;
  none with the three entries removed.
* **`v0.6.0`'s exported API is `v0.5.0`'s less package `command/cli`.**
  `git archive` of both tags, `go list ./...` and `go doc -short` of every
  public package: the package lists differ by `./command/cli` alone, and
  the API diff is its eight lines. `New[A]` takes the `json` rule
  (`TestNewTakesJSONRule`; S4-1 and S4-2 killed in Step 4).
* **Rule 2's checks are clean at every step, on macOS and the Windows
  test host.** Each step's record holds its checks. At the close, on
  macOS:
  * `make release-check` exit 0 ("go-precheck: 124 file(s) clean in 1
    module(s)");
  * `make lint` exit 0, three "0 issues."; `make vuln` exit 0;
  * with `GOWORK=off`, `-race`, `-shuffle=on -count=2` and `LC_ALL=C`
    each gave 15 packages `ok`; the tests in workspace mode, 15 `ok`;
    `go mod tidy -diff` was silent.

  On the Windows test host:
  go1.27.1 windows/amd64, `make pre-add-check` exit 0 ("124 file(s) clean
  in 1 module(s)"), `make lint` and `make vuln` exit 0, and `GOWORK=off
  go test -count=3 -shuffle=on ./...` exit 0, 15 `ok`.
* **The commands guide's examples for `flag`, Cobra and Kong compile.**
  Extracted from the guide again at the close, they vet. Each framework
  refuses the destructive command without `--yes` and runs it with it.
* **The consumer smoke test builds and runs against `v0.6.0`** (Step 6).
* **The identifier scan finds nothing, and CI is green after each push.**
  * Each step's diff was scanned: no match.
  * CI concluded `success` on each push's head: `bde54d0`, `7fe52e6` and
    `6bf8fad`, the last twice, once for `main` and once for the `v0.6.0`
    tag.
  * `376106e` and `60ed13e` went out inside those pushes.

Every Verification item holds: this PLAN is `complete`. Its deviation is
D1, the package-doc notice dropped for the `go.mod` deprecation.
[0011-PLAN-docs-accuracy-after-v0-5-0.md](0011-PLAN-docs-accuracy-after-v0-5-0.md)
resumes, re-scoped, by a revision there.

**Open after this PLAN:**

* **Integration helpers for a program's own CLI,** for a later record
  (MADR, Decision Outcome item 7). For example, launching the TUI from a
  subcommand with the caller's streams, or choosing interactive or plain
  output.
* **The proxy's cached `@latest` answer** for the two retired modules.
