import importlib.util
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
SCRIPT = ROOT / "scripts" / "run_cli_smoke.py"
SPEC = importlib.util.spec_from_file_location("run_cli_smoke", SCRIPT)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class CliSmokeRunnerTest(unittest.TestCase):
    def test_selects_go_configuration_reads(self):
        plan = MODULE.load_plan(ROOT / "docs" / "contracts" / "cli-smoke-targets.json")

        routes = MODULE.select_routes(plan, "go", "configuration-read")

        self.assertEqual(7, len(routes))
        self.assertTrue(all(route["owner"] == "go" for route in routes))
        self.assertTrue(all(route["execution_mode"] == "configuration-read" for route in routes))

    def test_requires_runtime_inputs_without_exposing_values(self):
        plan = MODULE.load_plan(ROOT / "docs" / "contracts" / "cli-smoke-targets.json")
        routes = MODULE.select_routes(plan, "go", "configuration-read")

        with self.assertRaises(MODULE.CliSmokeError) as context:
            MODULE.validate_environment(routes, {})

        message = str(context.exception)
        self.assertIn("IDELIUM_CLI_SMOKE_GO_BASE_URL", message)
        self.assertIn("IDELIUM_CLI_SMOKE_API_KEY", message)
        self.assertNotIn("secret", message)

    def test_legacy_key_environment_alias_is_supported(self):
        plan = MODULE.load_plan(ROOT / "docs" / "contracts" / "cli-smoke-targets.json")
        routes = MODULE.select_routes(plan, "go", "configuration-read")
        env = {
            "IDELIUM_CLI_SMOKE_GO_BASE_URL": "https://go.example.invalid",
            "IDELIUM_CLI_SMOKE_IDELIUM_KEY": "secret-value",
            "IDELIUM_CLI_SMOKE_ID_ENVIRONMENT": "16",
            "IDELIUM_CLI_SMOKE_ID_PLUGIN": "14",
            "IDELIUM_CLI_SMOKE_ID_PROJECT": "3",
            "IDELIUM_CLI_SMOKE_ID_STEP": "12",
            "IDELIUM_CLI_SMOKE_ID_TEST": "9",
            "IDELIUM_CLI_SMOKE_ID_TEST_CYCLE": "7",
        }

        MODULE.validate_environment(routes, env)

    def test_dry_run_renders_all_safe_read_urls(self):
        plan = MODULE.load_plan(ROOT / "docs" / "contracts" / "cli-smoke-targets.json")
        env = {
            "IDELIUM_CLI_SMOKE_GO_BASE_URL": "https://go.example.invalid",
            "IDELIUM_CLI_SMOKE_API_KEY": "secret-value",
            "IDELIUM_CLI_SMOKE_ID_ENVIRONMENT": "16",
            "IDELIUM_CLI_SMOKE_ID_PLUGIN": "14",
            "IDELIUM_CLI_SMOKE_ID_PROJECT": "3",
            "IDELIUM_CLI_SMOKE_ID_STEP": "12",
            "IDELIUM_CLI_SMOKE_ID_TEST": "9",
            "IDELIUM_CLI_SMOKE_ID_TEST_CYCLE": "7",
        }

        results = MODULE.run(plan, env, "go", "configuration-read", 1.0, dry_run=True)

        self.assertEqual(7, len(results))
        self.assertTrue(all(result.status == 0 for result in results))
        self.assertTrue(any(result.url.endswith("/api/ideliumcl/testcycle/7") for result in results))
        self.assertTrue(any(result.url.endswith("/api/ideliumcl/environments/3") for result in results))


if __name__ == "__main__":
    unittest.main()
