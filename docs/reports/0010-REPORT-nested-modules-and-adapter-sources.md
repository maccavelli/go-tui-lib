---
date: 2026-10-02
subject: the facts behind shipping go-tui-lib's Cobra, Kong and glamour adapters as nested Go modules with a committed go.work, and what each adapter's upstream source does that the library's rules must answer
examines: "the Go module reference, toolchain and workspace documentation and the Go Modules wiki (read from go.googlesource.com source); golang/go issues 50750 and 73655; actions/setup-go, golangci-lint and golangci-lint-action, golang/vuln and Dependabot documentation; charmbracelet/x, open-telemetry/opentelemetry-go, aws/aws-sdk-go-v2, kubernetes/kubernetes, hashicorp/vault, uber-go/zap, testcontainers-go, DataDog/dd-trace-go and golang/tools at HEAD on 2026-10-02; module source of github.com/spf13/cobra v1.10.2, github.com/spf13/pflag v1.0.10, charm.land/fang/v2 v2.0.1, github.com/alecthomas/kong v1.16.1 and charm.land/glamour/v2 v2.0.1; scratch experiments with go1.27.1, golangci-lint 2.14.0 and govulncheck 1.8.0"
associated-madr: "0010-MADR-nested-adapter-modules.md"
---
# Nested modules, a committed go.work, and the adapters' upstream sources

This report records findings. It decides nothing. The decisions it supports
are [0010-MADR-nested-adapter-modules.md](../decisions/0010-MADR-nested-adapter-modules.md),
amendment A1 of [0006-MADR-command-registry.md](../decisions/0006-MADR-command-registry.md)
and amendment A2 of [0009-MADR-streaming-content-engine.md](../decisions/0009-MADR-streaming-content-engine.md).

## Why, and how

On 2026-10-02 the owner asked whether the library is neutral between Cobra,
Viper, Kong or no framework, and said it "needs to support, and be fully
optimized for kong". Offered packages or nested modules for the adapters,
the owner decided:

> Nested modules. Glamour as nested module. Go.work in repo. Update the docs
> with facts grounded in sources and web research, best practices and
> idioms.

Three read-only passes ran on 2026-10-02:

- **Go's documentation and precedent repositories.** The Go documentation
  was read from its source repositories (`go.googlesource.com/website`,
  `_content/ref/mod.md` and others, and `go.googlesource.com/wiki`,
  `Modules.md`), so quotations are exact. Precedent repositories were
  shallow-cloned at HEAD.
- **Upstream module source.** Each adapter's dependency was downloaded at
  its latest version with `go mod download` into a scratch module cache,
  and read at the file and line cited.
- **Experiments.** Scratch modules, built with go1.27.1, tested what the
  documentation leaves open. No experiment touched this repository.

"Not verified" marks what was neither read in a source nor observed.

## 1. Multi-module repositories

- **The default is one module per repository.** The Go Modules wiki says
  "it is almost always easier and simpler to manage a single-module
  repository rather than multiple modules in an existing repository"
  (`Modules.md:1173`). It names the costs: "`go test ./...` from the
  repository root will no longer test everything in the repository", and
  "you might need to routinely manage the relationship between the modules
  via `replace` directives" (`Modules.md:1181-1182`).
- **The exception fits adapters.** A multi-module repository is reasonable
  "if you have a repository with a complex set of dependencies, but you
  have a client API with a smaller set of dependencies" (`Modules.md:1190`).
  The module reference calls nested modules "typically done for large
  repositories with multiple components that need to be released and
  versioned independently" (`ref/mod.md:3421-3427`).
- **New directories are the easy case.** "Add the package and the go.mod in
  the same commit, tag the commit, and push" (`Modules.md:1198`). Carving a
  module out of a package path an earlier tag already published risks an
  "ambiguous import" (`Modules.md:1213`). The only tag here, `v0.1.0`, holds
  `glyph`, `internal`, `layout`, `scripts`, `theme`, `tuitest` and
  `workspace` (`git ls-tree -d v0.1.0`), so every planned adapter directory
  is new.
