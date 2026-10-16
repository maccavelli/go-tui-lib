"""The gates a PLAN step runs, as data, and their runner.

Each gate is a name, an argument list, the environment changes it needs,
and the lines of its output worth quoting. The runner writes each gate's
whole output to a file, prints one line per gate as it ends,
``name: exit=N (Ns)``, with its quoted lines, goes on after a failure, and
fails if any gate failed
(docs/decisions/0018-MADR-python-verification-tools.md, item 2 and A2).
"""

from __future__ import annotations

import re
import sys
import time
from collections.abc import Callable, Iterable, Mapping, Sequence
from dataclasses import dataclass, field
from pathlib import Path

from . import proc
from .proc import say

#: An argument that becomes each Go file the step names.
GO_FILES = "{go_files}"
#: Text in an argument that becomes the step's Go files, space-separated.
GO_FILES_JOINED = "{go_files_joined}"

#: The lines of a failing gate's output that are printed.
TAIL = 15


@dataclass(frozen=True)
class Gate:
    """One gate."""

    name: str
    args: tuple[str, ...]
    env: Mapping[str, str | None] = field(default_factory=dict)
    #: The gate reads git tags or history, which a bare copy lacks.
    needs_history: bool = False
    #: The gate checks the step's Go files, and is skipped with none.
    needs_go_files: bool = False
    #: Any output is a failure, as gofmt -l's list of files is.
    fail_on_output: bool = False
    #: Patterns of the output lines worth quoting when the gate passes.
    quote: tuple[str, ...] = ()
    #: When it needs history and there is none, the environment it runs
    #: with instead of being skipped; None skips it.
    without_history: Mapping[str, str | None] | None = None
    #: In the default list. A gate that repeats another's checks is not,
    #: and runs only when --only names it (MADR A2).
    default: bool = True

    def argv(self, go_files: Sequence[str]) -> list[str]:
        """The argument list, with the step's Go files put in."""
        out: list[str] = []
        for a in self.args:
            if a == GO_FILES:
                out.extend(go_files)
            else:
                out.append(a.replace(GO_FILES_JOINED, " ".join(go_files)))
        return out


@dataclass(frozen=True)
class Outcome:
    """What one gate did."""

    gate: Gate
    status: int | None  # None when skipped
    reason: str = ""  # why it was skipped
    lines: tuple[str, ...] = ()
    out: Path | None = None
    seconds: float = 0.0

    @property
    def failed(self) -> bool:
        return self.status not in (None, 0)


# The Go gates run with GOWORK=off, as make and CI do; release-check runs
# the tests again in workspace mode.
OFF: Mapping[str, str | None] = {"GOWORK": "off"}


def _cross_vet(goos: str, goarch: str) -> Gate:
    return Gate(f"vet-{goos}-{goarch}", ("go", "vet", "./..."), {"GOWORK": "off", "CGO_ENABLED": "0", "GOOS": goos, "GOARCH": goarch})


def default_gates(repo: Path, python: str = sys.executable) -> list[Gate]:
    """Every gate of a PLAN step, in order: the Go gates make and CI run,
    then CI's other checks, then these tools' own tests. Each check runs
    once (MADR A2): release-check runs the pre-add check over every module
    and file, which is golangci-lint, go vet, go test, tidy, govulncheck,
    workspace mode, the modules check, the API diff and the examples, so
    the gates that repeat it are defined but not in the default list.
    modernize is make lint's go fix -diff, which release-check lacks; vuln
    fails where the pre-add check skips an unreachable database; gofmt
    covers the step's untracked files."""
    gates = [
        Gate("gofmt", ("gofmt", "-l", GO_FILES), needs_go_files=True, fail_on_output=True),
        Gate("modernize", ("make", "modernize")),
        _cross_vet("freebsd", "amd64"),
        _cross_vet("openbsd", "amd64"),
        _cross_vet("linux", "386"),
        Gate("race", ("go", "test", "-race", "-count=1", "./..."), OFF),
        Gate("shuffle", ("go", "test", "-shuffle=on", "-count=2", "./..."), OFF),
        Gate("lc-all-c", ("go", "test", "-count=1", "./..."), {"GOWORK": "off", "LC_ALL": "C"}),
        Gate("vuln", ("make", "vuln"), quote=(r"No vulnerabilities found",)),
        Gate("release-check", ("make", "release-check"), needs_history=True, quote=(r"^go-precheck: .*clean",),
             without_history={"GO_PRECHECK_SKIP_APICHECK": "1"}),
        # The repeats, runnable by name with --only.
        Gate("pre-add-check", ("make", "pre-add-check", f"FILES={GO_FILES_JOINED}"), needs_go_files=True, quote=(r"^go-precheck: .*clean",), default=False),
        Gate("lint", ("make", "lint"), quote=(r"\d+ issues",), default=False),
        Gate("workspace-mode", ("go", "test", "-count=1", "./..."), {"GOWORK": None}, default=False),
        Gate("tidy", ("make", "tidy-check"), default=False),
        Gate("modules", (python, "scripts/go-modules.py", "--check"), default=False),
        Gate("apicheck", ("make", "apicheck"), needs_history=True, quote=(r"^apicheck:",), default=False),
        Gate("examples", ("make", "examples"), quote=(r"^examples:",), default=False),
        Gate("shellcheck", ("shellcheck", *sorted(str(p.relative_to(repo)) for p in (repo / "scripts").glob("*.sh")))),
        Gate("markdownlint", ("npx", "--no-install", "markdownlint-cli2"), quote=(r"^Summary:",)),
        Gate("actionlint", ("actionlint",)),
    ]
    for t in sorted((repo / "scripts").glob("*_test.py")):
        gates.append(Gate(f"script-test-{t.stem.removesuffix('_test')}", (python, str(t.relative_to(repo))), quote=(r"passed, \d+ failed",)))
    gates.append(Gate("tools-tests", (python, "-m", "unittest", "discover", "-s", "tools", "-t", "."), quote=(r"^Ran \d+ tests?",)))
    return gates


