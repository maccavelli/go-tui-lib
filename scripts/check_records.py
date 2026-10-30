#!/usr/bin/env python3
"""Checks the numbered records, the records index and every relative link in
the tracked Markdown (docs/decisions/0022-MADR-align-docs-ci-and-disclosure-gates.md;
docs/decisions/0022-PLAN-align-docs-ci-and-disclosure-gates.md Phase 1).

Usage:
  scripts/check_records.py [--root DIR]          check, and print each error
  scripts/check_records.py --next [--root DIR]   print the next record number

The files are those `git ls-files` lists, so an untracked draft is neither
checked nor counted. A record is docs/decisions/NNNN-MADR-slug.md or
NNNN-PLAN-slug.md, or docs/reports/NNNN-REPORT-slug.md or NNNN-GATES-slug.md
(AGENTS.md, "Records"). The checks:

* names: a Markdown file in docs/decisions or docs/reports that is not named
  as a record, or a file anywhere named like one in another case or width;
* placement: a record outside its kind's directory;
* pairs: a number claimed by two MADRs, a PLAN with no MADR of its number,
  and a GATES with no PLAN of its number; several PLANs and REPORTs may
  share a number;
* index: docs/README.md's "Records" table has one row per record, whose
  number and kind are the file's and whose link names it, and no row for a
  file that is not a record;
* links: in every tracked Markdown file outside a testdata directory, a
  relative link to a file that does not exist, and a link whose #fragment
  names no heading or HTML anchor of its Markdown target.

Anchors are GitHub's: a heading's text with its inline markup removed,
lowercased, every character but letters, digits, "_", "-" and spaces
dropped, spaces made hyphens, and a repeated heading given -1, -2, and so
on. Links and headings inside fenced code and code spans are ignored.

--next prints one more than the highest number any tracked record uses, so
a gap stays a gap.

Exit codes: 0 clean · 1 an error found · 2 usage error, or git failed.
"""

import argparse
import os
import posixpath
import re
import subprocess
import sys

RECORD = re.compile(r"^(\d{4})-(MADR|PLAN|REPORT|GATES)-([a-z0-9]+(?:-[a-z0-9]+)*)\.md$")
LOOKALIKE = re.compile(r"^\d+-(madr|plan|report|gates)-", re.IGNORECASE)
KIND_DIR = {"MADR": "docs/decisions", "PLAN": "docs/decisions", "REPORT": "docs/reports", "GATES": "docs/reports"}
RECORD_DIRS = ("docs/decisions", "docs/reports")
INDEX = "docs/README.md"
INDEX_HEADING = "## Records"
INDEX_ROW = re.compile(r"^\|\s*(\d{4})\s*\|\s*(MADR|PLAN|REPORT|GATES)\s*\|.*?\]\(([^)\s]+)\)")

LINK = re.compile(r"\]\(([^)]+)\)")
SCHEME = re.compile(r"^[A-Za-z][A-Za-z0-9+.-]*:")
FENCE = re.compile(r"^\s*(```|~~~)")
HEADING = re.compile(r"^ {0,3}#{1,6}(?:[ \t]+(.*?))?(?:[ \t]+#+)?[ \t]*$")
HTML_ANCHOR = re.compile(r"<a\s+(?:[^>]*\s)?(?:id|name)\s*=\s*[\"']([^\"']+)[\"']", re.IGNORECASE)


