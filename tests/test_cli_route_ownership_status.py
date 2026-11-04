import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "build_cli_route_ownership_status.py"


class CliRouteOwnershipStatusTest(unittest.TestCase):
    def run_check(self, *targets: Path) -> subprocess.CompletedProcess:
        args = [sys.executable, str(SCRIPT), "--check", "--matrix", str(ROOT / "docs/contracts/migration-ownership-matrix.json")]
        for target in targets:
            args += ["--target", str(target)]
        return subprocess.run(args, capture_output=True, text=True, cwd=ROOT)

    def test_committed_docs_match_matrix(self) -> None:
        result = self.run_check(ROOT / "README.md", ROOT / "docs/migration/progress.md")
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_drift_fails(self) -> None:
        text = (ROOT / "README.md").read_text(encoding="utf-8")
        with tempfile.TemporaryDirectory() as tmp:
            target = Path(tmp) / "README.md"
            target.write_text(text.replace("`implemented` | Go", "`owned` | Go", 1), encoding="utf-8")
            result = self.run_check(target)
        self.assertEqual(result.returncode, 1)
        self.assertIn("diverges", result.stderr)

    def test_no_stale_claim(self) -> None:
        readme = (ROOT / "README.md").read_text(encoding="utf-8")
        self.assertNotIn("No Laravel business route is owned by Go yet", readme)
        matrix = json.loads((ROOT / "docs/contracts/migration-ownership-matrix.json").read_text())
        self.assertTrue(all(a["rollout_state"] == "laravel-primary" for a in matrix["aggregates"]))


if __name__ == "__main__":
    unittest.main()
