#!/usr/bin/env python3
"""Checks the tracked text for real user-home paths
(docs/decisions/0022-MADR-align-docs-ci-and-disclosure-gates.md;
docs/decisions/0022-PLAN-align-docs-ci-and-disclosure-gates.md Phase 1).

The portable half of the disclosure rule (AGENTS.md, "Identifiers"): it
needs no private data, so full CI and the Docs workflow both run it. The
owner's own patterns stay in the global pre-push guard, which runs before
every push and is never copied here.

Usage: scripts/check_disclosure.py [--root DIR]

It reads every file `git ls-files` lists and refuses a home path whose
account is not a placeholder:

  macos-home    /Users/<name>
  linux-home    /home/<name>
  windows-home  C:\\Users\\<name>, any drive letter, either separator, with
                its backslashes JSON-escaped or not

An account is a placeholder when it is in angle brackets, <user>, or the
same JSON-escaped, \\u003cuser\\u003e; when it starts with $ or %, as $USER
or %USERNAME% do; when it is only dots or ellipses, C:\\Users\\...; or when
it is one of PLACEHOLDERS, the generic names the tree's tests and examples
use. A path inside a longer path or a URL, such
as /tmp/home/x, is not a home path.

A file with a NUL byte is binary, and skipped. A tracked file that cannot be
read, or is not UTF-8 and has no NUL byte, is refused as "unreadable", since
a check that cannot read a file has not checked it.

Each finding prints as FILE:LINE: CATEGORY, never the text found, so the
output can go into a record or a CI log.

Exit codes: 0 clean · 1 a finding · 2 usage error, or git failed.
"""

import argparse
import os
import re
import subprocess
import sys

# The generic accounts the tree's tests and examples use: tools/verify's
# redaction tests use the cryptographers' four.
PLACEHOLDERS = frozenset({"user", "u", "alice", "bob", "carol", "dave"})

_NAME = r"([^\s/\\\"'`<>()\[\]{},;]+|<[^\s<>/\\]*>|\\u003c[^\s\\/]*?\\u003e)"
# One backslash is tried before two, so that a JSON-escaped placeholder after
# a separator, \Users\\u003cuser\u003e, keeps its own backslash.
_SEP = r"(?:\\|\\\\|/)"
FORMS = (
    ("windows-home", re.compile(r"(?<![\w])[A-Za-z]:" + _SEP + r"Users" + _SEP + _NAME, re.IGNORECASE)),
    ("macos-home", re.compile(r"(?<![A-Za-z]:)(?<![\w.-])/Users/" + _NAME)),
    ("linux-home", re.compile(r"(?<![\w.-])/home/" + _NAME)),
)


def placeholder(name):
    """Whether an account name is a placeholder, not a real account."""
    low = name.lower()
    return (low.startswith(("<", "\\u003c", "$", "%")) or low in PLACEHOLDERS
            or not low.strip(".…"))


def scan_text(text):
    """The sorted (line, category) of each real home path in text."""
    found = set()
    for category, rx in FORMS:
        for m in rx.finditer(text):
            if not placeholder(m.group(1)):
                found.add((text.count("\n", 0, m.start()) + 1, category))
    return sorted(found)


def scan(root, names):
    """The (file, line, category) findings in names, relative to root."""
    out = []
    for name in names:
        try:
            with open(os.path.join(root, name), "rb") as f:
                data = f.read()
        except OSError:
            out.append((name, 0, "unreadable"))
            continue
        if b"\0" in data:
            continue
        try:
            text = data.decode("utf-8")
        except UnicodeDecodeError:
            out.append((name, 0, "unreadable"))
            continue
        out.extend((name, line, category) for line, category in scan_text(text))
    return out


def tracked(root):
    """The files git tracks under root, or None when git fails."""
    try:
        r = subprocess.run(["git", "-C", root, "ls-files", "-z"], capture_output=True, check=False)
    except OSError:
        return None
    if r.returncode != 0:
        return None
    return [n for n in r.stdout.decode("utf-8", "surrogateescape").split("\0") if n]


def main(argv):
    for stream in (sys.stdout, sys.stderr):
        stream.reconfigure(newline="\n")
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--root", default=os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    args = ap.parse_args(argv)
    names = tracked(args.root)
    if names is None:
        print("check-disclosure: git ls-files failed", file=sys.stderr)
        return 2
    findings = scan(args.root, names)
    for name, line, category in findings:
        print(f"{name}:{line}: {category}")
    print(f"check-disclosure: {len(names)} tracked file(s), {len(findings)} finding(s)")
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
