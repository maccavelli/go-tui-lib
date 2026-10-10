---
status: in-progress
date: 2026-10-10
associated-madr: "0014-MADR-native-integration-api.md"
---
# Implement W4: renames with deprecation aliases, opaque options, one enum helper, JSON v2, structured errors, and the planned records' names; release `v0.9.0` and `v0.10.0`

Associated MADR: [0014-MADR-native-integration-api.md](0014-MADR-native-integration-api.md),
workstream W4, as amended by A1. Evidence:
[0014-REPORT-api-assessment-and-integration-research.md](../reports/0014-REPORT-api-assessment-and-integration-research.md)
§2.4, C1–C13.

## Goal

By `v0.10.0`, one name means one thing across the library. In detail:

* **The names.**
  * Every rename lands in `v0.9.0` beside the old name, which is marked
    deprecated.
  * The old names go in `v0.10.0`, and the collision check's temporary
    entries go with them.
* **The internals.** Option types are opaque. Every exported enum has
  text forms through one helper. `command` and `termcap` use JSON v2
  only.
* **Errors and hooks.** `when` and `termcap` have structured errors, and
  every hook has a `…Func` adapter.
* **The planned records.** 0007, 0008 and 0009 take their names from the
  glossary before they are built.

## Scope

### Facts this PLAN starts from (2026-10-07)

| Fact | Where it was read |
| :--- | :--- |
| `command.Decision` and its four constants, `Gate.Decide(…) (Decision, error)`, and the field `Record.Decision`; the audit's slog key is `"decision"`; an unexported `verdict` function exists | `command/gate.go:11-18`, `:44`, `:107`; `command/audit.go:23`, `:47` |
| a probe: `type Decision = Verdict` keeps every old use compiling, including old `Gate` implementations; `apidiff` sees only `Verdict: added` | a scratch module |
| `Request.Context` and `Invocation.Context` are `when.Context` fields; the owner chose `WhenContext` (A1.1) | `command/handler.go:35`, `:52` |
| a probe: keeping both fields and reading new-then-old compiles old literals; `apidiff` sees `Request.WhenContext: added` | a scratch module |
| `termcap.New` has no non-test caller | `termcap/prober.go:191`; search |
| `layout`'s preset options have bare names: `SidebarWidth`, `BottomHeight`, `MainSize`, `BottomSpan`, `Footer`, `Gap`, `Breakpoints`, `NoResponsive` | `layout/presets.go:28`, `:56-88` |
| `workspace.WithMouse(on bool)`, default on; `termsvc.WithGate(func(Notification) bool)`; `WrapOption`'s `On…` (callbacks) and `With…` (values) already fit W0.4 | `workspace/workspace.go:242-244`, `:268`; `termsvc/notify.go:130-132`; `workspace/model.go:31-73` |
| six option types are functions over exported structs: `command.Option`, `RegistryOption`, `termcap.Option`, `termsvc.NotifyOption`, `workspace.Option`, `workspace.WrapOption[M]`; no test, guide or example writes its own literal | `command/args.go:29`, `registry.go:67`; `termcap/prober.go:48`; `termsvc/notify.go:118`; `workspace/workspace.go:184`, `model.go:31`; search |
| a probe: an interface option with an unexported method keeps every library option compiling and refuses a third party's; `apidiff` reports it incompatible, and no alias route exists | a scratch module |
| enum text is written three ways: termcap's generics (seven enums), command's slices and switches, termsvc's slices; tests pin the out-of-range text: `"Kind(9)"`, `"Origin(99)"` and termcap's `"9"` | `termcap/termcap.go:124-145`; `command/command.go:123-288`; `termsvc/notify.go:92`, `beacon.go:54`; `command/registry_test.go:316,321`, `termcap/termcap_test.go:63` |
| the one-message `tea.Cmd` helper is written three times, and inline twice more | `command/dispatch.go:196`, `termsvc/clipboard.go:96`, `termsvc/notify.go:199`; `termcap/prober.go:523,529`, `termsvc/links.go:69` |
| `command` imports JSON v1 only for `RawMessage`, which is `= jsontext.Value`; `termcap` uses v1 to marshal `Caps` and the report | `command/audit.go:22`, `handler.go:31,48`, `command.go:74`; `termcap/caps.go:84,111`, `report.go:72` |
| v1→v2, measured: `<>&` not escaped; nil slices give `[]`; field names case-sensitive; duplicate keys refused; invalid UTF-8 refused. Only `"{}"` and a `"findings": [` substring pin termcap's JSON | a probe; `termcap/termcap_test.go:133-138`, `findings_test.go:101` |
| `when`'s errors come from one `scanner.errorf`, at 19 sites; two size errors have no offset; one wraps a regex error with `%v`; termcap's text unmarshalers share one error site | `when/lex.go:46-48`, `when/when.go:50,61,66`, `when/parse.go:163`; `termcap/termcap.go:141` |
| hooks without `…Func` adapters: `command.Auditor`, `termsvc.Backend`, `termsvc.Clipboard` (W2 adds `GateFunc`) | `command/audit.go:11-13`; `termsvc/notify.go:113-115`; `termsvc/clipboard.go:55-57` |
| width is measured four ways: workspace's method (WcWidth, GraphemeWidth after mode 2027), `ansi.StringWidth` in tuitest, `ansi.Truncate` in termsvc, and termcap's report | `workspace/workspace.go:245-253`; `tuitest/tuitest.go:215`; `termsvc/termsvc.go:115`; `termcap/report.go:215-226` |
| the stale `--args` wording | `command/slash.go:12-13`; `docs/guides/commands.md:171-172` |
| collisions planned in 0007, 0008 and 0009 | 0007-MADR `:259-442`; 0008-MADR `:167-302`; 0009-MADR `:185-760`; the W4 fact sheet |

### Facts re-read before execution (2026-10-09)

W2 (0014-PLAN-component-native-forms), W3 (0014-PLAN-hardening) and the
Go 1.27.2 floor (0016-PLAN-go-1-27-2-for-go-2026-6604) landed after this
PLAN was written, and `v0.8.0` is tagged. The agent read every row above
again on `b6e3cef`. The table above keeps the 2026-10-07 readings; this
one gives today's lines and what changed. The steps below are annotated
where it changes them.

| Fact | Today |
| :--- | :--- |
| `Decision`, its four constants, `Gate.Decide`, `GateFunc` (W2), `Record.Decision` and the slog key `"decision"` hold; the unexported `verdict` function holds | `command/gate.go:11-18`, `:45`, `:49-52`, `:128`; `command/audit.go:23`, `:47` |
| `Request.Context` and `Invocation.Context` hold; `admit` reads `req.Context` and builds the `Invocation` with it; the builtin list reads `inv.Context` | `command/handler.go:37`, `:54`; `command/dispatch.go:161`, `:167`; `command/builtin.go:28` |
| `termcap.New` still has no non-test caller | `termcap/prober.go:194` |
| `layout`'s bare preset options hold | `layout/presets.go:56-88` |
| `workspace.WithMouse(on bool)` and `termsvc.WithGate` hold | `workspace/workspace.go:316`; `termsvc/notify.go:134` |
| The six option types over exported structs hold. Since this PLAN, `command.LoadOption` (W3) and `launch.Option` (0013) are interfaces already, and `layout.PresetOption`, `termcap.ReportOption`, `termsvc.CopyOption`, `theme.Option` and `workspace.CommandOption` are functions over unexported structs, which no third party can write; they stay | `command/args.go:35`, `registry.go:121`; `termcap/prober.go:49`; `termsvc/notify.go:120`; `workspace/workspace.go:200`, `model.go:32` |
| **Changed, by W2:** `internal/enum` takes `~uint8` and `~int` types, and already serves termcap's seven enums, `command.Format`, `glyph.Tier`, `launch.Choice` and `Target`, and `theme`'s two. 21 enums still lack text forms: `command`'s `Kind`, `Danger`, `Surface`, `Mode`, `SourceKind`, `Origin` and `Decision` (which has `ACPKind`, not `String`); `termsvc`'s eight; `layout`'s three; `workspace`'s two; `when.Kind`. The pins `"Kind(9)"`, `"Origin(99)"` and termcap's `"9"` hold | `internal/enum/enum.go`; `command/command.go:117-289`; `command/registry_test.go:316`, `:321`; `termcap/termcap_test.go:63` |
| **New, from W3:** `FuzzEnumText` pins that each termcap name `UnmarshalText` takes is the name `MarshalText` gives back | `termcap/fuzz_test.go` |
| The one-message `tea.Cmd` helper is still written three times and inline twice | `command/dispatch.go:251`; `termsvc/clipboard.go:121`, `notify.go:258`; `termcap/prober.go:528`; `termsvc/links.go:70` |
| **Changed:** `command` imports JSON v1 in four files, not three: `args.go` (`ArgsOf` returns `json.RawMessage`) as well as `audit.go`, `command.go` (`Schema = json.RawMessage`) and `handler.go`. W2 added v2 imports beside them. `termcap` still marshals `Caps` and the report with v1. `layout/state.go` also uses v1; it is not in this PLAN's scope | `command/args.go:5`, `:94`; `command/command.go:77`; `termcap/caps.go:84`, `:111`; `report.go:72`; `layout/state.go:4` |
| `when`'s 19 `errorf` sites hold; the two size errors are `fmt.Errorf` with no offset; `parse.go:163` wraps the regex error with `%v`; termcap's unmarshalers share one site, now `enum.Unmarshal` | `when/lex.go:46`; `when/when.go:53`, `:69`; `when/parse.go:163`; `termcap/termcap.go:133` |
| Hooks without `…Func` adapters: `command.Auditor`, `termsvc.Backend`, `termsvc.Clipboard`. W2 added `GateFunc` | `command/audit.go:11`; `termsvc/notify.go:115`; `termsvc/clipboard.go:61` |
| **Changed, by W3:** the workspace cuts every line and label with `sanitize.Truncate`, which never returns text wider than asked: rune by rune under WcWidth, and checked under GraphemeWidth, where `clip` also keeps a line's ends from joining a border (0014-PLAN-hardening D8). Tests measure with W2's `tuitest.Fits` under the case's method; `termsvc` and `termcap`'s report truncate by grapheme | `workspace/render.go:174`, `:299`, `:337`; `internal/sanitize/sanitize.go`; `tuitest/case.go:28`; `termsvc/termsvc.go:104`; `termcap/report.go:215-226` |
| The stale `--args` wording holds | `command/slash.go:13`; `docs/guides/commands.md:191` |
| **Changed:** the collision check allows six shared names, not five: `Terminal`, in `launch/launchtest` and `termcap/termcaptest`, was added by 0013-PLAN-cli-integration-helpers D5. Its `Decision` entry still says "until v0.10.0" | `internal/conformance/conformance_test.go:470-482`; `docs/glossary.md:28` |
| **Changed:** the glossary's "Reserved by accepted records" still lists names W2 and W3 have since built: `Format`, `Param`, `PanicError`, `PlainViewer`, `GlyphThemeBuilder` and `LoadOption`. 0007, 0008 and 0009 have not changed since this PLAN | `docs/glossary.md:31-44`; `git log` |
| **A conflict, resolved by the MADR:** 0014-MADR-native-integration-api A1.5 says W3's deprecated variables, `termcaptest.RunTimeout` and the seven `workspace.Key…`, are removed in `v0.9.0`; so do AGENTS.md's "API conventions" rule 1 (one minor release) and 0014-PLAN-hardening Step 8. This PLAN's Step 9 removed them in `v0.10.0`. They move to Step 4, in `v0.9.0` | `docs/decisions/0014-MADR-native-integration-api.md` A1.5; `AGENTS.md`; Step 9 below |
| The preconditions hold: `v0.8.0` is tagged at `f6c9f9b`, and `scripts/apicheck.allow` lists nothing | `git tag`; the file |

