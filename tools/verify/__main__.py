"""python3 -m tools.verify <command>: see the package's documentation."""

from __future__ import annotations

import argparse
import sys
import tempfile
from collections.abc import Callable, Sequence
from pathlib import Path

from . import gates, mutate, tree


def _names(values: Sequence[str] | None) -> list[str]:
    """Gate names from repeated, comma-separated options."""
    return [n for v in values or () for n in v.split(",") if n]


def cmd_gates(args: argparse.Namespace, gate_list: Callable[[Path], list[gates.Gate]] = gates.default_gates) -> int:
    repo = tree.root(Path.cwd())
    all_gates = gate_list(repo)
    if args.list:
        for g in all_gates:
            print(g.name)
        return 0
    try:
        chosen = gates.select(all_gates, _names(args.only), _names(args.skip))
    except gates.SelectError as err:
        print(f"gates: {err}", file=sys.stderr)
        return 2
    out_dir = Path(args.out) if args.out else Path(tempfile.mkdtemp(prefix=f"verify-{args.label}-"))
    print(f"gates: {args.label}: outputs in {out_dir}")
    outcomes = gates.run(chosen, repo=repo, files=args.files, out_dir=out_dir, history=not args.no_history)
    return 1 if any(o.failed for o in outcomes) else 0


def cmd_mutate(args: argparse.Namespace) -> int:
    repo = tree.root(Path.cwd())
    try:
        mutations = mutate.load(Path(args.file))
    except mutate.MutationError as err:
        print(f"mutate: {err}", file=sys.stderr)
        return 2
    work = Path(args.work) if args.work else Path(tempfile.mkdtemp(prefix="verify-mutate-"))
    outcomes = mutate.run(mutations, repo=repo, work=work, keep=args.keep)
    return 0 if all(o.verdict == mutate.KILLED for o in outcomes) else 1


def parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(prog="python3 -m tools.verify", description="Verify a PLAN step.")
    sub = p.add_subparsers(dest="command", required=True)
    g = sub.add_parser("gates", help="run every gate of a step")
    g.add_argument("files", nargs="*", help="the step's files; its Go files go to gofmt and pre-add-check")
    g.add_argument("--label", default="gates", help="names the run and its output directory")
    g.add_argument("--out", help="where each gate's output is kept (default: a new temporary directory)")
    g.add_argument("--only", action="append", help="run only these gates, comma-separated")
    g.add_argument("--skip", action="append", help="skip these gates, comma-separated")
    g.add_argument("--no-history", action="store_true", help="skip the gates that need git tags or history")
    g.add_argument("--list", action="store_true", help="list the gates and stop")
    g.set_defaults(func=cmd_gates)
    m = sub.add_parser("mutate", help="run mutation proofs: each named test must fail")
    m.add_argument("file", help="the JSON list of mutations")
    m.add_argument("--work", help="where the copies go (default: a new temporary directory)")
    m.add_argument("--keep", action="store_true", help="keep each mutated copy")
    m.set_defaults(func=cmd_mutate)
    return p


def main(argv: Sequence[str] | None = None) -> int:
    args = parser().parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
