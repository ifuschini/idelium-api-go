#!/usr/bin/env python3
"""Check that the reviewed Laravel schema baseline remains frozen."""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-dir", type=Path, default=Path("../idelium-api/database/migrations"))
    parser.add_argument("--baseline", type=Path, default=Path("docs/contracts/go-baseline-migration.json"))
    parser.add_argument("--output-json", type=Path, default=Path("docs/contracts/laravel-schema-freeze.json"))
    parser.add_argument("--output-markdown", type=Path, default=Path("docs/contracts/laravel-schema-freeze.md"))
    parser.add_argument("--check", action="store_true")
    return parser.parse_args()


def sha256_text(text: str) -> str:
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def current_migrations(source_dir: Path) -> tuple[str, list[dict[str, Any]]]:
    if not source_dir.is_dir():
        raise SystemExit(f"Laravel migration directory does not exist: {source_dir}")
    aggregate = hashlib.sha256()
    migrations = []
    for path in sorted(source_dir.glob("*.php")):
        text = path.read_text(encoding="utf-8")
        digest = sha256_text(text)
        aggregate.update(path.name.encode("utf-8"))
        aggregate.update(b"\0")
        aggregate.update(digest.encode("utf-8"))
        aggregate.update(b"\0")
        migrations.append(
            {
                "file": path.name,
                "name": path.stem,
                "sha256": digest,
                "bytes": len(text.encode("utf-8")),
            }
        )
    if not migrations:
        raise SystemExit(f"No Laravel migrations found in {source_dir}")
    return aggregate.hexdigest(), migrations


def build_report(source_dir: Path, baseline: dict[str, Any]) -> dict[str, Any]:
    current_digest, current = current_migrations(source_dir)
    expected_by_file = {item["file"]: item for item in baseline["migrations"]}
    current_by_file = {item["file"]: item for item in current}

    added = sorted(set(current_by_file) - set(expected_by_file))
    removed = sorted(set(expected_by_file) - set(current_by_file))
    changed = sorted(
        file_name
        for file_name in set(expected_by_file) & set(current_by_file)
        if expected_by_file[file_name]["sha256"] != current_by_file[file_name]["sha256"]
    )

    violations = []
    for file_name in added:
        violations.append({"file": file_name, "type": "added"})
    for file_name in removed:
        violations.append({"file": file_name, "type": "removed"})
    for file_name in changed:
        violations.append({"file": file_name, "type": "changed"})

    status = "frozen" if not violations and current_digest == baseline["aggregate_sha256"] else "changed"
    return {
        "schema_version": 1,
        "freeze_id": "laravel-schema-freeze-2026-08-26",
        "status": status,
        "baseline_id": baseline["baseline_id"],
        "source_directory": baseline["source_directory"],
        "expected": {
            "migration_count": baseline["migration_count"],
            "aggregate_sha256": baseline["aggregate_sha256"],
        },
        "current": {
            "migration_count": len(current),
            "aggregate_sha256": current_digest,
        },
        "policy": {
            "new_laravel_migrations_allowed": False,
            "laravel_migration_edits_allowed": False,
            "schema_owner": "laravel-until-handover",
            "go_baseline_application_enabled": False,
            "dual_writes_allowed": False,
            "exception_process": "Open a versioned schema-change issue and update the reviewed Go baseline after approval.",
        },
        "violations": violations,
        "redaction": "Only migration file names, hashes, sizes, and aggregate counts are recorded.",
    }


def render_markdown(report: dict[str, Any]) -> str:
    lines = [
        "# Laravel Schema Freeze",
        "",
        "This generated report freezes the Laravel migration tree after the reviewed",
        "Go baseline has been produced. It prevents accidental Laravel schema drift",
        "during the final handover wave.",
        "",
        "## Status",
        "",
        "| Field | Value |",
        "| --- | --- |",
        f"| Freeze status | `{report['status']}` |",
        f"| Baseline ID | `{report['baseline_id']}` |",
        f"| Expected migrations | {report['expected']['migration_count']} |",
        f"| Current migrations | {report['current']['migration_count']} |",
        f"| Expected aggregate SHA-256 | `{report['expected']['aggregate_sha256']}` |",
        f"| Current aggregate SHA-256 | `{report['current']['aggregate_sha256']}` |",
        "",
        "## Policy",
        "",
        "- New Laravel migrations are not allowed during schema handover.",
        "- Edits to reviewed Laravel migrations are not allowed.",
        "- Go baseline application remains disabled until the bridge, empty-install,",
        "  upgrade, route-cutover, and rollback rehearsal gates pass.",
        "- Dual writes remain prohibited.",
        "- Any exception must be handled as a reviewed, versioned schema-change",
        "  issue that updates the Go baseline deliberately.",
        "",
        "## Violations",
        "",
        "| File | Type |",
        "| --- | --- |",
    ]
    if report["violations"]:
        for violation in report["violations"]:
            lines.append(f"| `{violation['file']}` | `{violation['type']}` |")
    else:
        lines.append("| none | none |")
    lines.extend(
        [
            "",
            "## Regeneration",
            "",
            "```sh",
            "python3 scripts/check_laravel_schema_freeze.py",
            "python3 scripts/check_laravel_schema_freeze.py --check",
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
    if args.check and not args.source_dir.is_dir():
        if not args.output_json.exists():
            print(f"Laravel schema freeze report is missing: {args.output_json}", file=sys.stderr)
            return 1
        report = load_json(args.output_json)
    else:
        report = build_report(args.source_dir, load_json(args.baseline))
    json_content = json.dumps(report, indent=2, sort_keys=True) + "\n"
    markdown_content = render_markdown(report)
    ok_json = write_or_check(args.output_json, json_content, args.check)
    ok_markdown = write_or_check(args.output_markdown, markdown_content, args.check)
    if report["status"] != "frozen":
        print("Laravel schema freeze has violations.", file=sys.stderr)
        return 1
    return 0 if ok_json and ok_markdown else 1


if __name__ == "__main__":
    raise SystemExit(main())
