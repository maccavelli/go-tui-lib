"""tools.verify.gates and the gates command: the runner and the list."""

from __future__ import annotations

import contextlib
import io
import os
import unittest
from pathlib import Path

from tools.verify import __main__ as cli
from tools.verify import gates
from tools.verify.tests.support import RepoCase, py

OK = py("print('fine'); print('Summary: 0 issues')")
FAIL = py("import sys; print('broke at line 9'); sys.exit(3)")


class TestRunner(RepoCase):
    def run_gates(self, gate_list: list[gates.Gate], files: list[str] | None = None, history: bool = True) -> tuple[list[gates.Outcome], list[str]]:
        lines: list[str] = []
        out = gates.run(gate_list, repo=self.repo, files=files or [], out_dir=self.tmp / "out", history=history, emit=lines.append)
        return out, lines

    def test_a_failure_does_not_stop_the_run(self) -> None:
        outcomes, lines = self.run_gates([gates.Gate("first", OK), gates.Gate("broken", FAIL), gates.Gate("last", OK)])
        self.assertEqual([o.status for o in outcomes], [0, 3, 0])
        self.assertIn("broken: exit=3", lines)
        self.assertIn("  | broke at line 9", lines)
        self.assertIn("last: exit=0", lines)
        self.assertEqual(lines[-1], "gates: 2 passed, 1 failed, 0 skipped")
        self.assertTrue((self.tmp / "out" / "broken.out").is_file())

    def test_history_gates_are_skipped_without_history(self) -> None:
        g = gates.Gate("tags", OK, needs_history=True)
        outcomes, lines = self.run_gates([g], history=False)
        self.assertIsNone(outcomes[0].status)
        self.assertIn("tags: skipped (needs git history)", lines)
        outcomes, _ = self.run_gates([g], history=True)
        self.assertEqual(outcomes[0].status, 0)

    def test_go_file_gates_need_go_files(self) -> None:
        g = gates.Gate("fmt", py("import sys; print(sys.argv[1:])") + (gates.GO_FILES,), needs_go_files=True)
        _, lines = self.run_gates([g], files=["README.md"])
        self.assertIn("fmt: skipped (no Go files)", lines)
        outcomes, _ = self.run_gates([g], files=["a.go", "README.md", "b_test.go"])
        self.assertIn("['a.go', 'b_test.go']", outcomes[0].out.read_text())

    def test_output_fails_a_gate_that_wants_none(self) -> None:
        outcomes, lines = self.run_gates([gates.Gate("gofmt", py("print('a.go')"), fail_on_output=True)])
        self.assertEqual(outcomes[0].status, 1)
        self.assertIn("  | a.go", lines)

    def test_quoted_lines_of_a_passing_gate(self) -> None:
        _, lines = self.run_gates([gates.Gate("lint", OK, quote=(r"^Summary:",))])
        self.assertEqual(lines[:2], ["lint: exit=0", "  | Summary: 0 issues"])


class TestGate(unittest.TestCase):
    def test_argv_puts_in_the_go_files(self) -> None:
        g = gates.Gate("g", ("make", "pre-add-check", f"FILES={gates.GO_FILES_JOINED}", gates.GO_FILES))
        self.assertEqual(g.argv(["a.go", "b.go"]), ["make", "pre-add-check", "FILES=a.go b.go", "a.go", "b.go"])

    def test_select(self) -> None:
        gs = [gates.Gate(n, OK) for n in ("a", "b", "c")]
        self.assertEqual([g.name for g in gates.select(gs)], ["a", "b", "c"])
        self.assertEqual([g.name for g in gates.select(gs, only=["c", "a"])], ["a", "c"])
        self.assertEqual([g.name for g in gates.select(gs, skip=["b"])], ["a", "c"])
        with self.assertRaisesRegex(gates.SelectError, "no gate z"):
            gates.select(gs, only=["z"])


class TestDefaultGates(unittest.TestCase):
    def setUp(self) -> None:
        self.gates = {g.name: g for g in gates.default_gates(Path(__file__).resolve().parents[3])}

    def test_history(self) -> None:
        needing = {n for n, g in self.gates.items() if g.needs_history}
        self.assertEqual(needing, {"apicheck", "release-check"})

    def test_go_files(self) -> None:
        self.assertEqual({n for n, g in self.gates.items() if g.needs_go_files}, {"gofmt", "pre-add-check"})

    def test_the_step_gates_are_all_there(self) -> None:
        for n in ("lint", "vet-linux-386", "race", "shuffle", "lc-all-c", "workspace-mode", "tidy", "vuln", "modules", "examples", "markdownlint", "actionlint", "tools-tests", "script-test-go-precheck"):
            self.assertIn(n, self.gates)
        self.assertEqual(self.gates["workspace-mode"].env, {"GOWORK": None})
        self.assertEqual(self.gates["race"].env, {"GOWORK": "off"})

    def test_no_gate_runs_a_shell(self) -> None:
        for g in self.gates.values():
            self.assertNotIn(os.path.basename(g.args[0]), {"sh", "bash", "zsh", "cmd", "powershell"}, g.name)


class TestCommand(RepoCase):
    def command(self, gate_list: list[gates.Gate], *argv: str) -> tuple[int, str, str]:
        args = cli.parser().parse_args(["gates", "--out", str(self.tmp / "out"), *argv])
        cwd = os.getcwd()
        os.chdir(self.repo)
        self.addCleanup(os.chdir, cwd)
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            status = cli.cmd_gates(args, gate_list=lambda _repo: gate_list)
        return status, out.getvalue(), err.getvalue()

    def test_exit_status(self) -> None:
        self.assertEqual(self.command([gates.Gate("a", OK)])[0], 0)
        status, out, _ = self.command([gates.Gate("a", OK), gates.Gate("b", FAIL)])
        self.assertEqual(status, 1)
        self.assertIn("b: exit=3", out)

    def test_only_skip_and_unknown(self) -> None:
        gs = [gates.Gate("a", OK), gates.Gate("b", FAIL)]
        status, out, _ = self.command(gs, "--only", "a")
        self.assertEqual(status, 0)
        self.assertNotIn("b:", out)
        self.assertEqual(self.command(gs, "--skip", "b")[0], 0)
        status, _, err = self.command(gs, "--only", "nope")
        self.assertEqual(status, 2)
        self.assertIn("no gate nope", err)


if __name__ == "__main__":
    unittest.main()
