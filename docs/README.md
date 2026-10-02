# go-tui-lib documentation

## Records

Decisions and their implementation plans, in one numbered sequence.

| No. | Kind | Title | Status |
| :--- | :--- | :--- | :--- |
| 0001 | MADR | [Scaffold go-tui-lib as a Go 1.27.1 Charm v2 library to the go-core-lib standard, with honest gates until the first package lands](decisions/0001-MADR-scaffold-charm-tui-library.md) | accepted |
| 0001 | PLAN | [Implement the go-tui-lib Go 1.27.1 Charm v2 library scaffold](decisions/0001-PLAN-scaffold-charm-tui-library.md) | complete |
| 0001 | REPORT | [TUI working example, fleet consumers and the planned packages](reports/0001-REPORT-tui-working-example-and-consumers.md) | — |
| 0002 | MADR | [Build multi-pane terminal workspaces from a pure layout solver and a Bubble Tea pane host, with pi-go's agent session as the first consumer](decisions/0002-MADR-multi-pane-workspace-layouts.md) | accepted; A1 proposed |
| 0002 | PLAN | [Implement multi-pane workspaces (`v0.1.0`)](decisions/0002-PLAN-multi-pane-workspace-layouts.md) | complete |
| 0002 | PLAN | [Harden multi-pane workspaces (`v0.1.1`)](decisions/0002-PLAN-harden-workspace-v0-1-1.md) | proposed |
| 0003 | REPORT | [Agent TUI ecosystem research, an audit of `v0.1.0`, and a source pass over four agent TUIs](reports/0003-REPORT-agent-tui-ecosystem-research.md) | — |
| 0004 | MADR | [Draw the workspace on an ultraviolet cell buffer, follow the terminal's width method and theme, and adopt Go 1.27 idioms in the public API](decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md) | proposed |
| 0004 | PLAN | [Implement direct cell drawing, terminal-following width and theme, and Go 1.27 accessors (`v0.2.0`)](decisions/0004-PLAN-integrate-charm-v2-and-go-1-27.md) | proposed |
| 0005 | MADR | [Detect terminal capabilities with one sentinel-terminated probe that observes Bubble Tea's own queries, and offer notifications, clipboard, links and prompt marks as commands built from the result](decisions/0005-MADR-terminal-capabilities-and-services.md) | proposed |
| 0005 | PLAN | [Implement terminal capabilities and services (`termcap`, `termsvc`)](decisions/0005-PLAN-terminal-capabilities-and-services.md) | proposed |
| 0006 | MADR | [Make one command registry the source of every action, for keys, palette, slash commands, the shell and agents, on open standards](decisions/0006-MADR-command-registry.md) | proposed |
| 0006 | PLAN | [Implement the command registry (`command`, `when`, `command/cli`)](decisions/0006-PLAN-command-registry.md) | proposed |
| 0007 | MADR | [Bind keys to command IDs through a context-aware keymap engine, with chords, a leader key and VS Code-format user keymaps](decisions/0007-MADR-keymap-engine.md) | proposed |
| 0007 | PLAN | [Implement the keymap engine](decisions/0007-PLAN-keymap-engine.md) | proposed |
| 0008 | MADR | [Find and run anything from one command palette, built on an in-repository fuzzy matcher and asynchronous, scoped providers](decisions/0008-MADR-command-palette.md) | proposed |
| 0008 | PLAN | [Implement the command palette and the fuzzy matcher](decisions/0008-PLAN-command-palette.md) | proposed |
| 0009 | MADR | [Stream agent output through a stable-prefix Markdown engine, a frame scheduler, an input filter and a safe-text sanitizer](decisions/0009-MADR-streaming-content-engine.md) | proposed |
| 0009 | PLAN | [Implement the streaming content engine](decisions/0009-PLAN-streaming-content-engine.md) | proposed |

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
| know who owns the screen, the output and ctrl+c | [AGENTS.md, TUI conventions](../AGENTS.md#tui-conventions) |
| test a component across colour, charset and width | [AGENTS.md, TUI conventions](../AGENTS.md#tui-conventions), rule 6 |
| add a package | write its MADR and PLAN first: [AGENTS.md](../AGENTS.md#madr-and-plan-before-mutating-work) |
| run the checks before committing | `make pre-add-check`; see [AGENTS.md, Pre-add checks](../AGENTS.md#pre-add-checks) |
| know why the gates fail on an empty module | [0001-MADR, §8](decisions/0001-MADR-scaffold-charm-tui-library.md#8-push-and-identity) and [Consequences](decisions/0001-MADR-scaffold-charm-tui-library.md#consequences) |
| know where the TUI rules came from | [0001-MADR, §6](decisions/0001-MADR-scaffold-charm-tui-library.md#6-repository-conventions-for-tui-packages): ocp-login's records |
| see what ocp-login's TUI design offers for extraction | [0001-REPORT, §1](reports/0001-REPORT-tui-working-example-and-consumers.md#1-the-working-example-ocp-login-by-pattern) |
| know who will use this library | [0001-REPORT, §2 and §3](reports/0001-REPORT-tui-working-example-and-consumers.md#2-the-fleets-tui-programs) |
| see the planned `updatetea` API | [0001-REPORT, §4](reports/0001-REPORT-tui-working-example-and-consumers.md#4-updatetea-as-go-core-lib-plans-it) |
| know how the multi-pane workspace is designed | [0002-MADR](decisions/0002-MADR-multi-pane-workspace-layouts.md) |
| know how this scaffold differs from go-core-lib's | [0001-MADR, §5](decisions/0001-MADR-scaffold-charm-tui-library.md#5-deliberate-differences-from-go-core-lib) |
| know what was wrong in `v0.1.0`, and how `v0.1.1` fixes it | [0003-REPORT, §1](reports/0003-REPORT-agent-tui-ecosystem-research.md#1-audit-of-v010) and [0002-PLAN-harden-workspace-v0-1-1.md](decisions/0002-PLAN-harden-workspace-v0-1-1.md) |
| see how other agent TUIs solved streaming, keymaps, palettes and terminal features | [0003-REPORT](reports/0003-REPORT-agent-tui-ecosystem-research.md) |
| know which terminal standards Bubble Tea v2 already handles | [0003-REPORT, §5](reports/0003-REPORT-agent-tui-ecosystem-research.md#5-terminal-standards) |
| know the order the planned packages land in | [0003-REPORT, §7](reports/0003-REPORT-agent-tui-ecosystem-research.md#7-candidates-and-what-the-owner-chose) |
| see what the Kilo, Grok Build, opencode and codex TUIs add beyond the planned records | [0003-REPORT, §8](reports/0003-REPORT-agent-tui-ecosystem-research.md#8-second-pass-the-kilo-grok-build-opencode-and-codex-tuis) |
| know which terminals need which workarounds | [0003-REPORT, §9](reports/0003-REPORT-agent-tui-ecosystem-research.md#9-per-terminal-quirks-the-sources-work-around) |
| see the candidate packages after 0009, and the modules they would need | [0003-REPORT, §11](reports/0003-REPORT-agent-tui-ecosystem-research.md#11-candidates-from-the-second-pass) |
| see the planned command registry, and how agents and the shell run commands | [0006-MADR](decisions/0006-MADR-command-registry.md) |
| see the planned keymap and `keybindings.json` format | [0007-MADR](decisions/0007-MADR-keymap-engine.md) |
| see the planned command palette and fuzzy matcher | [0008-MADR](decisions/0008-MADR-command-palette.md) |
| see how streamed Markdown, frames and untrusted text will be handled | [0009-MADR](decisions/0009-MADR-streaming-content-engine.md) |
| see how terminal capabilities will be detected | [0005-MADR](decisions/0005-MADR-terminal-capabilities-and-services.md) |
| know why the workspace will draw with ultraviolet directly | [0004-MADR](decisions/0004-MADR-integrate-charm-v2-and-go-1-27.md) |
