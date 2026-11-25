"""The client domain rules: Swift and Kotlin domain modules held to docs/server/architecture.md "Domain model"
('Client modules').

The modules are found by their names, never through a list: a Swift target named `*Domain` or `*Adapt` in
`apple/Packages/*/Package.swift` (and `PersonaDomainKit`), a Gradle project `android/core/<ctx>/domain`, the package
`persona.core.<ctx>.adapt` under `android/core/<ctx>/src/main/kotlin`, and `android/core/domainkit`. The existing
`:core:domain` module (package `persona.core.domain`) is not one. A directory, source file or import that is named like
a domain module the manifests do not declare is a finding (`swift-discovery`, `kotlin-discovery`), so a module the
scan missed fails instead of passing as "no modules".

Every function here takes text (or a symbol graph already read from JSON) and returns findings; it reads no file and
keeps no state. Walking the tree is `scan_apple` and `scan_android`, at the end, and the suites in
test_client_domain_modules.py run them. Comments and string literals are blanked before a rule reads source, so a
banned word in a comment or a string is never a finding.

A finding prints as one line: `<path>:<line>: RULE <id> <what>. WHY ... FIX ... GOOD ... see docs/server/architecture.md
"Domain model" ('Client modules')`. GOOD names a compliant twin under scripts/test/fixtures/client_domain/planted, which
the guard's own tests hold at no findings. There is no ignore marker.
"""
from bisect import bisect_right
from collections import namedtuple
import gzip
import json
import os
from pathlib import Path
import re

DOC = "see docs/server/architecture.md \"Domain model\" ('Client modules')"
GOOD = "scripts/test/fixtures/client_domain/planted/"
_SWIFT_GOOD = GOOD + "apple/Packages/PersonaPlanted/"
_KOTLIN_GOOD = GOOD + "android/core/planted/"


class Finding(namedtuple("Finding", "path line rule what")):
    """One broken rule at one place: a repository-relative path, a 1-based line (0 when the place is a whole file)."""

    def message(self):
        why, fix, good = RULES[self.rule]
        return f"{self.path}:{self.line}: RULE {self.rule} {self.what}. WHY {why}. FIX {fix}. GOOD {good}. {DOC}"


