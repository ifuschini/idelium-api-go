import json
import subprocess
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
FREEZE_PATH = ROOT / "docs/contracts/laravel-schema-freeze.json"
BASELINE_PATH = ROOT / "docs/contracts/go-baseline-migration.json"


def load_json(path):
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


class LaravelSchemaFreezeTests(unittest.TestCase):
    def test_freeze_report_is_generated_and_current(self):
        result = subprocess.run(
            ["python3", "scripts/check_laravel_schema_freeze.py", "--check"],
            cwd=ROOT,
            text=True,
            capture_output=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)

    def test_laravel_schema_is_frozen_against_reviewed_baseline(self):
        freeze = load_json(FREEZE_PATH)
        baseline = load_json(BASELINE_PATH)

        self.assertEqual(freeze["status"], "frozen")
        self.assertEqual(freeze["baseline_id"], baseline["baseline_id"])
        self.assertEqual(
            freeze["expected"]["aggregate_sha256"],
            baseline["aggregate_sha256"],
        )
        self.assertEqual(
            freeze["expected"]["migration_count"],
            baseline["migration_count"],
        )
        self.assertEqual(freeze["violations"], [])

    def test_freeze_policy_prevents_unsafe_schema_drift(self):
        freeze = load_json(FREEZE_PATH)
        policy = freeze["policy"]

        self.assertFalse(policy["new_laravel_migrations_allowed"])
        self.assertFalse(policy["laravel_migration_edits_allowed"])
        self.assertFalse(policy["go_baseline_application_enabled"])
        self.assertFalse(policy["dual_writes_allowed"])
        self.assertIn("versioned schema-change issue", policy["exception_process"])

    def test_freeze_report_contains_no_sensitive_values(self):
        text = FREEZE_PATH.read_text(encoding="utf-8").lower()
        for marker in ["bearer ", "idelium-key:", "api_key=", "password="]:
            self.assertNotIn(marker, text)


if __name__ == "__main__":
    unittest.main()
