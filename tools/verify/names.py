"""A record's planned type names, checked against the exported type names
of every public package, before the record is accepted.

internal/conformance fails when two public packages export a type of the
same name unless its sharedNames allows the name for exactly those
packages (AGENTS.md, "API conventions" rule 3). names applies the same
rule to names a record only plans, so a collision is found while the
record can still change.
"""

from __future__ import annotations

import json
import re
from collections.abc import Callable, Mapping
from dataclasses import dataclass
from pathlib import Path

#: The file that holds sharedNames, relative to the repository.
CONFORMANCE = Path("internal/conformance/conformance_test.go")

_TYPE = re.compile(r"^type ([A-Z]\w*)", re.MULTILINE)
_GROUP = re.compile(r"^type \(\n(.*?)^\)", re.MULTILINE | re.DOTALL)
_GROUPED = re.compile(r"^\t([A-Z]\w*)[\s\[]", re.MULTILINE)
_SHARED = re.compile(r"^var sharedNames = map\[string\]\[\]string\{\n(.*?)^\}", re.MULTILINE | re.DOTALL)
_ENTRY = re.compile(r'^\t"(\w+)":\s*\{([^}]*)\}', re.MULTILINE)


class NamesError(Exception):
    """The planned names, or sharedNames, could not be read."""


@dataclass(frozen=True)
class Clash:
    """A planned name another package already exports."""

    package: str
    name: str
    others: tuple[str, ...]

    def __str__(self) -> str:
        return f"{self.package}.{self.name}: also in {', '.join(self.others)}"


def _public(rel: Path) -> bool:
    parts = rel.parts
    return not any(p in ("internal", "testdata", "vendor", "node_modules") or p.startswith((".", "_")) for p in parts)


def exported_types(repo: Path) -> dict[str, set[str]]:
    """Each exported type name of the public packages, with the packages,
    by directory, that declare it. Tests and internal packages are left
    out, as the conformance scan leaves them."""
    out: dict[str, set[str]] = {}
    for f in repo.rglob("*.go"):
        rel = f.relative_to(repo)
        if f.name.endswith("_test.go") or not _public(rel.parent):
            continue
        src = f.read_text(encoding="utf-8", errors="replace")
        names = set(_TYPE.findall(src))
        for group in _GROUP.findall(src):
            names.update(_GROUPED.findall(group))
        pkg = rel.parent.as_posix()
        for n in names:
            out.setdefault(n, set()).add(pkg)
    return out


def shared_names(repo: Path) -> dict[str, frozenset[str]]:
    """internal/conformance's sharedNames: each name and the packages
    allowed to share it, "*" for any."""
    try:
        src = (repo / CONFORMANCE).read_text(encoding="utf-8")
    except OSError as err:
        raise NamesError(f"{CONFORMANCE}: {err}") from None
    m = _SHARED.search(src)
    if not m:
        raise NamesError(f"{CONFORMANCE}: no sharedNames map")
    return {name: frozenset(re.findall(r'"([^"]+)"', pkgs)) for name, pkgs in _ENTRY.findall(m.group(1))}


def check(planned: Mapping[str, list[str]], exported: Mapping[str, set[str]], shared: Mapping[str, frozenset[str]]) -> list[Clash]:
    """The planned names that clash: another package exports the name, and
    sharedNames does not allow it for exactly those packages."""
    clashes = []
    for pkg, names in sorted(planned.items()):
        for n in names:
            others = exported.get(n, set()) - {pkg}
            if not others:
                continue
            allowed = shared.get(n, frozenset())
            if "*" in allowed or allowed == frozenset(others | {pkg}):
                continue
            clashes.append(Clash(pkg, n, tuple(sorted(others))))
    return clashes


def load(path: Path) -> dict[str, list[str]]:
    """A planned-names file: a JSON object of package directory to the
    type names a record plans for it."""
    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as err:
        raise NamesError(f"{path}: {err}") from None
    if not isinstance(raw, dict) or not all(isinstance(v, list) and all(isinstance(n, str) for n in v) for v in raw.values()):
        raise NamesError(f"{path}: want an object of package to a list of type names")
    return raw


def run(repo: Path, planned: Mapping[str, list[str]], *, emit: Callable[[str], None] = print) -> list[Clash]:
    """Check planned against the repository, and report."""
    clashes = check(planned, exported_types(repo), shared_names(repo))
    for c in clashes:
        emit(f"names: {c}")
    total = sum(len(v) for v in planned.values())
    emit(f"names: {len(clashes)} clash(es) in {total} planned name(s)" if clashes else f"names: {total} planned name(s), clean")
    return clashes
