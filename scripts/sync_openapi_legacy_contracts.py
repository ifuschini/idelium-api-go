#!/usr/bin/env python3
"""Synchronize OpenAPI placeholder contracts for Laravel-owned routes."""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Any, Iterable


BEGIN = "  # BEGIN GENERATED LARAVEL COMPATIBILITY CONTRACTS"
END = "  # END GENERATED LARAVEL COMPATIBILITY CONTRACTS"
HTTP_METHODS = {"DELETE", "GET", "PATCH", "POST", "PUT"}
GO_CUTOVER_GATED_ROUTES = {
    "PUT /api/admin/identity/accounts/{user}/break-glass",
    "POST /api/admin/identity/accounts/{user}/break-glass/test",
    "GET|HEAD /api/admin/identity/providers",
    "POST /api/admin/identity/providers",
    "POST /api/admin/identity/providers/{identityProvider}/scim/users",
    "POST /api/admin/profile/mfa/confirm",
    "POST /api/admin/profile/mfa/enroll",
    "POST /api/admin/profile/mfa/step-up",
    "POST /api/oidc/token-exchange",
    "POST /api/sso/{identityProvider}/oidc/callback",
    "POST /api/sso/{identityProvider}/saml/callback",
    "POST /api/sso/{identityProvider}/start",
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Add OpenAPI contracts for all production-visible Laravel routes."
    )
    parser.add_argument("--inventory", type=Path, required=True)
    parser.add_argument("--consumer-map", type=Path, required=True)
    parser.add_argument("--openapi", type=Path, required=True)
    parser.add_argument("--check", action="store_true")
    return parser.parse_args()


def load_inventory(path: Path) -> list[dict[str, Any]]:
    document = json.loads(path.read_text(encoding="utf-8"))
    routes = document.get("routes")
    if not isinstance(routes, list):
        raise ValueError(f"{path} must contain a routes array.")
    return [
        route
        for route in routes
        if route.get("authentication_mode") != "development-only"
    ]


def load_consumer_index(path: Path) -> dict[tuple[str, str], list[str]]:
    document = json.loads(path.read_text(encoding="utf-8"))
    routes = document.get("routes")
    if not isinstance(routes, list):
        raise ValueError(f"{path} must contain a routes array.")
    return {
        (route["method"], route["path"]): [
            consumer["id"] for consumer in route.get("consumers", [])
        ]
        for route in routes
    }


def gateway_path(path: str) -> str:
    return path.removeprefix("/api") or "/"


def route_methods(method: str) -> list[str]:
    return [part for part in method.split("|") if part in HTTP_METHODS]


def existing_paths(source: str) -> set[str]:
    paths: set[str] = set()
    source = remove_generated_block(source)
    for line in source.splitlines():
        match = re.match(r"^  (/[^:]+):\s*$", line)
        if match:
            paths.add(match.group(1))
    return paths


def operation_id(method: str, path: str) -> str:
    parts = re.findall(r"[A-Za-z0-9]+", f"{method.lower()} {path}")
    if not parts:
        return "legacyCompatibilityRoute"
    first, rest = parts[0], parts[1:]
    return first + "".join(part[:1].upper() + part[1:] for part in rest)


def yaml_string(value: Any) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if value is None:
        return "null"
    text = str(value)
    escaped = text.replace("\\", "\\\\").replace('"', '\\"')
    return f'"{escaped}"'


def scalar_list(values: Iterable[str]) -> str:
    return "[" + ", ".join(yaml_string(value) for value in values) + "]"


