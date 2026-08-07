# SBOM and Vulnerability Gates

The image job produces auditable supply-chain evidence and fails closed on
actionable vulnerabilities.

## Gates

1. `govulncheck v1.1.4` analyzes Go dependencies and reachable application
   code during the quality job.
2. Syft `v1.50.0` generates a CycloneDX JSON SBOM from the exact runtime image
   built by CI.
3. The SBOM is uploaded as `idelium-api-go-sbom` for 14 days; a missing file
   fails the job.
4. Trivy `0.73.0` scans the runtime image and fails on fixed `HIGH` or
   `CRITICAL` vulnerabilities. Unfixed findings remain visible but do not make
   a release permanently impossible without an available remediation.

Syft and Trivy images are pinned to immutable multi-platform digests. The
artifact uploader is pinned to a full action commit. Version or policy changes
must update the workflow contract tests and include review of release notes and
the new vulnerability baseline.

## Handling findings

Do not suppress a fixed high-severity finding without a documented, time-bound
risk acceptance. Update the direct dependency, base image, or build tool and
rerun all gates. SBOMs must not contain credentials or deployment secrets; CI
build arguments are limited to public build identity.

This ticket does not change the public API, database, route ownership, or
tenant data. Differential, migration, and cross-tenant tests are not
applicable. Rollback is the dedicated workflow commit, but disabling a
security gate requires explicit security and engineering approval.
