#!/usr/bin/env python3
"""Check local Markdown links and fragments in the repository."""

from __future__ import annotations

import html
import re
import sys
from pathlib import Path
from urllib.parse import unquote, urlsplit


MARKDOWN_LINK = re.compile(r"!?\[[^]]*\]\(\s*(?:<([^>]+)>|([^\s)]+))")
REFERENCE_LINK = re.compile(r"(?m)^\s{0,3}\[[^]]+\]:\s*(?:<([^>]+)>|([^\s]+))")
HTML_LINK = re.compile(r"(?i)\b(?:href|src)\s*=\s*[\"']([^\"']+)[\"']")
HEADING = re.compile(r"^\s{0,3}(#{1,6})\s+(.+?)\s*#*\s*$")
HTML_ID = re.compile(r"(?i)\b(?:id|name)\s*=\s*[\"']([^\"']+)[\"']")


def markdown_files(root: Path) -> list[Path]:
    ignored = {".git", ".venv", "node_modules"}
    return sorted(
        path
        for path in root.rglob("*.md")
        if not ignored.intersection(path.relative_to(root).parts)
    )


def destinations(source: str) -> list[str]:
    found = []
    for pattern in (MARKDOWN_LINK, REFERENCE_LINK):
        found.extend(match.group(1) or match.group(2) for match in pattern.finditer(source))
    found.extend(match.group(1) for match in HTML_LINK.finditer(source))
    return found


def heading_anchors(source: str) -> set[str]:
    anchors = set()
    counts: dict[str, int] = {}
    for line in source.splitlines():
        match = HEADING.match(line)
        if match:
            title = re.sub(r"`([^`]*)`", r"\1", match.group(2))
            title = re.sub(r"\[([^]]+)\]\([^)]*\)", r"\1", title)
            slug = html.unescape(title).lower()
            slug = re.sub(r"[^\w -]", "", slug)
            slug = re.sub(r"\s+", "-", slug).strip("-")
            count = counts.get(slug, 0)
            counts[slug] = count + 1
            anchors.add(slug if count == 0 else f"{slug}-{count}")
        anchors.update(html.unescape(value) for value in HTML_ID.findall(line))
    return anchors


def check_link(markdown_file: Path, destination: str, root: Path) -> str | None:
    parsed = urlsplit(destination)
    if parsed.scheme or parsed.netloc:
        return None

    raw_path = unquote(parsed.path)
    target = (root / raw_path.lstrip("/")) if raw_path.startswith("/") else markdown_file.parent / raw_path
    target = target.resolve()
    if not target.exists() or (target != root and root not in target.parents):
        return f"{markdown_file.relative_to(root)}: broken link target {destination}"

    fragment = unquote(parsed.fragment)
    if fragment and target.is_file() and target.suffix.lower() in {".md", ".html", ".htm"}:
        anchors = heading_anchors(target.read_text(encoding="utf-8"))
        if fragment not in anchors:
            return f"{markdown_file.relative_to(root)}: broken link fragment {destination}"
    return None


def find_broken_links(root: Path) -> list[str]:
    errors = []
    for markdown_file in markdown_files(root):
        source = markdown_file.read_text(encoding="utf-8")
        for destination in destinations(source):
            error = check_link(markdown_file, destination, root)
            if error:
                errors.append(error)
    return errors


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    errors = find_broken_links(root)
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    print("All local Markdown links are valid.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
