---
status: in-progress
date: 2026-10-05
associated-madr: "0006-MADR-command-registry.md"
---
# Implement the command registry (`when`, `command`, `command/cli`, and the Cobra and Kong front ends)

Associated MADR: [0006-MADR-command-registry.md](0006-MADR-command-registry.md)

## Revisions

* **2026-10-02.** The owner answered the MADR's Q1–Q4, and the MADR is
  `accepted`. Q4 added a Cobra front end beside `command/cli`.
* **Later on 2026-10-02.** MADR amendment A1 and
  [0010-MADR-nested-adapter-modules.md](0010-MADR-nested-adapter-modules.md)
  moved the Cobra front end into the nested module `command/cobracmd`,
  without fang, added a Kong front end, `command/kongcmd`, and aligned the
  argument tags with Kong's.
* **2026-10-03.** The owner accepted A1 and approved
  [0010-PLAN-nested-adapter-modules.md](0010-PLAN-nested-adapter-modules.md).
* **2026-10-05.** Rewritten after an audit of the tree at `v0.3.0`
  (`81a61de`), recorded as MADR amendment A2, at the owner's request
  ("assess the codebase for current facts … ensure the plan is accurate,
  updated, and is actionable, detailed, and deterministic"). What changed,
  and why:
  * the preconditions that 0004 and 0010 set are met, so the conditional
    and "blocked" wording is gone, and Steps 10 and 11 wait only for the
    `v0.4.0` tag;
  * A1's tags are the only tag vocabulary; §4's `doc` and `arg:"…"` are
    not built;
  * `workspace.resize` takes `split`, `workspace.layout.reset` is
    `SetState` of the zero state, `workspace.theme.set` calls a new
    `Workspace.SetBackground`, and `SchemaOf` supports `map[string]T`
    (A2, Q5);
  * the exporters target MCP 2026-07-28 (A2, Q6);
  * the registry's own commands move from Step 3 to Step 4, because
    `command.describe` takes an argument and `New[A]` arrives in Step 4;
  * each fuzz target joins `make fuzz` in the step that adds it, so CI
    fuzzes it from that step on;
  * every step now lists its files, its exported names, its tests by name,
    its golden files, its mutations, its commands and its done criteria;
  * a Step 12 closes the PLAN after both front ends, which the old Step 9
    did by reference.

  The text this revision replaces is in git at `4c6261e`.

## Goal

Ship `when`, `command` and `command/cli` in the root module's `v0.4.0`,
with the workspace's built-in commands, so that one definition per action
serves keys, the palette, slash commands, help, the shell and agents; then
ship the Cobra and Kong front ends as the nested modules
`command/cobracmd` and `command/kongcmd`, each at `v0.1.0` (MADR §1–§11,
A1, A2).

Done means every item under Verification holds, CI is green on the pushed
tree, and the owner has tagged `v0.4.0`, `command/cobracmd/v0.1.0` and
`command/kongcmd/v0.1.0`.

## Scope

### Facts this PLAN starts from (2026-10-05)

| Fact | Where it was read |
| :--- | :--- |
| Go 1.27.1; `encoding/json/v2` builds with no `GOEXPERIMENT` | `go version`; a scratch build (MADR A2, item 1) |
| the root module is at `v0.3.0`; its `go.mod` is unchanged since `v0.2.0` | `git tag`; `go.mod` |
| 0002, 0004, 0005 and 0010 are `complete` | `docs/README.md` |
| `go.work` lists `.` only; `scripts/go-modules.sh` lists the modules for CI's matrix | `go.work`; `.github/workflows/ci.yml` |
| `.golangci.yml` confines Cobra and pflag to `command/cobracmd/**`, Kong to `command/kongcmd/**` | `.golangci.yml`, the `cobra` and `kong` depguard rules |
| `internal/conformance` scans every package of every module | `internal/conformance/conformance_test.go` |
| `make fuzz` runs `scripts/go-fuzz.sh` once per package, one line each | `Makefile`, target `fuzz` |
| golden files are `tuitest.Text` or `tuitest.Golden`, rewritten with `-tuitest.update` | `tuitest/tuitest.go` |
| the workspace methods the built-ins wrap, and their signatures | `go doc ./workspace` (MADR A2, item 2) |

### In scope

| Step | Paths | Delivers | Released in |
| :--- | :--- | :--- | :--- |
| 1 | `docs/decisions/0006-*`, `docs/README.md` | the records (this revision) | — |
| 2 | `when/`, `Makefile` | the expression language | `v0.4.0` |
| 3 | `command/` | the command, the registry, dispatch, the gate, the audit trail | `v0.4.0` |
| 4 | `command/`, `Makefile` | arguments: `New[A]`, `SchemaOf`, strict decoding, slash parsing; the registry's own commands | `v0.4.0` |
| 5 | `command/` | sources: command files, MCP prompts, ACP commands, clashes | `v0.4.0` |
| 6 | `command/` | the MCP and ACP exporters and the manifest | `v0.4.0` |
| 7 | `workspace/` | `Commands`, `WhenContext`, `Contexter`, the context keys, `SetBackground` | `v0.4.0` |
| 8 | `command/cli/` | the standard-library front end | `v0.4.0` |
| 9 | `README.md`, `docs/` | documentation; the root release | `v0.4.0` |
| 10 | `command/cobracmd/`, `go.work`, docs | the Cobra front end, a nested module | `command/cobracmd/v0.1.0` |
| 11 | `command/kongcmd/`, `go.work`, docs | the Kong front end, a nested module | `command/kongcmd/v0.1.0` |
| 12 | `docs/` | close-out | — |

The root `go.mod` and `go.sum` do not change in any step: every import in
Steps 2 to 9 is the standard library or `charm.land/bubbletea/v2`, which the
root already requires. The root `go.mod` never names Cobra or Kong.

### Out of scope

* The keymap engine
  ([0007-PLAN-keymap-engine.md](0007-PLAN-keymap-engine.md)) and the
  palette ([0008-PLAN-command-palette.md](0008-PLAN-command-palette.md)).
  The workspace's `KeyMap` stays as it is until 0007 moves it.
* A permission dialog: this PLAN defines `Gate` and the default refusal.
* An MCP server or an ACP transport, the MCP list envelope (`ttlMs`,
  `cacheScope`) and `subscriptions/listen`: the host's (MADR A2, Q6).
* `terminal.*` context keys from `termcap` (MADR A2, item 5).
* YAML or TOML command files (MADR Q1), fang (0010-MADR Q1), a third-party
  Kong completion module (0010-MADR Q2), and shell completion scripts from
  `command/cli`.
* Any change in pi-go.
* `git push` and tags, which are the owner's.

## Rules for every step

1. **Order.** A step starts when the previous one is committed. Its
   packages compile, its tests pass, and its checks are clean before it
   ends.
2. **Exported API.** A step adds exactly the exported names it lists. A
   name it needs and does not list stops the step for the owner, and is
   recorded as a deviation.
3. **Checks,** with their output in the execution record:
   1. `gofmt -l` on the step's Go files prints nothing;
   2. `make pre-add-check FILES="<the step's Go files>"`;
   3. `make lint`, which runs `make modernize` and golangci-lint for
      `GOOS=linux`, `darwin` and `windows`;
   4. `GOWORK=off go test -race -count=1 ./...`;
   5. `GOWORK=off go test -shuffle=on -count=2 ./...`;
   6. `LC_ALL=C GOWORK=off go test -count=1 ./...`;
   7. `GOWORK=off go mod tidy -diff`;
   8. `make fuzz FUZZTIME=20s` when the step adds a fuzz target;
   9. on the Windows test host, on a copy of the tree: `make
      pre-add-check`, `make lint`, `make vuln`, and the step's packages'
      tests with `-shuffle=on -count=3`;
   10. `markdownlint-cli2 --config .markdownlint-cli2.jsonc` and a
       relative-link check on the Markdown the step changes;
   11. the identifier scan of the diff: no hostname, account name or
       real-machine path.
4. **Mutation proofs.** Each listed mutation is applied to a scratch copy
   of the tree, never the tree itself, and the named test must fail; the
   failing line goes in the execution record. A mutation that survives, or
   does not compile, stops the step: the test is strengthened, or the
   mutation replaced, and either is recorded. After a step changes code
   that an earlier step's mutation anchors on, the earlier steps' sets run
   again.
5. **Golden files** are written with `-tuitest.update`, and every one is
   read before the step ends.
6. **Conventions** (0001-MADR §6, AGENTS.md). Nothing writes to
   `os.Stdout` or `os.Stderr`, reads the process environment, starts a
   process, sets the alternate screen or installs a signal handler; every
   glyph comes from `glyph`; output goes to the writers a caller gives.
   `internal/conformance` covers each new package with no change to it.
7. **Deviations.** Anything the PLAN does not say, or that contradicts a
   fact here, stops the step: the agent gives the evidence and the
   options, the owner picks, and the choice is recorded in this PLAN, and
   in the MADR when a decision or a fact changes, before work continues.
8. **Commits.** One commit per step, made by the owner with `git commit
   --no-edit`, unless the owner asks the agent to commit in that turn. The
   agent stages nothing otherwise.

## Implementation Steps

### Step 1: records

* MADR amendment A2 is accepted, and this revision is written. When the
  owner approves execution, this PLAN is set `in-progress`, with
  `docs/README.md`'s row, at the start of Step 2.
* **Done when:** the owner approves execution.

### Step 2: `when`

**Files:** `when/when.go` (package documentation, `Expr`, `Parse`,
`MustParse`, `Check`), `when/lex.go`, `when/parse.go`, `when/eval.go`,
`when/value.go`, `when/key.go`; tests `when/when_test.go`,
`when/fuzz_test.go`, `when/bench_test.go`; `Makefile`. (*D1: and the
regression input `when/testdata/fuzz/FuzzParse/01e90e822ccb8b53`. D3: and
`.github/workflows/ci.yml`.*)

**Exported names** (MADR §8):

```go
func Parse(src string) (Expr, error)
func MustParse(src string) Expr
type Expr struct{ /* compiled */ }
func (e Expr) Eval(c Context) bool
func (e Expr) String() string
func (e Expr) Keys() iter.Seq[string]
func Check(e Expr, known Keys) []error

type Context interface{ Value(key string) (Value, bool) }
type Kind uint8                // KindBool, KindNumber, KindString, KindList
type Value struct{ /* one of the four kinds */ }
func BoolValue(b bool) Value
func NumberValue(f float64) Value
func StringValue(s string) Value
func ListValue(l []string) Value
func (v Value) Kind() Kind
func (v Value) Bool() bool       // the truth of a key alone (§8)
func (v Value) Number() (float64, bool)
func (v Value) String() string   // canonical text
func (v Value) List() ([]string, bool)
type Map map[string]Value
func (m Map) Value(key string) (Value, bool)
func Layered(cs ...Context) Context
type Keys map[string]Kind
type Key[T bool | int | float64 | string | []string] struct{ Name, Doc string }
func NewKey[T bool | int | float64 | string | []string](name, doc string) Key[T]
func (k Key[T]) Set(m Map, v T)
func (k Key[T]) Get(c Context) (T, bool)
func (k Key[T]) Kind() Kind
```

An `int` is held as `KindNumber`. `MaxSource = 4096` and `MaxDepth = 64`
are exported constants.

**Build:** a hand-written lexer, and a recursive-descent parser of the
MADR's grammar with its precedence. A regex compiles with `regexp` at parse
time. `String` writes the canonical form: single spaces around binary
operators, parentheses only where precedence needs them, strings in single
quotes.

**Tests** (`when/when_test.go`):

| Test | Shows |
| :--- | :--- |
| `TestOperators` | every operator of §8, alone, on a `Map` |
| `TestPrecedence` | `!foo && bar` is `(!foo) && bar`; `foo \|\| bar && baz` is `foo \|\| (bar && baz)` |
| `TestUnsetKeys` | an unset key alone is false; it is unequal to every value, so `x != 'a'` is true |
| `TestIn` | `in` and `not in` over a list key, and over an unset key |
| `TestTerms` | quoted strings with spaces, barewords, integers, decimals, `true`, `false` |
| `TestNoSpacesNeeded` | `a==1&&b!=2` parses as `a == 1 && b != 2` |
| `TestLayered` | the first context that holds a key wins |
| `TestCheck` | an unknown key, and a number compared with a string, are reported |
| `TestLimits` | 4097 bytes of source, and 65 nested parentheses, are errors, not panics (*D1: and a source within 4096 bytes whose canonical form is longer*) |
| `TestRegexIsRE2` | `=~ /(a+)+$/` evaluates in linear time on a long input; a bad regex is a parse error |
| `TestStringRoundTrip` | `Parse(e.String()).String() == e.String()` over the operator table (*D2: and the two evaluate alike on every assignment of `true` and `false` to the keys*) |
| `TestTypedKeys` | `Key[T].Set` and `Get` for each `T`; `Get` of the wrong kind is false |

`FuzzParse` (`when/fuzz_test.go`): no panic; for every parsed input,
`Parse(e.String())` succeeds and gives the same `String()` (*D1: true
once `Parse` bounds the canonical form, MADR A3*); `Eval` on a
`Map` built from the fuzz input does not panic (*D2: and gives the same
result for the expression and its canonical form*). Seeds: the operator
table. Benchmarks `BenchmarkParse` and `BenchmarkEval` on a five-clause
expression.

**Makefile:** the `fuzz` target gains
`@./scripts/go-fuzz.sh -t $(FUZZTIME) -m 1 ./when`, and its help text
names `when`.

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| `&&` binds looser than `\|\|` | `TestPrecedence` |
| an unset key compares equal to `''` | `TestUnsetKeys` |
| the depth limit is not checked | `TestLimits` |
| `Layered` takes the last match | `TestLayered` |
| `String` drops needed parentheses | `TestStringRoundTrip` |
| `not in` is evaluated as `in` | `TestIn` |
| *D1:* the canonical form's length is not checked | `TestLimits` |

**Done when:** the checks of Rule 3 are clean, `make fuzz` covers `when`,
and every mutation is killed.

### Step 3: `command` core

**Files:** `command/command.go` (package documentation, `ID`, `Command`,
the enumerations), `command/handler.go`, `command/registry.go`,
`command/dispatch.go`, `command/gate.go`, `command/audit.go`,
`command/messages.go`; tests `command/registry_test.go`,
`command/dispatch_test.go`, `command/gate_test.go`,
`command/bench_test.go`.

**Exported names** (MADR §2, §3, §5, §7):

* `ID` with `Valid() error` and `Segments() iter.Seq[string]`;
  `Command` with the fields of §2; `Schema = json.RawMessage`.
* `Kind` (`Action`, `Prompt`, `Forward`), `Danger` (`ReadOnly`, `UI`,
  `Mutating`, `Destructive`), `Surface` (`SurfaceKey`, `SurfacePalette`,
  `SurfaceSlash`, `SurfaceCLI`, `SurfaceAgent`, and `AllSurfaces`), `Mode`
  (`Loop`, `Async`), `Scope` (a string; `Global` is `""`), `SourceKind`
  (`Builtin`, `User`, `Project`, `MCP`, `ACP`, `Plugin`), `Source`,
  `Origin` (`OriginKey`, `OriginMouse`, `OriginPalette`, `OriginSlash`,
  `OriginCLI`, `OriginAgent`, `OriginProgram`), each with `String`.
* `Handler`, `HandlerFunc`, `Invocation`, `Result`, `Request`.
* `Registry`, `NewRegistry`, `RegistryOption`, `WithGate`, `WithAuditor`,
  `WithPrefixer`, and the methods `Register`, `Remove`, `Lookup`, `Slash`,
  `All`, `Available`, `Version`, `Watch`, `Dispatch`, `Run`, `Cancel`,
  `CancelAll`. (`ReplaceSource` is Step 5's; `ParseSlash` Step 4's.)
* `Decision` (`AllowOnce`, `AllowAlways`, `RejectOnce`, `RejectAlways`)
  with `ACPKind() string`; `Gate`; `Auditor`; `Record`; `SlogAuditor`.
* The messages `ResultMsg`, `PromptMsg`, `ChangedMsg`, `ConflictMsg`,
  `QuitRequestMsg`; `Conflict`.

**Build:**

* The snapshot is an immutable slice sorted by ID and a map from ID and
  from slash name, behind an `atomic.Pointer`. A write takes a mutex,
  copies, bumps `Version`, publishes, and closes the channel `Watch`
  waits on.
* `Dispatch` checks `When` and `Surfaces` for the request's origin and
  context, asks the policy (§7), audits, and then runs a `Loop` command at
  once, returning its `Result.Cmd` batched with a `ResultMsg`, or returns
  a `tea.Cmd` that runs an `Async` command under a context `Cancel`
  reaches. `Run` does the same synchronously, for any mode.
* The policy (§7): `ReadOnly` and `UI` run for every origin; `Mutating`
  from `OriginAgent`, and `Destructive` from `OriginAgent` or `OriginCLI`,
  ask the gate, and with none are `RejectOnce`. `AllowAlways` and
  `RejectAlways` are remembered per command ID and caller.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestIDValid` | `workspace.focus.next` is valid; an upper-case letter, an empty segment, a leading `-` and 129 bytes are not |
| `TestRegisterDuplicate` | a second command with one ID is an error |
| `TestAllSorted`, `TestAvailable` | `All` is in ID order; `Available` honours `When` and `Surfaces` |
| `TestDispatchLoop` | a `Loop` command runs during `Dispatch`; its `Result.Cmd` and a `ResultMsg` come back |
| `TestDispatchAsync` | an `Async` command runs only when the returned `tea.Cmd` runs |
| `TestRunMatchesDispatch` | `Run` and `Dispatch` give the same `Result` for one request |
| `TestCancel`, `TestExclusive` | a running async command sees its context cancelled |
| `TestPolicy` | every origin against every danger, without a gate and with one |
| `TestAlways` | `AllowAlways` is remembered per command and caller; `RejectOnce` is not |
| `TestSlogAuditor` | one record per dispatch on the given logger, never the default logger |
| `TestSnapshotsUnderRace` | under `-race`, `All` and `Lookup` from many goroutines during writes see whole snapshots, and `Version` never falls |
| `TestWatch` | `Watch`'s command returns `ChangedMsg` after a change, not before |

Benchmarks: `BenchmarkLookup`, `BenchmarkAvailable` over 500 commands,
`BenchmarkDispatchLoop` of a no-op command.

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| the default policy allows `Mutating` for agents | `TestPolicy` |
| `AllowAlways` is not remembered | `TestAlways` |
| `Version` is not bumped on `Remove` | `TestWatch` |
| `Exclusive` is ignored | `TestExclusive` |
| a write changes the published snapshot in place | `TestSnapshotsUnderRace` |
| `Dispatch` skips `When` | `TestAvailable` or `TestDispatchLoop` |

**Done when:** the checks are clean and every mutation is killed.

### Step 4: arguments, and the registry's own commands

**Files:** `command/args.go` (`New`, `Option` and its constructors,
`NoArgs`, `ArgError`), `command/schema.go` (`SchemaOf`, the tags),
`command/decode.go`, `command/slash.go` (`ParseSlash`),
`command/builtin.go`; tests `command/schema_test.go`,
`command/decode_test.go`, `command/slash_test.go`,
`command/builtin_test.go`, `command/fuzz_test.go`; goldens
`command/testdata/golden/schema-*.golden`; `Makefile`.

**Exported names** (MADR §4, A1, A2):

* `New[A any](id ID, title string, run func(context.Context, *Invocation, A) (Result, error), opts ...Option) (Command, error)`,
  `SchemaOf[A any]() (Schema, error)`, `NoArgs`, `ArgError{Path, Reason}`.
* `Option`, one constructor per `Command` field New does not take:
  `WithDescription`, `WithCategory`, `WithSlash(name string, aliases ...string)`,
  `WithArgHint`, `WithOutput(Schema)`, `WithWhen`, `WithScope`,
  `WithDanger`, `WithIdempotent`, `WithOpenWorld`, `WithSurfaces`,
  `WithMode`, `WithExclusive`, `WithWhileBusy`, `WithHidden`, `WithMeta(key string, v any)`.
* `(*Registry).ParseSlash(line string) (Request, error)`.

**The tags** are A1's only: `json`, `help`, `default`, `enum:"a,b"`,
`arg:""` (positional, in field order), `short`, `hidden`, `placeholder`,
`group`, and `schema:"min=…,max=…,minLen=…,maxLen=…,secret"`. `SchemaOf`
supports string, bool, the integer and float kinds, slices, nested structs,
pointers, `map[string]T` (A2), `time.Duration` (a string), and a type with
its own `JSONSchema() Schema`. It refuses a scalar `enum` field that is
neither required nor defaulted (A1), and any other type, with an error.

**Decoding:** `encoding/json/v2` with `json.RejectUnknownMembers(true)`,
then `type`, `required`, `enum`, `minimum`, `maximum`, `minLength`,
`maxLength`, `items` and `additionalProperties: false` are checked. An
`*ArgError` carries a JSON Pointer path.

**Slash parsing:** quotes group words; positionals fill `arg` fields in
field order; `name=value` fills any field; a command with one string
argument takes the whole tail. The result is the JSON `--args` would send.

**The registry's own commands,** registered by `NewRegistry`:
`command.list` (`NoArgs`, `ReadOnly`, every surface), `command.describe`
(`{id string}` positional, `ReadOnly`, every surface) and `app.quit`
(`NoArgs`, `UI`, no CLI surface), which only sends `QuitRequestMsg`.

**Goldens** (`tuitest.Text`): `schema-flat`, `schema-nested`,
`schema-slices`, `schema-pointers`, `schema-enum`, `schema-bounds`,
`schema-duration`, `schema-map`, `schema-custom`, `schema-noargs`. Each is
valid JSON naming `$schema` 2020-12; `schema-noargs` is `{"type":
"object", "additionalProperties": false}` with `$schema`.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestSchemaGolden` | the ten goldens |
| `TestSchemaRefuses` | a channel field, and an enum neither required nor defaulted, are errors, not panics |
| `TestSchemaKeepsUnknownKeys` | a `schema` key Kong does not read, and the `json` name, survive in the schema |
| `TestDecodeStrict` | an unknown member, a wrong type, a missing required field, a number out of range and a value outside its enum are each an `*ArgError` found with `errors.AsType`, naming its path |
| `TestSlashForms` | `/resize sidebar 4`, `/resize split=sidebar delta=4` and `/resize "sidebar" "4"` give the same JSON; a stray positional is an error |
| `TestSlashWholeTail` | a one-string command takes the whole tail |
| `TestAuditMasksSecrets` | a field tagged `schema:"secret"` is masked in every audit `Record` |
| `TestBuiltins` | `command.list` lists every command once; `command.describe` returns a command's ID, schema and danger; `app.quit` sends `QuitRequestMsg` and nothing else |

`FuzzParseSlash`: no panic, and every accepted line decodes against its
command's schema. The `fuzz` target gains
`@./scripts/go-fuzz.sh -t $(FUZZTIME) -m 1 ./command`.

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| `omitzero` fields are marked required | `TestSchemaGolden` |
| unknown members are accepted | `TestDecodeStrict` |
| `maximum` is not checked | `TestDecodeStrict` |
| positionals fill fields in reverse order | `TestSlashForms` |
| `arg` presence is ignored, so a positional becomes a flag | `TestSlashForms` |
| `enum` is split on `\|` | `TestSchemaGolden` |
| a map field is refused | `TestSchemaGolden` |
| `secret` is not masked | `TestAuditMasksSecrets` |

**Done when:** the checks are clean, `make fuzz` covers `command`, every
golden is read, and every mutation is killed.

### Step 5: sources and loaders

**Files:** `command/source.go` (`ReplaceSource`, the prefixer),
`command/loaddir.go`, `command/frontmatter.go`, `command/expand.go`,
`command/mcpprompt.go`, `command/acp.go` (`ACPCommand`, `FromACP`); tests
`command/loaddir_test.go`, `command/frontmatter_test.go`,
`command/sources_test.go`; a test tree `command/testdata/commands/`; goldens
`command/testdata/golden/loaddir.golden`.

**Exported names** (MADR §6): `LoadDir(fsys fs.FS, src Source)
([]Command, []error)`; `MCPPrompt`, `MCPPromptArgument`, `PromptGetter`,
`FromMCPPrompts`; `ACPCommand`, `ACPCommandInput`, `FromACP`;
`(*Registry).ReplaceSource(src Source, cmds []Command) []Conflict`.

**Build:** the front matter is MADR §6's strict subset; `$NAME`
placeholders become required string arguments; `$ARGUMENTS` is the slash
tail; expansion is text substitution only; a loaded command with no
`danger` is `Mutating`; IDs are `user.…`, `project.…`, `mcp.<server>.…`,
`acp.<agent>.…`, `plugin.<name>.…`; a loaded slash name that a built-in
holds becomes `<prefix>:<name>` and a `Conflict`, also sent as
`ConflictMsg`.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestLoadDirGolden` | the commands of `testdata/commands/`, with subdirectories, front matter and placeholders |
| `TestFrontMatterErrors` | an unknown key, a duplicate key, a nested value, no closing `---`, a bad `danger` and a bad `when` each name the file and line |
| `TestNoFrontMatter` | a file without it is a prompt whose description is its first line |
| `TestRootRefusesSymlink` | through `os.Root.FS()` on a temporary directory, a symlink out of the root is refused |
| `TestExpand` | `$FOCUS` and `$ARGUMENTS` are substituted; `$lower`, `$$`, `!{ls}` and `$(ls)` stay as text |
| `TestDefaultDanger` | a file without `danger` is `Mutating` |
| `TestFromMCPPrompts` | `mcp.<server>.<name>` prompts whose schema has the prompt's arguments, `required` kept |
| `TestFromACP` | `acp.<agent>.<name>` forwards; running one sends `PromptMsg{Text: "/name args"}` |
| `TestSlashClash` | a loaded `/review` beside a built-in one is `/user:review`, with a `Conflict` and a `ConflictMsg` |
| `TestReplaceSource` | the old commands go in the same version the new ones arrive |

`FuzzFrontMatter`: no panic; every accepted input re-serialises to an
equivalent front matter.

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| a built-in loses a slash clash | `TestSlashClash` |
| unknown front-matter keys are ignored | `TestFrontMatterErrors` |
| a command without `danger` defaults to `ReadOnly` | `TestDefaultDanger` |
| `ReplaceSource` keeps the old commands | `TestReplaceSource` |
| `!{…}` is expanded | `TestExpand` |

**Done when:** the checks are clean, the golden is read, and every
mutation is killed.

### Step 6: exporters

**Files:** `command/mcp.go`, `command/acpexport.go`,
`command/manifest.go`; tests `command/export_test.go`; goldens
`command/testdata/golden/mcp-tools.golden`, `mcp-call-*.golden`,
`acp-commands.golden`, `manifest.golden`.

**Exported names** (MADR §7, A2): `MCPTool`, `MCPToolAnnotations`,
`MCPCallResult`, `MCPContent`, `Manifest`, `ManifestCommand`; the methods
`MCPTools(ctx when.Context) []MCPTool`, `CallMCP(ctx context.Context, name
string, arguments json.RawMessage, caller string) MCPCallResult`,
`ACPCommands(ctx when.Context) []ACPCommand`, `Manifest() Manifest`.

**The target is MCP 2026-07-28** (A2, Q6). Before the first golden, the
step fetches `schema/2026-07-28/schema.ts` and records, in the execution
record, the field names of `Tool`, `ToolAnnotations` and
`CallToolResult`; the tests hold them as literal lists, and no test reads
the network.

* `MCPTool`: `name` (the ID), `title`, `description`, `inputSchema`,
  `outputSchema` (omitted when empty), `annotations`, `_meta`.
* `MCPCallResult`: `content`, `structuredContent` (omitted when empty),
  `isError`, and `resultType: "complete"`.
* Danger maps to annotations as MADR §2's table says.
* `MCPTools` and `ACPCommands` list in ID order, hidden commands and
  commands without `SurfaceAgent` left out.
* `CallMCP` runs as `OriginAgent`; a bad argument, an unknown name and a
  refusal are `isError: true` with a text the model can act on, never a
  Go error.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestExportGolden` | the goldens, for a catalogue of one command per kind and danger level |
| `TestMCPFieldNames` | decoded into `map[string]any`, each object's keys are a subset of 2026-07-28's, and the required ones are present |
| `TestACPFieldNames` | `name`, `description`, `input`, `hint`, and nothing else |
| `TestCallMCPErrors` | a bad argument, an unknown name and a gate refusal each give `isError: true` and a message |
| `TestCallMCPIsAgent` | the default policy applies to `CallMCP` |
| `TestExportFilters` | hidden and non-agent commands are not exported |
| `TestManifestRoundTrip` | the manifest round-trips through JSON |

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| `Destructive` exports `destructiveHint: false` | `TestExportGolden` |
| `CallMCP` runs as `OriginProgram` | `TestCallMCPIsAgent` |
| hidden commands are exported | `TestExportFilters` |
| `resultType` is left out | `TestMCPFieldNames` |
| tools are listed in insertion order | `TestExportGolden` |

**Done when:** the checks are clean, the field names are recorded, the
goldens are read, and every mutation is killed.

### Step 7: workspace integration

**Files:** `workspace/commands.go`, `workspace/context.go`,
`workspace/background.go`; tests `workspace/commands_test.go`,
`workspace/context_test.go`, `workspace/background_test.go`; goldens
`workspace/testdata/golden/commands-*`.

**Exported names** (MADR §8, §9, A2):

* `Commands(w *Workspace, o ...CommandOption) []command.Command`,
  `CommandOption`, `WithLayouts(map[string]layout.Node) CommandOption`;
* `(*Workspace).WhenContext() when.Context`; `Contexter interface{
  WhenContext() when.Context }`;
* the typed keys, as package variables: `KeyFocusedPane`
  (`workspace.focusedPane`, string), `KeyZoomed` (`workspace.zoomed`,
  bool), `KeyHiddenPanes` (`workspace.hiddenPanes`, list), `KeyOverlay`
  (`workspace.overlay`, string), `KeyModal` (`workspace.modal`, bool),
  `KeyWidth` (`workspace.width`, int), `KeyHeight` (`workspace.height`,
  int); and `ContextKeys() when.Keys`;
* `(*Workspace).SetBackground(bg theme.Background) tea.Cmd` (A2, Q5).

**The built-ins,** each `Loop` mode, every surface but CLI, Danger `UI`
unless marked:

| ID | Slash | Arguments | Calls |
| :--- | :--- | :--- | :--- |
| `workspace.focus` | `focus` | `pane` (`arg`) | `Focus`; a pane not in the plan is an `ArgError` |
| `workspace.focus.next`, `.prev` | `next`, `prev` | — | `FocusNext`, `FocusPrev` |
| `workspace.zoom` | `zoom` | `pane` (`arg`, `omitzero`) | `Zoom(pane)`, the focused pane when empty |
| `workspace.toggle` | `toggle` | `pane` (`arg`) | `Toggle` |
| `workspace.resize` | `resize` | `split` (`arg`), `delta` (`arg`, `-200..200`) | `Resize`; a split that is not a resizable separator of `Plan()` is an `ArgError` |
| `workspace.layout.use` | `layout` | `name` (`arg`, an enum of `WithLayouts`'s keys) | `SetLayout`; registered only when `WithLayouts` names a layout |
| `workspace.layout.reset` | — | — | `SetState(layout.State{})` |
| `workspace.state.get` | — | — | `State()`, as `Value`; `ReadOnly` |
| `workspace.state.set` | — | `state` (a `layout.State`) | `SetState` |
| `workspace.panes` | `panes` | — | each pane's ID, title (`Titled`, else the ID), rectangle (`Plan().Panes`), focus and hidden state (`Plan().Hidden`), in focus-ring order; `ReadOnly` |
| `workspace.overlay.close` | `close` | — | `Pop`; with no overlay, a `Result.Text` that says so |
| `workspace.theme.set` | `theme` | `background` (`arg`, `dark`, `light`, `auto`) | `SetBackground(theme.Dark)`, `(theme.Light)` or `(theme.Unknown)` |

**`SetBackground`:** a new `pinnedBg` field, and the last reported
background kept apart from the one in use. `theme.Dark` or `theme.Light`
pins it, rebuilds, and a later `tea.BackgroundColorMsg` changes only the
reported one; `theme.Unknown` unpins, rebuilds from the reported one, and
returns `tea.RequestBackgroundColor` unless `WithoutBackgroundQuery`; on a
workspace built `WithTheme` it records the choice and returns nil.

**`WhenContext`** layers the top overlay's `Contexter`, then the focused
pane's, then the workspace's keys.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestBuiltinsMatchMethods` | each built-in, dispatched through a registry, gives the same frame and `State` as its method |
| `TestPanesCommand` | every pane, visible and hidden, with its rectangle and focus |
| `TestResizeSplit` | `sidebar` moves; a pane name or a positional split is an `ArgError` |
| `TestLayoutUseEnum` | the schema's enum is exactly `WithLayouts`'s keys, sorted; without them the command is absent |
| `TestLayoutReset` | a zoomed, resized and hidden layout returns to the zero state |
| `TestWhenContextLayers` | overlay, then focused pane, then workspace |
| `TestAgentMayZoom` | `workspace.zoom` runs for `OriginAgent` under the default policy |
| `TestKeysStillWork` | the existing key bindings still work |
| `TestSetBackgroundPins` | a pinned dark survives a light `BackgroundColorMsg` |
| `TestSetBackgroundAuto` | `Unknown` rebuilds from the reported background and returns the query, or nil with `WithoutBackgroundQuery` |
| `TestSetBackgroundFixedTheme` | on a `WithTheme` workspace the frame does not change |
| `TestCommandsGolden` | the agent-session example zoomed, restored and set light through `Dispatch`, across the matrix at 80 and 160 columns |

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| `WhenContext` puts the workspace keys first | `TestWhenContextLayers` |
| `workspace.panes` leaves hidden panes out | `TestPanesCommand` |
| `workspace.focus.next` moves twice | `TestBuiltinsMatchMethods` |
| a background message overrides a pinned background | `TestSetBackgroundPins` |
| `auto` sends no query | `TestSetBackgroundAuto` |
| `workspace.resize` accepts any split | `TestResizeSplit` |

**Done when:** the checks are clean, the goldens are read, and every
mutation is killed. `workspace`'s new imports are `command` and `when`
only.

### Step 8: `command/cli`

**Files:** `command/cli/cli.go`, `command/cli/flags.go`,
`command/cli/help.go`; tests `command/cli/cli_test.go`,
`command/cli/example_test.go`; goldens
`command/cli/testdata/golden/help-*.golden`.

**Exported names** (MADR §10): `Run(ctx context.Context, r
*command.Registry, args []string, stdout, stderr io.Writer, o ...Option)
int`; `Option`, `WithName(string)`, `WithConfirm(func(prompt string)
bool)`, `WithContext(when.Context)`, `WithWidth(int)`; the exit-code
constants `ExitOK = 0`, `ExitFailed = 1`, `ExitUsage = 2`,
`ExitRefused = 3`.

**Build:** word and dotted forms; a flag per schema property, an array
repeating its flag; `--args`, `--json`, `--yes`; the verbs `list`,
`describe`, `schema`, `help`. Help is plain ASCII at the option's width
(80 by default). A shell invocation is `OriginCLI`; `Result.Cmd` is
ignored; commands without `SurfaceCLI` are neither listed nor run.

**Tests:**

| Test | Shows |
| :--- | :--- |
| `TestHelpGolden` | the registry's help and one command's, at 60 and 100 columns, ASCII only |
| `TestFlagsEqualArgs` | `--delta 4` and `--args '{"delta":4}'` build the same request |
| `TestFlagKinds` | a string, integer, boolean, enum and array property each parse; a bad value exits 2 with the `ArgError` on stderr |
| `TestJSONOutput` | `--json` writes `Result.Value`; otherwise `Result.Text` |
| `TestDestructiveNeedsYes` | without `--yes` or `WithConfirm`, exit 3 |
| `TestCLISurface` | a command without `SurfaceCLI` is neither listed nor run |
| `TestWritersOnly` | everything goes to the given buffers |

`ExampleRun` builds a registry with two commands and runs `list` and one
command with `--json`.

**Mutations:**

| Mutation | Must fail |
| :--- | :--- |
| `--yes` is not required | `TestDestructiveNeedsYes` |
| a usage error exits 1 | `TestFlagKinds` |
| hidden commands are listed | `TestHelpGolden` |
| `--json` writes `Result.Text` | `TestJSONOutput` |

**Done when:** the checks are clean, the goldens are read, and every
mutation is killed.

### Step 9: documentation, and the root release `v0.4.0`

* **`docs/guides/commands.md` (new):** defining a command and `New[A]`
  with the tags; kinds, danger and the default gate; dispatching from
  `Update`, and `Run` for the shell and tests; command files and their
  front matter; MCP prompts and ACP commands; exporting to an agent, with
  what a host still owns under MCP 2026-07-28 (the list envelope,
  `subscriptions/listen`); `when` and the context keys;
  `workspace.Commands` and `SetBackground`; `command/cli`.
* **`docs/architecture.md`:** the three packages and their imports,
  `workspace`'s two new imports, the tree. **`docs/README.md`:** "I want
  to…" rows for adding a command, letting an agent drive the TUI, and
  running commands from the shell. **`README.md`:** Status.
* **Release notes** in the execution record.
* **Checks:** Rule 3, and `make release-check`.
* **The release:** the owner commits, pushes and, with CI green, tags
  `v0.4.0`. The agent then runs the consumer smoke test of
  `docs/guides/releasing.md` against `v0.4.0`, importing `command`,
  `when` and `command/cli`, and records it.
* **Done when:** `v0.4.0` is tagged and the smoke test builds.

### Step 10: `command/cobracmd`, the Cobra front end

**Starts when** `v0.4.0` is tagged.

* **Before the first file,** record in the execution record: the newest
  `github.com/spf13/cobra` and `github.com/spf13/pflag` versions and
  their `go.mod`s, the licence of each module the new `go.mod` adds, and
  `GOWORK=off govulncheck ./...` in the new module. A finding stops the
  step for the owner. 0010-MADR §5 named Cobra v1.10.2 and pflag v1.0.10.
* **Files:** `command/cobracmd/go.mod` (`module
  github.com/maccavelli/go-tui-lib/command/cobracmd`, `go 1.27.1`,
  requiring `github.com/maccavelli/go-tui-lib v0.4.0` and Cobra, no
  `replace`), `go.sum`, `cobracmd.go`, `flags.go`, `help.go`,
  `complete.go`; `command/cobracmd/docs/docs.go`; tests, and the shared
  front-end cases in `command/cobracmd/frontend_test.go`; goldens under
  `command/cobracmd/testdata/golden/`. `go work use ./command/cobracmd`.
  `AGENTS.md` and `docs/architecture.md` drop "(planned)" from the
  module's row.
* **Exported names** (MADR A1): `New`, `Mount`, `Run`, `Option`,
  `WithName`, `WithConfirm`, `WithContext`, `WithTheme`, `WithGlyphs`,
  `WithWidth`; in `docs`, `Man(w io.Writer, root *cobra.Command, date
  time.Time) error` and `Markdown(w io.Writer, root *cobra.Command) error`.
* **Build:** MADR A1's tree, flags, completion, verbs, exit codes, help and
  rules, as written there.
* **Tests:** the shared front-end cases agree with `command/cli`; no
  hidden or non-CLI command in the tree; nested commands and groups;
  `Mount` beside a program's own commands; an enum flag refuses and
  completes; golden help across 0001-MADR §6's matrix at 60 and 100
  columns; golden completion scripts for bash, zsh, fish and PowerShell;
  golden man and Markdown pages, unchanged between two runs; `Run` reads no
  `os.Args` and writes to no standard stream; a second `New` in one
  process works and registers no completion twice.
* **Mutations:** a flag's type from the Go field instead of the schema;
  `Destructive` without `--yes`; `Run` without `SetArgs`; a group added
  after its children; depguard allowing Cobra in the root.
* **Checks:** Rule 3, in the module's directory with `GOWORK=off`, and
  the tests again in workspace mode; `scripts/go-modules.sh --check`.
* **The release:** the owner commits, pushes and tags
  `command/cobracmd/v0.1.0`; the agent runs the smoke test against it.
* **Done when:** the tag exists and the smoke test builds.

### Step 11: `command/kongcmd`, the Kong front end

**Starts when** `v0.4.0` is tagged; it does not wait for Step 10.

* **Before the first file,** record the newest
  `github.com/alecthomas/kong` and its `go.mod`, each licence, and
  `GOWORK=off govulncheck ./...`. 0010-MADR §5 named Kong v1.16.1.
* **Files:** `command/kongcmd/go.mod` (as Step 10's, with Kong),
  `go.sum`, `kongcmd.go`, `grammar.go`, `help.go`, `complete.go`,
  `resolver.go`; tests, the shared front-end cases in
  `command/kongcmd/frontend_test.go`; goldens under
  `command/kongcmd/testdata/golden/`. `go work use ./command/kongcmd`.
  `AGENTS.md` and `docs/architecture.md` drop "(planned)".
* **Exported names** (MADR A1): `New`, `Adapter`, `Adapter.Options`,
  `Adapter.Parser`, `Adapter.Run`, `Adapter.Resolver`, `Option`,
  `WithName`, `WithConfirm`, `WithContext`, `WithTheme`, `WithGlyphs`,
  `WithWidth`.
* **Build:** MADR A1's mounting, grammars, dispatch, groups, help,
  settings, completion and rules, as written there.
* **Tests:** golden completion scripts for each shell; the shared
  front-end cases; one struct, two readers (`SchemaOf` and the Kong
  grammar agree on names, required fields, enums, defaults and positional
  order); nested `cmd` selection; `--help` returns 0 and parsing stops;
  mounting beside a program's own grammar, with `handled` false for its
  commands; help at the option's width whatever `$COLUMNS` says, golden
  across the matrix at 60 and 100 columns; no standard stream and no
  `os.Args`; `Resolver` fills a flag and a command-line flag wins; no `env`
  tag unless asked.
* **Mutations:** `Exit` returns instead of panicking; dispatch from the
  first registry command instead of `ctx.Selected()`; an enum's default
  dropped; `kong.Writers` unset; depguard allowing Kong in the root.
* **Checks:** as Step 10.
* **The release:** the owner tags `command/kongcmd/v0.1.0`; the agent
  runs the smoke test against it.
* **Done when:** the tag exists and the smoke test builds.

### Step 12: close-out

* Verification, item by item, in the execution record; this PLAN set
  `complete`, with `docs/README.md`'s row.
* **Done when:** every Verification item holds.

## Verification

* Every step's mutations are killed, and each earlier set again after a
  later change moves its anchors.
* Rule 3's checks are clean at every step, on the macOS development host
  and the Windows test host.
* `make fuzz` covers `layout`, `when` and `command`, and CI's `gates` job
  runs it.
* The root `go.mod` is unchanged from `v0.3.0` through `v0.4.0`, and never
  names Cobra or Kong; `go mod tidy -diff` is clean in every module.
* Each nested `go.mod` requires `v0.4.0`, has no `replace`, and passes
  every gate of 0010-MADR §4 with `GOWORK=off` and in workspace mode.
* depguard keeps Cobra and pflag in `command/cobracmd`, Kong in
  `command/kongcmd`, and refuses the Charm v1 paths, mcplib, the MCP
  go-sdk and go-llmprovider-sdk everywhere.
* `internal/conformance` finds nothing in `when`, `command`, `command/cli`,
  `cobracmd` and `kongcmd`.
* The shared front-end cases pass in all three front ends.
* The MCP field names match 2026-07-28's schema, and the ACP ones the ACP
  pages, as recorded in Step 6.
* The consumer smoke test builds against `v0.4.0`,
  `command/cobracmd/v0.1.0` and `command/kongcmd/v0.1.0`.
* The identifier scan finds nothing, and CI is green after each push.

## Rollout and Rollback

* **Rollout:** Steps 2 to 9 are pushed and `v0.4.0` is tagged; then each
  front end lands, requires `v0.4.0`, and is tagged under its prefix
  (0010-MADR §3). pi-go adopts the registry under its own records.
* **Rollback:** before a push, each step is one local commit; after it, a
  patch release fixes forward. The new packages are additive, so a
  consumer that does not import them is unaffected. Step 7 adds names to
  `workspace` and changes no existing one; removing them breaks only their
  callers. A front end is fixed forward under its own prefix while a
  consumer pins its previous tag; tags are never moved or deleted.

## Execution Record

### Step 2: `when`

#### Deviations

* **D1 (2026-10-05): the canonical form can outgrow the source limit.**
  `go test -run '^$' -fuzz=FuzzParse -fuzztime=30s ./when` failed: a
  minimized source under 4096 bytes, `A<000\…` with thousands of
  backslashes in a bareword, prints `A < '000\\…'`, "which does not
  parse: when: source of 6718 bytes, more than 4096". The canonical form
  quotes a bareword, doubles its backslashes and adds spaces around
  operators, so the PLAN's fuzz property and MADR §8's round trip were
  false at the limit. Options given: bound the canonical form too
  (recommended), or limit the source only and weaken the property to
  canonical forms that fit. **The owner picked "Bound the canonical form
  too".** `Parse` now refuses an expression whose canonical form is longer
  than `MaxSource`; recorded as MADR amendment A3. Added to the step:
  a `TestLimits` case, a seventh mutation, and the failing input as a
  regression seed in `when/testdata/fuzz/FuzzParse/`.
* **D2 (2026-10-05): a mutation survived.** "`String` drops needed
  parentheses" (`wrap`'s `n.prec() < need` made `n.prec() < need-10`)
  passed `TestStringRoundTrip`, which checked only that the canonical text
  reprints unchanged: `a && (b || !c)` printed `a && b || !c`, which is
  stable and means something else. Under Rule 4 the test was
  strengthened: `TestStringRoundTrip` also evaluates the expression and
  its re-parsed canonical form on every assignment of `true` and `false`
  to its keys, over three more sources that need parentheses; and
  `FuzzParse` fails when the two evaluate differently on its contexts.
  Seen to fail: the set below, and the same mutation fuzzed on a scratch
  copy (`-fuzz '^FuzzParse$' -fuzztime 20s`) failed on a seed in 0.07 s:
  `fuzz_test.go:51: "a && (b || !c)" prints "a && b || !c", which
  evaluates differently`.
* **D3 (2026-10-05): CI's fuzz step named only `layout`.** `make fuzz`
  runs in CI, so CI fuzzed `when` with no change; but the step's comment
  said only `layout` had fuzz targets, and on a failure it uploaded only
  `layout/testdata/fuzz/`, so a failing `when` input would be lost.
  `.github/workflows/ci.yml` was not in the step's files. Options given:
  add it to Step 2 (recommended), or leave it as a known gap. **The owner
  picked "Add ci.yml to Step 2".** The comment names both packages, and
  the artifact's `path` lists `layout/testdata/fuzz/` and
  `when/testdata/fuzz/`. `actionlint` is not installed here; the file
  parses as YAML, and the step's `path` reads back as the two lines.

#### What was built

* `when/when.go`, `lex.go`, `parse.go`, `eval.go`, `value.go`, `key.go`:
  the exported names listed above and no others, plus the constants
  `MaxSource` and `MaxDepth`. A hand-written scanner and a
  recursive-descent parser; a regex compiles with `regexp` at parse time,
  with the flags `i`, `m` and `s` written as `(?ims)`. `Parse` refuses a
  canonical form over `MaxSource` (D1). The package imports only the
  standard library.
* `when/when_test.go`: the twelve tests of the table, and
  `TestParseErrors`, `TestKeysInOrder` and `TestZeroExpr`.
  `when/fuzz_test.go`: `FuzzParse`, seeded from the operator table and
  five more sources. `when/bench_test.go`: `BenchmarkParse` and
  `BenchmarkEval`. `when/testdata/fuzz/FuzzParse/01e90e822ccb8b53`: D1's
  input.
* `Makefile`: the `fuzz` target runs `./scripts/go-fuzz.sh -t
  $(FUZZTIME) -m 1 ./when` after `./layout`, and its help text names
  `when`. `.github/workflows/ci.yml`: D3.
* Lint findings fixed on the way, before the checks below: `goconst` on
  `"true"` and `"false"` (now `trueWord` and `falseWord`), and
  `gocritic`'s `regexpSimplify` (`[0-9]` is `\d`, which RE2 limits to
  ASCII).

#### Checks (Rule 3)

1. `gofmt -l when`: no output.
2. `make pre-add-check FILES="<the nine when/*.go files>"`: exit 0,
   "go-precheck: 9 file(s) clean in 1 module(s) (gofmt, golangci-lint, go
   vet, go test, go mod tidy, govulncheck)."
3. `make lint`: exit 0; "0 issues." for `GOOS=linux`, `darwin` and
   `windows`.
4. `GOWORK=off go test -race -count=1 ./...`: exit 0, 14 packages `ok`,
   `when` among them; `internal/conformance` passes with `when` in its
   scan.
5. `GOWORK=off go test -shuffle=on -count=2 ./...`: exit 0, 14 packages
   `ok`.
6. `LC_ALL=C GOWORK=off go test -count=1 ./...`: exit 0, 14 packages `ok`.
7. `GOWORK=off go mod tidy -diff`: exit 0, no output.
8. `make fuzz FUZZTIME=20s`: exit 0; "go-fuzz: 1 fuzz targets ran clean
   in ./layout", "go-fuzz: 1 fuzz targets ran clean in ./when"
   (`FuzzParse`, 5,701,018 executions). No new input was written.
9. Windows test host, go1.27.1 windows/amd64, on a copy of the tree:
   `make pre-add-check` exit 0 ("83 file(s) clean in 1 module(s)"),
   `make lint` exit 0 ("0 issues." for each target), `make vuln` exit 0
   ("No vulnerabilities found."), `GOWORK=off go test -count=3
   -shuffle=on ./when/` exit 0.
10. `markdownlint-cli2 --config .markdownlint-cli2.jsonc` on
    `docs/README.md` and this pair: "0 issues in 0 files"; the
    configuration excludes MADR and PLAN files, so only `docs/README.md`
    is linted. The relative-link check of the three files: 0 broken.
11. Identifier scan of the diff and the new files, for the local account
    name, the test host's name, home-directory paths and the employer's
    domain: no match.

Benchmarks, on the development host (`go test -run '^$' -bench .
-benchmem ./when`): `BenchmarkParse` 5278 ns/op, 11173 B/op, 69 allocs/op;
`BenchmarkEval` 110.2 ns/op, 32 B/op, 1 allocs/op.

#### Mutations (Rule 4)

Each on a scratch copy of the tree, with `go test -count=1 -run '^<Test>$'
./when`. All seven killed:

| Mutation | Applied as | Failing line |
| :--- | :--- | :--- |
| `&&` binds looser than `\|\|` | `or` loops on `tAnd` building `andNode`, `and` on `tOr` building `orNode` | `when_test.go:67: foo \|\| bar && baz read as (foo \|\| bar) && baz` |
| an unset key compares equal to `''` | `cmpNode.eval` on an unset key uses `StringValue("")` | `when_test.go:93: an unset key equals ''` |
| the depth limit is not checked | `p.depth > MaxDepth*1000` | `when_test.go:220: 65 nested parentheses parsed` |
| `Layered` takes the last match | the loop runs from the last context | `when_test.go:155: k = bottom, true; want the first context's` |
| `String` drops needed parentheses | `n.prec() < need-10` | `when_test.go:275: "!(a && b)" prints "!a && b", which differs on map[a:false b:false]` (after D2) |
| `not in` is evaluated as `in` | `return in` | `when_test.go:105: "item not in l" = true, want false` |
| D1: the canonical form's length is not checked | `n > MaxSource*100` | `when_test.go:212: a source of 4096 bytes printing longer than MaxSource: <nil>` |
