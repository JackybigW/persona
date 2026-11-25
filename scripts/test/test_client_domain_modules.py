"""The client domain guard: Swift and Kotlin domain modules held to docs/server/architecture.md "Domain model"
('Client modules').

The rules are client_domain_rules.py's; this runs them.

- AppleDomain scans every Swift target named `*Domain` or `*Adapt` in apple/Packages/*/Package.swift, and
  PersonaDomainKit, with what reads only text: the manifests, the sources, scripts/ci/apple.sh's domain package step,
  and the SPI imports, visitor conformances and dropped results in the rest of apple/. scripts/ci/apple.sh runs it
  again after building each domain package with -emit-symbol-graph (PERSONA_DOMAIN_SYMBOL_GRAPHS, the public API
  rules) and after the device build (PERSONA_APP_BUILD_LOGS, the app's unused domain results).
- AndroidDomain scans every Gradle project android/core/<ctx>/domain, the packages persona.core.<ctx>.adapt and
  android/core/domainkit, and the unused-result flags in android/build-logic. scripts/ci/android.sh's lint block
  runs it.
- RuleTests runs every rule on the planted trees under scripts/test/fixtures/client_domain and asserts the exact
  findings (expected.txt), and that the compliant twins there, the "Domain model" examples included, have none.

With no domain module in the repository the scan classes print "no domain modules yet" and pass, after checking that
nothing named like one went unfound. CI's Scripts tests job runs the whole file before merge; Swift's symbol-graph
rules run after merge, in the Apple lane.

    python3 -B scripts/test/test_client_domain_modules.py [AppleDomain|AndroidDomain|RuleTests]
    python3 -B scripts/test/test_client_domain_modules.py --swift-packages   # what apple.sh builds, one per line
    python3 -B scripts/test/test_client_domain_modules.py --planted-graphs   # rewrite the planted graphs (needs Swift)
"""
from collections import Counter
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))
import client_domain_rules as rules  # noqa: E402

ROOT = Path(__file__).resolve().parents[2]
FIXTURES = ROOT / "scripts" / "test" / "fixtures" / "client_domain"
PLANTED = FIXTURES / "planted"
MISSED = FIXTURES / "missed"
GRAPHS = FIXTURES / "graphs"
APP_LOG = FIXTURES / "app-build.log"
EXPECTED = FIXTURES / "expected.txt"
ARCHITECTURE = ROOT / "docs" / "server" / "architecture.md"
# The planted code that breaks nothing: every rule's compliant twin.
COMPLIANT = ("apple/Packages/PersonaDomainKit/", "apple/Packages/PersonaPlanted/", "apple/Persona/PlantedRow.swift",
             "android/core/domainkit/", "android/core/planted/", "android/feature/planted/PlantedUse.kt")


def build_logs(value):
    """The app build logs PERSONA_APP_BUILD_LOGS names: a file, or a folder of *.xcactivitylog and *.log files."""
    if not value:
        return []
    path = Path(value)
    if path.is_dir():
        return sorted(p for p in path.rglob("*") if p.suffix in (".xcactivitylog", ".log"))
    return [path] if path.is_file() else []


def report(label, scan):
    if not scan.modules:
        print(f"{label}: no domain modules yet")
        return
    names = ", ".join(getattr(m, "name", None) or f"{m.kind} {m.ctx}" for m in scan.modules)
    print(f"{label}: {len(scan.modules)} modules ({names}); {scan.inspected} sources and graphs read")


def failure(findings):
    return "\n" + "\n".join(f.message() for f in findings)


