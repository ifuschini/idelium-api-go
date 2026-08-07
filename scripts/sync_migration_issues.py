#!/usr/bin/env python3
"""Materialize the migration backlog as linked GitHub issues.

The script treats docs/migration/epics.md as the source of truth and creates a
three-level hierarchy: wave epic, domain track, and implementation ticket.
Existing issues are matched by their exact generated title, so repeated runs
are safe.
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Iterable


ROOT = Path(__file__).resolve().parents[1]
BACKLOG_PATH = ROOT / "docs" / "migration" / "epics.md"
MANIFEST_PATH = ROOT / "docs" / "migration" / "github-issues.json"
PROGRESS_PATH = ROOT / "docs" / "migration" / "progress.md"


@dataclass
class IssueRef:
    number: int
    title: str
    url: str


@dataclass
class Track:
    name: str
    tickets: list[str] = field(default_factory=list)
    issue: IssueRef | None = None


@dataclass
class Wave:
    number: int
    name: str
    goal: str
    exit_criteria: list[str]
    tracks: list[Track]
    tickets: list[str]
    issue: IssueRef | None = None


STOP_WORDS = {
    "a", "add", "and", "api", "for", "from", "in", "of", "or", "the",
    "to", "with", "reads", "writes", "migrate", "migration", "endpoints",
    "endpoint", "logic", "workflows", "tests", "test",
}


TRACK_HINTS: dict[str, tuple[str, ...]] = {
    "route and consumer inventory": ("route", "consumer", "inventory", "map ", "classify"),
    "compatibility contract backlog": ("contract", "undocumented consumer", "compatibility"),
    "golden fixture strategy": ("fixture", "golden", "sanitize"),
    "rollout, ownership, and rollback governance": ("ownership", "adr", "release", "rollback", "approval"),
    "runtime lifecycle": ("process", "shutdown", "worker", "migrate process", "lifecycle"),
    "configuration and secret loading": ("configuration", "secret-file", "secret loading", "database configuration"),
    "http safety and observability": ("health", "readiness", "logging", "correlation", "panic", "headers", "observability"),
    "mysql connectivity": ("mysql", "database", "schema"),
    "ci and container build": ("ci ", "ci gate", "dockerfile", "image", "sbom", "vulnerability", "makefile", "license", "repository directives"),
    "openapi generation": ("openapi", "generated server", "drift"),
    "laravel-go golden comparison": ("golden", "comparator", "normalize", "performance baseline", "fixture redaction"),
    "tenant isolation test library": ("tenant-isolation", "cross-tenant"),
    "web and cli smoke targeting": ("web smoke", "cli smoke"),
    "side-effect comparison": ("side-effect", "side effect", "mutation"),
    "operational metadata": ("platform type", "platform status", "operational"),
    "platform catalog reads": ("location", "brand", "model", "operating-system", "os-version", "browser", "catalog"),
    "safe lookup reads": ("differential", "lookup"),
    "shadow traffic and route switching": ("gateway", "route ownership", "shadow"),
    "api-key authentication": ("api-key", "api key", "authentication"),
    "test cycle graph reads": ("test-cycle", "test cycle", "graph equivalence", "complete remote cli execution"),
    "test, step, plugin, and environment reads": ("test read", "step read", "plugin", "environment"),
    "missing-reference diagnostics": ("missing-reference", "missing reference", "diagnostic"),
    "cross-tenant denial": ("cross-tenant", "tenant-scoped"),
    "performed cycle lifecycle": ("performed-cycle", "performed cycle"),
    "performed test lifecycle": ("performed-test", "performed test"),
    "performed step lifecycle": ("performed-step", "performed step"),
    "runtime metadata snapshots": ("snapshot", "platform", "browser", "operating system", "device"),
    "postman execution detail": ("postman", "request-level", "payload"),
    "failure finalization": ("terminal state", "failed", "interrupted", "idempotency", "retry"),
    "projects": ("project",),
    "environments": ("environment",),
    "plugins": ("plugin",),
    "steps, json, wizard, and dsl": ("step", "json", "wizard", "dsl"),
    "tests and step membership": ("test reads", "step membership"),
    "test cycles and ordering": ("test-cycle", "test cycle", "test ordering"),
    "imports and launcher setup": ("import", "launcher", "end-to-end web authoring"),
    "performed result exploration": ("performed", "exploration"),
    "exports and downloads": ("export", "download"),
    "artifact lifecycle": ("artifact", "retention", "archive", "restore", "legal hold", "deletion"),
    "grid query snapshots": ("grid", "bulk job"),
    "integrations and deliveries": ("integration", "delivery", "secret rotation"),
    "audit and asset review workflows": ("audit", "asset impact", "asset version", "drain laravel jobs"),
    "schedules and matrices": ("schedule", "matrix"),
    "worker claims and leases": ("claim", "row-lock", "lease"),
    "heartbeats and cancellation": ("heartbeat", "cancellation"),
    "run tokens": ("run-token", "run token"),
    "agents": ("agent registration", "agent status"),
    "runner-only endpoints": ("runner-only", "runner only", "clock-skew", "failure-injection", "load"),
    "login, logout, sessions, cookies, csrf": ("login", "logout", "session", "csrf", "browser-auth"),
    "current user, menu, customer switching": ("current-user", "current user", "capability", "customer switching"),
    "accounts, roles, profiles, password policy": ("account", "role", "profile", "password policy"),
    "customer administration and legacy api-key lifecycle": ("customer administration", "legacy api-key"),
    "service accounts and credentials": ("service account", "scoped credential"),
    "mfa, sso, oidc, scim, workload identity, and break-glass controls": ("mfa", "step-up", "sso", "oidc", "scim", "workload", "break-glass", "auth bridge"),
    "go migration baseline": ("baseline migration", "bridge command", "schema"),
    "empty install and upgrade verification": ("empty install", "upgrade"),
    "route ownership cutover": ("route ownership", "staging"),
    "laravel queue drain and write freeze": ("freeze", "queue", "maintenance"),
    "docker default image switch": ("docker default", "go api image"),
    "rollback rehearsal and operations documentation": ("rollback", "backup", "recovery", "operations documentation", "release"),
}


def run_gh(args: list[str], *, input_text: str | None = None, retries: int = 4) -> str:
    command = ["gh", *args]
    for attempt in range(1, retries + 1):
        result = subprocess.run(
            command,
            cwd=ROOT,
            input=input_text,
            text=True,
            capture_output=True,
            check=False,
        )
        if result.returncode == 0:
            return result.stdout.strip()
        if attempt == retries:
            raise RuntimeError(
                f"Command failed after {retries} attempts: {' '.join(command)}\n"
                f"{result.stderr.strip()}"
            )
        time.sleep(min(2**attempt, 12))
    raise AssertionError("unreachable")


def parse_bullets(block: str, heading: str) -> list[str]:
    match = re.search(
        rf"\n{re.escape(heading)}:\n\n(?P<body>(?:- .+(?:\n  .+)*\n)+)",
        block,
    )
    if not match:
        return []
    items: list[str] = []
    current: list[str] = []
    for line in match.group("body").splitlines():
        if line.startswith("- "):
            if current:
                items.append(" ".join(current))
            current = [line[2:].strip()]
        elif line.startswith("  "):
            current.append(line.strip())
    if current:
        items.append(" ".join(current))
    return items


def parse_backlog(path: Path) -> list[Wave]:
    source = path.read_text(encoding="utf-8")
    headings = list(re.finditer(r"^## Wave (\d+) Epic: (.+)$", source, re.MULTILINE))
    waves: list[Wave] = []
    for index, heading in enumerate(headings):
        end = headings[index + 1].start() if index + 1 < len(headings) else len(source)
        block = source[heading.end():end]
        goal_match = re.search(r"\nGoal: (.+(?:\n(?!\n)[^\n]+)*)", block)
        goal = " ".join(goal_match.group(1).splitlines()) if goal_match else heading.group(2)
        tracks = [Track(name=value.rstrip(".")) for value in parse_bullets(block, "Tracks")]
        tickets = [value.rstrip(".") for value in parse_bullets(block, "Tickets")]
        exit_criteria = [value.rstrip(".") for value in parse_bullets(block, "Exit criteria")]
        wave = Wave(
            number=int(heading.group(1)),
            name=heading.group(2).strip(),
            goal=goal,
            exit_criteria=exit_criteria,
            tracks=tracks,
            tickets=tickets,
        )
        assign_tickets(wave)
        waves.append(wave)
    return waves


def tokens(value: str) -> set[str]:
    words = re.findall(r"[a-z0-9]+", value.lower())
    return {word for word in words if len(word) > 2 and word not in STOP_WORDS}


def track_score(track: Track, ticket: str) -> int:
    track_key = track.name.lower()
    ticket_key = ticket.lower()
    score = 4 * len(tokens(track_key) & tokens(ticket_key))
    for hint in TRACK_HINTS.get(track_key, ()):
        if hint in ticket_key:
            score += 8
    return score


def assign_tickets(wave: Wave) -> None:
    if not wave.tracks:
        return
    for ticket in wave.tickets:
        scored = [(track_score(track, ticket), -index, track) for index, track in enumerate(wave.tracks)]
        _, _, selected = max(scored, key=lambda item: (item[0], item[1]))
        selected.tickets.append(ticket)


def generated_title(kind: str, wave: Wave, name: str) -> str:
    return f"[{kind}][Wave {wave.number}] {name}"


def load_existing(repo: str) -> dict[str, IssueRef]:
    payload = run_gh([
        "issue", "list", "--repo", repo, "--state", "all", "--limit", "1000",
        "--json", "number,title,url",
    ])
    return {
        item["title"]: IssueRef(item["number"], item["title"], item["url"])
        for item in json.loads(payload or "[]")
    }


def ensure_labels(repo: str) -> None:
    labels = {
        "migration-epic": ("5319e7", "Top-level Laravel-to-Go migration wave"),
        "migration-track": ("8250df", "Domain track within a migration wave"),
        "migration-ticket": ("1d76db", "Reviewable Laravel-to-Go implementation ticket"),
        "compatibility": ("fbca04", "Preserve Laravel, Web, CLI, runner, and Docker behavior"),
        "tenant-isolation": ("b60205", "Requires explicit tenant ownership enforcement"),
    }
    for wave_number in range(11):
        labels[f"wave-{wave_number}"] = ("d4c5f9", f"Migration wave {wave_number}")
    for name, (color, description) in labels.items():
        run_gh([
            "label", "create", name, "--repo", repo, "--color", color,
            "--description", description, "--force",
        ])


def create_or_reuse(
    repo: str,
    existing: dict[str, IssueRef],
    title: str,
    body: str,
    labels: Iterable[str],
) -> IssueRef:
    if title in existing:
        issue = existing[title]
        print(f"REUSE #{issue.number}: {title}", flush=True)
        return issue
    with tempfile.NamedTemporaryFile("w", encoding="utf-8", suffix=".md") as body_file:
        body_file.write(body)
        body_file.flush()
        args = ["issue", "create", "--repo", repo, "--title", title, "--body-file", body_file.name]
        for label in labels:
            args.extend(["--label", label])
        url = run_gh(args)
    number = int(url.rstrip("/").rsplit("/", 1)[-1])
    issue = IssueRef(number=number, title=title, url=url)
    existing[title] = issue
    print(f"CREATE #{number}: {title}", flush=True)
    time.sleep(0.8)
    return issue


def update_body(repo: str, issue: IssueRef, body: str) -> None:
    with tempfile.NamedTemporaryFile("w", encoding="utf-8", suffix=".md") as body_file:
        body_file.write(body)
        body_file.flush()
        run_gh(["issue", "edit", str(issue.number), "--repo", repo, "--body-file", body_file.name])
    print(f"LINK   #{issue.number}: {issue.title}", flush=True)
    time.sleep(0.5)


def wave_stub_body(wave: Wave) -> str:
    return f"""## Goal