- **Module path and tags.** A nested module's path is the repository's
  path plus its subdirectory, and "the module subdirectory … also serves as
  a prefix for semantic version tags" (`ref/mod.md:57-60`). "Each tag name
  must be prefixed with the module subdirectory, followed by a slash"
  (`ref/mod.md:3336-3342`), for example `gopls/v0.4.0`. A consumer still
  writes the version without the prefix in its `require` line
  (`doc/modules/managing-source.md:153-161`).
- **Major versions.** "Major version suffixes are not allowed at major
  versions `v0` or `v1`" (`ref/mod.md:241`). A later `v2` of a nested module
  lives at `<dir>` or `<dir>/v2` and is tagged `<dir>/v2.x.y`
  (`ref/mod.md:3429-3434`, `3344-3347`).
- **Tags are permanent.** "Once a tag is created, it should not be deleted
  or changed" (`ref/mod.md:3355-3359`).
- **The root module's zip excludes nested modules.** "Module zip files do
  not include the contents of `vendor` directories or any nested modules"
  (`ref/mod.md:3456-3458`).
- **Internal packages.** A nested module may import the root's `internal/`
  packages, because "packages in one module are allowed to import internal
  packages from another module as long as they share the same path prefix
  up to the internal/ path component" (`Modules.md:1255`). Because minimal
  version selection can pair an adapter release with any newer root, an
  adapter that did so could break when the root changed an internal API.
  That is an inference, not a documented rule.

## 2. Module graph pruning, measured

- **The rule.** For a main module at `go 1.17` or later, "the module graph
  used for minimal version selection includes only the immediate
  requirements for each module dependency that specifies `go 1.17` or
  higher", and "modules whose requirements have been pruned out still
  appear in the module graph and are still reported by `go list -m all`"
  (`ref/mod.md:1157-1163`, `1175-1177`).
- **Experiment.** Two libraries were published through a file-based
  `GOPROXY`, each with a package `plain` and no other import. In `flatlib`,
  the root `go.mod` required `github.com/alecthomas/kong v1.16.1`, which
  only a second package imported. In `nestlib`, the root required nothing,
  as if Kong lived in a nested module. A consumer imported only
  `<lib>/plain`, then ran `go get` and `go mod tidy`:

  | | Consumer `go.mod` | Consumer `go.sum` | `go list -m all` |
  | :--- | :--- | :--- | :--- |
  | `flatlib` | the library only | the library, and `github.com/alecthomas/kong v1.16.1/go.mod` | the library and Kong |
  | `nestlib` | the library only | the library only | the library only |

  A requirement of the root module therefore reaches every consumer's
  `go.sum` and module graph, though it is never compiled or downloaded as
  source. Only a nested module keeps it out of both.

## 3. go.work

- **Committing it.** The module reference says, verbatim
  (`ref/mod.md:1285-1302`):

  > It is generally inadvisable to commit go.work files into version
  > control systems, for two reasons:
  >
  > - A checked-in `go.work` file might override a developer's own
  >   `go.work` file from a parent directory, causing confusion when their
  >   `use` directives don't apply.
  > - A checked-in `go.work` file may cause a continuous integration (CI)
  >   system to select and thus test the wrong versions of a module's
  >   dependencies. CI systems should generally not be allowed to use the
  >   `go.work` file so that they can test the behavior of the module as it
  >   would be used when required by other modules, where a `go.work` file
  >   within the module has no effect.
  >
  > That said, there are some cases where committing a `go.work` file makes
  > sense. For example, when the modules in a repository are developed
  > exclusively with each other but not together with external modules,
  > there may not be a reason the developer would want to use a different
  > combination of modules in a workspace. In that case, the module author
  > should ensure the individual modules are tested and released properly.

  `go help work`, the workspaces tutorial and the Go blog do not discuss
  committing it. The blog does recommend a workspace over `replace` for
  "multiple interdependent modules in the same repository"
  (`blog/get-familiar-with-workspaces.md:96-100`).