class SelectError(Exception):
    """--only or --skip named a gate that does not exist."""


def select(gates: Sequence[Gate], only: Iterable[str] = (), skip: Iterable[str] = ()) -> list[Gate]:
    """The gates named in only, or the default ones when it is empty, less
    those in skip, in their own order."""
    names = {g.name for g in gates}
    only, skip = set(only), set(skip)
    if unknown := (only | skip) - names:
        raise SelectError(f"no gate {', '.join(sorted(unknown))}; gates: {', '.join(g.name for g in gates)}")
    return [g for g in gates if (g.name in only if only else g.default) and g.name not in skip]


def run(
    gates: Sequence[Gate],
    *,
    repo: Path,
    files: Sequence[str],
    out_dir: Path,
    history: bool = True,
    emit: Callable[[str], None] = say,
) -> list[Outcome]:
    """Run gates in repo for the step's files, each output to out_dir, and
    report each as it ends, with its time. A failure never stops the run."""
    go_files = [f for f in files if f.endswith(".go")]
    outcomes = []
    for g in gates:
        env = g.env
        if g.needs_history and not history:
            if g.without_history is None:
                o = Outcome(g, None, "needs git history")
                _report(o, emit)
                outcomes.append(o)
                continue
            env = {**g.env, **g.without_history}
        if g.needs_go_files and not go_files:
            o = Outcome(g, None, "no Go files")
        else:
            o = _run_one(g, env=env, repo=repo, go_files=go_files, out_dir=out_dir)
        _report(o, emit)
        outcomes.append(o)
    passed = sum(o.status == 0 for o in outcomes)
    failed = sum(o.failed for o in outcomes)
    skipped = sum(o.status is None for o in outcomes)
    total = sum(o.seconds for o in outcomes)
    emit(f"gates: {passed} passed, {failed} failed, {skipped} skipped in {total:.0f}s")
    return outcomes


def _run_one(g: Gate, *, env: Mapping[str, str | None], repo: Path, go_files: Sequence[str], out_dir: Path) -> Outcome:
    start = time.monotonic()
    r = proc.run(g.argv(go_files), cwd=repo, out=out_dir / f"{g.name}.out", env=env)
    seconds = time.monotonic() - start
    text = r.text()
    status = r.status
    if status == 0 and g.fail_on_output and text.strip():
        status = 1
    lines = text.splitlines()
    if status != 0:
        quoted = lines[-TAIL:]
    else:
        quoted = [ln for ln in lines if any(re.search(p, ln) for p in g.quote)]
    return Outcome(g, status, lines=tuple(quoted), out=r.out, seconds=seconds)


def _report(o: Outcome, emit: Callable[[str], None]) -> None:
    if o.status is None:
        emit(f"{o.gate.name}: skipped ({o.reason})")
        return
    emit(f"{o.gate.name}: exit={o.status} ({o.seconds:.0f}s)")
    for ln in o.lines:
        emit(f"  | {ln}")
