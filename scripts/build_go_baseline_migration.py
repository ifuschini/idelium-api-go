#!/usr/bin/env python3
"""Build the reviewed Go baseline migration manifest from Laravel migrations."""

from __future__ import annotations

import argparse
import hashlib
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


SCHEMA_VERSION = 1
BASELINE_ID = "go-baseline-2026-08-25"
BASELINE_DATE = "2026-08-25"
SOURCE_RUNTIME = "idelium-api Laravel migrations"
TARGET_RUNTIME = "idelium-api-go"


def sha256_text(text: str) -> str:
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def build_manifest(source_dir: Path) -> dict[str, Any]:
    if not source_dir.is_dir():
        raise SystemExit(f"Laravel migration directory does not exist: {source_dir}")

    migrations = []
    aggregate = hashlib.sha256()
    for path in sorted(source_dir.glob("*.php")):
        text = path.read_text(encoding="utf-8")
        digest = sha256_text(text)
        aggregate.update(path.name.encode("utf-8"))
        aggregate.update(b"\0")
        aggregate.update(digest.encode("utf-8"))
        aggregate.update(b"\0")
        migrations.append(
            {
                "name": path.stem,
                "file": path.name,
                "sha256": digest,
                "bytes": len(text.encode("utf-8")),
            }
        )

    if not migrations:
        raise SystemExit(f"No Laravel migrations found in {source_dir}")

    return {
        "schema_version": SCHEMA_VERSION,
        "baseline_id": BASELINE_ID,
        "generated_on": BASELINE_DATE,
        "source_runtime": SOURCE_RUNTIME,
        "target_runtime": TARGET_RUNTIME,
        "source_directory": "../idelium-api/database/migrations",
        "migration_count": len(migrations),
        "aggregate_sha256": aggregate.hexdigest(),
        "review_status": "review-required-before-apply",
        "handover_policy": {
            "laravel_remains_schema_owner": True,
            "go_baseline_application_enabled": False,
            "dual_writes_allowed": False,
            "rollback": "Do not apply this baseline until bridge, empty-install, upgrade, and rollback rehearsal tickets pass.",
        },
        "redaction": "Migration source hashes and file sizes are recorded; no tenant data, credentials, or payload values are included.",
        "migrations": migrations,
    }


def render_markdown(manifest: dict[str, Any]) -> str:
    lines = [
        "# Go baseline migration manifest",
        "",
        "This document is generated from the reviewed Laravel migration source tree.",
        "It defines the immutable baseline that `idelium-api-go` will use during",
        "schema ownership handover. The baseline is intentionally not applied by",
        "this ticket; Laravel remains the schema owner until the bridge, empty",
        "install, upgrade, cutover, and rollback gates pass.",
        "",
        "## Summary",
        "",
        f"- Baseline ID: `{manifest['baseline_id']}`",
        f"- Generated on: `{manifest['generated_on']}`",
        f"- Source runtime: {manifest['source_runtime']}",
        f"- Target runtime: {manifest['target_runtime']}",
        f"- Migration count: {manifest['migration_count']}",
        f"- Aggregate SHA-256: `{manifest['aggregate_sha256']}`",
        f"- Review status: `{manifest['review_status']}`",
        "",
        "## Handover policy",
        "",
        "- Laravel remains the schema owner during coexistence.",
        "- Go baseline application is disabled until the Wave 10 bridge and",
        "  verification tickets are complete.",
        "- Dual writes are not allowed.",
        "- Rollback is a Git revert while this manifest is review-only; after",
        "  schema handover, rollback must use the documented last dual-runtime",
        "  release path.",
        "",
        "## Redaction",
        "",
        manifest["redaction"],
        "",
        "## Included Laravel migrations",
        "",
        "| # | Migration | SHA-256 | Size |",
        "| ---: | --- | --- | ---: |",
    ]
    for index, migration in enumerate(manifest["migrations"], start=1):
        lines.append(
            f"| {index} | `{migration['file']}` | `{migration['sha256']}` | {migration['bytes']} |"
        )
    lines.extend(
        [
            "",
            "## Regeneration",
            "",
            "```sh",
            "python3 scripts/build_go_baseline_migration.py \\",
            "  --source-dir ../idelium-api/database/migrations \\",
            "  --output-json docs/contracts/go-baseline-migration.json \\",
            "  --output-markdown docs/contracts/go-baseline-migration.md",
            "```",
        ]
    )
    return "\n".join(lines) + "\n"


def write_if_changed(path: Path, contents: str) -> None:
    if path.exists() and path.read_text(encoding="utf-8") == contents:
        return
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(contents, encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(description="Build Go baseline migration evidence.")
    parser.add_argument("--source-dir", type=Path, default=Path("../idelium-api/database/migrations"))
    parser.add_argument("--output-json", type=Path, default=Path("docs/contracts/go-baseline-migration.json"))
    parser.add_argument("--output-markdown", type=Path, default=Path("docs/contracts/go-baseline-migration.md"))
    parser.add_argument("--output-embedded-json", type=Path, default=Path("internal/migrations/baseline_manifest.json"))
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()

    if args.check:
        if not args.source_dir.is_dir():
            if not args.output_json.exists():
                raise SystemExit(f"Go baseline migration manifest is missing: {args.output_json}")
            manifest = json.loads(args.output_json.read_text(encoding="utf-8"))
            json_contents = json.dumps(manifest, indent=2, sort_keys=True) + "\n"
            markdown_contents = render_markdown(manifest)
        else:
            manifest = build_manifest(args.source_dir)
            json_contents = json.dumps(manifest, indent=2, sort_keys=True) + "\n"
            markdown_contents = render_markdown(manifest)
        failures = []
        if not args.output_json.exists() or args.output_json.read_text(encoding="utf-8") != json_contents:
            failures.append(str(args.output_json))
        if not args.output_markdown.exists() or args.output_markdown.read_text(encoding="utf-8") != markdown_contents:
            failures.append(str(args.output_markdown))
        if (
            not args.output_embedded_json.exists()
            or args.output_embedded_json.read_text(encoding="utf-8") != json_contents
        ):
            failures.append(str(args.output_embedded_json))
        if failures:
            raise SystemExit("Go baseline migration artifacts are stale: " + ", ".join(failures))
        return

    manifest = build_manifest(args.source_dir)
    json_contents = json.dumps(manifest, indent=2, sort_keys=True) + "\n"
    markdown_contents = render_markdown(manifest)
    write_if_changed(args.output_json, json_contents)
    write_if_changed(args.output_markdown, markdown_contents)
    write_if_changed(args.output_embedded_json, json_contents)
    print(
        f"Created Go baseline {manifest['baseline_id']} with "
        f"{manifest['migration_count']} Laravel migrations at "
        f"{datetime.now(timezone.utc).isoformat()}."
    )


if __name__ == "__main__":
    main()