- **`GOWORK`.** "If `GOWORK` is set to `off`, the command will be in a
  single-module context. If it is empty or not provided, the command will
  search the current working directory, and then successive parent
  directories, for a file `go.work`" (`ref/mod.md:1237-1247`).
- **Commands that ignore the workspace.** "`go mod init`, `go mod why`,
  `go mod edit`, `go mod tidy`, `go mod vendor`, and `go get` always operate
  on a single main module" (`ref/mod.md:1234-1235`). golang/go#50750, "go
  mod tidy ignores go.work file", was closed as not planned on 2025-05-10:
  "a module can't be tidy unless it requires all of its dependencies".
- **`go` and `toolchain` lines.** "A workspace's `go` line must declare a
  version greater than or equal to the `go` version declared by each of the
  modules listed in `use` statements" (`doc/toolchain.md:156-157`).
  Toolchain selection "consults the `toolchain` and `go` lines in the
  current workspace's `go.work` file or, when there is no workspace, the
  main module's `go.mod` file" (`doc/toolchain.md:241-243`). "When a
  workspace is in use, `godebug` directives in `go.mod` files are ignored"
  (`ref/mod.md:1384`).
- **`go work use` and `go work sync`.** `use` "updates the go line in
  go.work to specify a version at least as new as all the go lines in the
  used modules" (`go help work use`), and "does not add modules contained in
  subdirectories of its argument directory" (`ref/mod.md:1391-1393`).
  `sync` rewrites each workspace module's `go.mod` "with the dependencies
  relevant to that module upgraded to match the workspace build list"
  (`ref/mod.md:2872-2874`).
- **`go.work.sum`** "keeps track of hashes used by the workspace that are
  not in collective workspace modules' go.sum files"
  (`ref/mod.md:1282-1283`).
- **Experiments, go1.27.1, a root module and a nested `cobra/` module in a
  workspace:**
  - `go test ./...` in `cobra/` passed in workspace mode while its `go.mod`
    required a root version that does not exist. `go mod tidy -diff` in
    `cobra/` then tried to download that version and failed.
  - With no `require` of the root at all, `go build ./...` in `cobra/`
    passed in workspace mode and failed with `GOWORK=off`: "no required
    module provides package".
  - `GOFLAGS=-mod=mod` is refused in workspace mode: "-mod may only be set
    to readonly or vendor when in workspace mode".

  A workspace therefore hides both a missing requirement and an
  unpublished one. Only a `GOWORK=off` run sees what a consumer gets.

## 4. Tools across modules

- **`./...` stops at a nested module, with or without go.work.** `go help
  packages`: wildcard patterns exclude "directories that contain a go.mod
  file". At the root, `go list ./...`, `go test ./...`, `go vet ./...`,
  `golangci-lint run ./...` and `govulncheck ./...` covered only the root
  module's packages in the experiment, and missed an issue planted in
  `cobra/`.
