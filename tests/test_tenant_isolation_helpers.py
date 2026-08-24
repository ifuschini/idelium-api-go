import re
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
HELPER = ROOT / "internal" / "testsupport" / "tenant" / "tenant.go"
HELPER_TEST = ROOT / "internal" / "testsupport" / "tenant" / "tenant_test.go"
DOC = ROOT / "docs" / "contracts" / "tenant-isolation-test-helpers.md"


class TenantIsolationHelperContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.source = HELPER.read_text(encoding="utf-8")
        cls.test_source = HELPER_TEST.read_text(encoding="utf-8")
        cls.doc = DOC.read_text(encoding="utf-8")

    def test_helper_exposes_the_required_cross_tenant_contract(self):
        for symbol in (
            "type Scope struct",
            "type OwnedRecord struct",
            "type LookupFunc func",
            "func NewScope",
            "func AssertOwnedRecords",
            "func AssertOwnerCanRead",
            "func AssertForeignTenantCannotRead",
            "func AssertTenantIsolation",
        ):
            with self.subTest(symbol=symbol):
                self.assertIn(symbol, self.source)

    def test_negative_check_fails_closed_when_tenants_are_not_distinct(self):
        self.assertRegex(
            self.source,
            r"if owner\.TenantID == foreign\.TenantID \{\n\t\tt\.Fatal",
        )
        self.assertIn("AssertDistinctScopes(t, owner, foreign)", self.source)
        self.assertIn("RejectsSharedTenantSetup", self.test_source)

    def test_diagnostics_do_not_echo_sensitive_runtime_values(self):
        forbidden_fragments = (
            "%v",
            "resourceID",
            "TenantID)",
            "ActorID)",
            "password leaked",
        )
        diagnostics = "\n".join(
            line
            for line in self.source.splitlines()
            if "Fatal" in line or "Fatalf" in line
        )
        for fragment in forbidden_fragments:
            with self.subTest(fragment=fragment):
                self.assertNotIn(fragment, diagnostics)
        self.assertIn("failed with %T", diagnostics)

    def test_tests_cover_success_denial_and_safe_diagnostics(self):
        for name in (
            "TestAssertTenantIsolationAcceptsOwnerAndDeniesForeignTenant",
            "TestAssertOwnedRecordsAcceptsOnlyActiveTenantRecords",
            "TestAssertOwnerCanReadReportsLookupFailuresWithoutValues",
            "TestAssertForeignTenantCannotReadRejectsSharedTenantSetup",
        ):
            with self.subTest(name=name):
                self.assertIn(f"func {name}", self.test_source)

    def test_documentation_records_adoption_and_rollback_policy(self):
        normalized_doc = re.sub(r"\s+", " ", self.doc)
        for phrase in (
            "ownership predicate in the same query or transaction",
            "a foreign tenant cannot resolve the owner resource",
            "Diagnostics avoid tenant IDs",
            "Database-backed tests must seed at least two synthetic tenants",
            "OpenAPI and Laravel-Go differential comparisons are not applicable",
            "Rollback is a normal revert",
        ):
            with self.subTest(phrase=phrase):
                self.assertIn(phrase, normalized_doc)

    def test_no_todos_or_legacy_domains_were_introduced(self):
        combined = "\n".join((self.source, self.test_source, self.doc))
        self.assertNotRegex(combined, re.compile(r"\bTODO\b", re.IGNORECASE))
        self.assertNotIn("idelium.io", combined)


if __name__ == "__main__":
    unittest.main()
