"""Throwaway repositories and commands for the tests."""

from __future__ import annotations

import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


def git(repo: Path, *args: str) -> str:
    """Run git in repo, failing the test on an error, and return stdout."""
    p = subprocess.run(["git", *args], cwd=repo, capture_output=True, text=True, check=False)
    if p.returncode != 0:
        raise AssertionError(f"git {' '.join(args)}: {p.stderr}")
    return p.stdout


class RepoCase(unittest.TestCase):
    """A test with a fresh, empty git repository in self.repo."""

    def setUp(self) -> None:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.tmp = Path(tmp.name)
        self.repo = self.tmp / "repo"
        self.repo.mkdir()
        git(self.repo, "init", "-q")

    def write(self, name: str, text: str = "x\n") -> Path:
        path = self.repo / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text)
        return path

    def commit(self, message: str = "c") -> None:
        """Commit everything, with no hooks, signing or real identity: the
        machine's hooks are not this repository's to run."""
        hooks = self.tmp / "no-hooks"
        hooks.mkdir(exist_ok=True)
        git(self.repo, "add", "-A")
        git(self.repo, "-c", f"core.hooksPath={hooks}", "-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "-m", message)


def py(code: str) -> tuple[str, ...]:
    """The argument list that runs code in this Python."""
    return (sys.executable, "-c", code)
