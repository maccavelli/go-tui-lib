"""Run one command from an argument list, its output to a file and its exit
status kept before anything reads the output."""

from __future__ import annotations

import os
import shutil
import subprocess
from collections.abc import Mapping, Sequence
from dataclasses import dataclass
from pathlib import Path

#: The status a command that cannot be started is given, as a shell does.
NOT_FOUND = 127


@dataclass(frozen=True)
class Result:
    """A finished command."""

    args: tuple[str, ...]
    status: int
    out: Path

    @property
    def ok(self) -> bool:
        return self.status == 0

    def text(self) -> str:
        """The command's output, stdout and stderr together."""
        return self.out.read_text(encoding="utf-8", errors="replace")


def environ(changes: Mapping[str, str | None], base: Mapping[str, str] = os.environ) -> dict[str, str]:
    """base with changes applied: a value sets a variable, None removes it."""
    env = dict(base)
    for k, v in changes.items():
        if v is None:
            env.pop(k, None)
        else:
            env[k] = v
    return env


def resolve(name: str, env: Mapping[str, str]) -> str:
    """name as a path, found on env's PATH when it is a bare name: on
    Windows a bare name finds only a .exe, and npx, for one, is npx.CMD."""
    if os.path.dirname(name):
        return name
    return shutil.which(name, path=env.get("PATH")) or name


def run(args: Sequence[str], *, cwd: Path, out: Path, env: Mapping[str, str | None] | None = None) -> Result:
    """Run args in cwd, with env's changes to this process's environment,
    writing stdout and stderr to out. A command that cannot be started
    gets NOT_FOUND, and the reason is written to out."""
    out.parent.mkdir(parents=True, exist_ok=True)
    full = environ(env or {})
    argv = [resolve(args[0], full), *args[1:]]
    with out.open("wb") as f:
        try:
            p = subprocess.run(argv, cwd=cwd, env=full, stdout=f, stderr=subprocess.STDOUT, check=False)
        except OSError as err:
            f.write(f"cannot run {args[0]}: {err}\n".encode())
            return Result(tuple(args), NOT_FOUND, out)
    return Result(tuple(args), p.returncode, out)
