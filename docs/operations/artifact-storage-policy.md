# Artifact size and storage policy

The Go API now carries the Laravel-compatible artifact policy used by result
write migrations. Artifact descriptor routes still remain Laravel-owned until
Wave 7, but Go result-write code must validate captured artifacts through this
policy before any descriptor or inline artifact is accepted.

## Default limits

| Policy | Environment override | Default |
| --- | --- | ---: |
| Descriptor artifact size | `IDELIUM_ARTIFACT_MAX_SIZE_BYTES` | `52428800` bytes |
| Inline result artifact size | `IDELIUM_ARTIFACT_INLINE_MAX_BYTES` | `262144` bytes |
| Artifact nodes per result payload | `IDELIUM_ARTIFACT_COLLECTION_MAX_ITEMS` | `50` |
| Retention window | `IDELIUM_ARTIFACT_RETENTION_DAYS` | `30` days |

## Allowed content types

- `application/json`
- `application/junit+xml`
- `application/xml`
- `text/markdown`
- `text/html`
- `text/plain`
- `image/png`
- `image/jpeg`

## Stable validation codes

| Code | Meaning |
| --- | --- |
| `ARTIFACT_FIELD_REQUIRED` | Required descriptor metadata is missing. |
| `ARTIFACT_CONTENT_TYPE_NOT_ALLOWED` | The descriptor content type is not in the allow-list. |
| `ARTIFACT_SIZE_INVALID` | The descriptor size is negative. |
| `ARTIFACT_SIZE_LIMIT_EXCEEDED` | The descriptor exceeds the configured storage limit. |
| `ARTIFACT_CHECKSUM_INVALID` | The checksum is not a SHA-256 hex digest. |
| `ARTIFACT_INLINE_SIZE_LIMIT_EXCEEDED` | An inline artifact in a result payload exceeds the configured inline limit. |
| `ARTIFACT_COLLECTION_LIMIT_EXCEEDED` | A result payload contains more artifact nodes than allowed. |

## Redaction and diagnostics

Policy violations include only safe fields: the field name, stable code, message,
limit, and actual numeric size or count. They never include storage keys,
checksums, tenant identifiers, file paths, cookies, headers, payload bodies, or
credential material.

## Rollback

No traffic is moved by this policy slice. Rollback is a normal Git revert of the
policy package and configuration wiring. Laravel remains the fallback owner for
artifact descriptor HTTP routes until the Wave 7 artifact migration tickets are
completed.
