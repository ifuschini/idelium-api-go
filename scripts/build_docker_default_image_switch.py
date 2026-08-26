#!/usr/bin/env python3
"""Build the Docker default image switch plan for the Go API cutover."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
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
        default=Path("docs/contracts/docker-default-image-switch.json"),
    )
    parser.add_argument(
        "--output-markdown",
        type=Path,
        default=Path("docs/contracts/docker-default-image-switch.md"),
    )
    parser.add_argument("--check", action="store_true")
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def build_plan(maintenance: dict[str, Any], route_cutover: dict[str, Any]) -> dict[str, Any]:
    blockers = []
    if maintenance["status"] != "ready":
        blockers.append(
            {
                "control": "laravel-readonly-maintenance",
                "reason": "The Laravel read-only maintenance gate is not ready.",
            }
        )
    if route_cutover["status"] != "ready":
        blockers.append(
            {
                "control": "staging-route-cutover",
                "reason": "The staging route cutover manifest still has Laravel blockers.",
                "count": route_cutover["summary"]["laravel_blocker_routes"],
            }
        )

    status = "ready" if not blockers else "blocked"
    return {
        "schema_version": 1,
        "switch_id": "docker-default-api-image-switch",
        "status": status,
        "production_enabled": False,
        "current_default": {
            "repository": "idelium-docker",
            "api_service": "Laravel API image remains the default while this plan is blocked.",
            "fallback_owner": "laravel",
        },
        "target_default": {
            "repository": "idelium-docker",
            "api_service": "idelium/api-go",
            "image_reference_policy": "pin-by-immutable-digest",
            "runtime_user": "65532:65532",
            "readiness_path": "/readyz",
            "liveness_path": "/healthz",
        },
        "preconditions": {
            "maintenance_status": maintenance["status"],
            "route_cutover_status": route_cutover["status"],
            "laravel_blocker_routes": route_cutover["summary"]["laravel_blocker_routes"],
            "go_owned_routes": route_cutover["summary"]["go_owned_routes"],
            "go_fail_closed_routes": route_cutover["summary"]["go_fail_closed_routes"],
        },
        "switch_controls": [
            "Build and publish the exact Go API runtime image by immutable digest.",
            "Update idelium-docker defaults only after maintenance and route-cutover gates are ready.",
            "Keep the Laravel API image reference available as rollback fallback.",
            "Require Go readiness before web and CLI traffic is admitted.",
            "Run Docker quickstart, Web smoke, CLI smoke, and route cutover checks after the switch.",
        ],
        "rollback": {
            "strategy": "Restore the previous Laravel API image default, route traffic back to Laravel, and keep Go image available for diagnostics.",
            "requires_database_restore": False,
            "dual_writes_allowed": False,
        },
        "blockers": blockers,
        "redaction": "The switch plan contains only image names, control states, and route counts. It does not contain credentials or runtime secrets.",
    }


def render_markdown(plan: dict[str, Any]) -> str:
    preconditions = plan["preconditions"]
    lines = [
        "# Docker Default API Image Switch",
        "",
        "This generated plan governs the switch from the Laravel API image to the",
        "Go API image as the default backend in `idelium-docker`. It deliberately",
        "keeps production disabled until the read-only maintenance and staging",
        "route cutover gates are ready.",
        "",
        "## Status",
        "",
        "| Field | Value |",
        "| --- | --- |",
        f"| Switch status | `{plan['status']}` |",
        f"| Production enabled | `{str(plan['production_enabled']).lower()}` |",
        f"| Target API image | `{plan['target_default']['api_service']}` |",
        f"| Image reference policy | `{plan['target_default']['image_reference_policy']}` |",
        f"| Runtime user | `{plan['target_default']['runtime_user']}` |",
        f"| Readiness path | `{plan['target_default']['readiness_path']}` |",
        f"| Maintenance status | `{preconditions['maintenance_status']}` |",
        f"| Route cutover status | `{preconditions['route_cutover_status']}` |",
        f"| Laravel blocker routes | {preconditions['laravel_blocker_routes']} |",
        "",
        "## Switch controls",
        "",
    ]
    for control in plan["switch_controls"]:
        lines.append(f"- {control}")
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
            "python3 scripts/build_docker_default_image_switch.py",
            "python3 scripts/build_docker_default_image_switch.py --check",
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
    plan = build_plan(load_json(args.maintenance), load_json(args.route_cutover))
    json_content = json.dumps(plan, indent=2, sort_keys=True) + "\n"
    markdown_content = render_markdown(plan)
    ok_json = write_or_check(args.output_json, json_content, args.check)
    ok_markdown = write_or_check(args.output_markdown, markdown_content, args.check)
    return 0 if ok_json and ok_markdown else 1


if __name__ == "__main__":
    raise SystemExit(main())
