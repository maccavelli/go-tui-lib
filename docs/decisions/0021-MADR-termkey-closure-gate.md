---
status: accepted
date: 2026-10-25
decision-makers: the repository owner
consulted: "0020-MADR-charm-free-cores-glyphs-and-safetext.md D7, D9 and D11; 0020-PLAN-charm-free-cores-glyphs-and-safetext.md, the Step 11 record; this repository at 81177eb; golangci-lint 2.14.0 and its depguard"
informed: gobble-cli
---
# Hold `termkey` to the standard library and `internal/enum` with a closure test and a strict depguard rule

## Context and Problem Statement

[0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md)
D7 creates `termkey` and says: "It imports the standard library and
`internal/enum`, for its enums' tokens, and nothing else. `keymap`
imports it. A line editor that needs keys without bindings imports
`termkey` alone." That is narrower than the rule every core follows (D1):
no Charm module but `x/ansi`.

Two gates hold the cores today, and both check the wide rule only:

* **depguard's `charmfree` rule** (`.golangci.yml`) is a `lax` deny list
  of Bubble Tea, Bubbles, Lip Gloss, `colorprofile`, `ultraviolet`, `x/term`
  and `internal/teamsg`, over each core's own non-test files;
* **`TestCoresAreCharmFree`** (`internal/conformance/charmfree_test.go`)
  lists each core's closure with `go list -deps` for linux, darwin and
  windows, and fails on a package for which `charm(p)` is true; `charm`
  is false for `github.com/charmbracelet/x/ansi` and everything beneath
  it, by design, and for every module that is not Charm's.

