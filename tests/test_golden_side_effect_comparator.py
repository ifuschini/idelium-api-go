import copy
import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT = ROOT / "scripts" / "compare_golden_side_effects.py"
SPEC = importlib.util.spec_from_file_location("compare_golden_side_effects", SCRIPT)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


def fixture():
    return {
        "fixtureVersion": "1.0",
        "id": "project-create-authorized",
        "route": {
            "method": "POST",
            "path": "/api/admin/projects",
            "trustPath": "browser-session",
            "tenantOwned": True,
        },
        "normalizations": [
            {"path": "$.sideEffects[0].row.id", "rule": "Normalize generated ID."},
            {"path": "$.sideEffects[0].row.created_at", "rule": "Normalize timestamp."},
        ],
        "sideEffects": [
            {
                "table": "projects",
                "operation": "insert",
                "tenantId": "fixture-tenant-alpha",
                "naturalKey": "fixture-project-alpha",
                "row": {
                    "id": 1001,
                    "project": "fixture-project-alpha",
                    "description": "Synthetic project created by the migration comparator.",
                    "created_at": "2026-01-01T00:00:00Z",
                },
            }
        ],
    }


class GoldenSideEffectComparatorTest(unittest.TestCase):
    def test_identical_mutation_side_effects_pass(self):
        result = MODULE.compare(fixture(), fixture())

        self.assertTrue(result.passed)
        self.assertEqual([], result.differences)

    def test_committed_project_create_fixtures_match(self):
        expected = json.loads(
            (ROOT / "testdata" / "golden" / "project-create.fixture.json").read_text(
                encoding="utf-8"
            )
        )
        actual = json.loads(
            (ROOT / "testdata" / "golden" / "project-create-go.fixture.json").read_text(
                encoding="utf-8"
            )
        )

        result = MODULE.compare(expected, actual)

        self.assertTrue(result.passed)

    def test_committed_parallel_worker_update_fixtures_match(self):
        expected = json.loads((ROOT / "testdata" / "golden" / "parallel-run-worker-update.fixture.json").read_text(encoding="utf-8"))
        actual = json.loads((ROOT / "testdata" / "golden" / "parallel-run-worker-update-go.fixture.json").read_text(encoding="utf-8"))
        result = MODULE.compare(expected, actual)
        self.assertTrue(result.passed)

    def test_side_effect_mismatches_are_reported_by_path(self):
        expected = fixture()
        actual = copy.deepcopy(expected)
        actual["sideEffects"][0]["table"] = "tests"
        actual["sideEffects"][0]["row"]["description"] = "changed"

        result = MODULE.compare(expected, actual)

        self.assertFalse(result.passed)
        paths = {difference.path for difference in result.differences}
        self.assertIn("$.sideEffects[0].table", paths)
        self.assertIn("$.sideEffects[0].row.description", paths)

    def test_declared_nondeterministic_side_effect_values_are_normalized(self):
        expected = fixture()
        actual = fixture()
        actual["sideEffects"][0]["row"]["id"] = 9999
        actual["sideEffects"][0]["row"]["created_at"] = "2026-08-24T12:30:00Z"

        result = MODULE.compare(expected, actual)

        self.assertTrue(result.passed)

    def test_missing_declared_normalization_path_fails_safely(self):
        expected = fixture()
        actual = fixture()
        expected["normalizations"].append(
            {"path": "$.sideEffects[0].row.missing_id", "rule": "Normalize generated ID."}
        )

        result = MODULE.compare(expected, actual)

        self.assertFalse(result.passed)
        self.assertEqual("$.sideEffects[0].row.missing_id", result.differences[0].path)

    def test_safe_read_fixtures_are_rejected(self):
        expected = fixture()
        actual = fixture()
        expected["route"]["method"] = "GET"

        result = MODULE.compare(expected, actual)

        self.assertFalse(result.passed)
        self.assertEqual("$.expected.route.method", result.differences[0].path)

    def test_mutation_without_side_effects_is_rejected(self):
        expected = fixture()
        actual = fixture()
        actual["sideEffects"] = []

        result = MODULE.compare(expected, actual)

        self.assertFalse(result.passed)
        self.assertEqual("$.actual.sideEffects", result.differences[0].path)

    def test_diagnostics_do_not_include_sensitive_values(self):
        expected = fixture()
        actual = fixture()
        actual["sideEffects"][0]["row"]["description"] = "secret changed value"

        result = MODULE.compare(expected, actual).to_dict()
        serialized = str(result)

        self.assertIn("$.sideEffects[0].row.description", serialized)
        self.assertNotIn("secret changed value", serialized)


if __name__ == "__main__":
    unittest.main()
