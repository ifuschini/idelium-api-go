from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[1]
OPENAPI = ROOT / "api" / "openapi.yaml"


class ServiceAccountCutoverGateTest(unittest.TestCase):
    def setUp(self):
        self.source = OPENAPI.read_text(encoding="utf-8")

    def test_service_account_routes_document_go_cutover_gate(self):
        expected_paths = [
            "/admin/service-accounts",
            "/admin/service-accounts/{serviceAccount}/revoke",
        ]
        for path in expected_paths:
            with self.subTest(path=path):
                index = self.source.find(f"  {path}:")
                self.assertNotEqual(-1, index, f"{path} is missing from OpenAPI")
                block = self.source[index : self.source.find("\n  /", index + 1)]
                self.assertIn("x-idelium-go-cutover-gate: true", block)
                self.assertIn(
                    'x-idelium-go-cutover-error-code: "SERVICE_ACCOUNT_MIGRATION_DISABLED"',
                    block,
                )
                self.assertIn('"501":', block)


if __name__ == "__main__":
    unittest.main()