{wave.goal}

## Hierarchy

This issue is the parent migration epic for Wave {wave.number}. Domain tracks and implementation tickets are linked after materialization.

## Exit criteria

{bullet_list(wave.exit_criteria)}

## Governance

- Source of truth: [`docs/migration/epics.md`](../blob/main/docs/migration/epics.md)
- Strategy: [`MIGRATION_PLAN.md`](../blob/main/MIGRATION_PLAN.md)
- Every completed ticket must end with a dedicated commit.
- Route ownership and rollback must remain explicit throughout coexistence.
"""


def track_stub_body(wave: Wave, track: Track) -> str:
    assert wave.issue
    return f"""## Parent epic

- {markdown_issue(wave.issue)}

## Objective

Deliver the **{track.name}** domain track within Wave {wave.number}: {wave.name}.

## Scope

- Preserve the Laravel compatibility contract for the affected behavior.
- Keep route and mutation ownership explicit and reversible.
- Apply tenant isolation, input validation, redaction, and safe diagnostics where relevant.
- Complete the linked implementation tickets with dedicated commits.

## Acceptance criteria

- All linked tickets are complete and verified.
- OpenAPI and compatibility documentation reflect externally visible behavior.
- Relevant unit, integration, differential, and negative cross-tenant tests pass.
- Rollout and rollback implications are recorded.