### Preconditions

* **`v0.8.0` is tagged** (0014-PLAN-hardening).
* **`scripts/apicheck.allow` is empty.**

### In scope

| Step | Delivers | Release |
| :--- | :--- | :--- |
| 1 | approval | — |
| 2 | the glossary updated; 0007, 0008 and 0009 amended to it | — |
| 3 | `command`: `Verdict`, `WhenContext`, `Record.Verdict` | `v0.9.0` |
| 4 | `termcap.NewProber`; `layout`'s `With…` options; `workspace.WithoutMouse`; `termsvc.WithNotifyFilter` | `v0.9.0` |
| 5 | opaque options | `v0.9.0` |
| 6 | `internal/enum` for every enum; `internal/teamsg` | `v0.9.0` |
| 7 | JSON v2 only; `when.SyntaxError`; `termcap.ErrUnknownName`; the `…Func` adapters; the width rule; the `--args` wording | `v0.9.0` |
| 8 | the release `v0.9.0` | `v0.9.0` |
| 9 | every deprecated name removed | `v0.10.0` |
| 10 | the release `v0.10.0`, and close-out | `v0.10.0` |

### Out of scope

* **Renaming the five deliberate collisions** (`Context`, `Kind`,
  `Option`, `Origin`, `Pane`). The glossary records them as deliberate,
  per W0's PLAN. Renaming any needs the owner and a record. (2026-10-09:
  six, with `Terminal`, the two test kits' fake terminal, from
  0013-PLAN-cli-integration-helpers D5.)
* **`layout/state.go`'s JSON v1** (added 2026-10-09). The goal names
  `command` and `termcap` only.
* **W5's records.**

## Implementation Steps

### Rules

1. **A step starts when the previous one is committed.** The agent
   commits on `main` only when the owner asks in that turn, with `git
   commit --no-edit`. The owner pushes and tags.
2. **Checks.** The Go checks of
   [0014-PLAN-component-native-forms.md](0014-PLAN-component-native-forms.md)
   Rule 2, with two differences:
   * `make apicheck` may show only the incompatible lines this PLAN
     lists, each entered in `scripts/apicheck.allow` in that step's
     commit, under a header that names this PLAN's filename and the step;
   * the collision check passes.
3. **Every deprecated name** has `// Deprecated: use X (0014-MADR W4).`
   It still compiles and works in `v0.9.x`, and a test proves it.
4. **Mutations** run on scratch copies.
5. **Anything unplanned stops the step.**

### Step 1: records

* **This PLAN, approved.**
* **Its row in `docs/README.md`.**

### Step 2: the glossary and the planned records

* **`docs/glossary.md`** gains:
  * **`Verdict`:** a gate's answer.
  * **`WhenContext`:** a `when.Context` of key values. `When` stays a
    when-expression.
  * **`Decision`:** `launch`'s start-up outcome only.
  * **`Policy`:** `termsvc`'s notification policy only.
  * **`Plan`, `Rule`, `Span`:** `layout`'s only.
  * **`Result`, `Conflict`, `Scope`, `Source`:** `command`'s only.
  * **`Query`:** a `termcap` query only.
  * **`Tier`:** `glyph`'s only.
  * **`Filter`:** a message filter only.
  * **`Renderer`:** the streaming interface; an adapter's
    `NewRenderer` returns its own type.
  * **`New`:** only when it returns the type named after the package.
* **Amendments, each dated, citing this PLAN.** Each changes only names,
  never a design.
  * **0007-MADR** (`keymap`) (2026-10-09, D1: `command.Scope` itself;
    `Options` → `Settings`; `Match` → `Resolved`):
    * `Context` → `Scope` (or `command.Scope` itself);
    * `Origin` → `Location`;
    * `Conflict` → `BindingConflict`;
    * `Rule` → `Binding`;
    * `Result` → `Outcome`;
    * `Options` → `Config`;
    * `Match` and `Matcher` → `Hit` and `Dispatcher`;
    * the `Rule.Context` and `Conflict.Context` fields → `Scope`;
    * the `Unsupported` constant → `NotSupported`.
  * **0008-MADR** (`fuzzy`, `palette`) (2026-10-09, D1: also
    `Palette` → `Picker`, with `New` → `NewPicker`):
    * `fuzzy.New` → `NewMatcher`, with `With…` options;
    * `Tier` → `Grade`;
    * `Span` → `Range`;
    * `palette.Query` → `Input`, with its `Context` field →
      `WhenContext`;
    * `palette.Scope` → `Availability`;
    * `Index.With` → `Update`;
    * `…Func` adapters for its hooks.
  * **0009-MADR:**
    * `safetext.Policy` → `Rules`;
    * `frame.Policy` → `Pacer`, with `Policy.Decide` → `Pace`;
    * `frame.Plan` → `Release`;
    * `Plan.Mode` → `Gear`;
    * `frame.New` → `NewScheduler`;
    * `frame.Msg` → `FrameMsg`;
    * `safetext.Filter` → `Sanitizer`;
    * `inputfilter.New` → `NewFilter`;
    * `stream.New` → `NewDoc`;
    * `stream.Pane` → `DocPane`;
    * glamourmd `New` → `NewRenderer` and `FromTheme` → `WithTheme`;
    * a `HighlighterFunc` adapter;
    * `safetext`'s overlap with `internal/sanitize` noted for its PLAN.

* **Added 2026-10-09:** the glossary moves `Format`, `Param`,
  `PanicError`, `PlainViewer`, `GlyphThemeBuilder` and `LoadOption` from
  "Reserved by accepted records" to "Built from those records", since W2
  and W3 built them; and its deliberate collisions name six, with
  `Terminal`.

**Done when** the glossary's names and the three amendments agree, and
each record's index row says "A<n> recorded".

### Step 3: `command`'s renames

* **`Verdict`.** `type Verdict uint8` holds the constants, which are now
  typed `Verdict`. `type Decision = Verdict` stays, deprecated. `Gate`'s
  method returns `Verdict`; the alias keeps old implementations
  satisfying it. The unexported `verdict` function becomes `settle`.
* **`Record.Verdict`.** It is added beside `Record.Decision`, which is
  deprecated. Both are set while the old name exists. The slog key stays
  `"decision"` in `v0.9.x` and becomes `"verdict"` in `v0.10.0`, named in
  the notes.
* **`WhenContext`.**
  * `Request.WhenContext` and `Invocation.WhenContext` are added; the old
    `Context` fields stay, deprecated.
  * `admit` reads `WhenContext`, else `Context`
    (`command/dispatch.go:107`; 2026-10-09: `:161`).
  * The `Invocation` it builds sets both (`:113`; 2026-10-09: `:167`).
  * `builtin.go:27` reads `WhenContext` (2026-10-09: `:28`).
