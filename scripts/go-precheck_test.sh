#!/usr/bin/env bash
# Tests for go-precheck.sh's gofmt step and its choice of files, on
# throwaway repositories in a temporary directory, never in this one
# (docs/decisions/0015-PLAN-precheck-gofmt-errors.md Step 2).
#
# gofmt, go vet and go test are the real ones; golangci-lint is a stub that
# passes, and the network gates (govulncheck, the API diff gate, the
# framework examples) are skipped. Each case gets a repository of its own:
# a module with one package and its test, and a copy of the scripts.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PRECHECK="${PRECHECK:-$ROOT/scripts/go-precheck.sh}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
unset GOWORK

printf '#!/bin/sh\nexit 0\n' >"$WORK/golangci-lint"
chmod +x "$WORK/golangci-lint"
export GOLANGCI_LINT="$WORK/golangci-lint"
export GO_PRECHECK_SKIP_VULN=1 GO_PRECHECK_SKIP_APICHECK=1 GO_PRECHECK_SKIP_EXAMPLES=1

PASS=0
FAIL=0

check() { # name want got
	if [ "$2" = "$3" ]; then
		echo "  ok   $1"
		PASS=$((PASS + 1))
	else
		echo "  FAIL $1: want $2, got $3"
		sed 's/^/         /' "$WORK/out"
		FAIL=$((FAIL + 1))
	fi
}

has() { # name pattern
	if grep -q -F -e "$2" "$WORK/out"; then
		echo "  ok   $1"
		PASS=$((PASS + 1))
	else
		echo "  FAIL $1: no line holds $2 in:"
		sed 's/^/         /' "$WORK/out"
		FAIL=$((FAIL + 1))
	fi
}

hasnt() { # name pattern
	if grep -q -F -e "$2" "$WORK/out"; then
		echo "  FAIL $1: a line holds $2 in:"
		sed 's/^/         /' "$WORK/out"
		FAIL=$((FAIL + 1))
	else
		echo "  ok   $1"
		PASS=$((PASS + 1))
	fi
}

g() { # git, as a throwaway identity
	git -c user.name=precheck -c user.email=precheck@example.invalid -c commit.gpgsign=false "$@"
}

# repo DIR: a repository holding module example.com/r with package p and
# its test, go.work, and the two scripts, committed.
repo() {
	mkdir -p "$1/p" "$1/scripts"
	cp "$PRECHECK" "$1/scripts/go-precheck.sh"
	cp "$ROOT/scripts/go-modules.py" "$1/scripts/go-modules.py"
	(
		cd "$1"
		git init -q
		printf 'module example.com/r\n\ngo 1.27.1\n' >go.mod
		printf 'go 1.27.1\n\nuse .\n' >go.work
		printf 'package p\n\n// F is a function.\nfunc F() int { return 1 }\n' >p/p.go
		printf 'package p\n\nimport "testing"\n\nfunc TestF(t *testing.T) {\n\tif F() != 1 {\n\t\tt.Fatal(F())\n\t}\n}\n' >p/p_test.go
		g add -A
		g commit -q -m base
	)
}

# run DIR [FILES...]: the precheck's exit status, with output in $WORK/out.
run() {
	local dir="$1"
	shift
	set +e
	(cd "$dir" && ./scripts/go-precheck.sh "$@") </dev/null >"$WORK/out" 2>&1
	local rc=$?
	set -e
	echo "$rc"
}

broken='package main

func main( {'

# 1. A clean tree.
repo "$WORK/a"
check "a clean tree passes" 0 "$(run "$WORK/a")"
has "it says clean" "file(s) clean"

# 2. An unformatted file fails, named under gofmt.
repo "$WORK/b"
printf 'package p\n\n// F is a function.\nfunc  F() int { return 1 }\n' >"$WORK/b/p/p.go"
check "an unformatted file fails" 1 "$(run "$WORK/b" p/p.go)"
has "it is named" "these files are not formatted"

# 3. A file under testdata that does not parse fails at gofmt, in a file
# list: go vet and go test never see testdata, so only gofmt can.
repo "$WORK/c"
mkdir -p "$WORK/c/testdata/x"
printf '%s\n' "$broken" >"$WORK/c/testdata/x/broken.go"
check "a testdata file that does not parse fails" 1 "$(run "$WORK/c" testdata/x/broken.go)"
has "gofmt's failure is named" "gofmt: failed"
has "gofmt's message is shown" "expected ')'"

# 4. The same file, tracked, fails with no list.
(cd "$WORK/c" && g add -A && g commit -q -m broken)
check "a tracked file that does not parse fails, no list" 1 "$(run "$WORK/c")"
has "gofmt's failure is named, no list" "gofmt: failed"

# 5. A tracked test file deleted from the work tree is not checked.
repo "$WORK/d"
rm "$WORK/d/p/p_test.go"
check "a deleted tracked file is skipped, no list" 0 "$(run "$WORK/d")"
hasnt "gofmt is not given it" "lstat"

# 6. A file list naming a missing file and a good one.
repo "$WORK/e"
check "a missing file in a list is ignored" 0 "$(run "$WORK/e" p/gone.go p/p.go)"

echo "go-precheck_test: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
