# Golden Fixture Redaction

Captured fixtures must be sanitized before they are committed. The redaction
tool removes or replaces sensitive values and records every action in the
fixture `redactions` manifest.

The tool removes:

- authorization, proxy authorization, cookies, CSRF headers, API key headers,
  and `Set-Cookie`;
- fields whose names indicate credentials, sessions, tokens, passwords, private
  keys, API keys, or client secrets;
- credential-like strings such as bearer tokens, basic authorization values,
  JWTs, and private-key blocks.

Credential-like strings found inside otherwise safe fields are replaced with the
standard `[REDACTED]` marker. Sensitive fields and headers are removed entirely.
Diagnostics and tests report only fixture paths and rules, never the secret
value.

## Usage

```sh
python3 scripts/redact_golden_fixture.py \
  --input /tmp/candidate.fixture.json \
  --output /tmp/sanitized.fixture.json
```

Validate the result before committing:

```sh
python3 scripts/validate_golden_fixtures.py /tmp/sanitized.fixture.json
```

The validator remains the commit gate. The redaction tool helps produce safe
fixtures, but it does not replace human review of the final diff.

## Deployment and Rollback

This change adds offline fixture tooling only. It does not change runtime
routing, database schema, tenant data, or Laravel behavior. Rollback is a Git
revert of the tool, tests, and documentation.