* **Uses.** Every use in the library, its tests and the guides moves to
  the new names.
* **Tests:**
  * `TestOldDecisionStillWorks`: an old-signature `Gate` and the old
    constants;
  * `TestWhenContextFallsBack`: `Context` alone, `WhenContext` alone,
    both;
  * `TestRecordCarriesBoth`.
* **Mutation S3-1:** `admit` ignores the old field.
  `TestWhenContextFallsBack`.

### Step 4: the other renames

* **`termcap`:** `NewProber`. `New` calls it, deprecated.
* **`layout`:** `WithSidebarWidth`, `WithBottomHeight`, `WithMainSize`,
  `WithBottomSpan`, `WithFooter`, `WithGap`, `WithBreakpoints` and
  `WithoutResponsive`. The old names are deprecated wrappers.
* **`workspace`:** `WithoutMouse()`. `WithMouse(on)` is deprecated. The
  rule, now in `AGENTS.md`:
  * `On…` names a callback;
  * `With…` sets a value or a provider;
  * `Without…` turns off a default.
* **`termsvc`:** `WithNotifyFilter(func(Notification) bool)`. `WithGate`
  is deprecated, because "gate" is `command`'s permission concept.
* **Uses.** The library, its tests and the guides move to the new names.
* **W3's deprecated variables are removed** (added 2026-10-09, by
  0014-MADR-native-integration-api A1.5): `termcaptest.RunTimeout` and
  `workspace.KeyFocusedPane`, `KeyZoomed`, `KeyHiddenPanes`,
  `KeyOverlay`, `KeyModal`, `KeyWidth` and `KeyHeight`, deprecated in
  `v0.8.0`. `runTimeout` reads the terminal's timeout, else
  `DefaultRunTimeout`. `workspace/contextkeys_test.go`, which reassigns
  the variables, is removed with them; `TestKeyFunctions` stays. The
  eight `apidiff` lines go into `scripts/apicheck.allow` in this step's
  commit, and the release notes name the removals.
* **Test:** `TestDeprecatedOptionsStillWork`, one case per old name.

### Step 5: opaque options

* **The six types** become `type X interface{ apply(*T) }`, each with an
  unexported adapter `optionFunc`:
  * `command.Option`, `RegistryOption`;
  * `termcap.Option`;
  * `termsvc.NotifyOption`;
  * `workspace.Option`, `WrapOption[M]`.
* **Every constructor** returns the adapter. The type names stay.
* **Breaking.** A third party's own `func(*T)` value no longer converts.
  The six `apidiff` lines go into `scripts/apicheck.allow`, and the
  release notes name the change.
* **Test:** `TestOptionsAreOpaque`, in a compile test. A `func(*T)`
  literal assigned to each type fails to compile. It is checked with
  `go/types` on a planted source, as conformance plants its sources.

### Step 6: `internal/enum` everywhere, and `internal/teamsg`

* **`internal/enum`.** It grows the type
  `Names[T ~uint8 | ~int]{Type string; Tokens []string}`, with
  `String`, `Marshal` and `Unmarshal`. Out of range, `String` gives the
  stringer form `Type(N)`.
  * termcap's `"9"` test (`termcap/termcap_test.go:63`) changes to
    `"Mux(9)"`, a behaviour change named in the notes.
  * The functions from 0013's PLAN stay as wrappers.
* **It serves:**
  * `termcap`: its seven enums;
  * `command`: `Kind`, `Danger` (0 is `undeclared`), `Mode`,
    `SourceKind`, `Origin`, `Verdict` (whose tokens are its ACP kinds);
  * `termsvc`: `SkipReason`, `ActivityState`, `Urgency`, `Protocol`,
    `Policy`, `Status`, `Route`, `Display`;
  * `layout`: `SizeKind`, `Axis`, `Span`;
  * `workspace`: `Chrome`, `AnchorKind`;
  * `when`: `Kind`;
  * and `theme`'s two from W2.

  2026-10-09: `termcap`'s seven, `command.Format`, `glyph.Tier`,
  `launch.Choice` and `Target`, and `theme`'s two already go through
  `internal/enum`, by W2; they move to `Names` with the rest.
  `FuzzEnumText` (W3) keeps pinning termcap's exact names.

  Each enum gets `String`, `MarshalText` and `UnmarshalText`; an existing
  `String` keeps its tokens.
* **`command.Surface`,** a bit set, gets a `SurfaceNames` helper of its
  own: `"key|palette"`.
* **`internal/teamsg.Cmd[M any](m M) tea.Cmd`** replaces the three
  helpers and the two inline forms.
* **Tests:**
  * `TestEveryEnumRoundTrips`, generated over every enum the collision
    check's scan finds with an integer underlying type;
  * `TestOutOfRangeText`.
* **Mutation S6-1:** an enum skips `UnmarshalText`.
  `TestEveryEnumRoundTrips`.
* **2026-10-10, D2:** `Names` carries a `Pkg` field as well, so the
  errors keep their texts; the twelve enums with no text yet take the
  tokens D2 lists; `Surface`'s helper is an `enum.Bits[T]` type beside
  `Names`; and `teamsg.Cmd` replaces a sixth site,
  `termcap/prober.go:547`.

### Step 7: JSON v2, errors, adapters, width, wording

* **JSON.**
  * `command`'s v1 imports go: `RawMessage` is `jsontext.Value` under its
    v2 name. (2026-10-09: in four files, `args.go` among them, whose
    `ArgsOf` returns `json.RawMessage`; the type is the same, so its
    signature does not change.)
  * `termcap`'s `Caps` and report marshal with v2.
  * The measured differences are named in the notes: `<>&` are no longer
    escaped; nil slices become `[]`; names match case-sensitively;
    duplicate keys and invalid UTF-8 are refused (W3 already sanitizes the
    strings).
  * `TestCapsJSONStable` pins the report's JSON as a new golden before
    the switch.
* **`when.SyntaxError{Offset int; Msg string; Err error}`,** with `Error`
  and `Unwrap`, comes from `scanner.errorf`.
  * The two size errors give `Offset: -1`.
  * `parse.go:163` wraps the regex error with `Err`.
  * `TestSyntaxErrorOffsets` covers every site.
* **`termcap.ErrUnknownName`** is wrapped by every text unmarshaler.
* **The `…Func` adapters:** `command.AuditorFunc`, `termsvc.BackendFunc`
  and `termsvc.ClipboardFunc`, modelled on `HandlerFunc`.
* **Width.** `AGENTS.md` and the glossary state one rule:
  * the workspace measures with its method, WcWidth until the terminal
    reports mode 2027 and GraphemeWidth after (2026-10-09: and cuts with
    `sanitize.Truncate`, never wider than asked, closing a line's ends
    under GraphemeWidth, by 0014-PLAN-hardening D8);
  * tests measure with `tuitest.Fits` under the case's method;
  * `termsvc` and `termcap` truncate by grapheme.

  The code is unchanged; only the documentation is unified.
* **The `--args` wording** is fixed in `command/slash.go` and the guide.
* **2026-10-10, D3:** v2 refuses invalid UTF-8 when writing as well, and
  `ParseTmux` does not sanitize: `ParseTmux` cleans its fields, and
  `termcap` writes its JSON allowing invalid UTF-8, as U+FFFD, while it
  reads strictly. `ErrUnknownName` is also wrapped by `Caps`'s unknown
  colour profile.

### Step 8: the release `v0.9.0`

1. **The documents.** The guides, `docs/architecture.md` and `README.md`
   use the new names, and give a "Migrating to `v0.9.0`" table of old to
   new names.
2. **The tag.** The owner commits, has the agent run the disclosure
   guard, pushes, and with CI green tags `v0.9.0`.
3. **The smoke test** builds a program on the old names, which warns
   under `staticcheck`'s SA1019 but compiles, and one on the new names.
4. **The notes** list every deprecation, the opaque options, and the
   JSON and enum-text behaviour changes.
5. **`scripts/apicheck.allow`** is emptied in the close-out commit. The
   base is now `v0.9.0`, so the entries are stale.

### Step 9: the removals (`v0.10.0`)

* **What goes:**
  * `command.Decision` and `Record.Decision`;
  * `Request.Context` and `Invocation.Context`;
  * `termcap.New`;
  * `layout`'s bare option names;
  * `workspace.WithMouse`;
  * `termsvc.WithGate`;
  * ~~`termcaptest.RunTimeout`;~~
  * ~~the seven `workspace.Key*` variables, deprecated by W3.~~
    (2026-10-09: both removed in Step 4, in `v0.9.0`, by
    0014-MADR-native-integration-api A1.5.)
* **The audit's slog key** becomes `"verdict"`.
* **The collision check's `Decision` entry,** dated for `v0.10.0` by
  0013's PLAN, is removed. The check must then pass with `Decision` in
  `launch` alone.
* **`scripts/apicheck.allow`** lists each removal under this step.
* **Mutation S9-1:** a scratch copy keeps `command.Decision`. The
  collision check fails once its entry is gone.

### Step 10: the release `v0.10.0`, and close-out

