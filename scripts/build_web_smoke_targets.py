#!/usr/bin/env python3
"""Build the Web smoke-test target plan from route ownership contracts."""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
from collections import Counter
from pathlib import Path
from typing import Any


SCHEMA_VERSION = 1
CONSUMER_ID = "idelium-web"
OWNER_BASE_URL_ENV = {
    "laravel": "IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL",
    "go": "IDELIUM_WEB_SMOKE_GO_BASE_URL",
}
SAFE_READ_METHODS = {"GET", "HEAD", "GET|HEAD"}


class SmokeTargetError(ValueError):
    """Report structural smoke-target errors without runtime values."""


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--consumer-map", type=Path, default=Path("docs/contracts/consumer-route-map.json"))
    parser.add_argument("--ownership-matrix", type=Path, default=Path("docs/contracts/migration-ownership-matrix.json"))
    parser.add_argument("--output-json", type=Path, default=Path("docs/contracts/web-smoke-targets.json"))
    parser.add_argument("--output-md", type=Path, default=Path("docs/contracts/web-smoke-targets.md"))
    parser.add_argument("--check", action="store_true", help="Fail when generated files are not up to date.")
    return parser.parse_args()


def read_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def route_id(method: str, path: str) -> str:
    return f"{method} {path}"


def smoke_method(method: str) -> str:
    return "GET" if method == "GET|HEAD" else method


def has_consumer(route: dict[str, Any], consumer_id: str) -> bool:
    return any(consumer.get("id") == consumer_id for consumer in route.get("consumers", []))


def build_targets(consumer_map: dict[str, Any], ownership_matrix: dict[str, Any]) -> dict[str, Any]:
    ownership_by_route = {route["route_id"]: route for route in ownership_matrix.get("routes", [])}
    targets: list[dict[str, Any]] = []
    errors: list[str] = []

    for route in consumer_map.get("routes", []):
        if not has_consumer(route, CONSUMER_ID):
            continue
        method = route["method"]
        path = route["path"]
        identifier = route_id(method, path)
        ownership = ownership_by_route.get(identifier)
        if ownership is None:
            errors.append(f"{identifier}: missing ownership matrix route")
            continue
        owner = ownership.get("owner")
        if owner not in OWNER_BASE_URL_ENV:
            errors.append(f"{identifier}: unsupported route owner")
            continue
        operation_kind = ownership.get("operation_kind")
        read_only = method in SAFE_READ_METHODS and operation_kind == "read"
        targets.append(
            {
                "route_id": identifier,
                "method": method,
                "smoke_method": smoke_method(method),
                "path": path,
                "aggregate": ownership.get("aggregate"),
                "operation_kind": operation_kind,
                "owner": owner,
                "target_base_url_env": OWNER_BASE_URL_ENV[owner],
                "tenant_context": ownership.get("tenant_context"),
                "trust_path": route.get("trust_path"),
                "authentication_mode": route.get("authentication_mode"),
                "execution_mode": "safe-read" if read_only else "synthetic-mutation",
                "requires_synthetic_session": route.get("authentication_mode") == "browser-session",
            }
        )

    if errors:
        raise SmokeTargetError("\n".join(errors))

    targets.sort(key=lambda item: (item["owner"], item["path"], item["method"]))
    owner_counts = Counter(target["owner"] for target in targets)
    mode_counts = Counter(target["execution_mode"] for target in targets)
    return {
        "schema_version": SCHEMA_VERSION,
        "consumer": CONSUMER_ID,
        "route_inventory_digest_sha256": consumer_map.get("route_inventory_digest_sha256"),
        "ownership_matrix_digest_sha256": sha256_bytes(
            json.dumps(ownership_matrix, sort_keys=True, separators=(",", ":")).encode("utf-8")
        ),
        "target_policy": {
            "route_owner_is_authoritative": True,
            "fail_closed_on_unknown_owner": True,
            "secrets_in_plan_allowed": False,
            "base_url_env_by_owner": OWNER_BASE_URL_ENV,
            "laravel_remains_fallback_owner": True,
        },
        "summary": {
            "route_count": len(targets),
            "owner_counts": dict(sorted(owner_counts.items())),
            "execution_mode_counts": dict(sorted(mode_counts.items())),
        },
        "routes": targets,
    }


def render_markdown(plan: dict[str, Any]) -> str:
    lines = [
        "# Web Smoke Target Plan",
        "",
        "This generated plan defines how Idelium Web smoke tests choose Laravel",
        "or Go for each route consumed by the Web console during the strangler",
        "migration. The route owner recorded in the migration ownership matrix is",
        "authoritative; unknown owners fail closed instead of falling back silently.",
        "",
        "## Execution policy",
        "",
        f"- Consumer: `{plan['consumer']}`",
        "- Laravel base URL environment variable: `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL`",
        "- Go base URL environment variable: `IDELIUM_WEB_SMOKE_GO_BASE_URL`",
        "- Smoke plans must not contain credentials, cookies, CSRF tokens, session IDs, or payload secrets.",
        "- Browser-session routes require a synthetic test session created outside this generated plan.",
        "- Mutation routes must use isolated synthetic tenants and reversible fixture data.",
        "",
        "## Summary",
        "",
        "| Metric | Value |",
        "| --- | --- |",
        f"| Routes | {plan['summary']['route_count']} |",
        f"| Owners | {', '.join(f'{owner}: {count}' for owner, count in plan['summary']['owner_counts'].items())} |",
        f"| Execution modes | {', '.join(f'{mode}: {count}' for mode, count in plan['summary']['execution_mode_counts'].items())} |",
        "",
        "## Routes",
        "",
        "| Method | Path | Owner | Target env | Mode | Tenant | Aggregate |",
        "| --- | --- | --- | --- | --- | --- | --- |",
    ]
    for route in plan["routes"]:
        lines.append(
            f"| `{route['smoke_method']}` | `{route['path']}` | `{route['owner']}` | "
            f"`{route['target_base_url_env']}` | `{route['execution_mode']}` | "
            f"{'yes' if route['tenant_context'] else 'no'} | `{route['aggregate']}` |"
        )
    lines.extend(
        [
            "",
            "## Compatibility and rollback",
            "",
            "This plan does not move traffic by itself. It gives Web smoke tests a",
            "deterministic target for each route while Laravel remains the fallback owner",
            "during coexistence. Rollback is a normal revert of the generated plan or a",
            "route-owner change in the ownership matrix before smoke execution.",
            "",
        ]
    )
    return "\n".join(lines)


def write_or_check(path: Path, content: str, check: bool) -> list[str]:
    if check:
        if not path.exists() or path.read_text(encoding="utf-8") != content:
            return [f"{path}: generated content is out of date"]
        return []
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
    return []


def main() -> int:
    args = parse_args()
    try:
        plan = build_targets(read_json(args.consumer_map), read_json(args.ownership_matrix))
    except SmokeTargetError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    json_content = json.dumps(plan, indent=2, sort_keys=True) + "\n"
    md_content = render_markdown(plan)
    errors = []
    errors.extend(write_or_check(args.output_json, json_content, args.check))
    errors.extend(write_or_check(args.output_md, md_content, args.check))
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
