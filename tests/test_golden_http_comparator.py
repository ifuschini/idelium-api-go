import copy
import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT_PATH = ROOT / "scripts" / "compare_golden_http.py"
SPEC = importlib.util.spec_from_file_location("compare_golden_http", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


def fixture():
    return {
        "fixtureVersion": "1.0",
        "id": "platform-status-authorized",
        "route": {
            "method": "GET",
            "path": "/api/admin/platforms/status",
            "trustPath": "browser-session",
            "tenantOwned": True,
        },
        "request": {
            "headers": {
                "Authorization": "Bearer secret-token",
                "Accept": "application/json",
            },
            "query": {},
            "body": None,
        },
        "response": {
            "status": 200,
            "headers": {
                "Content-Type": "application/json",
                "X-Correlation-ID": "not-compared",
            },
            "body": [{"id": 1, "name": "active"}],
        },
        "normalizations": [],
        "sideEffects": [],
    }


class GoldenHTTPComparatorTest(unittest.TestCase):
    def test_identical_safe_read_fixtures_pass(self):
        result = MODULE.compare(fixture(), fixture())
        self.assertTrue(result.passed)
        self.assertEqual([], result.differences)

    def test_committed_platform_status_fixtures_match(self):
        expected = json.loads(
            (ROOT / "testdata" / "golden" / "platform-status.fixture.json").read_text(
                encoding="utf-8"
            )
        )
        actual = json.loads(
            (
                ROOT / "testdata" / "golden" / "platform-status-go.fixture.json"
            ).read_text(encoding="utf-8")
        )

        result = MODULE.compare(expected, actual)

        self.assertTrue(result.passed)

    def test_committed_parallel_run_results_fixtures_match(self):
        expected = json.loads((ROOT / "testdata" / "golden" / "parallel-run-results.fixture.json").read_text(encoding="utf-8"))
        actual = json.loads((ROOT / "testdata" / "golden" / "parallel-run-results-go.fixture.json").read_text(encoding="utf-8"))
        result = MODULE.compare(expected, actual)
        self.assertTrue(result.passed)

    def test_status_header_and_body_mismatches_are_reported_by_path(self):
        expected = fixture()
        actual = fixture()
        actual["response"]["status"] = 404
        actual["response"]["headers"]["Content-Type"] = "text/plain"
        actual["response"]["body"][0]["name"] = "inactive"

        result = MODULE.compare(expected, actual)

        self.assertFalse(result.passed)
        paths = {difference.path for difference in result.differences}
        self.assertIn("$.response.status", paths)
        self.assertIn("$.response.headers.content-type", paths)
        self.assertIn("$.response.body[0].name", paths)

    def test_declared_nondeterministic_values_are_normalized_before_compare(self):
        expected = fixture()
        actual = fixture()
        expected["response"]["body"][0] = {
            "id": "123e4567-e89b-12d3-a456-426614174000",
            "createdAt": "2026-01-01T00:00:00Z",
            "correlationId": "correlation-laravel",
        }
        actual["response"]["body"][0] = {
            "id": "987e6543-e21b-12d3-a456-426614174999",
            "createdAt": "2026-08-24T12:30:00Z",
            "correlationId": "correlation-go",
        }
        normalizations = [
            {"path": "$.response.body[0].id", "rule": "Normalize generated UUID."},
            {"path": "$.response.body[0].createdAt", "rule": "Normalize timestamp."},
            {
                "path": "$.response.body[0].correlationId",
                "rule": "Normalize correlation identifier.",
            },
        ]
        expected["normalizations"] = normalizations
        actual["normalizations"] = normalizations

        result = MODULE.compare(expected, actual)

        self.assertTrue(result.passed)
        self.assertEqual([], result.differences)

    def test_missing_declared_normalization_path_fails_safely(self):
        expected = fixture()
        actual = fixture()
        expected["normalizations"] = [
            {"path": "$.response.body[0].missingId", "rule": "Normalize UUID."}
        ]

        result = MODULE.compare(expected, actual)

        self.assertFalse(result.passed)
        self.assertEqual("$.response.body[0].missingId", result.differences[0].path)

    def test_mutations_are_rejected_by_safe_read_comparator(self):
        expected = fixture()
        actual = fixture()
        expected["route"]["method"] = "POST"

        result = MODULE.compare(expected, actual)

        self.assertFalse(result.passed)
        self.assertEqual("$.expected.route.method", result.differences[0].path)

    def test_side_effects_are_rejected_by_safe_read_comparator(self):
        expected = fixture()
        actual = fixture()
        actual["sideEffects"] = [{"table": "projects", "operation": "insert"}]

        result = MODULE.compare(expected, actual)

        self.assertFalse(result.passed)
        self.assertEqual("$.actual.sideEffects", result.differences[0].path)

    def test_diagnostics_do_not_include_sensitive_values(self):
        expected = fixture()
        actual = copy.deepcopy(expected)
        actual["response"]["body"][0]["name"] = "changed"

        result = MODULE.compare(expected, actual).to_dict()
        serialized = str(result)

        self.assertNotIn("Bearer secret-token", serialized)
        self.assertNotIn("changed", serialized)
        self.assertIn("$.response.body[0].name", serialized)


if __name__ == "__main__":
    unittest.main()
