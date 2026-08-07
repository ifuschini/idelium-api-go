#!/usr/bin/env python3
"""Validate committed golden fixtures without exposing their values."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any, Iterable


MAX_FILE_BYTES = 256 * 1024
MAX_BODY_BYTES = 64 * 1024
MAX_HEADERS = 50
MAX_SIDE_EFFECTS = 100
TRUST_PATHS = {
    "browser-session",
    "api-key",
    "run-token",
    "internal-service",
    "public-operational",
}
HTTP_METHODS = {"DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"}
FORBIDDEN_HEADERS = {
    "authorization",
    "cookie",
    "proxy-authorization",
    "set-cookie",
    "x-api-key",
    "x-csrf-token",
    "x-xsrf-token",
}
SENSITIVE_KEYS = {
    "accesstoken",
    "apikey",
    "authorization",
    "clientsecret",
    "cookie",
    "password",
    "privatekey",
    "refreshtoken",
    "secret",
    "session",
    "sessionid",
    "token",
}
SECRET_PATTERNS = (
    re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
    re.compile(r"\bBearer\s+[A-Za-z0-9._~+/=-]{8,}", re.IGNORECASE),
    re.compile(r"\bBasic\s+[A-Za-z0-9+/=]{8,}", re.IGNORECASE),
    re.compile(r"\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b"),
)
FIXTURE_ID = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
HEX_40 = re.compile(r"^[0-9a-f]{40}$")
HEX_64 = re.compile(r"^[0-9a-f]{64}$")


class FixtureValidationError(ValueError):
    """Report only structural locations and rules, never fixture values."""


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "paths",
        nargs="+",
        type=Path,
        help="Fixture JSON files or directories containing *.fixture.json files.",
    )
    return parser.parse_args()


def normalized_key(value: str) -> str:
    return re.sub(r"[^a-z0-9]", "", value.lower())


def require(condition: bool, location: str, message: str, errors: list[str]) -> None:
    if not condition:
        errors.append(f"{location}: {message}")


def require_object(value: Any, location: str, errors: list[str]) -> dict[str, Any]:
    if not isinstance(value, dict):
        errors.append(f"{location}: must be an object")
        return {}
    return value


def validate_identity(value: Any, location: str, errors: list[str]) -> None:
    if value is None:
        return
    identity = require_object(value, location, errors)
    require(identity.get("synthetic") is True, location, "must declare synthetic=true", errors)
    identifier = identity.get("id")
    require(
        isinstance(identifier, str) and identifier.startswith("fixture-"),
        f"{location}.id",
        "must use the fixture- synthetic identifier prefix",
        errors,
    )


def validate_headers(value: Any, location: str, errors: list[str]) -> None:
    headers = require_object(value, location, errors)
    require(len(headers) <= MAX_HEADERS, location, f"must contain at most {MAX_HEADERS} headers", errors)
    for name, header_value in headers.items():
        require(isinstance(name, str), location, "header names must be strings", errors)
        if isinstance(name, str) and name.lower() in FORBIDDEN_HEADERS:
            errors.append(f"{location}.{name}: sensitive headers must be removed")
        require(isinstance(header_value, str), f"{location}.{name}", "header values must be strings", errors)


def inspect_sensitive_keys(value: Any, location: str, errors: list[str]) -> None:
    if isinstance(value, dict):
        for key, child in value.items():
            child_location = f"{location}.{key}"
            if normalized_key(str(key)) in SENSITIVE_KEYS:
                errors.append(f"{child_location}: sensitive fields must be removed")
            inspect_sensitive_keys(child, child_location, errors)
    elif isinstance(value, list):
        for index, child in enumerate(value):
            inspect_sensitive_keys(child, f"{location}[{index}]", errors)
    elif isinstance(value, str):
        if any(pattern.search(value) for pattern in SECRET_PATTERNS):
            errors.append(f"{location}: value resembles a credential and must be removed")


def serialized_size(value: Any) -> int:
    return len(json.dumps(value, ensure_ascii=False, separators=(",", ":")).encode("utf-8"))


def validate_transformations(value: Any, location: str, errors: list[str]) -> None:
    if not isinstance(value, list):
        errors.append(f"{location}: must be an array")
        return
    for index, item in enumerate(value):
        record = require_object(item, f"{location}[{index}]", errors)
        require(isinstance(record.get("path"), str) and bool(record.get("path")), f"{location}[{index}].path", "is required", errors)
        require(isinstance(record.get("rule"), str) and bool(record.get("rule")), f"{location}[{index}].rule", "is required", errors)


def validate_fixture_data(data: Any) -> list[str]:
    errors: list[str] = []
    root = require_object(data, "$", errors)
    required = {
        "fixtureVersion",
        "id",
        "description",
        "source",
        "route",
        "context",
        "request",
        "response",
        "normalizations",
        "redactions",
        "sideEffects",
    }
    for field in sorted(required - root.keys()):
        errors.append(f"$.{field}: is required")

    require(root.get("fixtureVersion") == "1.0", "$.fixtureVersion", "must equal 1.0", errors)
    fixture_id = root.get("id")
    require(isinstance(fixture_id, str) and bool(FIXTURE_ID.fullmatch(fixture_id)), "$.id", "must be a lowercase kebab-case identifier", errors)
    require(isinstance(root.get("description"), str) and bool(root.get("description")), "$.description", "is required", errors)

    source = require_object(root.get("source"), "$.source", errors)
    require(source.get("runtime") in {"laravel", "go"}, "$.source.runtime", "must be laravel or go", errors)
    require(isinstance(source.get("repository"), str) and source.get("repository", "").startswith("idelium/"), "$.source.repository", "must identify an Idelium repository", errors)
    require(isinstance(source.get("revision"), str) and bool(HEX_40.fullmatch(source.get("revision", ""))), "$.source.revision", "must be an immutable 40-character Git revision", errors)
    require(isinstance(source.get("capturedAt"), str) and source.get("capturedAt", "").endswith("Z"), "$.source.capturedAt", "must be a UTC timestamp", errors)
    require(isinstance(source.get("routeInventoryDigestSha256"), str) and bool(HEX_64.fullmatch(source.get("routeInventoryDigestSha256", ""))), "$.source.routeInventoryDigestSha256", "must be a SHA-256 digest", errors)

    route = require_object(root.get("route"), "$.route", errors)
    require(route.get("method") in HTTP_METHODS, "$.route.method", "must be a supported HTTP method", errors)
    require(isinstance(route.get("path"), str) and route.get("path", "").startswith("/"), "$.route.path", "must be an absolute route path", errors)
    require(route.get("trustPath") in TRUST_PATHS, "$.route.trustPath", "must use a canonical trust path", errors)
    require(isinstance(route.get("tenantOwned"), bool), "$.route.tenantOwned", "must be a boolean", errors)

    context = require_object(root.get("context"), "$.context", errors)
    validate_identity(context.get("tenant"), "$.context.tenant", errors)
    validate_identity(context.get("actor"), "$.context.actor", errors)
    if route.get("tenantOwned") is True:
        require(context.get("tenant") is not None, "$.context.tenant", "is required for a tenant-owned route", errors)

    request = require_object(root.get("request"), "$.request", errors)
    response = require_object(root.get("response"), "$.response", errors)
    validate_headers(request.get("headers"), "$.request.headers", errors)
    validate_headers(response.get("headers"), "$.response.headers", errors)
    status = response.get("status")
    require(isinstance(status, int) and 100 <= status <= 599, "$.response.status", "must be an HTTP status code", errors)
    for location, body in (("$.request.body", request.get("body")), ("$.response.body", response.get("body"))):
        require(serialized_size(body) <= MAX_BODY_BYTES, location, f"must not exceed {MAX_BODY_BYTES} serialized bytes", errors)
        inspect_sensitive_keys(body, location, errors)
    inspect_sensitive_keys(request.get("query", {}), "$.request.query", errors)

    validate_transformations(root.get("normalizations"), "$.normalizations", errors)
    validate_transformations(root.get("redactions"), "$.redactions", errors)
    side_effects = root.get("sideEffects")
    require(isinstance(side_effects, list), "$.sideEffects", "must be an array", errors)
    if isinstance(side_effects, list):
        require(len(side_effects) <= MAX_SIDE_EFFECTS, "$.sideEffects", f"must contain at most {MAX_SIDE_EFFECTS} records", errors)
        inspect_sensitive_keys(side_effects, "$.sideEffects", errors)
    return errors


def fixture_paths(paths: Iterable[Path]) -> list[Path]:
    files: list[Path] = []
    for path in paths:
        if path.is_dir():
            files.extend(path.rglob("*.fixture.json"))
        else:
            files.append(path)
    return sorted(set(files))


def validate_file(path: Path) -> list[str]:
    if path.stat().st_size > MAX_FILE_BYTES:
        return [f"$: fixture must not exceed {MAX_FILE_BYTES} bytes"]
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError):
        return ["$: fixture must contain valid UTF-8 JSON"]
    return validate_fixture_data(data)


def main() -> int:
    files = fixture_paths(parse_args().paths)
    if not files:
        print("No golden fixtures were found.", file=sys.stderr)
        return 1
    failed = False
    for path in files:
        errors = validate_file(path)
        if errors:
            failed = True
            for error in errors:
                print(f"{path}: {error}", file=sys.stderr)
        else:
            print(f"Validated {path}")
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
