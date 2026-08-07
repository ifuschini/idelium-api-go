#!/usr/bin/env python3
"""Validate route-switch governance without inspecting production data."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


ZERO_TOLERANCE_THRESHOLDS = {
    "cross_tenant_access_events",
    "credential_exposure_events",
    "lost_or_duplicate_writes",
    "side_effect_mismatches",
    "consumer_smoke_failures",
}
REQUIRED_APPROVALS = {
    "read-route": {"api-maintainer", "operations-on-call"},
    "mutation-aggregate": {
        "api-maintainer",
        "database-owner",
        "operations-on-call",
        "security-reviewer",
    },
}
EXPECTED_TRAFFIC_STAGES = [0, 0, 5, 25, 50, 100]


def validate_policy(policy: Any) -> list[str]:
    errors: list[str] = []
    if not isinstance(policy, dict):
        return ["$: policy must be an object"]
    ownership = policy.get("ownership", {})
    if ownership.get("dual_writes_allowed") is not False:
        errors.append("$.ownership.dual_writes_allowed: must be false")
    if ownership.get("fallback_owner") != "laravel":
        errors.append("$.ownership.fallback_owner: must be laravel during coexistence")

    roles = set(policy.get("approver_roles", []))
    profiles = policy.get("approval_profiles", {})
    for name, required in REQUIRED_APPROVALS.items():
        profile = profiles.get(name, {})
        configured = set(profile.get("required_roles", []))
        if not required <= configured:
            errors.append(f"$.approval_profiles.{name}: missing required independent roles")
        if profile.get("minimum_approvals", 0) < len(required):
            errors.append(f"$.approval_profiles.{name}.minimum_approvals: is too low")
        if not configured <= roles:
            errors.append(f"$.approval_profiles.{name}: contains an undeclared role")

    stages = policy.get("stages", [])
    stage_ids = [stage.get("id") for stage in stages if isinstance(stage, dict)]
    if len(stage_ids) != len(set(stage_ids)):
        errors.append("$.stages: stage identifiers must be unique")
    traffic = [stage.get("traffic_percent") for stage in stages if isinstance(stage, dict)]
    if traffic != EXPECTED_TRAFFIC_STAGES:
        errors.append("$.stages: traffic must progress through 0, 0, 5, 25, 50, and 100 percent")
    if any(not isinstance(stage.get("minimum_observation_minutes"), int) or stage.get("minimum_observation_minutes") < 0 for stage in stages if isinstance(stage, dict)):
        errors.append("$.stages: observation windows must be non-negative integer minutes")

    evidence = set(policy.get("evidence", []))
    required_evidence = {
        item
        for stage in stages
        if isinstance(stage, dict)
        for item in stage.get("required_evidence", [])
    }
    if evidence != required_evidence:
        errors.append("$.evidence: every required evidence type must be assigned to a stage")

    thresholds = policy.get("stop_thresholds", {})
    for name in ZERO_TOLERANCE_THRESHOLDS:
        if thresholds.get(name) != 0:
            errors.append(f"$.stop_thresholds.{name}: must have zero tolerance")

    rollback = policy.get("rollback", {})
    if rollback.get("automatic_on_stop_threshold") is not True:
        errors.append("$.rollback.automatic_on_stop_threshold: must be true")
    if rollback.get("gateway_owner_after_rollback") != "laravel":
        errors.append("$.rollback.gateway_owner_after_rollback: must be laravel")
    if rollback.get("database_restore_required") is not False:
        errors.append("$.rollback.database_restore_required: must be false")
    if rollback.get("reverse_application_replay_allowed") is not False:
        errors.append("$.rollback.reverse_application_replay_allowed: must be false")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("policy", type=Path)
    args = parser.parse_args()
    try:
        policy = json.loads(args.policy.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError):
        print("Policy must contain valid UTF-8 JSON.", file=sys.stderr)
        return 1
    errors = validate_policy(policy)
    for error in errors:
        print(error, file=sys.stderr)
    if not errors:
        print(f"Validated {args.policy}")
    return 1 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main())
