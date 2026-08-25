#!/usr/bin/env python3
"""Run opt-in Idelium CLI remote smoke checks against the route owner plan."""

from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any


DEFAULT_PLAN = Path("docs/contracts/cli-smoke-targets.json")
API_KEY_ENV = "IDELIUM_CLI_SMOKE_API_KEY"
LEGACY_API_KEY_ENV = "IDELIUM_CLI_SMOKE_IDELIUM_KEY"
PATH_PARAM_ENV = {
    "idEnvironment": "IDELIUM_CLI_SMOKE_ID_ENVIRONMENT",
    "idPlugin": "IDELIUM_CLI_SMOKE_ID_PLUGIN",
    "idProject": "IDELIUM_CLI_SMOKE_ID_PROJECT",
    "idStep": "IDELIUM_CLI_SMOKE_ID_STEP",
    "idTest": "IDELIUM_CLI_SMOKE_ID_TEST",
    "idTestCycle": "IDELIUM_CLI_SMOKE_ID_TEST_CYCLE",
}
SUCCESS_STATUS_BY_MODE = {
    "configuration-read": {200},
}


class CliSmokeError(RuntimeError):
    """Report CLI smoke failures without exposing credentials."""


@dataclass(frozen=True)
class SmokeResult:
    route_id: str
    status: int
    url: str


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--plan", type=Path, default=DEFAULT_PLAN)
    parser.add_argument("--owner", default="go", choices=["go", "laravel"])
    parser.add_argument("--mode", default="configuration-read")
    parser.add_argument("--timeout", type=float, default=10.0)
    parser.add_argument("--dry-run", action="store_true", help="Print planned requests without calling the remote API.")
    return parser.parse_args()


def load_plan(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def select_routes(plan: dict[str, Any], owner: str, mode: str) -> list[dict[str, Any]]:
    routes = [
        route
        for route in plan.get("routes", [])
        if route.get("owner") == owner and route.get("execution_mode") == mode
    ]
    if not routes:
        raise CliSmokeError(f"No CLI smoke routes match owner={owner!r} and mode={mode!r}.")
    return routes


def required_env_for_routes(routes: list[dict[str, Any]]) -> list[str]:
    required = {API_KEY_ENV}
    required.update(route["target_base_url_env"] for route in routes)
    for route in routes:
        for parameter in PATH_PARAM_ENV:
            if "{" + parameter + "}" in route["path"]:
                required.add(PATH_PARAM_ENV[parameter])
    return sorted(required)


def read_api_key(env: dict[str, str]) -> str | None:
    return env.get(API_KEY_ENV) or env.get(LEGACY_API_KEY_ENV)


def validate_environment(routes: list[dict[str, Any]], env: dict[str, str]) -> None:
    missing = [name for name in required_env_for_routes(routes) if name not in env]
    if API_KEY_ENV in missing and LEGACY_API_KEY_ENV in env:
        missing.remove(API_KEY_ENV)
    if missing:
        raise CliSmokeError("Missing required environment variables: " + ", ".join(missing))
    if not read_api_key(env):
        raise CliSmokeError(f"Missing required environment variable: {API_KEY_ENV}")


def render_path(path: str, env: dict[str, str]) -> str:
    rendered = path
    for parameter, variable in PATH_PARAM_ENV.items():
        rendered = rendered.replace("{" + parameter + "}", urllib.parse.quote(env[variable], safe=""))
    return rendered


def target_url(route: dict[str, Any], env: dict[str, str]) -> str:
    base_url = env[route["target_base_url_env"]].rstrip("/")
    path = render_path(route["path"], env)
    return base_url + path


def expected_statuses(route: dict[str, Any]) -> set[int]:
    return SUCCESS_STATUS_BY_MODE.get(route["execution_mode"], {200, 201, 204})


def execute_route(route: dict[str, Any], env: dict[str, str], timeout: float) -> SmokeResult:
    url = target_url(route, env)
    request = urllib.request.Request(
        url,
        method=route["smoke_method"],
        headers={
            "Accept": "application/json",
            "Idelium-Key": read_api_key(env) or "",
            "X-Correlation-ID": "cli-smoke-go-configuration-read",
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            status = response.status
            response.read(1024)
    except urllib.error.HTTPError as exc:
        status = exc.code
    except urllib.error.URLError as exc:
        raise CliSmokeError(f"{route['route_id']}: remote API is not reachable: {exc.reason}") from exc

    if status not in expected_statuses(route):
        raise CliSmokeError(f"{route['route_id']}: expected {sorted(expected_statuses(route))}, got HTTP {status}")
    return SmokeResult(route_id=route["route_id"], status=status, url=url)


def run(plan: dict[str, Any], env: dict[str, str], owner: str, mode: str, timeout: float, dry_run: bool) -> list[SmokeResult]:
    routes = select_routes(plan, owner, mode)
    validate_environment(routes, env)

    results = []
    for route in routes:
        if dry_run:
            results.append(SmokeResult(route_id=route["route_id"], status=0, url=target_url(route, env)))
            continue
        results.append(execute_route(route, env, timeout))
    return results


def main() -> int:
    args = parse_args()
    try:
        results = run(load_plan(args.plan), os.environ, args.owner, args.mode, args.timeout, args.dry_run)
    except CliSmokeError as exc:
        print(f"CLI smoke failed: {exc}", file=sys.stderr)
        return 1

    for result in results:
        status = "DRY-RUN" if result.status == 0 else f"HTTP {result.status}"
        print(f"{status} {result.route_id} {result.url}")
    print(f"CLI smoke completed: {len(results)} route(s) checked.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
