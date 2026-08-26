#!/usr/bin/env python3
"""Build the Laravel read-only maintenance gate for final handover."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--schema-freeze",
        type=Path,
        default=Path("docs/contracts/laravel-schema-freeze.json"),
    )
    parser.add_argument(
        "--route-cutover",
        type=Path,
        default=Path("docs/contracts/staging-route-cutover.json"),
    )
    parser.add_argument(
        "--output-json",
        type=Path,
        default=Path("docs/contracts/laravel-readonly-maintenance.json"),
    )
    parser.add_argument(
        "--output-markdown",
        type=Path,
        default=Path("docs/contracts/laravel-readonly-maintenance.md"),
    )
    parser.add_argument("--check", action="store_true")
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def build_plan(schema_freeze: dict[str, Any], route_cutover: dict[str, Any]) -> dict[str, Any]:
    blockers = []
    if schema_freeze["status"] != "frozen":
        blockers.append(
            {
                "control": "schema-freeze",
                "reason": "Laravel schema freeze is not frozen.",
            }
        )
    if route_cutover["summary"]["laravel_blocker_routes"] > 0:
        blockers.append(
            {
                "control": "route-cutover",
                "reason": "Routes without Go implementation or fail-closed gates remain on Laravel.",
                "count": route_cutover["summary"]["laravel_blocker_routes"],
            }
        )

    status = "ready" if not blockers else "blocked"
    return {
        "schema_version": 1,
        "maintenance_id": "laravel-readonly-maintenance",
        "status": status,
        "production_enabled": False,
        "timebox": {
            "max_duration_minutes": 60,
            "requires_approved_start": True,
            "requires_approved_end": True,
            "default_state": "not-scheduled",
        },
        "preconditions": {
            "schema_freeze_status": schema_freeze["status"],
            "route_cutover_status": route_cutover["status"],
            "laravel_blocker_routes": route_cutover["summary"]["laravel_blocker_routes"],
            "go_owned_routes": route_cutover["summary"]["go_owned_routes"],
            "go_fail_closed_routes": route_cutover["summary"]["go_fail_closed_routes"],
        },
        "controls": [
            {
                "name": "gateway-mutation-block",
                "owner": "gateway",
                "state": "planned",
                "description": "Block Laravel-owned mutation traffic at the gateway during the approved window.",
            },
            {
                "name": "queue-drain",
                "owner": "laravel-operations",
                "state": "planned",
                "description": "Drain Laravel queues before entering read-only and keep workers stopped during the window.",
            },
            {
                "name": "scheduled-job-pause",
                "owner": "laravel-operations",
                "state": "planned",
                "description": "Pause Laravel scheduled jobs that can mutate data until archival is complete or rollback starts.",
            },
            {
                "name": "go-route-verification",
                "owner": "idelium-api-go",
                "state": "planned",
                "description": "Verify Go-owned and Go fail-closed routes before read-only maintenance begins.",
            },
            {
                "name": "operator-broadcast",
                "owner": "release-management",
                "state": "planned",
                "description": "Announce the time-boxed maintenance window and expected read-only behavior.",
            },
        ],
        "exit_criteria": [
            "No Laravel schema freeze violations exist.",
            "No unresolved route cutover blockers exist.",
            "Laravel queues are drained and workers are stopped.",
            "Go-owned routes pass the staging smoke plan.",
            "Rollback owner and gateway switchback command are confirmed.",
        ],
        "rollback": {
            "strategy": "Remove the gateway mutation block, resume Laravel workers and scheduled jobs, and route traffic back to Laravel before retrying the handover.",
            "requires_database_restore": False,
            "dual_writes_allowed": False,
        },
        "blockers": blockers,
        "redaction": "The maintenance plan records only aggregate status and control names. It contains no credentials, cookies, authorization headers, or payload data.",
    }


def render_markdown(plan: dict[str, Any]) -> str:
    preconditions = plan["preconditions"]
    lines = [
        "# Laravel Read-only Maintenance Gate",
        "",
        "This generated gate controls the time-boxed Laravel read-only period used",
        "before archival. It does not move traffic by itself; it defines the",
        "preconditions and operational controls that must be true before operators",
        "enter the maintenance window.",
        "",
        "## Status",
        "",
        "| Field | Value |",
        "| --- | --- |",
        f"| Maintenance status | `{plan['status']}` |",
        f"| Production enabled | `{str(plan['production_enabled']).lower()}` |",
        f"| Default state | `{plan['timebox']['default_state']}` |",
        f"| Max duration | {plan['timebox']['max_duration_minutes']} minutes |",
        f"| Schema freeze | `{preconditions['schema_freeze_status']}` |",
        f"| Route cutover | `{preconditions['route_cutover_status']}` |",
        f"| Laravel blocker routes | {preconditions['laravel_blocker_routes']} |",
        f"| Go-owned routes | {preconditions['go_owned_routes']} |",
        f"| Go fail-closed routes | {preconditions['go_fail_closed_routes']} |",
        "",
        "## Controls",
        "",
        "| Control | Owner | State | Description |",
        "| --- | --- | --- | --- |",
    ]
    for control in plan["controls"]:
        lines.append(
            f"| `{control['name']}` | `{control['owner']}` | `{control['state']}` | "
            f"{control['description']} |"
        )
    lines.extend(
        [
            "",
            "## Exit criteria",
            "",
        ]
    )
    for item in plan["exit_criteria"]:
        lines.append(f"- {item}")
    lines.extend(
        [
            "",
            "## Current blockers",
            "",
            "| Control | Reason | Count |",
            "| --- | --- | ---: |",
        ]
    )
    if plan["blockers"]:
        for blocker in plan["blockers"]:
            lines.append(
                f"| `{blocker['control']}` | {blocker['reason']} | {blocker.get('count', 1)} |"
            )
    else:
        lines.append("| none | none | 0 |")
    lines.extend(
        [
            "",
            "## Rollback",
            "",
            plan["rollback"]["strategy"],
            "",
            f"- Requires database restore: `{str(plan['rollback']['requires_database_restore']).lower()}`",
            f"- Dual writes allowed: `{str(plan['rollback']['dual_writes_allowed']).lower()}`",
            "",
            "## Regeneration",
            "",
            "```sh",
            "python3 scripts/build_laravel_readonly_maintenance.py",
            "python3 scripts/build_laravel_readonly_maintenance.py --check",
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
    plan = build_plan(load_json(args.schema_freeze), load_json(args.route_cutover))
    json_content = json.dumps(plan, indent=2, sort_keys=True) + "\n"
    markdown_content = render_markdown(plan)
    ok_json = write_or_check(args.output_json, json_content, args.check)
    ok_markdown = write_or_check(args.output_markdown, markdown_content, args.check)
    return 0 if ok_json and ok_markdown else 1


if __name__ == "__main__":
    raise SystemExit(main())
