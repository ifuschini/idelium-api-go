import json
import subprocess
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MANIFEST_PATH = ROOT / "docs/contracts/staging-route-cutover.json"
MATRIX_PATH = ROOT / "docs/contracts/migration-ownership-matrix.json"


def load_json(path):
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


class StagingRouteCutoverTests(unittest.TestCase):
    def test_manifest_is_generated_and_current(self):
        result = subprocess.run(
            ["python3", "scripts/build_staging_route_cutover.py", "--check"],
            cwd=ROOT,
            text=True,
            capture_output=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)

    def test_staging_cutover_stays_blocked_until_all_routes_are_covered(self):
        manifest = load_json(MANIFEST_PATH)

        self.assertEqual(manifest["status"], "blocked")
        self.assertFalse(manifest["production_enabled"])
        self.assertFalse(manifest["policy"]["dual_writes_allowed"])
        self.assertTrue(manifest["policy"]["production_cutover_requires_zero_blockers"])
        self.assertGreater(manifest["summary"]["laravel_blocker_routes"], 0)
        self.assertEqual(
            manifest["summary"]["laravel_blocker_routes"],
            len(manifest["blockers"]),
        )

    def test_go_owned_routes_are_ready_for_staging(self):
        manifest = load_json(MANIFEST_PATH)
        matrix = load_json(MATRIX_PATH)
        matrix_go_routes = {
            route["route_id"]
            for route in matrix["routes"]
            if route["owner"] == "go"
        }
        ready_routes = {
            route["route_id"]
            for route in manifest["routes"]
            if route["staging_state"] == "ready"
        }

        self.assertEqual(matrix_go_routes, ready_routes)
        self.assertEqual(manifest["summary"]["go_owned_routes"], len(matrix_go_routes))

    def test_fail_closed_routes_have_stable_diagnostics(self):
        manifest = load_json(MANIFEST_PATH)
        gated_routes = [
            route
            for route in manifest["routes"]
            if route["staging_state"] == "gated"
        ]

        self.assertGreater(len(gated_routes), 0)
        for route in gated_routes:
            self.assertEqual(route["staging_owner"], "go-fail-closed")
            self.assertEqual(route["routing_action"], "send-to-go-gate")
            self.assertRegex(route["error_code"], r"^[A-Z0-9_]+$")

    def test_manifest_contains_no_sensitive_values(self):
        text = MANIFEST_PATH.read_text(encoding="utf-8").lower()
        for marker in ["bearer ", "idelium-key:", "api_key=", "password="]:
            self.assertNotIn(marker, text)


if __name__ == "__main__":
    unittest.main()
