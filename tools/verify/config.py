"""The local config: the hosts tools.verify runs on, and the identifier
patterns its scan looks for.

The file is the owner's, outside the repository, and never committed: it
names hosts and accounts that no committed file may
(docs/decisions/0018-MADR-python-verification-tools.md, item 4).
verify.example.json, beside this module, shows its shape with
placeholders.
"""

from __future__ import annotations

import json
import os
import re
from collections.abc import Mapping
from dataclasses import dataclass
from pathlib import Path

#: The kinds of host remote knows how to drive.
KINDS = ("posix", "windows")


class ConfigError(Exception):
    """The config is missing, malformed, or lacks what was asked of it."""


@dataclass(frozen=True)
class Host:
    """A host the gates run on."""

    name: str
    kind: str
    ssh: str
    path_prepend: tuple[str, ...] = ()
    python: str = "python3"


@dataclass(frozen=True)
class Config:
    """The parsed config."""

    path: Path
    hosts: Mapping[str, Host]
    identifiers: tuple[re.Pattern[str], ...]

    def host(self, name: str) -> Host:
        """The host called name, or a ConfigError naming the file."""
        try:
            return self.hosts[name]
        except KeyError:
            known = ", ".join(sorted(self.hosts)) or "none"
            raise ConfigError(f"{self.path}: no host {name!r} (hosts: {known})") from None


def default_path(env: Mapping[str, str] = os.environ) -> Path:
    """$XDG_CONFIG_HOME/go-tui-lib/verify.json, else ~/.config/…."""
    base = env.get("XDG_CONFIG_HOME") or str(Path.home() / ".config")
    return Path(base) / "go-tui-lib" / "verify.json"


def load(path: Path | None = None, env: Mapping[str, str] = os.environ) -> Config:
    """Read and check the config at path, or at default_path(env)."""
    path = path or default_path(env)
    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        raise ConfigError(f"{path}: no config; copy tools/verify/verify.example.json there and fill it in") from None
    except (OSError, json.JSONDecodeError) as err:
        raise ConfigError(f"{path}: {err}") from None
    if not isinstance(raw, dict):
        raise ConfigError(f"{path}: the config is not a JSON object")
    return Config(path=path, hosts=_hosts(path, raw.get("hosts", {})), identifiers=_patterns(path, raw.get("identifiers", [])))


def _hosts(path: Path, raw: object) -> dict[str, Host]:
    if not isinstance(raw, dict):
        raise ConfigError(f"{path}: hosts is not an object")
    hosts = {}
    for name, h in raw.items():
        if not isinstance(h, dict):
            raise ConfigError(f"{path}: host {name!r} is not an object")
        kind, ssh = h.get("kind"), h.get("ssh")
        if kind not in KINDS:
            raise ConfigError(f"{path}: host {name!r} has kind {kind!r}; want one of {', '.join(KINDS)}")
        if not isinstance(ssh, str) or not ssh:
            raise ConfigError(f"{path}: host {name!r} has no ssh destination")
        prepend = h.get("path_prepend", [])
        if not isinstance(prepend, list) or not all(isinstance(p, str) for p in prepend):
            raise ConfigError(f"{path}: host {name!r} path_prepend is not a list of strings")
        python = h.get("python", "python3")
        if not isinstance(python, str) or not python:
            raise ConfigError(f"{path}: host {name!r} python is not a string")
        hosts[name] = Host(name=name, kind=kind, ssh=ssh, path_prepend=tuple(prepend), python=python)
    return hosts


def _patterns(path: Path, raw: object) -> tuple[re.Pattern[str], ...]:
    if not isinstance(raw, list) or not all(isinstance(p, str) for p in raw):
        raise ConfigError(f"{path}: identifiers is not a list of strings")
    out = []
    for i, p in enumerate(raw):
        try:
            out.append(re.compile(p, re.IGNORECASE))
        except re.error as err:
            raise ConfigError(f"{path}: identifier pattern {i} does not compile: {err}") from None
    return tuple(out)