class AppleDomain(unittest.TestCase):
    """The Swift domain targets in apple/Packages, and what the rest of apple/ does with them."""

    def test_domain_targets_follow_the_rules(self):
        graphs = os.environ.get("PERSONA_DOMAIN_SYMBOL_GRAPHS")
        if graphs and not Path(graphs).is_dir():
            self.fail(f"PERSONA_DOMAIN_SYMBOL_GRAPHS={graphs} is not a folder: run the domain package step first")
        logs = build_logs(os.environ.get("PERSONA_APP_BUILD_LOGS"))
        if os.environ.get("PERSONA_APP_BUILD_LOGS") and not logs:
            self.fail(f"PERSONA_APP_BUILD_LOGS={os.environ['PERSONA_APP_BUILD_LOGS']} holds no build log")
        apple_sh = (ROOT / "scripts" / "ci" / "apple.sh").read_text(encoding="utf-8")
        scan = rules.scan_apple(ROOT, apple_sh=apple_sh, graphs_dir=graphs, build_logs=logs)
        report("AppleDomain", scan)
        if scan.modules and not graphs:
            print("AppleDomain: the public API rules need the symbol graphs scripts/ci/apple.sh writes")
        self.assertEqual([], scan.findings, failure(scan.findings))

    def test_kit_exceptions_name_the_kits_declarations(self):
        """An exception the kit no longer needs fails: each names a declaration of the shape its rule refuses."""
        targets, _ = rules.swift_targets(ROOT)
        kit = next((t for t in targets if t.role == "kit"), None)
        if kit is None:
            print("AppleDomain: no PersonaDomainKit yet, so no kit exception to check")
            return
        text = rules.strip("\n".join(p.read_text(encoding="utf-8")
                                     for p in sorted((ROOT / kit.sources).rglob("*.swift"))), "swift")
        shapes = {"swift-public-enum": r"\bpublic\s+(?:indirect\s+)?enum\s+{name}\b",
                  "swift-generic": r"\b(?:enum|struct|class)\s+{name}\s*<"}
        for rule, target, path in rules.KIT_EXCEPTIONS:
            shape = shapes[rule].format(name=re.escape(path.split(".")[-1]))
            self.assertTrue(re.search(shape, text),
                            f"KIT_EXCEPTIONS keeps ({rule}, {path}), which {target} no longer needs: drop the line")


class AndroidDomain(unittest.TestCase):
    """The Kotlin domain modules, the kit and the adapt packages under android/core, and what the rest of android/
    does with them."""

    def test_domain_modules_follow_the_rules(self):
        scan = rules.scan_android(ROOT)
        report("AndroidDomain", scan)
        self.assertEqual([], scan.findings, failure(scan.findings))


def planted_findings():
    """Every finding on the planted trees, as expected.txt writes them."""
    lines = []
    for tree, scan in (
        ("planted", rules.scan_apple(PLANTED, graphs_dir=GRAPHS, build_logs=[APP_LOG])),
        ("planted", rules.scan_android(PLANTED, build_logic=False)),
        ("missed", rules.scan_apple(MISSED)),
        ("missed", rules.scan_android(MISSED, build_logic=False)),
    ):
        lines.extend(f"{tree} {f.rule} {f.path}:{f.line} {f.what}" for f in scan.findings)
    return sorted(lines)


def doc_section(title):
    """The text of an H3 of the "Domain model" H2 in docs/server/architecture.md."""
    text = ARCHITECTURE.read_text(encoding="utf-8")
    model = re.search(r"(?m)^## Domain model\n(.*?)(?=^## )", text, re.S)
    if not model:
        return None
    section = re.search(r"(?m)^### " + re.escape(title) + r"\n(.*?)(?=^### |\Z)", model.group(1), re.S)
    return section.group(1) if section else None


