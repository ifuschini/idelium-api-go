import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "compare_parallel_run_fixtures.py"


class ParallelRunFixtureComparatorTests(unittest.TestCase):
    def test_reports_missing_go_fixture_without_leaking_payloads(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            laravel = root / "laravel"
            go = root / "go"
            laravel.mkdir()
            go.mkdir()
            fixture = {
                "route": {"method": "GET", "path": "/parallel-runs", "trustPath": "api-key", "tenantOwned": True},
                "response": {"status": 200, "headers": {}, "body": {}},
                "sideEffects": [],
            }
            (laravel / "list.fixture.json").write_text(json.dumps(fixture), encoding="utf-8")
            plan = root / "plan.json"
            plan.write_text(json.dumps({"routes": [{"id": "list", "method": "GET", "output": "list.fixture.json"}]}), encoding="utf-8")
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--plan", str(plan), "--laravel-dir", str(laravel), "--go-dir", str(go)],
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(result.returncode, 1)
            self.assertIn("missing Go fixture", result.stderr)
            self.assertNotIn("token", result.stderr.lower())

    def test_compares_a_safe_read_pair(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            laravel = root / "laravel"
            go = root / "go"
            laravel.mkdir()
            go.mkdir()
            fixture = {
                "route": {"method": "GET", "path": "/parallel-runs", "trustPath": "api-key", "tenantOwned": True},
                "response": {"status": 200, "headers": {"Content-Type": "application/json"}, "body": {"data": []}},
                "sideEffects": [],
            }
            (laravel / "list.fixture.json").write_text(json.dumps(fixture), encoding="utf-8")
            (go / "list-go.fixture.json").write_text(json.dumps(fixture), encoding="utf-8")
            plan = root / "plan.json"
            plan.write_text(json.dumps({"routes": [{"id": "list", "method": "GET", "output": "list.fixture.json"}]}), encoding="utf-8")
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--plan", str(plan), "--laravel-dir", str(laravel), "--go-dir", str(go)],
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("Compared 1", result.stdout)


if __name__ == "__main__":
    unittest.main()
