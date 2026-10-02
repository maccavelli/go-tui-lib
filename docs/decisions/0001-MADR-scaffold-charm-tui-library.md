---
status: accepted
date: 2026-10-01
decision-makers: go-tui-lib maintainers
consulted: go-core-lib repository conventions; ocp-login (the working example for the Charm v2 stack)
informed: fleet programs with a terminal UI (ocp-login, ocp-login-macos, mcp-server-magicdev, mcp-server-magictools, mcp-server-recall, mcp-server-socratic-thinker)
---
# Scaffold go-tui-lib as a Go 1.27.1 Charm v2 library to the go-core-lib standard, with honest gates until the first package lands

## Context and Problem Statement

`go-tui-lib` (`https://github.com/maccavelli/go-tui-lib`) is a public
repository whose only commit, `ff5c504`, adds a `README.md`. That README
states the purpose: reusable Go code for terminal interfaces on Bubble Tea,
Lip Gloss and the rest of the Charm stack, extracted from fleet programs
"while those projects are worked through", so that each interaction pattern
has one shared implementation.

go-core-lib already plans the first consumer. Its
`docs/decisions/0004-MADR-evolve-selfupdate-api-and-tui-support.md` §1 puts
the Bubble Tea adapter for `selfupdate` in `go-tui-lib/updatetea`, "Charm,
in go-tui-lib only", decided by "a record in go-tui-lib". Its Phase 2
Confirmation asks for an example Bubble Tea program here that drives a real
update.

On 2026-10-01 the owner asked for the scaffolding first: "prepare the tooling
and environment for the project to begin. get the scaffolding in place before
taking other action. ensure the repo adheres to standards and our best
practices". They then added: "use the ocp-login charm/lipgloss as the working
example". This record decides the scaffold. Every package, including
`updatetea` and any extraction from ocp-login, is a later record.

Evidence gathered for this record (read-only, 2026-10-01):

* **The repository.**
  * `main` tracks `origin/main` at `ff5c504`. The hooks path resolves to
    `~/.global-git-hooks`.
  * There is no local `user.name` / `user.email`, unlike go-core-lib. The
    first commit's author differs from the identity go-core-lib sets
    locally, which is also the global one.
  * GitHub reports the repository public, with default branch `main` and no
    licence.
* **The closest precedent is go-core-lib itself.** Its scaffold is
  `docs/decisions/0001-MADR-scaffold-shared-go-library.md`, and the
  standard has moved on since. At `f200c51` it carries:
  * `AGENTS.md` and the agent pointers;
  * `.gitignore`, `.gitattributes` and the fleet `.markdownlint-cli2.jsonc`;
  * `.golangci.yml`: golangci-lint v2, 20 linters, `revive`'s `exported`,
    `package-comments` and `var-naming` in place of `golint`, and errcheck
    `check-blank`;
  * a `Makefile` that lints for linux, darwin and windows with
    `CGO_ENABLED=0`;
  * `scripts/go-precheck.sh`, a GitHub Actions CI on three operating
    systems with SHA-pinned actions, golangci-lint v2.14.0 and govulncheck
    v1.7.0, and an Apache-2.0 `LICENSE`.
