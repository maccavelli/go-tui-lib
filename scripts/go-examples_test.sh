#!/usr/bin/env bash
# Tests for go-examples.sh, on throwaway trees in a temporary directory,
# never this one (docs/decisions/0014-PLAN-api-policy-gates.md Step 5).
#
# Each tree holds one program with no requirement, so the tests need no
# network: it prints its argument to stdout and exits with status 3.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EXAMPLES="${EXAMPLES:-$ROOT/scripts/go-examples.sh}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
unset GOWORK GO_PRECHECK_SKIP_EXAMPLES

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

has() { # name pattern: whether the last run's output matches
	if grep -q -F -e "$2" "$WORK/out"; then
		echo "  ok   $1"
		PASS=$((PASS + 1))
	else
		echo "  FAIL $1: no line holds $2 in:"
		sed 's/^/         /' "$WORK/out"
		FAIL=$((FAIL + 1))
	fi
}

# tree DIR: a tree with one program, echo, one passing case, and a guide
# whose Go block is the program's region.
tree() {
	mkdir -p "$1/testdata/frameworks/echo" "$1/docs/guides"
	printf 'module example.com/frameworks\n\ngo 1.27.1\n\nreplace github.com/maccavelli/go-tui-lib => @ROOT@\n' >"$1/testdata/frameworks/go.mod.tmpl"
	: >"$1/testdata/frameworks/go.sum"
	cat >"$1/testdata/frameworks/echo/main.go" <<'EOF'
// Command echo prints its argument.
package main

import (
	"fmt"
	"os"
)

func main() {
	// guide:echo
	if len(os.Args) > 1 {
		fmt.Println(os.Args[1])
	}
	// guide:end
	os.Exit(3)
}
EOF
	printf 'echo hello => 3 stdout hello\n' >"$1/testdata/frameworks/cases.txt"
	cat >"$1/docs/guides/g.md" <<'EOF'
# A guide

<!-- from: testdata/frameworks/echo/main.go#echo -->

```go
if len(os.Args) > 1 {
    fmt.Println(os.Args[1])
}
```
EOF
}

# run DIR [ARGS]: the script's exit status, with output in $WORK/out.
run() {
	local dir="$1"
	shift
	set +e
	EXAMPLES_ROOT="$dir" "$EXAMPLES" "$@" >"$WORK/out" 2>&1
	local rc=$?
	set -e
	echo "$rc"
}

# 1. A passing case, and an excerpt equal to its region once dedented and
# with tabs as four spaces.
tree "$WORK/a"
check "a passing tree passes" 0 "$(run "$WORK/a")"
has "the case ran" "1 case(s) run"
has "the excerpt was checked" "1 excerpt(s) checked"

# 2. A wrong exit status fails.
tree "$WORK/b"
printf 'echo hello => 0 stdout hello\n' >"$WORK/b/testdata/frameworks/cases.txt"
check "a wrong exit fails" 1 "$(run "$WORK/b")"
has "the exit is named" "exit 3, want 0"

# 3. The substring in the wrong stream fails.
tree "$WORK/c"
printf 'echo hello => 3 stderr hello\n' >"$WORK/c/testdata/frameworks/cases.txt"
check "a substring in the other stream fails" 1 "$(run "$WORK/c")"
has "the stream is named" 'stderr does not hold "hello"'

# 4. An excerpt one byte away from its region fails, in --check-guide too.
tree "$WORK/d"
sed 's/Println/Printl/' "$WORK/d/docs/guides/g.md" >"$WORK/d/g.md" && mv "$WORK/d/g.md" "$WORK/d/docs/guides/g.md"
check "a one-byte excerpt difference fails" 1 "$(run "$WORK/d" --check-guide)"
has "the difference is shown" "+    fmt.Printl(os.Args[1])"

# 5. A region the program does not have fails.
tree "$WORK/e"
sed 's/#echo/#missing/' "$WORK/e/docs/guides/g.md" >"$WORK/e/g.md" && mv "$WORK/e/g.md" "$WORK/e/docs/guides/g.md"
check "a missing region fails" 1 "$(run "$WORK/e" --check-guide)"
has "the region is named" 'has no region "// guide:missing"'

# 6. A from line with no Go block after it fails.
tree "$WORK/f"
printf '\n<!-- from: testdata/frameworks/echo/main.go#echo -->\n\nNo block.\n' >>"$WORK/f/docs/guides/g.md"
check "a from line without a block fails" 1 "$(run "$WORK/f" --check-guide)"
has "it is named" "is not followed by a Go block"

# 7. --check-guide neither builds nor runs: a program that does not compile
# passes it, and fails the full run with exit 2.
tree "$WORK/g"
printf 'package main\n\nfunc main() { undefined() }\n' >"$WORK/g/testdata/frameworks/echo/broken.go"
check "--check-guide does not build" 0 "$(run "$WORK/g" --check-guide)"
check "a build failure exits 2" 2 "$(run "$WORK/g")"
has "the update is suggested" "scripts/go-examples.sh --update"

# 8. --update writes go.mod.tmpl and go.sum back, with @ROOT@ kept.
tree "$WORK/h"
check "--update passes" 0 "$(run "$WORK/h" --update)"
if grep -q '=> @ROOT@$' "$WORK/h/testdata/frameworks/go.mod.tmpl" && ! grep -q -F "$WORK" "$WORK/h/testdata/frameworks/go.mod.tmpl"; then
	check "--update keeps @ROOT@" yes yes
else
	check "--update keeps @ROOT@" yes no
fi

# 9. The skip switch.
set +e
GO_PRECHECK_SKIP_EXAMPLES=1 EXAMPLES_ROOT="$WORK/g" "$EXAMPLES" >"$WORK/out" 2>&1
rc=$?
set -e
check "the skip switch passes" 0 "$rc"

echo "go-examples_test: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
