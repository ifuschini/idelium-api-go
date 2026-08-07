import pathlib
import re
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / ".github" / "workflows" / "ci.yml"


class CIContractTests(unittest.TestCase):
    def setUp(self) -> None:
        self.workflow = WORKFLOW.read_text(encoding="utf-8")

    def test_ci_has_bounded_independent_gates(self) -> None:
        for job in ("quality", "integration", "image"):
            self.assertRegex(self.workflow, rf"(?m)^  {job}:$")
        self.assertEqual(3, self.workflow.count("timeout-minutes: 15"))
        self.assertIn("cancel-in-progress: true", self.workflow)

    def test_quality_gate_covers_every_required_check(self) -> None:
        for command in (
            "go mod verify",
            "make format-check",
            "make vet",
            "make test",
            "make test-race",
            "make contract-test",
            "make build",
        ):
            self.assertIn(command, self.workflow)

    def test_actions_and_service_images_are_immutable(self) -> None:
        action_references = re.findall(r"uses:\s+([^\s]+)", self.workflow)
        self.assertGreater(len(action_references), 0)
        for reference in action_references:
            self.assertRegex(reference, r"@[0-9a-f]{40}$")

        self.assertRegex(
            self.workflow,
            r"image: mariadb:10\.6\.22@sha256:[0-9a-f]{64}",
        )
        self.assertIn("go-version: 1.26.5", self.workflow)

    def test_image_gate_builds_and_checks_non_root_user(self) -> None:
        self.assertIn("docker build", self.workflow)
        self.assertIn("docker image inspect", self.workflow)
        self.assertIn('= "65532:65532"', self.workflow)

    def test_supply_chain_gates_are_versioned_and_fail_closed(self) -> None:
        self.assertIn("golang.org/x/vuln/cmd/govulncheck@v1.1.4", self.workflow)
        self.assertRegex(
            self.workflow,
            r"anchore/syft:v1\.50\.0@sha256:[0-9a-f]{64}",
        )
        self.assertRegex(
            self.workflow,
            r"aquasec/trivy:0\.73\.0@sha256:[0-9a-f]{64}",
        )
        self.assertIn("--output cyclonedx-json", self.workflow)
        self.assertIn("--exit-code 1", self.workflow)
        self.assertIn("--severity HIGH,CRITICAL", self.workflow)
        self.assertIn("if-no-files-found: error", self.workflow)


if __name__ == "__main__":
    unittest.main()
