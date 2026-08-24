import copy
import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT = ROOT / "scripts" / "build_cli_smoke_targets.py"
PLAN = ROOT / "docs" / "contracts" / "cli-smoke-targets.json"
DOC = ROOT / "docs" / "contracts" / "cli-smoke-targets.md"
SPEC = importlib.util.spec_from_file_location("build_cli_smoke_targets", SCRIPT)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class CliSmokeTargetsTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.consumer_map = json.loads(
            (ROOT / "docs" / "contracts" / "consumer-route-map.json").read_text(encoding="utf-8")
        )
        cls.ownership_matrix = json.loads(
            (ROOT / "docs" / "contracts" / "migration-ownership-matrix.json").read_text(encoding="utf-8")
        )
        cls.plan = json.loads(PLAN.read_text(encoding="utf-8"))
        cls.doc = DOC.read_text(encoding="utf-8")

    def test_generated_plan_is_current(self):
        expected = MODULE.build_targets(self.consumer_map, self.ownership_matrix)
        self.assertEqual(expected, self.plan)

    def test_every_cli_route_has_an_explicit_runtime_target(self):
        self.assertEqual(13, self.plan["summary"]["route_count"])
        for route in self.plan["routes"]:
            with self.subTest(route=route["route_id"]):
                self.assertIn(route["owner"], {"laravel", "go"})
                self.assertEqual(
                    MODULE.OWNER_BASE_URL_ENV[route["owner"]],
                    route["target_base_url_env"],
                )
                self.assertIn(
                    route["execution_mode"],
                    {"configuration-read", "result-reporting-write"},
                )
                self.assertTrue(route["requires_runtime_credential"])

    def test_result_writes_are_identified_for_synthetic_fixture_use(self):
        writes = [
            route
            for route in self.plan["routes"]
            if route["execution_mode"] == "result-reporting-write"
        ]
        self.assertGreater(len(writes), 0)
        self.assertTrue(all(route["operation_kind"] == "mutation" for route in writes))

    def test_unknown_owner_fails_closed(self):
        matrix = copy.deepcopy(self.ownership_matrix)
        for route in matrix["routes"]:
            if route["route_id"] == self.plan["routes"][0]["route_id"]:
                route["owner"] = "unknown-runtime"
                break
        with self.assertRaises(MODULE.SmokeTargetError):
            MODULE.build_targets(self.consumer_map, matrix)

    def test_documentation_records_targeting_and_rollback_policy(self):
        normalized_doc = " ".join(self.doc.split())
        for phrase in (
            "route owner recorded in the migration ownership matrix is authoritative",
            "IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL",
            "IDELIUM_CLI_SMOKE_GO_BASE_URL",
            "Runtime credentials stay outside the generated plan",
            "Laravel remains the fallback owner",
            "Rollback is a normal revert",
        ):
            with self.subTest(phrase=phrase):
                self.assertIn(phrase, normalized_doc)


if __name__ == "__main__":
    unittest.main()
