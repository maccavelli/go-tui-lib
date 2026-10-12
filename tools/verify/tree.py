"""The working tree's files: tracked and untracked, not ignored, and
present, so a deleted file that is not staged is never copied
(docs/decisions/0017-MADR-python-repository-scripts.md, item 4)."""

from __future__ import annotations

import shutil
import subprocess
import tarfile
from pathlib import Path


class TreeError(Exception):
    """git could not list the tree."""


def root(start: Path) -> Path:
    """The top of the repository that holds start."""
    p = subprocess.run(["git", "rev-parse", "--show-toplevel"], cwd=start, capture_output=True, text=True, check=False)
    if p.returncode != 0:
        raise TreeError(f"{start}: not in a git work tree: {p.stderr.strip()}")
    return Path(p.stdout.strip())


def files(repo: Path) -> list[str]:
    """The repository's files, relative to repo, sorted: tracked and
    untracked, ignored ones left out, and only those that exist."""
    p = subprocess.run(["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard"], cwd=repo, capture_output=True, check=False)
    if p.returncode != 0:
        raise TreeError(f"{repo}: git ls-files: {p.stderr.decode(errors='replace').strip()}")
    names = {n for n in p.stdout.decode().split("\0") if n}
    return sorted(n for n in names if (repo / n).is_file())


def copy(repo: Path, dest: Path) -> list[str]:
    """Copy the repository's files to dest, keeping their paths and modes,
    and return their names."""
    names = files(repo)
    for n in names:
        target = dest / n
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(repo / n, target)
    return names


def tar(repo: Path, dest: Path) -> list[str]:
    """Write the repository's files to a gzipped tar at dest, and return
    their names."""
    names = files(repo)
    with tarfile.open(dest, "w:gz") as t:
        for n in names:
            t.add(repo / n, arcname=n, recursive=False)
    return names
