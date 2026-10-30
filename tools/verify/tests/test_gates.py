"""tools.verify.gates and the gates command: the runner and the list."""

from __future__ import annotations

import contextlib
import io
import os
import subprocess
import sys
import time
import unittest
from pathlib import Path

from tools.verify import __main__ as cli
from tools.verify import gates, proc
from tools.verify.tests.support import RepoCase, py

ROOT = Path(__file__).resolve().parents[3]
OK = py("print('fine'); print('Summary: 0 issues')")
FAIL = py("import sys; print('broke at line 9'); sys.exit(3)")
#: A gate line's time, whatever a loaded host makes it.
TIME = r"\(\d+s\)"


class TestRunner(RepoCase):
    def run_gates(self, gate_list: list[gates.Gate], files: list[str] | None = None, history: bool = True) -> tuple[list[gates.Outcome], list[str]]:
        lines: list[str] = []
        out = gates.run(gate_list, repo=self.repo, files=files or [], out_dir=self.tmp / "out", history=history, emit=lines.append)
        return out, lines

    def test_a_failure_does_not_stop_the_run(self) -> None:
        outcomes, lines = self.run_gates([gates.Gate("first", OK), gates.Gate("broken", FAIL), gates.Gate("last", OK)])
        self.assertEqual([o.status for o in outcomes], [0, 3, 0])
        self.assertRegex(lines[1], rf"^broken: exit=3 {TIME}$")
        self.assertEqual(lines[2], "  | broke at line 9")
        self.assertRegex(lines[3], rf"^last: exit=0 {TIME}$")
        self.assertRegex(lines[-1], rf"^gates: 2 passed, 1 failed, 0 skipped in \d+s$")
        self.assertTrue((self.tmp / "out" / "broken.out").is_file())

    def test_each_gate_is_timed(self) -> None:
        outcomes, lines = self.run_gates([gates.Gate("slow", py("import time; time.sleep(1.2)")), gates.Gate("quick", OK)])
        self.assertGreaterEqual(outcomes[0].seconds, 1.2)
        self.assertEqual(lines[0], f"slow: exit=0 ({outcomes[0].seconds:.0f}s)")
        self.assertEqual(lines[-1], f"gates: 2 passed, 0 failed, 0 skipped in {sum(o.seconds for o in outcomes):.0f}s")

    def test_history_gates_are_skipped_without_history(self) -> None:
        g = gates.Gate("tags", OK, needs_history=True)
        outcomes, lines = self.run_gates([g], history=False)
        self.assertIsNone(outcomes[0].status)
        self.assertIn("tags: skipped (needs git history)", lines)
        outcomes, _ = self.run_gates([g], history=True)
        self.assertEqual(outcomes[0].status, 0)

    def test_without_history_a_gate_may_run_differently(self) -> None:
        code = "import os, sys; sys.exit(0 if os.environ.get('VERIFY_TEST_NO_HISTORY') == '1' else 5)"
        g = gates.Gate("tags", py(code), needs_history=True, without_history={"VERIFY_TEST_NO_HISTORY": "1"})
        outcomes, lines = self.run_gates([g], history=False)
        self.assertEqual(outcomes[0].status, 0)
        self.assertRegex(lines[0], rf"^tags: exit=0 {TIME}$")
        outcomes, _ = self.run_gates([g], history=True)
        self.assertEqual(outcomes[0].status, 5)

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
        self.assertRegex(lines[0], rf"^lint: exit=0 {TIME}$")
        self.assertEqual(lines[1], "  | Summary: 0 issues")


def go_test(head: list[str], oks: int, status: int = 1) -> tuple[str, ...]:
    """A gate that prints head, then oks packages' ok lines and FAIL, as
    go test ./... does when an early package fails."""
    body = [*head, *(f"ok  \tpkg/p{i:02}\t0.1s" for i in range(oks)), "FAIL"]
    return py(f"import sys; print({chr(10).join(body)!r}); sys.exit({status})")


