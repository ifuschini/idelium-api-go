import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT_PATH = ROOT / "scripts" / "sync_openapi_legacy_contracts.py"
SPEC = importlib.util.spec_from_file_location("sync_openapi_legacy_contracts", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class OpenAPILegacyContractsTest(unittest.TestCase):
    def setUp(self):
        self.openapi = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
        self.routes = MODULE.load_inventory(ROOT / "docs" / "contracts" / "laravel-routes.json")
        consumer_index = MODULE.load_consumer_index(
            ROOT / "docs" / "contracts" / "consumer-route-map.json"
        )
        for route in self.routes:
            route["consumer_ids"] = consumer_index.get((route["method"], route["path"]), [])

    def test_openapi_is_synchronized_with_laravel_inventory(self):
        generated = MODULE.insert_generated_block(
            self.openapi,
            MODULE.generated_paths(self.routes, MODULE.existing_paths(self.openapi)),
        )
        self.assertEqual(self.openapi, generated)

    def test_every_production_visible_laravel_route_has_an_openapi_operation(self):
        operations = self._openapi_operations(self.openapi)
        missing = []
        for route in self.routes:
            path = MODULE.gateway_path(route["path"])
            if any((method, path) in operations for method in MODULE.route_methods(route["method"])):
                continue
            missing.append(f"{route['method']} {route['path']}")
        self.assertEqual([], missing)

    def test_generated_contracts_preserve_migration_metadata(self):
        self.assertIn("x-idelium-laravel-route", self.openapi)
        self.assertIn("x-idelium-authentication-mode", self.openapi)
        self.assertIn("x-idelium-tenant-context", self.openapi)
        self.assertIn("x-idelium-consumers", self.openapi)
        self.assertIn("LegacyCompatibilityResponse", self.openapi)

    def test_no_development_only_routes_are_documented(self):
        inventory = json.loads(
            (ROOT / "docs" / "contracts" / "laravel-routes.json").read_text(
                encoding="utf-8"
            )
        )
        development_paths = {
            MODULE.gateway_path(route["path"])
            for route in inventory["routes"]
            if route["authentication_mode"] == "development-only"
        }
        documented_paths = {
            path
            for _, path in self._openapi_operations(self.openapi)
        }
        self.assertTrue(development_paths.isdisjoint(documented_paths))

    def _openapi_operations(self, source):
        operations = set()
        current_path = None
        for line in source.splitlines():
            if line.startswith("  /") and line.rstrip().endswith(":"):
                current_path = line.strip()[:-1]
                continue
            if current_path and line.startswith("    "):
                method = line.strip()[:-1].upper()
                if method in MODULE.HTTP_METHODS:
                    operations.add((method, current_path))
        return operations


if __name__ == "__main__":
    unittest.main()
