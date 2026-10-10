---
status: proposed
date: 2026-10-09
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
  * **0007-MADR** (`keymap`):
    * `Context` → `Scope` (or `command.Scope` itself);
    * `Origin` → `Location`;
    * `Conflict` → `BindingConflict`;
    * `Rule` → `Binding`;
    * `Result` → `Outcome`;
    * `Options` → `Config`;
    * `Match` and `Matcher` → `Hit` and `Dispatcher`;
    * the `Rule.Context` and `Conflict.Context` fields → `Scope`;
    * the `Unsupported` constant → `NotSupported`.
  * **0008-MADR** (`fuzzy`, `palette`):
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
  before this re-read. This amendment awaits the owner's approval before
  Step 2.