class TestFailureLines(RepoCase):
    """A failing gate quotes the lines that name its failure, then its
    tail (MADR A3)."""

    def quoted(self, g: tuple[str, ...]) -> list[str]:
        outcomes = gates.run([gates.Gate("t", g)], repo=self.repo, files=[], out_dir=self.tmp / "out", emit=lambda _: None)
        return list(outcomes[0].lines)

    def tail(self, oks: int) -> list[str]:
        return [*(f"ok  \tpkg/p{i:02}\t0.1s" for i in range(oks - gates.TAIL + 1, oks)), "FAIL"]

    def test_an_early_failure_is_named(self) -> None:
        head = ["=== RUN   TestX", "--- FAIL: TestX (0.00s)", "    x_test.go:12: got 1, want 2", "FAIL\tpkg/early\t0.2s"]
        lines = self.quoted(go_test(head, 30))
        self.assertEqual(lines, [*head[1:], *self.tail(30)])

    def test_a_failure_line_in_the_tail_is_quoted_once(self) -> None:
        lines = self.quoted(go_test(["--- FAIL: TestY (0.00s)"], 3))
        self.assertEqual(lines.count("--- FAIL: TestY (0.00s)"), 1)
        self.assertEqual(lines[0], "--- FAIL: TestY (0.00s)")
        self.assertEqual(len(lines), 5)

    def test_the_failure_lines_are_limited(self) -> None:
        head = [f"--- FAIL: Test{i:02} (0.00s)" for i in range(gates.FAIL_QUOTE + 10)]
        lines = self.quoted(go_test(head, 30))
        self.assertEqual(lines[: gates.FAIL_QUOTE], head[: gates.FAIL_QUOTE])
        self.assertEqual(lines[gates.FAIL_QUOTE], "… 10 more failure lines")
        self.assertEqual(lines[gates.FAIL_QUOTE + 1 :], self.tail(30))

    def test_a_panic_and_a_position_are_quoted(self) -> None:
        head = ["panic: runtime error: index out of range [3] with length 3", "palette/x.go:12:3: undefined: y"]
        lines = self.quoted(go_test(head, 30))
        self.assertEqual(lines[:2], head)

    def test_a_passing_gate_quotes_only_its_quote_lines(self) -> None:
        g = go_test(["--- FAIL: TestPlanted (0.00s)", "Ran 3 tests"], 30, status=0)
        outcomes = gates.run([gates.Gate("t", g, quote=(r"^Ran \d+",))], repo=self.repo, files=[], out_dir=self.tmp / "out", emit=lambda _: None)
        self.assertEqual(list(outcomes[0].lines), ["Ran 3 tests"])


#: A child that runs a quick gate, then one that sleeps, and prints as the
#: runner does by default.
CHILD = """
import sys
from pathlib import Path
from tools.verify import gates
gates.run(
    [gates.Gate("quick", (sys.executable, "-c", "pass")), gates.Gate("slow", (sys.executable, "-c", "import time; time.sleep(3)"))],
    repo=Path(sys.argv[1]), files=[], out_dir=Path(sys.argv[2]),
)
"""


class TestLiveOutput(RepoCase):
    def test_each_line_arrives_as_its_gate_ends(self) -> None:
        # stdout is a pipe here, as it is under remote: the quick gate's line
        # must come through while the slow gate still runs. That is measured
        # as the wait from it to the end of the output, since poll() can see
        # a child that has flushed at exit as still running.
        env = proc.environ({"PYTHONUNBUFFERED": None})
        with subprocess.Popen(
            [sys.executable, "-c", CHILD, str(self.repo), str(self.tmp / "out")],
            cwd=ROOT, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
        ) as p:
            assert p.stdout is not None
            first = p.stdout.readline()
            start = time.monotonic()
            rest = p.stdout.read()
            wait = time.monotonic() - start
        self.assertEqual(p.returncode, 0, first + rest)
        self.assertRegex(first, rf"^quick: exit=0 {TIME}\n$")
        self.assertGreater(wait, 1.5, "the first line came only when the run ended")
        self.assertRegex(rest, rf"(?m)^slow: exit=0 {TIME}$")


