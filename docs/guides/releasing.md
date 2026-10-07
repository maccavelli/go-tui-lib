# Releasing

This repository can hold more than one Go module: the root,
`github.com/maccavelli/go-tui-lib`, and, when an adapter needs a dependency
the root must not carry, a nested module beside it, such as the planned
`stream/glamourmd`. Each module is released on its own, with its own
tags. Why it is built this way is in
[0010-MADR](../decisions/0010-MADR-nested-adapter-modules.md); the evidence
is in [0010-REPORT](../reports/0010-REPORT-nested-modules-and-adapter-sources.md).

Tags and pushes are the owner's. Nothing below runs without that.

Before every push, the disclosure guard runs over the outgoing commits
(AGENTS.md, Identifiers), and is never bypassed with `--no-verify`:

```bash
echo "refs/heads/main $(git rev-parse HEAD) refs/heads/main $(git rev-parse origin/main)" |
  python3 ~/.global-git-hooks/github-disclosure.py pre-push origin "$(git remote get-url origin)"
```

## The rules

- **A tag names one module.** The root's tags are `vX.Y.Z`. A nested
  module's tags carry its directory: `stream/glamourmd/v0.1.0`. Each
  module's versions are independent and start at `v0`. A consumer writes
  the version alone: `go get
  github.com/maccavelli/go-tui-lib/stream/glamourmd@v0.1.0`.
- **A published tag is never moved or deleted.** A proxy and the checksum
  database may already hold it. A mistake is fixed by a later version.
- **An adapter requires a published root version.** Its `go.mod` names a
  `vX.Y.Z` tag of the root, never a pseudo-version, and has no `replace`.
  `scripts/go-precheck.sh` fails otherwise.
- **`go.work` is for development,** and is committed; `go.work.sum` is
  ignored, since only ad-hoc workspace-mode commands write it. It lets a
  root change and an adapter change be built and tested together
  before either is tagged. Every gate
  also runs with `GOWORK=off`, which is what a consumer builds.

## Add a module

1. Write its records first. A module that requires anything new needs a
   MADR that names it (AGENTS.md, Dependencies).
2. Create it, require the root at its latest tag, and add it to the
   workspace:

   ```bash
   mkdir -p <dir>
   (cd <dir> &&
     go mod init github.com/maccavelli/go-tui-lib/<dir> &&
     GOWORK=off go get github.com/maccavelli/go-tui-lib@vX.Y.Z)
   go work use ./<dir>
   ```

3. If it brings a dependency only it may import, add a depguard rule to
   `.golangci.yml` over `$all` less `!**/<dir>/**`, as the `glamour` rule
   does.
4. Run `scripts/go-modules.sh --check` and `make release-check`. Both now
   cover the new module. Commit `go.work` with the module.

## Release the root alone

1. `make release-check`, `make lint` and `make vuln` pass.
2. The disclosure guard passes, the owner pushes `main`, and CI passes:
   every `test (<module>, <os>)` entry, `modules` and `gates`.
3. The owner tags the commit CI passed, and pushes the tag:

   ```bash
   git tag -a vX.Y.Z -m "vX.Y.Z" <commit>
   git push origin vX.Y.Z
   ```

4. Run the consumer smoke test below for the root.

An adapter that should build against the new root then moves its
requirement, as step 2 of "Release a change that spans the root and an
adapter" shows.

## Release an adapter alone

1. The adapter's `go.mod` already requires a published root tag.
2. `make release-check` passes, the disclosure guard passes before the
   push, and CI passes on the pushed commit.
3. The owner tags it with the directory prefix, and pushes the tag:

   ```bash
   git tag -a stream/glamourmd/vA.B.C -m "stream/glamourmd/vA.B.C" <commit>
   git push origin stream/glamourmd/vA.B.C
   ```

4. Run the consumer smoke test for the adapter.

## Release a change that spans the root and an adapter

The change lands in two steps, because an adapter can only require a root
that is already published.

1. **The root first.** The root's part lands, CI passes, and the owner tags
   `vX.Y.Z`, as above. While developing, `go.work` builds the adapter
   against the unreleased root. Until this tag exists, the adapter's
   `GOWORK=off` gates fail, by design: it does not yet build against any
   root a consumer can get.
2. **Then the adapter.** Move its requirement to the new tag:

   ```bash
   cd stream/glamourmd
   GOWORK=off go get github.com/maccavelli/go-tui-lib@vX.Y.Z
   GOWORK=off go mod tidy
   ```

   The adapter's part lands, CI passes, and the owner tags
   `stream/glamourmd/vA.B.C`.

## Retire a module

A published module cannot be withdrawn: the proxy and the checksum
database keep every version. It is retired in two steps
([0012-MADR](../decisions/0012-MADR-bring-your-own-cli.md)):

1. **Deprecate and retract it.** Its `go.mod` gets a `// Deprecated:`
   comment above `module`, saying what to use instead, and a `retract`
   directive for every version, the new one included:

   ```text
   // Deprecated: <what to use instead>.
   module github.com/maccavelli/go-tui-lib/<dir>

   // <why>.
   retract [v0.1.0, v0.1.1]
   ```

   CI passes, and the owner tags the new version, `<dir>/v0.1.1`, which
   carries the retraction.
2. **Then remove it** from `main`: the directory, its `go.work` entry and
   its depguard rule. Its tags stay.

Afterwards, `go list -m -u` shows a user of an old version `(retracted)
(deprecated)`, and `go get <module>@latest` finds no version once `main`
no longer holds the module. The module proxy may answer `@latest` from
its cache for a while.

## The consumer smoke test

After a tag, prove that a consumer can fetch and build it. Work in a scratch
module outside the repository, so that no `go.work` applies:

```bash
cd "$(mktemp -d)"
go mod init example.com/smoke
cat >main.go <<'EOF'
package main

import _ "github.com/maccavelli/go-tui-lib/stream/glamourmd"

func main() {}
EOF
go get github.com/maccavelli/go-tui-lib/stream/glamourmd@vA.B.C
go mod tidy
test -z "$(go env GOWORK)"
go vet ./...
go build ./...
go run .
```

`go env GOWORK` must be empty, so that no workspace stands in for the
published version, and the program is vetted and run, not only built.

The program comes first and `go mod tidy` is not optional: `go get` of a
module path records the module, not the `go.sum` entries of the packages
the program imports, and without them `go build` fails with "missing
go.sum entry".

For the root, `go get github.com/maccavelli/go-tui-lib@vX.Y.Z` and import
one of its packages that has outside dependencies, such as `workspace`,
instead. A package with none, such as `glyph`, builds even when those
entries are missing, so it proves less. If `go get` cannot find a
nested module's tag, check that the tag's prefix is the module's directory
exactly. The Go module reference says: "Each tag name must be prefixed
with the module subdirectory, followed by a slash" (0010-REPORT §1).
