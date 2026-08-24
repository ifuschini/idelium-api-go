import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT_PATH = ROOT / "scripts" / "redact_golden_fixture.py"
SPEC = importlib.util.spec_from_file_location("redact_golden_fixture", SCRIPT_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.path.insert(0, str(ROOT / "scripts"))
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)

VALIDATOR_PATH = ROOT / "scripts" / "validate_golden_fixtures.py"
VALIDATOR_SPEC = importlib.util.spec_from_file_location(
    "validate_golden_fixtures_for_redaction", VALIDATOR_PATH
)
VALIDATOR = importlib.util.module_from_spec(VALIDATOR_SPEC)
assert VALIDATOR_SPEC.loader is not None
sys.modules[VALIDATOR_SPEC.name] = VALIDATOR
VALIDATOR_SPEC.loader.exec_module(VALIDATOR)


def candidate_fixture():
    return {
        "fixtureVersion": "1.0",
        "id": "sensitive-candidate",
        "description": "Candidate fixture with synthetic secrets that must be removed.",
        "source": {
            "runtime": "laravel",
            "repository": "idelium/idelium-api",
            "revision": "ef97a6b3f42105cad583b3f1a68acf637a9d086e",
            "capturedAt": "2026-01-01T00:00:00Z",
            "routeInventoryDigestSha256": "0ba5e5d3ff27afdba1e3c32619af26e66ff26022570a6a730b1ce93145ccffd9",
        },
        "route": {
            "method": "GET",
            "path": "/api/admin/platforms/status",
            "trustPath": "browser-session",
            "tenantOwned": True,
        },
        "context": {
            "tenant": {"id": "fixture-tenant-alpha", "synthetic": True},
            "actor": {"id": "fixture-actor-admin", "synthetic": True},
        },
        "request": {
            "headers": {
                "Accept": "application/json",
                "Authorization": "Bearer synthetic-secret",
                "Cookie": "session=synthetic-secret",
            },
            "query": {"apiKey": "synthetic-key"},
            "body": {
                "password": "synthetic-password",
                "note": "Bearer synthetic-body-token",
            },
        },
        "response": {
            "status": 200,
            "headers": {
                "Content-Type": "application/json",
                "Set-Cookie": "session=synthetic-secret",
            },
            "body": {"token": "synthetic-token", "status": "ok"},
        },
        "normalizations": [],
        "redactions": [],
        "sideEffects": [{"sessionId": "synthetic-session", "status": "created"}],
    }


class GoldenFixtureRedactionTest(unittest.TestCase):
    def test_redacts_sensitive_headers_fields_and_strings(self):
        sanitized = MODULE.redact_fixture(candidate_fixture())

        serialized = json.dumps(sanitized)
        self.assertNotIn("Authorization", sanitized["request"]["headers"])
        self.assertNotIn("Cookie", sanitized["request"]["headers"])
        self.assertNotIn("Set-Cookie", sanitized["response"]["headers"])
        self.assertNotIn("apiKey", sanitized["request"]["query"])
        self.assertNotIn("password", sanitized["request"]["body"])
        self.assertNotIn("token", sanitized["response"]["body"])
        self.assertNotIn("sessionId", sanitized["sideEffects"][0])
        self.assertNotIn("synthetic-secret", serialized)
        self.assertIn(MODULE.REDACTED, serialized)
        self.assertGreaterEqual(len(sanitized["redactions"]), 7)

    def test_redacted_fixture_passes_committed_validator(self):
        sanitized = MODULE.redact_fixture(candidate_fixture())

        errors = VALIDATOR.validate_fixture_data(sanitized)

        self.assertEqual([], errors)

    def test_cli_writes_sanitized_fixture(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            input_path = Path(temp_dir) / "candidate.fixture.json"
            output_path = Path(temp_dir) / "sanitized.fixture.json"
            input_path.write_text(
                json.dumps(candidate_fixture()), encoding="utf-8"
            )

            exit_code = MODULE.main_for_test(input_path, output_path)

            self.assertEqual(0, exit_code)
            self.assertTrue(output_path.exists())


if __name__ == "__main__":
    unittest.main()
