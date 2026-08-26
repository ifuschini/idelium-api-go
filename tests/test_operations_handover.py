import json
import subprocess
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
HANDOVER_PATH = ROOT / "docs/contracts/operations-handover.json"


def load_json(path):
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


class OperationsHandoverTests(unittest.TestCase):
    def test_handover_pack_is_generated_and_current(self):
        result = subprocess.run(
            ["python3", "scripts/build_operations_handover.py", "--check"],
            cwd=ROOT,
            text=True,
            capture_output=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)

    def test_handover_remains_blocked_until_all_gates_are_ready(self):
        handover = load_json(HANDOVER_PATH)

        self.assertEqual(handover["status"], "blocked")
        self.assertFalse(handover["production_enabled"])
        self.assertEqual(handover["gate_statuses"]["schema_freeze"], "frozen")
        self.assertIn("route_cutover", handover["gate_statuses"])
        self.assertGreaterEqual(len(handover["blockers"]), 1)

    def test_backup_recovery_and_release_controls_are_recorded(self):
        handover = load_json(HANDOVER_PATH)

        self.assertTrue(handover["backup"]["required_before_window"])
        self.assertTrue(handover["backup"]["restore_test_required"])
        self.assertIn("application database snapshot", handover["backup"]["scope"])
        self.assertFalse(handover["recovery"]["database_restore_required_for_route_rollback"])
        self.assertFalse(handover["recovery"]["reverse_application_replay_allowed"])
        self.assertEqual(handover["release"]["docker_default_target"], "idelium/api-go")
        self.assertIn("cli-smoke", handover["release"]["smoke_required"])

    def test_operations_include_maintenance_controls_and_rollback_steps(self):
        handover = load_json(HANDOVER_PATH)

        self.assertLessEqual(handover["operations"]["maintenance_max_minutes"], 60)
        self.assertIn("queue-drain", handover["operations"]["controls"])
        self.assertIn("gateway-switchback", handover["operations"]["rollback_steps"])

    def test_handover_pack_contains_no_sensitive_values(self):
        text = HANDOVER_PATH.read_text(encoding="utf-8").lower()
        for marker in ["bearer ", "idelium-key:", "api_key=", "password="]:
            self.assertNotIn(marker, text)


if __name__ == "__main__":
    unittest.main()
