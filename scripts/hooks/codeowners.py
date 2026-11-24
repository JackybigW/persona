#!/usr/bin/env python3
"""Generate the Persona Core block of .github/CODEOWNERS from schema/core-paths.txt (#14916).

    python3 scripts/hooks/codeowners.py            print the block
    python3 scripts/hooks/codeowners.py --write    rewrite the block in .github/CODEOWNERS between the two marker lines
    python3 scripts/hooks/codeowners.py --check    exit 1 when the file's block is out of date

CODEOWNERS cannot reference another file, so the block is generated. The last matching pattern wins: a directory that
is mostly core is owned as a whole, a module directory inside it gets a later pattern with no owner (which un-assigns
it), and a core directory inside that gets an owner again. Patterns are directory level; a test-only change in a core
directory still requests review (CODEOWNERS cannot tell it apart; core-change-check.py can). Whether a code-owner
review is required depends on the branch ruleset, which this repository's administrator sets.
Owners must be users with write access, or GitHub lists them in repos/<repo>/codeowners/errors.
"""
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts" / "test"))
import core_paths as cp  # noqa: E402

OWNERS = ("@chyroc", "@szupzj18")
FILE = ROOT / ".github" / "CODEOWNERS"
BEGIN = "# BEGIN persona-core (generated from schema/core-paths.txt by scripts/hooks/codeowners.py; do not edit)"
END = "# END persona-core"
OWN_PERCENT = 75  # a directory with at least this share of core files is owned whole, its modules un-owned
MIN_WILDCARD = 3  # this many non-core sibling directories become one `dir/*/` line


def build_tree(files, core):
    tree = {}
    for f in files:
        node = tree
        for part in f.split("/")[:-1]:
            node = node.setdefault(part, {})
        node.setdefault("\0files", []).append((f, core(f)))
    return tree


def counts(node):
    total = core = 0
    for k, v in node.items():
        if k == "\0files":
            total += len(v)
            core += sum(1 for _, c in v if c)
        else:
            t, c = counts(v)
            total, core = total + t, core + c
    return total, core


def lines_for(node, path, owned, out):
    """Append patterns for node's content; `owned` says whether the context already has an owner."""
    dirs = sorted(k for k in node if k != "\0files")
    stat = {d: counts(node[d]) for d in dirs}

    def child(d):
        return f"{path}/{d}" if path else d

    for f, c in sorted(node.get("\0files", [])):
        if c and not owned:
            out.append((f"/{f}", True))
    if owned:
        gone = [d for d in dirs if stat[d][1] == 0]
        kept = [d for d in dirs if stat[d][1] > 0]
        if len(gone) >= MIN_WILDCARD and len(gone) > len(kept):
            # Mostly modules: un-own every child, then own the core ones again.
            out.append((f"/{path}/*/" if path else "/*/", False))
            for d in kept:
                out.append((f"/{child(d)}/", True))
        else:
            for d in gone:
                out.append((f"/{child(d)}/", False))
        for d in kept:
            if stat[d][1] != stat[d][0]:
                lines_for(node[d], child(d), True, out)
        return
    for d in dirs:
        total, core = stat[d]
        if core == 0:
            continue
        if core * 100 >= total * OWN_PERCENT:
            out.append((f"/{child(d)}/", True))
            if core != total:
                lines_for(node[d], child(d), True, out)
        else:
            lines_for(node[d], child(d), False, out)


def block(files=None):
    c = cp.Core()
    files = files if files is not None else cp.tracked()
    out = []
    lines_for(build_tree(files, lambda f: c.is_core(f, tests_count=True)), "", False, out)
    # A line that repeats an earlier one adds nothing; the last matching pattern wins.
    lines, seen = [], set()
    for pat, own in out:
        if (pat, own) in seen:
            continue
        seen.add((pat, own))
        lines.append(f"{pat} {' '.join(OWNERS)}" if own else pat)
    return "\n".join([BEGIN, *lines, END]) + "\n"


def current_text():
    return FILE.read_text(encoding="utf-8")


def replace(text, new_block):
    if BEGIN in text and END in text:
        head, rest = text.split(BEGIN, 1)
        _, tail = rest.split(END + "\n", 1) if END + "\n" in rest else (rest.split(END, 1)[0], rest.split(END, 1)[1])
        return head + new_block + tail
    return text.rstrip("\n") + "\n\n" + new_block


def extract(text):
    if BEGIN not in text or END not in text:
        return None
    return BEGIN + text.split(BEGIN, 1)[1].split(END, 1)[0] + END + "\n"


if __name__ == "__main__":
    mode = sys.argv[1] if len(sys.argv) > 1 else ""
    new = block()
    if mode == "--write":
        FILE.write_text(replace(current_text(), new), encoding="utf-8")
    elif mode == "--check":
        if extract(current_text()) != new:
            sys.exit("CODEOWNERS core block is out of date: run python3 scripts/hooks/codeowners.py --write")
    else:
        print(new, end="")