Implementation tickets are linked after materialization.
"""


def ticket_body(wave: Wave, track: Track, ticket: str) -> str:
    assert wave.issue and track.issue
    return f"""## Parent hierarchy

- Epic: {markdown_issue(wave.issue)}
- Track: {markdown_issue(track.issue)}

## Objective

{ticket}.

## Scope

- Implement this change as a small, independently reviewable migration slice.
- Preserve current Laravel-facing status codes, response fields, authorization semantics, and side effects unless a versioned compatibility decision says otherwise.
- Enforce tenant ownership in the same query or transaction for every tenant-owned resource.
- Keep sensitive credentials, headers, cookies, session identifiers, and payload secrets redacted.

## Acceptance criteria

- The behavior is implemented and its public contract is documented in OpenAPI when HTTP-visible.
- Unit tests cover success, validation failures, and safe error diagnostics.
- Database-backed behavior includes integration tests and negative cross-tenant coverage where applicable.
- Laravel-Go differential coverage is added or an explicit reason is recorded when comparison is not applicable.
- `make verify` passes; database-backed work also passes MySQL integration and migration checks.
- Route ownership, deployment impact, and rollback steps are documented when traffic or writes move.
- The completed work is committed in a dedicated commit and this issue is closed with verification evidence.

## Compatibility and rollback

