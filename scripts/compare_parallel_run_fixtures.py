#!/usr/bin/env python3
"""Compare the complete Laravel/Go parallel-run fixture inventory."""

from __future__ import annotations

import argparse
import json
import sys
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
            if route["method"] in {"GET", "HEAD"}:
                result = compare_http(load_http(expected), load_http(actual))
            else:
                result = compare_mutation(load_mutation(expected), load_mutation(actual))
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