# Each rule: why it holds, the change that fixes a finding, and the compliant twin that shows it done.
RULES = {
    "swift-discovery": (
        "the guard finds domain targets only by their names in a package manifest, and one it cannot see is checked by "
        "nothing",
        "declare the target with `.target(name: \"Persona<Ctx>Domain\")` (or `...Adapt`) in its package's "
        "Package.swift, with its sources in Sources/<name>, or rename what only looks like one",
        _SWIFT_GOOD + "Package.swift"),
    "swift-manifest": (
        "a Domain target is a functional core: it links only the kit and shared kernels, and the compiler settings "
        "stop Foundation, concurrency and dropped results from reaching it",
        "depend only on PersonaDomainKit and shared-kernel *Domain targets, and pass the core settings: "
        ".swiftLanguageMode(.v6), .defaultIsolation(nil), the upcoming features InternalImportsByDefault, "
        "MemberImportVisibility and ExistentialAny, and .unsafeFlags([\"-warnings-as-errors\", \"-Xfrontend\", "
        "\"-disable-implicit-concurrency-module-import\", \"-Xfrontend\", "
        "\"-disable-implicit-string-processing-module-import\"])",
        _SWIFT_GOOD + "Package.swift"),
    "swift-testable": (
        "domain tests are black-box and the public API rules read the symbol graph scripts/ci/apple.sh emits; a "
        "package built another way skips both",
        "give the package a Persona<Ctx>DomainTests target, and keep the domain package step in scripts/ci/apple.sh: "
        "`swift test --disable-testable-imports` with -emit-symbol-graph for every package "
        "test_client_domain_modules.py --swift-packages lists, then AppleDomain",
        _SWIFT_GOOD + "Package.swift"),
    "swift-import": (
        "a core computes only from its arguments, so it imports no platform module",
        "import only PersonaDomainKit and shared-kernel *Domain modules, and move what needs another module into the "
        "*Adapt target",
        _SWIFT_GOOD + "Sources/PersonaPlantedDomain/Item.swift"),
    "swift-banned-call": (
        "a core takes time, randomness and output from its caller and stops only through an outcome",
        "take the value as a parameter (Now, an id made by the adapter), return an Outcome, and log in the application",
        _SWIFT_GOOD + "Sources/PersonaPlantedDomain/Item.swift (snoozed)"),
    "swift-throws": (
        "an expected failure is a value the caller has to read, and a thrown Error is neither typed nor visited",
        "return Outcome<Value, <Ctx>Fail> and read the failure through FailVisitor",
        _SWIFT_GOOD + "Sources/PersonaPlantedDomain/Item.swift (PlantedFail)"),
    "swift-conformance": (
        "codecs, descriptions and raw values read a value's state from outside, around its behaviour",
        "drop the conformance; the *Adapt target converts to the wire type and back",
        _SWIFT_GOOD + "Sources/PersonaPlantedAdapt/ItemWire.swift"),
    "swift-public-enum": (
        "a public enum lets the application switch on a domain value instead of asking it",
        "make it a struct over a private enum with `public static let` values and accept(_ v: inout some XVisitor)",
        _SWIFT_GOOD + "Sources/PersonaPlantedDomain/Item.swift (Urgency)"),
    "swift-mutating": (
        "a transition returns the next value, so an earlier value never changes under its holder",
        "return the next value from a non-mutating method; `mutating` stays only on a visitor protocol's requirements",
        _SWIFT_GOOD + "Sources/PersonaPlantedDomain/Item.swift (snoozed)"),
    "swift-stored-public": (
        "a stored public property, or a setter, is a getter or setter on the value's state",
        "make it `private let`, and add the behaviour the caller needs (a predicate, a transition, accept)",
        _SWIFT_GOOD + "Sources/PersonaPlantedDomain/Item.swift (needsOwner)"),
    "swift-primitive-api": (
        "a String or Int in the public API is a concept without a type, and every caller decides on it",
        "take and return value types; raw input enters only through a static parse, and a Bool only as a computed "
        "predicate or a function's result",
        _SWIFT_GOOD + "Sources/PersonaPlantedDomain/Item.swift (ItemID.parse)"),
    "swift-accessor": (
        "a member that hands state out lets the caller decide what the value should decide",
        "replace it with the behaviour the caller needs: a Bool predicate, a transition returning Self or Outcome, or "
        "accept(_ v: inout some XVisitor); `id` stays",
        _SWIFT_GOOD + "Sources/PersonaPlantedDomain/Item.swift (needsOwner)"),
    "swift-generic": (
        "a generic domain type carries a caller's object inside the core, where the core can neither check nor own it",
        "use a concrete value type, or the kit's Slot for the caller's place",
        _SWIFT_GOOD + "Sources/PersonaPlantedDomain/Item.swift"),
    "swift-public-evolve": (
        "applying an event outside its decider can put a value in a state no rule allowed",
        "keep evolve internal and expose decide",
        _SWIFT_GOOD + "Sources/PersonaPlantedDomain/Item.swift (snoozed)"),
    "swift-factory": (
        "values are built only through constructors named for how they are built, with at most four inputs",
        "construct through a static parse (raw input), new (domain values) or restore (stored state) on a private "
        "init, with a staged builder (newBuilder, restoreBuilder) above four inputs",
        _SWIFT_GOOD + "Sources/PersonaPlantedDomain/Item.swift (Item.new)"),
    "swift-spi-import": (
        "the boundary conversions are for adapters only, and only the kit's adapter reads the clock",
        "convert in a *Adapt target, which alone writes `@_spi(Boundary) import`; `@_spi(Clock) import` belongs to "
        "PersonaDomainKitAdapt",
        _SWIFT_GOOD + "Sources/PersonaPlantedAdapt/ItemWire.swift"),
    "swift-visitor-impl": (
        "code that reads a domain value through a visitor converts it, which is the adapter's job",
        "implement the visitor in the *Adapt target (or the domain tests) and call the behaviour the app needs",
        _SWIFT_GOOD + "Sources/PersonaPlantedAdapt/ItemWire.swift"),
    "swift-app-discard": (
        "a domain transition returns the next value or a failure, and dropping it loses the change or the failure",
        "use the result: keep the next value, or read the outcome",
        GOOD + "apple/Persona/PlantedRow.swift"),
    "kotlin-discovery": (
        "the guard finds domain modules only by their names in settings.gradle.kts and their source paths, and one it "
        "cannot see is checked by nothing",
        "include the module as \":core:<ctx>:domain\" in android/settings.gradle.kts with its package "
        "persona.core.<ctx>.domain, keep adapters in android/core/<ctx>/src/main/kotlin/persona/core/<ctx>/adapt, or "
        "rename what only looks like one",
        _KOTLIN_GOOD + "domain/build.gradle.kts"),
    "kotlin-build-flags": (
        "the compiler holds the module to explicit visibility, warnings as errors and checked results, and the "
        "Android-wide flags make a dropped domain result an error in every module",
        "apply persona.jvm.library, call explicitApi(), set allWarningsAsErrors to true and add "
        "-Xreturn-value-checker=full; keep -Xreturn-value-checker=check and "
        "-Xwarning-level=RETURN_VALUE_NOT_USED:error in android/build-logic for every module",
        _KOTLIN_GOOD + "domain/build.gradle.kts"),
    "kotlin-module": (
        "a domain module is a functional core that links only the kit and shared kernels",
        "depend only on project(\":core:domainkit\") and shared-kernel domain modules; move the rest to "
        ":core:<ctx>'s adapt package",
        _KOTLIN_GOOD + "domain/build.gradle.kts"),
    "kotlin-var": (
        "a core value never changes in place",
        "use val and return the next value from a transition",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt (snoozed)"),
    "kotlin-ctor-property": (
        "a public constructor property is a getter on the value's state",
        "make it private val (or internal val for the module's peers) and add the behaviour the caller needs",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt"),
    "kotlin-public-constructor": (
        "a public constructor builds a value without the checks of parse, new or restore",
        "declare `private constructor` and construct through companion parse, new or restore",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt (ItemId)"),
    "kotlin-data-class": (
        "a data class exposes its state through componentN, copy and toString, around its constructors",
        "write a class with a private constructor and private vals, with equals and hashCode where identity matters",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt"),
    "kotlin-public-sealed": (
        "a public sealed type, enum class or data object lets the application `when` on a domain value",
        "make it a class with a private constructor, companion values and accept(visitor); a sealed type or enum "
        "class stays internal or private inside it",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt (Urgency)"),
    "kotlin-primitive-api": (
        "a String or Int in the public API is a concept without a type, and every caller decides on it",
        "take and return value types; raw input enters only through companion parse, and a Boolean only as a "
        "computed predicate or a function's result",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt (ItemId.parse)"),
    "kotlin-accessor": (
        "a member that hands state out lets the caller decide what the value should decide",
        "replace it with the behaviour the caller needs: a computed Boolean, a transition returning the class or its "
        "Outcome, or accept(visitor); id() and a @DomainBoundary wire() stay",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt (needsOwner)"),
    "kotlin-import": (
        "a core computes only from its arguments, so it uses no platform or library API",
        "import only persona.core.domainkit, shared-kernel domain packages and its own package; move the rest to the "
        "adapt package",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt"),
    "kotlin-banned-call": (
        "a core takes time, ids and output from its caller",
        "take Now and ids as parameters from the adapter, and log in the application",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt (snoozed)"),
    "kotlin-serializable": (
        "a serializable core type is a stored or wire shape, which lives in the store or adapt package",
        "keep the @Serializable record in :core:<ctx>'s store or adapt package and convert in the adapter",
        _KOTLIN_GOOD + "src/main/kotlin/persona/core/planted/adapt/ItemWire.kt"),
    "kotlin-optin": (
        "the boundary conversions are for adapters only, and only the kit's clock adapter reads the clock",
        "opt in to DomainBoundary only in persona.core.<ctx>.adapt; ClockRead only in persona.core.domainkit.adapt; "
        "never through a compiler flag",
        _KOTLIN_GOOD + "src/main/kotlin/persona/core/planted/adapt/ItemWire.kt"),
    "kotlin-throw": (
        "an expected failure is a value, and a programming error is a Defect",
        "return Outcome for an expected failure and `throw Defect(...)` for a programming error; no !!, error(), "
        "require() or check()",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt (PlantedFail)"),
    "kotlin-public-evolve": (
        "applying an event outside its decider can put a value in a state no rule allowed",
        "keep evolve internal and expose decide",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt (snoozed)"),
    "kotlin-visitor-impl": (
        "code that reads a domain value through a visitor converts it, which is the adapter's job",
        "implement the visitor in persona.core.<ctx>.adapt (or a test) and call the behaviour the app needs",
        _KOTLIN_GOOD + "src/main/kotlin/persona/core/planted/adapt/ItemWire.kt"),
    "kotlin-factory": (
        "values are built only through constructors named for how they are built, with at most four inputs",
        "name a companion factory parse, new, restore, newBuilder or restoreBuilder (a closed set's value keeps its "
        "own name), and use a staged builder above four inputs",
        _KOTLIN_GOOD + "domain/src/main/kotlin/persona/core/planted/domain/Item.kt (Item.new)"),
}

# Named, reviewed exceptions for the kit's documented API, by rule, target and declaration path. A line nothing needs
# fails once the kit exists (test_client_domain_modules.py), so the list only holds what the kit declares.
KIT_EXCEPTIONS = {
    ("swift-public-enum", "PersonaDomainKit", "Outcome"): "the unpack: the one public enum, switched on at the edge",
    ("swift-generic", "PersonaDomainKit", "Outcome"): "carries every core's value and failure family",
}

# A factory named for how it builds instead of parse, new, restore, newBuilder or restoreBuilder, or a constructor in
# disguise. A closed set's value keeps its own name (ItemCommand.markDone(by:)).
BANNED_FACTORY = re.compile(
    r"^(?:make|create|build|init|from|to|get|set|reconstruct|must|of|valueOf|invoke)(?:[A-Z_]|$)")
MAX_PARAMS = 4
SWIFT_PRIMITIVES = frozenset((
    "String", "Character", "Substring", "Bool", "Int", "Int8", "Int16", "Int32", "Int64", "Int128", "UInt", "UInt8",
    "UInt16", "UInt32", "UInt64", "UInt128", "Double", "Float", "Float16", "Float80", "CGFloat", "Decimal",
    "TimeInterval", "Date", "UUID", "Data", "URL", "Any", "AnyObject", "AnyHashable"))
KOTLIN_PRIMITIVES = frozenset((
    "String", "CharSequence", "Char", "Boolean", "Byte", "Short", "Int", "Long", "UByte", "UShort", "UInt", "ULong",
    "Float", "Double", "Number", "Any", "ByteArray", "ShortArray", "IntArray", "LongArray", "FloatArray", "DoubleArray",
    "CharArray", "BooleanArray"))
_BUILD_DIRS = frozenset(("build", "DerivedData", "node_modules", "xcuserdata"))
_BUILD_SUFFIXES = (".xcodeproj", ".xcworkspace", ".xcframework", ".xcassets", ".lproj", ".xcresult")


# ---------------------------------------------------------------------------------------------------------------------
# Comments and string literals


class _Stripper:
    """Finds comment and string-content ranges in Swift or Kotlin source. Block comments nest in both languages, and
    interpolated code inside a string (Swift \\(...), Kotlin ${...}) is code again."""

    _SWIFT = re.compile(r"//|/\*|#+\"|\"|[(){}\[\]]")
    _KOTLIN = re.compile(r"//|/\*|\"|'|[(){}\[\]]")
    _BLOCK = re.compile(r"/\*|\*/")

    def __init__(self, text, swift):
        self.text, self.n, self.swift = text, len(text), swift
        self.ranges = []  # (start, end, is_comment), in order and apart
        self._patterns = {}

    def code(self, i, inside=False):
        """Scan code from i. Inside an interpolation, stop after the bracket that closes it; return where it stopped."""
        text, depth = self.text, 0
        pattern = self._SWIFT if self.swift else self._KOTLIN
        while True:
            m = pattern.search(text, i)
            if not m:
                return self.n
            token, start = m.group(), m.start()
            if token == "//":
                end = text.find("\n", start)
                end = self.n if end < 0 else end
                self.ranges.append((start, end, True))
                i = end
            elif token == "/*":
                end = self._block_end(start)
                self.ranges.append((start, end, True))
                i = end
            elif token in "([{":
                depth += 1
                i = m.end()
            elif token in ")]}":
                if inside and depth == 0:
                    return m.end()
                depth -= 1
                i = m.end()
            else:
                i = self._swift_string(start, token) if self.swift else self._kotlin_string(start, token)

    def _block_end(self, start):
        depth, i = 0, start
        while True:
            m = self._BLOCK.search(self.text, i)
            if not m:
                return self.n
            depth += 1 if m.group() == "/*" else -1
            i = m.end()
            if depth == 0:
                return i

    def _pattern(self, key, source):
        if key not in self._patterns:
            self._patterns[key] = re.compile(source)
        return self._patterns[key]

    def _swift_string(self, start, token):
        text, hashes = self.text, len(token) - 1
        i = start + len(token)
        multi = text.startswith("\"\"", i)
        if multi:
            i += 2
        close = ("\"\"\"" if multi else "\"") + "#" * hashes
        escape = "\\" + "#" * hashes
        source = re.escape(close) + "|" + re.escape(escape) + ("" if multi else r"|\n")
        pattern = self._pattern(("s", hashes, multi), source)
        content = i
        while True:
            m = pattern.search(text, i)
            if not m:
                self.ranges.append((content, self.n, False))
                return self.n
            if m.group() == "\n":
                self.ranges.append((content, m.start(), False))
                return m.start()
            if m.group() == close:
                self.ranges.append((content, m.start(), False))
                return m.end()
            after = m.end()
            if after < self.n and text[after] == "(":
                self.ranges.append((content, m.start(), False))
                i = content = self.code(after + 1, inside=True)
            else:
                i = after + 1

    def _kotlin_string(self, start, token):
        text = self.text
        if token == "'":  # a char literal: 'a', '\n', '\'' or '\u0041'
            body = start + 1
            end = text.find("'", body + (2 if text.startswith("\\", body) else 1))
            if end < 0 or end - start > 8 or "\n" in text[start:end]:
                return start + 1
            self.ranges.append((body, end, False))
            return end + 1
        raw = text.startswith("\"\"\"", start)
        i = content = start + (3 if raw else 1)
        pattern = self._pattern(("k", raw), r"\"\"\"+|\$\{" if raw else r"\\.|\"|\$\{|\n")
        while True:
            m = pattern.search(text, i)
            if not m:
                self.ranges.append((content, self.n, False))
                return self.n
            found = m.group()
            if found == "${":
                self.ranges.append((content, m.start(), False))
                i = content = self.code(m.end(), inside=True)
            elif found.startswith("\"\"\""):
                self.ranges.append((content, m.end() - 3, False))
                return m.end()
            elif found == "\"" or found == "\n":
                self.ranges.append((content, m.start(), False))
                return m.end() if found == "\"" else m.start()
            else:
                i = m.end()


def strip(text, language, keep_strings=False):
    """The source with comments, and string contents unless keep_strings, turned to spaces. Newlines stay, so offsets
    and line numbers are the source's."""
    stripper = _Stripper(text, language == "swift")
    stripper.code(0)
    out, last = [], 0
    for start, end, comment in stripper.ranges:
        if keep_strings and not comment:
            continue
        out.append(text[last:start])
        out.append(re.sub(r"[^\n]", " ", text[start:end]))
        last = end
    out.append(text[last:])
    return "".join(out)


# ---------------------------------------------------------------------------------------------------------------------
# Tokens

Token = namedtuple("Token", "v line start end first")
_TOKEN = re.compile(r"`[^`\n]+`|[A-Za-z_][A-Za-z0-9_]*|\d[\w.]*|->|::|\?\.|\?:|!!|\.\.[.<]?|&&|\|\||[=!<>]=|\S")


def tokens(text):
    """Tokens with their 1-based line and whether each is the first on its line."""
    newlines = [m.start() for m in re.finditer("\n", text)]
    out, last_line = [], 0
    for m in _TOKEN.finditer(text):
        line = bisect_right(newlines, m.start()) + 1
        value = m.group()
        if value.startswith("`"):
            value = value[1:-1]
        out.append(Token(value, line, m.start(), m.end(), line != last_line))
        last_line = line
    return out


def _line_of(text, offset):
    return text.count("\n", 0, offset) + 1


def _ident(value):
    return bool(value) and (value[0].isalpha() or value[0] == "_")


# ---------------------------------------------------------------------------------------------------------------------
# Kotlin declarations


class KtDecl:
    """A Kotlin declaration the rules read: a class, interface or object with its members, a function, a property or a
    secondary constructor. Function and accessor bodies are skipped: what they declare is not API."""

    def __init__(self, kind, name, line, mods, annos, parent):
        self.kind, self.name, self.line, self.mods, self.annos, self.parent = kind, name, line, mods, annos, parent
        self.params, self.type, self.receiver, self.supers, self.children = [], None, None, [], []
        self.ctor = None  # a class's primary constructor: None without one, else its visibility ("" for the default)
        self.getter = self.initializer = False
        self.init_first = ""  # a property initializer's first token

    @property
    def public(self):
        own = not (self.mods & {"private", "internal", "protected"})
        return own and (self.parent is None or self.parent.public)

    def walk(self):
        for child in self.children:
            yield child
            yield from child.walk()


KtParam = namedtuple("KtParam", "name mods annos binding type line")
_KT_MODIFIERS = frozenset((
    "public", "private", "internal", "protected", "open", "final", "abstract", "sealed", "data", "enum", "value",
    "inline", "override", "companion", "inner", "annotation", "const", "lateinit", "operator", "infix", "suspend",
    "tailrec", "external", "expect", "actual", "noinline", "crossinline", "vararg"))
_KT_KEYWORDS = frozenset(("class", "interface", "object", "fun", "val", "var", "constructor", "init", "typealias"))
_KT_STARTERS = _KT_MODIFIERS | _KT_KEYWORDS | {"@", "}"}
_KT_ANNOTATION_TARGETS = frozenset(("file", "get", "set", "param", "field", "property", "receiver", "setparam",
                                    "delegate"))


class _KtParser:
    def __init__(self, toks):
        self.t, self.n = toks, len(toks)
        self.package, self.imports = "", []

    def v(self, i):
        return self.t[i].v if i < self.n else ""

    def new_line(self, i):
        return i < self.n and self.t[i].first

    def members(self, owner, i):
        while i < self.n:
            value = self.v(i)
            if value == "}" and owner.kind != "file":
                return i + 1
            if value in (";", "}"):
                i += 1
                continue
            if owner.kind == "file" and value in ("package", "import"):
                j = i + 1
                while j < self.n and not self.t[j].first:
                    j += 1
                name = "".join(tok.v for tok in self.t[i + 1:j])
                if value == "package":
                    self.package = name
                else:
                    self.imports.append((re.sub(r"(?<=[\w*])as[A-Za-z_]\w*$", "", name), self.t[i].line))
                i = j
                continue
            j, annos = self.annotations(i)
            k, mods = self.modifier_run(j)
            keyword = self.v(k)
            if keyword == "fun" and self.v(k + 1) == "interface":
                mods = mods | {"fun"}
                k += 1
                keyword = "interface"
            if keyword not in _KT_KEYWORDS:  # an expression or an enum entry, not a declaration
                i = max(self.skip_expression(j), i + 1)
                continue
            if keyword in ("class", "interface", "object"):
                i = self.klass(owner, k, keyword, mods, annos)
            elif keyword == "fun":
                i = self.fun(owner, k, mods, annos)
            elif keyword in ("val", "var"):
                i = self.prop(owner, k, keyword, mods, annos)
            elif keyword == "constructor":
                decl = KtDecl("constructor", "constructor", self.t[k].line, mods, annos, owner)
                i, decl.params = self.params(k + 1)
                owner.children.append(decl)
                if self.v(i) == ":":
                    i = self.skip_expression(i + 1, stops=("{",))
                if self.v(i) == "{":
                    i = self.skip_brackets(i)
            elif keyword == "init" and self.v(k + 1) == "{":
                i = self.skip_brackets(k + 1)
            else:
                i = self.skip_expression(k + 1)
        return i

    def annotations(self, i):
        names = set()
        while self.v(i) == "@":
            i += 1
            if self.v(i) in _KT_ANNOTATION_TARGETS and self.v(i + 1) == ":":
                i += 2
            if self.v(i) == "[":  # @[A B]
                end = self.skip_brackets(i)
                names.update(tok.v for tok in self.t[i:end] if _ident(tok.v))
                i = end
                continue
            name = ""
            while _ident(self.v(i)):
                name = self.v(i)
                i += 1
                if self.v(i) == "." and _ident(self.v(i + 1)):
                    i += 1
                    continue
                break
            names.add(name)
            if self.v(i) == "<":
                i = self.skip_angles(i)
            if self.v(i) == "(" and i < self.n and self.t[i].start == self.t[i - 1].end:
                i = self.skip_brackets(i)
        return i, names

    def modifier_run(self, i):
        j, mods = i, set()
        while self.v(j) in _KT_MODIFIERS and self.v(j + 1) not in (":", "=", "(", ".", ","):
            mods.add(self.v(j))
            j += 1
            if self.v(j) == "@":
                j, _ = self.annotations(j)
        if self.v(j) in _KT_KEYWORDS:
            return j, mods
        return i, set()

    def skip_brackets(self, i):
        """From an opening (, [ or {, the index after its closing partner."""
        depth = 0
        while i < self.n:
            value = self.v(i)
            if value in ("(", "[", "{"):
                depth += 1
            elif value in (")", "]", "}"):
                depth -= 1
                if depth == 0:
                    return i + 1
            i += 1
        return i

    def skip_angles(self, i):
        depth = 0
        while i < self.n:
            value = self.v(i)
            if value == "<":
                depth += 1
            elif value == ">":
                depth -= 1
                if depth == 0:
                    return i + 1
            elif value in ("{", ";", "=") or (value == ")" and depth <= 1):
                return i
            i += 1
        return i

    def skip_expression(self, i, stops=(), accessors=False):
        """Past an expression or statement: to a ; or } at its depth, a stop token, or a line that starts a
        declaration (with accessors, also one that starts get or set)."""
        depth, begin = 0, i
        while i < self.n:
            value = self.v(i)
            if depth == 0 and i > begin and self.new_line(i):
                if value in _KT_STARTERS:
                    return i
                if accessors and (value in ("get", "set") or (value in ("private", "internal", "protected", "public")
                                                              and self.v(i + 1) in ("get", "set"))):
                    return i
            if depth == 0 and (value in (";", "}") or value in stops):
                return i
            if value in ("(", "[", "{"):
                depth += 1
            elif value in (")", "]", "}"):
                depth -= 1
            i += 1
        return i

    def type_until(self, i, stops):
        """The type's tokens from i to a stop token at depth 0, or to the end of its line."""
        begin, depth = i, 0
        while i < self.n:
            value = self.v(i)
            if depth == 0 and (value in stops or (i > begin and self.new_line(i))):
                break
            if value in ("(", "[", "<"):
                depth += 1
            elif value in (")", "]", ">"):
                if depth == 0:
                    break
                depth -= 1
            elif value == "{" and depth == 0:
                break
            i += 1
        return i, [tok.v for tok in self.t[begin:i]]

    def params(self, i):
        """Parameters of the list opening at i: (index after it, [KtParam])."""
        if self.v(i) != "(":
            return i, []
        end = self.skip_brackets(i)
        chunks, chunk, depth = [], [], 0
        for tok in self.t[i + 1:end - 1]:
            if tok.v in ("(", "[", "{", "<"):
                depth += 1
            elif tok.v in (")", "]", "}", ">"):
                depth -= 1
            if tok.v == "," and depth == 0:
                chunks.append(chunk)
                chunk = []
            else:
                chunk.append(tok)
        if chunk:
            chunks.append(chunk)
        out = []
        for chunk in chunks:
            sub = _KtParser(chunk)
            j, annos = sub.annotations(0)
            mods = set()
            while sub.v(j) in _KT_MODIFIERS:
                mods.add(sub.v(j))
                j += 1
            binding = sub.v(j) if sub.v(j) in ("val", "var") else None
            if binding:
                j += 1
            name = sub.v(j)
            j += 1
            type_tokens = []
            if sub.v(j) == ":":
                j, type_tokens = sub.type_until(j + 1, ("=",))
            out.append(KtParam(name, mods, annos, binding, type_tokens, chunk[0].line))
        return end, out

    def klass(self, owner, k, keyword, mods, annos):
        i = k + 1
        name = ""
        if _ident(self.v(i)) and self.v(i) not in ("constructor",):
            name = self.v(i)
            i += 1
        elif "companion" in mods:
            name = "Companion"
        decl = KtDecl(keyword, name, self.t[k].line, mods, annos, owner)
        owner.children.append(decl)
        if self.v(i) == "<":
            i = self.skip_angles(i)
        j, _ = self.annotations(i)
        visibility = ""
        while self.v(j) in ("private", "internal", "public", "protected"):
            visibility = self.v(j)
            j += 1
        if self.v(j) == "constructor":
            decl.ctor = visibility
            i = j + 1
        if self.v(i) == "(" and not self.new_line(i):
            if decl.ctor is None:
                decl.ctor = ""
            i, decl.params = self.params(i)
        if self.v(i) == ":":
            i = self.supertypes(decl, i + 1)
        if self.v(i) == "where":
            i = self.skip_expression(i + 1, stops=("{",))
        if self.v(i) == "{":
            if "enum" in mods:
                i = self.skip_brackets(i)
            else:
                i = self.members(decl, i + 1)
        return i

    def supertypes(self, decl, i):
        """The names after a class header's colon, up to its body or the end of the declaration."""
        while i < self.n and not (self.new_line(i) and self.v(i) in _KT_STARTERS):
            while _ident(self.v(i)) and self.v(i + 1) == ".":
                i += 2  # a qualified name: the last part names the type
            if not _ident(self.v(i)):
                break
            decl.supers.append(self.v(i))
            i += 1
            if self.v(i) == "<":
                i = self.skip_angles(i)
            if self.v(i) == "(":
                i = self.skip_brackets(i)
            if self.v(i) == "by":
                i = self.skip_expression(i + 1, stops=(",", "{"))
            if self.v(i) != ",":
                break
            i += 1
        return i

    def signature(self, i):
        """The receiver and name before a function's ( or a property's : =: (index at the stop, receiver, name)."""
        begin, depth = i, 0
        while i < self.n:
            value = self.v(i)
            if value == "<":
                depth += 1
            elif value == ">":
                depth -= 1
            elif depth == 0 and (value in ("(", ":", "=", "{", ";", "}") or value == "by"
                                 or (i > begin and self.new_line(i))):
                break
            i += 1
        parts = [tok.v for tok in self.t[begin:i]]
        name, receiver = (parts[-1] if parts else ""), None
        depth = 0
        for index in range(len(parts) - 1, -1, -1):
            if parts[index] == ">":
                depth += 1
            elif parts[index] == "<":
                depth -= 1
            elif parts[index] == "." and depth == 0:
                receiver = parts[:index]
                break
        return i, receiver, name

    def fun(self, owner, k, mods, annos):
        i = k + 1
        if self.v(i) == "<":
            i = self.skip_angles(i)
        i, receiver, name = self.signature(i)
        decl = KtDecl("fun", name, self.t[k].line, mods, annos, owner)
        decl.receiver = receiver
        owner.children.append(decl)
        i, decl.params = self.params(i)
        if self.v(i) == ":":
            i, decl.type = self.type_until(i + 1, ("{", "=", "where"))
        if self.v(i) == "where":
            i = self.skip_expression(i + 1, stops=("{", "="))
        if self.v(i) == "{":
            i = self.skip_brackets(i)
        elif self.v(i) == "=":
            i = self.skip_expression(i + 1)
        return i

    def prop(self, owner, k, keyword, mods, annos):
        i = k + 1
        if self.v(i) == "<":
            i = self.skip_angles(i)
        i, receiver, name = self.signature(i)
        decl = KtDecl(keyword, name, self.t[k].line, mods, annos, owner)
        decl.receiver = receiver
        owner.children.append(decl)
        if self.v(i) == ":":
            i, decl.type = self.type_until(i + 1, ("=", "by", "get", "set", ";", "}"))
        if self.v(i) in ("=", "by"):
            decl.initializer = True
            decl.init_first = self.v(i + 1)
            i = self.skip_expression(i + 1, accessors=True)
        while True:
            j, _ = self.annotations(i)
            while self.v(j) in ("private", "internal", "protected", "public", "override", "inline"):
                j += 1
            accessor = self.v(j) in ("get", "set") and (self.v(j + 1) in ("(", "=", "{", ";", "}")
                                                        or self.new_line(j + 1))
            if not accessor:
                break
            if self.v(j) == "get":
                decl.getter = True
            i = j + 1
            if self.v(i) == "(":
                i = self.skip_brackets(i)
            if self.v(i) == ":":
                i, _ = self.type_until(i + 1, ("{", "="))
            if self.v(i) == "{":
                i = self.skip_brackets(i)
            elif self.v(i) == "=":
                i = self.skip_expression(i + 1, accessors=True)
        return i


def parse_kotlin(stripped):
    """(package, [(import, line)], file declaration) of stripped Kotlin source."""
    parser = _KtParser(tokens(stripped))
    root = KtDecl("file", "", 1, set(), set(), None)
    parser.members(root, 0)
    return parser.package, parser.imports, root


def _type_root(type_tokens):
    """The outermost type's name: Outcome for persona.core.domainkit.Outcome<Item, Fail>?, None for none."""
    root = None
    for value in type_tokens:
        if value in ("<", "("):
            break
        if _ident(value):
            root = value
    return root


def _first_type_argument(type_tokens):
    if "<" not in type_tokens:
        return None
    return _type_root(type_tokens[type_tokens.index("<") + 1:])


# ---------------------------------------------------------------------------------------------------------------------
# Kotlin rules


def _kt_owner(decl):
    """The class a member answers for: its class, the class of its companion object, or an extension's receiver."""
    if decl.receiver:
        return _type_root(decl.receiver)
    parent = decl.parent
    if parent is not None and parent.kind == "object" and "companion" in parent.mods and parent.parent is not None:
        return parent.parent.name
    if parent is not None and parent.kind in ("class", "interface", "object"):
        return parent.name
    return None


def _in_companion(decl):
    return decl.parent is not None and decl.parent.kind == "object" and "companion" in decl.parent.mods


def _returns_own(type_tokens, owner):
    root = _type_root(type_tokens or [])
    return owner is not None and (root == owner or (root == "Outcome" and _first_type_argument(type_tokens) == owner))


def kotlin_source_findings(path, text, module):
    """Findings for one Kotlin source file of a domain module (module "domain") or of the kit (module "kit")."""
    stripped = strip(text, "kotlin")
    _, imports, root = parse_kotlin(stripped)
    found = []
    add = lambda line, rule, what: found.append(Finding(path, line, rule, what))  # noqa: E731
    toks = tokens(stripped)
    for name, line in imports:
        kit = name.startswith("persona.core.domainkit.") and not name.startswith("persona.core.domainkit.adapt.")
        kernel = re.match(r"persona\.core\.[a-z0-9_]+\.domain\.", name) and not name.startswith("persona.core.domain.")
        if not (kit or (kernel and module != "kit")):
            add(line, "kotlin-import", f"imports {name}")
    _kotlin_token_rules(toks, module, add)
    for decl in root.walk():
        _kotlin_decl_rules(decl, add)
    return sorted(set(found))


_KT_FQN = re.compile(r"^(?:java|javax|kotlinx|android|androidx|dalvik|okhttp3|retrofit2)$")
_KT_FQN_KOTLIN = frozenset(("random", "io", "concurrent", "time", "system", "reflect", "coroutines"))
_KT_BANNED_CALLS = {
    ("Instant", "now"): "reads the clock", ("System", None): "reads the platform",
    ("UUID", "randomUUID"): "makes an id", ("Random", None): "draws a random number",
    ("Clock", None): "reads the clock", ("Thread", None): "uses a thread",
    ("LocalDateTime", "now"): "reads the clock", ("LocalDate", "now"): "reads the clock",
    ("ZonedDateTime", "now"): "reads the clock", ("OffsetDateTime", "now"): "reads the clock",
}
_KT_BANNED_BARE = {"println": "prints", "print": "prints", "runBlocking": "blocks a thread", "readLine": "reads input",
                   "Random": "draws a random number"}
_KT_HIDDEN_THROWS = frozenset(("error", "require", "requireNotNull", "check", "checkNotNull", "TODO"))
_KT_CODEC_NAMES = frozenset(("Serializable", "SerialName", "Parcelize", "Parcelable", "Transient"))


def _kotlin_token_rules(toks, module, add):
    n = len(toks)
    head = ""
    for i, tok in enumerate(toks):
        if tok.first:
            head = tok.v
        if head in ("package", "import"):
            continue  # kotlin-import reads these lines
        value, prev = tok.v, (toks[i - 1].v if i else "")
        after = toks[i + 1].v if i + 1 < n else ""
        if value == "var" and prev != ".":
            add(tok.line, "kotlin-var", "declares a var")
        elif value == "throw" and not (after == "Defect" and i + 2 < n and toks[i + 2].v == "("):
            add(tok.line, "kotlin-throw", "throws something other than a Defect")
        elif value == "!!":
            add(tok.line, "kotlin-throw", "uses !!, which throws")
        elif value in _KT_HIDDEN_THROWS and after == "(" and prev not in (".", "fun", "::"):
            add(tok.line, "kotlin-throw", f"calls {value}(), which throws")
        elif value in _KT_BANNED_BARE and after == "(" and prev not in (".", "fun", "::"):
            add(tok.line, "kotlin-banned-call", f"calls {value}() ({_KT_BANNED_BARE[value]})")
        elif prev != "." and i + 2 < n and toks[i + 1].v == "." and (value, toks[i + 2].v) in _KT_BANNED_CALLS:
            member = toks[i + 2].v
            add(tok.line, "kotlin-banned-call", f"calls {value}.{member} ({_KT_BANNED_CALLS[value, member]})")
        elif prev != "." and after == "." and (value, None) in _KT_BANNED_CALLS:
            add(tok.line, "kotlin-banned-call", f"uses {value} ({_KT_BANNED_CALLS[value, None]})")
        elif value in _KT_CODEC_NAMES and prev in ("@", ".", ":", ","):
            add(tok.line, "kotlin-serializable", f"uses {value}")
        elif value == "OptIn" and prev in ("@", ".", ":") and after == "(":
            end = i + 2
            while end < n and toks[end].v != ")":
                end += 1
            marks = {t.v for t in toks[i + 2:end]} & {"DomainBoundary", "ClockRead"}
            if marks:
                add(tok.line, "kotlin-optin", f"opts in to {', '.join(sorted(marks))} in a domain module")
        elif value == "ClockRead" and prev == "@" and module != "kit":
            add(tok.line, "kotlin-optin", "marks a declaration @ClockRead outside the kit")
        if prev != "." and _KT_FQN.match(value) and after == ".":
            add(tok.line, "kotlin-import", f"names {value}.{toks[i + 2].v if i + 2 < n else ''} by its full name")
        if value == "kotlin" and prev != "." and after == "." and i + 2 < n and toks[i + 2].v in _KT_FQN_KOTLIN:
            add(tok.line, "kotlin-import", f"names kotlin.{toks[i + 2].v} by its full name")


def _kotlin_decl_rules(decl, add):
    public = decl.public
    if decl.kind in ("class", "interface", "object"):
        label = decl.name or "an object"
        if "data" in decl.mods and decl.kind == "class":
            add(decl.line, "kotlin-data-class", f"{label} is a data class")
        if public and ("sealed" in decl.mods or "enum" in decl.mods or (decl.kind == "object" and "data" in decl.mods)):
            shape = ("an enum class" if "enum" in decl.mods else "a data object" if "data" in decl.mods
                     else "a sealed type")
            add(decl.line, "kotlin-public-sealed", f"{label} is public and {shape}")
        if public and decl.kind == "class":
            for param in decl.params:
                if param.binding and not (param.mods & {"private", "internal"}):
                    add(param.line, "kotlin-ctor-property",
                        f"{label}'s constructor property {param.name} is visible outside the module")
            constructible = not (decl.mods & {"abstract", "sealed", "annotation", "enum"})
            secondaries = [c for c in decl.children if c.kind == "constructor"]
            if constructible:
                if decl.ctor in ("", "public") or (decl.ctor is None and not secondaries):
                    add(decl.line, "kotlin-public-constructor", f"{label} has a public constructor")
                for ctor in secondaries:
                    if not (ctor.mods & {"private", "internal", "protected"}):
                        add(ctor.line, "kotlin-public-constructor", f"{label} has a public secondary constructor")
        return
    if decl.kind == "constructor":
        return
    if not public or decl.parent is None:
        return
    owner = _kt_owner(decl)
    boundary = "DomainBoundary" in decl.annos
    override = "override" in decl.mods
    name = decl.name
    if decl.kind == "fun":
        label = f"{owner}.{name}()" if owner else f"{name}()"
        # The platform's contracts (equals, hashCode, compareTo), the identity and a staged builder's last step.
        identity = (override and name in ("equals", "hashCode", "compareTo") or (name == "id" and not decl.params)
                    or (name == "build" and not decl.params and (owner or "").endswith("Builder")))
        if name in ("evolve", "apply"):
            add(decl.line, "kotlin-public-evolve", f"{label} is public")
        if len(decl.params) > MAX_PARAMS:
            add(decl.line, "kotlin-factory", f"{label} takes {len(decl.params)} parameters")
        if _in_companion(decl) and _returns_own(decl.type, owner) and BANNED_FACTORY.match(name):
            add(decl.line, "kotlin-factory", f"{label} builds a {owner} under a name other than parse, new or restore")
        if not boundary and not identity:
            parse = name == "parse" and _in_companion(decl)
            loose = [p.name for p in decl.params if set(p.type) & KOTLIN_PRIMITIVES] if not parse else []
            result = set(decl.type or []) & KOTLIN_PRIMITIVES
            if loose:
                add(decl.line, "kotlin-primitive-api", f"{label} takes {', '.join(loose)} as a primitive")
            if result and decl.type != ["Boolean"]:
                add(decl.line, "kotlin-primitive-api", f"{label} returns {' '.join(decl.type)}")
        if not decl.params and not identity and not (boundary and name == "wire"):
            root = _type_root(decl.type or [])
            if not (decl.type is None or root in ("Boolean", "Unit") and len(decl.type) == 1
                    or _returns_own(decl.type, owner)):
                add(decl.line, "kotlin-accessor", f"{label} hands out {' '.join(decl.type)}")
        return
    # A val or var: the stored or computed value of a class, a companion value, or a package-level value.
    label = f"{owner}.{name}" if owner else name
    typed = decl.type or []
    computed_bool = decl.getter and not decl.initializer and typed == ["Boolean"]
    companion_value = _in_companion(decl) and (
        _type_root(typed) == owner or (not typed and decl.initializer and decl.init_first == owner))
    if set(typed) & KOTLIN_PRIMITIVES and not computed_bool and not boundary:
        add(decl.line, "kotlin-primitive-api", f"{label} is public and has the type {' '.join(typed)}")
    if not computed_bool and not companion_value:
        what = "a package-level value" if decl.parent.kind == "file" else "state visible outside the module"
        add(decl.line, "kotlin-accessor", f"{label} is {what}; a public val is a computed Boolean")


def kotlin_build_findings(path, text, kit, kernels):
    """Findings for a domain module's (or the kit's) build.gradle.kts. kernels: the Gradle paths of every domain module
    (shared kernels it may depend on)."""
    source = strip(text, "kotlin", keep_strings=True)
    found = []
    add = lambda what, rule="kotlin-build-flags", line=0: found.append(Finding(path, line, rule, what))  # noqa: E731
    flat = re.sub(r"\s+", "", source)
    plugins = re.findall(r"\bid\(\"([^\"]+)\"\)|`([\w-]+)`|\bkotlin\(\"([^\"]+)\"\)|\balias\(([^)]*)\)",
                         _block(source, "plugins"))
    names = [next(part for part in match if part) for match in plugins]
    if "persona.jvm.library" not in names:
        add("does not apply persona.jvm.library")
    for other in names:
        if other != "persona.jvm.library":
            add(f"applies the plugin {other}", "kotlin-module")
    if "explicitApi()" not in flat:
        add("does not call explicitApi()")
    if "allWarningsAsErrors.set(true)" not in flat and "allWarningsAsErrors=true" not in flat:
        add("does not set allWarningsAsErrors to true")
    if "\"-Xreturn-value-checker=full\"" not in flat:
        add("does not add -Xreturn-value-checker=full")
    dependencies = _block(source, "dependencies")
    for line_number, line in enumerate(dependencies.splitlines(), _block_line(source, "dependencies")):
        m = re.match(r"\s*\"?([A-Za-z_]\w*)\"?\s*\((.*)\)\s*$", line)
        if not m or m.group(1).startswith(("test", "androidTest")):
            continue
        target = re.fullmatch(r"project\(\s*\"([^\"]+)\"\s*\)|projects((?:\.\w+)+)", m.group(2).strip())
        project = None
        if target:
            project = target.group(1) or ":" + ":".join(target.group(2).strip(".").split("."))
        if kit or project is None or not (project == ":core:domainkit" or project in kernels):
            add(f"depends on {m.group(2).strip()}", "kotlin-module", line_number)
    return found


def _block(source, name):
    m = re.search(r"(?m)^\s*" + name + r"\s*\{", source)
    if not m:
        return ""
    depth, i = 0, m.end() - 1
    while i < len(source):
        if source[i] == "{":
            depth += 1
        elif source[i] == "}":
            depth -= 1
            if depth == 0:
                return source[m.end():i]
        i += 1
    return source[m.end():]


def _block_line(source, name):
    m = re.search(r"(?m)^\s*" + name + r"\s*\{", source)
    return _line_of(source, m.end()) if m else 0


def kotlin_build_logic_findings(files):
    """The Android-wide flags in android/build-logic. files: {repository-relative path: text} of ReturnValues.kt and the
    application, library and JVM conventions."""
    found = []
    values = next((text for path, text in files.items() if path.endswith("ReturnValues.kt")), None)
    values_path = next((path for path in files if path.endswith("ReturnValues.kt")),
                       "android/build-logic/convention/src/main/kotlin/persona/buildlogic/ReturnValues.kt")
    if values is None:
        found.append(Finding(values_path, 0, "kotlin-build-flags", "is missing"))
    else:
        for flag in ("-Xreturn-value-checker=check", "-Xwarning-level=RETURN_VALUE_NOT_USED:error"):
            if f"\"{flag}\"" not in strip(values, "kotlin", keep_strings=True):
                found.append(Finding(values_path, 0, "kotlin-build-flags", f"no longer adds {flag}"))
        for m in re.finditer(r"\"(:core:[\w-]+:domain|:core:domainkit)\"", values):
            found.append(Finding(values_path, _line_of(values, m.start()), "kotlin-build-flags",
                                 f"keeps {m.group(1)}'s dropped results as warnings"))
    for path, text in files.items():
        if path.endswith(".gradle.kts") and "configureReturnValueChecks()" not in strip(text, "kotlin"):
            found.append(Finding(path, 0, "kotlin-build-flags", "does not call configureReturnValueChecks()"))
    return found


def kotlin_outside_findings(path, text, place, visitors):
    """Findings for Android code outside the domain modules' main sources. place: "adapt", "kit-adapt", "test" or
    "other". visitors: the domain visitor interfaces' names."""
    stripped = strip(text, "kotlin")
    found = []
    for m in re.finditer(r"\b(DomainBoundary|ClockRead)\b", stripped):
        mark = m.group(1)
        allowed = place == "kit-adapt" or (place == "adapt" and mark == "DomainBoundary")
        if path.endswith(".kts") or not allowed:
            found.append(Finding(path, _line_of(stripped, m.start()), "kotlin-optin", f"names {mark}"))
    if visitors and place not in ("adapt", "kit-adapt", "test") and path.endswith(".kt"):
        _, _, root = parse_kotlin(stripped)
        for decl in root.walk():
            hit = sorted(set(decl.supers) & visitors)
            if hit:
                found.append(Finding(path, decl.line, "kotlin-visitor-impl",
                                     f"{decl.name or 'an object'} implements {', '.join(hit)}"))
        for m in re.finditer(r"\bobject\s*:\s*([\w.<>, ]+?)\s*[{(]", stripped):
            hit = sorted(set(re.findall(r"\w+", m.group(1))) & visitors)
            if hit:
                found.append(Finding(path, _line_of(stripped, m.start()), "kotlin-visitor-impl",
                                     f"an object implements {', '.join(hit)}"))
    return sorted(set(found))


def kotlin_visitors(texts):
    """The visitor interfaces the domain modules declare."""
    names = set()
    for text in texts:
        names.update(re.findall(r"\binterface\s+(\w+Visitor)\b", strip(text, "kotlin")))
    return names


# ---------------------------------------------------------------------------------------------------------------------
# Swift manifests

SwiftTarget = namedtuple("SwiftTarget", "package name role sources manifest line deps settings")
_TARGET_CALL = re.compile(r"\.(target|testTarget|executableTarget|binaryTarget|macro|plugin|systemLibrary)\s*\(")
DOMAIN_TARGET = re.compile(r"^Persona\w*Domain$")
ADAPT_TARGET = re.compile(r"^Persona\w*Adapt$")
KIT = "PersonaDomainKit"
KIT_ADAPT = "PersonaDomainKitAdapt"


def swift_role(name, call="target"):
    if call == "testTarget":
        return "test-domain" if re.match(r"^Persona\w*Domain(?:Kit)?Tests$", name) else "other"
    if name == KIT:
        return "kit"
    if name == KIT_ADAPT:
        return "kit-adapt"
    if DOMAIN_TARGET.match(name):
        return "domain"
    if ADAPT_TARGET.match(name):
        return "adapt"
    return "other"


def _balanced(text, i, open_char="(", close_char=")"):
    """The text between the bracket at i and its partner (strings already blanked or kept whole)."""
    depth, j, in_string = 0, i, False
    while j < len(text):
        c = text[j]
        if c == "\"" and text[j - 1] != "\\":
            in_string = not in_string
        elif not in_string and c == open_char:
            depth += 1
        elif not in_string and c == close_char:
            depth -= 1
            if depth == 0:
                return text[i + 1:j]
        j += 1
    return text[i + 1:]


def _argument(arguments, start):
    """One argument's text: from start to the comma that ends it at depth 0, or to the end."""
    depth, in_string = 0, False
    for j in range(start, len(arguments)):
        c = arguments[j]
        if c == "\"" and arguments[j - 1] != "\\":
            in_string = not in_string
        elif in_string:
            continue
        elif c in "([{":
            depth += 1
        elif c in ")]}":
            depth -= 1
        elif c == "," and depth == 0:
            return arguments[start:j]
    return arguments[start:]


def _manifest_value(arguments, label, manifest):
    """A labelled argument's arrays: an array literal, or arrays the manifest declares by name, joined by +."""
    m = re.search(r"\b" + label + r"\s*:\s*", arguments)
    if not m:
        return ""
    out = []
    for part in _argument(arguments, m.end()).split("+"):
        part = part.strip()
        if part.startswith("["):
            out.append(_balanced(part, 0, "[", "]"))
        elif re.fullmatch(r"\w+", part):
            declared = re.search(r"\b(?:let|var)\s+" + part + r"\b[^=]*=\s*\[", manifest)
            if declared:
                out.append(_balanced(manifest, declared.end() - 1, "[", "]"))
    return ",".join(out)


def parse_manifest(package, manifest_path, text):
    """Every target a Package.swift declares (comments stripped, strings kept)."""
    source = strip(text, "swift", keep_strings=True)
    targets = []
    for m in _TARGET_CALL.finditer(source):
        arguments = _balanced(source, m.end() - 1)
        name = re.search(r"\bname\s*:\s*\"([^\"]+)\"", arguments)
        if not name:
            continue
        path = re.search(r"\bpath\s*:\s*\"([^\"]+)\"", arguments)
        default = "Tests/" if m.group(1) == "testTarget" else "Sources/"
        sources = package + "/" + (path.group(1).strip("/") if path else default + name.group(1))
        deps_text = _manifest_value(arguments, "dependencies", source)
        deps_text = re.sub(r"\bpackage\s*:\s*\"[^\"]*\"", "", deps_text)
        deps = re.findall(r"\"([^\"]+)\"", deps_text)
        settings = _manifest_value(arguments, "swiftSettings", source)
        targets.append(SwiftTarget(package, name.group(1), swift_role(name.group(1), m.group(1)), sources,
                                   manifest_path, _line_of(source, m.start()), deps, settings))
    return targets


_SWIFT_SETTINGS = (
    (".swiftLanguageMode(.v6)", ".swiftLanguageMode(.v6)"),
    (".defaultIsolation(nil)", ".defaultIsolation(nil)"),
    (".enableUpcomingFeature(\"InternalImportsByDefault\")", "the upcoming feature InternalImportsByDefault"),
    (".enableUpcomingFeature(\"MemberImportVisibility\")", "the upcoming feature MemberImportVisibility"),
    (".enableUpcomingFeature(\"ExistentialAny\")", "the upcoming feature ExistentialAny"),
)
_SWIFT_FLAGS = (("-warnings-as-errors",), ("-Xfrontend", "-disable-implicit-concurrency-module-import"),
                ("-Xfrontend", "-disable-implicit-string-processing-module-import"))


def swift_manifest_findings(target):
    """A Domain target's (or the kit's) dependencies and settings."""
    found = []
    add = lambda what: found.append(Finding(target.manifest, target.line, "swift-manifest", what))  # noqa: E731
    for dep in target.deps:
        if target.role == "kit" or not (dep == KIT or DOMAIN_TARGET.match(dep)):
            add(f"{target.name} depends on {dep}")
    flat = re.sub(r"\s+", "", target.settings)
    for needle, label in _SWIFT_SETTINGS:
        if needle not in flat:
            add(f"{target.name}'s swiftSettings lack {label}")
    flags = []
    for m in re.finditer(r"\.unsafeFlags\(\[", flat):
        flags.extend(re.findall(r"\"([^\"]*)\"", _balanced(flat, m.end() - 1, "[", "]")))
    for group in _SWIFT_FLAGS:
        if not any(tuple(flags[i:i + len(group)]) == group for i in range(len(flags))):
            add(f"{target.name}'s unsafeFlags lack {' '.join(group)}")
    return found


def swift_testable_findings(apple_sh, packages, path="scripts/ci/apple.sh"):
    """scripts/ci/apple.sh tests every domain package black-box, emits its symbol graph and runs AppleDomain."""
    found = []
    lines = apple_sh.splitlines()
    if not any("test_client_domain_modules.py" in line and "--swift-packages" in line for line in lines):
        found.append(Finding(path, 0, "swift-testable", "no longer lists the domain packages with --swift-packages"))
    step = [line for line in lines if "swift test" in line and "-emit-symbol-graph-dir" in line]
    if not step:
        found.append(Finding(path, 0, "swift-testable", "has no `swift test` step that emits the domain symbol graphs"))
    for number, line in enumerate(lines, 1):
        if "swift test" in line and "-emit-symbol-graph-dir" in line:
            for flag in ("--disable-testable-imports", "-symbol-graph-minimum-access-level"):
                if flag not in line:
                    found.append(Finding(path, number, "swift-testable", f"tests the domain packages without {flag}"))
        for package in packages:
            if re.search(re.escape(package) + r"[\"'/\s]", line + " ") and "--package-path" in line \
                    and "--disable-testable-imports" not in line:
                found.append(Finding(path, number, "swift-testable",
                                     f"tests {package} without --disable-testable-imports"))
    if not any("AppleDomain" in line and "PERSONA_DOMAIN_SYMBOL_GRAPHS" in line for line in lines):
        found.append(Finding(path, 0, "swift-testable", "no longer runs AppleDomain over the domain symbol graphs"))
    return found


# ---------------------------------------------------------------------------------------------------------------------
# Swift sources

_SWIFT_IMPORT = re.compile(
    r"(?m)^[ \t]*((?:@[\w]+(?:\([^)\n]*\))?[ \t]+)*)(?:(?:public|package|internal|fileprivate|private)[ \t]+)?"
    r"import[ \t]+(?:(?:typealias|struct|class|enum|protocol|let|var|func)[ \t]+)?([A-Za-z_]\w*)")
_SWIFT_BANNED = (
    (re.compile(r"(?<![\w.])(print|debugPrint|dump)\s*\("), "prints"),
    (re.compile(r"(?<![\w.])(fatalError|preconditionFailure)\s*\("), "traps"),
    (re.compile(r"\.random\s*\("), "draws a random number"),
    (re.compile(r"\b(SystemRandomNumberGenerator)\b"), "draws a random number"),
    (re.compile(r"(?<![\w.])(Mirror)\s*\("), "reflects on a value"),
    (re.compile(r"\b(ContinuousClock|SuspendingClock)\b"), "reads the clock"),
)
_SWIFT_CONFORMANCES = re.compile(r"\b(Codable|Encodable|Decodable|CustomStringConvertible|CustomDebugStringConvertible|"
                                 r"LosslessStringConvertible|RawRepresentable|CustomReflectable|CodingKey)\b")
_SWIFT_MODIFIERS = frozenset(("public", "open", "internal", "fileprivate", "private", "package", "static", "class",
                              "final", "override", "mutating", "nonmutating", "lazy", "weak", "unowned", "indirect",
                              "nonisolated", "isolated", "consuming", "borrowing", "dynamic", "required",
                              "convenience", "optional", "prefix", "postfix", "infix", "distributed"))
_SWIFT_ACCESS = frozenset(("public", "open", "internal", "fileprivate", "private", "package"))
_SWIFT_TYPE_KINDS = frozenset(("protocol", "struct", "class", "enum", "actor", "extension"))
_SWIFT_HEAD_KEYWORDS = _SWIFT_TYPE_KINDS | frozenset((
    "func", "init", "deinit", "subscript", "var", "let", "get", "set", "didSet", "willSet", "if", "else", "guard",
    "for", "while", "repeat", "switch", "case", "default", "do", "catch", "defer", "in", "return"))


def _swift_modifiers(toks, i):
    """The modifiers and attributes right before token i (`private(set)` included)."""
    mods, j = set(), i - 1
    while j >= 0:
        value = toks[j].v
        if value == ")" and j >= 3 and [t.v for t in toks[j - 2:j]] == ["(", "set"] and toks[j - 3].v in _SWIFT_ACCESS:
            mods.add(toks[j - 3].v + "(set)")
            j -= 4
        elif value in _SWIFT_MODIFIERS:
            mods.add(value)
            j -= 1
        elif value == ")":  # an attribute's arguments
            depth = 0
            while j >= 0:
                if toks[j].v == ")":
                    depth += 1
                elif toks[j].v == "(":
                    depth -= 1
                    if depth == 0:
                        break
                j -= 1
            j -= 1
            if j >= 1 and toks[j - 1].v == "@":
                j -= 2
            else:
                break
        elif j >= 1 and toks[j - 1].v == "@" and _ident(value):
            mods.add("@" + value)
            j -= 2
        else:
            break
    return mods


class _SwiftScope(namedtuple("_SwiftScope", "kind name public exposed")):
    """A brace scope: its kind (a type kind, "code" or "file"), the type it declares or extends, whether its members
    default to public (a public extension), and whether anything in it can be public API (no enclosing internal type)."""


def _swift_scopes(toks):
    """For each token index, the stack of scopes it is in (as a tuple), computed in one pass."""
    stack = [_SwiftScope("file", None, False, True)]
    head = 0
    at = []
    for i, tok in enumerate(toks):
        at.append(tuple(stack))
        if tok.v == "{":
            kind, name, public, exposed = "code", None, False, False
            keyword_at = None
            for j in range(i - 1, head - 1, -1):
                if toks[j].v in _SWIFT_HEAD_KEYWORDS and not (toks[j].v == "class" and j + 1 < i
                                                               and toks[j + 1].v in ("func", "var", "let")):
                    keyword_at = j
                    break
            if keyword_at is not None and toks[keyword_at].v in _SWIFT_TYPE_KINDS:
                kind = toks[keyword_at].v
                parts = []
                for tok2 in toks[keyword_at + 1:i]:
                    if tok2.v in (":", "<", "where", "{"):
                        break
                    if _ident(tok2.v):
                        parts.append(tok2.v)
                name = parts[-1] if parts else None
                mods = _swift_modifiers(toks, keyword_at)
                public = bool(mods & {"public", "open"})
                exposed = stack[-1].exposed and (kind == "extension" or _swift_public(mods, stack[-1]))
            stack.append(_SwiftScope(kind, name, public, exposed))
            head = i + 1
        elif tok.v == "}":
            if len(stack) > 1:
                stack.pop()
            head = i + 1
        elif tok.v == ";":
            head = i + 1
    return at


def _swift_public(mods, scope):
    if mods & {"public", "open"}:
        return True
    if mods & {"internal", "fileprivate", "private", "package"}:
        return False
    return scope.kind == "extension" and scope.public


def swift_source_findings(path, text, target):
    """Findings for one Swift source file of a Domain target or of the kit."""
    stripped = strip(text, "swift")
    found = []
    add = lambda line, rule, what: found.append(Finding(path, line, rule, what))  # noqa: E731
    for m in _SWIFT_IMPORT.finditer(stripped):
        attributes, module = m.group(1), m.group(2)
        line = _line_of(stripped, m.start(2))
        if "@_spi" in attributes:
            add(line, "swift-spi-import", f"{target.name} imports {module} with {attributes.strip()}")
        elif "@testable" in attributes or not (module == KIT or (DOMAIN_TARGET.match(module) and target.role != "kit")):
            add(line, "swift-import", f"{target.name} imports {module}")
    for pattern, why in _SWIFT_BANNED:
        for m in pattern.finditer(stripped):
            used = m.group(1) if m.groups() else m.group()
            add(_line_of(stripped, m.start()), "swift-banned-call", f"uses {used} ({why})")
    for m in re.finditer(r"\b(throws|rethrows|throw|Error)\b", stripped):
        add(_line_of(stripped, m.start()), "swift-throws", f"uses {m.group(1)}")
    for m in _SWIFT_CONFORMANCES.finditer(stripped):
        add(_line_of(stripped, m.start()), "swift-conformance", f"names {m.group(1)}")
    toks = tokens(stripped)
    scopes = _swift_scopes(toks)
    n = len(toks)
    for i, tok in enumerate(toks):
        stack = scopes[i]
        scope = stack[-1]
        in_code = any(s.kind == "code" for s in stack)
        if tok.v == "mutating" and not in_code:
            j = i + 1
            while j < n and toks[j].v in _SWIFT_MODIFIERS:
                j += 1
            if j < n and toks[j].v == "func" and scope.kind != "protocol":
                name = toks[j + 1].v if j + 1 < n else ""
                add(tok.line, "swift-mutating", f"declares mutating func {name}")
        elif tok.v == "enum" and not in_code and scope.exposed and i + 1 < n and _ident(toks[i + 1].v):
            name = toks[i + 1].v
            label = f"{scope.name}.{name}" if scope.name else name
            if _swift_public(_swift_modifiers(toks, i), scope) and not _kit_exception(
                    "swift-public-enum", target.name, tuple(label.split("."))):
                add(tok.line, "swift-public-enum", f"{label} is a public enum")
        elif tok.v in ("let", "var") and not in_code and scope.exposed \
                and scope.kind in (_SWIFT_TYPE_KINDS - {"protocol"}) | {"file"}:
            mods = _swift_modifiers(toks, i)
            if not _swift_public(mods, scope) or i + 1 >= n:
                continue
            name = toks[i + 1].v
            label = f"{scope.name}.{name}" if scope.name else name
            if any(mod.endswith("(set)") for mod in mods):
                add(tok.line, "swift-stored-public", f"{label} has a setter")
                continue
            j, annotated = i + 2, None
            if j < n and toks[j].v == ":":
                j += 1
                begin = j
                depth = 0
                while j < n and not (depth == 0 and (toks[j].v in ("=", "{") or toks[j].first)):
                    depth += toks[j].v in ("<", "[", "(")
                    depth -= toks[j].v in (">", "]", ")")
                    j += 1
                annotated = [t.v for t in toks[begin:j]]
            # What follows the name and type on its line: = for a stored value, { for accessors (or observers).
            follows = toks[j].v if j < n and (toks[j].v in ("=", "{") or not toks[j].first) else ""
            if tok.v == "let":
                if annotated:
                    own = scope.name is not None and _type_root(annotated) in (scope.name, "Self")
                else:
                    own = (scope.name is not None and follows == "=" and j + 1 < n
                           and toks[j + 1].v in (scope.name, "Self"))
                if not ({"static", "class"} & mods and own):
                    add(tok.line, "swift-stored-public", f"{label} is a stored public property")
            elif follows == "{":
                inner = toks[j + 1].v if j + 1 < n else ""
                if inner in ("didSet", "willSet"):
                    add(tok.line, "swift-stored-public", f"{label} is a stored public property")
                else:
                    end, depth = j, 0
                    while end < n:
                        depth += toks[end].v == "{"
                        depth -= toks[end].v == "}"
                        if depth == 0:
                            break
                        if depth == 1 and toks[end].v == "set" and toks[end - 1].v in ("{", "}"):
                            add(tok.line, "swift-stored-public", f"{label} has a public setter")
                            break
                        end += 1
            else:
                add(tok.line, "swift-stored-public", f"{label} is a stored public property")
    return sorted(set(found))


def _kit_exception(rule, target, path):
    return (rule, target, ".".join(path)) in KIT_EXCEPTIONS


def swift_public_functions(texts):
    """The names of the functions the Domain targets declare public, for the app's dropped results."""
    names = set()
    for text in texts:
        stripped = strip(text, "swift")
        names.update(re.findall(r"\b(?:public|open)\s+(?:[\w()]+\s+)*?func\s+([A-Za-z_]\w*)", stripped))
    return names


def swift_visitors(texts):
    names = set()
    for text in texts:
        names.update(re.findall(r"\bprotocol\s+(\w+Visitor)\b", strip(text, "swift")))
    return names


_SWIFT_CONFORMANCE_HEAD = re.compile(
    r"\b(?:struct|class|enum|actor|extension)\s+[\w.]+(?:\s*<[^>{]*>)?\s*:\s*([^{]*)\{")


def swift_outside_findings(path, text, role, visitors, functions, app):
    """Findings for a Swift file outside the Domain targets and the kit. role: the target role of the file ("adapt",
    "kit-adapt", "test-domain" or "other"); app: whether the file is app code (apple/Persona, apple/Shared)."""
    stripped = strip(text, "swift")
    found = []
    for m in _SWIFT_IMPORT.finditer(stripped):
        attributes, module = m.group(1), m.group(2)
        if not (module == KIT or DOMAIN_TARGET.match(module)):
            continue
        spis = set(re.findall(r"@_spi\(\s*(\w+)\s*\)", attributes))
        allowed = {"Boundary"} if role == "adapt" else {"Boundary", "Clock"} if role == "kit-adapt" else set()
        for spi in sorted(spis - allowed):
            found.append(Finding(path, _line_of(stripped, m.start(2)), "swift-spi-import",
                                 f"imports {module} with @_spi({spi})"))
    if visitors and role not in ("adapt", "kit-adapt", "test-domain"):
        for m in _SWIFT_CONFORMANCE_HEAD.finditer(stripped):
            hit = sorted(set(re.findall(r"\w+", m.group(1).split("where")[0])) & visitors)
            if hit:
                found.append(Finding(path, _line_of(stripped, m.start()), "swift-visitor-impl",
                                     f"conforms to {', '.join(hit)}"))
    if app and functions and re.search(r"\bimport\s+(?:PersonaDomainKit|Persona\w*Domain)\b", stripped):
        for m in re.finditer(r"(?<![\w.])_\s*=\s*([^\n;]*)", stripped):
            called = [name for name in re.findall(r"\b([A-Za-z_]\w*)\s*\(", m.group(1)) if name in functions]
            if called:
                found.append(Finding(path, _line_of(stripped, m.start()), "swift-app-discard",
                                     f"drops the result of {called[0]}"))
    return sorted(set(found))


_UNUSED = re.compile(r"(?:(?P<file>/[^\s:]+\.swift):(?P<line>\d+):(?:\d+:)?\s*)?(?:warning: )?result of (?:call to "
                     r"'(?P<call>[^'(]+)(?:\([^']*\))?'|'(?P<init>[^']+)' initializer) is unused")
_APPLE_PATH = re.compile(r"/(apple/[A-Z].*)$")


def swift_build_log_findings(name, text, functions):
    """Unused-result warnings in an app build log (plain or an Xcode activity log's text) naming a domain function."""
    found = []
    for m in _UNUSED.finditer(text):
        called = m.group("call") or m.group("init")
        if called not in functions:
            continue
        path, line = name, 0
        inside = _APPLE_PATH.search(m.group("file") or "")
        if inside:
            path, line = inside.group(1), int(m.group("line"))
        found.append(Finding(path, line, "swift-app-discard", f"drops the result of {called} (build warning)"))
    return sorted(set(found))


def read_build_log(path):
    """A build log as text: a plain log, or an Xcode .xcactivitylog (gzip)."""
    data = Path(path).read_bytes()
    if data[:2] == b"\x1f\x8b":
        data = gzip.decompress(data)
    return data.decode("utf-8", errors="replace")


# ---------------------------------------------------------------------------------------------------------------------
# Swift symbol graphs


def _fragments(fragments):
    return "".join(f.get("spelling", "") for f in fragments or [])


def _types(fragments):
    return [f["spelling"] for f in fragments or [] if f.get("kind") == "typeIdentifier"]


def _graph_location(symbol, root_marker="/apple/Packages/"):
    location = symbol.get("location") or {}
    uri = location.get("uri", "")
    path = uri.split("file://", 1)[-1]
    if root_marker in path:
        path = "apple/Packages/" + path.rsplit(root_marker, 1)[1]
    line = (location.get("position") or {}).get("line")
    return path, (line + 1 if isinstance(line, int) else 0)


def swift_graph_findings(target, graphs):
    """The public API rules over a Domain target's (or the kit's) symbol graphs: the parsed JSON of
    <Target>.symbols.json and every <Target>@<Other>.symbols.json."""
    symbols, parents = {}, {}
    for graph in graphs:
        for symbol in graph.get("symbols", []):
            symbols[symbol["identifier"]["precise"]] = symbol
        for relation in graph.get("relationships", []):
            if relation.get("kind") == "memberOf":
                parents[relation["source"]] = relation["target"]
    found = []
    for precise, symbol in symbols.items():
        # Members the compiler adds (a protocol's default implementations, Codable's init(from:)) have no place in the
        # source; the conformance that brought them is a finding of its own.
        if "::SYNTHESIZED::" in precise or not symbol.get("location"):
            continue
        if symbol.get("accessLevel") not in ("public", "open"):
            continue
        kind = symbol["kind"]["identifier"]
        components = symbol.get("pathComponents") or []
        path, line = _graph_location(symbol)
        name = components[-1] if components else ""
        base = name.split("(", 1)[0]
        owner = components[-2] if len(components) > 1 else None
        label = ".".join(components)
        add = lambda rule, what: found.append(Finding(path, line, rule, what))  # noqa: E731
        decl = _fragments(symbol.get("declarationFragments"))
        signature = symbol.get("functionSignature") or {}
        params = signature.get("parameters") or []
        returns = signature.get("returns") or []
        own_generics = _own_generics(symbol, symbols.get(parents.get(precise)))
        if kind == "swift.enum" and not _kit_exception("swift-public-enum", target, components):
            add("swift-public-enum", f"{label} is a public enum")
        generic = own_generics or kind == "swift.associatedtype"
        if generic and not _kit_exception("swift-generic", target, components[:1]):
            add("swift-generic", f"{label} is generic")
        if kind in ("swift.property", "swift.type.property", "swift.typealias", "swift.var"):
            stored_let = re.match(r"^(?:static |class )?let ", decl)
            stored_var = re.match(r"^(?:static |class )?var ", decl) and "{" not in decl
            setter = "{ get set }" in decl
            declared = _types(symbol.get("declarationFragments"))
            if kind != "swift.typealias" and (stored_let or stored_var or setter):
                own = declared[:1] in ([owner], ["Self"]) and len(declared) == 1
                if not (stored_let and decl.startswith(("static ", "class ")) and own):
                    what = "has a public setter" if setter else "is a stored public property"
                    add("swift-stored-public", f"{label} {what}")
            computed_bool = "{ get }" in decl and declared == ["Bool"]
            if set(declared) & SWIFT_PRIMITIVES and not computed_bool:
                add("swift-primitive-api", f"{label} has the type {' '.join(declared)}")
            if kind != "swift.typealias" and owner and base != "id" and not (
                    declared == ["Bool"] or declared[:1] in ([owner], ["Self"]) and len(declared) == 1):
                add("swift-accessor", f"{label} hands out {' '.join(declared) or 'state'}")
            continue
        if kind not in ("swift.method", "swift.type.method", "swift.func", "swift.init", "swift.subscript",
                        "swift.type.subscript", "swift.func.op"):
            continue
        param_types = [_types(p.get("declarationFragments")) for p in params]
        result = _types(returns)
        if base in ("evolve", "apply") and kind != "swift.init":
            add("swift-public-evolve", f"{label} is public")
        if len(params) > MAX_PARAMS:
            add("swift-factory", f"{label} takes {len(params)} parameters")
        if kind == "swift.init":
            add("swift-factory", f"{label} is a public initializer")
        builds_own = result[:1] in ([owner], ["Self"]) or (
            result[:1] == ["Outcome"] and result[1:2] in ([owner], ["Self"]))
        if kind == "swift.type.method" and builds_own and BANNED_FACTORY.match(base):
            add("swift-factory", f"{label} builds a {owner} under a name other than parse, new or restore")
        parse = kind == "swift.type.method" and base == "parse"
        loose = [p.get("name", "_") for p, kinds in zip(params, param_types) if set(kinds) & SWIFT_PRIMITIVES]
        if loose and not parse:
            add("swift-primitive-api", f"{label} takes {', '.join(loose)} as a primitive")
        if set(result) & SWIFT_PRIMITIVES and result != ["Bool"]:
            add("swift-primitive-api", f"{label} returns {_fragments(returns)}")
        built = base == "build" and owner is not None and owner.endswith("Builder")
        if owner and not params and kind in ("swift.method", "swift.type.method") and base != "id" and not built:
            plain = _fragments(returns).strip()
            if not (plain in ("", "()", "Void") or result == ["Bool"] or builds_own):
                add("swift-accessor", f"{label} hands out {plain}")
    return sorted(set(found))


def _own_generics(symbol, parent):
    """The generic parameters a symbol declares itself, not the ones of the type it is a member of."""
    def parameters(s):
        return {(p.get("name"), p.get("depth")) for p in ((s or {}).get("swiftGenerics") or {}).get("parameters") or []}
    return parameters(symbol) - parameters(parent)


# ---------------------------------------------------------------------------------------------------------------------
# The tree: finding the modules and running every rule

Scan = namedtuple("Scan", "modules findings inspected")


def _walk(top, suffixes):
    """Repository files under top with one of suffixes, build output and hidden folders left out."""
    for directory, folders, files in os.walk(top):
        folders[:] = sorted(f for f in folders if not f.startswith(".") and f not in _BUILD_DIRS
                            and not f.endswith(_BUILD_SUFFIXES))
        for name in sorted(files):
            if name.endswith(suffixes):
                yield Path(directory) / name


def _rel(root, path):
    return Path(path).relative_to(root).as_posix()


def _read(path):
    return Path(path).read_text(encoding="utf-8", errors="replace")


def swift_targets(root):
    """Every target of every apple/Packages/*/Package.swift, and the problems finding the domain ones."""
    root = Path(root)
    targets, problems = [], []
    packages = root / "apple" / "Packages"
    manifests = sorted(packages.glob("*/Package.swift")) if packages.is_dir() else []
    texts = {}
    for manifest in manifests:
        package = _rel(root, manifest.parent)
        texts[package] = _read(manifest)
        targets.extend(parse_manifest(package, _rel(root, manifest), texts[package]))
    declared = {t.name for t in targets}
    for package, text in texts.items():
        source = strip(text, "swift", keep_strings=True)
        for m in re.finditer(r"\"(Persona\w*(?:Domain|Adapt)|PersonaDomainKit\w*)\"", source):
            if m.group(1) not in declared:
                problems.append(Finding(package + "/Package.swift", _line_of(source, m.start()), "swift-discovery",
                                        f"names {m.group(1)}, which no .target(name:) declares"))
    by_sources = {t.sources: t for t in targets}
    if (root / "apple").is_dir():
        for directory, folders, _ in os.walk(root / "apple"):
            folders[:] = [f for f in folders if not f.startswith(".") and f not in _BUILD_DIRS
                          and not f.endswith(_BUILD_SUFFIXES)]
            for folder in folders:
                if (Path(directory) / folder / "Package.swift").is_file():
                    continue  # a package's own folder (apple/Packages/PersonaDomainKit), not a target's sources
                if swift_role(folder) in ("domain", "adapt", "kit", "kit-adapt"):
                    rel = _rel(root, Path(directory) / folder)
                    if rel not in by_sources:
                        problems.append(Finding(rel, 0, "swift-discovery",
                                                f"is named like a domain target, but no package declares {folder} "
                                                "with these sources"))
    return targets, problems


def scan_apple(root, apple_sh=None, graphs_dir=None, build_logs=()):
    """Every Swift rule over the tree at root. apple_sh: scripts/ci/apple.sh's text (None leaves the step check out);
    graphs_dir: where apple.sh wrote the symbol graphs (None leaves the public API rules out); build_logs: app build
    logs to read for unused domain results."""
    root = Path(root)
    targets, findings = swift_targets(root)
    modules = [t for t in targets if t.role in ("domain", "kit", "adapt", "kit-adapt")]
    cores = [t for t in modules if t.role in ("domain", "kit")]
    inspected = 0
    core_texts = []
    for target in cores:
        findings.extend(swift_manifest_findings(target))
        files = list(_walk(root / target.sources, (".swift",))) if (root / target.sources).is_dir() else []
        if not files:
            findings.append(Finding(target.sources, 0, "swift-discovery", f"{target.name} has no Swift sources here"))
        for file in files:
            text = _read(file)
            core_texts.append(text)
            inspected += 1
            findings.extend(swift_source_findings(_rel(root, file), text, target))
    visitors = swift_visitors(core_texts)
    functions = swift_public_functions(core_texts)
    core_dirs = {t.sources for t in cores}
    roles = {t.sources: t.role for t in targets}
    declared = {t.name for t in targets}
    if (root / "apple").is_dir():
        for file in _walk(root / "apple", (".swift",)):
            rel = _rel(root, file)
            owner = next((sources for sources in roles if rel.startswith(sources + "/")), None)
            if owner in core_dirs:
                continue
            data = file.read_bytes()
            for m in _SWIFT_DOMAIN_IMPORT.finditer(data):
                module = m.group(1).decode()
                if module not in declared:
                    findings.append(Finding(rel, data.count(b"\n", 0, m.start(1)) + 1, "swift-discovery",
                                            f"imports {module}, which no package declares"))
            app = rel.startswith(("apple/Persona/", "apple/Shared/"))
            if b"_spi" not in data and not any(v.encode() in data for v in visitors) \
                    and not (app and b"_ =" in data):
                continue
            findings.extend(swift_outside_findings(rel, data.decode("utf-8", errors="replace"),
                                                   roles.get(owner, "other"), visitors, functions, app))
    for package in sorted({t.package for t in cores}):
        if not any(t.package == package and t.role == "test-domain" for t in targets):
            findings.append(Finding(package + "/Package.swift", 0, "swift-testable",
                                    "declares no Persona<Ctx>DomainTests target, so `swift test` has no test to run"))
    packages = sorted({t.package for t in modules})
    if apple_sh is not None:
        findings.extend(swift_testable_findings(apple_sh, packages))
    if graphs_dir is not None:
        for target in cores:
            main = Path(graphs_dir) / f"{target.name}.symbols.json"
            if not main.is_file():
                findings.append(Finding(target.sources, 0, "swift-testable",
                                        f"has no symbol graph {main.name} in {graphs_dir}"))
                continue
            graphs = [json.loads(main.read_text(encoding="utf-8"))]
            graphs += [json.loads(p.read_text(encoding="utf-8"))
                       for p in sorted(Path(graphs_dir).glob(f"{target.name}@*.symbols.json"))]
            inspected += 1
            findings.extend(swift_graph_findings(target.name, graphs))
    for log in build_logs:
        findings.extend(swift_build_log_findings(str(log), read_build_log(log), functions))
    return Scan(modules, sorted(set(findings)), inspected)


_KT_DOMAIN_PACKAGE = re.compile(rb"(?m)^\s*package\s+persona\.core\.([a-z0-9_]+)\.(domain|adapt)\b")
_KT_DOMAIN_IMPORT = re.compile(rb"(?m)^\s*import\s+persona\.core\.(?:([a-z0-9_]+)\.domain|(domainkit))\b")
_SWIFT_DOMAIN_IMPORT = re.compile(rb"(?m)^[ \t]*(?:@\w+(?:\([^)\n]*\))?[ \t]+)*(?:(?:public|package|internal|fileprivate|"
                                  rb"private)[ \t]+)?import[ \t]+(PersonaDomainKit\w*|Persona\w*Domain)\b")
_KT_KIT_PACKAGE = re.compile(rb"(?m)^\s*package\s+persona\.core\.domainkit\b")


KotlinModule = namedtuple("KotlinModule", "ctx kind path")  # kind: "domain", "kit" or "adapt"


def android_modules(root):
    """The Kotlin domain modules, the kit and the adapt packages, and the problems finding them."""
    root = Path(root)
    settings = root / "android" / "settings.gradle.kts"
    included = set()
    problems = []
    if settings.is_file():
        source = strip(_read(settings), "kotlin", keep_strings=True)
        included = set(re.findall(r"\"(:[\w:-]+)\"", _block_calls(source, "include")))
        for m in re.finditer(r"project\(\s*\"(:core:[\w-]+:domain|:core:domainkit)\"\s*\)\s*\.\s*projectDir", source):
            problems.append(Finding("android/settings.gradle.kts", _line_of(source, m.start()), "kotlin-discovery",
                                    f"moves {m.group(1)} away from its conventional folder"))
    modules = []
    for project in sorted(included):
        m = re.fullmatch(r":core:([a-z0-9_]+):domain", project)
        if m:
            modules.append(KotlinModule(m.group(1), "domain", f"android/core/{m.group(1)}/domain"))
        elif project == ":core:domainkit":
            modules.append(KotlinModule("domainkit", "kit", "android/core/domainkit"))
    core = root / "android" / "core"
    if core.is_dir():
        for ctx_dir in sorted(p for p in core.iterdir() if p.is_dir()):
            ctx = ctx_dir.name
            adapt = ctx_dir / "src" / "main" / "kotlin" / "persona" / "core" / ctx / "adapt"
            if adapt.is_dir() and f":core:{ctx}" in included:
                modules.append(KotlinModule(ctx, "adapt", _rel(root, adapt)))
            domain = ctx_dir / "domain"
            if domain.is_dir() and f":core:{ctx}:domain" not in included:
                problems.append(Finding(_rel(root, domain), 0, "kotlin-discovery",
                                        f"is named like a domain module, but settings.gradle.kts does not include "
                                        f":core:{ctx}:domain"))
        if (core / "domainkit").is_dir() and ":core:domainkit" not in included:
            problems.append(Finding("android/core/domainkit", 0, "kotlin-discovery",
                                    "is the kit's folder, but settings.gradle.kts does not include :core:domainkit"))
    return modules, problems


def _block_calls(source, name):
    out = []
    for m in re.finditer(r"\b" + name + r"\s*\(", source):
        out.append(_balanced(source, m.end() - 1))
    return "\n".join(out)


def _android_place(rel, modules):
    """Where an Android file is: ("domain"|"kit"|"adapt"|"kit-adapt"|"test"|"other", the module or None)."""
    for module in modules:
        if not rel.startswith(module.path + "/"):
            continue
        rest = rel[len(module.path) + 1:]
        if module.kind == "adapt":
            return "kit-adapt" if module.ctx == "domainkit" else "adapt", module
        if rest.startswith("src/main/"):
            if module.kind == "kit" and "/persona/core/domainkit/adapt/" in "/" + rest:
                return "kit-adapt", module
            return module.kind, module
        if rest.startswith("src/"):
            return "test", module
        return "build", module
    if re.search(r"/src/(test|androidTest|testFixtures)\w*/", rel):
        m = re.search(r"/persona/core/([a-z0-9_]+)/adapt/", rel)
        return ("adapt" if m else "test"), None
    return "other", None


def scan_android(root, build_logic=True):
    """Every Kotlin rule over the tree at root."""
    root = Path(root)
    modules, findings = android_modules(root)
    kernels = {f":core:{m.ctx}:domain" for m in modules if m.kind == "domain"}
    contexts = {m.ctx for m in modules if m.kind == "domain"}
    kit_found = any(m.kind == "kit" for m in modules)
    inspected = 0
    core_texts = []
    seen = set()
    for module in modules:
        if module.kind in ("domain", "kit"):
            build = root / module.path / "build.gradle.kts"
            if not build.is_file():
                findings.append(Finding(module.path, 0, "kotlin-discovery", "has no build.gradle.kts"))
            else:
                findings.extend(kotlin_build_findings(_rel(root, build), _read(build), module.kind == "kit", kernels))
            main = root / module.path / "src" / "main"
            files = list(_walk(main, (".kt",))) if main.is_dir() else []
            if not files:
                findings.append(Finding(module.path, 0, "kotlin-discovery", "has no Kotlin sources in src/main"))
            for file in files:
                rel = _rel(root, file)
                place, _ = _android_place(rel, modules)
                if place == "kit-adapt":
                    continue
                text = _read(file)
                core_texts.append(text)
                seen.add(rel)
                inspected += 1
                findings.extend(kotlin_source_findings(rel, text, module.kind))
    visitors = kotlin_visitors(core_texts)
    android = root / "android"
    if android.is_dir():
        for file in _walk(android, (".kt", ".kts")):
            rel = _rel(root, file)
            data = file.read_bytes()
            place, module = _android_place(rel, modules)
            for m in _KT_DOMAIN_PACKAGE.finditer(data):
                ctx, kind = m.group(1).decode(), m.group(2).decode()
                expected = f"android/core/{ctx}/domain/" if kind == "domain" else \
                    f"android/core/{ctx}/src/main/kotlin/persona/core/{ctx}/adapt/"
                if ctx != "domain" and module is None and not re.search(r"/src/\w*[tT]est\w*/", rel):
                    findings.append(Finding(rel, data.count(b"\n", 0, m.end()) + 1, "kotlin-discovery",
                                            f"is in package persona.core.{ctx}.{kind}, which the guard reads only in "
                                            f"{expected} of an included module"))
            for m in _KT_DOMAIN_IMPORT.finditer(data):
                ctx = (m.group(1) or b"").decode()
                missing = (m.group(2) and not kit_found) or (ctx and ctx != "domain" and ctx not in contexts)
                if missing:
                    name = "persona.core.domainkit" if m.group(2) else f"persona.core.{ctx}.domain"
                    findings.append(Finding(rel, data.count(b"\n", 0, m.end()) + 1, "kotlin-discovery",
                                            f"imports {name}, which no included module holds"))
            if _KT_KIT_PACKAGE.search(data) and not rel.startswith("android/core/domainkit/"):
                findings.append(Finding(rel, 0, "kotlin-discovery", "is in package persona.core.domainkit outside "
                                                                    "android/core/domainkit"))
            if rel in seen:
                continue
            if b"DomainBoundary" not in data and b"ClockRead" not in data and not any(
                    v.encode() in data for v in visitors):
                continue
            findings.extend(kotlin_outside_findings(rel, data.decode("utf-8", errors="replace"), place, visitors))
    if build_logic:
        logic = root / "android" / "build-logic" / "convention" / "src" / "main" / "kotlin"
        files = {}
        for name in ("persona/buildlogic/ReturnValues.kt", "persona.android.application.gradle.kts",
                     "persona.android.library.gradle.kts", "persona.jvm.library.gradle.kts"):
            path = logic / name
            if path.is_file():
                files[_rel(root, path)] = _read(path)
        findings.extend(kotlin_build_logic_findings(files))
    return Scan(modules, sorted(set(findings)), inspected)
