#!/usr/bin/env python3
"""Build the route-level consumer map from versioned matching rules."""

from __future__ import annotations

import argparse
import fnmatch
import json
from collections import Counter
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Build the Idelium route consumer map.")
    parser.add_argument("--inventory", type=Path, required=True)
    parser.add_argument("--rules", type=Path, required=True)
    parser.add_argument("--output-json", type=Path, required=True)
    parser.add_argument("--output-markdown", type=Path, required=True)
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object.")
    return value


def build_map(inventory: dict[str, Any], rules: dict[str, Any]) -> dict[str, Any]:
    consumers = rules.get("consumers")
    if not isinstance(consumers, list) or not consumers:
        raise ValueError("Consumer rules must define at least one consumer.")

    consumer_ids = [consumer.get("id") for consumer in consumers]
    if any(not consumer_id for consumer_id in consumer_ids):
        raise ValueError("Every consumer must have a non-empty id.")
    if len(consumer_ids) != len(set(consumer_ids)):
        raise ValueError("Consumer ids must be unique.")

    match_counts: Counter[tuple[str, str]] = Counter()
    mapped_routes = []
    for route in inventory["routes"]:
        route_consumers = []
        for consumer in consumers:
            matched_patterns = []
            for pattern in consumer.get("path_patterns", []):
                if fnmatch.fnmatchcase(route["path"], pattern):
                    matched_patterns.append(pattern)
                    match_counts[(consumer["id"], pattern)] += 1
            if matched_patterns:
                route_consumers.append(
                    {
                        "id": consumer["id"],
                        "relationship": consumer["relationship"],
                        "evidence": consumer["evidence"],
                        "matched_patterns": matched_patterns,
                    }
                )
        mapped_routes.append(
            {
                "method": route["method"],
                "path": route["path"],
                "authentication_mode": route["authentication_mode"],
                "current_owner": route["current_owner"],
                "consumers": route_consumers,
            }
        )

    unmatched_rules = [
        f"{consumer['id']}:{pattern}"
        for consumer in consumers
        for pattern in consumer.get("path_patterns", [])
        if match_counts[(consumer["id"], pattern)] == 0
    ]
    if unmatched_rules:
        raise ValueError(
            "Consumer rules matched no Laravel route: " + ", ".join(unmatched_rules)
        )

    mapped_count = sum(bool(route["consumers"]) for route in mapped_routes)
    return {
        "schema_version": 1,
        "route_inventory_digest_sha256": inventory["route_digest_sha256"],
        "sources": rules["sources"],
        "route_count": len(mapped_routes),
        "mapped_route_count": mapped_count,
        "unmapped_route_count": len(mapped_routes) - mapped_count,
        "routes": mapped_routes,
        "unresolved_references": rules.get("unresolved_references", []),
    }


def markdown(document: dict[str, Any]) -> str:
    consumer_counts: Counter[str] = Counter()
    for route in document["routes"]:
        for consumer in route["consumers"]:
            consumer_counts[consumer["id"]] += 1

    lines = [
        "# Laravel Route Consumer Map",
        "",
        "This generated baseline maps current Idelium consumers to the Laravel routes",
        "they invoke directly or through documented workflows. A missing consumer means",
        "that the repository scan found no current usage; it does not authorize route",
        "removal without compatibility review.",
        "",
        "## Coverage summary",
        "",
        f"- Laravel routes: **{document['route_count']}**",
        f"- Routes with an identified consumer: **{document['mapped_route_count']}**",
        f"- Routes without an identified consumer: **{document['unmapped_route_count']}**",
        "",
        "| Consumer | Mapped routes | Source baseline |",
        "| --- | ---: | --- |",
    ]
    for consumer, count in sorted(consumer_counts.items()):
        lines.append(f"| `{consumer}` | {count} | `{document['sources'][consumer]}` |")

    lines.extend(
        [
            "",
            "## Route-level mapping",
            "",
            "| Method | Path | Authentication | Consumers |",
            "| --- | --- | --- | --- |",
        ]
    )
    for route in document["routes"]:
        consumers = ", ".join(
            f"`{entry['id']}` ({entry['relationship']})"
            for entry in route["consumers"]
        ) or "—"
        lines.append(
            f"| `{route['method']}` | `{route['path']}` | "
            f"`{route['authentication_mode']}` | {consumers} |"
        )

    lines.extend(
        [
            "",
            "## Consumer references without a registered Laravel route",
            "",
            "These references are migration gaps, not active Laravel contracts.",
            "",
            "| Consumer | Referenced path | Reason |",
            "| --- | --- | --- |",
        ]
    )
    for reference in document["unresolved_references"]:
        lines.append(
            f"| `{reference['consumer']}` | `{reference['path']}` | "
            f"{reference['reason']} |"
        )

    lines.extend(
        [
            "",
            "## Governance",
            "",
            "This documentation-only map moves no traffic and changes no schema. Rollback",
            "is a Git revert. Before a route moves to Go, its mapped consumers must have a",
            "versioned compatibility contract and differential tests. Unmapped routes require",
            "an explicit retain, deprecate, or remove decision; absence of observed usage is",
            "not sufficient evidence for deletion.",
            "",
            "Regenerate this map after updating either the Laravel inventory or the",
            "consumer rules:",
            "",
            "```sh",
            "python3 scripts/build_consumer_route_map.py \\",
            "  --inventory docs/contracts/laravel-routes.json \\",
            "  --rules docs/contracts/consumer-route-rules.json \\",
            "  --output-json docs/contracts/consumer-route-map.json \\",
            "  --output-markdown docs/contracts/consumer-route-map.md",
            "```",
            "",
        ]
    )
    return "\n".join(lines)


def main() -> int:
    args = parse_args()
    document = build_map(load_json(args.inventory), load_json(args.rules))
    args.output_json.write_text(json.dumps(document, indent=2) + "\n", encoding="utf-8")
    args.output_markdown.write_text(markdown(document), encoding="utf-8")
    print(
        f"Mapped {document['mapped_route_count']} of {document['route_count']} "
        "Laravel routes."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
