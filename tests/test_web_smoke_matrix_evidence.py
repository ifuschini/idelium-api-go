import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
LARAVEL = ROOT.parent / "idelium-api"
FIXTURES = ROOT / "testdata/golden/laravel-parallel-runs"


class WebSmokeMatrixEvidenceTest(unittest.TestCase):
    def test_browser_fixture_set_covers_all_operations(self):
        expected = {"browser-create", "browser-matrix", "parallel-run-list", "browser-show", "browser-claim", "browser-results", "browser-heartbeat", "browser-worker-update", "browser-cancel"}
        actual = {path.name.removesuffix("-go.fixture.json") for path in FIXTURES.glob("browser-*-go.fixture.json")}
        if (FIXTURES / "parallel-run-list-go.fixture.json").is_file():
            actual.add("parallel-run-list")
        self.assertEqual(actual, expected)

    def test_laravel_sources_and_redaction_contract_exist(self):
        self.assertTrue((LARAVEL / "tests/Feature/BrowserSessionAuthenticationTest.php").is_file())
        self.assertTrue((LARAVEL / "tests/Feature/ParallelRunScheduleApiTest.php").is_file())
        source = (ROOT / "docs/contracts/web-smoke-matrix-evidence.md").read_text(encoding="utf-8").lower()
        for marker in ("csrf", "cookie", "foreign-tenant", "compare_parallel_run_fixtures.py"):
            self.assertIn(marker, source)


if __name__ == "__main__":
    unittest.main()
