import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
LARAVEL = ROOT.parent / "idelium-api"
MATRIX = ROOT / "docs" / "contracts" / "security-regression-matrix.md"


class SecurityRegressionMatrixTest(unittest.TestCase):
    def test_matrix_references_both_runtime_sources_and_required_cases(self):
        matrix = MATRIX.read_text(encoding="utf-8")
        required = (
            "BrowserSessionAuthenticationTest.php",
            "IdeliumCliTenantIsolationTest.php",
            "ParallelRunScheduleApiTest.php",
            "RunTokenTest.php",
            "internal/browserauth/handler_test.go",
            "internal/persistence/mysql/database_integration_test.go",
            "internal/persistence/mysql/worker_token_test.go",
        )
        for source in required:
            self.assertIn(source, matrix)

    def test_laravel_security_sources_exist_and_matrix_contains_no_secret_values(self):
        for relative in (
            "tests/Feature/BrowserSessionAuthenticationTest.php",
            "tests/Feature/IdeliumCliTenantIsolationTest.php",
            "tests/Feature/ParallelRunScheduleApiTest.php",
            "tests/Feature/RunTokenTest.php",
        ):
            self.assertTrue((LARAVEL / relative).is_file(), relative)
        matrix = MATRIX.read_text(encoding="utf-8").lower()
        for forbidden in ("authorization: bearer", "cookie:", "api_key=", "password="):
            self.assertNotIn(forbidden, matrix)


if __name__ == "__main__":
    unittest.main()
