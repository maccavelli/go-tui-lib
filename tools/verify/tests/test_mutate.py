"""tools.verify.mutate: each mutation on its own copy, its test must fail."""

from __future__ import annotations

import contextlib
import io
import json
import os
import shutil
import unittest

from tools.verify import __main__ as cli
from tools.verify import mutate
from tools.verify.tests.support import RepoCase, py

# check.py fails unless calc.py's add adds.
CHECK = "import sys; sys.path.insert(0, '.'); import calc; sys.exit(0 if calc.add(2, 3) == 5 else 1)"


class TestRun(RepoCase):
    def setUp(self) -> None:
        super().setUp()
        self.write("calc.py", "def add(a, b):\n    # adds\n    return a + b\n")
        self.write("twice.py", "x = 1\nx = 1\n")

    def mutation(self, name: str, file: str, old: str, new: str, **more: object) -> dict[str, object]:
        return {"name": name, "file": file, "old": old, "new": new, "cmd": list(py(CHECK)), **more}

    def run_all(self, raw: list[dict[str, object]]) -> tuple[list[mutate.Outcome], list[str]]:
        lines: list[str] = []
        out = mutate.run(mutate.parse(raw), repo=self.repo, work=self.tmp / "work", emit=lines.append)
        return out, lines

    def test_killed_survived_and_errors(self) -> None:
        outcomes, lines = self.run_all([
            self.mutation("breaks add", "calc.py", "a + b", "a - b"),
            self.mutation("touches a comment", "calc.py", "# adds", "# sums"),
            self.mutation("anchor twice", "twice.py", "x = 1", "x = 2"),
            self.mutation("anchor absent", "calc.py", "a * b", "a"),
        ])
        self.assertEqual([o.verdict for o in outcomes], [mutate.KILLED, mutate.SURVIVED, mutate.ERROR, mutate.ERROR])
        self.assertIn("## breaks add: KILLED (exit 1)", lines)
        self.assertIn("## touches a comment: SURVIVED (exit 0)", lines)
        self.assertIn("    anchor twice: the anchor is found 2 times in twice.py; want once", lines)
        self.assertIn("    anchor absent: the anchor is found 0 times in calc.py; want once", lines)
        self.assertEqual(lines[-1], "mutate: 1 killed, 1 survived, 2 errors")

    def test_the_repository_is_untouched_and_copies_removed(self) -> None:
        self.run_all([self.mutation("breaks add", "calc.py", "a + b", "a - b")])
        self.assertIn("a + b", (self.repo / "calc.py").read_text())
        self.assertFalse((self.tmp / "work" / "mut0").exists())

    def test_extra_edits(self) -> None:
        self.write("other.py", "N = 0\n")
        outcomes, _ = self.run_all([{
            "name": "two files", "file": "calc.py", "old": "a + b", "new": "a + b + other.N",
            "extra": [{"file": "calc.py", "old": "def add", "new": "import other\ndef add"}, {"file": "other.py", "old": "N = 0", "new": "N = 1"}],
            "cmd": list(py(CHECK)),
        }])
        self.assertEqual(outcomes[0].verdict, mutate.KILLED)

    def test_env_reaches_the_test(self) -> None:
        # The test passes only when MUT reaches it, so a neutral mutation
        # survives with env and is killed, by a KeyError, without it.
        check = "import os, sys; sys.exit(0 if os.environ['MUT'] == 'on' else 1)"
        outcomes, _ = self.run_all([{"name": "env", "file": "calc.py", "old": "# adds", "new": "# sums", "cmd": list(py(check)), "env": {"MUT": "on"}}])
        self.assertEqual(outcomes[0].verdict, mutate.SURVIVED)

    @unittest.skipUnless(shutil.which("go"), "needs go")
    def test_a_go_test(self) -> None:
        self.write("go.mod", "module example.com/m\n\ngo 1.24\n")
        self.write("add.go", "package m\n\nfunc Add(a, b int) int { return a + b }\n")
        self.write("add_test.go", "package m\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(2, 3) != 5 {\n\t\tt.Fatal(\"Add(2, 3) is not 5\")\n\t}\n}\n")
        lines: list[str] = []
        raw = [{"name": "go breaks", "file": "add.go", "old": "a + b", "new": "a - b", "test": ["./..."], "env": {"GOWORK": "off", "GOFLAGS": "-mod=mod"}}]
        outcomes = mutate.run(mutate.parse(raw), repo=self.repo, work=self.tmp / "work", emit=lines.append)
        self.assertEqual(outcomes[0].verdict, mutate.KILLED)
        self.assertTrue(any("Add(2, 3) is not 5" in ln for ln in lines), lines)


class TestParse(unittest.TestCase):
    def test_errors(self) -> None:
        good = {"name": "n", "file": "f", "old": "o", "new": "x"}
        for raw, want in [
            ({}, "a mutation file is a JSON list"),
            ([{"file": "f"}], "mutation 0: no name"),
            ([good], "give test or cmd"),
            ([{**good, "test": ["./..."], "cmd": ["x"]}], "give test or cmd"),
            ([{**good, "cmd": []}], "not a list of strings"),
            ([{**good, "old": "", "cmd": ["x"]}], "empty anchor"),
            ([{**good, "cmd": ["x"], "env": {"A": 1}}], "env is not"),
            ([{**good, "cmd": ["x"], "extra": [{"file": "g"}]}], "file, old and new"),
        ]:
            with self.subTest(raw=raw), self.assertRaisesRegex(mutate.MutationError, want):
                mutate.parse(raw)

    def test_go_test_arguments(self) -> None:
        m = mutate.parse([{"name": "n", "file": "f", "old": "o", "new": "x", "test": ["-run", "TestX", "./pkg/"]}])[0]
        self.assertEqual(m.args, ("go", "test", "-count=1", "-run", "TestX", "./pkg/"))


class TestCommand(RepoCase):
    def test_exit_status(self) -> None:
        self.write("calc.py", "def add(a, b):\n    return a + b\n")
        cwd = os.getcwd()
        os.chdir(self.repo)
        self.addCleanup(os.chdir, cwd)
        for raw, want in [
            ([{"name": "k", "file": "calc.py", "old": "a + b", "new": "a - b", "cmd": list(py(CHECK))}], 0),
            ([{"name": "s", "file": "calc.py", "old": "a + b", "new": "b + a", "cmd": list(py(CHECK))}], 1),
        ]:
            spec = self.tmp / "m.json"
            spec.write_text(json.dumps(raw))
            with contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(cli.main(["mutate", str(spec), "--work", str(self.tmp / "work")]), want)
        spec.write_text("{")
        with contextlib.redirect_stderr(io.StringIO()) as err:
            self.assertEqual(cli.main(["mutate", str(spec)]), 2)
        self.assertIn("mutate:", err.getvalue())


if __name__ == "__main__":
    unittest.main()
