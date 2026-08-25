#!/usr/bin/env python3
"""Compare sanitized Laravel and Go CLI configuration graph fixtures."""

from __future__ import annotations

import argparse
import copy
import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


NORMALIZED_SOURCE = {
    "runtime": "[NORMALIZED_RUNTIME]",
    "repository": "[NORMALIZED_REPOSITORY]",
    "revision": "[NORMALIZED_REVISION]",
    "capturedAt": "[NORMALIZED_TIMESTAMP]",
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
                {"path": difference.path, "reason": difference.reason}
                for difference in self.differences
            ],
        }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--expected", type=Path, required=True)
    parser.add_argument("--actual", type=Path, required=True)
    return parser.parse_args()


def load_fixture(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def normalized_fixture(fixture: dict[str, Any]) -> dict[str, Any]:
    result = copy.deepcopy(fixture)
    source = result.setdefault("source", {})
    for field_name, marker in NORMALIZED_SOURCE.items():
        if field_name in source:
            source[field_name] = marker
    return result


def ids(items: list[dict[str, Any]]) -> set[int]:
    return {int(item["id"]) for item in items}


def validate_graph(fixture: dict[str, Any], label: str) -> list[Difference]:
    differences: list[Difference] = []
    graph = fixture.get("graph", {})
    customer_id = graph.get("customerId")
    project_id = graph.get("projectId")

    tests = graph.get("tests", [])
    steps = graph.get("steps", [])
    plugins = graph.get("plugins", [])
    environments = graph.get("environments", [])
    test_ids = ids(tests)
    step_ids = ids(steps)

    cycle = graph.get("cycle", {})
    for test_id in cycle.get("configTestIds", []):
        if int(test_id) not in test_ids:
            differences.append(Difference(f"$.{label}.graph.cycle.configTestIds", f"Referenced test {test_id} is missing."))

    for test in tests:
        if test.get("idCostumer") != customer_id:
            differences.append(Difference(f"$.{label}.graph.tests[{test.get('id')}].idCostumer", "Test customer does not match graph customer."))
        if test.get("idProject") != project_id:
            differences.append(Difference(f"$.{label}.graph.tests[{test.get('id')}].idProject", "Test project does not match graph project."))
        for step_id in test.get("configStepIds", []):
            if int(step_id) not in step_ids:
                differences.append(Difference(f"$.{label}.graph.tests[{test.get('id')}].configStepIds", f"Referenced step {step_id} is missing."))

    for collection_name, collection in (
        ("steps", steps),
        ("plugins", plugins),
        ("environments", environments),
    ):
        for item in collection:
            if item.get("idCostumer") != customer_id:
                differences.append(Difference(f"$.{label}.graph.{collection_name}[{item.get('id')}].idCostumer", "Resource customer does not match graph customer."))
            if item.get("idProject") != project_id:
                differences.append(Difference(f"$.{label}.graph.{collection_name}[{item.get('id')}].idProject", "Resource project does not match graph project."))

    return differences


def compare_values(expected: Any, actual: Any, path: str, differences: list[Difference]) -> None:
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
    differences.extend(validate_graph(expected, "expected"))
    differences.extend(validate_graph(actual, "actual"))
    expected = normalized_fixture(expected)
    actual = normalized_fixture(actual)
    compare_values(expected, actual, "$", differences)
    return Comparison(passed=not differences, differences=differences)


def main() -> int:
    result = compare(load_fixture(parse_args().expected), load_fixture(parse_args().actual))
    print(json.dumps(result.to_dict(), indent=2, sort_keys=True))
    return 0 if result.passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
