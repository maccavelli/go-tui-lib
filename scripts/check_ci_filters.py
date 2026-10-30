#!/usr/bin/env python3
"""Checks that ci.yml's path filters and the Docs workflow agree, so that a
docs-only change skips no check that reads it
(docs/decisions/0022-MADR-align-docs-ci-and-disclosure-gates.md;
docs/decisions/0022-PLAN-align-docs-ci-and-disclosure-gates.md Phase 2).

Usage: scripts/check_ci_filters.py [--root DIR]

It reads .github/workflows/ci.yml and docs.yml as YAML, with the reader
below, the Makefile's recipes, and the files `git ls-files` lists, and
requires each part:

  filters  CI's push and pull_request paths-ignore lists are equal and not
           empty, and CI has no paths filter; CI keeps every trigger in
           CI_TRIGGERS, push to main and to v* tags and every pull request,
           which GitHub never path-filters for a tag
  docs     Docs' push and pull_request paths are equal, and are CI's
           ignored paths and DOCS_EXTRA, no more and no fewer
  tracked  each ignored pattern matches a tracked file, and Docs runs on
           every tracked file CI ignores
  readers  CI ignores no tracked file under a testdata directory, which
           go test reads, such as command/testdata/**/*.md; for each of
           READERS whose input CI ignores, Docs runs its make target and
           the Makefile's recipe for it runs the reader; and every tracked
           file a Go file names by a literal path, as a string or as the
           literal parts of a filepath.Join, is either not ignored or read
           by one of READERS, or listed in NOT_READS as named and not
           read; a NOT_READS line that no longer matches fails as stale
  gates    both workflows run each of DOC_TARGETS through make, neither
           names markdownlint-cli2 itself, and the Makefile pins it once
  pins     every action Docs uses is used by CI at the same pin, and Docs
           reads the repository only (permissions: contents: read)

The reader is the subset of YAML the workflows use, so the check needs only
the standard library: block mappings and sequences, plain and quoted
scalars, flow sequences of scalars, block scalars (| and >) and comments.
An anchor, an alias, a tag, a flow mapping, a document marker, a duplicate
key or a tab in indentation is refused, not guessed at. It is ported from
acp-go-sdk2's scripts/check_ci_filters.py (that repository's
0037-PLAN-align-docs-ci-and-disclosure-gates.md).

Exit codes: 0 the contract holds · 1 it does not · 2 a file cannot be read,
a workflow is outside the subset, or git failed.
"""

import argparse
import os
import posixpath
import re
import subprocess
import sys

CI = ".github/workflows/ci.yml"
DOCS = ".github/workflows/docs.yml"
DOCS_EXTRA = (".markdownlint-cli2.jsonc", DOCS)
DOC_TARGETS = ("markdownlint", "check-records", "docs-check", "disclosure-check")

# The triggers CI must keep: (event, key, value). A trigger added to ci.yml
# is added here in the same change, so that losing it fails.
CI_TRIGGERS = (("push", "branches", "main"), ("push", "tags", "v*"), ("pull_request", None, None))

# The checks that read tracked documents, by the file that reads them: what
# it reads, the make target Docs runs it through, and a word that target's
# recipe must hold.
READERS = (
    ("internal/conformance/defaults_test.go", ("docs/guides/defaults.md", "docs/decisions/*.md"),
     "docs-check", "TestDefaultsInventory"),
    ("scripts/go-examples.py", ("docs/guides/*.md",), "docs-check", "--check-guide"),
)

# Literal paths in Go that name a tracked document without reading it, each
# with the reason. A new literal that names an ignored file fails until it
# is a reader in READERS or a line here.
NOT_READS = {
    ("fuzzy/example_test.go", "README.md"): "a candidate string the example matches",
    ("fuzzy/example_test.go", "docs/guides/commands.md"): "a candidate string the example matches",
    ("palette/providers_test.go", "README.md"): "a key of an in-memory fstest.MapFS",
    ("internal/conformance/defaults_test.go", "AGENTS.md"): "a name the test looks for in a row's status",
}


class YAMLError(ValueError):
    """A workflow outside the YAML subset the reader handles."""


