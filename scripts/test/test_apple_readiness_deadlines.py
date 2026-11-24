"""scripts/ci/apple.sh's hosted iOS readiness: the deadline and result-inspection cases of test_apple_readiness.py.

Split from that suite (#13293), then split by fixture cost (#14321) so each file stays well under the scripts lane's
120-second limit per suite (scripts/ci/python-tests.py). The fixture is test_apple_readiness.py's
AppleReadinessTests; only the two process-deadline tests defined here run from this file (load_tests).
"""
import unittest

import test_apple_readiness as readiness

AppleReadinessTests = readiness.AppleReadinessTests  # the cases below build fresh fixtures of it


class AppleReadinessDeadlineTests(readiness.AppleReadinessTests):
    def test_probe_and_result_inspection_share_one_deadline(self):
        for key, pid_name, product_hang in (("PROBE_HANG", "probe.pid", False),
                                            ("RESULT_HANG", "result.pid", False),
                                            ("PROBE_HANG", "probe.pid", True)):
            with self.subTest(phase=key, product_hang=product_hang):
                case = AppleReadinessTests()
                case.setUp()
                try:
                    result, _ = case.run_preparation(**{key: "1", "TEST_STEP_TIMEOUT": "1",
                                                        "PRODUCT_HANG": "1" if product_hang else "0"})
                    self.assertEqual(result.returncode, 124, result.stdout + result.stderr)
                    self.assertFalse((case.root / "full-suite-started").exists())
                    self.assertTrue((case.root / pid_name).exists())
                    # The deadline terminates the probe/result process, not just its shell.
                    self.assertEqual((case.root / pid_name.replace(".pid", ".stopped")).read_text(), "terminated")
                    observer = case.output / "diagnostics/live-readiness"
                    if product_hang:
                        self.assertIn("xcodebuild-finished", (observer / "observer.status.txt").read_text())
                        product = observer / "product-0-architecture.txt"
                        self.assertEqual(product.read_text(), "unavailable: command timed out\n")
                        self.assertEqual(product.with_suffix(".txt.status.txt").read_text(), "124\n")
                    else:
                        self.assertIn("xcodebuild-finished", (observer / "observer.status.txt").read_text())
                        case.assert_collected_products(observer)
                    self.assertTrue((observer / "host-final.json").is_file())
                    case.assert_recorded_processes_reaped()
                finally:
                    case.doCleanups()

    def test_start_is_inside_the_existing_permission_deadline(self):
        result, calls = self.run_preparation(BOOT_HANG="1")
        self.assertEqual(result.returncode, 124, result.stdout + result.stderr)
        self.assertEqual(len(calls), 2)
        self.assertEqual((self.root / "boot.stopped").read_text(), "terminated")
        self.assertFalse((self.root / "privacy-attempted").exists())
        self.assertFalse((self.root / "full-suite-started").exists())
        self.assertEqual((self.root / "budgets").read_text().splitlines()[-1],
                         "Grant Reminders access to the owned test app|300")


def load_tests(loader, tests, pattern):
    """Only this file's tests: the fixture class it extends runs its own in test_apple_readiness.py."""
    suite = unittest.TestSuite()
    for name in sorted(n for n in vars(AppleReadinessDeadlineTests) if n.startswith("test_")):
        suite.addTest(AppleReadinessDeadlineTests(name))
    return suite


if __name__ == "__main__":
    unittest.main()
