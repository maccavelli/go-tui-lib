"""tools.verify.scan: the identifier scan of a change."""

from __future__ import annotations

import contextlib
import io
import json
import os
import re
import unittest

from tools.verify import __main__ as cli
from tools.verify import scan
from tools.verify.tests.support import RepoCase, git

SECRET = "plantedsecret"
PATTERNS = (re.compile("nothing-here"), re.compile(SECRET, re.IGNORECASE))


class TestScan(RepoCase):
    def setUp(self) -> None:
        super().setUp()
        self.write("old.txt", f"committed {SECRET} line\nsecond\n")
        self.write("staged.txt", "a\nb\n")
        self.write("unstaged.txt", "a\nb\nc\n")
        self.commit()

    def run_scan(self, base: str = "HEAD") -> tuple[list[scan.Hit], list[str]]:
        lines: list[str] = []
        return scan.run(self.repo, PATTERNS, base=base, emit=lines.append), lines

    def test_staged_unstaged_and_untracked(self) -> None:
        self.write("staged.txt", f"a\nnew {SECRET.upper()}\nb\n")
        git(self.repo, "add", "staged.txt")
        self.write("unstaged.txt", f"a\nb\nc\nalso {SECRET}\n")
        self.write("dir/untracked.md", f"one\ntwo\n{SECRET} three\n")
        self.write("ignored.log", SECRET)
        self.write(".gitignore", "*.log\n")
        hits, lines = self.run_scan()
        # The change's lines come first, by path, then the untracked files'.
        self.assertEqual([str(h) for h in hits], [
            "staged.txt:2: matches identifier pattern 1",
            "unstaged.txt:4: matches identifier pattern 1",
            "dir/untracked.md:3: matches identifier pattern 1",
        ])
        self.assertEqual(lines[-1], "scan: 3 line(s) in 3 file(s)")
        self.assertFalse(any(SECRET in ln.lower() for ln in lines), "the scan printed what it matched")

    def test_unchanged_lines_are_not_reported(self) -> None:
        hits, lines = self.run_scan()
        self.assertEqual(hits, [])
        self.assertEqual(lines, ["scan: clean (2 pattern(s))"])

    def test_a_base_reaches_back(self) -> None:
        first = git(self.repo, "rev-parse", "HEAD").strip()
        self.write("later.txt", f"{SECRET}\n")
        self.commit()
        self.assertEqual(self.run_scan()[0], [])
        hits, _ = self.run_scan(base=first)
        self.assertEqual([str(h) for h in hits], ["later.txt:1: matches identifier pattern 1"])
        with self.assertRaisesRegex(scan.ScanError, "no-such-ref: no such commit"):
            self.run_scan(base="no-such-ref")

    def test_binary_files_are_skipped(self) -> None:
        (self.repo / "blob.bin").write_bytes(b"\0\1" + SECRET.encode())
        self.assertEqual(self.run_scan()[0], [])


class TestNoCommitYet(RepoCase):
    def test_everything_is_new(self) -> None:
        self.write("a.txt", f"{SECRET}\n")
        git(self.repo, "add", "a.txt")
        self.assertEqual([str(h) for h in scan.run(self.repo, PATTERNS, emit=lambda _: None)], ["a.txt:1: matches identifier pattern 1"])


class TestCommand(RepoCase):
    def test_exit_status_and_config(self) -> None:
        cwd = os.getcwd()
        os.chdir(self.repo)
        self.addCleanup(os.chdir, cwd)
        cfg = self.tmp / "verify.json"
        cfg.write_text(json.dumps({"identifiers": [SECRET]}))
        with contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(cli.main(["scan", "--config", str(cfg)]), 0)
            self.write("x.txt", SECRET)
            self.assertEqual(cli.main(["scan", "--config", str(cfg)]), 1)
        cfg.write_text(json.dumps({"identifiers": []}))
        with contextlib.redirect_stderr(io.StringIO()) as err:
            self.assertEqual(cli.main(["scan", "--config", str(cfg)]), 2)
        self.assertIn("no identifier patterns", err.getvalue())


if __name__ == "__main__":
    unittest.main()
