#!/usr/bin/env python3
"""Build the final operations handover documentation gate."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--schema-freeze", type=Path, default=Path("docs/contracts/laravel-schema-freeze.json"))
    parser.add_argument("--route-cutover", type=Path, default=Path("docs/contracts/staging-route-cutover.json"))
    parser.add_argument("--maintenance", type=Path, default=Path("docs/contracts/laravel-readonly-maintenance.json"))
    parser.add_argument("--docker-switch", type=Path, default=Path("docs/contracts/docker-default-image-switch.json"))
    parser.add_argument("--rollback", type=Path, default=Path("docs/contracts/rollback-rehearsal.json"))
    parser.add_argument("--output-json", type=Path, default=Path("docs/contracts/operations-handover.json"))
    parser.add_argument("--output-markdown", type=Path, default=Path("docs/contracts/operations-handover.md"))
    parser.add_argument("--check", action="store_true")
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def build_handover(
    schema_freeze: dict[str, Any],
    route_cutover: dict[str, Any],
    maintenance: dict[str, Any],
    docker_switch: dict[str, Any],
    rollback: dict[str, Any],
) -> dict[str, Any]:
    gate_statuses = {
        "schema_freeze": schema_freeze["status"],
        "route_cutover": route_cutover["status"],
        "laravel_readonly_maintenance": maintenance["status"],
        "docker_default_image_switch": docker_switch["status"],
        "rollback_rehearsal": rollback["status"],
    }
    blockers = [
        {"gate": gate, "status": status}
        for gate, status in gate_statuses.items()
        if status != "ready" and not (gate == "schema_freeze" and status == "frozen")
    ]
    status = "ready" if not blockers else "blocked"
    return {
        "schema_version": 1,
        "handover_id": "final-operations-handover",
        "status": status,
        "production_enabled": False,
        "gate_statuses": gate_statuses,
        "backup": {
            "required_before_window": True,
            "scope": [
                "application database snapshot",
                "object-storage artifacts and retention metadata",
                "current Laravel API image digest",
                "candidate Go API image digest",
                "gateway route ownership configuration",
            ],
            "restore_test_required": True,
        },
        "recovery": {
            "database_restore_required_for_route_rollback": False,
            "route_switchback_owner": "laravel",
            "max_recovery_minutes": rollback["safety"]["max_recovery_minutes"],
            "reverse_application_replay_allowed": False,
        },
        "release": {
            "docker_default_target": docker_switch["target_default"]["api_service"],
            "image_reference_policy": docker_switch["target_default"]["image_reference_policy"],
            "readiness_path": docker_switch["target_default"]["readiness_path"],
            "smoke_required": ["docker-quickstart", "web-smoke", "cli-smoke", "route-cutover-check"],
        },
        "operations": {
            "maintenance_max_minutes": maintenance["timebox"]["max_duration_minutes"],
            "controls": [control["name"] for control in maintenance["controls"]],
            "rollback_steps": [step["name"] for step in rollback["steps"]],
        },
        "blockers": blockers,
        "redaction": "The handover pack records only gate states, control names, route counts, and image reference policies. It contains no credentials, cookies, authorization headers, or payload data.",
    }


def render_markdown(handover: dict[str, Any]) -> str:
    lines = [
        "# Final Operations Handover",
        "",
        "This generated pack summarizes backup, recovery, release, and operations",
        "requirements for the Laravel-to-Go API handover.",
        "",
        "## Status",
        "",
        "| Field | Value |",
        "| --- | --- |",
        f"| Handover status | `{handover['status']}` |",
        f"| Production enabled | `{str(handover['production_enabled']).lower()}` |",
        "",
        "## Gate statuses",
        "",
        "| Gate | Status |",
        "| --- | --- |",
    ]
    for gate, status in handover["gate_statuses"].items():
        lines.append(f"| `{gate}` | `{status}` |")
    lines.extend(
        [
            "",
            "## Backup scope",
            "",
        ]
    )
    for item in handover["backup"]["scope"]:
        lines.append(f"- {item}")
    lines.extend(
        [
            "",
            "## Recovery",
            "",
            f"- Route switchback owner: `{handover['recovery']['route_switchback_owner']}`",
            f"- Database restore required for route rollback: `{str(handover['recovery']['database_restore_required_for_route_rollback']).lower()}`",
            f"- Reverse application replay allowed: `{str(handover['recovery']['reverse_application_replay_allowed']).lower()}`",
            f"- Max recovery objective: {handover['recovery']['max_recovery_minutes']} minutes",
            "",
            "## Release",
            "",
            f"- Docker default target: `{handover['release']['docker_default_target']}`",
            f"- Image reference policy: `{handover['release']['image_reference_policy']}`",
            f"- Readiness path: `{handover['release']['readiness_path']}`",
            "",
            "Required smoke checks:",
        ]
    )
    for item in handover["release"]["smoke_required"]:
        lines.append(f"- `{item}`")
    lines.extend(
        [
            "",
            "## Operations",
            "",
            f"- Maintenance window cap: {handover['operations']['maintenance_max_minutes']} minutes",
            "- Maintenance controls:",
        ]
    )
    for item in handover["operations"]["controls"]:
        lines.append(f"  - `{item}`")
    lines.append("- Rollback steps:")
    for item in handover["operations"]["rollback_steps"]:
        lines.append(f"  - `{item}`")
    lines.extend(
        [
            "",
            "## Current blockers",
            "",
            "| Gate | Status |",
            "| --- | --- |",
        ]
    )
    if handover["blockers"]:
        for blocker in handover["blockers"]:
            lines.append(f"| `{blocker['gate']}` | `{blocker['status']}` |")
    else:
        lines.append("| none | ready |")
    lines.extend(
        [
            "",
            "## Regeneration",
            "",
            "```sh",
            "python3 scripts/build_operations_handover.py",
            "python3 scripts/build_operations_handover.py --check",
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
    handover = build_handover(
        load_json(args.schema_freeze),
        load_json(args.route_cutover),
        load_json(args.maintenance),
        load_json(args.docker_switch),
        load_json(args.rollback),
    )
    json_content = json.dumps(handover, indent=2, sort_keys=True) + "\n"
    markdown_content = render_markdown(handover)
    ok_json = write_or_check(args.output_json, json_content, args.check)
    ok_markdown = write_or_check(args.output_markdown, markdown_content, args.check)
    return 0 if ok_json and ok_markdown else 1


if __name__ == "__main__":
    raise SystemExit(main())
