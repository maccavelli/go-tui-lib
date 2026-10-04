# Releasing

This repository holds more than one Go module: the root,
`github.com/maccavelli/go-tui-lib`, and nested adapter modules beside it,
such as `command/kongcmd`. Each module is released on its own, with its own
tags. Why it is built this way is in
[0010-MADR](../decisions/0010-MADR-nested-adapter-modules.md); the evidence
is in [0010-REPORT](../reports/0010-REPORT-nested-modules-and-adapter-sources.md).

Tags and pushes are the owner's. Nothing below runs without that.

## The rules

- **A tag names one module.** The root's tags are `vX.Y.Z`. A nested
  module's tags carry its directory: `command/kongcmd/v0.1.0`. Each module's
  versions are independent and start at `v0`. A consumer writes the version
  alone: `go get github.com/maccavelli/go-tui-lib/command/kongcmd@v0.1.0`.
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
   mkdir -p command/kongcmd && cd command/kongcmd
   go mod init github.com/maccavelli/go-tui-lib/command/kongcmd
   GOWORK=off go get github.com/maccavelli/go-tui-lib@vX.Y.Z
   cd ../..
   go work use ./command/kongcmd
   ```

3. If it brings a dependency only it may import, add a depguard rule to
   `.golangci.yml` over `$all` less `!**/<dir>/**`, as the `cobra`, `kong`
   and `glamour` rules do.
4. Run `scripts/go-modules.sh --check` and `make release-check`. Both now
   cover the new module. Commit `go.work` with the module.

## Release the root alone

1. `make release-check`, `make lint` and `make vuln` pass.
2. The owner pushes `main`, and CI passes: every `test (<module>, <os>)`
   entry, `modules` and `gates`.
3. The owner tags the commit CI passed, and pushes the tag:

   ```bash
   git tag -a vX.Y.Z <commit>
   git push origin vX.Y.Z
   ```

4. Run the consumer smoke test below for the root.

An adapter that should build against the new root then moves its
requirement, as step 2 of "Release a change that spans the root and an
adapter" shows.

## Release an adapter alone

1. The adapter's `go.mod` already requires a published root tag.
2. `make release-check` passes, and CI passes on the pushed commit.
3. The owner tags it with the directory prefix, and pushes the tag:

   ```bash
   git tag -a command/kongcmd/vA.B.C <commit>
   git push origin command/kongcmd/vA.B.C
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
   cd command/kongcmd
   GOWORK=off go get github.com/maccavelli/go-tui-lib@vX.Y.Z
   GOWORK=off go mod tidy
   ```

   The adapter's part lands, CI passes, and the owner tags
   `command/kongcmd/vA.B.C`.

## The consumer smoke test

After a tag, prove that a consumer can fetch and build it. Work in a scratch
module outside the repository, so that no `go.work` applies:

```bash
cd "$(mktemp -d)"
go mod init example.com/smoke
go get github.com/maccavelli/go-tui-lib/command/kongcmd@vA.B.C
cat >main.go <<'EOF'
package main

import _ "github.com/maccavelli/go-tui-lib/command/kongcmd"

func main() {}
EOF
go build ./...
```

For the root, `go get github.com/maccavelli/go-tui-lib@vX.Y.Z` and import
one of its packages, such as `glyph`, instead. If `go get` cannot find a
nested module's tag, check that the tag's prefix is the module's directory
exactly. The Go module reference says: "Each tag name must be prefixed
with the module subdirectory, followed by a slash" (0010-REPORT §1).