class RuleTests(unittest.TestCase):
    """Every rule on planted inputs: each fails once with its message, and its compliant twin passes."""

    maxDiff = None

    def test_planted_findings_are_exact(self):
        lines = EXPECTED.read_text(encoding="utf-8").splitlines()
        expected = [line for line in lines if line and not line.startswith("#")]
        actual = planted_findings()
        missing = sorted((Counter(expected) - Counter(actual)).elements())
        extra = sorted((Counter(actual) - Counter(expected)).elements())
        self.assertEqual(([], []), (missing, extra),
                         "planted findings differ from scripts/test/fixtures/client_domain/expected.txt\n"
                         + "\n".join(["missing:"] + missing + ["unexpected:"] + extra))

    def test_compliant_twins_have_no_findings(self):
        planted = [line for line in planted_findings() if line.startswith("planted ")]
        on_twins = [line for line in planted if line.split(" ")[2].startswith(COMPLIANT)]
        self.assertEqual([], on_twins)

    def test_the_named_twins_are_compliant_planted_code(self):
        """The cases a rule must let through, each in the planted code test_compliant_twins_have_no_findings holds."""
        swift = "apple/Packages/PersonaPlanted/Sources/PersonaPlantedDomain/Item.swift"
        kotlin = "android/core/planted/domain/src/main/kotlin/persona/core/planted/domain/Item.kt"
        twins = {
            "a computed id": (swift, "public var id: ItemID { identity }"),
            "a public static let value": (swift, "public static let routine = Urgency(.routine)"),
            "an @_spi(Boundary) member": (swift, "@_spi(Boundary) public var wire: String { item }"),
            "a banned word inside a comment": (swift, "prints (print(...))"),
            "a computed Boolean predicate": (swift, "public var breaksQuiet: Bool { urgency == .urgent }"),
            "an internal sealed interface inside a domain class": (kotlin, "internal sealed interface Case {"),
            "a computed Boolean property": (kotlin, "public val breaksQuiet: Boolean get() = urgency == Case.URGENT"),
            "a companion parse": (kotlin, "public fun parse(raw: String): Outcome<ItemId, PlantedFail> ="),
            "a @DomainBoundary wire()": (kotlin, "@DomainBoundary public fun wire(): String = item"),
        }
        for case, (path, snippet) in twins.items():
            self.assertTrue(path.startswith(COMPLIANT), case)
            self.assertIn(snippet, (PLANTED / path).read_text(encoding="utf-8"), case)

    def test_every_rule_fails_on_a_planted_input_and_names_a_twin(self):
        failed = {line.split(" ")[1] for line in planted_findings()}
        self.assertEqual(sorted(rules.RULES), sorted(failed))
        for rule, (why, fix, good) in rules.RULES.items():
            path = good.split(" (", 1)[0]
            self.assertTrue((ROOT / path).exists(), f"{rule}'s GOOD names {path}, which does not exist")
            self.assertTrue(path.startswith(rules.GOOD), f"{rule}'s GOOD names {path}, outside the planted tree")

    def test_messages_say_rule_why_fix_good_and_the_doc(self):
        finding = rules.Finding("apple/Packages/PersonaX/Sources/PersonaXDomain/X.swift", 3, "swift-accessor",
                                "X.doneAt hands out Instant")
        message = finding.message()
        self.assertTrue(message.startswith("apple/Packages/PersonaX/Sources/PersonaXDomain/X.swift:3: RULE "
                                           "swift-accessor X.doneAt hands out Instant. WHY "), message)
        for part in (". FIX ", ". GOOD ", ". see docs/server/architecture.md \"Domain model\" ('Client modules')"):
            self.assertIn(part, message)
        self.assertTrue(message.endswith(rules.DOC))

    def test_modules_are_found_by_name(self):
        apple = rules.scan_apple(PLANTED, graphs_dir=GRAPHS)
        self.assertEqual(["PersonaBrokenDomain", "PersonaDomainKit", "PersonaDomainKitAdapt", "PersonaPlantedAdapt",
                          "PersonaPlantedDomain"], sorted(t.name for t in apple.modules))
        android = rules.scan_android(PLANTED, build_logic=False)
        self.assertEqual(["adapt domainkit", "adapt planted", "domain broken", "domain planted", "kit domainkit"],
                         sorted(f"{m.kind} {m.ctx}" for m in android.modules))

    def test_a_missed_module_fails_instead_of_passing_as_none(self):
        apple, android = rules.scan_apple(MISSED), rules.scan_android(MISSED, build_logic=False)
        self.assertEqual(([], []), (apple.modules, android.modules))
        self.assertEqual({"swift-discovery"}, {f.rule for f in apple.findings})
        self.assertEqual({"kotlin-discovery"}, {f.rule for f in android.findings})

    def test_no_modules_and_nothing_named_like_one_pass(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "apple/Packages/PersonaKit/Sources/PersonaKit").mkdir(parents=True)
            (root / "apple/Packages/PersonaKit/Package.swift").write_text(
                '// swift-tools-version: 6.2\nimport PackageDescription\n'
                'let package = Package(name: "PersonaKit", targets: [.target(name: "PersonaKit")])\n')
            (root / "android/core/domain/src/main/kotlin/persona/core/domain/inbox").mkdir(parents=True)
            (root / "android/settings.gradle.kts").write_text('include(":app", ":core:domain")\n')
            # The existing :core:domain module (persona.core.domain.*) is not a domain module by these names.
            (root / "android/core/domain/src/main/kotlin/persona/core/domain/inbox/InboxBar.kt").write_text(
                "package persona.core.domain.inbox\n\ndata class InboxBar(var query: String)\n")
            apple, android = rules.scan_apple(root), rules.scan_android(root, build_logic=False)
            self.assertEqual(([], [], [], []), (apple.modules, apple.findings, android.modules, android.findings))

    def test_comments_and_string_literals_are_blanked(self):
        swift = ('let a = "print(\\(b.count) /* x */)" // fatalError(\n'
                 '/* outer /* inner */ print( */ let c = #"Mirror("#\n')
        stripped = rules.strip(swift, "swift")
        self.assertNotIn("print", stripped)
        self.assertNotIn("fatalError", stripped)
        self.assertNotIn("Mirror", stripped)
        self.assertIn("b.count", stripped)  # interpolated code stays code
        self.assertEqual(swift.count("\n"), stripped.count("\n"))
        kotlin = 'val a = "println(${b.size})" + """var ${c}""" + \'"\' // throw\n/* /* var */ println( */ val d = 1\n'
        stripped = rules.strip(kotlin, "kotlin")
        self.assertNotIn("println", stripped)
        self.assertNotIn("throw", stripped)
        self.assertEqual(["val", "val"], re.findall(r"\b(?:val|var)\b", stripped))
        self.assertIn("b.size", stripped)
        self.assertEqual(rules.strip(kotlin, "kotlin", keep_strings=True).count("println"), 1)

    def test_apple_sh_runs_the_domain_package_step(self):
        real = (ROOT / "scripts" / "ci" / "apple.sh").read_text(encoding="utf-8")
        self.assertEqual([], rules.swift_testable_findings(real, ["apple/Packages/PersonaPlanted"]))
        broken = "\n".join(line.replace("--disable-testable-imports", "") for line in real.splitlines())
        broken += '\n  ci_run x 60 xcrun swift test --package-path "$SOURCE/apple/Packages/PersonaPlanted"\n'
        whats = sorted(f.what for f in rules.swift_testable_findings(broken, ["apple/Packages/PersonaPlanted"]))
        self.assertEqual(["tests apple/Packages/PersonaPlanted without --disable-testable-imports",
                          "tests the domain packages without --disable-testable-imports"], whats)
        bare = rules.swift_testable_findings("#!/bin/bash\n", [])
        self.assertEqual(3, len(bare), bare)

    def test_build_logic_keeps_the_unused_result_errors(self):
        logic = ROOT / "android/build-logic/convention/src/main/kotlin"
        names = ("persona/buildlogic/ReturnValues.kt", "persona.android.application.gradle.kts",
                 "persona.android.library.gradle.kts", "persona.jvm.library.gradle.kts")
        files = {f"android/build-logic/convention/src/main/kotlin/{n}": (logic / n).read_text(encoding="utf-8")
                 for n in names}
        self.assertEqual([], rules.kotlin_build_logic_findings(files))
        path = "android/build-logic/convention/src/main/kotlin/persona/buildlogic/ReturnValues.kt"
        lost = dict(files)
        lost[path] = files[path].replace('"-Xwarning-level=RETURN_VALUE_NOT_USED:error",', "").replace(
            "private val stillWarnings: Map<String, List<String>> = mapOf(",
            'private val stillWarnings: Map<String, List<String>> = mapOf(\n    ":core:inbox:domain" to listOf("x"),')
        jvm = "android/build-logic/convention/src/main/kotlin/persona.jvm.library.gradle.kts"
        lost[jvm] = files[jvm].replace("configureReturnValueChecks()", "")
        whats = sorted(f"{f.path.rsplit('/', 1)[1]}: {f.what}" for f in rules.kotlin_build_logic_findings(lost))
        self.assertEqual(["ReturnValues.kt: keeps :core:inbox:domain's dropped results as warnings",
                          "ReturnValues.kt: no longer adds -Xwarning-level=RETURN_VALUE_NOT_USED:error",
                          "persona.jvm.library.gradle.kts: does not call configureReturnValueChecks()"], whats)

    def test_kit_exceptions_are_needed_by_the_planted_kit(self):
        kit = next(t for t in rules.swift_targets(PLANTED)[0] if t.role == "kit")
        graphs = [json.loads((GRAPHS / "PersonaDomainKit.symbols.json").read_text(encoding="utf-8"))]
        texts = {p: p.read_text(encoding="utf-8") for p in sorted((PLANTED / kit.sources).rglob("*.swift"))}
        saved = dict(rules.KIT_EXCEPTIONS)
        try:
            for key in saved:
                rules.KIT_EXCEPTIONS.clear()
                rules.KIT_EXCEPTIONS.update({k: v for k, v in saved.items() if k != key})
                found = rules.swift_graph_findings(kit.name, graphs)
                for path, text in texts.items():
                    found += rules.swift_source_findings(path.name, text, kit)
                self.assertIn(key[0], {f.rule for f in found if key[2] in f.what}, f"{key} is not needed")
        finally:
            rules.KIT_EXCEPTIONS.clear()
            rules.KIT_EXCEPTIONS.update(saved)

    def test_planted_graphs_match_the_planted_sources(self):
        """A planted Swift edit without --planted-graphs leaves graphs that point at other lines."""
        for graph in sorted(GRAPHS.glob("*.symbols.json")):
            for symbol in json.loads(graph.read_text(encoding="utf-8"))["symbols"]:
                path, line = rules._graph_location(symbol)
                if not line:
                    continue
                source = (PLANTED / path).read_text(encoding="utf-8").splitlines()
                name = symbol["pathComponents"][-1].split("(", 1)[0]
                self.assertIn(name if name != "init" else "init", source[line - 1],
                              f"{graph.name}: {'.'.join(symbol['pathComponents'])} is not on {path}:{line}; "
                              "run test_client_domain_modules.py --planted-graphs")

    def test_domain_model_client_examples_are_compliant_twins(self):
        section = doc_section("Client modules")
        self.assertIsNotNone(section, "docs/server/architecture.md has no 'Client modules' under 'Domain model'")
        swift = re.findall(r"```swift\n(.*?)```", section, re.S)
        kotlin = re.findall(r"```kotlin\n(.*?)```", section, re.S)
        self.assertTrue(swift and kotlin, "'Client modules' shows a Swift and a Kotlin example")
        twins = {p: p.read_text(encoding="utf-8") for p in sorted(PLANTED.rglob("*"))
                 if p.is_file() and p.relative_to(PLANTED).as_posix().startswith(COMPLIANT)}
        for code in swift + kotlin:
            self.assertTrue(any(code in text for text in twins.values()),
                            "this 'Client modules' example is in no compliant planted file; copy it from one:\n" + code)
        target = rules.SwiftTarget("apple/Packages/PersonaDoc", "PersonaDocDomain", "domain",
                                   "apple/Packages/PersonaDoc/Sources/PersonaDocDomain", "", 0, [], "")
        for number, code in enumerate(swift, 1):
            self.assertEqual([], rules.swift_source_findings(f"swift example {number}", code, target))
        for number, code in enumerate(kotlin, 1):
            if "plugins {" in code:
                self.assertEqual([], rules.kotlin_build_findings(f"kotlin example {number}", code, False, set()))
            else:
                self.assertEqual([], rules.kotlin_source_findings(f"kotlin example {number}", code, "domain"))

