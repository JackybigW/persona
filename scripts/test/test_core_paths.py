"""schema/core-paths.txt (#14916): it parses, every explicit path exists, module directories are never core, and sample
paths classify as expected. Run: python3 scripts/test/test_core_paths.py"""
import fnmatch
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))
import core_paths as cp  # noqa: E402
import module_classification as mc  # noqa: E402

FILES = cp.tracked()
CORE = cp.Core()

SAMPLES = {
    True: [
        "server/internal/hub/hub.go", "server/internal/modapi/modapi.go", "server/internal/modules/all/all.go",
        "server/internal/modules/all/node/node.go", "server/go.mod", "sdk/go/ext/ext.go",
        "web/src/views/chat/ChatView.tsx", "web/src/core/module-host.ts", "web/src/main.tsx",
        "android/app/src/main/kotlin/A.kt", "android/core/moduleapi/src/main/kotlin/A.kt",
        "apple/Persona/Views/Chat/ChatView.swift", "apple/Shared/Design/Theme.swift", "apple/Persona/App/Modules.swift",
        "apple/modules.yml", "schema/core-paths.txt", "schema/module-classification.txt", "scripts/hooks/merge-pr.sh",
        ".github/CODEOWNERS", "server/internal/proto/proto.go",
        "server/internal/archguard/archguard_test.go",  # a guard: its tests count
        "scripts/test/test_module_flags.py",
    ],
    False: [
        "docs/architecture/core-and-modules.md", "README.md", "server/internal/modules/voice/voice.go",
        "server/internal/modules/voice/module.json", "web/src/modules/lark/module.ts", "web/src/views/knowledge/Page.tsx",
        "android/module/voice/src/main/kotlin/A.kt", "apple/Persona/Views/Mail/MailView.swift", "apple/Persona/Voice/V.swift",
        "apple/Modules/Health/H.swift", "apple/project.yml", "web/package.json", "android/build.gradle.kts",
        # tests-only changes inside a core path
        "server/internal/hub/hub_test.go", "server/internal/hub/testdata/x.json", "web/src/views/chat/ChatView.test.tsx",
        "android/app/src/test/kotlin/A.kt", "android/core/hub/src/androidTest/kotlin/A.kt",
        "apple/PersonaTests/ChatTests.swift", "server/internal/modapi/modapi_test.go", "web/src/core/module-host.test.ts",
    ],
}


class CorePathsTests(unittest.TestCase):
    def test_parses(self):
        classified, explicit = cp.parse(cp.FILE.read_text(encoding="utf-8"))
        self.assertEqual(set(classified), set(cp.KINDS))
        self.assertTrue(explicit)

    def test_bad_lines_are_refused(self):
        base = "classified go server\nclassified web web/src\nclassified gradle android\nclassified apple apple\n"
        for bad in ("nonsense x\n", "core\n", "core /abs\n", "core a/../b\n", "classified go server\n"):
            with self.assertRaises(cp.FormatError, msg=bad):
                cp.parse(base + bad)
        with self.assertRaises(cp.FormatError):
            cp.parse("classified go server\n")

    def test_every_explicit_path_exists(self):
        for e in CORE.explicit:
            if "*" in e.path:
                hit = any(fnmatch.fnmatchcase(f, e.path) for f in FILES)
            else:
                hit = e.path in FILES or any(f.startswith(e.path.rstrip("/") + "/") for f in FILES)
            self.assertTrue(hit, f"core-paths.txt line {e.line}: `{e.kind} {e.path}` matches no tracked file (stale: "
                                 "delete the line or point it at the new place)")

    def test_explicit_paths_are_never_module_files(self):
        for e in CORE.explicit:
            for f in FILES:
                if e.matches(f):
                    self.assertIn(CORE.classification(f), (mc.CORE, None),
                                  f"line {e.line}: {f} belongs to module {CORE.classification(f)!r}")

    def test_module_directories_are_never_core(self):
        for f in FILES:
            cls = CORE.classification(f)
            if cls not in (mc.CORE, None):
                self.assertFalse(CORE.is_core(f, tests_count=True), f"{f} is in module {cls} but counts as core")

    def test_modules_dirs_are_modules(self):
        mods = [f for f in FILES if f.startswith(("server/internal/modules/", "web/src/modules/", "android/module/"))]
        self.assertTrue(mods)
        for f in mods:
            if f.startswith("server/internal/modules/all/") or f.count("/") < 4 and f.startswith("web/src/modules/"):
                continue  # the composition roots and the module list, which are core
            self.assertFalse(cp.is_core(f), f)

    def test_samples(self):
        for want, paths in SAMPLES.items():
            for p in paths:
                self.assertEqual(cp.is_core(p), want, p)

    def test_core_files_keeps_order(self):
        self.assertEqual(cp.core_files(["docs/a.md", "server/go.mod", "web/src/modules/lark/a.ts", "apple/modules.yml"]),
                         ["server/go.mod", "apple/modules.yml"])

    def test_test_only_rule(self):
        for p in ("a/b_test.go", "x/testdata/f.json", "web/src/a.test.tsx", "android/app/src/test/A.kt",
                  "android/app/src/debugUnitTest/A.kt", "apple/PersonaUITests/A.swift", "apple/Shared/FooTests.swift"):
            self.assertTrue(cp.is_test_only(p), p)
        for p in ("server/internal/hub/hub.go", "android/app/src/main/A.kt", "web/src/test-utils.ts"):
            self.assertFalse(cp.is_test_only(p), p)

    def test_every_tracked_file_is_decided(self):
        # The reader must run over the whole repository without error.
        self.assertGreater(len([f for f in FILES if cp.is_core(f)]), 1000)


if __name__ == "__main__":
    unittest.main()
