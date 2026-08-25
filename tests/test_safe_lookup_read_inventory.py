import json
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
INVENTORY_PATH = ROOT / "docs" / "contracts" / "safe-lookup-read-inventory.json"
OWNERSHIP_PATH = ROOT / "docs" / "contracts" / "migration-ownership-matrix.json"


class SafeLookupReadInventoryTest(unittest.TestCase):
    def setUp(self):
        self.inventory = json.loads(INVENTORY_PATH.read_text(encoding="utf-8"))
        matrix = json.loads(OWNERSHIP_PATH.read_text(encoding="utf-8"))
        self.routes = matrix["routes"]

    def wave_three_reads(self):
        return {
            route["route_id"]: route
            for route in self.routes
            if route["migration_wave"] == 3 and route["method"] == "GET|HEAD"
        }

    def test_inventory_covers_every_wave_three_read_decision(self):
        migrated = {
            item["route_id"] for item in self.inventory["migrated_safe_lookup_reads"]
        }
        deferred = {item["route_id"] for item in self.inventory["deferred_candidates"]}

        self.assertEqual(set(self.wave_three_reads()), migrated | deferred)
        self.assertFalse(migrated & deferred)

    def test_migrated_safe_lookup_reads_are_go_owned(self):
        routes = self.wave_three_reads()
        for item in self.inventory["migrated_safe_lookup_reads"]:
            with self.subTest(route=item["route_id"]):
                self.assertEqual("go-owned", routes[item["route_id"]]["rollout_status"])

    def test_deferred_candidates_remain_laravel_owned_with_reason(self):
        routes = self.wave_three_reads()
        for item in self.inventory["deferred_candidates"]:
            with self.subTest(route=item["route_id"]):
                self.assertEqual(
                    "laravel-owned", routes[item["route_id"]]["rollout_status"]
                )
                self.assertTrue(item["reason"])


if __name__ == "__main__":
    unittest.main()