Laravel remains the fallback owner until the corresponding Wave {wave.number} exit criteria pass. Do not introduce dual writes. Any schema change must remain backward compatible during coexistence.

## References

- [`docs/migration/epics.md`](../blob/main/docs/migration/epics.md)
- [`MIGRATION_PLAN.md`](../blob/main/MIGRATION_PLAN.md)
"""


def wave_final_body(wave: Wave) -> str:
    tracks = "\n".join(f"- [ ] {markdown_issue(track.issue)}" for track in wave.tracks if track.issue)
    return f"""## Goal

{wave.goal}

## Domain tracks

{tracks}

## Exit criteria

{bullet_list(wave.exit_criteria)}

## Governance

- Source of truth: [`docs/migration/epics.md`](../blob/main/docs/migration/epics.md)
- Strategy: [`MIGRATION_PLAN.md`](../blob/main/MIGRATION_PLAN.md)
- Every completed ticket must end with a dedicated commit.
- All track issues must be closed before this epic is closed.
- Route ownership and rollback must remain explicit throughout coexistence.
"""


def track_final_body(wave: Wave, track: Track, ticket_refs: dict[str, IssueRef]) -> str:
    assert wave.issue
    ticket_lines = "\n".join(
        f"- [ ] {markdown_issue(ticket_refs[ticket])}" for ticket in track.tickets
    ) or "- No implementation tickets are currently assigned."
    return f"""## Parent epic

- {markdown_issue(wave.issue)}

## Objective

Deliver the **{track.name}** domain track within Wave {wave.number}: {wave.name}.

## Implementation tickets

{ticket_lines}

## Acceptance criteria

- Every linked ticket is complete and includes verification evidence.
- OpenAPI and compatibility documentation reflect externally visible behavior.
- Relevant unit, integration, differential, and negative cross-tenant tests pass.
- Route ownership, deployment impact, and rollback implications are recorded.
- This track is closed only after its ticket checklist is complete.
"""


def bullet_list(items: list[str]) -> str:
    return "\n".join(f"- {item}." for item in items) if items else "- Defined by the linked implementation tickets."


def markdown_issue(issue: IssueRef | None) -> str:
    if issue is None:
        return "Unmaterialized issue"
    return f"[#{issue.number} {issue.title}]({issue.url})"


def write_manifest(repo: str, waves: list[Wave], ticket_refs: dict[str, IssueRef]) -> None:
    payload = {
        "repository": repo,
        "source": "docs/migration/epics.md",
        "waves": [
            {
                "number": wave.number,
                "name": wave.name,
                "issue": vars(wave.issue) if wave.issue else None,
                "tracks": [
                    {
                        "name": track.name,
                        "issue": vars(track.issue) if track.issue else None,
                        "tickets": [vars(ticket_refs[ticket]) for ticket in track.tickets],
                    }
                    for track in wave.tracks
                ],
            }
            for wave in waves
        ],
    }
    MANIFEST_PATH.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")

    total_tracks = sum(len(wave.tracks) for wave in waves)
    total_tickets = sum(len(wave.tickets) for wave in waves)
    progress = f"""# Idelium API Go Migration Progress