def _strip_comment(s):
    """s without its comment: a # at its start or after a space, outside quotes."""
    quote = ""
    for i, c in enumerate(s):
        if quote:
            if c == quote:
                quote = ""
        elif c in "'\"" and (i == 0 or s[i - 1] in " [,:"):
            quote = c
        elif c == "#" and (i == 0 or s[i - 1] in " \t"):
            return s[:i].rstrip()
    return s.rstrip()


def _split_flow(s):
    """The items inside a flow sequence, split at commas outside quotes."""
    items, cur, quote = [], "", ""
    for c in s:
        if quote:
            cur += c
            if c == quote:
                quote = ""
        elif c in "'\"":
            quote = c
            cur += c
        elif c == ",":
            items.append(cur.strip())
            cur = ""
        else:
            cur += c
    if quote:
        raise YAMLError(f"an unclosed quote in [{s}]")
    if cur.strip():
        items.append(cur.strip())
    return items


def scalar(s):
    """A scalar, or a flow sequence of scalars, of the subset."""
    s = s.strip()
    if s.startswith("[") and s.endswith("]"):
        return [scalar(x) for x in _split_flow(s[1:-1])]
    if len(s) >= 2 and s[0] == s[-1] == "'":
        return s[1:-1].replace("''", "'")
    if len(s) >= 2 and s[0] == s[-1] == '"':
        return s[1:-1].replace('\\"', '"').replace("\\\\", "\\")
    if s[:1] in ("{", "&", "*", "!", "%", "@", "`", "'", '"', "["):
        raise YAMLError(f"outside the subset: {s[:40]}")
    return s


def _key(s):
    """A mapping entry's key and the rest of its line, or None."""
    m = re.match(r"^([A-Za-z0-9_.-]+):(?:\s+(.*))?$", s)
    return (m.group(1), m.group(2) or "") if m else None


def load(text):
    """The value of a workflow file in the subset."""
    rows = []
    raw = text.split("\n")
    for i, line in enumerate(raw):
        if line.rstrip() in ("---", "..."):
            raise YAMLError(f"line {i + 1}: a document marker")
        if "\t" in line[: len(line) - len(line.lstrip())]:
            raise YAMLError(f"line {i + 1}: a tab in indentation")
        body = _strip_comment(line)
        if body.strip():
            rows.append([len(body) - len(body.lstrip()), body.strip(), i])
    if not rows:
        return None
    value, end = _block(rows, 0, rows[0][0], raw)
    if end != len(rows):
        raise YAMLError(f"line {rows[end][2] + 1}: unexpected indentation")
    return value


def _block_scalar(raw, rows, j, indent, style):
    """The block scalar after rows[j-1]: the raw lines indented more than indent."""
    k = rows[j - 1][2] + 1
    lines = []
    while k < len(raw) and (not raw[k].strip() or len(raw[k]) - len(raw[k].lstrip()) > indent):
        lines.append(raw[k])
        k += 1
    while lines and not lines[-1].strip():
        lines.pop()
    pad = min((len(x) - len(x.lstrip()) for x in lines if x.strip()), default=0)
    text = "\n".join(x[pad:] for x in lines)
    if style.startswith(">"):
        text = " ".join(x.strip() for x in text.split("\n"))
    while j < len(rows) and rows[j][2] < k:
        j += 1
    return text + ("" if style.endswith("-") else "\n"), j


def _value(rows, j, indent, rest, raw):
    """The value after a key or a dash: rest, a block scalar, or a nested block."""
    if rest in ("|", "|-", ">", ">-"):
        return _block_scalar(raw, rows, j, indent, rest)
    if rest:
        return scalar(rest), j
    if j < len(rows) and rows[j][0] > indent:
        return _block(rows, j, rows[j][0], raw)
    if j < len(rows) and rows[j][0] == indent and rows[j][1].startswith("- "):
        return _block(rows, j, indent, raw)
    return None, j


