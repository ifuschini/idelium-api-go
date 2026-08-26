#!/usr/bin/env python3
"""Build the route and aggregate ownership matrix for migration governance."""

from __future__ import annotations

import argparse
import json
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any


MUTATION_METHODS = {"DELETE", "PATCH", "POST", "PUT"}
OWNERS = {"laravel", "go"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--backlog", type=Path, required=True)
    parser.add_argument("--output-json", type=Path, required=True)
    parser.add_argument("--output-markdown", type=Path, required=True)
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def is_mutation(method: str) -> bool:
    return bool(set(method.split("|")) & MUTATION_METHODS)


def aggregate_for(item: dict[str, Any]) -> str:
    path = item["path"]
    mutation = is_mutation(item["method"])

    if path == "/" or path in {"/api/clear", "/api/csrf-cookie", "/api/sanctum/csrf-cookie"}:
        return "operations"
    if path.startswith("/api/admin/accounts"):
        return "accounts"
    if path.startswith("/api/admin/agents") or path.startswith("/api/ideliumcl/agents"):
        return "agent-registry"
    if path.startswith("/api/admin/apikey"):
        return "legacy-api-keys"
    if path.startswith("/api/admin/costumers") or path.startswith("/api/menu/header"):
        return "customers"
    if path.startswith("/api/admin/environments") or path.startswith("/api/ideliumcl/environment"):
        return "environments"
    if path.startswith("/api/admin/grid"):
        return "grid-jobs"
    if path.startswith("/api/admin/identity") or path.startswith("/api/oidc/") or path.startswith("/api/sso/"):
        return "enterprise-identity"
    if path == "/api/admin/importtest" or path.startswith("/api/admin/tests"):
        if "testsperfomed" in path:
            return "execution-results"
        return "tests"
    if path == "/api/admin/launchtest":
        return "test-launches"
    if path.startswith("/api/admin/platforms"):
        return "platform-catalog"
    if path.startswith("/api/admin/plugins") or path.startswith("/api/ideliumcl/plugin"):
        return "plugins"
    if path.startswith("/api/admin/profile") or path in {"/api/login", "/api/logout", "/api/user"}:
        return "browser-identity"
    if path.startswith("/api/admin/projects/") and "/asset-" in path:
        return "asset-versions"
    if path.startswith("/api/admin/projects/") and ("/integrations" in path or "/integration-deliveries" in path):
        return "integrations"
    if "/parallel-runs" in path or path.startswith("/api/ideliumrunner/"):
        return "parallel-runs"
    if path.startswith("/api/admin/projects/") and "/performed-test-cycles/" in path:
        return "artifacts"
    if path.startswith("/api/admin/projects"):
        return "projects"
    if path.startswith("/api/admin/result-exports"):
        return "result-exports"
    if path == "/api/admin/roles" or path == "/api/me/capabilities" or path == "/api/menu/sidebar":
        return "access-control"
    if path.startswith("/api/admin/service-accounts"):
        return "service-accounts"
    if "stepsperfomed" in path or "testcyclesperfomed" in path:
        return "execution-results"
    if path.startswith("/api/admin/steps"):
        return "steps"
    if path.startswith("/api/admin/testcycles"):
        return "test-cycles"
    if path == "/api/audit-events":
        return "audit-events"
    if path == "/api/ideliumcl/test" and mutation:
        return "cli-performed-tests"
    if path == "/api/ideliumcl/testcycle" and mutation:
        return "cli-performed-cycles"
    if path == "/api/ideliumcl/step" and mutation:
        return "cli-performed-steps"
    if path.startswith("/api/ideliumcl/step/"):
        return "steps"
    if path.startswith("/api/ideliumcl/test/"):
        return "tests"
    if path.startswith("/api/ideliumcl/testcycle/"):
        return "test-cycles"
    raise ValueError(f"No aggregate ownership rule matches {item['method']} {path}.")


def effective_owner(item: dict[str, Any]) -> str:
    rollout_status = item["rollout_status"]
    if rollout_status == "laravel-owned":
        return "laravel"
    if rollout_status == "go-owned":
        return "go"
    raise ValueError(
        f"Unsupported rollout status for {item['id']}; ownership must be explicit."
    )


def build_matrix(backlog: dict[str, Any]) -> dict[str, Any]:
    route_records = []
    aggregate_routes: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for item in backlog["items"]:
        record = {
            "route_id": item["id"],
            "method": item["method"],
            "path": item["path"],
            "aggregate": aggregate_for(item),
            "operation_kind": "mutation" if is_mutation(item["method"]) else "read",
            "owner": effective_owner(item),
            "rollout_status": item["rollout_status"],
            "tenant_context": item["tenant_context"],
            "migration_wave": item["migration_wave"],
        }
        route_records.append(record)
        aggregate_routes[record["aggregate"]].append(record)

    aggregates = []
    for name, routes in sorted(aggregate_routes.items()):
        mutations = [route for route in routes if route["operation_kind"] == "mutation"]
        mutation_owners = sorted({route["owner"] for route in mutations})
        if len(mutation_owners) > 1:
            raise ValueError(f"Aggregate {name} has multiple mutation owners.")
        if any(owner not in OWNERS for owner in mutation_owners):
            raise ValueError(f"Aggregate {name} has an unsupported mutation owner.")
        aggregates.append(
            {
                "aggregate": name,
                "mutation_owner": mutation_owners[0] if mutation_owners else None,
                "route_count": len(routes),
                "mutation_route_count": len(mutations),
                "tenant_scoped_route_count": sum(bool(route["tenant_context"]) for route in routes),
                "rollout_state": "laravel-primary",
                "dual_writes_allowed": False,
                "fallback_owner": "laravel",
            }
        )

    return {
        "schema_version": 1,
        "compatibility_backlog_digest": backlog["route_inventory_digest_sha256"],
        "ownership_policy": {
            "allowed_owners": sorted(OWNERS),
            "single_mutation_owner_required": True,
            "dual_writes_allowed": False,
            "default_fallback_owner": "laravel",
        },
        "summary": {
            "aggregate_count": len(aggregates),
            "route_count": len(route_records),
            "mutation_route_count": sum(route["operation_kind"] == "mutation" for route in route_records),
            "route_owner_counts": dict(sorted(Counter(route["owner"] for route in route_records).items())),
        },
        "aggregates": aggregates,
        "routes": route_records,
    }


def render_markdown(matrix: dict[str, Any]) -> str:
    lines = [
        "# Migration Ownership Matrix",
        "",
        "This generated matrix assigns every production-visible Laravel route to",
        "one aggregate and one effective owner. An aggregate may have no mutations,",
        "but an aggregate with mutations must have exactly one mutation owner.",
        "Application-level dual writes are prohibited.",
        "",
        "## Current summary",
        "",
        "| Measure | Value |",
        "| --- | ---: |",
        f"| Aggregates | {matrix['summary']['aggregate_count']} |",
        f"| Routes | {matrix['summary']['route_count']} |",
        f"| Mutation routes | {matrix['summary']['mutation_route_count']} |",
        "",
        "## Aggregate ownership",
        "",
        "| Aggregate | Mutation owner | Routes | Mutations | Tenant-scoped | State |",
        "| --- | --- | ---: | ---: | ---: | --- |",
    ]
    for aggregate in matrix["aggregates"]:
        lines.append(
            f"| `{aggregate['aggregate']}` | {aggregate['mutation_owner'] or 'none'} | "
            f"{aggregate['route_count']} | {aggregate['mutation_route_count']} | "
            f"{aggregate['tenant_scoped_route_count']} | {aggregate['rollout_state']} |"
        )
    lines.extend(
        [
            "",
            "## Ownership transition rule",
            "",
            "A route changes to `go` only after its compatibility, authorization, tenant",
            "isolation, side-effect, observability, and rollback evidence is approved.",
            "All mutation routes in the same transaction-owning aggregate move together,",
            "unless an ADR defines a smaller aggregate boundary. The gateway is the only",
            "switch: neither runtime may replicate application writes into the other.",
            "",
            "During rollback, route ownership returns to Laravel before the Go writer is",
            "disabled. Shared schema changes must remain backward compatible, so rollback",
            "does not require database restoration or reverse data replication.",
            "",
            "## Route assignments",
            "",
            "| Method | Path | Aggregate | Kind | Owner | Tenant | Wave |",
            "| --- | --- | --- | --- | --- | --- | ---: |",
        ]
    )
    for route in matrix["routes"]:
        lines.append(
            f"| `{route['method']}` | `{route['path']}` | `{route['aggregate']}` | "
            f"{route['operation_kind']} | {route['owner']} | "
            f"{'yes' if route['tenant_context'] else 'no'} | {route['migration_wave']} |"
        )
    lines.extend(
        [
            "",
            "## Deployment and rollback",
            "",
            "This baseline is governance-only: it moves no traffic, performs no writes,",
            "and changes no database schema. Laravel remains the effective and fallback",
            "owner for every recorded route. Deployment publishes the matrix; rollback is",
            "a Git revert. Differential HTTP testing is not applicable until a route is",
            "implemented in both runtimes.",
            "",
        ]
    )
    return "\n".join(lines)


def main() -> int:
    args = parse_args()
    matrix = build_matrix(load_json(args.backlog))
    args.output_json.parent.mkdir(parents=True, exist_ok=True)
    args.output_json.write_text(json.dumps(matrix, indent=2) + "\n", encoding="utf-8")
    args.output_markdown.write_text(render_markdown(matrix), encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
