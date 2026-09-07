import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
WORKSPACE = ROOT.parent
PINNED_REF = "ref: 68d22b2"


class CrossRepositoryCIContractTest(unittest.TestCase):
    def test_consumer_workflows_pin_go_revision_and_run_contract_checks(self):
        workflows = {
            "idelium-docker": "go test ./internal/app ./internal/browserauth ./internal/cliapi",
            "idelium-web": "build_web_smoke_targets.py --check",
            "idelium-cli": "build_cli_smoke_targets.py --check",
        }
        for repository, check in workflows.items():
            with self.subTest(repository=repository):
                workflow = (WORKSPACE / repository / ".github/workflows/ci.yml").read_text(encoding="utf-8")
                self.assertIn("repository: ifuschini/idelium-api-go", workflow)
                self.assertIn(PINNED_REF, workflow)
                self.assertIn(check, workflow)
                self.assertNotIn("idelium-api-go:latest", workflow)


if __name__ == "__main__":
    unittest.main()
