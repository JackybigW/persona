"""The Persona Core block of .github/CODEOWNERS is generated from schema/core-paths.txt (#14916): it is current, it owns
every core file, and it leaves every module file without an owner. Run: python3 scripts/test/test_codeowners.py"""
from pathlib import Path
import re
import sys
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts" / "hooks"))
sys.path.insert(0, str(ROOT / "scripts" / "test"))
import codeowners as co  # noqa: E402
import core_paths as cp  # noqa: E402


def owned(path, patterns):
    """Whether the last matching pattern gives path an owner (CODEOWNERS: `/dir/`, `/dir/*/`, `/file`)."""
    result = False
    for pat, own in patterns:
        pat = pat.lstrip("/")
        if pat.endswith("/"):
            rx = re.escape(pat.rstrip("/")).replace(r"\*", "[^/]+") + "/"
            hit = re.match(rx, path) is not None
        else:
            hit = path == pat
        if hit:
            result = own
    return result


class CodeownersTests(unittest.TestCase):
    def test_generated_block_is_current(self):
        text = (ROOT / ".github" / "CODEOWNERS").read_text(encoding="utf-8")
        self.assertEqual(co.extract(text), co.block(),
                         "the CODEOWNERS core block is out of date: run python3 scripts/hooks/codeowners.py --write")

    def test_super_rules_line_is_kept(self):
        self.assertIn("/docs/SUPER_RULES.md @chyroc", (ROOT / ".github" / "CODEOWNERS").read_text(encoding="utf-8").splitlines())

    def test_owners_are_the_two_named(self):
        for line in co.extract((ROOT / ".github" / "CODEOWNERS").read_text(encoding="utf-8")).splitlines():
            if line and not line.startswith("#") and len(line.split()) > 1:
                self.assertEqual(line.split()[1:], ["@chyroc", "@szupzj18"], line)

    def test_block_owns_core_and_not_modules(self):
        text = (ROOT / ".github" / "CODEOWNERS").read_text(encoding="utf-8")
        patterns = []
        for line in co.extract(text).splitlines():
            if line and not line.startswith("#"):
                patterns.append((line.split()[0], len(line.split()) > 1))
        c = cp.Core()
        for f in cp.tracked():
            cls = c.classification(f)
            if c.is_core(f, tests_count=True):
                self.assertTrue(owned(f, patterns), f"core file {f} has no owner")
            elif cls not in (None, "core"):
                self.assertFalse(owned(f, patterns), f"module file {f} ({cls}) has an owner")

    def test_few_patterns(self):
        self.assertLess(len(co.block().splitlines()), 90)

    def test_replace_keeps_the_rest(self):
        old = "/docs/SUPER_RULES.md @chyroc\n\n" + co.BEGIN + "\n/x/ @a\n" + co.END + "\n# tail\n"
        new = co.replace(old, co.BEGIN + "\n/y/ @a\n" + co.END + "\n")
        self.assertEqual(new, "/docs/SUPER_RULES.md @chyroc\n\n" + co.BEGIN + "\n/y/ @a\n" + co.END + "\n# tail\n")
        self.assertEqual(co.replace("/docs/SUPER_RULES.md @chyroc\n", "B\n"), "/docs/SUPER_RULES.md @chyroc\n\nB\n")


if __name__ == "__main__":
    unittest.main()
