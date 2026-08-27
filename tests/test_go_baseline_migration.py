import importlib.util
import json
from pathlib import Path
import subprocess
import unittest


ROOT = Path(__file__).resolve().parents[1]
SCRIPT_PATH = ROOT / "scripts" / "build_go_baseline_migration.py"
SPEC = importlib.util.spec_from_file_location("build_go_baseline_migration", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class GoBaselineMigrationTest(unittest.TestCase):
    def setUp(self):
        self.manifest_path = ROOT / "docs" / "contracts" / "go-baseline-migration.json"
        self.embedded_path = ROOT / "internal" / "migrations" / "baseline_manifest.json"
        self.manifest = json.loads(self.manifest_path.read_text(encoding="utf-8"))

    def test_manifest_tracks_current_laravel_migration_source(self):
        source_dir = ROOT.parent / "idelium-api" / "database" / "migrations"
        if not source_dir.is_dir():
            self.skipTest("Laravel migration source tree is not mounted")
        expected = MODULE.build_manifest(source_dir)
        self.assertEqual(expected, self.manifest)

    def test_embedded_manifest_matches_reviewed_contract_manifest(self):
        embedded = json.loads(self.embedded_path.read_text(encoding="utf-8"))
        self.assertEqual(self.manifest, embedded)

    def test_manifest_is_policy_safe_before_cutover(self):
        policy = self.manifest["handover_policy"]
        self.assertTrue(policy["laravel_remains_schema_owner"])
        self.assertFalse(policy["go_baseline_application_enabled"])
        self.assertFalse(policy["dual_writes_allowed"])
        self.assertEqual(69, self.manifest["migration_count"])
        self.assertEqual(69, len(self.manifest["migrations"]))
        serialized = json.dumps(self.manifest).lower()
        for unsafe_marker in ("password=", "authorization:", "cookie:", "bearer "):
            self.assertNotIn(unsafe_marker, serialized)

    def test_generator_check_detects_stale_artifacts(self):
        result = subprocess.run(
            [
                "python3",
                str(SCRIPT_PATH),
                "--source-dir",
                str(ROOT.parent / "idelium-api" / "database" / "migrations"),
                "--output-json",
                str(self.manifest_path),
                "--output-markdown",
                str(ROOT / "docs" / "contracts" / "go-baseline-migration.md"),
                "--output-embedded-json",
                str(self.embedded_path),
                "--check",
            ],
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual("", result.stderr)
        self.assertEqual(0, result.returncode, result.stderr + result.stdout)


if __name__ == "__main__":
    unittest.main()