* **The working example is ocp-login**, on the Charm v2 stack. ocp-login is
  an org-internal program, and this repository is public, so it is described
  here by its design patterns only, never by its files.
  * **Stack.** Charm v2 under `charm.land` (Bubble Tea, Lip Gloss,
    Bubbles), with `colorprofile` and `x/ansi`, and fang for its command
    layer.
  * **v1 is banned by a test.** A test refuses the Charm v1 import paths.
  * **Its generic patterns** are:
    * a layered theme: raw palette, semantic roles, built styles;
    * a three-state background (light, dark, unknown), with an unknown
      palette that is legible on both;
    * colour-profile resolution that honours `NO_COLOR` and a non-TTY;
    * width-capped measurement, with wrap and truncate on `x/ansi`;
    * a glyph table with a Unicode form and an ASCII twin;
    * rendering primitives, and a help footer that drops whole hints;
    * a frame composed on a Lip Gloss canvas;
    * spinners and progress bars;
    * prompts, including a masked secret field that can be wiped;
    * a non-interactive fallback for every interactive path.
  * **Its coupling is in names and strings, not types.** Its UI layer
    imports no domain package.
  * **How it renders.**
    * Short-lived programs render inline (its `0004`).
    * A long-lived region uses the alternate screen and replays its
      record to the main screen on exit (`0011`).
    * Its update command takes no alternate screen (`0024`).
    * Bubble Tea's own signal handler is off. The command owns `ctrl+c`
      and exits 130.
  * **How it tests.**
    * Hand-written goldens over colour × charset × width.
    * Guard tests for ASCII degradation, glyph ownership, and contrast
      after ANSI-256 conversion.
    * A PTY harness that replays measured terminal profiles.
    * A C-locale test job in CI.
  * **Its toolchain is Go 1.26.6,** below the fleet's 1.27.1.
* **Other fleet TUI programs are on Charm v1.** These are
  mcp-server-magicdev, mcp-server-magictools, mcp-server-recall and
  mcp-server-socratic-thinker: `github.com/charmbracelet/bubbletea` v1.3.10,
  `lipgloss` v1.1.0, Go 1.26.6.
* **Current releases** (module proxy, 2026-10-01):
  * `charm.land/bubbletea/v2` v2.0.10, `charm.land/lipgloss/v2` v2.0.6,
    `charm.land/bubbles/v2` v2.2.1;
  * `github.com/charmbracelet/colorprofile` v0.4.3,
    `github.com/charmbracelet/x/ansi` v0.11.8;
  * golangci-lint v2.14.0, govulncheck v1.8.0. The installed toolchain is
    `go1.27.1`.
* **Every gate fails on a module with no packages.** go-core-lib's 0001
  measured this on Go 1.27.1: `go test` and `go vet` exit 1,
  golangci-lint exits 5, govulncheck exits 2, and `go mod tidy -diff` and
  `go mod verify` exit 0. This record's PLAN measures it again here.

## Decision Drivers

* **One standard across the fleet's libraries,** so that an agent or
  maintainer moving between go-core-lib and go-tui-lib finds the same
  files, rules and commands.
* **One Charm generation.** Mixing v1 and v2 types in one program does not
  compile across the boundary, and the fleet's newest TUI work, ocp-login,
  is on v2.
* **A dependency position just above go-core-lib.** `updatetea` needs
  `selfupdate`, and go-core-lib must never import Charm.
* **Gates that report the truth.** A check that passes because it was
  taught to pass on nothing is not a check.
* **The TUI disciplines ocp-login paid to learn,** fixed as repository rules
  before the first package rather than rediscovered in each one.
* **The first package records decide only their own content.**

## Considered Options

* **A. The go-core-lib scaffold, adapted for a Charm v2 library, on an empty module; gates stay honest and fail until the first package; ocp-login's generic TUI disciplines become repository conventions.**
* **B. As A, but leave TUI conventions to each package's record.**
* **C. Copy ocp-login's tooling** (its GitLab CI, mutation lane, coverage floor, skip ledger, PTY harness) as the scaffold.
* **D. No separate scaffold.** Scaffold inside the first package's pair (`updatetea`, or an extraction from ocp-login).

## Decision Outcome

Chosen option: "A", because:

* it meets the fleet standard now and invents no code and no guard;
* the one red signal (no package yet) points at the work that removes it;
* it fixes the cross-package TUI rules once, where every later package
  inherits them.

### 1. Identity and scope

* **Module path:** `github.com/maccavelli/go-tui-lib`.
* **A library only.** No binary, no `cmd/`, no packaging or release targets.
  Runnable examples are `Example` functions or test programs.
* **Scope:** reusable terminal-UI code on the Charm v2 stack, such as
  themes, layout, glyph tables, components, prompts, rendering primitives,
  test helpers, and adapters that bind a go-core-lib package to Bubble Tea
  (`updatetea`). UI-free code belongs in go-core-lib.

