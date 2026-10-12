"""Run a step's gates on another host: the same gates, by the same code.

remote uploads a tar of the tree and a git bundle of HEAD and the tags
with scp, then feeds _bootstrap.py's source to the host's python3 on
stdin through ssh, with argument lists only. What the host prints comes
back line by line, with home paths and the config's identifiers redacted
(docs/decisions/0018-MADR-python-verification-tools.md, item 3).
"""

from __future__ import annotations

import json
import re
import subprocess
import tempfile
from collections.abc import Callable, Iterable, Sequence
from dataclasses import dataclass
from pathlib import Path

from . import tree
from .config import Host

#: A label names the uploads, so it is kept to a file name's characters.
LABEL = re.compile(r"[A-Za-z0-9._-]+")

#: ssh's own lines, which say nothing of the run.
NOISE = re.compile(r"post-quantum|store now, decrypt later|pq\.html|upgraded|Pseudo-terminal", re.IGNORECASE)

#: Home directories, whose last part is an account name.
HOMES = (
    (re.compile(r"/home/[^/\s]+"), "/home/<user>"),
    (re.compile(r"/Users/[^/\s]+"), "/Users/<user>"),
    (re.compile(r"([A-Za-z]:)([\\/]+)Users([\\/]+)[^\\/\s]+", re.IGNORECASE), r"\1\2Users\3<user>"),
)

BOOTSTRAP = Path(__file__).with_name("_bootstrap.py")


class RemoteError(Exception):
    """The run could not be set up."""


@dataclass(frozen=True)
class Transport:
    """The ssh and scp commands, each with its options; tests swap in
    fakes."""

    ssh: tuple[str, ...] = ("ssh", "-o", "BatchMode=yes", "-T")
    scp: tuple[str, ...] = ("scp", "-q", "-o", "BatchMode=yes")


def redact(line: str, identifiers: Iterable[re.Pattern[str]] = ()) -> str | None:
    """line with home paths and identifiers replaced, or None for ssh's
    noise."""
    if NOISE.search(line):
        return None
    for pattern, repl in HOMES:
        line = pattern.sub(repl, line)
    for pattern in identifiers:
        line = pattern.sub("<redacted>", line)
    return line


#: The bootstrap's last lines, which payload replaces with the call.
FOOTER = 'if __name__ == "__main__":\n    sys.exit(main(json.load(sys.stdin)))\n'


def payload(params: dict) -> str:
    """The program the host's python3 reads on stdin: the bootstrap, then
    its call with params."""
    source = BOOTSTRAP.read_text(encoding="utf-8")
    if not source.endswith(FOOTER):
        raise RemoteError(f"{BOOTSTRAP.name} does not end with the call payload replaces")
    return source[: -len(FOOTER)] + f"sys.exit(main(json.loads({json.dumps(params)!r})))\n"


def bundle(repo: Path, dest: Path) -> None:
    """A git bundle of HEAD and every tag, so the copy on the host has the
    history the API diff reads."""
    p = subprocess.run(["git", "bundle", "create", str(dest), "HEAD", "--tags"], cwd=repo, capture_output=True, text=True, check=False)
    if p.returncode != 0:
        raise RemoteError(f"git bundle create: {p.stderr.strip()}")


def run(
    host: Host,
    gate_args: Sequence[str],
    *,
    repo: Path,
    label: str,
    identifiers: Iterable[re.Pattern[str]] = (),
    transport: Transport = Transport(),
    emit: Callable[[str], None] = print,
) -> int:
    """Run the gates on host, with gate_args, and return their exit
    status."""
    if not LABEL.fullmatch(label):
        raise RemoteError(f"label {label!r}: use letters, digits, '.', '_' and '-'")
    identifiers = tuple(identifiers)
    with tempfile.TemporaryDirectory() as tmp:
        work = Path(tmp)
        tgz, bdl = work / f"verify-{label}.tgz", work / f"verify-{label}.bundle"
        tree.tar(repo, tgz)
        bundle(repo, bdl)
        for f in (tgz, bdl):
            p = subprocess.run([*transport.scp, str(f), f"{host.ssh}:{f.name}"], capture_output=True, text=True, check=False)
            if p.returncode != 0:
                emit(f"remote: {host.name}: scp {f.name} failed ({p.returncode})")
                for ln in p.stderr.splitlines():
                    if (r := redact(ln, identifiers)) is not None:
                        emit(f"  | {r}")
                return p.returncode
        params = {"label": label, "tar": tgz.name, "bundle": bdl.name, "gates": list(gate_args), "path_prepend": list(host.path_prepend)}
        with subprocess.Popen(
            [*transport.ssh, host.ssh, host.python, "-"],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
            text=True, encoding="utf-8", errors="replace",
        ) as p:
            assert p.stdin is not None and p.stdout is not None
            p.stdin.write(payload(params))
            p.stdin.close()
            for line in p.stdout:
                if (r := redact(line.rstrip("\r\n"), identifiers)) is not None:
                    emit(r)
            status = p.wait()
    emit(f"remote: {host.name}: exit={status}")
    return status
