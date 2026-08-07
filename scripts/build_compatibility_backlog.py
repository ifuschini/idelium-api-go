#!/usr/bin/env python3
"""Build the initial compatibility-contract backlog from route evidence."""

from __future__ import annotations

import argparse
import json
import re
from collections import Counter
from pathlib import Path
from typing import Any


HTTP_METHODS = {"delete", "get", "head", "options", "patch", "post", "put"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Build compatibility-contract work items for Laravel routes."
    )
    parser.add_argument("--inventory", type=Path, required=True)
    parser.add_argument("--consumer-map", type=Path, required=True)
    parser.add_argument("--openapi", type=Path, required=True)
    parser.add_argument("--output-json", type=Path, required=True)
    parser.add_argument("--output-markdown", type=Path, required=True)
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def openapi_operations(source: str) -> set[tuple[str, str]]:
    operations: set[tuple[str, str]] = set()
    current_path: str | None = None
    for line in source.splitlines():
        path_match = re.match(r"^  (/[^:]+):\s*$", line)
        if path_match:
            current_path = path_match.group(1)
            continue
        method_match = re.match(r"^    ([a-z]+):\s*$", line)
        if current_path and method_match and method_match.group(1) in HTTP_METHODS:
            operations.add((method_match.group(1).upper(), current_path))
    return operations


def gateway_path(path: str) -> str:
    return path.removeprefix("/api") or "/"


def route_methods(method: str) -> set[str]:
    return {part for part in method.split("|") if part != "HEAD"}


def is_documented(route: dict[str, Any], operations: set[tuple[str, str]]) -> bool:
    path = gateway_path(route["path"])
    return any((method, path) in operations for method in route_methods(route["method"]))


def migration_wave(route: dict[str, Any]) -> int:
    path = route["path"]
    methods = route_methods(route["method"])

    if path in {"/api/admin/platforms/types", "/api/admin/platforms/status"}:
        return 3
    if path.startswith("/api/ideliumcl/projects/") or path == "/api/ideliumcl/agents/register":
        return 8
    if path.startswith("/api/ideliumcl/"):
        return 4 if methods == {"GET"} else 5
    if path.startswith("/api/ideliumrunner/"):
        return 8
    if "/parallel-runs" in path or path.startswith("/api/admin/agents"):
        return 8
    if any(
        marker in path
        for marker in (
            "/stepsperfomed/",
            "/testsperfomed/",
            "/testcyclesperfomed/",
            "/result-exports",
            "/artifacts",
            "/asset-impact/",
            "/asset-versions/",
            "/grid/",
            "/integrations",
            "/integration-deliveries",
        )
    ):
        return 7
    if path == "/api/admin/launchtest":
        return 8
    if path.startswith("/api/admin/platforms/"):
        return 3 if methods == {"GET"} else 6
    if any(
        path.startswith(prefix)
        for prefix in (
            "/api/admin/projects",
            "/api/admin/environments",
            "/api/admin/plugins",
            "/api/admin/steps",
            "/api/admin/tests",
            "/api/admin/testcycles",
            "/api/admin/importtest",
        )
    ):
        return 6
    if route["trust_path"] in {"browser-session", "internal-service"}:
        return 9
    return 0


def priority(route: dict[str, Any], consumer_ids: list[str]) -> str:
    if route["path"] in {"/api/clear", "/api/oidc/token-exchange"}:
        return "critical"
    if route["trust_path"] in {"api-key", "run-token", "internal-service"}:
        return "high"
    if consumer_ids:
        return "high"
    if route["trust_path"] == "public-operational":
        return "high"
    return "normal"


def build_backlog(
    inventory: dict[str, Any],
    consumer_map: dict[str, Any],
    operations: set[tuple[str, str]],
) -> dict[str, Any]:
    if inventory["route_digest_sha256"] != consumer_map["route_inventory_digest_sha256"]:
        raise ValueError("Consumer map does not match the Laravel route inventory.")

    consumer_index = {
        (route["method"], route["path"]): [entry["id"] for entry in route["consumers"]]
        for route in consumer_map["routes"]
    }
    items = []
    exclusions = []
    for route in inventory["routes"]:
        identity = (route["method"], route["path"])
        if route["authentication_mode"] == "development-only":
            exclusions.append(
                {
                    "method": route["method"],
                    "path": route["path"],
                    "reason": "Development-only Ignition route; prohibit it in production rather than migrate it.",
                }
            )
            continue

        consumer_ids = consumer_index[identity]
        documented = is_documented(route, operations)
        items.append(
            {
                "id": f"{route['method']} {route['path']}",
                "method": route["method"],
                "path": route["path"],
                "controller": route["controller"],
                "current_owner": route["current_owner"],
                "trust_path": route["trust_path"],
                "authentication_mode": route["authentication_mode"],
                "tenant_context": route["tenant_context"],
                "consumer_ids": consumer_ids,
                "migration_wave": migration_wave(route),
                "priority": priority(route, consumer_ids),
                "openapi_status": "documented" if documented else "pending",
                "fixture_status": "pending",
                "differential_test_status": "pending",
                "security_review_status": "pending",
                "rollout_status": "laravel-owned",
                "required_contract_evidence": [
                    "request-and-validation",
                    "response-and-status-codes",
                    "authorization-and-tenant-isolation",
                    "side-effects-and-idempotency",
                    "redaction-and-audit",
                    "sanitized-laravel-fixture",
                    "laravel-go-differential-test",
                    "consumer-smoke-test",
                    "rollout-and-rollback",
                ],
            }
        )

    return {
        "schema_version": 1,
        "route_inventory_digest_sha256": inventory["route_digest_sha256"],
        "public_route_count": len(items),
        "excluded_development_route_count": len(exclusions),
        "items": items,
        "exclusions": exclusions,
    }


