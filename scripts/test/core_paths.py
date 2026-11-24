"""Read schema/core-paths.txt: which paths are Persona Core (#14916; the file's header describes the format).

    is_core(path)        True when a change to this file counts as a change to the core (a test-only file does not)
    core_files(paths)    the paths that are core, in order

CLI (the numbers in a PR description):  python3 scripts/test/core_paths.py --table
Used by test_core_paths.py, scripts/hooks/codeowners.py and scripts/hooks/core-change-check.py.
"""
from dataclasses import dataclass
import fnmatch
from functools import lru_cache
from pathlib import Path
import re
import subprocess
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import module_classification as mc  # noqa: E402

ROOT = mc.ROOT
FILE = ROOT / "schema" / "core-paths.txt"
KINDS = ("go", "web", "gradle", "apple")


class FormatError(ValueError):
    pass


@dataclass(frozen=True)
class Explicit:
    kind: str  # "core" or "guard"
    path: str
    line: int

    def matches(self, path):
        if "*" in self.path:
            return fnmatch.fnmatchcase(path, self.path)
        return path == self.path or path.startswith(self.path.rstrip("/") + "/")


def parse(text):
    """(classified: kind -> roots, explicit entries); FormatError on a malformed line."""
    classified, explicit = {}, []
    for n, raw in enumerate(text.splitlines(), 1):
        words = raw.split("#", 1)[0].split()
        if not words:
            continue
        if words[0] == "classified":
            if len(words) < 3 or words[1] not in KINDS or words[1] in classified:
                raise FormatError(f"line {n}: expected `classified <{'|'.join(KINDS)}> <root>...`, once per kind: {raw!r}")
            classified[words[1]] = tuple(w.rstrip("/") for w in words[2:])
        elif words[0] in ("core", "guard"):
            if len(words) != 2 or words[1].startswith("/") or ".." in words[1].split("/"):
                raise FormatError(f"line {n}: expected `{words[0]} <repo-relative path>`: {raw!r}")
            explicit.append(Explicit(words[0], words[1], n))
        else:
            raise FormatError(f"line {n}: unknown directive {words[0]!r}: {raw!r}")
    missing = [k for k in KINDS if k not in classified]
    if missing:
        raise FormatError(f"missing `classified` line for {', '.join(missing)}")
    return classified, explicit


_TEST_FILE = re.compile(r"(_test\.go|\.test\.[cm]?[jt]sx?|\.spec\.[cm]?[jt]sx?|(Tests?|Spec)\.(swift|kt))\Z")
_TEST_DIR = re.compile(r"(^|/)(testdata|__tests__)/|/src/(test|androidTest|[A-Za-z]*Test)/")


def is_test_only(path):
    """A file that only tests something: tests may be added to the core freely."""
    parts = path.split("/")
    if _TEST_FILE.search(parts[-1]) or _TEST_DIR.search(path):
        return True
    return parts[0] == "apple" and any(re.search(r"Tests?\Z", d) for d in parts[:-1])


class Core:
    def __init__(self, root=ROOT):
        self.root = root
        self.classified, self.explicit = parse((root / "schema" / "core-paths.txt").read_text(encoding="utf-8"))
        _, self.rules = mc.parse((root / "schema" / "module-classification.txt").read_text(encoding="utf-8"))
        self.projects = sorted(
            mc.gradle_projects((root / "android" / "settings.gradle.kts").read_text(encoding="utf-8")),
            key=lambda p: -len(p))
        self._class = lru_cache(maxsize=None)(self._class_of_dir)

    def _class_of_dir(self, kind, d):
        return mc.classify(d, kind, self.rules)[0]

    def _under(self, kind, path):
        return any(path == r or path.startswith(r + "/") for r in self.classified[kind])

    def classification(self, path):
        """"core", a module id, or None for a file the classification does not cover."""
        for kind in ("go", "web"):
            if self._under(kind, path):
                d = path.rsplit("/", 1)[0] if "/" in path else ""
                while d:
                    c = self._class(kind, d)
                    if c is not None:
                        return c
                    d = d.rsplit("/", 1)[0] if "/" in d else ""
                return mc.CORE  # a file no classified directory sits above: server/go.mod, migrations
        if self._under("gradle", path):
            for p in self.projects:
                d = "android/" + p.strip(":").replace(":", "/")
                if path.startswith(d + "/"):
                    return self._class("gradle", p)
            return None
        if self._under("apple", path):
            d = path.rsplit("/", 1)[0]
            while "/" in d:
                c = self._class("apple", d)
                if c is not None:
                    return c
                d = d.rsplit("/", 1)[0]
        return None

    def is_core(self, path, tests_count=False):
        cls = self.classification(path)
        if cls is not None and cls != mc.CORE:
            return False  # a module's file is never core
        for e in self.explicit:
            if e.matches(path) and (e.kind == "guard" or tests_count or not is_test_only(path)):
                return True
        return cls == mc.CORE and (tests_count or not is_test_only(path))


@lru_cache(maxsize=None)
def _default():
    return Core()


def is_core(path):
    return _default().is_core(path)


def core_files(paths):
    return [p for p in paths if is_core(p)]


def tracked(root=ROOT):
    """Tracked files, and new files not ignored (a file added in this change counts before it is committed)."""
    out = subprocess.run(["git", "-C", str(root), "ls-files", "-z", "--cached", "--others", "--exclude-standard"], check=True, capture_output=True).stdout.decode()
    return [p for p in out.split("\0") if p]


def table(files):
    """(area, pattern, count) rows: the core files by area, at directory level."""
    c, rows = _default(), {}
    for f in files:
        if not c.is_core(f, tests_count=True):
            continue
        parts = f.split("/")
        if f.startswith("server/internal/") and len(parts) > 3:
            key = ("Server", "server/internal/" + parts[2] + "/")
        elif parts[0] in ("server", "sdk"):
            key = ("Server", "/".join(parts[:2 if len(parts) > 2 else 1]) + ("/" if len(parts) > 2 else ""))
        elif parts[0] == "web":
            key = ("Web", "/".join(parts[:3]) + "/" if len(parts) > 3 else f)
        elif parts[0] == "android":
            key = ("Android", "/".join(parts[:3]) + "/" if len(parts) > 3 else f)
        elif parts[0] == "apple":
            key = ("Apple", "/".join(parts[:3]) + "/" if len(parts) > 3 else f)
        else:
            key = ("Foundation", f)
        rows[key] = rows.get(key, 0) + 1
    return sorted((a, p, n) for (a, p), n in rows.items())


if __name__ == "__main__":
    if "--table" in sys.argv:
        files = tracked()
        total = {}
        for area, pat, n in table(files):
            total[area] = total.get(area, 0) + n
        for area, n in sorted(total.items()):
            print(f"{area}\t{n}")
        print("total core files (tests included)\t%d of %d tracked" % (sum(total.values()), len(files)))
    else:
        print("usage: core_paths.py --table")
