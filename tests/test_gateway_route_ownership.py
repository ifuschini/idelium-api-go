import json
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
GATEWAY_PATH = ROOT / "docs" / "contracts" / "gateway-route-ownership.json"
ROLLOUT_PATH = ROOT / "docs" / "contracts" / "route-rollout-overrides.json"


class GatewayRouteOwnershipTest(unittest.TestCase):
    def setUp(self):
        self.gateway = json.loads(GATEWAY_PATH.read_text(encoding="utf-8"))
        self.rollout = json.loads(ROLLOUT_PATH.read_text(encoding="utf-8"))

    def test_gateway_routes_match_go_owned_platform_catalog_rollout(self):
        configured = {route["route_id"] for route in self.gateway["routes"]}
        go_owned_catalog_routes = {
            route_id
            for route_id, owner in self.rollout["routes"].items()
            if owner == "go-owned" and "/api/admin/platforms/" in route_id
        }

        self.assertEqual(go_owned_catalog_routes, configured)

    def test_gateway_routes_are_read_only_and_reversible(self):
        for route in self.gateway["routes"]:
            with self.subTest(route=route["route_id"]):
                self.assertEqual(["GET", "HEAD"], route["methods"])
                self.assertEqual("go", route["owner"])
                self.assertEqual("laravel", route["fallback_owner"])
                self.assertTrue(route["public_path"].startswith("/api/"))
                self.assertEqual(route["public_path"][4:], route["go_path"])

    def test_gateway_upstreams_are_externalized(self):
        upstreams = self.gateway["upstreams"]

        self.assertEqual("IDELIUM_API_GO_BASE_URL", upstreams["go"]["base_url_env"])
        self.assertEqual(
            "IDELIUM_LARAVEL_BASE_URL", upstreams["laravel"]["base_url_env"]
        )
        self.assertFalse(self.gateway["rollback"]["requires_database_restore"])
        self.assertFalse(self.gateway["rollback"]["requires_dual_writes"])


if __name__ == "__main__":
    unittest.main()