def generated_operation(route: dict[str, Any], method: str) -> list[str]:
    path = gateway_path(route["path"])
    path_params = re.findall(r"{([^}]+)}", path)
    consumer_ids = route.get("consumer_ids", [])
    route_id = f"{route['method']} {route['path']}"
    go_cutover_gated = route_id in GO_CUTOVER_GATED_ROUTES
    responses = ['        "200":', "          $ref: \"#/components/responses/LegacyCompatibilityResponse\""]
    if go_cutover_gated:
        responses.extend(
            [
                '        "501":',
                "          $ref: \"#/components/responses/LegacyCompatibilityError\"",
            ]
        )
    if route["authentication_mode"] != "public":
        responses.extend(
            [
                '        "401":',
                "          $ref: \"#/components/responses/Unauthorized\"",
                '        "403":',
                "          $ref: \"#/components/responses/Forbidden\"",
            ]
        )
    if method in {"PATCH", "POST", "PUT"}:
        responses.extend(
            [
                '        "422":',
                "          $ref: \"#/components/responses/ValidationError\"",
            ]
        )
    responses.extend(
        [
            "        default:",
            "          $ref: \"#/components/responses/LegacyCompatibilityError\"",
        ]
    )

    lines = [
        f"    {method.lower()}:",
        f"      operationId: {operation_id(method, path)}",
        "      summary: Laravel compatibility contract",
        "      description: >-",
        "        Placeholder contract for an externally visible Laravel route. It keeps",
        "        consumers, authentication, tenant scope, and migration ownership visible",
        "        until this route is implemented natively in Go.",
        "      tags: [Laravel Compatibility]",
        f"      x-idelium-laravel-route: {yaml_string(route['path'])}",
        f"      x-idelium-controller: {yaml_string(route['controller'])}",
        f"      x-idelium-current-owner: {yaml_string(route['current_owner'])}",
        f"      x-idelium-trust-path: {yaml_string(route['trust_path'])}",
        f"      x-idelium-authentication-mode: {yaml_string(route['authentication_mode'])}",
        f"      x-idelium-tenant-context: {yaml_string(route['tenant_context'])}",
        f"      x-idelium-consumers: {scalar_list(consumer_ids)}",
    ]
    if go_cutover_gated:
        lines.extend(
            [
                "      x-idelium-go-cutover-gate: true",
                "      x-idelium-go-cutover-error-code: \"IDENTITY_MIGRATION_DISABLED\"",
            ]
        )
    if path_params:
        lines.append("      parameters:")
        for name in path_params:
            lines.extend(
                [
                    f"        - name: {name}",
                    "          in: path",
                    "          required: true",
                    "          schema:",
                    "            type: string",
                ]
            )
    if method in {"PATCH", "POST", "PUT"}:
        lines.extend(
            [
                "      requestBody:",
                "        required: false",
                "        content:",
                "          application/json:",
                "            schema:",
                "              $ref: \"#/components/schemas/LegacyCompatibilityPayload\"",
                "          multipart/form-data:",
                "            schema:",
                "              $ref: \"#/components/schemas/LegacyCompatibilityPayload\"",
            ]
        )
    lines.append("      responses:")
    lines.extend(responses)
    return lines


def generated_paths(routes: list[dict[str, Any]], documented_paths: set[str]) -> str:
    grouped: dict[str, list[tuple[dict[str, Any], str]]] = {}
    for route in routes:
        path = gateway_path(route["path"])
        if path in documented_paths:
            continue
        grouped.setdefault(path, [])
        for method in route_methods(route["method"]):
            grouped[path].append((route, method))

    lines = [BEGIN]
    for path, operations in grouped.items():
        lines.append(f"  {path}:")
        for route, method in operations:
            lines.extend(generated_operation(route, method))
    lines.append(END)
    return "\n".join(lines)


def remove_generated_block(source: str) -> str:
    if BEGIN not in source and END not in source:
        return source
    pattern = re.compile(rf"\n?{re.escape(BEGIN)}.*?{re.escape(END)}\n?", re.DOTALL)
    return pattern.sub("\n", source)


def insert_generated_block(source: str, generated: str) -> str:
    stripped = remove_generated_block(source)
    marker = "\ncomponents:\n"
    if marker not in stripped:
        raise ValueError("OpenAPI document must contain a top-level components section.")
    prefix, suffix = stripped.split(marker, 1)
    return prefix.rstrip() + "\n" + generated + "\n" + marker.lstrip() + suffix


def main() -> int:
    args = parse_args()
    routes = load_inventory(args.inventory)
    consumer_index = load_consumer_index(args.consumer_map)
    for route in routes:
        route["consumer_ids"] = consumer_index.get((route["method"], route["path"]), [])
    source = args.openapi.read_text(encoding="utf-8")
    updated = insert_generated_block(source, generated_paths(routes, existing_paths(source)))
    if args.check:
        if source != updated:
            raise SystemExit("api/openapi.yaml is not synchronized with Laravel routes.")
        return 0
    args.openapi.write_text(updated, encoding="utf-8")
    print(f"Synchronized {len(routes)} Laravel compatibility route contracts.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
