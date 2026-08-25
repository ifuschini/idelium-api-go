#!/usr/bin/env python3
"""Compare offline shadow-read fixtures for Go-owned safe GET routes."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

from compare_golden_http import compare, load_fixture


SAFE_METHODS = {"GET", "HEAD"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--config",
        type=Path,
        default=Path("docs/contracts/shadow-read-comparison.json"),
        help="Shadow-read comparison plan.",
    )
    parser.add_argument(
        "--gateway",
        type=Path,
        default=Path("docs/contracts/gateway-route-ownership.json"),
        help="Gateway ownership contract used to validate route coverage.",
    )
    parser.add_argument(
        "--root",
        type=Path,
        default=Path("."),
        help="Repository root used to resolve fixture paths.",
    )
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def validate_plan(
    plan: dict[str, Any],
    gateway: dict[str, Any],
    root: Path,
) -> list[str]:
    errors: list[str] = []
    routes = plan.get("routes", [])
    if not isinstance(routes, list):
        return ["$.routes: must be an array"]

    planned_route_ids = set()
    for index, route in enumerate(routes):
        location = f"$.routes[{index}]"
        if not isinstance(route, dict):
            errors.append(f"{location}: must be an object")
            continue
        route_id = route.get("route_id")
        if not isinstance(route_id, str) or not route_id:
            errors.append(f"{location}.route_id: is required")
        else:
            planned_route_ids.add(route_id)
        if route.get("method") not in SAFE_METHODS:
            errors.append(f"{location}.method: must be GET or HEAD")
        for field in ("expected_fixture", "actual_fixture"):
            value = route.get(field)
            if not isinstance(value, str) or not value:
                errors.append(f"{location}.{field}: is required")
                continue
            if not (root / value).is_file():
                errors.append(f"{location}.{field}: fixture file does not exist")

    gateway_route_ids = {
        route["route_id"]
        for route in gateway.get("routes", [])
        if isinstance(route, dict) and route.get("owner") == "go"
    }
    missing = sorted(gateway_route_ids - planned_route_ids)
    unexpected = sorted(planned_route_ids - gateway_route_ids)
    for route_id in missing:
        errors.append(f"$.routes: missing gateway route {route_id}")
    for route_id in unexpected:
        errors.append(f"$.routes: unexpected non-gateway route {route_id}")
    return errors


def compare_plan(plan: dict[str, Any], root: Path) -> dict[str, Any]:
    route_results: list[dict[str, Any]] = []
    for route in plan["routes"]:
        expected = load_fixture(root / route["expected_fixture"])
        actual = load_fixture(root / route["actual_fixture"])
        comparison = compare(expected, actual)
        route_results.append(
            {
                "route_id": route["route_id"],
                "passed": comparison.passed,
                "differences": [
                    {"path": item.path, "reason": item.reason}
                    for item in comparison.differences
                ],
            }
        )
    return {
        "passed": all(route["passed"] for route in route_results),
        "routes": route_results,
    }


def main() -> int:
    args = parse_args()
    root = args.root.resolve()
    plan = load_json(root / args.config)
    gateway = load_json(root / args.gateway)
    errors = validate_plan(plan, gateway, root)
    if errors:
        print(json.dumps({"passed": False, "errors": errors}, indent=2))
        return 2

    report = compare_plan(plan, root)
    print(json.dumps(report, indent=2))
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