So `termkey` could import `x/ansi`, another module, or another of this
module's packages, such as `glyph`, and both gates would pass. The
0020-PLAN's Step 11 record names the gap: the rule "holds today (`go list
-deps` above), but the gates check only that the closure has no Charm
module but `x/ansi`: nothing refuses an `x/ansi` import in `termkey`."

**Today's closure,** `GOWORK=off go list -deps ./termkey` at `81177eb`:
outside the standard library, `termkey` and `internal/enum` only; and
`internal/enum`'s is itself alone. The rule holds; nothing keeps it.

**A probe,** on a throwaway clone at `81177eb`, with golangci-lint 2.14.0:
a depguard rule `termkeystd`, `list-mode: strict`, over
`**/termkey/*.go` and `!$test`, allowing `$gostd` and
`github.com/maccavelli/go-tui-lib/v2/internal/enum`:
* `termkey` as it is: `0 issues.`, exit 0;
* with `github.com/charmbracelet/x/ansi` imported in `termkey/key.go`:
  exit 1, `termkey/key.go:6:2: import 'github.com/charmbracelet/x/ansi' is
  not allowed from list 'termkeystd' (depguard)`, the one depguard issue;
  `charmfree` reported nothing.

How should the repository hold `termkey` to D7's rule?

## Decision Drivers

* **D7's rule, as written,** over the whole closure: a package `termkey`
  imports could take on a dependency of its own.
* **A direct import fails at lint,** in the editor, before the tests run,
  as D11 gives the cores `charmfree` beside the closure test.
* **No new tool and no new module.** `internal/conformance` runs
  `go list` already; depguard is configured already.
* **Nothing goes stale unseen.** A rename, or an allowance nothing uses
  any more, fails, as `cores` does today.
* **Every new check is seen failing** on a broken copy first (AGENTS.md,
  and the global rules).

## Considered Options

* **A. A closure test with a table of narrow closures, and a strict
  depguard rule for `termkey`**
* **B. The closure test alone**
* **C. The depguard rule alone**
* **D. Amend 0020-MADR D7 to the wide rule**
* **E. Change nothing**

## Decision Outcome

Chosen option: "A. A closure test with a table of narrow closures, and a
strict depguard rule for `termkey`", because it is D11's pattern applied to
D7's narrower rule: the test holds the whole closure on three targets, and
the rule fails a direct import at lint, so neither half leaves the other's
hole.

* **N1. The table.** `internal/conformance` gains `narrow`: each entry a
  core, the packages of this module its closure may hold besides itself,
  and the record that says so. It starts with one entry: `termkey`, which
  may hold `internal/enum` (0020-MADR D7).
* **N2. The test, `TestNarrowClosures`.** For each entry and each of
  linux, darwin and windows, every package in the core's non-test closure
  is in the standard library (`go list`'s `.Standard`), or the core
  itself, or in its allowance. Otherwise it fails, naming the core, the
  package and the shortest import chain to it, as `TestCoresAreCharmFree`
  does. It also fails:
  * when an entry names a package that `cores` does not list, so a narrow
    core is always a core;
  * when an entry's allowance names a package its closure does not hold,
    so the table keeps no dead entry.
* **N3. The depguard rule, `termkeystd`.** `list-mode: strict` over
  `**/termkey/*.go` and `!$test`, allowing `$gostd` and
  `github.com/maccavelli/go-tui-lib/v2/internal/enum`, each with a
  description citing D7 and this record.
* **N4. Joining.** A core joins `narrow`, and gets a rule like N3, by the
  record that states its narrower rule. This record adds `termkey` alone.
* **N5. Out of scope:** 0020-MADR D9 says that `safetext` "imports the
  standard library, `glyph` and `internal/enum` only", a rule of the same
  kind that no gate checks either. Its closure holds it today (`safetext`,
  `glyph` and `internal/enum`). It is not this record's: N4 lets a later
  record, or an amendment here, add it.

### Consequences

* Good, because an `x/ansi` import in `termkey`, or a new dependency of
  `internal/enum`, fails a gate before it is committed, on every target.
* Good, because the table states each narrow rule where the test reads
  it, with its record, so the rule and its check cannot drift apart.
* Neutral, because `TestNarrowClosures` reuses `TestCoresAreCharmFree`'s
  `go list` helper over its own few packages: three more short runs, one
  for each target.
* Bad, because two lists, `narrow` and the depguard rule, name the same
  allowance; a change to one needs the other. The rule's comment says so,
  as `charmfree`'s does for `cores`.
* Bad, because a later wish of `termkey`'s for a module, even a small
  one, now needs a record; that is the point of D7, and the cost is said.

### Confirmation

The PLAN sees each check fail on a scratch copy:
* `termkey` importing `x/ansi`: `TestNarrowClosures` and `termkeystd`
  fail, and `TestCoresAreCharmFree` and `charmfree` still pass, which is
  the gap closed;
* `internal/enum` importing `x/ansi`: `TestNarrowClosures` fails, with the
  chain `termkey → internal/enum → …`, and `termkeystd` passes, which is
  why the closure test is needed;
* `termkey` importing `glyph`: both fail;
* `narrow` naming a package `cores` does not list, and an allowance the
  closure does not hold: `TestNarrowClosures` fails as stale;
* a synthetic graph test of the walk, so that a broken predicate fails
  without a module of its own.

## Pros and Cons of the Options

### A. A closure test with a table of narrow closures, and a strict depguard rule for `termkey`

* Good, because it checks D7's rule over the whole closure, on three
  targets, and a direct import at lint.
* Good, because the probe shows the rule works in this repository's
  golangci-lint, and the test needs nothing `internal/conformance` does
  not run already.
* Bad, because the allowance is written twice.

### B. The closure test alone

* Good, because one list and one place.
* Bad, because a direct import fails only when the tests run, not in the
  editor or at lint, unlike D11's pair.

### C. The depguard rule alone

* Good, because it fails at once, in the editor.
* Bad, because depguard reads direct imports only (0020-MADR D11's own
  argument): an import `internal/enum` took would reach `termkey`
  unseen.

### D. Amend 0020-MADR D7 to the wide rule

* Good, because the record would then describe what the gates check.
* Bad, because it gives up D7's reason for `termkey`: the least a line
  editor can import to read keys.

### E. Change nothing

* Good, because no work.
* Bad, because D7's rule holds by accident, and the Step 11 record says
  so.

## More Information

* 0020-MADR D7 (the rule), D9 (`safetext`'s, N5), D11 (the closure gate
  and `charmfree`, whose pattern this follows).
* The gap, named:
  [0020-PLAN-charm-free-cores-glyphs-and-safetext.md](0020-PLAN-charm-free-cores-glyphs-and-safetext.md),
  the Step 11 record, "Not done here".
* depguard v2's `strict` mode and its `$gostd` allowance, as golangci-lint
  2.14.0 runs them, by the probe above.
* The PLAN:
  [0021-PLAN-termkey-closure-gate.md](0021-PLAN-termkey-closure-gate.md).
