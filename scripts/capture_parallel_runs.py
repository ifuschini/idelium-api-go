#!/usr/bin/env python3
"""Capture Laravel parallel-run responses into sanitized golden fixtures."""

from __future__ import annotations

import argparse
import json
import os
import urllib.request
import urllib.error
import re
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

SENSITIVE = {"authorization", "cookie", "set-cookie", "token", "apikey", "password", "secret", "session"}
PLACEHOLDER = re.compile(r"\{\{([A-Za-z0-9_.-]+)\}\}")


def sanitize(value: Any) -> Any:
    if isinstance(value, dict):
        return {k: sanitize(v) for k, v in value.items() if not any(part in k.lower() for part in SENSITIVE)}
    if isinstance(value, list):
        return [sanitize(item) for item in value]
    return value


def resolve(value: Any, variables: dict[str, str]) -> Any:
    if isinstance(value, str):
        return PLACEHOLDER.sub(lambda match: variables.get(match.group(1), match.group(0)), value)
    if isinstance(value, dict):
        return {key: resolve(item, variables) for key, item in value.items()}
    if isinstance(value, list):
        return [resolve(item, variables) for item in value]
    return value


def extract(value: Any, path: str) -> str:
    current = value
    for part in path.split("."):
        if not isinstance(current, dict) or part not in current:
            raise ValueError(f"capture path not found: {path}")
        current = current[part]
    if not isinstance(current, str) or not current:
        raise ValueError(f"capture path is not a non-empty string: {path}")
    return current


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--plan", type=Path, required=True)
    parser.add_argument("--output-dir", type=Path, required=True)
    args = parser.parse_args()
    plan = json.loads(args.plan.read_text(encoding="utf-8"))
    base_url = os.environ.get(plan["baseUrlEnv"], "").rstrip("/")
    if not base_url:
        raise SystemExit(f"Missing required environment variable: {plan['baseUrlEnv']}")
    cookie = os.environ.get("CAPTURE_BROWSER_COOKIE", "")
    xsrf = os.environ.get("CAPTURE_XSRF_TOKEN", "")
    api_key = os.environ.get("CAPTURE_API_KEY", "")
    run_token = os.environ.get("CAPTURE_RUN_TOKEN", "")
    variables: dict[str, str] = {"runToken": run_token} if run_token else {}
    revision = os.environ.get("CAPTURE_LARAVEL_REVISION", "")
    if len(revision) != 40:
        raise SystemExit("CAPTURE_LARAVEL_REVISION must be an immutable 40-character Git revision")
    args.output_dir.mkdir(parents=True, exist_ok=True)
    for route in plan["routes"]:
        path = resolve(route["path"], variables)
        headers = {"Accept": "application/json", "X-Correlation-ID": "laravel-capture"}
        if route["authentication"] == "browser-session":
            if not cookie:
                raise SystemExit("Missing CAPTURE_BROWSER_COOKIE for browser-session route")
            headers["Cookie"] = cookie
            if xsrf:
                headers["X-XSRF-TOKEN"] = xsrf
            route_token = variables.get("runToken", run_token)
            if route.get("requiresRunToken"):
                if not route_token:
                    raise SystemExit("Missing CAPTURE_RUN_TOKEN for run-token protected browser route")
                headers["Idelium-Run-Token"] = route_token
            if route.get("requiresWorkerToken"):
                worker_token = variables.get("workerToken", "")
                if not worker_token:
                    raise SystemExit("Missing worker token from claim response")
                headers["Idelium-Worker-Token"] = worker_token
        elif route["authentication"] == "api-key":
            if not api_key:
                raise SystemExit("Missing CAPTURE_API_KEY for api-key route")
            headers["Idelium-Key"] = api_key
        elif route["authentication"] == "run-token":
            route_token = variables.get("runToken", run_token)
            if not route_token:
                raise SystemExit("Missing CAPTURE_RUN_TOKEN for run-token route")
            headers["Idelium-Run-Token"] = route_token
        if route.get("requiresRunToken") and "Idelium-Run-Token" not in headers:
            route_token = variables.get("runToken", run_token)
            if not route_token:
                raise SystemExit("Missing run token for protected route")
            headers["Idelium-Run-Token"] = route_token
        if route.get("requiresWorkerToken"):
            worker_token = variables.get("workerToken", "")
            if not worker_token:
                raise SystemExit("Missing worker token from claim response")
            headers["Idelium-Worker-Token"] = worker_token
        body = resolve(route.get("body"), variables)
        encoded_body = None
        if body is not None:
            encoded_body = json.dumps(body).encode("utf-8")
            headers["Content-Type"] = "application/json"
        request = urllib.request.Request(base_url + path, data=encoded_body, headers=headers, method=route["method"])
        try:
            with urllib.request.urlopen(request, timeout=15) as response:
                raw = response.read()
                body = json.loads(raw.decode("utf-8")) if raw else None
                status = response.status
        except urllib.error.HTTPError as error:
            raw = error.read()
            try:
                body = sanitize(json.loads(raw.decode("utf-8"))) if raw else None
            except (UnicodeDecodeError, json.JSONDecodeError):
                body = None
            raise SystemExit(f"{route['id']}: HTTP {error.code}, response={body}") from error
        if status != route["expectedStatus"]:
            raise SystemExit(f"{route['id']}: expected HTTP {route['expectedStatus']}, got {status}")
        capture = route.get("capture")
        if capture:
            captured = extract(body, capture["path"])
            variables[capture["as"]] = captured
            if capture["as"] == "runToken":
                variables["runTokenId"] = captured.split(".", 1)[0]
        fixture = {
            "fixtureVersion": "1.0",
            "id": route["id"],
            "description": "Sanitized Laravel parallel-run capture.",
            "source": {"runtime": "laravel", "repository": "idelium/idelium-api", "revision": revision, "capturedAt": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"), "routeInventoryDigestSha256": "dc2a622d5825effb47aa810dfbc296348c7bcc4b16b4692c31094ed1534bbc48"},
            "route": {"method": route["method"], "path": route["path"], "trustPath": route["authentication"], "tenantOwned": True},
            "context": {"tenant": {"id": "fixture-tenant-9001", "synthetic": True}, "actor": {"id": "fixture-browser-user-9001", "synthetic": True}},
            "request": {"headers": {"Accept": "application/json"}, "query": {}, "body": None},
            "response": {"status": status, "headers": {"Content-Type": "application/json"}, "body": sanitize(body)},
            "normalizations": [], "redactions": [], "sideEffects": [],
        }
        (args.output_dir / route["output"]).write_text(json.dumps(fixture, indent=2) + "\n", encoding="utf-8")
        print(f"captured {route['id']} HTTP {status}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
