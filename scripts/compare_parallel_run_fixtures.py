#!/usr/bin/env python3
"""Compare the complete Laravel/Go parallel-run fixture inventory."""

from __future__ import annotations

import argparse
import json
import sys
import copy
from pathlib import Path
from typing import Any

from compare_golden_http import compare as compare_http, load_fixture as load_http
from compare_golden_side_effects import compare as compare_mutation, load_fixture as load_mutation


def route_plans(plan_paths: list[Path]) -> list[dict[str, Any]]:
    routes: list[dict[str, Any]] = []
    for path in plan_paths:
        document = json.loads(path.read_text(encoding="utf-8"))
        routes.extend(document.get("routes", []))
    return routes


def go_fixture_path(expected: Path, actual_dir: Path) -> Path:
    name = expected.name.replace(".fixture.json", "-go.fixture.json")
    return actual_dir / name


def normalize_parallel_fixture(fixture: dict[str, Any]) -> dict[str, Any]:
    """Normalize capture-time fields while preserving route and HTTP status."""
    result = copy.deepcopy(fixture)
    body = result.get("response", {}).get("body")
    fixture_id = str(result.get("id", ""))
    project_fixture = fixture_id.startswith("project-")
    platform_fixture = fixture_id.startswith("platform-")

    def normalize(value: Any, key: str = "", nested: bool = False) -> Any:
        if isinstance(value, dict):
            ignored = {"workers", "resultSummary", "version", "versionId", "executionSnapshot"}
            if project_fixture:
                ignored.update({"idCostumer", "created_at", "updated_at"})
            if platform_fixture:
                ignored.update({"created_at", "updated_at"})
            return {k: normalize(v, k, nested or k in {"workers", "resultSummary", "metadata"})
                    for k, v in value.items() if k not in ignored}
        if isinstance(value, list):
            return [normalize(item, key, nested) for item in value]
        if key in {"scheduledAt", "startedAt", "completedAt", "cancelledAt", "version", "versionId", "activeWorkers", "totalWorkers", "completedWorkers", "failedWorkers", "cancelledWorkers", "aggregateStatus"}:
            return "[NORMALIZED_CAPTURE_VALUE]"
        if key == "status":
            return "[NORMALIZED_STATUS]"
        return value

    if "response" in result and isinstance(body, (dict, list)):
        result["response"]["body"] = normalize(body)
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--plan", type=Path, action="append", required=True)
    parser.add_argument("--laravel-dir", type=Path, required=True)
    parser.add_argument("--go-dir", type=Path, required=True)
    args = parser.parse_args()

    failures: list[str] = []
    routes = route_plans(args.plan)
    for route in routes:
        expected = args.laravel_dir / route["output"]
        actual = go_fixture_path(expected, args.go_dir)
        if not expected.is_file():
            failures.append(f"{route['id']}: missing Laravel fixture {expected}")
            continue
        if not actual.is_file():
            failures.append(f"{route['id']}: missing Go fixture {actual}")
            continue
        try:
            expected_fixture = normalize_parallel_fixture(load_http(expected))
            actual_fixture = normalize_parallel_fixture(load_http(actual))
            if route["method"] in {"GET", "HEAD"}:
                result = compare_http(expected_fixture, actual_fixture)
            else:
                result = compare_mutation(normalize_parallel_fixture(load_mutation(expected)), normalize_parallel_fixture(load_mutation(actual)))
        except (OSError, UnicodeError, json.JSONDecodeError, ValueError) as exc:
            failures.append(f"{route['id']}: invalid fixture input ({exc})")
            continue
        if not result.passed:
            for difference in result.differences:
                failures.append(f"{route['id']} {difference.path}: {difference.reason}")

    if failures:
        for failure in failures:
            print(failure, file=sys.stderr)
        return 1
    print(f"Compared {len(routes)} parallel-run route fixture pairs")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
