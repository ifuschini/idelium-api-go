import json
import subprocess
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PLAN_PATH = ROOT / "docs/contracts/docker-default-image-switch.json"


def load_json(path):
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


class DockerDefaultImageSwitchTests(unittest.TestCase):
    def test_switch_plan_is_generated_and_current(self):
        result = subprocess.run(
            ["python3", "scripts/build_docker_default_image_switch.py", "--check"],
            cwd=ROOT,
            text=True,
            capture_output=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)

    def test_switch_remains_blocked_until_cutover_gates_are_ready(self):
        plan = load_json(PLAN_PATH)

        self.assertEqual(plan["status"], "blocked")
        self.assertFalse(plan["production_enabled"])
        self.assertEqual(plan["current_default"]["fallback_owner"], "laravel")
        self.assertGreater(plan["preconditions"]["laravel_blocker_routes"], 0)
        self.assertTrue(
            any(blocker["control"] == "staging-route-cutover" for blocker in plan["blockers"])
        )

    def test_target_image_policy_is_reproducible_and_non_root(self):
        plan = load_json(PLAN_PATH)
        target = plan["target_default"]

        self.assertEqual(target["api_service"], "idelium/api-go")
        self.assertEqual(target["image_reference_policy"], "pin-by-immutable-digest")
        self.assertEqual(target["runtime_user"], "65532:65532")
        self.assertEqual(target["readiness_path"], "/readyz")
        self.assertEqual(target["liveness_path"], "/healthz")

    def test_rollback_does_not_require_database_restore_or_dual_writes(self):
        plan = load_json(PLAN_PATH)

        self.assertFalse(plan["rollback"]["requires_database_restore"])
        self.assertFalse(plan["rollback"]["dual_writes_allowed"])
        self.assertIn("Laravel API image", plan["rollback"]["strategy"])

    def test_switch_plan_contains_no_sensitive_values(self):
        text = PLAN_PATH.read_text(encoding="utf-8").lower()
        for marker in ["bearer ", "idelium-key:", "api_key=", "password="]:
            self.assertNotIn(marker, text)


if __name__ == "__main__":
    unittest.main()
