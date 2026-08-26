import copy
import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT_PATH = ROOT / "scripts" / "build_ownership_matrix.py"
SPEC = importlib.util.spec_from_file_location("build_ownership_matrix", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class OwnershipMatrixTest(unittest.TestCase):
    def setUp(self):
        self.backlog = MODULE.load_json(
            ROOT / "docs" / "contracts" / "compatibility-backlog.json"
        )
        self.matrix = MODULE.build_matrix(self.backlog)

    def test_assigns_every_route_to_exactly_one_aggregate(self):
        self.assertEqual(self.backlog["public_route_count"], len(self.matrix["routes"]))
        self.assertEqual(
            len(self.matrix["routes"]),
            len({route["route_id"] for route in self.matrix["routes"]}),
        )

    def test_each_mutating_aggregate_has_one_owner_and_no_dual_writes(self):
        mutating = [
            aggregate
            for aggregate in self.matrix["aggregates"]
            if aggregate["mutation_route_count"]
        ]
        self.assertTrue(mutating)
        self.assertTrue(all(aggregate["mutation_owner"] in MODULE.OWNERS for aggregate in mutating))
        self.assertTrue(all(not aggregate["dual_writes_allowed"] for aggregate in mutating))
        go_mutating = [aggregate for aggregate in mutating if aggregate["mutation_owner"] == "go"]
        self.assertEqual(["cli-performed-tests"], [aggregate["aggregate"] for aggregate in go_mutating])

    def test_rejects_split_mutation_ownership(self):
        backlog = copy.deepcopy(self.backlog)
        account_mutations = [
            item
            for item in backlog["items"]
            if item["path"].startswith("/api/admin/accounts")
            and MODULE.is_mutation(item["method"])
        ]
        account_mutations[0]["rollout_status"] = "go-owned"
        with self.assertRaisesRegex(ValueError, "multiple mutation owners"):
            MODULE.build_matrix(backlog)

    def test_rejects_implicit_or_unknown_ownership_state(self):
        backlog = copy.deepcopy(self.backlog)
        backlog["items"][0]["rollout_status"] = "shadow"
        with self.assertRaisesRegex(ValueError, "ownership must be explicit"):
            MODULE.build_matrix(backlog)

    def test_committed_matrix_matches_current_backlog(self):
        committed = json.loads(
            (ROOT / "docs" / "contracts" / "migration-ownership-matrix.json").read_text(
                encoding="utf-8"
            )
        )
        self.assertEqual(self.matrix, committed)


if __name__ == "__main__":
    unittest.main()