### 2. Layout

* **One top-level directory per capability,** with the package named after
  the directory. There is no root package.
* **Shared unexported helpers** go under `internal/`.
* **One documentation tree:** `README.md` → `docs/README.md`,
  `docs/architecture.md`, `docs/decisions/`, `docs/reports/`,
  `docs/guides/`. Nothing in the repository is adjacent to its purpose.
  `docs/reports/` and `docs/guides/` are created by their first document.
* **Records** use one repository-wide `NNNN` sequence, starting at `0001`
  (this record).

### 3. Toolchain and dependencies

* **`go.mod` says `go 1.27.1`,** with no `toolchain` line, as go-core-lib.
* **The scaffold adds no requirement and no `go.sum`.** A module is required
  only in the commit that adds its first import, and only when a MADR here
  names it.
* **The stack this record names,** so that a package record need not decide
  it again:
  * `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2` and
    `charm.land/bubbles/v2`;
  * `github.com/charmbracelet/colorprofile` and
    `github.com/charmbracelet/x/ansi`;
  * `github.com/maccavelli/go-core-lib`.

  Each enters at its newest release when its first import lands. Any other
  module, including fang, cobra, a PTY library or a colour library, needs
  its own record.
* **Never imported:**
  * the Charm v1 paths `github.com/charmbracelet/bubbletea`,
    `github.com/charmbracelet/lipgloss` and
    `github.com/charmbracelet/bubbles`;
  * `github.com/maccavelli/mcplib`;
  * `github.com/modelcontextprotocol/go-sdk`;
  * `github.com/maccavelli/go-llmprovider-sdk`.
