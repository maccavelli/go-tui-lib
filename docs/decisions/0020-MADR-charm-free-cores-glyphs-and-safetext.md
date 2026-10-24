---
status: accepted
date: 2026-10-23
decision-makers: the repository owner
consulted: "issues #2, #3, #4, #5, #11 and #12 of this repository's tracker, filed by gobble-cli on 2026-10-23, with their comments; gobble-cli docs/decisions/0032-MADR-go-tui-lib-integration-boundary.md and its amendment A1, read at gobble-cli 09673ba; this repository at 63c68d6; the module sources of charm.land/bubbletea/v2 v2.0.10, github.com/charmbracelet/colorprofile v0.4.3, github.com/charmbracelet/ultraviolet v0.0.0-20260811164956-006e29f97886 and github.com/charmbracelet/x/ansi v0.11.8; the external sources listed under More Information"
informed: gobble-cli, pi-go
---
# Give a line-mode CLI Charm-free cores of `termcap`, `termsvc`, `command` and `keymap`, with their Bubble Tea forms in sub-packages, status and meter glyphs in `glyph`, and `safetext` as the one sanitizer

## Context and Problem Statement

On 2026-10-23 the owner asked for an assessment of the open issues, and
then: "draft the madr, add #11 to scope for it also in addition to
safetext, ground in codebase facts and research and ensure it is detailed
and complete." The MADR the assessment proposed covers issues #2 to #5,
which ask for `termcap`, `termsvc`, `command` and `keymap` without Bubble
Tea. The owner added #11, status and meter glyphs in `glyph`, and #12,
the sanitizer exported as `safetext`.

All six issues come from one decision in another repository. gobble-cli
`docs/decisions/0032-MADR-go-tui-lib-integration-boundary.md` (accepted
2026-10-22) lets gobble's native line-mode CLI, `internal/cli`, import a
go-tui-lib package only when that package's whole import closure holds no
Charm module. Its amendment A1 admits `github.com/charmbracelet/x/ansi`,
with the text modules it requires, and nothing else of Charm: no
`charm.land/...` (Bubble Tea, Bubbles, Lip Gloss), no
`github.com/charmbracelet/colorprofile`, no
`github.com/charmbracelet/ultraviolet`, no `github.com/charmbracelet/x/term`
and nothing else under `github.com/charmbracelet/` (its A1.1). gobble
builds no copy of what go-tui-lib does. It waits for the library, and
files what it needs here (its D5). The first comment on each of #2 to #10
records the A1 relaxation, and each issue's first text predates it.

Every fact below was read on 2026-10-23, at `63c68d6`, from each package's
non-test files, and from `GOWORK=off go list -deps`.

### Which packages link which Charm modules today

| Package | Charm modules in its import closure |
| :--- | :--- |
| `glyph`, `when`, `layout` | none |
| `fuzzy`, `internal/sanitize` | `x/ansi` |
| `termcap`, `termsvc`, `command` | Bubble Tea, `colorprofile`, `ultraviolet`, `x/ansi` |
| `keymap` | Bubble Tea, Bubbles, Lip Gloss, `colorprofile`, `ultraviolet`, `x/ansi` |

So gobble's CLI can use `glyph`, `when`, `layout` and, under A1, `fuzzy`
today. `fuzzy` is not yet in a tag (it came after `v0.12.0`), which is
all #6 asks for. #6 is out of this record's scope: it needs a release, not
a decision.

### Where the Charm imports are

The packages import each other: `termsvc` imports `termcap`; `keymap`
imports `command` and `termcap`; `launch` imports `command` and `termcap`;
`workspace` imports `command` and `keymap`. A core is only Charm-free when
every core it imports is, so the order is `termcap` before `termsvc`, and
`termcap` and `command` before `keymap`.