def swift_packages():
    targets, _ = rules.swift_targets(ROOT)
    return sorted({t.package for t in targets if t.role in ("domain", "kit", "adapt", "kit-adapt")})


def planted_graphs():
    """Build the planted Swift packages with the symbol-graph flags apple.sh passes and keep, for each Domain target
    and the kit, what the rules read, with paths from apple/Packages."""
    targets, _ = rules.swift_targets(PLANTED)
    cores = {t.name for t in targets if t.role in ("domain", "kit")}
    packages = sorted({t.package for t in targets if t.role == "domain"})
    with tempfile.TemporaryDirectory(prefix="planted-graphs-") as tmp:
        copy = Path(tmp) / "apple" / "Packages"
        shutil.copytree(PLANTED / "apple" / "Packages", copy)
        out = Path(tmp) / "graphs"
        out.mkdir()
        for package in packages:
            name = package.rsplit("/", 1)[1]
            subprocess.run(["xcrun", "swift", "build", "--package-path", str(copy / name), "--scratch-path",
                            str(Path(tmp) / "scratch" / name), "-Xswiftc", "-emit-symbol-graph", "-Xswiftc",
                            "-emit-symbol-graph-dir", "-Xswiftc", str(out), "-Xswiftc",
                            "-symbol-graph-minimum-access-level", "-Xswiftc", "public"], check=True, timeout=600)
        for old in GRAPHS.glob("*.symbols.json"):
            old.unlink()
        GRAPHS.mkdir(exist_ok=True)
        for graph in sorted(out.glob("*.symbols.json")):
            if graph.name.split(".")[0].split("@")[0] not in cores:
                continue
            data = trim(json.loads(graph.read_text(encoding="utf-8")))
            (GRAPHS / graph.name).write_text(compact(data), encoding="utf-8")
            print(f"wrote {(GRAPHS / graph.name).relative_to(ROOT)}")


