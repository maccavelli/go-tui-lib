---
status: complete
date: 2026-10-25
associated-madr: "0021-MADR-termkey-closure-gate.md"
---
# Implement: hold `termkey` to the standard library and `internal/enum`

Associated MADR: [0021-MADR-termkey-closure-gate.md](0021-MADR-termkey-closure-gate.md)

## Goal

`termkey`'s non-test closure holds only the standard library, `termkey`
and `internal/enum`, as
[0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md)
D7 says, and two gates keep it so: `internal/conformance`'s
`TestNarrowClosures` over the whole closure on linux, darwin and windows,
and depguard's `termkeystd` rule over `termkey`'s own imports
(0021-MADR N1 to N4).

## Scope

### Facts the steps rest on (read 2026-10-25, at `81177eb`)

| Fact | Where |
| :--- | :--- |
| `termkey`'s closure outside the standard library: `termkey`, `internal/enum`; `internal/enum`'s: itself | `GOWORK=off go list -deps ./termkey`, `./internal/enum` |
| `listDeps` prints `ImportPath`, an error and `Imports`, tab-separated; it does not print `Standard` | `internal/conformance/charmfree_test.go:81-111` |
| `charmChains` is a breadth-first walk that stops at a package `charm` is true for | `internal/conformance/charmfree_test.go:113-149` |
| `cores` lists twelve directories, `termkey` among them | `internal/conformance/charmfree_test.go:18-31` |
| depguard's rules are `lax` deny lists today; `charmfree`'s comment asks that it and `cores` be kept the same | `.golangci.yml:32-137` |
| a `strict` rule allowing `$gostd` and `internal/enum` passes `termkey` and refuses `x/ansi` in it | 0021-MADR, the probe |
| the architecture doc describes the closure test and `charmfree` | `docs/architecture.md:90-96`, `:286`, `:338-341` |
| `AGENTS.md`'s Dependencies section names each depguard rule that confines an import | `AGENTS.md`, "Dependencies" |

### In scope

* `internal/conformance`: `listDeps` reads `.Standard`; the walk takes its
  stopping predicate; `narrow` and `TestNarrowClosures`; a synthetic test
  of the walk.
* `.golangci.yml`: the `termkeystd` rule.
* `AGENTS.md`, "Dependencies": one sentence on `termkey`'s rule.
* `docs/architecture.md`: the Tooling bullet and the depguard list.
* `docs/README.md`: this pair's rows.

### Out of scope

* `safetext`'s rule of 0020-MADR D9 (0021-MADR N5).
* Any change to `termkey`'s code or API: the step adds checks only, so
  `make apicheck` stays clean and nothing is released.

## Rules for every step

1. **The gate.** No step starts until the owner approves this PLAN.
2. **Gates on every host** at the end of each step that changes a file
   outside `docs/`: `python3 -m tools.verify gates` over the step's files
   on macOS, and `python3 -m tools.verify remote <host> -- <files>` on the
   Linux and Windows hosts, each summary line in the record.
3. **Every new check is seen failing** on a scratch copy first
   (`python3 -m tools.verify mutate`), its failure line in the record. A
   mutation that survives, does not compile, or crashes by itself is
   replaced, and both are recorded.
4. **The identifier scan,** `python3 -m tools.verify scan`, is clean at
   each handoff, and nothing is staged: the owner commits each step.

## Implementation Steps

### Step 1: records

**Files:** `docs/decisions/0021-MADR-termkey-closure-gate.md`,
`docs/decisions/0021-PLAN-termkey-closure-gate.md`, `docs/README.md` (two
rows, the MADR `proposed` and the PLAN `proposed`).

This is AGENTS.md's bootstrap exception: records only, no source.

### Step 2: the gate

**`internal/conformance/charmfree_test.go`:**

* `depGraph` gains `standard map[string]bool`, and `listDeps`'s format
  prints `{{.Standard}}` as a fourth field, so one `go list` run serves
  both tests.
