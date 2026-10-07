---
status: accepted
date: 2026-10-07
decision-makers: owner
consulted: 0014-REPORT-api-assessment-and-integration-research.md, 0003-REPORT-agent-tui-ecosystem-research.md (§11), 0012-MADR-bring-your-own-cli.md, 0013-MADR-cli-integration-helpers.md, Codex, Grok, goose, opencode, Kilo Code and Pi sources, the Go CLI frameworks' sources
informed: pi-go, gobble
---
# Make go-tui-lib's API natively integrable from any Go CLI: native types and component forms, hardening, and canonicalization during `v0`

## Context and Problem Statement

[0012-MADR-bring-your-own-cli.md](0012-MADR-bring-your-own-cli.md) made
go-tui-lib a TUI layer that a Go program stacks on its own CLI.
[0013-MADR-cli-integration-helpers.md](0013-MADR-cli-integration-helpers.md),
still proposed, designs `launch`: decide between a TUI and plain output, run
the TUI on the program's streams, fall back to the CLI mode, and map the
exit.

On 2026-10-07 the owner widened the aim:

> i want a wide, broadly compatible scope to integration helpers, much like
> i want a broadly compatible, flexible, extensible, api for the go-tui-lib
> library. integration helpers should have fully native API support backing
> them. examples of go cli programs that would want to leverage this library
> could be using native go terminal, cobra, kong, or other common go cli
> frameworks.

They also asked for an API assessment, research on how others build these
layers, and ideas for future-proofing, hardening, standardizing and
canonicalizing. The evidence is
[0014-REPORT-api-assessment-and-integration-research.md](../reports/0014-REPORT-api-assessment-and-integration-research.md).
In short:

* **0013's fallback has two holes.**
  * A Loop command run after the program has ended never runs, and its
    caller waits for ever (REPORT I1).
  * Modes set outside Bubble Tea, such as the prober's mode 2031, survive a
    crash (I2).
* **The library has no native form for most of what a CLI touches.** It
  lacks:
  * glyphs, profile and size for `workspace` (I3);
  * plain rendering (I4);
  * a result writer, exit statuses, an argument-vector parser and a
    parameter description in `command` (I5–I8);
  * enum text for flags (I9);
  * environment-only capabilities (I10);
  * byte forms of the terminal services (I11);
  * a testing kit (I12).
* **Native support needs no framework import.** One standard-library flag
  type, one tagged struct, one method-set stream source and an
  `ExitCode()` convention fit stdlib `flag`, cobra/pflag, kong, urfave/cli
  v3, ff v4, go-flags and go-arg, with one line of glue at most. Each was
  tested in a scratch program (REPORT §3).
* **The same names already mean different things** across packages, and
  0013 and the planned 0007–0009 add more (C1).
* **Hardening:** eleven small, independent findings (H1–H11).
* **Nothing guards the API between releases:** no API diff, no deprecation
  policy, no stability tiers (REPORT §2.6).