**`termcap`** (issue #2).

* `prober.go` imports Bubble Tea, `x/ansi`, `internal/sanitize`,
  `internal/teamsg` and `internal/termevent`, which imports ultraviolet.
  The `Prober` is a Bubble Tea component throughout: `Init` (`:215`),
  `Quit` (`:242`) and `Update` (`:252`) return `tea.Cmd`, and
  `IsReplyFragment` (`:409`) takes a `tea.Msg`. `Reply.Msg` (`:35`) is a
  `tea.Msg`. `Query.Parse` takes a `Reply`, and `WithQuery` takes a
  `Query`.
* `views.go` imports Bubble Tea for one type:
  `KeyboardCaps.Enhancements` (`:16`) is a `tea.KeyboardEnhancements`, and
  `KeyboardFlags` (`:38`) returns one.
* `caps.go` imports `colorprofile` for one field: `Caps.Profile` (`:45`)
  is a `colorprofile.Profile`. The JSON form writes it by
  `colorprofile.Profile.String()` (`:79-80`), whose tokens are `Unknown`,
  `NoTTY`, `Ascii`, `ANSI`, `ANSI256` and `TrueColor`.
* `report.go` imports `x/ansi`, and `env.go`, `identity.go` and `tmux.go`
  import `internal/sanitize`. Both are allowed under gobble's A1.
* What a line-mode CLI wants, `EnvCaps`, `FromEnv`, `Caps`, its views,
  `Findings` and `Report`, names no Charm type except `Caps.Profile` and
  `KeyboardCaps.Enhancements`.

**`termsvc`** (issue #3).

* `termsvc.go` imports `x/ansi`, `internal/sanitize` and `termcap`. It
  holds the byte functions: `Wrap`, `SanitizeTitle`, `PromptStart`,
  `CommandStart`, `CommandExecuted`, `CommandFinished`.
* `beacon.go`, `clipboard.go`, `links.go` and `notify.go` import Bubble
  Tea, four of them through `internal/teamsg` as well. The functions that
  name a Bubble Tea type: `ActivityBeacon` (`beacon.go:135`), `Copy` and
  `CopyContext` (`clipboard.go:123`, `:130`), `LinkPolicy.Open`
  (`links.go:82`), and `Notifier.Update`, `Notify` and `NotifyContext`
  (`notify.go:255`, `:272`, `:279`).
* The byte forms a CLI needs exist already: `Notifier.Sequence`,
  `CopySequence`, `CopyPlan`, `Link`, `Activity`, `Pointer`. But a CLI
  gives a `Notifier` its capabilities through
  `Update(termcap.CapsMsg{…})` (the package documentation says so), which
  is a method that takes a `tea.Msg`.
* `termsvc` imports `termcap`, so it is Charm-free only once `termcap` is.

**`command`** (issue #4).

* `Result.Cmd` (`handler.go:45`) is a `tea.Cmd`, and `Result` is what
  every `Handler` returns, so the handler interface itself names Bubble
  Tea. Inside the repository, `workspace/commands.go` sets it at nine
  sites (`:91`, `:142`, `:162`, `:179`, `:193`, `:203`, `:213`, `:249`,
  `:276`), and the built-in `app.quit` at `builtin.go:57`.
* `Registry.Dispatch` (`dispatch.go:24`) returns a `tea.Cmd`;
  `Registry.Attach` (`dispatch.go:80`) and `WithLoop`
  (`registry.go:156`) take `func(tea.Msg)`; `Registry.Watch` and
  `WatchContext` (`registry.go:467`, `:472`) return a `tea.Cmd` built from
  a `tea.BatchMsg` (`:485`); `LoopMsg.Run` (`messages.go:45`) returns a
  `tea.Cmd`.
* What a line-mode CLI wants is already in files with no Charm import:
  `Registry.ParseArgs` and `Registry.Complete` (`cliargs.go:42`, `:288`),
  `Params` (`cliargs.go:244`), `Registry.ParseSlash` (`slash.go:27`),
  `WriteResult` (`result.go:55`), `Registry.Run` (`dispatch.go:51`), the
  gate's `Verdict`, and the ACP and MCP forms (`acp.go`, `acpexport.go`,
  `mcp.go`, `mcpprompt.go`). `registry.go` links Bubble Tea, so the
  package does.
* The message types, `ResultMsg`, `PromptMsg`, `ChangedMsg`,
  `ConflictMsg` and `QuitRequestMsg`, are plain structs that name no
  Bubble Tea type. `LoopMsg` holds a `func() tea.Cmd`.

**`keymap`** (issue #5).

* The package is 2,717 non-test lines. Keys are Bubble Tea's:
  `Normalize` (`normalize.go:33`) takes and returns a `tea.Key`;
  `Stroke.Mod` (`notation.go:65`) is a `tea.KeyMod`, and `Stroke.Code`
  (`:66`) holds Bubble Tea's key constants; `Stroke.Matches`
  (`notation.go:400`) takes a `tea.Key`; `Explanation.Raw` and
  `Normalized` (`whichkey.go:78-79`) are `tea.Key`s, and `Explain`
  (`:95`) takes one; `Outcome.Replay` (`matcher.go:26`) is a
  `[]tea.KeyPressMsg`; `Dispatcher.Update` (`matcher.go:104`) takes a
  `tea.Msg` and returns a `tea.Cmd`; `Settings.Tick` (`keymap.go:164`)
  is a function of Bubble Tea types.
* `help.go` imports Bubbles: `Keymap.KeyBinding` (`:59`) returns a
  `key.Binding` and `Keymap.Help` (`:90`) a `help.KeyMap`.
  `Stroke.BubblesKeys` writes the strings Bubbles compares.
* The canonical notation is Bubble Tea's `Key.Keystroke()` form
  (`docs/decisions/0007-MADR-keymap-engine.md`), and `notation.go` and
  `label.go` build their name tables from Bubble Tea's key constants
  (`notation.go` names `tea.` 43 times).
* Bubble Tea's key codes and modifiers are ultraviolet's, re-exported:
  `KeyUp = uv.KeyUp` and the rest (`bubbletea key.go:12-41`), and
  `type KeyMod = uv.KeyMod` (`mod.go:6`). `tea.Key` is Bubble Tea's own
  struct (`key.go:302`).
* What a line-mode CLI wants, `LoadVSCode`, `ExportVSCode`, `Schema`,
  `ParseStroke`, `ParseSequence` and `Stroke.Label`, decodes no key. It
  links Bubble Tea because `Stroke` names `tea.KeyMod`.
* Issue #5 also asks for a key decoder from raw bytes, legacy and
  enhanced, for gobble's own line editor. go-tui-lib has none: Bubble
  Tea's is ultraviolet's, in ultraviolet's root package `uv`
  (`decoder.go`, 2,102 lines, with `key_table.go`, 664 lines, MIT
  licence), exported as `uv.EventDecoder`. That package imports
  `github.com/charmbracelet/x/windows` and `github.com/rivo/uniseg`, and,
  in other files of the same package, `colorprofile`; ultraviolet has no
  tag at all. gobble refuses it. This repository lets only
  `internal/cells` and `internal/termevent` import it
  (`docs/decisions/0005-MADR-terminal-capabilities-and-services.md` §1).
  `github.com/charmbracelet/x/input`, the decoder Bubble Tea used before
  ultraviolet (it moved in September 2025), is its own module, but it
  requires `x/windows`, and the `x` repository calls its packages
  experimental, with no promise of compatibility.

**`internal/teamsg`** is 11 lines: `Cmd[M](m M) tea.Cmd`. Every package
that sends a message through a command uses it, so it moves with the
commands.

### The sanitizer (issue #12)

* `internal/sanitize` (272 lines, `x/ansi`) is the library's one
  sanitizer (`docs/decisions/0014-PLAN-hardening.md` Step 2, finding C6).
  It exports `Line`, `Styled`, `Truncate`, `Token` and `HasControl`. The
  library calls them 28 times: `Line` 19 times (in `workspace`, `termsvc`,
  `termcap` and `command`), `Truncate` 4, `Token` 2, `HasControl` 2 and
  `Styled` 1.
* It removes (`sanitize.go:26-38`): C0, DEL, C1, the Bidi_Control set
  (U+061C, U+200E, U+200F, U+202A–U+202E, U+2066–U+2069), U+200B,
  U+FEFF, U+2060–U+2064, U+00AD and the tag characters U+E0000–U+E007F.
  It drops invalid UTF-8 (`:53`). It turns each run of `\r`, `\n`, `\t`,
  U+2028 and U+2029 into one space (`:40-66`). `Line` strips with
  `ansi.Strip` and `Styled` parses with `ansi.DecodeSequence`, keeping SGR
  with at most 32 parameters and OSC 8 with printable-ASCII parameters,
  whatever the scheme (`:158-194`). `HasControl` is `Line(s) != s`
  (`:272`).
* `safetext` is already decided:
  `docs/decisions/0009-MADR-streaming-content-engine.md` §2 (accepted)
  gives its API and grammar, A1 (accepted) adds the `Command()` and
  `StatusLine()` presets, `Interpret`, and the `Escape`, `KeepSGR` and
  `LinkSchemes` fields, and A4 renames `Policy` to `Rules` and `Filter` to
  `Sanitizer`. §1 says it imports only the standard library and `glyph`,
  and that `glyph.Set` gains `Control` (`␛`, ASCII `^`). A4 leaves to the
  PLAN whether `safetext` wraps `internal/sanitize` or replaces it, "so
  that the library keeps one sanitizer".
* `0009-PLAN-streaming-content-engine.md` is `proposed`, and it still
  uses the names A4 replaced. `safetext` is its Step 2, before `frame`,
  `inputfilter` and `stream`, and depends on none of them. A1's additions
  are its Step A1.1, after the others. `Interpret` advances by cell width
  with `x/ansi` (A1), so a `safetext` that holds it is no longer
  standard-library only.
* 0009 §2 and `internal/sanitize` do not agree:
  * invalid UTF-8: 0009 makes it U+FFFD; `internal/sanitize` drops it;
  * what is removed: 0009's `Bidi` field names only U+202A–U+202E and
    U+2066–U+2069; `internal/sanitize` also removes U+061C, U+200E,
    U+200F, the zero-width and format characters, and the tags;
  * line breaks: 0009's `Flatten` makes each `\n` and `\t` one space;
    `internal/sanitize` makes each run one space, and treats U+2028 and
    U+2029 as breaks;
  * links: 0009's `StatusLine()` keeps OSC 8 only for http, https and
    mailto with a host; `Styled` keeps any printable-ASCII target.
* gobble's comment on #12 asks for `safetext` as its own step ahead of
  the rest of 0009, for one sanitizer, for a combination that keeps `\n`
  and `\t` and shows every other control escaped (tool output), and gives
  measurements of `internal/sanitize` at `v0.12.0` with four changes tried
  on a scratch copy. Those figures are gobble's, taken on darwin/arm64
  with Go 1.27, three runs each; this record has not reproduced them.

| Input, 4 KiB | Function | `v0.12.0` | with gobble's changes |
| :--- | :--- | ---: | ---: |
| clean ASCII | `Line`, `Styled`, `HasControl` | 2.0 µs, 0 allocs | 2.0 µs, 0 allocs |
| clean CJK and emoji | `Line` | 7.9 µs, 0 allocs | 7.9 µs, 0 allocs |
| tool output | `Line` | 45.7 µs, 9 allocs | 6.5 µs, 1 alloc |
| tool output | `Styled` | 49.0 µs, 1 alloc | 5.9 µs, 1 alloc |
| tool output | `HasControl` | 42.6 µs, 9 allocs | 8 ns, 0 allocs |
| styled (SGR, OSC 8) | `Line` | 22.4 µs, 8 allocs | 14.6 µs, 8 allocs |
| styled | `HasControl` | 21.1 µs, 8 allocs | 4 ns, 0 allocs |
| hostile (CSI, OSC, bidi, raw C1) | `Styled` | 38.1 µs, 1 alloc | 33.6 µs, 1 alloc |

  Its differential fuzz found one trap: on invalid UTF-8 a raw C1 byte
  can open a sequence for `x/ansi`'s parser, so `"\x90q\x1b\\x"` gave
  `q\x` before the change and `qx` after it.

### The glyphs (issue #11)

* `glyph.Set` (`glyph.go`) has 18 single glyphs besides the four
  borders, among them `Focus` (`▸`, `>`), `Ellipsis` (`…`, `~`) and
  `Bullet` (`•`, `*`). It has no status glyph and no meter cell.
* The package's rule: each glyph is one rune and one cell wide in both
  sets, so a view laid out for one set fits the other; `Keys` is the one
  exception (`docs/decisions/0007-MADR-keymap-engine.md` A8). The tests
  walk the struct by reflection: `TestEveryGlyphIsOneCell`,
  `TestOneCellInBothMethods`, `TestTierWidths` and `TestASCIISetIsASCII`
  cover a new field with no change, and two of them count 70 glyphs
  (`glyph_test.go:54`, `:110`).
* `Ellipsis` is a truncation tail inside a cell budget:
  `workspace/render.go:299` and `:337` pass it to `sanitize.Truncate`,
  and `workspace/plain.go:125` uses it for the help view.
* gobble draws seven glyphs of its own, and asks for five more meanings:

| gobble | Unicode | ASCII | `glyph.Set` today |
| :--- | :--- | :--- | :--- |
| Bullet | `•` | `*` | `Bullet`, the same |
| Arrow, a tool call's header | `▸` | `>` | `Focus`, the same glyphs, another meaning |
| Ellipsis | `…` | `...` | `Ellipsis`, ASCII `~` |
| Check, done | `✓` | `+` | none |
| Cross, failed | `✗` | `x` | none |
| Bar, a meter's filled part | `━` | `#` | none (`Heavy.Top` is `━`) |
| BarEmpty, its empty part | `╌` | `-` | none |

### The gates and the policy

* **Nothing checks an import closure.** depguard in `.golangci.yml`
  checks each file's own imports (its source loops over each file's
  import specs and never reads the dependency graph): the `forbidden`
  rule refuses the Charm v1 paths everywhere, `ultraviolet` is allowed
  only under
  `internal/cells` and `internal/termevent`, and `xterm` only under
  `launch`. `internal/conformance` type-checks every package and resolves
  every use, but it checks uses, not what a package links.
* **API rule 1** (AGENTS.md, "API conventions") keeps a renamed name for
  one minor release as a wrapper, an alias or a deprecated constant, and
  keeps both fields of a renamed struct field. A record that needs an
  exception says so.
* **A wrapper cannot stay behind here.** A sub-package that adapts
  `termcap` to Bubble Tea imports `termcap`. An alias or wrapper for it
  in `termcap` would make `termcap` import the sub-package: an import
  cycle, which Go refuses. And it would put Bubble Tea back in the closure
  this record removes. A field whose type changes, such as `Caps.Profile`
  or `Stroke.Mod`, has no compatible form at all.
* **API rule 3** forbids two public packages exporting a type of the same
  name unless `docs/glossary.md` and `sharedNames` allow it, and the test
  counts an alias as a declaration (`conformance_test.go:357-359`). The
  name `Key` is `when.Key` already, and `Profile` is
  `termcap/termcaptest.Profile`. `Query` is `termcap`'s in the glossary.
* **API rule 5** gives every exported enum stable lowercase tokens from
  `internal/enum`. `colorprofile`'s tokens are mixed-case.
* **Consumers.** Neither gobble-cli (at `09673ba`) nor pi-go (at
  `3ff552f`, which has no Go module yet) requires go-tui-lib. No known
  module calls the APIs that move.
* **Release state.** `v0.12.0` is the latest tag.
  `0008-PLAN-command-palette.md` is in progress, its `palette` steps
  (5 to 9) still to run, and `palette` will dispatch through `command`
  and `keymap`. `0009-PLAN-streaming-content-engine.md` is proposed.

### The question

How does go-tui-lib give a line-mode CLI the headless parts of
`termcap`, `termsvc`, `command` and `keymap`, with a key decoder, the
glyphs it draws and the sanitizer it cleans text with, each linking no
Charm module but `x/ansi`, without forking any of them and without
taking anything from the TUI that uses them today?

## Decision Drivers

* **One implementation for both modes.** gobble's 0032-MADR D3: a
  line-mode CLI and the TUI share each element. A second copy of the
  facts, the registry, the notation or the sanitizer, in either
  repository, is what both records set out to prevent.
* **The closure, not the import.** A consumer's binary links the whole
  closure. A gate that reads direct imports, as depguard does, lets a
  core regain Bubble Tea through any core it imports.
* **The TUI keeps every capability.** Nothing a Bubble Tea program can do
  today goes away; only where it is imported from may change.
* **The API policy.** One name means one thing (rule 3); options stay
  opaque (rule 4); enums have lowercase tokens (rule 5); deprecation
  (rule 1) is followed where it can be, and an exception is stated where
  it cannot.
* **gobble's timeline.** gobble waits for these tags (its D3, D7). Every
  release the work spreads over is one more release gobble waits.
* **Small, ordered steps.** Each package's split is reviewable on its own,
  and leaves the tree green.
* **Untrusted text is a security boundary.** Model output, tool output,
  file names and terminal replies reach the screen. A sanitizer that
  removes less than today's, or that two modes run differently, is a
  regression.

## Considered Options

* **A. Charm-free cores at today's import paths, the Bubble Tea forms in
  a sub-package of each, one breaking minor release per split, and a
  closure gate.**
* **B. New Charm-free packages beside today's, which keep their paths as
  the Bubble Tea face through aliases.**
* **C. The same split, over two releases: the core forms and the
  sub-packages added first, the Bubble Tea names deprecated, and removed
  in the next minor release.**
* **D. Build tags that leave the Bubble Tea files out of a CLI's build.**
* **E. Erase the Bubble Tea types to `any` in place, so the packages stop
  importing Bubble Tea without moving anything.**
* **F. Change nothing in these packages; gobble keeps its own copies.**

A separate Go module for the adapters (0010-MADR's pattern) is not an
option of its own. What a binary compiles and links is its import
closure, so a package split gives gobble everything its boundary asks
for, with none of the two-step releases. A module split would also keep
the Charm modules out of gobble's module graph and `go.sum`, where module
graph pruning leaves them listed (Go Modules Reference, "Module graph
pruning"); gobble's 0032-MADR D1 and A1 ask about the closure, not the
graph.

## Decision Outcome

*The owner answered Q1 to Q8 on 2026-10-23 (Owner questions, below).
Under Q1 option A's splits land together, in one breaking release rather
than one release each; D10 and D12 say so.*

Chosen option: **"A"**, because it is the only option that gives gobble
clean closures without a release of transitional names, keeps every import path a CLI
already uses, and keeps every Bubble Tea capability in a package of its
own, without aliases the glossary must carry. B doubles the API and
still breaks every method that names Bubble Tea; C keeps the policy's
form but, with no known consumer of the moved names, buys a release of
transitional names for nobody, and delays gobble a release; D and E give
up type safety or build sanity; F contradicts both repositories' records.

The decision has twelve parts, D1 to D12.

### D1. The boundary: what "Charm-free" means here

* **A core package's import closure**, its non-test packages and every
  package they import, transitively, as `GOWORK=off go list -deps`
  reports it, **holds no package under `charm.land/` or
  `github.com/charmbracelet/`, except `github.com/charmbracelet/x/ansi`
  and the packages beneath it** (`x/ansi/parser`). The modules `x/ansi`
  requires (`github.com/clipperhouse/displaywidth`,
  `github.com/clipperhouse/uax29/v2`, `github.com/lucasb-eyer/go-colorful`,
  `github.com/mattn/go-runewidth`) are allowed. This is gobble's A1.1
  boundary, exactly.
* **The cores**, after this record: `glyph`, `when`, `layout`, `fuzzy`,
  `safetext`, `termkey` (D7), `termcap`, `termsvc`, `command` and
  `keymap`, and the internal packages they import.
* **The rest stay TUI packages:** `theme`, `workspace`, `launch`
  (`x/term`, Bubble Tea), `tuitest` (`colorprofile`), the adapters of D3,
  and the test kits `termcap/termcaptest` and `launch/launchtest`.
* **Test files do not count.** A core's tests may import the adapter or
  Bubble Tea where they test the pairing, because no consumer links a
  test. The gate (D11) reads non-test closures.
* A package joins or leaves the cores only by a record.

### D2. The split rule: what moves

* **A declaration moves to the package's adapter when its signature, its
  fields or its method set names a Charm type, and only then.** Types
  that name none stay where they are, the Bubble Tea message structs
  among them (`termcap.CapsMsg`, `command.ResultMsg`, `PromptMsg`,
  `ChangedMsg`, `ConflictMsg`, `QuitRequestMsg`, `keymap.TimeoutMsg`).
  They are plain structs; a TUI still switches on them where it does
  today.
* **Every effect gets a core form the adapter wraps.** Where a function
  returns a `tea.Cmd` today, the core gains the synchronous step under it,
  bytes to write, a result, or a deadline, and the adapter turns that step
  into the command. Logic is never written twice: the adapter calls the
  core.
* **A field whose type is a Charm type changes to a core type**, with the
  same values, and the adapter converts. D4 to D7 name each.
* **Message delivery moves.** `internal/teamsg` is used only by adapters
  after this record.

### D3. The adapters: one sub-package each, named `<pkg>tea`

| Core | Adapter | Holds |
| :--- | :--- | :--- |
| `termcap` | `termcap/termcaptea` | the `Prober`, `Query`, `Reply`, `WithQuery`, `KeyboardEnhancements` |
| `termsvc` | `termsvc/termsvctea` | the commands: notify, copy, open, the beacon tick, and the notifier's observer |
| `command` | `command/commandtea` | `Dispatch`, `Watch`, `Attach`, `WithLoop`, the `Effect` conversions |
| `keymap` | `keymap/keymaptea` | the key conversions, the dispatcher's `Update`, the timer, `Help`, `KeyBinding`, `BubblesKeys` |

* **The name is the package's plus `tea`,** as `termcaptest` and
  `launchtest` are the package's plus `test`. It never shadows the `tea`
  identifier a TUI file imports Bubble Tea as, and it says which framework
  it adapts. A later adapter for another framework would be `<pkg><name>`
  beside it.
* **Each adapter imports its core and Bubble Tea,** and nothing imports
  an adapter but programs, `workspace`, `launch`, the future `palette`,
  and tests.
* **Each adapter states its stability line** and follows every TUI and
  API convention as any public package does.
* **A moved type keeps its name** (`termcaptea.Prober`), so the glossary
  records only the move, not a new meaning.

### D4. `termcap` (issue #2)

* **The `Prober` moves to `termcaptea`,** whole: `NewProber`, `Init`,
  `Update`, `Quit`, `Restore`, `Caps`, `SetTmux`, `IsReplyFragment`. So do
  `Query`, `Reply` and `WithQuery`, which carry `tea.Msg`, and
  `views.go`'s `KeyboardFlags`, as
  `termcaptea.KeyboardEnhancements(c termcap.Caps) tea.KeyboardEnhancements`.
  The glossary's `Query` row moves to `termcaptea`.
* **`termcap.Option` stays in `termcap`, and stays opaque.** `EnvCaps`
  takes it, and so does the prober. The two packages share the resolved
  settings through an internal package, so no exported type or method
  carries them (API rule 4). `WithQuery`, which only shapes a probe,
  becomes `termcaptea.WithQuery` and returns a `termcap.Option` through
  that internal package.
* **`Caps.Profile` becomes a `termcap.ColorProfile`,** an enum with
  colorprofile's six values in colorprofile's order (`unknown`, `notty`,
  `ascii`, `ansi`, `ansi256`, `truecolor`), from `internal/enum` (API
  rule 5). `termcaptea` converts both ways, and a test pins the
  conversion against every `colorprofile` value. The JSON form writes the
  lowercase tokens and still reads colorprofile's spellings, and
  `ReportSchemaVersion` goes to 2, because the value of a field changes
  its spelling.
* **`KeyboardCaps.Enhancements` becomes `KeyboardCaps.Request`,** the
  Kitty keyboard flags to ask for (the progressive-enhancement bits, as
  `Accepted` already reports them). A Bubble Tea program gets its
  `tea.KeyboardEnhancements` from `termcaptea.KeyboardEnhancements`, and
  a line-mode CLI writes `CSI > flags u` itself.
* **What stays:** `Caps` and its views, `EnvCaps`, `FromEnv`, `Identity`,
  `Env`, `Fact`, `Findings`, `Report`, `ParseTmux`, `TmuxQuery`, every
  enum and reason token, `CapsMsg` and `ColorSchemeMsg`.
* The package documentation stops saying the environment "comes from
  `tea.EnvMsg`"; it says the program supplies it, as `Env(msg)` does in a
  TUI and `os.Environ()` does in a CLI.

### D5. `termsvc` (issue #3)

* **The commands move to `termsvctea`:** `ActivityBeacon`, `Copy`,
  `CopyContext`, `Open` (from `LinkPolicy.Open`), `Notify`,
  `NotifyContext`, and `Observe(n, msg)` in place of `Notifier.Update`.
  Functions over the core types, not wrapper types, so `Notifier` keeps
  one meaning.
* **The core `Notifier` gains its inputs and its synchronous send:**
  `SetCaps(termcap.Caps)` and `SetFocus(focused bool)` in place of the
  messages `Update` reads, and a send that returns the bytes to write and
  the `NotifyResultMsg`, calling the backend when one is set. `Sequence`
  stays as it is. The adapter's `Notify` is that send, then `tea.Raw` and
  the message.
* **Copy gains the same core form:** the bytes and the `CopiedMsg`,
  through the program's `Clipboard` when one is set. `CopySequence` and
  `CopyPlan` stay.
* **What stays:** every byte function, `LinkPolicy.Openable`, every enum,
  `Backend`, `Clipboard` and their `…Func` adapters, and the message
  structs.
* `termsvc`'s core is Charm-free once `termcap`'s is, so it ships in the
  same release or later.

### D6. `command` (issue #4)

* **`Result.Cmd` becomes a `command.Effect`:**

  ```go
  // Effect is a follow-up a host runs off its loop: the value it returns is
  // a message for the host. A Bubble Tea host runs it as a tea.Cmd. The
  // shell and agents ignore it, as they ignore Result.Cmd today.
  type Effect func() any
  ```

  The field keeps its name, `Cmd`, so a handler that sets none compiles
  unchanged. A TUI handler that sets one wraps its command:
  `commandtea.Effect(cmd)`, which keeps a nil command nil. A `tea.Cmd`
  is a `func() tea.Msg`, and `tea.Msg` is ultraviolet's named
  `Event` interface, so a `tea.Cmd` value does not convert to `func() any`
  directly; the wrapper is one closure. The built-in `app.quit` returns
  `func() any { return QuitRequestMsg{} }`.
* **`Registry.Dispatch` keeps its logic in the core,** policy, `Loop` and
  `Async` and the `ResultMsg` that follows, and returns the effects to run
  as `Effect` values. `commandtea.Dispatch(r, ctx, req) tea.Cmd` batches
  them. The core's `Dispatch` result is not a `tea.Cmd` any more, so a TUI
  calls the adapter.
* **The loop takes `func(any)`.** `Registry.Attach(send func(any))` and
  `WithLoop(send func(any))` stay in the core; Bubble Tea's `Program.Send`
  is a `func(tea.Msg)`, so `commandtea.Attach(r, p.Send)` and
  `commandtea.WithLoop(p.Send)` wrap it. `LoopMsg.Run` returns an
  `Effect`, and `commandtea.RunLoop(m)` the `tea.Cmd`.
* **`Watch` and `WatchContext` move to `commandtea`.** The core gains the
  blocking wait for the next change, with its conflicts, that they are
  built on.
* **What stays:** the registry, its snapshots, `Run`, `ParseArgs`,
  `ParseSlash`, `Complete`, `Params`, `WriteResult`, the gate and the
  policy, ACP and MCP, every message struct, and every option but the
  loop's signature.

### D7. `keymap` and a new `termkey` (issue #5)

* **A new core package, `termkey`, holds a terminal's keys:** the decoded
  key, its modifiers, the key codes and their canonical names, and a
  decoder from bytes. It imports the standard library and
  `internal/enum`, for its enums' tokens, and nothing else. `keymap`
  imports it.
  A line editor that needs keys without bindings imports `termkey` alone.
* **Its names avoid the glossary's.** The decoded key is
  `termkey.KeyEvent` (`Key` is `when.Key`), with `Code`, `Mod`, `Text`,
  `ShiftedCode`, `BaseCode`, and whether it is a release or a repeat; its
  modifiers are `termkey.Mod`; the decoder is `termkey.Decoder`, and what
  it yields is `termkey.Input`, a key, a paste, or a sequence it does not
  decode, with its bytes. The PLAN checks every name against
  `docs/glossary.md` before it is built.
* **The codes are ultraviolet's values.** `termkey`'s key codes and
  modifier bits take the numeric values `uv.KeyUp`, `uv.ModCtrl` and the
  rest have, so `keymaptea`'s conversion from `tea.Key` is field for field.
  A test in `keymaptea` walks every code and modifier Bubble Tea exports
  and fails on one that converts to another value, or that `termkey`
  names differently from `tea.Key.Keystroke()`.
* **The decoder** reads, as a state machine fed bytes, with no goroutine
  and no clock:
  * UTF-8 text, and C0 controls as ctrl and their letter, as `Normalize`
    reads them today;
  * `ESC` before a key as alt and that key;
  * the legacy CSI and SS3 forms of the arrows, Home, End, Insert,
    Delete, Page Up and Down and the function keys, with the xterm
    modifier parameter (`CSI 1 ; m X`, `CSI n ; m ~`);
  * xterm's modifyOtherKeys form, `CSI 27 ; m ; code ~`, and its
    formatOtherKeys variant `CSI code ; m u`, which reads as the Kitty
    form below;
  * the Kitty keyboard protocol's
    `CSI code : alternates ; mods : event ; text u`, with its modifier
    bits (shift 1, alt 2, ctrl 4, super 8, hyper 16, meta 32), and the
    flags reply `CSI ? flags u`;
  * bracketed paste, `CSI 200 ~` to `CSI 201 ~` (DECSET 2004), as one
    paste input, bounded in bytes;
  * any other CSI, OSC, DCS or APC as one input with its raw bytes,
    bounded, so a terminal's reply never reads as keys.

  A lone `ESC` is ambiguous until more bytes come or none do. The decoder
  holds it, says that it holds one, and gives it up as the escape key when
  the caller says the input went idle, the caller choosing how long. The
  practice varies: tmux's `escape-time` defaults to 10 ms, ultraviolet's
  `DefaultEscTimeout` and Neovim's `ttimeoutlen` to 50 ms, and Vim's
  `defaults.vim` sets 100 ms. `termkey` exports ultraviolet's 50 ms as its
  suggested default, so the two modes wait alike. Windows'
  win32-input-mode (`CSI ? 9001 h`, keys as `CSI Vk ; Sc ; Uc ; Kd ; Cs ;
  Rc _`) is out of scope: the decoder passes such a sequence through as
  raw bytes, and a later record may decode it.
* **The decoder's behaviour is pinned against Bubble Tea's.** A
  differential test under `internal/termevent`, the one place ultraviolet
  may be imported, feeds both decoders the same corpus and compares the
  keys. A difference is a test failure, unless the test's table records
  it with a reason. A fuzz target holds that the decoder never panics,
  consumes every byte, and decodes a split stream as the whole.
* **`keymap`'s core uses `termkey`:** `Stroke.Mod` is a `termkey.Mod`;
  `Normalize`, `Stroke.Matches`, `Explain` and `Explanation` take and
  hold `termkey.KeyEvent`s; `Outcome.Replay` is a `[]termkey.KeyEvent`.
  The notation and labels read `termkey`'s names, which are Bubble Tea's
  `Keystroke()` spellings by the test above, so every stored binding and
  every user's `keybindings.json` reads as before.
* **The dispatcher's core is a function of keys and time.** The core
  `Dispatcher` takes a key, the path and the context, and returns the
  `Outcome` and the deadline of a pending sequence, if one is pending; a
  `TimeoutMsg`-equivalent call ends it. `keymaptea.Update(d, msg, p, ctx)`
  is today's `Update`: it converts the key, calls the core, and starts the
  `tea.Tick`. `Settings.Tick` moves to a `keymaptea` option.
* **Bubbles moves to `keymaptea`:** `Help`, `KeyBinding` and
  `BubblesKeys`, as functions over the core `Keymap` and `Stroke`.
* **What stays:** `LoadVSCode`, `ExportVSCode`, `Schema`, `ParseStroke`,
  `ParseSequence`, `Stroke.Label`, `Sequence`, `Binding`, `Default`,
  `Keymap` and its queries, `FactsFrom`, `Features`, `KeyFacts`, the
  conflicts and the which-key data.

### D8. `glyph`: status and meter glyphs, and `Control` (issue #11)

`glyph.Set` gains eight fields, every one a single rune one cell wide in
both sets, under both width methods. The East_Asian_Width column is from
Unicode 18.0's `EastAsianWidth.txt`. None of the first seven is an emoji
in `emoji-data.txt`; `␛`, which 0009-MADR chose, the PLAN checks there
before it is built.

| Field | Means | Unicode | EAW | ASCII |
| :--- | :--- | :--- | :--- | :--- |
| `Done` | a step or item that succeeded | `✓` U+2713 CHECK MARK | N | `+` |
| `Failed` | a step or item that failed | `✗` U+2717 BALLOT X | N | `x` |
| `Running` | a step in progress | `●` U+25CF BLACK CIRCLE | A | `*` |
| `Pending` | a step not yet started | `○` U+25CB WHITE CIRCLE | A | `-` |
| `Heading` | the first line of an item, such as a tool call's | `▸` U+25B8 | N | `>` |
| `MeterFill` | a meter's filled cell | `━` U+2501 | A | `#` |
| `MeterTrack` | a meter's empty cell | `╌` U+254C | N | `-` |
| `Control` | where `safetext` removed a sequence (0009-MADR §1) | `␛` U+241B | N | `^` |

* **gobble's glyphs are kept where it has them** (`✓ +`, `✗ x`, `▸ >`,
  `━ #`, `╌ -`), so gobble changes no output when it adopts the set.
* **The light check and ballot X, not the heavy ones.** `✔` U+2714 is an
  emoji (`Emoji=Yes`) and is drawn two cells wide when a variation
  selector follows it; `✓` and `✗` are not emoji. npm's `figures` uses the
  heavy forms; GitHub's `gh` uses `✓`, as gobble does.
* **Ambiguous width is not a new risk.** `●`, `○` and `━` are
  East_Asian_Width A, as `…`, `•` and every box-drawing glyph of the set
  already are. UAX #11 says to treat an ambiguous character as narrow
  where the context is not known to be East Asian, and both width methods
  of `x/ansi` measure them as one cell, which `TestOneCellInBothMethods`
  holds. A terminal set to draw ambiguous characters wide misdraws the
  existing set already; the legacy tier (`TierLegacy`) is where a later
  record would answer that.
* **`Heading` is its own field, though it draws as `Focus` does.** One
  field means one thing: a theme that changes the focus mark must not
  change every tool call's header. `Mask` and `Bullet` share a glyph the
  same way today.
* **The status glyphs never carry meaning by colour alone** (TUI rule 4):
  each has a distinct shape in both sets. `Running` and `Pending` are the
  two #11 left open ("and perhaps running and pending"); they complete
  the set a step list needs.
* **`Ellipsis` keeps `~` in the ASCII set.** It is the tail a cut adds
  inside a cell budget (`workspace/render.go:299`, `:337`). Three dots
  would take two cells more than `…` and break the rule every test of
  the package holds, that a view laid out for one set fits the other. A
  three-dot tail has precedent: GitHub's `go-gh` truncates with `...`,
  and only where the width leaves two cells besides it, cutting with no
  tail otherwise. That is a width-dependent tail, which is what the
  one-cell rule exists to avoid. A tilde is a recognised truncation mark
  as well: MS-DOS and Windows short file names cut long names to
  `PROGRA~1`. Text that continues in prose, such as "Thinking...", is the
  program's text, not a glyph.
* The tests' count of 70 goes to 78, and a test like `TestPromptAndMask`
  holds that each new field's two forms differ, and that the four status
  glyphs differ from each other within each set.

### D9. `safetext`: shipped by this record, as the one sanitizer (issue #12)

0009-MADR §2, with A1's presets and fields and A4's names, stays the
design. This record ships it ahead of the rest of 0009, decides what A4
left open, and settles the four points where 0009 and `internal/sanitize`
disagree.

* **What ships here:** `Rules` with the fields of §2 and A1 (`Tabs`,
  `Newlines`, `Flatten`, `Bidi`, `Visible`, `Glyphs`, `MaxString`,
  `Escape`, `KeepSGR`, `LinkSchemes`); the presets `Text()`, `Strict()`,
  `Line()`, `Command()` and `StatusLine()`; `Clean`; the streaming
  `Sanitizer`; `SafeURL` and `URLs`; and `glyph.Set.Control`. **What
  stays with 0009-PLAN:** `Interpret`, its Step A1.1, which needs `x/ansi`
  and which nothing here needs. So `safetext` imports the standard
  library, `glyph` and `internal/enum` only.
* **One added function:** `IsClean(s string, r Rules) bool`, true when
  `Clean(s, r)` would return `s`, without allocating and stopping at the
  first change. Validators refuse text rather than clean it
  (`termsvc/links.go`, `command/registry.go`); today they clean it in
  full to find out (`HasControl` is `Line(s) != s`), which gobble
  measured at 42.6 µs for 4 KiB of tool output.
* **S1. What is removed is at least what is removed today.** Under every
  preset, `safetext` removes, or under `Escape` shows escaped, every
  character `internal/sanitize` drops: C0 but the `\t` and `\n` a rule
  keeps, DEL, C1, U+200B, U+FEFF, U+2060–U+2064, U+00AD, the tags
  U+E0000–U+E007F, and the Bidi_Control set. `Rules.Bidi` keeps the
  whole Bidi_Control set (U+061C, U+200E, U+200F, U+202A–U+202E,
  U+2066–U+2069, the twelve code points of Unicode's `PropList.txt`), not
  the nine §2 listed, so that prose in Arabic and Hebrew keeps its marks
  as well as its embeddings. These are the characters of Trojan Source
  (CVE-2021-42574). Unicode's UTS #55 advises showing them rather than
  forbidding them, which `Escape` does (S4). ZWJ, ZWNJ and the variation
  selectors stay, as today.
* **S2. Invalid UTF-8 becomes U+FFFD, everywhere,** as §2 says, **one
  U+FFFD for each maximal subpart** (owner, Q8). This is the practice of
  Unicode's core specification, §3.9.6, "U+FFFD Substitution of Maximal
  Subparts", which the WHATWG Encoding standard follows. A maximal subpart
  is the longest prefix of a well-formed sequence (Table 3-7) that the
  input begins with, or one byte when no well-formed sequence begins
  there: `E2 82` cut short is one U+FFFD, and `C0 AF` is two.
  * Neither standard-library helper does this. `utf8.DecodeRune` returns
    one error per byte, so `E2 82` gives two, and `strings.ToValidUTF8`
    replaces a whole run with one, so `C0 AF` gives one. `safetext`
    therefore decodes with Table 3-7's byte ranges itself, in its single
    pass.
  * The tests hold it to Table 3-8's worked example and to a reference
    implementation of Table 3-7 written plainly in the test, under the
    fuzz target, which also holds that `Clean` is idempotent on the
    result.
  * Dropping invalid bytes, as `internal/sanitize` does, can join the
    bytes either side into a sequence that was not there.
  * A raw byte 0x80–0x9F that continues no sequence is invalid UTF-8, so
    it becomes U+FFFD and opens no sequence; the C1 code points, properly
    encoded, are parsed as their 7-bit forms, as §2 says. The terminal
    never receives a raw C1 byte either way. That is the trap xterm's
    documentation names: C1 controls are valid UTF-8 continuation bytes,
    so a C1 control can be mistaken for badly formed UTF-8 (`ctlseqs`,
    "C1 (8-Bit) Control Characters").
  * The library's visible output changes where it met invalid UTF-8: a
    dropped byte now shows as `�`.
* **S3. `Flatten` turns each run of `\r`, `\n`, `\t`, U+2028 and U+2029
  into one space,** as `internal/sanitize` does, not each one. With
  `Newlines`, U+2028 and U+2029 become `\n`.
* **S4. `Escape` shows what the rules would remove,** as `\xNN`,
  `\uNNNN` or `\U00NNNNNN`, and an escape sequence as its bytes so
  written; what `Tabs` and `Newlines` keep stays raw; `Flatten` still
  wins. So the tool-output combination gobble asks for is
  `Rules{Tabs: true, Newlines: true, Escape: true}`: lines and tabs kept,
  every other control and the bidi controls shown. `Command()` is
  `Escape` with neither, everything on one line, as A1 says.
* **S5. `internal/sanitize` wraps `safetext`.** Its `Line`, `Styled` and
  `HasControl` become calls of `safetext` with rules equal to their
  behaviour (`Line()`; `KeepSGR` and OSC 8 for `Styled`, with
  `LinkSchemes` left empty meaning any printable-ASCII target, as
  `Styled` keeps today; `IsClean` for `HasControl`). `Truncate` and
  `Token` stay in `internal/sanitize`: they cut and filter by width and
  alphabet, which is not sanitising, and `Truncate` needs `x/ansi`. The
  28 calls keep their call sites. A later record may move them to
  `safetext` directly; this one does not, to keep the change small.
  *A3.2 corrects "equal to their behaviour": equal on well-formed input;
  on malformed input 0009-MADR §2's grammar applies, in five classes A3.1
  names. A3.3 adds that `KeepSGR` keeps nothing written with a C1.*
* **The performance bar** (gobble's proposed criteria, adopted):
  * clean input: `Clean` returns `s` itself, with no allocation;
  * input that needs cleaning: one pass and one allocation;
  * `Sanitizer.Write` allocates nothing in the steady state;
  * benchmarks in the package for clean ASCII, clean CJK and emoji, tool
    output, styled text and hostile text, 4 KiB each, as gobble's table
    has them, with that table recorded as the outside baseline;
  * no benchmark of `internal/sanitize`'s three wrapped functions slower
    than at `v0.12.0` on the same machine, measured before and after.
* **The checks.** 0009-PLAN Step 2's tests, `FuzzClean` and the
  differential against `ansi.Strip`, all of them; and **a differential
  fuzz against `internal/sanitize` as it is at `v0.12.0`,** kept as a
  test-only copy, since once `internal/sanitize` wraps `safetext` it
  cannot be its own oracle. It compares every input on which the two are
  meant to agree, and its table names each class S1 to S4 change.

### D10. Migration: one breaking minor release for every split, by exception to rule 1

*A4, 2026-10-24: under `v1`, a breaking minor release goes against Go's
import compatibility rule; A4.1, open, decides release 2's form.*

* **This record grants the exception API rule 1 allows a record to
  grant,** for the declarations D4 to D7 move or retype. A wrapper cannot
  be left in the core (an import cycle), a retyped field has no
  compatible form, and no known module calls them.
* **Every split lands in one minor release** (owner, Q1: "One big-bang
  release"): `termcap`, `termsvc`, `command`, `termkey` and `keymap`
  together. A TUI caller migrates once, and no package is half moved
  across two releases. The additive work of D8, D9 and D11 is not a split,
  and ships before it (D12).
* **The release notes carry a migration table:** each moved or retyped
  name, its new home or type, and the one-line change a caller makes. The
  PLAN lists every incompatible change in a `scripts/apicheck.allow` block
  naming the PLAN and step, and the release's close-out removes it
  (rule 2).
* **What is added is added compatibly:** `safetext`, `termkey`, the glyph
  fields, `IsClean`, the core forms of D5 to D7. They need no exception.
* **The glossary** records `termcaptea.Query`'s move, the new names
  (`ColorProfile`, `Effect`, `KeyEvent`, `Mod`, `Decoder`, `Input`,
  `Rules`, `Sanitizer`), and the moved types' new packages, in the change
  that builds each.

### D11. The closure gate

* **`internal/conformance` gains `TestCoresAreCharmFree`:** for each core
  of D1, it runs `go list -deps` with `GOWORK=off`, for `GOOS=linux`,
  `darwin` and `windows` (build constraints are read for one target at a
  time), and fails on any package under `charm.land/` or
  `github.com/charmbracelet/` that is not `x/ansi` or beneath it, naming
  the core, the package and the import chain. Its list of cores fails as
  stale when a listed package does not exist, so a rename cannot drop a
  core silently. `internal/conformance` runs `go list` already
  (`goList`, `conformance_test.go:382`), so the test adds no tool and no
  module.
* **Why a closure test and not only a lint rule.** depguard and
  gomodguard read direct imports only. Go's own `go/build/deps_test.go`
  gets the transitive property another way, by checking each package's
  direct imports against a policy graph closed under transitivity, and
  Kubernetes' `import-boss` has a `Transitive` flag. Listing the closure
  is the simplest form that fits a list of ten packages, and it reports
  the chain that broke the rule.
* **depguard gains a rule, `charmfree`,** that refuses Bubble Tea,
  Bubbles, Lip Gloss, `colorprofile`, `ultraviolet` and `x/term` in the
  cores' own non-test files, so a direct import fails at lint, in the
  editor, before the test runs.
* **Each is seen failing first,** on a scratch copy that adds a Bubble Tea
  import to a core, and one that adds it to a package a core imports, as
  the global rules require.
* **A core joins the gate in the commit that makes it clean.** `glyph`,
  `when`, `layout`, `fuzzy` and `internal/sanitize` are clean today, and
  join with the gate itself.

### D12. Order and releases

*Revised 2026-10-23 for the owner's answer to Q1, which put every split
in one release; the draft had a release for `termcap`, `termsvc` and
`command` and another for `termkey` and `keymap`.*

*A4, 2026-10-24: release 1 is `v1.13.0`, tagged in place of `v0.13.0`;
release 2's version and form wait for A4.1.*

1. **Release 1, additive:** the gate and the clean packages, then `glyph`
   (D8) and `safetext` (D9). It also carries `fuzzy`, and so answers #6,
   #11 and #12.
2. **Release 2, breaking, every split:** `termcap`, then `termsvc`, then
   `command` (D4 to D6), then `termkey` and `keymap` (D7). It answers #2,
   #3, #4 and #5.

* **Each package is a PLAN phase,** in that order, with the tree green and
  the pre-add gate passed at its end, though release 2 is tagged only
  after the last. A split phase leaves `main` with a moved API and no tag
  for it, which is the owner's to release or hold.
* **`palette` (0008-PLAN Steps 5 to 9) waits for this record** (owner,
  Q7), so it is written once, against `commandtea` and `keymaptea`.
* **0009-PLAN** gets a dated note, when this record is accepted, that its
  Step 2 is carried out by this record's PLAN, with the names of A4, and
  that A1.1 adds `Interpret` and nothing else.

### Consequences

* Good, because gobble's CLI can use `termcap`'s facts and report,
  `termsvc`'s sequences, `command`'s registry, `keymap`'s files and
  notation, a key decoder, the glyph set and the sanitizer, and builds a
  copy of none of them.
* Good, because a TUI keeps every capability, one import away, with the
  same message types it switches on today.
* Good, because the closure gate makes the boundary a property of the
  build, which a later change cannot lose without a failing test.
* Good, because the library gets one sanitizer that removes no less than
  today's, maps invalid UTF-8 to U+FFFD, and stops cleaning text in full
  just to find out whether it is clean.
* Good, because a line editor, gobble's or the library's own later, gets
  a decoder pinned to Bubble Tea's behaviour, so the two modes read keys
  alike.
* Bad, because, with every split in one release (Q1), gobble gets
  `termcap`, `termsvc` and `command` only when `keymap` and `termkey`, the
  largest part, are done as well; and release 2 is one large review.
* Good, because a TUI caller migrates once, in one release, with one
  migration table.
* Bad, because every TUI caller of a moved name changes an import, and
  every TUI handler that returns a follow-up wraps it in
  `commandtea.Effect`: nine sites in `workspace` today.
* Bad, because two decoders exist, ultraviolet's for the TUI and
  `termkey`'s for line mode, and they can drift; the differential test
  catches the drift but someone must act on it when ultraviolet moves.
* Bad, because four adapter packages add to the API surface and to the
  documentation.
* Bad, because `Result.Cmd`'s static type weakens from `tea.Cmd` to
  `func() any`: a TUI can no longer tell at compile time that an effect
  returns a Bubble Tea message.
* Bad, because the doctor report's JSON `profile` changes spelling and
  its `schema_version` goes to 2.
* Bad, because the library's output changes wherever it met invalid UTF-8,
  from nothing to `�`; golden files that hold such input change.
* Neutral, because `launch`, `theme`, `workspace` and `tuitest` stay TUI
  packages; `launch` links `x/term`, which gobble refuses, and nothing
  here changes that.
* Neutral, because gobble's module graph and `go.sum` still list the
  Charm modules the root module requires, though its CLI compiles none of
  them; only a module split would remove them, and gobble's boundary does
  not ask for it.

### Confirmation

* `TestCoresAreCharmFree` and the `charmfree` depguard rule pass at the
  end of each phase, after each was seen failing on a scratch copy.
* `make pre-add-check`, `make lint` (three GOOS), `make test`, the race
  run, `LC_ALL=C` and the Windows host pass at each phase, as AGENTS.md
  and the PLAN require.
* `make apicheck` reports exactly the changes the PLAN's
  `scripts/apicheck.allow` block lists, and no other.
* The `colorprofile` round trip, the `termkey` code and name table
  against `tea.Key.Keystroke()`, and the decoder differential against
  ultraviolet pass.
* `safetext`'s fuzz targets join `make fuzz` and CI's fuzz step; the
  differential against the `v0.12.0` copy of `internal/sanitize` runs in
  `go test`; the benchmarks meet D9's bar, with their output in the PLAN's
  execution record.
* Every golden file that changes is read before it is committed, and the
  PLAN's execution record lists each and why.
* gobble confirms adoption in its own repository (its 0032-MADR D7); this
  repository does not.

## Pros and Cons of the Options

### A. Charm-free cores at today's paths, adapters in sub-packages, one breaking release per split, a closure gate

* Good, because the import paths gobble's CLI wants are the ones the
  issues name, unchanged.
* Good, because each declaration has one home and one name; the glossary
  gains no shared name.
* Good, because the split rule (D2) is mechanical, so review checks it,
  not taste.
* Good, because gobble gets each split in one release.
* Neutral, because the exception to rule 1 is explicit and bounded to
  the declarations D4 to D7 name.
* Bad, because TUI code changes imports for the moved names.

### B. New Charm-free packages beside today's, with aliases in the old paths

* Good, because a TUI importing `termcap` today compiles unchanged for
  types.
* Bad, because a method that names Bubble Tea, such as
  `Registry.Dispatch` or `Notifier.Update`, cannot be added to an aliased
  type from another package; those break anyway.
* Bad, because every type has two names in two packages, which
  `TestNoTypeNameMeansTwoThings` refuses unless the glossary lists dozens
  of deliberate clashes, against rule 3.
* Bad, because gobble's CLI imports new paths, and the docs explain two
  packages per capability.

### C. The same split over two releases, with deprecation

* Good, because it follows rule 1's letter.
* Bad, because the deprecated forms can only live in the adapter's
  direction, so the old names at the old paths keep Bubble Tea in the
  closure for that release, and gobble waits a release more for each
  split.
* Bad, because retyped fields need a second field each for one release
  (`Caps.Profile` beside `Caps.ColorProfile`, `Stroke.Mod` beside a new
  one), and `Stroke` is a value type compared with `==`, so two modifier
  fields change what equal means.
* Bad, because transitional aliases in the adapters count as second
  declarations for `TestNoTypeNameMeansTwoThings`, and the glossary must
  carry them for a release.
* Bad, because no known module calls the names it protects.
* Neutral, because the usual tools for a gradual move work only in the
  other direction. The type-alias design (Cox and Griesemer, 2016) and
  `go fix`'s `//go:fix inline` (Go 1.26) let an old package forward to a
  new one, which means the old package imports the new. Here the old
  package is the core, and forwarding from it to the adapter is the cycle
  above.

### D. Build tags

* Good, because nothing moves.
* Bad, because a build tag applies to the whole build: a program that is
  both a CLI and a TUI, as gobble is, cannot build one package both ways.
* Bad, because every gate, lint, vet, test and apicheck, would run twice,
  and the tagged-out API would be invisible to `go doc`.

### E. Erase Bubble Tea types to `any` in place

* Good, because no import path changes.
* Bad, because the `Prober` and the `Notifier` must switch on Bubble Tea's
  message types, so they import Bubble Tea anyway.
* Bad, because every signature loses its type: `Update(msg any) any`
  compiles for anything and checks nothing.

### F. Change nothing

* Good, because it costs this repository nothing now.
* Bad, because gobble's 0032-MADR (D3, D5) and this repository's purpose,
  one canonical implementation, both say not to; gobble's CLI would keep
  its own copies for ever.

### The sub-decisions

* **The adapter's name.** `<pkg>tea` was chosen over a package named
  `tea`, which shadows Bubble Tea's conventional import name in every file
  that needs both, and over one adapter package for all four, which would
  mix four unrelated APIs and force a TUI to import all of them.
* **`Result.Cmd`.** `Effect func() any` was chosen over `any`, which keeps
  every handler compiling but checks nothing, so a handler that set a
  `func() tea.Msg` instead of a `tea.Cmd` would fail only at run time;
  and over a type parameter on `Registry`, which would spread to every
  type that holds a registry.
* **The decoder.** Writing `termkey` and pinning it to ultraviolet by a
  differential test was chosen over three others:
  * importing `uv.EventDecoder`, which links `colorprofile` and
    `x/windows` and an untagged module;
  * importing `x/input`, which requires `x/windows`, is labelled
    experimental, and is no longer what Bubble Tea runs, so it would not
    keep the two modes alike;
  * porting ultraviolet's `decoder.go` (MIT, so allowed with its notice).
    The port would agree by construction at first, but it would bring the
    Windows console-record path that needs `x/windows`, and 2,700 lines
    written for a full-screen reader. It would drift from ultraviolet's
    pseudo-version all the same.
* **The sanitizer.** Wrapping (S5) was chosen over replacing the 28 calls
  at once, which would put a behaviour change and a call-site change in
  one diff, and over keeping two sanitizers, which A4 rules out.
* **`ColorProfile`'s tokens.** Lowercase (rule 5) with the old spellings
  still read was chosen over keeping colorprofile's mixed case, which
  would need a second exception, to rule 5, for no gain.

## Owner questions

Each was put to the owner on 2026-10-23 with three options, the
recommendation first. The owner answered all eight that day. Six answers
are the recommendation. Two are not, Q1 and Q8, and D9, D10, D12 and the
Consequences above follow those two answers.

* **Q1. The exception to rule 1 (D10).** Options: the exception with one
  breaking minor release per split (recommended); option C, two releases
  with deprecation; the exception with every split in one breaking
  release. **Answer: "One big-bang release".** The exception is granted,
  and `termcap`, `termsvc`, `command`, `termkey` and `keymap` land
  together in one breaking release (D10, D12).
* **Q2. The adapter names (D3).** Options: `<pkg>tea` (recommended);
  `<pkg>/bubbletea`; one `tuitea` package for all four. **Answer:
  `<pkg>tea`.**
* **Q3. `Result.Cmd` (D6).** Options: `Effect func() any` (recommended);
  `Cmd any`; no `Cmd` on `Result`, with effects registered through the
  adapter. **Answer: `Effect func() any`.**
* **Q4. The key type and decoder (D7).** Options: a new `termkey` package
  (recommended); both inside `keymap`; `termkey` with the key type only
  and the decoder deferred. **Answer: a new `termkey` package.**
* **Q5. The status glyphs (D8).** Options: all four, with `● *` and
  `○ -` (recommended); all four, from East_Asian_Width N code points only;
  `Done` and `Failed` only. **Answer: all four, `● *` and `○ -`.**
* **Q6. `Ellipsis` (D8).** Options: keep `~` and give #11 the reason,
  with no new field (recommended); keep `~` and add a wide field exempt
  from the one-cell rule; change the ASCII form to `...`. **Answer: keep
  `~`, and explain.**
* **Q7. The order with `palette` (D12).** Options: this record's releases
  before 0008-PLAN Step 5 (recommended); `palette` first, on today's APIs;
  the two interleaved. **Answer: this record first.**
* **Q8. Invalid UTF-8 (D9 S2).** Options: one U+FFFD per run of invalid
  bytes, as `strings.ToValidUTF8` (recommended); one U+FFFD per maximal
  subpart, Unicode §3.9.6 and WHATWG practice; keep dropping it. **Answer:
  "U+FFFD per maximal subpart".** D9 S2 says how.

## Amendments

### A1 (2026-10-23): what the PLAN's drafting found

*Status: A1.1 answered, (c) chosen, and A2 builds it; A1.2 and A1.3
recorded, A1.3 revised by A2.* The owner accepted this
record on 2026-10-23 ("accept the madr"). Drafting
[0020-PLAN-charm-free-cores-glyphs-and-safetext.md](0020-PLAN-charm-free-cores-glyphs-and-safetext.md)
against the tree at `63c68d6` found one point where D4, as written, cannot
be built, and fixed the core forms D2 asks for. Nothing else changes.

**A1.1. `termcap`'s options split in two (changes D4).**

* **Found.** `termcap.Option` is `interface{ apply(*Prober) }`
  (`termcap/prober.go:51`): every option sets a field of the `Prober`
  itself. `EnvCaps` (`termcap/envcaps.go:16-22`) builds a disabled
  `Prober`, calls its unexported `start`, and returns its `Caps`. Once the
  `Prober` is in `termcaptea`, `termcaptea` cannot call an unexported
  method of `termcap.Option`, and a method of `termcap` cannot take a
  `termcaptea` type without an import cycle. D4's "the two packages share
  the resolved settings through an internal package" has no form that
  keeps the option opaque under API rule 4: an internal package cannot
  name `termcap.Option`, and an unexported method can be called only by
  its own package.
* **The options are already two sets.** `EnvCaps`'s documentation says so:
  `WithGOOS`, `WithAppearanceEnv` and `WithOverride` shape the
  environment's facts; `WithTimeout`, `WithQuery`, `WithoutHeuristic`,
  `WithoutColorSchemeUpdates` and `WithoutBackgroundRequest` "change
  nothing" there, and the hooks `WithAppearanceHook` and `WithConsoleHost`
  "are not run". `WithDisabled` exists only for a prober.
* **Options considered:**
  * **(a) Split the options** *(recommended)*. `termcap.Option` keeps the
    three environment options, opaque over an unexported settings struct.
    The eight probe options move to `termcaptea` as `termcaptea.Option`,
    opaque the same way, with `termcaptea.WithEnv(o ...termcap.Option)`
    to pass the environment's options to a prober. `EnvCaps` no longer
    builds a prober; the prober calls `EnvCaps` for the environment's
    facts on its first `tea.EnvMsg`. Two exported functions join the
    core, so the prober needs nothing unexported:
    `Overridden(c Caps, o ...Option) Caps`, the overrides of `o` applied
    to `c` in order, which the prober runs on each `Caps` it hands out,
    as `WithOverride` says; and `BrandFromVersion(v string) Brand`,
    today's unexported `brandFromVersion` (`termcap/identity.go:278`),
    which reads an XTVERSION reply.
  * **(b) Keep one `termcap.Option`, with an exported method whose
    parameter is an internal type.** No program outside the module can
    implement it, since it cannot name the type, so it is opaque in
    effect. But it breaks the letter of API rule 4 (an unexported
    method), and `go doc` shows an internal type in the option's method
    set.
  * **(c) Move the probe engine into the core,** as an exported
    message-free state machine (events in, bytes out), with
    `termcaptea.Prober` a thin wrapper; the options stay `termcap.Option`.
    A CLI could then probe as well. It adds an engine's worth of exported
    API that no issue asks for, and the largest change of the three.
* **Recommended: (a).** It follows the documented split, keeps rule 4's
  letter, and adds two small functions a CLI can use as well. Under (a)
  the planned names are unchanged but for `termcaptea.Option`, which is
  `Option`, shared by every package (glossary).
* **Answer (owner, 2026-10-23): "option c, with additional research as to
  best practice and standards adherence, idiomatic implementation,
  forward-thinking and future-proof API".** The owner chose (c) after a
  comparison of (a) and (c): (c) lets a line-mode CLI query the terminal,
  keeps every option in `termcap`, and moves only the `Prober` type, at
  the cost of reply parsing in the core and a larger API. A2 is the
  design. (a)'s `Overridden`, `BrandFromVersion` and `termcaptea.Option`
  are not built.

**A1.2. The core forms, as built (fills in D4 to D7).** Each follows D2:
a core keeps its names where it can, and returns a value the adapter
turns into a command.

* **`command`.** `Registry.Dispatch(ctx, req) Effect` and
  `Registry.Watch(ctx) Effect` keep their names and return an `Effect`;
  `WatchContext` goes, since `Watch` takes the context. An `Effect`'s
  value is a message for the host, `nil` for none, or an `[]Effect` the
  host runs as one batch, as `tea.Batch` does, which is how an `Async`
  command's follow-ups and a change's `ConflictMsg` arrive.
  `LoopMsg.Run() Effect`. `commandtea.Cmd(e Effect) tea.Cmd` converts any
  `Effect`, and `commandtea.Dispatch`, `Watch`, `WatchContext` and
  `RunLoop` are it over the core.
* **`termsvc`.** No new type.
  `(*Notifier).Prepare(ctx, x) (seq string, deliver func() NotifyResultMsg)`
  does, on the caller's goroutine, what `NotifyContext` does before its
  command: it cleans, applies the policy, numbers and encodes. `deliver`
  calls the backend, bounded by ctx and `WithBackendTimeout`, or returns
  the result at once. `PrepareCopy(ctx, c, text, o ...CopyOption) (Route,
  func() CopiedMsg)` is the same for a copy. The adapter writes `seq` with
  `tea.Raw`, and an OSC 52 copy with `tea.SetClipboard` as today, and
  runs `deliver` in the command, so a backend still runs off the loop.
  `SetCaps(termcap.Caps)` and `SetFocus(focused bool)` replace the
  messages `Update` reads.
* **`keymap`.** `(*Dispatcher).Key(k termkey.KeyEvent, p Path, ctx
  when.Context) Outcome` and `(*Dispatcher).Expire(m TimeoutMsg) Outcome`.
  `Outcome` gains `Wait time.Duration` and `Timeout TimeoutMsg`: when
  `Wait` is positive a sequence is pending, and the host hands `Timeout`
  back to `Expire` after `Wait`. `(*Keymap).Shown(id, p) []Binding` is the
  bindings help shows, which `keymaptea.Help` and `KeyBinding` need.
  `keymaptea.Update(d, msg, p, ctx) (keymap.Outcome, tea.Cmd)` starts
  `tea.Tick(o.Wait, …)` for a pending sequence. `Settings.Tick`, "a clock
  for tests" (0007-MADR A4 item 5), goes without a replacement: the one
  test that sets it (`workspace/bindings_test.go:174`) sets
  `Settings.Timeout` to a nanosecond instead, and `tea.Tick` then fires at
  once. `keymaptea.FromTea(tea.Key) termkey.KeyEvent` and
  `ToTea(termkey.KeyEvent) tea.Key` convert field for field, so a replayed
  key reaches a pane as the same `tea.KeyPressMsg`.
* **`termkey`.** `KeyEvent{Code, Mod, Text, ShiftedCode, BaseCode,
  IsRepeat, Release}`; `Mod` a `uint16` bit set with ultraviolet's bit
  values (shift 1, alt 2, ctrl 4, meta 8, hyper 16, super 32, caps lock
  64, num lock 128, scroll lock 256); `Decoder` with `Decode`, `Pending`
  and `Flush`; `Input` with an `InputKind` of `InputKey`, `InputPaste` or
  `InputUnknown`; `DefaultEscTimeout`, 50 ms.

**A1.3. The names check.** `python3 -m tools.verify names` over the
planned names (`safetext`: `Rules`, `Sanitizer`; `termkey`: `KeyEvent`,
`Mod`, `Decoder`, `Input`, `InputKind`; `termcap`: `ColorProfile`;
`command`: `Effect`; `termcaptea`: `Prober`, `Query`, `Reply`, `Option`;
and, as first drafted, `keymaptea`: `Option`, which A1.2 then dropped)
reported three clashes in fourteen names,
`termcaptea`'s `Prober`, `Query` and `Reply` with `termcap`'s. They are
the moves of D4, which leave `termcap` in the same change, so
`TestNoTypeNameMeansTwoThings` sees one of each; the PLAN checks it after
the move.

### A2 (2026-10-23): `termcap`'s probe as an I/O-free core engine (builds A1.1 (c); replaces D4's prober move)

*Status: accepted (2026-10-23).* The owner chose (c) and asked for its
design, then accepted it and approved the PLAN's execution: "i accept the
design, i approve executing the plan, proceed".

**What changes in the decision.** D4 moved the whole `Prober` to
`termcaptea`. Under A2 the probe's logic stays in `termcap`, as an
exported engine that does no I/O, reads no clock and starts no goroutine.
`termcaptea.Prober` is a thin Bubble Tea driver of it. A line-mode CLI
drives the same engine from its own raw-mode input. Every option stays a
`termcap.Option`, as today. D4's `ColorProfile`, `KeyboardCaps.Request`
and `ReportSchemaVersion` 2 stand.

#### A2.1 The evidence

Read 2026-10-23. The sources are listed under More Information.

* **The sans-IO discipline.** Cory Benfield's "Building Protocol Libraries
  The Right Way" (PyCon 2016) and the sans-io pages define a protocol
  implementation with no I/O: it is "defined entirely in terms of
  synchronous functions returning synchronous results" and must not
  "block or wait for any form of I/O". Bytes go in through one input, and
  bytes and events come out, returned or pulled. The stated gains: tests
  that are never flaky, fast and safe in parallel, and reuse "across all
  forms of flow control".
* **Time comes in, a deadline goes out.** The sans-io pages say nothing
  about clocks. quinn-proto, QUIC's I/O-free core in Rust, is "fully
  deterministic … does not get any relevant timestamps from the operating
  system": `handle_timeout(now)`, `poll_timeout() -> Option<Instant>`,
  `poll() -> Option<Event>`, with "input events [that] represent
  monotonically increasing time". In Go, `golang.org/x/time/rate` takes
  `t time.Time` (`AllowN(t, n)`, `ReserveN(t, n)`), and
  `golang.org/x/net/quic` passes `now time.Time` to every internal
  method, leaving `time.Now()` to its loop alone.
* **Go's own pulled-event engine.** `crypto/tls`'s QUIC API (Go 1.21,
  `quic.go`): feed bytes with
  `HandleData(level, data)`, then drain `NextEvent() QUICEvent` until it
  returns `Kind == QUICNoEvent`. `QUICEvent` is a struct with a
  `Kind QUICEventKind` and kind-specific fields. That form has grown
  three times without a break: Go 1.23 added two kinds and a field behind
  an opt-in flag (`QUICConfig.EnableSessionEvents`), Go 1.26 an error kind
  and a field, and Go 1.27 a config field. Two cautions from it: its
  `Start` runs a goroutine and its clock defaults to `time.Now`, which an
  I/O-free engine avoids; and its `Data []byte` is valid only until the
  next call, which a `string` avoids.
* **Sum types in a Go API.** The Go FAQ leaves variant types out because
  they "overlap in confusing ways with interfaces". The standard library
  uses both a struct with a `Kind` (`tls.QUICEvent`) and an interface
  sealed by an unexported method (`go/ast.Expr`, and `go/constant.Value`,
  which is sealed and has a `Kind`). Neither gives exhaustiveness. A new
  kind of a `Kind` struct reaches a caller's `switch` as a value it does
  not know; a new variant of a sealed interface reaches a type switch the
  same way. The struct form also allows a zero value meaning "no event",
  and keyed literals.
* **Keeping an API compatible** (Go blog, "Keeping Your Modules
  Compatible", 2020): "no backward-compatible change you can make to a
  function's signature … add a new function"; adding a method to an
  interface breaks its implementers; "you can almost always add a field",
  if "its zero value is meaningful and preserves the old behavior"; an
  opt-in method or option for new behaviour. Go 1 compatibility: keyed
  struct literals keep compiling when fields are added.
* **The DA1 sentinel is the standard practice.** The Kitty keyboard,
  graphics and notification specifications each say to send the query,
  then DA1, and to read a DA1 reply that arrives first as "not
  supported". terminal-colorsaurus does the same, so it "doesn't need to
  rely on a timeout", and keeps a 1 s timeout for SSH latency and
  terminals that do not answer DA1 (anyterm, Eterm, shellinabox).
  notcurses: "All known terminals respond to DA1." Neovim sends
  `CSI ? u` then `CSI c`. foot advises tertiary, secondary and primary DA
  in that order. termenv uses a cursor-position report instead, with a
  5 s per-byte timeout and a 25-byte reply cap, too small for OSC 4.
* **Pitfalls the sources name.**
  * Some terminals print what they do not consume: DECRQM's `$p` in
    Konsole and Terminal.app (notcurses); APC in the Linux console;
    JetBrains' terminal prints Kitty push and pop. Bubble Tea v2.0.10
    sends its DECRQM for 2026 and 2027 only where
    `shouldQuerySynchronizedOutput` allows: not over SSH, and not in
    Apple Terminal (`tea.go:980-994`).
  * Multiplexers: tmux answers DA1 itself, and passes a query to the
    outer terminal only through `\ePtmux;…\e\\` with `allow-passthrough`,
    off by default. GNU screen relays OSC 11 outward and answers DA1
    first, so the sentinel lies there (colorsaurus treats screen as
    unsupported).
  * Replies do not always mirror the query's terminator: Apple Terminal
    ends OSC replies with BEL even after an ST query, and old urxvt with a
    bare `ESC`. colorsaurus sends BEL, and accepts BEL or ESC.
  * Replies split across reads, even inside a UTF-8 sequence: "the PTY …
    has no idea about the boundaries of escape sequences … implement a
    proper finite state machine" (Windows Terminal, #19282). Windows
    Terminal duplicates DA replies under SSH (#19345).
  * A reply nobody reads ends up on the user's input line, so a program
    must read every reply it asked for, and must not query without input
    (Bubble Tea skips its queries then, #1590).
* **The grammars** (xterm `ctlseqs`, patch #411, and the specifications
  named): DA1 `CSI ? Ps ; … c`; DA2 `CSI > Pp ; Pv ; Pc c`; XTVERSION
  `CSI > 0 q` → `DCS > | text ST`; DECRQM `CSI ? Ps $ p` → `CSI ? Ps ; Pm
  $ y`, Pm 0 not recognised, 1 set, 2 reset, 3 permanently set, 4
  permanently reset; OSC 10, 11 and 4 with `?` reply with the sequence
  that would set the colour, OSC 4 once per index; colours in X11's
  `XParseColor` grammar, `rgb:h/h/h` with one to four hex digits per
  channel scaled to the channel, and `#RGB` to `#RRRRGGGGBBBB` as the most
  significant bits, not scaled; `CSI ? u` → `CSI ? flags u`; the Kitty
  graphics query → `APC G i=… ; OK|error ST`; OSC 99 `p=?` → its keys, of
  which "clients must ignore keys they do not understand"; DSR 996 →
  `CSI ? 997 ; 1|2 n`; modes 2026, 2027, 2031 and 2048 read by DECRQM,
  Pm 0 or 4 meaning unsupported. xterm sends 7-bit controls unless S8C1T
  is on; a C1 control can be mistaken for badly formed UTF-8.
* **Parser limits.** No source sets a normative reply length. The DEC
  parser keeps at most 16 parameters, alacritty's 32, and x/ansi's 32
  with a 64 KiB string buffer; tmux discards a sequence over 1 MiB.
  x/ansi's `DecodeSequence(b, state, p) (seq, width, n, newState)` takes
  the caller's state: it is itself sans-IO, and it is the tokenizer
  Bubble Tea's stack uses.
* **Go prior art for a CLI probe.** termenv (above); colorprofile makes no
  query at all, reading the environment and terminfo, and runs `tmux info`
  as a process; `x/term` is mode control only. None offers an I/O-free
  engine.

#### A2.2 The API

All in `termcap`; every type name passed `python3 -m tools.verify names`
with one clash, `termcaptea.Prober`, which is the move.

```go
// Probe learns a terminal's capabilities from one batch of queries that
// ends with DA1. It does no I/O, reads no clock and starts no goroutine:
// the host gives it the environment, each reply and the time, and drains
// the events it returns. It is not safe for concurrent use.
type Probe struct{ /* … */ }

func NewProbe(o ...Option) *Probe

func (p *Probe) Start(env Env, now time.Time)          // the environment's facts; queues the batch
func (p *Probe) HandleReply(r Reply, now time.Time)    // one reply, in the order it arrived
func (p *Probe) HandleTimeout(now time.Time)           // the deadline Timeout gave has passed
func (p *Probe) HandleHook(h ProbeHook, v, ok bool)    // a hook's answer (ProbeRunHook)
func (p *Probe) SetColorProfile(c ColorProfile)        // a host's own detection
func (p *Probe) SetTmux(f TmuxFacts)
func (p *Probe) Timeout() (deadline time.Time, ok bool) // when HandleTimeout is due
func (p *Probe) NextEvent() ProbeEvent                 // ProbeNoEvent when none is left
func (p *Probe) Awaiting() bool                        // a reply may still come
func (p *Probe) Caps() Caps                            // what is known now, overrides applied
func (p *Probe) Restore() string                       // bytes that reset every mode it set

type ProbeEventKind uint8
const (
	ProbeNoEvent     ProbeEventKind = iota // the zero value: nothing to do
	ProbeWrite                            // write Data to the terminal, in order
	ProbeRunHook                          // run Run, then pass its answer to HandleHook
	ProbeDone                             // the probe ended; Caps holds the facts
	ProbeColorScheme                      // a DSR 997 after the probe; Dark says which
)

type ProbeEvent struct {
	Kind ProbeEventKind
	Data string                   // ProbeWrite
	Hook ProbeHook                // ProbeRunHook
	Run  func() (v, ok bool)      // ProbeRunHook: the program's hook
	Caps Caps                     // ProbeDone
	Dark bool                     // ProbeColorScheme
}

type ProbeHook uint8 // HookAppearance, HookConsoleHost

// Reply is one terminal reply. ParseReply always sets Raw; a host that
// has only a decoded form, as Bubble Tea gives, sets the fields of its
// Kind and may leave Raw empty.
type Reply struct {
	Kind    ReplyKind
	Raw     string
	Params  []int       // DA1, DA2
	Mode    int         // DECRPM
	Setting ModeSetting // DECRPM
	OSC     int         // ReplyColor: 4, 10 or 11
	Index   int         // ReplyColor with OSC 4
	Color   color.Color // ReplyColor
	Flags   int         // ReplyKittyKeyboard
	Dark    bool        // ReplyColorScheme
	Text    string      // ReplyVersion; ReplyKittyGraphics's "OK" or error; ReplyNotify's keys
}

type ReplyKind uint8 // ReplyUnknown (the zero value), ReplyPrimaryAttributes,
// ReplySecondaryAttributes, ReplyVersion, ReplyMode, ReplyColor,
// ReplyKittyKeyboard, ReplyColorScheme, ReplyKittyGraphics, ReplyNotify

type ModeSetting uint8 // ModeNotRecognized, ModeSet, ModeReset,
// ModePermanentlySet, ModePermanentlyReset: DECRPM's Pm 0 to 4

// ParseReply reads one complete reply, as a decoder such as termkey's
// returns it. It never fails: a sequence it does not know is ReplyUnknown.
func ParseReply(raw string) Reply

// WithoutModeQuery leaves DECRQM for the modes out of the batch: the host
// asks for them itself and passes their reports to HandleReply.
func WithoutModeQuery(modes ...int) Option
```

`Query` keeps its shape, `Name`, `Seq func(Env, Mux) string`, `Gated`,
`Parse func(Reply, *Caps) bool`, and its `Reply` is the one above. Every
existing option stays: `WithTimeout`, `WithQuery`, `WithoutHeuristic`,
`WithOverride`, `WithoutColorSchemeUpdates`, `WithoutBackgroundRequest`,
`WithDisabled`, `WithGOOS`, `WithAppearanceEnv`, `WithAppearanceHook`,
`WithConsoleHost`, and the new `WithoutModeQuery`. `Option`'s method
becomes `apply(*Probe)`. `EnvCaps` is a disabled `Probe` started on the
environment, as it is a disabled `Prober` today.

#### A2.3 The rules the design follows, and why

1. **No I/O, no clock, no goroutine** (sans-IO; quinn-proto). Every
   method returns at once; `now` comes in, `Timeout` goes out; the
   program's hooks, which may start a process, are handed back as
   `ProbeRunHook` for the host to run where it likes. So every test is
   deterministic, and a transcript of a real terminal is a test.
2. **Pulled events in a struct with a `Kind`** (`tls.QUICEvent`). The zero
   `ProbeEvent` is "no event". A host drains `NextEvent` until it gets
   `ProbeNoEvent`, and **ignores a kind it does not know**, which the type's
   documentation says, so a later kind is an addition.
3. **Beyond the base contract, an event that needs the host to act is only
   sent after the host opts in** (Go 1.23's `EnableSessionEvents`). Every
   host handles `ProbeWrite` and `ProbeDone`; that is what driving a probe
   means. `ProbeRunHook` comes only for a hook the program set, and
   `ProbeColorScheme` is information a host may drop. A later kind that
   needs action gets an option of its own, so a host that ignores an
   unknown kind loses nothing.
4. **`Reply` has a `Kind`, and `ParseReply` never fails.** A reply this
   version does not know is `ReplyUnknown` with `Raw`, which a custom
   `Query` can read; a later version that knows it changes only its
   `Kind`, and `Raw` is still there.
5. **Fields only grow, with meaningful zero values** (the Go blog;
   `Caps`'s existing rule). `Reply`, `ProbeEvent` and `Caps` gain fields,
   never change one; the documentation says to build them with keyed
   literals (Go 1 compatibility).
6. **No dependency type in the API.** `ModeSetting` is `termcap`'s own,
   not `x/ansi`'s, so `x/ansi`, a pre-1.0 module, can change under the
   engine without changing its API. `x/ansi` is used inside: its
   `DecodeSequence` and `Parser` tokenize a reply, its constants are the
   queries.
7. **Strings out, not shared buffers.** `ProbeEvent.Data` and `Reply.Raw`
   are strings, so no caller can keep a buffer the engine reuses, the
   caution `QUICEvent.Data` documents.
8. **Not safe for concurrent use,** as `tls.QUICConn` is: one goroutine
   drives it, the TUI's loop or the CLI's.

#### A2.4 The standards it follows, and the changes from today

* **The batch:** the queries, then DA1, written as one `ProbeWrite`, in
  today's order. A reply to a query sent before DA1 marks it answered; DA1
  ends the probe, and each unanswered query is unsupported. A second DA1
  changes nothing, so a duplicated reply is harmless.
* **The timeout** stays `DefaultTimeout`, 2 s, between colorsaurus's 1 s
  and tmux's 5 s, and bounds the whole probe, not each byte.
* **OSC queries end with BEL,** as `x/ansi`'s constants already do, and
  `ParseReply` accepts BEL, ST, 8-bit ST and, for urxvt, a bare `ESC`.
  C1 introducers (0x9B CSI, 0x9D OSC, 0x90 DCS, 0x9F APC) are accepted as
  their 7-bit forms, as ultraviolet's decoder accepts them.
* **Colours follow `XParseColor` exactly:** `rgb:` with one to four hex
  digits per channel, scaled; `#RGB`, `#RRGGBB`, `#RRRGGGBBB` and
  `#RRRRGGGGBBBB` as the most significant bits; urxvt's `rgba:` read with
  its alpha dropped; the result a `color.RGBA64`, so a 16-bit reply keeps
  its precision. Today's `parseXColor` (`prober.go:737-770`) reads `rgb:`
  and `#rrggbb` only. This is a change: `#RGB` and the 12- and 16-bit `#`
  forms read where they did not.
* **DECRQM for 2026 and 2027.** Bubble Tea sends these itself, so
  `termcaptea` passes `WithoutModeQuery(2026, 2027)` and the reports
  arrive as replies, as today. A CLI's probe sends them, under Bubble
  Tea's own rule, `shouldQuerySynchronizedOutput`, copied with its
  source named: not over SSH, and not in Apple Terminal.
* **Unchanged on purpose:** today's DECRQM for 1004, 2031 and 2048 go to
  every terminal, Apple Terminal among them, though notcurses reports that
  Terminal.app prints `$p`; GNU screen is probed as any terminal, though
  its DA1 arrives before the outer terminal's replies. Each is a finding
  for a later record, not part of a refactor that must keep the TUI's
  behaviour.
* **Lengths:** a reply over 1 KiB is not parsed, and the query it would
  answer is marked `ReasonReplyTooLong`, as today.
* **Split replies.** The engine sees whole replies only; joining bytes is
  the decoder's job. `termcaptea` keeps `IsReplyFragment` for
  ultraviolet's escape-timeout split. A CLI using `termkey` does not call
  `Decoder.Flush` while `Probe.Awaiting()` is true, so a reply split
  across reads is joined, not given up as `InputUnknown` halves.

#### A2.5 What it costs

* Reply parsing moves into the core: about ten reply kinds that Bubble Tea
  and ultraviolet decode for the prober today, on `x/ansi`'s tokenizer.
* `termcaptea` translates Bubble Tea's decoded messages into `Reply`s:
  two input paths, which a differential test holds together (the PLAN).
* The exported API grows by `Probe` and its methods, `ProbeEvent`,
  `ProbeEventKind`, `ProbeHook`, `ReplyKind`, `ModeSetting`,
  `ParseReply` and `WithoutModeQuery`; `Reply` changes shape. Under the
  stability policy these are kept.
* A CLI that probes takes on raw mode and reading every reply. The
  engine's documentation says so, with the pitfalls of A2.1.

### A3 (2026-10-23): what 0020-PLAN Step 5's differential found (corrects D9 S5; adds to D9's `KeepSGR`)

*Status: accepted (2026-10-23).* At
[0020-PLAN-charm-free-cores-glyphs-and-safetext.md](0020-PLAN-charm-free-cores-glyphs-and-safetext.md)
Step 5, the differential against `v0.12.0` found differences beyond S2
that the PLAN had not foreseen. The agent stopped with the evidence below,
and the owner chose the recommended resolution of each of three questions
that day. The third, the step's file list, is the PLAN's alone and is
recorded there.

**A3.1 Found.** `internal/sanitize`, wrapping `safetext` as S5 says, was
compared with `v0.12.0`'s copy over 400,000 inputs generated from a
hostile alphabet, each difference shrunk to a minimal input: 1,633 minimal
inputs, every one in one of six classes.

| # | Class | In | Why |
| :--- | :--- | :--- | :--- |
| 1 | A C1 code point, U+0080–U+009F encoded as UTF-8, is read as its 7-bit form: U+0090 opens a DCS, U+009B a CSI, U+009C is ST | `Line`, `Styled` | 0009-MADR §2; `v0.12.0` dropped the code point alone and printed what followed it |
| 2 | BEL ends a DCS, SOS, PM or APC | both | §2; `x/ansi` keeps BEL as data, so `v0.12.0` hid the text after it |
| 3 | A byte of 0x80 or more inside a DCS, SOS, PM or APC is part of it | both | §2; `x/ansi`'s table leaves the string at UTF-8 and prints the rest |
| 4 | Inside an OSC, a character whose UTF-8 holds the byte 0x9C (`Ü` is `c3 9c`) does not end it | both | `safetext` decodes UTF-8 before the grammar, so a byte inside a character is not a C1 control; `x/ansi` reads the bytes one at a time and takes that 0x9C for ST |
| 5 | A malformed sequence splits as the DEC parser splits it: a C0 right after ESC, an nF escape, a DCS holding a C0, a private marker after a parameter | `Styled` | `v0.12.0`'s `Styled` tokenised with `ansi.DecodeSequence`, which splits these otherwise; in all 440 such minimal inputs the new `Styled` equals `v0.12.0`'s own `Line` |
| 6 | An SGR or OSC 8 introduced or ended by an encoded C1 was kept, rewritten in its 7-bit form (`"\u009bm"` became `"\x1b[m"`) | `Styled` | nothing decided it; A3.3 does |

Classes 1, 3 and 4 and part of 5 remove more than `v0.12.0` did; class 2
and parts of 1 and 5 remove less. Where the new output shows what
`v0.12.0` hid, what it shows is text: no difference put a control
character or an ESC in the output, but class 6's two. `HasControl` differs
on no input when compared with `v0.12.0`'s on the input itself:
`v0.12.0` counted invalid UTF-8 as a control too, since its `Line` dropped
it.

**A3.2 S5's rules equal the old behaviour on well-formed input (corrects
S5).** S5 says the wrapper's rules are "equal to their behaviour". That
holds where the input's sequences are well formed. On malformed input,
classes 1 to 5 apply, by 0009-MADR §2's grammar, which D9 makes the one
grammar; the owner accepted them as that grammar. So the library's visible
output changes on malformed input as well as on invalid UTF-8, and the
`v0.13.0` notes say so. D9's "The checks" stands as written: the
differential compares every input on which the two are meant to agree,
and names each class where they differ.

**A3.3 `KeepSGR` keeps only what is written with 7-bit ESC (adds to D9).**
A sequence introduced by an encoded C1 (U+009B, U+009D) or ended by one
(U+009C) is removed whole under `KeepSGR`, as any sequence it does not
keep is. Every sequence `KeepSGR` keeps is then the input's own bytes, as
`v0.12.0`'s `Styled` documented ("a control sequence introduced by an
8-bit C1 byte" is dropped). Keeping it rewritten in its 7-bit form, the
other option, was as safe, but a kept sequence would then differ from what
the program wrote.

### A4 (2026-10-24): release 1 is `v1.13.0` (changes D12's numbering; opens A4.1)

*Status: A4 accepted (2026-10-24); A4.1 answered (2026-10-24): release 2
is `v2.0.0` at `/v2`; A4.2 proposed (2026-10-24), for the owner.*

**Found.** At 0020-PLAN Step 6 the owner tagged `8be11e7`, the commit CI
passed, as `v1.13.0`, not `v0.13.0`, and pushed it. Before the consumer
smoke test the agent found:
* `git ls-remote` shows `refs/tags/v1.13.0` and no `v0.13.0`;
* `proxy.golang.org` lists `v1.13.0` and answers `@latest` with it,
  fetched at 00:52 UTC.

A published tag is never moved or deleted (`docs/guides/releasing.md`,
"The rules"), and the proxy keeps every version it has served. Go orders
`v1.13.0` above every `v0.x`, so a later `v0` tag would never be
`@latest`.

**The options** put to the owner, the recommended one first:
* (a) retract `v1.13.0` with a `v1.13.1` that retracts both, and tag
  `v0.13.0`;
* (b) keep `v1`, and number the releases from it;
* (c) leave it.

**Chosen: (b),** by the owner, 2026-10-24. Release 1 (D12) is `v1.13.0`.
Its content is the commit the PLAN planned for `v0.13.0`, unchanged.

**A4.1 (open): release 2 under `v1`.** D10 and D12 make release 2 one
breaking minor release, `v0.14.0`, by exception to API rule 1. Under
`v1` that is not available as written:
* a `v0.14.0` would sort below `v1.13.0`;
* Go's import compatibility rule ("If an old package and a new package
  have the same import path, the new package must be backwards
  compatible with the old package") lets a module break its API only
  under a new major version's path, `…/v2`.

The owner chooses among:
* release 2 as `v2.0.0` at the module path
  `github.com/maccavelli/go-tui-lib/v2`;
* the new cores beside today's packages, with the old names kept and
  deprecated, so that release 2 is `v1.14.0` and breaks nothing, which is
  this record's option B;
* a breaking `v1.14.0` against that rule.

Whichever is chosen also settles API rule 1's "removed in the next minor
release" (AGENTS.md) for `v1`, and `releasing.md`'s versions. Steps 7 to
13 of the PLAN wait for the answer.

**A4.1 answered: release 2 is `v2.0.0`, at the module path
`github.com/maccavelli/go-tui-lib/v2`,** chosen by the owner, 2026-10-24,
the recommended option. It is the one form Go's import compatibility rule
allows for a breaking change. D2 to D7's split ships as designed, and a
`v1` user's `go get -u` never meets it.
* *D10 under it:* the exception to API rule 1 stands, as a major release
  in place of a minor one. A `v1` caller moves once, to `/v2`, and the
  moves D4 to D7 list are its only API changes.
* *D12 under it:* release 2 is `v2.0.0`.
* *Costs:* every import path in the repository, the depguard and
  conformance paths, and the guides' import lines move to `/v2`. gobble
  and pi-go change their imports when they adopt `v2`.
* *What is still to be decided:* how the PLAN sequences the move, and
  what API rule 1 says under a major version. The next PLAN amendment
  proposes both, for the owner's approval.

**A4.2 (proposed, 2026-10-24): how the move lands, and the rules under a
major version.**

* **The move lands first,** as 0020-PLAN Step 6A, before any split. From
  that commit `main`'s `go.mod` names `…/v2`, and the go command refuses
  a `v1.x` tag of such a commit ("go.mod has post-v1 module path").
  Steps 7 to 12 then build the splits at the path they ship at. The other
  order would put the move in the release step: no step would change
  path, but `main` would carry a breaking API at the `v1` path for six
  steps, where a mistaken `v1` tag, like the one A4 records, would
  publish it.
* **The API diff gate across a major boundary.** apidiff compares types
  by import path. Compared with `v1.13.0` as it is, the `/v2` tree shows
  every signature that names another of the module's packages as
  changed. That was tried on a scratch copy, at `220506d` with every path
  moved: `./command.(*Registry).Available: changed from
  func(github.com/maccavelli/go-tui-lib/when.Context, …)` and many more.
  With the base's own import paths moved to the tree's module path before
  it is exported, the same comparison reported nothing. So
  `scripts/go-apicheck.py` does that move when the base's module path and
  the tree's differ only by a `/vN` suffix. Steps 7 to 12's allow-list
  blocks then list their real moves against `v1.13.0`, as the PLAN's Rule
  3 asks.
* **API rule 1 under a major version** (AGENTS.md). Within one major
  version nothing is removed or retyped: a renamed name keeps its old one,
  deprecated, until the next major version, and a removal needs a major
  release and a record.
* **API rule 2.** An `apicheck.allow` block is allowed only in a PLAN
  whose release changes the major version. From `v2.0.0` on, a minor
  release has 0 incompatible changes.
* **`v1` users** keep `v1.13.0`. A fix for them would be a `v1.13.x` from
  a branch cut at that tag; this record plans none.
* **0008-PLAN's release** (Step 9) becomes the next minor version after
  `v2.0.0`.

## More Information

* Issues of this repository: #2 (`termcap`), #3 (`termsvc`), #4
  (`command`), #5 (`keymap`), #11 (`glyph`), #12 (the sanitizer), each
  with gobble's comment of 2026-10-23 on the `x/ansi` relaxation; #6
  (`fuzzy`, a tag only) and #7 to #10 (line-mode components) are outside
  this record.
* gobble-cli `docs/decisions/0032-MADR-go-tui-lib-integration-boundary.md`:
  D1 to D7, and amendment A1 (A1.1 to A1.4).
* Records here:
  [0005-MADR-terminal-capabilities-and-services.md](0005-MADR-terminal-capabilities-and-services.md)
  (`termcap`, `termsvc`, ultraviolet's confinement);
  [0006-MADR-command-registry.md](0006-MADR-command-registry.md);
  [0007-MADR-keymap-engine.md](0007-MADR-keymap-engine.md) (the notation,
  A8's key labels);
  [0008-PLAN-command-palette.md](0008-PLAN-command-palette.md);
  [0009-MADR-streaming-content-engine.md](0009-MADR-streaming-content-engine.md)
  (§1, §2, A1, A4) and
  [0009-PLAN-streaming-content-engine.md](0009-PLAN-streaming-content-engine.md)
  (Step 2);
  [0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md)
  (W0, the API rules);
  [0014-PLAN-hardening.md](0014-PLAN-hardening.md) (Step 2, C6; Step 9,
  D8 and D9);
  [docs/glossary.md](../glossary.md).
* External sources, read 2026-10-23:
  * **Key encodings.** xterm's control sequences (`ctlseqs`, patch #411):
    PC-style function keys, the modifier parameter, modifyOtherKeys and
    formatOtherKeys, bracketed paste, and the C1 controls,
    <https://invisible-island.net/xterm/ctlseqs/ctlseqs.html>. The Kitty
    keyboard protocol, <https://sw.kovidgoyal.net/kitty/keyboard-protocol/>.
    fixterms, <http://www.leonerd.org.uk/hacks/fixterms/>. Windows
    Terminal's spec #4999, win32-input-mode,
    <https://github.com/microsoft/terminal/blob/main/doc/specs/%234999%20-%20Improved%20keyboard%20handling%20in%20Conpty.md>.
  * **Escape timeouts.** tmux `options-table.c` (`escape-time`); Vim and
    Neovim `options.txt` (`ttimeoutlen`) and Vim's `defaults.vim`;
    ultraviolet `terminal_reader.go` (`DefaultEscTimeout`).
  * **Text hazards.** CVE-2021-42574,
    <https://nvd.nist.gov/vuln/detail/CVE-2021-42574>, and Boucher and
    Anderson, "Trojan Source", <https://arxiv.org/abs/2111.00169>.
    Unicode `PropList.txt` (Bidi_Control),
    <https://www.unicode.org/Public/UCD/latest/ucd/PropList.txt>. UTS #55,
    <https://www.unicode.org/reports/tr55/>. The Unicode core
    specification, chapter 3, §3.9.6,
    <https://www.unicode.org/versions/latest/core-spec/chapter-3/>. David
    Leadbeater, "ANSI Terminal Security in 2023 and Finding 10 CVEs",
    <https://dgl.cx/2023/09/ansi-terminal-security>.
  * **Widths.** `EastAsianWidth.txt`,
    <https://www.unicode.org/Public/UCD/latest/ucd/EastAsianWidth.txt>;
    UAX #11, <https://www.unicode.org/reports/tr11/>; `emoji-data.txt`.
  * **Glyph precedent.** sindresorhus/figures `index.js`; log-symbols;
    cli/cli `pkg/iostreams/color.go`; cli/go-gh `pkg/text/text.go`.
  * **Dependency enforcement.** OpenPeeDeeP/depguard `depguard.go`;
    ryancurrah/gomodguard's README; Go's `src/go/build/deps_test.go`;
    Kubernetes `cmd/import-boss`; `go help list`; the Go Modules
    Reference, "Module graph pruning", <https://go.dev/ref/mod#graph-pruning>.
  * **Splitting and moving.** prometheus/client_golang (`prometheus` and
    `promhttp`); the Go wiki's Modules FAQ, "Should I have multiple
    modules in a single repository?"; the type-alias design,
    <https://github.com/golang/proposal/blob/master/design/18130-type-alias.md>;
    Russ Cox, "Codebase Refactoring (with help from Go)",
    <https://go.dev/talks/2016/refactor.article>; Go 1.26 release notes
    (`//go:fix inline`), <https://go.dev/doc/go1.26>.
* Sources for A2, read 2026-10-23:
  * **I/O-free engines.** <https://sans-io.readthedocs.io/how-to-sans-io.html>;
    Cory Benfield, "Building Protocol Libraries The Right Way", PyCon 2016;
    quinn-proto 0.11.19, <https://docs.rs/quinn-proto/latest/quinn_proto/struct.Connection.html>;
    Go's `crypto/tls` `quic.go` and `api/go1.21.txt`, `go1.23.txt`,
    `go1.26.txt`, `go1.27.txt`; `golang.org/x/time/rate` v0.16.0
    `rate.go`; `golang.org/x/net/quic` v0.61.0 `conn.go`, `conn_recv.go`.
  * **Go API design.** The Go FAQ, "Why does Go not have variant types?";
    "Keeping Your Modules Compatible", <https://go.dev/blog/module-compatibility>;
    Go 1 compatibility, <https://go.dev/doc/go1compat>; `go/ast` and
    `go/constant`.
  * **Probing practice.** The Kitty keyboard, graphics and desktop
    notification specifications, <https://sw.kovidgoyal.net/kitty/>;
    terminal-colorsaurus 1.0.3, `doc/feature-detection.md`,
    `doc/terminal-survey.md`, `src/quirks.rs`, `src/xterm.rs`; notcurses
    `src/lib/termdesc.c`, `src/lib/in.c`; Neovim `src/nvim/tui/tui.c`;
    foot's README and `doc/foot-ctlseqs.7.scd`; termenv v0.16.0
    `termenv_unix.go`; colorprofile v0.4.3 `env.go`; tmux `tmux.1`,
    `input.c`, `tty.c`, `options-table.c`; Bubble Tea v2.0.10 `tea.go`;
    Windows Terminal issues 19120, 19282 and 19345.
  * **Grammars.** xterm `ctlseqs` (patch #411); the Xlib manual, "Color
    Strings"; Contour's VT extensions (colour palette update notifications,
    synchronized output); the in-band resize specification,
    <https://gist.github.com/rockorager/e695fb2924d36b2bcf1fff4a3704bd83>;
    terminal-unicode-core's mode 2027.
  * **Parsers.** Paul Flo Williams, <https://vt100.net/emu/dec_ansi_parser>;
    alacritty vte 0.15.0; x/ansi v0.11.8 `parser.go`, `parser_decode.go`;
    ultraviolet `decoder.go`, `terminal_reader.go`.
* The PLAN:
  [0020-PLAN-charm-free-cores-glyphs-and-safetext.md](0020-PLAN-charm-free-cores-glyphs-and-safetext.md),
  written to the owner's answers to Q1 to Q8 and to A1.