* **`.golangci.yml` enforces the "never" list with `depguard`,** so a
  forbidden import fails `make lint`, the pre-add gate and CI. A test that
  scans sources (ocp-login's approach) needs a package to live in; a lint
  rule does not.
* **`go mod tidy -diff` is clean at every commit.**

### 4. Files taken from go-core-lib

These are taken with go-core-lib's wording at `f200c51`, changed only where
they name go-core-lib, `selfupdate` or its records:

* `AGENTS.md`;
* `.claude/.gitignore`, `.claude/rules/madr-and-plan-skill.md`,
  `.grok/rules/madr-plan-before-mutating-work.md`, `.opencode/rules.md`,
  `opencode.json`;
* `.gitignore` and `.markdownlint-cli2.jsonc`, verbatim;
* `LICENSE`, verbatim (§7);
* `.golangci.yml`, with `depguard` added (§3);
* `Makefile`, without `apicheck` and `fuzz`;
* `scripts/go-precheck.sh`, with its provenance comment re-pointed at this
  record;
* `.gitattributes`, keeping `* text=auto eol=lf` and the binary patterns,
  with the `selfupdate` fixture rule dropped. Golden files hold escape
  sequences and exact column widths, so they must never be converted;
* `.github/workflows/ci.yml`, keeping:
  * the three operating systems and the action SHAs;
  * `go test`, `-race` on Linux and macOS, and `-shuffle` on Linux;
  * cross `go vet` for freebsd, openbsd and linux/386;
  * `go vet`, `gofmt`, `go mod tidy -diff` and golangci-lint v2.14.0;
  * govulncheck;
  * shellcheck v0.11.0 (SHA-256 pinned), markdownlint-cli2 0.23.2 and
    actionlint v1.7.12.

  It drops every `selfupdate`-specific step: fuzz, the API gate, the
  release fixtures, the release guard, the tag rule and the workflow
  contract.

### 5. Deliberate differences from go-core-lib

* **`depguard`** in `.golangci.yml` (§3).
* **An `LC_ALL=C` test leg** on Linux, from ocp-login's C-locale CI job:
  terminal code that assumes a UTF-8 locale fails there.
* **No `apicheck` yet.** It compares with the newest `v1.*` tag, and there
  is none. The record that makes the first release adds it, as go-core-lib's
  `0004-PLAN-v1-1-0-core-api.md` did.
* **No `scripts/check-workflows.sh`.** Its test is written against
  go-core-lib's reusable release workflow, and this repository has none.
  actionlint already flags untrusted context interpolated into `run:`.
* **govulncheck at v1.8.0,** the current release (Q3). The owner made it
  the version for every Go host and for go-core-lib, under go-core-lib
  `docs/decisions/0007-MADR-adopt-govulncheck-v1-8.md`. The install hints
  copied from go-core-lib therefore name `@v1.8.0`.

### 6. Repository conventions for TUI packages

`AGENTS.md` carries these as rules every package follows. Each is a generic
lesson from ocp-login. Where a package needs an exception, its record says
so.

1. **Output goes where the caller says.** A package never writes to
   `os.Stdout` or `os.Stderr` itself. It renders to a string, an
   `io.Writer` the caller passes, or a Bubble Tea view.
2. **The caller owns the screen and the signals.** A component never sets
   the alternate screen and never installs a signal handler. Whether a
   program runs inline (ocp-login `0004`, `0024`) or in an alternate-screen
   region (`0011`) is the program's choice. `ctrl+c` reaches the program
   as a key or a cancel.
3. **Every glyph comes from a glyph table** with a Unicode form and an ASCII
   twin, including the glyphs Charm components draw by default (ocp-login
   `0033`).
4. **Colour degrades, and never carries meaning alone.** Output must stay
   correct under `NO_COLOR`, an ASCII colour profile and an unknown
   background (ocp-login `0014`, `0027`, `0037`).
5. **Width is the caller's, and is measured in cells.** Wrapping and
   truncation use `x/ansi`, and nothing assumes a terminal size.
6. **Rendering is tested across the matrix** {colour, no colour} × {UTF-8,
   ASCII} × at least two widths, with golden files under the package's
   `testdata/`. The `LC_ALL=C` leg runs those tests.

### 7. Licence

* **`LICENSE` is the Apache License 2.0** (owner question Q1). It is
  go-core-lib's file, byte for byte, which is `magic-cli-remote`'s. The
  repository is public and has no licence today.

### 8. Push and identity

* **The scaffold is not pushed on its own** (owner question Q2). Its
  commits stay local and are pushed with the first package, when the owner
  asks. CI therefore never runs on the empty module. This is go-core-lib's
  0001 §6, applied again.
* **The local `user.name` and `user.email` are set** to the identity the
  owner named (Q4), which is also the one go-core-lib sets locally.
  AGENTS.md then states, as go-core-lib's does, that the repository sets
  them and that agents do not override them.

### 9. Not added

* **No `.github/dependabot.yml`.** This is the fleet decision; SHA pins are
  updated by hand, as a recorded act.
* **No tag or version.** The first release is its package record's to
  decide.
* **No package, no `testdata/`, and nothing from ocp-login's tooling beyond
  §5's locale leg.** Its mutation lane, coverage floor, skip ledger, PTY
  harness and terminal-profile corpus are candidates for the record of the
  first package that needs them.
* **No change in go-core-lib or ocp-login.** The govulncheck move in
  go-core-lib is that repository's own record, 0007.

### Consequences

* Good, because every fleet convention an agent relies on is present, in
  the same files and words as go-core-lib.
* Good, because a v1 Charm import, or an import of a library this module
  must sit beside or below, fails lint before review.
* Good, because the TUI rules that cost ocp-login several records are
  decided once.
* Neutral, because the stack is named but not yet required. The module
  stays requirement-free until code imports it.
* Bad, because consumers must be on Go 1.27.1 to import this module, and
  ocp-login and the four Charm v1 programs are on 1.26.6. Importing
  go-core-lib already requires 1.27.1, so `updatetea`'s consumers face that
  anyway.
* Bad, because the Charm v1 programs must move to v2 before they can share
  code from here.
* Bad, because the scaffold commits wait unpushed until the first package.
  Until then, `make test`, `make vet`, `make lint` and `make vuln` fail with
  "no packages".

### Confirmation

* Each file in §4 is compared with its go-core-lib source:
  * the verbatim files and `LICENSE` with `cmp`;
  * the rest with `diff`, showing only the changes §4 and §5 allow.
* `go mod verify` and `go mod tidy -diff` exit 0, `gofmt -l .` is empty,
  and markdownlint-cli2 is clean over the non-record Markdown.
* The empty-module failures are reproduced on the committed tree and
  recorded in the PLAN.
* On a scratch clone with a planted package, `make pre-add-check` and
  `make lint` fail on:
  * an undocumented exported function;
  * an unformatted file;
  * an import of `github.com/charmbracelet/lipgloss`;
  * an import of `github.com/maccavelli/mcplib`.

  On the same clone with a clean package that imports
  `charm.land/lipgloss/v2`, every gate passes.
* The identifier scan finds no hostname, account name, org-internal path or
  real-machine absolute path in the committed files. ocp-login is cited by
  name and record number only.

## Pros and Cons of the Options

### A. go-core-lib scaffold plus TUI conventions

* Good, because it is the standard the sibling library already proves in
  CI.
* Good, because the conventions reach the first package before its first
  line.
* Neutral, because the conventions are written from one program's
  experience. Each is a generic rule, and a package record can amend one.
* Bad, because CI would be red on the empty module. §8 keeps it from
  running there.

### B. Conventions per package

* Good, because each rule is decided next to the code it shapes.
* Bad, because the second package can silently differ from the first: two
  glyph tables, two width rules, two ideas of who owns `ctrl+c`.

### C. ocp-login's tooling

* Good, because it is the most thorough TUI test regime in the fleet.
* Bad, because it is GitLab CI for an application, with a coverage floor,
  mutation lane and skip ledger built for one program's history.
* Bad, because its PTY harness and terminal corpus need code to test. Here
  they would be scaffolding for nothing.

### D. Scaffold inside the first package's pair

* Good, because CI is never red on a pushed commit.
* Bad, because one record then decides two unrelated things: repository
  standards, and a package's API. The owner asked for the scaffold first.

## Owner questions

*Answered 2026-10-01: "1. apache-2.0 2. hold locally until the first
package. 3. use 1.8.0 across the entire go environment … pin it in
go-core-lib for parity. 4. [the owner's name and email] for identity."*
Each answer is the recommendation, and Q3 is widened to every Go host and
to go-core-lib. The email is not repeated in this record.

* **Q1. Licence.** Recommended: Apache-2.0, go-core-lib's file. The
  alternative is no licence, as go-llmprovider-sdk has. A public repository
  without one grants no reuse rights.
* **Q2. Push.** Recommended: hold the scaffold until the first package, as
  go-core-lib did. The alternative is to push now and accept a red CI until
  then.
* **Q3. govulncheck.** Recommended: v1.8.0, the current release, with
  go-core-lib's v1.7.0 left to a go-core-lib record. The alternative is
  v1.7.0 for parity.
* **Q4. Local identity.** Recommended: set it to go-core-lib's local
  identity. The alternative is to inherit the global configuration.

## More Information

* go-core-lib `docs/decisions/0001-MADR-scaffold-shared-go-library.md`: the
  scaffold this one adapts.
* go-core-lib `docs/decisions/0004-MADR-evolve-selfupdate-api-and-tui-support.md`
  §1 and §4: `updatetea`, and the Phase 2 Confirmation it owes.
* ocp-login records `0002`, `0003`, `0004`, `0005`, `0011`, `0014`, `0024`,
  `0027`, `0033` and `0037`: the TUI disciplines in §6.
* magic-cli-remote
  `docs/decisions/0169-MADR-standardize-toolchains-on-current-supported-advisory-free-releases.md`:
  the fleet toolchain rule behind Go 1.27.1 and §5's govulncheck.
