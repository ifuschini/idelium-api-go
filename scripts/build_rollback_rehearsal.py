#!/usr/bin/env python3
"""Build the rollback rehearsal plan for the last dual-runtime release."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--docker-switch",
        type=Path,
        default=Path("docs/contracts/docker-default-image-switch.json"),
    )
    parser.add_argument(
        "--maintenance",
        type=Path,
        default=Path("docs/contracts/laravel-readonly-maintenance.json"),
    )
    parser.add_argument(
        "--route-cutover",
        type=Path,
        default=Path("docs/contracts/staging-route-cutover.json"),
    )
    parser.add_argument(
        "--output-json",
        type=Path,
        default=Path("docs/contracts/rollback-rehearsal.json"),
    )
    parser.add_argument(
        "--output-markdown",
        type=Path,
        default=Path("docs/contracts/rollback-rehearsal.md"),
    )
    parser.add_argument("--check", action="store_true")
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def build_plan(
    docker_switch: dict[str, Any],
    maintenance: dict[str, Any],
    route_cutover: dict[str, Any],
) -> dict[str, Any]:
    blockers = []
    for control, report in [
        ("docker-default-image-switch", docker_switch),
        ("laravel-readonly-maintenance", maintenance),
        ("staging-route-cutover", route_cutover),
    ]:
        if report["status"] != "ready":
            blockers.append(
                {
                    "control": control,
                    "reason": f"{control} is not ready.",
                }
            )

    status = "ready" if not blockers else "blocked"
    return {
        "schema_version": 1,
        "rehearsal_id": "last-dual-runtime-rollback",
        "status": status,
        "production_enabled": False,
        "target": {
            "release": "last-dual-runtime-release",
            "gateway_owner_after_rollback": "laravel",
            "restore_laravel_api_image": True,
            "resume_laravel_workers": True,
            "resume_laravel_scheduler": True,
        },
        "preconditions": {
            "docker_switch_status": docker_switch["status"],
            "maintenance_status": maintenance["status"],
            "route_cutover_status": route_cutover["status"],
            "laravel_blocker_routes": route_cutover["summary"]["laravel_blocker_routes"],
        },
        "steps": [
            {
                "order": 1,
                "name": "freeze-forward-rollout",
                "description": "Stop further Go promotion and declare rollback ownership.",
            },
            {
                "order": 2,
                "name": "gateway-switchback",
                "description": "Switch route ownership back to Laravel at the gateway.",
            },
            {
                "order": 3,
                "name": "restore-laravel-api-image",
                "description": "Restore the last dual-runtime Laravel API image default.",
            },
            {
                "order": 4,
                "name": "resume-laravel-processing",
                "description": "Resume Laravel queue workers and scheduled jobs after switchback.",
            },
            {
                "order": 5,
                "name": "smoke-and-observe",
                "description": "Run Docker quickstart, Web smoke, CLI smoke, and route ownership checks.",
            },
            {
                "order": 6,
                "name": "record-rehearsal-evidence",
                "description": "Record command output, route owner, image digest, and observation window results.",
            },
        ],
        "success_criteria": [
            "Gateway owner after rollback is Laravel.",
            "Laravel API image is the default API image.",
            "Laravel workers and scheduled jobs are resumed.",
            "Go routes can be drained without reverse data replay.",
            "Docker quickstart, Web smoke, and CLI smoke checks pass.",
            "No database restore is required.",
        ],
        "safety": {
            "requires_database_restore": False,
            "reverse_application_replay_allowed": False,
            "dual_writes_allowed": False,
            "max_recovery_minutes": 30,
        },
        "blockers": blockers,
        "redaction": "The rehearsal plan records only control states, ordered actions, and aggregate route counts. It contains no credentials, cookies, authorization headers, or payload data.",
    }


def render_markdown(plan: dict[str, Any]) -> str:
    preconditions = plan["preconditions"]
    lines = [
        "# Rollback Rehearsal Plan",
        "",
        "This generated plan rehearses rollback to the last dual-runtime release.",
        "It is intentionally blocked until the Docker default switch, Laravel",
        "read-only maintenance, and staging route cutover gates are ready.",
        "",
        "## Status",
        "",
        "| Field | Value |",
        "| --- | --- |",
        f"| Rehearsal status | `{plan['status']}` |",
        f"| Production enabled | `{str(plan['production_enabled']).lower()}` |",
        f"| Target release | `{plan['target']['release']}` |",
        f"| Gateway owner after rollback | `{plan['target']['gateway_owner_after_rollback']}` |",
        f"| Docker switch status | `{preconditions['docker_switch_status']}` |",
        f"| Maintenance status | `{preconditions['maintenance_status']}` |",
        f"| Route cutover status | `{preconditions['route_cutover_status']}` |",
        f"| Laravel blocker routes | {preconditions['laravel_blocker_routes']} |",
        f"| Max recovery objective | {plan['safety']['max_recovery_minutes']} minutes |",
        "",
        "## Ordered rehearsal steps",
        "",
        "| # | Step | Description |",
        "| ---: | --- | --- |",
    ]
    for step in plan["steps"]:
        lines.append(f"| {step['order']} | `{step['name']}` | {step['description']} |")
    lines.extend(
        [
            "",
            "## Success criteria",
            "",
        ]
    )
    for item in plan["success_criteria"]:
        lines.append(f"- {item}")
    lines.extend(
        [
            "",
            "## Safety",
            "",
            f"- Requires database restore: `{str(plan['safety']['requires_database_restore']).lower()}`",
            f"- Reverse application replay allowed: `{str(plan['safety']['reverse_application_replay_allowed']).lower()}`",
            f"- Dual writes allowed: `{str(plan['safety']['dual_writes_allowed']).lower()}`",
            "",
            "## Current blockers",
            "",
            "| Control | Reason |",
            "| --- | --- |",
        ]
    )
    if plan["blockers"]:
        for blocker in plan["blockers"]:
            lines.append(f"| `{blocker['control']}` | {blocker['reason']} |")
    else:
        lines.append("| none | none |")
    lines.extend(
        [
            "",
            "## Regeneration",
            "",
            "```sh",
            "python3 scripts/build_rollback_rehearsal.py",
            "python3 scripts/build_rollback_rehearsal.py --check",
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
    plan = build_plan(
        load_json(args.docker_switch),
        load_json(args.maintenance),
        load_json(args.route_cutover),
    )
    json_content = json.dumps(plan, indent=2, sort_keys=True) + "\n"
    markdown_content = render_markdown(plan)
    ok_json = write_or_check(args.output_json, json_content, args.check)
    ok_markdown = write_or_check(args.output_markdown, markdown_content, args.check)
    return 0 if ok_json and ok_markdown else 1


if __name__ == "__main__":
    raise SystemExit(main())
