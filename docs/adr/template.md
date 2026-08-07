# ADR-NNNN: Decision title

- Status: Proposed
- Date: YYYY-MM-DD
- Owners: Team or role
- Related issues: `#NNN`
- Supersedes: None
- Superseded by: None

## Context

Describe the problem, current constraints, affected consumers, and why a
decision is required now.

## Decision

State the decision and its boundaries. Identify route, aggregate, transaction,
and data ownership where applicable.

## Alternatives considered

Describe viable alternatives and why they were not selected.

## Consequences

List positive, negative, and operational consequences.

## Security and tenant isolation

Describe authentication, authorization, tenant ownership, credential handling,
redaction, and negative cross-tenant evidence.

## Compatibility

Describe preserved HTTP, CLI, Web, runner, database, configuration, and
persisted-data contracts. Record any versioned migration requirement.

## Deployment and rollback

Define prerequisites, approval gates, observable rollout steps, stop conditions,
fallback ownership, and rollback validation. State how writes remain
single-owner throughout the change.

## Verification

List unit, integration, differential, consumer, performance, and operational
evidence. Explain why any category is not applicable.
