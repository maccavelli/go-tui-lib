"""tools.verify.config: the local config's path and checks."""

from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

from tools.verify import config

EXAMPLE = Path(__file__).resolve().parents[1] / "verify.example.json"


class TestConfig(unittest.TestCase):
    def setUp(self) -> None:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.dir = Path(tmp.name)

    def write(self, data: object) -> Path:
        path = self.dir / "verify.json"
        path.write_text(data if isinstance(data, str) else json.dumps(data))
        return path

    def test_default_path(self) -> None:
        self.assertEqual(config.default_path({"XDG_CONFIG_HOME": "/cfg"}), Path("/cfg/go-tui-lib/verify.json"))
        self.assertEqual(config.default_path({}), Path.home() / ".config" / "go-tui-lib" / "verify.json")

    def test_the_example_loads(self) -> None:
        c = config.load(EXAMPLE)
        self.assertEqual(sorted(c.hosts), ["linux", "windows"])
        self.assertEqual(c.host("windows").kind, "windows")

    def test_a_host(self) -> None:
        c = config.load(self.write({"hosts": {"lin": {"kind": "posix", "ssh": "dest", "path_prepend": ["$HOME/bin"]}}, "identifiers": ["SeCrEt"]}))
        h = c.host("lin")
        self.assertEqual((h.kind, h.ssh, h.path_prepend, h.python), ("posix", "dest", ("$HOME/bin",), "python3"))
        self.assertTrue(c.identifiers[0].search("a secret here"))
        with self.assertRaisesRegex(config.ConfigError, r"no host 'win' \(hosts: lin\)"):
            c.host("win")

    def test_errors(self) -> None:
        for data, want in [
            ("{not json", "verify.json"),
            ([], "not a JSON object"),
            ({"hosts": []}, "hosts is not an object"),
            ({"hosts": {"h": {"kind": "vms", "ssh": "d"}}}, "kind 'vms'"),
            ({"hosts": {"h": {"kind": "posix"}}}, "no ssh destination"),
            ({"hosts": {"h": {"kind": "posix", "ssh": "d", "path_prepend": "bin"}}}, "path_prepend"),
            ({"identifiers": ["("]}, "pattern 0 does not compile"),
            ({"identifiers": "x"}, "identifiers is not a list"),
        ]:
            with self.subTest(data=data), self.assertRaisesRegex(config.ConfigError, want):
                config.load(self.write(data))

    def test_missing_file_names_the_path_and_the_example(self) -> None:
        with self.assertRaisesRegex(config.ConfigError, r"absent\.json: no config; copy tools/verify/verify\.example\.json"):
            config.load(self.dir / "absent.json")


if __name__ == "__main__":
    unittest.main()