def _block(rows, j, indent, raw):
    """The block mapping or sequence at rows[j], indented indent."""
    if rows[j][1] == "-" or rows[j][1].startswith("- "):
        out = []
        while j < len(rows) and rows[j][0] == indent and (rows[j][1] == "-" or rows[j][1].startswith("- ")):
            rest = rows[j][1][1:].strip()
            if _key(rest) and not rest.startswith(("'", '"', "[")):
                inner = indent + 2
                rows[j] = [inner, rest, rows[j][2]]
                item, j = _block(rows, j, inner, raw)
            else:
                item, j = _value(rows, j + 1, indent, rest, raw)
            out.append(item)
        return out, j
    out = {}
    while j < len(rows) and rows[j][0] == indent:
        kv = _key(rows[j][1])
        if not kv:
            raise YAMLError(f"line {rows[j][2] + 1}: not a mapping entry")
        key, rest = kv
        if key in out:
            raise YAMLError(f"line {rows[j][2] + 1}: a duplicate key, {key}")
        out[key], j = _value(rows, j + 1, indent, rest, raw)
    return out, j


def glob_re(pattern):
    """A GitHub path filter as a regular expression: ** any path, * a name, ? a character."""
    out, i = "", 0
    while i < len(pattern):
        if pattern.startswith("**/", i):
            out += "(?:.*/)?"
            i += 3
        elif pattern.startswith("**", i):
            out += ".*"
            i += 2
        elif pattern[i] == "*":
            out += "[^/]*"
            i += 1
        elif pattern[i] == "?":
            out += "[^/]"
            i += 1
        else:
            out += re.escape(pattern[i])
            i += 1
    return re.compile(out + r"\Z")


def matches(patterns, path):
    return any(glob_re(p).match(path) for p in patterns)


def steps(workflow):
    for job in ((workflow or {}).get("jobs") or {}).values():
        for step in (job or {}).get("steps") or []:
            if isinstance(step, dict):
                yield step


def run_lines(workflow):
    """Each line of each step's run, stripped."""
    return [x.strip() for s in steps(workflow) if isinstance(s.get("run"), str) for x in s["run"].split("\n")]


def make_targets(workflow):
    """The targets of every make command the workflow runs."""
    out = set()
    for line in run_lines(workflow):
        words = line.split()
        if words[:1] == ["make"]:
            out.update(w for w in words[1:] if not w.startswith("-") and "=" not in w)
    return out


def recipes(makefile):
    """Each Makefile target's recipe lines, joined."""
    out, cur = {}, None
    for line in makefile.split("\n"):
        m = re.match(r"^([A-Za-z0-9_-]+):(?!=)", line)
        if m:
            cur = m.group(1)
            out.setdefault(cur, "")
        elif line.startswith("\t") and cur:
            out[cur] += line + "\n"
        elif line.strip() and not line.startswith("#"):
            cur = None
    return out


STRING = re.compile(r'"((?:[^"\\\n]|\\.)*)"|`([^`]*)`')
JOIN = re.compile(r"filepath\.Join\(([^()]*)\)")


def go_reads(root, tracked):
    """(reader, path) for each tracked file a tracked Go file names by a
    literal path: a string, relative to the root or the Go file's directory,
    or the literal parts of a filepath.Join."""
    files = set(tracked)
    out = set()
    for go in tracked:
        if not go.endswith(".go") or "testdata" in go.split("/")[:-1]:
            continue
        with open(os.path.join(root, go), encoding="utf-8") as f:
            src = f.read()
        names = [a or b for a, b in STRING.findall(src)]
        for m in JOIN.finditer(src):
            parts = [a or b for a, b in STRING.findall(m.group(1))]
            if parts:
                names.append("/".join(parts))
        for name in names:
            if "/" not in name and "." not in name:
                continue
            for cand in (name, posixpath.normpath(posixpath.join(posixpath.dirname(go), name))):
                if cand in files:
                    out.add((go, cand))
    return out


