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
| `Decision` | `command` | a gate's answer: allow or reject, once or always. W4 of [0014-MADR](decisions/0014-MADR-native-integration-api.md) renames it `Verdict` in `v0.10.0`, and the clash ends |
| | `launch` | the start-up outcome: whether the TUI can start, where, and why |
| | `when` | the values a when-clause expression reads, by key |
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
| `Format` | `command` | 0014-MADR W2 | how `WriteResult` writes a result: text or JSON |
| `Param` | `command` | 0014-MADR W2 | one parameter of a command, for flags and completion |
| `PanicError` | `command` | 0014-MADR W2 | a handler's panic, as an error |
| `PlainViewer` | `workspace` | 0014-MADR W2 | a pane that can render itself without chrome or colour |
| `GlyphThemeBuilder` | `workspace` | 0014-MADR W2 | a theme builder that is given the glyphs |
| `LoadOption` | `command` | 0014-MADR W3 | an option of `LoadDirWith` |

## Built from those records

`launch`'s `Choice`, `Config`, `Target`, `Reason`, `Streams`,
`StreamSource`, `Flags`, `Restorer` and `ExitError`, and `glyph.Tier`, were
reserved here and are now built
([0013-PLAN](decisions/0013-PLAN-cli-integration-helpers.md)). Each has one
meaning, as its package documents it. `Decision` and `Terminal` are in
"Deliberate" above.

## Names the planned records must not take

[0007-MADR](decisions/0007-MADR-keymap-engine.md),
[0008-MADR](decisions/0008-MADR-command-palette.md) and
[0009-MADR](decisions/0009-MADR-streaming-content-engine.md) are accepted
but unbuilt. Each still names a type that collides with one above:

| Name | Taken by | Planned in |
| :--- | :--- | :--- |
| `Context` | `layout`, `when` | 0007 (`keymap.Context`) |
| `Origin` | `command`, `termcap` | 0007 (`keymap.Origin`) |
| `Conflict` | `command` | 0007 (`keymap.Conflict`) |
| `Policy` | `termsvc` | 0009 (`safetext.Policy`, `frame.Policy`) |

[0014-PLAN-canonicalization.md](decisions/0014-PLAN-canonicalization.md)
Step 2 amends those records with names from this page before they are
built.

## The rule for a new name

The record that introduces an exported type name checks it against this
page. A name already here means what this page says, or the record picks
another. A deliberate clash is added here and to `sharedNames` in the same
change.
