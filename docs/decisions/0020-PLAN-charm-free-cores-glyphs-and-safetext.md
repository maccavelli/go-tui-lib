---
status: in-progress
date: 2026-10-24
associated-madr: "0020-MADR-charm-free-cores-glyphs-and-safetext.md"
---
# Implement Charm-free cores of `termcap`, `termsvc`, `command` and `keymap`, the `termkey` decoder, the status and meter glyphs, and `safetext`

Associated MADR: [0020-MADR-charm-free-cores-glyphs-and-safetext.md](0020-MADR-charm-free-cores-glyphs-and-safetext.md)

Written 2026-10-23 against the tree at `63c68d6`, after the owner accepted
the MADR and answered its Q1 to Q8. It follows the MADR's D1 to D12 with
amendments A1 and A2.

*Revised 2026-10-23:* the owner answered A1.1 with (c), an I/O-free probe
engine in the core, and MADR A2 designs it. The old Step 7 (the prober
moved whole, options split) is replaced by Step 7, the engine in
`termcap`, and Step 8, `termcaptea` over it; the later steps are
renumbered 9 to 13.

*Later on 2026-10-23:* the owner accepted MADR A2 and approved execution
("i accept the design, i approve executing the plan, proceed"). This PLAN
is `in-progress`.

*Deviation, 2026-10-23, Step 5:* the differential against `v0.12.0` found
six classes of difference beyond S2; the owner's resolution is MADR A3 and
the Execution Record's Step 5 entry, and Steps 5 and 6 carry notes where
it changes them.

*Deviation, 2026-10-24, Step 6:* release 1 was tagged `v1.13.0` in place
of `v0.13.0`, and the owner kept it (MADR A4). Where this PLAN says
`v0.13.0`, read `v1.13.0`. Release 2's version and form, written below as
a breaking `v0.14.0`, wait for MADR A4.1, and Steps 7 to 13 are amended
for its answer before Step 7 starts. *Later on 2026-10-24:* A4.1 is
answered, `v2.0.0` at `/v2`. Step 6A, proposed with MADR A4.2, moves the
path and says how Steps 7 to 13 read under it; where they say `v0.14.0`,
read `v2.0.0`.

*Deviation, 2026-10-24, Step 7:* Step 7 as written removes `Prober` from
`termcap` a step before `termcaptea` exists, so `termcap`'s tests,
`termcaptest`'s and `launch`'s could not compile between the two steps.
The owner chose to run Steps 7 and 8 as one step and one commit; the
Execution Record's "Steps 7 and 8" entry has the evidence, and the facts
table and Step 8's files carry notes.

## Goal

Release two versions:

* **`v0.13.0`, additions only:** the closure gate, the eight glyphs of D8,
  `safetext`, `internal/sanitize` as a wrapper over it, and `fuzzy`, which
  0008-PLAN built and no tag carries yet. It answers issues #6, #11 and
  #12.
* **`v0.14.0`, one breaking release** (Q1): the Charm-free cores of
  `termcap`, `termsvc`, `command` and `keymap`, the new `termkey`, and
  their Bubble Tea forms in `termcaptea`, `termsvctea`, `commandtea` and
  `keymaptea`. It answers issues #2 to #5.

Done means every item under Verification holds, CI is green on the pushed
tree and on both tags, both tags are the owner's, and both consumer smoke
tests pass.

## Scope

### Facts the steps rest on (read 2026-10-23, at `63c68d6`)

The MADR's Context lists every fact with its file and line. The ones a step
acts on directly:

| Fact | Step |
| :--- | :--- |
| `GOWORK=off go list -deps`: `glyph`, `when`, `layout` link no Charm; `fuzzy`, `internal/sanitize` link `x/ansi` alone; `termcap`, `termsvc`, `command` link Bubble Tea, `colorprofile`, `ultraviolet`; `keymap` also Bubbles and Lip Gloss | 2, 7–11 |
| `internal/conformance` runs `go list` already (`goList`, `conformance_test.go:382`) | 2 |
| depguard v2.2.1 compiles each `files` pattern with `glob.Compile(exp, '/')`, so `*` never crosses a `/`; a leading `!` negates; `$test` expands to the test files (`settings.go:61-83`) | 2 |
| `glyph.Set` has 18 single glyphs; `TestEveryGlyphIsOneCell` and `TestTierWidths` count 70 (`glyph_test.go:54`, `:110`) | 3 |
| `internal/sanitize` (`sanitize.go`, 272 lines) is called 28 times: `Line` 19, `Truncate` 4, `Token` 2, `HasControl` 2, `Styled` 1 | 4, 5 |
| `Styled` keeps an SGR of at most 32 parameters (`maxSGRParams`, `sanitize.go:162`), because `x/ansi` v0.11.8's parser panics on the 33rd | 4, 5 |
| `termcap.Option` is `interface{ apply(*Prober) }` (`prober.go:51`); `EnvCaps` runs a disabled prober's `start` (`envcaps.go:16-22`); no other file of `termcap` uses the prober's internals | 7 |
| The prober reads its replies as Bubble Tea and ultraviolet decode them: `tea.EnvMsg`, `ColorProfileMsg`, `ModeReportMsg`, `BackgroundColorMsg`, `ForegroundColorMsg`, `KeyboardEnhancementsMsg`, `TerminalVersionMsg`, and through `internal/termevent` DA1, DA2, DSR 997, the Kitty graphics reply and unknown sequences' raw bytes (`prober.go:252-326`, `:595-809`) | 7, 8 |
| Its batch, in order: `CSI ? u`; DECRQM 2031, 2048, 1004; DSR 996; OSC 11; DA2; then gated: XTVERSION, OSC 99 `p=?`, the Kitty graphics query, OSC 10, OSC 4 × 16; then each `WithQuery`; then DA1 (`prober.go:556-585`, `:369`) | 7 |
| `parseXColor` reads `rgb:` with 1–4 hex digits and `#rrggbb` only (`prober.go:737-770`) | 7 |
| Bubble Tea v2.0.10 sends DECRQM 2026 and 2027 itself, only where `shouldQuerySynchronizedOutput` allows (`tea.go:980-994`, `:1118-1123`) | 7, 8 |
| `x/ansi` v0.11.8 exports `DecodeSequence(b, state, p)` and a `Parser` with `Params`, `Command` and `Data` (`parser_decode.go:70`, `parser.go:85-121`); its OSC query constants end with BEL (`background.go:120`, `:147`) | 7 |
| `termcaptest.Terminal` answers each query with raw reply bytes (`termcaptest.go:394-509`), through the unexported `answer` | 8 |
| Outside `termcap`, only `termcap/termcaptest/termcaptest.go` uses `Prober`; `termsvc/notify.go` names `Prober` and `EnvCaps` in its documentation; `launch` uses only `termcap.Env`. *Corrected 2026-10-24, and so already at `63c68d6`: `termcaptest.go` names `Prober` only in its documentation; `termcaptest_test.go:31` and `launch/consumer_test.go:372` call `NewProber`; `launch/restore.go:7` and `docs/guides/commands.md:769` name `*termcap.Prober`* | 8 |
| `Notifier.NotifyContext` checks the policy and numbers on the caller's goroutine, and calls a backend inside its command (`notify.go:279-300`); `CopyContext` writes OSC 52 through `tea.SetClipboard` (`clipboard.go:130-155`) | 8 |
| `Registry.Dispatch` runs a `Loop` command at once and an `Async` one inside its command, then batches `Result.Cmd`, `ResultMsg` and, for a `Prompt` or `Forward`, `PromptMsg` (`dispatch.go:24-40`, `:244-250`) | 9 |
| `Result{Cmd: …}` is set at nine sites in `workspace/commands.go` and once in `command/builtin.go:57`; `launch/run.go:98` calls `Attach(p.Send)`. *Corrected 2026-10-24, and so already at `63c68d6`: twelve sites in `workspace/commands.go`; `workspace/bindings.go:271` returns a handler's `res.Cmd` as a `tea.Cmd`; `docs/guides/keymaps.md:52` calls `Dispatch`* | 9 |
| ultraviolet's decoder is `uv.EventDecoder.Decode(buf []byte) (n int, Event)` (`decoder.go:234`); `uv.KeyMod` is an `int` with shift 1, alt 2, ctrl 4, meta 8, hyper 16, super 32, caps lock 64, num lock 128, scroll lock 256 (`key.go:12-32`); `KeyExtended = unicode.MaxRune + 1` (`key.go:50`); Bubble Tea re-exports both (`bubbletea key.go`, `mod.go`) | 10, 11 |
| `keymap/help.go` reads the unexported `(*Keymap).shown` (`keymap.go:589`); `workspace/bindings.go:221` calls `Dispatcher.Update`, `workspace/help.go:33` calls `Keymap.KeyBinding`; `workspace/bindings_test.go:174` is the one user of `Settings.Tick` | 11 |
| The framework examples under `testdata/frameworks` use `command.Registry`, `Request`, `Result`, `WriteResult` and `launch`, and none of the names that move | 9 |
| `0008-PLAN-command-palette.md` Steps 1 to 4 are done, 5 to 9 (`palette`, the `v0.13.0` release) are not | 1 |

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0020-*`, `0008-PLAN-*`, `0009-PLAN-*`, `docs/README.md`, `docs/glossary.md` | records only |
| 2 | `internal/conformance/charmfree_test.go`, `.golangci.yml` | the closure gate and the `charmfree` depguard rule |
| 3 | `glyph/` | the eight fields of D8 |
| 4 | `safetext/` | the package (D9) |
| 5 | `internal/sanitize/` | the wrapper (D9 S5), the `v0.12.0` oracle, the differential fuzz |
| 6 | `README.md`, `docs/` | release `v0.13.0` (tagged `v1.13.0`, MADR A4) |
| 6A | `go.mod`, the Go files, `.golangci.yml`, `scripts/go-apicheck.py`, `scripts/go-examples.py`, `testdata/frameworks/go.mod.tmpl`, `AGENTS.md`, `docs/` | the module path to `/v2` (MADR A4.1, A4.2) |
| 7 | `termcap/`, `internal/conformance/` | the I/O-free probe engine and `ParseReply` (D4, A2) |
| 8 | `termcap/termcaptea/`, `termcap/termcaptest/`, `termcap/` (tests moved), `internal/conformance/`, `launch/consumer_test.go`, `launch/restore.go`, `docs/` | `termcaptea.Prober` over the engine, the differential test (D4, A2); with Step 7, one step (2026-10-24) |
| 9 | `termsvc/`, `termsvc/termsvctea/`, `docs/` | the `termsvc` split (D5, A1.2) |
| 10 | `command/`, `command/commandtea/`, `workspace/commands.go`, `launch/run.go`, their tests, `docs/` | the `command` split (D6, A1.2) |
| 11 | `termkey/`, `internal/termevent/termkey_test.go` | the key type and decoder (D7, A1.2) |
| 12 | `keymap/`, `keymap/keymaptea/`, `workspace/bindings.go`, `workspace/help.go`, their tests, `docs/` | the `keymap` split (D7, A1.2) |
| 13 | `README.md`, `docs/`, `scripts/apicheck.allow` | release `v0.14.0`, close-out |

No module is added or removed in any step. `safetext` and `termkey`
import the standard library, `glyph` and `internal/enum`; every adapter
imports its core and modules the root already requires.

### Out of scope

* Issues #7 to #10, the line-mode components, and the `inline` record they
  need.
* `Interpret`, 0009-PLAN Step A1.1, and every other step of 0009-PLAN.
* Windows' win32-input-mode in `termkey` (D7).
* Moving `internal/sanitize`'s 28 call sites to `safetext` (D9 S5).
* `launch`, `theme`, `workspace` and `tuitest` becoming cores (D1).
* `palette` (0008-PLAN Steps 5 to 9), which waits for this PLAN (Q7).
* Any change in gobble-cli or pi-go, and posting on the issues: the agent
  drafts each comment in the execution record, and the owner posts it or
  says to.
* `git push` and tags, which are the owner's.

## Prerequisites

* MADR A2 is accepted before Step 7. Steps 1 to 6 do not depend on it.
* *(2026-10-24, MADR A4: the tag is `v1.13.0`; and A4.1 is answered,
  and Steps 7 to 13 amended for it, before Step 7.)*
* `v0.13.0` is tagged, and CI is green on the tag, before Step 7, since
  Steps 7 to 12 diff the API against it.
* `63c68d6`, or a later commit with no change under the paths above, is
  `main`'s head, so the facts stand. The agent re-reads the facts table
  before Step 2 and before Step 7, and records any that moved.
* `v0.12.0` is the latest tag, so `make apicheck` compares with it until
  `v0.13.0`, and `scripts/apicheck.allow` holds no block.
* The verification hosts are in the local `tools/verify` config, which is
  never committed.

## Rules for every step

1. **Order.** The steps run in order. Each step's packages compile, pass
   their tests and pass the gates before the next step starts. Steps 7 to
   12 leave `main` with a changed API and no tag for it; that is expected,
   and only Step 13 tags.
2. **Gates.** `python3 -m tools.verify gates` with its default list, over
   the step's Go files, on the macOS development host, and through
   `python3 -m tools.verify remote` on the Linux and Windows hosts the
   local config names. Nothing is skipped without a dated deviation entry.
   Output goes to a scratch file, and the exit status is captured before
   any filter. The record quotes each host's summary line
   ("gates: N passed, 0 failed, 0 skipped").
3. **The API diff.**
   * Steps 2 to 5: `make apicheck` against `v0.12.0` reports 0
     incompatible changes.
   * Steps 7 to 12: against `v0.13.0`, it reports exactly the lines the
     step adds to `scripts/apicheck.allow`, in a block whose comment names
     "0020-PLAN Step N". Each line matches a declaration the step's
     "Moves and changes" list names; a line that matches none is a
     deviation, not an allow-list entry.
4. **Mutation proofs.** Each step's mutations run through
   `python3 -m tools.verify mutate FILE.json`, a JSON list kept in the
   scratch directory, never committed. Each is applied to a fresh copy,
   never the tree, and must make the named test fail. A mutation that
   survives or does not compile is replaced, and both are recorded. The
   record quotes each failure line.
5. **New checks are seen failing first** (the global rule): each new test
   or rule is also run on a scratch copy broken on purpose, and the record
   says what the failure looked like.
6. **Conventions.** Every package follows AGENTS.md's TUI and API
   conventions: no write to `os.Stdout` or `os.Stderr`, no alternate
   screen, no signal handler, no environment read, no exit, no process;
   every glyph from `glyph`; widths in cells; options opaque; enums
   through `internal/enum` and in `internal/conformance/enums_test.go`'s
   table; each package's documentation ends with its stability line; each
   new type name checked with `python3 -m tools.verify names` and entered
   in `docs/glossary.md` in the step that builds it.
7. **Tests move with what they test.** A test of a declaration that moves
   to an adapter moves with it, unchanged but for imports and receivers. A
   core test that still imports Bubble Tea after its step is allowed only
   where it tests the pairing of core and adapter (D1); the record lists
   each such file.
8. **Golden files** are rewritten only with `-tuitest.update` or
   `TUITEST_UPDATE=1`, read in full before the step ends, and each
   changed file is listed in the record with the reason.
9. **Identifiers.** `python3 -m tools.verify scan` before each handoff:
   no hostname, account name, org-internal path or real-machine absolute
   path in the diff or in an untracked file.
10. **Commit.** One commit per step, made by the owner. At the end of each
    step the agent stops with the gates passed and the evidence in the
    execution record, and stages and commits nothing. The owner commits
    with `git commit --no-edit`.
11. **Deviations.** A finding outside a step stops the work. The agent
    gives the evidence, the resolutions (recommended first, with costs) and
    the cost of doing nothing; the owner's choice is a dated entry here,
    and an amendment of the MADR where a decision or a fact changes, before
    work continues.

## Implementation Steps

### Step 1: records

* MADR A2: the owner's acceptance is recorded, and its status becomes
  `accepted`; a change the owner asks for amends A2 and Steps 7 and 8
  first.
* This PLAN becomes `in-progress` once the owner approves execution, and
  its `docs/README.md` row says so.
* `0008-PLAN-command-palette.md` gains a dated note:
  * Steps 5 to 9 wait for this PLAN to be `complete` (0020-MADR Q7), and
    `palette` is written against `commandtea` and `keymaptea`;
  * `fuzzy` ships in `v0.13.0` by this PLAN's Step 6, so Step 9's
    release is the next minor release after `v0.14.0`, and its migration
    section names that version.
* `0009-PLAN-streaming-content-engine.md` gains a dated note: its Step 2
  (`safetext`) is carried out by this PLAN's Step 4, with 0009-MADR A4's
  names (`Rules`, `Sanitizer`, `NewSanitizer`) and 0020-MADR D9's
  decisions; its Step A1.1 adds `Interpret` and nothing else.
* `docs/glossary.md`, "Reserved by accepted records", gains the names
  0020-MADR A1.3 lists: `safetext`'s `Rules` and `Sanitizer`; `termkey`'s
  `KeyEvent`, `Mod`, `Decoder`, `Input` and `InputKind`; `termcap`'s
  `ColorProfile`, `Probe`, `ProbeEvent`, `ProbeEventKind`, `ProbeHook`,
  `ReplyKind` and `ModeSetting` (A2.2); `command.Effect`; and that
  `Prober` moves from `termcap` to `termcap/termcaptea` in `v0.14.0`.
  `Query` and `Reply` stay `termcap`'s.
* `python3 -m tools.verify names` runs over those names; the record quotes
  its output, whose only clash is `termcaptea.Prober`, the move.
* Records only: no source, test, CI or configuration change in this
  commit.

### Step 2: the closure gate

**Files:** `internal/conformance/charmfree_test.go` (new);
`.golangci.yml`.

* **`TestCoresAreCharmFree`:**
  * `cores` is a sorted list of package directories, at this step:
    `fuzzy`, `glyph`, `internal/enum`, `internal/sanitize`, `layout`,
    `when`.
  * For each of `linux`, `darwin` and `windows`, one run in the root
    module's directory of
    `go list -deps -f '{{.ImportPath}}{{"\t"}}{{join .Imports " "}}' ./<core>...`,
    one argument per core, with `GOWORK=off`, `CGO_ENABLED=0` and that
    `GOOS`. A runner of its own, built as `goList` is
    (`conformance_test.go:382-401`) but taking the arguments, fails the
    test with `go list`'s standard error.
  * From each core's import path, a breadth-first walk of the printed
    graph. For each package `p` it reaches for which `charm(p)` holds, the
    test fails with
    `<core> (GOOS=<goos>) links <p>: <core> → … → <p>`, the shortest
    chain, imports in lexical order where chains tie.
  * A core `go list` does not find fails the test:
    `cores lists <dir>, which go list does not find`.
* **`charm(p)`** is true when `p` starts with `charm.land/`, or starts with
  `github.com/charmbracelet/` and is neither
  `github.com/charmbracelet/x/ansi` nor under
  `github.com/charmbracelet/x/ansi/`.
* **`TestCharmPredicate`**, a table: `charm.land/bubbletea/v2`,
  `charm.land/lipgloss/v2`, `github.com/charmbracelet/colorprofile`,
  `github.com/charmbracelet/ultraviolet`, `github.com/charmbracelet/x/term`
  and `github.com/charmbracelet/x/ansix` are Charm;
  `github.com/charmbracelet/x/ansi`, `github.com/charmbracelet/x/ansi/parser`,
  `github.com/clipperhouse/uax29/v2/graphemes` and `strings` are not.
* **depguard rule `charmfree`** in `.golangci.yml`, after `xterm`, with a
  comment naming 0020-MADR D11:

  ```yaml
  charmfree:
    list-mode: lax
    files:
      - "**/fuzzy/*.go"
      - "**/glyph/*.go"
      - "**/internal/enum/*.go"
      - "**/internal/sanitize/*.go"
      - "**/layout/*.go"
      - "**/when/*.go"
      - "!$test"
    deny:
      - pkg: charm.land/bubbletea/v2
      - pkg: charm.land/bubbles/v2
      - pkg: charm.land/lipgloss/v2
      - pkg: github.com/charmbracelet/colorprofile
      - pkg: github.com/charmbracelet/ultraviolet
      - pkg: github.com/charmbracelet/x/term
      - pkg: github.com/maccavelli/go-tui-lib/internal/teamsg
  ```

  Each `deny` entry has a `desc` naming 0020-MADR D11. Later steps add
  each new core's pattern to `files` in the step that makes it clean.
* **Mutations:**

  | Mutation | Must fail |
  | :--- | :--- |
  | `glyph/glyph.go` imports `_ "charm.land/bubbletea/v2"` | `TestCoresAreCharmFree`: `glyph (GOOS=linux) links charm.land/bubbletea/v2: glyph → charm.land/bubbletea/v2` |
  | `internal/enum` imports the same | `TestCoresAreCharmFree`, with a chain through `internal/enum` for `glyph` |
  | `cores` gains `nosuch` | `TestCoresAreCharmFree`: `cores lists nosuch` |
  | `charm` lets through any `github.com/charmbracelet/x/` path | `TestCharmPredicate` (`x/term`, `x/ansix`) |
  | `glyph/glyph.go` imports `_ "charm.land/bubbletea/v2"` (cmd: `golangci-lint run -c .golangci.yml ./glyph/` with `GOWORK=off`) | depguard `charmfree` |

* **Controls,** on a scratch copy, each expected to pass: the same import
  in `glyph/glyph_test.go` passes lint (`!$test`); a new
  `termcap/termcaptea/x.go` importing Bubble Tea passes lint (`*` does not
  cross `/`).

### Step 3: `glyph`, the eight fields (D8, issue #11)

**Files:** `glyph/glyph.go`, `glyph/glyph_test.go`.

* `Set` gains, after `Mask` and before `Keys`, with one comment each as
  `glyph.go`'s fields have:

  | Field | `Unicode()` | `ASCII()` |
  | :--- | :--- | :--- |
  | `Done` | `✓` U+2713 | `+` |
  | `Failed` | `✗` U+2717 | `x` |
  | `Running` | `●` U+25CF | `*` |
  | `Pending` | `○` U+25CB | `-` |
  | `Heading` | `▸` U+25B8 | `>` |
  | `MeterFill` | `━` U+2501 | `#` |
  | `MeterTrack` | `╌` U+254C | `-` |
  | `Control` | `␛` U+241B | `^` |

* `TierLegacy` gives `ASCII()`, as today, so it needs no entry.
* **Tests:**
  * the walked count in `TestEveryGlyphIsOneCell` and `TestTierWidths`
    goes from 70 to 78, with the comment's arithmetic ("plus 26 single
    glyphs");
  * `TestStatusAndMeter`: the table above, exactly, for both sets; each
    field's two forms differ; `Done`, `Failed`, `Running` and `Pending`
    are four different strings in each set;
  * `TestControlIsNotEmoji`: `Control`'s code point is not in the
    `emoji-data.txt` `Emoji` property, checked by the execution against
    Unicode 18.0's file and recorded (the MADR left `␛` to the PLAN); the
    test pins the code point.
* **Mutations:** `ASCII().Running` is `-` (`TestStatusAndMeter`: four
  status glyphs, three distinct); `Unicode().Done` is `✔`
  (`TestStatusAndMeter`); `Unicode().MeterTrack` is `╌╌`
  (`TestEveryGlyphIsOneCell`); `ASCII().Control` is `␛`
  (`TestASCIISetIsASCII`).

### Step 4: `safetext` (D9, issue #12)

**Files:** `safetext/doc.go`, `rules.go`, `clean.go`, `utf8.go`,
`escape.go`, `sanitizer.go`, `url.go`, and their tests; `.golangci.yml`
(`charmfree` gains `**/safetext/*.go`);
`internal/conformance/charmfree_test.go` (`cores` gains `safetext`).

* **Imports:** the standard library, `glyph` and `internal/enum` only.
* **The API** (0009-MADR §2, A1, A4; 0020-MADR D9):

  ```go
  type Rules struct {
      Tabs, Newlines bool     // keep \t; keep \n, \r\n becoming \n and a lone \r dropped
      Flatten        bool     // each run of \r \n \t U+2028 U+2029 becomes one space; wins over Tabs and Newlines
      Bidi           bool     // keep the Bidi_Control set
      Visible        bool     // one Glyphs.Control in place of each removed sequence
      Glyphs         glyph.Set // Control's source; a Set whose Control is "" uses glyph.Unicode()
      MaxString      int      // bytes an OSC, DCS, SOS, PM or APC may run before it is abandoned; 0 means 64 KiB
      Escape         bool     // show what would be removed, escaped, instead of removing it
      KeepSGR        bool     // keep SGR, of at most 32 parameters, and OSC 8 links
      LinkSchemes    []string // with KeepSGR: the OSC 8 schemes kept, compared without case; empty keeps any printable-ASCII target
  }
  func Text() Rules       // Tabs, Newlines, Bidi
  func Strict() Rules     // Tabs, Newlines
  func Line() Rules       // Flatten
  func Command() Rules    // Escape
  func StatusLine() Rules // Flatten, KeepSGR, LinkSchemes http, https, mailto, a host required
  func Clean(s string, r Rules) string
  func IsClean(s string, r Rules) bool
  type Sanitizer struct{ /* parser state */ }
  func NewSanitizer(w io.Writer, r Rules) *Sanitizer
  func (s *Sanitizer) Write(p []byte) (int, error)
  func (s *Sanitizer) Close() error
  func SafeURL(u string) (string, bool)
  func URLs(s string) iter.Seq2[int, int]
  ```

