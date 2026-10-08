# go-tui-lib documentation

## Records

Decisions, their implementation plans, and reports, in one numbered sequence.
go-core-lib was renamed go-selfupdate-lib at `v1.5.0`. Records name it as
it was when they were written
([0001-MADR, A3](decisions/0001-MADR-scaffold-charm-tui-library.md#a3-2026-10-03-go-core-lib-is-now-go-selfupdate-lib)).

| No. | Kind | Title | Status |
| :--- | :--- | :--- | :--- |
| 0001 | MADR | [Scaffold go-tui-lib as a Go 1.27.1 Charm v2 library to the go-core-lib standard, with honest gates until the first package lands](decisions/0001-MADR-scaffold-charm-tui-library.md) | accepted; A1–A3 recorded |
| 0001 | PLAN | [Implement the go-tui-lib Go 1.27.1 Charm v2 library scaffold](decisions/0001-PLAN-scaffold-charm-tui-library.md) | complete |
| 0001 | REPORT | [TUI working example, fleet consumers and the planned packages](reports/0001-REPORT-tui-working-example-and-consumers.md) | — |
| 0002 | MADR | [Build multi-pane terminal workspaces from a pure layout solver and a Bubble Tea pane host, with pi-go's agent session as the first consumer](decisions/0002-MADR-multi-pane-workspace-layouts.md) | accepted; A1, A2 accepted |
| 0002 | PLAN | [Implement multi-pane workspaces (`v0.1.0`)](decisions/0002-PLAN-multi-pane-workspace-layouts.md) | complete |
| 0002 | PLAN | [Harden multi-pane workspaces (`v0.1.1`)](decisions/0002-PLAN-harden-workspace-v0-1-1.md) | complete |
| 0003 | REPORT | [Agent TUI ecosystem research, an audit of `v0.1.0`, and a source pass over four agent TUIs](reports/0003-REPORT-agent-tui-ecosystem-research.md) | — |
| 0004 | MADR | [Draw the workspace on an ultraviolet cell buffer, follow the terminal's width method and theme, and adopt Go 1.27 idioms in the public API](decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md) | accepted; A1, A2 accepted |
| 0004 | PLAN | [Implement direct cell drawing, terminal-following width and theme, and Go 1.27 accessors (`v0.2.0`)](decisions/0004-PLAN-integrate-charm-v2-and-go-1-27.md) | complete |
| 0005 | MADR | [Detect terminal capabilities with one sentinel-terminated probe that observes Bubble Tea's own queries, and offer notifications, clipboard, links and prompt marks as commands built from the result](decisions/0005-MADR-terminal-capabilities-and-services.md) | accepted; A1–A5 accepted |
| 0005 | PLAN | [Implement terminal capabilities and services (`termcap`, `termsvc`)](decisions/0005-PLAN-terminal-capabilities-and-services.md) | complete |
| 0006 | MADR | [Make one command registry the source of every action, for keys, palette, slash commands, the shell and agents, on open standards](decisions/0006-MADR-command-registry.md) | accepted; A1–A13 accepted; §10, A1's front ends and A10–A12 superseded by 0012 |
| 0006 | PLAN | [Implement the command registry (`when`, `command`, `command/cli`, and the Cobra and Kong front ends)](decisions/0006-PLAN-command-registry.md) | complete |
| 0007 | MADR | [Bind keys to command IDs through a context-aware keymap engine, with chords, a leader key and VS Code-format user keymaps](decisions/0007-MADR-keymap-engine.md) | accepted; A1, A2 accepted; Q8 open |
| 0007 | PLAN | [Implement the keymap engine](decisions/0007-PLAN-keymap-engine.md) | proposed |
| 0008 | MADR | [Find and run anything from one command palette, built on an in-repository fuzzy matcher and asynchronous, scoped providers](decisions/0008-MADR-command-palette.md) | accepted; A1 accepted |
| 0008 | PLAN | [Implement the command palette and the fuzzy matcher](decisions/0008-PLAN-command-palette.md) | proposed |
| 0009 | MADR | [Stream agent output through a stable-prefix Markdown engine, a frame scheduler, an input filter and a safe-text sanitizer](decisions/0009-MADR-streaming-content-engine.md) | accepted; A1–A3 accepted; Q9 open |
| 0009 | PLAN | [Implement the streaming content engine](decisions/0009-PLAN-streaming-content-engine.md) | proposed |
| 0010 | MADR | [Ship the Cobra, Kong and glamour adapters as nested Go modules, released apart from the root and developed through a committed go.work, with every gate run per module and without the workspace](decisions/0010-MADR-nested-adapter-modules.md) | accepted; A1–A3 accepted; the command adapters superseded by 0012 |
| 0010 | PLAN | [Implement the multi-module repository: go.work, per-module gates, and the release procedure](decisions/0010-PLAN-nested-adapter-modules.md) | complete |
| 0010 | REPORT | [Nested modules, a committed go.work, and the adapters' upstream sources](reports/0010-REPORT-nested-modules-and-adapter-sources.md) | — |
| 0011 | MADR | [Bring the living documents and the unexecuted records up to date after `v0.5.0`, correcting records only by amendment and front matter](decisions/0011-MADR-docs-accuracy-after-v0-5-0.md) | accepted |
| 0011 | PLAN | [Implement the documentation pass after `v0.5.0`](decisions/0011-PLAN-docs-accuracy-after-v0-5-0.md) | complete |
| 0012 | MADR | [Make go-tui-lib a TUI layer a program stacks on its own Go CLI, and retire the library's CLI front ends](decisions/0012-MADR-bring-your-own-cli.md) | accepted |
| 0012 | PLAN | [Retire the CLI front ends: retract the adapters, remove `command/cli` and A12's check, release `v0.6.0`](decisions/0012-PLAN-bring-your-own-cli.md) | complete |
| 0013 | MADR | [Add a `launch` package that decides between a TUI and plain output, and runs the TUI on the program's own streams](decisions/0013-MADR-cli-integration-helpers.md) | accepted; A1 accepted, with A1.7–A1.10 from execution |
| 0013 | PLAN | [Implement the `launch` package with native types for every CLI framework, and release `v0.7.0`](decisions/0013-PLAN-cli-integration-helpers.md) | complete; Step 11 released as `v0.7.1` |
| 0014 | MADR | [Make go-tui-lib's API natively integrable from any Go CLI: native types and component forms, hardening, and canonicalization during `v0`](decisions/0014-MADR-native-integration-api.md) | accepted; A1 recorded |
| 0014 | PLAN | [W0: the API diff gate, conformance bans, the collision check, the glossary, the framework-example harness and the conventions](decisions/0014-PLAN-api-policy-gates.md) | complete |
| 0014 | PLAN | [W2: native forms in `command`, `workspace`, `termcap`, `termsvc`, `theme` and `tuitest`](decisions/0014-PLAN-component-native-forms.md) | in-progress |
| 0014 | PLAN | [W3: bounded input, one sanitizer, overflow-safe layout, no mutable package variables, wider fuzzing; release `v0.8.0`](decisions/0014-PLAN-hardening.md) | proposed |
| 0014 | PLAN | [W4: renames with deprecation aliases, opaque options, one enum helper, JSON v2, structured errors; release `v0.9.0` and `v0.10.0`](decisions/0014-PLAN-canonicalization.md) | proposed |
| 0014 | REPORT | [An API assessment of `v0.6.0`, and how Go CLIs and agent tools integrate a TUI](reports/0014-REPORT-api-assessment-and-integration-research.md) | — |
| 0015 | MADR | [Make the pre-add check fail when gofmt itself fails, and skip a tracked file the work tree no longer has](decisions/0015-MADR-precheck-gofmt-errors.md) | accepted |
| 0015 | PLAN | [Implement: the pre-add check fails when gofmt fails, and skips a tracked file the work tree no longer has](decisions/0015-PLAN-precheck-gofmt-errors.md) | complete |

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](architecture.md) |
| build a main pane with a sidebar, a bottom pane and a footer | [guides/building-workspaces.md](guides/building-workspaces.md) |
| write a pane, and choose its optional interfaces | [guides/building-workspaces.md](guides/building-workspaces.md#write-a-pane) |
| make my own layout, or change a preset | [guides/building-workspaces.md](guides/building-workspaces.md#choose-a-layout) |
| save and restore what the user changed in the layout | [guides/building-workspaces.md](guides/building-workspaces.md#persist-the-layout) |
| open a dialog, picker or completion pop-up | [guides/building-workspaces.md](guides/building-workspaces.md#overlays) |
| know what my program, not the workspace, owns | [guides/building-workspaces.md](guides/building-workspaces.md#the-program-owns-the-program) |
| see a whole agent session | `ExampleWorkspace_agentSession` in `workspace/agent_test.go` |
| know which Charm version to use | [0001-MADR, §3](decisions/0001-MADR-scaffold-charm-tui-library.md#3-toolchain-and-dependencies) |
| know what this module may import, and what it never may | [AGENTS.md, Dependencies](../AGENTS.md#dependencies) |
| name an exported type, option, error or hook, deprecate a name, or read a package's stability line | [AGENTS.md, API conventions](../AGENTS.md#api-conventions) |
| know what a type name means, and which names are shared or reserved | [glossary.md](glossary.md) |
| know who owns the screen, the output and ctrl+c | [AGENTS.md, TUI conventions](../AGENTS.md#tui-conventions) |
| test a component across colour, charset and width | [AGENTS.md, TUI conventions](../AGENTS.md#tui-conventions), rule 6 |
| add a package | write its MADR and PLAN first: [AGENTS.md](../AGENTS.md#madr-and-plan-before-mutating-work) |
| run the checks before committing | `make pre-add-check`; see [AGENTS.md, Pre-add checks](../AGENTS.md#pre-add-checks) |
| know why the gates once failed on an empty module | [0001-MADR, §8](decisions/0001-MADR-scaffold-charm-tui-library.md#8-push-and-identity) and [Consequences](decisions/0001-MADR-scaffold-charm-tui-library.md#consequences) |
| know where the TUI rules came from | [0001-MADR, §6](decisions/0001-MADR-scaffold-charm-tui-library.md#6-repository-conventions-for-tui-packages): ocp-login's records |
| see what ocp-login's TUI design offers for extraction | [0001-REPORT, §1](reports/0001-REPORT-tui-working-example-and-consumers.md#1-the-working-example-ocp-login-by-pattern) |
| know who will use this library | [0001-REPORT, §2 and §3](reports/0001-REPORT-tui-working-example-and-consumers.md#2-the-fleets-tui-programs) |
| see the planned `updatetea` API | [0001-REPORT, §4](reports/0001-REPORT-tui-working-example-and-consumers.md#4-updatetea-as-go-core-lib-plans-it) |
| know how the multi-pane workspace is designed | [0002-MADR](decisions/0002-MADR-multi-pane-workspace-layouts.md) |
| know how this scaffold differs from go-core-lib's (now go-selfupdate-lib) | [0001-MADR, §5](decisions/0001-MADR-scaffold-charm-tui-library.md#5-deliberate-differences-from-go-core-lib) |
| know what was wrong in `v0.1.0`, and how `v0.1.1` fixes it | [0003-REPORT, §1](reports/0003-REPORT-agent-tui-ecosystem-research.md#1-audit-of-v010) and [0002-PLAN-harden-workspace-v0-1-1.md](decisions/0002-PLAN-harden-workspace-v0-1-1.md) |
| see how other agent TUIs solved streaming, keymaps, palettes and terminal features | [0003-REPORT](reports/0003-REPORT-agent-tui-ecosystem-research.md) |
| know which terminal standards Bubble Tea v2 already handles | [0003-REPORT, §5](reports/0003-REPORT-agent-tui-ecosystem-research.md#5-terminal-standards) |
| know the order the planned packages land in | [0003-REPORT, §7](reports/0003-REPORT-agent-tui-ecosystem-research.md#7-candidates-and-what-the-owner-chose) |
| see what the Kilo, Grok Build, opencode and codex TUIs add beyond the planned records | [0003-REPORT, §8](reports/0003-REPORT-agent-tui-ecosystem-research.md#8-second-pass-the-kilo-grok-build-opencode-and-codex-tuis) |
| know which terminals need which workarounds | [0003-REPORT, §9](reports/0003-REPORT-agent-tui-ecosystem-research.md#9-per-terminal-quirks-the-sources-work-around) |
| see the candidate packages after [0009-MADR-streaming-content-engine.md](decisions/0009-MADR-streaming-content-engine.md), and the modules they would need | [0003-REPORT, §11](reports/0003-REPORT-agent-tui-ecosystem-research.md#11-candidates-from-the-second-pass) |
| add a command, with its arguments, danger and availability | [guides/commands.md](guides/commands.md#define-a-command) |
| run a command from a key or `Update`, and react to its result | [guides/commands.md](guides/commands.md#run-commands-from-your-program) |
| let an agent drive the TUI over MCP, safely | [guides/commands.md](guides/commands.md#let-an-agent-drive-the-program) |
| load the user's and the project's command files, or an agent's commands | [guides/commands.md](guides/commands.md#command-files) |
| run the same commands from my own CLI (`flag`, Cobra, Kong) | [guides/commands.md](guides/commands.md#run-commands-from-your-own-cli) |
| see complete, tested programs on `flag`, Cobra, Kong and urfave/cli | `testdata/frameworks/`, built and run by `make examples` ([architecture.md, Tooling](architecture.md#tooling)) |
| know why go-tui-lib ships no CLI front end | [0012-MADR](decisions/0012-MADR-bring-your-own-cli.md) |
| start the TUI from my CLI with `--tui`, in `flag`, Cobra, Kong or urfave/cli, and fall back to the CLI when it cannot run | [guides/commands.md](guides/commands.md#start-the-tui-from-your-cli) |
| test my CLI's `--tui` path with fake terminals | `launch/launchtest`, and `ExampleTerminal` in `launch/launchtest/example_test.go` |
| know why `launch` decides and falls back as it does | [0013-MADR](decisions/0013-MADR-cli-integration-helpers.md) |
| see how the API will grow to fit any Go CLI framework natively, and the hardening and renames planned before `v1` | [0014-MADR](decisions/0014-MADR-native-integration-api.md) |
| see how flag, Cobra, Kong, urfave/cli and agent tools such as Codex, Grok and Pi integrate a TUI, and what the API assessment found | [0014-REPORT](reports/0014-REPORT-api-assessment-and-integration-research.md) |
| know why the command registry is built this way | [0006-MADR](decisions/0006-MADR-command-registry.md) |
| see the planned keymap and `keybindings.json` format | [0007-MADR](decisions/0007-MADR-keymap-engine.md) |
| see the planned command palette and fuzzy matcher | [0008-MADR](decisions/0008-MADR-command-palette.md) |
| see how streamed Markdown, frames and untrusted text will be handled | [0009-MADR](decisions/0009-MADR-streaming-content-engine.md) |
| know why terminal capabilities are detected this way | [0005-MADR](decisions/0005-MADR-terminal-capabilities-and-services.md) |
| learn what the terminal supports, over SSH and inside tmux | [guides/terminal-capabilities.md](guides/terminal-capabilities.md) |
| quit without leaving mode 2031 set | [guides/terminal-capabilities.md](guides/terminal-capabilities.md#quit-through-the-prober) |
| send a notification, copy to the clipboard, or show a link | [guides/terminal-capabilities.md](guides/terminal-capabilities.md#notifications-and-focus) |
| write a doctor command | [guides/terminal-capabilities.md](guides/terminal-capabilities.md#a-doctor-command) |
| test a program against fake terminals | [guides/terminal-capabilities.md](guides/terminal-capabilities.md#test-with-fake-terminals) |
| know why the workspace draws with ultraviolet directly | [0004-MADR](decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md) |
| make borders line up after emoji, or measure as the workspace does | [guides/building-workspaces.md](guides/building-workspaces.md#text-width) |
| follow the terminal's light or dark theme, or keep my own colours | [guides/building-workspaces.md](guides/building-workspaces.md#chrome-and-theme) |
| show a help footer for the workspace and the focused pane | [guides/building-workspaces.md](guides/building-workspaces.md#keys-and-the-mouse) |
| know why an adapter such as glamour is a separate Go module, and how it is released | [0010-MADR](decisions/0010-MADR-nested-adapter-modules.md) |
| add a nested module to the repository | [guides/releasing.md](guides/releasing.md#add-a-module) |
| release the root, an adapter, or a change that spans both | [guides/releasing.md](guides/releasing.md) |
| know which gates run per module, and why with `GOWORK=off` | [AGENTS.md, Modules](../AGENTS.md#modules) |
| use the command registry from Cobra or Kong | [guides/commands.md](guides/commands.md#run-commands-from-your-own-cli) |
| see what Cobra, fang, Kong and glamour do that the library's rules must answer | [0010-REPORT, §6–§10](reports/0010-REPORT-nested-modules-and-adapter-sources.md) |