This file records the current operational cursor for the Laravel-to-Go migration.
The detailed strategy remains in [`MIGRATION_PLAN.md`](../../MIGRATION_PLAN.md),
while [`epics.md`](epics.md) is the versioned backlog source.

## GitHub backlog

- Repository: https://github.com/{repo}
- Wave epics: {len(waves)}
- Domain tracks: {total_tracks}
- Implementation tickets: {total_tickets}
- Machine-readable mapping: [`github-issues.json`](github-issues.json)

## Current cursor

| Wave | GitHub epic | Status | Evidence |
| --- | --- | --- | --- |
"""
    for wave in waves:
        status = "In progress" if wave.number in {1, 3} else "Planned"
        evidence = (
            "`e4e5def feat: bootstrap Go API foundation`" if wave.number == 1 else
            "`d4b7f22 feat: add read-only platform catalogs`" if wave.number == 3 else
            "Backlog materialized"
        )
        progress += f"| Wave {wave.number} | [#{wave.issue.number}]({wave.issue.url}) | {status} | {evidence} |\n"
    progress += """

## Update policy

Update this cursor whenever a migration ticket is completed. Each completed
ticket must include verification evidence, a dedicated commit, and a GitHub
closure comment. Regenerate the mapping with:

```sh
python3 scripts/sync_migration_issues.py --repo ifuschini/idelium-api-go --apply
```
"""
    PROGRESS_PATH.write_text(progress, encoding="utf-8")


def materialize(repo: str, waves: list[Wave]) -> None:
    ensure_labels(repo)
    existing = load_existing(repo)
    ticket_refs: dict[str, IssueRef] = {}

    for wave in waves:
        title = generated_title("Epic", wave, wave.name)
        wave.issue = create_or_reuse(
            repo, existing, title, wave_stub_body(wave),
            ["migration-epic", f"wave-{wave.number}", "compatibility"],
        )

    for wave in waves:
        for track in wave.tracks:
            title = generated_title("Track", wave, track.name)
            track.issue = create_or_reuse(
                repo, existing, title, track_stub_body(wave, track),
                ["migration-track", f"wave-{wave.number}", "compatibility"],
            )

    for wave in waves:
        for track in wave.tracks:
            for ticket in track.tickets:
                title = generated_title("Ticket", wave, ticket)
                labels = ["migration-ticket", f"wave-{wave.number}", "compatibility"]
                tenant_terms = ("tenant", "customer", "account", "project", "credential", "auth")
                if any(term in ticket.lower() for term in tenant_terms):
                    labels.append("tenant-isolation")
                ticket_refs[ticket] = create_or_reuse(
                    repo, existing, title, ticket_body(wave, track, ticket), labels,
                )

    for wave in waves:
        for track in wave.tracks:
            update_body(repo, track.issue, track_final_body(wave, track, ticket_refs))
        update_body(repo, wave.issue, wave_final_body(wave))

    write_manifest(repo, waves, ticket_refs)
    print(
        f"DONE: {len(waves)} epics, "
        f"{sum(len(wave.tracks) for wave in waves)} tracks, "
        f"{sum(len(wave.tickets) for wave in waves)} tickets",
        flush=True,
    )


def print_plan(waves: list[Wave]) -> None:
    for wave in waves:
        print(f"Wave {wave.number}: {wave.name}")
        for track in wave.tracks:
            print(f"  Track: {track.name} ({len(track.tickets)} tickets)")
            for ticket in track.tickets:
                print(f"    - {ticket}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default="ifuschini/idelium-api-go", help="GitHub repository in owner/name form")
    parser.add_argument("--apply", action="store_true", help="Create and link GitHub issues")
    args = parser.parse_args()

    waves = parse_backlog(BACKLOG_PATH)
    if not waves:
        raise RuntimeError(f"No migration waves found in {BACKLOG_PATH}")
    if args.apply:
        materialize(args.repo, waves)
    else:
        print_plan(waves)
        print("\nDry run only. Pass --apply to create or update GitHub issues.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (RuntimeError, ValueError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        raise SystemExit(1)
