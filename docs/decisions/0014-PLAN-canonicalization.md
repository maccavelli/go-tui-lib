---
status: proposed
date: 2026-10-07
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
  per W0's PLAN. Renaming any needs the owner and a record.
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
    (`command/dispatch.go:107`).
  * The `Invocation` it builds sets both (`:113`).
  * `builtin.go:27` reads `WhenContext`.
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
    v2 name.
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
    reports mode 2027 and GraphemeWidth after;
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
  * `termcaptest.RunTimeout`;
  * the seven `workspace.Key*` variables, deprecated by W3.
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

Not started.