* **The grammar** is 0009-MADR §2's, exactly: ECMA-48 with the DEC
  parser's recovery; CSI, OSC (ended by BEL, ST or U+009C), DCS, SOS, PM,
  APC, nF/Fp/Fe/Fs escapes; any other byte after `ESC` dropped with it;
  CAN and SUB abort; `ESC` inside a sequence starts a new one; encoded C1
  code points read as their 7-bit forms; a whole sequence removed, not
  its `ESC` alone.
* **What is removed or escaped** (S1): C0 but what `Tabs` and `Newlines`
  keep, DEL, U+0080–U+009F, U+200B, U+FEFF, U+2060–U+2064, U+00AD,
  U+E0000–U+E007F, and unless `Bidi`, U+061C, U+200E, U+200F,
  U+202A–U+202E, U+2066–U+2069. ZWJ, ZWNJ and U+FE00–U+FE0F stay.
* **Invalid UTF-8** (S2): one U+FFFD per maximal subpart, decoded with the
  byte ranges of Unicode's Table 3-7 in `utf8.go`, in the same pass. A
  raw 0x80–0x9F that continues no sequence is one U+FFFD.
* **Escape** (S4): a rune that would be removed is written as
  `strconv.QuoteToASCII` writes it, without the quotes (`\x1b`, `\n`,
  `\u202e`, `\U000e0041`); an escape sequence is its `ESC` so written and
  its other bytes as text. What `Tabs` and `Newlines` keep stays raw;
  `Flatten` wins over all three. U+FFFD replacement still applies (S2).
* **`IsClean`** stops at the first byte that `Clean` would change, and
  allocates nothing.
