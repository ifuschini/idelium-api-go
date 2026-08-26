#!/usr/bin/env python3
"""Build the staging route cutover manifest for the Laravel-to-Go migration."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any


OPENAPI_METHODS = {"get", "post", "put", "patch", "delete"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--matrix",
        type=Path,
        default=Path("docs/contracts/migration-ownership-matrix.json"),
    )
    parser.add_argument(
        "--gateway",
        type=Path,
        default=Path("docs/contracts/gateway-route-ownership.json"),
    )
    parser.add_argument(
        "--openapi",
        type=Path,
        default=Path("api/openapi.yaml"),
    )
    parser.add_argument(
        "--output-json",
        type=Path,
        default=Path("docs/contracts/staging-route-cutover.json"),
    )
    parser.add_argument(
        "--output-markdown",
        type=Path,
        default=Path("docs/contracts/staging-route-cutover.md"),
    )
    parser.add_argument(
        "--check",
        action="store_true",
        help="Fail if the generated outputs differ from the files on disk.",
    )
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def operation_matches_route(operation_method: str, route_method: str) -> bool:
    return operation_method.upper() in route_method.split("|")


def route_key(method: str, path: str) -> str:
    return f"{method} {path}"


def parse_openapi_cutover_gates(openapi: Path) -> dict[str, dict[str, str]]:
    """Return Go fail-closed route gates from OpenAPI vendor extensions."""

    gates: dict[str, dict[str, str]] = {}
    current_public_path: str | None = None
    current_method: str | None = None
    block: list[str] = []

    def flush() -> None:
        nonlocal block, current_method, current_public_path
        if not current_public_path or not current_method or not block:
            block = []
            return
        text = "\n".join(block)
        if "x-idelium-go-cutover-gate: true" not in text:
            block = []
            return
        laravel_route = extract_yaml_string(text, "x-idelium-laravel-route")
        error_code = extract_yaml_string(text, "x-idelium-go-cutover-error-code")
        if not laravel_route:
            raise ValueError(
                f"OpenAPI gate for {current_method.upper()} {current_public_path} "
                "is missing x-idelium-laravel-route."
            )
        if not error_code:
            raise ValueError(
                f"OpenAPI gate for {current_method.upper()} {current_public_path} "
                "is missing x-idelium-go-cutover-error-code."
            )
        gates[route_key(current_method.upper(), laravel_route)] = {
            "public_path": current_public_path,
            "laravel_route": laravel_route,
            "error_code": error_code,
        }
        block = []

    for line in openapi.read_text(encoding="utf-8").splitlines():
        path_match = re.match(r"^  (/[^:]+):\s*$", line)
        if path_match:
            flush()
            current_public_path = path_match.group(1)
            current_method = None
            block = []
            continue

        method_match = re.match(r"^    (get|post|put|patch|delete):\s*$", line)
        if method_match and method_match.group(1) in OPENAPI_METHODS:
            flush()
            current_method = method_match.group(1)
            block = []
            continue

        if current_method:
            block.append(line)

    flush()
    return gates


def extract_yaml_string(text: str, key: str) -> str | None:
    match = re.search(rf'{re.escape(key)}:\s+"([^"]+)"', text)
    if match:
        return match.group(1)
    match = re.search(rf"{re.escape(key)}:\s+([^\\n]+)", text)
    if match:
        return match.group(1).strip()
    return None


def gateway_go_route_ids(gateway: dict[str, Any]) -> set[str]:
    route_ids = set()
    for route in gateway.get("routes", []):
        if route.get("owner") == "go":
            route_ids.add(route["route_id"])
    return route_ids


def matching_gate(
    route: dict[str, Any], gates: dict[str, dict[str, str]]
) -> dict[str, str] | None:
    for method in route["method"].split("|"):
        gate = gates.get(route_key(method, route["path"]))
        if gate:
            return gate
    return None


def build_manifest(
    matrix: dict[str, Any], gateway: dict[str, Any], openapi_gates: dict[str, dict[str, str]]
) -> dict[str, Any]:
    gateway_go = gateway_go_route_ids(gateway)
    routes: list[dict[str, Any]] = []
    blockers: list[dict[str, Any]] = []
    go_owned = 0
    go_gated = 0
    laravel_blocked = 0

    for route in matrix["routes"]:
        entry = {
            "route_id": route["route_id"],
            "method": route["method"],
            "path": route["path"],
            "aggregate": route["aggregate"],
            "operation_kind": route["operation_kind"],
            "tenant_context": route["tenant_context"],
            "migration_wave": route["migration_wave"],
            "current_owner": route["owner"],
            "dual_writes_allowed": False,
        }
        gate = matching_gate(route, openapi_gates)
        if route["owner"] == "go":
            entry.update(
                {
                    "staging_state": "ready",
                    "staging_owner": "go",
                    "routing_action": "send-to-go",
                    "gateway_route_configured": route["route_id"] in gateway_go,
                    "fallback_owner": "laravel",
                }
            )
            go_owned += 1
        elif gate:
            entry.update(
                {
                    "staging_state": "gated",
                    "staging_owner": "go-fail-closed",
                    "routing_action": "send-to-go-gate",
                    "gateway_route_configured": False,
                    "fallback_owner": "laravel",
                    "error_code": gate["error_code"],
                    "go_public_path": gate["public_path"],
                }
            )
            go_gated += 1
        else:
            entry.update(
                {
                    "staging_state": "blocked",
                    "staging_owner": "laravel",
                    "routing_action": "keep-on-laravel",
                    "gateway_route_configured": False,
                    "fallback_owner": "laravel",
                    "blocker": "Route has no Go implementation and no explicit Go fail-closed gate.",
                }
            )
            blockers.append(
                {
                    "route_id": route["route_id"],
                    "aggregate": route["aggregate"],
                    "migration_wave": route["migration_wave"],
                    "reason": entry["blocker"],
                }
            )
            laravel_blocked += 1
        routes.append(entry)

    route_count = len(routes)
    status = "ready" if laravel_blocked == 0 else "blocked"
    return {
        "schema_version": 1,
        "cutover_id": "staging-route-cutover",
        "status": status,
        "production_enabled": False,
        "summary": {
            "route_count": route_count,
            "go_owned_routes": go_owned,
            "go_fail_closed_routes": go_gated,
            "laravel_blocker_routes": laravel_blocked,
            "gateway_go_routes": len(gateway_go),
        },
        "policy": {
            "environment": "staging",
            "dual_writes_allowed": False,
            "default_fallback_owner": "laravel",
            "production_cutover_requires_zero_blockers": True,
            "unimplemented_go_routes_fail_closed": True,
            "secrets_in_manifest_allowed": False,
        },
        "blockers": blockers,
        "routes": routes,
    }


def render_markdown(manifest: dict[str, Any]) -> str:
    summary = manifest["summary"]
    lines = [
        "# Staging Route Cutover Manifest",
        "",
        "This generated manifest is the staging checklist for moving route",
        "ownership from Laravel to Go. It deliberately keeps production disabled",
        "until every route is either implemented in Go or exposed as an explicit",
        "Go fail-closed gate. Application-level dual writes remain prohibited.",
        "",
        "## Status",
        "",
        "| Field | Value |",
        "| --- | --- |",
        f"| Cutover status | `{manifest['status']}` |",
        f"| Production enabled | `{str(manifest['production_enabled']).lower()}` |",
        f"| Route count | {summary['route_count']} |",
        f"| Go-owned routes | {summary['go_owned_routes']} |",
        f"| Go fail-closed routes | {summary['go_fail_closed_routes']} |",
        f"| Laravel blocker routes | {summary['laravel_blocker_routes']} |",
        f"| Gateway Go routes | {summary['gateway_go_routes']} |",
        "",
        "## Staging policy",
        "",
        "- Staging may route `ready` entries to Go.",
        "- Staging may route `gated` entries to Go only when the expected",
        "  fail-closed diagnostic is acceptable for that rehearsal.",
        "- `blocked` entries stay on Laravel until a Go implementation or",
        "  fail-closed gate is merged.",
        "- Production cutover remains disabled until there are zero blockers.",
        "- Dual writes are not allowed.",
        "",
        "## Blocker summary by aggregate",
        "",
        "| Aggregate | Blocked routes |",
        "| --- | ---: |",
    ]

    aggregate_counts: dict[str, int] = {}
    for blocker in manifest["blockers"]:
        aggregate_counts[blocker["aggregate"]] = aggregate_counts.get(blocker["aggregate"], 0) + 1
    for aggregate, count in sorted(aggregate_counts.items()):
        lines.append(f"| `{aggregate}` | {count} |")
    if not aggregate_counts:
        lines.append("| none | 0 |")

    lines.extend(
        [
            "",
            "## Route decisions",
            "",
            "| Method | Path | Aggregate | State | Staging owner | Action |",
            "| --- | --- | --- | --- | --- | --- |",
        ]
    )
    for route in manifest["routes"]:
        lines.append(
            f"| `{route['method']}` | `{route['path']}` | `{route['aggregate']}` | "
            f"`{route['staging_state']}` | `{route['staging_owner']}` | "
            f"`{route['routing_action']}` |"
        )

    lines.extend(
        [
            "",
            "## Regeneration",
            "",
            "```sh",
            "python3 scripts/build_staging_route_cutover.py",
            "python3 scripts/build_staging_route_cutover.py --check",
            "```",
            "",
        ]
    )
    return "\n".join(lines)


def write_or_check(path: Path, content: str, check: bool) -> bool:
    current = path.read_text(encoding="utf-8") if path.exists() else None
    if check:
        if current != content:
            print(f"{path} is out of date.", file=sys.stderr)
            return False
        return True
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
    return True


def main() -> int:
    args = parse_args()
    manifest = build_manifest(
        load_json(args.matrix),
        load_json(args.gateway),
        parse_openapi_cutover_gates(args.openapi),
    )
    json_content = json.dumps(manifest, indent=2, sort_keys=True) + "\n"
    markdown_content = render_markdown(manifest)
    ok_json = write_or_check(args.output_json, json_content, args.check)
    ok_markdown = write_or_check(args.output_markdown, markdown_content, args.check)
    return 0 if ok_json and ok_markdown else 1


if __name__ == "__main__":
    raise SystemExit(main())
