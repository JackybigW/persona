#!/usr/bin/env python3
"""A PR that changes Persona Core says why (#14916; CONTRIBUTING.md "Changing Persona Core"):

    scripts/hooks/core-change-check.py [--snapshot <file>] <PR number>

Reads the PR's changed files, labels and description through the GitHub CLI. When it changes a core path (the list is
schema/core-paths.txt; a test-only change does not count) the PR must carry the label `core-change` and its description
must have a `## Why this must change the core` section with at least one real line. Anything else passes untouched:
module-only, docs-only and tests-only PRs. Exit 0 = ok, 1 = the PR needs the label or the section, 2 = could not read the PR.

The PR text is data: it is only searched, never run. scripts/hooks/merge-pr.sh calls this before it merges, and a workflow
may call it the same way (the proposal is in the issue).
"""
import json
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts" / "test"))
import core_paths  # noqa: E402

LABEL = "core-change"
HEADING = "## Why this must change the core"
LISTED = 20
PLACEHOLDER = re.compile(r"(|n/?a|none|nothing|no|tbd|todo|to do|-+|\.+|…|x|\?+|<[^>]*>|\[[^\]]*\])[.!]?", re.I)


def reason(body):
    """The first real line of the `## Why this must change the core` section, or None."""
    text = re.sub(r"<!--.*?-->", "", body or "", flags=re.S)
    fence, in_section = None, False
    for line in text.splitlines():
        stripped = line.strip()
        m = re.match(r"(`{3,}|~{3,})", stripped)
        if m:
            fence = None if fence and stripped.startswith(fence) else (fence or m.group(1))
            if in_section:
                continue
        if fence and not m:
            if in_section and stripped:
                return stripped  # code in the section is still an explanation
            continue
        if re.match(r"#{1,6}\s", stripped):
            if in_section:
                return None
            in_section = stripped == HEADING
            continue
        if in_section and stripped and not PLACEHOLDER.fullmatch(re.sub(r"^[-*>]\s*", "", stripped)):
            return stripped
    return None


def evaluate(files, labels, body):
    """(ok, message) for a PR with these changed files (renames: both names), label names and description."""
    touched = core_paths.core_files(files)
    if not touched:
        return True, "core-change-check: no core path changed; nothing to add"
    missing = []
    if LABEL not in labels:
        missing.append(f"add the label `{LABEL}`")
    if reason(body) is None:
        missing.append(f"add a section to the description with the exact heading `{HEADING}` and a line saying why the "
                       "core must change (not a placeholder)")
    shown = "\n".join(f"  {p}" for p in touched[:LISTED])
    more = f"\n  … and {len(touched) - LISTED} more" if len(touched) > LISTED else ""
    listing = f"{len(touched)} core file{'s' * (len(touched) != 1)} changed (schema/core-paths.txt):\n{shown}{more}"
    if missing:
        return False, ("core-change-check: this PR changes Persona Core and is not ready.\n"
                       + listing + "\nTo fix it: " + "; ".join(missing)
                       + ".\nSee CONTRIBUTING.md \"Changing Persona Core\".")
    return True, "core-change-check: core change with label and reason.\n" + listing


def gh(*args):
    return subprocess.run(["gh", *args], check=True, capture_output=True, text=True).stdout


def read_pr(number, snapshot=None):
    """(files, label names, body); labels and body from the snapshot file (merge-pr.sh's one read of the PR) when given."""
    repo = gh("repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner").strip()
    if snapshot:
        meta = json.loads(Path(snapshot).read_text(encoding="utf-8"))
    else:
        meta = json.loads(gh("pr", "view", number, "--json", "labels,body"))
    files = []
    pages = gh("api", "--paginate", f"repos/{repo}/pulls/{number}/files?per_page=100", "--jq",
               '.[] | [.filename, (.previous_filename // "")] | @tsv')
    for row in pages.splitlines():
        for name in row.split("\t"):
            if name and name not in files:
                files.append(name)
    return files, [l["name"] for l in meta.get("labels") or []], meta.get("body") or ""


def main(argv):
    args, snapshot = argv[1:], None
    if args[:1] == ["--snapshot"] and len(args) > 1:
        snapshot, args = args[1], args[2:]
    if len(args) != 1 or not args[0].isdigit():
        print("usage: scripts/hooks/core-change-check.py [--snapshot <pr view json file>] <PR number>", file=sys.stderr)
        return 2
    try:
        files, labels, body = read_pr(args[0], snapshot)
    except (subprocess.CalledProcessError, ValueError, OSError) as e:
        print(f"core-change-check: could not read PR #{args[0]}: {e}", file=sys.stderr)
        return 2
    ok, message = evaluate(files, labels, body)
    print(message, file=sys.stderr if not ok else sys.stdout)
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
