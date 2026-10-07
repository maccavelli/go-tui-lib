---
status: accepted
date: 2026-10-07
decision-makers: owner
consulted: 0006-MADR-command-registry.md (§1, §10, A1, A10, A11, A12), 0010-MADR-nested-adapter-modules.md (§1, §3, §5, §6), 0011-MADR-docs-accuracy-after-v0-5-0.md, the Go module reference on deprecation and retraction
informed: pi-go
---
# Make go-tui-lib a TUI layer a program stacks on its own Go CLI, and retire the library's CLI front ends

## Context and Problem Statement

[0006-MADR-command-registry.md](0006-MADR-command-registry.md) gave the
command registry three shell front ends:

* `command/cli` (§10), the standard-library one, in the root module since
  `v0.4.0`;
* `command/cobracmd` (A1, A10), a nested module released as
  `command/cobracmd/v0.1.0`;
* `command/kongcmd` (A1, A11), a nested module released as
  `command/kongcmd/v0.1.0`.

To serve `kongcmd`, A12 made `command.New[A]` refuse an argument struct
that Kong would read differently, released in `v0.5.0`.
[0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md)
gave the two adapters their modules, tags and gates.

On 2026-10-07, after all four releases, the owner said what the library
is for:

> So if we rip out the kong and cobra modules and focus only on creating
> go-tui-lib as an idempotent widely compatible broad api tui layer than
> can be integrated into go cli tooling that is actually what I was
> wanting. Bring your own go cli, stack on go-tui-lib and now you have a
> tui in addition to the native cli.

A program already has its CLI, in Cobra, Kong, urfave/cli or the
standard `flag` package. go-tui-lib should add a TUI beside it, and stay
out of the CLI. The library's own front ends do the opposite: they build
the program's CLI from the registry. The rest of the library already
fits.

* The TUI packages (`workspace`, `layout`, `theme`, `glyph`, `termcap`,
  `termsvc`, `tuitest`) import no CLI framework, and leave the screen,
  the signals and the writers to the caller.
* The registry runs the TUI's actions from keys, the palette, slash
  commands and agents. A program's own CLI runs the same actions by
  calling `Registry.Run` with `command.OriginCLI` from its own handler,
  with no adapter.

Two facts bound the change:

* A published version cannot be withdrawn. The module proxy and the
  checksum database keep `command/cobracmd/v0.1.0` and
  `command/kongcmd/v0.1.0`. A module is deprecated by a `// Deprecated:`
  comment on its `module` line, and a version is retracted by a `retract`
  directive, each in a later version of that module (the Go module
  reference, "Deprecation" and "retract directive").
* Removing an exported package, or relaxing `New[A]`, changes the root's
  API. In `v0` that is a minor release.

## Decision Drivers

* The program owns its CLI. The library gives it a TUI, and never a CLI.
* No CLI framework reaches the library's build or the program's
  `go.sum` through it.
* An abandoned module says so to its users, through the go command.
* One way to run an action from the shell: the program's own handler
  calls the registry.

## Considered Options

* Retire the CLI front ends: retract the two adapter modules, remove
  them, `command/cli` and A12's check, and document the program's own
  CLI as the shell surface.
* Keep the front ends as optional extras beside a bring-your-own-CLI
  guide.
* Keep `command/cli` alone, as a zero-dependency option.

## Decision Outcome

Chosen option: "Retire the CLI front ends", because the library's
purpose is a TUI that a program adds to the CLI it already has.

1. **The two adapter modules are retracted, then removed.** Each gets a
   last version, `command/cobracmd/v0.1.1` and `command/kongcmd/v0.1.1`,
   whose `go.mod` marks the module `// Deprecated:` with a pointer to
   this record, and retracts `[v0.1.0, v0.1.1]`. Then the two
   directories leave `main`, with their `go.work` entries. The tags stay,
   as the proxy requires.
2. **`command/cli` is removed** from the root. A program's own CLI
   handler calls `Registry.Run` with `command.OriginCLI`, and a guide
   shows how for the standard `flag` package, Cobra and Kong. The guide's
   examples are compiled in a scratch module; the library imports none
   of those frameworks.
