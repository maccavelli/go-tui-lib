"""tools.verify.remote and its bootstrap, through fake ssh and scp."""

from __future__ import annotations

import contextlib
import io
import json
import os
import re
import shutil
import sys
import tempfile
import unittest
from pathlib import Path

from tools.verify import __main__ as cli
from tools.verify import _bootstrap, remote, tree
from tools.verify.config import Host
from tools.verify.tests.support import RepoCase, git

TOOLS = Path(__file__).resolve().parents[2]
FAKES = Path(__file__).resolve().parent / "fakes"
TRANSPORT = remote.Transport(
    ssh=(sys.executable, str(FAKES / "ssh.py"), "-o", "BatchMode=yes", "-T"),
    scp=(sys.executable, str(FAKES / "scp.py"), "-q", "-o", "BatchMode=yes"),
)
HOST = Host(name="far", kind="posix", ssh="far-host", path_prepend=("$VERIFY_FAKE_HOME/bin",))


class RemoteCase(RepoCase):
    """A committed, tagged repository holding tools/, an untracked file,
    and a fake host's home, temporary directory and log."""

    def setUp(self) -> None:
        super().setUp()
        shutil.copytree(TOOLS, self.repo / "tools", ignore=shutil.ignore_patterns("__pycache__"))
        self.write("README", "r\n")
        self.commit("init")
        git(self.repo, "tag", "v0.0.1")
        self.write("new.txt", "untracked\n")
        self.home, self.ftmp, self.log = self.tmp / "home", self.tmp / "ftmp", self.tmp / "log.jsonl"
        self.home.mkdir()
        self.ftmp.mkdir()
        for k, v in {"VERIFY_FAKE_HOME": self.home, "VERIFY_FAKE_TMP": self.ftmp, "VERIFY_FAKE_LOG": self.log}.items():
            os.environ[k] = str(v)
            self.addCleanup(os.environ.pop, k, None)

    def remote(self, *gate_args: str, label: str = "lab") -> tuple[int, list[str]]:
        lines: list[str] = []
        status = remote.run(HOST, gate_args, repo=self.repo, label=label, transport=TRANSPORT, emit=lines.append)
        return status, lines

    def calls(self) -> list[dict[str, object]]:
        return [json.loads(ln) for ln in self.log.read_text().splitlines()]


class TestRoundTrip(RemoteCase):
    def test_gates_run_on_the_host(self) -> None:
        status, lines = self.remote("--list")
        self.assertEqual(status, 0, lines)
        self.assertTrue(any(ln.startswith("remote: lab: python ") for ln in lines), lines)
        self.assertIn("gofmt", lines)
        self.assertIn("tools-tests", lines)
        self.assertEqual(lines[-1], "remote: far: exit=0")

    def test_nothing_is_left_on_the_host(self) -> None:
        for args, want in ((("--list",), 0), (("--only", "nope"), 2)):
            with self.subTest(args=args):
                status, lines = self.remote(*args)
                self.assertEqual(status, want, lines)
                self.assertEqual(list(self.home.iterdir()), [], "uploads left in the home")
                self.assertEqual(list(self.ftmp.iterdir()), [], "a work or output directory left")

    def test_only_argument_lists(self) -> None:
        self.remote("--list")
        calls = self.calls()
        scps = [c["args"] for c in calls if c["tool"] == "scp"]
        self.assertEqual([a[-1] for a in scps], ["far-host:verify-lab.tgz", "far-host:verify-lab.bundle"])
        (ssh,) = [c["args"] for c in calls if c["tool"] == "ssh"]
        self.assertEqual(ssh, ["-o", "BatchMode=yes", "-T", "far-host", "python3", "-"])

    def test_a_bad_label(self) -> None:
        with self.assertRaisesRegex(remote.RemoteError, "label 'a b'"):
            self.remote("--list", label="a b")


