"""Mutation proofs: each mutation is applied to a fresh copy of the tree,
and the test it names must fail there.

A mutation file is a JSON list. Each mutation has a ``name``, a ``file``
relative to the repository, an ``old`` anchor that must occur in it
exactly once, and its ``new`` text; ``extra`` edits more files the same
way. The test is ``test``, the arguments that follow ``go test -count=1``,
or ``cmd``, a whole argument list, such as Python's unittest; ``env`` adds
to its environment.
"""

from __future__ import annotations

import json
import re
import shutil
from collections.abc import Callable, Mapping, Sequence
from dataclasses import dataclass, field
from pathlib import Path

from . import proc, tree

KILLED, SURVIVED, ERROR = "KILLED", "SURVIVED", "ERROR"

#: The lines of a test's output that say why it failed.
SIGNAL = re.compile(r"FAIL|panic|Error|want|--- |cannot|undefined|_test\.go:|stack overflow")
#: How many of them each mutation prints.
SHOWN = 8


class MutationError(Exception):
    """A mutation file, or one mutation, is malformed or does not apply."""


@dataclass(frozen=True)
class Edit:
    """One replacement in one file."""

    file: str
    old: str
    new: str


@dataclass(frozen=True)
class Mutation:
    """A named set of edits and the test that must fail with them."""

    name: str
    edits: tuple[Edit, ...]
    args: tuple[str, ...]
    env: Mapping[str, str] = field(default_factory=dict)


@dataclass(frozen=True)
class Outcome:
    """What one mutation did."""

    mutation: Mutation
    verdict: str
    status: int | None = None
    lines: tuple[str, ...] = ()


def _edit(name: str, raw: object) -> Edit:
    if not isinstance(raw, dict) or not all(isinstance(raw.get(k), str) for k in ("file", "old", "new")):
        raise MutationError(f"{name}: an edit needs file, old and new strings")
    if not raw["old"]:
        raise MutationError(f"{name}: an empty anchor in {raw['file']}")
    return Edit(raw["file"], raw["old"], raw["new"])


def parse(raw: object) -> list[Mutation]:
    """The mutations of a decoded mutation file."""
    if not isinstance(raw, list):
        raise MutationError("a mutation file is a JSON list")
    out = []
    for i, m in enumerate(raw):
        if not isinstance(m, dict) or not isinstance(m.get("name"), str):
            raise MutationError(f"mutation {i}: no name")
        name = m["name"]
        if ("test" in m) == ("cmd" in m):
            raise MutationError(f"{name}: give test or cmd, not both or neither")
        given = m.get("cmd", m.get("test"))
        if not isinstance(given, list) or not given or not all(isinstance(a, str) for a in given):
            raise MutationError(f"{name}: test or cmd is not a list of strings")
        args = given if "cmd" in m else ["go", "test", "-count=1", *given]
        env = m.get("env", {})
        if not isinstance(env, dict) or not all(isinstance(v, str) for v in env.values()):
            raise MutationError(f"{name}: env is not an object of strings")
        extra = m.get("extra", [])
        if not isinstance(extra, list):
            raise MutationError(f"{name}: extra is not a list")
        edits = (_edit(name, m), *(_edit(name, e) for e in extra))
        out.append(Mutation(name, edits, tuple(args), env))
    return out


def load(path: Path) -> list[Mutation]:
    """The mutations of the file at path."""
    try:
        return parse(json.loads(path.read_text(encoding="utf-8")))
    except (OSError, json.JSONDecodeError) as err:
        raise MutationError(f"{path}: {err}") from None


def apply(m: Mutation, root: Path) -> None:
    """Make m's edits under root, each anchor found exactly once first."""
    for e in m.edits:
        target = root / e.file
        try:
            text = target.read_text(encoding="utf-8")
        except OSError as err:
            raise MutationError(f"{m.name}: {e.file}: {err}") from None
        if (n := text.count(e.old)) != 1:
            raise MutationError(f"{m.name}: the anchor is found {n} times in {e.file}; want once")
        target.write_text(text.replace(e.old, e.new), encoding="utf-8")


def run(mutations: Sequence[Mutation], *, repo: Path, work: Path, emit: Callable[[str], None] = proc.say, keep: bool = False) -> list[Outcome]:
    """Run each mutation in its own copy of repo under work, and report it.
    A copy is removed after its test, unless keep."""
    outcomes = []
    for i, m in enumerate(mutations):
        dst = work / f"mut{i}"
        if dst.exists():
            shutil.rmtree(dst)
        tree.copy(repo, dst)
        try:
            apply(m, dst)
        except MutationError as err:
            o = Outcome(m, ERROR, lines=(str(err),))
        else:
            r = proc.run(m.args, cwd=dst, out=work / f"mut{i}.out", env=m.env)
            failing = [ln for ln in r.text().splitlines() if SIGNAL.search(ln)]
            o = Outcome(m, KILLED if r.status != 0 else SURVIVED, r.status, tuple(failing[:SHOWN]))
        if not keep:
            shutil.rmtree(dst)
        _report(o, emit)
        outcomes.append(o)
    counts = {v: sum(o.verdict == v for o in outcomes) for v in (KILLED, SURVIVED, ERROR)}
    emit(f"mutate: {counts[KILLED]} killed, {counts[SURVIVED]} survived, {counts[ERROR]} errors")
    return outcomes


def _report(o: Outcome, emit: Callable[[str], None]) -> None:
    status = "" if o.status is None else f" (exit {o.status})"
    emit(f"## {o.mutation.name}: {o.verdict}{status}")
    for ln in o.lines:
        emit(f"    {ln[:220]}")