def markdown(document: dict[str, Any]) -> str:
    wave_counts = Counter(item["migration_wave"] for item in document["items"])
    contract_counts = Counter(item["openapi_status"] for item in document["items"])
    priority_counts = Counter(item["priority"] for item in document["items"])
    lines = [
        "# Compatibility Contract Backlog",
        "",
        "This generated backlog creates one compatibility record for every Laravel route",
        "that is reachable outside development-only tooling. It is the contract gate for",
        "moving route ownership to Go; a route cannot move while required evidence remains",
        "pending.",
        "",
        "## Summary",
        "",
        f"- Public Laravel route records: **{document['public_route_count']}**",
        f"- Excluded development-only routes: **{document['excluded_development_route_count']}**",
        f"- OpenAPI-documented Laravel operations: **{contract_counts['documented']}**",
        f"- Operations awaiting OpenAPI contracts: **{contract_counts['pending']}**",
        "",
        "| Priority | Records |",
        "| --- | ---: |",
    ]
    lines.extend(
        f"| `{name}` | {count} |" for name, count in sorted(priority_counts.items())
    )
    lines.extend(["", "| Migration wave | Records |", "| --- | ---: |"])
    lines.extend(f"| Wave {wave} | {count} |" for wave, count in sorted(wave_counts.items()))
    lines.extend(
        [
            "",
            "## Contract gates",
            "",
            "Each record requires request validation, response/status behavior, authorization",
            "and tenant isolation, side effects and idempotency, redaction and audit behavior,",
            "a sanitized Laravel fixture, a Laravel-Go differential test, applicable consumer",
            "smoke coverage, and explicit rollout/rollback evidence.",
            "",
            "The two platform catalogue reads already present in the Go OpenAPI document are",
            "marked documented. Their fixtures and differential tests intentionally remain",
            "pending until the Wave 2 harness is available.",
            "",
            "## Route backlog",
            "",
            "| Priority | Wave | Method | Path | Trust path | Consumers | OpenAPI | Owner |",
            "| --- | ---: | --- | --- | --- | --- | --- | --- |",
        ]
    )
    for item in document["items"]:
        consumers = ", ".join(f"`{value}`" for value in item["consumer_ids"]) or "—"
        lines.append(
            f"| `{item['priority']}` | {item['migration_wave']} | `{item['method']}` | "
            f"`{item['path']}` | `{item['trust_path']}` | {consumers} | "
            f"`{item['openapi_status']}` | `{item['current_owner']}` |"
        )
    lines.extend(
        [
            "",
            "## Explicit exclusions",
            "",
            "| Method | Path | Decision |",
            "| --- | --- | --- |",
        ]
    )
    for exclusion in document["exclusions"]:
        lines.append(
            f"| `{exclusion['method']}` | `{exclusion['path']}` | {exclusion['reason']} |"
        )
    lines.extend(
        [
            "",
            "## Deployment and rollback",
            "",
            "This backlog changes no runtime behavior, traffic ownership, or database schema.",
            "Rollback is a Git revert. Subsequent route migrations must update the relevant",
            "record atomically with their contract, fixture, test, ownership, and rollback",
            "evidence.",
            "",
            "Regenerate the backlog whenever route, consumer, or OpenAPI evidence changes:",
            "",
            "```sh",
            "python3 scripts/build_compatibility_backlog.py \\",
            "  --inventory docs/contracts/laravel-routes.json \\",
            "  --consumer-map docs/contracts/consumer-route-map.json \\",
            "  --openapi api/openapi.yaml \\",
            "  --output-json docs/contracts/compatibility-backlog.json \\",
            "  --output-markdown docs/contracts/compatibility-backlog.md",
            "```",
            "",
        ]
    )
    return "\n".join(lines)


def main() -> int:
    args = parse_args()
    document = build_backlog(
        load_json(args.inventory),
        load_json(args.consumer_map),
        openapi_operations(args.openapi.read_text(encoding="utf-8")),
    )
    args.output_json.write_text(json.dumps(document, indent=2) + "\n", encoding="utf-8")
    args.output_markdown.write_text(markdown(document), encoding="utf-8")
    print(f"Created {document['public_route_count']} compatibility records.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