1. **The tag:** commit, the disclosure guard, push, CI, then `v0.10.0`.
2. **The smoke test** builds the new names, and the old-name program now
   fails to compile, as expected.
3. **The notes** list every removal.
4. **`scripts/apicheck.allow`** is emptied.
5. **This PLAN `complete`,** and 0014-MADR's W0–W4 noted as done in its
   index row.

## Verification

* **Every rename** has its new name in `v0.9.0`, its old name working
  through `v0.9.x` (Rule 3), and the old name gone in `v0.10.0`.
* **The six option types are opaque,** and the break is listed.
* **Every exported enum** round-trips its text, and `Surface` has its own
  helper.
* **`command` and `termcap` use JSON v2 only,** with the report's JSON
  pinned.
* **`when.SyntaxError`, `termcap.ErrUnknownName` and the `…Func`
  adapters** exist.
* **One width rule is documented.**
* **The glossary and 0007–0009** agree, and the collision check passes
  with no temporary entry.
* **`v0.9.0` and `v0.10.0` are tagged,** with smoke tests, and
  `make apicheck` showed only the listed lines.
* **Rule 2's checks are clean** on macOS and the Windows test host.

## Rollout and Rollback

* **Rollout:**
  * `v0.9.0` adds and deprecates;
  * `v0.10.0`, at least one release later, removes.
  * pi-go and gobble move to the new names during `v0.9.x`, which the
    smoke tests model.
* **Rollback:**
  * **Before a tag,** revert the step.
  * **After `v0.9.0`,** a rename that proves wrong is reversed in `v0.9.1`
    by un-deprecating the old name.
  * **After `v0.10.0`,** a removed name returns only by a record.

## Execution Record

### Step 1: records

* **Facts re-read (2026-10-09).** The agent read every fact of this PLAN
  again on `b6e3cef`, after W2, W3, 0016 and `v0.8.0`. The results are in
  "Facts re-read before execution (2026-10-09)", with the amended lines
  in "Out of scope" and Steps 2, 3, 4, 6, 7 and 9.
  * Every fact the steps rest on still holds, at new lines.
  * W2 and W3 changed five: `internal/enum` already serves twelve enums;
    `command` imports JSON v1 in four files; the workspace's width is
    W3's `sanitize.Truncate`; the collision check shares six names; and
    the glossary still reserves six names that are now built.
  * W3's deprecated variables are removed in `v0.9.0`, in Step 4, as
    0014-MADR-native-integration-api A1.5 says, not in `v0.10.0`.
* **No other record changes.** The MADR already decides each point.
* **The approval:** the owner wrote "approved to proceed" on 2026-10-09,
  before this re-read, and "committed. approved to proceed" after it
  (`b45e0ee`).
* **This PLAN** is `in-progress`, and its row in `docs/README.md` follows.

### Step 2: the glossary and the planned records

#### Deviations