def check(root, tracked):
    """The contract's problems, each naming its part."""
    def read(rel):
        with open(os.path.join(root, rel), encoding="utf-8") as f:
            return f.read()

    ci, docs = load(read(CI)) or {}, load(read(DOCS)) or {}
    makefile = recipes(read("Makefile"))
    pins = read("Makefile").count("markdownlint-cli2@")
    problems = []

    on, don = ci.get("on") or {}, docs.get("on") or {}
    push, pr = on.get("push") or {}, on.get("pull_request") or {}
    ignore = push.get("paths-ignore") or []
    if not ignore or ignore != (pr.get("paths-ignore") or []):
        problems.append("filters: CI's push and pull_request paths-ignore differ, or are empty")
    if "paths" in push or "paths" in pr:
        problems.append("filters: CI has a paths filter")
    for event, key, value in CI_TRIGGERS:
        if event not in on or (key and value not in ((on.get(event) or {}).get(key) or [])):
            problems.append(f"filters: CI lost its trigger: {event}" + (f" {key} {value}" if key else ""))

    dpush, dpr = (don.get("push") or {}).get("paths") or [], (don.get("pull_request") or {}).get("paths") or []
    if dpush != dpr:
        problems.append("docs: Docs' push and pull_request paths differ")
    want = set(ignore) | set(DOCS_EXTRA)
    if set(dpush) != want:
        problems.append(f"docs: Docs' paths are not CI's ignored paths and {', '.join(DOCS_EXTRA)}: "
                        f"missing {sorted(want - set(dpush))}, extra {sorted(set(dpush) - want)}")

    for p in ignore:
        if not any(glob_re(p).match(t) for t in tracked):
            problems.append(f"tracked: CI ignores {p!r}, which matches no tracked file")
    for t in tracked:
        if matches(ignore, t) and not matches(dpush, t):
            problems.append(f"tracked: CI ignores {t}, and Docs does not run on it")

    ignored = [t for t in tracked if matches(ignore, t)]
    for t in ignored:
        if "testdata" in t.split("/")[:-1]:
            problems.append(f"readers: CI ignores {t}, under a testdata directory go test reads")
    dtargets = make_targets(docs)
    for reader, inputs, target, word in READERS:
        if not any(matches(inputs, t) for t in ignored):
            continue
        if reader not in tracked:
            problems.append(f"readers: {reader}, a reader of {', '.join(inputs)}, is not tracked")
        if target not in dtargets:
            problems.append(f"readers: CI ignores {', '.join(inputs)}, and Docs does not run make {target}, "
                            f"which runs {reader}")
        if word not in makefile.get(target, ""):
            problems.append(f"readers: make {target} does not run {word}, the reader in {reader}")
    found = go_reads(root, tracked)
    for go, path in sorted(found):
        if (path in ignored and (go, path) not in NOT_READS
                and not any(r == go and matches(i, path) for r, i, _, _ in READERS)):
            problems.append(f"readers: CI ignores {path}, which {go} names, and no reader of it runs in Docs")
    for go, path in sorted(set(NOT_READS) - found):
        problems.append(f"readers: NOT_READS lists {go} naming {path}, which it no longer does")

    for name, wf in (("CI", ci), ("Docs", docs)):
        for target in DOC_TARGETS:
            if target not in make_targets(wf):
                problems.append(f"gates: {name} does not run make {target}")
        if any("markdownlint-cli2" in line for line in run_lines(wf)):
            problems.append(f"gates: {name} runs markdownlint-cli2 itself, not through make markdownlint")
    if pins != 1:
        problems.append(f"gates: the Makefile names markdownlint-cli2@ {pins} time(s), not once")

    cuses = {s["uses"] for s in steps(ci) if isinstance(s.get("uses"), str)}
    for s in steps(docs):
        uses = s.get("uses")
        if isinstance(uses, str) and uses not in cuses:
            action = uses.split("@")[0]
            have = sorted(u for u in cuses if u.split("@")[0] == action)
            problems.append(f"pins: Docs uses {uses}, and CI {have or 'not ' + action}")
    if docs.get("permissions") != {"contents": "read"}:
        problems.append("pins: Docs' permissions are not contents: read")
    return problems


def main(argv):
    for stream in (sys.stdout, sys.stderr):
        stream.reconfigure(newline="\n")
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--root", default=os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    args = ap.parse_args(argv)
    try:
        r = subprocess.run(["git", "-C", args.root, "ls-files", "-z"], capture_output=True, check=False)
    except OSError:
        r = None
    if r is None or r.returncode != 0:
        print("check-ci-filters: git ls-files failed", file=sys.stderr)
        return 2
    tracked = [n for n in r.stdout.decode("utf-8", "surrogateescape").split("\0") if n]
    try:
        problems = check(args.root, tracked)
    except (OSError, UnicodeDecodeError, YAMLError) as e:
        print(f"check-ci-filters: {e}", file=sys.stderr)
        return 2
    for p in problems:
        print(f"check-ci-filters: {p}")
    print(f"check-ci-filters: {len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
