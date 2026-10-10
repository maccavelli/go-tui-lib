# Glossary

One name means one thing across go-tui-lib
([0014-MADR-native-integration-api.md](decisions/0014-MADR-native-integration-api.md)
W0.3). `internal/conformance`'s `TestNoTypeNameMeansTwoThings` fails when
two public packages export a type of the same name, unless this page lists
the name as deliberate and the test's `sharedNames` holds its exact
packages. A new package with an allowed name still fails.

## Deliberate: one name in several packages

Each is qualified by its package wherever it is used. The packages listed
are the only ones; none is added.

| Name | Packages | Means |
| :--- | :--- | :--- |
| `Option` | any | each package's own option type, passed to its constructor |
| `Context` | `layout` | the state a layout tree is arranged with, collecting the plan as each node places its leaves |
| | `when` | the values a when-clause expression reads, by key |
| `Decision` | `command` | a gate's answer: allow or reject, once or always. W4 of [0014-MADR](decisions/0014-MADR-native-integration-api.md) renames it `Verdict` in `v0.9.0`, keeps `Decision` as a deprecated alias, and removes it in `v0.10.0`, when the clash ends |
| | `launch` | the start-up outcome: whether the TUI can start, where, and why |
| `Kind` | `command` | what running a command does: `Action`, `Prompt` or `Forward` |
| | `when` | the type of a `when.Value` |
| `Origin` | `command` | what asked for a command to run: a key, the palette, a slash line, the program's CLI, an agent |
| | `termcap` | where a capability fact came from, weakest first |
| `Pane` | `layout` | a leaf node of a layout tree, by pane ID |
| | `workspace` | the component interface a workspace hosts |
| `Terminal` | `launch/launchtest` | a fake terminal stream for tests: it says it is a terminal, has a size, and takes typed keys |
| | `termcap/termcaptest` | a fake terminal for tests that answers a program's queries as a terminal profile does |

## Reserved by accepted records

These names are taken by records that are accepted but not yet built. A
later record does not reuse them for something else.

| Name | Package | Record | Means |
| :--- | :--- | :--- | :--- |
| `Verdict` | `command` | 0014-MADR W4 | a gate's answer, today `command.Decision` |
| `WhenContext` | `command` | 0014-MADR W4 | `Request`'s and `Invocation`'s `when.Context` field, today `Context` |

## Built from those records

`launch`'s `Choice`, `Config`, `Target`, `Reason`, `Streams`,
`StreamSource`, `Flags`, `Restorer` and `ExitError`, and `glyph.Tier`, were
reserved here and are now built
([0013-PLAN](decisions/0013-PLAN-cli-integration-helpers.md)). So are
`command`'s `Format`, `Param`, `PanicError` and `LoadOption`, and
`workspace`'s `PlainViewer` and `GlyphThemeBuilder`, in `v0.8.0`
([0014-PLAN-component-native-forms](decisions/0014-PLAN-component-native-forms.md),
[0014-PLAN-hardening](decisions/0014-PLAN-hardening.md)). Each has one
meaning, as its package documents it. `Decision` and `Terminal` are in
"Deliberate" above.

## One meaning each

These names are taken, each for one meaning
([0014-PLAN-canonicalization](decisions/0014-PLAN-canonicalization.md)
Step 2). A planned record that needs the idea uses the name below, or one
of its own.

| Name | Means | Belongs to |
| :--- | :--- | :--- |
| `Verdict` | a gate's answer | `command` |
| `WhenContext` | a `when.Context` of key values; `When` stays a when-expression | `command`, and `workspace`'s method |
| `Decision` | the start-up outcome, once `command`'s is `Verdict` | `launch` |
| `Policy` | the notification policy | `termsvc` |
| `Plan`, `Rule`, `Span` | a solved layout, a responsive rule, how far a bottom pane spans | `layout` |
| `Result`, `Conflict`, `Scope`, `Source` | what a command produced, a slash name two commands claim, where a command applies, where it came from | `command` |
| `Query` | a capability query | `termcap` |
| `Tier` | a glyph set's tier | `glyph` |
| `Filter` | a message filter, as `launch.WithFilter` takes | a message filter only |
| `Renderer` | the streaming interface that turns Markdown into styled text | `stream`; an adapter's own `Renderer` implements it |
| `New` | a constructor of the type named after its package; any other constructor is `NewX` | every package |

## Names the planned records must not take

[0007-MADR](decisions/0007-MADR-keymap-engine.md),
[0008-MADR](decisions/0008-MADR-command-palette.md) and
[0009-MADR](decisions/0009-MADR-streaming-content-engine.md) are accepted
but unbuilt. Their amendments of 2026-10-09 (0007 A3, 0008 A2, 0009 A4,
from [0014-PLAN-canonicalization.md](decisions/0014-PLAN-canonicalization.md)
Step 2) renamed every type that collided with one above, or with another
planned one: `keymap.Context`, `Origin`, `Conflict`, `Rule`, `Result`,
`Options`, `Match` and `Matcher`; `fuzzy.Tier` and `Span`;
`palette.Query`, `Scope` and `Palette`; `safetext.Policy` and `Filter`;
and `frame.Policy`, `Plan` and `Msg`.

One planned pair is deliberate: `Renderer`, the interface in `stream`,
and the type in `stream/glamourmd` that implements it. It is added to
"Deliberate" and to `sharedNames` when the adapter is built.

## Width: one rule

Width is measured in cells, by one rule
([0014-PLAN-canonicalization](decisions/0014-PLAN-canonicalization.md)
Step 7; `AGENTS.md`, "TUI conventions" rule 5):

| Who | Measures and cuts with |
| :--- | :--- |
| `workspace` | its width method: `ansi.WcWidth` until the terminal reports grapheme clustering (mode 2027), `ansi.GraphemeWidth` after, unless `WithWidthMethod` fixes it; it cuts with `internal/sanitize`'s `Truncate`, never wider than asked |
| a test | `tuitest.Fits`, under the case's method |
| `termsvc`, `termcap`'s report | grapheme clusters, with `ansi.Truncate` |

## The rule for a new name

The record that introduces an exported type name checks it against this
page. A name already here means what this page says, or the record picks
another. A deliberate clash is added here and to `sharedNames` in the same
change.
