import copy
import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT_PATH = ROOT / "scripts" / "validate_golden_fixtures.py"
SPEC = importlib.util.spec_from_file_location("validate_golden_fixtures", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class GoldenFixturePolicyTest(unittest.TestCase):
    def setUp(self):
        self.fixture_path = (
            ROOT / "testdata" / "golden" / "platform-status.fixture.json"
        )
        self.fixture = json.loads(self.fixture_path.read_text(encoding="utf-8"))

    def validate(self, update):
        fixture = copy.deepcopy(self.fixture)
        update(fixture)
        return MODULE.validate_fixture_data(fixture)

    def test_committed_fixture_is_sanitized_and_valid(self):
        self.assertEqual([], MODULE.validate_file(self.fixture_path))

    def test_rejects_sensitive_headers_without_echoing_the_value(self):
        secret = "Bearer deliberately-sensitive-test-value"
        errors = self.validate(
            lambda fixture: fixture["request"]["headers"].update(
                {"Authorization": secret}
            )
        )
        diagnostic = "\n".join(errors)
        self.assertIn("sensitive headers must be removed", diagnostic)
        self.assertNotIn(secret, diagnostic)

    def test_rejects_nested_sensitive_payload_fields(self):
        errors = self.validate(
            lambda fixture: fixture["response"].update(
                {"body": {"result": {"access_token": "not-printed"}}}
            )
        )
        self.assertIn(
            "$.response.body.result.access_token: sensitive fields must be removed",
            errors,
        )
        self.assertNotIn("not-printed", "\n".join(errors))

    def test_rejects_non_synthetic_tenant_identity(self):
        errors = self.validate(
            lambda fixture: fixture["context"].update(
                {"tenant": {"id": "customer-42", "synthetic": False}}
            )
        )
        self.assertTrue(any("synthetic=true" in error for error in errors))
        self.assertTrue(any("fixture-" in error for error in errors))

    def test_rejects_oversized_response_body(self):
        errors = self.validate(
            lambda fixture: fixture["response"].update(
                {"body": "x" * (MODULE.MAX_BODY_BYTES + 1)}
            )
        )
        self.assertTrue(any("must not exceed" in error for error in errors))

    def test_rejects_missing_tenant_context_for_tenant_route(self):
        errors = self.validate(
            lambda fixture: fixture["context"].update({"tenant": None})
        )
        self.assertIn(
            "$.context.tenant: is required for a tenant-owned route",
            errors,
        )


if __name__ == "__main__":
    unittest.main()