* **Tests:**
  * 0009-PLAN Step 2's table, a row per sequence class: CSI; OSC ended by
    BEL, by ST and by U+009C; DCS; SOS; PM; APC; nF escapes; a lone `ESC`;
    CAN and SUB; `ESC` inside a sequence; each C1 code point; each C0
    control; DEL; `\r\n`; a lone `\r`; invalid UTF-8; the bidi controls
    under `Text()` and `Strict()`;
  * codex's injection string
    `"\x1b[2J\x1b]52;c;Y2xpcA==\x07\u009dhidden\u009c\r\n\t"` cleans to
    `"\n\t"` under `Text()`;
  * an unterminated OSC hides at most `MaxString` bytes, and the text after
    survives;
  * `Visible` writes one `Control` per removed sequence, `^` under
    `glyph.ASCII()`;
  * S1: every code point `internal/sanitize`'s `dropped` names
    (`sanitize.go:26-38`) is removed under `Line()`, and under `Text()`
    but for the twelve Bidi_Control code points, which `Text()` keeps;
  * S2: Table 3-8's example, its bytes and expected code points quoted
    from the specification in the test's comment; `E2 82` at the end is
    one U+FFFD and `C0 AF` two; and `TestMaximalSubparts`, every
    one-, two- and three-byte prefix over the bytes 0x80–0xFF against a
    reference implementation of Table 3-7 written plainly in the test;
  * S3: `"a\r\n\t\u2028b"` is `"a b"` under `Line()`, and `"a\n\nb"` is
    `"a\n\nb"` under `Text()`;
  * S4: `Rules{Tabs: true, Newlines: true, Escape: true}` keeps `\n` and
    `\t`, writes `\x1b[2J` for `ESC [2J`, and `\u202e` for U+202E;
    `Command()` writes `\n` for a newline;
  * `KeepSGR`: an SGR of 32 parameters is kept and one of 33 is removed;
    OSC 8 with any printable-ASCII target is kept with empty
    `LinkSchemes`; `StatusLine()` keeps an `https://example.org` link and
    removes `file:///etc` and `https://` with no host;
  * `SafeURL` refuses `javascript:`, a URL holding ESC, BEL or a space, and
    one holding `ESC \`; `URLs` returns spans that slice the cleaned text
    into its URLs;
  * `IsClean(s, r) == (Clean(s, r) == s)` over the table's inputs.
* **`FuzzClean(data []byte, preset uint8, split uint16)`,** for each of the
  five presets and `Rules{Tabs: true, Newlines: true, Escape: true}`:
  * the output is valid UTF-8;
  * without `Escape` and `KeepSGR`, it holds no 0x1B, no U+0080–U+009F and
    no C0 control but `\t` and `\n` where kept;
  * `Clean(Clean(x)) == Clean(x)`;
  * writing `x[:i]`, then `x[i:]`, then `Close`, through a `Sanitizer`
    gives `Clean(x)`, `i = split % (len(x)+1)`;
  * `IsClean(x) == (Clean(x) == x)`.

  `make fuzz` and CI's fuzz step find it on their own.
* **The differential against `x/ansi`** (0009-PLAN Step 2): over generated
  well-formed CSI, OSC and DCS sequences mixed with valid UTF-8 text,
  `Clean(x, Strict())` equals `ansi.Strip(x)` with every C0 control but
  `\t` and `\n` removed. `x/ansi` is imported by this test file only.
* **Allocations,** asserted with `testing.AllocsPerRun(100, …)`:
  `Clean` on each clean input of the benchmarks allocates 0; on each
  input that needs cleaning, 1; `Sanitizer.Write` to `io.Discard`, after
  one warm-up write, 0; `IsClean`, 0.
* **Benchmarks** (`b.Loop`, `-benchmem`): `Clean` and `IsClean` over five
  inputs of 4 KiB built by the test from fixed strings, with no
  randomness: clean ASCII, clean CJK and emoji, tool output (lines of
  text with `\t`, `\r\n` and SGR), styled (SGR and OSC 8), and hostile
  (CSI, OSC, bidi controls and raw C1 bytes); and `Clean` over 1 MiB of
  prose and 1 MiB that is half escape sequences. The record holds their
  output beside gobble's table in the MADR. Times are recorded, not
  asserted.
* **Mutations:** BEL no longer ends an OSC; encoded C1 code points pass
  through; the `Sanitizer` forgets its state between writes; `MaxString`
  is ignored; `SafeURL` accepts any scheme; invalid UTF-8 is replaced one
  U+FFFD per byte (`TestMaximalSubparts`); `Flatten` turns each break, not
  each run, into a space (S3's test); `Escape` drops instead of escaping
  (S4's test); `KeepSGR` keeps 33 parameters; `IsClean` reports true for
  a string with U+200B. Each names its failing test in the record.

### Step 5: `internal/sanitize` wraps `safetext` (D9 S5)

**Files:** `internal/sanitize/sanitize.go`; new
`internal/sanitize/oracle_test.go`, `internal/sanitize/bench_test.go`;
`internal/sanitize/fuzz_test.go`.

*Deviation, 2026-10-23 (Execution Record, Step 5; MADR A3):* also
`internal/sanitize/sanitize_test.go` (its changed rows, and a row for each
class of MADR A3.1); `safetext/clean.go`, `safetext/rules.go` and
`safetext/safetext_test.go` (MADR A3.3: `KeepSGR` keeps nothing written
with a C1); `termcap/json_test.go` and `termsvc/termsvc_test.go` (their
assertions of the old output); and `termsvc/termsvc.go` (`strip`'s
comment, which says invalid UTF-8 is removed).

* **First, the benchmarks, on the unchanged implementation.**
  `bench_test.go` adds `BenchmarkLine`, `BenchmarkStyled` and
  `BenchmarkHasControl` over Step 4's five 4 KiB inputs.
  `go test -run '^$' -bench . -benchmem -count=10 ./internal/sanitize`,
  output to a scratch file, recorded as "before".
* **The oracle.** `oracle_test.go` holds `internal/sanitize/sanitize.go`
  as `git show v0.12.0:internal/sanitize/sanitize.go` prints it, its
  `Line`, `Styled`, `HasControl` and their helpers renamed with an
  `oracle` prefix, nothing else changed, with a comment naming the tag.
* **Then the wrapper:**
  * `Line(s)` is `safetext.Clean(s, safetext.Line())`;
  * `Styled(s)` is `safetext.Clean(s, safetext.Rules{Flatten: true,
    KeepSGR: true})`;
  * `HasControl(s)` is `!safetext.IsClean(s, safetext.Line())`;
  * `Truncate` and `Token` are unchanged;
  * the package comment says that `safetext` does the cleaning, and that
    `Truncate` and `Token` stay here (0020-MADR D9 S5).
* **`FuzzAgainstV0120(data []byte)`:** let `f` be the test's reference
  maximal-subpart replacement (Step 4's). For every input,
  `Line(x) == oracleLine(f(x))`, `Styled(x) == oracleStyled(f(x))` and
  `HasControl(x) == oracleHasControl(f(x))`. For valid UTF-8, `f(x) == x`,
  so the two agree exactly; for invalid UTF-8 they differ only by S2. A
  difference beyond that is a defect in Step 4's package, fixed there,
  unless `safetext` removes something the oracle kept, which is recorded as
  a deviation for the owner.

  *Deviation, 2026-10-23 (MADR A3):* the equations above cannot all hold.
  `HasControl`'s is wrong as written, since `v0.12.0` also counted invalid
  UTF-8 as a control; and MADR A3.1's classes 1 to 5 differ by the
  grammar. The check is restated as four exact properties. For every input
  `x`:
  * `HasControl(x) == oracleHasControl(x)`;
  * `Line(Styled(x)) == Line(x)`;
  * unless `lineDeparts(x)`, `Line(x) == oracleLine(f(x))`;
  * unless `lineDeparts(x)` or not `plainest(x)`,
    `Styled(x) == oracleStyled(f(x))`.

  `lineDeparts` holds when `x` has a class 1 to 4 trigger:
  * a C1 code point;
  * a DCS, SOS, PM or APC introducer, after any C0 or DEL, followed later
    by BEL or a byte of 0x80 or more;
  * an OSC introducer, after any C0 or DEL, followed later by a byte of
    0x80 or more (*the C0 and DEL added the same day, after the fuzzer
    found the shape; Execution Record, Step 5*).

  `plainest` holds when every ESC begins one of three forms:
  * a CSI: a private marker first, if any, then digits, `;` and `:`, then
    intermediates, then a final byte;
  * an OSC of printable ASCII, ended by BEL or `ESC \`;
  * a two-byte escape.

  Outside `plainest` is class 5. Each clause cites its rule in the test.
  `TestDepartures` in `sanitize_test.go` pins one row per class: the input,
  the new output and `v0.12.0`'s.
* **The existing tests** of `internal/sanitize` pass unchanged, except
  those that pin invalid UTF-8 being dropped, which change to U+FFFD; the
  record lists each.
* **After:** the same benchmark run, recorded as "after". Pass: for each
  benchmark and input, the median of the ten after-runs is at most 1.10
  times the median of the ten before-runs, and allocations per operation
  are no more than before.
* **The tree's tests** all pass; any golden file that changes holds
  invalid UTF-8 in its input, and is read and listed.
* **Mutations:** `Styled` drops `KeepSGR` (`FuzzAgainstV0120`'s seed
  corpus, and `sanitize_test.go`); `HasControl` is `IsClean` without the
  negation; `Line` uses `safetext.Text()`.
  *Deviation, 2026-10-23 (MADR A3):* also, `KeepSGR` keeps an SGR
  introduced by U+009B (`safetext`'s `TestKeepSGR`); `KeepSGR` keeps an
  OSC 8 ended by U+009C (the same); and `lineDeparts` without its C1
  clause (`FuzzAgainstV0120`'s seed corpus).

### Step 6: release `v0.13.0`

* **Docs:**
  * `docs/architecture.md`: `safetext`'s row and imports; `glyph`'s new
    fields; `internal/sanitize` as a wrapper; the closure gate and the
    cores' list;
  * `docs/guides/migrating.md`, "Migrating to `v0.13.0`": *Added*
    (`safetext`, the eight glyph fields, `fuzzy`), *Changed* (the library's
    output shows invalid UTF-8 as U+FFFD, one per maximal subpart, where it
    dropped it; *and, by the Step 5 deviation, MADR A3.1's classes on
    malformed input: an encoded C1 read as its 7-bit form, BEL ending a
    DCS, SOS, PM or APC, non-ASCII inside them, a character's 0x9C byte
    not ending an OSC, and the styled path splitting malformed sequences
    as the plain path does*);
  * `docs/glossary.md`: `Rules` and `Sanitizer` move from "Reserved" to
    "Built";
  * `README.md` Status; `docs/README.md` rows.

  *Deviation, 2026-10-23 (Execution Record, Step 6):* `v0.13.0` also
  carries 0008-PLAN Steps 2 to 4, which are on `main` and in no tag:
  `glyph`'s `Prompt` and `Mask`; `workspace`'s `CommandConsumer`,
  `CommandMsg`, `Close`, `Bindings` and `Theme`; and `fuzzy`, of which
  this list named only the package. `architecture.md` and *Added* name
  them too. 0008-PLAN Step 9's guide, `docs/guides/command-palette.md`,
  stays with that PLAN.
* **The release,** as `docs/guides/releasing.md` "Release the root alone"
  says:
  * release notes in the execution record;
  * `python3 -m tools.verify gates` on every host, with `release-check`
    and `apicheck` against `v0.12.0` (0 incompatible changes);
  * the disclosure guard over the outgoing commits (AGENTS.md,
    "Identifiers"), then the owner's push;
  * CI green on `main`; the owner's annotated tag `v0.13.0`; CI green on
    the tag.
* **The consumer smoke test,** as `releasing.md` "The consumer smoke test"
  says, in a scratch module outside the repository with `GOWORK` empty:
  `go get github.com/maccavelli/go-tui-lib@v0.13.0` from the proxy; a
  program that cleans a hostile string with `safetext`, draws
  `glyph.For(false).Done`, and ranks with `fuzzy`; built and run; then
  `go list -deps` over it, checked with Step 2's `charm` rule: no Charm but
  `x/ansi`. Negative control: the same program importing `termcap` shows
  `charm.land/bubbletea/v2` in the list.
* **Issue comments** for #6, #11 and #12, drafted in the record for the
  owner.

### Step 6A: the module path moves to `/v2` (MADR A4.1, A4.2)

*Proposed 2026-10-24, and approved by the owner that day ("committed,
approved to proceed"). It is lettered, not numbered, so that Steps 7 to 13
and MADR A2's references to them keep their numbers.*

**Files:**
* `go.mod`: `module github.com/maccavelli/go-tui-lib/v2`;
* every tracked `.go` file that imports one of the module's packages.
  At `220506d` the module path appears 293 times in Go files;
* `.golangci.yml`: `charmfree`'s `internal/teamsg` path;
* `testdata/frameworks/go.mod.tmpl`, and `scripts/go-examples.py` and its
  test: the `replace` and `require` of the root;
* `scripts/go-apicheck.py` and `scripts/go-apicheck_test.py`: the move
  across a major boundary;
* `AGENTS.md`: the module path, and API rules 1 and 2 as MADR A4.2 words
  them;
* `docs/guides/releasing.md`: a section "Release a new major version",
  and the root's tags `vX.Y.Z` with `/v2` in its module path;
* the import lines and module path in `README.md`, `docs/architecture.md`
  and `docs/guides/*.md`.

Records under `docs/decisions/` and `docs/reports/` keep the path they
were written with, since they are history.

* **The move:**
  * done by a script in the scratch directory, never committed. It
    rewrites `github.com/maccavelli/go-tui-lib` not followed by `/v2` to
    `github.com/maccavelli/go-tui-lib/v2`, in the files above only;
  * it asserts each file's count before it writes;
  * it prints the counts, which the record holds;
  * `gofmt` and `goimports` then re-sort the import blocks.
* **The API diff gate across a major boundary:**
  * `go-apicheck.py` reads the base's module path from its `go.mod`.
    When it differs from the tree's only by a `/vN` suffix, the gate moves
    the base's own import paths to the tree's path, in its `.go` files and
    `go.mod`, before exporting it;
  * otherwise it is unchanged;
  * the script's docstring and `docs/architecture.md`'s Tooling say so.
* **Tests:** `go-apicheck_test.py` gains three cases, on throwaway
  repositories, as its others do:
  * a module tagged `v1.0.0` and moved to `/v2` with no API change is
    clean;
  * the same with one function removed reports that function, and only
    it;
  * a module whose path changes otherwise, not by a major suffix, is
    compared as before.
* **Checks:**
  * `GOWORK=off go list -m` prints `github.com/maccavelli/go-tui-lib/v2`;
  * `go build`, `go vet` and `go test` pass for every module, in both
    modes, and `go mod tidy -diff` is clean;
  * `make apicheck` reports "against v1.13.0, 0 incompatible change(s)";
  * `make examples` is clean;
  * no tracked file outside `docs/decisions/` and `docs/reports/` names
    the module path without `/v2`, by a scan the record quotes.
* **Mutations:**
  * the gate without the boundary move: the second new test fails on the
    noise lines, and `make apicheck` on a scratch copy reports every
    cross-package signature;
  * the boundary move applied whatever the paths: the third new test
    fails;
  * one `.go` file left at the old path: `go build` fails.
* **Gates** on every host, as Rule 2. Nothing is tagged: `v2.0.0` is Step
  13's.

**Under this step, the later steps change as follows:**
* *Rule 3, Steps 7 to 12:* against `v1.13.0`, through the boundary move;
  each step's `apicheck.allow` block lists its real moves.
* *Steps 7 to 12:* their files, names and tests are unchanged; their
  imports read `/v2`.
* *Step 13* releases `v2.0.0`:
  * `docs/guides/migrating.md` gains "Migrating to `v2.0.0`", whose first
    item is the import path, then D10's table;
  * the smoke test runs
    `go get github.com/maccavelli/go-tui-lib/v2@v2.0.0`;
  * the owner's tag is `v2.0.0`;
  * the close-out removes the allow-list blocks, as written.

### Step 7: `termcap`, the I/O-free probe engine (D4, A2)

*Starts only once MADR A2 is accepted.*

*Deviation, 2026-10-24:* Steps 7 and 8 run as one step, with one commit,
one set of gates and one `apicheck.allow` block, whose comment names
"0020-PLAN Steps 7 and 8". Each step's tests and mutations stay as
written, and Rule 1's "each step compiles" holds at the end of the pair.

**Moves and changes** (with Step 8's, the `apicheck.allow` block's whole
list for `termcap`):

* removed from `termcap`: `Prober`, `NewProber` and their methods (to
  `termcaptea` in Step 8); `KeyboardFlags` (Step 8's
  `termcaptea.KeyboardEnhancements`);
* changed in `termcap`: `Reply` (its `Msg tea.Msg` removed; the fields of
  A2.2 added); `Caps.Profile` from `colorprofile.Profile` to
  `ColorProfile`; `KeyboardCaps.Enhancements` removed and
  `KeyboardCaps.Request int` added; `Option`'s unexported method;
  `ReportSchemaVersion` from 1 to 2;
* added to `termcap`: `Probe`, `NewProbe` and the methods of A2.2;
  `ProbeEvent`, `ProbeEventKind` and its constants; `ProbeHook`
  (`HookAppearance`, `HookConsoleHost`); `ReplyKind` and its constants;
  `ModeSetting` and its constants; `ParseReply`; `WithoutModeQuery`;
  `ColorProfile` and its constants.
* unchanged: every other option, `Query`, `EnvCaps`, `Caps`, its views,
  `Findings`, `Report`, every enum and reason token, `CapsMsg`,
  `ColorSchemeMsg`, `DefaultTimeout`.

**Files:**

* `reply.go` (new): `Reply`, `ReplyKind`, `ModeSetting`, `ParseReply`.
* `xcolor.go` (new): the `XParseColor` reader, from `prober.go`'s
  `parseXColor`, extended per A2.4.
* `probe.go` (new): `Probe`, `ProbeEvent`, `ProbeEventKind`, `ProbeHook`,
  and the engine, from `prober.go` with every `tea` and `termevent` use
  replaced by `Reply` and events.
* `options.go` (new): `Option interface{ apply(*Probe) }`, every option
  moved from `prober.go` unchanged in meaning, and `WithoutModeQuery`.
* `queries.go` (new): the batch and its parsers (`prober.go:550-834`),
  over `Reply`.
* `envcaps.go`: `EnvCaps` runs `NewProbe(append(opts, WithDisabled(),
  WithGOOS(goos))...)`, `Start(env, time.Time{})`, and returns `Caps()`.
  Its tests pass unchanged but for imports.
* `colorprofile.go` (new), `caps.go`, `views.go`, `report.go`: as below.
* `prober.go` leaves `termcap` (Step 8 takes its Bubble Tea half).
* `.golangci.yml`: `charmfree` gains `**/termcap/*.go`;
  `charmfree_test.go`'s `cores` gains `termcap`.

**`ParseReply(raw string) Reply`:**

* Tokenizes with `x/ansi`'s `DecodeSequence` and a `Parser` sized for
  32 parameters and 1 KiB of data. A string holding more than one
  sequence is read as its first; `Raw` is always the input.
* Accepts 7-bit introducers and the 8-bit forms 0x9B, 0x9D, 0x90, 0x9F;
  ends strings at BEL, `ESC \`, 0x9C, or a bare trailing `ESC` (urxvt).
* Kinds, each from the grammar of A2.1:
  * `CSI ? Ps ; … c` → `ReplyPrimaryAttributes`, `Params`;
  * `CSI > Pp ; Pv ; Pc c` → `ReplySecondaryAttributes`, `Params`;
  * `DCS > | text ST` → `ReplyVersion`, `Text`;
  * `CSI ? Ps ; Pm $ y` → `ReplyMode`, `Mode`, `Setting` (`Pm` 0–4; any
    other `Pm` is `ReplyUnknown`);
  * `OSC 10 ; spec`, `OSC 11 ; spec`, `OSC 4 ; index ; spec` →
    `ReplyColor`, `OSC`, `Index`, `Color` (an unreadable spec is
    `ReplyUnknown`);
  * `CSI ? flags u` → `ReplyKittyKeyboard`, `Flags`;
  * `CSI ? 997 ; 1 n` / `; 2 n` → `ReplyColorScheme`, `Dark`;
  * `APC G … ; text ST` → `ReplyKittyGraphics`, `Text`;
  * `OSC 99 ; meta ; keys` → `ReplyNotify`, `Text` the keys;
  * anything else, or a sequence over 1 KiB, → `ReplyUnknown`.

**The `XParseColor` reader** (A2.4): `rgb:` with 1–4 hex digits per
channel, each scaled to 16 bits as `v * 0xffff / (16^n - 1)`; `#` with 3,
6, 9 or 12 hex digits, each channel's digits the most significant bits of
16, so `#3a7` is `#3000a0007000`; `rgba:` read as `rgb:` with the fourth
channel dropped; the result a `color.RGBA64`. Anything else is not a
colour.

**The engine** (`probe.go`), today's prober's behaviour, I/O-free:

* `Start(env, now)`: once only; `setEnv`; the JetBrains reasons and the
  editor gating of `prober.go:336-356`; then, unless disabled, one
  `ProbeWrite` with the batch of `queries.go` and DA1, in today's order,
  less each mode `WithoutModeQuery` names; DECRQM 2026 and 2027 join the
  batch, before DA1, unless excluded, and only where
  `shouldQuerySynchronizedOutput`'s rule (copied, with a comment naming
  Bubble Tea v2.0.10 `tea.go:980-994`) allows; the deadline becomes
  `now + timeout`; a `ProbeRunHook` for each hook the options set, the
  console host's only for `GOOS` `windows`. A disabled probe, or one in
  JetBrains, queues `ProbeDone` at once.
* `HandleReply(r, now)`: a reply over 1 KiB marks the query its prefix
  names with `ReasonReplyTooLong`; else each sent query's `Parse` is
  offered `r`, as `prober.go:298-303` does; the kinds `Update` handles
  itself today (`prober.go:270-319`) set their facts: `ReplyMode` 2026
  and 2027, `ReplyColor` 11 (and `Dark` unless a DSR 997 said),
  `ReplyColorScheme`, `ReplyPrimaryAttributes` (attributes, Sixel, the
  Apple fingerprint, then the sentinel). The sentinel marks every
  unanswered query with its silent meaning, sets `Complete`, and queues
  `ProbeDone`. After the probe, a `ReplyColorScheme` queues
  `ProbeColorScheme`, then, unless `WithoutBackgroundRequest`, a
  `ProbeWrite` of the OSC 11 query. Once `ColorSchemeReports` is
  supported, and unless `WithoutColorSchemeUpdates`, one `ProbeWrite` of
  mode 2031's set; `Restore` returns its reset.
* `HandleTimeout(now)`: before `ProbeDone`, and with `now` at or after
  the deadline, sets `TimedOut` and queues `ProbeDone`; otherwise nothing.
* `HandleHook(h, v, ok)`: the appearance hook's answer sets `Dark` at
  `Heuristic` with `ReasonDesktopAppearance`; the console host's sets
  `LegacyConsole` at `Queried`; `ok` false changes nothing.
* `Timeout()`: the deadline, and true, until `ProbeDone`; then false.
* `Awaiting()`: as `prober.go:442-450`.
* `Caps()`: a copy, `Attributes` cloned, overrides applied.
* `ProbeDone` is queued once; `ProbeEvent.Caps` is `Caps()` at that time.
* Documentation: the package comment and `Probe`'s say how a host drives
  it (A2.3), that a host ignores a kind it does not know, that a reply a
  host does not read reaches the user's input line, that a CLI probes
  only from raw mode and never without input, and that a CLI using
  `termkey` does not `Flush` while `Awaiting` (A2.4).

**`colorprofile.go`, `caps.go`, `views.go`, `report.go`:** D4 as written:
`ColorProfile` (`ProfileUnknown`, `ProfileNoTTY`, `ProfileASCII`,
`ProfileANSI`, `ProfileANSI256`, `ProfileTrueColor`; tokens `unknown`,
`notty`, `ascii`, `ansi`, `ansi256`, `truecolor`); `Caps.Profile
ColorProfile`, written by token and read by token or by colorprofile's
spellings (`Unknown`, `NoTTY`, `Ascii`, `ANSI`, `ANSI256`, `TrueColor`);
`KeyboardCaps.Request`, `2` when `keyboardReason(c)` is `""`, else `0`,
documented for a CLI as `CSI > (1|Request) u` unless `Kitty.Reason` says
the terminal wants none; `ReportSchemaVersion = 2`.

**Tests** (all in `termcap`, none importing Bubble Tea):

* `TestParseReply`: a row for each kind and each terminator (BEL, ST,
  0x9C, bare `ESC`) and each 8-bit introducer; DECRPM `Pm` 0–4 and 5;
  OSC 4 with indexes 0 and 15; an unreadable colour; a DA1 with no
  parameters; a reply of 1 KiB and of 1 KiB + 1; `Raw` equal to the input
  in every row.
* `TestXParseColor`: `rgb:f/0/8`, `rgb:ff/a5/0`, `rgb:fff/000/888`,
  `rgb:ffff/0000/8080`, `#3a7` = `#3000a0007000`, `#ff8000`, `#fff000888`,
  `#ffff00008080`, `rgba:ff/ff/ff/80`, and refusals: `rgb:fffff/0/0`,
  `rgb:g/0/0`, `#12345`, `cmyk:…`.
* `TestProbeBatch`: the first `ProbeWrite` after `Start` is today's batch,
  byte for byte, for a plain environment, for tmux (passthrough-wrapped
  queries), for Apple Terminal and SSH (gated queries left out), and with
  `WithoutModeQuery(2026, 2027)`, with and without 2026 and 2027.
* `TestProbeTranscripts`: for each profile of `termcap/termcaptest`
  (`Kitty`, `XTerm`, `Tmux(true)`, `Tmux(false)`, `DA1Only`,
  `AppleTerminalSSH`, `JetBrains`, `Silent`), a fixed transcript: the
  reply bytes that profile gives to each query of the batch, copied into
  the test from `termcaptest.go:394-509` by the execution and recorded.
  The probe is driven with them, each `ParseReply`d, and the final `Caps`
  equal the `Caps` today's prober tests pin for that profile. Step 8
  replaces the copied bytes with `Terminal.Reply`, so the two cannot
  drift.
* `TestProbeTimeout`: no reply; `HandleTimeout` before the deadline does
  nothing, at it queues one `ProbeDone` with `TimedOut`; a second does
  nothing.
* `TestProbeDuplicateDA1`: a second DA1 changes nothing (Windows
  Terminal under SSH, A2.1).
* `TestProbeHooks`: a hook's `ProbeRunHook` is queued only when set; the
  console host only on Windows; `HandleHook` sets the facts named above.
* `TestProbeNoEvent`: a fresh probe, and one fully drained, return the
  zero `ProbeEvent`.
* `TestEnvCaps…`: every existing `EnvCaps` test, unchanged.
* `TestColorProfileTokens`, `TestCapsJSONReadsOldProfile`,
  `TestReportSchemaVersion`.
* **`FuzzParseReply(raw string)`:** no panic; `Raw == raw`; a `ReplyColor`
  holds a non-nil colour; a `ReplyMode`'s `Setting` is 0–4.
* **`FuzzProbe(data []byte)`:** `data` read as a script of operations
  (start, reply bytes, timeout at a time, hook), times only rising: no
  panic; at most one `ProbeDone`; after `ProbeDone`, `Timeout` is false;
  every fact's origin is one of `Origin`'s values.
* `internal/conformance`: `enums_test.go` gains `termcap.ColorProfile`,
  `ProbeEventKind`, `ProbeHook`, `ReplyKind` and `ModeSetting`;
  `options_test.go`'s `termcap.Option` case targets `*termcap.Probe` with
  `termcap.WithGOOS("linux")`.

**Mutations:** the sentinel leaves unanswered queries unknown
(`TestProbeTranscripts`, `DA1Only`); `ParseReply` refuses BEL-terminated
OSC (`TestParseReply`); `#3a7` is scaled, not taken as the high bits
(`TestXParseColor`); `HandleTimeout` ignores the deadline
(`TestProbeTimeout`); `ProbeDone` is queued twice on a second DA1
(`TestProbeDuplicateDA1`); `WithoutModeQuery` is ignored
(`TestProbeBatch`); the console hook is queued on Linux
(`TestProbeHooks`); `ParseReply` reads `Pm` 5 as a setting
(`TestParseReply`); `termcap/probe.go` imports Bubble Tea
(`TestCoresAreCharmFree`, `charmfree`).

### Step 8: `termcaptea`, the Bubble Tea driver (D4, A2)

**Files:** `termcap/termcaptea/doc.go`, `prober.go`, `convert.go`,
`keyboard.go`, `profile.go`, and their tests;
`termcap/termcaptest/termcaptest.go`; the `termcap` tests that move;
`internal/conformance/options_test.go`; `termsvc/notify.go` (its
documentation); `docs/guides/terminal-capabilities.md`;
`docs/architecture.md`; `docs/glossary.md`. *Added 2026-10-24, by the
corrected fact:* `termcap/termcaptest/termcaptest_test.go` and
`launch/consumer_test.go` (each `termcap.NewProber` to
`termcaptea.NewProber`, and the profile compared as a `ColorProfile`);
`launch/restore.go` and `docs/guides/commands.md` (the documentation's
`*termcap.Prober` to `*termcaptea.Prober`).

* **`Prober`,** with today's exported surface: `NewProber(o
  ...termcap.Option) *Prober`, `Init`, `Update`, `Quit`, `Restore`,
  `Caps`, `SetTmux`, `IsReplyFragment`, each with today's documentation.
  It holds a `termcap.Probe` built with the caller's options and
  `WithoutModeQuery(2026, 2027)`, since Bubble Tea sends those itself.
  * `Init`: `nil` when disabled. Otherwise it records `t0 := time.Now()`
    and returns a `tea.Tick` to `t0 + timeout`, and the first `tea.EnvMsg`
    calls `Start(env, t0)`, so the deadline runs from `Init`, as it does
    today (`prober.go:215-220`), though the batch waits for the
    environment.
  * `Update(msg)`: `tea.EnvMsg` → `Start(termcap.Env(m), now)`;
    `tea.ColorProfileMsg` → `SetColorProfile(FromColorProfile(m.Profile))`;
    the probe's own tick → `HandleTimeout(now)`; a hook's answer →
    `HandleHook`; any message `convert.go` turns into a `Reply` →
    `HandleReply`. Then every event is drained: `ProbeWrite` →
    `tea.Raw(Data)`; `ProbeRunHook` → a command that runs `Run` and
    returns the answer as a message; `ProbeDone` → `CapsMsg`, held until
    `tea.ColorProfileMsg` has arrived or the probe timed out, as
    `prober.go:527-538` holds it; `ProbeColorScheme` → `ColorSchemeMsg`.
    While `Timeout()` is true, a `tea.Tick` to its deadline. The probe
    never hides a message, as today.
  * `Quit` and `IsReplyFragment`: today's code, `IsReplyFragment` over
    `Probe.Awaiting`.
* **`convert.go`:** `tea.BackgroundColorMsg` → `ReplyColor` 11;
  `tea.ForegroundColorMsg` → 10; `tea.KeyboardEnhancementsMsg` →
  `ReplyKittyKeyboard`; `tea.ModeReportMsg` → `ReplyMode`;
  `tea.TerminalVersionMsg` → `ReplyVersion`; through
  `internal/termevent`: DA1, DA2, DSR 997 and the Kitty graphics reply to
  their kinds, and an unknown sequence to `termcap.ParseReply(raw)`.
* **`keyboard.go`:** `KeyboardEnhancements(c termcap.Caps)
  tea.KeyboardEnhancements`, `ReportEventTypes` when
  `c.Keyboard().Request&2 != 0`.
* **`profile.go`:** `FromColorProfile` and `ToColorProfile`.
* **`termcaptest`:** `Terminal` gains `Reply(query string) string`, the
  bytes `answer` gives, exported for tests that drive a `termcap.Probe`
  directly; its `Run` uses `termcaptea.Prober`.
* **Tests moved** to `termcaptea`, by Rule 7: `prober_test.go`,
  `discipline_test.go`, `example_test.go`, and the Bubble Tea cases of
  `fuzz_test.go`, `identity_test.go`, `sanitize_test.go`,
  `report_golden_test.go` and `envcaps_test.go`. Each runs unchanged but
  for imports.
* **`TestCoreAndAdapterAgree`** (in `termcaptea`), the differential of A2.5:
  for each `termcaptest` profile, the `Caps` of `termcaptest.Run` over a
  model holding a `Prober`, and the `Caps` of a `termcap.Probe` driven
  directly through `Terminal.Reply` and `ParseReply` with the same
  options, are equal but for `Profile`, which only Bubble Tea detects,
  and with colours compared at 8 bits per channel, the precision of the
  colours Bubble Tea's decoded messages carry; the core's 16-bit values
  are pinned by `TestXParseColor`. `TestProbeTranscripts` of Step 7 is
  rewritten to use `Terminal.Reply`.
  *Deviation, 2026-10-24:* the TUI run's `KeyboardFlags` depends on
  whether Bubble Tea's first-render push of flag 1 reaches the terminal
  before the probe's `CSI ? u`, so the core is driven instead with the
  replies a fresh `Terminal` gives, through `Reply`, to the sequences the
  TUI run wrote, in its order (`Terminal.Sequences`). `Reply` keeps the
  flag stack as `answer` does. Every field but `Profile` is compared,
  `KeyboardFlags` included.
* **`TestColorProfileRoundTrip`:** for every `colorprofile` value from
  `Unknown` to `TrueColor`, `ToColorProfile(FromColorProfile(p)) == p`, and
  the token equals `strings.ToLower(p.String())`.
* **`TestKeyboardEnhancements`:** `Request` 2 gives `ReportEventTypes`, 0
  gives none, for each keyboard reason of `views.go`.
* `internal/conformance/options_test.go`: no new case; `termcap.Option`
  stays the one option type.
* `docs/guides/terminal-capabilities.md`: the prober from `termcaptea`; a
  new section, "Probe from a line-mode CLI", with the drive loop of A2.3,
  raw mode, `termkey`, and the pitfalls of A2.1.
* `docs/glossary.md`: the new names as built; `Prober`'s move.
* **Golden files:** the goldens under `termcap/testdata/golden` that hold
  a JSON report change `schema_version` to 2 and the profile's spelling;
  each is read and listed. A golden that changes otherwise is a deviation.
* **Mutations:** `convert.go` drops `tea.ModeReportMsg`
  (`TestCoreAndAdapterAgree`, `Kitty`); the adapter does not pass
  `WithoutModeQuery(2026, 2027)` (the moved batch test of
  `prober_test.go`); `CapsMsg` is sent before `ColorProfileMsg` (the moved
  profile-wait test); `IsReplyFragment` ignores `Awaiting` (the moved
  fragment test); `KeyboardEnhancements` ignores `Request`
  (`TestKeyboardEnhancements`).

### Step 9: `termsvc` and `termsvctea` (D5, A1.2)

**Moves and changes:**

* removed from `termsvc`, added to `termsvc/termsvctea`: `ActivityBeacon()
  tea.Cmd`; `Copy(c, text, o...) tea.Cmd`; `CopyContext(ctx, c, text, o...)
  tea.Cmd`; `LinkPolicy.Open` as `Open(p termsvc.LinkPolicy, rawURL string)
  tea.Cmd`; `Notifier.Notify` and `NotifyContext` as `Notify(n, x)` and
  `NotifyContext(ctx, n, x)`; `Notifier.Update` as
  `Observe(n *termsvc.Notifier, msg tea.Msg)`.
* added to `termsvc`: `(*Notifier).SetCaps(termcap.Caps)`,
  `(*Notifier).SetFocus(focused bool)`, `(*Notifier).Prepare(ctx, x)
  (seq string, deliver func() NotifyResultMsg)`, and
  `PrepareCopy(ctx, c, text, o ...CopyOption) (Route, func() CopiedMsg)`.

**`termsvc`, the core:**

* `Prepare` is today's `NotifyContext` up to its command
  (`notify.go:280-299`): clean, skip, and for the terminal `bytesFor`, on
  the caller's goroutine. With a backend and no skip, `seq` is `""` and
  `deliver` calls the backend under `bounded(ctx, n.timeout)`. With a skip,
  `seq` is `""` and `deliver` returns the skipped result. Otherwise `seq`
  is the encoded sequence and `deliver` returns `Sent: true`.
* `PrepareCopy` is today's `CopyContext` up to its command: over
  `MaxCopyBytes` gives `RouteNone` and `Failed`; with `WithClipboard`,
  `RouteBackend` and a `deliver` that calls the clipboard, bounded; else
  `RouteOSC52Tmux` inside tmux or `RouteOSC52`, and an `Unconfirmed`
  result.
* `SetCaps` and `SetFocus` set what `Update` set (`notify.go:255-265`).
* The `Notifier`'s documentation says a CLI calls `SetCaps` with
  `termcap.EnvCaps`, then `Prepare`, writes `seq`, and calls `deliver`.
* `.golangci.yml`: `charmfree` gains `**/termsvc/*.go`; `cores` gains
  `termsvc`.

**`termsvc/termsvctea`:**

* `Notify` and `NotifyContext`: `Prepare` on the caller's goroutine; with
  `seq` empty, `func() tea.Msg { return deliver() }`; else
  `tea.Sequence(tea.Raw(seq), func() tea.Msg { return deliver() })`.
* `Copy` and `CopyContext`: `PrepareCopy`; `RouteBackend` and `RouteNone`
  return `func() tea.Msg { return deliver() }`; `RouteOSC52` returns
  `tea.Sequence(tea.SetClipboard(text), deliver-as-cmd)`; `RouteOSC52Tmux`
  returns `tea.Sequence(tea.Batch(tea.SetClipboard(text),
  tea.Raw(termsvc.Wrap(c, ansi.SetSystemClipboard(text)))), deliver-as-cmd)`,
  as `clipboard.go:149-154` does today.
* `Open`: `nil` unless `p.Openable(rawURL)`, else `teamsg.Cmd(termsvc.OpenURLMsg{URL: rawURL})`.
* `Observe`: `termcap.CapsMsg` calls `SetCaps`, `tea.FocusMsg`
  `SetFocus(true)`, `tea.BlurMsg` `SetFocus(false)`.
* `ActivityBeacon`: today's `beacon.go:135` body.
* **Tests moved,** by Rule 7: the Bubble Tea cases of `services_test.go`,
  `termsvc_test.go` and `forms_test.go`.
* **New tests:** `TestPrepareMatchesSequence` (for each protocol and
  policy, `Prepare`'s `seq` equals `Sequence`'s, with the same numbering
  across calls); `TestBackendOffTheLoop` (a backend that blocks until the
  test releases it: `NotifyContext` returns its command without calling
  it, and the command delivers `Sent` once released);
  `TestPrepareCopyRoutes` (each route and the too-large case);
  `TestObserve`.
* **Callers:** `docs/guides/terminal-capabilities.md` (`n.Notify(…)` →
  `termsvctea.Notify(n, …)`, `termsvc.Copy` → `termsvctea.Copy`);
  `docs/architecture.md`.
* **Mutations:** `Prepare` numbers twice per call
  (`TestPrepareMatchesSequence`); the adapter calls `deliver` before
  returning (`TestBackendOffTheLoop`); `PrepareCopy` sends an over-long
  text (`TestPrepareCopyRoutes`); `Observe` ignores `tea.BlurMsg`
  (`TestObserve`); `termsvc/notify.go` imports Bubble Tea
  (`TestCoresAreCharmFree`).

### Step 10: `command` and `commandtea` (D6, A1.2)

**Moves and changes:**

* added to `command`: `type Effect func() any`, documented: its value is a
  message for the host, `nil` for none, or an `[]Effect` the host runs as
  one batch.
* changed in `command`: `Result.Cmd` from `tea.Cmd` to `Effect`;
  `Registry.Dispatch` returns `Effect`; `Registry.Watch(ctx
  context.Context) Effect` replaces `Watch()` and `WatchContext(ctx)`;
  `Registry.Attach(send func(any))`; `WithLoop(send func(any))`;
  `LoopMsg.Run() Effect`.
* added in `command/commandtea`: `Effect(c tea.Cmd) command.Effect` (nil
  stays nil); `Cmd(e command.Effect) tea.Cmd` (nil stays nil; a value that
  is an `[]command.Effect` becomes `tea.Batch` of each converted, run
  at once, as `tea.Batch(...)()` is today); `Dispatch(r, ctx, req)`;
  `Watch(r)`; `WatchContext(r, ctx)`; `Attach(r, send func(tea.Msg))
  func()`; *(2026-10-24, the owner's choice: the context first,
  `Dispatch(ctx, r, req)` and `WatchContext(ctx, r)`, as Go's convention,
  revive's `context-as-argument` and `termsvctea.NotifyContext` have it)*; `WithLoop(send func(tea.Msg)) command.RegistryOption`;
  `RunLoop(m command.LoopMsg) tea.Cmd`.

**`command`, the core:**

* `dispatch.go`: `Dispatch` returns, where it returned
  `teamsg.Cmd(ResultMsg{…})`, `func() any { return ResultMsg{…} }`; where
  it returned `tea.Batch(effects…)`, `func() any { return effects(…) }`
  with `effects` returning `[]Effect`; and for `Async`, an `Effect` that
  executes, audits and returns `effects(…)`. `onLoop`'s `LoopMsg` carries
  `func() Effect`.
* `registry.go`: `Watch(ctx)` is today's `WatchContext` body, returning
  `ChangedMsg`, or `[]Effect{changed, conflict}` where it returned a
  `tea.BatchMsg`, or `nil` when ctx ends; `WithLoop` and `loopState.send`
  take `func(any)`.
* `messages.go`, `handler.go`, `builtin.go`: no Bubble Tea and no
  `internal/teamsg`; `app.quit` returns
  `Result{Cmd: func() any { return QuitRequestMsg{} }}`.
* Package documentation: `Dispatch` "returns the effect", and a Bubble Tea
  host calls `commandtea.Dispatch`.
* `.golangci.yml`: `charmfree` gains `**/command/*.go`; `cores` gains
  `command`.

**Callers:**

* `workspace/commands.go`: the nine `command.Result{Cmd: x}` become
  `command.Result{Cmd: commandtea.Effect(x)}`.
* `launch/run.go:98`: `commandtea.Attach(o.registry, p.Send)`.
* Tests: `workspace/commands_test.go` (`r.Dispatch` at `:61` →
  `commandtea.Dispatch(r, …)`; `command.WithLoop(func(m tea.Msg) …)` at
  `:375` → `commandtea.WithLoop(…)`; `lm.Run()` at `:390` →
  `commandtea.RunLoop(lm)`), `launch/run_test.go:62` and
  `launch/consumer_test.go:74` (`return m, msg.Run()` →
  `return m, commandtea.RunLoop(msg)`).
* Tests moved to `commandtea`, by Rule 7: the Bubble Tea cases of
  `dispatch_test.go`, `loop_test.go`, `builtin_test.go`, `registry_test.go`,
  `result_test.go`, `sources_test.go` and `export_test.go`.
* `docs/guides/commands.md:110-146`: `commandtea.WithLoop(program.Send)`,
  `case command.LoopMsg: return m, commandtea.RunLoop(msg)`, and
  `commandtea.Dispatch`. `make examples` passes, its excerpts unchanged.
* `docs/architecture.md:85`.
* *Added 2026-10-24, by the corrected fact:* `workspace/bindings.go:271`
  (`return commandtea.Cmd(res.Cmd)`), all twelve sites of
  `workspace/commands.go`, and `docs/guides/keymaps.md:52`
  (`commandtea.Dispatch`).

**New tests:** `TestCmdBatches` (an `Effect` returning two `Effect`s
gives a `tea.BatchMsg` of two commands whose messages are theirs, in
order); `TestEffectNil`; `TestCoreDispatchWithoutTea` (a core-only test
that runs `Dispatch`'s `Effect` and its nested `[]Effect` by hand and
sees `ResultMsg` and `PromptMsg`).

**Mutations:** `Cmd` drops a nested `[]Effect` (`TestCmdBatches`);
`Effect(nil)` is a non-nil function (`TestEffectNil`); `Dispatch` runs an
`Async` command before returning (the moved async test of
`dispatch_test.go`); `Watch` returns the conflict without `ChangedMsg`
(the moved watch test); `commandtea.Attach` drops a message (the moved
loop test); `command/registry.go` imports Bubble Tea
(`TestCoresAreCharmFree`).

### Step 11: `termkey` (D7, A1.2)

**Files:** `termkey/doc.go`, `key.go`, `names.go`, `decode.go`,
`input.go`, and their tests; `internal/termevent/termkey_test.go`;
`internal/conformance/enums_test.go`; `.golangci.yml` (`charmfree` gains
`**/termkey/*.go`); `cores` gains `termkey`.

* **`key.go`:** `type Mod uint16` with `ModShift` 1, `ModAlt` 2, `ModCtrl`
  4, `ModMeta` 8, `ModHyper` 16, `ModSuper` 32, `ModCapsLock` 64,
  `ModNumLock` 128, `ModScrollLock` 256, from `internal/enum`'s `Bits`
  with tokens `shift`, `alt`, `ctrl`, `meta`, `hyper`, `super`,
  `capslock`, `numlock`, `scrolllock`. The key codes: `KeyExtended =
  unicode.MaxRune + 1` and each constant of ultraviolet's `key.go` const
  block, in the same order and with the same value, under the same name
  (`KeyUp`, …, `KeyF63`, the keypad, media and modifier keys), and the
  C0-named runes (`KeyBackspace`, `KeyTab`, `KeyEnter`, `KeyEscape`,
  `KeySpace`) as Bubble Tea defines them. `type KeyEvent struct{ Code rune;
  Mod Mod; Text string; ShiftedCode, BaseCode rune; IsRepeat, Release bool
  }` with `Keystroke() string` and `String() string`, as `tea.Key`'s.
* **`names.go`:** the name of each code, as ultraviolet's key table writes
  it, which `Keystroke` and `keymap`'s notation read.
* **`input.go`:** `type InputKind uint8` (`InputKey`, `InputPaste`,
  `InputUnknown`; tokens `key`, `paste`, `unknown`) and
  `type Input struct{ Kind InputKind; Key KeyEvent; Text string; Raw
  []byte; Truncated bool }`.
* **`decode.go`:** `type Decoder struct{ MaxPaste, MaxSequence int; … }`,
  zero value usable (`MaxPaste` 0 means 1 MiB, `MaxSequence` 0 means
  ~~1 KiB~~ 4 KiB, over `ParseReply`'s 1 KiB, and a string cut on an
  `ESC` leaves it out: deviation of 2026-10-24 in the execution record);
  `func (d *Decoder) Decode(p []byte, out []Input) []Input`;
  `func (d *Decoder) Pending() bool`;
  `func (d *Decoder) Flush(out []Input) []Input`; and
  `const DefaultEscTimeout = 50 * time.Millisecond`. It decodes MADR D7's
  list. A paste over `MaxPaste` keeps its first `MaxPaste` bytes, sets
  `Truncated`, and consumes the rest up to `CSI 201 ~`. A sequence over
  `MaxSequence` is one `InputUnknown` holding its first `MaxSequence`
  bytes (one fewer when the cut falls on an `ESC`), the rest consumed up
  to its end. `Flush` gives up a held lone
  `ESC` as the escape key, and a held incomplete sequence as
  `InputUnknown` with its bytes. The Kitty flags reply `CSI ? flags u` is
  `InputUnknown`, never a key.
* **Tests:**
  * a table per encoding of D7, each row bytes in and inputs out: UTF-8
    text, each C0 control, `ESC` + key as alt, every legacy CSI and SS3
    key with modifier codes 2 to 8, modifyOtherKeys, formatOtherKeys,
    Kitty with each modifier bit, event type and alternate key,
    associated text, bracketed paste, the flags reply, an OSC and a DCS
    reply;
  * `TestPasteBounded` and `TestSequenceBounded`;
  * `TestFlushEscape` (a lone `ESC`: `Pending` true, then `Flush` gives
    escape) and `TestSplit` (every input of the table split at every
    byte decodes to the same inputs);
  * `FuzzDecode(data []byte, split uint16)`: no panic; after `Decode` and
    `Flush`, `Pending` is false; decoding `x[:i]` then `x[i:]` then
    `Flush` equals decoding `x` then `Flush`.
  * `TestRepliesReachParseReply` (MADR A2.4): every row of Step 7's
    `TestParseReply`, fed to a `Decoder` whole and split at every byte,
    comes out as one `InputUnknown` whose `Raw`, given to
    `termcap.ParseReply`, gives the same `Reply` as the row. The test
    imports `termcap` (tests do not count, D1); `termkey` itself does not.
* **`internal/termevent/termkey_test.go`** (the one place ultraviolet may
  be imported):
  * `TestKeyCodesMatchUltraviolet`: each `termkey` code and modifier bit
    has ultraviolet's value, and for each code,
    `termkey.KeyEvent{Code: c}.Keystroke() == uv.Key{Code: c}.Keystroke()`;
  * `TestDecoderMatchesUltraviolet`: every input of `termkey`'s table that
    is a key or a paste is decoded by `uv.EventDecoder` as well, and the
    key's fields, or the paste's text, are equal; a row in the test's
    `differ` table, with its reason, is the only exception allowed, and
    the record lists each.
* **Mutations:** the modifier parameter is read without the minus one
  (the legacy table); `ESC` before a key is not alt (the alt row); the
  paste is not bounded (`TestPasteBounded`); `Flush` drops a lone `ESC`
  (`TestFlushEscape`); `Decode` forgets a sequence split across calls
  (`TestSplit`); `ModMeta` is 16 (`TestKeyCodesMatchUltraviolet`).

### Step 12: `keymap` and `keymaptea` (D7, A1.2)

**Moves and changes:**

* changed in `keymap`: `Stroke.Mod` from `tea.KeyMod` to `termkey.Mod`;
  `Stroke.Code` holds `termkey`'s codes; `Normalize(k termkey.KeyEvent, f,
  goos) (termkey.KeyEvent, bool)`; `Stroke.Matches(k termkey.KeyEvent)`;
  `Explanation.Raw` and `Normalized` are `termkey.KeyEvent`s and
  `(*Dispatcher).Explain` takes one; `Outcome.Replay` is a
  `[]termkey.KeyEvent`; `Settings.Tick` removed.
* removed from `keymap`: `(*Dispatcher).Update`; `(*Keymap).Help`;
  `(*Keymap).KeyBinding`; `Stroke.BubblesKeys`.
* added to `keymap`: `(*Dispatcher).Key(k termkey.KeyEvent, p Path, ctx
  when.Context) Outcome`; `(*Dispatcher).Expire(m TimeoutMsg) Outcome`;
  `Outcome.Wait time.Duration` and `Outcome.Timeout TimeoutMsg`;
  `(*Keymap).Shown(id command.ID, p Path) []Binding`.
* added in `keymap/keymaptea`: `Update(d *keymap.Dispatcher, msg tea.Msg,
  p keymap.Path, ctx when.Context) (keymap.Outcome, tea.Cmd)`;
  `FromTea(tea.Key) termkey.KeyEvent`; `ToTea(termkey.KeyEvent) tea.Key`;
  `Help(k *keymap.Keymap, p keymap.Path, d keymap.Describer, ids
  ...command.ID) help.KeyMap`; `KeyBinding(k, id, p, d) key.Binding`;
  `BubblesKeys(s keymap.Stroke) []string`.

**`keymap`, the core:**

* `matcher.go`: `Key` is today's `Update` for a key (release from
  `k.Release`), and where `Update` returned `d.tick()`, `Key` sets
  `Wait` (the keymap's timeout, one second when unset) and `Timeout`
  (`TimeoutMsg{gen}`); `Expire` is `Update`'s `TimeoutMsg` case.
* `normalize.go`, `notation.go`, `label.go`, `facts.go`, `whichkey.go`:
  `termkey` in place of `tea` throughout; the name tables read
  `termkey`'s names; `facts.go`'s `ctrlQ` and `ctrlI` use `termkey.ModCtrl`.
* `keymap.go`: `Shown` exports `shown`'s result as `[]Binding`, the
  entries' embedded `Binding`s, in the same order.
* `help.go` leaves the package. `.golangci.yml`: `charmfree` gains
  `**/keymap/*.go`; `cores` gains `keymap`.

**`keymap/keymaptea`:**

* `Update`: a `tea.KeyPressMsg` or `tea.KeyReleaseMsg` goes to `d.Key`
  with `FromTea` (release set for the latter); a `keymap.TimeoutMsg` to
  `d.Expire`; any other message gives a zero `Outcome`. When `o.Wait > 0`
  the command is `tea.Tick(o.Wait, func(time.Time) tea.Msg { return
  o.Timeout })`.
* `FromTea` and `ToTea` copy `Code`, `Mod`, `Text`, `ShiftedCode`,
  `BaseCode` and `IsRepeat` field for field.
* `Help`, `KeyBinding`, `BubblesKeys`: today's `help.go` and
  `Stroke.BubblesKeys`, over `k.Shown`.
* **Tests:**
  * moved, by Rule 7: `help_test.go`, and the Bubble Tea cases of
    `matcher_test.go`, `normalize_test.go`, `notation_test.go` and
    `whichkey_test.go`;
  * `TestTeaConversion`: every `Key…` constant Bubble Tea's `key.go`
    declares, listed in the test as the execution reads that file (the
    count recorded), and every modifier, convert both ways unchanged, and
    `FromTea(k).Keystroke() == k.Keystroke()` for a table of keys with
    text, shifted and base codes;
  * `TestReplayIsLossless`: a chord that ends without a match replays
    `tea.KeyPressMsg`s equal to the ones sent.
* **Callers:** `workspace/bindings.go:221` calls `keymaptea.Update`, and
  replays `tea.KeyPressMsg(keymaptea.ToTea(r))`; `workspace/help.go:33`
  calls `keymaptea.KeyBinding`; `workspace/bindings_test.go:174`'s
  `chordKeymap` sets `Settings{Timeout: time.Nanosecond}` and drops its
  `tick`; `docs/guides/keymaps.md:165-180` (`keymap.Normalize` takes a
  `termkey.KeyEvent`; `keymaptea.KeyBinding`, `keymaptea.Help`;
  `Explain`); `docs/architecture.md`.
* **The notation does not change:** every `ParseStroke` and `String`
  case of `notation_test.go`, every `LoadVSCode` and `ExportVSCode` case
  of `vscode_test.go`, and every golden under `keymap/testdata/golden`,
  passes unchanged.
* **Mutations:** `FromTea` drops `BaseCode` (`TestTeaConversion`);
  `Key` sets no `Wait` for a prefix (the moved chord test);
  `Expire` ignores the generation (the moved stale-timer test); `Shown`
  returns entries in reverse (the moved help-order test); `Normalize`
  maps ctrl+5 to ctrl+5 (`normalize_test.go`); `keymap/notation.go`
  imports Bubble Tea (`TestCoresAreCharmFree`).

### Step 13: release `v0.14.0`, and the close-out

* **Docs:**
  * `docs/guides/migrating.md`, "Migrating to `v0.14.0`": the migration
    table of D10, one row for each moved or retyped name of Steps 7 to 12
    (old, new, the one-line change), and *Changed*: the JSON report's
    `schema_version` 2 and the profile's spelling;
  * `docs/architecture.md`: the cores, the adapters and `termkey`, with
    their imports;
  * `docs/glossary.md`: every new name as built; the moves;
  * the four adapters' guides' sections, already changed by their steps,
    read through once more;
  * `README.md` Status; `docs/README.md` rows.
* **The release:** as Step 6, with `apicheck` against `v0.13.0`
  reporting exactly the `scripts/apicheck.allow` blocks of Steps 7 to 12;
  the owner's push, CI, the annotated tag `v0.14.0`, CI on the tag.
* **The consumer smoke test,** in a scratch module with `GOWORK` empty,
  against `v0.14.0` from the proxy: a program that reads
  `termcap.EnvCaps` and writes `termcap.Report`, writes a
  `termsvc.Notifier`'s `Prepare` bytes, parses a slash line and writes a
  result with `command`, loads a `keybindings.json` with `keymap`,
  decodes bytes with `termkey`, and runs a `termcap.Probe` to `ProbeDone`
  over a fixed transcript of Kitty's replies, read through `termkey` and
  `ParseReply`, checking that `Caps.KittyKeyboard` is `Supported`; built
  and run; `go list -deps` over it,
  checked with Step 2's `charm` rule: no Charm but `x/ansi`. Negative
  control: the same program importing `termcap/termcaptea` shows
  `charm.land/bubbletea/v2`. A second program, a Bubble Tea one, runs a
  `termcaptea.Prober`, `commandtea.Dispatch` and `keymaptea.Update` over a
  `workspace`, under staticcheck.
* **The close-out:**
  * `scripts/apicheck.allow`'s blocks of Steps 7 to 12 are removed, and
    `make apicheck` against `v0.14.0` is clean;
  * 0020-MADR: D12's releases as tagged; the MADR's More Information
    names the two tags;
  * this PLAN becomes `complete`, and its `docs/README.md` row says so;
  * the issue comments for #2 to #5 drafted in the record for the owner.

## Verification

* `TestCoresAreCharmFree` passes with `cores` holding all twelve packages
  (`command`, `fuzzy`, `glyph`, `internal/enum`, `internal/sanitize`,
  `keymap`, `layout`, `safetext`, `termcap`, `termkey`, `termsvc`, `when`)
  for `linux`, `darwin` and `windows`, and each mutation of Step 2 is
  killed.
* `make lint` passes with the `charmfree` rule covering the same twelve.
* `python3 -m tools.verify gates` passes on every host at the end of every
  step, with the summary lines in the record.
* `make apicheck`: 0 incompatible changes through `v0.13.0`; exactly the
  listed blocks from Step 7 to Step 12; clean against `v0.14.0` after the
  close-out.
* Every mutation of every step is killed, each with its failure line in
  the record.
* `safetext`'s allocation assertions pass, and its benchmarks are
  recorded; `internal/sanitize`'s after-medians are within 1.10 of the
  before-medians, with no added allocation.
* `FuzzClean`, `FuzzAgainstV0120`, `FuzzParseReply`, `FuzzProbe` and
  `FuzzDecode` run in `make fuzz` and CI's fuzz step with no failure.
* `TestCoreAndAdapterAgree` passes for every `termcaptest` profile:
  `termcap.Probe` driven directly and `termcaptea.Prober` under Bubble Tea
  reach the same `Caps`.
* `TestRepliesReachParseReply` passes: every reply `termcap` parses
  reaches it whole through `termkey`, split at any byte.
* `TestDecoderMatchesUltraviolet` and `TestKeyCodesMatchUltraviolet` pass,
  with the `differ` table in the record.
* `make examples` passes, its excerpts unchanged.
* Both consumer smoke tests pass, with their negative controls seen to
  fail.
* `python3 -m tools.verify scan` is clean at each handoff.

## Rollout and Rollback

* **Rollout.** Two tags, `v0.13.0` then `v0.14.0`, each the owner's,
  each after CI is green on `main`. gobble adopts each by its own
  0032-MADR D7 steps; this repository changes nothing there.
* **Before a tag,** a step is undone by reverting its commit on `main`.
  Steps 7 to 12 are in order: reverting one reverts every later one first,
  since each later core imports the earlier (`termsvc` imports `termcap`;
  `keymap` imports `termcap`, `command` and `termkey`).
* **After a tag,** tags are never moved or deleted (AGENTS.md, "Modules").
  A defect is fixed in a patch release, `v0.13.x` or `v0.14.x`. A move
  that proves wrong is reversed in a later minor release, under API rule 1
  in the other direction: the name returns to the core where it can, or a
  record grants the exception again.
* **`v0.13.0`'s behaviour change** (U+FFFD for invalid UTF-8) is reverted,
  if it must be, by changing `safetext`'s S2 in a patch release; the
  oracle test of Step 5 then shows the old behaviour again.

## Execution Record

### Step 1: records (2026-10-23)

* **Approval:** the owner, 2026-10-23: "i accept the design, i approve
  executing the plan, proceed". MADR A2 is `accepted`; this PLAN is
  `in-progress`.
* **Prerequisites checked:** `main`'s head is `63c68d6`; the latest tag is
  `v0.12.0`; `scripts/apicheck.allow` holds no entry; the hooks path is the
  global hooks directory.
* **Records changed:**
  * `0020-MADR-charm-free-cores-glyphs-and-safetext.md`: A2's status
    `accepted`, with the owner's words;
  * this PLAN: `in-progress`, and this record;
  * `0008-PLAN-command-palette.md`: the dated note of Step 1 (Steps 5 to 9
    wait; the release after `v0.14.0`);
  * `0009-PLAN-streaming-content-engine.md`: the dated note of Step 1
    (Step 2 carried out by this PLAN's Step 4);
  * `docs/glossary.md`: "Reserved by accepted records" lists the sixteen
    names and `Prober`'s move;
  * `docs/README.md`: the MADR's and this PLAN's rows.
* **Names check,** `python3 -m tools.verify names` over the sixteen
  names, exit status 1:

  ```text
  names: termcap/termcaptea.Prober: also in termcap
  names: 1 clash(es) in 16 planned name(s)
  ```

  The one clash is the move of D4 and A2: `termcap.Prober` leaves
  `termcap` in Step 8, the step that adds `termcaptea.Prober`.
* **Checks:** markdownlint over the tree, as CI's docs job runs it; the
  identifier scan; a scan for raw bidi, format and control characters
  over the changed records. Their results are in the handoff.
* **Records only:** no source, test, CI or configuration file changed.
* Committed by the owner as `44a4890`.

### Step 2: the closure gate (2026-10-23)

* **Facts re-read:** `git diff 63c68d6 HEAD` over `internal/conformance`,
  `.golangci.yml` and the six cores is empty; the facts table stands.
* **`internal/conformance/charmfree_test.go`** (new): `cores` (`fuzzy`,
  `glyph`, `internal/enum`, `internal/sanitize`, `layout`, `when`);
  `charm`; `TestCharmPredicate`; `listDeps`, one
  `go list -e -deps` per GOOS over every core, so a missing core comes back
  as an error line rather than failing the run; `charmChains`, the
  breadth-first walk, which stops at a Charm package so each chain names
  the Charm package a non-Charm one imports; `TestCoresAreCharmFree`,
  one subtest per GOOS.
* **`.golangci.yml`:** the `charmfree` rule after `xterm`, as the step
  gives it, each `deny` with a `desc` naming 0020-MADR D11 (D2 for
  `internal/teamsg`), and a comment that its file list and `cores` stay
  the same.
* **On the tree:** `TestCharmPredicate` and `TestCoresAreCharmFree` pass,
  linux, darwin and windows; `golangci-lint` over the six cores reports
  `0 issues.`
* **Mutations,** through `python3 -m tools.verify mutate`, each killed:

  | Mutation | Killed by |
  | :--- | :--- |
  | `glyph/glyph.go` imports `_ "charm.land/bubbletea/v2"` | `TestCoresAreCharmFree`, for each GOOS: `glyph (GOOS=linux) links charm.land/bubbletea/v2: glyph → charm.land/bubbletea/v2` |
  | `internal/enum` imports the same | `TestCoresAreCharmFree`: `fuzzy (GOOS=linux) links charm.land/bubbletea/v2: fuzzy → internal/enum → charm.land/bubbletea/v2`, and the same for `glyph`, `layout`, `when` and `internal/enum` itself |
  | `cores` gains `nosuch` | `TestCoresAreCharmFree`: `cores lists nosuch, which go list does not find (stat …/nosuch: directory not found)` |
  | `charm` lets any `github.com/charmbracelet/x/` path through | `TestCharmPredicate`: `charm("github.com/charmbracelet/x/term") = false, want true`, and `x/ansix` |
  | `glyph/glyph.go` imports and uses Bubble Tea (lint) | depguard: `glyph/glyph.go:17:2: import 'charm.land/bubbletea/v2' is not allowed from list 'charmfree': no Bubble Tea in a Charm-free core; …` |
  | `charmfree` also covers `**/termcap/*.go` (lint over `./termcap/...`) | depguard, four issues: `termcap/caps.go:9:2` (`colorprofile`), `termcap/prober.go:13:2` (Bubble Tea), `termcap/prober.go:17:2` (`internal/teamsg`), `termcap/views.go:7:2` (Bubble Tea) |

  *Replaced, Rule 4:* the step's lint mutation as first written, a blank
  import `_ "charm.land/bubbletea/v2"` in `glyph/glyph.go`, failed lint
  through revive's `blank-imports` rule alone (`a blank import should be
  only in a main or test package…`), not through depguard, so it proved
  nothing about `charmfree`. It was replaced by a named import that a
  type alias uses, which depguard refuses (the fifth row; `unused` also
  reports the alias).
* **Controls,** read from the last mutation's output: no file under
  `termcap/termcaptest/`, whose `termcaptest.go` imports Bubble Tea and
  `colorprofile`, is flagged, so `*` does not reach a sub-package; and no
  `termcap` test file, several of which import Bubble Tea and
  `colorprofile`, is flagged, so `!$test` leaves tests out.
* **Gates,** `python3 -m tools.verify gates` over the step's two files,
  nothing skipped:
  * macOS: `gates: 19 passed, 0 failed, 0 skipped in 446s`;
    `release-check` "242 file(s) clean"; `make apicheck`
    "against v0.12.0, 0 incompatible change(s)";
  * the Linux host: `gates: 19 passed, 0 failed, 0 skipped in 981s`;
  * the Windows host: `gates: 19 passed, 0 failed, 0 skipped in 1982s`.

  The hosts' `release-check` counts 243 files, their copies' index
  tracking the new test file.
* Committed by the owner.

### Step 3: `glyph`, the eight fields (2026-10-23)

* **The emoji check** (the MADR left `␛` to this step): Unicode's
  `emoji-data.txt`, "Version: 18.0.0", dated 2026-01-30, downloaded on
  2026-10-23 and read by a script as data. U+2713, U+2717, U+25CF,
  U+25CB, U+25B8, U+2501, U+254C and U+241B have no property in it; the
  control, U+2714, reads `Emoji, Extended_Pictographic`. So none of the
  eight glyphs is an emoji.
* **`glyph/glyph.go`:** `Set` gains `Done`, `Failed`, `Running`,
  `Pending`, `Heading`, `MeterFill`, `MeterTrack` and `Control` after
  `Mask`, with their comments (D8, and 0009-MADR §1 for `Control`);
  `Unicode()` and `ASCII()` give the step's table. `TierLegacy` gives
  `ASCII()`, unchanged.
* **`glyph/glyph_test.go`:** the walked count in
  `TestEveryGlyphIsOneCell` and `TestTierWidths` goes from 70 to 78
  ("plus 26 single glyphs"); `TestStatusAndMeter` pins the table, that
  each field's two forms differ, and that the four status glyphs are
  four strings in each set; `TestControlIsNotEmoji` pins U+241B, the code
  point checked above. The existing reflection tests cover the new fields
  unchanged.
* **On the tree:** `go test ./...` passes in every package; no golden
  file changed, since no view draws the new fields yet.
* **Mutations,** each killed:

  | Mutation | Killed by |
  | :--- | :--- |
  | `ASCII().Running` is `-` | `TestStatusAndMeter`: `ASCII: the status glyph "-" stands for two states` |
  | `Unicode().Done` is `✔` | `TestStatusAndMeter`: `Unicode: {done:✔ …}, want {done:✓ …}` |
  | `Unicode().MeterTrack` is `╌╌` | `TestEveryGlyphIsOneCell`: `Unicode.MeterTrack = "╌╌" is 2 cells, want 1` |
  | `ASCII().Control` is `␛` | `TestASCIISetIsASCII`: `ASCII.Control = "␛" holds U+241B, which is not printable ASCII` |
  | `Unicode().Control` is `⎋` U+238B, a code point not checked | `TestControlIsNotEmoji`: `Unicode().Control = "⎋" ([U+238B]), want U+241B, …` |

  The fifth was added to the step's four so that the new
  `TestControlIsNotEmoji` is seen failing too.
* **Gates,** over `glyph/glyph.go` and `glyph/glyph_test.go`, nothing
  skipped: macOS `gates: 19 passed, 0 failed, 0 skipped in 412s`; the
  Linux host `… in 814s`; the Windows host `… in 1954s`; `release-check`
  "243 file(s) clean" on each; `make apicheck` "against v0.12.0, 0
  incompatible change(s)".
* Committed by the owner.

### Step 4: `safetext` (2026-10-23)

* **The package,** 7 files: `doc.go` (the grammar and what survives, the
  stability line), `rules.go` (`Rules`, the five presets,
  `DefaultMaxString`), `clean.go` (the parser, `Clean`, `IsClean`),
  `utf8.go` (maximal-subpart decoding), `escape.go` (the removed sets,
  `Escape`'s form), `sanitizer.go` (`Sanitizer`, `NewSanitizer`,
  `ErrClosed`), `url.go` (`SafeURL`, `URLs`). Imports: the standard
  library and `glyph`. `charmfree` covers `**/safetext/*.go`, and `cores`
  holds `safetext`; `internal/conformance` passes.
* **Exported beyond the step's list:** `DefaultMaxString`, the constant
  `MaxString`'s zero means, and `ErrClosed`, `Write`'s error after
  `Close`. Neither is a type, so the glossary does not list them.
* **The grammar, as built.** Reading x/ansi v0.11.8 showed two
  tokenizers, not one: `ansi.Strip` runs the DEC transition table
  (`parser/transition_table.go`), and `ansi.DecodeSequence` a simpler
  decoder; `internal/sanitize`'s `Line` uses the first and `Styled` the
  second, and they split malformed input differently (a C0 inside a CSI,
  an intermediate then a parameter, a DEL, UTF-8 inside an APC).
  `safetext` follows the transition table, as 0009-MADR §2 names it, with
  §2's rules where they differ from it:
  * a C1 code point is its 7-bit form (x/ansi prints U+009B as a
    character);
  * BEL ends a DCS, SOS, PM or APC as well as an OSC (x/ansi keeps it as
    data);
  * a string past `MaxString` is abandoned, and the byte past it is text;
  * every byte of a sequence is removed, so UTF-8 inside an SOS, PM or APC
    is part of it (x/ansi's table leaves such a string and prints the
    rest).

  Where §2 is silent, the table's own recovery is kept: a C0 control
  inside a CSI or an escape is executed as text and the sequence goes on;
  DEL inside one is ignored; a parameter byte after an intermediate ends a
  CSI there; an ESC right after a DCS's introducer is data. Step 5's
  differential against `Styled` will meet the classes where
  `DecodeSequence` splits malformed input differently; they are named
  there.
* **Invalid UTF-8.** The Unicode Standard 18.0's §3.9.6 has four worked
  examples, Tables 3-8 to 3-11, read from the specification on
  2026-10-23. The PLAN's "Table 3-8's example" is Table 3-8, non-shortest
  forms; `TestUnicodeTables` pins all four, and the MADR's two, `E2 82`
  and `C0 AF`.
* **Tests:** `TestSequences` (0009-PLAN Step 2's classes, the codex string),
  `TestControls`, `TestRemovedSet` (S1), `TestFlatten` (S3), `TestEscape`
  (S4, each escape as `strconv.QuoteToASCII`), `TestKeepSGR`,
  `TestVisible`, `TestMaxString`, `TestIsClean`, `TestCleanReturnsItsInput`
  (the same string data), `TestSanitizerClose`, `TestUnicodeTables`,
  `TestMaximalSubparts` (decode against a plain Table 3-7 over every one-,
  two- and three-byte string of 0x80–0xFF and every lead byte with every
  second byte), `TestDecodePartial`, `TestSafeURL`, `TestURLs`,
  `TestAgainstStrip` (0009-PLAN Step 2's differential, 2000 generated
  inputs from a fixed seed), `TestAllocations`. Every table case also runs
  through a `Sanitizer` split at every byte, and through `IsClean`.
* **`FuzzClean`:** 0009-PLAN Step 2's four properties and `IsClean`'s,
  and a fifth: a `Sanitizer` written one byte at a time equals `Clean`.
  Run locally for 120 s (9.2 million inputs) and, after the fast path
  below, for 180 s (30.9 million inputs), with no failure.
* **Performance, and what changed for it.** The first build met the
  allocation bar but not the speed Step 5 needs. That step's bar is
  `internal/sanitize` at the current commit, measured on this machine
  (a scratch copy of `HEAD` with the benchmark added; Apple M1 Pro,
  median of five):

  | 4 KiB input | `Line` | `Styled` | `HasControl` |
  | :--- | ---: | ---: | ---: |
  | clean ASCII | 1.93 µs | 1.92 µs | 1.92 µs |
  | clean CJK and emoji | 8.17 µs | 8.36 µs | 8.17 µs |
  | tool output | 37.75 µs | 46.73 µs | 39.39 µs |
  | styled | 17.69 µs | 13.32 µs | 17.82 µs |
  | hostile | 16.46 µs | 32.34 µs | 16.51 µs |

  The changes, each with the tests and the fuzz above passing: the
  machine is readied in place rather than copied; text that passes as it
  is goes out in one piece without being compared with itself; the
  standard library's decoder reads well-formed UTF-8, and Table 3-7's
  subparts are worked out only for the rest; a string's printable content
  and a CSI's parameters are taken in bulk; a whole CSI or OSC of the
  plainest form is decided in one step, as the parser would decide it,
  which the byte-at-a-time fuzz property holds; the link check is a
  scan of RFC 3986's scheme and authority, not `net/url`, which allocated
  on every link (`StatusLine` on the styled input made 261 allocations
  before it). `Rules.LinkSchemes`'s documentation says the rule exactly.
* **Benchmarks, after** (median of five, the same machine):

  | 4 KiB input | `Line()` | `Rules{Flatten, KeepSGR}` | `IsClean(Line())` |
  | :--- | ---: | ---: | ---: |
  | clean ASCII | 2.04 µs, 0 allocs | 2.04 µs, 0 allocs | 2.02 µs |
  | clean CJK and emoji | 8.64 µs, 0 allocs | 8.63 µs, 0 allocs | 8.60 µs |
  | tool output | 9.70 µs, 1 alloc | 10.86 µs, 1 alloc | 0.11 µs |
  | styled | 6.52 µs, 1 alloc | 6.31 µs, 0 allocs | 0.10 µs |
  | hostile | 14.38 µs, 1 alloc | 15.21 µs, 1 alloc | 0.17 µs |

  1 MiB of prose: 0.91 ms, 0 allocations; 1 MiB half escapes: 1.58 ms,
  1 allocation. A warm `Sanitizer.Write` allocates nothing on any input.
  Each is within 1.10 of the bar, clean ASCII the nearest (2.04 µs
  against 2.11 µs for `Styled`'s); Step 5 measures the wrappers
  themselves.
* **Mutations,** through `python3 -m tools.verify mutate`:

  | Mutation | Killed by |
  | :--- | :--- |
  | BEL no longer ends an OSC (the parser) | `TestSequences`: `U+009D is OSC: Clean("a\u009d0;t\ab") = "a", want "ab"` |
  | encoded C1 code points pass through | `TestSequences`: `U+009D is OSC: … = "a\u009d0;tb", want "ab"` |
  | the `Sanitizer` forgets its state between writes | `TestSequences`: `Sanitizer split at 1 of "\x1b]0;x\x1b\\\x1b\\b" gives "]0;xb"; Clean gives "b"` |
  | `MaxString` is ignored (the parser) | `TestMaxString`: `Clean = "a", want a6789b` |
  | `MaxString` is ignored (the whole-sequence path) | `TestMaxString`: `Clean = "ab", want a6789b: the string abandoned before its BEL` |
  | `SafeURL` accepts any scheme | `TestSafeURL`: `SafeURL("javascript:alert(1)") = …, true; want false` |
  | invalid UTF-8 is one U+FFFD per byte | `TestUnicodeTables`: `Table 3-11: … = "��������A", want "����A"`; `TestMaximalSubparts` |
  | `Flatten` turns each break, not each run, into a space | `TestFlatten`: `Line: Clean("a\n\x00\nb") = "a  b", want "a b"` |
  | `Escape` drops instead of escaping | `TestEscape` |
  | `KeepSGR` keeps 33 parameters | `TestKeepSGR`: `33 parameters: …` |
  | `KeepSGR` keeps an SGR with a prefix | `TestKeepSGR`: `a prefix is not an SGR: Clean("\x1b[?1mx") = "\x1b[?1mx", want "x"` |
  | `IsClean` true when only a prefix survives | `TestIsClean`: `IsClean("a\u200b", Text) = true, want false` |
  | the whole-sequence path keeps any OSC | `TestKeepSGR`: `other OSC removed: Clean("\x1b]0;title\ax") = "\x1b]0;title\ax", want "x"` |

  *Replaced, Rule 4:* "BEL no longer ends an OSC (the whole-sequence
  path)" survived. It is an equivalent mutant: with BEL refused there, the
  OSC falls to the parser, which removes it the same way, so only speed
  changes. It was replaced by the last row, a change the path's output
  shows. `TestIsClean` was added so that the step's `IsClean` mutation
  had a test that can see it: a removal at the very end, where the output
  is a prefix of the input.
* **Found while building, and fixed:** `TestMaxString`'s first expectation
  counted wrong (the hidden 8 bytes are `0;012345`); `TestAllocations`'s
  first form took the styled input for dirty under `StatusLine`, where it
  is clean; the editing tools turned `\uXXXX` escapes into the raw
  characters in ten places, bidirectional controls among them, which a
  scan of every new file found and a script turned back into escapes;
  none is left in any file of the step.
* **Lint:** `golangci-lint` for linux, darwin and windows, 0 issues, after
  five findings were fixed (an `errors.Is`, a one-case switch, two De
  Morgan rewrites, an unused helper); `modernize` asked for one range loop
  in a test.
* **Gates,** `python3 -m tools.verify gates` over the step's 15 files
  (`safetext/*.go`, `internal/conformance/charmfree_test.go`,
  `.golangci.yml`). The first run failed on every host at `modernize`
  alone (`go fix` asked for `for c := range rune(0x20)` in
  `TestControls`), which was applied; the second, nothing skipped: macOS
  `gates: 19 passed, 0 failed, 0 skipped in 384s`; the Linux host
  `… in 811s`; the Windows host `… in 1994s`. `release-check` is clean on
  each (243 files on macOS, 256 on the hosts, whose copies' index tracks
  the new files); `make apicheck` "against v0.12.0, 0 incompatible
  change(s)".
* Committed by the owner.

### Step 5: `internal/sanitize` wraps `safetext` (2026-10-23)

* **Benchmarks, before,** on the unchanged implementation, with
  `bench_test.go` added: `go test -run '^$' -bench . -benchmem -count=10
  ./internal/sanitize`, Apple M1 Pro, the median of ten:

  | 4 KiB input | `Line` | `Styled` | `HasControl` |
  | :--- | ---: | ---: | ---: |
  | clean ASCII | 1.94 µs, 0 allocs | 1.96 µs, 0 allocs | 1.93 µs, 0 allocs |
  | clean CJK and emoji | 8.23 µs, 0 allocs | 8.44 µs, 0 allocs | 8.19 µs, 0 allocs |
  | tool output | 38.20 µs, 9 allocs | 47.53 µs, 1 alloc | 37.85 µs, 9 allocs |
  | styled | 18.08 µs, 7 allocs | 13.54 µs, 0 allocs | 17.75 µs, 7 allocs |
  | hostile | 16.76 µs, 8 allocs | 32.56 µs, 1 alloc | 16.49 µs, 8 allocs |

* **The oracle.** `oracle_test.go` was written by a script from
  `git show v0.12.0:internal/sanitize/sanitize.go`, which renamed the ten
  identifiers of `Line`, `Styled`, `HasControl` and their helpers in code
  only, never in a comment or a string. `sanitize.go` had not changed since
  the tag. Beyond the step's text, the package comment, `Truncate`,
  `byRune` and `Token` are left out: the package keeps them, and a copy
  would not compile beside them. The file's comment says so.
* **Deviation (2026-10-23): the differential found six classes beyond
  S2.**
  * *Found.* With the wrapper in place, `internal/sanitize`'s tests failed
    on four rows, and the tree's on three assertions in two other
    packages:
    * `sanitize_test.go:23`, "C1, as UTF-8": `Line("a\u0085\u009bb") =
      "a", want "ab"`;
    * `sanitize_test.go:24`, "invalid UTF-8": `"a�b�", want "ab"`;
    * `sanitize_test.go:54`, "an 8-bit CSI": `Styled("a\x9b31mb") =
      "a�31mb", want "a31mb"`;
    * `sanitize_test.go:60`, "an nF escape dropped":
      `Styled("000\x1b X0000000") = "0000000000", want "000"`;
    * `termcap/json_test.go:96`: `ParseTmux` gave the version
      `"3.4�"`;
    * `termsvc/termsvc_test.go:239`: `strip left UTF-8 C1 controls:
      "abcd"`;
    * `termsvc/termsvc_test.go:249`: the link's text `"tet�"`, want
      `"tet"`.

    `FuzzAgainstV0120`'s seed `"a\xffb\xc3"` failed too:
    `HasControl = true; v0.12.0 gives false`. That is the PLAN's equation
    at fault, since `oracleHasControl(f(x))` is false where `v0.12.0`'s
    own `HasControl(x)` is true. No golden file changed.
  * *Classified* on a scratch copy of the tree, never the tree: 400,000
    inputs from a hostile alphabet, from a fixed seed, each difference
    shrunk to a minimal input. That gave 1,633 minimal inputs, in six
    classes, MADR A3.1's table:

    | # | Class | `Line` | `Styled` |
    | :--- | :--- | ---: | ---: |
    | 1 | C1 read as its 7-bit form | 289 | 406 |
    | 2 | BEL ends a DCS, SOS, PM or APC | 304 | 47 |
    | 3 | non-ASCII inside a DCS, SOS, PM or APC | 32 | 28 |
    | 4 | a character's 0x9C byte does not end an OSC | 43 | 43 |
    | 5 | `DecodeSequence`'s split of malformed input | 0 | 440 |
    | 6 | a C1 SGR or OSC 8 kept, rewritten | 0 | 2 |

    Each class 5 input's new `Styled` equals `v0.12.0`'s `Line`, 440 of
    440. Apart from class 6's two, no difference puts a control
    character in the output. `HasControl` compared with
    `oracleHasControl(x)` differs on none of the inputs.
  * *Resolutions offered,* each recommended first:
    1. On the classes:
       * (a) accept them as the grammar and restate the check;
       * (b) a model oracle that rewrites each input before `v0.12.0`
         sees it;
       * (c) make `safetext` match `v0.12.0`, which would reverse
         0009-MADR §2.
    2. On class 6:
       * (a) remove such a sequence whole;
       * (b) keep it rewritten.
    3. On the files outside the step:
       * (a) all three in this step;
       * (b) the tests now, and the comment in Step 9.
  * *Chosen* by the owner, 2026-10-23: (a) for each. The MADR records it
    as A3, and S5 carries a note. The step's notes above give the files
    added, the restated check and the mutations added.
  * *Before restating,* `Line(Styled(x)) == Line(x)` was tried on the
    scratch copy over 400,000 inputs and held on all. The predicates were
    tried over 2,000,000 inputs from two seeds and left no difference
    outside them. The first form missed two shapes, each found there and
    fixed:
    * a C0 or DEL between ESC and a string's introducer, which the
      grammar executes without leaving the escape;
    * a private marker after a parameter.

    The comparison covered about 40% of the inputs for `Line` and 27% for
    `Styled`. The two exact properties hold on every input.
* **Found while running the restated check, and fixed in the test.**
  `FuzzAgainstV0120` found `"0…0\x1b\x1e]ל0"` after 2.0 million inputs.
  `ESC RS ]` is an OSC, since the grammar executes the C0 without leaving
  the escape, and `ל` (U+05DC) is `d7 9c`, so this is class 4. The OSC
  clause had not skipped a C0 or DEL after ESC, as the string clause did.
  Both now share `introduced`, and the input is a seed. The step's note
  above now names the C0 and DEL in the OSC clause too, marked as added.
  `safetext` was not at fault.
* **The wrapper,** `sanitize.go`:
  * `Line` is `safetext.Clean(s, safetext.Line())`;
  * `Styled` is `safetext.Clean(s, styled)`, with
    `styled = safetext.Rules{Flatten: true, KeepSGR: true}`;
  * `HasControl` is `!safetext.IsClean(s, safetext.Line())`;
  * `Truncate`, `byRune` and `Token` are unchanged, byte for byte;
  * the package comment says that `safetext` does the cleaning and why
    `Truncate` and `Token` stay; `Line`'s and `HasControl`'s say invalid
    UTF-8 is U+FFFD;
  * the helpers `dropped`, `isBreak`, `text`, `plain`, `styledClean`,
    `kept` and `maxSGRParams` are gone; `x/ansi` stays for `Truncate`.
* **`safetext`, for MADR A3.3:** `machine.fromC1` is set while a C1 code
  point's 7-bit form is parsed. A CSI it opens is not a kept SGR, an OSC
  it opens is not a kept OSC 8, and an OSC 8 it ends is removed.
  `Rules.KeepSGR`'s documentation says so. Step 4's `TestKeepSGR` row
  "8-bit SGR, written 7-bit" (`"\u009b31mx"` to `"\x1b[31mx"`) pinned
  the behaviour A3.3 reverses; it is replaced by four rows: an SGR
  U+009B introduces, a link U+009D introduces, a link U+009C ends, and a
  7-bit link after a C1 SGR, which is kept.
* **Tests changed,** each with a comment naming its rule:
  * `sanitize_test.go`: "C1, as UTF-8" now holds U+0085 alone, and a new
    row "U+009B is CSI" gives `"a"` (class 1); "invalid UTF-8" gives
    `"a�b�"` (S2); "an 8-bit CSI" is now "a raw 0x9B byte",
    `"a�31mb"` (S2); "an nF escape dropped" gives `"0000000000"`
    (class 5); `TestDepartures` is new;
  * `termcap/json_test.go`: `ParseTmux`'s version is `"3.4�"` (S2);
  * `termsvc/termsvc_test.go`: `strip` of the encoded C1 controls is
    `"abcd"` (class 1); the link's text is `"tet�"` (S2);
  * `termsvc/termsvc.go`: `strip`'s comment says invalid UTF-8 shows as
    U+FFFD.

  No golden file changed. The tree's tests pass.
* **Fuzz,** each on a scratch copy of the tree:
  * `FuzzAgainstV0120`: 180 s, 11.3 million inputs, after the fix above,
    with no failure;
  * `FuzzClean`, since `clean.go` changed: 120 s, 20.5 million inputs,
    with no failure.
* **Benchmarks, after,** the same command and machine; the bar is the
  "before" median times 1.10, with no more allocations:

  | 4 KiB input | `Line` | `Styled` | `HasControl` |
  | :--- | ---: | ---: | ---: |
  | clean ASCII | 2.06 µs (1.06), 0 allocs | 2.10 µs (1.08), 0 allocs | 2.02 µs (1.04), 0 allocs |
  | clean CJK and emoji | 8.82 µs (1.07), 0 allocs | 8.82 µs (1.04), 0 allocs | 8.62 µs (1.05), 0 allocs |
  | tool output | 9.88 µs (0.26), 1 alloc | 10.90 µs (0.23), 1 alloc | 0.11 µs (0.00), 0 allocs |
  | styled | 6.62 µs (0.37), 1 alloc | 5.81 µs (0.43), 0 allocs | 0.10 µs (0.01), 0 allocs |
  | hostile | 14.86 µs (0.89), 1 alloc | 14.99 µs (0.46), 1 alloc | 0.18 µs (0.01), 0 allocs |

  All 15 pass. `Styled` on clean ASCII is the nearest, at 1.08.
* **Mutations,** through `python3 -m tools.verify mutate`, all killed:

  | Mutation | Killed by |
  | :--- | :--- |
  | `Styled` drops `KeepSGR` | `TestStyled`: `SGR kept: Styled("\x1b[38;2;1;2;3;1mred\x1b[m") = "red", want …` |
  | `Styled` drops `KeepSGR` | `FuzzAgainstV0120/seed#1`: `Styled(…) = "bold link"; v0.12.0 gives "\x1b[1mbold\x1b[m …"` |
  | `HasControl` is `IsClean` without the negation | `TestTokenAndHasControl`: `HasControl("plain text") = true, want false` |
  | `Line` uses `safetext.Text()` | `TestLine`: `line breaks and tabs, as one space: Line("a\r\n\t b\nc") = "a\n\t b\nc", want "a  b c"` |
  | `KeepSGR` keeps an SGR U+009B introduces | `TestKeepSGR`: `Clean("\u009b31mx") = "\x1b[31mx", want "x"` |
  | `KeepSGR` keeps an OSC 8 U+009C ends | `TestKeepSGR`: `Clean("\x1b]8;;https://e.org\u009cx") = "\x1b]8;;https://e.org\x1b\\x", want "x"` |
  | `KeepSGR` keeps a link U+009D introduces | `TestKeepSGR`: `Clean("\u009d8;;https://e.org\ax") = "\x1b]8;;https://e.org\ax", want "x"` |
  | `lineDeparts` without its C1 clause | `FuzzAgainstV0120/seed#26`: `Line("a\u0090q\u009cb") = "ab"; v0.12.0 gives "aqb"` |
  | `lineDeparts` without its string clause | `FuzzAgainstV0120/seed#27`: `Line("a\x1bXs\ab") = "ab"; v0.12.0 gives "a"` |
  | `lineDeparts` without its OSC clause | `FuzzAgainstV0120/seed#29`: `Line("a\x1b]0;Ü b") = "a"; v0.12.0 gives "a b"` |
  | `plainest` always true | `FuzzAgainstV0120/seed#30`: `Styled("a\x1b[0?mb") = "ab"; v0.12.0 gives "a?mb"` |
  | `HasControl` compared on `subparts(x)`, as the PLAN first wrote it | `FuzzAgainstV0120/seed#7`: `HasControl("a\xffb\xc3") = true; v0.12.0 gives false` |
  | `Styled` without `Flatten` (`Line(Styled(x)) == Line(x)`) | `FuzzAgainstV0120/seed#4`: `Line(Styled("a\r\n\t b\nc")) = "a bc"; Line gives "a  b c"` |
  | a `TestDepartures` row that no longer departs | `TestDepartures`: `class 2, BEL ends an SOS: v0.12.0's Line("a\x1bXs\ab") = "a", the row says "ab"` |
  | `introduced` skips no C0 or DEL after ESC | `FuzzAgainstV0120/seed#32`: `Line("0\x1b\x1e]ל0") = "0"; v0.12.0 gives "00"` |

  The step's three are the first four rows, the first run against both of
  its named tests; the rest are the deviation's. Each clause of the
  restated check is among them, so each new check was seen to fail.
* **Lint:** `modernize` asked, on the first gate run, for
  `slices.ContainsFunc` in `introduced`, which was applied.
* **Gates,** `python3 -m tools.verify gates` over the step's 14 files (the
  eleven Go files and the three records). The first run on macOS failed at
  `modernize` alone (above); the remote runs were stopped and all three
  rerun after the fix, nothing skipped: macOS
  `gates: 19 passed, 0 failed, 0 skipped in 388s`; the Linux host
  `… in 762s`; the Windows host `… in 2172s`. `release-check` is clean on
  each (258 files on the hosts); `make apicheck` "against v0.12.0, 0
  incompatible change(s)".
* **Scans:** `python3 -m tools.verify scan` "clean (5 pattern(s))"; no raw
  bidirectional, format or C1 character in any file of the step. The
  markdownlint gate's configuration leaves MADR and PLAN files out, so it
  checked `docs/README.md` only of the three records, as in earlier steps.
  Copies of the two records under other names, linted with the same
  rules, show only `*` list markers, which MADR requires, and hard tabs
  in A2.2's Go block, which predates this step.
* Committed by the owner.

### Step 6: release `v0.13.0` (2026-10-23)

* **Deviation (2026-10-23): the release carries more than the step's
  list.** Listing the exported API with apidiff, compatible changes
  included, from `v0.12.0` to `HEAD` (`220506d`) gives 17 additions. Nine
  are this PLAN's: `safetext`, and `glyph.Set`'s `Done`, `Failed`,
  `Running`, `Pending`, `Heading`, `MeterFill`, `MeterTrack` and
  `Control`. Eight are 0008-PLAN Steps 2 to 4's, on `main` since before
  this PLAN and in no tag:
  * `fuzzy`, which the step named;
  * `glyph.Set`'s `Prompt` and `Mask`;
  * `workspace`'s `CommandConsumer`, `CommandMsg`, `Close`,
    `(*Workspace).Bindings` and `(*Workspace).Theme`.

  The step's *Added* list named `fuzzy` alone of them. Release notes that
  leave out what a tag adds would be wrong, so they and `architecture.md`
  name all seventeen. 0008-PLAN Step 9's guide,
  `docs/guides/command-palette.md`, waits for `palette` there, as that
  PLAN's note says. The MADR's D12 says that release 1 "also carries
  `fuzzy`", which stays true, so the MADR is not amended.
* **Docs:**
  * `docs/architecture.md`:
    * `fuzzy` and `safetext` in the package tree and table;
    * `internal/sanitize`'s imports and row, as a wrapper of `safetext`;
    * `glyph`'s row with its ten new fields;
    * `workspace`'s row with its five additions;
    * a bullet naming the Charm-free cores;
    * `TestCoresAreCharmFree` under Tooling;
    * the `charmfree` rule under Dependencies;
    * the tree.
  * `docs/guides/migrating.md`, "Migrating to `v0.13.0`":
    * *Changed:* U+FFFD for invalid UTF-8; MADR A3.1's classes, put as
      what a program sees;
    * *Added:* the seventeen additions;
    * a line on the Charm-free cores.
  * `docs/glossary.md`: `Rules` and `Sanitizer` move from "Reserved" to
    "Built".
  * `README.md`: the current release is `v0.13.0`, and a Status bullet
    for `safetext`, `fuzzy` and the glyphs.
  * `docs/README.md`: this PLAN's row.
* **Release notes, `v0.13.0`,** for the owner's release:

  > **`v0.13.0`** adds packages and fields and removes nothing: `make
  > apicheck` reports 0 incompatible changes against `v0.12.0`.
  >
  > *Added*
  >
  > * `safetext`: `Clean`, `IsClean` and the streaming `Sanitizer` by
  >   `Rules`, with the presets `Text`, `Strict`, `Line`, `Command` and
  >   `StatusLine`; `SafeURL` and `URLs`. It imports only the standard
  >   library and `glyph` (#12).
  > * `fuzzy`: grapheme-cluster matching and ranking, and an `Index` for
  >   large lists (#6).
  > * `glyph.Set`: `Done`, `Failed`, `Running`, `Pending`, `Heading`,
  >   `MeterFill`, `MeterTrack` and `Control` (#11); `Prompt` and `Mask`.
  > * `workspace`: `Bindings`, `Theme`, `CommandConsumer`, `CommandMsg`
  >   and `Close`.
  >
  > *Changed*
  >
  > * Untrusted text the library shows has invalid UTF-8 as U+FFFD, one
  >   per maximal subpart, where it was dropped.
  > * Malformed escape sequences are read as the DEC parser reads them:
  >   * an encoded C1 control is its 7-bit form;
  >   * BEL ends a DCS, SOS, PM or APC;
  >   * a pane's view splits malformed input as a title does, and keeps
  >     no SGR or link written with a C1 control.
  >
  >   Well-formed text, styles and links are unchanged.
  > * `internal/sanitize` runs on `safetext`, and is faster: up to four
  >   times on tool output, and `HasControl` no longer cleans to decide.
  >
  > *Charm-free cores:* `fuzzy`, `glyph`, `layout`, `safetext` and `when`
  > link no Charm module but `x/ansi`, and a test holds them to it.
  >
  > [The migration guide](docs/guides/migrating.md#migrating-to-v0130)
  > has the details. Records:
  > `0020-MADR-charm-free-cores-glyphs-and-safetext.md` (D8, D9, D11,
  > A3), and `0008-PLAN-command-palette.md` Steps 2 to 4.

* **Issue comments,** drafted for the owner, to post once `v0.13.0` is
  tagged and the consumer smoke test passes:
  * **#6:** "`fuzzy` is in `v0.13.0`. Its closure links `x/ansi` and no
    other Charm module, which gobble's 0032-MADR A1 admits, and
    `internal/conformance`'s `TestCoresAreCharmFree` now holds it to
    that on Linux, macOS and Windows. The consumer smoke test ranked with
    it from the proxy's copy of the tag."
  * **#11:** "`v0.13.0`'s `glyph.Set` has `Done` (✓/+), `Failed` (✗/x),
    `Running` (●/\*) and `Pending` (○/-), each its own shape in both sets;
    `Heading` (▸/>), its own field though it draws as `Focus`; and
    `MeterFill` (━/#) and `MeterTrack` (╌/-). The ASCII ellipsis stays
    `~`. It is the tail a cut adds inside a cell budget, and `...` would
    take two cells more than `…`, so a view laid out for one set would no
    longer fit the other; a tilde is a known truncation mark too, as in
    `PROGRA~1` (0020-MADR D8, Q6)."
  * **#12:** "`safetext` is in `v0.13.0`, ahead of the rest of 0009, and
    `internal/sanitize` is now a wrapper over it: one sanitiser. It
    imports the standard library and `glyph` only. Invalid UTF-8 becomes
    U+FFFD, one per maximal subpart, so a raw 0x9B opens nothing; the C1
    code points are read as their 7-bit forms; `Escape` shows the
    bidirectional controls as `\u202e` and the rest; `IsClean` validates
    without cleaning. Against gobble's numbers, 4 KiB of tool output
    cleans in about 10 µs under `Line()`, and `IsClean` decides in 0.1
    µs. The differences from `v0.12.0` on malformed input are 0020-MADR
    A3."
* **Gates,** `python3 -m tools.verify gates` over the step's seven files,
  all documents: macOS `gates: 18 passed, 0 failed, 1 skipped in 402s`;
  the Linux host `… 1 skipped in 813s`; the Windows host `… 1 skipped in
  1968s`. `release-check` is "258 file(s) clean" on each, with `apicheck`
  among its checks; `make apicheck` alone reported "against v0.12.0, 0
  incompatible change(s)" at Step 5's commit, and this step changes no
  Go file.
  * *Skipped, 2026-10-23:* the `gofmt` gate, on each host, "no Go files":
    the step's files hold none. `release-check` ran gofmt over every
    tracked Go file, clean.
* **The disclosure guard** over the five commits ahead of `origin/main`
  (`44a4890` to `220506d`) passed, exit 0, with no finding. It runs again
  over this step's commit before the push.
* **Not done yet, and the owner's:**
  1. commit this step;
  2. the guard over the outgoing commits, which the agent runs again on
     request;
  3. the push of `main`, and CI green;
  4. the annotated tag `v0.13.0` on that commit, pushed, and CI green on
     the tag.

  Then the agent runs the consumer smoke test and its negative control,
  records them here, and the owner posts the issue comments above.
* **Deviation (2026-10-24): the tag is `v1.13.0`.** The owner committed
  the step as `8be11e7`. The guard passed over the six outgoing commits,
  `44a4890` to `8be11e7`, exit 0, with `main` not behind `origin/main`. CI
  and Docs passed on `main` at `8be11e7`. The owner then tagged that
  commit and pushed the tag as `v1.13.0`.
  * *Found* before the smoke test:
    * `git ls-remote` showed `refs/tags/v1.13.0` (an annotated tag, at
      `8be11e7`) and no `v0.13.0`;
    * `proxy.golang.org` listed `v1.13.0` and answered `@latest` with it.
  * *Options:*
    * (a) retract it with a `v1.13.1` and tag `v0.13.0`, recommended;
    * (b) keep `v1` and number from it;
    * (c) leave it.
  * *Chosen:* (b), by the owner. MADR A4 records it, and opens A4.1 on
    release 2's version and form, which Steps 7 to 13 wait for.
  * *Files added to the step:*
    * `README.md`, `docs/guides/migrating.md`, `docs/glossary.md` and
      `docs/README.md`, which now name `v1.13.0` where this step wrote
      `v0.13.0`;
    * `docs/decisions/0008-PLAN-command-palette.md`, a dated note;
    * the MADR, A4 and its notes on D10 and D12.
  * *The drafts above:* in the release notes and the three issue
    comments, read `v1.13.0` for `v0.13.0`, and the guide's anchor is
    `#migrating-to-v1130`. A reader of the tag's own `README.md` and
    `migrating.md` sees `v0.13.0`. That copy cannot change, and the GitHub
    release's notes can say so.
* **The consumer smoke test**, `releasing.md`'s procedure, run on
  2026-10-24 in a scratch module outside the repository by a script in the
  scratch directory.
  * *Steps:* `go mod init example.com/smoke`, then
    `go get github.com/maccavelli/go-tui-lib@v1.13.0`, which downloaded
    from `proxy.golang.org`, `go mod tidy`, an empty `go env GOWORK`,
    `go vet ./...`, `go build ./...` and `go run .`, each with exit 0. The
    module required is `github.com/maccavelli/go-tui-lib v1.13.0`.
  * *The program* cleans codex's injection string with invalid UTF-8 after
    it under `safetext.Text()`, checks `IsClean` on the result, draws
    `glyph.For(false).Done` and ranks `"rsm"` with `fuzzy`. It printed
    `clean "\n\tok�", done "+", best "resume"`.
  * *Its closure,* `go list -deps .` with `CGO_ENABLED=0`, judged by the
    closure gate's `charm` rule:
    * `GOOS=linux`: 133 packages, no Charm beyond `x/ansi`;
    * `darwin`: 131;
    * `windows`: 133;
    * on each, two `x/ansi` packages.

    `smoke: PASS`.
  * *The negative control:* the same procedure, with a program that
    imports `termcap` and `glyph`; each step had exit 0. Its closure holds
    `charm.land/bubbletea/v2`, `colorprofile`, `ultraviolet` and `x/term`
    on all three, with `x/termios` on linux and darwin and `x/windows` on
    each: 163, 161 and 168 packages. `control: PASS`.
* **CI on the tag:** in progress when this was written; the owner
  confirms it green.
* **Not done:** the issue comments are the owner's to post, as read above.
  The step's other items are done.
* **CI on the tag** `v1.13.0` passed (2026-10-24), as did CI and Docs on
  `main` at `8be11e7`. The owner committed the records of A4 and A4.1 as
  `ea559a2`.
* Committed by the owner.

### Step 6A: the module path moves to `/v2` (2026-10-24)

* **Approved** by the owner with MADR A4.2 ("committed, approved to
  proceed"), and both marked so.
* **The move,** by a script in the scratch directory, which checked each
  file's count before writing: 298 mentions in 153 files.
  * 149 Go files;
  * `go.mod`, 1;
  * `.golangci.yml`, 1, `charmfree`'s `internal/teamsg`;
  * `testdata/frameworks/go.mod.tmpl`, 2, with the root's pseudo-version
    now `v2.0.0-00010101000000-000000000000`, since a `/v2` path takes no
    `v0` version;
  * `scripts/go-examples_test.py`, 1.

  `gofmt -l` then listed nothing, and the import blocks kept their order.
  `GOWORK=off go list -m` prints `github.com/maccavelli/go-tui-lib/v2`.
* **`scripts/go-examples.py`:** `REPLACE` now matches the root's
  `replace` line at any major version's path, and `--update` writes back
  the path it found. Run with `--update` on a scratch copy, it wrote the
  template and `go.sum` unchanged, so the template's new form is what
  `go mod tidy` gives.
* **`scripts/go-apicheck.py`:**
  * `module_path` reads the base's `go.mod`;
  * `major_move` holds when two paths differ only by a `/vN` suffix;
  * `move_imports` then moves the base's own import paths in its `.go`
    files and `go.mod`, leaving any nested module alone, and the gate
    says so: `apicheck: .: v1.13.0 is github.com/maccavelli/go-tui-lib;
    its import paths moved to github.com/maccavelli/go-tui-lib/v2
    (294)`;
  * the docstring and `docs/architecture.md`'s Tooling describe it.
* **`scripts/go-apicheck_test.py`,** cases 12 to 14, written first and
  run against the unchanged script. That run gave
  `go-apicheck_test: 23 passed, 6 failed`: cases 12 and 13 exited 2 with
  `loading example.com/r/v2: found no packages for module example.com/r/v2`,
  and case 14 passed. After the change: `29 passed, 0 failed`.
* **Docs:**
  * `AGENTS.md`: the module path; the modules table; API rule 1, nothing
    removed within a major version; rule 2, an allow-list block only for
    a release that changes the major version, and the boundary move;
  * `docs/guides/releasing.md`: the root's path, its three `go get` lines,
    and the new section "Release a new major version";
  * `docs/architecture.md`: the path, and the gate's boundary move;
  * the two guides' `go doc` lines;
  * `README.md`'s Status: `main` is at `/v2` for `v2.0.0`, and rule 1.
* **The scan** of tracked files outside `docs/decisions/` and
  `docs/reports/` for the bare path finds 11 mentions, each meant:
  * four say what the path was up to `v1.13.0` (`AGENTS.md`,
    `docs/architecture.md`, `docs/guides/releasing.md`, and
    `docs/guides/migrating.md`'s `@latest` line, still true of `v1`);
  * six are a nested module's path (`stream/glamourmd`, `<dir>`);
  * one is the new section's `/vN` template.

  None is in code.
* **Checks:**
  * `go build`, `go vet` and `go test ./...` pass with `GOWORK=off`, and
    `go test ./...` in workspace mode, all exit 0;
  * `go mod tidy -diff` is clean;
  * `make lint`: 0 issues for linux, darwin and windows;
  * `make examples`: "46 case(s) run", "10 excerpt(s) checked", "clean";
  * `go-examples_test`: 19 passed, 0 failed;
  * `make apicheck`: "against v1.13.0, 0 incompatible change(s)",
    "clean".
* **Mutations,** through `python3 -m tools.verify mutate`:

  | Mutation | Killed by |
  | :--- | :--- |
  | the API gate without the boundary move | `go-apicheck_test`: `FAIL a move to /v2 with no API change passes: want 0, got 2` |
  | the boundary move applied whatever the paths | `go-apicheck_test`: `FAIL a path changed otherwise moves nothing` |
  | one `.go` file left at the old path (`safetext/rules.go`) | `go test ./safetext/`: `FAIL github.com/maccavelli/go-tui-lib/v2/safetext [setup failed]` |
  | the examples' `replace` line matched at the bare path only | `go-examples_test`: `FAIL --update keeps @ROOT@: want yes, got no` |

  The fourth is beyond the step's list, for the `go-examples.py` change.
  *Corrected, 2026-10-24:* the step expected `make apicheck` without the
  boundary move to report every cross-package signature. On a full
  scratch copy, tags included, it exits 2 instead: `exporting v1.13.0's
  API failed: … found no packages for module
  github.com/maccavelli/go-tui-lib/v2`, since the gate exports the base
  under the tree's path. The noise lines appear only when the base is
  exported under its own path, as in MADR A4.2's probe. Either way the
  gate fails without the move.
* **Gates,** `python3 -m tools.verify gates` over the step's 164 changed
  files, nothing skipped: macOS
  `gates: 19 passed, 0 failed, 0 skipped in 437s`; the Linux host
  `… in 878s`; the Windows host `… in 1779s`. `release-check` is "258
  file(s) clean" on each, with the API diff and examples gates.
* **Nothing is tagged:** `v2.0.0` is Step 13's. Steps 7 to 12 build at
  `/v2` and diff against `v1.13.0` through the boundary move.

### Steps 7 and 8: the probe engine and `termcaptea` (2026-10-24)

* The owner committed Step 6A as `f97a2e4` and said to proceed
  ("committed, proceed").
* **The facts, re-read before Step 7** (Prerequisites): `termcap` and
  `internal/termevent` changed after `63c68d6` only in import lines (Step
  5's `json_test.go` rows, and Step 6A's path), one for one, so every line
  the facts table cites still holds (`prober.go:51`, `:252-326`,
  `:336-356`, `:369`, `:442-450`, `:527-538`, `:556-585`, `:737-770`;
  `envcaps.go:16-22`). `go.mod` still requires Bubble Tea v2.0.10,
  `x/ansi` v0.11.8 and `colorprofile` v0.4.3. One row was wrong, and was
  already wrong at `63c68d6`; it is corrected in the table.
* **Deviation, 2026-10-24.**
  * *Found:* Step 7 removes `Prober`, `NewProber` and `KeyboardFlags` from
    `termcap`, gives `Option` the method `apply(*Probe)`, drops
    `Reply.Msg` and retypes `Caps.Profile`, but `termcaptea` only exists
    from Step 8. Between the two steps these would not compile:
    `termcap`'s `prober_test.go`, `discipline_test.go`,
    `example_test.go`, `identity_test.go`, `envcaps_test.go`,
    `report_golden_test.go` and `fuzz_test.go` (Step 8 moves them);
    `termcap/termcaptest/termcaptest_test.go:31` (`termcap.NewProber`),
    and its `:88`, which compares `Profile` with a
    `colorprofile.Profile`; and `launch/consumer_test.go:372`
    (`TestRunRestorerAfterCrash`). Rule 1 forbids a step that leaves
    `main` so. `launch/restore.go:7` and `docs/guides/commands.md:769`
    also name `*termcap.Prober`. These four files were in neither step's
    list.
  * *Resolutions offered:* (a) Step 7 rewrites `termcap.Prober` in place
    as a driver over the engine, and Step 8 moves it; (b) Steps 7 and 8
    as one step and one commit.
  * *Chosen:* (b), "Steps 7 and 8 as one". One commit, one set of gates,
    one `apicheck.allow` block named "0020-PLAN Steps 7 and 8"; both
    steps' tests and mutations as written; the four files added to Step
    8's.
  * The MADR is not amended: no decision changes, and the wrong fact is
    the PLAN's only.
* **Deviation, 2026-10-24, the differential.**
  * *Found:* `TestCoreAndAdapterAgree` as written compares every field but
    `Profile`, but the TUI run's `KeyboardFlags` is not deterministic.
    Bubble Tea v2.0.10's first render pushes Kitty flag 1
    (`cursed_renderer.go:425-430`, `:896`), and whether that reaches the
    fake terminal before the probe's `CSI ? u` decides the reply, 0 or 1.
    `termcap/report_golden_test.go:36-39` already pins the flags for that
    reason. Driven through `Terminal.Reply`, the core always sees 0.
  * *Resolutions offered:* (a) drive the core with the replies a fresh
    `Terminal` gives, through `Reply`, to the sequences the TUI run wrote,
    in its order; (b) pin `KeyboardFlags` with `WithOverride` in both paths.
  * *Chosen:* (a), "Replay the TUI's queries". `Terminal.Reply` keeps the
    Kitty flag stack as `answer` does; every field but `Profile` is
    compared, `KeyboardFlags` included. The core's own batch is covered by
    `TestProbeBatch` and `TestProbeTranscripts`.
* **`termcap`, the engine (Step 7).**
  * New: `reply.go` (`Reply`, `ReplyKind`, `ModeSetting`, `ParseReply`),
    `xcolor.go` (the `XParseColor` reader), `probe.go` (`Probe`,
    `ProbeEvent`, `ProbeEventKind`, `ProbeHook`, `CapsMsg`,
    `ColorSchemeMsg`, `DefaultTimeout`), `options.go` (every option, on
    `*Probe`, and `WithoutModeQuery`), `queries.go` (`Query`, the batch,
    its parsers over `Reply`, and `gatedAllowed`), `colorprofile.go`.
  * Changed: `caps.go` (`Profile ColorProfile`, written by token, read by
    token or colorprofile's spelling), `views.go` (`KeyboardCaps.Request`;
    `Enhancements` and `KeyboardFlags` gone), `report.go`
    (`ReportSchemaVersion = 2`), `envcaps.go` (a disabled `Probe`),
    `termcap.go` and `tmux.go` (documentation, and three shared token
    constants for `goconst`), `identity.go` (those constants).
  * `prober.go` is deleted; its Bubble Tea half is `termcaptea`'s.
  * `GOWORK=off go list -deps ./termcap`, for linux, darwin and windows,
    lists no Charm package but `x/ansi` and `x/ansi/parser`.
  * Choices made in execution, within A2:
    * `Probe` exports no timeout, and A2.2's API is kept, so
      `termcaptea.Prober.Init` learns the deadline by starting a scratch
      `Probe` of the same options at `Init`'s time and reading its
      `Timeout`; a disabled one has none, so `Init` is `nil`.
    * `ProbeDone`'s `Caps` is `Caps()` when `NextEvent` returns it, so a
      hook answered while the host drains is in it.
    * The probe's own DECRQM 2026 and 2027 read no silence into a missing
      reply, as tea's do not: `TestEachProfile` pins `SyncOutput`
      not-queried for `DA1Only`, and the report's text says "not every
      terminal answers".
    * `HandleReply` reads a report of mode 2026 or 2027, or of a mode
      `WithoutModeQuery` names, whoever asked; a report of another mode
      counts only for a query the batch sent, as today.
    * `isDark` is ultraviolet's `isDarkColor` (`decoder.go:1781`), the
      HSL lightness of the 8-bit channels under 0.5, written as
      `max + min < 255`; `TestDarkMatchesBubbleTea` holds it to tea's.
    * `x/ansi` v0.11.8's parser drops the last parameter when its buffer
      is exactly full (found by `TestParseReply`'s 32-parameter row), so
      the buffer holds 33 and a reply of more than 32 is refused before
      decoding, below the panic.
    * BEL ends a DCS or APC as it ends an OSC; `x/ansi` reads it as data,
      and `ParseReply` trims it.
    * An OSC 4 index is read with `strconv.Atoi`, leading zeros and all,
      as `parsePalette` read it; a stricter first draft failed the moved
      `TestPaletteAndForeground`.
  * Behaviour that differs from `v1.13.0`'s prober, each at an edge:
    * a deadline that fires before `tea.EnvMsg` no longer ends the probe
      with no environment; the tick is armed again for the deadline once
      the environment arrives, and ends it then;
    * a DSR 997 that arrives between DA1 and tea's colour profile now
      sends `ColorSchemeMsg`, before the held `CapsMsg`; it was dropped;
    * the follow-up OSC 11 after a DSR 997 is the probe's bytes through
      `tea.Raw`, not `tea.RequestBackgroundColor`: the same query;
    * a `tea.ModeReportMsg` of an ANSI mode, and any message that is not
      a reply, is not offered to a `Query`'s `Parse`, which reads a
      `Reply` with no `Msg`;
    * a colour of OSC 4 is a `color.RGBA64` at the reply's precision
      (A2.4); OSC 10 and 11 keep tea's colours in a TUI.
* **`termcaptea` (Step 8):** `doc.go`, `prober.go` (`Prober`, today's
  surface, over a `Probe` built with `WithoutModeQuery(2026, 2027)`),
  `convert.go`, `keyboard.go` (`KeyboardEnhancements`), `profile.go`
  (`FromColorProfile`, `ToColorProfile`). Each drain is one `tea.Sequence`
  of the probe's writes and messages in order, beside its hooks and any
  tick.
* **Outside `termcap`:** `termcaptest.Terminal.Reply`, under the
  terminal's lock, and its documentation; `termcaptest_test.go` on
  `termcaptea`, comparing `Profile` through `FromColorProfile`, with the
  new `TestTerminalReply`; `launch/consumer_test.go:372` and
  `launch/restore.go`; `termsvc/notify.go`'s documentation;
  `internal/conformance`'s `cores`, `enums` and the `termcap.Option` case
  on `*termcap.Probe`; `.golangci.yml`'s `charmfree` with
  `**/termcap/*.go`; `scripts/apicheck.allow`'s block.
* **Tests moved, by Rule 7.** Every name below runs in `termcaptea`, with
  `termcap.` qualifiers; the changes beyond those are listed.
  * `prober_test.go`, but `TestGatedAllowed`, which stays with the
    engine in `termcap/probe_test.go`. A local `fixed` helper;
    `backgroundRequests` also counts a write of the OSC 11 query alone,
    which is how the prober now writes the follow-up (the batch, which
    holds it among others, is not one).
  * `discipline_test.go`, but `TestAlacrittyVersionFromDA2`, which stays
    and reads `Keyboard().Request`. `queryFacts`, unexported, is the list
    of its seven facts; "nothing awaited" answers each query the batch
    sent, where it set an unexported flag; `Profile` is compared as a
    `ColorProfile`.
  * From `identity_test.go`: `TestWithGOOS`, `TestAppearanceChain`,
    `TestHooksAreNotCalledWhenNotSet` (on `hookMsg`), `TestConsoleHost`,
    `TestPaletteAndForeground` (the palette wanted as `RGBA64`, A2.4; its
    refusals stay in `termcap` as `TestPaletteRefusals`, through
    `ParseReply`) and `TestBrandFromXTVersion`. `TestKeyboardFlags` stays
    as `TestKeyboardRequest`; `TestParseTmux` sets the facts on a
    `Probe`; `TestParseXColor` gives way to `TestXParseColor`.
  * `FuzzProberReplies`, from `fuzz_test.go`.
  * `TestXTVersionSanitized`'s tea message; `termcap` keeps the
    environment's cases and reads a `ReplyVersion`.
  * The `Prober` half of the `EnvCaps` match, as
    `TestEnvCapsMatchesDisabledProber`; `termcap` keeps the facts, as
    `TestEnvCapsMatchesDisabledProbe`.
  * `ExampleProber`, with `KeyboardEnhancements`; `termcap` gains
    `ExampleProbe`, a CLI's drive loop with written-out replies.
  * `report_golden_test.go` and its 16 text goldens, moved from
    `termcap/testdata/golden` to `termcap/termcaptea/testdata/golden`.
* **Core test files that link Bubble Tea after the step (Rule 7):** no
  `termcap` test imports it. `termcap/envcaps_test.go` and
  `termcap/transcripts_test.go`, both `termcap_test`, import
  `termcaptest`, which links it, for its profiles and `Terminal.Reply`;
  `TestProbeTranscripts` drives the core with no program.
* **New tests:** `TestParseReply`, `TestReplyKindsAndSettings`,
  `TestXParseColor`, `TestProbeBatch`, `TestProbeReadsHostModes`,
  `TestProbeTimeout`, `TestProbeSentinel`, `TestProbeDuplicateDA1`,
  `TestProbeHooks`, `TestProbeNoEvent`, `TestProbeColorScheme`,
  `TestProbeBackground`, `TestProbeReplyTooLong`, `TestProbeJetBrains`,
  `TestProbeAwaiting`, `TestQuerySynchronizedOutput`,
  `TestProbeTranscripts`, `TestPaletteRefusals`, `TestColorProfileTokens`,
  `TestCapsJSONReadsOldProfile`, `TestReportSchemaVersion`, `ExampleProbe`,
  `FuzzParseReply`, `FuzzProbe`; in `termcaptea`,
  `TestCoreAndAdapterAgree`, `TestDarkMatchesBubbleTea`,
  `TestKeyboardEnhancements`, `TestColorProfileRoundTrip`; and
  `termcaptest`'s `TestTerminalReply`. `TestProbeTranscripts` uses
  `Terminal.Reply` at once: the steps run as one, so the copied
  transcript of Step 7 had no step to live in.
* **Golden files,** each read:
  * the 16 text reports, moved, differ from `HEAD`'s only in the
    profile's line, 32 lines in all, as `TrueColor` → `truecolor`,
    `ANSI256` → `ansi256` and `ANSI` → `ansi`;
  * `report-json-full.golden` and `report-json-zero.golden`:
    `"schema_version": 1` → `2`, and in the full one `"profile":
    "ANSI256"` → `"ansi256"`. Nothing else moved.
  * `report_test.go`'s and `termcap_test.go`'s wanted profile, and
    `TestCapsJSONRoundTrip`'s, change the same way.
* **Checks:**
  * `GOWORK=off go build ./...`, `go vet ./...`, `go test ./...`, and
    `go test ./...` in workspace mode: exit 0;
  * `TestCoreAndAdapterAgree` and `TestReportGolden`, `-count=30`: ok;
  * `golangci-lint` over the changed packages, for linux, darwin and
    windows: `0 issues`; the first run found `goconst` on the shared
    tokens, a `gosec` G115 on the checked DECRPM conversion (now a
    `nolint` with its reason) and `staticcheck` ST1018 on raw format
    characters the writing tool had decoded from `\u` escapes in the two
    `TestXTVersionSanitized` files, written back as escapes and compared
    with `HEAD`'s strings;
  * `make examples`: "46 case(s) run", "10 excerpt(s) checked", "clean";
  * `make apicheck`: "against v1.13.0, 8 incompatible change(s)",
    "clean", with the block;
  * `python3 -m tools.verify names` over the eight new type names:
    "8 planned name(s), clean";
  * `markdownlint-cli2` over the changed guides and indexes: "0 issues".
* **Mutations,** through `python3 -m tools.verify mutate`, each on a
  fresh copy: 41, all killed by the test named (`mutate: 41 killed, 0
  survived, 0 errors`, on the tree as handed off). Rows 7.x and 8.x are
  the PLAN's lists; 7.5b to 7.10, 8.6, 8.7 and c.1 to c.21 are added, so
  that every new or changed test is seen failing (Rule 5).

  | Mutation | Killed by |
  | :--- | :--- |
  | 7.1 the sentinel leaves unanswered queries unknown (TestProbeTranscripts, DA1 only) | `transcripts_test.go:114: xterm: DesktopNotify = {Value:unknown Origin:not-queried Reason:}, want {Value:unsupported Origin:query Reason:}` |
  | 7.2 ParseReply refuses a BEL-terminated OSC (TestParseReply) | `reply_test.go:99: OSC 11, BEL: ParseReply("\x1b]11;rgb:1e1e/1e1e/2e2e\a") = {Kind:unknown Raw:\x1b]11;rgb:1e1e/1e1e/2e2e\x07 Params:[] Mode:0 Setting:not…` |
  | 7.3 #3a7 is scaled, not taken as the high bits (TestXParseColor) | `xcolor_test.go:26: xParseColor("#3a7") = {13107 43690 30583 65535}, true; want #3000a0007000` |
  | 7.4 HandleTimeout ignores the deadline (TestProbeTimeout), replacing a mutation that did not compile | `probe_test.go:156: before the deadline: [{Kind:done Data: Hook:none Run:<nil> Caps:{Complete:false TimedOut:true Terminal:{Value:xterm-kitty Origin…` |
  | 7.5 ProbeDone is queued twice on a second DA1 (TestProbeDuplicateDA1) | `probe_test.go:221: first DA1 [{Kind:done Data: Hook:none Run:<nil> Caps:{Complete:true TimedOut:false Terminal:{Value:xterm-kitty Origin:env Reason…` |
  | 7.5b the same, seen by FuzzProbe's seeds | `fuzz_test.go:135: ProbeDone 2 times` |
  | 7.6 WithoutModeQuery is ignored (TestProbeBatch) | `probe_test.go:119: local, without 2026 and 2027: first event [{Kind:write Data:\x1b[?u\x1b[?2031$p\x1b[?2048$p\x1b[?1004$p\x1b[?996n\x1b]11;?\x07\x1b[>c\x1b[>q\x1b]99;i=termcap:…` |
  | 7.7 the console hook is queued on Linux (TestProbeHooks) | `probe_test.go:237: Linux: [{Kind:write Data:\x1b[?u\x1b[?2031$p\x1b[?2048$p\x1b[?1004$p\x1b[?996n\x1b]11;?\x07\x1b[>c\x1b[>q\x1b]99;i=termcap:p=?;\x07\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;A…` |
  | 7.8 ParseReply reads Pm 5 as a setting (TestParseReply) | `reply_test.go:99: DECRPM Pm 5: ParseReply("\x1b[?1004;5$y") = {Kind:mode Raw:\x1b[?1004;5$y Params:[] Mode:1004 Setting:ModeSetting(5) OSC:0 Index:0 C…` |
  | 7.8b the same, seen by FuzzParseReply's seeds | `fuzz_test.go:46: ParseReply("\x1b[?2031;5$y") is DECRPM with Pm 5` |
  | 7.9 termcap/probe.go imports Bubble Tea (TestCoresAreCharmFree) | `charmfree_test.go:170: termcap (GOOS=linux) links charm.land/bubbletea/v2: termcap → charm.land/bubbletea/v2` |
  | 7.9b the same, refused by depguard's charmfree | `termcap/probe.go:9:2: import 'charm.land/bubbletea/v2' is not allowed from list 'charmfree' … (depguard)` |
  | 7.10 ParseReply's colour is nil (FuzzParseReply's seeds) | `fuzz_test.go:43: ParseReply("\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\") is a colour with none` |
  | 8.1 convert.go drops tea.ModeReportMsg (TestCoreAndAdapterAgree) | `agree_test.go:87: kitty: the core and the Prober disagree:` |
  | 8.2 the adapter does not pass WithoutModeQuery(2026, 2027) (TestBatchNeverAsksWhatTeaAsks) | `prober_test.go:171: [TERM=xterm-kitty]: the batch asks "\x1b[?2026$p", which tea asks and acts on` |
  | 8.3 CapsMsg is sent before ColorProfileMsg (TestCapsMsgWaitsForTheColourProfile) | `discipline_test.go:232: 1 CapsMsg before the colour profile` |
  | 8.4 IsReplyFragment ignores Awaiting (TestIsReplyFragment) | `discipline_test.go:192: a fragment before the batch was sent` |
  | 8.5 KeyboardEnhancements ignores Request where Kitty is supported (TestKeyboardEnhancements), replacing a mutation that did not compile | `keyboard_test.go:45: terminal.leaks-releases: {ReportEventTypes:true ReportAlternateKeys:false ReportAllKeysAsEscapeCodes:false ReportAssociatedTex…` |
  | 8.6 Terminal.Reply forgets the Kitty flag stack (TestTerminalReply), replacing one the differential could not see | `termcaptest_test.go:271: step 3: Reply("\x1b[?u") = "\x1b[?0u", want "\x1b[?1u"` |
  | 8.7 the core's dark rule is max+min under 256, not 255 (TestDarkMatchesBubbleTea) | `agree_test.go:104: {255 0 0 255}: dark true, tea says false` |
  | c.1 two ColorProfile tokens swapped (TestColorProfileTokens) | `colorprofile_test.go:16: tokens unknown,notty,ascii,ansi256,ansi,truecolor` |
  | c.2 the same, seen by termcaptea's TestColorProfileRoundTrip | `profile_test.go:22: ANSI: token "ansi256", want "ansi"` |
  | c.3 the old spellings are not read (TestCapsJSONReadsOldProfile) | `colorprofile_test.go:30: "Unknown": unknown, json: cannot unmarshal JSON object into Go termcap.Caps: termcap: unknown colour profile "Unknown"; wa…` |
  | c.4 ReportSchemaVersion stays 1 (TestReportSchemaVersion) | `colorprofile_test.go:46: ReportSchemaVersion = 1, want 2` |
  | c.5 a fresh probe has an event (TestProbeNoEvent) | `probe_test.go:267: a fresh probe: {Kind:write Data: Hook:none Run:<nil> Caps:{Complete:false TimedOut:false Terminal:{Value: Origin:not-queried Rea…` |
  | c.6 no background query follows a DSR 997 (TestProbeColorScheme) | `probe_test.go:288: options 0: [{Kind:color-scheme Data: Hook:none Run:<nil> Caps:{Complete:false TimedOut:false Terminal:{Value: Origin:not-queried…` |
  | c.7 OSC 11 reads dark backwards (TestProbeBackground) | `probe_test.go:318: "\x1b]11;rgb:ff/00/00\a": Background {65535 0 0 65535}, Dark {Value:true Origin:query Reason:}; want dark false` |
  | c.8 a long 8-bit OSC is not matched to its query (TestProbeReplyTooLong) | `probe_test.go:334: an 8-bit OSC over 1 KiB: {Value:unknown Origin:not-queried Reason:}` |
  | c.9 JetBrains is probed (TestProbeJetBrains) | `probe_test.go:350: JetBrains: [{Kind:write Data:\x1b[?u\x1b[?2031$p\x1b[?2048$p\x1b[?1004$p\x1b[?996n\x1b]11;?\x07\x1b[>c\x1b[>q\x1b]99;i=termcap:p=?;\x07\x1b_Gi=31,s=1,v=1,a=q,t=d,f=…` |
  | c.10 Awaiting stays true once the batch went out (TestProbeAwaiting) | `probe_test.go:374: awaiting with every query answered` |
  | c.11 Bubble Tea's rule loses its Windows Terminal clause (TestQuerySynchronizedOutput) | `probe_test.go:418: [TERM=xterm-256color SSH_TTY=x WT_SESSION=x]: false, want true` |
  | c.12 a host's own modes are not read (TestProbeReadsHostModes) | `probe_test.go:136: SyncOutput {Value:supported Origin:query Reason:}, GraphemeWidth {Value:unsupported Origin:query Reason:}, ColorSchemeReports {V…` |
  | c.13 a reply after the sentinel is not read (TestProbeSentinel) | `probe_test.go:203: DesktopNotify = {Value:unsupported Origin:query Reason:} after a late reply, want supported` |
  | c.14 the palette takes an index past 15 (TestPaletteRefusals) | `--- FAIL: TestPaletteRefusals: panic: runtime error: index out of range [16] with length 16` |
  | c.15 Request asks for disambiguation, not event types (TestKeyboardRequest) | `identity_test.go:166: kitty: Request 1, reason ""; want 2, ""` |
  | c.16 Alacritty 0.14 counts as newer (TestAlacrittyVersionFromDA2) | `discipline_test.go:24: DA2 [0 1400 1]: event types true, want false` |
  | c.17 the batch leaves DA2 out (ExampleProbe) | `--- FAIL: ExampleProbe: its output differs from the Output comment` |
  | c.18 XTVERSION's name is not cleaned (termcap's TestXTVersionSanitized) | `sanitize_test.go:17: Terminal = {Value:evil\x1b[2J\u202eterm …}: the reply kept its controls` |
  | c.19 the same, seen by termcaptea's TestXTVersionSanitized | `sanitize_test.go:19: Terminal = {Value:evil\x1b[2J\u202eterm …}: the message kept its controls` |
  | c.20 EnvCaps drops the caller's options (TestEnvCapsMatchesDisabledProbe) | `envcaps_test.go:35: kitty, linux, appearance and override:` |
  | c.21 the same, seen by termcaptea's TestEnvCapsMatchesDisabledProber | `envcaps_test.go:38: kitty, linux, appearance and override:` |

  The first run had 20 (7.x and 8.x): 17 killed by their test, two killed
  only by a build failure, so replaced, and one survivor, so replaced:
  * 7.4 as first written left `deadline` unused, and did not compile;
  * 8.5 as first written left `ansi` unused, and did not compile;
  * 8.6 as first written, `Terminal.Reply` forgetting the Kitty flag
    stack, survived `TestCoreAndAdapterAgree -count=5`: in those runs tea's
    push reached the terminal after the probe's query, so the replay saw
    no difference. `TestTerminalReply` now pins the stack, and kills it.
* **`go fix`.** The first macOS gates run failed `modernize`: Go 1.27's
  `go fix` writes the batch's literals with promoted fields, `{Name: …}`
  for `{Query: Query{Name: …}}`, the form `prober.go` had. `go fix
  ./termcap/` applied it to `queries.go` alone; `go fix -diff ./...` is
  then empty for linux, darwin and windows, and the gates and the
  mutations above ran on that tree. The Linux and Windows runs begun
  before the fix were stopped and run again.
* **Gates,** `python3 -m tools.verify gates` over the step's 76 present
  files, nothing skipped: macOS `gates: 19 passed, 0 failed, 0 skipped in
  407s`; the Linux host `gates: 19 passed, 0 failed, 0 skipped in 925s`;
  the Windows host `gates: 19 passed, 0 failed, 0 skipped in 2074s`. `release-check` is "255 file(s) clean"
  on macOS and "282 file(s) clean" on Linux, with the API diff and
  examples gates.
* **Identifiers:** `python3 -m tools.verify scan`: `scan: clean (5
  pattern(s))`.
* **Not done here:** the release and its migration notes, which are Step
  13's. Nothing is tagged.

### Step 9: `termsvc` and `termsvctea` (2026-10-24)

* The owner committed Steps 7 and 8 as `a559d7c` and said to proceed
  ("committed, proceed").
* **The facts it rests on:** `NotifyContext` at `notify.go:280-300`,
  `CopyContext` at `clipboard.go:130-155` and `ActivityBeacon` at
  `beacon.go:135` stood as the table cites them; `Update` was at
  `notify.go:256-266`, one line lower after Step 8's documentation edit.
  Outside `termsvc` only the guides, `docs/architecture.md` and
  `internal/conformance/options_test.go` (`termsvc.NotifyOption`, which
  stays) name what moves.
* **`termsvc`, the core:**
  * `notify.go`: `Update`, `Notify` and `NotifyContext` gone;
    `SetCaps`, `SetFocus` and `Prepare(ctx, x) (seq, deliver)` added.
    `Prepare` is `NotifyContext` up to its command: cleaning, the skip,
    `bytesFor` and so the numbering on the caller's goroutine, and a
    backend called only by `deliver`, under `bounded(ctx, n.timeout)`.
    `Sequence` is unchanged. The `Notifier`'s and `WhenUnfocused`'s
    documentation say how a CLI and a TUI drive it.
  * `clipboard.go`: `Copy` and `CopyContext` gone; `PrepareCopy(ctx, c,
    text, o...) (Route, func() CopiedMsg)` added, with `RouteNone` and
    `Failed` over `MaxCopyBytes`, `RouteBackend` and a bounded clipboard
    call in `deliver`, else `RouteOSC52Tmux` inside tmux or `RouteOSC52`.
  * `links.go`: `LinkPolicy.Open` gone; `beacon.go`: `ActivityBeacon`
    gone; `termsvc.go`'s package comment.
  * `GOWORK=off go list -deps ./termsvc`, for linux, darwin and windows:
    no Charm package but `x/ansi` and `x/ansi/parser`, and no
    `internal/teamsg`.
  * `.golangci.yml`'s `charmfree` gains `**/termsvc/*.go`;
    `internal/conformance`'s `cores` gains `termsvc`.
* **`termsvc/termsvctea/termsvctea.go`:** `Observe`, `Notify`,
  `NotifyContext`, `Copy`, `CopyContext`, `Open` and `ActivityBeacon`, as
  the step writes them. A skipped notification's command is the
  `deliver` call, where today's was `teamsg.Cmd` of the same result.
* **Tests, by Rule 7:**
  * moved to `termsvctea`, changed only for receivers and `termsvc.`
    qualifiers: from `termsvc_test.go` the helpers and
    `TestNotificationBytes`, `TestFocusPolicies` (through `Observe`),
    `TestEmptyNotificationIsNotSent`, `TestBackend` and `TestCopy`; from
    `services_test.go` `TestCopyStatus` and `TestActivityBeacon`; all of
    `forms_test.go` (`TestSequenceMatchesNotify`, `TestCopySequence`,
    `TestBackendDeadline`, `TestBackendTimeout`) and of `adapter_test.go`;
    and `TestOpenable`'s `Open` half, as `TestOpen`;
  * kept in `termsvc`, reading what the core decides through `Prepare`,
    `deliver` and `PrepareCopy` where they read a command:
    `TestNotificationBytes`, `TestNotificationsAreWrappedInsideAMultiplexer`,
    `TestFocusPolicies` (through `SetFocus`), `TestEmptyNotificationIsNotSent`,
    `TestBackend` (and a skipped notification never reaching it),
    `TestControlBytesAreStripped`, `TestNotificationText`, `TestGate`,
    `TestBackendDeadline`, `TestBackendTimeout` and
    `TestBackendAndClipboardFuncs`; the rest of the package's tests
    unchanged. So the byte tables are pinned on both sides: the core's,
    and what the adapter writes.
  * new: `TestPrepareMatchesSequence` (every protocol, `Off` included,
    and every policy; one count across both, once per call) and
    `TestPrepareCopyRoutes` in `termsvc`; `TestBackendOffTheLoop`,
    `TestObserve`, `TestOpen` and `TestNotifyWritesThenReports` in
    `termsvctea`.
  * The four test functions holding `\u` escapes (`TestControlBytesAreStripped`,
    `TestSanitizeTitle`, `FuzzSanitizers`, `TestActivityVendorSanitized`)
    were copied byte for byte by a scratch script, since the writing tool
    decodes `\u` escapes (Steps 7 and 8); a scan finds no raw format or
    control character in either package.
  * No `termsvc` test imports Bubble Tea.
* **Callers:** `docs/guides/terminal-capabilities.md` (the services'
  bullets, the notifier example, the CLI's `Prepare` and `SetCaps`, the
  activity row) and `docs/architecture.md` (the dependency rows, the
  `termsvc` and `termsvctea` rows, the cores, the tree).
* **Checks:**
  * `GOWORK=off go build ./...` and `go vet ./...`: exit 0; `go test
    ./termsvc/...` three times, and with `-race`: ok;
    `go test ./internal/conformance/`: ok;
  * `golangci-lint` over `termsvc/...` and `internal/conformance/...` for
    linux, darwin and windows: `0 issues`; `go fix -diff ./...`: empty
    for each;
  * `make apicheck`: "against v1.13.0, 15 incompatible change(s)",
    "clean", with a block named "0020-PLAN Step 9" of the seven removals,
    each one the step's list names;
  * `markdownlint-cli2` over the changed guide, `architecture.md` and the
    index: "0 issues".
* **Mutations,** each on a fresh copy, 21, every one killed by the test
  named (`mutate: 21 killed, 0 survived, 0 errors`, then `mutate: 2
  killed, 0 survived, 0 errors` for 9.5 and 9.5b replaced). Rows 9.1 to
  9.5 are the step's list; the rest see each new or changed test fail
  (Rule 5):

  | Mutation | Killed by |
  | :--- | :--- |
  | 9.1 Prepare numbers twice per call (TestPrepareMatchesSequence) | `forms_test.go:42: protocol auto, policy always, {Title:build Body:passed ID: Urgency:normal}:` |
  | 9.2 the adapter calls deliver before returning (TestBackendOffTheLoop) | `termsvctea_test.go:37: NotifyContext waited for the backend` |
  | 9.3 PrepareCopy sends an over-long text (TestPrepareCopyRoutes) | `services_test.go:49: one byte over: osc52, {Status:unconfirmed Route:osc52 Err:<nil>}; want none, {Status:failed Route:none Err:<nil>}` |
  | 9.4 Observe ignores tea.BlurMsg (TestObserve) | `termsvctea_test.go:92: after blur: focused, want` |
  | 9.5 termsvc/notify.go imports and uses Bubble Tea (TestCoresAreCharmFree), replacing one that did not compile | `charmfree_test.go:171: termsvc (GOOS=linux) links charm.land/bubbletea/v2: termsvc → charm.land/bubbletea/v2` |
  | 9.5b the same, refused by depguard's charmfree | `termsvc/notify.go:9:2: import 'charm.land/bubbletea/v2' is not allowed from list 'charmfree' … (depguard)` |
  | 9.6 the result is delivered before the write (TestNotifyWritesThenReports) | `termsvctea_test.go:65: first {Notification:{Title:t Body: ID: Urgency:normal} Sent:true Skipped: Err:<nil>}, want the bell` |
  | 9.7 Open ignores the policy (TestOpen) | `termsvctea_test.go:117: Open returned a command for a refused URL` |
  | 9.8 SetFocus records the opposite (termsvc's TestFocusPolicies) | `termsvc_test.go:144: policy 0, focused: Sent true, Skipped ""; want skipped "focused"` |
  | 9.9 the same, seen through Observe by termsvctea's TestFocusPolicies | `termsvc_test.go:130: policy 3, focused: Sent true, Skipped ""; want skipped "focused"` |
  | 9.10 a skipped notification still calls the backend (termsvc's TestBackend) | `termsvc_test.go:192: a skipped notification reached the backend: "", {Notification:{Title:x Body: ID: Urgency:normal} Sent:true Skipped: Err:<nil>}…` |
  | 9.11 Prepare does not clean the text (TestNotificationText) | `services_test.go:178: title "a\n\n  b\r\nc\x1b[1m d", want line breaks collapsed and the sequence gone` |
  | 9.12 the bell is wrapped like the rest (termsvc's TestNotificationsAreWrappedInsideAMultiplexer) | `termsvc_test.go:115: tmux bell: "\x1bPtmux;\a\x1b\\", want a bare BEL, which tmux passes on` |
  | 9.13 OSC 777 keeps its semicolons (termsvc's TestNotificationBytes) | `termsvc_test.go:71: OSC 777:` |
  | 9.14 the same, seen by termsvctea's TestNotificationBytes | `termsvc_test.go:102: OSC 777:` |
  | 9.15 a copy's clipboard is not bounded (termsvc's TestBackendDeadline) | `forms_test.go:136: the default, clipboard: deadline -2562047h47m16.854775808s (false), want 5s` |
  | 9.16 a blocked clipboard is not ended by its timeout (termsvc's TestBackendTimeout) | `--- FAIL: TestBackendTimeout: panic: deadlock: all goroutines in bubble are blocked` |
  | 9.17 a clipboard's error is not reported (TestBackendAndClipboardFuncs) | `adapter_test.go:41: PrepareCopy reported backend, {Status:confirmed Route:backend Err:<nil>}` |
  | 9.18 a gated notification is prepared (termsvc's TestGate) | `services_test.go:196: gated: {Notification:{Title:working Body: ID: Urgency:normal} Sent:true Skipped: Err:<nil>}` |
  | 9.19 an empty notification is prepared (termsvc's TestEmptyNotificationIsNotSent) | `termsvc_test.go:156: a notification of controls only: "\a", {Notification:{Title: Body: ID: Urgency:normal} Sent:true Skipped: Err:<nil>}` |
  | 9.20 tmux copies are not wrapped again (termsvctea's TestCopy) | `termsvc_test.go:203: inside tmux, the extra copy is "\x1b]52;c;aGVsbG8=\a", want "\x1bPtmux;\x1b\x1b]52;c;aGVsbG8=\a\x1b\\"` |

  9.5 and 9.5b as first written added the import without using it, so
  the package did not compile, and each kill was a build or type-check
  failure rather than the check named; both were replaced by a form that
  uses the import, whose kills are the rows above.
* **Gates,** `python3 -m tools.verify gates` over the step's 21 files,
  nothing skipped: macOS `gates: 19 passed, 0 failed, 0 skipped in 459s`; the Linux host
  `gates: 19 passed, 0 failed, 0 skipped in 883s`; the Windows host
  `gates: 19 passed, 0 failed, 0 skipped in 1826s`. `release-check` is
  "282 file(s) clean" on macOS and "288 file(s) clean" on Linux and
  Windows, with the API diff and examples gates.
* **Identifiers:** `python3 -m tools.verify scan`: `scan: clean (5 pattern(s))`.

### Step 10: `command` and `commandtea` (2026-10-24)

* The owner committed Step 9 as `64d3f4d` and said to proceed
  ("committed, proceed").
* **Deviation, 2026-10-24, the callers.** The facts table's count was
  wrong, and so already at `63c68d6`: `workspace/commands.go` sets
  `Result{Cmd: …}` at twelve sites, not nine; `workspace/bindings.go:271`
  returns a handler's `res.Cmd` as a `tea.Cmd`; `docs/guides/keymaps.md:52`
  calls `Dispatch`. Each is a caller of what the step changes, with one
  correct resolution, converting it here, or the tree does not build: so
  it was recorded and not stopped for, as Step 6's uncounted additions
  were. The fact row and the step's callers carry the note.
* **Deviation, 2026-10-24, the context's place.** The step wrote
  `commandtea.Dispatch(r, ctx, req)` and `WatchContext(r, ctx)`; revive's
  `context-as-argument` refuses a context after another parameter. Offered:
  the context first, or the step's order with a `nolint`. *Chosen:* the
  context first, `Dispatch(ctx, r, req)` and `WatchContext(ctx, r)`, as
  Go's convention and `termsvctea.NotifyContext(ctx, n, x)` have it. The
  step carries the note.
* **`command`, the core:**
  * `handler.go`: `type Effect func() any`, documented; `Result.Cmd` is an
    `Effect`.
  * `dispatch.go`: `Dispatch` returns an `Effect`: a `ResultMsg` alone for
    a request that cannot run; for an `Async` command, one that executes,
    audits and returns `effects(…)`; else `effects(…)` at once, as a
    batch. `effects` is an `[]Effect` of `Result.Cmd`, the `ResultMsg` and,
    for a `Prompt` or `Forward`, the `PromptMsg`, a nil `Result.Cmd`
    staying a nil entry the host skips. `Attach(send func(any))`; `onLoop`'s
    `LoopMsg` carries a `func() Effect`.
  * `registry.go`: `Watch(ctx) Effect` is `WatchContext`'s body, giving
    `ChangedMsg`, or `[]Effect{changed, conflict}`, or nil once ctx ends;
    `WithLoop(send func(any))`, and `loopState.send` a `func(any)`.
  * `messages.go`: `LoopMsg{run func() Effect}` and `Run() Effect`;
    `builtin.go`: `app.quit`'s `Result{Cmd: func() any { return
    QuitRequestMsg{} }}`; `command.go`'s package comment and `Async`'s.
  * `GOWORK=off go list -deps ./command`, for linux, darwin and windows:
    no Charm package but `x/ansi` and `x/ansi/parser`, and no
    `internal/teamsg`.
  * `.golangci.yml`'s `charmfree` gains `**/command/*.go`;
    `internal/conformance`'s `cores` gains `command`.
* **`command/commandtea/commandtea.go`:** `Effect`, `Cmd` (an
  `[]command.Effect` run at once as `tea.Batch` of each converted, which
  gives `nil` when every entry is nil), `Dispatch`, `Watch`,
  `WatchContext`, `RunLoop`, `Attach` and `WithLoop`. `tea.Msg` is
  ultraviolet's `Event`, a defined `interface{}` and not an alias of
  `any`, so `Attach` and `WithLoop` wrap `send`.
* **Callers:** `workspace/commands.go`, the twelve sites as
  `commandtea.Effect(x)`; `workspace/bindings.go:271`,
  `commandtea.Cmd(res.Cmd)`; `launch/run.go:98`, `commandtea.Attach`;
  `workspace/commands_test.go`, `launch/run_test.go` and
  `launch/consumer_test.go` as the step says.
* **Tests, by Rule 7.** `Dispatch`, `Watch`, `Attach`, `WithLoop` and
  `LoopMsg.Run` stay in `command` with new types, and so do their tests:
  their only use of Bubble Tea was reading messages, so a scratch script
  read them through the core (the `collect` helper over `Effect`,
  `tea.Msg` as `any`, `tea.Cmd` as `Effect`, `teamsg.Cmd(x)` as an
  `Effect`, `Watch()` and `WatchContext(ctx)` as `Watch(ctx)`), in
  `builtin_test.go`, `dispatch_test.go`, `export_test.go`, `forms_test.go`,
  `loop_test.go`, `registry_test.go`, `result_test.go` and
  `sources_test.go`. No `command` test imports Bubble Tea. The tea forms
  have tests of their own in `commandtea`, as below.
* **New tests:** in `commandtea`, `TestCmdBatches`, `TestEffectNil`,
  `TestDispatch`, `TestDispatchAsync`, `TestWatch` and
  `TestAttachAndRunLoop`; in `command`, `TestCoreDispatchWithoutTea`; in
  `workspace`, `TestCommandEffectsReachTheProgram`. The last is beyond the
  step's list: no `workspace` test saw a command's effect reach the
  program, before this step or after it (with `commandtea.Effect` made to
  drop its message, and with the key path dropping its command's effect,
  every `workspace` test passed), so the twelve converted sites and
  `bindings.go` were held by the type checker alone. It focuses a pane
  whose `Focus` returns a command, through `commandtea.Dispatch` and
  through `alt+.`, and wants the pane's message.
* **Docs:** `docs/guides/commands.md` (the program example and a bullet on
  `command.Effect` and `commandtea`), `docs/guides/keymaps.md:52`,
  `docs/architecture.md` (the dependency rows, the `command` and
  `commandtea` rows, the cores, the imports, the tree), `docs/glossary.md`
  (`Effect`, reserved by A1.3, now built) and the index.
* **Checks:**
  * `gofmt -l .` empty; `GOWORK=off go vet ./...`: exit 0; `go test ./...`
    with `GOWORK=off` and in workspace mode: every package ok;
  * `golangci-lint` over `command/...`, `workspace/...`, `launch/...` and
    `internal/conformance/...` for linux, darwin and windows: `0 issues`;
    `go fix -diff ./...`: empty for each, once `go fix` had written one
    `reflect.TypeFor` in `commandtea_test.go`;
  * `make examples`: "46 case(s) run", "10 excerpt(s) checked", "clean",
    its excerpts unchanged;
  * `make apicheck`: "against v1.13.0, 22 incompatible change(s)",
    "clean", with a block named "0020-PLAN Step 10" of seven lines, each a
    declaration the step names;
  * `python3 -m tools.verify names` over `Effect`: "1 planned name(s),
    clean";
  * `markdownlint-cli2` over the changed guides, `architecture.md`, the
    glossary and the index: "0 issues".
* **Mutations,** each on a fresh copy, 17, every one killed by the test
  named. Rows 10.1 to 10.6 are the step's list; the rest see each new or
  changed test fail (Rule 5):

  | Mutation | Killed by |
  | :--- | :--- |
  | 10.1 Cmd drops a nested []Effect (TestCmdBatches), replacing one that crashed on a nil entry | `commandtea_test.go:83: a nested batch: [{}]` |
  | 10.2 Effect(nil) is a non-nil function (TestEffectNil) | `commandtea_test.go:94: Effect(nil) is not nil` |
  | 10.3 Dispatch runs an Async command before returning (command's TestDispatchAsync) | `dispatch_test.go:58: an Async command ran during Dispatch` |
  | 10.3b the same, seen by commandtea's TestDispatchAsync | `commandtea_test.go:148: the async command ran before its tea.Cmd did` |
  | 10.4 Watch returns the conflict without ChangedMsg (command's TestSlashClash) | `sources_test.go:56: Watch sent [{[{user.review review user:review builtin.review <nil>}]}], want ChangedMsg{2} too` |
  | 10.4b the same, seen by commandtea's TestWatch | `commandtea_test.go:197: a renaming change: command.ConflictMsg, want a tea.BatchMsg` |
  | 10.5 commandtea.Attach drops a message (TestAttachAndRunLoop) | `commandtea_test.go:250: Attach: the loop got nothing` |
  | 10.6 command/registry.go imports and uses Bubble Tea (TestCoresAreCharmFree) | `charmfree_test.go:172: command (GOOS=linux) links charm.land/bubbletea/v2: command → charm.land/bubbletea/v2` |
  | 10.6b the same, refused by depguard's charmfree | `command/registry.go:14:2: import 'charm.land/bubbletea/v2' is not allowed from list 'charmfree' … (depguard)` |
  | 10.7 the core's batch drops the PromptMsg (TestCoreDispatchWithoutTea) | `effect_test.go:39: a prompt's effect: 2 entries, first nil true; want 3, the first nil` |
  | 10.8 commandtea.Dispatch does not convert the batch (commandtea's TestDispatch) | `commandtea_test.go:117: Dispatch's message is []command.Effect, want a tea.BatchMsg` |
  | 10.9 RunLoop drops the command's effect (TestAttachAndRunLoop) | `commandtea_test.go:256: Attach: RunLoop's tea.Cmd sent [], want the effect's message` |
  | 10.10 workspace.focus's effect is lost (TestCommandEffectsReachTheProgram), replacing one TestBuiltinsMatchMethods could not see | `effect_test.go:72: workspace.focus through commandtea.Dispatch sent [{{workspace.focus {"pane":"logs"}  key  [map[workspace.focusedPane:session wor…` |
  | 10.11 a key's own command loses its effect (TestCommandEffectsReachTheProgram), replacing one that did not compile | `effect_test.go:78: alt+. moved focus logs → footer and sent [], want the new pane's focus message` |
  | 10.12 the core's batch drops its ResultMsg (command's TestDispatchLoop) | `dispatch_test.go:32: 0 ResultMsgs in [{}], want 1` |
  | 10.13 a LoopMsg's run loses its claim and runs nothing (command's TestAttachRunsOnLoop) | `loop_test.go:76: Run did not return` |
  | 10.14 commandtea.Effect runs c and drops its message (TestCommandEffectsReachTheProgram) | `effect_test.go:72: workspace.focus through commandtea.Dispatch sent [<nil> {{workspace.focus {"pane":"logs"}  key  [map[workspace.focusedPane:sessi…` |

  The first run had 16: 13 killed by their test; 10.1 killed by a nil
  dereference the mutation itself made, on the batch's nil entry, and
  10.11 by a build failure (the `commandtea` import left unused), so both
  replaced; and 10.10, `workspace.zoom`'s effect dropped, survived
  `TestBuiltinsMatchMethods`, which compares state that the command sets
  at once, so it was replaced by one on `workspace.focus` against the new
  test, with 10.14 added.
* **Gates,** `python3 -m tools.verify gates` over the step's 33 files,
  nothing skipped: macOS `gates: 19 passed, 0 failed, 0 skipped in 422s`; the Linux host
  `gates: 19 passed, 0 failed, 0 skipped in 961s`; the Windows host
  `gates: 19 passed, 0 failed, 0 skipped in 1815s`. `release-check` is
  "288 file(s) clean" on macOS and "292 file(s) clean" on Linux and
  Windows, with the API diff and examples gates.
* **Identifiers:** `python3 -m tools.verify scan`: `scan: clean (5 pattern(s))`.
* **Not done here:** the release and its migration notes, which are Step
  13's. Nothing is tagged.

### Step 11: `termkey` (2026-10-24)

* The owner committed Step 10 as `8644ae0` and said to proceed
  ("committed, proceed").
* **Deviation, 2026-10-24, the decoder's bound.**
  `TestRepliesReachParseReply`'s row "a reply of 1 KiB + 1" failed at
  every split. At the step's 1 KiB `MaxSequence` the decoder cut the
  1025-byte OSC 99 reply on the `ESC` of its `ESC \`, and
  `termcap.ParseReply`, which reads a bare `ESC` as an end (MADR A2.4),
  read the cut reply as a whole notification. Offered: leave out an `ESC`
  the cut falls on; raise the default to 4 KiB; or have hosts skip a
  `Truncated` input. The owner asked first for 4 KiB in both bounds. With
  both at 4 KiB the failure moves to a reply of 4097 bytes, and raising
  `ParseReply`'s cap would supersede
  `0005-MADR-terminal-capabilities-and-services.md`'s 1 KiB reply cap and
  let `termcaptea`'s fragment filter swallow up to 4 KiB of input. That
  went back to the owner with the cap's origin: 0005-MADR A1's reply caps
  took codex's 1 KiB probe budget
  (`0003-REPORT-agent-tui-ecosystem-research.md` §8.3), and every reply
  termcap asks for is far shorter. *Chosen:* `MaxSequence` 0 means 4 KiB,
  over `ParseReply`'s bound, and a string cut on an `ESC` leaves the `ESC`
  out, whatever bound a host sets; `ParseReply` keeps 1 KiB. No MADR
  decision or fact changes: D7 says only "bounded", and A2.4's 1 KiB is
  `ParseReply`'s. The step carries the note, and
  `TestRepliesReachParseReply` gains a row cut by a 1 KiB bound and one cut
  by the default.
* **The same run's second failing row,** "cut by an ESC before more", came
  from a change made during the step, not from the PLAN: the decoder had
  been changed to consume an `ESC` that cuts a string, as ultraviolet does,
  and `ParseReply` then read `OSC 11 … ESC` as a colour. The `ESC` is left
  to begin what follows, as ECMA-48 has it and as first written, and the
  `differ` table records the difference from ultraviolet. The step's own
  assertion decides it, so it was recorded and not stopped for.
* **Files beyond the step's list:** `termkey/codes.go`, the key codes,
  generated from ultraviolet's `key.go` const block in its order (139
  constants, with `KeyExtended`); `termkey/kitty.go`, the Kitty protocol's
  key map, ultraviolet's `kittyKeyMap` with its C0 additions; and
  `termkey/testdata/golden/corpus.golden`. The differential test replays
  "termkey's table", which lives in `termkey`'s test files and cannot be
  imported, so `TestCorpus` keeps the tables' inputs, a name and a quoted
  input a line, in a golden through `tuitest.Text`, and
  `internal/termevent` reads it; a row added without the golden fails
  `TestCorpus`. The golden was read in full: 489 rows.
* **`termkey`:**
  * `key.go`: `Mod`, a `uint16` `enum.Bits` with ultraviolet's values and
    the step's tokens; `KeyEvent`, with `Keystroke` and `String` as
    `tea.Key`'s; `names.go`, ultraviolet's `keyTypeString`.
  * `input.go`: `InputKind` (`key`, `paste`, `unknown`) and `Input`.
  * `decode.go`: `Decoder`, a state machine with no goroutine and no
    clock, reading D7's list. Beyond it, as ultraviolet reads them:
    urxvt's `CSI n $`, `CSI n ^` and `CSI n @`; `CSI 1 ; m R` as F3; an X10
    mouse report consumed with its three bytes; a C1 byte 0x80 to 0x9F as
    ctrl+alt and its letter; `ESC` and an introducer held until `Flush`,
    then alt and that key. A key a rune: `termkey` imports nothing for
    grapheme clusters. A paste keeps its bytes, invalid UTF-8 left out,
    and a stray `CSI 201 ~` is unknown. A string ends at BEL, ST, `ESC \`,
    CAN or SUB, a bare `ESC` as the input goes quiet, or an `ESC` before
    anything else, which is left out. Kitty's base-layout key no longer
    sets `ShiftedCode`, where ultraviolet's parser falls through.
  * `doc.go`: the package comment, the host's loop, A2.4's rule that a
    probing host does not `Flush` while its `Probe` awaits a reply, and
    the stability line.
  * `GOWORK=off go list -deps ./termkey`, for linux, darwin and windows:
    no package outside the standard library but `termkey` and
    `internal/enum`.
* **`internal/termevent/termkey_test.go`:** `TestKeyCodesMatchUltraviolet`,
  its 144 code rows generated from ultraviolet's names, each value and
  `Keystroke` equal, with the nine modifier bits and their names;
  `TestDecoderMatchesUltraviolet`, each corpus row through Bubble Tea's own
  reader, `uv.NewTerminalReader` with `StreamEvents`, read whole and then
  the end of the input, against `termkey`'s keys and pastes. A row in
  `differ` that agrees fails as stale. Its ten rows, each with its reason:
  the well-formed U+FFFD; a grapheme of two runes; Kitty's base key, twice;
  a paste holding an OSC; a paste not ended; a paste end with no start; an
  OSC cut by an `ESC`; 33 parameters; and win32-input-mode, which D7 leaves
  out of scope.
* **An upstream finding, not acted on.** Bubble Tea v2.0.10's reader panics
  on a CSI of 33 parameters: ultraviolet's `deserializeWin32Input`
  (`terminal_reader.go:463`, built on every OS) calls x/ansi v0.11.8's
  `DecodeSequence`, which runs past its 32-parameter buffer, the bug
  `termcap.ParseReply` already guards against. The differential test
  recovers the panic and lists the row. Reporting it upstream is the
  owner's call.
* **Conformance:** `enums` gains `termkey.InputKind` and `termkey.Mod`,
  and `bitSets` `termkey.Mod`. `TestOutOfRangeText` took bit 7 as free in
  every bit set, and `ModNumLock` is bit 7: it now takes the first bit
  above the named ones and fails when the type cannot hold it.
  `.golangci.yml`'s `charmfree` gains `**/termkey/*.go`, and `cores` gains
  `termkey`.
* **Docs:** `docs/architecture.md` (the dependency graph, the `termkey`
  row, the cores, the tree), `docs/glossary.md` (`KeyEvent`, `Mod`,
  `Decoder`, `Input` and `InputKind`, reserved by A1.3, now built, which
  leaves nothing reserved) and the index.
* **Checks:**
  * `gofmt -l` empty; `go vet` exit 0 on the packages changed;
  * `make lint`: `go fix -diff` empty and golangci-lint `0 issues` for
    linux, darwin and windows, after nine findings in the new code were
    fixed (`max` and `any` shadowed, a `rune` conversion, an empty-string
    test, and `csi`'s complexity, split with `scanParams`);
  * `make apicheck`: "against v1.13.0, 22 incompatible change(s)",
    "clean": the step only adds to the API, so it needs no block;
  * `python3 -m tools.verify names` over `KeyEvent`, `Mod`, `Decoder`,
    `Input` and `InputKind`: "5 planned name(s), clean";
  * `go test -race -count=10 ./termkey/ ./internal/termevent/`: ok, the
    differential test's reader goroutines included;
  * `FuzzDecode` for 90 s three times, the last on the final code: 22.4, 11.0 and
    9.7 million executions, no failure, no corpus written to the tree.
* **Mutations,** each on a fresh copy, 36, every one killed by the test
  named. Rows 11.1 to 11.6 are the step's list; the rest see each new
  test and assertion fail (Rule 5):

  | Mutation | Killed by |
  | :--- | :--- |
  | 11.1 the modifier parameter is read without the minus one (TestDecode/legacy) | `decode_test.go:334: CSI 1;2 A: "\x1b[1;2A" gives [key{alt+up raw "\x1b[1;2A"}], want [key{shift+up raw ""}]` |
  | 11.2 ESC before a key is not alt (TestDecode/alt) | `decode_test.go:334: alt+a: "\x1ba" gives [key{a raw "\x1ba"}], want [key{alt+a raw ""}]` |
  | 11.3 the paste is not bounded (TestPasteBounded) | `decode_test.go:428: a 16-byte paste over MaxPaste 8 gives [paste{"0123456789abcdef" truncated false} key{x raw "x"}], want [paste{"01234567" trunca…` |
  | 11.3b a paste of exactly MaxPaste is truncated (TestPasteBounded) | `decode_test.go:440: a paste of exactly MaxPaste gives [paste{"01234567" truncated true}], want it whole and not truncated` |
  | 11.4 Flush drops a lone ESC (TestFlushEscape) | `decode_test.go:399: Flush gives [], Pending false; want escape, and nothing pending` |
  | 11.4b Flush drops a paste not ended (TestFlushEscape) | `decode_test.go:414: a paste cut by a read: Flush gives [], Pending true; want [paste{"abc\x1b[20" truncated false}]` |
  | 11.5 Decode forgets a sequence split across calls (TestSplit) | `decode_test.go:379: alt+a: "\x1ba" split at 1 gives [key{a raw "a"}], want [key{alt+a raw ""}]` |
  | 11.5b the same, seen by FuzzDecode's seeds | `decode_test.go:510: "\x1ba" split at 1 gives [key{a raw "a"}], whole gives [key{alt+a raw "\x1ba"}]` |
  | 11.6 ModMeta is 16 (TestKeyCodesMatchUltraviolet) | `termkey_test.go:203: ModMeta: termkey 16, ultraviolet 8` |
  | 11.7 an upper-case letter keeps its case (TestDecode/text) | `decode_test.go:334: an upper-case letter: "A" gives [key{A raw "A"}], want [key{A raw ""}]` |
  | 11.8 FS to US are read 0x20 off (TestDecode/control) | `decode_test.go:334: FS: "\x1c" gives [key{ctrl+\| raw "\x1c"}], want [key{ctrl+\ raw ""}]` |
  | 11.9 SS3's rxvt arrows lose ctrl (TestDecode/ss3) | `decode_test.go:334: SS3 ctrl+up, rxvt: "\x1bOa" gives [key{up raw "\x1bOa"}], want [key{ctrl+up raw ""}]` |
  | 11.10 modifyOtherKeys gives shift no text (TestDecode/otherkeys) | `decode_test.go:334: modifyOtherKeys shift+A: "\x1b[27;2;65~" gives [key{shift+A raw "\x1b[27;2;65~"}], want [key{A raw ""}]` |
  | 11.11 Kitty's super reads as meta (TestDecode/kitty) | `decode_test.go:334: Kitty modifier bit 8: "\x1b[97;9u" gives [key{meta+a raw "\x1b[97;9u"}], want [key{super+a raw ""}]` |
  | 11.12 CSI 200 ~ does not begin a paste (TestDecode/paste) | `decode_test.go:334: a paste: "\x1b[200~hello\x1b[201~" gives [unknown{"\x1b[200~" truncated false} key{h raw "h"} key{e raw "e"} key{l raw "l"} key…` |
  | 11.13 a prefixed CSI is read as a key (TestDecode/reply) | `decode_test.go:334: the Kitty flags reply: "\x1b[?1u" gives [key{ctrl+a raw "\x1b[?1u"}], want [unknown{"\x1b[?1u" truncated false}]` |
  | 11.14 a key keeps no bytes (TestDecode, the bytes adding up) | `decode_test.go:348: the Kitty flags reply, then a key: the inputs' bytes add up to "\x1b[?31u", want "\x1b[?31ua"` |
  | 11.15 Keystroke puts alt before ctrl (TestKeystroke) | `key_test.go:27: alt+ctrl+shift+up.Keystroke() = "alt+ctrl+shift+up", want "ctrl+alt+shift+up"` |
  | 11.16 Keystroke names a modifier key's own modifier (TestKeystroke) | `key_test.go:27: shift+leftshift.Keystroke() = "shift+leftshift", want "leftshift"` |
  | 11.17 String gives a space's text (TestKeyString) | `key_test.go:43:  .String() = " ", want "space"` |
  | 11.18 Contains is true for any one modifier (TestMod) | `key_test.go:57: Contains is wrong for alt\|ctrl` |
  | 11.19 a row the corpus golden lacks (TestCorpus) | `decode_test.go:370: tuitest: corpus color.utf8.0: line 479 differs` |
  | 11.20 two rows of one name (TestCorpus) | `decode_test.go:364: two rows are named "a digit"` |
  | 11.21 a cut on an ESC keeps the ESC (TestSequenceBounded) | `decode_test.go:467: an OSC whose ST crosses the bound: gives [unknown{"\x1b]zzzzzzzzzzzzz\x1b" truncated true} key{x raw "x"}], want [unknown{"\x1b…` |
  | 11.21b the same, seen by TestRepliesReachParseReply | `reply_test.go:102: a reply of 1 KiB + 1, cut by a 1 KiB bound, split at 0: ParseReply(Raw) = {Kind:notify Raw: Params:[] Mode:0 Setting:not-recogni…` |
  | 11.22 MaxSequence 0 means 1 KiB (TestSequenceBounded) | `decode_test.go:485: an OSC over the default bound gives 1 inputs, the first of 1024 bytes; want one input of 4 KiB, truncated` |
  | 11.23 the ESC that cuts a string is consumed with it (TestRepliesReachParseReply) | `reply_test.go:102: cut by an ESC before more, split at 0: ParseReply(Raw) = {Kind:color Raw: Params:[] Mode:0 Setting:not-recognized OSC:11 Index:0…` |
  | 11.24 SS3's digits are not bounded (TestSequenceBounded) | `decode_test.go:474: an SS3 over the bound: holds 17 bytes, over MaxSequence` |
  | 11.24b a CSI of exactly MaxSequence is cut (TestSequenceBounded) | `decode_test.go:467: a CSI of exactly the bound: gives [unknown{"\x1b[1111111111111~" truncated true}], want [unknown{"\x1b[1111111111111~" truncate…` |
  | 11.25 caps lock types lower case (TestDecoderMatchesUltraviolet) | `termkey_test.go:372: Kitty modifier bit 64: "\x1b[97;65u": termkey gives [{paste:false text: key:{Code:97 Mod:64 Text:a ShiftedCode:0 BaseCode:0 Is…` |
  | 11.26 ultraviolet's fall-through restored, so a differ row is stale (TestDecoderMatchesUltraviolet) | `termkey_test.go:374: Kitty ctrl+ф, its base key: the decoders agree, but differ lists it (ultraviolet's Kitty parser falls through from the base-la…` |
  | 11.27 page up is named pageup (TestKeyCodesMatchUltraviolet) | `termkey_test.go:198: KeyPgUp: termkey names it "pageup", ultraviolet "pgup"` |
  | 11.28b the free bit is wider than the type holds (TestOutOfRangeText), replacing 11.28, which left `math/bits` unused and did not compile | `enums_test.go:233: command.Surface: the test wants bit 0 named and a free bit above the named ones, in 0x1f` |
  | 11.29 two of Mod's bits share a token (TestEveryEnumRoundTrips) | `enums_test.go:196: termkey.Mod: 64 and 128 share the token "capslock"` |
  | 11.30 termkey imports Bubble Tea (TestCoresAreCharmFree) | `charmfree_test.go:173: termkey (GOOS=linux) links charm.land/bubbletea/v2: termkey → charm.land/bubbletea/v2` |
  | 11.30b the same, refused by depguard's charmfree | `termkey/key.go:6:2: import 'charm.land/bubbletea/v2' is not allowed from list 'charmfree' … (depguard)` |

  The first run had 36, all killed, but 11.28 by a build failure (the
  mutation left `math/bits` unused), so it was replaced by 11.28b; and
  11.30b's depguard line was read from its kept copy, since the run
  printed none.
* **Gates,** `python3 -m tools.verify gates` over the step's 19 files,
  nothing skipped: macOS `gates: 19 passed, 0 failed, 0 skipped in 409s`; the Linux host
  `gates: 19 passed, 0 failed, 0 skipped in 900s`; the Windows host
  `gates: 19 passed, 0 failed, 0 skipped in 1801s`, the differential test
  among them, its reader on a plain `io.Reader` on each. `release-check`
  is "292 file(s) clean" on macOS, which lists tracked files and so not
  the step's eleven new Go files, and "303 file(s) clean" on Linux and
  Windows, whose copies hold them, with the API diff and examples gates.
* **Identifiers:** `python3 -m tools.verify scan`: `scan: clean (5 pattern(s))`,
  over the change and the untracked files.
* **Not done here:** `keymap` and `keymaptea`, with D7's walk of every
  code and modifier Bubble Tea exports, which are Step 12's; the release,
  Step 13's. D7's stricter rule for `termkey`, the standard library and
  `internal/enum` only, holds today (`go list -deps` above), but the gates
  check only that the closure has no Charm module but `x/ansi`: nothing
  refuses an `x/ansi` import in `termkey`. Nothing is tagged.
