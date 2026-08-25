import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
COMPARATOR_PATH = ROOT / "scripts" / "compare_golden_http.py"
SPEC = importlib.util.spec_from_file_location("compare_golden_http", COMPARATOR_PATH)
COMPARATOR = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = COMPARATOR
SPEC.loader.exec_module(COMPARATOR)

GOLDEN_DIR = ROOT / "testdata" / "golden"
ROLLOUT_PATH = ROOT / "docs" / "contracts" / "route-rollout-overrides.json"

CLI_CONFIGURATION_FIXTURE_PAIRS = {
    "GET|HEAD /api/ideliumcl/test/{idTest}": (
        "cli-test.fixture.json",
        "cli-test-go.fixture.json",
    ),
    "GET|HEAD /api/ideliumcl/testcycle/{idTestCycle}": (
        "cli-test-cycle.fixture.json",
        "cli-test-cycle-go.fixture.json",
    ),
}


class CLIConfigurationDifferentialCoverageTest(unittest.TestCase):
    def load_fixture(self, filename):
        return json.loads((GOLDEN_DIR / filename).read_text(encoding="utf-8"))

    def test_every_go_owned_cli_configuration_route_has_fixture_pair(self):
        rollout = json.loads(ROLLOUT_PATH.read_text(encoding="utf-8"))
        go_owned_cli_routes = {
            route
            for route, owner in rollout["routes"].items()
            if owner == "go-owned" and route in CLI_CONFIGURATION_FIXTURE_PAIRS
        }

        self.assertEqual(set(CLI_CONFIGURATION_FIXTURE_PAIRS), go_owned_cli_routes)

    def test_cli_configuration_fixture_pairs_preserve_laravel_contracts(self):
        for route, (expected_name, actual_name) in CLI_CONFIGURATION_FIXTURE_PAIRS.items():
            with self.subTest(route=route):
                expected = self.load_fixture(expected_name)
                actual = self.load_fixture(actual_name)

                comparison = COMPARATOR.compare(expected, actual)

                self.assertTrue(comparison.passed, comparison.to_dict())

    def test_cli_configuration_fixture_pairs_are_safe_tenant_reads(self):
        for route, fixture_names in CLI_CONFIGURATION_FIXTURE_PAIRS.items():
            with self.subTest(route=route):
                for fixture_name in fixture_names:
                    fixture = self.load_fixture(fixture_name)
                    self.assertEqual("GET", fixture["route"]["method"])
                    self.assertEqual([], fixture["sideEffects"])
                    self.assertEqual("api-key", fixture["route"]["trustPath"])
                    self.assertTrue(fixture["route"]["tenantOwned"])


if __name__ == "__main__":
    unittest.main()
