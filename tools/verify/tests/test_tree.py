"""tools.verify.tree: which files a copy of the tree holds."""

from __future__ import annotations

import os
import stat
import tarfile
import unittest

from tools.verify import tree
from tools.verify.tests.support import RepoCase, git


class TestFiles(RepoCase):
    def test_tracked_untracked_ignored_and_deleted(self) -> None:
        self.write(".gitignore", "*.log\n")
        self.write("kept.go")
        self.write("gone.go")
        self.write("sub/new.py")
        self.write("noise.log")
        git(self.repo, "add", ".gitignore", "kept.go", "gone.go")
        (self.repo / "gone.go").unlink()  # deleted, and not staged
        self.assertEqual(tree.files(self.repo), [".gitignore", "kept.go", "sub/new.py"])

    def test_root(self) -> None:
        (self.repo / "a" / "b").mkdir(parents=True)
        self.assertEqual(tree.root(self.repo / "a" / "b").resolve(), self.repo.resolve())
        outside = self.tmp / "outside"
        outside.mkdir()
        with self.assertRaises(tree.TreeError):
            tree.root(outside)


class TestCopyAndTar(RepoCase):
    def test_copy_keeps_paths_and_modes(self) -> None:
        script = self.write("scripts/run.py", "print(1)\n")
        os.chmod(script, 0o755)
        self.write("README")
        dest = self.tmp / "copy"
        self.assertEqual(tree.copy(self.repo, dest), ["README", "scripts/run.py"])
        self.assertEqual((dest / "scripts/run.py").read_text(), "print(1)\n")
        # The copy's permission bits are the source's: on POSIX that keeps
        # the executable bit set above; Windows has only read-only.
        self.assertEqual(stat.S_IMODE((dest / "scripts/run.py").stat().st_mode), stat.S_IMODE(script.stat().st_mode))

    def test_tar_holds_the_files(self) -> None:
        self.write("a.go")
        self.write("dir/b.md")
        out = self.tmp / "tree.tgz"
        tree.tar(self.repo, out)
        with tarfile.open(out) as t:
            self.assertEqual(sorted(t.getnames()), ["a.go", "dir/b.md"])


if __name__ == "__main__":
    unittest.main()
