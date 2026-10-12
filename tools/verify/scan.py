"""The identifier scan: every line a change adds, and every line of an
untracked file, checked against the local config's identifier patterns.

A hit is reported by file, line and the pattern's number, never by the
text that matched, so the scan's own output can be pasted into a record
(AGENTS.md, "Identifiers").
"""

from __future__ import annotations

import re
import subprocess
from collections.abc import Callable, Iterable, Iterator, Sequence
from dataclasses import dataclass
from pathlib import Path

#: git's empty tree, the base of a repository with no commit yet.
EMPTY_TREE = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

#: A hunk header of a zero-context diff: the new side's first line.
HUNK = re.compile(r"^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@")

#: How much of a file is read to tell text from binary.
SNIFF = 8192


class ScanError(Exception):
    """git could not produce the diff to scan."""


@dataclass(frozen=True)
class Hit:
    """A line that matches an identifier pattern."""

    file: str
    line: int
    pattern: int  # its index in the config's list

    def __str__(self) -> str:
        return f"{self.file}:{self.line}: matches identifier pattern {self.pattern}"


def _git(repo: Path, *args: str) -> subprocess.CompletedProcess[bytes]:
    return subprocess.run(["git", *args], cwd=repo, capture_output=True, check=False)


def base_of(repo: Path, base: str) -> str:
    """base, or the empty tree when base is HEAD and there is no commit."""
    if _git(repo, "rev-parse", "--verify", "--quiet", f"{base}^{{commit}}").returncode == 0:
        return base
    if base == "HEAD":
        return EMPTY_TREE
    raise ScanError(f"{base}: no such commit")


def added_lines(repo: Path, base: str = "HEAD") -> Iterator[tuple[str, int, str]]:
    """Each line the work tree adds against base, staged or not, as (file,
    line number, text). Binary files have no lines."""
    p = _git(repo, "-c", "core.quotePath=false", "diff", base_of(repo, base), "--unified=0", "--no-color", "--no-ext-diff", "--no-renames", "--text")
    if p.returncode != 0:
        raise ScanError(f"git diff: {p.stderr.decode(errors='replace').strip()}")
    file, at = "", 0
    for raw in p.stdout.decode("utf-8", errors="replace").splitlines():
        if raw.startswith("+++ "):
            file = raw[6:] if raw.startswith("+++ b/") else ""
        elif m := HUNK.match(raw):
            at = int(m.group(1))
        elif raw.startswith("+") and file:
            yield file, at, raw[1:]
            at += 1


def untracked_lines(repo: Path) -> Iterator[tuple[str, int, str]]:
    """Each line of each untracked, unignored text file."""
    p = _git(repo, "ls-files", "-z", "--others", "--exclude-standard")
    if p.returncode != 0:
        raise ScanError(f"git ls-files: {p.stderr.decode(errors='replace').strip()}")
    for name in filter(None, p.stdout.decode().split("\0")):
        path = repo / name
        if not path.is_file():
            continue
        data = path.read_bytes()
        if b"\0" in data[:SNIFF]:
            continue
        for n, text in enumerate(data.decode("utf-8", errors="replace").splitlines(), 1):
            yield name, n, text


def scan(lines: Iterable[tuple[str, int, str]], patterns: Sequence[re.Pattern[str]]) -> list[Hit]:
    """The lines that match a pattern, each once, with the first pattern it
    matches."""
    hits = []
    for file, n, text in lines:
        for i, p in enumerate(patterns):
            if p.search(text):
                hits.append(Hit(file, n, i))
                break
    return hits


def run(repo: Path, patterns: Sequence[re.Pattern[str]], *, base: str = "HEAD", emit: Callable[[str], None] = print) -> list[Hit]:
    """Scan the change against base and the untracked files, and report."""
    hits = scan([*added_lines(repo, base), *untracked_lines(repo)], patterns)
    for h in hits:
        emit(str(h))
    files = len({h.file for h in hits})
    emit(f"scan: {len(hits)} line(s) in {files} file(s)" if hits else f"scan: clean ({len(patterns)} pattern(s))")
    return hits
