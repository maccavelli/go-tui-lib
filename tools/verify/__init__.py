"""Verify a PLAN step: its gates, its mutation proofs, the same gates on
other hosts, and an identifier scan of its diff.

Run it from the repository root as ``python3 -m tools.verify <command>``
(docs/decisions/0018-MADR-python-verification-tools.md). Python 3.12 or
later and the standard library only; every command is an argument list,
never a shell string.
"""
