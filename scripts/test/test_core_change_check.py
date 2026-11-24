"""scripts/hooks/core-change-check.py (#14916): the rule on sample file lists, labels and descriptions, and the CLI against
a fake gh. Run: python3 scripts/test/test_core_change_check.py"""
import importlib.util
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("core_change_check", ROOT / "scripts/hooks/core-change-check.py")
ccc = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ccc)

CORE = "server/internal/hub/hub.go"
WHY = "Fixes #1\n\n## Why this must change the core\n\nThe hub must carry the new frame.\n"


class RuleTests(unittest.TestCase):
    def ok(self, files, labels=(), body=""):
        return ccc.evaluate(files, list(labels), body)

    def test_pr_without_core_paths_passes_untouched(self):
        for files in (["docs/a.md"], ["server/internal/modules/voice/v.go"], ["web/src/modules/lark/x.ts", "README.md"],
                      [CORE.replace(".go", "_test.go")], ["server/internal/hub/testdata/a.json"],
                      ["web/src/views/chat/Chat.test.tsx"], [],
                      # the default panel (tier panel) is not Core
                      ["web/src/views/chat/ChatView.tsx", "web/src/App.tsx"], ["android/feature/agents/src/main/kotlin/A.kt"],
                      ["android/app/src/main/kotlin/A.kt", "apple/Persona/Views/Chat/ChatView.swift"]):
            ok, message = self.ok(files)
            self.assertTrue(ok, (files, message))

    def test_core_change_needs_label_and_reason(self):
        for labels, body in (((), ""), ((), WHY), (("core-change",), "Fixes #1"), (("other",), WHY)):
            ok, message = self.ok([CORE], labels, body)
            self.assertFalse(ok, (labels, body))
            self.assertIn(CORE, message)
        ok, message = self.ok([CORE], ("core-change",), WHY)
        self.assertTrue(ok, message)

    def test_message_says_what_to_add(self):
        ok, message = self.ok([CORE])
        self.assertIn("`core-change`", message)
        self.assertIn("## Why this must change the core", message)
        ok, message = self.ok([CORE], ("core-change",), "")
        self.assertNotIn("add the label", message)
        self.assertIn("## Why this must change the core", message)
        ok, message = self.ok([CORE], (), WHY)
        self.assertIn("add the label", message)
        self.assertNotIn("add a section", message)

    def test_failure_names_the_tier_and_ignores_panel_files(self):
        ok, message = self.ok(["web/src/views/chat/ChatView.tsx", "web/src/lib/task-board.ts"])
        self.assertFalse(ok)
        self.assertIn("tier core", message)
        self.assertIn("web/src/lib/task-board.ts", message)
        self.assertNotIn("ChatView.tsx", message)
        for headless in ("android/core/hub/src/main/kotlin/A.kt", "apple/Packages/PersonaWire/Sources/A.swift", "native/app-core/src/lib.rs"):
            self.assertFalse(self.ok([headless])[0], headless)

    def test_lists_the_first_twenty(self):
        files = [f"server/internal/hub/f{i:02d}.go" for i in range(25)]
        ok, message = self.ok(files)
        self.assertIn("25 tier-core files", message)
        self.assertIn("f19.go", message)
        self.assertNotIn("f20.go", message)
        self.assertIn("and 5 more", message)

    def test_reason_rules(self):
        h = ccc.HEADING
        self.assertIsNone(ccc.reason(""))
        self.assertIsNone(ccc.reason(None))
        self.assertIsNone(ccc.reason(f"{h}\n"))
        for placeholder in ("N/A", "n/a", "none", "TBD", "TODO", "-", "...", "…", "<why>", "<!-- say why -->", "[why]", ""):
            self.assertIsNone(ccc.reason(f"{h}\n\n{placeholder}\n\n## Next\ntext"), placeholder)
        self.assertEqual(ccc.reason(f"{h}\n\n<!-- hint -->\nBecause the hub routes it.\n"), "Because the hub routes it.")
        self.assertEqual(ccc.reason(f"intro\n\n{h}  \nReason here\n## Verification\n"), "Reason here")

    def test_reason_must_be_inside_the_section(self):
        self.assertIsNone(ccc.reason(f"{ccc.HEADING}\n\n## Verification\n\nRan the tests.\n"))
        self.assertIsNone(ccc.reason("## Why this must change the core (draft)\n\nBecause.\n"))
        self.assertIsNone(ccc.reason("### Why this must change the core\n\nBecause.\n"))
        self.assertIsNone(ccc.reason("Because.\n"))

    def test_a_heading_in_a_comment_does_not_count(self):
        body = f"<!--\n{ccc.HEADING}\nBecause.\n-->\nFixes #1"
        self.assertIsNone(ccc.reason(body))

    def test_pr_text_is_data(self):
        body = f"{ccc.HEADING}\n\n$(touch /tmp/should-not-exist-core-change-check) `rm -rf /`\n"
        self.assertTrue(ccc.reason(body).startswith("$(touch"))
        self.assertFalse(os.path.exists("/tmp/should-not-exist-core-change-check"))


class CliTests(unittest.TestCase):
    def run_cli(self, state, *args):
        with tempfile.TemporaryDirectory() as d:
            gh = Path(d) / "gh"
            gh.write_text("#!" + sys.executable + """
import json, os, sys
state = json.load(open(os.environ["FAKE_GH_STATE"]))
a = sys.argv[1:]
if a[:2] == ["repo", "view"]:
    print("example/persona")
elif a[:2] == ["pr", "view"]:
    print(json.dumps({"labels": state["labels"], "body": state["body"]}))
elif a[:1] == ["api"]:
    print("\\n".join(state["files"]))
""")
            gh.chmod(gh.stat().st_mode | stat.S_IXUSR)
            (Path(d) / "state.json").write_text(json.dumps(state))
            env = dict(os.environ, PATH=d + os.pathsep + os.environ["PATH"], FAKE_GH_STATE=str(Path(d) / "state.json"))
            return subprocess.run([sys.executable, str(ROOT / "scripts/hooks/core-change-check.py"), *args], env=env,
                                  capture_output=True, text=True, timeout=60)

    def test_cli_verdicts(self):
        r = self.run_cli({"labels": [], "body": "", "files": [CORE]}, "9")
        self.assertEqual(r.returncode, 1, r.stdout + r.stderr)
        self.assertIn(CORE, r.stderr)
        r = self.run_cli({"labels": [{"name": "core-change"}], "body": WHY, "files": [CORE + "\t"]}, "9")
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
        r = self.run_cli({"labels": [], "body": "", "files": ["docs/a.md\t"]}, "9")
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)

    def test_cli_counts_the_old_name_of_a_rename(self):
        r = self.run_cli({"labels": [], "body": "", "files": ["web/src/modules/lark/A.tsx\tweb/src/lib/task-board.ts"]}, "9")
        self.assertEqual(r.returncode, 1, r.stdout + r.stderr)
        self.assertIn("web/src/lib/task-board.ts", r.stderr)

    def test_cli_usage_and_unreadable_pr(self):
        self.assertEqual(self.run_cli({}, "not-a-number").returncode, 2)
        self.assertEqual(self.run_cli({}, "9").returncode, 2)  # the fake state has no labels: the read fails


if __name__ == "__main__":
    unittest.main()
