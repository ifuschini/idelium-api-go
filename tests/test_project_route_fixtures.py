"""Regression coverage for the real Laravel/Go project route captures."""

from __future__ import annotations

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))

from compare_parallel_run_fixtures import normalize_parallel_fixture  # noqa: E402
from compare_golden_http import compare as compare_http  # noqa: E402
from compare_golden_side_effects import compare as compare_mutation  # noqa: E402


FIXTURE_DIR = ROOT / "testdata" / "golden" / "projects"
ROUTES = {
    "project-list": "read",
    "project-show": "read",
    "project-update": "mutation",
    "project-delete": "mutation",
}


def load(name: str, runtime: str) -> dict:
    suffix = "" if runtime == "laravel" else "-go"
    return json.loads((FIXTURE_DIR / f"{name}{suffix}.fixture.json").read_text(encoding="utf-8"))


def test_project_route_pairs_are_real_and_equivalent() -> None:
    for name, kind in ROUTES.items():
        laravel = load(name, "laravel")
        go = load(name, "go")
        assert laravel["source"]["runtime"] == "laravel"
        assert go["source"]["runtime"] == "go"
        assert laravel["response"]["status"] == go["response"]["status"] == 200
        assert laravel["route"] == go["route"]
        result = (compare_http if kind == "read" else compare_mutation)(
            normalize_parallel_fixture(laravel), normalize_parallel_fixture(go)
        )
        assert result.passed, result.differences


def test_project_update_fixture_preserves_sanitized_payload() -> None:
    fixture = load("project-update", "go")
    assert fixture["request"]["body"] == {
        "name": "fixture-project-updated",
        "description": "Disposable updated project",
    }
