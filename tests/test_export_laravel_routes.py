import hashlib
import importlib.util
import json
import sys
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).parents[1] / "scripts" / "export_laravel_routes.py"
SPEC = importlib.util.spec_from_file_location("export_laravel_routes", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class LaravelRouteExportTest(unittest.TestCase):
    def route(self, uri, action="App\\Http\\Controllers\\Example@index", middleware=None):
        return {
            "method": "GET|HEAD",
            "uri": uri,
            "name": "example.index",
            "action": action,
            "middleware": middleware or ["api"],
            "path": None,
        }

    def test_classifies_every_supported_trust_path(self):
        cases = [
            (
                self.route(
                    "api/admin/projects",
                    middleware=[
                        "api",
                        "App\\Http\\Middleware\\Authenticate:sanctum",
                        "App\\Http\\Middleware\\ResolveTenantContext",
                    ],
                ),
                "browser-session",
                True,
            ),
            (
                self.route(
                    "api/ideliumcl/test/1",
                    middleware=["api", "App\\Http\\Middleware\\AuthenticateIdeliumKey"],
                ),
                "api-key",
                False,
            ),
            (self.route("api/ideliumrunner/claim"), "run-token", False),
            (self.route("api/oidc/token-exchange"), "workload-identity-exchange", False),
            (self.route("api/sso/1/start"), "sso-bootstrap-or-callback", False),
            (self.route("api/login"), "browser-auth-bootstrap", False),
            (
                self.route(
                    "_ignition/health-check",
                    action="Spatie\\LaravelIgnition\\HealthCheckController",
                ),
                "development-only",
                False,
            ),
            (self.route("/", action="Closure"), "public", False),
        ]

        for route, expected_mode, expected_tenant_context in cases:
            with self.subTest(uri=route["uri"]):
                exported = MODULE.normalize_route(route)
                self.assertEqual(expected_mode, exported["authentication_mode"])
                self.assertEqual(expected_tenant_context, exported["tenant_context"])

    def test_rejects_duplicate_method_and_path(self):
        route = self.route("api/example")
        with self.assertRaisesRegex(ValueError, "Duplicate method/path"):
            MODULE.normalize_routes([route, route])

    def test_committed_inventory_is_complete_and_self_consistent(self):
        inventory_path = (
            Path(__file__).parents[1]
            / "docs"
            / "contracts"
            / "laravel-routes.json"
        )
        inventory = json.loads(inventory_path.read_text(encoding="utf-8"))
        routes = inventory["routes"]

        self.assertEqual(171, inventory["route_count"])
        self.assertEqual(inventory["route_count"], len(routes))
        self.assertTrue(all(route["authentication_mode"] for route in routes))
        self.assertTrue(all(route["current_owner"] for route in routes))
        self.assertTrue(all(route["controller"] for route in routes))
        identities = {(route["method"], route["path"]) for route in routes}
        self.assertEqual(len(routes), len(identities))

        payload = json.dumps(routes, sort_keys=True, separators=(",", ":")).encode()
        self.assertEqual(
            inventory["route_digest_sha256"],
            hashlib.sha256(payload).hexdigest(),
        )


if __name__ == "__main__":
    unittest.main()
