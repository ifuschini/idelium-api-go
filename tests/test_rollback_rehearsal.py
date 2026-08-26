import json
import subprocess
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PLAN_PATH = ROOT / "docs/contracts/rollback-rehearsal.json"


def load_json(path):
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


class RollbackRehearsalTests(unittest.TestCase):
    def test_rehearsal_plan_is_generated_and_current(self):
        result = subprocess.run(
            ["python3", "scripts/build_rollback_rehearsal.py", "--check"],
            cwd=ROOT,
            text=True,
            capture_output=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)

    def test_rehearsal_remains_blocked_until_prerequisites_are_ready(self):
        plan = load_json(PLAN_PATH)

        self.assertEqual(plan["status"], "blocked")
        self.assertFalse(plan["production_enabled"])
        self.assertEqual(plan["target"]["gateway_owner_after_rollback"], "laravel")
        self.assertGreater(plan["preconditions"]["laravel_blocker_routes"], 0)
        self.assertGreaterEqual(len(plan["blockers"]), 1)

    def test_rehearsal_steps_are_ordered_and_complete(self):
        plan = load_json(PLAN_PATH)
        steps = plan["steps"]
        self.assertEqual([step["order"] for step in steps], list(range(1, len(steps) + 1)))
        step_names = {step["name"] for step in steps}

        self.assertIn("freeze-forward-rollout", step_names)
        self.assertIn("gateway-switchback", step_names)
        self.assertIn("restore-laravel-api-image", step_names)
        self.assertIn("resume-laravel-processing", step_names)
        self.assertIn("smoke-and-observe", step_names)
        self.assertIn("record-rehearsal-evidence", step_names)

    def test_rehearsal_safety_forbids_restore_replay_and_dual_writes(self):
        plan = load_json(PLAN_PATH)
        safety = plan["safety"]

        self.assertFalse(safety["requires_database_restore"])
        self.assertFalse(safety["reverse_application_replay_allowed"])
        self.assertFalse(safety["dual_writes_allowed"])
        self.assertLessEqual(safety["max_recovery_minutes"], 30)

    def test_rehearsal_plan_contains_no_sensitive_values(self):
        text = PLAN_PATH.read_text(encoding="utf-8").lower()
        for marker in ["bearer ", "idelium-key:", "api_key=", "password="]:
            self.assertNotIn(marker, text)


if __name__ == "__main__":
    unittest.main()
