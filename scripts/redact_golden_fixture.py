#!/usr/bin/env python3
"""Redact sensitive values from a candidate golden fixture."""

from __future__ import annotations

import argparse
import copy
import json
from pathlib import Path
from typing import Any

from validate_golden_fixtures import (
    FORBIDDEN_HEADERS,
    SECRET_PATTERNS,
    SENSITIVE_KEYS,
    normalized_key,
)


REDACTED = "[REDACTED]"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Create a sanitized golden fixture from a candidate capture."
    )
    parser.add_argument("--input", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    return parser.parse_args()


def load_document(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def add_redaction(redactions: list[dict[str, str]], path: str, rule: str) -> None:
    record = {"path": path, "rule": rule}
    if record not in redactions:
        redactions.append(record)


def redact_headers(message: dict[str, Any], path: str, redactions: list[dict[str, str]]) -> None:
    headers = message.get("headers")
    if not isinstance(headers, dict):
        return
    for name in list(headers):
        if str(name).lower() in FORBIDDEN_HEADERS:
            del headers[name]
            add_redaction(
                redactions,
                f"{path}.headers.{name}",
                "Remove sensitive HTTP header before persisting the fixture.",
            )


def redact_tree(value: Any, path: str, redactions: list[dict[str, str]]) -> Any:
    if isinstance(value, dict):
        sanitized: dict[str, Any] = {}
        for key, child in value.items():
            child_path = f"{path}.{key}"
            if normalized_key(str(key)) in SENSITIVE_KEYS:
                add_redaction(
                    redactions,
                    child_path,
                    "Remove sensitive field before persisting the fixture.",
                )
                continue
            sanitized[key] = redact_tree(child, child_path, redactions)
        return sanitized
    if isinstance(value, list):
        return [
            redact_tree(child, f"{path}[{index}]", redactions)
            for index, child in enumerate(value)
        ]
    if isinstance(value, str) and any(pattern.search(value) for pattern in SECRET_PATTERNS):
        add_redaction(
            redactions,
            path,
            "Replace credential-like string before persisting the fixture.",
        )
        return REDACTED
    return value


def redact_fixture(document: dict[str, Any]) -> dict[str, Any]:
    sanitized = copy.deepcopy(document)
    redactions = list(sanitized.get("redactions", []))
    for message_name in ("request", "response"):
        message = sanitized.get(message_name)
        if isinstance(message, dict):
            redact_headers(message, f"$.{message_name}", redactions)
            if "query" in message:
                message["query"] = redact_tree(
                    message.get("query", {}), f"$.{message_name}.query", redactions
                )
            if "body" in message:
                message["body"] = redact_tree(
                    message.get("body"), f"$.{message_name}.body", redactions
                )
    if "sideEffects" in sanitized:
        sanitized["sideEffects"] = redact_tree(
            sanitized.get("sideEffects"), "$.sideEffects", redactions
        )
    sanitized["redactions"] = sorted(redactions, key=lambda item: (item["path"], item["rule"]))
    return sanitized


def write_redacted(input_path: Path, output_path: Path) -> None:
    sanitized = redact_fixture(load_document(input_path))
    output_path.write_text(json.dumps(sanitized, indent=2) + "\n", encoding="utf-8")


def main_for_test(input_path: Path, output_path: Path) -> int:
    write_redacted(input_path, output_path)
    return 0


def main() -> int:
    args = parse_args()
    write_redacted(args.input, args.output)
    print(f"Wrote sanitized fixture to {args.output}.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
