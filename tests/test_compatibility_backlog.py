import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT_PATH = ROOT / "scripts" / "build_compatibility_backlog.py"
SPEC = importlib.util.spec_from_file_location("build_compatibility_backlog", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class CompatibilityBacklogTest(unittest.TestCase):
    def setUp(self):
        self.inventory = MODULE.load_json(
            ROOT / "docs" / "contracts" / "laravel-routes.json"
        )
        self.consumer_map = MODULE.load_json(
            ROOT / "docs" / "contracts" / "consumer-route-map.json"
        )
        operations = MODULE.openapi_operations(
            (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
        )
        self.backlog = MODULE.build_backlog(
            self.inventory,
            self.consumer_map,
            operations,
        )

    def item(self, method, path):
        return next(
            item
            for item in self.backlog["items"]
            if item["method"] == method and item["path"] == path
        )

    def test_creates_a_record_for_every_non_development_route(self):
        self.assertEqual(168, self.backlog["public_route_count"])
        self.assertEqual(3, self.backlog["excluded_development_route_count"])
        self.assertEqual(
            self.inventory["route_count"],
            self.backlog["public_route_count"]
            + self.backlog["excluded_development_route_count"],
        )
        self.assertEqual(
            len(self.backlog["items"]),
            len({item["id"] for item in self.backlog["items"]}),
        )

    def test_marks_every_production_visible_route_as_openapi_documented(self):
        statuses = {item["openapi_status"] for item in self.backlog["items"]}
        self.assertEqual({"documented"}, statuses)
        self.assertEqual(168, len(self.backlog["items"]))

    def test_current_go_openapi_operations_remain_documented(self):
        self.assertEqual(
            "documented",
            self.item("GET|HEAD", "/api/admin/platforms/types")["openapi_status"],
        )
        self.assertEqual(
            "documented",
            self.item("GET|HEAD", "/api/admin/platforms/os/{idType}")["openapi_status"],
        )

    def test_routes_are_assigned_to_expected_migration_waves(self):
        cases = [
            ("GET|HEAD", "/api/admin/platforms/status", 3),
            ("GET|HEAD", "/api/ideliumcl/step/{idStep}", 4),
            ("POST", "/api/ideliumcl/step", 5),
            ("POST", "/api/admin/tests", 6),
            ("GET|HEAD", "/api/admin/testsperfomed/{idTestPerformed}", 7),
            ("POST", "/api/ideliumrunner/claim", 8),
            (
                "POST",
                "/api/ideliumcl/projects/{idProject}/parallel-runs",
                8,
            ),
            ("POST", "/api/login", 9),
            ("GET|HEAD", "/", 0),
        ]
        for method, path, wave in cases:
            with self.subTest(path=path):
                self.assertEqual(wave, self.item(method, path)["migration_wave"])

    def test_every_record_has_all_contract_gates(self):
        expected = {
            "request-and-validation",
            "response-and-status-codes",
            "authorization-and-tenant-isolation",
            "side-effects-and-idempotency",
            "redaction-and-audit",
            "sanitized-laravel-fixture",
            "laravel-go-differential-test",
            "consumer-smoke-test",
            "rollout-and-rollback",
        }
        self.assertTrue(
            all(
                set(item["required_contract_evidence"]) == expected
                for item in self.backlog["items"]
            )
        )

    def test_committed_backlog_matches_current_inputs(self):
        committed = json.loads(
            (ROOT / "docs" / "contracts" / "compatibility-backlog.json").read_text(
                encoding="utf-8"
            )
        )
        self.assertEqual(self.backlog, committed)


if __name__ == "__main__":
    unittest.main()