def compact(graph):
    """The graph as JSON with one symbol or relationship per line, so a change to the planted code reads as a diff."""
    def rows(items):
        return ",\n".join("  " + json.dumps(item, sort_keys=True) for item in items)
    return ('{"module": ' + json.dumps(graph["module"]) + ',\n "relationships": [\n' + rows(graph["relationships"])
            + '\n ],\n "symbols": [\n' + rows(graph["symbols"]) + "\n ]\n}\n")


def trim(graph):
    """What swift_graph_findings reads, with each location as file:///planted/apple/Packages/..."""
    def fragments(items):
        return [{"kind": f["kind"], "spelling": f["spelling"]} for f in items or []]

    symbols = []
    for symbol in graph.get("symbols", []):
        kept = {k: symbol[k] for k in ("kind", "identifier", "pathComponents", "accessLevel") if k in symbol}
        kept["kind"] = {"identifier": symbol["kind"]["identifier"]}
        kept["identifier"] = {"precise": symbol["identifier"]["precise"]}
        kept["declarationFragments"] = fragments(symbol.get("declarationFragments"))
        if "functionSignature" in symbol:
            signature = symbol["functionSignature"]
            kept["functionSignature"] = {
                "parameters": [{"name": p.get("name", ""),
                                "declarationFragments": fragments(p.get("declarationFragments"))}
                               for p in signature.get("parameters") or []],
                "returns": fragments(signature.get("returns")),
            }
        if symbol.get("swiftGenerics", {}).get("parameters"):
            kept["swiftGenerics"] = {"parameters": [{"name": p["name"], "depth": p["depth"]}
                                                    for p in symbol["swiftGenerics"]["parameters"]]}
        if "location" in symbol:
            uri = symbol["location"]["uri"].rsplit("/apple/Packages/", 1)[-1]
            kept["location"] = {"uri": "file:///planted/apple/Packages/" + uri,
                                "position": {"line": symbol["location"]["position"]["line"]}}
        symbols.append(kept)
    relationships = [{"kind": r["kind"], "source": r["source"], "target": r["target"]}
                     for r in graph.get("relationships", []) if r.get("kind") == "memberOf"]
    return {"module": {"name": graph["module"]["name"]},
            "symbols": sorted(symbols, key=lambda s: s["identifier"]["precise"]),
            "relationships": sorted(relationships, key=lambda r: (r["source"], r["target"]))}


if __name__ == "__main__":
    if sys.argv[1:] == ["--swift-packages"]:
        for package in swift_packages():
            print(package)
    elif sys.argv[1:] == ["--planted-graphs"]:
        planted_graphs()
    else:
        unittest.main()
