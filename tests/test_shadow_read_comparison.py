import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT_PATH = ROOT / "scripts" / "compare_shadow_reads.py"
SPEC = importlib.util.spec_from_file_location("compare_shadow_reads", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.path.insert(0, str(ROOT / "scripts"))
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class ShadowReadComparisonTest(unittest.TestCase):
    def setUp(self):
        self.plan = MODULE.load_json(
            ROOT / "docs" / "contracts" / "shadow-read-comparison.json"
        )
        self.gateway = MODULE.load_json(
            ROOT / "docs" / "contracts" / "gateway-route-ownership.json"
        )

    def test_committed_plan_is_valid(self):
        self.assertEqual([], MODULE.validate_plan(self.plan, self.gateway, ROOT))

    def test_committed_shadow_read_comparison_passes(self):
        report = MODULE.compare_plan(self.plan, ROOT)

        self.assertTrue(report["passed"], json.dumps(report, indent=2))

    def test_plan_rejects_mutation_methods(self):
        plan = json.loads(json.dumps(self.plan))
        plan["routes"][0]["method"] = "POST"

        errors = MODULE.validate_plan(plan, self.gateway, ROOT)

        self.assertIn("$.routes[0].method: must be GET or HEAD", errors)

    def test_plan_must_match_gateway_routes(self):
        plan = json.loads(json.dumps(self.plan))
        removed = plan["routes"].pop()["route_id"]

        errors = MODULE.validate_plan(plan, self.gateway, ROOT)

        self.assertIn(f"$.routes: missing gateway route {removed}", errors)


if __name__ == "__main__":
    unittest.main()
