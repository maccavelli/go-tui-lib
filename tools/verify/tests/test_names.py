"""tools.verify.names: planned type names against the public packages'."""

from __future__ import annotations

import contextlib
import io
import json
import os
import unittest
from pathlib import Path

from tools.verify import __main__ as cli
from tools.verify import names
from tools.verify.tests.support import RepoCase

CONFORMANCE = """package conformance

var sharedNames = map[string][]string{
\t"Option":  {"*"},
\t"Pane":    {"layout", "workspace"},
\t// a comment
\t"Terminal": {"launch/launchtest", "termcap/termcaptest"},
}
"""


class NamesCase(RepoCase):
    def setUp(self) -> None:
        super().setUp()
        self.write("internal/conformance/conformance_test.go", CONFORMANCE)
        self.write("a/a.go", "package a\n\ntype Foo struct{}\n\ntype Ranked[T any] struct{ v T }\n\ntype lower int\n")
        self.write("b/b.go", "package b\n\ntype (\n\tBar int\n\tBaz[T any] []T\n\tquiet int\n)\n")
        self.write("layout/l.go", "package layout\n\ntype Pane struct{}\n")
        self.write("launch/launchtest/t.go", "package launchtest\n\ntype Terminal struct{}\n")
        self.write("internal/z/z.go", "package z\n\ntype Hidden int\n")
        self.write("a/a_test.go", "package a\n\ntype TestOnly int\n")
        self.write("testdata/x/x.go", "package x\n\ntype Fixture int\n")


class TestExported(NamesCase):
    def test_public_exported_types(self) -> None:
        got = names.exported_types(self.repo)
        self.assertEqual(got["Foo"], {"a"})
        self.assertEqual(got["Ranked"], {"a"})
        self.assertEqual(got["Bar"], {"b"})
        self.assertEqual(got["Baz"], {"b"})
        for absent in ("lower", "quiet", "Hidden", "TestOnly", "Fixture"):
            self.assertNotIn(absent, got)

    def test_shared_names(self) -> None:
        self.assertEqual(names.shared_names(self.repo), {
            "Option": frozenset({"*"}),
            "Pane": frozenset({"layout", "workspace"}),
            "Terminal": frozenset({"launch/launchtest", "termcap/termcaptest"}),
        })


class TestCheck(NamesCase):
    def test_clashes_and_allowances(self) -> None:
        planned = {
            "palette": ["Foo", "Option", "Qux", "Hidden", "TestOnly", "Pane"],
            "workspace": ["Pane"],
            "termcap/termcaptest": ["Terminal"],
            "a": ["Foo"],
        }
        lines: list[str] = []
        clashes = names.run(self.repo, planned, emit=lines.append)
        self.assertEqual([str(c) for c in clashes], ["palette.Foo: also in a", "palette.Pane: also in layout"])
        self.assertEqual(lines[-1], "names: 2 clash(es) in 9 planned name(s)")

    def test_clean(self) -> None:
        lines: list[str] = []
        self.assertEqual(names.run(self.repo, {"palette": ["Picker", "Hit"]}, emit=lines.append), [])
        self.assertEqual(lines, ["names: 2 planned name(s), clean"])


class TestErrors(RepoCase):
    def test_no_conformance_file(self) -> None:
        with self.assertRaisesRegex(names.NamesError, "conformance_test.go"):
            names.shared_names(self.repo)

    def test_bad_planned_file(self) -> None:
        bad = self.tmp / "planned.json"
        bad.write_text(json.dumps({"palette": "Picker"}))
        with self.assertRaisesRegex(names.NamesError, "want an object"):
            names.load(bad)


class TestCommand(NamesCase):
    def test_exit_status(self) -> None:
        cwd = os.getcwd()
        os.chdir(self.repo)
        self.addCleanup(os.chdir, cwd)
        planned = self.tmp / "planned.json"
        for data, want in (({"palette": ["Picker"]}, 0), ({"palette": ["Foo"]}, 1)):
            planned.write_text(json.dumps(data))
            with contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(cli.main(["names", str(planned)]), want)
        with contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(cli.main(["names", str(self.tmp / "absent.json")]), 2)


class TestTheRepository(unittest.TestCase):
    def test_the_repositorys_shared_names_parse(self) -> None:
        repo = Path(__file__).resolve().parents[3]
        shared = names.shared_names(repo)
        self.assertEqual(shared["Option"], frozenset({"*"}))
        self.assertIn("Terminal", shared)


if __name__ == "__main__":
    unittest.main()
