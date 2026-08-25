import importlib.util
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
COMPARATOR_PATH = ROOT / "scripts" / "compare_golden_http.py"
SPEC = importlib.util.spec_from_file_location("compare_golden_http", COMPARATOR_PATH)
COMPARATOR = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = COMPARATOR
SPEC.loader.exec_module(COMPARATOR)

GOLDEN_DIR = ROOT / "testdata" / "golden"
ROLLOUT_PATH = ROOT / "docs" / "contracts" / "route-rollout-overrides.json"

CATALOG_FIXTURE_PAIRS = {
    "GET|HEAD /api/admin/platforms/browserversions/{idBrowser}": (
        "platform-browser-versions.fixture.json",
        "platform-browser-versions-go.fixture.json",
    ),
    "GET|HEAD /api/admin/platforms/browsers/{idOs}": (
        "platform-browsers.fixture.json",
        "platform-browsers-go.fixture.json",
    ),
    "GET|HEAD /api/admin/platforms/brands": (
        "platform-brands.fixture.json",
        "platform-brands-go.fixture.json",
    ),
    "GET|HEAD /api/admin/platforms/locations": (
        "platform-locations.fixture.json",
        "platform-locations-go.fixture.json",
    ),
    "GET|HEAD /api/admin/platforms/models/{idBrand}": (
        "platform-models.fixture.json",
        "platform-models-go.fixture.json",
    ),
    "GET|HEAD /api/admin/platforms/os/{idType}": (
        "platform-operating-systems.fixture.json",
        "platform-operating-systems-go.fixture.json",
    ),
    "GET|HEAD /api/admin/platforms/osversion/{idOs}": (
        "platform-operating-system-versions.fixture.json",
        "platform-operating-system-versions-go.fixture.json",
    ),
    "GET|HEAD /api/admin/platforms/status": (
        "platform-status.fixture.json",
        "platform-status-go.fixture.json",
    ),
    "GET|HEAD /api/admin/platforms/types": (
        "platform-types.fixture.json",
        "platform-types-go.fixture.json",
    ),
}


class PlatformCatalogDifferentialCoverageTest(unittest.TestCase):
    def load_fixture(self, filename):
        return json.loads((GOLDEN_DIR / filename).read_text(encoding="utf-8"))

    def test_every_go_owned_platform_catalog_route_has_fixture_pair(self):
        rollout = json.loads(ROLLOUT_PATH.read_text(encoding="utf-8"))
        go_owned_catalog_routes = {
            route
            for route, owner in rollout["routes"].items()
            if owner == "go-owned" and "/api/admin/platforms/" in route
        }

        self.assertEqual(set(CATALOG_FIXTURE_PAIRS), go_owned_catalog_routes)

    def test_platform_catalog_fixture_pairs_preserve_laravel_contracts(self):
        for route, (expected_name, actual_name) in CATALOG_FIXTURE_PAIRS.items():
            with self.subTest(route=route):
                expected = self.load_fixture(expected_name)
                actual = self.load_fixture(actual_name)

                comparison = COMPARATOR.compare(expected, actual)

                self.assertTrue(comparison.passed, comparison.to_dict())

    def test_platform_catalog_fixture_pairs_are_safe_reads_without_side_effects(self):
        for route, fixture_names in CATALOG_FIXTURE_PAIRS.items():
            with self.subTest(route=route):
                for fixture_name in fixture_names:
                    fixture = self.load_fixture(fixture_name)
                    self.assertEqual("GET", fixture["route"]["method"])
                    self.assertEqual([], fixture["sideEffects"])
                    self.assertEqual("browser-session", fixture["route"]["trustPath"])
                    self.assertTrue(fixture["route"]["tenantOwned"])


if __name__ == "__main__":
    unittest.main()
