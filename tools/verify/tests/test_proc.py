"""tools.verify.proc: a command's status and output."""

from __future__ import annotations

import os
import sys
import tempfile
import unittest
from pathlib import Path

from tools.verify import proc
from tools.verify.tests.support import py


class TestRun(unittest.TestCase):
    def setUp(self) -> None:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.dir = Path(tmp.name)

    def test_failing_status_and_output_kept(self) -> None:
        r = proc.run(py("import sys; print('out'); print('err', file=sys.stderr); sys.exit(3)"), cwd=self.dir, out=self.dir / "o/x.out")
        self.assertEqual(r.status, 3)
        self.assertFalse(r.ok)
        self.assertIn("out", r.text())
        self.assertIn("err", r.text())

    def test_missing_command(self) -> None:
        r = proc.run(["no-such-command-for-verify"], cwd=self.dir, out=self.dir / "x.out")
        self.assertEqual(r.status, proc.NOT_FOUND)
        self.assertIn("cannot run no-such-command-for-verify", r.text())

    @unittest.skipIf(os.name == "nt", "a script on PATH needs a shebang")
    def test_found_on_the_commands_own_path(self) -> None:
        bin_dir = self.dir / "bin"
        bin_dir.mkdir()
        tool = bin_dir / "verify-test-tool"
        tool.write_text(f"#!{sys.executable}\nprint('found')\n")
        tool.chmod(0o755)
        r = proc.run(["verify-test-tool"], cwd=self.dir, out=self.dir / "x.out", env={"PATH": f"{bin_dir}{os.pathsep}{os.environ['PATH']}"})
        self.assertEqual((r.status, r.text().strip()), (0, "found"))
        self.assertEqual(proc.resolve("./rel/tool", {"PATH": str(bin_dir)}), "./rel/tool")

    def test_environment_changes(self) -> None:
        os.environ["VERIFY_TEST_DROP"] = "here"
        self.addCleanup(os.environ.pop, "VERIFY_TEST_DROP", None)
        code = "import os; print(os.environ.get('VERIFY_TEST_SET'), os.environ.get('VERIFY_TEST_DROP'))"
        r = proc.run(py(code), cwd=self.dir, out=self.dir / "x.out", env={"VERIFY_TEST_SET": "set", "VERIFY_TEST_DROP": None})
        self.assertEqual(r.text().strip(), "set None")


if __name__ == "__main__":
    unittest.main()