* **D1 (2026-10-09): four of Step 2's names, chosen again.**
  * **Found,** checking every type the three records plan against the
    library's exported types on `b45e0ee` and against each other:
    * `keymap.Context` → `Scope` would collide with `command.Scope`,
      which means the same thing: where a command applies, `Global` or a
      pane, overlay or mode name.
    * `keymap.Options` → `Config` would collide with `launch.Config`,
      built by 0013 after this PLAN's names were chosen.
    * `keymap.Match` → `Hit` would collide with 0008's `palette.Hit`, and
      `Match` itself with 0008's `fuzzy.Match`.
    * 0008's `palette.Palette` would collide with `theme.Palette`; Step 2
      had not listed it.
  * **The owner's choices:** `keymap` uses `command.Scope` and
    `command.Global` itself (the PLAN's "or `command.Scope` itself");
    `Settings`; `Resolved`; and `palette.Picker`, built by `NewPicker`.
  * **The others:** a `keymap.Region` type; `BuildOptions`; `Found`; and
    `Palette` as a deliberate shared name.
  * Every other new name is free: none is an exported type in the
    library, and none is planned twice.

#### What was done

* **`docs/glossary.md`:**
  * the `when` row of "Deliberate" sits under `Context`, where it
    belongs; it had been listed under `Decision`;
  * `Format`, `Param`, `PanicError`, `PlainViewer`, `GlyphThemeBuilder`
    and `LoadOption` move from "Reserved" to "Built", since W2 and W3
    built them;
  * "One meaning each" gains Step 2's names: `Verdict`, `WhenContext`,
    `Decision`, `Policy`, `Plan`, `Rule`, `Span`, `Result`, `Conflict`,
    `Scope`, `Source`, `Query`, `Tier`, `Filter`, `Renderer` and `New`;
  * "Names the planned records must not take" records that the three
    amendments renamed every collision, and that `Renderer` in `stream`
    and `stream/glamourmd` is the one planned deliberate pair.
* **0007-MADR-keymap-engine A3, 0008-MADR-command-palette A2 and
  0009-MADR-streaming-content-engine A4,** each dated, citing this PLAN,
  and changing names only. Their PLANs take the names when they are
  next amended, before they run.
* **`docs/README.md`:** each record's row says its amendment is recorded,
  and this PLAN's row says `in-progress`.

### Step 3: `command`'s renames

#### Deviations

No deviation.

#### What was built

* **`command/gate.go`:**
  * `type Verdict uint8` holds `AllowOnce`, `AllowAlways`, `RejectOnce`
    and `RejectAlways`, now typed `Verdict`, and `ACPKind`;
  * `type Decision = Verdict`, marked `// Deprecated: use Verdict
    (0014-MADR W4).`;
  * `Gate.Decide` and `GateFunc` return `Verdict`; the alias keeps a gate
    written against `Decision` satisfying `Gate`;
  * the unexported `verdict` function is `settle`.
* **`command/audit.go`:** `Record.Verdict` is added, and
  `Record.Decision`, now of type `Verdict`, is deprecated. The registry
  sets both. The slog key stays `"decision"`; `SlogAuditor` reads
  `Verdict`, else `Decision`, so a `Record` a program builds with the old
  field alone still logs its answer.
* **`command/handler.go`:** `Request.WhenContext` and
  `Invocation.WhenContext` are added beside the deprecated `Context`
  fields.
* **`command/dispatch.go`:** `admit` reads `WhenContext`, else the
  deprecated `Context`, and builds the `Invocation` with both set to the
  same value.
* **`command/builtin.go`:** the list command reads `WhenContext`.
* **Uses:** `command`'s and `workspace`'s tests, and the commands guide's
  example, move to the new names. `docs/architecture.md` moves in
  Step 8, with the other documents.
* **`command/rename_test.go` (new),** in package `command`, so that
  staticcheck does not report the deprecated names it uses on purpose:
  * `TestOldDecisionStillWorks`: a `Gate` whose `Decide` returns
    `Decision`, for each of the four answers; a `GateFunc` literal
    returning `Decision`; and a function from `Decision` to `Verdict` that
    compiles only while they are one type;
  * `TestWhenContextFallsBack`: `Context` alone, `WhenContext` alone, both
    (`WhenContext` wins, even empty), and neither; the handler sees both
    fields, set;
  * `TestRecordCarriesBoth`: a granted request's record has `AllowAlways`
    in both fields, an unknown command's neither, and `SlogAuditor`
    writes `decision=reject_once` for a `Record` with either field.

#### Checks

* **Mutations,** on scratch copies of the tree; every one was killed by a
  failing test, none by a build failure:
  * **S3-1** (the PLAN's: `admit` ignores the old field): "Context
    alone, the old field: ran false … want ran true";
  * **S3-2** (`admit` reads the old field first, added): "both:
    WhenContext wins: ran false";
  * **S3-3** (the `Invocation` leaves the old field unset, added): "the
    Invocation's WhenContext map[on:true] and Context <nil>";
  * **S3-4** (the audit sets `Verdict` only, added): "Verdict 2,
    Decision 0";
  * **S3-5** (`SlogAuditor` ignores the old field, added): the log line
    for a `Record` with `Decision` alone holds no `reject_once`.
* **`apidiff`:** `make apicheck`, "against v0.8.0, 0 incompatible
  change(s)", so `scripts/apicheck.allow` gains nothing: `Decision`
  turning into an alias of `Verdict` is compatible, as this PLAN's probe
  found.
* **Rule 2 on macOS,** go1.27.2:
  * `make pre-add-check FILES=<the 15 Go files>`: "15 file(s) clean".
  * `make lint`: clean. On the first run, staticcheck's ST1023 and
    unconvert refused a typed `var` and a conversion in
    `TestOldDecisionStillWorks`; the function from `Decision` to
    `Verdict` replaced them.
  * The cross `go vet` for `freebsd/amd64`, `openbsd/amd64` and
    `linux/386`; `-race`, `-shuffle=on -count=2`, `LC_ALL=C` and workspace
    mode; `go mod tidy -diff`, `make vuln`, `make examples`,
    `scripts/go-modules.sh --check` and `make release-check`: clean.
  * CI's other checks: `shellcheck`, `markdownlint-cli2`, `actionlint`
    and `go-precheck_test.sh` ("12 passed"): clean.
* **The Windows test host,** go1.27.2 windows/amd64:
  * a first run, started before the lint fix, was stopped, and its files
    were removed from the host;
  * on the final tree: `make pre-add-check`, `make lint`, `make vuln` and
    `make examples`, each exit 0; `go test -count=2 -shuffle=on ./...`,
    exit 0; the step's tests, with `command`'s gate, audit and exit-code
    tests (the same pattern selects those seven on macOS, the three new among
    them), 7 passed, 0 failed; `go-fuzz_test.sh`, "25 passed, 0 failed"; `make fuzz
    FUZZTIME=3s`, "8 packages ran clean".

### Step 4: the other renames, and W3's deprecated variables

#### Deviations

No deviation. The removal of W3's variables is the PLAN as amended in
Step 1.

#### What was built

* **`termcap`:** `NewProber`; `New` calls it, marked `// Deprecated: use
  NewProber (0014-MADR W4).` `EnvCaps`, the tests, `launch`'s consumer
  test and the terminal-capabilities guide call `NewProber`.
* **`layout`:** `WithSidebarWidth`, `WithBottomHeight`, `WithMainSize`,
  `WithBottomSpan`, `WithFooter`, `WithGap`, `WithBreakpoints` and
  `WithoutResponsive`; the eight bare names are deprecated wrappers.
  `layout`'s and `workspace`'s tests, `workspace.go`'s comment and the
  building-workspaces guide use the new names.
* **`workspace`:** `WithoutMouse()`; `WithMouse(on)` is deprecated, since
  handling is on by default.
* **`termsvc`:** `WithNotifyFilter`; `WithGate` is a deprecated wrapper,
  since a gate is `command`'s permission concept.
* **`AGENTS.md`,** "API conventions" rule 4: `On…` names a callback,
  `With…` sets a value or a provider, and `Without…` turns off a default.
* **W3's deprecated variables, removed** (0014-MADR-native-integration-api
  A1.5): `termcaptest.RunTimeout`, so `runTimeout` is the terminal's
  timeout, else `DefaultRunTimeout`, and `TestSetRunTimeout`'s table and
  a test's deadline lose it; and the seven `workspace.Key…` variables,
  with `workspace/contextkeys_test.go`, which reassigned them.
  `TestKeyFunctions` stays.
* **`scripts/apicheck.allow`** lists the eight removals, under a header
  naming this PLAN and step.
* **`TestDeprecatedOptionsStillWork`,** in `layout`, `termcap`,
  `workspace` and `termsvc`, each in its package so that staticcheck does
  not report the deprecated names it uses on purpose: each old name does
  what its new one does. `layout`'s solves the preset with each old
  option and its new one, at a width where the option changes the plan,
  and fails if it does not.

#### Checks

* **Mutations,** on scratch copies of the tree; every one was killed by a
  failing test, none by a build failure:
  * **S4-1** (`Gap` drops its argument): "Gap: the old name solves to …";
  * **S4-2** (`NoResponsive` does nothing): "NoResponsive: the old name
    solves to …";
  * **S4-3** (`termcap.New` ignores its options): "New with options:
    timeout 2s";
  * **S4-4** (`WithMouse` ignores `on`): "WithMouse(true) after
    WithoutMouse: mouse false";
  * **S4-5** (`WithoutMouse` does nothing): "WithoutMouse: mouse true";
  * **S4-6** (`WithGate` sets no filter): "the filter is not set";
  * **S4-7** (`runTimeout` ignores the terminal's): "Run with a 50ms
    timeout took 10.001445292s".

  `layout`'s test first failed on its own check for `NoResponsive` at
  120 columns, where the default folds nothing; its case runs at 80.
* **`apidiff`:** `make apicheck`, "against v0.8.0, 8 incompatible
  change(s)", each the listed removal, and "clean". The new names are
  additions.
* **Rule 2 on macOS,** go1.27.2:
  * `make pre-add-check FILES=<the 32 Go files>`: "32 file(s) clean".
  * `make lint`, the cross `go vet`, `-race`, `-shuffle=on -count=2`,
    `LC_ALL=C`, workspace mode, `go mod tidy -diff`, `make vuln`, `make
    examples`, `scripts/go-modules.sh --check` and `make release-check`:
    clean.
  * CI's other checks: `shellcheck`, `markdownlint-cli2`, `actionlint`
    and `go-precheck_test.sh` ("12 passed"): clean.
* **The Windows test host,** go1.27.2 windows/amd64: `make pre-add-check`,
  `make lint`, `make vuln` and `make examples`, each exit 0; `go test
  -count=2 -shuffle=on ./...`, exit 0; the step's tests, with the context
  keys', the presets' and the profiles' tests, 10 passed, 0 failed; `make fuzz
  FUZZTIME=3s`, "8 packages ran clean".
  * `go-fuzz_test.sh` exited 123 there, and passed on macOS. The agent's
    Windows harness builds the index from the last commit, which still
    holds `workspace/contextkeys_test.go`, while the copied tree does not:
    case 6's `git ls-files` listed it, and `grep` failed on it ("No such
    file or directory"). Run again with the index from the copied tree
    (`git add -A`), it gave "25 passed, 0 failed". The harness syncs the
    index from now on.
  * **Noted, not changed:** case 6 aborts with no FAIL line when a
    tracked test file is deleted and not yet staged. It is
    0014-PLAN-hardening's code; the owner decides whether it is fixed.

### Step 5: opaque options

#### Deviations

No deviation.

#### What was built

* **The six types,** each `type X interface{ apply(*T) }` with an
  unexported function adapter whose `apply` calls it:
  * `command.Option` (`optionFunc`) and `RegistryOption`
    (`registryOptionFunc`);
  * `termcap.Option` (`optionFunc`);
  * `termsvc.NotifyOption` (`notifyOptionFunc`);
  * `workspace.Option` (`optionFunc`) and `WrapOption[M]`
    (`wrapOptionFunc[M]`).

  The type names stay. Each doc comment says the type is opaque.
* **Every constructor** returns the adapter: 61 functions, each one's
  `return func(…) {…}` wrapped by brace matching, with no other change.
  `termsvc.WithGate`, which returns `WithNotifyFilter`'s value, needed
  none. The six loops that apply options call `apply`.
* **`scripts/apicheck.allow`** lists the six type changes, under a header
  naming this PLAN and step; the release notes name the break.
* **`AGENTS.md`,** "API conventions" rule 4: option types are opaque, an
  interface with an unexported method; a function over an unexported
  struct is opaque already.
* **`internal/conformance/options_test.go` (new):** `TestOptionsAreOpaque`
  plants sources and type-checks them as the scan does, with the source
  importer. For each type, the library's own option type-checks, so a
  failing import cannot pass for opacity; and two programs' attempts are
  refused as not implementing the type: a `func(*T)` literal, and a type
  of the program's own with both an `apply` and an `Apply` method.
* **No test, guide or example** wrote an option literal of its own: the
  library's tests, `make examples` and the guides needed no change.

#### Checks

* **The probe:** `TestOptionsAreOpaque`, on a scratch copy of `ae6e76c`,
  the commit before this step: 12 failures, both attempts for each of
  the six types. It passes on the step's code.
* **Mutations,** on scratch copies of the tree, each reopening one type
  while the tree still builds; both were killed by the test:
  * **S5-1** (`workspace.Option`'s method exported as `Apply`): "a type
    of its own: <nil>". Only the second attempt catches it: a function
    literal has no `Apply` method either.
  * **S5-2** (`termsvc.NotifyOption` an alias of its function adapter):
    "a func literal: <nil>".
* **`apidiff`:** `make apicheck`, "against v0.8.0, 14 incompatible
  change(s)", the eight of Step 4 and the six listed here, and "clean".
* **Rule 2 on macOS,** go1.27.2:
  * `make pre-add-check FILES=<the 7 Go files>`: "7 file(s) clean".
  * `make lint`, the cross `go vet`, `-race`, `-shuffle=on -count=2`,
    `LC_ALL=C`, workspace mode, `go mod tidy -diff`, `make vuln`, `make
    examples`, `scripts/go-modules.sh --check` and `make release-check`
    ("197 file(s) clean"): clean.
  * CI's other checks: `shellcheck`, `markdownlint-cli2`, `actionlint`,
    `go-precheck_test.sh` ("12 passed") and `go-fuzz_test.sh` ("25
    passed"): clean.
* **The Windows test host,** go1.27.2 windows/amd64, with the index synced
  to the copied tree: `make pre-add-check`, `make lint`, `make vuln` and
  `make examples`, each exit 0; `go test -count=2 -shuffle=on ./...`,
  exit 0; `TestOptionsAreOpaque`, the deprecated-name tests and the
  conformance scan, 7 passed, 0 failed; `go-fuzz_test.sh`, "25 passed, 0 failed";
  `make fuzz FUZZTIME=3s`, "8 packages ran clean".

### Step 6: `internal/enum` everywhere, and `internal/teamsg`

#### Deviations

* **D2 (2026-10-10): four details Step 6 left open.**
  * **Found,** reading the step against `4aedfb2`:
    * `internal/enum`'s errors name the package as well as the enum,
      "termcap: unknown Mux", and its package comment promises a package
      that moves onto it keeps its texts. `Names{Type, Tokens}` has no
      package to name.
    * Twelve enums have no text at all, so Step 6 chooses their tokens,
      which become stable API: `command.Verdict` (the step says its ACP
      kinds, but not its zero's), `termsvc`'s `Status`, `Route`,
      `Display`, `Urgency`, `Protocol` and `Policy`, `layout`'s `Axis`,
      `SizeKind` and `Span`, `workspace`'s `Chrome` and `AnchorKind`, and
      `when.Kind`.
    * The step does not say where `Surface`'s helper lives.
    * A sixth one-message command: `termcap/prober.go:547`,
      `schemeChanged`'s `ColorSchemeMsg`, inline since `45ead16`, before
      this PLAN. The fact table names five sites.
  * **The owner's choices:**
    * `Names{Pkg, Type, Tokens}`: the errors keep their texts byte for
      byte, and `String`'s out-of-range form names the type alone,
      `Mux(9)`.
    * These tokens:

      | Enum | Tokens, from 0 |
      | :--- | :--- |
      | `command.Verdict` | `""`, as `ACPKind` gives and `SkipReason`'s zero prints; `allow_once`, `allow_always`, `reject_once`, `reject_always` |
      | `termsvc.Status` | `unconfirmed`, `confirmed`, `failed` |
      | `termsvc.Route` | `none`, `backend`, `tmux-buffer`, `osc52`, `osc52-tmux` |
      | `termsvc.Display` | `label-only`, `label-and-url` |
      | `termsvc.Urgency` | `normal`, `low`, `critical` |
      | `termsvc.Protocol` | `auto`, `osc99`, `osc777`, `osc9`, `bell`, `off` |
      | `termsvc.Policy` | `when-unfocused`, `always`, `never`, `unless-focused` |
      | `layout.Axis` | `horizontal`, `vertical` |
      | `layout.SizeKind` | `fill`, `fixed`, `percent`, `ratio` |
      | `layout.Span` | `full-width`, `under-main` |
      | `workspace.Chrome` | `borders`, `separators`, `none` |
      | `workspace.AnchorKind` | `center`, `below-cursor`, `on-pane` |
      | `when.Kind` | `nothing`, `boolean`, `number`, `string`, `list`: the words `when`'s errors already use |

      Every other enum keeps the tokens its `String` prints today.
    * `Surface`'s helper is an `enum.Bits[T]` type in `internal/enum`,
      beside `Names`, rather than an unexported helper in `command`.
    * `teamsg.Cmd` replaces all six sites.
  * **Also found, needing no choice:** moving onto `Names` changes the
    out-of-range text of every enum whose `String` printed the bare
    number: `termcap`'s seven, `command.Format`, `glyph.Tier`,
    `launch.Choice` and `Target`, `theme`'s two, and `termsvc`'s
    `ActivityState` and `SkipReason`. Three tests pin the old text:
    `termcap/termcap_test.go:63`, which the step names, and
    `glyph/glyph_test.go:105` and `theme/text_test.go:35`, which it does
    not; the agent found the third when the tests ran. Each changes to the
    `Type(N)` form, and the release notes name the change.

#### What was built

* **`internal/enum`:**
  * `Names[T]{Pkg, Type, Tokens}`, with `String`, `Marshal` and
    `Unmarshal`. `String` gives `Type(N)` for a value with no token; the
    errors are the package's existing texts.
  * `Bits[T ~uint8]{Pkg, Type, Tokens, None}` (D2), for a bit set: the
    tokens of the bits that are set, lowest first and joined with `|`,
    `None` for the empty set, and bits with no token last, in hex, which
    `Marshal` refuses. `Unmarshal` takes the tokens in any order, and
    refuses an empty or unknown one.
  * `Name`, `Marshal` and `Unmarshal` stay. `Marshal` and `Unmarshal`
    wrap `Names`; `Name` keeps the bare number, since it has no type
    name. No package calls them now.
* **Every exported enum is on a table, 34 in all.**
  * The 13 that were already on `internal/enum`, termcap's three helpers
    removed with them.
  * The 21 others: `command`'s `Kind`, `Danger`, `Mode`, `SourceKind`,
    `Origin` and `Verdict` on `Names`, and `Surface` on `Bits`;
    `termsvc`'s eight; `layout`'s three; `workspace`'s two; `when.Kind`.
    Each has `String`, `MarshalText` and `UnmarshalText`, with D2's
    tokens. `Verdict.ACPKind` is `String`, and `""` for a value with no
    token, as before. `when`'s `kindName` is gone; its errors print the
    `Kind`.
  * Each package names itself once, in `const pkgName`, for its tables'
    `Pkg` (`goconst`).
* **`internal/teamsg.Cmd[M any](m M) tea.Cmd`** replaces `command`'s
  `msgCmd`, `termsvc`'s `copied` and `result`, and the three inline forms
  in `termcap/prober.go` (two) and `termsvc/links.go`. `command`'s own
  test used `msgCmd` and uses `Cmd`.
* **The pins,** D2's three tests: `Tier(9)`, `Support(9)` and
  `Background(-1)` now expect the `Type(N)` form.
* **`AGENTS.md`,** "API conventions" rule 5: the tokens come from one
  table in `internal/enum`, `Names` or `Bits`; a value with no token
  prints `Type(N)`; `internal/conformance` checks the round trip.
* **`docs/architecture.md`:** the package graph gains the imports this
  step adds (`internal/enum` in `layout`, `when`, `termsvc` and
  `workspace`; `internal/teamsg` in `command`, `termcap` and `termsvc`),
  with rows and a tree line for `internal/teamsg`. Step 8 still owns the
  renames there.
* **Tests:**
  * `internal/conformance/enums_test.go` (new). The collision check's
    scan now keeps each type's object, and the enums are what it finds:
    every exported type of a public package with an integer underlying
    type, aliases aside, with its constants read from the type checker.
    A table of conversions builds their values; a scanned enum missing
    from the table fails, and so does an entry the scan does not find.
    * `TestEveryEnumRoundTrips`: each enum has `String` and
      `MarshalText`, and `UnmarshalText` on its pointer; each constant
      and the zero value has a lowercase token of its own, which
      `String` prints, `MarshalText` writes and `UnmarshalText` reads
      back into a variable that held another value.
    * `TestOutOfRangeText`: one past the last constant, and for an `int`
      enum one below the first, prints `Type(N)`, is refused by
      `MarshalText` with the package's and the type's names, and is not
      read back. For `Surface`, `SurfaceKey|0x80` prints `key|0x80`, and
      `MarshalText` refuses `0x80`.
  * `internal/enum`: `TestNames`, `TestBits` and `TestBitsEight`.
  * `internal/teamsg`: `TestCmd`.

#### Checks

* **The probe:** the two conformance tests, on a scratch copy of
  `4aedfb2`, the commit before this step: `TestEveryEnumRoundTrips`
  failed 76 times (29 missing `MarshalText`, 26 `String`, 21
  `UnmarshalText`), and `TestOutOfRangeText` 36 times (the bare number
  from the 13 enums already on `internal/enum`, and the missing
  methods). Both pass on the step's code, with no table entry stale and
  none missing: the scan found the 34 enums.
* **Mutations,** on scratch copies of the tree, each one building; all
  eight killed by an assertion:
  * **S6-1** (`termsvc.Status` skips `UnmarshalText`):
    "termsvc.Status has no UnmarshalText method".
  * **S6-2** (`layout.Axis`'s `UnmarshalText` does nothing):
    `UnmarshalText("horizontal") = vertical`.
  * **S6-3** (two `termsvc.Route` values share `osc52`): "3 and 4 share
    the token".
  * **S6-4** (`workspace.AnchorKind`'s table says `Anchor`):
    `"Anchor(3)", want "AnchorKind(3)"`.
  * **S6-5** (`Names.String` prints the bare number): `"5", want
    "Danger(5)"`.
  * **S6-6** (a `Danger` token in capitals): "want one lowercase token".
  * **S6-7** (`Bits.Unmarshal` skips an empty token): `TestBits`,
    `Unmarshal(""): <nil>`.
  * **S6-8** (`Bits.Marshal` writes unknown bits): `TestOutOfRangeText`,
    `command.Surface(0x81).MarshalText: <nil>`.
* **`apidiff`:** `make apicheck`, "against v0.8.0, 14 incompatible
  change(s)", Steps 4 and 5's, and "clean": the new methods are
  compatible, and `scripts/apicheck.allow` gains nothing.
* **Rule 2 on macOS,** go1.27.2:
  * `make pre-add-check FILES=<the 35 Go files>`: "35 file(s) clean".
  * `make lint`, the cross `go vet`, `-race`, `-shuffle=on -count=2`,
    `LC_ALL=C`, workspace mode, `go mod tidy -diff`, `make vuln`, `make
    examples`, `scripts/go-modules.sh --check` and `make release-check`
    ("198 file(s) clean"): clean.
  * CI's other checks: `shellcheck`, `markdownlint-cli2`, `actionlint`,
    `go-precheck_test.sh` ("12 passed") and `go-fuzz_test.sh` ("25
    passed"): clean.
* **The Windows test host,** go1.27.2 windows/amd64, with the index synced
  to the copied tree: `make pre-add-check`, `make lint`, `make vuln` and
  `make examples`, each exit 0; `go test -count=2 -shuffle=on ./...`,
  exit 0; the step's tests and the changed pins, 13 passed, 0 failed;
  `go-fuzz_test.sh`, "25 passed, 0 failed"; `make fuzz FUZZTIME=3s`, "8
  packages ran clean".
* **For Step 8's notes,** the behaviour changes: the 21 enums' new
  `MarshalText` turns their JSON from a number into the token, in a
  program's own values and in `command.WriteResult`'s; and the
  out-of-range text D2 lists.

### Step 7: JSON v2, errors, adapters, width, wording

#### Deviations

* **D3 (2026-10-10): v2 writes no invalid UTF-8, and one more unknown
  name.**
  * **Found,** on a scratch copy with `termcap` on v2:
    * v2 refuses invalid UTF-8 when it writes, not only when it reads.
      `Caps{Terminal: "wez\xffterm"}` and a `Caps` holding
      `ParseTmux("3.4\xff…")` make `Caps.MarshalJSON` fail ("invalid
      UTF-8 within \"/tmux/version\""), and so `Report`'s JSON, where v1
      wrote U+FFFD.
    * The step's "W3 already sanitizes the strings" holds for the
      environment and the terminal's replies, but not for `ParseTmux`
      (`termcap/tmux.go:50-54`), whose input is the output of a tmux the
      program runs; nor for a program's own `Caps` values and
      `WithOverride`, which no sanitizer sees. The first is a gap in
      0014-MADR C6, "one `internal/sanitize` for every untrusted
      string".
    * `Caps.UnmarshalJSON` reads one name besides the seven enums': the
      colour profile, refused as "termcap: unknown colour profile"
      (`termcap/caps.go:135`).
  * **The owner's choices:**
    * `ParseTmux` cleans each field with `sanitize.Line`, as every other
      source does; and `Caps.MarshalJSON` and `Report` write with
      `jsontext.AllowInvalidUTF8(true)`, so a report never fails on a
      value's bytes and writes U+FFFD as v1 did. Reading stays strict.
    * The colour profile's error wraps `ErrUnknownName` as well, its
      text unchanged.
  * The MADR's decisions stand, so it is not amended.

#### What was built

* **JSON.**
  * `command`: `args.go`, `audit.go`, `command.go` and `handler.go`
    import `encoding/json/jsontext` for `jsontext.Value`, in place of v1's
    `json.RawMessage`. In Go 1.27 the v1 name is an alias of it, so no
    signature changes, and `apidiff` reports none.
  * `termcap`: `Caps.MarshalJSON` and `UnmarshalJSON` use v2, and
    `Report` writes with `jsontext.WithIndent("  ")`. `Report` needs no
    `AllowInvalidUTF8` of its own: its caps come through
    `Caps.MarshalJSON`, and its findings are the package's own text, so
    the option there could never take effect.
  * The tests of both packages moved to v2 as well, so neither package
    imports v1. `layout/state.go` keeps v1, outside this PLAN's scope.
  * D3: `ParseTmux` cleans each field with `sanitize.Line`, and
    `Caps.MarshalJSON` writes invalid UTF-8 as U+FFFD.
* **`when.SyntaxError{Offset, Msg, Err}`** (`when/error.go`), with `Error`
  and `Unwrap`. `scanner.errorf` builds it, so its 18 sites give it; the
  regex site sets `Err` to `regexp`'s error, under `Msg` "a bad regex";
  the two size limits give `Offset: -1`. Every text is the one before,
  byte for byte. `Parse`'s comment says its error is a `*SyntaxError`.
* **`termcap.ErrUnknownName`,** "termcap: unknown name". `enum.Names`
  gains an `Unknown` sentinel, which `Unmarshal`'s error wraps, and
  `enum.Errorf` makes an error with a text of its own that wraps a
  sentinel, so no text changes. The seven enums' tables set it, and
  `Caps`'s unknown colour profile uses `enum.Errorf` (D3).
* **The adapters:** `command.AuditorFunc`, `termsvc.BackendFunc` and
  `termsvc.ClipboardFunc`, each a function type whose method calls it,
  as `HandlerFunc`. No exported type of these names existed, and the
  glossary lists none.
* **Width:** `AGENTS.md`'s "TUI conventions" rule 5, and a "Width: one
  rule" section in `docs/glossary.md`, state the rule: the workspace's
  method, `WcWidth` until mode 2027 and `GraphemeWidth` after, cutting
  with `sanitize.Truncate`; `tuitest.Fits` under the case's method in
  tests; grapheme clusters, with `ansi.Truncate`, in `termsvc` and
  `termcap`'s report. The agent read each claim against the code
  (`tuitest/case.go:28`, `termsvc/termsvc.go:104`,
  `termcap/report.go:215-226`). No code changed.
* **The `--args` wording:** `command/slash.go` and
  `docs/guides/commands.md` name the program's own command line
  (`ParseArgs`), which replaced `--args`.
* **Tests:**
  * `termcap/json_test.go` (new):
    * `TestCapsJSONStable`, the report's JSON as two golden files,
      `full()` with `<>&` in its terminal's name and the zero `Caps`,
      written on v1 before the switch;
    * `TestCapsJSONReadsStrictly`: a duplicate name and invalid UTF-8
      are refused, and a name in another case is not read;
    * `TestErrUnknownName`: the seven enums and two JSON inputs wrap it,
      with their texts; a bad colour does not;
    * `TestJSONWritesInvalidUTF8`: U+FFFD from `MarshalJSON` and
      `Report`; `ParseTmux` drops escapes, invalid UTF-8, a bidi control
      and BEL.
  * `when`: `TestSyntaxErrorOffsets`, 22 sources over the 21 sites
    (nesting both ways), each's offset, `Msg`, nil `Err` and text; and
    the regex's wrapped `*syntax.Error`.
  * `command`: `TestAuditorFunc`, through a registry. `termsvc`:
    `TestBackendAndClipboardFuncs`, through `Notify` and `Copy`, the
    clipboard's error reported as `Failed`.

#### Checks

* **The measured difference:** the golden written on v1 and the one on
  v2 differ in one line, `full()`'s terminal: v1 wrote `"WezTerm
  \u003c20240203\u003e \u0026 co"`, v2 `"WezTerm <20240203> & co"`.
  The zero report is the same. No field `termcap` writes is a nil slice
  without `omitzero`, so v2's `[]` for a nil slice changes nothing here.
* **The probe,** on a scratch copy of `1d6b438`, the commit before this
  step, with the new tests that compile there:
  * `TestCapsJSONReadsStrictly`: 3 failures; v1 read the duplicate
    (the last value won), turned `\xff` into U+FFFD, and read
    `"Complete"` and `"MUX"`.
  * `TestJSONWritesInvalidUTF8`: 1 failure, `ParseTmux` keeping `\xff`,
    the escape, U+202E and BEL.
  * `TestErrUnknownName`, `TestSyntaxErrorOffsets`, `TestAuditorFunc`
    and `TestBackendAndClipboardFuncs` use names `1d6b438` lacks; the
    mutations below show each failing.
* **Mutations,** on scratch copies of the tree, each one building; all
  ten killed by an assertion:
  * **S7-1** (`Names.Unmarshal` wraps no sentinel) and **S7-2** (the
    colour profile's neither): `TestErrUnknownName`, "want … wrapping
    ErrUnknownName".
  * **S7-3** (`errorf` gives offset 0): `"a & b" = 0 … want 2`.
  * **S7-4** (the regex error flattened into `Msg`): "a bad regex:
    &when.SyntaxError{… Err:error(nil)}".
  * **S7-5** (the source limit gives offset 0): `= 0 … want -1`.
  * **S7-6** (`MarshalJSON` refuses invalid UTF-8): "invalid UTF-8
    within \"/terminal/value\"".
  * **S7-7** (`ParseTmux` keeps the raw field): "ParseTmux kept
    untrusted bytes".
  * **S7-8** (`AuditorFunc.Audit` drops the record): "records []".
  * **S7-9** (`BackendFunc.Notify` does not call the function): "the
    backend was sent []".
  * **S7-10** (`ClipboardFunc.Copy` hides the error): "Copy reported
    [{Status:confirmed …}]".
* **`apidiff`:** `make apicheck`, "against v0.8.0, 14 incompatible
  change(s)", Steps 4 and 5's, and "clean": the new names are additions,
  and `jsontext.Value` is the type `json.RawMessage` names.
* **Rule 2 on macOS,** go1.27.2:
  * `make pre-add-check FILES=<the 26 Go files>`: "26 file(s) clean".
  * `make lint`, the cross `go vet`, `-race`, `-shuffle=on -count=2`,
    `LC_ALL=C`, workspace mode, `go mod tidy -diff`, `make vuln`, `make
    examples`, `scripts/go-modules.sh --check` and `make release-check`
    ("201 file(s) clean"): clean.
  * CI's other checks: `shellcheck`, `markdownlint-cli2`, `actionlint`,
    `go-precheck_test.sh` ("12 passed") and `go-fuzz_test.sh` ("25
    passed"): clean.
* **The Windows test host,** go1.27.2 windows/amd64, with the index synced
  to the copied tree: `make pre-add-check`, `make lint`, `make vuln` and
  `make examples`, each exit 0; `go test -count=2 -shuffle=on ./...`,
  exit 0; the step's tests, 12 passed, 0 failed; `go-fuzz_test.sh`, "25
  passed, 0 failed"; `make fuzz FUZZTIME=3s`, "8 packages ran clean".
* **For Step 8's notes:**
  * `termcap`'s JSON no longer escapes `<>&`, and reads strictly:
    duplicate names and invalid UTF-8 are errors, and names match in
    their own case only; a program's own invalid UTF-8 is written as
    U+FFFD.
  * `ParseTmux` cleans its fields.
  * `when.Parse`'s errors are `*SyntaxError`, with the same texts.
  * `termcap`'s unknown-name errors wrap `ErrUnknownName`.
  * The three `…Func` adapters are new.

### Step 8: the release `v0.9.0`

#### Deviations

No deviation. Where the step left a choice, the agent chose:

* **The migration table** is a new guide, `docs/guides/migrating.md`,
  with a section per release, so `v0.10.0`'s removals join it; `README.md`
  and `docs/README.md` link it.
* **`docs/guides/terminal-capabilities.md`** still said `WithGate(f)`,
  which Step 4's pass over the guides missed; it says
  `WithNotifyFilter(f)`.

#### The documents

* **`docs/guides/migrating.md` (new),** "Migrating to `v0.9.0`": the
  renamed names, old to new, through `v0.9.x`; the removed names and what
  to use; what changed (opaque options, enum text, `termcap`'s JSON v2,
  `ParseTmux`, `SyntaxError`, `ErrUnknownName`, `jsontext.Value`); and the
  additions.
* **`README.md`:** "The current release is `v0.9.0`", with the
  deprecation window and a link to the migration guide; a bullet, "One
  name for each idea, since `v0.9.0`"; an "I want to…" row.
* **`docs/README.md`:** an "I want to…" row for the migration guide.
* **`docs/architecture.md`:** `command`'s row says `Verdict`,
  `AuditorFunc` and `WhenContext`, where it said `Decision`, as Step 3
  left for this step; `when`'s, `SyntaxError`; `termcap`'s, `NewProber`,
  JSON v2 and `ErrUnknownName`; `termsvc`'s, `WithNotifyFilter`,
  `BackendFunc` and `ClipboardFunc`; `layout`'s, the presets' `With…`
  options.
* **`docs/glossary.md`:** `Verdict` and `WhenContext` move from
  "Reserved by accepted records", now empty, to "Built from those
  records".
* **The guides:** `terminal-capabilities.md` names `WithNotifyFilter`,
  `BackendFunc` and `ClipboardFunc`; `commands.md`, `AuditorFunc`.
  `make examples` stays "clean".

#### Release notes

`v0.9.0` gives the API one name for each idea (0014-MADR W4). Go 1.27.2 is
required. Against `v0.8.0`, apidiff reports 14 incompatible changes, each
listed in `scripts/apicheck.allow`, and 75 compatible ones.

* **Deprecated,** working through `v0.9.x` and removed in `v0.10.0`:
  * `command.Decision` (use `Verdict`), `Record.Decision` (use
    `Record.Verdict`), and `Request.Context` and `Invocation.Context`
    (use `WhenContext`). `SlogAuditor` logs under the key `decision`
    until `v0.10.0`, then `verdict`.
  * `termcap.New` (use `NewProber`).
  * `layout.SidebarWidth`, `BottomHeight`, `MainSize`, `BottomSpan`,
    `Footer`, `Gap`, `Breakpoints` and `NoResponsive` (use `With…`, and
    `WithoutResponsive`).
  * `workspace.WithMouse` (use `WithoutMouse`; the mouse is handled by
    default).
  * `termsvc.WithGate` (use `WithNotifyFilter`).
* **Removed,** deprecated in `v0.8.0`: `termcaptest.RunTimeout` and the
  seven `workspace.Key…` variables.
* **Breaking: opaque option types.** `command.Option`, `RegistryOption`,
  `termcap.Option`, `termsvc.NotifyOption`, `workspace.Option` and
  `WrapOption` are interfaces with an unexported method. A program's own
  `func(*T)` used as one no longer compiles; the package's functions
  still make every option.
* **Behaviour changes:**
  * **Enum text.** 21 enums gain `MarshalText` and `UnmarshalText`, and
    those without `String` gain it, with D2's tokens: their JSON is the
    token, where it was a number. Every enum's out-of-range value prints
    `Type(N)`; `termcap`'s seven, `command.Format`, `glyph.Tier`,
    `launch`'s two, `theme`'s two, and `termsvc`'s `ActivityState` and
    `SkipReason` printed the bare number.
  * **`termcap`'s JSON is v2:** `<>&` are not escaped; a duplicate name
    and invalid UTF-8 are refused on reading, and a name matches in its
    own case only; a program's own invalid UTF-8 is written as U+FFFD
    (D3).
  * **`termcap.ParseTmux`** cleans its fields with `internal/sanitize`
    (D3).
  * **`when.Parse`'s errors** are `*SyntaxError`, with the texts as
    before.
  * **`termcap`'s unknown-name errors** wrap `ErrUnknownName`, with the
    texts as before.
* **Additions:** `command.Verdict`, `Record.Verdict`,
  `Request.WhenContext`, `Invocation.WhenContext` and `AuditorFunc`;
  `termcap.NewProber` and `ErrUnknownName`; `layout`'s eight `With…` and
  `Without…` options; `workspace.WithoutMouse`; `termsvc.WithNotifyFilter`,
  `BackendFunc` and `ClipboardFunc`; `when.SyntaxError`; and the enums'
  text methods.
* **Unchanged for a program:** `command`'s `json.RawMessage` fields are
  spelled `jsontext.Value`, the same type in Go 1.27.

#### The tag, CI and the smoke test

* **The tag.** The owner committed the documents as `9a90b6d`, pushed
  `main`, and tagged `v0.9.0` there (an annotated tag). The owner pushed
  before the agent ran the disclosure guard, so the push's own pre-push
  guard was the first over these commits; the agent then ran it over
  `ae6e76c..9a90b6d`, which holds everything that push sent: exit 0. CI
  passed on `main`, run `38065389258`, and on the tag, run
  `38066341008`; and on `2c58073`, Step 7, run `38063219588`.
* **The consumer smoke test,** as the releasing guide gives it, in a
  scratch module outside the repository, with `go env GOWORK` empty: `go
  get github.com/maccavelli/go-tui-lib@v0.9.0` from the proxy, `go mod
  tidy`, then `go vet`, `go build` and `go run` of two programs.
  * **The old names:** `command.Decision` from a `GateFunc`,
    `Request.Context` and `Invocation.Context`, `Record.Decision`,
    `termcap.New`, `layout.SidebarWidth`, `Gap` and `NoResponsive`,
    `workspace.WithMouse(true)` and `termsvc.WithGate`. It builds and
    prints "smoke old names: ok": a destructive command an agent runs is
    asked of the gate and allowed, the handler sees the context, the
    record holds `AllowOnce`, `RenderPlain` draws the pane, and the
    filter skips with `SkipGated`. golangci-lint 2.14.0's staticcheck,
    with every issue shown, reports SA1019 for each of the 11 uses, with
    its `// Deprecated:` text. The standalone `staticcheck` on the host
    is too old to read Go 1.27's export data; golangci-lint's is the one
    the repository runs.
  * **The new names:** the same on `Verdict`, `WhenContext`,
    `Record.Verdict`, `AuditorFunc`, `NewProber`, the `With…` options,
    `WithoutMouse` and `WithNotifyFilter`; and what `v0.9.0` changed:
    `Danger`'s JSON is `"read-only"`, `Mux(9)` prints "Mux(9)",
    `Surface` reads "cli|key" and prints "key|cli", `Mux`'s unknown name
    wraps `ErrUnknownName` with its text, `Caps`'s JSON keeps
    `"a<b>&c"`, `when.Parse("a & b")` is a `*SyntaxError` at offset 2,
    and a `BackendFunc` and a `ClipboardFunc` receive `Notify`'s and
    `Copy`'s text. It prints "smoke new names: ok", and staticcheck
    reports nothing.
  * **The break:** a third program assigning `func(*workspace.Workspace)
    {}` to a `workspace.Option` fails to build: "does not implement
    workspace.Option (missing method apply)".
  * **The negative control:** a copy of the new-names program expecting
    `Danger`'s JSON as the number 1 failed with "FAIL: Danger's JSON =
    {\"D\":\"read-only\"}" and exit status 1.

#### `scripts/apicheck.allow`, emptied

* **Why now:** the API diff gate's base is the newest tag that HEAD does
  not contain. While HEAD is the tag, that is `v0.8.0`; from the next
  commit it is `v0.9.0`, where apidiff reports none of the 14 entries.
* **The proof, on a scratch clone of `9a90b6d`:** an empty commit after
  the tag, with the list as it was, fails: "against v0.9.0, 0
  incompatible change(s)", and the 14 entries "stale". With the list
  emptied, it passes: "against v0.9.0, 0 incompatible change(s)",
  "clean".
* **The change:** both of W4's blocks go; the file is the header alone,
  byte for byte as at `v0.8.0`. In the working tree, while HEAD is still
  the tag, `make apicheck` compares with `v0.8.0` and fails on the 14
  changes now unlisted; the close-out commit makes the base `v0.9.0`.

Step 8 is done.
