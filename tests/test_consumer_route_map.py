import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT_PATH = ROOT / "scripts" / "build_consumer_route_map.py"
SPEC = importlib.util.spec_from_file_location("build_consumer_route_map", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class ConsumerRouteMapTest(unittest.TestCase):
    def setUp(self):
        self.inventory = MODULE.load_json(
            ROOT / "docs" / "contracts" / "laravel-routes.json"
        )
        self.rules = MODULE.load_json(
            ROOT / "docs" / "contracts" / "consumer-route-rules.json"
        )
        self.route_map = MODULE.build_map(self.inventory, self.rules)

    def consumers_for(self, method, path):
        route = next(
            route
            for route in self.route_map["routes"]
            if route["method"] == method and route["path"] == path
        )
        return {consumer["id"] for consumer in route["consumers"]}

    def test_maps_representative_consumers(self):
        self.assertEqual(
            {"idelium-web", "idelium-docker"},
            self.consumers_for("GET|HEAD", "/api/sanctum/csrf-cookie"),
        )
        self.assertEqual(
            {"idelium-cli", "idelium-docker-wiki"},
            self.consumers_for("GET|HEAD", "/api/ideliumcl/step/{idStep}"),
        )
        self.assertEqual(
            {"idelium-runner"},
            self.consumers_for("POST", "/api/ideliumrunner/claim"),
        )
        self.assertEqual(
            {"idelium-web"},
            self.consumers_for("PUT", "/api/admin/platforms/os"),
        )

    def test_preserves_unmapped_routes_for_review(self):
        self.assertEqual(
            set(),
            self.consumers_for("GET|HEAD", "/api/audit-events"),
        )
        self.assertGreater(self.route_map["unmapped_route_count"], 0)

    def test_committed_map_matches_the_rules_and_inventory(self):
        committed = json.loads(
            (ROOT / "docs" / "contracts" / "consumer-route-map.json").read_text(
                encoding="utf-8"
            )
        )
        self.assertEqual(self.route_map, committed)


if __name__ == "__main__":
    unittest.main()
