import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
INTEGRATION_TEST = ROOT / "internal" / "persistence" / "mysql" / "database_integration_test.go"

RESOURCE_EXPECTATIONS = {
    "test cycle": (
        "TestCLITestCycleRepositoryIntegration",
        "expected cross-tenant test cycle to be hidden",
    ),
    "test": (
        "TestCLITestRepositoryIntegration",
        "expected cross-tenant test to be hidden",
    ),
    "step": (
        "TestCLIStepRepositoryIntegration",
        "expected cross-tenant step to be hidden",
    ),
    "plugin": (
        "TestCLIPluginRepositoryIntegration",
        "expected cross-tenant plugin to be hidden",
    ),
    "environment": (
        "TestCLIEnvironmentRepositoryIntegration",
        "expected cross-tenant environment to be hidden",
    ),
}


class CliCrossTenantDenialCoverageTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.source = INTEGRATION_TEST.read_text(encoding="utf-8")

    def test_every_cli_graph_resource_has_mysql_cross_tenant_denial_coverage(self):
        for resource, (test_name, diagnostic) in RESOURCE_EXPECTATIONS.items():
            with self.subTest(resource=resource):
                self.assertIn(test_name, self.source)
                self.assertIn(diagnostic, self.source)
                self.assertIn("cliapi.ErrNotFound", self.source)


if __name__ == "__main__":
    unittest.main()
