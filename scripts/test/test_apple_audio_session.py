#!/usr/bin/env python3
"""The Apple apps reach the shared audio session only through the core's broker (#13849 item 3, #13431;
docs/apple/ios.md, "Audio session").

`AudioSessionBroker` (apple/Shared/AudioSessionBroker.swift) is the one place that calls `AVAudioSession`: the voice
conversations, system calls, CarPlay, recordings, dictation, meeting playback and the screen's Picture in Picture go
through it, so it can keep a lease per client and, later, decide between them. This check fails on any other Swift file
under apple/ (tests excluded) that takes the shared session (`AVAudioSession.sharedInstance()`) or calls a method that
changes it on an `AVAudioSession` it was handed. Types, ports, categories and the session's notifications stay free to
use. Comments and string literals do not count. No baseline and no marker: the count is zero.

  python3 scripts/test/test_apple_audio_session.py          run the check (unittest)
  python3 scripts/test/test_apple_audio_session.py --list   print every call found, file:line
"""
from pathlib import Path
import re
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))
import test_apple_module_refs as refs  # noqa: E402  (its Swift blanking and file walk)

ROOT = refs.ROOT
BROKER = "apple/Shared/AudioSessionBroker.swift"
SKIP_PARTS = refs.SKIP_PARTS

SHARED = re.compile(r"\bAVAudioSession\s*\.\s*sharedInstance\s*\(")
# A delegate hands the session over (`didActivate audioSession: AVAudioSession`): a parameter of that type whose
# changing methods are called.
HANDED = re.compile(r"(\w+)\s*:\s*AVAudioSession\b(?!\s*\.)")
CHANGES = ("setCategory", "setActive", "setMode", "overrideOutputAudioPort", "setPreferredInput",
           "setPreferredSampleRate", "setPreferredIOBufferDuration", "setAllowHapticsAndSystemSoundsDuringRecording")


def calls(src):
    """[(line, what)] for each call to the shared session in one Swift file."""
    code = refs.blank(src)
    found = [(code.count("\n", 0, m.start()) + 1, "AVAudioSession.sharedInstance()") for m in SHARED.finditer(code)]
    for name in sorted(set(HANDED.findall(code))):
        for m in re.finditer(r"\b" + re.escape(name) + r"\s*\.\s*(" + "|".join(CHANGES) + r")\s*\(", code):
            found.append((code.count("\n", 0, m.start()) + 1, f"{name}.{m.group(1)}("))
    return sorted(found)


def scan():
    sites = []
    for rel in refs.swift_files():
        if rel.as_posix() == BROKER:
            continue
        for line, what in calls((ROOT / rel).read_text(encoding="utf-8", errors="replace")):
            sites.append(f"{rel}:{line} {what}")
    return sites


class AppleAudioSessionTests(unittest.TestCase):
    def test_only_the_broker_calls_the_audio_session(self):
        self.assertEqual(scan(), [], "call the shared audio session through AudioSessionBroker.shared, with your "
                         "AudioSessionClient (apple/Shared/AudioSessionBroker.swift)")

    def test_the_broker_exists_and_calls_it(self):
        src = (ROOT / BROKER).read_text(encoding="utf-8")
        self.assertTrue(calls(src), "the broker no longer reaches AVAudioSession")

    def test_what_counts(self):
        sample = '''
        // AVAudioSession.sharedInstance().setActive(true) in a comment
        let s = "AVAudioSession.sharedInstance()"
        let ports: [AVAudioSession.Port] = []
        NotificationCenter.default.addObserver(forName: AVAudioSession.routeChangeNotification, object: nil, queue: .main) { _ in }
        let a = AVAudioSession.sharedInstance()
        func manager(_ m: Any, didActivate audioSession: AVAudioSession) { try? audioSession.setActive(true) }
        func other(_ s: AVAudioSession) { _ = s.currentRoute }
        '''
        self.assertEqual(calls(sample), [(6, "AVAudioSession.sharedInstance()"), (7, "audioSession.setActive(")])


if __name__ == "__main__":
    if "--list" in sys.argv:
        print("\n".join(scan()) or "no direct audio session calls outside the broker")
        sys.exit(0)
    unittest.main()
