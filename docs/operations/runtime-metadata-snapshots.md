# Runtime Metadata Snapshots

## Purpose

Idelium stores runtime metadata snapshots so operators can understand where a
test run executed: environment, browser, browser version, operating system,
device, target type, and execution runtime. This makes result triage more useful
without requiring the Web UI to infer platform data from deleted or renamed
catalogue records.

## Source of truth

`idelium-cli` sends a non-sensitive `executionContext` object when it creates the
performed cycle through `POST /api/ideliumcl/testcycle`.

The accepted fields are intentionally descriptive:

- `environment` and `environmentName`
- `browser` and `browserVersion`
- `device`, `deviceName`, and `deviceType`
- `platformName` and `platformVersion`
- `runtime`

The API accepts additional fields for forward compatibility, but callers must
not place credentials or authorization material in this object.

## Validation and redaction

The Go API enforces these boundaries:

- `executionContext` must be a JSON object when present;
- the encoded object must be smaller than 64 KiB;
- sensitive keys are redacted before storage, including authorization headers,
  passwords, secrets, tokens, API keys, sessions, CSRF values, and cookies;
- runtime metadata never changes tenant ownership or authorization decisions.

## Persistence strategy

This repository is currently under a Laravel schema freeze for migration safety.
For that reason this slice does not introduce a new PHP migration. The Go API
uses opportunistic persistence instead:

1. If `performed_test_cycles.executionContext` exists, it stores the redacted
   JSON there.
2. Otherwise, if `performed_test_cycles.execution_context` exists, it stores the
   same JSON there.
3. Otherwise, if `performed_test_cycles.context` exists, it stores the same JSON
   there.
4. If no compatible column exists, the route keeps the legacy create behavior and
   does not fail the CLI run.

This lets environments with an updated schema expose metadata immediately while
older installations remain compatible.

## Verification

Coverage includes:

- handler validation for invalid `executionContext` payloads;
- handler redaction for sensitive runtime metadata keys;
- MySQL integration coverage for tenant-scoped performed-cycle creation without
  a snapshot column;
- MySQL integration coverage for snapshot persistence when a compatible JSON
  column exists.

Laravel differential coverage is not applicable for this field because Laravel
ignored the additional CLI metadata. The HTTP-visible response remains
Laravel-compatible.
