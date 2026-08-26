import json
import subprocess
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PLAN_PATH = ROOT / "docs/contracts/laravel-readonly-maintenance.json"
CUTOVER_PATH = ROOT / "docs/contracts/staging-route-cutover.json"


def load_json(path):
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


class LaravelReadonlyMaintenanceTests(unittest.TestCase):
    def test_maintenance_plan_is_generated_and_current(self):
        result = subprocess.run(
            ["python3", "scripts/build_laravel_readonly_maintenance.py", "--check"],
            cwd=ROOT,
            text=True,
            capture_output=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)

    def test_maintenance_is_blocked_until_cutover_blockers_are_resolved(self):
        plan = load_json(PLAN_PATH)
        cutover = load_json(CUTOVER_PATH)

        self.assertEqual(plan["status"], "blocked")
        self.assertFalse(plan["production_enabled"])
        self.assertEqual(
            plan["preconditions"]["laravel_blocker_routes"],
            cutover["summary"]["laravel_blocker_routes"],
        )
        self.assertGreater(plan["preconditions"]["laravel_blocker_routes"], 0)
        self.assertTrue(
            any(blocker["control"] == "route-cutover" for blocker in plan["blockers"])
        )

    def test_maintenance_window_is_time_boxed_and_reversible(self):
        plan = load_json(PLAN_PATH)

        self.assertLessEqual(plan["timebox"]["max_duration_minutes"], 60)
        self.assertTrue(plan["timebox"]["requires_approved_start"])
        self.assertTrue(plan["timebox"]["requires_approved_end"])
        self.assertFalse(plan["rollback"]["requires_database_restore"])
        self.assertFalse(plan["rollback"]["dual_writes_allowed"])

    def test_required_controls_are_present(self):
        plan = load_json(PLAN_PATH)
        control_names = {control["name"] for control in plan["controls"]}

        self.assertIn("gateway-mutation-block", control_names)
        self.assertIn("queue-drain", control_names)
        self.assertIn("scheduled-job-pause", control_names)
        self.assertIn("go-route-verification", control_names)
        self.assertIn("operator-broadcast", control_names)

    def test_maintenance_plan_contains_no_sensitive_values(self):
        text = PLAN_PATH.read_text(encoding="utf-8").lower()
        for marker in ["bearer ", "idelium-key:", "api_key=", "password="]:
            self.assertNotIn(marker, text)


if __name__ == "__main__":
    unittest.main()
