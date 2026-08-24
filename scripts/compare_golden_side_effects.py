#!/usr/bin/env python3
"""Compare sanitized Laravel and Go golden fixtures for mutation side effects."""

from __future__ import annotations

import argparse
import copy
import json
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


SAFE_READ_METHODS = {"GET", "HEAD"}
MUTATION_METHODS = {"DELETE", "PATCH", "POST", "PUT"}
NORMALIZATION_MARKERS = {
    "correlation": "[NORMALIZED_CORRELATION_ID]",
    "timestamp": "[NORMALIZED_TIMESTAMP]",
    "uuid": "[NORMALIZED_UUID]",
    "id": "[NORMALIZED_IDENTIFIER]",
}


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
        description="Compare sanitized mutation side effects between Laravel and Go fixtures."
    )
    parser.add_argument("--expected", type=Path, required=True)
    parser.add_argument("--actual", type=Path, required=True)
    return parser.parse_args()


def load_fixture(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def normalization_marker(rule: str, path: str) -> str:
    text = f"{rule} {path}".lower()
    for keyword, marker in NORMALIZATION_MARKERS.items():
        if keyword in text:
            return marker
    return "[NORMALIZED_VALUE]"


def path_tokens(path: str) -> list[str | int]:
    if not path.startswith("$."):
        raise ValueError(f"Unsupported normalization path: {path}")
    tokens: list[str | int] = []
    for part in path[2:].split("."):
        if not part:
            raise ValueError(f"Unsupported normalization path: {path}")
        match = re.match(r"^([^\[]+)((?:\[\d+\])*)$", part)
        if not match:
            raise ValueError(f"Unsupported normalization path: {path}")
        tokens.append(match.group(1))
        for index in re.findall(r"\[(\d+)\]", match.group(2)):
            tokens.append(int(index))
    return tokens


def set_path_value(document: Any, path: str, value: Any) -> bool:
    current = document
    tokens = path_tokens(path)
    for token in tokens[:-1]:
        if isinstance(token, int):
            if not isinstance(current, list) or token >= len(current):
                return False
            current = current[token]
            continue
        if not isinstance(current, dict) or token not in current:
            return False
        current = current[token]
    final = tokens[-1]
    if isinstance(final, int):
        if not isinstance(current, list) or final >= len(current):
            return False
        current[final] = value
        return True
    if not isinstance(current, dict) or final not in current:
        return False
    current[final] = value
    return True


def collect_normalizations(*fixtures: dict[str, Any]) -> dict[str, str]:
    normalizations: dict[str, str] = {}
    for fixture in fixtures:
        for entry in fixture.get("normalizations", []):
            path = entry.get("path")
            rule = entry.get("rule", "")
            if isinstance(path, str) and path.startswith("$."):
                normalizations[path] = normalization_marker(str(rule), path)
    return normalizations


def normalized_fixture(
    fixture: dict[str, Any],
    normalizations: dict[str, str],
    differences: list[Difference],
    label: str,
) -> dict[str, Any]:
    result = copy.deepcopy(fixture)
    for path, marker in sorted(normalizations.items()):
        if not path.startswith("$.sideEffects"):
            continue
        try:
            applied = set_path_value(result, path, marker)
        except ValueError:
            differences.append(
                Difference(f"$.{label}.normalizations", "Normalization path is not supported.")
            )
            continue
        if not applied:
            differences.append(
                Difference(path, "Normalization path is declared but not present in the fixture.")
            )
    return result


def ensure_mutation_fixture(fixture: dict[str, Any], label: str) -> list[Difference]:
    differences: list[Difference] = []
    route = fixture.get("route", {})
    method = route.get("method")
    if method in SAFE_READ_METHODS or method not in MUTATION_METHODS:
        differences.append(
            Difference(
                f"$.{label}.route.method",
                "Only DELETE, PATCH, POST, and PUT fixtures can use the side-effect comparator.",
            )
        )
    side_effects = fixture.get("sideEffects")
    if not isinstance(side_effects, list) or not side_effects:
        differences.append(
            Difference(f"$.{label}.sideEffects", "Mutation fixtures must declare side effects.")
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
    differences.extend(ensure_mutation_fixture(expected, "expected"))
    differences.extend(ensure_mutation_fixture(actual, "actual"))
    normalizations = collect_normalizations(expected, actual)
    expected = normalized_fixture(expected, normalizations, differences, "expected")
    actual = normalized_fixture(actual, normalizations, differences, "actual")

    for field_name in ("method", "path", "trustPath", "tenantOwned"):
        expected_value = expected.get("route", {}).get(field_name)
        actual_value = actual.get("route", {}).get(field_name)
        if expected_value != actual_value:
            differences.append(
                Difference(f"$.route.{field_name}", "Route contract metadata differs.")
            )

    compare_values(
        expected.get("sideEffects"),
        actual.get("sideEffects"),
        "$.sideEffects",
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
