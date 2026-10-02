# go-tui-lib documentation

## Records

Decisions and their implementation plans, in one numbered sequence.

| No. | Kind | Title | Status |
| :--- | :--- | :--- | :--- |
| 0001 | MADR | [Scaffold go-tui-lib as a Go 1.27.1 Charm v2 library to the go-core-lib standard, with honest gates until the first package lands](decisions/0001-MADR-scaffold-charm-tui-library.md) | accepted |
| 0001 | PLAN | [Implement the go-tui-lib Go 1.27.1 Charm v2 library scaffold](decisions/0001-PLAN-scaffold-charm-tui-library.md) | complete |

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](architecture.md) |
| know which Charm version to use | [0001-MADR, §3](decisions/0001-MADR-scaffold-charm-tui-library.md#3-toolchain-and-dependencies) |
| know what this module may import, and what it never may | [AGENTS.md, Dependencies](../AGENTS.md#dependencies) |
| know who owns the screen, the output and ctrl+c | [AGENTS.md, TUI conventions](../AGENTS.md#tui-conventions) |
| test a component across colour, charset and width | [AGENTS.md, TUI conventions](../AGENTS.md#tui-conventions), rule 6 |
| add a package | write its MADR and PLAN first: [AGENTS.md](../AGENTS.md#madr-and-plan-before-mutating-work) |
| run the checks before committing | `make pre-add-check`; see [AGENTS.md, Pre-add checks](../AGENTS.md#pre-add-checks) |
| know why the gates fail on an empty module | [0001-MADR, §8](decisions/0001-MADR-scaffold-charm-tui-library.md#8-push-and-identity) and [Consequences](decisions/0001-MADR-scaffold-charm-tui-library.md#consequences) |
| know where the TUI rules came from | [0001-MADR, §6](decisions/0001-MADR-scaffold-charm-tui-library.md#6-repository-conventions-for-tui-packages): ocp-login's records |
| know how this scaffold differs from go-core-lib's | [0001-MADR, §5](decisions/0001-MADR-scaffold-charm-tui-library.md#5-deliberate-differences-from-go-core-lib) |
