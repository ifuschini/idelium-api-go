import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
OPENAPI = ROOT / "api" / "openapi.yaml"

GO_OWNED_CLI_READS = {
    "/ideliumcl/environment/{idEnvironment}",
    "/ideliumcl/environments/{idProject}",
    "/ideliumcl/plugin/{idPlugin}",
    "/ideliumcl/plugins/{idProject}",
    "/ideliumcl/step/{idStep}",
    "/ideliumcl/test/{idTest}",
    "/ideliumcl/testcycle/{idTestCycle}",
}


class CliMissingReferenceDiagnosticsTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.openapi = OPENAPI.read_text(encoding="utf-8")

    def operation_block(self, path):
        marker = f"  {path}:"
        start = self.openapi.index(marker)
        next_path = self.openapi.find("\n  /", start + len(marker))
        if next_path == -1:
            next_path = self.openapi.find("\ncomponents:", start)
        return self.openapi[start:next_path]

    def test_go_owned_cli_reads_preserve_legacy_invalid_id_response(self):
        for path in sorted(GO_OWNED_CLI_READS):
            with self.subTest(path=path):
                operation = self.operation_block(path)
                response_404 = operation.split('"404":', 1)[1].split('"500":', 1)[0]
                self.assertIn("LegacyMessageResponse", response_404)
                self.assertNotIn("ErrorResponse", response_404)

    def test_single_resource_reads_document_missing_or_cross_tenant_behavior(self):
        for path in sorted(path for path in GO_OWNED_CLI_READS if "/plugin/" in path or "/environment/" in path or "/step/" in path or "/test/" in path or "/testcycle/" in path):
            with self.subTest(path=path):
                text = self.operation_block(path).lower()
                self.assertTrue("missing" in text or "outside" in text)
                self.assertTrue("tenant" in text or "invalid id" in text)

    def test_go_owned_cli_reads_are_marked_tenant_contextual(self):
        for path in sorted(GO_OWNED_CLI_READS):
            with self.subTest(path=path):
                operation = self.operation_block(path)
                self.assertIn("x-idelium-tenant-context: true", operation)
                self.assertIn('x-idelium-authentication-mode: "api-key"', operation)


if __name__ == "__main__":
    unittest.main()
