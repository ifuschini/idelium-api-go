import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT_PATH = ROOT / "scripts" / "build_performance_baseline.py"
SPEC = importlib.util.spec_from_file_location("build_performance_baseline", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class PerformanceBaselineTest(unittest.TestCase):
    def setUp(self):
        self.backlog = MODULE.load_json(ROOT / "docs" / "contracts" / "compatibility-backlog.json")
        self.report = MODULE.build_report(self.backlog)
        self.committed = json.loads(
            (ROOT / "docs" / "contracts" / "performance-baseline.json").read_text(
                encoding="utf-8"
            )
        )

    def test_committed_report_matches_current_backlog(self):
        self.assertEqual(self.report, self.committed)

    def test_report_contains_representative_reads_and_writes(self):
        classes = {item["class"] for item in self.report["cases"]}
        self.assertEqual({"read", "write"}, classes)
        self.assertGreaterEqual(
            len([item for item in self.report["cases"] if item["class"] == "read"]),
            4,
        )
        self.assertGreaterEqual(
            len([item for item in self.report["cases"] if item["class"] == "write"]),
            3,
        )

    def test_every_case_has_budget_sample_size_and_capture_status(self):
        for item in self.report["cases"]:
            with self.subTest(scenario=item["scenario"]):
                self.assertEqual("capture-required", item["baselineStatus"])
                self.assertLess(0, item["budgetMs"]["p50"])
                self.assertLessEqual(item["budgetMs"]["p50"], item["budgetMs"]["p95"])
                self.assertLessEqual(item["budgetMs"]["p95"], item["budgetMs"]["p99"])
                self.assertGreaterEqual(item["sampleSize"]["minimumRequests"], 100)

    def test_report_does_not_persist_payload_values_or_credentials(self):
        serialized = json.dumps(self.report).lower()
        forbidden = ("authorization", "cookie", "password", "secret", "token")
        for value in forbidden:
            with self.subTest(value=value):
                self.assertNotIn(value, serialized)


if __name__ == "__main__":
    unittest.main()