* `charmChains(g, from)` becomes `chains(g, from, stop func(string) bool)`:
  the same breadth-first walk, stopping at and returning each package
  `stop` is true for. `TestCoresAreCharmFree` calls it with `charm`, and
  its failures read as before.
* `narrow`, a slice of `{core string; allow []string; record string}`,
  each directory relative to the repository root:
  `{"termkey", []string{"internal/enum"}, "0020-MADR D7, 0021-MADR"}`.
  Its comment says that a core joins by the record that states its rule,
  with a depguard rule beside it (0021-MADR N4).
* `TestNarrowClosures`: for each of `charmGOOS`, `listDeps` over the
  narrow cores; then for each entry:
  * a core `cores` does not list fails ("narrow lists %s, which cores does
    not");
  * a core `go list` does not find fails as `cores`' stale case does;
  * every package `chains` reaches with `stop` true for a package that is
    not in the standard library, not the core and not in its allowance
    fails, naming the core, the target, the package and the chain, and
    the record;
  * an allowance the closure does not hold fails ("narrow allows %s for
    %s, which its closure does not hold").
* `TestNarrowChains`: a synthetic `depGraph` of a core, an allowed
  package, a standard one and a stray one, each reached by one chain:
  `chains` with the narrow predicate returns the stray package's chain
  alone, and the one through the allowed package to a stray one too.

**`.golangci.yml`:** `termkeystd`, after `charmfree`:

```yaml
        # termkey imports the standard library and internal/enum, and
        # nothing else (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md
        # D7; docs/decisions/0021-MADR-termkey-closure-gate.md). internal/conformance's
        # TestNarrowClosures checks the whole closure; this rule fails a
        # direct import at lint. Keep its allow list and narrow's the same.
        termkeystd:
          list-mode: strict
          files:
            - "**/termkey/*.go"
            - "!$test"
          allow:
            - $gostd
            - github.com/maccavelli/go-tui-lib/v2/internal/enum
```

**`AGENTS.md`, "Dependencies":** after the `x/term` sentence: "`termkey`
imports the standard library and `internal/enum` and nothing else: the
depguard rule `termkeystd` refuses any other import, and
`internal/conformance`'s `TestNarrowClosures` checks its closure
(`docs/decisions/0021-MADR-termkey-closure-gate.md`)."

**`docs/architecture.md`:** the Tooling bullet on `TestCoresAreCharmFree`
gains `TestNarrowClosures` and `narrow`; the depguard list gains
`termkeystd`.

**Checks:**
* `go test ./internal/conformance/` passes, with `TestNarrowClosures` and
  `TestNarrowChains` among the tests run (`-v`, their `PASS` lines);
* `make lint`: `0 issues.` for linux, darwin and windows;
* `make apicheck`: clean, no incompatible change;
* `markdownlint-cli2` over `AGENTS.md`, `architecture.md` and the index.

**Mutations** (Rule 3):
1. `termkey/key.go` imports and uses `x/ansi`: `TestNarrowClosures`
   fails, naming `x/ansi`; the same copy's `TestCoresAreCharmFree` passes,
   which the record quotes as the gap closed;
2. the same, seen by `termkeystd` (`golangci-lint run ./termkey/`); and by
   `charmfree`, which passes;
3. `internal/enum` imports and uses `x/ansi`: `TestNarrowClosures` fails
   with the chain `termkey → internal/enum → github.com/charmbracelet/x/ansi`;
   `golangci-lint run ./termkey/` passes, which the record quotes as why
   the closure test is needed;
4. `termkey/key.go` imports and uses `glyph`: `TestNarrowClosures` and
   `termkeystd` fail;
5. `narrow` names `termkeys`: `TestNarrowClosures` fails, as not in
   `cores`;
6. `narrow`'s allowance for `termkey` adds `glyph`: `TestNarrowClosures`
   fails, as an allowance the closure does not hold;
7. the narrow predicate takes every package for standard:
   `TestNarrowChains` fails;
8. `chains` walks on past a package `stop` is true for: `TestNarrowChains`
   fails, since the walk then returns the packages beneath the stray one
   too.

**Gates** on every host (Rule 2) over the step's files; the identifier
scan; the record.

## Verification

* `TestNarrowClosures` and `TestNarrowChains` pass on macOS, Linux and
  Windows, in the gates of each host.
* `make lint` passes with `termkeystd`.
* Each mutation of Step 2 is killed, with its failure line in the record,
  and mutations 1 and 3 also show which gate would have missed the import.
* `make apicheck` is clean: no exported name changes.

## Rollout and Rollback

* **Rollout.** The step lands on `main` as one commit, the owner's, after
  CI is green. It changes no exported API and no package's code, so it
  needs no tag; the next release carries it.
* **Rollback.** Reverting the step's commit removes both gates and leaves
  `termkey` as it was. A future need of `termkey`'s for another import
  is not a rollback: it needs a record that changes 0020-MADR D7, and that
  record changes `narrow` and `termkeystd` together.

## Execution Record

### Step 1: records (2026-10-25)

* The pair and its index rows, committed by the owner as `8d2cb67`.
* **Approval:** the owner, 2026-10-25: "committed proceed". The MADR
  is `accepted`; this PLAN is `in-progress`.

### Step 2: the gate (2026-10-25)

* **`internal/conformance/charmfree_test.go`,** as the step says:
  * `depGraph` gains `standard`, which `listDeps` fills from a fourth
    field, `{{.Standard}}`;
  * `charmChains` is `chains(g, from, stop)`, and `TestCoresAreCharmFree`
    calls it with `charm`, its messages unchanged;
  * `narrowCore`, `narrow` (one entry: `termkey`, allowing
    `internal/enum`, by "0020-MADR D7, 0021-MADR N1"), `strayFrom`, the
    narrow predicate, and `closure`;
  * `TestNarrowClosures` and `TestNarrowChains`.
* **`.golangci.yml`:** `termkeystd`, after `charmfree`, with its comment,
  as the step gives it.
* **`AGENTS.md`,** "Dependencies": the sentence on `termkey`'s rule.
  **`docs/architecture.md`:** a bullet beside `charmfree`'s in the
  depguard list, and a Tooling bullet on `TestNarrowClosures`.
  **`docs/README.md`:** the rows' statuses.
* **Checks:**
  * `GOWORK=off go test -count=1 -v -run
    'TestNarrow|TestCoresAreCharmFree|TestCharmPredicate'
    ./internal/conformance/`: `PASS` for `TestCharmPredicate`,
    `TestCoresAreCharmFree` (linux, darwin, windows),
    `TestNarrowClosures` (linux, darwin, windows) and `TestNarrowChains`;
  * `make lint`: `go fix -diff` empty, and `0 issues.` for linux, darwin
    and windows, `termkeystd` among the rules;
  * `markdownlint-cli2` over `AGENTS.md`, `architecture.md` and the index:
    "0 issues".
* **Mutations,** each on a fresh copy. Twelve that a check must kill, all
  killed by the check named; and two that a check must let through, both
  let through, which show what each gate alone would miss:

  | Mutation | Seen by |
  | :--- | :--- |
  | 21.1 termkey imports x/ansi (TestNarrowClosures) | `charmfree_test.go:256: termkey (GOOS=linux) links github.com/charmbracelet/x/ansi, which 0020-MADR D7, 0021-MADR N1 does not allow: termkey → github.com/charmbracelet/x/a…` |
  | 21.1p the same: TestCoresAreCharmFree passes, the gap the step closes | `SURVIVED (exit 0)` |
  | 21.2 the same, refused by depguard's termkeystd, and not by charmfree | `termkey/key.go:6:2: import 'github.com/charmbracelet/x/ansi' is not allowed from list 'termkeystd' (depguard)`, the one depguard issue |
  | 21.3 internal/enum imports x/ansi (TestNarrowClosures) | `charmfree_test.go:256: termkey (GOOS=linux) links github.com/charmbracelet/x/ansi, which 0020-MADR D7, 0021-MADR N1 does not allow: termkey → internal/enum → github.com/c…` |
  | 21.3p the same: golangci-lint over termkey passes, why the closure test is needed | `0 issues.`, `SURVIVED (exit 0)` |
  | 21.4 termkey imports glyph (TestNarrowClosures) | `charmfree_test.go:256: termkey (GOOS=linux) links github.com/maccavelli/go-tui-lib/v2/glyph, which 0020-MADR D7, 0021-MADR N1 does not allow: termkey → glyph` |
  | 21.4b the same, refused by depguard's termkeystd | `termkey/key.go:6:2: import 'github.com/maccavelli/go-tui-lib/v2/glyph' is not allowed from list 'termkeystd' (depguard)` |
  | 21.5 narrow names termkeys (TestNarrowClosures) | `charmfree_test.go:234: narrow lists termkeys, which cores does not` |
  | 21.6 narrow allows glyph too (TestNarrowClosures) | `charmfree_test.go:262: narrow allows glyph for termkey, which its closure does not hold (GOOS=linux)` |
  | 21.7 the narrow predicate takes every package for standard (TestNarrowChains) | `charmfree_test.go:290: chains = [], want [["m/core" "m/bad"] ["m/core" "m/ok" "x/mod"]]` |
  | 21.8 the walk goes on past a stopping package (TestNarrowChains) | `charmfree_test.go:289: chains = [["m/core" "m/bad"] ["m/core" "m/bad" "m/deep"] ["m/core" "m/ok" "x/mod"]], want [["m/core" "m/bad"] ["m/core" "m/ok" "x/mod"]]` |
  | 21.9 closure holds only its start (TestNarrowChains) | `charmfree_test.go:293: closure = map[m/core:true], want all seven packages` |
  | 21.9b the same, seen by TestNarrowClosures as a stale allowance | `charmfree_test.go:262: narrow allows internal/enum for termkey, which its closure does not hold (GOOS=linux)` |
  | 21.10 listDeps reads no package as standard (TestNarrowClosures) | `charmfree_test.go:256: termkey (GOOS=linux) links bytes, which 0020-MADR D7, 0021-MADR N1 does not allow: termkey → bytes` |

  21.9, 21.9b and 21.10 are beyond the step's eight, so that `closure` and
  the `.Standard` field, which the step adds, are seen failing too. The
  depguard lines of 21.2 and 21.4b were read from their kept copies; each
  copy also reports its mutation's own `unused` placeholder.
* **Gates,** `python3 -m tools.verify gates` over the step's seven files,
  nothing skipped: macOS `gates: 19 passed, 0 failed, 0 skipped in 575s`; the Linux host
  `gates: 19 passed, 0 failed, 0 skipped in 1001s`; the Windows host
  `gates: 19 passed, 0 failed, 0 skipped in 1827s`. `release-check` is
  "309 file(s) clean" on each, with `apicheck` (no incompatible change)
  and the examples gate.
* **Identifiers:** `python3 -m tools.verify scan`: `scan: clean (5 pattern(s))`.
* **CI on `81177eb`,** 0020-PLAN's close-out, passed (`modules`, `gates`,
  and `test` on three systems): the first run whose API gate took
  `v2.0.0` as its base, with the emptied allow-list.
* **Not done here:** nothing of this PLAN's. `safetext`'s rule stays out
  of scope (0021-MADR N5). Nothing is tagged: the step changes no exported
  name.
* **The PLAN is `complete`:** each item of Verification is met, as
  above. The owner commits the step.
