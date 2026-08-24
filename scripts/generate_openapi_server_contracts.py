#!/usr/bin/env python3
"""Generate Go server contract interfaces from the committed OpenAPI document."""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Any


HTTP_METHODS = {"delete", "get", "patch", "post", "put"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate Go server contract definitions from OpenAPI."
    )
    parser.add_argument("--openapi", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--check", action="store_true")
    return parser.parse_args()


def parse_scalar(value: str) -> Any:
    value = value.strip()
    if value == "true":
        return True
    if value == "false":
        return False
    if value == "[]":
        return []
    if value.startswith("[") and value.endswith("]"):
        inner = value[1:-1].strip()
        if not inner:
            return []
        if '"' in inner:
            return json.loads(value)
        return [part.strip() for part in inner.split(",")]
    if value.startswith('"') and value.endswith('"'):
        return json.loads(value)
    return value


def openapi_operations(source: str) -> list[dict[str, Any]]:
    operations: list[dict[str, Any]] = []
    current_path: str | None = None
    current_method: str | None = None
    current: dict[str, Any] | None = None
    in_tags = False

    for line in source.splitlines():
        path_match = re.match(r"^  (/[^:]*):\s*$", line)
        if path_match:
            if current:
                operations.append(current)
            current_path = path_match.group(1)
            current_method = None
            current = None
            in_tags = False
            continue

        method_match = re.match(r"^    ([a-z]+):\s*$", line)
        if current_path and method_match and method_match.group(1) in HTTP_METHODS:
            if current:
                operations.append(current)
            current_method = method_match.group(1).upper()
            current = {
                "method": current_method,
                "path": current_path,
                "operation_id": "",
                "tags": [],
                "laravel_route": "",
                "authentication_mode": "",
                "trust_path": "",
                "tenant_context": False,
                "consumers": [],
            }
            in_tags = False
            continue

        if not current:
            continue

        stripped = line.strip()
        if stripped.startswith("operationId: "):
            current["operation_id"] = stripped.removeprefix("operationId: ").strip()
            in_tags = False
        elif stripped.startswith("tags: "):
            value = stripped.removeprefix("tags: ").strip()
            current["tags"] = [str(item) for item in parse_scalar(value)]
            in_tags = False
        elif stripped.startswith("x-idelium-laravel-route: "):
            current["laravel_route"] = parse_scalar(
                stripped.removeprefix("x-idelium-laravel-route: ")
            )
        elif stripped.startswith("x-idelium-authentication-mode: "):
            current["authentication_mode"] = parse_scalar(
                stripped.removeprefix("x-idelium-authentication-mode: ")
            )
        elif stripped.startswith("x-idelium-trust-path: "):
            current["trust_path"] = parse_scalar(
                stripped.removeprefix("x-idelium-trust-path: ")
            )
        elif stripped.startswith("x-idelium-tenant-context: "):
            current["tenant_context"] = parse_scalar(
                stripped.removeprefix("x-idelium-tenant-context: ")
            )
        elif stripped.startswith("x-idelium-consumers: "):
            current["consumers"] = parse_scalar(
                stripped.removeprefix("x-idelium-consumers: ")
            )
        elif stripped == "tags:":
            current["tags"] = []
            in_tags = True
        elif in_tags and stripped.startswith("- "):
            current["tags"].append(stripped.removeprefix("- ").strip())
        elif stripped and not stripped.startswith("- "):
            in_tags = False

    if current:
        operations.append(current)

    missing = [
        f"{operation['method']} {operation['path']}"
        for operation in operations
        if not operation["operation_id"]
    ]
    if missing:
        raise ValueError("OpenAPI operations missing operationId: " + ", ".join(missing))
    return operations


def exported_name(operation_id: str) -> str:
    parts = re.findall(r"[A-Za-z0-9]+", operation_id)
    if not parts:
        raise ValueError(f"Invalid operationId: {operation_id}")
    return "".join(part[:1].upper() + part[1:] for part in parts)


def go_string(value: str) -> str:
    return json.dumps(value)


def go_string_slice(values: list[str]) -> str:
    if not values:
        return "nil"
    return "[]string{" + ", ".join(go_string(value) for value in values) + "}"


def go_bool(value: bool) -> str:
    return "true" if value else "false"


def render(operations: list[dict[str, Any]]) -> str:
    lines = [
        "// Code generated by scripts/generate_openapi_server_contracts.py; DO NOT EDIT.",
        "package openapicontract",
        "",
        "import \"net/http\"",
        "",
        "// Operation describes one HTTP operation declared in api/openapi.yaml.",
        "type Operation struct {",
        "\tMethod             string",
        "\tPath               string",
        "\tOperationID        string",
        "\tTags               []string",
        "\tLaravelRoute       string",
        "\tAuthenticationMode string",
        "\tTrustPath          string",
        "\tTenantContext      bool",
        "\tConsumers          []string",
        "}",
        "",
        "// ServerInterface is the generated HTTP handler surface for the OpenAPI contract.",
        "type ServerInterface interface {",
    ]
    for operation in operations:
        lines.append(
            f"\t{exported_name(operation['operation_id'])}(http.ResponseWriter, *http.Request)"
        )
    lines.extend(
        [
            "}",
            "",
            "// Operations lists every OpenAPI operation in stable document order.",
            "var Operations = []Operation{",
        ]
    )
    for operation in operations:
        lines.extend(
            [
                "\t{",
                f"\t\tMethod:             {go_string(operation['method'])},",
                f"\t\tPath:               {go_string(operation['path'])},",
                f"\t\tOperationID:        {go_string(operation['operation_id'])},",
                f"\t\tTags:               {go_string_slice(operation['tags'])},",
                f"\t\tLaravelRoute:       {go_string(operation['laravel_route'])},",
                f"\t\tAuthenticationMode: {go_string(operation['authentication_mode'])},",
                f"\t\tTrustPath:          {go_string(operation['trust_path'])},",
                f"\t\tTenantContext:      {go_bool(operation['tenant_context'])},",
                f"\t\tConsumers:          {go_string_slice(operation['consumers'])},",
                "\t},",
            ]
        )
    lines.extend(["}", ""])
    return "\n".join(lines)


def main() -> int:
    args = parse_args()
    source = args.openapi.read_text(encoding="utf-8")
    generated = render(openapi_operations(source))
    if args.check:
        existing = args.output.read_text(encoding="utf-8")
        if existing != generated:
            raise SystemExit(f"{args.output} is not synchronized with {args.openapi}.")
        return 0
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(generated, encoding="utf-8")
    print(f"Generated {args.output}.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