- **Workspace-wide patterns.** The pattern `work` means "all packages in
  the main module (or workspace modules)" (`go help packages`); `go list
  work` and `go vet work` covered both modules. golangci-lint reads `work`
  as a directory and fails. `go list -m` (without `all`) in workspace mode
  prints every workspace module, which golangci-lint-action's own
  "Go Workspace Example" uses to build a matrix
  (golangci-lint-action `README.md:126-153`).
- **golangci-lint configuration** is searched "in all directories from the
  directory of the first analyzed path up to the root"
  (golangci-lint `docs/content/docs/configuration/file.md:13`). Passing
  `-c` to the root's file is explicit.
- **govulncheck** documents running "from the module directory"
  (golang/vuln `cmd/govulncheck/doc.go:25-29`). It ran in workspace mode in
  the experiment; its documentation does not mention workspaces (not
  verified officially).
- **actions/setup-go** accepts "the go.mod, go.work, .go-version, or
  .tool-versions file" as `go-version-file` (`action.yml:8`), and reads a
  `go.work`'s `toolchain` or `go` line (`src/installer.ts:648-666`). Its
  module cache keys on `go.mod` by default (`README.md:28`), and
  `cache-dependency-path` "accepts glob patterns and multi-line values"
  (`docs/advanced-usage.md:258-278`), so `**/go.sum` covers every module.
- **Dependabot.** The `directories` key "supports globbing and the wildcard
  character `*`", and `group-by: dependency-name` creates "a single pull
  request for each dependency update across all specified directories"
  (GitHub's Dependabot options reference). dependabot-core gained go.work
  support in PR #14909 (2026-05-05); whether that is generally available on
  github.com is not verified. This repository has no Dependabot
  configuration today.

## 5. How other repositories do it

| Repository | go.work committed | Tags | CI per module | Lint and vuln |
| :--- | :--- | :--- | :--- | :--- |
| charmbracelet/x | no (`.gitignore:24-25`) | `ansi/v0.11.8`, one prefix per module | one generated workflow per module, `working-directory: ./ansi`, `go-version-file: ./ansi/go.mod`, ubuntu, macOS and Windows (`.github/workflows/ansi.yml:17-30`) | a reusable lint workflow per directory; no govulncheck seen |
| opentelemetry-go | no; generated by `make go-work` | `vX.Y.Z` and `sdk/metric/vX.Y.Z` | Make targets over `find . -name go.mod` (`Makefile:7`, `163-169`) | `cd $(DIR) && golangci-lint run`, `govulncheck ./...` and tidy per directory (`Makefile:220-253`) |
| hashicorp/vault | yes, with `go.work.sum` | `api/v1.23.0`, `sdk/v0.26.0` | `GO_CMD?=GOWORK=off go` (`Makefile:28`); workflows set `GOWORK: "off"` | tidy per `go list -m` directory with `GOWORK=off` (`scripts/go-helper.sh:110-116`) |
| kubernetes | yes, generated, with `go.work.sum` | staging modules are published elsewhere | `GOWORK=off` for tidy and pinning (`hack/update-vendor.sh:30-31`) | n/a |
| DataDog/dd-trace-go | yes (`.gitignore:3-5`) | `contrib/net/http/v2.12.0-dev.3` | local `replace` in every nested module (`contrib/net/http/go.mod:83`) | `GOWORK=off` in some jobs |
| uber-go/zap | no | `exp/v0.3.0` | `MODULE_DIRS` loop (`Makefile:10`, `54`) | `golangci-lint run --path-prefix` per module; tidy and diff per module (`Makefile:22-39`) |

- **The release order.** A nested module needs a root version that exists.
  Three patterns are in use:
  - **Lockstep tags with a permanent local `replace`:** opentelemetry-go
    (`sdk/go.mod:5`), testcontainers-go, zap's `exp`, aws-sdk-go-v2. Root
    and nested tags point at one commit.
  - **The wiki's recipe:** a temporary `replace` to test, dropped before
    tagging both on one commit (`Modules.md:1228-1243`).
  - **Published versions only:** charmbracelet/x's nested modules require
    only released versions (`vt/go.mod:7`), so a change that spans two
    modules is released in two steps. gopls keeps `replace
    golang.org/x/tools => ..` on its main branch (`gopls/go.mod:40`), and
    its release tag requires a published pseudo-version instead; how its
    tooling drops the `replace` is not verified.
  - golang/go#50750 recommends "adding a `replace` directive until there's
    a module version that could be properly `require`d".
- **Naming.** Adapter modules commonly take a package name that does not
  shadow the library they wrap: `otelhttp`, `otelgin`, `zapgrpc`. A package
  named `cobra` would collide with `spf13/cobra` at every import site; that
  is a convention, not a rule.
- **A committed go.work is harmless to consumers.** The proxy zip of
  `github.com/DataDog/dd-trace-go/v2@v2.10.1` contains `go.work` and
  `go.work.sum`, and no nested module's files; a `go.work` "within the
  module has no effect" for a consumer (`ref/mod.md:1295`).

## 6. Cobra and pflag

`github.com/spf13/cobra` v1.10.2 (2025-12-03) and `github.com/spf13/pflag`
v1.0.10 (2025-09-02).

- **Requirements.** Cobra: `go 1.15`; `cpuguy83/go-md2man/v2`,
  `inconshreveable/mousetrap`, `spf13/pflag`, `go.yaml.in/yaml/v3`. pflag
  requires nothing. Neither imports Charm. `cobra/doc` is a package of the
  same module; md2man, blackfriday and YAML are compiled only by importers
  of `cobra/doc`.
- **The API an adapter uses** (`command.go`): `Use`, `Aliases`, `Short`,
  `Long`, `Example`, `GroupID`, `ValidArgsFunction`, `Args`, `Deprecated`,
  `Annotations`, `RunE`, `PersistentPreRunE`, `Hidden`, `SilenceErrors`,
  `SilenceUsage` (64-247); `Group` and `AddGroup` (45, 1396); `SetArgs`,
  `SetOut`, `SetErr`, `SetIn`, `SetHelpFunc`, `SetFlagErrorFunc` (281-376);
  `ExecuteContextC` (1078). Completion: `CompletionFunc`,
  `ShellCompDirective`, `FixedCompletions`, `RegisterFlagCompletionFunc`
  (`completions.go:45-170`), and script generators that take an
  `io.Writer`. Flag groups: `MarkFlagRequired`, `MarkFlagsRequiredTogether`,
  `MarkFlagsOneRequired`, `MarkFlagsMutuallyExclusive` (`flag_groups.go:33-81`).
  pflag's `Value` interface supports custom types (`flag.go:210`); it has
  no enum type.
- **Behaviour the library's rules must answer:**
  - Without `SetArgs`, `ExecuteC` reads `os.Args[1:]` (`command.go:1105-1106`).
  - Writers fall back to `os.Stdout`, `os.Stderr` and `os.Stdin` when not
    set (`command.go:393-440`).
  - `os.Exit` is reached only through `cobra.CheckErr` (`cobra.go:235-238`),
    the Windows mousetrap when started from Explorer (`command_win.go:30-39`,
    disabled by the global `MousetrapHelpText = ""`), and `doc.GenYaml` on
    a marshalling error (`doc/yaml_docs.go:138-140`).
  - A failing `__complete` writes to `os.Stderr` directly
    (`completions.go:968`), which no setter redirects.
  - `checkCommandGroups` panics on an undefined `GroupID`
    (`command.go:1205-1210`); `MarkFlagsMutuallyExclusive` panics on an
    unknown flag (`flag_groups.go:69-71`).
  - `RegisterFlagCompletionFunc` stores into a process-wide map keyed by
    flag, with no delete, and refuses a second registration
    (`completions.go:38-41`, `178`).
  - `EnableTraverseRunHooks`, `EnablePrefixMatching` and
    `MousetrapHelpText` are process-wide variables (`cobra.go:55-72`).
  - `doc.GenMarkdown` stamps `time.Now()` unless `DisableAutoGenTag`
    (`doc/md_docs.go:112-113`); `GenMan` honours `SOURCE_DATE_EPOCH`
    (`doc/man_docs.go:126-134`).
  - No signal handlers, and no terminal queries.

## 7. fang

- **The module path is `charm.land/fang/v2`**, at v2.0.1 (2026-03-11).
  `github.com/charmbracelet/fang` stops at v1.0.0. The v2 `go.mod` declares
  `module charm.land/fang/v2`, so only that path can be required
  (`UPGRADE_GUIDE_V2.md:7-12`).
- **Requirements:** `go 1.25.0`; `charm.land/lipgloss/v2`, `colorprofile`,
  `x/ansi`, `x/term`, `muesli/mango-cobra`, `muesli/roff`, `spf13/cobra`,
  `spf13/pflag`, `golang.org/x/sys` and `golang.org/x/text`, among others.
  No Charm v1 import path appears in its `go.sum`.
- **API** (`fang.go`): `Execute(ctx, root, ...Option)` (110), and the
  options `WithoutCompletions`, `WithoutManpage`, `WithColorSchemeFunc`,
  `WithVersion`, `WithoutVersion`, `WithCommit`, `WithErrorHandler`,
  `WithNotifySignal` (42-103). There is no option for writers, width or
  background.
- **What it does that the library's rules forbid, with no option to turn
  off:**
  - **It queries the terminal.** `mustColorscheme` calls
    `lipgloss.HasDarkBackground(os.Stdin, os.Stdout)` whenever stdout is a
    terminal (`theme.go:116-121`), which writes an OSC 11 query
    (`lipgloss/v2@v2.0.1/query.go:34`, `83`). It runs on every help render
    and every error render (`fang.go:132`, `175`), whatever writers were
    set.
  - **It reads the terminal's width** from `term.GetSize(os.Stdout.Fd())`,
    capped at 120 and cached for the process (`help.go:30-40`).
  - **Its `man` command writes to `os.Stdout`** (`fang.go:156`), not to the
    command's writer.
  - Its signal handling is opt-in through `WithNotifySignal`
    (`fang.go:167-171`). It never calls `os.Exit`.
- Its help renderer is unexported (`help.go:42`), so it cannot be reused
  without `Execute`.

## 8. Kong

`github.com/alecthomas/kong` v1.16.1 (2026-08-09). Two probe programs were
built and run with go1.27.1.

- **Requirements:** `go 1.20`; only test-only modules. The runtime imports
  nothing outside the standard library.
- **API** (`options.go`): `DynamicCommand(name, help, group, cmd, tags...)`
  (100), `Name`, `Description`, `Writers` (215), `Exit` (51), `Bind`,
  `BindTo`, `BindFor[T]`, `BindToProvider` (234-284), `Help`,
  `ConfigureHelp`, `ExplicitGroups` (291-383), `Resolvers` (417),
  `Configuration(loader, paths...)` (459), `PostBuild` (125). Hook
  interfaces run in the order BeforeReset, BeforeResolve, BeforeApply,
  AfterApply (`kong.go:323-356`); `AfterRun` runs in `Context.Run`.
- **Tags** (`tag.go:249-345`): `cmd`, `arg`, `required`, `optional`,
  `default`, `name`, `help`, `type`, `env`, `short`, `hidden`, `sep`,
  `group`, `xor`, `and`, `prefix`, `embed`, `negatable`, `aliases`,
  `placeholder`, `enum`, `passthrough`. `t.Arg = t.Has("arg")` (260): any
  field with an `arg` tag is positional, whatever its value. Enum values are
  split on `,` (`model.go:281`, `291`), and a scalar enum must be `required`
  or have a `default` (`tag.go:326-329`). Unknown tag keys are kept and read
  through `Tag.Get` (`tag.go:58-59`).
- **Runtime grammars work.** A command built with `reflect.StructOf`,
  including nested `cmd` fields, parsed and validated in the probes. A
  `StructOf` type has no methods, so `Context.Run` fails with "no Run()
  method found in hierarchy"; dispatch must come from `ctx.Selected()`.
  Kong has no map-based grammar. A `passthrough` command captures raw
  arguments; through `DynamicCommand` tags it needs `cmd:""` as well
  (observed error otherwise: "passthrough only makes sense for positional
  arguments or commands").
- **Behaviour the library's rules must answer:**
  - Defaults are `Exit: os.Exit`, `Stdout: os.Stdout`, `Stderr: os.Stderr`
    (`kong.go:84-86`). `Exit` is called after `--help` (`help.go:28`), by
    `VersionFlag`, `Fatalf` and `FatalIfErrorf`.
  - **If `Exit` returns, parsing continues.** With an `Exit` that returned,
    `--help` printed help and `Parse` then returned a parse error. With an
    `Exit` that panics a sentinel the caller recovers, it stopped cleanly.
  - Help width comes from `$COLUMNS` and then a `TIOCGWINSZ` ioctl when the
    writer is an `*os.File` (`guesswidth_unix.go:13-41`); it is 80 for any
    other writer.
  - `Name` defaults to `filepath.Base(os.Args[0])` (`kong.go:118`).
  - `env` tags read `os.LookupEnv`; the file mappers read `os.Stdin` for
    `-` (`mapper.go:635`, `741`).
- **No built-in shell completion.** `github.com/willabides/kongplete`
  v0.4.0 was last released in 2023 against Kong v0.8.1;
  `github.com/jotaen/kong-completion` v0.0.14 (2026-04-30) requires Kong
  v1.13.0 and `posener/complete`, and its completion command calls
  `ctx.Exit(0)`.

## 9. glamour

`charm.land/glamour/v2` v2.0.1 (2026-06-12) is the latest; there is no v3.

- **Requirements:** `go 1.25.8`; `charm.land/lipgloss/v2`,
  `alecthomas/chroma/v2`, `x/ansi`, `microcosm-cc/bluemonday`,
  `yuin/goldmark`, `yuin/goldmark-emoji`, `golang.org/x/text`. No Charm v1
  path in its `go.sum`.
- **"Glamour is now pure."** v2 removed `WithAutoStyle` and
  `WithColorProfile` (`UPGRADE_GUIDE_V2.md:30-42`, `73-96`): it never
  queries the terminal, and colour downsampling is the caller's. Its only
  process reads are `GLAMOUR_STYLE` through `WithEnvironmentConfig`
  (`glamour.go:280`) and style files through `WithStylePath`.
- **API** (`glamour.go`): `NewTermRenderer` (68), `WithStyles` (146),
  `WithWordWrap` (173), `WithTableWrap`, `WithInlineTableLinks`,
  `WithPreservedNewLines`, `WithEmoji`, `WithChromaFormatter` (183-216),
  `Render` and `RenderBytes` (267-273). Built-in styles: ascii, dark,
  light, notty, pink, dracula, tokyo-night.
- **A `TermRenderer` is not safe for concurrent use.** Its `ANSIRenderer`
  shares one `RenderContext` whose block stack and table every render
  mutates (`ansi/renderer.go:28-37`, `ansi/context.go:11-27`).
- **The code-block theme is process-wide, first wins.** A style with
  `CodeBlock.Chroma` registers a chroma style named `charm` into chroma's
  global registry once per process (`ansi/codeblock.go:84-125`). The dark,
  light, dracula and tokyo-night styles all set it, so whichever renders a
  code block first fixes code colours for every later renderer.
- **Glyphs and margins.** The default styles draw non-ASCII glyphs, such
  as `[✓]` followed by a space (`styles/styles.go:20`), and a document margin of 2
  (`styles.go:13`, `46`). Tables default to `lipgloss.NormalBorder()`
  unless separators are set (`ansi/table.go:106-119`).

## 10. What this means for the library's rules

| Rule (0001-MADR §6) | Cobra | fang | Kong | glamour |
| :--- | :--- | :--- | :--- | :--- |
| 1, output to the caller's writers | met with `SetOut`/`SetErr`; a failing `__complete` writes `os.Stderr` | broken by `man` | met with `Writers` | met: returns strings |
| 1, no `os.Exit` | met unless `CheckErr` or the mousetrap | met | met only with an `Exit` that panics and is recovered | met |
| 2, the caller probes the terminal | met | broken: OSC 11 on every help and error | met | met |
| 5, width is the caller's | met | broken: stdout's size | met only with a custom help printer | met: `WithWordWrap` |
| read only what the caller gives | met with `SetArgs` | — | met with `Name`; `env` tags read the environment | met without the environment options |