class Checker:
    def __init__(self, root, files):
        self.root = root
        self.files = set(files)
        self.errors = []
        self.anchor_cache = {}

    def error(self, msg):
        self.errors.append(msg)

    def read(self, rel):
        try:
            with open(os.path.join(self.root, rel), encoding="utf-8") as f:
                return f.read().splitlines()
        except (OSError, UnicodeDecodeError) as e:
            self.error(f"{rel}: unreadable: {type(e).__name__}")
            return None

    def records(self):
        return sorted(f for f in self.files if f.endswith(".md") and RECORD.match(posixpath.basename(f)))

    def check_names(self):
        for f in sorted(self.files):
            base = posixpath.basename(f)
            if RECORD.match(base):
                continue
            in_dir = f.endswith(".md") and posixpath.dirname(f) in RECORD_DIRS
            if in_dir or LOOKALIKE.match(base):
                self.error(f"{f}: misnamed record: want NNNN-MADR|PLAN|REPORT|GATES-slug.md, slug lowercase")

    def check_structure(self):
        by_num = {}
        for f in self.records():
            num, kind, _ = RECORD.match(posixpath.basename(f)).groups()
            by_num.setdefault(num, {}).setdefault(kind, []).append(f)
            if posixpath.dirname(f) != KIND_DIR[kind]:
                self.error(f"{f}: a {kind} belongs in {KIND_DIR[kind]}/")
        for num in sorted(by_num):
            kinds = by_num[num]
            if len(kinds.get("MADR", [])) > 1:
                self.error(f"{num}: claimed by {len(kinds['MADR'])} MADRs: {', '.join(kinds['MADR'])}")
            if kinds.get("PLAN") and not kinds.get("MADR"):
                self.error(f"{num}: a PLAN with no MADR: {', '.join(kinds['PLAN'])}")
            if kinds.get("GATES") and not kinds.get("PLAN"):
                self.error(f"{num}: a GATES with no PLAN: {', '.join(kinds['GATES'])}")

    def check_index(self):
        if INDEX not in self.files:
            self.error(f"{INDEX}: missing, and every record needs a row in it")
            return
        lines = self.read(INDEX)
        if lines is None:
            return
        rows = {}
        in_records = False
        for i, line in enumerate(lines, 1):
            if line.startswith("## "):
                in_records = line.strip() == INDEX_HEADING
                continue
            m = INDEX_ROW.match(line) if in_records else None
            if not m:
                continue
            num, kind, target = m.groups()
            rel = posixpath.normpath(posixpath.join(posixpath.dirname(INDEX), target))
            rec = RECORD.match(posixpath.basename(rel))
            if not rec or (rec.group(1), rec.group(2)) != (num, kind):
                self.error(f"{INDEX}:{i}: the row says {num} {kind}, and links {target}")
                continue
            if rel in rows:
                self.error(f"{INDEX}:{i}: a second row for {rel}; the first is line {rows[rel]}")
                continue
            rows[rel] = i
        for f in self.records():
            if f not in rows:
                self.error(f"{INDEX}: no row for {f}")

    def anchors(self, rel):
        if rel in self.anchor_cache:
            return self.anchor_cache[rel]
        out = set()
        seen = {}
        in_fence = False
        for line in self.read(rel) or []:
            if FENCE.match(line):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            out.update(HTML_ANCHOR.findall(line))
            m = HEADING.match(line)
            if m:
                slug = heading_slug(m.group(1) or "")
                n = seen.get(slug, 0)
                seen[slug] = n + 1
                out.add(slug if n == 0 else f"{slug}-{n}")
        self.anchor_cache[rel] = out
        return out

    def check_links(self, rel):
        lines = self.read(rel)
        if lines is None:
            return
        in_fence = False
        for i, line in enumerate(lines, 1):
            if FENCE.match(line):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            for m in LINK.finditer(strip_code_spans(line)):
                target = m.group(1).strip()
                if not target or SCHEME.match(target) or any(c.isspace() for c in target):
                    continue
                path, _, fragment = target.partition("#")
                if path:
                    resolved = posixpath.normpath(posixpath.join(posixpath.dirname(rel), path))
                    full = os.path.join(self.root, resolved)
                    if resolved.startswith("../") or not os.path.exists(full):
                        self.error(f"{rel}:{i}: broken link: {target}")
                        continue
                else:
                    resolved = rel
                if fragment and resolved.endswith(".md") and os.path.isfile(os.path.join(self.root, resolved)):
                    if fragment not in self.anchors(resolved):
                        self.error(f"{rel}:{i}: broken anchor: {target}")

    def markdown(self):
        return sorted(f for f in self.files
                      if f.endswith(".md") and "testdata" not in f.split("/")[:-1])

    def run(self):
        self.check_names()
        self.check_structure()
        self.check_index()
        for f in self.markdown():
            self.check_links(f)
        return self.errors


def strip_code_spans(line):
    """line with each `code span` removed, so a link written as an example
    inside one is not read as a link."""
    return "".join(p for i, p in enumerate(line.split("`")) if i % 2 == 0)


def heading_slug(text):
    text = re.sub(r"!?\[([^\]]*)\]\([^)]*\)", r"\1", text)
    text = re.sub(r"<[^>]+>", "", text)
    text = text.replace("`", "").replace("*", "")
    text = re.sub(r"(?<!\w)_+|_+(?!\w)", "", text)
    text = re.sub(r"[^\w\- ]", "", text.strip().lower())
    return text.replace(" ", "-")


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
    ap.add_argument("--next", action="store_true", help="print the next record number")
    args = ap.parse_args(argv)
    files = tracked(args.root)
    if files is None:
        print("check-records: git ls-files failed", file=sys.stderr)
        return 2
    c = Checker(args.root, files)
    if args.next:
        nums = [int(RECORD.match(posixpath.basename(f)).group(1)) for f in c.records()]
        print(f"{max(nums, default=0) + 1:04d}")
        return 0
    errors = c.run()
    for e in errors:
        print(e)
    print(f"check-records: {len(c.records())} record(s), {len(c.markdown())} Markdown file(s), {len(errors)} error(s)")
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
