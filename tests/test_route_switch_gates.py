import copy
import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
POLICY_PATH = ROOT / "docs" / "operations" / "route-switch-gates.json"
SCRIPT_PATH = ROOT / "scripts" / "validate_route_switch_gates.py"
SPEC = importlib.util.spec_from_file_location("validate_route_switch_gates", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class RouteSwitchGateTest(unittest.TestCase):
    def setUp(self):
        self.policy = json.loads(POLICY_PATH.read_text(encoding="utf-8"))

    def test_committed_policy_is_valid(self):
        self.assertEqual([], MODULE.validate_policy(self.policy))

    def test_rejects_dual_writes(self):
        policy = copy.deepcopy(self.policy)
        policy["ownership"]["dual_writes_allowed"] = True
        self.assertIn(
            "$.ownership.dual_writes_allowed: must be false",
            MODULE.validate_policy(policy),
        )

    def test_rejects_nonzero_tolerance_for_cross_tenant_access(self):
        policy = copy.deepcopy(self.policy)
        policy["stop_thresholds"]["cross_tenant_access_events"] = 1
        self.assertIn(
            "$.stop_thresholds.cross_tenant_access_events: must have zero tolerance",
            MODULE.validate_policy(policy),
        )

    def test_rejects_missing_mutation_approval_role(self):
        policy = copy.deepcopy(self.policy)
        policy["approval_profiles"]["mutation-aggregate"]["required_roles"].remove(
            "security-reviewer"
        )
        self.assertTrue(
            any(
                "mutation-aggregate" in error and "missing" in error
                for error in MODULE.validate_policy(policy)
            )
        )

    def test_rejects_unsafe_rollback_owner(self):
        policy = copy.deepcopy(self.policy)
        policy["rollback"]["gateway_owner_after_rollback"] = "go"
        self.assertIn(
            "$.rollback.gateway_owner_after_rollback: must be laravel",
            MODULE.validate_policy(policy),
        )


if __name__ == "__main__":
    unittest.main()
