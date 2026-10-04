#!/usr/bin/env bash
# Offline tests for go-modules.sh, on throwaway repositories in a temporary
# directory, never in this one
# (docs/decisions/0010-PLAN-nested-adapter-modules.md Phase 3).
#
# The --check failures are the cases that matter: a check that passed with a
# module missing from go.work would let a gate skip that module.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODULES="${MODULES:-$ROOT/scripts/go-modules.sh}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

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

# repo DIR: a git repository whose root is a module, with a go.work.
repo() {
	mkdir -p "$1"
	git -C "$1" init -q
	printf 'module example.com/root\n\ngo 1.27.1\n' >"$1/go.mod"
	printf 'package root\n' >"$1/root.go"
	printf 'go 1.27.1\n\nuse .\n' >"$1/go.work"
}

# nested DIR SUB: a module at DIR/SUB.
nested() {
	mkdir -p "$1/$2"
	printf 'module example.com/root/%s\n\ngo 1.27.1\n' "$2" >"$1/$2/go.mod"
	printf 'package %s\n' "$(basename "$2")" >"$1/$2/sub.go"
}

# run DIR ARGS...: the script's exit status, with output in $WORK/out.
# RUN_GO, when set, is the go command the script runs.
run() {
	local dir="$1"
	shift
	set +e
	(cd "$dir" && GO="${RUN_GO:-go}" GOWORK=off "$MODULES" "$@") >"$WORK/out" 2>&1
	local rc=$?
	set -e
	echo "$rc"
}

# 1. A root and one nested module, both in go.work: two lines, and --check
#    passes. GOWORK=off in the caller does not hide the nested module.
repo "$WORK/two"
nested "$WORK/two" sub
printf 'go 1.27.1\n\nuse (\n\t.\n\t./sub\n)\n' >"$WORK/two/go.work"
git -C "$WORK/two" add -A
check "listing exits 0" 0 "$(run "$WORK/two")"
check "listing prints the root and sub" ".,sub" "$(sort "$WORK/out" | paste -sd, -)"
check "--check passes" 0 "$(run "$WORK/two" --check)"

# 2. A nested go.mod missing from go.work: --check fails and names it.
repo "$WORK/missing"
nested "$WORK/missing" sub
git -C "$WORK/missing" add -A
check "a module missing from go.work fails --check" 1 "$(run "$WORK/missing" --check)"
check "the failure names it" 1 "$(grep -c 'sub/go.mod is tracked, but go.work does not list sub' "$WORK/out" || true)"

# 3. A go.work entry with no go.mod: --check fails.
repo "$WORK/ghost"
mkdir -p "$WORK/ghost/ghost"
printf 'go 1.27.1\n\nuse (\n\t.\n\t./ghost\n)\n' >"$WORK/ghost/go.work"
git -C "$WORK/ghost" add -A
check "a go.work entry with no go.mod fails --check" 1 "$(run "$WORK/ghost" --check)"
check "the failure names it" 1 "$(grep -c 'cannot load module ghost listed in go.work' "$WORK/out" || true)"

# 4. A go.mod that is not tracked: --check fails, because the module would
#    not reach a commit.
repo "$WORK/untracked"
nested "$WORK/untracked" sub
printf 'go 1.27.1\n\nuse (\n\t.\n\t./sub\n)\n' >"$WORK/untracked/go.work"
git -C "$WORK/untracked" add go.mod go.work root.go
check "an untracked module in go.work fails --check" 1 "$(run "$WORK/untracked" --check)"
check "the failure names it" 1 "$(grep -c 'go.work lists sub, which has no tracked go.mod' "$WORK/out" || true)"

# 5. A go.mod under testdata, or under a "_" or "." directory, is not a
#    module the go command builds: --check skips it.
repo "$WORK/ignored"
nested "$WORK/ignored" testdata/fixture
nested "$WORK/ignored" _scratch
git -C "$WORK/ignored" add -A
check "ignored go.mod files pass --check" 0 "$(run "$WORK/ignored" --check)"

# 6. No go.work: --check fails, and listing is a usage error.
repo "$WORK/nowork"
rm "$WORK/nowork/go.work"
git -C "$WORK/nowork" add -A
check "no go.work fails --check" 1 "$(run "$WORK/nowork" --check)"
check "no go.work fails listing" 2 "$(run "$WORK/nowork")"

# 7. An unknown argument is a usage error.
check "an unknown argument exits 2" 2 "$(run "$WORK/two" --bogus)"

# 8. The go command spells the module directories differently from git, as
#    on Windows, where go prints C:\Users\... and git C:/Users/...
#    (docs/decisions/0010-PLAN-nested-adapter-modules.md deviation D1). On a
#    host where go's spelling is already git's, a go that reports them
#    through a symlink to the repository stands in for it. Either way the
#    list and --check must not change.
TOP="$(git -C "$WORK/two" rev-parse --show-toplevel)"
SPELL=go
if [ "$(GOWORK="$TOP/go.work" go list -m -f '{{.Dir}}' | head -1)" = "$TOP" ]; then
	ln -s "$TOP" "$WORK/two-link"
	SPELL="$WORK/spell-go"
	printf '#!/usr/bin/env bash\n"%s" "$@" | sed "s#^%s#%s#"\n' \
		"$(command -v go)" "$TOP" "$WORK/two-link" >"$SPELL"
	chmod +x "$SPELL"
fi
check "go's spelling differs from git's" 0 "$(GOWORK="$TOP/go.work" "$SPELL" list -m -f '{{.Dir}}' | grep -cxF "$TOP" || true)"
check "listing under another spelling exits 0" 0 "$(RUN_GO="$SPELL" run "$WORK/two")"
check "listing under another spelling prints the root and sub" ".,sub" "$(sort "$WORK/out" | paste -sd, -)"
check "--check under another spelling passes" 0 "$(RUN_GO="$SPELL" run "$WORK/two" --check)"

echo "go-modules_test: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
