#!/usr/bin/env python3
"""Export and classify the authoritative Laravel route list."""

from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
from collections import Counter
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable


@dataclass(frozen=True)
class Source:
    label: str
    revision: str


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Export Laravel routes with authentication and ownership metadata."
    )
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument("--input", type=Path, help="Laravel route:list JSON file.")
    source.add_argument(
        "--docker-container",
        help="Running Laravel container used to execute artisan route:list --json.",
    )
    parser.add_argument("--source-label", required=True)
    parser.add_argument("--source-revision", required=True)
    parser.add_argument("--output-json", type=Path, required=True)
    parser.add_argument("--output-markdown", type=Path, required=True)
    return parser.parse_args()


def load_routes(args: argparse.Namespace) -> list[dict[str, Any]]:
    if args.input:
        raw = args.input.read_text(encoding="utf-8")
    else:
        completed = subprocess.run(
            [
                "docker",
                "exec",
                args.docker_container,
                "php",
                "artisan",
                "route:list",
                "--json",
            ],
            check=True,
            capture_output=True,
            text=True,
        )
        raw = completed.stdout

    parsed = json.loads(raw)
    if not isinstance(parsed, list):
        raise ValueError("Laravel route export must be a JSON array.")
    return parsed


def authentication_mode(route: dict[str, Any]) -> str:
    middleware = route["middleware"]
    uri = route["uri"].lstrip("/")

    if any("AuthenticateIdeliumKey" in item for item in middleware):
        return "api-key"
    if any("Authenticate:sanctum" in item for item in middleware):
        return "browser-session"
    if uri.startswith("api/ideliumrunner/"):
        return "run-token"
    if uri == "api/oidc/token-exchange":
        return "workload-identity-exchange"
    if uri.startswith("api/sso/"):
        return "sso-bootstrap-or-callback"
    if uri in {"api/login", "api/sanctum/csrf-cookie"}:
        return "browser-auth-bootstrap"
    if uri.startswith("_ignition/"):
        return "development-only"
    return "public"


def current_owner(route: dict[str, Any]) -> str:
    action = route["action"]
    if action.startswith("App\\") or route.get("path"):
        return "laravel-application"
    if action.startswith("Spatie\\"):
        return "development-dependency"
    return "laravel-framework"


def trust_path(authentication: str) -> str:
    mapping = {
        "api-key": "api-key",
        "browser-auth-bootstrap": "browser-session",
        "browser-session": "browser-session",
        "development-only": "public-operational",
        "public": "public-operational",
        "run-token": "run-token",
        "sso-bootstrap-or-callback": "browser-session",
        "workload-identity-exchange": "internal-service",
    }
    try:
        return mapping[authentication]
    except KeyError as error:
        raise ValueError(
            f"Authentication mode has no canonical trust path: {authentication}."
        ) from error


def normalize_route(route: dict[str, Any]) -> dict[str, Any]:
    required = {"method", "uri", "action", "middleware"}
    missing = required.difference(route)
    if missing:
        raise ValueError(f"Route entry is missing fields: {', '.join(sorted(missing))}.")
    if not isinstance(route["middleware"], list):
        raise ValueError("Route middleware must be a JSON array.")

    authentication = authentication_mode(route)
    normalized = {
        "method": route["method"],
        "path": "/" + route["uri"].lstrip("/"),
        "name": route.get("name"),
        "controller": route["action"],
        "authentication_mode": authentication,
        "trust_path": trust_path(authentication),
        "tenant_context": any(
            "ResolveTenantContext" in item for item in route["middleware"]
        ),
        "current_owner": current_owner(route),
        "middleware": route["middleware"],
        "source": route.get("path"),
    }
    return normalized


def normalize_routes(routes: Iterable[dict[str, Any]]) -> list[dict[str, Any]]:
    normalized = [normalize_route(route) for route in routes]
    normalized.sort(key=lambda route: (route["path"], route["method"], route["name"] or ""))

    identities = [(route["method"], route["path"]) for route in normalized]
    duplicates = [identity for identity, count in Counter(identities).items() if count > 1]
    if duplicates:
        rendered = ", ".join(f"{method} {path}" for method, path in duplicates)
        raise ValueError(f"Duplicate method/path route identities: {rendered}.")
    return normalized


