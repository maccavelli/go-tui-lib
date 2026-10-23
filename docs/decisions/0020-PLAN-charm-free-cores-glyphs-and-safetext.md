---
status: in-progress
date: 2026-10-23
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
| Outside `termcap`, only `termcap/termcaptest/termcaptest.go` uses `Prober`; `termsvc/notify.go` names `Prober` and `EnvCaps` in its documentation; `launch` uses only `termcap.Env` | 8 |
| `Notifier.NotifyContext` checks the policy and numbers on the caller's goroutine, and calls a backend inside its command (`notify.go:279-300`); `CopyContext` writes OSC 52 through `tea.SetClipboard` (`clipboard.go:130-155`) | 8 |
| `Registry.Dispatch` runs a `Loop` command at once and an `Async` one inside its command, then batches `Result.Cmd`, `ResultMsg` and, for a `Prompt` or `Forward`, `PromptMsg` (`dispatch.go:24-40`, `:244-250`) | 9 |
| `Result{Cmd: …}` is set at nine sites in `workspace/commands.go` and once in `command/builtin.go:57`; `launch/run.go:98` calls `Attach(p.Send)` | 9 |
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
| 6 | `README.md`, `docs/` | release `v0.13.0` |
| 7 | `termcap/`, `internal/conformance/` | the I/O-free probe engine and `ParseReply` (D4, A2) |
| 8 | `termcap/termcaptea/`, `termcap/termcaptest/`, `termcap/` (tests moved), `internal/conformance/`, `docs/` | `termcaptea.Prober` over the engine, the differential test (D4, A2) |
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

### Step 6: release `v0.13.0`

* **Docs:**
  * `docs/architecture.md`: `safetext`'s row and imports; `glyph`'s new
    fields; `internal/sanitize` as a wrapper; the closure gate and the
    cores' list;
  * `docs/guides/migrating.md`, "Migrating to `v0.13.0`": *Added*
    (`safetext`, the eight glyph fields, `fuzzy`), *Changed* (the library's
    output shows invalid UTF-8 as U+FFFD, one per maximal subpart, where it
    dropped it);
  * `docs/glossary.md`: `Rules` and `Sanitizer` move from "Reserved" to
    "Built";
  * `README.md` Status; `docs/README.md` rows.
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

### Step 7: `termcap`, the I/O-free probe engine (D4, A2)

*Starts only once MADR A2 is accepted.*

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
`docs/architecture.md`; `docs/glossary.md`.

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
  func()`; `WithLoop(send func(tea.Msg)) command.RegistryOption`;
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
  1 KiB); `func (d *Decoder) Decode(p []byte, out []Input) []Input`;
  `func (d *Decoder) Pending() bool`;
  `func (d *Decoder) Flush(out []Input) []Input`; and
  `const DefaultEscTimeout = 50 * time.Millisecond`. It decodes MADR D7's
  list. A paste over `MaxPaste` keeps its first `MaxPaste` bytes, sets
  `Truncated`, and consumes the rest up to `CSI 201 ~`. A sequence over
  `MaxSequence` is one `InputUnknown` holding its first `MaxSequence`
  bytes, the rest consumed up to its end. `Flush` gives up a held lone
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