class TestGate(unittest.TestCase):
    def test_argv_puts_in_the_go_files(self) -> None:
        g = gates.Gate("g", ("make", "pre-add-check", f"FILES={gates.GO_FILES_JOINED}", gates.GO_FILES))
        self.assertEqual(g.argv(["a.go", "b.go"]), ["make", "pre-add-check", "FILES=a.go b.go", "a.go", "b.go"])

    def test_select(self) -> None:
        gs = [gates.Gate("a", OK), gates.Gate("b", OK), gates.Gate("c", OK, default=False)]
        self.assertEqual([g.name for g in gates.select(gs)], ["a", "b"])
        self.assertEqual([g.name for g in gates.select(gs, only=["c", "a"])], ["a", "c"])
        self.assertEqual([g.name for g in gates.select(gs, skip=["b"])], ["a"])
        with self.assertRaisesRegex(gates.SelectError, "no gate z"):
            gates.select(gs, only=["z"])


#: The default list (MADR A2), less the script tests, which follow scripts/.
DEFAULT = {
    "gofmt", "modernize", "vet-freebsd-amd64", "vet-openbsd-amd64", "vet-linux-386", "race", "shuffle", "lc-all-c",
    "vuln", "release-check", "shellcheck", "markdownlint", "actionlint", "tools-tests",
}
#: The gates whose checks release-check runs, by name only.
REPEATS = ["pre-add-check", "lint", "workspace-mode", "tidy", "modules", "apicheck", "examples"]


class TestDefaultGates(unittest.TestCase):
    def setUp(self) -> None:
        self.list = gates.default_gates(ROOT)
        self.gates = {g.name: g for g in self.list}

    def test_history(self) -> None:
        needing = {n for n, g in self.gates.items() if g.needs_history}
        self.assertEqual(needing, {"apicheck", "release-check"})
        self.assertEqual(self.gates["release-check"].without_history, {"GO_PRECHECK_SKIP_APICHECK": "1"})
        self.assertIsNone(self.gates["apicheck"].without_history)

    def test_go_files(self) -> None:
        self.assertEqual({n for n, g in self.gates.items() if g.needs_go_files}, {"gofmt", "pre-add-check"})

    def test_the_default_list(self) -> None:
        chosen = [g.name for g in gates.select(self.list)]
        self.assertEqual({n for n in chosen if not n.startswith("script-test-")}, DEFAULT)
        self.assertIn("script-test-go-precheck", chosen)
        self.assertEqual([n for n in chosen if n in REPEATS], [])
        self.assertEqual(sorted(g.name for g in gates.select(self.list, only=REPEATS)), sorted(REPEATS))

    def test_the_step_gates_are_all_there(self) -> None:
        for n in (*DEFAULT, *REPEATS, "script-test-go-precheck"):
            self.assertIn(n, self.gates)
        self.assertEqual(self.gates["workspace-mode"].env, {"GOWORK": None})
        self.assertEqual(self.gates["race"].env, {"GOWORK": "off"})

    def test_no_gate_runs_a_shell(self) -> None:
        for g in self.gates.values():
            self.assertNotIn(os.path.basename(g.args[0]), {"sh", "bash", "zsh", "cmd", "powershell"}, g.name)

    def test_markdownlint_runs_the_makefiles_pin(self) -> None:
        # The gate runs make's pinned markdownlint-cli2, never the newest
        # release npx would choose (docs/decisions/0018-MADR-python-verification-tools.md A4).
        self.assertEqual(self.gates["markdownlint"].args, ("make", "markdownlint"))
        makefile = (ROOT / "Makefile").read_text(encoding="utf-8")
        self.assertRegex(makefile, r"(?m)^MARKDOWNLINT \?= npx --yes markdownlint-cli2@\d+\.\d+\.\d+$")


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