def build_document(routes: list[dict[str, Any]], source: Source) -> dict[str, Any]:
    payload = json.dumps(routes, sort_keys=True, separators=(",", ":")).encode()
    return {
        "schema_version": 1,
        "source": {"label": source.label, "revision": source.revision},
        "route_count": len(routes),
        "route_digest_sha256": hashlib.sha256(payload).hexdigest(),
        "routes": routes,
    }


def markdown(document: dict[str, Any]) -> str:
    routes = document["routes"]
    auth_counts = Counter(route["authentication_mode"] for route in routes)
    trust_path_counts = Counter(route["trust_path"] for route in routes)
    owner_counts = Counter(route["current_owner"] for route in routes)
    tenant_count = sum(route["tenant_context"] for route in routes)

    lines = [
        "# Laravel Route Inventory",
        "",
        "This file is generated by `scripts/export_laravel_routes.py`. Do not edit the",
        "route table manually. Regenerate it from Laravel's authoritative",
        "`php artisan route:list --json` output.",
        "",
        "## Provenance",
        "",
        f"- Source: `{document['source']['label']}`",
        f"- Source revision: `{document['source']['revision']}`",
        f"- Route count: **{document['route_count']}**",
        f"- Normalized route digest: `{document['route_digest_sha256']}`",
        "",
        "The inventory includes application, framework, and development-only routes so",
        "that no reachable Laravel route disappears from migration governance.",
        "",
        "## Classification summary",
        "",
        "### Authentication modes",
        "",
        "| Mode | Routes |",
        "| --- | ---: |",
    ]
    lines.extend(f"| `{mode}` | {count} |" for mode, count in sorted(auth_counts.items()))
    lines.extend(
        [
            "",
            "### Canonical trust paths",
            "",
            "Every route is assigned to exactly one migration trust path. Browser",
            "authentication bootstrap and SSO callbacks belong to the browser-session",
            "path even though they execute before a session exists. Workload identity",
            "exchange belongs to the internal-service path. Framework, development, and",
            "otherwise public routes remain visible as public-operational endpoints.",
            "",
            "| Trust path | Routes |",
            "| --- | ---: |",
        ]
    )
    lines.extend(
        f"| `{path}` | {count} |" for path, count in sorted(trust_path_counts.items())
    )
    lines.extend(
        [
            "",
            "### Current owners",
            "",
            "| Owner | Routes |",
            "| --- | ---: |",
        ]
    )
    lines.extend(f"| `{owner}` | {count} |" for owner, count in sorted(owner_counts.items()))
    lines.extend(
        [
            "",
            f"Routes with explicit Laravel tenant context: **{tenant_count}**.",
            "",
            "## Complete route inventory",
            "",
            "| Method | Path | Controller or action | Trust path | Authentication detail | Tenant context | Current owner |",
            "| --- | --- | --- | --- | --- | --- | --- |",
        ]
    )
    for route in routes:
        action = route["controller"].replace("|", "\\|")
        lines.append(
            f"| `{route['method']}` | `{route['path']}` | `{action}` | "
            f"`{route['trust_path']}` | `{route['authentication_mode']}` | "
            f"{'yes' if route['tenant_context'] else 'no'} | "
            f"`{route['current_owner']}` |"
        )
    lines.extend(
        [
            "",
            "## Regeneration",
            "",
            "From a running local Idelium stack:",
            "",
            "```sh",
            "python3 scripts/export_laravel_routes.py \\",
            "  --docker-container idelium-ideliumapi-1 \\",
            "  --source-label idelium/idelium-api \\",
            "  --source-revision <laravel-commit> \\",
            "  --output-json docs/contracts/laravel-routes.json \\",
            "  --output-markdown docs/contracts/laravel-routes.md",
            "```",
            "",
            "Review the diff before committing. A changed digest means the Laravel route",
            "surface changed and the compatibility backlog must be reconciled.",
            "",
        ]
    )
    return "\n".join(lines)


def main() -> int:
    args = parse_args()
    routes = normalize_routes(load_routes(args))
    document = build_document(routes, Source(args.source_label, args.source_revision))
    args.output_json.parent.mkdir(parents=True, exist_ok=True)
    args.output_markdown.parent.mkdir(parents=True, exist_ok=True)
    args.output_json.write_text(json.dumps(document, indent=2) + "\n", encoding="utf-8")
    args.output_markdown.write_text(markdown(document), encoding="utf-8")
    print(f"Exported {len(routes)} Laravel routes.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
