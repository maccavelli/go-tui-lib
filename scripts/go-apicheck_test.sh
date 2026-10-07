#!/usr/bin/env bash
# Tests for go-apicheck.sh, on throwaway repositories in a temporary
# directory, never in this one
# (docs/decisions/0014-PLAN-api-policy-gates.md Step 4).
#
# The removal cases are the ones that matter: a gate that passed an
# unlisted incompatible change, or compared a tag with itself, would be no
# gate at all. apidiff is installed once, at the Makefile's pin, and every
# case uses that binary.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APICHECK="${APICHECK:-$ROOT/scripts/go-apicheck.sh}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
unset GOWORK

PASS=0
FAIL=0

check() { # name want got
	if [ "$2" = "$3" ]; then
		echo "  ok   $1"
		PASS=$((PASS + 1))
	else
		echo "  FAIL $1: want $2, got $3"
		FAIL=$((FAIL + 1))
	fi
}

has() { # name pattern: whether the last run's output matches
	if grep -q -e "$2" "$WORK/out"; then
		echo "  ok   $1"
		PASS=$((PASS + 1))
	else
		echo "  FAIL $1: no line matches $2 in:"
		sed 's/^/         /' "$WORK/out"
		FAIL=$((FAIL + 1))
	fi
}

# apidiff is installed as go-apicheck.sh installs it, through the configured
# proxy. The cases then run with GOPROXY=off: the throwaway modules require
# nothing, so the go command must never need the network
# (docs/decisions/0014-PLAN-api-policy-gates.md, deviation D9).
version="$(sed -n 's/^APIDIFF_VERSION[[:space:]]*?=[[:space:]]*//p' "$ROOT/Makefile" | head -1)"
GOBIN="$WORK/bin" GOWORK=off go install "golang.org/x/exp/cmd/apidiff@$version"
APIDIFF_BIN="$WORK/bin/apidiff"
[ -x "$APIDIFF_BIN.exe" ] && APIDIFF_BIN="$APIDIFF_BIN.exe"
export GOPROXY=off

g() { # git, as a throwaway identity
	git -c user.name=apicheck -c user.email=apicheck@example.invalid -c commit.gpgsign=false -c tag.gpgsign=false "$@"
}

# repo DIR: a repository whose root is module example.com/r, with package p
# exporting F and G, a go.work and an empty allow file, committed.
repo() {
	mkdir -p "$1/p"
	(
		cd "$1"
		git init -q
		printf 'module example.com/r\n\ngo 1.27.1\n' >go.mod
		printf 'go 1.27.1\n\nuse .\n' >go.work
		printf 'package p\n\n// F is kept.\nfunc F() {}\n\n// G is removed by some cases.\nfunc G() {}\n' >p/p.go
		: >allow
		g add -A
		g commit -q -m base
	)
}

# run DIR [APIDIFF]: the script's exit status, with output in $WORK/out.
run() {
	local dir="$1" bin="${2:-$APIDIFF_BIN}"
	set +e
	(cd "$dir" && APIDIFF="$bin" APICHECK_ALLOW="$dir/allow" "$APICHECK") >"$WORK/out" 2>&1
	local rc=$?
	set -e
	echo "$rc"
}

# 1. No tag: the module was never released, and is skipped.
repo "$WORK/a"
check "no previous tag passes" 0 "$(run "$WORK/a")"
has "no previous tag is said" 'no previous tag, skipped'

# 2. Tagged, unchanged: clean. A tag on HEAD is never the base (case 7),
# so each tag here is followed by a commit.
(cd "$WORK/a" && g tag v0.1.0 && g commit -q --allow-empty -m next)
check "an unchanged API passes" 0 "$(run "$WORK/a")"
has "against the tag" 'against v0.1.0, 0 incompatible'

# 3. An addition is compatible.
printf '\n// H is added.\nfunc H() {}\n' >>"$WORK/a/p/p.go"
check "an addition passes" 0 "$(run "$WORK/a")"

# 4. A removal fails, naming the change.
printf 'package p\n\n// F is kept.\nfunc F() {}\n' >"$WORK/a/p/p.go"
check "a removal fails" 1 "$(run "$WORK/a")"
has "the removal is named" 'p.G: removed'

# 5. The same removal, listed, passes.
line="$(grep 'p.G: removed' "$WORK/out" | head -1 | sed 's/^[[:space:]]*//' || true)"
printf '# a test\n%s\n' "$line" >"$WORK/a/allow"
check "a listed removal passes" 0 "$(run "$WORK/a")"

# 6. A listed line apidiff no longer prints is stale, and fails.
(cd "$WORK/a" && g checkout -q -- p/p.go)
check "a stale entry fails" 1 "$(run "$WORK/a")"
has "the stale entry is named" 'stale'
: >"$WORK/a/allow"

# 7. On a tagged commit, the base is the tag before it.
printf 'package p\n\n// F is kept.\nfunc F() {}\n' >"$WORK/a/p/p.go"
(cd "$WORK/a" && g commit -q -am "remove G" && g tag v0.2.0)
check "a tag on HEAD compares with the tag before" 1 "$(run "$WORK/a")"
has "against the earlier tag" 'against v0.1.0'

# 8. A package moved into a nested module is gone from the root. With
# GOWORK=off the root's export no longer holds it; in workspace mode it
# would, and the removal would pass unseen.
repo "$WORK/b"
mkdir -p "$WORK/b/sub"
printf 'package sub\n\n// S is exported.\nfunc S() {}\n' >"$WORK/b/sub/s.go"
(cd "$WORK/b" && g add -A && g commit -q -m sub && g tag v0.1.0)
printf 'module example.com/r/sub\n\ngo 1.27.1\n' >"$WORK/b/sub/go.mod"
printf 'go 1.27.1\n\nuse (\n\t.\n\t./sub\n)\n' >"$WORK/b/go.work"
(cd "$WORK/b" && g add -A && g commit -q -m "sub is a module")
check "a package moved out of the root fails" 1 "$(run "$WORK/b")"
has "the moved package is named" 'sub.*removed'
has "the new module has no tag yet" 'sub: no previous tag, skipped'

# 9. A nested module is compared with its own prefixed tag.
(cd "$WORK/b" && g tag sub/v0.1.0 && g tag v0.2.0 && g commit -q --allow-empty -m next)
printf 'package sub\n\n// T replaces S.\nfunc T() {}\n' >"$WORK/b/sub/s.go"
check "a nested module's removal fails" 1 "$(run "$WORK/b")"
has "it is compared with sub/v0.1.0" 'sub: against sub/v0.1.0, 1 incompatible'
has "it is listed under the module" "^  sub	.*S: removed"

# 10. A failing apidiff is a tool failure.
printf '#!/bin/sh\nexit 1\n' >"$WORK/fail.sh"
chmod +x "$WORK/fail.sh"
check "a failing apidiff exits 2" 2 "$(run "$WORK/a" "$WORK/fail.sh")"

echo "go-apicheck_test: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
