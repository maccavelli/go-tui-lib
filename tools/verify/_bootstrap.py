"""What remote runs on a host: remote.py sends this module's source to the
host's python3 on stdin, with a call to main appended, so nothing of it is
a shell string (docs/decisions/0018-MADR-python-verification-tools.md,
item 3).

It unpacks the tree remote uploaded into a new temporary directory, makes
a repository there from the uploaded git bundle, with the copied files as
its work tree and index, runs ``python3 -m tools.verify gates`` in it, and
removes the directory, the gates' outputs and the uploads, whatever
happened. It uses only the standard library: it runs before the tree it
unpacks can be imported.
"""

from __future__ import annotations

import json
import os
import shutil
import stat
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path


def _force(func, path, _exc):
    """rmtree's onexc: make a read-only file, such as a git object on
    Windows, writable, and try again."""
    os.chmod(path, stat.S_IWRITE)
    func(path)


def remove(path: Path) -> None:
    """Remove path and everything under it, read-only files included."""
    if path.exists():
        shutil.rmtree(path, onexc=_force)


def _git(work: Path, *args: str) -> None:
    p = subprocess.run(["git", *args], cwd=work, capture_output=True, text=True, check=False)
    if p.returncode != 0:
        raise RuntimeError(f"git {' '.join(args)}: {p.stderr.strip()}")


def prepare(params: dict, home: Path) -> Path:
    """The unpacked tree, in a new directory, with its history: HEAD and
    the tags from the bundle, and an index that matches the copied files.
    The uploads are removed either way."""
    tgz, bundle = home / params["tar"], home / params["bundle"]
    work = Path(tempfile.mkdtemp(prefix=f"verify-{params['label']}-"))
    try:
        with tarfile.open(tgz) as t:
            t.extractall(work, filter="data")
        _git(work, "init", "-q")
        _git(work, "fetch", "-q", str(bundle), "HEAD:refs/heads/verify", "refs/tags/*:refs/tags/*")
        _git(work, "symbolic-ref", "HEAD", "refs/heads/verify")
        _git(work, "reset", "-q")
        _git(work, "add", "-A")
    except BaseException:
        remove(work)
        raise
    finally:
        tgz.unlink(missing_ok=True)
        bundle.unlink(missing_ok=True)
    return work


def environment(params: dict) -> dict[str, str]:
    """This process's environment, with the host's directories first on
    PATH."""
    env = dict(os.environ)
    prepend = [os.path.expanduser(os.path.expandvars(p)) for p in params["path_prepend"]]
    if prepend:
        env["PATH"] = os.pathsep.join([*prepend, env.get("PATH", "")])
    return env


def main(params: dict) -> int:
    """Prepare the tree, run the gates in it, and clean up. The exit
    status is the gates'."""
    work = prepare(params, Path.home())
    out = Path(tempfile.mkdtemp(prefix=f"verify-{params['label']}-out-"))
    try:
        print(f"remote: {params['label']}: python {sys.version.split()[0]}, work in {work}", flush=True)
        args = [sys.executable, "-m", "tools.verify", "gates", "--label", params["label"], "--out", str(out), *params["gates"]]
        return subprocess.run(args, cwd=work, env=environment(params), check=False).returncode
    finally:
        remove(work)
        remove(out)


if __name__ == "__main__":
    sys.exit(main(json.load(sys.stdin)))
