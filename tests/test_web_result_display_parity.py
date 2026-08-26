import json
import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
CONTRACT = ROOT / "docs" / "contracts" / "web-result-display-parity.json"


class WebResultDisplayParityTests(unittest.TestCase):
    def setUp(self):
        self.contract = json.loads(CONTRACT.read_text(encoding="utf-8"))

    def test_contract_contains_required_web_views(self):
        required = {
            "test-running-tab",
            "test-results-tab",
            "cycle-list-selection",
            "run-list-selection",
            "test-list-selection",
            "step-by-step-results",
            "step-timing-timeline",
            "environment-and-target-summary",
            "postman-request-detail-modal",
            "safe-diagnostics",
        }

        self.assertEqual(required, set(self.contract["requiredWebViews"]))

    def test_every_fixture_has_environment_target_and_timing(self):
        for fixture in self.contract["fixtures"]:
            execution = fixture["execution"]
            cycle = execution["cycle"]
            self.assertGreater(cycle["durationMs"], 0, fixture["id"])
            self.assertIn(cycle["status"], {"passed", "failed", "cancelled", "pending"})
            self.assertIn("name", execution["environment"])

            target = execution["target"]
            for field in [
                "platform",
                "browser",
                "browserVersion",
                "operatingSystem",
                "operatingSystemVersion",
                "device",
            ]:
                self.assertIn(field, target, fixture["id"])

            for test in execution["tests"]:
                self.assertGreaterEqual(test["durationMs"], 0, test["name"])
                for step in test["steps"]:
                    self.assertGreaterEqual(step["durationMs"], 0, step["name"])
                    self.assertGreaterEqual(step["gapAfterPreviousMs"], 0, step["name"])
                    self.assertIn(step["status"], {"passed", "failed", "skipped", "cancelled", "pending"})

    def test_postman_fixture_exposes_request_level_display_data(self):
        postman_fixture = next(
            fixture for fixture in self.contract["fixtures"] if fixture["id"] == "go-produced-postman-run"
        )
        steps = postman_fixture["execution"]["tests"][0]["steps"]
        request_rows = steps[0]["postmanRequests"]

        self.assertGreaterEqual(len(request_rows), 2)
        for row in request_rows:
            for field in [
                "name",
                "method",
                "url",
                "status",
                "durationMs",
                "assertionsPassed",
                "assertionsTotal",
                "responseSummary",
            ]:
                self.assertIn(field, row)
            self.assertTrue(row["url"].startswith("https://"))
            self.assertLessEqual(row["assertionsPassed"], row["assertionsTotal"])

    def test_contract_does_not_store_sensitive_values(self):
        serialized = json.dumps(self.contract["fixtures"], sort_keys=True).lower()
        unsafe_fragments = [
            "authorization",
            "bearer ",
            "cookie",
            "csrf",
            "password",
            "secret",
            "session",
            "set-cookie",
            "token",
        ]
        for fragment in unsafe_fragments:
            self.assertNotIn(fragment, serialized)

    def test_failure_fixture_has_safe_diagnostics_without_stack_traces(self):
        browser_fixture = next(
            fixture for fixture in self.contract["fixtures"] if fixture["id"] == "go-produced-browser-run"
        )
        failed_steps = [
            step
            for test in browser_fixture["execution"]["tests"]
            for step in test["steps"]
            if step["status"] == "failed"
        ]

        self.assertGreaterEqual(len(failed_steps), 1)
        for step in failed_steps:
            self.assertGreaterEqual(len(step["diagnostics"]), 1)
            diagnostic_payload = json.dumps(step["diagnostics"], sort_keys=True).lower()
            self.assertNotIn("stacktrace", diagnostic_payload)
            self.assertNotIn("/users/", diagnostic_payload)


if __name__ == "__main__":
    unittest.main()
