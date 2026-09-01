# Idelium JSON import cutover

Issue #135 is implemented by the Go browser import handler and MySQL repository.
The import validates a non-empty Idelium step array, validates executable actions and
Postman collection payloads, verifies project ownership in the same transaction, and
creates steps and the test atomically. Laravel remains the rollback owner until the
gateway switch; no dual writes are enabled.
