#!/usr/bin/env python3
"""Compare sanitized Laravel and Go HTTP golden fixtures for safe reads."""

from __future__ import annotations

import argparse
import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


SAFE_READ_METHODS = {"GET", "HEAD"}
COMPARABLE_HEADERS = {"cache-control", "content-type", "location"}


@dataclass(frozen=True)
class Difference:
    path: str
    reason: str


@dataclass
class Comparison:
    passed: bool
    differences: list[Difference] = field(default_factory=list)

    def to_dict(self) -> dict[str, Any]:
        return {
            "passed": self.passed,
            "differences": [
                {"path": item.path, "reason": item.reason} for item in self.differences
            ],
        }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Compare sanitized Laravel and Go HTTP golden fixtures."
    )
    parser.add_argument("--expected", type=Path, required=True)
    parser.add_argument("--actual", type=Path, required=True)
    return parser.parse_args()


def load_fixture(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def comparable_headers(headers: dict[str, Any]) -> dict[str, str]:
    normalized = {str(key).lower(): str(value) for key, value in headers.items()}
    return {
        key: value
        for key, value in normalized.items()
        if key in COMPARABLE_HEADERS
    }


def ensure_safe_read(fixture: dict[str, Any], label: str) -> list[Difference]:
    differences: list[Difference] = []
    route = fixture.get("route", {})
    method = route.get("method")
    if method not in SAFE_READ_METHODS:
        differences.append(
            Difference(f"$.{label}.route.method", "Only GET and HEAD fixtures can use the safe-read comparator.")
        )
    side_effects = fixture.get("sideEffects", [])
    if side_effects:
        differences.append(
            Difference(f"$.{label}.sideEffects", "Safe-read fixtures must not declare side effects.")
        )
    return differences


def compare_values(
    expected: Any,
    actual: Any,
    path: str,
    differences: list[Difference],
) -> None:
    if type(expected) is not type(actual):
        differences.append(Difference(path, "JSON type differs."))
        return
    if isinstance(expected, dict):
        expected_keys = set(expected)
        actual_keys = set(actual)
        for missing in sorted(expected_keys - actual_keys):
            differences.append(Difference(f"{path}.{missing}", "Expected field is missing."))
        for unexpected in sorted(actual_keys - expected_keys):
            differences.append(Difference(f"{path}.{unexpected}", "Unexpected field is present."))
        for key in sorted(expected_keys & actual_keys):
            compare_values(expected[key], actual[key], f"{path}.{key}", differences)
        return
    if isinstance(expected, list):
        if len(expected) != len(actual):
            differences.append(Difference(path, "Array length differs."))
            return
        for index, expected_item in enumerate(expected):
            compare_values(expected_item, actual[index], f"{path}[{index}]", differences)
        return
    if expected != actual:
        differences.append(Difference(path, "Value differs."))


def compare(expected: dict[str, Any], actual: dict[str, Any]) -> Comparison:
    differences: list[Difference] = []
    differences.extend(ensure_safe_read(expected, "expected"))
    differences.extend(ensure_safe_read(actual, "actual"))

    for field_name in ("method", "path", "trustPath", "tenantOwned"):
        expected_value = expected.get("route", {}).get(field_name)
        actual_value = actual.get("route", {}).get(field_name)
        if expected_value != actual_value:
            differences.append(
                Difference(f"$.route.{field_name}", "Route contract metadata differs.")
            )

    expected_response = expected.get("response", {})
    actual_response = actual.get("response", {})
    if expected_response.get("status") != actual_response.get("status"):
        differences.append(Difference("$.response.status", "HTTP status differs."))

    compare_values(
        comparable_headers(expected_response.get("headers", {})),
        comparable_headers(actual_response.get("headers", {})),
        "$.response.headers",
        differences,
    )
    compare_values(
        expected_response.get("body"),
        actual_response.get("body"),
        "$.response.body",
        differences,
    )

    return Comparison(passed=not differences, differences=differences)


def main() -> int:
    args = parse_args()
    comparison = compare(load_fixture(args.expected), load_fixture(args.actual))
    print(json.dumps(comparison.to_dict(), indent=2))
    return 0 if comparison.passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
