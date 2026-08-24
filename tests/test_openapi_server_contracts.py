import importlib.util
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT_PATH = ROOT / "scripts" / "generate_openapi_server_contracts.py"
SPEC = importlib.util.spec_from_file_location("generate_openapi_server_contracts", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class OpenAPIServerContractsTest(unittest.TestCase):
    def setUp(self):
        self.openapi = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
        self.generated = (
            ROOT / "internal" / "openapicontract" / "generated_routes.go"
        ).read_text(encoding="utf-8")
        self.operations = MODULE.openapi_operations(self.openapi)

    def test_generated_go_contract_is_synchronized_with_openapi(self):
        self.assertEqual(self.generated, MODULE.render(self.operations))

    def test_generated_contract_contains_every_openapi_operation(self):
        self.assertEqual(171, len(self.operations))
        for operation in self.operations:
            with self.subTest(operation=operation["operation_id"]):
                self.assertIn(
                    f'OperationID:        "{operation["operation_id"]}"',
                    self.generated,
                )
                self.assertIn(f'Path:               "{operation["path"]}"', self.generated)

    def test_server_interface_uses_exported_operation_handlers(self):
        self.assertIn("type ServerInterface interface", self.generated)
        self.assertIn("GetLiveness(http.ResponseWriter, *http.Request)", self.generated)
        self.assertIn("ListPlatformTypes(http.ResponseWriter, *http.Request)", self.generated)
        self.assertIn("PostLogin(http.ResponseWriter, *http.Request)", self.generated)

    def test_laravel_compatibility_metadata_is_generated(self):
        self.assertIn('LaravelRoute:       "/api/login"', self.generated)
        self.assertIn('AuthenticationMode: "public"', self.generated)
        self.assertIn('AuthenticationMode: "api-key"', self.generated)
        self.assertIn("TenantContext:      true", self.generated)
        self.assertIn('Consumers:          []string{"idelium-cli", "idelium-docker-wiki"}', self.generated)


if __name__ == "__main__":
    unittest.main()
