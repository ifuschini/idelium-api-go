# Audit event cutover

The Go service owns the browser `GET /api/audit-events` compatibility route.
Reads require `audit_events.read`, always include `activeTenantId` in the SQL
predicate, support the Laravel filter set, and return at most 200 newest-first
events.

Audit writes remain internal to the domain mutation that caused them. Go uses a
shared recursive redaction policy before persistence, and read serialization
applies the same policy again to protect against unsafe historical rows. The Go
repository exposes no update or delete operation for `audit_events`; records are
append-only. Database errors fail the owning transaction safely and diagnostics
do not include event payloads.

No schema change or dual write is introduced. Rollback consists of routing the
read endpoint back to Laravel. Both runtimes continue to append to the shared
table only while they own the corresponding domain mutation; they must never
record a second audit event for a mirrored write.
