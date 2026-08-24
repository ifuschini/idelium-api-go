#!/usr/bin/env python3
"""Build the representative performance baseline report."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


CASES = [
    {
        "route_id": "GET|HEAD /api/admin/platforms/status",
        "scenario": "platform-status-read",
        "class": "read",
        "budget_ms": {"p50": 75, "p95": 200, "p99": 400},
        "payload_profile": "small-catalog",
    },
    {
        "route_id": "GET|HEAD /api/admin/platforms/types",
        "scenario": "platform-type-read",
        "class": "read",
        "budget_ms": {"p50": 75, "p95": 200, "p99": 400},
        "payload_profile": "small-catalog",
    },
    {
        "route_id": "GET|HEAD /api/ideliumcl/testcycle/{idTestCycle}",
        "scenario": "cli-test-cycle-graph-read",
        "class": "read",
        "budget_ms": {"p50": 150, "p95": 500, "p99": 900},
        "payload_profile": "medium-execution-graph",
    },
    {
        "route_id": "GET|HEAD /api/ideliumcl/step/{idStep}",
        "scenario": "cli-step-read",
        "class": "read",
        "budget_ms": {"p50": 125, "p95": 400, "p99": 800},
        "payload_profile": "dsl-or-json-step",
    },
    {
        "route_id": "GET|HEAD /api/admin/testsperfomed/{idTestPerformed}",
        "scenario": "web-test-result-detail-read",
        "class": "read",
        "budget_ms": {"p50": 175, "p95": 650, "p99": 1200},
        "payload_profile": "result-detail",
    },
    {
        "route_id": "POST /api/admin/tests",
        "scenario": "web-test-create-write",
        "class": "write",
        "budget_ms": {"p50": 225, "p95": 900, "p99": 1500},
        "payload_profile": "test-definition",
    },
    {
        "route_id": "POST /api/ideliumcl/step",
        "scenario": "cli-step-result-write",
        "class": "write",
        "budget_ms": {"p50": 200, "p95": 800, "p99": 1500},
        "payload_profile": "step-result",
    },
    {
        "route_id": "POST /api/admin/launchtest",
        "scenario": "web-launch-test-write",
        "class": "write",
        "budget_ms": {"p50": 300, "p95": 1200, "p99": 2000},
        "payload_profile": "launch-command",
    },
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Build performance baseline evidence.")
    parser.add_argument("--backlog", type=Path, required=True)
    parser.add_argument("--output-json", type=Path, required=True)
    parser.add_argument("--output-markdown", type=Path, required=True)
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def build_report(backlog: dict[str, Any]) -> dict[str, Any]:
    index = {item["id"]: item for item in backlog["items"]}
    missing = [item["route_id"] for item in CASES if item["route_id"] not in index]
    if missing:
        raise ValueError("Baseline references unknown route IDs: " + ", ".join(missing))

    cases = []
    for item in CASES:
        route = index[item["route_id"]]
        cases.append(
            {
                "scenario": item["scenario"],
                "class": item["class"],
                "routeId": item["route_id"],
                "method": route["method"],
                "path": route["path"],
                "trustPath": route["trust_path"],
                "authenticationMode": route["authentication_mode"],
                "tenantOwned": route["tenant_context"],
                "migrationWave": route["migration_wave"],
                "consumers": route["consumer_ids"],
                "payloadProfile": item["payload_profile"],
                "baselineStatus": "capture-required",
                "budgetMs": item["budget_ms"],
                "sampleSize": {
                    "minimumRequests": 100,
                    "warmupRequests": 10,
                    "runs": 3,
                },
                "failurePolicy": "Fail migration gate when p95 exceeds budget by more than 20% or error rate is non-zero.",
            }
        )

    return {
        "schemaVersion": 1,
        "routeInventoryDigestSha256": backlog["route_inventory_digest_sha256"],
        "measurementPolicy": {
            "environment": "isolated non-production stack with synthetic tenants",
            "clock": "monotonic client-side duration around the HTTP request",
            "redaction": "No request or response payload values are written to the baseline report.",
            "rollback": "Performance evidence changes no traffic ownership and rolls back by Git revert.",
        },
        "cases": cases,
    }


def markdown(report: dict[str, Any]) -> str:
    lines = [
        "# Performance Baseline Report",
        "",
        "This generated report identifies the representative read and write scenarios",
        "that must be measured before migrated Go routes can replace Laravel-owned",
        "behavior. The current baseline status is `capture-required` because this",
        "repository does not contain live Laravel timing captures.",
        "",
        "## Measurement Policy",
        "",
        f"- Environment: {report['measurementPolicy']['environment']}",
        f"- Clock: {report['measurementPolicy']['clock']}",
        f"- Redaction: {report['measurementPolicy']['redaction']}",
        f"- Rollback: {report['measurementPolicy']['rollback']}",
        "",
        "## Representative Cases",
        "",
        "| Class | Scenario | Route | Trust path | Consumers | p95 budget | Status |",
        "| --- | --- | --- | --- | --- | ---: | --- |",
    ]
    for item in report["cases"]:
        consumers = ", ".join(f"`{value}`" for value in item["consumers"]) or "-"
        lines.append(
            f"| `{item['class']}` | `{item['scenario']}` | `{item['routeId']}` | "
            f"`{item['trustPath']}` | {consumers} | {item['budgetMs']['p95']} ms | "
            f"`{item['baselineStatus']}` |"
        )
    lines.extend(
        [
            "",
            "## Gate Policy",
            "",
            "A route cannot move traffic to Go when its representative scenario exceeds",
            "the p95 budget by more than 20% or records a non-zero error rate. Mutation",
            "routes must also pass the side-effect comparator before performance evidence",
            "can be accepted.",
            "",
            "Regenerate this report with:",
            "",
            "```sh",
            "python3 scripts/build_performance_baseline.py \\",
            "  --backlog docs/contracts/compatibility-backlog.json \\",
            "  --output-json docs/contracts/performance-baseline.json \\",
            "  --output-markdown docs/contracts/performance-baseline.md",
            "```",
            "",
        ]
    )
    return "\n".join(lines)


def main() -> int:
    args = parse_args()
    report = build_report(load_json(args.backlog))
    args.output_json.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    args.output_markdown.write_text(markdown(report), encoding="utf-8")
    print(f"Created {len(report['cases'])} performance baseline cases.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
