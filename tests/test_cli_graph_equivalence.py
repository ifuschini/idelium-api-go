import copy
import importlib.util
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT = ROOT / "scripts" / "compare_cli_graph.py"
SPEC = importlib.util.spec_from_file_location("compare_cli_graph", SCRIPT)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class CliGraphEquivalenceTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.laravel = MODULE.load_fixture(ROOT / "testdata" / "golden" / "cli-graph-cycle.fixture.json")
        cls.go = MODULE.load_fixture(ROOT / "testdata" / "golden" / "cli-graph-cycle-go.fixture.json")

    def test_laravel_and_go_graphs_are_equivalent_for_the_same_cycle(self):
        comparison = MODULE.compare(self.laravel, self.go)

        self.assertTrue(comparison.passed, comparison.to_dict())

    def test_missing_referenced_test_is_reported(self):
        fixture = copy.deepcopy(self.go)
        fixture["graph"]["tests"] = []

        comparison = MODULE.compare(self.laravel, fixture)

        self.assertFalse(comparison.passed)
        self.assertTrue(any("Referenced test 9 is missing" in difference.reason for difference in comparison.differences))

    def test_cross_project_resource_is_reported(self):
        fixture = copy.deepcopy(self.go)
        fixture["graph"]["steps"][0]["idProject"] = 999

        comparison = MODULE.compare(self.laravel, fixture)

        self.assertFalse(comparison.passed)
        self.assertTrue(any("Resource project does not match graph project" in difference.reason for difference in comparison.differences))

    def test_payload_drift_is_reported(self):
        fixture = copy.deepcopy(self.go)
        fixture["graph"]["cycle"]["configTestIds"] = [10]

        comparison = MODULE.compare(self.laravel, fixture)

        self.assertFalse(comparison.passed)
        self.assertTrue(any(difference.path.endswith("configTestIds") for difference in comparison.differences))


if __name__ == "__main__":
    unittest.main()