* **The landscape confirms the foundations.** Charm v2 is dominant in Go;
  every active Go agent tool found is on it. tcell v3 is the only
  modernised alternative. The field has converged on main-screen
  rendering with native scrollback, which the owner already put in scope
  as `inline` ([0003-REPORT](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
  §11.5).

How should go-tui-lib grow its API so that every package offers native
surfaces to a program's own CLI, while it stays a TUI layer that imports no
CLI framework, and becomes harder to misuse and easier to keep stable?

## Decision Drivers

* **Native, not adapted.** A program in any common Go CLI framework uses
  go-tui-lib's types directly in its flags, streams, errors and tests. Any
  glue is one line, in the program.
* **No CLI framework in the build.** 0012's depguard rule stands.
* **The caller owns streams, signals, screen and exit** (`AGENTS.md`, TUI
  conventions 1 and 2). The library spawns no process
  ([0003-REPORT](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
  §11.5).
* **The fallback promise holds in every outcome** (0013, owner answers 6
  and 8).
* **Stable for its consumers.** pi-go and gobble get a migration window
  for every rename, and a breaking change cannot slip into a release
  unnoticed.
* **One name, one meaning,** across the library.
* **Bounded and fuzzed** wherever input comes from a terminal, a file, an
  agent or a remote client.
* **The planned records still fit:** `termmode`, `inline`, `transcript`,
  `composer`, `agentui`, theme v2 and the harness
  ([0003-REPORT](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
  §11).

## Considered Options

* **A. A staged programme.**
  1. Policy gates.
  2. Native integration, through an amended 0013.
  3. Native forms in each package.
  4. Hardening.
  5. Canonicalization with deprecation aliases.

  Future-proofing goes into the planned records.
* **B. 0013 alone,** with the rest handled as it comes up.
* **C. Per-framework adapter modules,** nested modules as 0010 allows, one
  per framework.
* **D. Freeze `v0`,** and redesign everything in one `v1` cut.
* **E. A different foundation:** tcell v3, or a renderer of our own.

## Decision Outcome

Chosen option: **"A. A staged programme"**. It is the only option that
closes 0013's holes, gives every package a native surface, and keeps the
API stable for pi-go and gobble while it changes. B leaves the gaps. C
contradicts 0012 for glue the probes measured at three lines or fewer. D
lets the collisions grow until `v1`. E discards the stack the whole Go
field builds on.

The programme has six workstreams. W1 is 0013, amended. Each of the others
gets its own PLAN under this number, with a slug for its scope, written
when this record is accepted. W5 names records rather than deciding their
designs.

### W0. Policies and gates (first)

1. **Deprecation with aliases.**
   * A renamed function, type or constant keeps its old name for one
     minor release, as a wrapper, a type alias or a constant, marked
     `// Deprecated: use X`. It is removed in the next minor release.
   * A renamed struct field keeps both fields for one minor release, and
     the code reads the new field, falling back to the old.
   * Release notes list every deprecation and every removal.
2. **An API diff gate.**
   * `apidiff` (`golang.org/x/exp/cmd/apidiff`) compares the exported API
     with the latest release tag, in CI and in `make release-check`.
   * An incompatible change fails the gate unless the PLAN being executed
     lists it.
   * The tool runs through `go run` at a pinned version. It is never added
     to `go.mod`, so it enters no consumer's build. golang.org/x/exp has
     no tagged releases, so the pin is a pseudo-version, named in the
     Makefile.
3. **One name, one meaning.**
   * A glossary in `docs/` names each exported concept once.
   * `internal/conformance` gains a check that fails when two packages
     export a type with the same name. Per-package option types (`Option`)
     and names the glossary lists as deliberate are allowed.
   * New records take their names from the glossary.
4. **Conventions, written into `AGENTS.md`:**
   * **Options:** `With…`, `Without…` and `On…` functions, over opaque
     option types.
   * **Enums:** every exported enum has `String`, `MarshalText` and
     `UnmarshalText`, with stable lowercase tokens, through one
     `internal/enum` helper.
   * **Errors:** sentinel `Err…` values, typed errors with `Unwrap`, and
     `ExitCode() int` on an error that ends a CLI.
   * **Hooks:** an interface for each hook, with a `…Func` adapter.
   * **Constructors:** `NewX`, with `MustNewX` where a failure is a
     programming error.
   * **The environment:** `termcap.Env` and nothing else. Its lookups
     ignore case on Windows.
5. **Conformance, extended.**
   * The scan's must-read list comes from `go list` rather than a
     constant (H8).
   * The scan also refuses `os.Getenv`, `os.LookupEnv`, `os.Environ`,
     `os.Exit` and `os/exec` outside tests. That enforces the no-spawn
     rule and the environment convention in every package.
   * The one use today is `tuitest`'s update switch, which reads
     `TUITEST_UPDATE` (`tuitest/tuitest.go:127`). `tuitest` runs only
     under `go test`, so the scan lists it as the one allowed exception.
     A search of the non-test Go files found no other call.
6. **Stability lines.** Each package's documentation says whether it is
   stable, experimental or internal-facing.
7. **The toolchain floor.**
   * The library requires the Go release in `go.mod`, today 1.27.1.
   * It moves only by a record, and it moves to the newest patch of a
     release Go still supports.
   * Features that need the floor, such as JSON v2 and the generic method
     `PaneAs`, are named in that record.

### W1. Native integration surface (0013, amended)

0013 gains an amendment, A1. It changes these things, among others:

* **The fixes.**
  * **I1:** `Run` attaches a registry for the program's lifetime and
    detaches it on every outcome, and `command`'s loop learns when its
    program is gone.
  * **I2:** a `Restorer` seam. `*termcap.Prober` already satisfies it.
    `Run` writes its bytes before restoring the terminal, on every
    outcome.
  * An `OnStart(*tea.Program)` hook, so an agent goroutine can `Send`.
* **A flag type.** `launch.Mode` (`auto|tui|plain`) has the method set
  REPORT §3.2 measured: `String`, `Set`, `Type`, `Get`, `MarshalText`,
  `UnmarshalText`, `UnmarshalFlag`, `MarshalFlag`.
* **A tagged struct.** `launch.Flags{Mode Mode; TUI bool}` carries kong,
  go-flags and ff tags, and has a `RegisterFlags(*flag.FlagSet)` method
  for stdlib, cobra (`AddGoFlagSet`) and ff.
* **Streams.**
  * `launch.FromSource` takes any value with `InOrStdin`, `OutOrStdout`
    and `ErrOrStderr`, which is what `*cobra.Command` has.
  * Every other framework writes a one-line `Streams{…}`.
* **The exit.** `ExitCode` honours `interface{ ExitCode() int }` first.
* **Glyph tiers.** The glyph choice becomes a `glyph.Tier`.
* **Reasons.** They become stable text tokens.
* **No child process.** The colour profile comes from
  `colorprofile.Env` and the terminal test, so `launch` spawns nothing.
* **Testing.** A `launch/launchtest` package with fake terminals.
* **Collision-free names.** 0013's `Policy`, `Route` and `Want` take names
  from the glossary.
* **Tier-1 framework examples** gate each release.

### W2. Native forms in every package

| Package | Adds | Closes |
| :--- | :--- | :--- |
| `command` | `Registry.ParseArgs(id, []string, Origin)`; `Params(Command)` and `Registry.Complete`; `WriteResult(io.Writer, Result, Format)`, sharing `CallMCP`'s encoder; `ExitCode()` on `ErrUnknown`, `ErrUnavailable`, `ErrRefused` and `*ArgError`; `ErrPanicked`, with `recover` around every handler; a loop that can be detached, and a `Done` channel; `WatchContext`; `GateFunc`, `AllowIf`, `ArgsOf[A]`, `MustNew[A]` | I1, I5–I8, I14, I15, H2 |
| `workspace` | `WithGlyphs`, `WithProfile`, `WithBackground`, `WithSize`; the first theme built through the builder; an optional `PlainViewer` interface and `RenderPlain(width)`; help styles from the glyph set | I3, I4, I13 |
| `termcap` | `EnvCaps(env, goos) Caps`, the environment's facts without a program | I10 |
| `termsvc` | byte-returning forms of a notification and a copy; context-aware backends | I11, H6 |
| `glyph`, `theme` | `glyph.Tier` (Unicode, legacy console, ASCII) and `Tier.Set()` (A1.2), with `For(bool)` kept; enum text forms | I9, L1 |
| `tuitest` | `Case.Profile()` and `Case.Glyphs()`; an assertion that every line fits the width in cells | C9 |

The CP437 glyphs themselves, the theme roles and the golden matrix's
legacy row belong to theme v2.

### W3. Hardening

* **H1:** bound the workspace's view cache to the current generation and
  one size per pane.
* **H3:** clamp the window size at a documented `MaxCells`.
* **H4:** `LoadDir` limits on file size, file count and depth.
* **H5:** a limit on argument JSON, and a fuzz target for argument
  preparation and masking.
* **H10:** saturating arithmetic in `layout`'s `Ratio` and `Percent`.
* **H11:** the mutable package variables become functions or options.
* **C6:** one `internal/sanitize` for every untrusted string, and
  workspace titles and badges flattened and sanitized.
* **H9:** fuzz targets for:
  * `termcap`'s replies, `ParseTmux` and its text forms;
  * `termsvc`'s `ParseActivity`, `Link` and `SanitizeTitle`, checking
    that no control character survives;
  * `layout.State`;
  * `workspace`'s `clip` and `Render`, checking that no line exceeds its
    pane;
  * `launch`'s environment parsing.

### W4. Canonicalization, with deprecation aliases

* **Renames** (W0.1 transition):
  * `command.Decision` becomes `Verdict`, with the gate's constants;
  * `Request.Context` and `Invocation.Context` become `WhenContext`
    (A1.1; the text first said `When`);
  * `termcap.New` becomes `NewProber`;
  * `layout`'s preset options become `With…`;
  * `workspace`'s `On…` and `With…` follow W0.4.
* **Opaque options** for the five option types that expose their structs.
  The type names stay; only a third party's own option function breaks,
  and the release notes say so.
* **Smaller items:**
  * `internal/enum` replaces the three enum implementations;
  * one message helper (C5);
  * JSON v2 throughout `command` and `termcap`;
  * `when.SyntaxError{Offset}`, and `termcap.ErrUnknownName`;
  * `…Func` adapters for every hook;
  * one documented width method;
  * the stale `--args` wording.
* **Planned records first.** 0007, 0008 and 0009 are amended to the
  glossary while they are unexecuted, which costs nothing (REPORT C1).

### W5. Future-proofing: the records it feeds

Each is its own record. This one decides only that they are needed, and
what they must plug into.

* **`termmode`** ([0003-REPORT](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
  §11.2, item 11) owns the mode plan, the restore bytes and the Windows
  console helpers. It plugs into W1's `Restorer` seam.
* **`inline`** (item 12): main-screen rendering with native scrollback.
  `launch` passes it through unchanged.
* **`transcript`** (item 6) is to define a transcript as an event model
  independent of rendering. One source then feeds:
  * the TUI;
  * the plain and JSONL output;
  * a linear, accessible form;
  * an ACP client.

  This is the shape of Codex's app-server, Grok's ACP and Pi's session
  stream (REPORT §4.2).
* **`composer` and `agentui`** (items 13 and 7) include a prompter that
  works in every mode. In print mode it answers with a policy; in RPC mode
  a dialog becomes a request (Pi, Grok, goose).
* **The harness** (item 10) grows from `launchtest` to the `x/vt`
  emulator and a PTY module, which the owner already allowed.
* **Theme v2** (item 9) takes the CP437 tier contents and role tokens
  (REPORT F7).
* **Later `termcap` additions:**
  * an opt-in for mode 2048;
  * an override for synchronized output over SSH, which Bubble Tea leaves
    off unless it recognises the terminal (REPORT §5.2);
  * OSC 7501 behind an option, once terminals support it;
  * an accessibility signal for W1's decision.
* **A `Component` interface**, `Update(tea.Msg) tea.Cmd` plus `View`,
  shared by `Prober`, `Notifier`, `Workspace` and the planned widgets.
  It is decided in `transcript`'s record, its first user.

### Release sequence

| Release | Workstreams |
| :--- | :--- |
| `v0.7.0` | W0's gates; W1; the parts of W2 that W1 needs (the detachable loop, `glyph.Tier`, enum text for `launch.Mode`) |
| `v0.8.0` | the rest of W2; W3 |
| `v0.9.0` | W4's new names beside the old, deprecated |
| `v0.10.0` | W4's old names removed |
| later | W5's records, in 0003-REPORT §11.4's order |

### What this changes in other records

* **0013:** amendment A1 (W1). Its §12 exclusions of a result writer,
  plain rendering and command exit statuses move here, to W2. Its PLAN is
  revised after A1 is accepted.
* **0007, 0008, 0009:** amended to the glossary (W4).
* **0003-REPORT §11's roadmap** stands, with W5's additions.
* **`AGENTS.md`:** the conventions of W0.4 and the conformance bans of
  W0.5, in W0's PLAN.

### Consequences

* Good, because a program in any common Go CLI framework uses
  go-tui-lib's own types in its flags, streams, errors and tests, and the
  library still imports no framework.
* Good, because 0013's fallback holds in every outcome once I1 and I2 are
  closed.
* Good, because the API diff gate and the alias policy let the API change
  during `v0` without surprising pi-go or gobble.
* Good, because collisions are stopped by a check rather than by review.
* Good, because input from terminals, files, agents and remote clients is
  bounded and fuzzed.
* Bad, because the programme spans four minor releases, and W4 deprecates
  names that consumers use today.
* Bad, because opaque options break any option function a third party
  wrote, which no alias can cover.
* Bad, because the conformance bans and the collision check reject code
  that compiles. Each exception has to be listed.
* Neutral, because W5 decides no designs; each of its records still needs
  its own decision.

### Confirmation

Each workstream's PLAN carries its tests, mutations and gates. Across the
programme:

* the API diff gate fails on a planted incompatible change in a scratch
  copy, and passes on the release's listed changes;
* the collision check fails on a planted duplicate type name;
* the conformance bans each fail on a planted call;
* the tier-1 framework examples (flag, cobra, kong, urfave/cli v3) are
  compiled and run in a scratch module on every release;
* every deprecated name still compiles for one minor release, and is gone
  in the next;
* the hardening findings each have a test that failed before the fix.

## Pros and Cons of the Options

### A. A staged programme

* Good, because each workstream is small enough to review, and W1 ships
  first.
* Good, because it follows the roadmap the owner already chose.
* Bad, because it is long, and its parts depend on one another (W1 needs
  pieces of W2).

### B. 0013 alone

* Good, because `launch` ships soonest.
* Bad, because the CLI mode still lacks native forms for results, exits,
  arguments and plain rendering, and the holes I1 and I2 need `command`
  and `termcap` changes anyway.

### C. Per-framework adapter modules

* Good, because each framework gets an exact fit, such as a kong
  `TypeMapper` for a tri-state flag.
* Bad, because it brings back what 0012 retired, for glue the probes
  measured at three lines or fewer (REPORT §3.2).

### D. Freeze `v0`, redesign for `v1`

* Good, because there is one migration.
* Bad, because the collisions and gaps grow while pi-go and gobble build
  on them.

### E. A different foundation

* Good, because tcell v3 is modern and has broad protocol coverage.
* Bad, because Charm v2 carries the Go ecosystem (REPORT §5.1), the
  library's API is built on it, and a renderer of our own repeats work the
  `inline` record can do on top of Bubble Tea.

## Amendments

### A1 (2026-10-07): corrections found while writing the PLANs

The owner accepted this record on 2026-10-07. Writing its PLANs found five
places where the text could not be built as written, or where a name
clashed. Each is corrected here; the rest of the record stands.

1. **`Request.Context` and `Invocation.Context` become `WhenContext`, not
   `When`** (W4, the owner's answer).
   * `When` already names a when-expression string:
     * `Command.When` (`command/command.go:89`);
     * `WithWhen(expr)` (`command/args.go:93`);
     * `layout.Rule.When` (`layout/layout.go:154`);
     * and the planned 0007 and 0008 rules.
   * `workspace` already calls the `when.Context` concept `WhenContext()`
     (`workspace/context.go:12,52`).
   * The owner chose `WhenContext` over `When` and over keeping `Context`.
2. **`glyph.For(Tier)` cannot exist beside `For(bool)`** (W2).
   * Go has no overloading.
   * A generic `For[T bool | Tier]` compiles, but `apidiff` reports a
     function turning generic as incompatible, because `f := glyph.For`
     stops compiling.
   * The tier's glyph set is therefore `Tier.Set()`, and `For(bool)`
     stays.
3. **The loop's done channel stays inside `command`** (W2). `Attach`
   returns only `detach`. The fallback to the caller happens inside `Run`,
   so a program never needs the channel.
4. **0013's `Filter` option is `WithFilter`,** under W0.4's naming rule.
   It is also the fourth exported `Filter` among the accepted and planned
   records.
5. **H11's package variables** are deprecated in `v0.8.0` with W3, and
   removed in `v0.9.0`. A variable cannot become a function of the same
   name, so W0.1's one-minor transition is the only route.

**The exit statuses of `command`'s errors** (W2), which the record left
open, follow 0006-MADR §10's codes:

| Error | Status |
| :--- | ---: |
| `ErrUnknown` | 2 |
| `*ArgError` | 2 |
| `ErrUnavailable` | 1 |
| `ErrRefused` | 3 |
| `ErrPanicked` | 2, matching 0013 §9's panic status |

## More Information

### Owner answers, 2026-10-07

The owner accepted this record on 2026-10-07: "accepted, write the actionable and comprehensive plans".

Picked from options, each the recommendation:

1. **Records:** a REPORT and this umbrella MADR. 0013 is amended and stays
   the `launch` record. The alternatives were:
   * one MADR superseding 0013;
   * widening 0013 alone.
2. **Breaking changes during `v0`:** now, with deprecation aliases and an
   API diff gate. The alternatives were:
   * one `v1` cut;
   * additive changes only, with a glossary for the collisions.
3. **Framework support:**
   * Tier 1, gated: stdlib `flag`, cobra, kong and urfave/cli v3. By
     pkg.go.dev counts, these cover nearly all importers of the
     frameworks surveyed.
   * Tier 2, documented with the probe results: ff v4, go-flags and
     go-arg.

   The alternatives were all seven in tier 1, or flag and cobra alone.

### Applied from an earlier owner decision

The library spawns no process
([0003-REPORT](../reports/0003-REPORT-agent-tui-ecosystem-research.md)
§11.5). 0013 §5 had `launch` call `colorprofile.Detect`, which runs `tmux
info` inside tmux (`colorprofile@v0.4.3/env.go:248`). Under this record,
`launch` uses `colorprofile.Env`, which reads only the environment, and the
conformance ban of W0.5 makes the rule a check. Refining the profile at run
time stays with `termcap`, whose `TmuxQuery` returns the command for the
program to run.