3. **A12's check is removed.** `New[A]` goes back to `v0.4.0`'s rule: the
   `json` tag decides a property's name and whether it is required. The
   Kong-style tags (`help`, `arg`, `enum`, `default`, `short`, `hidden`,
   `placeholder`, `group`, `schema`) stay the schema's vocabulary. The
   `required:""` and `optional:""` tags `v0.5.0` added to two `workspace`
   structs are removed, since nothing reads them.
4. **depguard refuses Cobra, pflag and Kong everywhere,** as it refuses
   fang: the library never imports a CLI framework. glamour keeps its
   one-module rule for `stream/glamourmd`, still planned under
   [0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md).
5. **`command.SurfaceCLI` and `command.OriginCLI` stay.** They now mean
   "the program's own CLI": a command can still be kept off it, and a
   request from it is still audited and gated as a shell request.
6. **The release is `v0.6.0`.** It removes `command/cli` and A12's check,
   and nothing else in the API.
7. **The documents say what the library is:** a TUI layer for a Go
   program that brings its own CLI. The design of further integration
   helpers is a later record. Those might launch the TUI from a CLI
   subcommand with the caller's streams, or choose between interactive
   and plain output.

### What this supersedes

* In [0006-MADR-command-registry.md](0006-MADR-command-registry.md): §10's
  `command/cli`; A1's Cobra and Kong front ends, and its "one conformance
  test for three front ends"; A10 and A11 entirely; A12 entirely. §4's
  tag vocabulary, as A1 aligned it, stays. A13 in that record points
  here.
* In [0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md):
  the two command adapters of §1, their dependencies in §5, and the Cobra
  and Kong rules of §6. The nested-module layout, `go.work`, the
  per-module gates and the two-step release stay, for
  `stream/glamourmd`. A3 in that record points here.
* [0011-MADR-docs-accuracy-after-v0-5-0.md](0011-MADR-docs-accuracy-after-v0-5-0.md)
  is not superseded. Its PLAN pauses until this one is complete, and then
  drops the findings this record makes moot.

### Consequences

* Good, because a program adds a TUI without the library touching its
  CLI, and no CLI framework enters its build through go-tui-lib.
* Good, because the registry has one shell path, the program's own
  handler, which is the same call an agent or a key makes.
* Good, because `go get @latest` and `go list -m -u` tell a user of an
  adapter that it is retracted and why.
* Bad, because two modules released the day before are withdrawn, and
  `v0.6.0` removes an exported package and relaxes `New[A]` within a day
  of `v0.5.0`.
* Bad, because a program that wanted the generated CLI writes its own
  handlers. The guide shows the few lines each framework needs.
* Neutral, because the 0006 and 0010 records keep their history; the
  superseded parts are marked, not rewritten.

### Confirmation

[0012-PLAN-bring-your-own-cli.md](0012-PLAN-bring-your-own-cli.md)
verifies:

* the retraction through the proxy from a scratch module;
* the removal by the gates and `scripts/go-modules.sh --check`;
* `v0.6.0`'s API by `go doc`;
* the guide's examples by compiling them;
* the consumer smoke test against `v0.6.0`.

## Pros and Cons of the Options

### Retire the CLI front ends

* Good, because it matches the library's purpose.
* Bad, because it withdraws released modules and API.

### Keep the front ends as optional extras

* Good, because nothing released is withdrawn.
* Bad, because the library keeps owning a CLI it should not, and two
  modules of maintenance follow it.

### Keep `command/cli` alone

* Good, because it costs no dependency.
* Bad, because it is still the library's CLI, a second shell path beside
  the program's own.

## More Information

* **Owner answers, 2026-10-07** (picked from options), each the
  recommendation:
  * retract, then remove the adapters;
  * remove `command/cli`;
  * remove A12's check;
  * this record for the removal, and a later one for integration
    helpers.

  The alternatives were:
  * deprecate without retracting, or just remove;
  * keep `command/cli` as an option;
  * keep A12's check, or make it opt-in;
  * design the integration helpers in this record too.