class TestBootstrap(RemoteCase):
    def upload(self) -> dict[str, object]:
        tree.tar(self.repo, self.home / "t.tgz")
        remote.bundle(self.repo, self.home / "t.bundle")
        return {"label": "lab", "tar": "t.tgz", "bundle": "t.bundle", "gates": [], "path_prepend": []}

    def test_the_copy_has_history_and_an_index(self) -> None:
        work = _bootstrap.prepare(self.upload(), self.home)
        self.addCleanup(_bootstrap.remove, work)
        self.assertEqual(git(work, "rev-parse", "HEAD"), git(self.repo, "rev-parse", "HEAD"))
        self.assertIn("v0.0.1", git(work, "tag").split())
        self.assertEqual(git(work, "ls-files", "new.txt").strip(), "new.txt")
        self.assertEqual(list(self.home.iterdir()), [])

    def test_a_failed_prepare_removes_its_directory(self) -> None:
        params = self.upload()
        (self.home / "t.bundle").write_text("not a bundle")
        old = tempfile.tempdir
        tempfile.tempdir = str(self.ftmp)
        self.addCleanup(setattr, tempfile, "tempdir", old)
        with self.assertRaisesRegex(RuntimeError, "git fetch"):
            _bootstrap.prepare(params, self.home)
        self.assertEqual(list(self.ftmp.iterdir()), [])
        self.assertEqual(list(self.home.iterdir()), [])

    def test_path_prepend(self) -> None:
        env = _bootstrap.environment({"path_prepend": ["$VERIFY_FAKE_HOME/bin"]})
        self.assertTrue(env["PATH"].startswith(f"{self.home}/bin{os.pathsep}"))

    def test_payload_calls_main_with_the_params(self) -> None:
        source = remote.payload({"label": "x", "tar": "a", "bundle": "b", "gates": ["--list"], "path_prepend": []})
        self.assertNotIn("json.load(sys.stdin)", source)
        self.assertTrue(source.endswith("sys.exit(main(json.loads('{\"label\": \"x\", \"tar\": \"a\", \"bundle\": \"b\", \"gates\": [\"--list\"], \"path_prepend\": []}')))\n"))
        compile(source, "payload", "exec")


class TestRedact(unittest.TestCase):
    def test_homes_identifiers_and_noise(self) -> None:
        ids = (re.compile("secret-host", re.IGNORECASE),)
        for line, want in [
            ("work in /home/alice/tmp/x", "work in /home/<user>/tmp/x"),
            ("at /Users/bob/go", "at /Users/<user>/go"),
            (r"in C:\Users\carol\AppData\Local\Temp", r"in C:\Users\<user>\AppData\Local\Temp"),
            ("in c:/Users/dave/x", "in c:/Users/<user>/x"),
            ("ran on Secret-Host today", "ran on <redacted> today"),
            ("** WARNING: connection is not using a post-quantum key exchange algorithm.", None),
            ("Pseudo-terminal will not be allocated because stdin is not a terminal.", None),
            ("gofmt: exit=0", "gofmt: exit=0"),
        ]:
            with self.subTest(line=line):
                self.assertEqual(remote.redact(line, ids), want)


class TestCommand(RepoCase):
    def test_config_errors(self) -> None:
        cwd = os.getcwd()
        os.chdir(self.repo)
        self.addCleanup(os.chdir, cwd)
        cfg = self.tmp / "verify.json"
        cfg.write_text(json.dumps({"hosts": {"lin": {"kind": "posix", "ssh": "d"}}}))
        for argv, want in [
            (["remote", "win", "--config", str(cfg), "--", "--list"], "no host 'win' (hosts: lin)"),
            (["remote", "lin", "--config", str(self.tmp / "absent.json")], "absent.json: no config"),
        ]:
            with self.subTest(argv=argv), contextlib.redirect_stderr(io.StringIO()) as err:
                self.assertEqual(cli.main(argv), 2)
                self.assertIn(want, err.getvalue())


if __name__ == "__main__":
    unittest.main()
