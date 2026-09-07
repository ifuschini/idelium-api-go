import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
LARAVEL = ROOT.parent / "idelium-api"
MATRIX = ROOT / "docs" / "contracts" / "cli-runner-smoke-matrix.md"
FIXTURES = ROOT / "testdata" / "golden" / "laravel-parallel-runs"


class CliRunnerSmokeMatrixTest(unittest.TestCase):
    def test_matrix_sources_and_go_fixtures_are_present(self):
        matrix = MATRIX.read_text(encoding="utf-8")
        for marker in (
            "ParallelRunScheduleApiTest.php",
            "RunTokenTest.php",
            "runner-claim-go.fixture.json",
            "runner-heartbeat-go.fixture.json",
            "runner-update-go.fixture.json",
            "compare_parallel_run_fixtures.py",
        ):
            self.assertIn(marker, matrix)
        self.assertTrue((LARAVEL / "tests/Feature/RunTokenTest.php").is_file())
        self.assertTrue((LARAVEL / "tests/Feature/ParallelRunScheduleApiTest.php").is_file())

    def test_go_fixture_set_contains_stateful_routes_without_secrets(self):
        fixtures = list(FIXTURES.glob("*-go.fixture.json"))
        self.assertEqual(len(fixtures), 23)
        for fixture in fixtures:
            content = fixture.read_text(encoding="utf-8").lower()
            self.assertNotIn("authorization", content)
            self.assertNotIn("cookie", content)
            self.assertNotIn("password", content)


if __name__ == "__main__":
    unittest.main()
